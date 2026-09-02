package resolver

import (
	"context"
	"encoding/json"
	"github.com/indu-forge/collector-engine/internal/loader"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveNATSReadsContainedSecretOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nats.json"), []byte(`{"schemaVersion":"nats-credential.v1","authType":"token","token":"hidden","username":"","password":""}`), 0600); err != nil {
		t.Fatal(err)
	}
	resource, _ := json.Marshal(map[string]string{"url": "nats://127.0.0.1:4222", "accountId": "account-a"})
	r, err := New(&loader.Loaded{Index: loader.Index{SchemaVersion: "collector-runtime-index.v1", Resources: map[string]json.RawMessage{"site-resource://site/nats": resource}, Secrets: map[string]string{"secret://site/nats": "nats.json"}}, IndexDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.ResolveNATS(context.Background(), "site-resource://site/nats", "secret://site/nats", "account-a")
	if err != nil || got.URL == "" || len(got.Options) != 1 {
		t.Fatalf("resolve failed: %v", err)
	}
	r.index.Secrets["secret://site/nats"] = "../nats.json"
	if _, err := r.ResolveNATS(context.Background(), "site-resource://site/nats", "secret://site/nats", "account-a"); err == nil {
		t.Fatal("must reject path escape")
	}
}

func TestResolveNATSAcceptsProjectedSecretLinkOnlyWithinIndexDirectory(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "..2026_09_02")
	if err := os.Mkdir(dataDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "nats.json"), []byte(`{"schemaVersion":"nats-credential.v1","authType":"token","token":"hidden","username":"","password":""}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("..2026_09_02", filepath.Join(dir, "..data")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("..data/nats.json", filepath.Join(dir, "nats.json")); err != nil {
		t.Fatal(err)
	}
	resource, _ := json.Marshal(map[string]string{"url": "nats://127.0.0.1:4222", "accountId": "account-a"})
	r, err := New(&loader.Loaded{Index: loader.Index{SchemaVersion: "collector-runtime-index.v1", Resources: map[string]json.RawMessage{"site-resource://site/nats": resource}, Secrets: map[string]string{"secret://site/nats": "nats.json"}}, IndexDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ResolveNATS(context.Background(), "site-resource://site/nats", "secret://site/nats", "account-a"); err != nil {
		t.Fatalf("projected Secret link must be accepted: %v", err)
	}
	outside := filepath.Join(t.TempDir(), "nats.json")
	if err := os.WriteFile(outside, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "nats.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "nats.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := r.ResolveNATS(context.Background(), "site-resource://site/nats", "secret://site/nats", "account-a"); err == nil {
		t.Fatal("escaped Secret link must be rejected")
	}
}
