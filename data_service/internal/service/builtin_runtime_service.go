package service

import (
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

const (
	defaultBuiltinSQLLimit = 100
	maxBuiltinSQLLimit     = 500
)

type builtinRuntimeTx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Commit(context.Context) error
	Rollback(context.Context) error
}

// BuiltinRuntimeService 承载四类 IF 内置运行库的开发态测试能力。
type BuiltinRuntimeService struct {
	devPool           *pgxpool.Pool
	metaPool          *pgxpool.Pool
	realtimeClient    redis.UniversalClient
	realtimeKeyPrefix string
}

type BuiltinRuntimeOptions struct {
	DevPool           *pgxpool.Pool
	MetaPool          *pgxpool.Pool
	RealtimeClient    redis.UniversalClient
	RealtimeKeyPrefix string
}

// wrapBuiltinSQLExecutionError 保留工作台用户可操作的 PostgreSQL 错误信息，
// 同时继续包装原始错误供服务端日志和 errors.Is/errors.As 使用。
func wrapBuiltinSQLExecutionError(err error) error {
	message := "执行内置运行库 SQL 失败"
	var pgErr *pgconn.PgError
	if stderrors.As(err, &pgErr) {
		detail := strings.TrimSpace(pgErr.Message)
		if pgErr.Position > 0 {
			detail = fmt.Sprintf("%s（位置 %d）", detail, pgErr.Position)
		}
		if value := strings.TrimSpace(pgErr.Detail); value != "" {
			detail += "；" + value
		}
		if value := strings.TrimSpace(pgErr.Hint); value != "" {
			detail += "；提示：" + value
		}
		if detail != "" {
			message += "：" + detail
		}
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message, err)
}

func NewBuiltinRuntimeService(options BuiltinRuntimeOptions) *BuiltinRuntimeService {
	realtimeKeyPrefix := strings.Trim(strings.TrimSpace(options.RealtimeKeyPrefix), ":")
	if realtimeKeyPrefix == "" {
		realtimeKeyPrefix = "ifdev"
	}
	return &BuiltinRuntimeService{
		devPool:           options.DevPool,
		metaPool:          options.MetaPool,
		realtimeClient:    options.RealtimeClient,
		realtimeKeyPrefix: realtimeKeyPrefix,
	}
}

type BuiltinSQLExecuteInput struct {
	Store      string
	SQL        string
	Parameters []any
	Limit      int
}

type BuiltinSQLExecuteResult struct {
	Columns       []string          `json:"columns"`
	ColumnTypes   map[string]string `json:"columnTypes"`
	Rows          []map[string]any  `json:"rows"`
	RowCount      int               `json:"rowCount"`
	ExecutionTime int64             `json:"executionTime"`
	Truncated     bool              `json:"truncated"`
	TruncatedBy   string            `json:"truncatedBy,omitempty"`
	Limits        SQLResultLimits   `json:"limits"`
}

type BuiltinTimeseriesSampleInput struct {
	Table string
	Point string
	Value any
	TS    *time.Time
}

type BuiltinRealtimeSetInput struct {
	Key        string
	Value      any
	TtlSeconds int
	RuntimeKey string
}

type BuiltinRealtimeValue struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value any    `json:"value"`
	TTL   int64  `json:"ttl"`
}

