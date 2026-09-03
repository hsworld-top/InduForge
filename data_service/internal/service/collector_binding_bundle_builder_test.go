package service

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/indu-forge/data_service/internal/repository"
	collectorsecurity "github.com/indu-forge/data_service/internal/security"
)

const (
	bundleProjectID = "11111111-1111-4111-8111-111111111111"
	bundleModbusID  = "22222222-2222-4222-8222-222222222222"
	bundleOPCUAID   = "33333333-3333-4333-8333-333333333333"
)

func TestCollectorBindingBundleBuildsModbusAndOPCUAWithoutSecretLeak(t *testing.T) {
	cipher := bundleCipher(t)
	secret, err := cipher.Encrypt([]byte("password-value"))
	if err != nil {
		t.Fatal(err)
	}
	snapshot := bundleSnapshot([]repository.SnapshotCollectorConnectionRecord{
		{ID: bundleModbusID, DriverID: "modbus.tcp", Config: map[string]any{"host": "10.0.0.8", "port": 502}, IsEnabled: true},
		{ID: bundleOPCUAID, DriverID: "opcua.standard", Config: map[string]any{"host": "opc.example", "port": 4840, "endpointPath": "/ua", "securityMode": "None", "securityPolicy": "None", "authenticationType": "anonymous"}, IsEnabled: true},
	}, []repository.SnapshotCollectorSecretRecord{{ConnectionID: bundleModbusID, SecretKey: "password", EncryptedValue: secret, EncryptionKeyVersion: "v1"}})
	input := bundleInput(t, snapshot)
	var jsonbFormatted bytes.Buffer
	if err := json.Indent(&jsonbFormatted, input.SourceSnapshot, "", "  "); err != nil {
		t.Fatal(err)
	}
	input.SourceSnapshot = jsonbFormatted.Bytes()
	bundle, err := mustBuilder(t, cipher).Build(snapshot, input)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(bundle.SecretFiles["connection-"+bundleModbusID+".json"]); got != `{"password":"password-value"}` {
		t.Fatalf("secret 文件不符合预期: %s", got)
	}
	if _, exists := bundle.SecretFiles["connection-"+bundleOPCUAID+".json"]; exists {
		t.Fatal("无密钥 OPC UA 连接不应产生 secret 文件")
	}
	var index struct {
		Resources map[string]json.RawMessage `json:"resources"`
		Secrets   map[string]string          `json:"secrets"`
	}
	if err := json.Unmarshal(bundle.Index, &index); err != nil {
		t.Fatal(err)
	}
	if string(index.Resources["site-resource://collector/"+bundleModbusID]) != `{"host":"10.0.0.8","port":502}` {
		t.Fatalf("Modbus resource 错误: %s", index.Resources["site-resource://collector/"+bundleModbusID])
	}
	if string(index.Resources["site-resource://collector/"+bundleOPCUAID]) != `{"url":"opc.tcp://opc.example:4840/ua"}` {
		t.Fatalf("OPC UA resource 错误: %s", index.Resources["site-resource://collector/"+bundleOPCUAID])
	}
	if index.Secrets["secret://collector/"+bundleModbusID] != "secrets/connection-"+bundleModbusID+".json" {
		t.Fatal("连接 secret index 错误")
	}
	for _, raw := range [][]byte{bundle.Binding, bundle.Index} {
		if bytes.Contains(raw, []byte("password-value")) {
			t.Fatalf("binding/index 泄露 secret: %s", raw)
		}
	}
	if bytes.Contains(bundle.Index, []byte("token")) {
		t.Fatal("NATS token 不应出现在 index")
	}
}

func TestCollectorBindingBundleRejectsDecryptFailureAndSnapshotMismatch(t *testing.T) {
	cipher := bundleCipher(t)
	snapshot := bundleSnapshot([]repository.SnapshotCollectorConnectionRecord{{ID: bundleModbusID, DriverID: "modbus.tcp", Config: map[string]any{"host": "10.0.0.8", "port": 502}, IsEnabled: true}}, []repository.SnapshotCollectorSecretRecord{{ConnectionID: bundleModbusID, SecretKey: "password", EncryptedValue: []byte("bad"), EncryptionKeyVersion: "v1"}})
	if _, err := mustBuilder(t, cipher).Build(snapshot, bundleInput(t, snapshot)); err == nil || strings.Contains(err.Error(), "bad") {
		t.Fatalf("应脱敏拒绝解密失败: %v", err)
	}
	snapshot.Snapshot.CollectorSecrets = nil
	in := bundleInput(t, snapshot)
	var source map[string]any
	if err := json.Unmarshal(in.SourceSnapshot, &source); err != nil {
		t.Fatal(err)
	}
	artifact := source["artifact"].(map[string]any)
	artifact["connections"] = []any{}
	raw, _ := json.Marshal(artifact)
	source["artifact"] = json.RawMessage(raw)
	source["size"] = len(raw)
	source["sha256"] = sha256Text(raw)
	in.SourceSnapshot, _ = json.Marshal(source)
	if _, err := mustBuilder(t, cipher).Build(snapshot, in); err == nil || !strings.Contains(err.Error(), "不一致") {
		t.Fatalf("应拒绝快照不一致: %v", err)
	}
}

