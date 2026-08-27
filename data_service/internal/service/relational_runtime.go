package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/denisenkom/go-mssqldb/msdsn"
	mysql "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

var (
	readOnlySQLPrefixPattern       = regexp.MustCompile(`(?is)^\s*(select|with)\b`)
	postgresDollarQuoteOpenPattern = regexp.MustCompile(`^\$(?:[A-Za-z_][A-Za-z0-9_]*)?\$`)
)

// RelationalTable 表示关系库表元信息。
type RelationalTable struct {
	Schema     string                        `json:"schema"`
	Name       string                        `json:"name"`
	Type       string                        `json:"type"`
	Kind       string                        `json:"kind"`
	Timeseries *RelationalTimeseriesMetadata `json:"timeseries,omitempty"`
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
	Columns     []RelationalTableColumn       `json:"columns"`
	Indexes     []RelationalIndex             `json:"indexes"`
	ForeignKeys []RelationalForeignKey        `json:"foreignKeys"`
	Timeseries  *RelationalTimeseriesMetadata `json:"timeseries,omitempty"`
}

// RelationalTimeseriesMetadata 描述 TimescaleDB hypertable 的平台配置。
// 这些字段来自结构化建表，前端用它生成时序模板和展示真实时序策略。
type RelationalTimeseriesMetadata struct {
	TimeColumn       string   `json:"timeColumn"`
	DimensionColumns []string `json:"dimensionColumns"`
	ChunkInterval    string   `json:"chunkInterval"`
	RetentionDays    *int     `json:"retentionDays,omitempty"`
}

// UpdateRelationalTableInput 表示表结构修改的目标状态。
// 服务端会与当前结构做差异比较，只执行低风险 DDL，避免前端直接拼接 ALTER SQL。
type UpdateRelationalTableInput struct {
	Columns []CreateRelationalTableColumnInput
	Indexes []CreateRelationalTableIndexInput
}

// RelationalPagination 表示分页元信息。
type RelationalPagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// RelationalTablePage 表示可增长的数据库对象分页结果。
type RelationalTablePage struct {
	Items      []RelationalTable    `json:"items"`
	Pagination RelationalPagination `json:"pagination"`
}

// RelationalTableData 表示表数据预览响应。
type RelationalTableData struct {
	Columns    []string             `json:"columns"`
	Rows       [][]any              `json:"rows"`
	Pagination RelationalPagination `json:"pagination"`
}

// RelationalQueryResult 表示 SQL 工作台执行结果。
type RelationalQueryResult struct {
	Columns       []string          `json:"columns"`
	ColumnTypes   map[string]string `json:"columnTypes"`
	Rows          [][]any           `json:"rows"`
	RowCount      int               `json:"rowCount"`
	ExecutionTime int64             `json:"executionTime"`
	Truncated     bool              `json:"truncated"`
	TruncatedBy   string            `json:"truncatedBy,omitempty"`
	Limits        SQLResultLimits   `json:"limits"`
}

// CreateRelationalTableInput 是工作台结构化建表的领域输入。
// 输入只表达表、字段和索引意图；具体 DDL 由服务端按连接类型生成，避免前端拼接 SQL。
type CreateRelationalTableInput struct {
	Name       string
	Kind       string
	Columns    []CreateRelationalTableColumnInput
	Indexes    []CreateRelationalTableIndexInput
	Timeseries *CreateRelationalTimeseriesInput
}

// CreateRelationalTableColumnInput 表示建表字段定义。
// Type 为受控类型标识，Length/Precision/Scale 只在对应类型需要时生效。
type CreateRelationalTableColumnInput struct {
	Name          string
	Type          string
	Length        *int
	Precision     *int
	Scale         *int
	Nullable      bool
	Primary       bool
	AutoIncrement bool
	DefaultValue  string
	Comment       string
}

// CreateRelationalTableIndexInput 表示建表索引定义。
// Type 仅支持 index/unique；主键由字段 Primary 汇总生成。
type CreateRelationalTableIndexInput struct {
	Name    string
	Type    string
	Columns []string
}

// CreateRelationalTimeseriesInput 为 TimescaleDB hypertable 建表扩展。
type CreateRelationalTimeseriesInput struct {
	TimeColumn       string
	DimensionColumns []CreateRelationalTableColumnInput
	ChunkInterval    string
	RetentionDays    *int
}

type relationalRuntimeConfig struct {
	DBType      string
	DatabaseURL string
	Host        string
	Port        int
	Database    string
	Username    string
	Password    string
	Schema      string
	SSLMode     string
	SSLCA       string
	SSLCert     string
	SSLKey      string
	Charset     string
}