type BuiltinRealtimeKeyDefinition struct {
	ID                string    `json:"id"`
	Key               string    `json:"key"`
	ValueType         string    `json:"valueType"`
	DefaultTtlSeconds int       `json:"defaultTtlSeconds"`
	Description       string    `json:"description"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (s *BuiltinRuntimeService) ExecuteSQL(ctx context.Context, projectID string, input BuiltinSQLExecuteInput) (*BuiltinSQLExecuteResult, error) {
	execCtx, cancel := context.WithTimeout(ctx, developmentSQLTimeout*time.Second)
	defer cancel()
	ctx = execCtx
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if s == nil || s.devPool == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态关系/时序运行库未初始化")
	}
	sqlText := strings.TrimSpace(input.SQL)
	if sqlText == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 不能为空")
	}
	if err := validateBuiltinSQL(sqlText); err != nil {
		return nil, err
	}

	schemaSuffix, err := builtinSQLSchemaSuffix(input.Store)
	if err != nil {
		return nil, err
	}
	schemaName := deriveBuiltinProjectSchema(projectID, schemaSuffix)
	if err := s.ensureSchema(ctx, schemaName); err != nil {
		return nil, err
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultBuiltinSQLLimit
	}
	if limit > maxBuiltinSQLLimit {
		limit = maxBuiltinSQLLimit
	}

	startedAt := time.Now()
	tx, err := s.devPool.Begin(ctx)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启内置运行库 SQL 事务失败", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %s, public", pgx.Identifier{schemaName}.Sanitize())); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "设置内置运行库 schema 失败", err)
	}

	if isBuiltinQuerySQL(sqlText) {
		rows, err := tx.Query(ctx, sqlText, input.Parameters...)
		if err != nil {
			return nil, wrapBuiltinSQLExecutionError(err)
		}
		defer rows.Close()
		result, err := collectBuiltinSQLRows(rows, limit)
		if err != nil {
			return nil, err
		}
		// 截断结果时仍可能有未读取的数据，提交事务前必须关闭游标释放连接。
		rows.Close()
		result.ExecutionTime = time.Since(startedAt).Milliseconds()
		if err := tx.Commit(ctx); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交内置运行库 SQL 事务失败", err)
		}
		return result, nil
	}

	commandTag, err := tx.Exec(ctx, sqlText, input.Parameters...)
	if err != nil {
		return nil, wrapBuiltinSQLExecutionError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交内置运行库 SQL 事务失败", err)
	}
	return &BuiltinSQLExecuteResult{
		Columns:       []string{},
		ColumnTypes:   map[string]string{},
		Rows:          []map[string]any{},
		RowCount:      int(commandTag.RowsAffected()),
		ExecutionTime: time.Since(startedAt).Milliseconds(),
		Limits:        developmentSQLLimits(),
	}, nil
}

func (s *BuiltinRuntimeService) ExecuteSQLInSchema(ctx context.Context, schemaName string, sqlText string, parameters []any, limit int) (*BuiltinSQLExecuteResult, error) {
	execCtx, cancel := context.WithTimeout(ctx, developmentSQLTimeout*time.Second)
	defer cancel()
	ctx = execCtx
	if s == nil || s.devPool == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态关系/时序运行库未初始化")
	}
	sqlText = strings.TrimSpace(sqlText)
	if sqlText == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 不能为空")
	}
	if err := validateBuiltinSQL(sqlText); err != nil {
		return nil, err
	}
	schemaName = sanitizeBuiltinIdentifier(schemaName, "")
	if schemaName == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库 schema 无效")
	}
	if err := s.ensureSchema(ctx, schemaName); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultBuiltinSQLLimit
	}
	if limit > maxBuiltinSQLLimit {
		limit = maxBuiltinSQLLimit
	}

	startedAt := time.Now()
	tx, err := s.devPool.Begin(ctx)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启内置运行库 SQL 事务失败", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %s, public", pgx.Identifier{schemaName}.Sanitize())); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "设置内置运行库 schema 失败", err)
	}
	if isBuiltinQuerySQL(sqlText) {
		rows, err := tx.Query(ctx, sqlText, parameters...)
		if err != nil {
			return nil, wrapBuiltinSQLExecutionError(err)
		}
		defer rows.Close()
		result, err := collectBuiltinSQLRows(rows, limit)
		if err != nil {
			return nil, err
		}
		// 截断结果时仍可能有未读取的数据，提交事务前必须关闭游标释放连接。
		rows.Close()
		result.ExecutionTime = time.Since(startedAt).Milliseconds()
		if err := tx.Commit(ctx); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交内置运行库 SQL 事务失败", err)
		}
		return result, nil
	}
	commandTag, err := tx.Exec(ctx, sqlText, parameters...)
	if err != nil {
		return nil, wrapBuiltinSQLExecutionError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交内置运行库 SQL 事务失败", err)
	}
	return &BuiltinSQLExecuteResult{
		Columns:       []string{},
		ColumnTypes:   map[string]string{},
		Rows:          []map[string]any{},
		RowCount:      int(commandTag.RowsAffected()),
		ExecutionTime: time.Since(startedAt).Milliseconds(),
		Limits:        developmentSQLLimits(),
	}, nil
}

func (s *BuiltinRuntimeService) ListTablesInSchema(ctx context.Context, schemaName string) ([]RelationalTable, error) {
	if s == nil || s.devPool == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态关系/时序运行库未初始化")
	}
	schemaName = sanitizeBuiltinIdentifier(schemaName, "")
	if schemaName == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库 schema 无效")
	}
	if err := s.ensureSchema(ctx, schemaName); err != nil {
		return nil, err
	}
	rows, err := s.devPool.Query(ctx, `
		SELECT t.table_schema, t.table_name, t.table_type, COALESCE(m.kind, ''), COALESCE(m.timeseries, '{}'::jsonb)
		FROM information_schema.tables t
		LEFT JOIN if_table_metadata m
		  ON m.schema_name = t.table_schema
		 AND m.table_name = t.table_name
		WHERE t.table_schema = $1
		  AND t.table_type IN ('BASE TABLE', 'VIEW')
		  AND t.table_name <> 'if_table_metadata'
		ORDER BY t.table_name
	`, schemaName)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询内置运行库表列表失败", err)
	}
	defer rows.Close()
	tables := make([]RelationalTable, 0)
	for rows.Next() {
		table := RelationalTable{}
		var storedKind string
		var timeseriesBytes []byte
		if err := rows.Scan(&table.Schema, &table.Name, &table.Type, &storedKind, &timeseriesBytes); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取内置运行库表列表失败", err)
		}
		table.Kind = relationalTableKind("builtin.relation", "postgresql", table.Type, table.Name)
		if storedKind != "" {
			table.Kind = storedKind
		}
		table.Timeseries = decodeRelationalTimeseriesMetadata(timeseriesBytes)
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历内置运行库表列表失败", err)
	}
	return tables, nil
}

func (s *BuiltinRuntimeService) CreateTableInSchema(ctx context.Context, schemaName string, input CreateRelationalTableInput) error {
	if s == nil || s.devPool == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态关系/时序运行库未初始化")
	}
	schemaName = sanitizeBuiltinIdentifier(schemaName, "")
	if schemaName == "" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库 schema 无效")
	}
	if err := s.ensureSchema(ctx, schemaName); err != nil {
		return err
	}
	statements, err := buildBuiltinCreateTableStatements(schemaName, input)
	if err != nil {
		return err
	}
	tx, err := s.devPool.Begin(ctx)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启内置运行库建表事务失败", err)
	}
	defer rollbackBuiltinTxQuietly(ctx, tx)
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "创建内置运行库表失败", err)
		}
	}
	timeseriesPayload, err := json.Marshal(timeseriesMetadataFromCreateInput(input))
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "序列化时序表元数据失败", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO if_table_metadata (schema_name, table_name, kind, timeseries)
		VALUES ($1, $2, $3, $4::jsonb)
		ON CONFLICT (schema_name, table_name)
		DO UPDATE SET kind = EXCLUDED.kind, timeseries = EXCLUDED.timeseries, updated_at = now()
	`, schemaName, input.Name, input.Kind, string(timeseriesPayload)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "记录内置运行库表元数据失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交内置运行库建表事务失败", err)
	}
	return nil
}

