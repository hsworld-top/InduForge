package ops

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestFoundationMigrationColumnsBelongToServicesTable(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位测试文件")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), "..", "..", "db", "schema", "core-schema.sql"))
	if err != nil {
		t.Fatal(err)
	}
	schema := string(raw)
	nodesStart := strings.Index(schema, "CREATE TABLE runtime_environment_nodes")
	servicesStart := strings.Index(schema, "CREATE TABLE runtime_environment_services")
	eventsStart := strings.Index(schema, "CREATE TABLE runtime_environment_events")
	if nodesStart < 0 || servicesStart <= nodesStart || eventsStart <= servicesStart {
		t.Fatal("运行环境表基线不完整")
	}
	if strings.Contains(schema[nodesStart:servicesStart], "previous_node_id") {
		t.Fatal("previous_node_id 不应属于环境节点关系表")
	}
	if !strings.Contains(schema[servicesStart:eventsStart], "previous_node_id uuid REFERENCES host_nodes") {
		t.Fatal("基础服务表缺少迁移源节点字段")
	}
}

func TestAcceptClusterStateIdentity(t *testing.T) {
	tests := []struct {
		name  string
		state ClusterState
		want  bool
	}{
		{name: "current cluster state", state: ClusterState{NodeID: "node-new", ClusterID: "cluster-new", ObservedState: "ready"}, want: true},
		{name: "old uninstall tombstone", state: ClusterState{NodeID: "node-old", ClusterID: "cluster-old", ObservedState: "not-installed"}, want: true},
		{name: "old running cluster", state: ClusterState{NodeID: "node-old", ClusterID: "cluster-old", ObservedState: "ready"}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := acceptClusterStateIdentity(test.state, "node-new", "cluster-new"); got != test.want {
				t.Fatalf("acceptClusterStateIdentity()=%v want=%v", got, test.want)
			}
		})
	}
}

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

func TestMapDeploymentCreateErrorPreservesProjectAndPortIsolation(t *testing.T) {
	if err := mapDeploymentCreateError(errors.New("duplicate key violates unique constraint project_deployments_tenant_project_key")); !errors.Is(err, ErrDeploymentExists) {
		t.Fatalf("same project conflict=%v", err)
	}
	if err := mapDeploymentCreateError(errors.New("duplicate key violates unique constraint project_deployments_tenant_node_access_port_key")); !errors.Is(err, ErrNodePortConflict) {
		t.Fatalf("same node port conflict=%v", err)
	}
	if err := mapDeploymentCreateError(&pgconn.PgError{ConstraintName: "project_deployments_tenant_node_access_port_key"}); !errors.Is(err, ErrNodePortConflict) {
		t.Fatalf("postgres port conflict=%v", err)
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

func TestTimeSyncEventClassDoesNotCreateCalibrationNoise(t *testing.T) {
	if timeSyncEventClass("synchronized") != timeSyncEventClass("adjusting") {
		t.Fatal("同步完成与平滑校准之间不应反复产生运维事件")
	}
	if timeSyncEventClass("failed") == timeSyncEventClass("synchronized") {
		t.Fatal("时间同步异常与正常状态必须产生运维事件")
	}
}

func TestFoundationMigrationAdvancesWholeEnvironmentGeneration(t *testing.T) {
	if !contains(advanceFoundationGenerationSQL, "SET desired_generation=$1") || !contains(advanceFoundationGenerationSQL, "WHERE environment_id=$2") {
		t.Fatalf("基础服务迁移必须统一推进环境代次: %s", advanceFoundationGenerationSQL)
	}
	if contains(advanceFoundationGenerationSQL, "service_type") {
		t.Fatalf("环境代次不能只推进被移动的单项服务: %s", advanceFoundationGenerationSQL)
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
