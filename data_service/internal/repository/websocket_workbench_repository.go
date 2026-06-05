package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// WebSocketSessionGroupRecord 表示 WebSocket 工作台左侧会话分组。
type WebSocketSessionGroupRecord struct {
	ID           string
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	SortOrder    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// WebSocketSessionRecord 表示一个开发态 WebSocket 会话模板。
type WebSocketSessionRecord struct {
	ID             string
	ProjectID      string
	ConnectionID   string
	GroupID        *string
	Name           string
	URL            string
	Headers        []any
	Auth           map[string]any
	Protocols      []any
	Messages       []any
	Settings       map[string]any
	Enabled        bool
	SortOrder      int
	LastMessage    any
	LastDiagnostic *string
	Quality        string
	LastMessageAt  *time.Time
	DataPointID    *string
	DataPointPath  *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateWebSocketSessionGroupParams struct {
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	SortOrder    int
	UserID       string
}

type UpdateWebSocketSessionGroupParams struct {
	ProjectID string
	GroupID   string
	ParentID  *string
	Name      string
	SortOrder int
	UserID    string
}

type CreateWebSocketSessionParams struct {
	ProjectID       string
	ConnectionID    string
	GroupID         *string
	Name            string
	URL             string
	Headers         []any
	Auth            map[string]any
	Protocols       []any
	Messages        []any
	Settings        map[string]any
	Enabled         bool
	SortOrder       int
	DataPointPath   string
	DataPointConfig map[string]any
	DefaultValue    *string
	UserID          string
}

type UpdateWebSocketSessionParams struct {
	ID              string
	ProjectID       string
	GroupID         *string
	Name            string
	URL             string
	Headers         []any
	Auth            map[string]any
	Protocols       []any
	Messages        []any
	Settings        map[string]any
	Enabled         bool
	SortOrder       int
	DataPointPath   string
	DataPointConfig map[string]any
	DefaultValue    *string
	UserID          string
}

// WebSocketSessionPreviewSnapshot 描述短连接预览后要持久化的最后消息。
type WebSocketSessionPreviewSnapshot struct {
	SessionID       string
	LastMessage     any
	LastDiagnostic  *string
	Quality         string
	LastMessageAt   time.Time
	DefaultValue    *string
	DataPointConfig map[string]any
	UserID          string
}

// WebSocketWorkbenchRepository 封装 WebSocket 工作台参数化 SQL。
type WebSocketWorkbenchRepository struct {
	pool *pgxpool.Pool
}

func NewWebSocketWorkbenchRepository(pool *pgxpool.Pool) *WebSocketWorkbenchRepository {
	return &WebSocketWorkbenchRepository{pool: pool}
}

func (r *WebSocketWorkbenchRepository) ListGroups(ctx context.Context, projectID, connectionID string) ([]WebSocketSessionGroupRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, connection_id, parent_id, name, sort_order, created_at, updated_at
		FROM data_websocket_session_groups
		WHERE project_id = $1 AND connection_id = $2
		ORDER BY sort_order ASC, updated_at DESC
	`, projectID, connectionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 WebSocket 会话分组失败", err)
	}
	defer rows.Close()

	result := make([]WebSocketSessionGroupRecord, 0)
	for rows.Next() {
		record, scanErr := scanWebSocketSessionGroupRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 WebSocket 会话分组失败", err)
	}
	return result, nil
}

func (r *WebSocketWorkbenchRepository) GetGroup(ctx context.Context, projectID, groupID string) (*WebSocketSessionGroupRecord, error) {
	record, err := scanWebSocketSessionGroupRecord(r.pool.QueryRow(ctx, `
		SELECT id, project_id, connection_id, parent_id, name, sort_order, created_at, updated_at
		FROM data_websocket_session_groups
		WHERE project_id = $1 AND id = $2
	`, projectID, groupID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "WebSocket 会话分组不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 WebSocket 会话分组失败", err)
	}
	return &record, nil
}

func (r *WebSocketWorkbenchRepository) CreateGroup(ctx context.Context, params CreateWebSocketSessionGroupParams) (*WebSocketSessionGroupRecord, error) {
	record, err := scanWebSocketSessionGroupRecord(r.pool.QueryRow(ctx, `
		INSERT INTO data_websocket_session_groups (
			project_id, connection_id, parent_id, name, sort_order, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		RETURNING id, project_id, connection_id, parent_id, name, sort_order, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.ParentID, params.Name, params.SortOrder, params.UserID))
	if err != nil {
		return nil, translateWebSocketWorkbenchWriteError(err, "创建 WebSocket 会话分组失败")
	}
	return &record, nil
}

func (r *WebSocketWorkbenchRepository) UpdateGroup(ctx context.Context, params UpdateWebSocketSessionGroupParams) (*WebSocketSessionGroupRecord, error) {
	record, err := scanWebSocketSessionGroupRecord(r.pool.QueryRow(ctx, `
		UPDATE data_websocket_session_groups
		SET parent_id = $3,
		    name = $4,
		    sort_order = $5,
		    updated_by = $6,
		    updated_at = now()
		WHERE project_id = $1 AND id = $2
		RETURNING id, project_id, connection_id, parent_id, name, sort_order, created_at, updated_at
	`, params.ProjectID, params.GroupID, params.ParentID, params.Name, params.SortOrder, params.UserID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "WebSocket 会话分组不存在")
		}
		return nil, translateWebSocketWorkbenchWriteError(err, "更新 WebSocket 会话分组失败")
	}
	return &record, nil
}