func buildBuiltinCreateTableStatements(schemaName string, input CreateRelationalTableInput) ([]string, error) {
	if input.Kind == "hypertable" {
		return buildPostgresCreateHypertableStatements(schemaName, input)
	}
	ddl, err := buildCreateTableDDL("postgresql", schemaName, input)
	if err != nil {
		return nil, err
	}
	return []string{ddl}, nil
}

func timeseriesMetadataFromCreateInput(input CreateRelationalTableInput) *RelationalTimeseriesMetadata {
	if input.Kind != "hypertable" || input.Timeseries == nil {
		return nil
	}
	dimensions := make([]string, 0, len(input.Timeseries.DimensionColumns))
	for _, column := range input.Timeseries.DimensionColumns {
		dimensions = append(dimensions, column.Name)
	}
	return &RelationalTimeseriesMetadata{
		TimeColumn:       input.Timeseries.TimeColumn,
		DimensionColumns: dimensions,
		ChunkInterval:    normalizeTimeseriesInterval(input.Timeseries.ChunkInterval, "1 day"),
		RetentionDays:    normalizeRetentionDays(input.Timeseries.RetentionDays),
	}
}

func decodeRelationalTimeseriesMetadata(payload []byte) *RelationalTimeseriesMetadata {
	if len(payload) == 0 || string(payload) == "{}" || string(payload) == "null" {
		return nil
	}
	var metadata RelationalTimeseriesMetadata
	if err := json.Unmarshal(payload, &metadata); err != nil {
		return nil
	}
	if metadata.TimeColumn == "" {
		return nil
	}
	return &metadata
}