type relationalRuntime struct {
	dbType             string
	searchPath         string
	pgPool             *pgxpool.Pool
	sqlDB              *sql.DB
	mysqlTLSConfigName string
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
	ColumnTypes() ([]string, error)
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
		if err := applyPostgreSQLTLSConfig(poolConfig, runtimeConfig); err != nil {
			return nil, err
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
		databaseURL, tlsConfigName, err := prepareMySQLDSN(runtimeConfig)
		if err != nil {
			return nil, err
		}

		db, err := sql.Open("mysql", databaseURL)
		if err != nil {
			if tlsConfigName != "" {
				mysql.DeregisterTLSConfig(tlsConfigName)
			}
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MySQL 连接串格式无效", err)
		}
		db.SetMaxOpenConns(4)
		db.SetMaxIdleConns(1)
		db.SetConnMaxIdleTime(30 * time.Second)
		db.SetConnMaxLifetime(5 * time.Minute)
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			if tlsConfigName != "" {
				mysql.DeregisterTLSConfig(tlsConfigName)
			}
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接 MySQL 失败", err)
		}

		return &relationalRuntime{
			dbType:             runtimeConfig.DBType,
			searchPath:         runtimeConfig.Database,
			sqlDB:              db,
			mysqlTLSConfigName: tlsConfigName,
		}, nil
	case "sqlserver":
		databaseURL := runtimeConfig.DatabaseURL
		if databaseURL == "" {
			databaseURL = buildSQLServerDSN(runtimeConfig)
		}

		connector, err := prepareSQLServerConnector(databaseURL, runtimeConfig)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL Server 连接串格式无效", err)
		}
		db := sql.OpenDB(connector)
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

	sslConfig := mapFromAny(config["sslConfig"])
	sslMode := strings.ToLower(strings.TrimSpace(toString(sslConfig["mode"])))
	if sslMode == "" {
		sslMode = "disable"
	}
	if !validRelationalTLSMode(dbType, sslMode) {
		return relationalRuntimeConfig{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TLS 模式与当前数据库类型不匹配")
	}

	return relationalRuntimeConfig{
		DBType:      dbType,
		DatabaseURL: databaseURL,
		Host:        getString("host"),
		Port:        getInt(defaultRelationalPort(dbType), "port"),
		Database:    getString("database", "dbname"),
		Username:    getString("username", "user"),
		Password:    getString("password"),
		Schema:      schema,
		SSLMode:     sslMode,
		SSLCA:       toString(sslConfig["ca"]),
		SSLCert:     toString(sslConfig["cert"]),
		SSLKey:      toString(sslConfig["key"]),
		Charset:     getString("charset"),
	}, nil
}

func validRelationalTLSMode(dbType, mode string) bool {
	if dbType == "sqlserver" {
		return mode == "disable" || mode == "require" || mode == "verify-full"
	}
	return mode == "disable" || mode == "prefer" || mode == "require" || mode == "verify-ca" || mode == "verify-full"
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
	if config.SSLMode != "disable" {
		query.Set("encrypt", "true")
	} else {
		query.Set("encrypt", "disable")
	}
	if config.SSLMode == "require" {
		query.Set("TrustServerCertificate", "true")
	}

	return (&url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(config.Username, config.Password),
		Host:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		RawQuery: query.Encode(),
	}).String()
}

// buildRelationalTLSConfig 将统一关系库 TLS 配置转换为 Go TLS 配置。
// require 只保证链路加密；verify-ca 校验证书链；verify-full 同时校验主机名。
func buildRelationalTLSConfig(config relationalRuntimeConfig) (*tls.Config, error) {
	if config.SSLMode == "disable" || config.SSLMode == "prefer" {
		return nil, nil
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: config.Host}
	if config.SSLMode == "require" {
		tlsConfig.InsecureSkipVerify = true //nolint:gosec // require 语义仅要求加密，不校验证书。
	}
	if strings.TrimSpace(config.SSLCA) != "" {
		roots, err := x509.SystemCertPool()
		if err != nil || roots == nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM([]byte(config.SSLCA)) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "CA 证书不是有效的 PEM 内容")
		}
		tlsConfig.RootCAs = roots
	}
	certText := strings.TrimSpace(config.SSLCert)
	keyText := strings.TrimSpace(config.SSLKey)
	if (certText == "") != (keyText == "") {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "客户端证书和私钥必须同时配置")
	}
	if certText != "" {
		certificate, err := tls.X509KeyPair([]byte(config.SSLCert), []byte(config.SSLKey))
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "客户端证书或私钥无效", err)
		}
		tlsConfig.Certificates = []tls.Certificate{certificate}
	}
	if config.SSLMode == "verify-ca" {
		tlsConfig.InsecureSkipVerify = true //nolint:gosec // 由 VerifyConnection 校验证书链，但按模式忽略主机名。
		tlsConfig.VerifyConnection = func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) == 0 {
				return errors.New("服务端未提供证书")
			}
			intermediates := x509.NewCertPool()
			for _, certificate := range state.PeerCertificates[1:] {
				intermediates.AddCert(certificate)
			}
			_, err := state.PeerCertificates[0].Verify(x509.VerifyOptions{
				Roots: tlsConfig.RootCAs, Intermediates: intermediates,
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			})
			return err
		}
	}
	return tlsConfig, nil
}

