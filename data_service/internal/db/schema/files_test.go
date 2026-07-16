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