func (s *BuiltinRuntimeService) getTableTimeseriesMetadata(ctx context.Context, schemaName, tableName string) (*RelationalTimeseriesMetadata, error) {
	var payload []byte
	err := s.devPool.QueryRow(ctx, `
		SELECT COALESCE(timeseries, '{}'::jsonb)
		FROM if_table_metadata
		WHERE schema_name = $1 AND table_name = $2
	`, schemaName, tableName).Scan(&payload)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取时序表元数据失败", err)
	}
	return decodeRelationalTimeseriesMetadata(payload), nil
}

func (s *BuiltinRuntimeService) GetTableStructureInSchema(ctx context.Context, schemaName, tableName string) (*RelationalTableStructure, error) {
	if s == nil || s.devPool == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态关系/时序运行库未初始化")
	}
	schemaName = sanitizeBuiltinIdentifier(schemaName, "")
	tableName = sanitizeBuiltinIdentifier(tableName, "")
	if schemaName == "" || tableName == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库表名无效")
	}
	rows, err := s.devPool.Query(ctx, `
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
		  ON st.schemaname = c.table_schema
		 AND st.relname = c.table_name
		LEFT JOIN pg_catalog.pg_description pgd
		  ON pgd.objoid = st.relid
		 AND pgd.objsubid = c.ordinal_position
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`, schemaName, tableName)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询内置运行库表结构失败", err)
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
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取内置运行库表结构失败", err)
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
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历内置运行库表结构失败", err)
	}
	indexes, err := s.listPostgresIndexesInSchema(ctx, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	timeseries, err := s.getTableTimeseriesMetadata(ctx, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	return &RelationalTableStructure{Columns: columns, Indexes: indexes, ForeignKeys: []RelationalForeignKey{}, Timeseries: timeseries}, nil
}

func (s *BuiltinRuntimeService) UpdateTableStructureInSchema(ctx context.Context, schemaName, tableName string, input UpdateRelationalTableInput) (*RelationalTableStructure, error) {
	if s == nil || s.devPool == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态关系/时序运行库未初始化")
	}
	schemaName = sanitizeBuiltinIdentifier(schemaName, "")
	tableName = sanitizeBuiltinIdentifier(tableName, "")
	if schemaName == "" || tableName == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库表名无效")
	}
	current, err := s.GetTableStructureInSchema(ctx, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	statements, err := buildPostgresUpdateTableDDL(schemaName, tableName, current, input)
	if err != nil {
		return nil, err
	}
	tx, err := s.devPool.Begin(ctx)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启内置运行库表结构事务失败", err)
	}
	defer rollbackBuiltinTxQuietly(ctx, tx)
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "修改内置运行库表结构失败", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE if_table_metadata
		SET updated_at = now()
		WHERE schema_name = $1 AND table_name = $2
	`, schemaName, tableName); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新内置运行库表元数据失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交内置运行库表结构事务失败", err)
	}
	return s.GetTableStructureInSchema(ctx, schemaName, tableName)
}

func (s *BuiltinRuntimeService) listPostgresIndexesInSchema(ctx context.Context, schemaName, tableName string) ([]RelationalIndex, error) {
	rows, err := s.devPool.Query(ctx, `
		SELECT indexname, indexdef
		FROM pg_indexes
		WHERE schemaname = $1
		  AND tablename = $2
		ORDER BY indexname
	`, schemaName, tableName)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询内置运行库表索引失败", err)
	}
	defer rows.Close()

	indexes := make([]RelationalIndex, 0)
	for rows.Next() {
		var name, indexDef string
		if err := rows.Scan(&name, &indexDef); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取内置运行库表索引失败", err)
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
				columnName := strings.Trim(strings.TrimSpace(item), `"`)
				if columnName != "" {
					columns = append(columns, columnName)
				}
			}
		}
		indexes = append(indexes, RelationalIndex{Name: name, Type: indexType, Method: method, Columns: columns})
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历内置运行库表索引失败", err)
	}
	return indexes, nil
}

