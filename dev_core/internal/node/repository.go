package node

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	dbsqlc "github.com/indu-forge/dev_core/internal/platform/db/sqlc"
	"github.com/jackc/pgx/v5"
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

func (r *PostgreSQLRepository) List(ctx context.Context, tenantID string, filter ListFilter) ([]Node, int64, error) {
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		return nil, 0, ErrNotFound
	}
	params := dbsqlc.ListNodesParams{
		TenantID: tenantUUID, Status: filter.Status, ApprovalStatus: filter.ApprovalStatus,
		Keyword: filter.Keyword, PageOffset: int32((filter.Page - 1) * filter.Limit), PageLimit: int32(filter.Limit),
	}
	rows, err := r.queries.ListNodes(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := r.queries.CountNodes(ctx, dbsqlc.CountNodesParams{TenantID: tenantUUID, Status: filter.Status, ApprovalStatus: filter.ApprovalStatus, Keyword: filter.Keyword})
	if err != nil {
		return nil, 0, err
	}
	items := make([]Node, 0, len(rows))
	for _, row := range rows {
		items = append(items, nodeFromList(row))
	}
	return items, total, nil
}

func (r *PostgreSQLRepository) Approve(ctx context.Context, tenantID, nodeID, userID string) (Node, error) {
	tenantUUID, nodeUUID, userUUID, err := parseTriple(tenantID, nodeID, userID)
	if err != nil {
		return Node{}, ErrNotFound
	}
	row, err := r.queries.ApproveNode(ctx, dbsqlc.ApproveNodeParams{UserID: userUUID, NodeID: nodeUUID, TenantID: tenantUUID})
	if err != nil {
		return Node{}, mapNotFound(err)
	}
	return nodeFromModel(row), nil
}

func (r *PostgreSQLRepository) Reject(ctx context.Context, tenantID, nodeID, userID string) (Node, error) {
	tenantUUID, nodeUUID, userUUID, err := parseTriple(tenantID, nodeID, userID)
	if err != nil {
		return Node{}, ErrNotFound
	}
	row, err := r.queries.RejectNode(ctx, dbsqlc.RejectNodeParams{UserID: userUUID, NodeID: nodeUUID, TenantID: tenantUUID})
	if err != nil {
		return Node{}, mapNotFound(err)
	}
	return nodeFromModel(row), nil
}

func (r *PostgreSQLRepository) Delete(ctx context.Context, tenantID, nodeID string) error {
	tenantUUID, nodeUUID, err := parsePair(tenantID, nodeID)
	if err != nil {
		return ErrNotFound
	}
	rows, err := r.queries.DeleteNode(ctx, dbsqlc.DeleteNodeParams{NodeID: nodeUUID, TenantID: tenantUUID})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgreSQLRepository) Register(ctx context.Context, input RegisterInput) (Node, error) {
	nodeUUID, tenantUUID, userUUID, err := parseTriple(input.NodeID, input.Registrant.TenantID, input.Registrant.ID)
	if err != nil {
		return Node{}, ErrNotFound
	}
	metadata, err := json.Marshal(map[string]any{
		"description": input.Description, "port": input.Port, "mode": input.Mode, "userAgent": input.UserAgent,
		"registrant": map[string]any{"id": input.Registrant.ID, "username": input.Registrant.Username},
	})
	if err != nil {
		return Node{}, err
	}
	status := "offline"
	approvalStatus := "pending"
	approvedAt := pgtype.Timestamptz{}
	approvedBy := pgtype.UUID{}
	if input.AutoApprove {
		approvalStatus = "approved"
		approvedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		approvedBy = userUUID
	}
	row, err := r.queries.RegisterNode(ctx, dbsqlc.RegisterNodeParams{
		NodeID: nodeUUID, TenantID: tenantUUID, Name: input.Name, Status: status, ApprovalStatus: approvalStatus,
		RegistrationTokenHash: nullableText(input.RegistrationHash), AgentVersion: nullableText(input.AgentVersion),
		IpAddress: nullableText(input.IPAddress), Metadata: metadata, ApprovedAt: approvedAt, ApprovedBy: approvedBy,
	})
	if err != nil {
		return Node{}, err
	}
	return nodeFromModel(row), nil
}

func (r *PostgreSQLRepository) GetApprovalStatus(ctx context.Context, nodeID string) (ApprovalStatus, error) {
	nodeUUID, err := parseUUID(nodeID)
	if err != nil {
		return ApprovalStatus{}, ErrNotFound
	}
	row, err := r.queries.GetNodeApprovalStatus(ctx, nodeUUID)
	if err != nil {
		return ApprovalStatus{}, mapNotFound(err)
	}
	return ApprovalStatus{NodeID: uuidString(row.ID), NodeName: row.Name, Status: row.Status, ApprovalStatus: row.ApprovalStatus, ApprovedAt: timePointer(row.ApprovedAt), RejectedAt: timePointer(row.RejectedAt), UpdatedAt: row.UpdatedAt.Time}, nil
}

