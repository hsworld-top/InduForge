package service

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

// publicConnectionTypeCategoryMap 只保留通用连接接口允许新建/改型的类型。
// 说明：kafka/http/websocket/redis 与 opcua/modbus/s7/tdengine 均走各自协议专用配置接口。
var publicConnectionTypeCategoryMap = map[string]string{
	"relational": "database",
	"mqtt":       "message",
	"websocket":  "protocol",
	"http":       "api",
}

// reservedPhase2ConnectionTypes 用于阻止工业协议从通用连接入口写入，避免只生成 metadata 而缺失专用配置表。
var reservedPhase2ConnectionTypes = map[string]string{
	"opcua":    "OPC UA",
	"modbus":   "Modbus",
	"s7":       "S7",
	"tdengine": "TDengine",
}

var allowedConnectionStatus = map[string]struct{}{
	"connected":    {},
	"disconnected": {},
	"error":        {},
	"unknown":      {},
}

// Connection 表示面向 HTTP 层返回的连接对象。
type Connection struct {
	ID               string         `json:"id"`
	ProjectID        string         `json:"projectId"`
	TenantID         string         `json:"tenantId"`
	Name             string         `json:"name"`
	Type             string         `json:"type"`
	Status           string         `json:"status"`
	Config           map[string]any `json:"config"`
	RelationalConfig map[string]any `json:"relationalConfig,omitempty"`
	MqttConfig       map[string]any `json:"mqttConfig,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

// ConnectionTestResult 表示连接测试响应。
type ConnectionTestResult struct {
	Connected bool   `json:"connected"`
	DBType    string `json:"dbType"`
	Message   string `json:"message"`
}

// CreateConnectionInput 表示创建连接的业务输入。
type CreateConnectionInput struct {
	Name   string
	Type   string
	Status string
	Config map[string]any
}

// UpdateConnectionInput 表示更新连接的业务输入。
type UpdateConnectionInput struct {
	Name      *string
	Type      *string
	Status    *string
	Config    map[string]any
	HasConfig bool
}

// ConnectionService 承载连接领域的基本业务校验与映射。
type ConnectionService struct {
	repository *repository.ConnectionRepository
}

// NewConnectionService 创建连接服务。
func NewConnectionService(repo *repository.ConnectionRepository) *ConnectionService {
	return &ConnectionService{repository: repo}
}

// ListConnections 查询项目下的连接列表。
func (s *ConnectionService) ListConnections(ctx context.Context, projectID, tenantID string) ([]Connection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}

	records, err := s.repository.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	connections := make([]Connection, 0, len(records))
	for _, record := range records {
		connections = append(connections, toConnection(record, tenantID))
	}

	return connections, nil
}

// CreateConnection 创建项目连接。
func (s *ConnectionService) CreateConnection(ctx context.Context, projectID, tenantID, userID string, input CreateConnectionInput) (*Connection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	name, err := normalizeConnectionName(input.Name)
	if err != nil {
		return nil, err
	}
	connectionType, category, err := normalizeConnectionType(input.Type)
	if err != nil {
		return nil, err
	}
	status, err := normalizeConnectionStatus(input.Status)
	if err != nil {
		return nil, err
	}
	config, err := normalizeConnectionConfig(input.Config)
	if err != nil {
		return nil, err
	}

	record, err := s.repository.Create(ctx, repository.CreateConnectionParams{
		ProjectID: projectID,
		UserID:    userID,
		Name:      name,
		Type:      connectionType,
		Category:  category,
		Status:    status,
		Config:    config,
	})
	if err != nil {
		return nil, err
	}

	connection := toConnection(*record, tenantID)
	return &connection, nil
}

// UpdateConnection 更新已有连接。
func (s *ConnectionService) UpdateConnection(ctx context.Context, projectID, connectionID, tenantID, userID string, input UpdateConnectionInput) (*Connection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	if input.Name == nil && input.Type == nil && input.Status == nil && !input.HasConfig {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "至少需要提供一个待更新字段")
	}

	current, err := s.repository.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}

	nextName := current.Name
	if input.Name != nil {
		nextName, err = normalizeConnectionName(*input.Name)
		if err != nil {
			return nil, err
		}
	}

	nextType := current.Type
	nextCategory := current.Category
	if nextCategory == "" {
		nextCategory = deriveStoredConnectionCategory(current.Type)
	}
	if input.Type != nil {
		nextType, nextCategory, err = normalizeConnectionType(*input.Type)
		if err != nil {
			return nil, err
		}
	}

	nextStatus := current.Status
	if input.Status != nil {
		nextStatus, err = normalizeConnectionStatus(*input.Status)
		if err != nil {
			return nil, err
		}
	}

	nextConfig := current.Config
	if input.HasConfig {
		nextConfig, err = normalizeConnectionConfig(input.Config)
		if err != nil {
			return nil, err
		}
	}

	record, err := s.repository.Update(ctx, repository.UpdateConnectionParams{
		ID:        connectionID,
		ProjectID: projectID,
		UserID:    userID,
		Name:      nextName,
		Type:      nextType,
		Category:  nextCategory,
		Status:    nextStatus,
		Config:    nextConfig,
	})
	if err != nil {
		return nil, err
	}

	connection := toConnection(*record, tenantID)
	return &connection, nil
}

// DeleteConnection 删除项目连接。
func (s *ConnectionService) DeleteConnection(ctx context.Context, projectID, connectionID string) error {
	if err := validateProjectID(projectID); err != nil {
		return err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return err
	}

	return s.repository.Delete(ctx, projectID, connectionID)
}

// TestConnection 使用临时连接配置测试外部关系库连接。
func (s *ConnectionService) TestConnection(ctx context.Context, projectID string, input CreateConnectionInput) (*ConnectionTestResult, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}

	connectionType, _, err := normalizeConnectionType(input.Type)
	if err != nil {
		return nil, err
	}
	if connectionType != "relational" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前仅支持关系型连接测试")
	}

	runtime, err := connectRelationalRuntime(ctx, input.Config)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()

	if err := runtime.Ping(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接测试失败", err)
	}

	runtimeConfig, parseErr := parseRelationalRuntimeConfig(input.Config)
	if parseErr != nil {
		return nil, parseErr
	}

	return &ConnectionTestResult{
		Connected: true,
		DBType:    runtimeConfig.DBType,
		Message:   "数据库连接成功",
	}, nil
}

// UpdateConnectionStatus 更新连接状态。
func (s *ConnectionService) UpdateConnectionStatus(ctx context.Context, projectID, connectionID, status string) (*Connection, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	nextStatus, err := normalizeConnectionStatus(status)
	if err != nil {
		return nil, err
	}

	record, err := s.repository.UpdateStatus(ctx, projectID, connectionID, nextStatus)
	if err != nil {
		return nil, err
	}

	current, err := s.repository.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	current.Status = record.Status
	current.UpdatedAt = record.UpdatedAt
	connection := toConnection(*current, "")
	return &connection, nil
}

// ListTables 返回外部关系库表列表。
func (s *ConnectionService) ListTables(ctx context.Context, projectID, connectionID string) ([]RelationalTable, error) {
	connection, err := s.loadRelationalConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}

	runtime, err := connectRelationalRuntime(ctx, connection.Config)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()

	schema := runtime.SearchPath()
	query, args := relationalListTablesQuery(runtime.DBType(), schema)
	rows, err := runtime.Query(ctx, query, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询表列表失败", err)
	}
	defer rows.Close()

	tables := make([]RelationalTable, 0)
	for rows.Next() {
		table := RelationalTable{}
		if err := rows.Scan(&table.Schema, &table.Name, &table.Type); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取表列表失败", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历表列表失败", err)
	}

	return tables, nil
}

// GetTableStructure 返回指定表结构。
func (s *ConnectionService) GetTableStructure(ctx context.Context, projectID, connectionID, tableName string) (*RelationalTableStructure, error) {
	connection, err := s.loadRelationalConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	normalizedTableName, err := normalizeRuntimeIdentifier(tableName, "tableName")
	if err != nil {
		return nil, err
	}

	runtime, err := connectRelationalRuntime(ctx, connection.Config)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()

	schema := runtime.SearchPath()

	columns, err := s.listTableColumns(ctx, runtime, schema, normalizedTableName)
	if err != nil {
		return nil, err
	}
	indexes, err := s.listTableIndexes(ctx, runtime, schema, normalizedTableName)
	if err != nil {
		return nil, err
	}
	foreignKeys, err := s.listTableForeignKeys(ctx, runtime, schema, normalizedTableName)
	if err != nil {
		return nil, err
	}

	return &RelationalTableStructure{
		Columns:     columns,
		Indexes:     indexes,
		ForeignKeys: foreignKeys,
	}, nil
}

// GetTableData 返回指定表数据预览。
func (s *ConnectionService) GetTableData(ctx context.Context, projectID, connectionID, tableName string, page, limit int) (*RelationalTableData, error) {
	connection, err := s.loadRelationalConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	normalizedTableName, err := normalizeRuntimeIdentifier(tableName, "tableName")
	if err != nil {
		return nil, err
	}

	page, limit = normalizePageAndSize(page, limit, 100, 500)

	runtime, err := connectRelationalRuntime(ctx, connection.Config)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()

	schema := runtime.SearchPath()
	qualifiedTableName := relationalQualifiedTableName(runtime.DBType(), schema, normalizedTableName)

	var total int
	if err := runtime.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", qualifiedTableName)).Scan(&total); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计表数据失败", err)
	}

	rows, err := runtime.Query(ctx, relationalTableDataQuery(runtime.DBType(), qualifiedTableName, page, limit))
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询表数据失败", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取表字段元信息失败", err)
	}

	resultRows := make([][]any, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取表数据失败", err)
		}
		row := make([]any, 0, len(values))
		for _, value := range values {
			row = append(row, normalizeQueryValue(value))
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历表数据失败", err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return &RelationalTableData{
		Columns: columns,
		Rows:    resultRows,
		Pagination: RelationalPagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// ExecuteSQL 执行只读 SQL。
func (s *ConnectionService) ExecuteSQL(ctx context.Context, projectID, connectionID, sqlText string, parameters []any) (*RelationalQueryResult, error) {
	connection, err := s.loadRelationalConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if err := ensureReadOnlySQLText(sqlText); err != nil {
		return nil, err
	}

	runtime, err := connectRelationalRuntime(ctx, connection.Config)
	if err != nil {
		return nil, err
	}
	defer runtime.Close()

	tx, err := runtime.BeginReadOnlyTx(ctx)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "创建只读事务失败", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	startTime := time.Now()
	rows, err := tx.Query(ctx, sqlText, parameters...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "执行 SQL 失败", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 SQL 字段元信息失败", err)
	}

	resultRows := make([][]any, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 SQL 结果失败", err)
		}
		row := make([]any, 0, len(values))
		for _, value := range values {
			row = append(row, normalizeQueryValue(value))
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 SQL 结果失败", err)
	}

	return &RelationalQueryResult{
		Columns:       columns,
		Rows:          resultRows,
		RowCount:      len(resultRows),
		ExecutionTime: time.Since(startTime).Milliseconds(),
	}, nil
}

func (s *ConnectionService) loadRelationalConnection(ctx context.Context, projectID, connectionID string) (*repository.ConnectionRecord, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}

	connection, err := s.repository.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if connection.Type != "relational" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前仅支持关系型连接")
	}
	return connection, nil
}

func normalizeRuntimeIdentifier(value, fieldName string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fieldName+" 不能为空")
	}
	if strings.ContainsAny(value, "\"'; \t\r\n") {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fieldName+" 包含非法字符")
	}
	return value, nil
}

func (s *ConnectionService) listTableColumns(ctx context.Context, runtime *relationalRuntime, schema, tableName string) ([]RelationalTableColumn, error) {
	query, args := relationalTableColumnsQuery(runtime.DBType(), schema, tableName)
	rows, err := runtime.Query(ctx, query, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询表字段失败", err)
	}
	defer rows.Close()

	columns := make([]RelationalTableColumn, 0)
	for rows.Next() {
		column := RelationalTableColumn{}
		var (
			maxLength sql.NullInt64
			precision sql.NullInt64
			scale     sql.NullInt64
			nullable  bool
			comment   sql.NullString
			defValue  sql.NullString
			isPrimary bool
			isUnique  bool
			autoIncr  bool
		)
		if err := rows.Scan(
			&column.Name,
			&column.Type,
			&maxLength,
			&precision,
			&scale,
			&nullable,
			&defValue,
			&comment,
			&isPrimary,
			&isUnique,
			&autoIncr,
		); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取表字段失败", err)
		}
		column.MaxLength = nullInt64ToInt(maxLength)
		column.NumericPrecision = nullInt64ToInt(precision)
		column.NumericScale = nullInt64ToInt(scale)
		column.Nullable = nullable
		column.DefaultValue = nullStringPointer(defValue)
		column.Comment = nullStringPointer(comment)
		column.IsPrimary = isPrimary
		column.IsUnique = isUnique
		column.AutoIncrement = autoIncr
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历表字段失败", err)
	}

	return columns, nil
}

func (s *ConnectionService) listTableIndexes(ctx context.Context, runtime *relationalRuntime, schema, tableName string) ([]RelationalIndex, error) {
	if runtime.DBType() == "postgresql" {
		return s.listPostgresTableIndexes(ctx, runtime, schema, tableName)
	}

	query, args := relationalTableIndexesQuery(runtime.DBType(), schema, tableName)
	rows, err := runtime.Query(ctx, query, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询表索引失败", err)
	}
	defer rows.Close()

	indexMap := make(map[string]*RelationalIndex)
	for rows.Next() {
		var name, indexType, method, columnName string
		if err := rows.Scan(&name, &indexType, &method, &columnName); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取表索引失败", err)
		}

		current, ok := indexMap[name]
		if !ok {
			current = &RelationalIndex{
				Name:    name,
				Type:    indexType,
				Method:  strings.ToLower(method),
				Columns: []string{},
			}
			indexMap[name] = current
		}
		if strings.TrimSpace(columnName) != "" {
			current.Columns = append(current.Columns, columnName)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历表索引失败", err)
	}

	indexes := make([]RelationalIndex, 0, len(indexMap))
	for _, index := range indexMap {
		indexes = append(indexes, *index)
	}
	return indexes, nil
}

func (s *ConnectionService) listTableForeignKeys(ctx context.Context, runtime *relationalRuntime, schema, tableName string) ([]RelationalForeignKey, error) {
	query, args := relationalTableForeignKeysQuery(runtime.DBType(), schema, tableName)
	rows, err := runtime.Query(ctx, query, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询表外键失败", err)
	}
	defer rows.Close()

	foreignKeys := make([]RelationalForeignKey, 0)
	for rows.Next() {
		foreignKey := RelationalForeignKey{}
		if err := rows.Scan(
			&foreignKey.Name,
			&foreignKey.ColumnName,
			&foreignKey.ReferencedTable,
			&foreignKey.ReferencedColumn,
			&foreignKey.UpdateRule,
			&foreignKey.DeleteRule,
		); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取表外键失败", err)
		}
		foreignKeys = append(foreignKeys, foreignKey)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历表外键失败", err)
	}

	return foreignKeys, nil
}

func nullInt64ToInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	result := int(value.Int64)
	return &result
}

func nullStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	text := value.String
	return &text
}

func toConnection(record repository.ConnectionRecord, tenantID string) Connection {
	relationalConfig := map[string]any(nil)
	mqttConfig := map[string]any(nil)
	switch record.Type {
	case "relational":
		relationalConfig = cloneMap(record.Config)
	case "mqtt":
		mqttConfig = cloneMap(record.Config)
	}
	return Connection{
		ID:               record.ID,
		ProjectID:        record.ProjectID,
		TenantID:         tenantID,
		Name:             record.Name,
		Type:             record.Type,
		Status:           record.Status,
		Config:           cloneMap(record.Config),
		RelationalConfig: relationalConfig,
		MqttConfig:       mqttConfig,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
	}
}

func validateProjectID(projectID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(projectID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "projectId 格式无效", err)
	}
	return nil
}

func validateConnectionID(connectionID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(connectionID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "connectionId 格式无效", err)
	}
	return nil
}

func validateUserID(userID string) error {
	if _, err := uuid.Parse(strings.TrimSpace(userID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "用户标识格式无效", err)
	}
	return nil
}

func normalizeConnectionName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接名称不能为空")
	}
	if len([]rune(name)) > 100 {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接名称长度不能超过 100 个字符")
	}
	return name, nil
}

func normalizeConnectionType(connectionType string) (string, string, error) {
	connectionType = strings.TrimSpace(strings.ToLower(connectionType))
	if displayName, ok := reservedPhase2ConnectionTypes[connectionType]; ok {
		return "", "", newPhaseBoundaryProtocolError(displayName)
	}
	category, ok := publicConnectionTypeCategoryMap[connectionType]
	if !ok {
		return "", "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接类型不受支持")
	}
	return connectionType, category, nil
}

func deriveStoredConnectionCategory(connectionType string) string {
	switch strings.TrimSpace(strings.ToLower(connectionType)) {
	case "relational":
		return "database"
	case "mqtt":
		return "message"
	case "kafka", "http", "websocket", "redis", "opcua", "modbus", "s7", "tdengine":
		return "protocol"
	default:
		return ""
	}
}

func normalizeConnectionStatus(status string) (string, error) {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "unknown", nil
	}
	if _, ok := allowedConnectionStatus[status]; !ok {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接状态不受支持")
	}
	return status, nil
}

func normalizeConnectionConfig(config map[string]any) (map[string]any, error) {
	if config == nil {
		return map[string]any{}, nil
	}
	return cloneMap(config), nil
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}

	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func (s *ConnectionService) listPostgresTableIndexes(ctx context.Context, runtime *relationalRuntime, schema, tableName string) ([]RelationalIndex, error) {
	rows, err := runtime.Query(ctx, `
        SELECT indexname, indexdef
        FROM pg_indexes
        WHERE schemaname = $1
          AND tablename = $2
        ORDER BY indexname
    `, schema, tableName)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询表索引失败", err)
	}
	defer rows.Close()

	indexes := make([]RelationalIndex, 0)
	for rows.Next() {
		var (
			name     string
			indexDef string
		)
		if err := rows.Scan(&name, &indexDef); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取表索引失败", err)
		}

		indexType := "INDEX"
		if strings.Contains(indexDef, "PRIMARY KEY") {
			indexType = "PRIMARY"
		} else if strings.Contains(indexDef, "UNIQUE INDEX") {
			indexType = "UNIQUE"
		}

		method := "btree"
		if methodMatch := regexp.MustCompile(`USING ([a-zA-Z0-9_]+)`).FindStringSubmatch(indexDef); len(methodMatch) == 2 {
			method = strings.ToLower(methodMatch[1])
		}

		columnPart := regexp.MustCompile(`\((.*)\)`).FindStringSubmatch(indexDef)
		columns := []string{}
		if len(columnPart) == 2 {
			for _, item := range strings.Split(columnPart[1], ",") {
				columnName := strings.Trim(strings.TrimSpace(item), "\"")
				if columnName != "" {
					columns = append(columns, columnName)
				}
			}
		}

		indexes = append(indexes, RelationalIndex{
			Name:    name,
			Type:    indexType,
			Method:  method,
			Columns: columns,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历表索引失败", err)
	}

	return indexes, nil
}

func relationalListTablesQuery(dbType, schema string) (string, []any) {
	switch dbType {
	case "mysql", "sqlserver":
		return `
        SELECT TABLE_SCHEMA, TABLE_NAME, TABLE_TYPE
        FROM information_schema.tables
        WHERE TABLE_SCHEMA = ?
          AND TABLE_TYPE IN ('BASE TABLE', 'VIEW')
        ORDER BY TABLE_NAME
    `, []any{schema}
	default:
		return `
        SELECT table_schema, table_name, table_type
        FROM information_schema.tables
        WHERE table_schema = $1
          AND table_type IN ('BASE TABLE', 'VIEW')
        ORDER BY table_name
    `, []any{schema}
	}
}

func relationalQualifiedTableName(dbType, schema, tableName string) string {
	switch dbType {
	case "mysql":
		return fmt.Sprintf("`%s`.`%s`", escapeMySQLIdentifier(schema), escapeMySQLIdentifier(tableName))
	case "sqlserver":
		return fmt.Sprintf("[%s].[%s]", escapeSQLServerIdentifier(schema), escapeSQLServerIdentifier(tableName))
	default:
		return pgx.Identifier{schema, tableName}.Sanitize()
	}
}

func relationalTableDataQuery(dbType, qualifiedTableName string, page, limit int) string {
	offset := (page - 1) * limit
	switch dbType {
	case "sqlserver":
		return fmt.Sprintf(
			"SELECT * FROM %s ORDER BY (SELECT NULL) OFFSET %d ROWS FETCH NEXT %d ROWS ONLY",
			qualifiedTableName,
			offset,
			limit,
		)
	default:
		return fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d", qualifiedTableName, limit, offset)
	}
}

func relationalTableColumnsQuery(dbType, schema, tableName string) (string, []any) {
	switch dbType {
	case "mysql":
		return `
        SELECT
            COLUMN_NAME,
            COLUMN_TYPE,
            CHARACTER_MAXIMUM_LENGTH,
            NUMERIC_PRECISION,
            NUMERIC_SCALE,
            CASE WHEN IS_NULLABLE = 'YES' THEN TRUE ELSE FALSE END AS nullable,
            COLUMN_DEFAULT,
            COLUMN_COMMENT,
            CASE WHEN COLUMN_KEY = 'PRI' THEN TRUE ELSE FALSE END AS is_primary,
            CASE WHEN COLUMN_KEY = 'UNI' THEN TRUE ELSE FALSE END AS is_unique,
            CASE WHEN EXTRA LIKE '%auto_increment%' THEN TRUE ELSE FALSE END AS auto_increment
        FROM information_schema.columns
        WHERE TABLE_SCHEMA = ?
          AND TABLE_NAME = ?
        ORDER BY ORDINAL_POSITION
    `, []any{schema, tableName}
	case "sqlserver":
		return `
        SELECT
            c.COLUMN_NAME,
            c.DATA_TYPE,
            c.CHARACTER_MAXIMUM_LENGTH,
            c.NUMERIC_PRECISION,
            c.NUMERIC_SCALE,
            CASE WHEN c.IS_NULLABLE = 'YES' THEN CAST(1 AS bit) ELSE CAST(0 AS bit) END AS nullable,
            CAST(c.COLUMN_DEFAULT AS nvarchar(4000)) AS column_default,
            CAST(ep.value AS nvarchar(4000)) AS column_comment,
            CASE WHEN pk.COLUMN_NAME IS NOT NULL THEN CAST(1 AS bit) ELSE CAST(0 AS bit) END AS is_primary,
            CASE WHEN uq.COLUMN_NAME IS NOT NULL THEN CAST(1 AS bit) ELSE CAST(0 AS bit) END AS is_unique,
            CAST(COLUMNPROPERTY(OBJECT_ID(QUOTENAME(c.TABLE_SCHEMA) + '.' + QUOTENAME(c.TABLE_NAME)), c.COLUMN_NAME, 'IsIdentity') AS bit) AS auto_increment
        FROM INFORMATION_SCHEMA.COLUMNS c
        LEFT JOIN (
            SELECT ku.TABLE_SCHEMA, ku.TABLE_NAME, ku.COLUMN_NAME
            FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
            JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku
              ON tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME
             AND tc.TABLE_SCHEMA = ku.TABLE_SCHEMA
             AND tc.TABLE_NAME = ku.TABLE_NAME
            WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
        ) pk
          ON pk.TABLE_SCHEMA = c.TABLE_SCHEMA
         AND pk.TABLE_NAME = c.TABLE_NAME
         AND pk.COLUMN_NAME = c.COLUMN_NAME
        LEFT JOIN (
            SELECT ku.TABLE_SCHEMA, ku.TABLE_NAME, ku.COLUMN_NAME
            FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
            JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku
              ON tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME
             AND tc.TABLE_SCHEMA = ku.TABLE_SCHEMA
             AND tc.TABLE_NAME = ku.TABLE_NAME
            WHERE tc.CONSTRAINT_TYPE = 'UNIQUE'
        ) uq
          ON uq.TABLE_SCHEMA = c.TABLE_SCHEMA
         AND uq.TABLE_NAME = c.TABLE_NAME
         AND uq.COLUMN_NAME = c.COLUMN_NAME
        LEFT JOIN sys.extended_properties ep
          ON ep.major_id = OBJECT_ID(QUOTENAME(c.TABLE_SCHEMA) + '.' + QUOTENAME(c.TABLE_NAME))
         AND ep.minor_id = COLUMNPROPERTY(OBJECT_ID(QUOTENAME(c.TABLE_SCHEMA) + '.' + QUOTENAME(c.TABLE_NAME)), c.COLUMN_NAME, 'ColumnId')
         AND ep.name = 'MS_Description'
        WHERE c.TABLE_SCHEMA = @schema
          AND c.TABLE_NAME = @table
        ORDER BY c.ORDINAL_POSITION
    `, []any{schema, tableName}
	default:
		return `
        SELECT
            c.column_name,
            c.data_type,
            c.character_maximum_length,
            c.numeric_precision,
            c.numeric_scale,
            c.is_nullable = 'YES' AS nullable,
            c.column_default,
            pgd.description,
            EXISTS (
                SELECT 1
                FROM information_schema.table_constraints tc
                JOIN information_schema.key_column_usage kcu
                  ON tc.constraint_name = kcu.constraint_name
                 AND tc.table_schema = kcu.table_schema
                 AND tc.table_name = kcu.table_name
                WHERE tc.table_schema = c.table_schema
                  AND tc.table_name = c.table_name
                  AND tc.constraint_type = 'PRIMARY KEY'
                  AND kcu.column_name = c.column_name
            ) AS is_primary,
            EXISTS (
                SELECT 1
                FROM information_schema.table_constraints tc
                JOIN information_schema.key_column_usage kcu
                  ON tc.constraint_name = kcu.constraint_name
                 AND tc.table_schema = kcu.table_schema
                 AND tc.table_name = kcu.table_name
                WHERE tc.table_schema = c.table_schema
                  AND tc.table_name = c.table_name
                  AND tc.constraint_type = 'UNIQUE'
                  AND kcu.column_name = c.column_name
            ) AS is_unique,
            POSITION('nextval(' IN COALESCE(c.column_default, '')) > 0 AS auto_increment
        FROM information_schema.columns c
        LEFT JOIN pg_catalog.pg_statio_all_tables st
          ON st.relname = c.table_name
        LEFT JOIN pg_catalog.pg_description pgd
          ON pgd.objoid = st.relid
         AND pgd.objsubid = c.ordinal_position
        WHERE c.table_schema = $1
          AND c.table_name = $2
        ORDER BY c.ordinal_position
    `, []any{schema, tableName}
	}
}

func relationalTableIndexesQuery(dbType, schema, tableName string) (string, []any) {
	switch dbType {
	case "mysql":
		return `
        SELECT
            INDEX_NAME,
            CASE
                WHEN INDEX_NAME = 'PRIMARY' THEN 'PRIMARY'
                WHEN NON_UNIQUE = 0 THEN 'UNIQUE'
                ELSE 'INDEX'
            END AS index_type,
            INDEX_TYPE,
            COLUMN_NAME
        FROM information_schema.statistics
        WHERE TABLE_SCHEMA = ?
          AND TABLE_NAME = ?
        ORDER BY INDEX_NAME, SEQ_IN_INDEX
    `, []any{schema, tableName}
	case "sqlserver":
		return `
        SELECT
            i.name,
            CASE
                WHEN i.is_primary_key = 1 THEN 'PRIMARY'
                WHEN i.is_unique = 1 THEN 'UNIQUE'
                ELSE 'INDEX'
            END AS index_type,
            i.type_desc,
            c.name
        FROM sys.indexes i
        JOIN sys.index_columns ic
          ON i.object_id = ic.object_id
         AND i.index_id = ic.index_id
        JOIN sys.columns c
          ON ic.object_id = c.object_id
         AND ic.column_id = c.column_id
        JOIN sys.objects o
          ON i.object_id = o.object_id
        JOIN sys.schemas s
          ON o.schema_id = s.schema_id
        WHERE s.name = @schema
          AND o.name = @table
          AND i.name IS NOT NULL
        ORDER BY i.name, ic.key_ordinal
    `, []any{schema, tableName}
	default:
		return "", nil
	}
}

func relationalTableForeignKeysQuery(dbType, schema, tableName string) (string, []any) {
	switch dbType {
	case "mysql":
		return `
        SELECT
            kcu.CONSTRAINT_NAME,
            kcu.COLUMN_NAME,
            kcu.REFERENCED_TABLE_NAME,
            kcu.REFERENCED_COLUMN_NAME,
            rc.UPDATE_RULE,
            rc.DELETE_RULE
        FROM information_schema.KEY_COLUMN_USAGE kcu
        JOIN information_schema.REFERENTIAL_CONSTRAINTS rc
          ON rc.CONSTRAINT_SCHEMA = kcu.CONSTRAINT_SCHEMA
         AND rc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
        WHERE kcu.TABLE_SCHEMA = ?
          AND kcu.TABLE_NAME = ?
          AND kcu.REFERENCED_TABLE_NAME IS NOT NULL
        ORDER BY kcu.CONSTRAINT_NAME, kcu.ORDINAL_POSITION
    `, []any{schema, tableName}
	case "sqlserver":
		return `
        SELECT
            fk.name,
            pc.name AS column_name,
            ro.name AS referenced_table,
            rc.name AS referenced_column,
            fk.update_referential_action_desc,
            fk.delete_referential_action_desc
        FROM sys.foreign_keys fk
        JOIN sys.foreign_key_columns fkc
          ON fk.object_id = fkc.constraint_object_id
        JOIN sys.objects po
          ON fk.parent_object_id = po.object_id
        JOIN sys.schemas ps
          ON po.schema_id = ps.schema_id
        JOIN sys.columns pc
          ON fkc.parent_object_id = pc.object_id
         AND fkc.parent_column_id = pc.column_id
        JOIN sys.objects ro
          ON fk.referenced_object_id = ro.object_id
        JOIN sys.columns rc
          ON fkc.referenced_object_id = rc.object_id
         AND fkc.referenced_column_id = rc.column_id
        WHERE ps.name = @schema
          AND po.name = @table
        ORDER BY fk.name, fkc.constraint_column_id
    `, []any{schema, tableName}
	default:
		return `
        SELECT
            tc.constraint_name,
            kcu.column_name,
            ccu.table_name AS referenced_table,
            ccu.column_name AS referenced_column,
            rc.update_rule,
            rc.delete_rule
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu
          ON tc.constraint_name = kcu.constraint_name
         AND tc.table_schema = kcu.table_schema
        JOIN information_schema.constraint_column_usage ccu
          ON ccu.constraint_name = tc.constraint_name
         AND ccu.table_schema = tc.table_schema
        JOIN information_schema.referential_constraints rc
          ON rc.constraint_name = tc.constraint_name
         AND rc.constraint_schema = tc.table_schema
        WHERE tc.table_schema = $1
          AND tc.table_name = $2
          AND tc.constraint_type = 'FOREIGN KEY'
        ORDER BY tc.constraint_name, kcu.ordinal_position
    `, []any{schema, tableName}
	}
}

func escapeMySQLIdentifier(value string) string {
	return strings.ReplaceAll(value, "`", "``")
}

func escapeSQLServerIdentifier(value string) string {
	return strings.ReplaceAll(value, "]", "]]")
}