func (s *BuiltinRuntimeService) RenameTableInSchema(ctx context.Context, schemaName, oldName, newName string) error {
	if s == nil || s.devPool == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态关系/时序运行库未初始化")
	}
	schemaName = sanitizeBuiltinIdentifier(schemaName, "")
	oldName = sanitizeBuiltinIdentifier(oldName, "")
	newName = sanitizeBuiltinIdentifier(newName, "")
	if schemaName == "" || oldName == "" || newName == "" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库表名无效")
	}
	if _, err := s.devPool.Exec(ctx, fmt.Sprintf(
		"ALTER TABLE %s RENAME TO %s",
		pgx.Identifier{schemaName, oldName}.Sanitize(),
		pgx.Identifier{newName}.Sanitize(),
	)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "重命名内置运行库表失败", err)
	}
	if _, err := s.devPool.Exec(ctx, `
		UPDATE if_table_metadata
		SET table_name = $3, updated_at = now()
		WHERE schema_name = $1 AND table_name = $2
	`, schemaName, oldName, newName); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新内置运行库表元数据失败", err)
	}
	return nil
}

func (s *BuiltinRuntimeService) DeleteTableInSchema(ctx context.Context, schemaName, tableName string) error {
	if s == nil || s.devPool == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态关系/时序运行库未初始化")
	}
	schemaName = sanitizeBuiltinIdentifier(schemaName, "")
	tableName = sanitizeBuiltinIdentifier(tableName, "")
	if schemaName == "" || tableName == "" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库表名无效")
	}
	if _, err := s.devPool.Exec(ctx, fmt.Sprintf(
		"DROP TABLE %s",
		pgx.Identifier{schemaName, tableName}.Sanitize(),
	)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "删除内置运行库表失败", err)
	}
	if _, err := s.devPool.Exec(ctx, `
		DELETE FROM if_table_metadata
		WHERE schema_name = $1 AND table_name = $2
	`, schemaName, tableName); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除内置运行库表元数据失败", err)
	}
	return nil
}

func rollbackBuiltinTxQuietly(ctx context.Context, tx builtinRuntimeTx) {
	if tx != nil {
		_ = tx.Rollback(ctx)
	}
}