func (r *WebSocketWorkbenchRepository) DeleteGroup(ctx context.Context, projectID, groupID, userID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 WebSocket 会话分组删除事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	if _, err := tx.Exec(ctx, `
		UPDATE data_websocket_sessions
		SET group_id = NULL, updated_by = $3, updated_at = now()
		WHERE project_id = $1 AND group_id = $2
	`, projectID, groupID, userID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "移动 WebSocket 会话到根集合失败", err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM data_websocket_session_groups WHERE project_id = $1 AND id = $2`, projectID, groupID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 WebSocket 会话分组失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "WebSocket 会话分组不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 WebSocket 会话分组删除事务失败", err)
	}
	return nil
}

func (r *WebSocketWorkbenchRepository) ListSessionsPage(ctx context.Context, projectID, connectionID string, groupID *string, search string, page, pageSize int) ([]WebSocketSessionRecord, int, error) {
	page, pageSize = normalizePageAndSize(page, pageSize, 20, 100)
	where := []string{"s.project_id = $1", "s.connection_id = $2"}
	args := []any{projectID, connectionID}
	if groupID != nil {
		normalizedGroupID := strings.TrimSpace(*groupID)
		if normalizedGroupID == "__ungrouped" {
			where = append(where, "s.group_id IS NULL")
		} else if normalizedGroupID != "" {
			args = append(args, normalizedGroupID)
			where = append(where, fmt.Sprintf("s.group_id = $%d", len(args)))
		}
	}
	if keyword := strings.TrimSpace(search); keyword != "" {
		args = append(args, "%"+keyword+"%")
		where = append(where, fmt.Sprintf("(s.name ILIKE $%d OR s.url ILIKE $%d)", len(args), len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_websocket_sessions s WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计 WebSocket 会话失败", err)
	}

	queryArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.project_id, s.connection_id, s.group_id, s.name, s.url,
		       s.headers, s.auth, s.protocols, s.messages, s.settings, s.enabled,
		       s.sort_order, s.last_message, s.last_diagnostic, s.quality, s.last_message_at,
		       dp.id, dp.path, s.created_at, s.updated_at
		FROM data_websocket_sessions s
		LEFT JOIN data_points dp
		  ON dp.project_id = s.project_id
		 AND dp.source_type = 'websocket.session'
		 AND dp.source_config->>'sessionId' = s.id::text
		WHERE `+whereSQL+`
		ORDER BY s.sort_order ASC, s.updated_at DESC
		LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2), queryArgs...)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 WebSocket 会话失败", err)
	}
	defer rows.Close()

	result := make([]WebSocketSessionRecord, 0)
	for rows.Next() {
		record, scanErr := scanWebSocketSessionRecord(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 WebSocket 会话失败", err)
	}
	return result, total, nil
}

func (r *WebSocketWorkbenchRepository) GetSession(ctx context.Context, projectID, sessionID string) (*WebSocketSessionRecord, error) {
	record, err := scanWebSocketSessionRecord(r.pool.QueryRow(ctx, `
		SELECT s.id, s.project_id, s.connection_id, s.group_id, s.name, s.url,
		       s.headers, s.auth, s.protocols, s.messages, s.settings, s.enabled,
		       s.sort_order, s.last_message, s.last_diagnostic, s.quality, s.last_message_at,
		       dp.id, dp.path, s.created_at, s.updated_at
		FROM data_websocket_sessions s
		LEFT JOIN data_points dp
		  ON dp.project_id = s.project_id
		 AND dp.source_type = 'websocket.session'
		 AND dp.source_config->>'sessionId' = s.id::text
		WHERE s.project_id = $1 AND s.id = $2
	`, projectID, sessionID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "WebSocket 会话不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 WebSocket 会话失败", err)
	}
	return &record, nil
}

func (r *WebSocketWorkbenchRepository) CreateSessionWithDataPoint(ctx context.Context, params CreateWebSocketSessionParams) (*WebSocketSessionRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 WebSocket 会话创建事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	record := WebSocketSessionRecord{}
	err = tx.QueryRow(ctx, `
		INSERT INTO data_websocket_sessions (
			project_id, connection_id, group_id, name, url, headers, auth, protocols,
			messages, settings, enabled, sort_order, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, $8::jsonb, $9::jsonb, $10::jsonb, $11, $12, $13, $13)
		RETURNING id, project_id, connection_id, group_id, name, url, headers, auth,
		          protocols, messages, settings, enabled, sort_order, last_message,
		          last_diagnostic, quality, last_message_at, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.GroupID, params.Name, params.URL, mustMarshalJSONArray(params.Headers), mustMarshalJSONObject(params.Auth), mustMarshalJSONArray(params.Protocols), mustMarshalJSONArray(params.Messages), mustMarshalJSONObject(params.Settings), params.Enabled, params.SortOrder, params.UserID).Scan(
		&record.ID, &record.ProjectID, &record.ConnectionID, &record.GroupID, &record.Name, &record.URL,
		newJSONScanner(&record.Headers), newJSONScanner(&record.Auth), newJSONScanner(&record.Protocols),
		newJSONScanner(&record.Messages), newJSONScanner(&record.Settings), &record.Enabled, &record.SortOrder,
		newJSONScanner(&record.LastMessage), &record.LastDiagnostic, &record.Quality, &record.LastMessageAt,
		&record.CreatedAt, &record.UpdatedAt,
	)
	if err != nil {
		return nil, translateWebSocketWorkbenchWriteError(err, "创建 WebSocket 会话失败")
	}

	dataPointID, dataPointPath, err := upsertWebSocketSessionDataPoint(ctx, tx, record, params.DataPointPath, params.DataPointConfig, params.DefaultValue, params.UserID)
	if err != nil {
		return nil, err
	}
	record.DataPointID = &dataPointID
	record.DataPointPath = &dataPointPath

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 WebSocket 会话创建事务失败", err)
	}
	return &record, nil
}

