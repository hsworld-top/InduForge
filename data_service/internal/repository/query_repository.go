package repository

import (
	"context"
	"database/sql"
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

// QueryRecord 表示 data_queries 表在仓储层的投影结果。
type QueryRecord struct {
	ID              string
	ProjectID       string
	ConnectionID    string
	Name            string
	Description     *string
	Category        *string
	GroupID         *string
	QueryType       string
	Config          map[string]any
	Transformer     *string
	IsEnabled       bool
	TimeoutMS       int
	CacheEnabled    bool
	CacheTtlSeconds int
	CreatedBy       string
	UpdatedBy       *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// QueryListFilter 表示查询列表的分页与过滤条件。
type QueryListFilter struct {
	ConnectionID string
	QueryType    string
	Search       string
	IsEnabled    *bool
	Page         int
	PageSize     int
}

// QueryRepository 封装 data_queries 的参数化 SQL 访问。
type QueryRepository struct {
	pool *pgxpool.Pool
}

// NewQueryRepository 创建查询仓储。
func NewQueryRepository(pool *pgxpool.Pool) *QueryRepository {
	return &QueryRepository{pool: pool}
}

// ListByProject 按项目分页查询 data_queries。
// 查询路径：project_id + 可选过滤条件 + created_at 排序，主要命中 data_queries_project_connection_idx / data_queries_type_enabled_idx。
// 潜在性能风险：search 使用 ILIKE 会降低索引命中率，数据量上来后可能退化为顺序扫描，必要时再补充全文检索或专用搜索索引。
func (r *QueryRepository) ListByProject(ctx context.Context, projectID string, filter QueryListFilter) ([]QueryRecord, int, error) {
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 20, 100)
	whereSQL, args := buildQueryWhereClause(projectID, filter)

	countSQL := `SELECT COUNT(*) FROM data_queries WHERE ` + whereSQL
	var total int
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计查询列表失败", err)
	}

	selectArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, connection_id, name, description, category, group_id, query_type, config, transformer,
               is_enabled, timeout_ms, cache_enabled, cache_ttl_seconds, created_by, updated_by, created_at, updated_at
        FROM data_queries
        WHERE `+whereSQL+`
        ORDER BY created_at DESC
        LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2)+`
    `, selectArgs...)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询查询列表失败", err)
	}
	defer rows.Close()

	records := make([]QueryRecord, 0)
	for rows.Next() {
		record, scanErr := scanQueryRecord(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历查询列表失败", err)
	}

	return records, total, nil
}

// GetByProjectAndID 按项目与主键读取单条查询。
// 查询路径：project_id + id，主命中 data_queries_pkey，project_id 作为边界约束避免跨项目误读。
// 潜在性能风险：若后续频繁按 project_id + id 复合访问，可再评估联合索引，但当前主键命中已足够稳定。
func (r *QueryRepository) GetByProjectAndID(ctx context.Context, projectID, queryID string) (*QueryRecord, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, project_id, connection_id, name, description, category, group_id, query_type, config, transformer,
               is_enabled, timeout_ms, cache_enabled, cache_ttl_seconds, created_by, updated_by, created_at, updated_at
        FROM data_queries
        WHERE project_id = $1 AND id = $2
    `, projectID, queryID)

	record, err := scanQueryRecord(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByID 按主键读取单条查询。
// 查询路径：id，主命中 data_queries_pkey；适合无 projectId 路径先取出项目边界再做 claims 校验。
// 潜在性能风险：如果后续大量依赖 project_id 过滤，仍建议在业务侧尽量保留项目边界参数。
func (r *QueryRepository) GetByID(ctx context.Context, queryID string) (*QueryRecord, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, project_id, connection_id, name, description, category, group_id, query_type, config, transformer,
               is_enabled, timeout_ms, cache_enabled, cache_ttl_seconds, created_by, updated_by, created_at, updated_at
        FROM data_queries
        WHERE id = $1
    `, queryID)

	record, err := scanQueryRecord(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// Create 写入一条新的查询记录。
// 查询路径：project_id + name 唯一约束，主要命中 data_queries_project_name_key，写入时同时依赖 data_queries_connection_project_idx 的关联校验。
// 潜在性能风险：写入阶段会触发 JSONB 编码与唯一约束检查；若配置体积变大，应关注 config 字段体积与冲突重试成本。
func (r *QueryRepository) Create(ctx context.Context, params CreateQueryParams) (*QueryRecord, error) {
	configBytes, err := marshalJSONObject(params.Config)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, `
        INSERT INTO data_queries (
            project_id,
            connection_id,
            name,
            description,
            category,
            group_id,
            query_type,
            config,
            transformer,
            is_enabled,
            timeout_ms,
            cache_enabled,
            cache_ttl_seconds,
            created_by,
            updated_by
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11, $12, $13, $14, $14)
        RETURNING id, project_id, connection_id, name, description, category, group_id, query_type, config, transformer,
                  is_enabled, timeout_ms, cache_enabled, cache_ttl_seconds, created_by, updated_by, created_at, updated_at
    `, params.ProjectID, params.ConnectionID, params.Name, params.Description, params.Category, params.GroupID, params.QueryType, string(configBytes), params.Transformer, params.IsEnabled, params.TimeoutMS, params.CacheEnabled, params.CacheTtlSeconds, params.UserID)

	record, err := scanQueryRecord(row)
	if err != nil {
		return nil, translateQueryWriteError(err)
	}

	return &record, nil
}

