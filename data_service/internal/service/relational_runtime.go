package service

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

var readOnlySQLPrefixPattern = regexp.MustCompile(`(?is)^\s*(select|with)\b`)

// RelationalTable 表示关系库表元信息。
type RelationalTable struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	Type   string `json:"type"`
}

// RelationalTableColumn 表示表字段元信息。
type RelationalTableColumn struct {
	Name             string  `json:"name"`
	Type             string  `json:"type"`
	MaxLength        *int    `json:"maxLength"`
	NumericPrecision *int    `json:"numericPrecision"`
	NumericScale     *int    `json:"numericScale"`
	Nullable         bool    `json:"nullable"`
	DefaultValue     *string `json:"defaultValue"`
	IsPrimary        bool    `json:"isPrimary"`
	IsUnique         bool    `json:"isUnique"`
	AutoIncrement    bool    `json:"autoIncrement"`
	Comment          *string `json:"comment"`
}

// RelationalIndex 表示表索引元信息。
type RelationalIndex struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Method  string   `json:"method"`
	Columns []string `json:"columns"`
}

// RelationalForeignKey 表示外键元信息。
type RelationalForeignKey struct {
	Name             string `json:"name"`
	ColumnName       string `json:"columnName"`
	ReferencedTable  string `json:"referencedTable"`
	ReferencedColumn string `json:"referencedColumn"`
	UpdateRule       string `json:"updateRule"`
	DeleteRule       string `json:"deleteRule"`
}

// RelationalTableStructure 表示表结构响应。
type RelationalTableStructure struct {
	Columns     []RelationalTableColumn `json:"columns"`
	Indexes     []RelationalIndex       `json:"indexes"`
	ForeignKeys []RelationalForeignKey  `json:"foreignKeys"`
}

// RelationalPagination 表示分页元信息。
type RelationalPagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// RelationalTableData 表示表数据预览响应。
type RelationalTableData struct {
	Columns    []string             `json:"columns"`
	Rows       [][]any              `json:"rows"`
	Pagination RelationalPagination `json:"pagination"`
}

// RelationalQueryResult 表示只读 SQL 执行结果。
type RelationalQueryResult struct {
	Columns       []string `json:"columns"`
	Rows          [][]any  `json:"rows"`
	RowCount      int      `json:"rowCount"`
	ExecutionTime int64    `json:"executionTime"`
}

type relationalRuntimeConfig struct {
	DBType                 string
	DatabaseURL            string
	Host                   string
	Port                   int
	Database               string
	Username               string
	Password               string
	Schema                 string
	SSLMode                string
	Charset                string
	Encrypt                bool
	TrustServerCertificate bool
}

type relationalRuntime struct {
	dbType     string
	searchPath string
	pgPool     *pgxpool.Pool
	sqlDB      *sql.DB
}

type relationalTx interface {
	Query(context.Context, string, ...any) (relationalRows, error)
	QueryRow(context.Context, string, ...any) relationalRow
	Rollback(context.Context) error
}

type relationalRows interface {
	Next() bool
	Scan(...any) error
	Close()
	Err() error
	Columns() ([]string, error)
	Values() ([]any, error)
}

type relationalRow interface {
	Scan(...any) error
}