func TestCollectorBindingBundleIsDeterministicAndUsesUUIDFileName(t *testing.T) {
	cipher := bundleCipher(t)
	secret, err := cipher.Encrypt([]byte(`{"username":"operator"}`))
	if err != nil {
		t.Fatal(err)
	}
	snapshot := bundleSnapshot([]repository.SnapshotCollectorConnectionRecord{{ID: bundleModbusID, DriverID: "modbus.tcp", Config: map[string]any{"host": "10.0.0.8", "port": 502}, IsEnabled: true}}, []repository.SnapshotCollectorSecretRecord{{ConnectionID: bundleModbusID, SecretKey: "login", EncryptedValue: secret, EncryptionKeyVersion: "v1"}})
	builder := mustBuilder(t, cipher)
	input := bundleInput(t, snapshot)
	first, err := builder.Build(snapshot, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := builder.Build(snapshot, input)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Binding, second.Binding) || !bytes.Equal(first.Index, second.Index) || first.BindingSHA256 != second.BindingSHA256 || first.BindingSize != len(first.Binding) {
		t.Fatal("相同输入必须产生确定 bundle/hash")
	}
	if _, ok := first.SecretFiles["connection-"+bundleModbusID+".json"]; !ok {
		t.Fatal("secret 文件名必须由 UUID 确定")
	}
}

func bundleCipher(t *testing.T) *collectorsecurity.CollectorSecretCipher {
	t.Helper()
	c, err := collectorsecurity.NewCollectorSecretCipher(bytes.Repeat([]byte{7}, 32), "v1")
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func mustBuilder(t *testing.T, c *collectorsecurity.CollectorSecretCipher) *CollectorBindingBundleBuilder {
	t.Helper()
	b, err := NewCollectorBindingBundleBuilder(c)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func bundleSnapshot(connections []repository.SnapshotCollectorConnectionRecord, secrets []repository.SnapshotCollectorSecretRecord) TenantProjectSnapshot {
	return TenantProjectSnapshot{TenantID: "tenant-a", ProjectID: bundleProjectID, Snapshot: &repository.ProjectSnapshot{CollectorConnections: connections, CollectorSecrets: secrets}}
}
func bundleInput(t *testing.T, snapshot TenantProjectSnapshot) DeploymentBindingInput {
	t.Helper()
	list := make([]any, 0, len(snapshot.Snapshot.CollectorConnections))
	for _, c := range snapshot.Snapshot.CollectorConnections {
		if c.IsEnabled {
			list = append(list, map[string]any{"connectionId": c.ID, "protocolFamily": "modbus", "driverId": c.DriverID, "driverVersion": "1.0.0", "schemaVersion": 1, "enabled": true, "requiredOperations": []any{"point.read"}, "defaultAcquisition": map[string]any{"intervalMs": 1000, "deadband": 0, "changeOnly": false}})
		}
	}
	artifact := map[string]any{"schemaVersion": "collector-runtime-artifact.v1", "artifactId": "artifact-a", "artifactRevision": 1, "projectId": snapshot.ProjectID, "collectorVersion": "1.0.0", "connections": list, "pointMappings": []any{}, "wal": map[string]any{}}
	raw, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	source, err := json.Marshal(map[string]any{"schemaVersion": "collector-runtime-artifact.v1", "projectId": snapshot.ProjectID, "artifactRevision": 1, "sha256": sha256Text(raw), "size": len(raw), "artifact": json.RawMessage(raw)})
	if err != nil {
		t.Fatal(err)
	}
	return DeploymentBindingInput{DeploymentID: "deployment-a", EnvironmentID: "prod", NodeID: "node-a", ReleaseID: "release-a", AccountID: "account-a", BindingRevision: 1, OwnershipEpoch: 1, NATSServerResourceRef: "site-resource://site/nats", NATSCredentialSecretRef: "secret://site/nats", NATSEndpoint: "nats://nats.example:4222", SourceSnapshot: source}
}