func applyPostgreSQLTLSConfig(poolConfig *pgxpool.Config, config relationalRuntimeConfig) error {
	if config.SSLMode == "disable" || config.SSLMode == "prefer" {
		return nil
	}
	tlsConfig, err := buildRelationalTLSConfig(config)
	if err != nil {
		return err
	}
	poolConfig.ConnConfig.TLSConfig = tlsConfig
	for _, fallback := range poolConfig.ConnConfig.Fallbacks {
		if fallback.TLSConfig != nil {
			fallback.TLSConfig = tlsConfig.Clone()
		}
	}
	return nil
}

func prepareMySQLDSN(config relationalRuntimeConfig) (string, string, error) {
	dsn := config.DatabaseURL
	if dsn == "" {
		dsn = buildMySQLDSN(config)
	}
	parsed, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MySQL 连接串格式无效", err)
	}
	switch config.SSLMode {
	case "disable":
		parsed.TLSConfig = "false"
	case "prefer":
		parsed.TLSConfig = "preferred"
	default:
		tlsConfig, err := buildRelationalTLSConfig(config)
		if err != nil {
			return "", "", err
		}
		name := "induforge-" + uuid.NewString()
		if err := mysql.RegisterTLSConfig(name, tlsConfig); err != nil {
			return "", "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "注册 MySQL TLS 配置失败", err)
		}
		parsed.TLSConfig = name
		return parsed.FormatDSN(), name, nil
	}
	return parsed.FormatDSN(), "", nil
}

func prepareSQLServerConnector(dsn string, config relationalRuntimeConfig) (*mssql.Connector, error) {
	parsed, _, err := msdsn.Parse(dsn)
	if err != nil {
		return nil, err
	}
	if config.SSLMode == "disable" {
		parsed.Encryption = msdsn.EncryptionDisabled
		parsed.TLSConfig = nil
		return mssql.NewConnectorConfig(parsed), nil
	}
	tlsConfig, err := buildRelationalTLSConfig(config)
	if err != nil {
		return nil, err
	}
	// SQL Server 的 TLS 记录需要关闭动态分片，以匹配 TDS 包边界。
	tlsConfig.DynamicRecordSizingDisabled = true
	parsed.Encryption = msdsn.EncryptionRequired
	parsed.TLSConfig = tlsConfig
	return mssql.NewConnectorConfig(parsed), nil
}

func sqlExecutionErrorMessage(prefix string, err error) string {
	detail := ""
	if err != nil {
		detail = strings.Join(strings.Fields(err.Error()), " ")
	}
	if detail == "" {
		return prefix
	}
	const maxDetailLength = 420
	detailRunes := []rune(detail)
	if len(detailRunes) > maxDetailLength {
		detail = string(detailRunes[:maxDetailLength]) + "…"
	}
	return prefix + "：" + detail
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
	if r.mysqlTLSConfigName != "" {
		mysql.DeregisterTLSConfig(r.mysqlTLSConfigName)
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

func (r *relationalRuntime) Exec(ctx context.Context, query string, args ...any) error {
	_, err := r.ExecAffected(ctx, query, args...)
	return err
}

func (r *relationalRuntime) ExecAffected(ctx context.Context, query string, args ...any) (int64, error) {
	if r == nil {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "关系库运行时未初始化")
	}
	if r.pgPool != nil {
		tag, err := r.pgPool.Exec(ctx, query, adaptRuntimeArgs(r.dbType, query, args)...)
		return tag.RowsAffected(), err
	}
	result, err := r.sqlDB.ExecContext(ctx, query, adaptRuntimeArgs(r.dbType, query, args)...)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, nil
	}
	return affected, nil
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
func (r *pgxRowsAdapter) ColumnTypes() ([]string, error) {
	types := make([]string, 0, len(r.rows.FieldDescriptions()))
	for _, field := range r.rows.FieldDescriptions() {
		name := ""
		if dataType, ok := r.rows.Conn().TypeMap().TypeForOID(field.DataTypeOID); ok {
			name = dataType.Name
		}
		types = append(types, name)
	}
	return types, nil
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
func (r *sqlRowsAdapter) ColumnTypes() ([]string, error) {
	columnTypes, err := r.rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	types := make([]string, 0, len(columnTypes))
	for _, columnType := range columnTypes {
		types = append(types, columnType.DatabaseTypeName())
	}
	return types, nil
}
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

	// 保留驱动原始值，调用方会结合 ColumnTypes 做最终规范化。
	// 若在这里提前把 []byte 转为字符串，BLOB/VARBINARY 会丢失类型信息并产生乱码。
	return values, nil
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
	names := make([]string, 0, len(matches))
	seen := map[string]struct{}{}
	for _, match := range matches {
		name := strings.ToLower(match[1])
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, match[1])
	}
	if len(names) != len(args) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL Server 参数数量与 SQL 中的 @param 不一致")
	}

	namedArgs := make([]any, 0, len(names))
	for index, name := range names {
		namedArgs = append(namedArgs, sql.Named(name, args[index]))
	}
	return namedArgs, nil
}

