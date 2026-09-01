package provisioner

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/indu-forge/runtime-engine/internal/binding"
)

type fakeDBAdmin struct {
	exists                bool
	created, role, schema int
	fail                  string
}

func (f *fakeDBAdmin) DatabaseExists(context.Context, string) (bool, error) {
	if f.fail == "check" {
		return false, errors.New("db unavailable")
	}
	return f.exists, nil
}
func (f *fakeDBAdmin) CreateDatabase(context.Context, string, string) error {
	if f.fail == "create" {
		return errors.New("db unavailable")
	}
	f.created++
	f.exists = true
	return nil
}
func (f *fakeDBAdmin) EnsureRole(context.Context, string, string) error {
	if f.fail == "role" {
		return errors.New("db unavailable")
	}
	f.role++
	return nil
}
func (f *fakeDBAdmin) InitializeSchema(context.Context, string, string, string) error {
	if f.fail == "schema" {
		return errors.New("db unavailable")
	}
	f.schema++
	return nil
}
func (f *fakeDBAdmin) Close() {}

func TestProvisionStateCreateAndIdempotent(t *testing.T) {
	in, credentials := stateInput()
	fake := &fakeDBAdmin{}
	if err := ProvisionStateWithAdmin(context.Background(), in, credentials, fake); err != nil {
		t.Fatal(err)
	}
	if fake.created != 1 || fake.role != 1 || fake.schema != 1 {
		t.Fatalf("unexpected calls: %#v", fake)
	}
	if err := ProvisionStateWithAdmin(context.Background(), in, credentials, fake); err != nil {
		t.Fatal(err)
	}
	if fake.created != 1 || fake.role != 2 || fake.schema != 2 {
		t.Fatalf("idempotent calls: %#v", fake)
	}
}
func TestProvisionStateConflictAndRollback(t *testing.T) {
	in, credentials := stateInput()
	fake := &fakeDBAdmin{fail: "schema"}
	err := ProvisionStateWithAdmin(context.Background(), in, credentials, fake)
	if err == nil || !strings.Contains(err.Error(), "runtime_engine") || strings.Contains(err.Error(), credentials.Password) {
		t.Fatalf("err = %v", err)
	}
	if fake.created != 1 || fake.role != 1 {
		t.Fatalf("prior operations not recorded: %#v", fake)
	}
	credentials.Schema = "bad-name"
	if err := ProvisionStateWithAdmin(context.Background(), in, credentials, fake); err == nil {
		t.Fatal("unsafe identifier accepted")
	}
}
func TestDecodePostgresBootstrapStrict(t *testing.T) {
	raw := []byte(`{"schemaVersion":"postgres-bootstrap.v1","maintenanceDsn":"postgres://admin:secret@db/postgres","database":"induforge_runtime","schema":"runtime_project","username":"runtime_project","password":"secret","extra":true}`)
	if _, err := decodePostgresBootstrap(raw); err == nil {
		t.Fatal("unknown field accepted")
	}
	valid := strings.Replace(string(raw), `,"extra":true`, "", 1)
	if _, err := decodePostgresBootstrap([]byte(valid)); err != nil {
		t.Fatal(err)
	}
}
func TestStateErrorsDoNotLeakSecret(t *testing.T) {
	in, credentials := stateInput()
	fake := &fakeDBAdmin{fail: "check"}
	err := ProvisionStateWithAdmin(context.Background(), in, credentials, fake)
	if err == nil || strings.Contains(err.Error(), credentials.Password) || strings.Contains(err.Error(), "postgres://") {
		t.Fatalf("unsafe err: %v", err)
	}
}

func TestProvisionStateRejectsNonContractDatabaseOrSchema(t *testing.T) {
	in, credentials := stateInput()
	fake := &fakeDBAdmin{}
	credentials.Database = "ifrt_0000000000000000"
	if err := ProvisionStateWithAdmin(context.Background(), in, credentials, fake); err == nil {
		t.Fatal("forged database accepted")
	}
	in.Binding.StateStore.Schema = "project_schema"
	credentials.Database = expectedRuntimeDatabase(in.Binding.ProjectID, in.Binding.SiteID)
	if err := ProvisionStateWithAdmin(context.Background(), in, credentials, fake); err == nil {
		t.Fatal("dynamic schema accepted")
	}
}
func stateInput() (Input, PostgresBootstrapCredentials) {
	project, environment := "project-a", "environment-a"
	return Input{Binding: binding.Input{ProjectID: project, SiteID: environment, StateStore: binding.StateStoreInput{Schema: "runtime_engine"}}}, PostgresBootstrapCredentials{SchemaVersion: "postgres-bootstrap.v1", MaintenanceDSN: "postgres://admin:secret@db/postgres", Database: expectedRuntimeDatabase(project, environment), Schema: "runtime_engine", Username: "runtime_project", Password: "runtime-secret"}
}