func connectRelationalRuntime(ctx context.Context, config map[string]any) (*relationalRuntime, error) {
	runtimeConfig, err := parseRelationalRuntimeConfig(config)
	if err != nil {
		return nil, err
	}

	switch runtimeConfig.DBType {
	case "postgresql":
		databaseURL := runtimeConfig.DatabaseURL
		if databaseURL == "" {
			databaseURL = buildPostgreSQLURL(runtimeConfig)
		}

		poolConfig, err := pgxpool.ParseConfig(databaseURL)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "PostgreSQL 连接串格式无效", err)
		}
		poolConfig.MaxConns = 4
		poolConfig.MinConns = 0
		poolConfig.MaxConnIdleTime = 30 * time.Second
		poolConfig.MaxConnLifetime = 5 * time.Minute

		pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "创建 PostgreSQL 连接池失败", err)
		}

		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接 PostgreSQL 失败", err)
		}

		searchPath := strings.TrimSpace(runtimeConfig.Schema)
		if searchPath == "" {
			searchPath = "public"
		}
		if searchPath != "" {
			if _, err := pool.Exec(ctx, fmt.Sprintf("SET search_path TO %s", pgx.Identifier{searchPath}.Sanitize())); err != nil {
				pool.Close()
				return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "设置 search_path 失败", err)
			}
		}

		return &relationalRuntime{
			dbType:     runtimeConfig.DBType,
			searchPath: searchPath,
			pgPool:     pool,
		}, nil
	case "mysql":
		databaseURL := runtimeConfig.DatabaseURL
		if databaseURL == "" {
			databaseURL = buildMySQLDSN(runtimeConfig)
		}

		db, err := sql.Open("mysql", databaseURL)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MySQL 连接串格式无效", err)
		}
		db.SetMaxOpenConns(4)
		db.SetMaxIdleConns(1)
		db.SetConnMaxIdleTime(30 * time.Second)
		db.SetConnMaxLifetime(5 * time.Minute)
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接 MySQL 失败", err)
		}

		return &relationalRuntime{
			dbType:     runtimeConfig.DBType,
			searchPath: runtimeConfig.Database,
			sqlDB:      db,
		}, nil
	case "sqlserver":
		databaseURL := runtimeConfig.DatabaseURL
		if databaseURL == "" {
			databaseURL = buildSQLServerDSN(runtimeConfig)
		}

		db, err := sql.Open("sqlserver", databaseURL)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL Server 连接串格式无效", err)
		}
		db.SetMaxOpenConns(4)
		db.SetMaxIdleConns(1)
		db.SetConnMaxIdleTime(30 * time.Second)
		db.SetConnMaxLifetime(5 * time.Minute)
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接 SQL Server 失败", err)
		}

		searchPath := strings.TrimSpace(runtimeConfig.Schema)
		if searchPath == "" {
			searchPath = "dbo"
		}

		return &relationalRuntime{
			dbType:     runtimeConfig.DBType,
			searchPath: searchPath,
			sqlDB:      db,
		}, nil
	default:
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前仅支持 PostgreSQL / MySQL / SQL Server 连接探查")
	}
}

func parseRelationalRuntimeConfig(config map[string]any) (relationalRuntimeConfig, error) {
	if config == nil {
		return relationalRuntimeConfig{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接配置不能为空")
	}

	getString := func(keys ...string) string {
		for _, key := range keys {
			raw, ok := config[key]
			if !ok {
				continue
			}
			switch typed := raw.(type) {
			case string:
				if text := strings.TrimSpace(typed); text != "" {
					return text
				}
			}
		}
		return ""
	}

	getInt := func(defaultValue int, keys ...string) int {
		for _, key := range keys {
			raw, ok := config[key]
			if !ok || raw == nil {
				continue
			}
			switch typed := raw.(type) {
			case float64:
				return int(typed)
			case int:
				return typed
			case int32:
				return int(typed)
			case int64:
				return int(typed)
			case string:
				parsed, err := strconv.Atoi(strings.TrimSpace(typed))
				if err == nil {
					return parsed
				}
			}
		}
		return defaultValue
	}

	getBool := func(keys ...string) bool {
		for _, key := range keys {
			raw, ok := config[key]
			if !ok || raw == nil {
				continue
			}
			switch typed := raw.(type) {
			case bool:
				return typed
			case string:
				return strings.EqualFold(strings.TrimSpace(typed), "true")
			}
		}
		return false
	}

	dbType := strings.ToLower(strings.TrimSpace(getString("dbType", "db_type")))
	if dbType == "" {
		dbType = "postgresql"
	}

	schema := getString("schema", "searchPath", "search_path")
	databaseURL := getString("databaseUrl", "databaseURL", "dsn")
	if databaseURL == "" {
		host := getString("host")
		database := getString("database", "dbname")
		username := getString("username", "user")
		if host == "" || database == "" || username == "" {
			return relationalRuntimeConfig{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接配置缺少关系库必填字段")
		}
	}

	sslMode := "disable"
	if getBool("ssl") {
		sslMode = "require"
	}

	return relationalRuntimeConfig{
		DBType:                 dbType,
		DatabaseURL:            databaseURL,
		Host:                   getString("host"),
		Port:                   getInt(defaultRelationalPort(dbType), "port"),
		Database:               getString("database", "dbname"),
		Username:               getString("username", "user"),
		Password:               getString("password"),
		Schema:                 schema,
		SSLMode:                sslMode,
		Charset:                getString("charset"),
		Encrypt:                getBool("encrypt"),
		TrustServerCertificate: getBool("trustServerCertificate"),
	}, nil
}

func defaultRelationalPort(dbType string) int {
	switch dbType {
	case "mysql":
		return 3306
	case "sqlserver":
		return 1433
	default:
		return 5432
	}
}

func buildPostgreSQLURL(config relationalRuntimeConfig) string {
	query := url.Values{}
	query.Set("sslmode", config.SSLMode)
	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(config.Username, config.Password),
		Host:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Path:     config.Database,
		RawQuery: query.Encode(),
	}).String()
}

