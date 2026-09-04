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

const staleProductionBuildTimeout = 15 * time.Minute

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
	return Project{ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Name: row.Name, Code: row.Code, WorkspacePath: row.WorkspacePath, CreatedBy: uuidString(row.CreatedBy), Visibility: row.Visibility, AuthoringEpoch: row.AuthoringEpoch}, nil
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
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, project.ID); err != nil {
		return Version{}, err
	}
	var locked string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM projects WHERE tenant_id=$1 AND id=$2 FOR UPDATE`, tenant, projectID).Scan(&locked); err != nil {
		return Version{}, mapNotFound(err)
	}
	var restoring bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM authoring_restore_tasks WHERE project_id=$1 AND state IN ('queued','staging','restoring_workspace','restoring_scenes','restoring_data','finalizing','compensating'))`, project.ID).Scan(&restoring); err != nil {
		return Version{}, err
	}
	if restoring {
		return Version{}, ErrAuthoringBusy
	}
	// 发布请求中断时无法保证失败回写可达。只收口超过最大受控构建窗口的旧记录，
	// 不影响仍在执行的 Release Builder；版本号继续单调递增且不可复用。
	if _, err = tx.Exec(ctx, `
		UPDATE application_versions
		SET status='failed', error_message='正式 Release 构建超时，已由下次发布恢复', completed_at=now(), updated_at=now()
		WHERE tenant_id=$1 AND project_id=$2 AND status='building'
		  AND created_at < now() - $3::interval
	`, tenant, projectID, fmt.Sprintf("%d seconds", int(staleProductionBuildTimeout.Seconds()))); err != nil {
		return Version{}, err
	}
	var building bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM application_versions WHERE tenant_id=$1 AND project_id=$2 AND status='building')`, tenant, projectID).Scan(&building); err != nil {
		return Version{}, err
	}
	if building {
		return Version{}, ErrAuthoringBusy
	}
	var patch int
	// 软删除版本仍受 (project_id, version) 唯一约束保护，正式版本号必须保持单调且不可复用。
	if err = tx.QueryRow(ctx, `SELECT COALESCE(max((regexp_match(version, '^1\.0\.([0-9]+)$'))[1]::integer),-1) FROM application_versions WHERE project_id=$1 AND version <> '__DEV__'`, projectID).Scan(&patch); err != nil {
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
	row, err := r.queries.MarkApplicationVersionReady(ctx, dbsqlc.MarkApplicationVersionReadyParams{
		ArtifactBucket: nullableText(input.Bucket), ArtifactKey: nullableText(input.ArtifactKey), ArtifactHash: nullableText(input.ArtifactHash),
		ArtifactSize: pgtype.Int8{Int64: input.ArtifactSize, Valid: true}, Manifest: manifest, ManifestHash: nullableText(input.ManifestHash),
		ChecksumsHash: nullableText(input.ChecksumsHash), SigningKeyID: nullableText(input.SigningKeyID),
		AuthoringSnapshotSchema: nullableText(input.AuthoringSnapshotSchema), AuthoringSnapshotBucket: nullableText(input.AuthoringSnapshotBucket),
		AuthoringSnapshotKey: nullableText(input.AuthoringSnapshotKey), AuthoringSnapshotHash: nullableText(input.AuthoringSnapshotHash),
		AuthoringSnapshotCipherHash: nullableText(input.AuthoringSnapshotCipherHash), AuthoringSnapshotSize: pgtype.Int8{Int64: input.AuthoringSnapshotSize, Valid: true},
		AuthoringSnapshotKeyID: nullableText(input.AuthoringSnapshotKeyID), AuthoringProjectRevision: nullableText(input.AuthoringProjectRevision),
		Restorable: input.Restorable,
		VersionID:  version, TenantID: tenant,
	})
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

func (r *PostgreSQLRepository) GetRestoreTask(ctx context.Context, tenantID, taskID string) (RestoreTask, error) {
	tenant, task, err := parsePair(tenantID, taskID)
	if err != nil {
		return RestoreTask{}, ErrNotFound
	}
	var item RestoreTask
	var started, completed pgtype.Timestamptz
	err = r.pool.QueryRow(ctx, `SELECT id,tenant_id,project_id,application_version_id,requested_by,COALESCE(current_project_revision,''),expected_authoring_epoch,target_authoring_epoch,state,stage,COALESCE(backup_bucket,''),COALESCE(backup_key,''),COALESCE(backup_hash,''),COALESCE(backup_cipher_hash,''),COALESCE(backup_size,0),COALESCE(backup_key_id,''),COALESCE(backup_project_revision,''),rolled_back,workspace_was_running,COALESCE(error_message,''),started_at,completed_at,created_at,updated_at FROM authoring_restore_tasks WHERE id=$1 AND tenant_id=$2`, task, tenant).Scan(&item.ID, &item.TenantID, &item.ProjectID, &item.VersionID, &item.RequestedBy, &item.CurrentProjectRevision, &item.ExpectedEpoch, &item.TargetEpoch, &item.State, &item.Stage, &item.BackupBucket, &item.BackupKey, &item.BackupHash, &item.BackupCipherHash, &item.BackupSize, &item.BackupKeyID, &item.BackupProjectRevision, &item.RolledBack, &item.WorkspaceWasRunning, &item.ErrorMessage, &started, &completed, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return RestoreTask{}, mapNotFound(err)
	}
	item.StartedAt = timePointer(started)
	item.CompletedAt = timePointer(completed)
	return item, nil
}

func (r *PostgreSQLRepository) DeleteVersion(ctx context.Context, tenantID, versionID string) error {
	item, err := r.GetVersion(ctx, tenantID, versionID)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, item.ProjectID); err != nil {
		return err
	}
	command, err := tx.Exec(ctx, `UPDATE application_versions v SET status='deleted',deleted_at=now(),updated_at=now() WHERE v.id=$1 AND v.tenant_id=$2 AND NOT EXISTS(SELECT 1 FROM node_deployments d WHERE d.application_version_id=v.id AND d.deleted_at IS NULL AND d.status IN ('pending','deploying','running')) AND NOT EXISTS(SELECT 1 FROM authoring_restore_tasks t WHERE t.application_version_id=v.id AND t.state IN ('queued','staging','restoring_workspace','restoring_scenes','restoring_data','finalizing','compensating'))`, versionID, tenantID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return ErrVersionInUse
	}
	return tx.Commit(ctx)
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
		ArtifactSize: int64Value(row.ArtifactSize), ManifestHash: textString(row.ManifestHash), ChecksumsHash: textString(row.ChecksumsHash), SigningKeyID: textString(row.SigningKeyID), ErrorMessage: textString(row.ErrorMessage), CompletedAt: timePointer(row.CompletedAt),
		Restorable: row.Restorable, AuthoringProjectRevision: textString(row.AuthoringProjectRevision), AuthoringSnapshotSchema: textString(row.AuthoringSnapshotSchema), AuthoringSnapshotBucket: textString(row.AuthoringSnapshotBucket), AuthoringSnapshotKey: textString(row.AuthoringSnapshotKey), AuthoringSnapshotHash: textString(row.AuthoringSnapshotHash), AuthoringSnapshotCipherHash: textString(row.AuthoringSnapshotCipherHash), AuthoringSnapshotKeyID: textString(row.AuthoringSnapshotKeyID), AuthoringSnapshotSize: int64Value(row.AuthoringSnapshotSize),
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
