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

// HTTPRequestGroupRecord 表示 HTTP 工作台左侧集合树分组。
type HTTPRequestGroupRecord struct {
	ID           string
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	SortOrder    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// HTTPRequestRecord 表示一个可保存、可发送的 HTTP 接口请求。
type HTTPRequestRecord struct {
	ID            string
	ProjectID     string
	ConnectionID  string
	GroupID       *string
	Name          string
	Method        string
	URL           string
	Params        []any
	Headers       []any
	Auth          map[string]any
	BodyType      string
	Body          map[string]any
	Settings      map[string]any
	Enabled       bool
	SortOrder     int
	LastResponse  any
	Quality       string
	LastSentAt    *time.Time
	DataPointID   *string
	DataPointPath *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CreateHTTPRequestGroupParams 描述 HTTP 请求分组创建参数。
type CreateHTTPRequestGroupParams struct {
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	SortOrder    int
	UserID       string
}

// UpdateHTTPRequestGroupParams 描述 HTTP 请求分组更新参数。
type UpdateHTTPRequestGroupParams struct {
	ProjectID string
	GroupID   string
	ParentID  *string
	Name      string
	SortOrder int
	UserID    string
}

// CreateHTTPRequestParams 描述 HTTP 请求创建参数。
type CreateHTTPRequestParams struct {
	ProjectID       string
	ConnectionID    string
	GroupID         *string
	Name            string
	Method          string
	URL             string
	Params          []any
	Headers         []any
	Auth            map[string]any
	BodyType        string
	Body            map[string]any
	Settings        map[string]any
	Enabled         bool
	SortOrder       int
	DataPointPath   string
	DataPointConfig map[string]any
	DefaultValue    *string
	UserID          string
}

// UpdateHTTPRequestParams 描述 HTTP 请求更新参数。
type UpdateHTTPRequestParams struct {
	ID              string
	ProjectID       string
	GroupID         *string
	Name            string
	Method          string
	URL             string
	Params          []any
	Headers         []any
	Auth            map[string]any
	BodyType        string
	Body            map[string]any
	Settings        map[string]any
	Enabled         bool
	SortOrder       int
	DataPointPath   string
	DataPointConfig map[string]any
	DefaultValue    *string
	UserID          string
}

// HTTPRequestSendSnapshot 描述一次发送后要保存的最后响应快照。
type HTTPRequestSendSnapshot struct {
	RequestID       string
	LastResponse    any
	Quality         string
	LastSentAt      time.Time
	DefaultValue    *string
	DataPointConfig map[string]any
	UserID          string
}

// HTTPWorkbenchRepository 封装 HTTP 工作台参数化 SQL。
type HTTPWorkbenchRepository struct {
	pool *pgxpool.Pool
}

// NewHTTPWorkbenchRepository 创建 HTTP 工作台仓储。
func NewHTTPWorkbenchRepository(pool *pgxpool.Pool) *HTTPWorkbenchRepository {
	return &HTTPWorkbenchRepository{pool: pool}
}

// ListGroups 返回当前 HTTP 接入源下的请求分组。
func (r *HTTPWorkbenchRepository) ListGroups(ctx context.Context, projectID, connectionID string) ([]HTTPRequestGroupRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, connection_id, parent_id, name, sort_order, created_at, updated_at
		FROM data_http_request_groups
		WHERE project_id = $1 AND connection_id = $2
		ORDER BY sort_order ASC, updated_at DESC
	`, projectID, connectionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 HTTP 请求分组失败", err)
	}
	defer rows.Close()

	result := make([]HTTPRequestGroupRecord, 0)
	for rows.Next() {
		record, scanErr := scanHTTPRequestGroupRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 HTTP 请求分组失败", err)
	}
	return result, nil
}

// GetGroup 返回单个 HTTP 请求分组。
func (r *HTTPWorkbenchRepository) GetGroup(ctx context.Context, projectID, groupID string) (*HTTPRequestGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, connection_id, parent_id, name, sort_order, created_at, updated_at
		FROM data_http_request_groups
		WHERE project_id = $1 AND id = $2
	`, projectID, groupID)
	record, err := scanHTTPRequestGroupRecord(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "HTTP 请求分组不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 HTTP 请求分组失败", err)
	}
	return &record, nil
}