// validateSQLParameterCount 在进入驱动前校验值参数数量，避免把表名、列名或 SQL 片段误当成可绑定参数。
func validateSQLParameterCount(dbType, query string, args []any) error {
	normalized := stripSQLLiteralsAndComments(query)
	expected := 0
	switch strings.ToLower(strings.TrimSpace(dbType)) {
	case "postgres", "postgresql", "builtin.relation", "builtin.timeseries":
		matches := regexp.MustCompile(`\$(\d+)`).FindAllStringSubmatch(normalized, -1)
		seen := map[int]struct{}{}
		for _, match := range matches {
			index, err := strconv.Atoi(match[1])
			if err != nil || index < 1 {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "PostgreSQL 参数必须使用 $1、$2…")
			}
			seen[index] = struct{}{}
			if index > expected {
				expected = index
			}
		}
		for index := 1; index <= expected; index++ {
			if _, ok := seen[index]; !ok {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "PostgreSQL 参数序号必须从 $1 连续递增")
			}
		}
	case "sqlserver", "mssql":
		matches := regexp.MustCompile(`@([A-Za-z_][A-Za-z0-9_]*)`).FindAllStringSubmatch(normalized, -1)
		seen := map[string]struct{}{}
		for _, match := range matches {
			seen[strings.ToLower(match[1])] = struct{}{}
		}
		expected = len(seen)
	default:
		expected = strings.Count(normalized, "?")
	}
	if expected != len(args) {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("SQL 值参数数量不匹配：SQL 需要 %d 个，实际提交 %d 个", expected, len(args)))
	}
	return nil
}

func stripSQLLiteralsAndComments(query string) string {
	var builder strings.Builder
	inSingle, inDouble, inLineComment, inBlockComment := false, false, false, false
	for index := 0; index < len(query); index++ {
		current := query[index]
		next := byte(0)
		if index+1 < len(query) {
			next = query[index+1]
		}
		if inLineComment {
			if current == '\n' {
				inLineComment = false
				builder.WriteByte(current)
			} else {
				builder.WriteByte(' ')
			}
			continue
		}
		if inBlockComment {
			if current == '*' && next == '/' {
				inBlockComment = false
				builder.WriteString("  ")
				index++
			} else {
				builder.WriteByte(' ')
			}
			continue
		}
		if !inSingle && !inDouble && current == '-' && next == '-' {
			inLineComment = true
			builder.WriteString("  ")
			index++
			continue
		}
		if !inSingle && !inDouble && current == '/' && next == '*' {
			inBlockComment = true
			builder.WriteString("  ")
			index++
			continue
		}
		if !inSingle && !inDouble && current == '$' {
			// PostgreSQL 函数体通常使用 $$ 或 $tag$ 引用；其中的 $1 是函数形参，
			// 不能计入工作台外层 SQL 的绑定参数数量。
			delimiter := postgresDollarQuoteOpenPattern.FindString(query[index:])
			if delimiter != "" {
				bodyStart := index + len(delimiter)
				closingOffset := strings.Index(query[bodyStart:], delimiter)
				end := len(query)
				if closingOffset >= 0 {
					end = bodyStart + closingOffset + len(delimiter)
				}
				builder.WriteString(strings.Repeat(" ", end-index))
				index = end - 1
				continue
			}
		}
		if !inDouble && current == '\'' {
			if inSingle && next == '\'' {
				builder.WriteString("  ")
				index++
				continue
			}
			inSingle = !inSingle
			builder.WriteByte(' ')
			continue
		}
		if !inSingle && current == '"' {
			inDouble = !inDouble
			builder.WriteByte(' ')
			continue
		}
		if inSingle || inDouble {
			builder.WriteByte(' ')
		} else {
			builder.WriteByte(current)
		}
	}
	return builder.String()
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
