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
		AlarmGroups:          []AlarmGroupRecord{{ID: groupID, ProjectID: "project", Name: "生产线", FullPath: "生产线", CreatedAt: now, UpdatedAt: now}},
		AlarmChannels:        []AlarmNotificationChannelRecord{{ID: channelID, ProjectID: "project", Name: "值班群", ChannelType: "dingtalk", Config: map[string]any{"webhookUrl": "https://example.com"}, SecretStatus: map[string]any{"signingSecret": true}, IsEnabled: true, CreatedAt: now, UpdatedAt: now}},
		AlarmChannelSecrets:  []SnapshotAlarmChannelSecretRecord{{ChannelID: channelID, SecretKey: "signingSecret", EncryptedValue: []byte{1, 2, 3}, EncryptionKeyVersion: "v1", CreatedAt: now, UpdatedAt: now}},
		AlarmItems:           []AlarmItemRecord{{ID: "00000000-0000-0000-0000-000000000030", ProjectID: "project", DatapointID: alarmSnapshotString("dp"), Path: alarmSnapshotString("line.temperature"), GroupID: &groupID, DisplayName: "质量报警", NameKey: "质量报警", Mode: "point", AlarmType: "quality", EvaluationMode: "single", TriggerFingerprint: strings.Repeat("a", 64), NotificationMode: "custom", NotificationChannelIDs: []string{channelID}, IsEnabled: true, Revision: 3, Contract: map[string]any{"schemaVersion": "alarm.item.v1"}, Conditions: []AlarmItemConditionRecord{{ID: "condition", Kind: "quality", Operator: "in", Severity: "warning", Params: map[string]any{"qualities": []any{"bad", "unknown"}}}}, CreatedAt: now, UpdatedAt: now}},
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
	if restored.AlarmItems[0].Conditions[0].Kind != "quality" {
		t.Fatalf("alarm condition kind round trip lost: %#v", restored.AlarmItems[0].Conditions)
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
		AlarmItems:          []AlarmItemRecord{{ID: "alarm-item", DatapointID: alarmSnapshotString("dp-1"), Path: alarmSnapshotString("line.temperature"), DisplayName: "数据陈旧报警", Mode: "point", AlarmType: "stale", EvaluationMode: "single", NotificationMode: "inherit", Contract: map[string]any{"schema": "alarm.item.v1"}, Conditions: []AlarmItemConditionRecord{{ID: "condition", Kind: "stale", Operator: "age_gte", Severity: "warning", Params: map[string]any{"maxAgeMs": 60000.0}}}}},
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
	if !strings.Contains(string(payload), `"kind":"stale"`) || !strings.Contains(string(payload), `"maxAgeMs":60000`) {
		t.Fatalf("artifact lost stale alarm semantics: %s", payload)
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

func TestProjectSnapshotStrongOutputsAndCollectorConfigRoundTrip(t *testing.T) {
	snapshot := ProjectSnapshot{
		Queries: []QueryRecord{{
			ID: "query-1", ProjectID: "project", ConnectionID: "connection-1", Name: "温度查询",
			Outputs: []SourceOutputMappingRecord{{
				ID: "mapping-1", DataPointID: "point-1", DataPointPath: "query.temperature", Key: "temperature",
				DisplayName: "温度", Selector: SourceOutputSelectorRecord{Kind: "path", Segments: []any{"data.with.dot", 0, "value"}},
				DataType: "float32", SortOrder: 0,
			}},
		}},
		HTTPRequests: []SnapshotHTTPRequestRecord{{
			ID: "http-1", ConnectionID: "connection-http", Name: "HTTP 温度", Method: "GET", URL: "https://example.invalid/value", Enabled: true,
			Outputs: []SourceOutputMappingRecord{{ID: "mapping-http", DataPointID: "point-http", Key: "temperature", DisplayName: "温度", Selector: SourceOutputSelectorRecord{Kind: "path", Segments: []any{"payload", "temperature"}}, DataType: "float32"}},
		}},
		WebSocketSessions: []SnapshotWebSocketSessionRecord{{
			ID: "ws-1", ConnectionID: "connection-ws", Name: "WS 温度", URL: "wss://example.invalid/stream", Enabled: true,
			Outputs: []SourceOutputMappingRecord{{ID: "mapping-ws", DataPointID: "point-ws", Key: "temperature", DisplayName: "温度", Selector: SourceOutputSelectorRecord{Kind: "path", Segments: []any{"items", 0, "temperature"}}, DataType: "float32"}},
		}},
		RealtimeKeys: []SnapshotRealtimeKeyRecord{{
			ID: "key-1", ConnectionID: "connection-redis", Provider: "redis", KeyPath: "factory:temperature", RedisType: "string", ValueType: "json",
			Outputs: []SourceOutputMappingRecord{{ID: "mapping-key", DataPointID: "point-key", Key: "temperature", DisplayName: "温度", Selector: SourceOutputSelectorRecord{Kind: "path", Segments: []any{"value.with.dot"}}, DataType: "float32"}},
		}},
		ComputeUnits: []ComputeUnitRecord{{
			ID: "compute-1", Name: "温度换算", Language: "js", ScriptCode: "return { celsius: 21 }", TriggerType: "manual", TriggerConfig: map[string]any{}, InputBindings: map[string]any{}, Dependencies: []any{"compute-upstream"}, IsEnabled: true,
			Outputs: []ComputeOutputRecord{{ID: "compute-output-1", DatapointID: "point-compute", OutputKey: "celsius", Name: "摄氏温度", Path: "calc.temperature.celsius", DataType: "float32", NullPolicy: "skip", SortOrder: 0}},
		}},
		HistoryStorage: []HistoryStorageConfigRecord{{ID: "history-compute", ComputeUnitID: snapshotString("compute-1"), IsEnabled: true, WriteMode: "on_change", OfflineBehavior: "store_stale"}},
		CollectorConnections: []SnapshotCollectorConnectionRecord{{
			ID: "collector-1", Name: "S7", Code: "s7", ProtocolFamily: "siemens", DriverID: "siemens.s7-tcp",
			DriverVersion: "1.0.0", SchemaVersion: 1, Config: map[string]any{"host": "127.0.0.1"},
			Metadata: map[string]any{}, IsEnabled: true,
			DefaultAcquisition: map[string]any{"intervalMs": 1000.0, "timeoutMs": 3000.0, "retryCount": 1.0},
		}},
		CollectorPoints: []SnapshotCollectorPointRecord{{
			ID: "collector-point-1", ConnectionID: "collector-1", Code: "temperature", Name: "温度",
			Address: map[string]any{"area": "DB", "dbNumber": 1.0, "byteOffset": 0.0}, AddressText: "DB1.DBD0",
			AddressSchemaVersion: 1, DataType: "float32", ElementCount: 1, ReadOptions: map[string]any{},
			AcquisitionMode: "override", AcquisitionOverrides: map[string]any{"intervalMs": 5000.0}, Enabled: true, Metadata: map[string]any{},
		}},
		CollectorSecrets: []SnapshotCollectorSecretRecord{{
			ConnectionID: "collector-1", SecretKey: "password", EncryptedValue: []byte("ciphertext"), EncryptionKeyVersion: "v1",
		}},
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var restored ProjectSnapshot
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatal(err)
	}
	segments := restored.Queries[0].Outputs[0].Selector.Segments
	if len(segments) != 3 || segments[0] != "data.with.dot" || segments[1] != float64(0) {
		t.Fatalf("strong output selector segments lost: %#v", segments)
	}
	if restored.CollectorConnections[0].DefaultAcquisition["intervalMs"] != float64(1000) || restored.CollectorPoints[0].AcquisitionMode != "override" {
		t.Fatalf("collector acquisition config lost: %#v %#v", restored.CollectorConnections, restored.CollectorPoints)
	}
	if restored.HTTPRequests[0].Outputs[0].Selector.Segments[0] != "payload" || restored.WebSocketSessions[0].Outputs[0].Selector.Segments[1] != float64(0) || restored.RealtimeKeys[0].Outputs[0].Selector.Segments[0] != "value.with.dot" {
		t.Fatalf("workbench strong output mappings lost: %#v %#v %#v", restored.HTTPRequests, restored.WebSocketSessions, restored.RealtimeKeys)
	}
	if len(restored.ComputeUnits[0].Outputs) != 1 || restored.ComputeUnits[0].Outputs[0].NullPolicy != "skip" || len(restored.HistoryStorage) != 1 || restored.HistoryStorage[0].ComputeUnitID == nil {
		t.Fatalf("compute outputs or history scope lost: %#v %#v", restored.ComputeUnits, restored.HistoryStorage)
	}
	artifact := BuildProjectArtifactV1("project", &restored, time.Now())
	artifactPayload, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(artifactPayload), "ciphertext") || len(artifact.Collectors.SecretRefs) != 1 {
		t.Fatalf("collector artifact leaked ciphertext or lost secret refs: %s", artifactPayload)
	}
	if len(artifact.Queries) != 1 || len(artifact.Queries[0].Outputs) != 1 || len(artifact.Collectors.Points) != 1 || len(artifact.Compute) != 1 || artifact.Compute[0].Outputs[0].NullPolicy != "skip" || len(artifact.HistoryStorage) != 1 {
		t.Fatalf("artifact lost strong mappings or collector config: %#v", artifact)
	}
}

func snapshotString(value string) *string { return &value }
