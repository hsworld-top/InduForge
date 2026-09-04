package schema

import (
	"regexp"
	"strings"
	"testing"
)

func TestEmbeddedSchemaContainsFinalStructure(t *testing.T) {
	payload, err := Files.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	baseline := string(payload)
	for _, extension := range []string{"pgcrypto", "pg_trgm"} {
		if !strings.Contains(baseline, "CREATE EXTENSION IF NOT EXISTS "+extension) {
			t.Fatalf("数据库最终基线缺少扩展 %s", extension)
		}
	}
	for _, table := range []string{
		"data_project_tenant_bindings",
		"data_connections",
		"data_collector_connections",
		"data_collector_points",
		"data_collector_point_debug_snapshots",
		"collector_dev_agents",
		"data_history_storage_configs",
		"data_history_storage_targets",
		"data_source_output_mappings",
		"data_compute_dependencies",
		"data_authoring_fences",
		"data_compute_unit_outputs",
		"data_connection_secrets",
	} {
		if !strings.Contains(baseline, "CREATE TABLE "+table+" (") {
			t.Fatalf("数据库结构基线缺少表 %s", table)
		}
	}
	forbidden := regexp.MustCompile(`(?m)^(UPDATE|DELETE FROM|DROP TABLE|TRUNCATE)\b`)
	if forbidden.MatchString(baseline) {
		t.Fatalf("数据库结构基线不得包含历史数据迁移或删除逻辑")
	}
	if strings.Contains(baseline, "schema_migrations") {
		t.Fatalf("数据库结构基线不得包含迁移版本表")
	}
	for _, legacy := range []string{"alarm_low", "alarm_high", "data_alarm_policies", "data_alarm_configurations", "data_storage_policies"} {
		if strings.Contains(baseline, legacy) {
			t.Fatalf("数据库最终基线仍包含旧字段或旧模型 %s", legacy)
		}
	}
	if count := strings.Count(baseline, "CREATE TABLE "); count != 60 {
		t.Fatalf("数据库最终基线表数量应为 60，实际为 %d", count)
	}
}

func TestAuthoringFenceCompatibilityRequirementsRemainInFinalSchema(t *testing.T) {
	payload, err := Files.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	baseline := string(payload)
	for _, expected := range []string{
		"authoring_epoch bigint DEFAULT 1 NOT NULL",
		"CREATE TABLE data_authoring_fences (",
		"token_hash bytea NOT NULL",
	} {
		if !strings.Contains(baseline, expected) {
			t.Fatalf("authoring fence compatibility baseline missing %q", expected)
		}
	}
}

func TestDatapointAndOutputSchemasUseCanonicalTypes(t *testing.T) {
	payload, err := Files.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	baseline := string(payload)
	for _, table := range []string{"data_points", "data_mqtt_tags", "data_source_output_mappings", "data_compute_unit_outputs"} {
		start := strings.Index(baseline, "CREATE TABLE "+table+" (")
		if start < 0 {
			t.Fatalf("数据库结构基线缺少表 %s", table)
		}
		end := strings.Index(baseline[start:], "\n);")
		if end < 0 {
			t.Fatalf("无法读取表 %s 定义", table)
		}
		definition := baseline[start : start+end]
		for _, expected := range []string{"'bool'", "'int8'", "'uint64'", "'float32'", "'float64'", "'decimal'", "'bytes'", "'datetime'", "'object'", "'array'"} {
			if !strings.Contains(definition, expected) {
				t.Fatalf("表 %s 缺少规范数据类型 %s", table, expected)
			}
		}
		for _, legacy := range []string{"'number'", "'double'", "'integer'", "'boolean'", "'json'"} {
			if strings.Contains(definition, legacy) {
				t.Fatalf("表 %s 仍接受旧类型别名 %s", table, legacy)
			}
		}
	}
}

func TestCollectorPointDebugSnapshotSchemaUsesCascadeDelete(t *testing.T) {
	payload, err := Files.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	baseline := string(payload)
	if !strings.Contains(baseline, "CREATE TABLE data_collector_point_debug_snapshots (") {
		t.Fatal("数据库结构基线缺少采集变量调试快照表")
	}
	if !strings.Contains(baseline, "FOREIGN KEY (point_id) REFERENCES data_collector_points(id) ON DELETE CASCADE") {
		t.Fatal("采集变量删除时必须级联删除调试快照")
	}
}

func TestCollectorPointSchemaEnforcesConnectionNameUniqueness(t *testing.T) {
	payload, err := Files.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	baseline := string(payload)
	expected := "CREATE UNIQUE INDEX data_collector_points_connection_name_key ON data_collector_points USING btree (project_id, connection_id, lower(name));"
	if !strings.Contains(baseline, expected) {
		t.Fatal("采集变量名称必须在同一连接内忽略大小写唯一")
	}
}

func TestCollectorConnectionSchemaHasEnabledAndAcquisitionDefaults(t *testing.T) {
	payload, err := Files.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	baseline := string(payload)
	start := strings.Index(baseline, "CREATE TABLE data_collector_connections (")
	end := strings.Index(baseline[start:], "\n);")
	if start < 0 || end < 0 {
		t.Fatal("未找到工业连接表定义")
	}
	definition := baseline[start : start+end]
	for _, expected := range []string{"is_enabled boolean", "default_acquisition jsonb"} {
		if !strings.Contains(definition, expected) {
			t.Fatalf("工业连接表缺少阶段 D 最终字段 %s", expected)
		}
	}
}
func TestCollectorTaskSchemaAllowsConnectionSessionOperations(t *testing.T) {
	payload, err := Files.ReadFile("schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	baseline := string(payload)
	start := strings.Index(baseline, "CREATE TABLE collector_dev_tasks (")
	end := strings.Index(baseline[start:], "\n);")
	if start < 0 || end < 0 {
		t.Fatal("未找到采集调试任务表定义")
	}
	definition := baseline[start : start+end]
	for _, operation := range []string{"connection.open", "connection.close"} {
		if !strings.Contains(definition, operation) {
			t.Fatalf("采集调试任务操作约束缺少 %s", operation)
		}
	}
}
