package ops

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRuntimeFoundationProvisionerDeclaresOnlyNonSensitiveIntent(t *testing.T) {
	input := runtimeFoundationFixture(t)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	provisioner, err := newRuntimeFoundationProvisioner(t.TempDir(), "linux", func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	state, err := provisioner.Ensure(input)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != runtimeFoundationDeclared || state.Intent.Foundation.Postgres.AllocationID == "" || state.Intent.Foundation.NATSJetStream.AllocationID == "" {
		t.Fatalf("unexpected declared foundation state: %+v", state)
	}
	if state.Intent.Secrets[0].Ref == "" || len(state.Intent.Secrets) != 5 {
		t.Fatalf("secret references were not retained: %+v", state.Intent.Secrets)
	}
	path, err := provisioner.StatePath(input.DeploymentID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "postgres://") || strings.Contains(string(raw), "password") || strings.Contains(string(raw), "runtime foundation ready") {
		t.Fatalf("state must never contain plaintext connection data or running claim: %s", raw)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("foundation state must use private file mode: info=%v err=%v", info, err)
	}
}

func TestRuntimeFoundationProvisionerIsConcurrentAndRevisionSafe(t *testing.T) {
	input := runtimeFoundationFixture(t)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	provisioner, err := newRuntimeFoundationProvisioner(t.TempDir(), "linux", func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	const workers = 12
	errs := make(chan error, workers)
	states := make(chan RuntimeFoundationState, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			state, err := provisioner.Ensure(input)
			states <- state
			errs <- err
		}()
	}
	group.Wait()
	close(errs)
	close(states)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	for state := range states {
		if state.Intent.Revision != 1 || state.CreatedAt != now.Format(time.RFC3339) {
			t.Fatalf("retry returned unexpected state: %+v", state)
		}
	}

	changed := input
	changed.Revision = 2
	changed.ActivationID = "77777777-7777-4777-8777-777777777777"
	changed.Foundation.Postgres.Revision++
	updated, err := provisioner.Ensure(changed)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Intent.Revision != 2 || updated.CreatedAt != now.Format(time.RFC3339) {
		t.Fatalf("newer revision was not atomically retained: %+v", updated)
	}
	if _, err := provisioner.Ensure(input); err == nil {
		t.Fatal("revision rollback must be rejected")
	}
	sameRevisionDifferentIntent := changed
	sameRevisionDifferentIntent.Binding.BindingSHA256 = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if _, err := provisioner.Ensure(sameRevisionDifferentIntent); err == nil {
		t.Fatal("same revision with a different intent must be rejected")
	}
}

func TestRuntimeFoundationProvisionerRejectsUnsupportedAndSensitiveInput(t *testing.T) {
	input := runtimeFoundationFixture(t)
	provisioner, err := newRuntimeFoundationProvisioner(t.TempDir(), "windows", time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provisioner.Ensure(input); err == nil {
		t.Fatal("Windows must fail closed until a supported foundation exists")
	}

	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	withPlaintext := strings.Replace(string(raw), `"ref":"secret-runtime-api-postgres"`, `"ref":"secret-runtime-api-postgres","value":"not-allowed"`, 1)
	if _, err := DecodeRuntimeActivationFoundationInput([]byte(withPlaintext)); err == nil {
		t.Fatal("unknown plaintext secret field must be rejected during strict decode")
	}
	withURL := strings.Replace(string(raw), "secret-runtime-api-postgres", "postgres://admin:password@remote.example/db", 1)
	decoded, err := DecodeRuntimeActivationFoundationInput([]byte(withURL))
	if err != nil {
		t.Fatal(err)
	}
	linux, err := newRuntimeFoundationProvisioner(t.TempDir(), "linux", func() time.Time {
		return time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := linux.Ensure(decoded); err == nil {
		t.Fatal("external secret URL must be rejected")
	}
}

func runtimeFoundationFixture(t *testing.T) RuntimeActivationFoundationInput {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "contracts", "runtime", "fixtures", "runtime-activation-v1.valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	input, err := DecodeRuntimeActivationFoundationInput(raw)
	if err != nil {
		t.Fatal(err)
	}
	return input
}
