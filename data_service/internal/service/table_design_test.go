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

func intPointer(value int) *int {
	return &value
}