func buildMySQLDSN(config relationalRuntimeConfig) string {
	query := url.Values{}
	query.Set("parseTime", "true")
	query.Set("loc", "Local")
	if strings.TrimSpace(config.Charset) != "" {
		query.Set("charset", strings.TrimSpace(config.Charset))
	}

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		query.Encode(),
	)
}

func buildSQLServerDSN(config relationalRuntimeConfig) string {
	query := url.Values{}
	query.Set("database", config.Database)
	if config.Encrypt {
		query.Set("encrypt", "true")
	} else {
		query.Set("encrypt", "disable")
	}
	if config.TrustServerCertificate {
		query.Set("TrustServerCertificate", "true")
	}

	return (&url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(config.Username, config.Password),
		Host:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		RawQuery: query.Encode(),
	}).String()
}

func (r *relationalRuntime) Close() {
	if r == nil {
		return
	}
	if r.pgPool != nil {
		r.pgPool.Close()
	}
	if r.sqlDB != nil {
		_ = r.sqlDB.Close()
	}
}

func (r *relationalRuntime) SearchPath() string {
	if r == nil {
		return ""
	}
	return r.searchPath
}

func (r *relationalRuntime) DBType() string {
	if r == nil {
		return ""
	}
	return r.dbType
}

func (r *relationalRuntime) Ping(ctx context.Context) error {
	if r == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "关系库运行时未初始化")
	}
	if r.pgPool != nil {
		return r.pgPool.Ping(ctx)
	}
	if r.sqlDB != nil {
		return r.sqlDB.PingContext(ctx)
	}
	return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "关系库运行时未初始化")
}

func (r *relationalRuntime) Query(ctx context.Context, query string, args ...any) (relationalRows, error) {
	if r.pgPool != nil {
		rows, err := r.pgPool.Query(ctx, query, adaptRuntimeArgs(r.dbType, query, args)...)
		if err != nil {
			return nil, err
		}
		return &pgxRowsAdapter{rows: rows}, nil
	}

	rows, err := r.sqlDB.QueryContext(ctx, query, adaptRuntimeArgs(r.dbType, query, args)...)
	if err != nil {
		return nil, err
	}
	return &sqlRowsAdapter{rows: rows}, nil
}

func (r *relationalRuntime) QueryRow(ctx context.Context, query string, args ...any) relationalRow {
	if r.pgPool != nil {
		return &pgxRowAdapter{row: r.pgPool.QueryRow(ctx, query, adaptRuntimeArgs(r.dbType, query, args)...)}
	}

	return &sqlRowAdapter{row: r.sqlDB.QueryRowContext(ctx, query, adaptRuntimeArgs(r.dbType, query, args)...)}
}

func (r *relationalRuntime) BeginReadOnlyTx(ctx context.Context) (relationalTx, error) {
	if r.pgPool != nil {
		tx, err := r.pgPool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
		if err != nil {
			return nil, err
		}
		return &pgxTxAdapter{
			dbType: r.dbType,
			tx:     tx,
		}, nil
	}

	tx, err := r.sqlDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	return &sqlTxAdapter{
		dbType: r.dbType,
		tx:     tx,
	}, nil
}

type pgxTxAdapter struct {
	dbType string
	tx     pgx.Tx
}

func (t *pgxTxAdapter) Query(ctx context.Context, query string, args ...any) (relationalRows, error) {
	rows, err := t.tx.Query(ctx, query, adaptRuntimeArgs(t.dbType, query, args)...)
	if err != nil {
		return nil, err
	}
	return &pgxRowsAdapter{rows: rows}, nil
}

func (t *pgxTxAdapter) QueryRow(ctx context.Context, query string, args ...any) relationalRow {
	return &pgxRowAdapter{row: t.tx.QueryRow(ctx, query, adaptRuntimeArgs(t.dbType, query, args)...)}
}

func (t *pgxTxAdapter) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

type sqlTxAdapter struct {
	dbType string
	tx     *sql.Tx
}