// CreateGroup 创建 HTTP 请求分组。
func (r *HTTPWorkbenchRepository) CreateGroup(ctx context.Context, params CreateHTTPRequestGroupParams) (*HTTPRequestGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO data_http_request_groups (
			project_id, connection_id, parent_id, name, sort_order, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		RETURNING id, project_id, connection_id, parent_id, name, sort_order, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.ParentID, params.Name, params.SortOrder, params.UserID)
	record, err := scanHTTPRequestGroupRecord(row)
	if err != nil {
		return nil, translateHTTPWorkbenchWriteError(err, "创建 HTTP 请求分组失败")
	}
	return &record, nil
}

// UpdateGroup 更新 HTTP 请求分组。
func (r *HTTPWorkbenchRepository) UpdateGroup(ctx context.Context, params UpdateHTTPRequestGroupParams) (*HTTPRequestGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE data_http_request_groups
		SET parent_id = $3,
		    name = $4,
		    sort_order = $5,
		    updated_by = $6,
		    updated_at = now()
		WHERE project_id = $1 AND id = $2
		RETURNING id, project_id, connection_id, parent_id, name, sort_order, created_at, updated_at
	`, params.ProjectID, params.GroupID, params.ParentID, params.Name, params.SortOrder, params.UserID)
	record, err := scanHTTPRequestGroupRecord(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "HTTP 请求分组不存在")
		}
		return nil, translateHTTPWorkbenchWriteError(err, "更新 HTTP 请求分组失败")
	}
	return &record, nil
}

// DeleteGroup 删除 HTTP 请求分组，并把组内接口移动到根集合。
func (r *HTTPWorkbenchRepository) DeleteGroup(ctx context.Context, projectID, groupID, userID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 HTTP 请求分组删除事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	if _, err := tx.Exec(ctx, `
		UPDATE data_http_requests
		SET group_id = NULL, updated_by = $3, updated_at = now()
		WHERE project_id = $1 AND group_id = $2
	`, projectID, groupID, userID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "移动 HTTP 请求到根集合失败", err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM data_http_request_groups WHERE project_id = $1 AND id = $2`, projectID, groupID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 HTTP 请求分组失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "HTTP 请求分组不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 HTTP 请求分组删除事务失败", err)
	}
	return nil
}