func (r *WebSocketWorkbenchRepository) UpdateSessionWithDataPoint(ctx context.Context, params UpdateWebSocketSessionParams) (*WebSocketSessionRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 WebSocket 会话更新事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	record := WebSocketSessionRecord{}
	err = tx.QueryRow(ctx, `
		UPDATE data_websocket_sessions
		SET group_id = $3,
		    name = $4,
		    url = $5,
		    headers = $6::jsonb,
		    auth = $7::jsonb,
		    protocols = $8::jsonb,
		    messages = $9::jsonb,
		    settings = $10::jsonb,
		    enabled = $11,
		    sort_order = $12,
		    updated_by = $13,
		    updated_at = now()
		WHERE project_id = $1 AND id = $2
		RETURNING id, project_id, connection_id, group_id, name, url, headers, auth,
		          protocols, messages, settings, enabled, sort_order, last_message,
		          last_diagnostic, quality, last_message_at, created_at, updated_at
	`, params.ProjectID, params.ID, params.GroupID, params.Name, params.URL, mustMarshalJSONArray(params.Headers), mustMarshalJSONObject(params.Auth), mustMarshalJSONArray(params.Protocols), mustMarshalJSONArray(params.Messages), mustMarshalJSONObject(params.Settings), params.Enabled, params.SortOrder, params.UserID).Scan(
		&record.ID, &record.ProjectID, &record.ConnectionID, &record.GroupID, &record.Name, &record.URL,
		newJSONScanner(&record.Headers), newJSONScanner(&record.Auth), newJSONScanner(&record.Protocols),
		newJSONScanner(&record.Messages), newJSONScanner(&record.Settings), &record.Enabled, &record.SortOrder,
		newJSONScanner(&record.LastMessage), &record.LastDiagnostic, &record.Quality, &record.LastMessageAt,
		&record.CreatedAt, &record.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "WebSocket 会话不存在")
		}
		return nil, translateWebSocketWorkbenchWriteError(err, "更新 WebSocket 会话失败")
	}

	dataPointID, dataPointPath, err := upsertWebSocketSessionDataPoint(ctx, tx, record, params.DataPointPath, params.DataPointConfig, params.DefaultValue, params.UserID)
	if err != nil {
		return nil, err
	}
	record.DataPointID = &dataPointID
	record.DataPointPath = &dataPointPath

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 WebSocket 会话更新事务失败", err)
	}
	return &record, nil
}

