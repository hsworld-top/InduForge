package repository

import (
	"strings"
	"testing"
)

func TestHistoryStorageDataPointOriginsIncludesStructuredSources(t *testing.T) {
	for _, fragment := range []string{
		"data_queries source_query",
		"data_mqtt_subscriptions direct_subscription",
		"data_mqtt_tags tag",
		"data_mqtt_subscriptions tag_subscription",
		"data_collector_points collector_point",
	} {
		if !strings.Contains(historyStorageDataPointOriginsCTE, fragment) {
			t.Fatalf("统一归属解析缺少 %q", fragment)
		}
	}
}

func TestBuildDataPointWhereClauseAccessSourceIncludesStructuredSources(t *testing.T) {
	whereSQL, args, err := buildDataPointWhereClause("project-1", DataPointListFilter{AccessSourceID: "00000000-0000-0000-0000-000000000001"})
	if err != nil {
		t.Fatalf("build where clause failed: %v", err)
	}
	for _, fragment := range []string{"data_queries source_query", "data_mqtt_subscriptions subscription", "data_mqtt_tags tag"} {
		if !strings.Contains(whereSQL, fragment) {
			t.Fatalf("接入源筛选缺少 %q", fragment)
		}
	}
	if len(args) != 2 || args[1] != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("unexpected args: %#v", args)
	}
}
