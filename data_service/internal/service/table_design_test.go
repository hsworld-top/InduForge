package service

import (
	"strings"
	"testing"
)

func TestNormalizeCreateTableInputRejectsInvalidIdentifier(t *testing.T) {
	_, err := normalizeCreateTableInput("relational", CreateRelationalTableInput{
		Name: "bad table",
		Columns: []CreateRelationalTableColumnInput{
			{Name: "id", Type: "bigint", Primary: true},
		},
	})
	if err == nil {
		t.Fatalf("expected invalid identifier error")
	}
}

func TestNormalizeCreateTableInputAllowsSuperTableOnlyForTimeseries(t *testing.T) {
	_, err := normalizeCreateTableInput("postgresql", CreateRelationalTableInput{
		Name: "metrics",
		Kind: "super_table",
		Columns: []CreateRelationalTableColumnInput{
			{Name: "ts", Type: "timestamptz", Primary: true},
		},
	})
	if err == nil {
		t.Fatalf("expected super table scope error")
	}
}

func TestBuildPostgresCreateTableDDL(t *testing.T) {
	input, err := normalizeCreateTableInput("builtin.relation", CreateRelationalTableInput{
		Name: "device_data",
		Columns: []CreateRelationalTableColumnInput{
			{Name: "id", Type: "bigint", Primary: true, AutoIncrement: true},
			{Name: "name", Type: "varchar", Length: intPointer(80), Nullable: false},
		},
		Indexes: []CreateRelationalTableIndexInput{
			{Name: "idx_device_data_name", Type: "index", Columns: []string{"name"}},
		},
	})
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	ddl, err := buildCreateTableDDL("postgresql", "public", input)
	if err != nil {
		t.Fatalf("build ddl failed: %v", err)
	}
	for _, expected := range []string{
		`CREATE TABLE "public"."device_data"`,
		`"id" bigserial NOT NULL`,
		`PRIMARY KEY ("id")`,
		`CREATE INDEX "idx_device_data_name"`,
	} {
		if !strings.Contains(ddl, expected) {
			t.Fatalf("ddl missing %q: %s", expected, ddl)
		}
	}
}

func TestBuildPostgresUpdateTableDDLAllowsSafeChanges(t *testing.T) {
	comment := "旧备注"
	current := &RelationalTableStructure{
		Columns: []RelationalTableColumn{
			{Name: "id", Type: "bigint", Nullable: false, IsPrimary: true, AutoIncrement: true},
			{Name: "name", Type: "character varying", Nullable: true, Comment: &comment},
			{Name: "legacy", Type: "text", Nullable: true},
		},
		Indexes: []RelationalIndex{
			{Name: "idx_device_name", Type: "INDEX", Method: "btree", Columns: []string{"name"}},
		},
	}
	input, err := normalizeUpdateTableInput(UpdateRelationalTableInput{
		Columns: []CreateRelationalTableColumnInput{
			{Name: "id", Type: "bigint", Primary: true, AutoIncrement: true, Nullable: false},
			{Name: "name", Type: "varchar", Nullable: false, Comment: "新备注"},
			{Name: "score", Type: "double", Nullable: true},
		},
		Indexes: []CreateRelationalTableIndexInput{
			{Name: "idx_device_score", Type: "index", Columns: []string{"score"}},
		},
	})
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	ddl, err := buildPostgresUpdateTableDDL("public", "device_data", current, input)
	if err != nil {
		t.Fatalf("build ddl failed: %v", err)
	}
	joined := strings.Join(ddl, ";\n")
	for _, expected := range []string{
		`ALTER TABLE "public"."device_data" ALTER COLUMN "name" SET NOT NULL`,
		`COMMENT ON COLUMN "public"."device_data"."name" IS '新备注'`,
		`ALTER TABLE "public"."device_data" ADD COLUMN "score" double precision`,
		`ALTER TABLE "public"."device_data" DROP COLUMN "legacy"`,
		`DROP INDEX "public"."idx_device_name"`,
		`CREATE INDEX "idx_device_score"`,
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("ddl missing %q: %s", expected, joined)
		}
	}
}

func TestBuildPostgresUpdateTableDDLRejectsUnsafeChanges(t *testing.T) {
	current := &RelationalTableStructure{
		Columns: []RelationalTableColumn{
			{Name: "id", Type: "bigint", Nullable: false, IsPrimary: true},
			{Name: "name", Type: "text", Nullable: true},
		},
	}
	tests := []struct {
		name  string
		input UpdateRelationalTableInput
	}{
		{
			name: "change type",
			input: UpdateRelationalTableInput{Columns: []CreateRelationalTableColumnInput{
				{Name: "id", Type: "bigint", Primary: true},
				{Name: "name", Type: "int", Nullable: true},
			}},
		},
		{
			name: "drop primary column",
			input: UpdateRelationalTableInput{Columns: []CreateRelationalTableColumnInput{
				{Name: "name", Type: "text", Nullable: true},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := normalizeUpdateTableInput(tt.input)
			if err != nil {
				t.Fatalf("normalize failed: %v", err)
			}
			if _, err := buildPostgresUpdateTableDDL("public", "device_data", current, input); err == nil {
				t.Fatalf("expected unsafe change error")
			}
		})
	}
}

func TestEnsureSQLWorkbenchDatabaseBoundary(t *testing.T) {
	allowed := []string{
		"SHOW DATABASES",
		"SELECT datname FROM pg_database",
		"INSERT INTO device_data(name) VALUES ('a')",
		"DROP TABLE device_data",
		"ALTER TABLE device_data ADD COLUMN value double precision",
	}
	for _, sqlText := range allowed {
		if err := ensureSQLWorkbenchDatabaseBoundary(sqlText); err != nil {
			t.Fatalf("expected %q to be allowed: %v", sqlText, err)
		}
	}

	blocked := []string{
		"DROP DATABASE prod",
		"create database demo",
		"ALTER DATABASE demo SET timezone TO 'UTC'",
		"DROP SCHEMA public",
		"CREATE SCHEMA staging",
		"USE other_database",
	}
	for _, sqlText := range blocked {
		if err := ensureSQLWorkbenchDatabaseBoundary(sqlText); err == nil {
			t.Fatalf("expected %q to be blocked", sqlText)
		}
	}
}

func intPointer(value int) *int {
	return &value
}
