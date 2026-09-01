package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	dbsqlc "github.com/indu-forge/dev_core/internal/platform/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgreSQLRepository struct {
	pool    *pgxpool.Pool
	queries *dbsqlc.Queries
}

func NewPostgreSQLRepository(pool *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{pool: pool, queries: dbsqlc.New(pool)}
}

func (r *PostgreSQLRepository) GetProject(ctx context.Context, tenantID, projectID string) (Project, error) {
	tenantUUID, projectUUID, err := parsePair(tenantID, projectID)
	if err != nil {
		return Project{}, ErrNotFound
	}
	row, err := r.queries.GetProject(ctx, dbsqlc.GetProjectParams{ProjectID: projectUUID, TenantID: tenantUUID})
	if err != nil {
		return Project{}, mapNotFound(err)
	}
	return Project{ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Name: row.Name, Code: row.Code, WorkspacePath: row.WorkspacePath, CreatedBy: uuidString(row.CreatedBy), Visibility: row.Visibility}, nil
}

func (r *PostgreSQLRepository) ListVersions(ctx context.Context, tenantID, projectID string, page, limit int) ([]Version, int64, error) {
	tenantUUID, projectUUID, err := parsePair(tenantID, projectID)
	if err != nil {
		return nil, 0, ErrNotFound
	}
	rows, err := r.queries.ListApplicationVersions(ctx, dbsqlc.ListApplicationVersionsParams{ProjectID: projectUUID, TenantID: tenantUUID, PageOffset: int32((page - 1) * limit), PageLimit: int32(limit)})
	if err != nil {
		return nil, 0, err
	}
	total, err := r.queries.CountApplicationVersions(ctx, dbsqlc.CountApplicationVersionsParams{ProjectID: projectUUID, TenantID: tenantUUID})
	if err != nil {
		return nil, 0, err
	}
	items := make([]Version, 0, len(rows))
	for _, row := range rows {
		items = append(items, versionFromModel(row))
	}
	return items, total, nil
}

func (r *PostgreSQLRepository) GetVersion(ctx context.Context, tenantID, versionID string) (Version, error) {
	tenantUUID, versionUUID, err := parsePair(tenantID, versionID)
	if err != nil {
		return Version{}, ErrNotFound
	}
	row, err := r.queries.GetApplicationVersion(ctx, dbsqlc.GetApplicationVersionParams{VersionID: versionUUID, TenantID: tenantUUID})
	if err != nil {
		return Version{}, mapNotFound(err)
	}
	return versionFromModel(row), nil
}

func (r *PostgreSQLRepository) GetDeployment(ctx context.Context, tenantID, nodeDeploymentID string) (Deployment, error) {
	tenantUUID, deploymentUUID, err := parsePair(tenantID, nodeDeploymentID)
	if err != nil {
		return Deployment{}, ErrNotFound
	}
	row, err := r.queries.GetNodeDeployment(ctx, dbsqlc.GetNodeDeploymentParams{NodeDeploymentID: deploymentUUID, TenantID: tenantUUID})
	if err != nil {
		return Deployment{}, mapNotFound(err)
	}
	return deploymentFromModel(row), nil
}

func (r *PostgreSQLRepository) CreateVersion(ctx context.Context, input CreateVersionInput) (Version, error) {
	tenantUUID, projectUUID, userUUID, err := parseTriple(input.Project.TenantID, input.Project.ID, input.CreatedBy)
	if err != nil {
		return Version{}, ErrNotFound
	}
	manifest, err := json.Marshal(input.Manifest)
	if err != nil {
		return Version{}, err
	}
	row, err := r.queries.CreateApplicationVersion(ctx, dbsqlc.CreateApplicationVersionParams{
		TenantID: tenantUUID, ProjectID: projectUUID, Version: input.Version,
		Name: nullableText(input.Name), Description: nullableText(input.Description), SourceHash: nullableText(input.SourceHash),
		ArtifactBucket: nullableText(input.Bucket), ArtifactKey: nullableText(input.ArtifactKey), ArtifactHash: nullableText(input.ArtifactHash),
		ArtifactSize: pgtype.Int8{Int64: input.ArtifactSize, Valid: true}, Manifest: manifest, UserID: userUUID,
	})
	if err != nil {
		return Version{}, mapConstraint(err)
	}
	return versionFromModel(row), nil
}

