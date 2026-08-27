package service

import (
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseRelationalRuntimeConfigSupportsMySQL(t *testing.T) {
	config, err := parseRelationalRuntimeConfig(map[string]any{
		"dbType":   "mysql",
		"host":     "localhost",
		"port":     3306,
		"database": "demo",
		"username": "root",
		"password": "secret",
		"charset":  "utf8mb4",
	})
	if err != nil {
		t.Fatalf("expected mysql config to parse, got error: %v", err)
	}

	if config.DBType != "mysql" {
		t.Fatalf("expected dbType mysql, got %s", config.DBType)
	}
	if config.Port != 3306 {
		t.Fatalf("expected port 3306, got %d", config.Port)
	}
	if config.Charset != "utf8mb4" {
		t.Fatalf("expected charset utf8mb4, got %s", config.Charset)
	}
}

func TestParseRelationalRuntimeConfigSupportsSQLServer(t *testing.T) {
	config, err := parseRelationalRuntimeConfig(map[string]any{
		"dbType":    "sqlserver",
		"host":      "localhost",
		"port":      1433,
		"database":  "master",
		"username":  "sa",
		"password":  "secret",
		"sslConfig": map[string]any{"mode": "disable"},
	})
	if err != nil {
		t.Fatalf("expected sqlserver config to parse, got error: %v", err)
	}

	if config.DBType != "sqlserver" {
		t.Fatalf("expected dbType sqlserver, got %s", config.DBType)
	}
	if config.Port != 1433 {
		t.Fatalf("expected port 1433, got %d", config.Port)
	}
	if config.SSLMode != "disable" {
		t.Fatalf("expected SQL Server TLS mode disable, got %s", config.SSLMode)
	}
}

func TestParseRelationalRuntimeConfigUsesUnifiedTLSConfig(t *testing.T) {
	tests := []struct {
		name     string
		dbType   string
		mode     string
		wantFail bool
	}{
		{name: "postgres verify ca", dbType: "postgresql", mode: "verify-ca"},
		{name: "mysql prefer", dbType: "mysql", mode: "prefer"},
		{name: "sqlserver verify full", dbType: "sqlserver", mode: "verify-full"},
		{name: "sqlserver rejects prefer", dbType: "sqlserver", mode: "prefer", wantFail: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := parseRelationalRuntimeConfig(map[string]any{
				"dbType": test.dbType, "host": "db.example.com", "database": "demo",
				"username": "user", "sslConfig": map[string]any{"mode": test.mode},
			})
			if (err != nil) != test.wantFail {
				t.Fatalf("parse error = %v, wantFail = %v", err, test.wantFail)
			}
			if err == nil && config.SSLMode != test.mode {
				t.Fatalf("expected mode %s, got %s", test.mode, config.SSLMode)
			}
		})
	}
}

func TestBuildRelationalTLSConfigValidatesCertificatePair(t *testing.T) {
	_, err := buildRelationalTLSConfig(relationalRuntimeConfig{
		Host: "db.example.com", SSLMode: "verify-full", SSLCert: "certificate-only",
	})
	if err == nil || !strings.Contains(err.Error(), "必须同时配置") {
		t.Fatalf("expected certificate pair validation error, got %v", err)
	}
}

func TestBuildRelationalTLSConfigMapsVerificationModes(t *testing.T) {
	requireConfig, err := buildRelationalTLSConfig(relationalRuntimeConfig{Host: "db.example.com", SSLMode: "require"})
	if err != nil || requireConfig == nil || !requireConfig.InsecureSkipVerify {
		t.Fatalf("require mode should encrypt without certificate verification: %#v %v", requireConfig, err)
	}
	verifyConfig, err := buildRelationalTLSConfig(relationalRuntimeConfig{Host: "db.example.com", SSLMode: "verify-full"})
	if err != nil || verifyConfig == nil || verifyConfig.InsecureSkipVerify || verifyConfig.ServerName != "db.example.com" {
		t.Fatalf("verify-full mode should verify hostname: %#v %v", verifyConfig, err)
	}
}

