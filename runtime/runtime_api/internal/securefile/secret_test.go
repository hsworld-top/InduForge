package securefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadSecretAcceptsAtomicWriterAndRejectsEscape(t *testing.T) {
	root := t.TempDir()
	version := filepath.Join(root, "..2026_09_02")
	if err := os.Mkdir(version, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(version, "token.json"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Base(version), filepath.Join(root, "..data")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..data", "token.json"), filepath.Join(root, "token.json")); err != nil {
		t.Fatal(err)
	}
	payload, err := ReadSecret(filepath.Join(root, "token.json"), 1<<20)
	if err != nil || string(payload) != "secret" {
		t.Fatalf("payload=%q err=%v", payload, err)
	}
	if err := os.Chmod(filepath.Join(version, "token.json"), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSecret(filepath.Join(root, "token.json"), 1<<20); err != nil {
		t.Fatalf("matching read-only group rejected: %v", err)
	}
	if err := os.Chmod(filepath.Join(version, "token.json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSecret(filepath.Join(root, "token.json"), 1<<20); err == nil {
		t.Fatal("other-readable secret accepted")
	}

	escapeRoot := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(escapeRoot, "..data")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..data", "token.json"), filepath.Join(escapeRoot, "token.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSecret(filepath.Join(escapeRoot, "token.json"), 1<<20); err == nil {
		t.Fatal("escaped AtomicWriter link accepted")
	}
}
