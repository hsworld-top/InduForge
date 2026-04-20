package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	Name             string `json:"name"`
	Type             string `json:"type"`
	MaxLength        *int   `json:"maxLength"`
	NumericPrecision *int   `json:"numericPrecision"`
	NumericScale     *int   `json:"numericScale"`
	Nullable         bool   `json:"nullable"`
	DefaultValue     *string `json:"defaultValue"`
	IsPrimary        bool   `json:"isPrimary"`
	IsUnique         bool   `json:"isUnique"`
	AutoIncrement    bool   `json:"autoIncrement"`
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
	Columns     []RelationalTableColumn  `json:"columns"`
	Indexes     []RelationalIndex        `json:"indexes"`
	ForeignKeys []RelationalForeignKey   `json:"foreignKeys"`
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
	Columns    []string               `json:"columns"`
	Rows       [][]any                `json:"rows"`
	Pagination RelationalPagination   `json:"pagination"`
}

// RelationalQueryResult 表示只读 SQL 执行结果。
type RelationalQueryResult struct {
	Columns       []string `json:"columns"`
	Rows          [][]any  `json:"rows"`
	RowCount      int      `json:"rowCount"`
	ExecutionTime int64    `json:"executionTime"`
}

type relationalRuntimeConfig struct {
	DBType     string
	DatabaseURL string
	Host       string
	Port       int
	Database   string
	Username   string
	Password   string
	Schema     string
	SSLMode    string
}

func connectRelationalRuntime(ctx context.Context, config map[string]any) (*pgxpool.Pool, string, func(), error) {
	runtimeConfig, err := parseRelationalRuntimeConfig(config)
	if err != nil {
		return nil, "", nil, err
	}
	if runtimeConfig.DBType != "postgresql" {
		return nil, "", nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前仅支持 PostgreSQL 连接探查")
	}

	databaseURL := runtimeConfig.DatabaseURL
	if databaseURL == "" {
		databaseURL = buildPostgreSQLURL(runtimeConfig)
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, "", nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "PostgreSQL 连接串格式无效", err)
	}
	poolConfig.MaxConns = 4
	poolConfig.MinConns = 0
	poolConfig.MaxConnIdleTime = 30 * time.Second
	poolConfig.MaxConnLifetime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, "", nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "创建 PostgreSQL 连接池失败", err)
	}
	cleanup := func() {
		pool.Close()
	}

	if err := pool.Ping(ctx); err != nil {
		cleanup()
		return nil, "", nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接 PostgreSQL 失败", err)
	}

	searchPath := strings.TrimSpace(runtimeConfig.Schema)
	if searchPath != "" {
		if _, err := pool.Exec(ctx, fmt.Sprintf("SET search_path TO %s", pgx.Identifier{searchPath}.Sanitize())); err != nil {
			cleanup()
			return nil, "", nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "设置 search_path 失败", err)
		}
	}

	return pool, searchPath, cleanup, nil
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
			return relationalRuntimeConfig{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接配置缺少 PostgreSQL 必填字段")
		}
	}

	sslMode := "disable"
	if getBool("ssl") {
		sslMode = "require"
	}

	return relationalRuntimeConfig{
		DBType:      dbType,
		DatabaseURL: databaseURL,
		Host:        getString("host"),
		Port:        getInt(5432, "port"),
		Database:    getString("database", "dbname"),
		Username:    getString("username", "user"),
		Password:    getString("password"),
		Schema:      schema,
		SSLMode:     sslMode,
	}, nil
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
