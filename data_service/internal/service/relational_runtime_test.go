package service

import (
	"database/sql"
	"reflect"
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
		"dbType":                 "sqlserver",
		"host":                   "localhost",
		"port":                   1433,
		"database":               "master",
		"username":               "sa",
		"password":               "secret",
		"encrypt":                false,
		"trustServerCertificate": true,
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
	if !config.TrustServerCertificate {
		t.Fatalf("expected trustServerCertificate to be true")
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