// ListRequestsPage 返回当前 HTTP 接入源下接口请求的一页数据。
func (r *HTTPWorkbenchRepository) ListRequestsPage(ctx context.Context, projectID, connectionID string, groupID *string, search string, page, pageSize int) ([]HTTPRequestRecord, int, error) {
	page, pageSize = normalizePageAndSize(page, pageSize, 20, 100)
	where := []string{"req.project_id = $1", "req.connection_id = $2"}
	args := []any{projectID, connectionID}
	if groupID != nil {
		normalizedGroupID := strings.TrimSpace(*groupID)
		if normalizedGroupID == "__ungrouped" {
			where = append(where, "req.group_id IS NULL")
		} else if normalizedGroupID != "" {
			args = append(args, normalizedGroupID)
			where = append(where, fmt.Sprintf("req.group_id = $%d", len(args)))
		}
	}
	if keyword := strings.TrimSpace(search); keyword != "" {
		args = append(args, "%"+keyword+"%")
		where = append(where, fmt.Sprintf("(req.name ILIKE $%d OR req.url ILIKE $%d OR req.method ILIKE $%d)", len(args), len(args), len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_http_requests req WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计 HTTP 请求失败", err)
	}

	queryArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.pool.Query(ctx, `
		SELECT req.id, req.project_id, req.connection_id, req.group_id, req.name, req.method, req.url,
		       req.params, req.headers, req.auth, req.body_type, req.body, req.settings, req.enabled,
		       req.sort_order, req.last_response, req.quality, req.last_sent_at, dp.id, dp.path,
		       req.created_at, req.updated_at
		FROM data_http_requests req
		LEFT JOIN data_points dp
		  ON dp.project_id = req.project_id
		 AND dp.source_type = 'http.request'
		 AND dp.source_config->>'requestId' = req.id::text
		WHERE `+whereSQL+`
		ORDER BY req.sort_order ASC, req.updated_at DESC
		LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2), queryArgs...)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 HTTP 请求失败", err)
	}
	defer rows.Close()

	result := make([]HTTPRequestRecord, 0)
	for rows.Next() {
		record, scanErr := scanHTTPRequestRecord(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 HTTP 请求失败", err)
	}
	return result, total, nil
}

// GetRequest 返回单个 HTTP 请求。
func (r *HTTPWorkbenchRepository) GetRequest(ctx context.Context, projectID, requestID string) (*HTTPRequestRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT req.id, req.project_id, req.connection_id, req.group_id, req.name, req.method, req.url,
		       req.params, req.headers, req.auth, req.body_type, req.body, req.settings, req.enabled,
		       req.sort_order, req.last_response, req.quality, req.last_sent_at, dp.id, dp.path,
		       req.created_at, req.updated_at
		FROM data_http_requests req
		LEFT JOIN data_points dp
		  ON dp.project_id = req.project_id
		 AND dp.source_type = 'http.request'
		 AND dp.source_config->>'requestId' = req.id::text
		WHERE req.project_id = $1 AND req.id = $2
	`, projectID, requestID)
	record, err := scanHTTPRequestRecord(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "HTTP 请求不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 HTTP 请求失败", err)
	}
	return &record, nil
}

// CreateRequestWithDataPoint 创建 HTTP 请求并同步 http.request 数据点。
func (r *HTTPWorkbenchRepository) CreateRequestWithDataPoint(ctx context.Context, params CreateHTTPRequestParams) (*HTTPRequestRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 HTTP 请求创建事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	record := HTTPRequestRecord{}
	err = tx.QueryRow(ctx, `
		INSERT INTO data_http_requests (
			project_id, connection_id, group_id, name, method, url, params, headers,
			auth, body_type, body, settings, enabled, sort_order, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9::jsonb, $10, $11::jsonb, $12::jsonb, $13, $14, $15, $15)
		RETURNING id, project_id, connection_id, group_id, name, method, url, params, headers,
		          auth, body_type, body, settings, enabled, sort_order, last_response,
		          quality, last_sent_at, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.GroupID, params.Name, params.Method, params.URL, mustMarshalJSONArray(params.Params), mustMarshalJSONArray(params.Headers), mustMarshalJSONObject(params.Auth), params.BodyType, mustMarshalJSONObject(params.Body), mustMarshalJSONObject(params.Settings), params.Enabled, params.SortOrder, params.UserID).Scan(
		&record.ID, &record.ProjectID, &record.ConnectionID, &record.GroupID, &record.Name, &record.Method, &record.URL,
		newJSONScanner(&record.Params), newJSONScanner(&record.Headers), newJSONScanner(&record.Auth), &record.BodyType,
		newJSONScanner(&record.Body), newJSONScanner(&record.Settings), &record.Enabled, &record.SortOrder,
		newJSONScanner(&record.LastResponse), &record.Quality, &record.LastSentAt, &record.CreatedAt, &record.UpdatedAt,
	)
	if err != nil {
		return nil, translateHTTPWorkbenchWriteError(err, "创建 HTTP 请求失败")
	}

	dataPointID, dataPointPath, err := upsertHTTPRequestDataPoint(ctx, tx, record, params.DataPointPath, params.DataPointConfig, params.DefaultValue, params.UserID)
	if err != nil {
		return nil, err
	}
	record.DataPointID = &dataPointID
	record.DataPointPath = &dataPointPath

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 HTTP 请求创建事务失败", err)
	}
	return &record, nil
}

// UpdateRequestWithDataPoint 更新 HTTP 请求并同步 http.request 数据点。
func (r *HTTPWorkbenchRepository) UpdateRequestWithDataPoint(ctx context.Context, params UpdateHTTPRequestParams) (*HTTPRequestRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 HTTP 请求更新事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	record := HTTPRequestRecord{}
	err = tx.QueryRow(ctx, `
		UPDATE data_http_requests
		SET group_id = $3,
		    name = $4,
		    method = $5,
		    url = $6,
		    params = $7::jsonb,
		    headers = $8::jsonb,
		    auth = $9::jsonb,
		    body_type = $10,
		    body = $11::jsonb,
		    settings = $12::jsonb,
		    enabled = $13,
		    sort_order = $14,
		    updated_by = $15,
		    updated_at = now()
		WHERE project_id = $1 AND id = $2
		RETURNING id, project_id, connection_id, group_id, name, method, url, params, headers,
		          auth, body_type, body, settings, enabled, sort_order, last_response,
		          quality, last_sent_at, created_at, updated_at
	`, params.ProjectID, params.ID, params.GroupID, params.Name, params.Method, params.URL, mustMarshalJSONArray(params.Params), mustMarshalJSONArray(params.Headers), mustMarshalJSONObject(params.Auth), params.BodyType, mustMarshalJSONObject(params.Body), mustMarshalJSONObject(params.Settings), params.Enabled, params.SortOrder, params.UserID).Scan(
		&record.ID, &record.ProjectID, &record.ConnectionID, &record.GroupID, &record.Name, &record.Method, &record.URL,
		newJSONScanner(&record.Params), newJSONScanner(&record.Headers), newJSONScanner(&record.Auth), &record.BodyType,
		newJSONScanner(&record.Body), newJSONScanner(&record.Settings), &record.Enabled, &record.SortOrder,
		newJSONScanner(&record.LastResponse), &record.Quality, &record.LastSentAt, &record.CreatedAt, &record.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "HTTP 请求不存在")
		}
		return nil, translateHTTPWorkbenchWriteError(err, "更新 HTTP 请求失败")
	}

	dataPointID, dataPointPath, err := upsertHTTPRequestDataPoint(ctx, tx, record, params.DataPointPath, params.DataPointConfig, params.DefaultValue, params.UserID)
	if err != nil {
		return nil, err
	}
	record.DataPointID = &dataPointID
	record.DataPointPath = &dataPointPath

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 HTTP 请求更新事务失败", err)
	}
	return &record, nil
}

// DeleteRequestWithDataPoint 删除 HTTP 请求并标记对应数据点失效。
func (r *HTTPWorkbenchRepository) DeleteRequestWithDataPoint(ctx context.Context, projectID, requestID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 HTTP 请求删除事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	if _, err := tx.Exec(ctx, `
		UPDATE data_points
		SET status = 'invalid', updated_at = now()
		WHERE project_id = $1
		  AND source_type = 'http.request'
		  AND source_config->>'requestId' = $2
	`, projectID, requestID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "标记 HTTP 请求数据点失效失败", err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM data_http_requests WHERE project_id = $1 AND id = $2`, projectID, requestID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 HTTP 请求失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "HTTP 请求不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 HTTP 请求删除事务失败", err)
	}
	return nil
}

