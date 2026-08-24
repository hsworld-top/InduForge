package repository

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestProjectSnapshotAlarmJSONRoundTrip(t *testing.T) {
	groupID := "00000000-0000-0000-0000-000000000010"
	channelID := "00000000-0000-0000-0000-000000000020"
	now := time.Date(2026, time.August, 18, 0, 0, 0, 0, time.UTC)
	snapshot := ProjectSnapshot{
		AlarmGroups:          []AlarmPolicyGroupRecord{{ID: groupID, ProjectID: "project", Name: "生产线", FullPath: "生产线", CreatedAt: now, UpdatedAt: now}},
		AlarmChannels:        []AlarmNotificationChannelRecord{{ID: channelID, ProjectID: "project", Name: "值班群", ChannelType: "dingtalk", Config: map[string]any{"webhookUrl": "https://example.com"}, SecretStatus: map[string]any{"signingSecret": true}, IsEnabled: true, CreatedAt: now, UpdatedAt: now}},
		AlarmChannelSecrets:  []SnapshotAlarmChannelSecretRecord{{ChannelID: channelID, SecretKey: "signingSecret", EncryptedValue: []byte{1, 2, 3}, EncryptionKeyVersion: "v1", CreatedAt: now, UpdatedAt: now}},
		AlarmItems:           []AlarmItemRecord{{ID: "00000000-0000-0000-0000-000000000030", ProjectID: "project", DatapointID: alarmSnapshotString("dp"), Path: alarmSnapshotString("line.temperature"), GroupID: &groupID, DisplayName: "温度报警", NameKey: "温度报警", Mode: "point", AlarmType: "threshold", EvaluationMode: "highest_matching", TriggerFingerprint: strings.Repeat("a", 64), NotificationMode: "custom", NotificationChannelIDs: []string{channelID}, IsEnabled: true, Revision: 3, Contract: map[string]any{"schemaVersion": "alarm.item.v1"}, Conditions: []AlarmItemConditionRecord{{ID: "condition", Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 80.0}}}, CreatedAt: now, UpdatedAt: now}},
		AlarmHistorySettings: &AlarmHistorySettingsRecord{ProjectID: "project", IsEnabled: true, RetentionDays: nil, StoreNotificationDeliveries: true, CreatedAt: now, UpdatedAt: now},
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	var restored ProjectSnapshot
	if err = json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if len(restored.AlarmItems) != 1 || len(restored.AlarmItems[0].Conditions) != 1 {
		t.Fatalf("alarm item round trip lost normalized details: %#v", restored.AlarmItems)
	}
	if len(restored.AlarmChannelSecrets) != 1 || string(restored.AlarmChannelSecrets[0].EncryptedValue) != string([]byte{1, 2, 3}) {
		t.Fatalf("alarm encrypted secret round trip lost: %#v", restored.AlarmChannelSecrets)
	}
	if restored.AlarmHistorySettings == nil || restored.AlarmHistorySettings.RetentionDays != nil || !restored.AlarmHistorySettings.StoreNotificationDeliveries {
		t.Fatalf("alarm history settings round trip lost: %#v", restored.AlarmHistorySettings)
	}
}

func TestProjectArtifactAlarmItemContractOnlyContainsSecretReferences(t *testing.T) {
	snapshot := &ProjectSnapshot{
		AlarmItems:          []AlarmItemRecord{{ID: "alarm-item", DatapointID: alarmSnapshotString("dp-1"), Path: alarmSnapshotString("line.temperature"), DisplayName: "报警", Mode: "point", AlarmType: "threshold", EvaluationMode: "single", NotificationMode: "inherit", Contract: map[string]any{"schema": "alarm.item.v1"}}},
		AlarmChannelSecrets: []SnapshotAlarmChannelSecretRecord{{ChannelID: "channel", SecretKey: "signingSecret", EncryptedValue: []byte("ciphertext-must-not-publish"), EncryptionKeyVersion: "v1"}},
	}
	artifact := BuildProjectArtifactV1("project", snapshot, time.Now())
	payload, err := json.Marshal(artifact)
	if err != nil {
		t.Fatalf("marshal artifact: %v", err)
	}
	if artifact.Alarms.SchemaVersion != "alarm.item.v1" || len(artifact.Alarms.Items) != 1 || len(artifact.Alarms.SecretRefs) != 1 {
		t.Fatalf("unexpected alarm artifact: %#v", artifact.Alarms)
	}
	if !artifact.Alarms.HistoryStorage.Enabled || artifact.Alarms.HistoryStorage.RetentionDays == nil || *artifact.Alarms.HistoryStorage.RetentionDays != 30 || !artifact.Alarms.HistoryStorage.StoreNotificationDeliveries {
		t.Fatalf("unexpected default alarm history artifact: %#v", artifact.Alarms.HistoryStorage)
	}
	if strings.Contains(string(payload), "ciphertext-must-not-publish") || strings.Contains(string(payload), "encryptedValue") {
		t.Fatalf("artifact leaked encrypted secret payload: %s", payload)
	}
	if !strings.Contains(string(payload), `"alarmItemId":"alarm-item"`) || !strings.Contains(string(payload), `"datapointId":"dp-1"`) || !strings.Contains(string(payload), `"evaluationMode":"single"`) {
		t.Fatalf("artifact lost stable alarm identity or evaluation mode: %s", payload)
	}
}

func alarmSnapshotString(value string) *string { return &value }

func TestProjectArtifactUsesExplicitAlarmHistorySettings(t *testing.T) {
	snapshot := &ProjectSnapshot{
		AlarmHistorySettings: &AlarmHistorySettingsRecord{
			ProjectID:                   "project",
			IsEnabled:                   false,
			RetentionDays:               nil,
			StoreNotificationDeliveries: false,
		},
	}
	artifact := BuildProjectArtifactV1("project", snapshot, time.Now())
	if artifact.Alarms.HistoryStorage.Enabled || artifact.Alarms.HistoryStorage.RetentionDays != nil || artifact.Alarms.HistoryStorage.StoreNotificationDeliveries {
		t.Fatalf("unexpected explicit alarm history artifact: %#v", artifact.Alarms.HistoryStorage)
	}
}
