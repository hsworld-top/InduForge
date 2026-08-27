package db

import (
	"strings"
	"testing"

	coreschema "github.com/indu-forge/dev_core/db/schema"
)

func TestSchemaContainsRequiredControlPlaneTables(t *testing.T) {
	for _, table := range requiredTables {
		if !strings.Contains(coreschema.CoreSQL, "CREATE TABLE "+table+" ") {
			t.Errorf("schema missing table %s", table)
		}
	}
}

func TestSchemaDoesNotContainLegacyDesignerTables(t *testing.T) {
	for _, table := range []string{"design_pages", "design_project_settings", "design_asset_folders", "design_assets"} {
		if strings.Contains(coreschema.CoreSQL, table) {
			t.Errorf("schema must not contain legacy designer table %s", table)
		}
	}
}

func TestSceneContentUsesExactlyOneStorageBackend(t *testing.T) {
	expected := "CHECK ((json_content IS NOT NULL AND object_key IS NULL) OR (json_content IS NULL AND object_key IS NOT NULL))"
	if !strings.Contains(coreschema.CoreSQL, expected) {
		t.Fatal("scene content objects must use exactly one storage backend")
	}
}

func TestSceneDraftFilesAreScopedToOneScene(t *testing.T) {
	for _, expected := range []string{
		"scene_document_id uuid NOT NULL",
		"UNIQUE (scene_document_id, logical_path)",
		"CREATE TABLE scene_asset_bindings",
		"CREATE TABLE scene_revision_assets",
	} {
		if !strings.Contains(coreschema.CoreSQL, expected) {
			t.Fatalf("scene asset schema missing %q", expected)
		}
	}
}

func TestDeletedSceneFileNodesReleaseContentObjects(t *testing.T) {
	expected := "deleted_at IS NOT NULL AND content_object_id IS NULL AND content_hash IS NULL"
	if !strings.Contains(coreschema.CoreSQL, expected) {
		t.Fatal("deleted scene file nodes must release their content object references")
	}
}

func TestActiveSceneNamesAreUniqueWithinProject(t *testing.T) {
	expected := "CREATE UNIQUE INDEX scene_documents_project_name_active_uidx ON scene_documents (project_id, lower(name)) WHERE deleted_at IS NULL"
	if !strings.Contains(coreschema.CoreSQL, expected) {
		t.Fatal("active scene names must be case-insensitively unique within a project")
	}
}

func TestSchemaDoesNotContainRuntimeMigrationStatements(t *testing.T) {
	upper := strings.ToUpper(coreschema.CoreSQL)
	for _, statement := range []string{"ALTER TABLE", "DROP TABLE", "UPDATE ", "DELETE FROM"} {
		if strings.Contains(upper, statement) {
			t.Errorf("schema must be an empty-database baseline, found %s", statement)
		}
	}
}
