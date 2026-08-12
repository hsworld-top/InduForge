package auditlog

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

type PostgreSQLRepository struct{ queries *dbsqlc.Queries }

func NewPostgreSQLRepository(pool *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{queries: dbsqlc.New(pool)}
}

func (r *PostgreSQLRepository) Create(ctx context.Context, item Log) error {
	tenantID, err := parseUUID(item.TenantID)
	if err != nil {
		return err
	}
	userID, err := parseUUID(item.UserID)
	if err != nil {
		return err
	}
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return err
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	return r.queries.CreateAuditLog(ctx, dbsqlc.CreateAuditLogParams{
		ID: uuidValue(uuid.New()), TenantID: tenantID, UserID: userID, Level: item.Level,
		Action: textValue(item.Action), Resource: textValue(item.Resource), ResourceID: textValue(item.ResourceID),
		Message: item.Message, RequestID: textValue(item.RequestID), Method: textValue(item.Method), Path: textValue(item.Path),
		Result: textValue(item.Result), Ip: textValue(item.IP), UserAgent: textValue(item.UserAgent), Metadata: metadata,
		CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
	})
}

func (r *PostgreSQLRepository) List(ctx context.Context, tenantID string, filter Filter) ([]Log, int64, error) {
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		return nil, 0, ErrNotFound
	}
	params := listParams(tenantUUID, filter)
	rows, err := r.queries.ListAuditLogs(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := r.queries.CountAuditLogs(ctx, dbsqlc.CountAuditLogsParams{
		TenantID: params.TenantID, Level: params.Level, Action: params.Action, Resource: params.Resource,
		UserID: params.UserID, Keyword: params.Keyword, StartTime: params.StartTime, EndTime: params.EndTime,
	})
	if err != nil {
		return nil, 0, err
	}
	items := make([]Log, 0, len(rows))
	for _, row := range rows {
		items = append(items, logFromFields(row.ID, row.TenantID, row.UserID, row.Level, row.Action, row.Resource, row.ResourceID, row.Message, row.RequestID, row.Method, row.Path, row.Result, row.Ip, row.UserAgent, row.Metadata, row.CreatedAt, row.Username, row.FullName))
	}
	return items, total, nil
}

func (r *PostgreSQLRepository) Get(ctx context.Context, tenantID, id string) (Log, error) {
	tenantUUID, logUUID, err := parsePair(tenantID, id)
	if err != nil {
		return Log{}, ErrNotFound
	}
	row, err := r.queries.GetAuditLog(ctx, dbsqlc.GetAuditLogParams{LogID: logUUID, TenantID: tenantUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Log{}, ErrNotFound
		}
		return Log{}, err
	}
	return logFromFields(row.ID, row.TenantID, row.UserID, row.Level, row.Action, row.Resource, row.ResourceID, row.Message, row.RequestID, row.Method, row.Path, row.Result, row.Ip, row.UserAgent, row.Metadata, row.CreatedAt, row.Username, row.FullName), nil
}

func (r *PostgreSQLRepository) DeleteBefore(ctx context.Context, tenantID string, before time.Time) (int64, error) {
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		return 0, ErrNotFound
	}
	return r.queries.DeleteAuditLogsBefore(ctx, dbsqlc.DeleteAuditLogsBeforeParams{TenantID: tenantUUID, BeforeTime: timeValue(&before)})
}

func (r *PostgreSQLRepository) Export(ctx context.Context, tenantID string, filter Filter) ([]Log, error) {
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		return nil, ErrNotFound
	}
	params := listParams(tenantUUID, filter)
	rows, err := r.queries.ExportAuditLogs(ctx, dbsqlc.ExportAuditLogsParams{
		TenantID: params.TenantID, Level: params.Level, Action: params.Action, Resource: params.Resource,
		UserID: params.UserID, Keyword: params.Keyword, StartTime: params.StartTime, EndTime: params.EndTime,
	})
	if err != nil {
		return nil, err
	}
	items := make([]Log, 0, len(rows))
	for _, row := range rows {
		items = append(items, logFromFields(row.ID, row.TenantID, row.UserID, row.Level, row.Action, row.Resource, row.ResourceID, row.Message, row.RequestID, row.Method, row.Path, row.Result, row.Ip, row.UserAgent, row.Metadata, row.CreatedAt, row.Username, row.FullName))
	}
	return items, nil
}