func (t *sqlTxAdapter) Query(ctx context.Context, query string, args ...any) (relationalRows, error) {
	rows, err := t.tx.QueryContext(ctx, query, adaptRuntimeArgs(t.dbType, query, args)...)
	if err != nil {
		return nil, err
	}
	return &sqlRowsAdapter{rows: rows}, nil
}

func (t *sqlTxAdapter) QueryRow(ctx context.Context, query string, args ...any) relationalRow {
	return &sqlRowAdapter{row: t.tx.QueryRowContext(ctx, query, adaptRuntimeArgs(t.dbType, query, args)...)}
}

func (t *sqlTxAdapter) Rollback(ctx context.Context) error {
	return t.tx.Rollback()
}

type pgxRowsAdapter struct {
	rows pgx.Rows
}

func (r *pgxRowsAdapter) Next() bool             { return r.rows.Next() }
func (r *pgxRowsAdapter) Scan(dest ...any) error { return r.rows.Scan(dest...) }
func (r *pgxRowsAdapter) Close()                 { r.rows.Close() }
func (r *pgxRowsAdapter) Err() error             { return r.rows.Err() }
func (r *pgxRowsAdapter) Columns() ([]string, error) {
	columns := make([]string, 0, len(r.rows.FieldDescriptions()))
	for _, field := range r.rows.FieldDescriptions() {
		columns = append(columns, string(field.Name))
	}
	return columns, nil
}
func (r *pgxRowsAdapter) Values() ([]any, error) {
	return r.rows.Values()
}

type sqlRowsAdapter struct {
	rows *sql.Rows
}

func (r *sqlRowsAdapter) Next() bool                 { return r.rows.Next() }
func (r *sqlRowsAdapter) Scan(dest ...any) error     { return r.rows.Scan(dest...) }
func (r *sqlRowsAdapter) Close()                     { _ = r.rows.Close() }
func (r *sqlRowsAdapter) Err() error                 { return r.rows.Err() }
func (r *sqlRowsAdapter) Columns() ([]string, error) { return r.rows.Columns() }
func (r *sqlRowsAdapter) Values() ([]any, error) {
	columns, err := r.rows.Columns()
	if err != nil {
		return nil, err
	}
	values := make([]any, len(columns))
	dest := make([]any, len(columns))
	for i := range values {
		dest[i] = &values[i]
	}
	if err := r.rows.Scan(dest...); err != nil {
		return nil, err
	}

	result := make([]any, 0, len(values))
	for _, value := range values {
		result = append(result, normalizeQueryValue(value))
	}
	return result, nil
}

type pgxRowAdapter struct {
	row pgx.Row
}

func (r *pgxRowAdapter) Scan(dest ...any) error { return r.row.Scan(dest...) }

type sqlRowAdapter struct {
	row *sql.Row
}

func (r *sqlRowAdapter) Scan(dest ...any) error { return r.row.Scan(dest...) }

func adaptRuntimeArgs(dbType, query string, args []any) []any {
	if strings.EqualFold(dbType, "sqlserver") {
		namedArgs, err := prepareSQLServerNamedArgs(query, args)
		if err == nil {
			return namedArgs
		}
	}
	return args
}

func prepareSQLServerNamedArgs(query string, args []any) ([]any, error) {
	paramRegex := regexp.MustCompile(`@([A-Za-z_][A-Za-z0-9_]*)`)
	matches := paramRegex.FindAllStringSubmatch(query, -1)
	if len(matches) == 0 {
		return args, nil
	}
	if len(matches) != len(args) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL Server 参数数量与 SQL 中的 @param 不一致")
	}

	namedArgs := make([]any, 0, len(matches))
	for index, match := range matches {
		namedArgs = append(namedArgs, sql.Named(match[1], args[index]))
	}
	return namedArgs, nil
}

func ensureReadOnlySQLText(sqlText string) error {
	normalized := strings.TrimSpace(sqlText)
	if normalized == "" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 不能为空")
	}
	if !readOnlySQLPrefixPattern.MatchString(normalized) {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "仅允许执行只读 SELECT / WITH 查询")
	}
	lowerText := strings.ToLower(normalized)
	for _, keyword := range []string{" insert ", " update ", " delete ", " drop ", " alter ", " truncate ", " create "} {
		if strings.Contains(" "+lowerText+" ", keyword) {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 包含写入或结构变更语句")
		}
	}
	return nil
}
