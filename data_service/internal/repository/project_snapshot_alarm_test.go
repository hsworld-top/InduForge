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
		AlarmPolicyGroups:   []AlarmPolicyGroupRecord{{ID: groupID, ProjectID: "project", Name: "生产线", FullPath: "生产线", CreatedAt: now, UpdatedAt: now}},
		AlarmChannels:       []AlarmNotificationChannelRecord{{ID: channelID, ProjectID: "project", Name: "值班群", ChannelType: "dingtalk", Config: map[string]any{"webhookUrl": "https://example.com"}, SecretStatus: map[string]any{"signingSecret": true}, IsEnabled: true, CreatedAt: now, UpdatedAt: now}},
		AlarmChannelSecrets: []SnapshotAlarmChannelSecretRecord{{ChannelID: channelID, SecretKey: "signingSecret", EncryptedValue: []byte{1, 2, 3}, EncryptionKeyVersion: "v1", CreatedAt: now, UpdatedAt: now}},
		AlarmPolicies:       []AlarmPolicyRecord{{ID: "00000000-0000-0000-0000-000000000030", ProjectID: "project", GroupID: &groupID, Name: "温度报警", Mode: "per_target", NotificationMode: "custom", NotificationChannelIDs: []string{channelID}, IsEnabled: true, Revision: 3, Contract: map[string]any{"schemaVersion": "alarm.policy.v1"}, Bindings: []AlarmPolicyBindingRecord{{ID: "binding", DatapointID: "dp", Role: "target"}}, Conditions: []AlarmPolicyConditionRecord{{ID: "condition", Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 80.0}}}, CreatedAt: now, UpdatedAt: now}},
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	var restored ProjectSnapshot
	if err = json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if len(restored.AlarmPolicies) != 1 || len(restored.AlarmPolicies[0].Bindings) != 1 || len(restored.AlarmPolicies[0].Conditions) != 1 {
		t.Fatalf("alarm policy round trip lost normalized details: %#v", restored.AlarmPolicies)
	}
	if len(restored.AlarmChannelSecrets) != 1 || string(restored.AlarmChannelSecrets[0].EncryptedValue) != string([]byte{1, 2, 3}) {
		t.Fatalf("alarm encrypted secret round trip lost: %#v", restored.AlarmChannelSecrets)
	}
}

func TestProjectArtifactAlarmPolicyContractOnlyContainsSecretReferences(t *testing.T) {
	snapshot := &ProjectSnapshot{
		AlarmPolicies:       []AlarmPolicyRecord{{ID: "policy", Name: "报警", Mode: "per_target", NotificationMode: "inherit", Contract: map[string]any{"schemaVersion": "alarm.policy.v1"}}},
		AlarmChannelSecrets: []SnapshotAlarmChannelSecretRecord{{ChannelID: "channel", SecretKey: "signingSecret", EncryptedValue: []byte("ciphertext-must-not-publish"), EncryptionKeyVersion: "v1"}},
	}
	artifact := BuildProjectArtifactV1("project", snapshot, time.Now())
	payload, err := json.Marshal(artifact)
	if err != nil {
		t.Fatalf("marshal artifact: %v", err)
	}
	if artifact.Alarms.SchemaVersion != "alarm.policy.v1" || len(artifact.Alarms.Policies) != 1 || len(artifact.Alarms.SecretRefs) != 1 {
		t.Fatalf("unexpected alarm artifact: %#v", artifact.Alarms)
	}
	if strings.Contains(string(payload), "ciphertext-must-not-publish") || strings.Contains(string(payload), "encryptedValue") {
		t.Fatalf("artifact leaked encrypted secret payload: %s", payload)
	}
}