func (s *BuiltinRuntimeService) WriteTimeseriesSample(ctx context.Context, projectID string, input BuiltinTimeseriesSampleInput) (*BuiltinSQLExecuteResult, error) {
	tableName := sanitizeBuiltinIdentifier(input.Table, "point_samples")
	point := strings.TrimSpace(input.Point)
	if point == "" {
		point = "demo.point"
	}
	ts := time.Now().UTC()
	if input.TS != nil && !input.TS.IsZero() {
		ts = input.TS.UTC()
	}
	createSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (ts timestamptz NOT NULL, point text NOT NULL, value jsonb NOT NULL)`, pgx.Identifier{tableName}.Sanitize())
	if _, err := s.ExecuteSQL(ctx, projectID, BuiltinSQLExecuteInput{Store: "timeseries", SQL: createSQL}); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(input.Value)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "样例值必须可序列化为 JSON", err)
	}
	insertSQL := fmt.Sprintf(`INSERT INTO %s (ts, point, value) VALUES ($1, $2, $3::jsonb)`, pgx.Identifier{tableName}.Sanitize())
	return s.ExecuteSQL(ctx, projectID, BuiltinSQLExecuteInput{
		Store:      "timeseries",
		SQL:        insertSQL,
		Parameters: []any{ts, point, string(payload)},
	})
}

func (s *BuiltinRuntimeService) SetRealtimeKey(ctx context.Context, projectID string, input BuiltinRealtimeSetInput) (*BuiltinRealtimeValue, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if s == nil || s.realtimeClient == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态实时运行库未初始化")
	}
	key, err := deriveBuiltinRealtimeKey(s.realtimeKeyPrefix, projectID, input.RuntimeKey, input.Key)
	if err != nil {
		return nil, err
	}
	valueBytes, err := json.Marshal(input.Value)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时库 value 必须可序列化为 JSON", err)
	}
	ttl := time.Duration(input.TtlSeconds) * time.Second
	if input.TtlSeconds < 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TTL 不能小于 0")
	}
	if err := s.realtimeClient.Set(ctx, key, string(valueBytes), ttl).Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入实时库 key 失败", err)
	}
	if _, err := s.UpsertRealtimeKeyDefinition(ctx, projectID, input.RuntimeKey, input.Key, "json", input.TtlSeconds, ""); err != nil {
		return nil, err
	}
	return s.GetRealtimeKey(ctx, projectID, input.RuntimeKey, input.Key)
}

func (s *BuiltinRuntimeService) ListRealtimeKeyDefinitions(ctx context.Context, projectID, connectionID string) ([]BuiltinRealtimeKeyDefinition, error) {
	metaPool := s.metaDB()
	if metaPool == nil {
		return []BuiltinRealtimeKeyDefinition{}, nil
	}
	rows, err := metaPool.Query(ctx, `
		SELECT id::text, key_path, value_type, default_ttl_seconds, description, created_at, updated_at
		FROM data_realtime_keys
		WHERE project_id = $1 AND connection_id = $2 AND provider = 'builtin'
		ORDER BY key_path
	`, projectID, connectionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询实时库 key 定义失败", err)
	}
	defer rows.Close()
	keys := make([]BuiltinRealtimeKeyDefinition, 0)
	for rows.Next() {
		item := BuiltinRealtimeKeyDefinition{}
		if err := rows.Scan(&item.ID, &item.Key, &item.ValueType, &item.DefaultTtlSeconds, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取实时库 key 定义失败", err)
		}
		keys = append(keys, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历实时库 key 定义失败", err)
	}
	return keys, nil
}

func (s *BuiltinRuntimeService) UpsertRealtimeKeyDefinition(ctx context.Context, projectID, connectionID, keyPath, valueType string, ttlSeconds int, description string) (*BuiltinRealtimeKeyDefinition, error) {
	metaPool := s.metaDB()
	if metaPool == nil {
		return nil, nil
	}
	keyPath = strings.Trim(strings.TrimSpace(keyPath), "/")
	if keyPath == "" || strings.Contains(keyPath, "..") {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时库 key 不合法")
	}
	valueType = strings.TrimSpace(valueType)
	if valueType == "" {
		valueType = "json"
	}
	if ttlSeconds < 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TTL 不能小于 0")
	}
	row := metaPool.QueryRow(ctx, `
		INSERT INTO data_realtime_keys (project_id, connection_id, provider, key_path, redis_type, value_type, default_ttl_seconds, description)
		VALUES ($1, $2, 'builtin', $3, 'string', $4, $5, $6)
		ON CONFLICT (project_id, connection_id, key_path)
		DO UPDATE SET value_type = EXCLUDED.value_type,
		              default_ttl_seconds = EXCLUDED.default_ttl_seconds,
		              description = EXCLUDED.description,
		              updated_at = now()
		RETURNING id::text, key_path, value_type, default_ttl_seconds, description, created_at, updated_at
	`, projectID, connectionID, keyPath, valueType, ttlSeconds, strings.TrimSpace(description))
	item := BuiltinRealtimeKeyDefinition{}
	if err := row.Scan(&item.ID, &item.Key, &item.ValueType, &item.DefaultTtlSeconds, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "保存实时库 key 定义失败", err)
	}
	return &item, nil
}

func (s *BuiltinRuntimeService) GetRealtimeKey(ctx context.Context, projectID, runtimeKey, key string) (*BuiltinRealtimeValue, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if s == nil || s.realtimeClient == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态实时运行库未初始化")
	}
	fullKey, err := deriveBuiltinRealtimeKey(s.realtimeKeyPrefix, projectID, runtimeKey, key)
	if err != nil {
		return nil, err
	}
	raw, err := s.realtimeClient.Get(ctx, fullKey).Result()
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "读取实时库 key 失败", err)
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		value = raw
	}
	ttl, _ := s.realtimeClient.TTL(ctx, fullKey).Result()
	return &BuiltinRealtimeValue{
		Key:   key,
		Type:  "string",
		Value: value,
		TTL:   int64(ttl.Seconds()),
	}, nil
}

func (s *BuiltinRuntimeService) DeleteRealtimeKey(ctx context.Context, projectID, runtimeKey, key string) (map[string]bool, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if s == nil || s.realtimeClient == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态实时运行库未初始化")
	}
	fullKey, err := deriveBuiltinRealtimeKey(s.realtimeKeyPrefix, projectID, runtimeKey, key)
	if err != nil {
		return nil, err
	}
	if err := s.realtimeClient.Del(ctx, fullKey).Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "删除实时库 key 失败", err)
	}
	return map[string]bool{"deleted": true}, nil
}

func (s *BuiltinRuntimeService) metaDB() *pgxpool.Pool {
	if s == nil {
		return nil
	}
	if s.metaPool != nil {
		return s.metaPool
	}
	return s.devPool
}

func (s *BuiltinRuntimeService) ensureSchema(ctx context.Context, schemaName string) error {
	if s == nil || s.devPool == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态运行库未初始化")
	}
	identifier := pgx.Identifier{schemaName}.Sanitize()
	if _, err := s.devPool.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, identifier)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "创建开发态 schema 失败", err)
	}
	if _, err := s.devPool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS if_table_metadata (
			schema_name text NOT NULL,
			table_name text NOT NULL,
			kind text NOT NULL DEFAULT 'table',
			timeseries jsonb NOT NULL DEFAULT '{}'::jsonb,
			updated_at timestamptz NOT NULL DEFAULT now(),
			PRIMARY KEY (schema_name, table_name)
		)
	`); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "创建开发态表元数据失败", err)
	}
	return nil
}

