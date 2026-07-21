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
	for _, table := range []string{
		"data_connections",
		"data_collector_connections",
		"data_collector_points",
		"data_collector_point_debug_snapshots",
		"collector_dev_agents",
		"data_storage_policies",
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

func TestCollectorConnectionSchemaHasNoConnectionEnabledColumn(t *testing.T) {
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
	if strings.Contains(definition, "enabled boolean") {
		t.Fatal("工业连接表不应包含连接级 enabled 字段")
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
