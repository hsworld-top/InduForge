package ops

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestContainsAllPreventsCapabilityEscalation(t *testing.T) {
	if !containsAll([]string{CapabilityProjectEntry, CapabilityDataRuntime}, []string{CapabilityProjectEntry, CapabilityDataRuntime}) {
		t.Fatal("required capabilities should match")
	}
	if containsAll([]string{CapabilityProjectEntry}, []string{CapabilityProjectEntry, CapabilityDataRuntime}) {
		t.Fatal("missing capability must reject claim")
	}
}
func TestAgentDesiredStateIsPinnedToPhysicalNode(t *testing.T) {
	query := pendingServicesSQL
	if !(contains(query, "d.node_id=$1") && contains(query, "s.node_id=$1")) {
		t.Fatal("desired-state query must never fan out by cluster or role")
	}
}

func TestMapDeploymentCreateErrorPreservesNodeProjectIsolation(t *testing.T) {
	if err := mapDeploymentCreateError(errors.New("duplicate key violates unique constraint project_deployments_tenant_project_key")); !errors.Is(err, ErrDeploymentExists) {
		t.Fatalf("same project conflict=%v", err)
	}
	if err := mapDeploymentCreateError(errors.New("duplicate key violates unique constraint project_deployments_tenant_node_key")); !errors.Is(err, ErrNodeProjectConflict) {
		t.Fatalf("different project on node conflict=%v", err)
	}
	if err := mapDeploymentCreateError(&pgconn.PgError{ConstraintName: "project_deployments_tenant_node_key"}); !errors.Is(err, ErrNodeProjectConflict) {
		t.Fatalf("postgres node conflict=%v", err)
	}
}

func TestDeploymentCreationLocksTenantProjectAndPhysicalNode(t *testing.T) {
	for name, query := range map[string]string{
		"project": lockDeploymentProjectSQL,
		"node":    lockDeploymentNodeSQL,
	} {
		if !contains(query, "tenant_id=$1") || !contains(query, "id=$2") || !contains(query, "FOR UPDATE") {
			t.Fatalf("%s lock query must be tenant-scoped and transactional: %s", name, query)
		}
	}
	if !contains(lockDeploymentNodeSQL, "capabilities @> $3::jsonb") || !contains(lockDeploymentNodeSQL, "observed_status='online'") {
		t.Fatalf("node lock must revalidate deployment eligibility: %s", lockDeploymentNodeSQL)
	}
}

func contains(s, part string) bool {
	for i := 0; i+len(part) <= len(s); i++ {
		if s[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