func (r *PostgreSQLRepository) BeginProductionBuild(ctx context.Context, project Project, userID, name, description string) (Version, error) {
	tenant, projectID, user, err := parseTriple(project.TenantID, project.ID, userID)
	if err != nil {
		return Version{}, ErrNotFound
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Version{}, err
	}
	defer tx.Rollback(ctx)
	var locked string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM projects WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, tenant, projectID).Scan(&locked); err != nil {
		return Version{}, mapNotFound(err)
	}
	var patch int
	if err = tx.QueryRow(ctx, `SELECT COALESCE(max((regexp_match(version, '^1\\.0\\.([0-9]+)$'))[1]::integer),-1) FROM application_versions WHERE project_id=$1 AND deleted_at IS NULL AND version <> '__DEV__'`, projectID).Scan(&patch); err != nil {
		return Version{}, err
	}
	version := fmt.Sprintf("1.0.%d", patch+1)
	var result dbsqlc.ApplicationVersion
	err = tx.QueryRow(ctx, `INSERT INTO application_versions(tenant_id,project_id,version,name,description,status,created_by) VALUES($1,$2,$3,$4,$5,'building',$6) RETURNING id,tenant_id,project_id,version,name,description,status,source_hash,artifact_bucket,artifact_key,artifact_hash,artifact_size,manifest,build_log,error_message,completed_at,deleted_at,created_at,updated_at`, tenant, projectID, version, nullableText(name), nullableText(description), user).Scan(&result.ID, &result.TenantID, &result.ProjectID, &result.Version, &result.Name, &result.Description, &result.Status, &result.SourceHash, &result.ArtifactBucket, &result.ArtifactKey, &result.ArtifactHash, &result.ArtifactSize, &result.Manifest, &result.BuildLog, &result.ErrorMessage, &result.CompletedAt, &result.DeletedAt, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return Version{}, mapConstraint(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return Version{}, err
	}
	return versionFromModel(result), nil
}

func (r *PostgreSQLRepository) MarkVersionReady(ctx context.Context, tenantID, versionID string, input VersionReadyInput) (Version, error) {
	tenant, version, err := parsePair(tenantID, versionID)
	if err != nil {
		return Version{}, ErrNotFound
	}
	manifest, err := json.Marshal(input.Manifest)
	if err != nil {
		return Version{}, err
	}
	row, err := r.queries.MarkApplicationVersionReady(ctx, dbsqlc.MarkApplicationVersionReadyParams{ArtifactBucket: nullableText(input.Bucket), ArtifactKey: nullableText(input.ArtifactKey), ArtifactHash: nullableText(input.ArtifactHash), ArtifactSize: pgtype.Int8{Int64: input.ArtifactSize, Valid: true}, Manifest: manifest, VersionID: version, TenantID: tenant})
	if err != nil {
		return Version{}, mapNotFound(err)
	}
	return versionFromModel(row), nil
}
func (r *PostgreSQLRepository) MarkVersionFailed(ctx context.Context, tenantID, versionID, message string) (Version, error) {
	tenant, version, err := parsePair(tenantID, versionID)
	if err != nil {
		return Version{}, ErrNotFound
	}
	row, err := r.queries.MarkApplicationVersionFailed(ctx, dbsqlc.MarkApplicationVersionFailedParams{VersionID: version, TenantID: tenant, ErrorMessage: nullableText(message)})
	if err != nil {
		return Version{}, mapNotFound(err)
	}
	return versionFromModel(row), nil
}

func (r *PostgreSQLRepository) DeleteVersion(ctx context.Context, tenantID, versionID string) error {
	if _, err := r.GetVersion(ctx, tenantID, versionID); err != nil {
		return err
	}
	tenantUUID, versionUUID, _ := parsePair(tenantID, versionID)
	rows, err := r.queries.DeleteApplicationVersion(ctx, dbsqlc.DeleteApplicationVersionParams{VersionID: versionUUID, TenantID: tenantUUID})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrVersionInUse
	}
	return nil
}

