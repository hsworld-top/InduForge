package service

import "testing"

func TestCanonicalSQLColumnType(t *testing.T) {
	tests := map[string]string{
		"varchar":         "string",
		"text":            "string",
		"bool":            "bool",
		"int4":            "int32",
		"BIGINT UNSIGNED": "uint64",
		"numeric":         "decimal",
		"float8":          "float64",
		"timestamptz":     "datetime",
		"jsonb":           "object",
		"_varchar":        "array",
		"unknown_db_type": "string",
		"":                "string",
	}
	for databaseType, expected := range tests {
		if actual := canonicalSQLColumnType(databaseType); actual != expected {
			t.Fatalf("canonicalSQLColumnType(%q) = %q, want %q", databaseType, actual, expected)
		}
	}
}

func TestCanonicalSQLColumnTypesKeepsColumnAlignment(t *testing.T) {
	actual := canonicalSQLColumnTypes([]string{"name", "count", "missing"}, []string{"varchar", "int8"})
	if actual["name"] != "string" || actual["count"] != "int64" || actual["missing"] != "string" {
		t.Fatalf("unexpected canonical column types: %#v", actual)
	}
}

func TestBuiltinSQLResultToRelationalNormalizesNilColumnTypes(t *testing.T) {
	result := builtinSQLResultToRelational(&BuiltinSQLExecuteResult{
		Columns:  []string{},
		Rows:     []map[string]any{},
		RowCount: 1,
	})
	if result.ColumnTypes == nil || len(result.ColumnTypes) != 0 {
		t.Fatalf("expected empty column type map, got %#v", result.ColumnTypes)
	}
}

func TestSQLWorkbenchStatementReturnsRows(t *testing.T) {
	returnsRows := []string{
		"SELECT * FROM devices",
		"EXPLAIN SELECT * FROM devices",
		"VALUES (1), (2)",
		"INSERT INTO devices(name) VALUES ('returning') RETURNING id, name",
		"UPDATE devices SET active = true OUTPUT inserted.id",
		"DELETE FROM devices WHERE id = 1 RETURNING id",
	}
	for _, sqlText := range returnsRows {
		if !sqlWorkbenchStatementReturnsRows(sqlText) {
			t.Fatalf("expected rows for %q", sqlText)
		}
	}

	commands := []string{
		"CREATE TABLE devices(id bigint)",
		"INSERT INTO devices(name) VALUES ('returning')",
		"UPDATE devices SET active = true",
		"DELETE FROM devices WHERE id = 1",
	}
	for _, sqlText := range commands {
		if sqlWorkbenchStatementReturnsRows(sqlText) {
			t.Fatalf("expected command result for %q", sqlText)
		}
	}
}