func validateBuiltinSQL(sqlText string) error {
	normalized := strings.ToLower(strings.TrimSpace(sqlText))
	blocked := []string{
		"drop database",
		"create database",
		"alter system",
		"copy ",
		" program ",
		"pg_catalog",
		"information_schema",
		"pg_toast",
		"if_core",
		"if_data",
		"if_dev_data",
		"postgres.",
		"template0",
		"template1",
	}
	for _, token := range blocked {
		if strings.Contains(normalized, token) {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 包含不允许访问的系统资源或危险语句")
		}
	}
	if strings.Contains(normalized, "p_") && strings.Contains(normalized, "_app.") {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 不允许显式访问工程 schema")
	}
	if strings.Contains(normalized, "p_") && strings.Contains(normalized, "_ts.") {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 不允许显式访问工程 schema")
	}
	return nil
}

func builtinSQLSchemaSuffix(store string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(store)) {
	case "relation":
		return "app", nil
	case "timeseries":
		return "ts", nil
	default:
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置 SQL 运行库类型不受支持")
	}
}

func isBuiltinQuerySQL(sqlText string) bool {
	return sqlWorkbenchStatementReturnsRows(sqlText)
}

func collectBuiltinSQLRows(rows pgx.Rows, limit int) (*BuiltinSQLExecuteResult, error) {
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, 0, len(fieldDescriptions))
	databaseTypes := make([]string, 0, len(fieldDescriptions))
	for _, field := range fieldDescriptions {
		columns = append(columns, field.Name)
		typeName := ""
		if dataType, ok := rows.Conn().TypeMap().TypeForOID(field.DataTypeOID); ok {
			typeName = dataType.Name
		}
		databaseTypes = append(databaseTypes, typeName)
	}

	resultRows := make([]map[string]any, 0)
	resultBytes := 0
	truncatedBy := ""
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取内置运行库 SQL 结果失败", err)
		}
		item := map[string]any{}
		for index, column := range columns {
			if index < len(values) {
				item[column] = normalizeSQLValue(values[index], databaseTypes[index])
			}
		}
		accepted, reason, nextBytes := admitSQLResultRow(len(resultRows), resultBytes, limit, developmentSQLMaxBytes, item)
		if !accepted {
			truncatedBy = reason
			break
		}
		resultRows = append(resultRows, item)
		resultBytes = nextBytes
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "遍历内置运行库 SQL 结果失败", err)
	}
	return &BuiltinSQLExecuteResult{
		Columns:     columns,
		ColumnTypes: canonicalSQLColumnTypes(columns, databaseTypes),
		Rows:        resultRows,
		RowCount:    len(resultRows),
		Truncated:   truncatedBy != "",
		TruncatedBy: truncatedBy,
		Limits:      developmentSQLLimits(),
	}, nil
}

func deriveBuiltinRealtimeKey(prefix, projectID, runtimeKey, key string) (string, error) {
	runtimeKey = strings.Trim(strings.TrimSpace(runtimeKey), ":")
	if runtimeKey == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时库缺少系统标识")
	}
	key = strings.Trim(strings.TrimSpace(key), "/")
	if key == "" || strings.Contains(key, "..") {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时库 key 不合法")
	}
	return fmt.Sprintf("%s:%s:%s:%s", prefix, projectID, runtimeKey, key), nil
}

func sanitizeBuiltinIdentifier(value string, defaultValue string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue
	}
	normalized := schemaUnsafePattern.ReplaceAllString(strings.ToLower(value), "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return defaultValue
	}
	return normalized
}

type pgxCommandTag interface {
	RowsAffected() int64
}

var _ pgxCommandTag = pgconn.CommandTag{}