// SaveRequestSendSnapshot 保存一次 Send 结果，并把成功响应写入对应数据点默认值。
func (r *HTTPWorkbenchRepository) SaveRequestSendSnapshot(ctx context.Context, projectID string, snapshot HTTPRequestSendSnapshot) error {
	responsePayload, err := json.Marshal(snapshot.LastResponse)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP 响应快照格式无效", err)
	}
	configPayload, err := json.Marshal(snapshot.DataPointConfig)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP 数据点 sourceConfig 格式无效", err)
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 HTTP 响应写回事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	tag, err := tx.Exec(ctx, `
		UPDATE data_http_requests
		SET last_response = $3::jsonb,
		    quality = $4,
		    last_sent_at = $5,
		    updated_by = $6,
		    updated_at = now()
		WHERE project_id = $1 AND id = $2
	`, projectID, snapshot.RequestID, string(responsePayload), snapshot.Quality, snapshot.LastSentAt, snapshot.UserID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "保存 HTTP 响应快照失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "HTTP 请求不存在")
	}

	if snapshot.DefaultValue != nil {
		_, err = tx.Exec(ctx, `
			UPDATE data_points
			SET default_value = $3,
			    source_config = $4::jsonb,
			    updated_by = $5,
			    updated_at = now()
			WHERE project_id = $1
			  AND source_type = 'http.request'
			  AND source_config->>'requestId' = $2
		`, projectID, snapshot.RequestID, snapshot.DefaultValue, string(configPayload), snapshot.UserID)
		if err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写回 HTTP 请求数据点失败", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 HTTP 响应写回事务失败", err)
	}
	return nil
}