func TestPrepareSQLServerNamedArgsUsesPlaceholderOrder(t *testing.T) {
	args, err := prepareSQLServerNamedArgs(
		"SELECT * FROM devices WHERE status = @status AND code = @code",
		[]any{"active", "A-01"},
	)
	if err != nil {
		t.Fatalf("expected sqlserver named args to build, got error: %v", err)
	}

	expected := []any{
		sql.Named("status", "active"),
		sql.Named("code", "A-01"),
	}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("unexpected named args: %#v", args)
	}
}

func TestSQLServerMetadataQueriesUseNamedArguments(t *testing.T) {
	query, args := relationalListTablesQuery("sqlserver", "dbo")
	if !strings.Contains(query, "@schema") || !reflect.DeepEqual(adaptRuntimeArgs("sqlserver", query, args), []any{sql.Named("schema", "dbo")}) {
		t.Fatalf("unexpected table-list query or args: %q %#v", query, args)
	}

	metadataQueries := []struct {
		name  string
		build func(string, string, string) (string, []any)
	}{
		{name: "columns", build: relationalTableColumnsQuery},
		{name: "indexes", build: relationalTableIndexesQuery},
		{name: "foreign keys", build: relationalTableForeignKeysQuery},
	}
	expected := []any{sql.Named("schema", "dbo"), sql.Named("table", "sqltest_types")}
	for _, test := range metadataQueries {
		t.Run(test.name, func(t *testing.T) {
			query, args := test.build("sqlserver", "dbo", "sqltest_types")
			if !strings.Contains(query, "@schema") || !strings.Contains(query, "@table") || !reflect.DeepEqual(adaptRuntimeArgs("sqlserver", query, args), expected) {
				t.Fatalf("unexpected metadata query or args: %q %#v", query, args)
			}
		})
	}
}

func TestValidateSQLParameterCountByDialect(t *testing.T) {
	tests := []struct {
		name, dialect, query string
		args                 []any
		wantError            bool
	}{
		{name: "mysql values", dialect: "mysql", query: "SELECT * FROM devices WHERE id = ? AND note = '?'", args: []any{1}},
		{name: "postgres values", dialect: "postgresql", query: "SELECT * FROM devices WHERE id = $1 AND state = $2", args: []any{1, "ok"}},
		{name: "postgres function body positional arg ignored", dialect: "postgresql", query: "CREATE FUNCTION double_value(integer) RETURNS integer LANGUAGE sql AS $body$ SELECT $1 * 2 $body$", args: nil},
		{name: "postgres outer arg with function body", dialect: "postgresql", query: "SELECT $1; DO $$ BEGIN RAISE NOTICE '$2'; END $$", args: []any{1}},
		{name: "postgres gap", dialect: "postgresql", query: "SELECT * FROM devices WHERE id = $2", args: []any{1, 2}, wantError: true},
		{name: "sqlserver repeated name", dialect: "sqlserver", query: "SELECT * FROM devices WHERE id = @p1 OR parent_id = @p1", args: []any{1}},
		{name: "comment ignored", dialect: "mysql", query: "SELECT 1 -- ?\nWHERE id = ?", args: []any{1}},
		{name: "count mismatch", dialect: "mysql", query: "SELECT * FROM devices WHERE id = ?", args: nil, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateSQLParameterCount(test.dialect, test.query, test.args)
			if (err != nil) != test.wantError {
				t.Fatalf("validateSQLParameterCount() error = %v, wantError = %v", err, test.wantError)
			}
		})
	}
}

func TestSQLExecutionErrorMessageKeepsDriverDetailAndLimitsLength(t *testing.T) {
	message := sqlExecutionErrorMessage("执行 SQL 失败", errors.New("duplicate key value violates unique constraint sqltest_types_pkey"))
	if !strings.Contains(message, "duplicate key") || !strings.HasPrefix(message, "执行 SQL 失败：") {
		t.Fatalf("unexpected message: %s", message)
	}
	longMessage := sqlExecutionErrorMessage("执行 SQL 失败", errors.New(strings.Repeat("x", 600)))
	if len([]rune(longMessage)) > 440 || !strings.HasSuffix(longMessage, "…") {
		t.Fatalf("expected limited message, got length %d", len([]rune(longMessage)))
	}
}
