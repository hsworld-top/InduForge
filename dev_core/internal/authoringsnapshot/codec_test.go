package authoringsnapshot

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestSealOpenAndOwnershipBinding(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	snapshot := Snapshot{SchemaVersion: SchemaVersion, TenantID: "tenant-a", ProjectID: "project-a", ProjectRevision: "sha256:revision-3", CapturedAt: time.Date(2026, 9, 4, 1, 2, 3, 0, time.UTC), Workspace: map[string]string{"src/main.ts": "YQ=="}, WorkspaceModes: map[string]uint32{"src/main.ts": 0o644}, Scenes: json.RawMessage(`{"items":[]}`), Data: json.RawMessage(`{"datapoints":[]}`)}
	sealed, err := Seal(snapshot, key, bytes.NewReader(bytes.Repeat([]byte{1}, 32)))
	if err != nil || sealed.ContentSHA256 == "" || sealed.CipherSHA256 == "" || sealed.CiphertextSize != int64(len(sealed.Bytes)) {
		t.Fatalf("Seal() = %#v, %v", sealed, err)
	}
	got, err := Open(sealed.Bytes, key, "tenant-a", "project-a", "sha256:revision-3")
	if err != nil || got.ProjectRevision != "sha256:revision-3" || got.Workspace["src/main.ts"] != "YQ==" {
		t.Fatalf("Open() = %#v, %v", got, err)
	}
	if _, err = VerifyAndOpen(sealed.Bytes, key, "tenant-a", "project-a", "sha256:revision-3", StoredMetadata{ContentSHA256: sealed.ContentSHA256, CipherSHA256: sealed.CipherSHA256, CiphertextSize: sealed.CiphertextSize}); err != nil {
		t.Fatalf("VerifyAndOpen() error = %v", err)
	}
	if _, err = VerifyAndOpen(sealed.Bytes, key, "tenant-a", "project-a", "sha256:revision-3", StoredMetadata{ContentSHA256: sealed.ContentSHA256, CipherSHA256: "bad", CiphertextSize: sealed.CiphertextSize}); err == nil {
		t.Fatal("数据库密文摘要不匹配必须失败")
	}
	if _, err = Open(sealed.Bytes, key, "tenant-b", "project-a", "sha256:revision-3"); err == nil {
		t.Fatal("跨租户密文必须认证失败")
	}
	sealed.Bytes[len(sealed.Bytes)-1] ^= 1
	if _, err = Open(sealed.Bytes, key, "tenant-a", "project-a", "sha256:revision-3"); err == nil {
		t.Fatal("篡改密文必须认证失败")
	}
}

func TestSealRejectsPartialSnapshot(t *testing.T) {
	_, err := Seal(Snapshot{SchemaVersion: SchemaVersion}, bytes.Repeat([]byte{1}, 32), nil)
	if err == nil {
		t.Fatal("不完整快照不得封装")
	}
}