func upsertHTTPRequestDataPoint(ctx context.Context, tx pgx.Tx, record HTTPRequestRecord, path string, sourceConfig map[string]any, defaultValue *string, userID string) (string, string, error) {
	config := cloneProtocolMap(sourceConfig)
	config["requestId"] = record.ID
	configPayload, err := json.Marshal(config)
	if err != nil {
		return "", "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP 请求数据点 sourceConfig 格式无效", err)
	}
	status := "active"
	if !record.Enabled {
		status = "inactive"
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
		  AND source_type = 'http.request'
		  AND source_config->>'requestId' = $9
		RETURNING id, path
	`, record.ProjectID, path, record.Name, record.ConnectionID, string(configPayload), defaultValue, status, userID, record.ID).Scan(&dataPointID, &dataPointPath)
	if err == nil {
		return dataPointID, dataPointPath, nil
	}
	if err != pgx.ErrNoRows {
		return "", "", apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "同步 HTTP 请求数据点失败", err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO data_points (
			project_id, path, name, source_type, source_id, source_config,
			data_type, default_value, refresh_mode, status, created_by, updated_by
		)
		VALUES ($1, $2, $3, 'http.request', $4, $5::jsonb, 'object', $6, 'manual', $7, $8, $8)
		ON CONFLICT (project_id, path)
		DO UPDATE SET
			name = EXCLUDED.name,
			source_type = EXCLUDED.source_type,
			source_id = EXCLUDED.source_id,
			source_config = EXCLUDED.source_config,
			data_type = EXCLUDED.data_type,
			default_value = COALESCE(EXCLUDED.default_value, data_points.default_value),
			refresh_mode = EXCLUDED.refresh_mode,
			status = EXCLUDED.status,
			updated_by = EXCLUDED.updated_by,
			updated_at = now()
		RETURNING id, path
	`, record.ProjectID, path, record.Name, record.ConnectionID, string(configPayload), defaultValue, status, userID).Scan(&dataPointID, &dataPointPath)
	if err != nil {
		return "", "", apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "同步 HTTP 请求数据点失败", err)
	}
	return dataPointID, dataPointPath, nil
}

func scanHTTPRequestGroupRecord(row pgx.Row) (HTTPRequestGroupRecord, error) {
	record := HTTPRequestGroupRecord{}
	if err := row.Scan(&record.ID, &record.ProjectID, &record.ConnectionID, &record.ParentID, &record.Name, &record.SortOrder, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return record, err
	}
	return record, nil
}

func scanHTTPRequestRecord(row pgx.Row) (HTTPRequestRecord, error) {
	record := HTTPRequestRecord{}
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.GroupID,
		&record.Name,
		&record.Method,
		&record.URL,
		newJSONScanner(&record.Params),
		newJSONScanner(&record.Headers),
		newJSONScanner(&record.Auth),
		&record.BodyType,
		newJSONScanner(&record.Body),
		newJSONScanner(&record.Settings),
		&record.Enabled,
		&record.SortOrder,
		newJSONScanner(&record.LastResponse),
		&record.Quality,
		&record.LastSentAt,
		&record.DataPointID,
		&record.DataPointPath,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return record, err
	}
	normalizeHTTPRequestJSONDefaults(&record)
	return record, nil
}

func normalizeHTTPRequestJSONDefaults(record *HTTPRequestRecord) {
	if record.Params == nil {
		record.Params = []any{}
	}
	if record.Headers == nil {
		record.Headers = []any{}
	}
	if record.Auth == nil {
		record.Auth = map[string]any{}
	}
	if record.Body == nil {
		record.Body = map[string]any{}
	}
	if record.Settings == nil {
		record.Settings = map[string]any{}
	}
}

type jsonScanner struct {
	target any
}

func newJSONScanner(target any) *jsonScanner {
	return &jsonScanner{target: target}
}

func (s *jsonScanner) Scan(value any) error {
	if value == nil {
		return nil
	}
	payload, ok := value.([]byte)
	if !ok {
		text, ok := value.(string)
		if !ok {
			return fmt.Errorf("JSON 字段类型无效")
		}
		payload = []byte(text)
	}
	if len(payload) == 0 {
		return nil
	}
	return json.Unmarshal(payload, s.target)
}

func mustMarshalJSONObject(value map[string]any) string {
	payload, _ := json.Marshal(value)
	return string(payload)
}

func mustMarshalJSONArray(value []any) string {
	payload, _ := json.Marshal(value)
	return string(payload)
}

func translateHTTPWorkbenchWriteError(err error, fallback string) error {
	var pgErr *pgconn.PgError
	if err != nil && strings.Contains(err.Error(), "JSON 字段类型无效") {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, fallback, err)
	}
	if ok := strings.TrimSpace(fallback); ok == "" {
		fallback = "HTTP 工作台写入失败"
	}
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "同名 HTTP 请求或分组已存在")
		case "23503":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "关联的 HTTP 接入源或分组不存在")
		}
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, fallback, err)
}
