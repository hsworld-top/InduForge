package binding

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/indu-forge/runtime-engine/internal/resolver"
)

func TestBuildResolverIndexComputeAndAlarm(t *testing.T) {
	compute, err := BuildResolverIndex(validInput(roleCompute))
	if err != nil {
		t.Fatalf("build compute index: %v", err)
	}
	if _, ok := compute.Resources["site-resource://site-a/compute-sandbox"]; !ok {
		t.Fatalf("compute index missing sandbox resource: %#v", compute.Resources)
	}
	if got := compute.Secrets["secret://site-a/deployment-a/compute-sandbox"]; got != "secrets/compute-sandbox.json" {
		t.Fatalf("compute sandbox secret must remain a mounted path, got %q", got)
	}
	if got := string(compute.Resources["site-resource://site-a/postgres"]); got != `{"schema":"runtime"}` {
		t.Fatalf("state-store schema resource mismatch: %s", got)
	}

	alarm, err := BuildResolverIndex(validInput(roleAlarm))
	if err != nil {
		t.Fatalf("build alarm index: %v", err)
	}
	if _, ok := alarm.Resources["site-resource://site-a/compute-sandbox"]; ok {
		t.Fatalf("alarm index must not declare sandbox: %#v", alarm.Resources)
	}
	if len(alarm.Secrets) != 2 {
		t.Fatalf("alarm index secret count = %d, want 2", len(alarm.Secrets))
	}

	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, indexFileName)
	raw, err := marshalIndex(compute)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Open(path, false); err != nil {
		t.Fatalf("formal resolver rejected generated index: %v", err)
	}
}

func TestBuildResolverIndexRejectsMissingReferenceAndContainsNoSecretValue(t *testing.T) {
	input := validInput(roleCompute)
	input.JetStream.CredentialSecretFile = ""
	if _, err := BuildResolverIndex(input); err == nil || !strings.Contains(err.Error(), "credentialSecretFile") {
		t.Fatalf("expected missing mounted secret path error, got %v", err)
	}

	index, err := BuildResolverIndex(validInput(roleCompute))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := marshalIndex(index)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"token", "password", "dsn", "postgres://"} {
		if bytes.Contains(raw, []byte(forbidden)) {
			t.Fatalf("resolver index contains forbidden secret value field %q: %s", forbidden, raw)
		}
	}
}

func TestBuildResolverIndexIsDeterministic(t *testing.T) {
	first, err := marshalIndexMust(validInput(roleCompute))
	if err != nil {
		t.Fatal(err)
	}
	second, err := marshalIndexMust(validInput(roleCompute))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("same input yielded different index bytes:\n%s\n%s", first, second)
	}
}

func TestWriteBundleUsesPrivateFilesAndRestoresOldBundleOnInstallFailure(t *testing.T) {
	input := validInput(roleCompute)
	config, err := BuildEngineConfig(input)
	if err != nil {
		t.Fatal(err)
	}
	index, err := BuildResolverIndex(input)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "bundle")
	if err := WriteBundle(config, index, target); err != nil {
		t.Fatalf("write initial bundle: %v", err)
	}
	oldConfig, err := os.ReadFile(filepath.Join(target, configFileName))
	if err != nil {
		t.Fatal(err)
	}
	oldIndex, err := os.ReadFile(filepath.Join(target, indexFileName))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{configFileName, indexFileName} {
		info, err := os.Stat(filepath.Join(target, name))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("%s permissions = %o, want 0600", name, info.Mode().Perm())
		}
	}

	oldRename := renameBundle
	calls := 0
	renameBundle = func(source, destination string) error {
		calls++
		if calls == 2 {
			return errors.New("injected rename failure")
		}
		return oldRename(source, destination)
	}
	t.Cleanup(func() { renameBundle = oldRename })
	if err := WriteBundle(config, index, target); err == nil || !strings.Contains(err.Error(), "已恢复旧 bundle") {
		t.Fatalf("expected recoverable install failure, got %v", err)
	}
	gotConfig, err := os.ReadFile(filepath.Join(target, configFileName))
	if err != nil {
		t.Fatal(err)
	}
	gotIndex, err := os.ReadFile(filepath.Join(target, indexFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(oldConfig, gotConfig) || !bytes.Equal(oldIndex, gotIndex) {
		t.Fatal("failed bundle write polluted old bundle")
	}
}

func marshalIndexMust(input Input) ([]byte, error) {
	index, err := BuildResolverIndex(input)
	if err != nil {
		return nil, err
	}
	return marshalIndex(index)
}

func marshalIndex(index ResolverIndex) ([]byte, error) {
	return json.Marshal(index)
}
