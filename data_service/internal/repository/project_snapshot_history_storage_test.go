package repository

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestProjectSnapshotHistoryStorageJSONRoundTrip(t *testing.T) {
	accessSourceID := "00000000-0000-0000-0000-000000000001"
	datapointID := "00000000-0000-0000-0000-000000000002"
	days := int64(30)
	now := time.Date(2026, time.August, 18, 0, 0, 0, 0, time.UTC)
	snapshot := ProjectSnapshot{HistoryStorage: []HistoryStorageConfigRecord{
		{
			ID: "00000000-0000-0000-0000-000000000010", ProjectID: "00000000-0000-0000-0000-000000000099",
			AccessSourceID: &accessSourceID, IsEnabled: true, WriteMode: "on_change", OfflineBehavior: "store_stale",
			CreatedBy: accessSourceID, CreatedAt: now, UpdatedAt: now,
			Targets: []HistoryStorageTargetRecord{
				{ID: "00000000-0000-0000-0000-000000000020", ConnectionID: "00000000-0000-0000-0000-000000000030", IsPrimary: true, SortOrder: 0, RetentionDays: nil},
				{ID: "00000000-0000-0000-0000-000000000021", ConnectionID: "00000000-0000-0000-0000-000000000031", SortOrder: 1, RetentionDays: &days},
			},
		},
		{
			ID: "00000000-0000-0000-0000-000000000011", ProjectID: "00000000-0000-0000-0000-000000000099",
			DatapointID: &datapointID, IsEnabled: false, WriteMode: "on_change", OfflineBehavior: "store_stale",
			CreatedBy: accessSourceID, CreatedAt: now, UpdatedAt: now,
		},
	}}

	payload, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	var restored ProjectSnapshot
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if len(restored.HistoryStorage) != 2 || len(restored.HistoryStorage[0].Targets) != 2 {
		t.Fatalf("unexpected history storage round trip: %#v", restored.HistoryStorage)
	}
	if restored.HistoryStorage[0].Targets[0].RetentionDays != nil || restored.HistoryStorage[0].Targets[1].SortOrder != 1 {
		t.Fatalf("永久保留或目标顺序丢失: %#v", restored.HistoryStorage[0].Targets)
	}
	if restored.HistoryStorage[1].DatapointID == nil || restored.HistoryStorage[1].IsEnabled {
		t.Fatalf("单点关闭覆盖丢失: %#v", restored.HistoryStorage[1])
	}
}

func TestProjectArtifactExcludesDatapointHistoryStorage(t *testing.T) {
	snapshot := &ProjectSnapshot{HistoryStorage: []HistoryStorageConfigRecord{{ID: "history-config-1"}}}
	artifact := BuildProjectArtifactV1("project-1", snapshot, time.Now())
	payload, err := json.Marshal(artifact)
	if err != nil {
		t.Fatalf("marshal artifact: %v", err)
	}
	if strings.Contains(string(payload), "history-config-1") {
		t.Fatalf("发布 Artifact 不应包含数据点历史存储配置: %s", payload)
	}
	if !strings.Contains(string(payload), `"historyStorage":{"enabled":true,"retentionDays":30,"storeNotificationDeliveries":true}`) {
		t.Fatalf("发布 Artifact 应包含报警历史默认设置: %s", payload)
	}
}