func (r *PostgreSQLRepository) ListProjectDeployments(ctx context.Context, tenantID, projectID string) ([]Deployment, error) {
	tenantUUID, projectUUID, err := parsePair(tenantID, projectID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := r.queries.ListProjectNodeDeployments(ctx, dbsqlc.ListProjectNodeDeploymentsParams{ProjectID: projectUUID, TenantID: tenantUUID})
	if err != nil {
		return nil, err
	}
	items := make([]Deployment, 0, len(rows))
	for _, row := range rows {
		items = append(items, deploymentFromList(row))
	}
	return items, nil
}

func (r *PostgreSQLRepository) Deploy(ctx context.Context, tenantID string, input DeployInput) ([]Deployment, error) {
	if len(input.NodeIDs) == 0 {
		return nil, fmt.Errorf("至少选择一个目标节点")
	}
	tenantUUID, projectUUID, userUUID, err := parseTriple(tenantID, input.Project.ID, input.DeployedBy)
	if err != nil {
		return nil, ErrNotFound
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := r.queries.WithTx(tx)
	configJSON, err := json.Marshal(input.RuntimeConfig)
	if err != nil {
		return nil, err
	}
	versionID := pgtype.UUID{}
	versionText := "DEV"
	if input.Version != nil {
		versionID, err = parseUUID(input.Version.ID)
		if err != nil {
			return nil, ErrNotFound
		}
		versionText = input.Version.Version
	}
	items := make([]Deployment, 0, len(input.NodeIDs))
	for _, rawNodeID := range input.NodeIDs {
		nodeUUID, err := parseUUID(rawNodeID)
		if err != nil {
			return nil, ErrNotFound
		}
		if _, err := queries.GetApprovedDeploymentNode(ctx, dbsqlc.GetApprovedDeploymentNodeParams{NodeID: nodeUUID, TenantID: tenantUUID}); err != nil {
			return nil, mapNotFound(err)
		}
		if err := queries.RemoveActiveNodeDeployment(ctx, dbsqlc.RemoveActiveNodeDeploymentParams{NodeID: nodeUUID, ProjectID: projectUUID}); err != nil {
			return nil, err
		}
		created, err := queries.CreateNodeDeployment(ctx, dbsqlc.CreateNodeDeploymentParams{
			TenantID: tenantUUID, NodeID: nodeUUID, ProjectID: projectUUID, ApplicationVersionID: versionID,
			Version: nullableText(versionText), Mode: input.Mode, RuntimeConfig: configJSON, UserID: userUUID,
		})
		if err != nil {
			return nil, err
		}
		payload, _ := json.Marshal(map[string]any{
			"nodeDeploymentId": uuidString(created.ID), "projectId": input.Project.ID, "version": versionText,
			"mode": input.Mode, "artifactUrl": input.ArtifactURL, "artifactKey": input.ArtifactKey, "artifactHash": input.ArtifactHash,
			"runtimeConfig": input.RuntimeConfig,
		})
		if _, err := queries.CreateNodeCommand(ctx, dbsqlc.CreateNodeCommandParams{
			TenantID: tenantUUID, NodeID: nodeUUID, NodeDeploymentID: created.ID, ProjectID: projectUUID,
			CommandType: "deploy", Payload: payload,
		}); err != nil {
			return nil, err
		}
		items = append(items, deploymentFromModel(created))
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *PostgreSQLRepository) Operate(ctx context.Context, tenantID, nodeDeploymentID, operation string) (Deployment, error) {
	tenantUUID, deploymentUUID, err := parsePair(tenantID, nodeDeploymentID)
	if err != nil {
		return Deployment{}, ErrNotFound
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Deployment{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := r.queries.WithTx(tx)
	current, err := queries.GetNodeDeployment(ctx, dbsqlc.GetNodeDeploymentParams{NodeDeploymentID: deploymentUUID, TenantID: tenantUUID})
	if err != nil {
		return Deployment{}, mapNotFound(err)
	}
	status := map[string]string{"start": "deploying", "stop": "stopped", "restart": "deploying", "remove": "removed"}[operation]
	updated, err := queries.UpdateNodeDeploymentState(ctx, dbsqlc.UpdateNodeDeploymentStateParams{Status: status, NodeDeploymentID: deploymentUUID, TenantID: tenantUUID})
	if err != nil {
		return Deployment{}, mapNotFound(err)
	}
	payload, _ := json.Marshal(map[string]any{"nodeDeploymentId": nodeDeploymentID, "projectId": uuidString(current.ProjectID)})
	if _, err := queries.CreateNodeCommand(ctx, dbsqlc.CreateNodeCommandParams{
		TenantID: tenantUUID, NodeID: current.NodeID, NodeDeploymentID: current.ID, ProjectID: current.ProjectID,
		CommandType: operation, Payload: payload,
	}); err != nil {
		return Deployment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Deployment{}, err
	}
	return deploymentFromModel(updated), nil
}

func versionFromModel(row dbsqlc.ApplicationVersion) Version {
	item := Version{
		ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), ProjectID: uuidString(row.ProjectID), Version: row.Version,
		Name: textString(row.Name), Description: textString(row.Description), Status: row.Status,
		SourceHash: textString(row.SourceHash), ArtifactKey: textString(row.ArtifactKey), ArtifactBucket: textString(row.ArtifactBucket), ArtifactHash: textString(row.ArtifactHash),
		ArtifactSize: int64Value(row.ArtifactSize), ErrorMessage: textString(row.ErrorMessage), CompletedAt: timePointer(row.CompletedAt),
		CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
	decodeJSONValue(row.Manifest, &item.Manifest)
	return item
}

func deploymentFromModel(row dbsqlc.NodeDeployment) Deployment {
	item := Deployment{
		ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), NodeID: uuidString(row.NodeID), ProjectID: uuidString(row.ProjectID),
		ApplicationVersionID: uuidString(row.ApplicationVersionID), Version: textString(row.Version), Mode: row.Mode, Status: row.Status,
		ErrorMessage: textString(row.ErrorMessage), CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
	decodeJSONValue(row.RuntimeConfig, &item.RuntimeConfig)
	decodeJSONValue(row.RuntimeMetrics, &item.RuntimeMetrics)
	return item
}

func deploymentFromList(row dbsqlc.ListProjectNodeDeploymentsRow) Deployment {
	item := Deployment{
		ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), NodeID: uuidString(row.NodeID), ProjectID: uuidString(row.ProjectID),
		ApplicationVersionID: uuidString(row.ApplicationVersionID), Version: textString(row.Version), Mode: row.Mode, Status: row.Status,
		ErrorMessage: textString(row.ErrorMessage), NodeName: row.NodeName, NodeStatus: row.NodeStatus,
		IPAddress: textString(row.IpAddress), ProjectName: row.ProjectName, ProjectCode: row.ProjectCode,
		ArtifactHash: textString(row.ArtifactHash), CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
	decodeJSONValue(row.RuntimeConfig, &item.RuntimeConfig)
	decodeJSONValue(row.RuntimeMetrics, &item.RuntimeMetrics)
	decodeJSONValue(row.Manifest, &item.Manifest)
	return item
}

func decodeJSONValue(value any, target any) {
	if raw, ok := value.([]byte); ok {
		_ = json.Unmarshal(raw, target)
		return
	}
	raw, err := json.Marshal(value)
	if err == nil {
		_ = json.Unmarshal(raw, target)
	}
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func mapConstraint(err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" {
		return ErrAlreadyExists
	}
	return err
}

func parseUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func parsePair(first, second string) (pgtype.UUID, pgtype.UUID, error) {
	firstUUID, err := parseUUID(first)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	secondUUID, err := parseUUID(second)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	return firstUUID, secondUUID, nil
}

func parseTriple(first, second, third string) (pgtype.UUID, pgtype.UUID, pgtype.UUID, error) {
	firstUUID, secondUUID, err := parsePair(first, second)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
	}
	thirdUUID, err := parseUUID(third)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, pgtype.UUID{}, err
	}
	return firstUUID, secondUUID, thirdUUID, nil
}

func uuidString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return uuid.UUID(value.Bytes).String()
}

func textString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullableText(value string) pgtype.Text { return pgtype.Text{String: value, Valid: value != ""} }

func int64Value(value pgtype.Int8) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}

func timePointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