func (r *WebSocketWorkbenchRepository) DeleteSessionWithDataPoint(ctx context.Context, projectID, sessionID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 WebSocket 会话删除事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	if _, err := tx.Exec(ctx, `
		UPDATE data_points
		SET status = 'invalid', updated_at = now()
		WHERE project_id = $1
		  AND source_type = 'websocket.session'
		  AND source_config->>'sessionId' = $2
	`, projectID, sessionID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "标记 WebSocket 会话数据点失效失败", err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM data_websocket_sessions WHERE project_id = $1 AND id = $2`, projectID, sessionID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 WebSocket 会话失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "WebSocket 会话不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 WebSocket 会话删除事务失败", err)
	}
	return nil
}

func (r *WebSocketWorkbenchRepository) SavePreviewSnapshot(ctx context.Context, projectID string, snapshot WebSocketSessionPreviewSnapshot) error {
	messagePayload, err := json.Marshal(snapshot.LastMessage)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket 消息快照格式无效", err)
	}
	configPayload, err := json.Marshal(snapshot.DataPointConfig)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket 数据点 sourceConfig 格式无效", err)
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 WebSocket 消息写回事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	tag, err := tx.Exec(ctx, `
		UPDATE data_websocket_sessions
		SET last_message = $3::jsonb,
		    last_diagnostic = $4,
		    quality = $5,
		    last_message_at = $6,
		    updated_by = $7,
		    updated_at = now()
		WHERE project_id = $1 AND id = $2
	`, projectID, snapshot.SessionID, string(messagePayload), snapshot.LastDiagnostic, snapshot.Quality, snapshot.LastMessageAt, snapshot.UserID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "保存 WebSocket 消息快照失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "WebSocket 会话不存在")
	}

	if snapshot.DefaultValue != nil {
		_, err = tx.Exec(ctx, `
			UPDATE data_points
			SET default_value = $3,
			    source_config = $4::jsonb,
			    updated_by = $5,
			    updated_at = now()
			WHERE project_id = $1
			  AND source_type = 'websocket.session'
			  AND source_config->>'sessionId' = $2
		`, projectID, snapshot.SessionID, snapshot.DefaultValue, string(configPayload), snapshot.UserID)
		if err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写回 WebSocket 会话数据点失败", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 WebSocket 消息写回事务失败", err)
	}
	return nil
}

func upsertWebSocketSessionDataPoint(ctx context.Context, tx pgx.Tx, record WebSocketSessionRecord, path string, sourceConfig map[string]any, defaultValue *string, userID string) (string, string, error) {
	config := cloneProtocolMap(sourceConfig)
	config["sessionId"] = record.ID
	configPayload, err := json.Marshal(config)
	if err != nil {
		return "", "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket 会话数据点 sourceConfig 格式无效", err)
	}
	status := "active"
	if !record.Enabled {
		status = "inactive"
	}
	allocatedPath, err := allocateGeneratedDataPointPath(ctx, tx, record.ProjectID, path, "websocket.session", record.ID)
	if err != nil {
		return "", "", err
	}

	var dataPointID, dataPointPath string
	err = tx.QueryRow(ctx, `
		UPDATE data_points
		SET path = $2,
		    name = $3,
		    source_id = $4,
		    source_config = $5::jsonb,
		    data_type = 'object',
		    default_value = COALESCE($6, default_value),
		    refresh_mode = 'manual',
		    status = $7,
		    updated_by = $8,
		    updated_at = now()
		WHERE project_id = $1
		  AND source_type = 'websocket.session'
		  AND source_config->>'sessionId' = $9
		RETURNING id, path
	`, record.ProjectID, path, record.Name, record.ConnectionID, string(configPayload), defaultValue, status, userID, record.ID).Scan(&dataPointID, &dataPointPath)
	if err == nil {
		return dataPointID, dataPointPath, nil
	}
	if err != pgx.ErrNoRows {
		return "", "", apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "同步 WebSocket 会话数据点失败", err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO data_points (
			project_id, path, name, source_type, source_id, source_config,
			data_type, default_value, refresh_mode, status, display_order, created_by, updated_by
		)
		VALUES (
			$1, $2, $3, 'websocket.session', $4, $5::jsonb, 'object', $6, 'manual', $7,
			COALESCE((SELECT MAX(display_order) + 1 FROM data_points WHERE project_id = $1), 0),
			$8, $8
		)
		RETURNING id, path
	`, record.ProjectID, allocatedPath, record.Name, record.ConnectionID, string(configPayload), defaultValue, status, userID).Scan(&dataPointID, &dataPointPath)
	if err != nil {
		return "", "", apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "同步 WebSocket 会话数据点失败", err)
	}
	return dataPointID, dataPointPath, nil
}

