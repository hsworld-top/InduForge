package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/auth"
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

const maxSavedSQLBytes = 1024 * 1024

// Query 表示面向 HTTP 层返回的查询对象。
type Query struct {
	ID              string         `json:"id"`
	ProjectID       string         `json:"projectId"`
	ConnectionID    string         `json:"connectionId"`
	Name            string         `json:"name"`
	Description     *string        `json:"description"`
	Category        *string        `json:"category"`
	GroupID         *string        `json:"groupId"`
	QueryType       string         `json:"queryType"`
	Config          map[string]any `json:"config"`
	Transformer     *string        `json:"transformer"`
	IsEnabled       bool           `json:"isEnabled"`
	TimeoutMS       int            `json:"timeoutMs"`
	CacheEnabled    bool           `json:"cacheEnabled"`
	CacheTtlSeconds int            `json:"cacheTtlSeconds"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	Outputs         []SourceOutput `json:"outputs"`
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
	Data          any               `json:"data"`
	Columns       []string          `json:"columns"`
	ColumnTypes   map[string]string `json:"columnTypes"`
	ExecutionTime int64             `json:"executionTime"`
	RowCount      int               `json:"rowCount"`
	Truncated     bool              `json:"truncated"`
	TruncatedBy   string            `json:"truncatedBy,omitempty"`
	Limits        SQLResultLimits   `json:"limits"`
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
	GroupID         *string
	ConnectionID    string
	QueryType       string
	Config          map[string]any
	Transformer     *string
	IsEnabled       *bool
	TimeoutMS       *int
	CacheEnabled    *bool
	CacheTtlSeconds *int
	Outputs         []SourceOutputInput
}

// UpdateQueryInput 表示更新查询时的业务输入。
type UpdateQueryInput struct {
	Name            *string
	Description     *string
	Category        *string
	GroupID         *string
	ConnectionID    *string
	QueryType       *string
	Config          map[string]any
	HasConfig       bool
	Transformer     *string
	IsEnabled       *bool
	TimeoutMS       *int
	CacheEnabled    *bool
	CacheTtlSeconds *int
	Outputs         []SourceOutputInput
	HasOutputs      bool
}

// ExecuteQueryInput 表示查询执行时的输入。
type ExecuteQueryInput struct {
	Parameters map[string]any
}

// QueryService 承载查询领域的校验、映射与执行逻辑。
type QueryService struct {
	repository  *repository.QueryRepository
	connections *repository.ConnectionRepository
	datapoints  *repository.DataPointRepository
	builtin     *BuiltinRuntimeService
	secrets     *repository.ConnectionSecretRepository
}

// NewQueryService 创建查询服务。
// 第三个参数可传入 DataPointRepository，用于同步 db.query 数据点；保留 any 是为了兼容既有测试装配。
func NewQueryService(repo *repository.QueryRepository, connectionRepo *repository.ConnectionRepository, dataPointRepo any) *QueryService {
	datapoints, _ := dataPointRepo.(*repository.DataPointRepository)
	return &QueryService{
		repository:  repo,
		connections: connectionRepo,
		datapoints:  datapoints,
	}
}

// SetBuiltinRuntime 注入内置关系/时序运行库，让已保存查询作为数据点执行时可覆盖内置库。
func (s *QueryService) SetBuiltinRuntime(builtin *BuiltinRuntimeService) {
	if s != nil {
		s.builtin = builtin
	}
}

func (s *QueryService) SetSecretRepository(secrets *repository.ConnectionSecretRepository) {
	if s != nil {
		s.secrets = secrets
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

	connection, err := s.connections.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
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

	createParams := repository.CreateQueryParams{
		ProjectID:       projectID,
		ConnectionID:    connectionID,
		UserID:          userID,
		Name:            name,
		Description:     cloneOptionalString(input.Description),
		Category:        cloneOptionalString(input.Category),
		GroupID:         cloneOptionalString(input.GroupID),
		QueryType:       queryType,
		Config:          config,
		Transformer:     cloneOptionalString(input.Transformer),
		IsEnabled:       isEnabled,
		TimeoutMS:       timeoutMS,
		CacheEnabled:    cacheEnabled,
		CacheTtlSeconds: cacheTtlSeconds,
	}
	var record *repository.QueryRecord
	if len(input.Outputs) == 0 {
		// 查询定义与数据点显式解耦：普通保存只写查询，用户主动生成时才创建输出映射。
		record, err = s.repository.Create(ctx, createParams)
	} else {
		outputs, normalizeErr := normalizeSourceOutputs(input.Outputs, defaultQueryOutput(name))
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		record, err = s.repository.CreateWithOutput(ctx, createParams, buildQueryOutputDataPoint(*connection, name, cloneOptionalString(input.Description), config, outputs, userID, queryType == "sql"))
	}
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

// GetQuery 按主键读取查询定义，并校验查询所属工程在当前令牌范围内。
func (s *QueryService) GetQuery(ctx context.Context, claims *auth.Claims, queryID string) (*Query, error) {
	record, err := s.loadQueryForClaims(ctx, claims, queryID)
	if err != nil {
		return nil, err
	}
	query := toQuery(*record)
	return &query, nil
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

	if err := s.repository.Delete(ctx, record.ProjectID, queryID); err != nil {
		return err
	}
	return nil
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
	if input.Name == nil && input.Description == nil && input.Category == nil && input.GroupID == nil && input.ConnectionID == nil && input.QueryType == nil && !input.HasConfig && input.Transformer == nil && input.IsEnabled == nil && input.TimeoutMS == nil && input.CacheEnabled == nil && input.CacheTtlSeconds == nil && !input.HasOutputs {
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

	nextGroupID := cloneOptionalString(current.GroupID)
	if input.GroupID != nil {
		nextGroupID = cloneOptionalString(input.GroupID)
		if nextGroupID != nil && *nextGroupID == "" {
			nextGroupID = nil
		}
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

	connection, err := s.connections.GetByProjectAndID(ctx, current.ProjectID, nextConnectionID)
	if err != nil {
		return nil, err
	}
	updateParams := repository.UpdateQueryParams{
		ID:              current.ID,
		ProjectID:       current.ProjectID,
		ConnectionID:    nextConnectionID,
		UserID:          userID,
		Name:            nextName,
		Description:     nextDescription,
		Category:        nextCategory,
		GroupID:         nextGroupID,
		QueryType:       nextQueryType,
		Config:          nextConfig,
		Transformer:     nextTransformer,
		IsEnabled:       nextIsEnabled,
		TimeoutMS:       nextTimeoutMS,
		CacheEnabled:    nextCacheEnabled,
		CacheTtlSeconds: nextCacheTtlSeconds,
	}
	outputs := sourceOutputParamsFromRecords(current.Outputs)
	if input.HasOutputs {
		outputs, err = normalizeSourceOutputs(input.Outputs, defaultQueryOutput(nextName))
		if err != nil {
			return nil, err
		}
	}
	var updated *repository.QueryRecord
	if input.HasOutputs {
		updated, err = s.repository.UpdateWithOutput(ctx, updateParams, buildQueryOutputDataPoint(*connection, nextName, nextDescription, nextConfig, outputs, userID, nextQueryType == "sql"))
	} else {
		// 未显式提交 outputs 时保留现有映射，不让查询保存隐式创建或改写数据点。
		updated, err = s.repository.Update(ctx, updateParams)
	}
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// executeRecord 负责真正执行已保存 SQL。
// 数据点可显式建模为写操作：普通 DML 返回影响行数，RETURNING/OUTPUT 返回结果集。
func (s *QueryService) executeRecord(ctx context.Context, record repository.QueryRecord, parameters map[string]any) (*QueryExecutionResult, error) {
	if record.QueryType != "sql" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前仅支持 sql 查询执行")
	}

	sqlText, err := extractQuerySQL(record.Config)
	if err != nil {
		return nil, err
	}
	args, err := buildQueryExecutionArgs(record.Config, parameters)
	if err != nil {
		return nil, err
	}

	timeoutMS := record.TimeoutMS
	if timeoutMS <= 0 || timeoutMS > developmentSQLTimeout*1000 {
		timeoutMS = 30000
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()

	if result, ok, err := s.executeBuiltinRecord(execCtx, record, sqlText, args); ok || err != nil {
		return result, err
	}
	if result, ok, err := s.executeTDengineRecord(execCtx, record, sqlText, args); ok || err != nil {
		return result, err
	}

	runtime, err := s.executionRuntimeForRecord(execCtx, record)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()
	if err := validateSQLParameterCount(runtime.DBType(), sqlText, args); err != nil {
		return nil, err
	}

	start := time.Now()
	if !sqlWorkbenchStatementReturnsRows(sqlText) {
		affected, err := runtime.ExecAffected(execCtx, sqlText, args...)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, sqlExecutionErrorMessage("执行 SQL 失败", err), err)
		}
		return &QueryExecutionResult{
			Data:          []map[string]any{},
			Columns:       []string{},
			ColumnTypes:   map[string]string{},
			ExecutionTime: time.Since(start).Milliseconds(),
			RowCount:      int(affected),
			Limits:        developmentSQLLimits(),
		}, nil
	}

	rows, err := runtime.Query(execCtx, sqlText, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, sqlExecutionErrorMessage("执行查询失败", err), err)
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取查询字段元信息失败", err)
	}
	databaseTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取查询字段类型失败", err)
	}
	resultRows := make([]map[string]any, 0)
	resultBytes := 0
	truncatedBy := ""
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取查询结果失败", err)
		}

		row := make(map[string]any, len(values))
		for i, field := range columnNames {
			databaseType := ""
			if i < len(databaseTypes) {
				databaseType = databaseTypes[i]
			}
			row[field] = normalizeSQLValue(values[i], databaseType)
		}
		accepted, reason, nextBytes := admitSQLResultRow(len(resultRows), resultBytes, developmentSQLMaxRows, developmentSQLMaxBytes, row)
		if !accepted {
			truncatedBy = reason
			break
		}
		resultRows = append(resultRows, row)
		resultBytes = nextBytes
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历查询结果失败", err)
	}

	executionTime := time.Since(start).Milliseconds()
	return &QueryExecutionResult{
		Data:          resultRows,
		Columns:       columnNames,
		ColumnTypes:   canonicalSQLColumnTypes(columnNames, databaseTypes),
		ExecutionTime: executionTime,
		RowCount:      len(resultRows),
		Truncated:     truncatedBy != "",
		TruncatedBy:   truncatedBy,
		Limits:        developmentSQLLimits(),
	}, nil
}

func (s *QueryService) executeTDengineRecord(ctx context.Context, record repository.QueryRecord, sqlText string, args []any) (*QueryExecutionResult, bool, error) {
	connection, err := s.connections.GetByProjectAndID(ctx, record.ProjectID, record.ConnectionID)
	if err != nil {
		return nil, false, err
	}
	if connection.Type != "tdengine" {
		return nil, false, nil
	}
	if err := validateTDengineSingleSQL(sqlText); err != nil {
		return nil, true, err
	}
	if err := validateSQLParameterCount("tdengine", sqlText, args); err != nil {
		return nil, true, err
	}
	if s.secrets != nil {
		values, resolveErr := s.secrets.ResolveAll(ctx, connection.ID)
		if resolveErr != nil {
			return nil, true, resolveErr
		}
		connection.Config = injectConnectionSecrets(connection.Config, values)
	}
	runtime, err := connectTDengineRuntime(ctx, connection.Config)
	if err != nil {
		return nil, true, err
	}
	defer runtime.Close()
	started := time.Now()
	if !sqlWorkbenchStatementReturnsRows(sqlText) {
		affected, err := runtime.execAffected(ctx, sqlText, args...)
		if err != nil {
			return nil, true, err
		}
		return &QueryExecutionResult{Data: []map[string]any{}, Columns: []string{}, ColumnTypes: map[string]string{}, ExecutionTime: time.Since(started).Milliseconds(), RowCount: int(affected), Limits: developmentSQLLimits()}, true, nil
	}
	columns, columnTypes, rows, truncatedBy, err := runtime.queryBounded(ctx, sqlText, developmentSQLMaxRows, developmentSQLMaxBytes, args...)
	if err != nil {
		return nil, true, err
	}
	resultRows := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		item := map[string]any{}
		for index, column := range columns {
			if index < len(row) {
				item[column] = normalizeQueryValue(row[index])
			}
		}
		resultRows = append(resultRows, item)
	}
	return &QueryExecutionResult{Data: resultRows, Columns: columns, ColumnTypes: columnTypes, ExecutionTime: time.Since(started).Milliseconds(), RowCount: len(resultRows), Truncated: truncatedBy != "", TruncatedBy: truncatedBy, Limits: developmentSQLLimits()}, true, nil
}

// executionRuntimeForRecord 根据 query 关联的连接配置返回执行 SQL 的目标运行时。
// 当前保持“每次执行临时建连接”的简单策略，待真实流量确认后再考虑按 dbType 做连接缓存。
func (s *QueryService) executionRuntimeForRecord(ctx context.Context, record repository.QueryRecord) (*relationalRuntime, error) {
	connection, err := s.connections.GetByProjectAndID(ctx, record.ProjectID, record.ConnectionID)
	if err != nil {
		return nil, err
	}
	if connection.Type != "relational" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前仅支持关系型连接执行 SQL")
	}
	// 连接记录只保存非敏感配置；已保存查询经数据点 GET 回放时也必须注入密钥，
	// 否则会与工作台执行链路产生差异并使用空密码连接外部数据库。
	if s.secrets != nil {
		values, resolveErr := s.secrets.ResolveAll(ctx, connection.ID)
		if resolveErr != nil {
			return nil, resolveErr
		}
		connection.Config = injectConnectionSecrets(connection.Config, values)
	}
	return connectRelationalRuntime(ctx, connection.Config)
}

func (s *QueryService) executeBuiltinRecord(ctx context.Context, record repository.QueryRecord, sqlText string, args []any) (*QueryExecutionResult, bool, error) {
	connection, err := s.connections.GetByProjectAndID(ctx, record.ProjectID, record.ConnectionID)
	if err != nil {
		return nil, false, err
	}
	schemaName, ok, err := builtinSQLSchemaFromRecord(connection)
	if err != nil || !ok {
		return nil, ok, err
	}
	if s.builtin == nil {
		return nil, true, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态内置运行库未初始化")
	}
	if err := validateSQLParameterCount("postgresql", sqlText, args); err != nil {
		return nil, true, err
	}
	result, err := s.builtin.ExecuteSQLInSchema(ctx, schemaName, sqlText, args, 500)
	if err != nil {
		return nil, true, err
	}
	return builtinQueryResult(result), true, nil
}

func buildQueryOutputDataPoint(connection repository.ConnectionRecord, name string, description *string, config map[string]any, outputs []repository.SourceOutputMappingParam, userID string, enabled bool) repository.QueryOutputDataPointParams {
	supported := connection.Type == "relational" || connection.Type == "tdengine" || connection.Type == "builtin.relation" || connection.Type == "builtin.timeseries"
	pathPrefix := "db." + normalizeDatapointSegment(connection.Name) + "." + normalizeDatapointSegment(name)
	sourceConfig := map[string]any{
		"mode":         "query",
		"connectionId": connection.ID,
	}
	if parameters, ok := config["parameters"]; ok {
		sourceConfig["parameters"] = parameters
	}
	return repository.QueryOutputDataPointParams{
		Enabled:      enabled && supported,
		PathPrefix:   pathPrefix,
		Description:  cloneOptionalString(description),
		SourceConfig: sourceConfig,
		Outputs:      outputs,
		UserID:       userID,
	}
}

func defaultQueryOutput(name string) SourceOutputInput {
	return SourceOutputInput{Key: "result", DisplayName: name, Selector: SourceOutputSelector{Kind: "whole"}, DataType: "object"}
}

func sourceOutputParamsFromRecords(records []repository.SourceOutputMappingRecord) []repository.SourceOutputMappingParam {
	result := make([]repository.SourceOutputMappingParam, 0, len(records))
	for _, record := range records {
		id := record.ID
		result = append(result, repository.SourceOutputMappingParam{ID: &id, Key: record.Key, DisplayName: record.DisplayName,
			Selector: record.Selector, DataType: record.DataType, Unit: record.Unit, PrecisionNum: record.PrecisionNum,
			SortOrder: record.SortOrder})
	}
	return result
}

func builtinQueryResult(result *BuiltinSQLExecuteResult) *QueryExecutionResult {
	if result == nil {
		return &QueryExecutionResult{Data: []map[string]any{}, Columns: []string{}, ExecutionTime: 0, RowCount: 0, Limits: developmentSQLLimits()}
	}
	rows := make([]map[string]any, 0, len(result.Rows))
	for _, row := range result.Rows {
		next := make(map[string]any, len(row))
		for key, value := range row {
			next[key] = normalizeQueryValue(value)
		}
		rows = append(rows, next)
	}
	return &QueryExecutionResult{
		Data:          rows,
		Columns:       result.Columns,
		ColumnTypes:   result.ColumnTypes,
		ExecutionTime: result.ExecutionTime,
		RowCount:      result.RowCount,
		Truncated:     result.Truncated,
		TruncatedBy:   result.TruncatedBy,
		Limits:        result.Limits,
	}
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
		GroupID:         cloneOptionalString(record.GroupID),
		QueryType:       record.QueryType,
		Config:          cloneMap(record.Config),
		Transformer:     cloneOptionalString(record.Transformer),
		IsEnabled:       record.IsEnabled,
		TimeoutMS:       record.TimeoutMS,
		CacheEnabled:    record.CacheEnabled,
		CacheTtlSeconds: record.CacheTtlSeconds,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
		Outputs:         sourceOutputsFromRecords(record.Outputs),
	}
}

func normalizeQueryName(name string) (string, error) {
	original := name
	name = strings.TrimSpace(name)
	if name == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "查询名称不能为空")
	}
	if original != name {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "查询名称不能包含首尾空格")
	}
	if len([]rune(name)) > 100 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "查询名称长度不能超过 100 个字符")
	}
	if containsControlCharacter(name) {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "查询名称不能包含换行、制表符或其他控制字符")
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

	return normalizeSQLText(sqlText)
}

func normalizeSQLText(sqlText string) (string, error) {
	sqlText = strings.TrimSpace(sqlText)
	if sqlText == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 不能为空")
	}
	if len([]byte(sqlText)) > maxSavedSQLBytes {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 内容不能超过 1 MiB")
	}
	return sqlText, nil
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
	return normalizeSQLValue(value, "")
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