func (r *PostgreSQLRepository) Heartbeat(ctx context.Context, nodeID, tokenHash string, input HeartbeatInput) (Node, []Command, error) {
	nodeUUID, err := parseUUID(nodeID)
	if err != nil {
		return Node{}, nil, ErrNotFound
	}
	metrics, err := json.Marshal(input.Metrics)
	if err != nil {
		return Node{}, nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Node{}, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := r.queries.WithTx(tx)
	row, err := queries.UpdateNodeHeartbeat(ctx, dbsqlc.UpdateNodeHeartbeatParams{Metrics: metrics, AgentVersion: nullableText(input.AgentVersion), NodeID: nodeUUID, RegistrationTokenHash: nullableText(tokenHash)})
	if err != nil {
		return Node{}, nil, mapNotFound(err)
	}
	commandRows, err := queries.ListPendingNodeCommands(ctx, nodeUUID)
	if err != nil {
		return Node{}, nil, err
	}
	commands := make([]Command, 0, len(commandRows))
	for _, commandRow := range commandRows {
		payload := map[string]any{}
		_ = json.Unmarshal(commandRow.Payload, &payload)
		commands = append(commands, Command{Type: commandRow.CommandType, Payload: payload})
		if err := queries.MarkNodeCommandIssued(ctx, commandRow.ID); err != nil {
			return Node{}, nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Node{}, nil, err
	}
	return nodeFromModel(row), commands, nil
}

func (r *PostgreSQLRepository) Offline(ctx context.Context, nodeID, tokenHash, reason string) (Node, error) {
	nodeUUID, err := parseUUID(nodeID)
	if err != nil {
		return Node{}, ErrNotFound
	}
	metadata, _ := json.Marshal(map[string]any{"offlineReason": reason, "offlineAt": time.Now().UTC().Format(time.RFC3339)})
	row, err := r.queries.UpdateNodeOffline(ctx, dbsqlc.UpdateNodeOfflineParams{Metadata: metadata, NodeID: nodeUUID, RegistrationTokenHash: nullableText(tokenHash)})
	if err != nil {
		return Node{}, mapNotFound(err)
	}
	return nodeFromModel(row), nil
}

func (r *PostgreSQLRepository) ReportDeployment(ctx context.Context, nodeID, tokenHash string, input DeploymentReport) (DeploymentUpdate, error) {
	nodeUUID, deploymentUUID, err := parsePair(nodeID, input.DeploymentID)
	if err != nil {
		return DeploymentUpdate{}, ErrNotFound
	}
	if _, err := r.queries.GetNodeByToken(ctx, dbsqlc.GetNodeByTokenParams{NodeID: nodeUUID, RegistrationTokenHash: nullableText(tokenHash)}); err != nil {
		return DeploymentUpdate{}, mapNotFound(err)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return DeploymentUpdate{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := r.queries.WithTx(tx)
	row, err := queries.UpdateNodeReportedDeployment(ctx, dbsqlc.UpdateNodeReportedDeploymentParams{
		Status: input.Status, ErrorMessage: nullableText(input.ErrorMessage), Message: nullableText(input.Message),
		StartedAt: timeValue(input.StartedAt), StoppedAt: timeValue(input.StoppedAt), NodeDeploymentID: deploymentUUID, NodeID: nodeUUID,
	})
	if err != nil {
		return DeploymentUpdate{}, mapNotFound(err)
	}
	commandStatus := "completed"
	if input.Status == "failed" {
		commandStatus = "failed"
	}
	if input.Status == "running" || input.Status == "stopped" || input.Status == "failed" || input.Status == "removed" {
		if err := queries.CompleteNodeDeploymentCommands(ctx, dbsqlc.CompleteNodeDeploymentCommandsParams{Status: commandStatus, LastError: nullableText(input.ErrorMessage), NodeDeploymentID: deploymentUUID}); err != nil {
			return DeploymentUpdate{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return DeploymentUpdate{}, err
	}
	return DeploymentUpdate{
		ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), NodeID: uuidString(row.NodeID), ProjectID: uuidString(row.ProjectID),
		Status: row.Status, ErrorMessage: textString(row.ErrorMessage), StartedAt: timePointer(row.StartedAt), StoppedAt: timePointer(row.StoppedAt),
	}, nil
}

func nodeFromModel(row dbsqlc.Node) Node {
	item := Node{
		ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Name: row.Name, Code: textString(row.Code),
		NodeType: row.NodeType, Status: row.Status, ApprovalStatus: row.ApprovalStatus,
		AgentVersion: textString(row.AgentVersion), OS: textString(row.Os), Architecture: textString(row.Architecture),
		Hostname: textString(row.Hostname), IPAddress: textString(row.IpAddress), LastHeartbeatAt: timePointer(row.LastHeartbeatAt),
		ApprovedAt: timePointer(row.ApprovedAt), RejectedAt: timePointer(row.RejectedAt), CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
	decodeJSON(row.Capabilities, &item.Capabilities)
	decodeJSON(row.Metrics, &item.Metrics)
	decodeJSON(row.Metadata, &item.Metadata)
	return item
}

func nodeFromList(row dbsqlc.ListNodesRow) Node {
	item := Node{
		ID: uuidString(row.ID), TenantID: uuidString(row.TenantID), Name: row.Name, Code: textString(row.Code),
		NodeType: row.NodeType, Status: row.Status, ApprovalStatus: row.ApprovalStatus,
		AgentVersion: textString(row.AgentVersion), OS: textString(row.Os), Architecture: textString(row.Architecture),
		Hostname: textString(row.Hostname), IPAddress: textString(row.IpAddress), LastHeartbeatAt: timePointer(row.LastHeartbeatAt),
		ApprovedAt: timePointer(row.ApprovedAt), RejectedAt: timePointer(row.RejectedAt), CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
	decodeJSON(row.Capabilities, &item.Capabilities)
	decodeJSON(row.Metrics, &item.Metrics)
	decodeJSON(row.Metadata, &item.Metadata)
	decodeJSON(row.Deployments, &item.Deployments)
	if item.Deployments == nil {
		item.Deployments = []map[string]any{}
	}
	return item
}

func decodeJSON(value any, target any) {
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

func timePointer(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func nullableText(value string) pgtype.Text { return pgtype.Text{String: value, Valid: value != ""} }

func timeValue(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}