func (r *PostgreSQLRepository) Stats(ctx context.Context, tenantID string, start, end *time.Time) (Stats, error) {
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		return Stats{}, ErrNotFound
	}
	startValue, endValue := timeValue(start), timeValue(end)
	levels, err := r.queries.GetAuditLogLevelStats(ctx, dbsqlc.GetAuditLogLevelStatsParams{TenantID: tenantUUID, StartTime: startValue, EndTime: endValue})
	if err != nil {
		return Stats{}, err
	}
	trends, err := r.queries.GetAuditLogTrendStats(ctx, dbsqlc.GetAuditLogTrendStatsParams{TenantID: tenantUUID, StartTime: startValue, EndTime: endValue})
	if err != nil {
		return Stats{}, err
	}
	actions, err := r.queries.GetAuditLogActionStats(ctx, dbsqlc.GetAuditLogActionStatsParams{TenantID: tenantUUID, StartTime: startValue, EndTime: endValue})
	if err != nil {
		return Stats{}, err
	}
	result := Stats{LevelStats: map[string]int64{}, TrendStats: make([]map[string]any, 0, len(trends)), ActionStats: make([]map[string]any, 0, len(actions))}
	for _, item := range levels {
		result.LevelStats[item.Level] = item.Count
	}
	for _, item := range trends {
		result.TrendStats = append(result.TrendStats, map[string]any{"date": item.Date, "count": item.Count})
	}
	for _, item := range actions {
		result.ActionStats = append(result.ActionStats, map[string]any{"action": item.Action, "count": item.Count})
	}
	return result, nil
}

func (r *PostgreSQLRepository) Recent(ctx context.Context, tenantID string, limit int) ([]Log, error) {
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		return nil, ErrNotFound
	}
	rows, err := r.queries.ListRecentAuditActivities(ctx, dbsqlc.ListRecentAuditActivitiesParams{TenantID: tenantUUID, ItemLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	items := make([]Log, 0, len(rows))
	for _, row := range rows {
		items = append(items, Log{ID: uuidString(row.ID), UserID: uuidString(row.UserID), Username: textString(row.Username), FullName: textString(row.FullName), Message: row.Message, Action: textString(row.Action), CreatedAt: row.CreatedAt.Time})
	}
	return items, nil
}

func listParams(tenantID pgtype.UUID, filter Filter) dbsqlc.ListAuditLogsParams {
	return dbsqlc.ListAuditLogsParams{
		TenantID: tenantID, Level: filter.Level, Action: filter.Action, Resource: filter.Resource, UserID: filter.UserID,
		Keyword: filter.Keyword, StartTime: timeValue(filter.StartTime), EndTime: timeValue(filter.EndTime),
		PageOffset: int32((filter.Page - 1) * filter.Limit), PageLimit: int32(filter.Limit),
	}
}

func logFromFields(id, tenantID, userID pgtype.UUID, level string, action, resource, resourceID pgtype.Text, message string, requestID, method, requestPath, result, ip, userAgent pgtype.Text, metadata []byte, createdAt pgtype.Timestamptz, username, fullName pgtype.Text) Log {
	item := Log{
		ID: uuidString(id), TenantID: uuidString(tenantID), UserID: uuidString(userID), Username: textString(username), FullName: textString(fullName),
		Level: level, Action: textString(action), Resource: textString(resource), ResourceID: textString(resourceID), Message: message,
		RequestID: textString(requestID), Method: textString(method), Path: textString(requestPath), Result: textString(result),
		IP: textString(ip), UserAgent: textString(userAgent), CreatedAt: createdAt.Time,
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	return item
}

func timeValue(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
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

func textValue(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func uuidValue(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}