// Update 更新一条查询记录。
// 查询路径：project_id + id，主命中 data_queries_pkey；更新时仍保留项目边界，防止误写其他项目。
// 潜在性能风险：每次更新都会重写 JSONB 配置与时间戳，若配置较大且更新频繁，写放大会明显增加。
func (r *QueryRepository) Update(ctx context.Context, params UpdateQueryParams) (*QueryRecord, error) {
	configBytes, err := marshalJSONObject(params.Config)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, `
        UPDATE data_queries
        SET connection_id = $3,
            name = $4,
            description = $5,
            category = $6,
            group_id = $7,
            query_type = $8,
            config = $9::jsonb,
            transformer = $10,
            is_enabled = $11,
            timeout_ms = $12,
            cache_enabled = $13,
            cache_ttl_seconds = $14,
            updated_by = $15,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, connection_id, name, description, category, group_id, query_type, config, transformer,
                  is_enabled, timeout_ms, cache_enabled, cache_ttl_seconds, created_by, updated_by, created_at, updated_at
    `, params.ProjectID, params.ID, params.ConnectionID, params.Name, params.Description, params.Category, params.GroupID, params.QueryType, string(configBytes), params.Transformer, params.IsEnabled, params.TimeoutMS, params.CacheEnabled, params.CacheTtlSeconds, params.UserID)

	record, err := scanQueryRecord(row)
	if err != nil {
		return nil, translateQueryWriteError(err)
	}

	return &record, nil
}

// Delete 按项目与主键删除查询。
// 查询路径：project_id + id，主命中 data_queries_pkey；删除前保留 project_id 边界，避免无意删除别的项目记录。
// 潜在性能风险：单条删除代价很低，但如果未来需要级联同步大量 datapoints，应注意级联逻辑不要在单次请求中放大。
func (r *QueryRepository) Delete(ctx context.Context, projectID, queryID string) error {
	commandTag, err := r.pool.Exec(ctx, `
        DELETE FROM data_queries
        WHERE project_id = $1 AND id = $2
    `, projectID, queryID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除查询失败", err)
	}
	if commandTag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "查询不存在")
	}
	return nil
}

// CreateQueryParams 表示查询写入时的仓储参数。
type CreateQueryParams struct {
	ProjectID       string
	ConnectionID    string
	UserID          string
	Name            string
	Description     *string
	Category        *string
	GroupID         *string
	QueryType       string
	Config          map[string]any
	Transformer     *string
	IsEnabled       bool
	TimeoutMS       int
	CacheEnabled    bool
	CacheTtlSeconds int
}

// UpdateQueryParams 表示查询更新时的仓储参数。
type UpdateQueryParams struct {
	ID              string
	ProjectID       string
	ConnectionID    string
	UserID          string
	Name            string
	Description     *string
	Category        *string
	GroupID         *string
	QueryType       string
	Config          map[string]any
	Transformer     *string
	IsEnabled       bool
	TimeoutMS       int
	CacheEnabled    bool
	CacheTtlSeconds int
}

type queryScannable interface {
	Scan(dest ...any) error
}

func scanQueryRecord(row queryScannable) (QueryRecord, error) {
	var (
		record      QueryRecord
		description sql.NullString
		category    sql.NullString
		groupID     sql.NullString
		transformer sql.NullString
		updatedBy   sql.NullString
		configBytes []byte
	)

	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.Name,
		&description,
		&category,
		&groupID,
		&record.QueryType,
		&configBytes,
		&transformer,
		&record.IsEnabled,
		&record.TimeoutMS,
		&record.CacheEnabled,
		&record.CacheTtlSeconds,
		&record.CreatedBy,
		&updatedBy,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return QueryRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "查询不存在")
		}
		return QueryRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取查询失败", err)
	}

	record.Description = nullStringToPtr(description)
	record.Category = nullStringToPtr(category)
	record.GroupID = nullStringToPtr(groupID)
	record.Transformer = nullStringToPtr(transformer)
	record.UpdatedBy = nullStringToPtr(updatedBy)
	record.Config = map[string]any{}
	if len(configBytes) > 0 {
		if err := json.Unmarshal(configBytes, &record.Config); err != nil {
			return QueryRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析查询配置失败", err)
		}
		if record.Config == nil {
			record.Config = map[string]any{}
		}
	}

	return record, nil
}

func buildQueryWhereClause(projectID string, filter QueryListFilter) (string, []any) {
	clauses := []string{"project_id = $1"}
	args := []any{projectID}

	addStringClause := func(column, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		args = append(args, strings.TrimSpace(value))
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	addStringClause("connection_id", filter.ConnectionID)
	addStringClause("query_type", filter.QueryType)

	if filter.IsEnabled != nil {
		args = append(args, *filter.IsEnabled)
		clauses = append(clauses, fmt.Sprintf("is_enabled = $%d", len(args)))
	}

	search := strings.TrimSpace(filter.Search)
	if search != "" {
		args = append(args, "%"+search+"%")
		clauseIndex := len(args)
		clauses = append(clauses, fmt.Sprintf("(name ILIKE $%d OR COALESCE(description, '') ILIKE $%d)", clauseIndex, clauseIndex))
	}

	return strings.Join(clauses, " AND "), args
}

func normalizePageAndSize(page, pageSize, defaultPageSize, maxPageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func marshalJSONObject(value map[string]any) ([]byte, error) {
	if value == nil {
		value = map[string]any{}
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "JSON 对象格式无效", err)
	}
	return bytes, nil
}

func nullStringToPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func translateQueryWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "查询名称已存在")
		case "23503":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "关联的连接不存在")
		}
	}
	return err
}
