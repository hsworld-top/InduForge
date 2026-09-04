package authoringsnapshot

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestKeyringRetainsOldKeyAfterRotation(t *testing.T) {
	oldKey, newKey := bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{2}, 32)
	oldRing, err := NewKeyring("old", map[string]string{"old": base64.StdEncoding.EncodeToString(oldKey)})
	if err != nil {
		t.Fatal(err)
	}
	oldID, sealingKey, _ := oldRing.Current()
	snapshot := Snapshot{SchemaVersion: SchemaVersion, TenantID: "tenant", ProjectID: "project", ProjectRevision: "revision", CapturedAt: time.Now().UTC(), Workspace: map[string]string{"a": "YQ=="}, WorkspaceModes: map[string]uint32{"a": 0o644}, Scenes: json.RawMessage(`{}`), Data: json.RawMessage(`{}`)}
	sealed, err := Seal(snapshot, sealingKey, bytes.NewReader(bytes.Repeat([]byte{3}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := NewKeyring("new", map[string]string{"old": base64.StdEncoding.EncodeToString(oldKey), "new": base64.StdEncoding.EncodeToString(newKey)})
	if err != nil {
		t.Fatal(err)
	}
	readingKey, err := rotated.Resolve(oldID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Open(sealed.Bytes, readingKey, "tenant", "project", "revision"); err != nil {
		t.Fatalf("轮换后旧快照不可恢复: %v", err)
	}
	if _, err = rotated.Resolve("removed"); err == nil {
		t.Fatal("未知 keyId 必须失败")
	}
}
