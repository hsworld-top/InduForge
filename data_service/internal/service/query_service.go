package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/db/postgres"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

var allowedQueryTypes = map[string]struct{}{
	"sql":      {},
	"tags":     {},
	"http":     {},
	"mqtt_pub": {},
	"mqtt_sub": {},
}

// Query 表示面向 HTTP 层返回的查询对象。
type Query struct {
	ID              string         `json:"id"`
	ProjectID       string         `json:"projectId"`
	ConnectionID    string         `json:"connectionId"`
	Name            string         `json:"name"`
	Description     *string        `json:"description"`
	Category        *string        `json:"category"`
	QueryType       string         `json:"queryType"`
	Config          map[string]any `json:"config"`
	Transformer     *string        `json:"transformer"`
	IsEnabled       bool           `json:"isEnabled"`
	TimeoutMS       int            `json:"timeoutMs"`
	CacheEnabled    bool           `json:"cacheEnabled"`
	CacheTtlSeconds int            `json:"cacheTtlSeconds"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// QueryPagination 表示查询列表分页信息。
type QueryPagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// QueryListResult 表示查询列表与分页结果。
type QueryListResult struct {
	Queries    []Query         `json:"queries"`
	Pagination QueryPagination `json:"pagination"`
}

// QueryExecutionResult 表示查询执行后的返回结果。
type QueryExecutionResult struct {
	Data          any   `json:"data"`
	ExecutionTime int64 `json:"executionTime"`
	RowCount      int   `json:"rowCount"`
}

// QueryListFilter 表示 service 层对查询列表的入口参数。
type QueryListFilter struct {
	ConnectionID string
	QueryType    string
	Search       string
	IsEnabled    *bool
	Page         int
	PageSize     int
}

// CreateQueryInput 表示创建查询时的业务输入。
type CreateQueryInput struct {
	Name            string
	Description     *string
	Category        *string
	ConnectionID    string
	QueryType       string
	Config          map[string]any
	Transformer     *string
	IsEnabled       *bool
	TimeoutMS       *int
	CacheEnabled    *bool
	CacheTtlSeconds *int
}

// UpdateQueryInput 表示更新查询时的业务输入。
type UpdateQueryInput struct {
	Name            *string
	Description     *string
	Category        *string
	ConnectionID    *string
	QueryType       *string
	Config          map[string]any
	HasConfig       bool
	Transformer     *string
	IsEnabled       *bool
	TimeoutMS       *int
	CacheEnabled    *bool
	CacheTtlSeconds *int
}

// ExecuteQueryInput 表示查询执行时的输入。
type ExecuteQueryInput struct {
	Parameters map[string]any
}

// QueryService 承载查询领域的校验、映射与执行逻辑。
type QueryService struct {
	repository  *repository.QueryRepository
	connections *repository.ConnectionRepository
}

// NewQueryService 创建查询服务。
// 第三个参数保留是为了兼容现有装配签名，当前不直接复用服务元数据库连接池执行 SQL。
func NewQueryService(repo *repository.QueryRepository, connectionRepo *repository.ConnectionRepository, _ *pgxpool.Pool) *QueryService {
	return &QueryService{
		repository:  repo,
		connections: connectionRepo,
	}
}

// ListQueries 查询项目下的查询列表。
func (s *QueryService) ListQueries(ctx context.Context, projectID string, filter QueryListFilter) (*QueryListResult, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}

	records, total, err := s.repository.ListByProject(ctx, projectID, repository.QueryListFilter{
		ConnectionID: strings.TrimSpace(filter.ConnectionID),
		QueryType:    strings.TrimSpace(filter.QueryType),
		Search:       strings.TrimSpace(filter.Search),
		IsEnabled:    filter.IsEnabled,
		Page:         filter.Page,
		PageSize:     filter.PageSize,
	})
	if err != nil {
		return nil, err
	}

	queries := make([]Query, 0, len(records))
	for _, record := range records {
		queries = append(queries, toQuery(record))
	}

	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 1, 100)
	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	return &QueryListResult{
		Queries: queries,
		Pagination: QueryPagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// CreateQuery 创建项目下的查询定义。
func (s *QueryService) CreateQuery(ctx context.Context, projectID, userID string, input CreateQueryInput) (*Query, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	name, err := normalizeQueryName(input.Name)
	if err != nil {
		return nil, err
	}
	connectionID, err := normalizeConnectionID(input.ConnectionID)
	if err != nil {
		return nil, err
	}
	queryType, err := normalizeQueryType(input.QueryType)
	if err != nil {
		return nil, err
	}
	config, err := normalizeQueryConfig(input.Config)
	if err != nil {
		return nil, err
	}
	if queryType == "sql" {
		if _, err := extractQuerySQL(config); err != nil {
			return nil, err
		}
	}

	if _, err := s.connections.GetByProjectAndID(ctx, projectID, connectionID); err != nil {
		return nil, err
	}

	isEnabled := true
	if input.IsEnabled != nil {
		isEnabled = *input.IsEnabled
	}
	timeoutMS := 30000
	if input.TimeoutMS != nil {
		timeoutMS = *input.TimeoutMS
	}
	cacheEnabled := false
	if input.CacheEnabled != nil {
		cacheEnabled = *input.CacheEnabled
	}
	cacheTtlSeconds := 300
	if input.CacheTtlSeconds != nil {
		cacheTtlSeconds = *input.CacheTtlSeconds
	}

	record, err := s.repository.Create(ctx, repository.CreateQueryParams{
		ProjectID:       projectID,
		ConnectionID:    connectionID,
		UserID:          userID,
		Name:            name,
		Description:     cloneOptionalString(input.Description),
		Category:        cloneOptionalString(input.Category),
		QueryType:       queryType,
		Config:          config,
		Transformer:     cloneOptionalString(input.Transformer),
		IsEnabled:       isEnabled,
		TimeoutMS:       timeoutMS,
		CacheEnabled:    cacheEnabled,
		CacheTtlSeconds: cacheTtlSeconds,
	})
	if err != nil {
		return nil, err
	}

	query := toQuery(*record)
	return &query, nil
}

// ExecuteQuery 按查询主键执行查询，并先校验当前 claims 的项目边界。
func (s *QueryService) ExecuteQuery(ctx context.Context, claims *auth.Claims, queryID string, input ExecuteQueryInput) (*QueryExecutionResult, error) {
	record, err := s.loadQueryForClaims(ctx, claims, queryID)
	if err != nil {
		return nil, err
	}
	if !record.IsEnabled {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "查询已被禁用")
	}

	return s.executeRecord(ctx, *record, input.Parameters)
}

// ExecuteQueryForProject 按项目与查询主键执行查询，适用于 datapoint value 的内部调用。
func (s *QueryService) ExecuteQueryForProject(ctx context.Context, projectID, queryID string, input ExecuteQueryInput) (*QueryExecutionResult, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateQueryID(queryID); err != nil {
		return nil, err
	}

	record, err := s.repository.GetByProjectAndID(ctx, projectID, queryID)
	if err != nil {
		return nil, err
	}
	if !record.IsEnabled {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "查询已被禁用")
	}

	return s.executeRecord(ctx, *record, input.Parameters)
}

// UpdateQuery 按查询主键更新查询，并先校验当前 claims 的项目边界。
func (s *QueryService) UpdateQuery(ctx context.Context, claims *auth.Claims, queryID, userID string, input UpdateQueryInput) (*Query, error) {
	record, err := s.loadQueryForClaims(ctx, claims, queryID)
	if err != nil {
		return nil, err
	}

	updated, err := s.updateRecord(ctx, record, userID, input)
	if err != nil {
		return nil, err
	}

	query := toQuery(*updated)
	return &query, nil
}

// DeleteQuery 按查询主键删除查询，并先校验当前 claims 的项目边界。
func (s *QueryService) DeleteQuery(ctx context.Context, claims *auth.Claims, queryID string) error {
	record, err := s.loadQueryForClaims(ctx, claims, queryID)
	if err != nil {
		return err
	}

	return s.repository.Delete(ctx, record.ProjectID, queryID)
}

// updateRecord 负责把更新输入与现有记录合并后持久化。
// 关键分支：如果更新后类型仍为 sql，则必须保留可执行 SQL；否则执行入口会在运行时失败。
func (s *QueryService) updateRecord(ctx context.Context, current *repository.QueryRecord, userID string, input UpdateQueryInput) (*repository.QueryRecord, error) {
	if current == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "查询不存在")
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if input.Name == nil && input.Description == nil && input.Category == nil && input.ConnectionID == nil && input.QueryType == nil && !input.HasConfig && input.Transformer == nil && input.IsEnabled == nil && input.TimeoutMS == nil && input.CacheEnabled == nil && input.CacheTtlSeconds == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "至少需要提供一个待更新字段")
	}

	nextName := current.Name
	if input.Name != nil {
		normalized, err := normalizeQueryName(*input.Name)
		if err != nil {
			return nil, err
		}
		nextName = normalized
	}

	nextDescription := cloneOptionalString(current.Description)
	if input.Description != nil {
		nextDescription = cloneOptionalString(input.Description)
	}

	nextCategory := cloneOptionalString(current.Category)
	if input.Category != nil {
		nextCategory = cloneOptionalString(input.Category)
	}

	nextConnectionID := current.ConnectionID
	if input.ConnectionID != nil {
		normalized, err := normalizeConnectionID(*input.ConnectionID)
		if err != nil {
			return nil, err
		}
		nextConnectionID = normalized
		if _, err := s.connections.GetByProjectAndID(ctx, current.ProjectID, nextConnectionID); err != nil {
			return nil, err
		}
	}

	nextQueryType := current.QueryType
	if input.QueryType != nil {
		normalized, err := normalizeQueryType(*input.QueryType)
		if err != nil {
			return nil, err
		}
		nextQueryType = normalized
	}

	nextConfig := cloneMap(current.Config)
	if input.HasConfig {
		normalized, err := normalizeQueryConfig(input.Config)
		if err != nil {
			return nil, err
		}
		nextConfig = normalized
	}
	if nextQueryType == "sql" {
		if _, err := extractQuerySQL(nextConfig); err != nil {
			return nil, err
		}
	}

	nextTransformer := cloneOptionalString(current.Transformer)
	if input.Transformer != nil {
		nextTransformer = cloneOptionalString(input.Transformer)
	}

	nextIsEnabled := current.IsEnabled
	if input.IsEnabled != nil {
		nextIsEnabled = *input.IsEnabled
	}

	nextTimeoutMS := current.TimeoutMS
	if input.TimeoutMS != nil {
		nextTimeoutMS = *input.TimeoutMS
	}

	nextCacheEnabled := current.CacheEnabled
	if input.CacheEnabled != nil {
		nextCacheEnabled = *input.CacheEnabled
	}

	nextCacheTtlSeconds := current.CacheTtlSeconds
	if input.CacheTtlSeconds != nil {
		nextCacheTtlSeconds = *input.CacheTtlSeconds
	}

	updated, err := s.repository.Update(ctx, repository.UpdateQueryParams{
		ID:              current.ID,
		ProjectID:       current.ProjectID,
		ConnectionID:    nextConnectionID,
		UserID:          userID,
		Name:            nextName,
		Description:     nextDescription,
		Category:        nextCategory,
		QueryType:       nextQueryType,
		Config:          nextConfig,
		Transformer:     nextTransformer,
		IsEnabled:       nextIsEnabled,
		TimeoutMS:       nextTimeoutMS,
		CacheEnabled:    nextCacheEnabled,
		CacheTtlSeconds: nextCacheTtlSeconds,
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// executeRecord 负责真正执行 sql 查询。
// 关键分支：先强制只读 SQL，再进入只读事务执行，防止查询接口被滥用于写操作。
func (s *QueryService) executeRecord(ctx context.Context, record repository.QueryRecord, parameters map[string]any) (*QueryExecutionResult, error) {
	if record.QueryType != "sql" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前仅支持 sql 查询执行")
	}

	sqlText, err := extractQuerySQL(record.Config)
	if err != nil {
		return nil, err
	}
	if err := ensureReadOnlySQL(sqlText); err != nil {
		return nil, err
	}

	args, err := buildQueryExecutionArgs(record.Config, parameters)
	if err != nil {
		return nil, err
	}

	timeoutMS := record.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 30000
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	execPool, release, err := s.executionPoolForRecord(execCtx, record)
	if err != nil {
		return nil, err
	}
	defer release()

	tx, err := execPool.BeginTx(execCtx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "创建只读执行会话失败", err)
	}
	defer func() {
		rollbackErr := tx.Rollback(execCtx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			// 只读查询场景下，回滚错误不影响业务返回。
		}
	}()

	start := time.Now()
	rows, err := tx.Query(execCtx, sqlText, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "执行查询失败", err)
	}
	defer rows.Close()

	columns := rows.FieldDescriptions()
	resultRows := make([]map[string]any, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取查询结果失败", err)
		}

		row := make(map[string]any, len(values))
		for i, field := range columns {
			row[string(field.Name)] = normalizeQueryValue(values[i])
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历查询结果失败", err)
	}

	executionTime := time.Since(start).Milliseconds()
	return &QueryExecutionResult{
		Data:          resultRows,
		ExecutionTime: executionTime,
		RowCount:      len(resultRows),
	}, nil
}

// executionPoolForRecord 根据 query 关联的连接配置返回执行 SQL 的目标连接池。
// 查询路径说明：
// 1. 先按 project_id + connection_id 读取连接（命中 data_connections 主键路径）。
// 2. 再从 connection metadata 中读取 databaseUrl/searchPath 生成目标池。
// 性能风险：当前为“每次执行临时建池”，简单但开销偏高；后续可引入连接池缓存。
func (s *QueryService) executionPoolForRecord(ctx context.Context, record repository.QueryRecord) (*pgxpool.Pool, func(), error) {
	connection, err := s.connections.GetByProjectAndID(ctx, record.ProjectID, record.ConnectionID)
	if err != nil {
		return nil, nil, err
	}
	if connection.Type != "relational" {
		return nil, nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前仅支持关系型连接执行 SQL")
	}

	databaseURL, searchPath := extractExecutionDatabaseConfig(connection.Config)
	if databaseURL == "" {
		return nil, nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接配置缺少 databaseUrl，无法执行查询")
	}

	pool, err := postgres.NewPool(ctx, postgres.PoolConfig{
		DatabaseURL: databaseURL,
		SearchPath:  searchPath,
	})
	if err != nil {
		return nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "创建执行目标连接池失败", err)
	}

	return pool, pool.Close, nil
}

func extractExecutionDatabaseConfig(config map[string]any) (string, string) {
	if config == nil {
		return "", ""
	}

	findString := func(keys ...string) string {
		for _, key := range keys {
			raw, ok := config[key]
			if !ok {
				continue
			}
			text, ok := raw.(string)
			if !ok {
				continue
			}
			text = strings.TrimSpace(text)
			if text != "" {
				return text
			}
		}
		return ""
	}

	databaseURL := findString("databaseUrl", "databaseURL", "dsn")
	searchPath := findString("searchPath", "schema")
	return databaseURL, searchPath
}

// loadQueryForClaims 读取查询并校验当前 claims 的项目边界。
func (s *QueryService) loadQueryForClaims(ctx context.Context, claims *auth.Claims, queryID string) (*repository.QueryRecord, error) {
	if claims == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成认证")
	}
	if err := validateQueryID(queryID); err != nil {
		return nil, err
	}

	record, err := s.repository.GetByID(ctx, queryID)
	if err != nil {
		return nil, err
	}
	if !claims.HasProjectAccess(record.ProjectID) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodePermissionProjectMismatch, http.StatusForbidden, "项目范围不足")
	}

	return record, nil
}

// toQuery 将仓储记录映射为 HTTP 返回对象。
func toQuery(record repository.QueryRecord) Query {
	return Query{
		ID:              record.ID,
		ProjectID:       record.ProjectID,
		ConnectionID:    record.ConnectionID,
		Name:            record.Name,
		Description:     cloneOptionalString(record.Description),
		Category:        cloneOptionalString(record.Category),
		QueryType:       record.QueryType,
		Config:          cloneMap(record.Config),
		Transformer:     cloneOptionalString(record.Transformer),
		IsEnabled:       record.IsEnabled,
		TimeoutMS:       record.TimeoutMS,
		CacheEnabled:    record.CacheEnabled,
		CacheTtlSeconds: record.CacheTtlSeconds,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}
}

func normalizeQueryName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "查询名称不能为空")
	}
	if len([]rune(name)) > 200 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "查询名称长度不能超过 200 个字符")
	}
	return name, nil
}

func normalizeQueryType(queryType string) (string, error) {
	queryType = strings.TrimSpace(strings.ToLower(queryType))
	if _, ok := allowedQueryTypes[queryType]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "查询类型不受支持")
	}
	return queryType, nil
}

func normalizeQueryConfig(config map[string]any) (map[string]any, error) {
	if config == nil {
		return map[string]any{}, nil
	}
	return cloneMap(config), nil
}

func extractQuerySQL(config map[string]any) (string, error) {
	if config == nil {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "sql 配置不能为空")
	}

	raw, ok := config["sql"]
	if !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "sql 配置缺失")
	}

	sqlText, ok := raw.(string)
	if !ok || strings.TrimSpace(sqlText) == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "sql 配置无效")
	}

	return strings.TrimSpace(sqlText), nil
}

func ensureReadOnlySQL(sqlText string) error {
	normalized := strings.ToLower(strings.TrimSpace(sqlText))
	if normalized == "" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "sql 不能为空")
	}
	if strings.HasPrefix(normalized, "select") || strings.HasPrefix(normalized, "with") || strings.HasPrefix(normalized, "values") || strings.HasPrefix(normalized, "show") || strings.HasPrefix(normalized, "explain") {
		return nil
	}
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前仅允许只读 SQL")
}

func buildQueryExecutionArgs(config map[string]any, parameters map[string]any) ([]any, error) {
	rawDefs, ok := config["parameters"]
	if !ok || rawDefs == nil {
		if len(parameters) > 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前查询未定义参数")
		}
		return []any{}, nil
	}

	defs, ok := rawDefs.([]any)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "parameters 配置格式无效")
	}

	args := make([]any, 0, len(defs))
	for _, rawDef := range defs {
		defMap, ok := rawDef.(map[string]any)
		if !ok {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "参数定义格式无效")
		}

		name, _ := defMap["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "参数定义缺少 name")
		}

		value, ok := parameters[name]
		if !ok {
			if defaultValue, exists := defMap["default"]; exists {
				value = defaultValue
			} else {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "缺少查询参数: "+name)
			}
		}

		args = append(args, value)
	}

	return args, nil
}

func normalizeQueryValue(value any) any {
	switch v := value.(type) {
	case []byte:
		return string(v)
	default:
		return value
	}
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	next := strings.TrimSpace(*value)
	return &next
}

func normalizeConnectionID(connectionID string) (string, error) {
	connectionID = strings.TrimSpace(connectionID)
	if connectionID == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "connectionId 不能为空")
	}
	if _, err := uuid.Parse(connectionID); err != nil {
		return "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "connectionId 格式无效", err)
	}
	return connectionID, nil
}

func validateQueryID(queryID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(queryID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "queryId 格式无效", err)
	}
	return nil
}

// normalizePageAndSize 统一处理分页参数，避免列表接口在不同服务间出现默认值不一致。
func normalizePageAndSize(page, pageSize, defaultPage, maxPageSize int) (int, int) {
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = maxPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}
