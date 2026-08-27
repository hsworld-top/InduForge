package service

import (
	"strings"
	"testing"
)

func TestValidateTDengineReadOnlySQL(t *testing.T) {
	allowed := []string{
		"SELECT * FROM metrics",
		"-- comment\nSHOW STABLES",
		"SELECT 'delete from metrics' AS text",
		"WITH latest AS (SELECT * FROM metrics) SELECT * FROM latest",
		"DESCRIBE device_metrics;",
	}
	for _, sqlText := range allowed {
		if err := validateTDengineReadOnlySQL(sqlText); err != nil {
			t.Fatalf("expected allowed SQL %q: %v", sqlText, err)
		}
	}

	blocked := []string{
		"INSERT INTO metrics VALUES(now, 1)",
		"SELECT * FROM metrics; DROP TABLE metrics",
		"WITH removed AS (DELETE FROM metrics) SELECT * FROM removed",
		"/* unterminated",
	}
	for _, sqlText := range blocked {
		if err := validateTDengineReadOnlySQL(sqlText); err == nil {
			t.Fatalf("expected blocked SQL %q", sqlText)
		}
	}
}

func TestValidateTDengineSingleSQLAllowsDMLButBlocksBoundaryAndMultipleStatements(t *testing.T) {
	allowed := []string{
		"INSERT INTO device_a VALUES(now, 1)",
		"DELETE FROM device_a WHERE ts < now - 1d",
	}
	for _, sqlText := range allowed {
		if err := validateTDengineSingleSQL(sqlText); err != nil {
			t.Fatalf("expected allowed SQL %q: %v", sqlText, err)
		}
	}
	blocked := []string{
		"SELECT * FROM device_a; DELETE FROM device_a",
		"USE another_database",
		"CREATE DATABASE another_database",
	}
	for _, sqlText := range blocked {
		if err := validateTDengineSingleSQL(sqlText); err == nil {
			t.Fatalf("expected blocked SQL %q", sqlText)
		}
	}
}

func TestBuildTDengineWSSDSN(t *testing.T) {
	dsn, err := buildTDengineDSN(map[string]any{
		"protocol": "wss", "host": "td.example.com", "port": 6041,
		"username": "root", "password": "secret", "databaseName": "demo",
		"timezone": "Asia/Shanghai", "tlsSkipVerify": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dsn, "@wss(td.example.com:6041)/demo") || !strings.Contains(dsn, "skipVerify=true") {
		t.Fatalf("unexpected wss dsn: %s", dsn)
	}
}

func TestBindTDengineQueryOnlyReplacesValuePlaceholders(t *testing.T) {
	query, err := bindTDengineQuery(
		"SELECT * FROM metrics WHERE device = ? AND note = '?' -- ?\nAND active = ?",
		[]any{"pump'a", true},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := "SELECT * FROM metrics WHERE device = 'pump''a' AND note = '?' -- ?\nAND active = TRUE"
	if query != want {
		t.Fatalf("unexpected bound query:\n%s\nwant:\n%s", query, want)
	}
}

func TestBindTDengineQueryRejectsParameterCountMismatch(t *testing.T) {
	if _, err := bindTDengineQuery("SELECT * FROM metrics WHERE device = ?", nil); err == nil {
		t.Fatal("expected parameter count error")
	}
}

func TestBindTDengineQueryDoesNotPanicOnUnclosedComment(t *testing.T) {
	query, err := bindTDengineQuery("SELECT 1 /* ?", nil)
	if err != nil {
		t.Fatal(err)
	}
	if query != "SELECT 1 /* ?" {
		t.Fatalf("unexpected query: %q", query)
	}
}
