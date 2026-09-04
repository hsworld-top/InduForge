package main

import (
	"context"
	"testing"

	platformdb "github.com/indu-forge/dev_core/internal/platform/db"
)

type recordingBuiltinBindingEnsurer struct {
	projectID string
	tenantID  string
	epoch     string
}

func (r *recordingBuiltinBindingEnsurer) EnsureProjectTenantBindingAtEpoch(_ context.Context, projectID, tenantID, epoch string) error {
	r.projectID, r.tenantID, r.epoch = projectID, tenantID, epoch
	return nil
}

func TestEnsureBuiltinProjectTenantBindingUsesPersistedTenantAndEpoch(t *testing.T) {
	ensurer := &recordingBuiltinBindingEnsurer{}
	if err := ensureBuiltinProjectTenantBinding(context.Background(), ensurer, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", 3); err != nil {
		t.Fatal(err)
	}
	if ensurer.projectID != platformdb.BuiltinDemoProjectID || ensurer.tenantID != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" || ensurer.epoch != "epoch-3" {
		t.Fatalf("unexpected binding request: %#v", ensurer)
	}
}

func TestEnsureBuiltinProjectTenantBindingRejectsInvalidPersistedEpoch(t *testing.T) {
	if err := ensureBuiltinProjectTenantBinding(context.Background(), &recordingBuiltinBindingEnsurer{}, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", 0); err == nil {
		t.Fatal("expected invalid persisted epoch to be rejected")
	}
}
