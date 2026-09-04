package db

import (
	"regexp"
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

func TestRuntimeSchemaValidationCoversAuthoringRestoreBoundary(t *testing.T) {
	for table, columns := range map[string][]string{
		"projects":             {"authoring_epoch"},
		"application_versions": {"authoring_snapshot_schema", "restorable"},
		"project_deployments":  {"deletion_requested_at"},
	} {
		validated := requiredColumns[table]
		for _, column := range columns {
			if !containsString(validated, column) {
				t.Fatalf("runtime schema validation missing %s.%s", table, column)
			}
		}
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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
	for _, statement := range []string{"ALTER TABLE", "DROP TABLE", `UPDATE\s`, "DELETE FROM"} {
		if regexp.MustCompile(`(?m)^\s*`+statement).FindStringIndex(upper) != nil {
			t.Errorf("schema must be an empty-database baseline, found %s", statement)
		}
	}
}

func TestAuthoringRestoreFinalSchemaRequiresCompleteSnapshotMetadata(t *testing.T) {
	for _, expected := range []string{
		"authoring_snapshot_schema text",
		"authoring_snapshot_cipher_hash text",
		"restorable boolean NOT NULL DEFAULT false",
		"CREATE TABLE authoring_restore_tasks",
		"CREATE UNIQUE INDEX authoring_restore_tasks_active_project_uidx",
		"CREATE TABLE authoring_restore_task_events",
		"FOREIGN KEY (application_version_id, project_id, tenant_id) REFERENCES application_versions (id, project_id, tenant_id)",
		"UNIQUE (id, project_id, tenant_id)",
	} {
		if !strings.Contains(coreschema.CoreSQL, expected) {
			t.Fatalf("authoring restore final schema missing %q", expected)
		}
	}
}
