package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadNATSTokenRequiresStrictTokenCredential(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nats.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":"nats-credential.v1","authType":"token","token":"project-token"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	token, err := loadNATSToken(path)
	if err != nil || token != "project-token" {
		t.Fatalf("token=%q err=%v", token, err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":"nats-credential.v1","authType":"token","token":"project-token","extra":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadNATSToken(path); err == nil {
		t.Fatal("unknown field must be rejected")
	}
}