func scanWebSocketSessionGroupRecord(row pgx.Row) (WebSocketSessionGroupRecord, error) {
	record := WebSocketSessionGroupRecord{}
	err := row.Scan(&record.ID, &record.ProjectID, &record.ConnectionID, &record.ParentID, &record.Name, &record.SortOrder, &record.CreatedAt, &record.UpdatedAt)
	return record, err
}

func scanWebSocketSessionRecord(row pgx.Row) (WebSocketSessionRecord, error) {
	record := WebSocketSessionRecord{}
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.GroupID,
		&record.Name,
		&record.URL,
		newJSONScanner(&record.Headers),
		newJSONScanner(&record.Auth),
		newJSONScanner(&record.Protocols),
		newJSONScanner(&record.Messages),
		newJSONScanner(&record.Settings),
		&record.Enabled,
		&record.SortOrder,
		newJSONScanner(&record.LastMessage),
		&record.LastDiagnostic,
		&record.Quality,
		&record.LastMessageAt,
		&record.DataPointID,
		&record.DataPointPath,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return record, err
	}
	normalizeWebSocketSessionJSONDefaults(&record)
	return record, nil
}

func normalizeWebSocketSessionJSONDefaults(record *WebSocketSessionRecord) {
	if record.Headers == nil {
		record.Headers = []any{}
	}
	if record.Auth == nil {
		record.Auth = map[string]any{}
	}
	if record.Protocols == nil {
		record.Protocols = []any{}
	}
	if record.Messages == nil {
		record.Messages = []any{}
	}
	if record.Settings == nil {
		record.Settings = map[string]any{}
	}
}

func translateWebSocketWorkbenchWriteError(err error, fallback string) error {
	var pgErr *pgconn.PgError
	if err != nil && strings.Contains(err.Error(), "JSON 字段类型无效") {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, fallback, err)
	}
	if strings.TrimSpace(fallback) == "" {
		fallback = "WebSocket 工作台写入失败"
	}
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "同名 WebSocket 会话或分组已存在")
		case "23503":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "关联的 WebSocket 接入源或分组不存在")
		}
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, fallback, err)
}
