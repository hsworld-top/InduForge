package deployment

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEd25519SigningKeyAcceptsControlledPKCS8PEM(t *testing.T) {
	_, key, _ := ed25519.GenerateKey(rand.Reader)
	raw, _ := x509.MarshalPKCS8PrivateKey(key)
	path := filepath.Join(t.TempDir(), "release.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: raw}), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadEd25519SigningKey(path, "release-key-v1")
	if err != nil || !loaded.Key.Equal(key) || loaded.KeyID != "release-key-v1" {
		t.Fatalf("受控 Ed25519 密钥加载失败: %v", err)
	}
}
func TestLoadEd25519SigningKeyRejectsUnsafeFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.pem")
	if err := os.WriteFile(path, []byte("bad"), 0o666); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadEd25519SigningKey(path, "key"); err == nil {
		t.Fatal("宽松权限或非法 PEM 不得接受")
	}
}
