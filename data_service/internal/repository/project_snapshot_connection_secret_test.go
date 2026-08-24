package repository

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestConnectionSecretSnapshotAndArtifactNeverExposePlaintext(t *testing.T) {
	snapshot := &ProjectSnapshot{
		Connections: []ConnectionRecord{{
			ID: "connection-1", Name: "TDengine", Type: "tdengine", Status: "active",
			Config: map[string]any{
				"host": "127.0.0.1", "password": "must-not-leak", "dsn": "root:must-not-leak@example",
				"options": map[string]any{"token": "must-not-leak", "timezone": "Asia/Shanghai"},
			},
		}},
		ConnectionSecrets: []SnapshotConnectionSecretRecord{{
			ConnectionID: "connection-1", SecretKey: "password", EncryptedValue: []byte{1, 2, 3, 4},
			EncryptionKeyVersion: "v1", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}},
	}
	if !connectionConfigContainsPlaintextSecret(snapshot.Connections[0].Config, "") {
		t.Fatal("plaintext connection config must be rejected on import")
	}
	snapshot.Connections[0].Config = scrubArtifactConnectionSecrets(snapshot.Connections[0].Config)
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(snapshotJSON, []byte("must-not-leak")) {
		t.Fatalf("snapshot contains plaintext: %s", snapshotJSON)
	}
	var restored ProjectSnapshot
	if err := json.Unmarshal(snapshotJSON, &restored); err != nil {
		t.Fatal(err)
	}
	if len(restored.ConnectionSecrets) != 1 || !bytes.Equal(restored.ConnectionSecrets[0].EncryptedValue, []byte{1, 2, 3, 4}) {
		t.Fatalf("encrypted secret roundtrip failed: %#v", restored.ConnectionSecrets)
	}

	artifact := BuildProjectArtifactV1("project-1", snapshot, time.Now())
	artifactJSON, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(artifactJSON, []byte("must-not-leak")) || bytes.Contains(artifactJSON, []byte("encryptedValue")) {
		t.Fatalf("artifact leaked secret material: %s", artifactJSON)
	}
	if len(artifact.SecretRefs) != 1 || artifact.SecretRefs[0].SecretKey != "password" || artifact.SecretRefs[0].KeyVersion != "v1" {
		t.Fatalf("artifact secret refs mismatch: %#v", artifact.SecretRefs)
	}
	if len(artifact.Protocols.TDengine) != 1 || artifact.Protocols.TDengine[0].Config["password"] != nil || artifact.Protocols.TDengine[0].Config["dsn"] != nil {
		t.Fatalf("artifact protocol config not scrubbed: %#v", artifact.Protocols.TDengine)
	}
}
