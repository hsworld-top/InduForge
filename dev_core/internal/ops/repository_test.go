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
	if !(contains(query, "s.node_id=$1") && !contains(query, "d.node_id")) {
		t.Fatal("desired-state query must never fan out by cluster or role")
	}
}

func TestDevelopmentDeploymentDescriptorIsDurableAndReleaseIsPinned(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位测试文件")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), "..", "..", "db", "schema", "core-schema.sql"))
	if err != nil {
		t.Fatal(err)
	}
	schema := string(raw)
	if !contains(schema, "artifact_descriptor jsonb NOT NULL") || !contains(schema, "project_deployments_artifact_source_check") || !contains(schema, "deployment_bindings_artifact_source_check") {
		t.Fatal("开发制品描述必须持久化，并与正式版本来源互斥")
	}
	if !contains(deploymentSelect, "CASE WHEN d.mode='development' THEN '__DEV__'") {
		t.Fatal("开发部署列表必须只呈现服务端固定 __DEV__ 标签")
	}
	if !contains(`artifact_mode=CASE WHEN d.mode='development' THEN 'development' ELSE 'release' END`, "development") {
		t.Fatal("Agent 必须按绑定制品来源解析，不能强制依赖 application_versions")
	}
}

func TestDeploymentInsertReturningKeepsLatestRunPlaceholder(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位仓储实现文件")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), "repository.go"))
	if err != nil {
		t.Fatal(err)
	}
	// 新建事务尚未写入 deployment_runs，因此 Version 后必须保留空的
	// LatestRunID 占位，确保 RETURNING 与 scanDeployment 的 21 列严格对齐。
	if !strings.Contains(string(raw), "COALESCE(application_version_id::text,''),'','',$5") {
		t.Fatal("部署 INSERT RETURNING 缺少 LatestRunID 占位列")
	}
}

func TestMapDeploymentCreateErrorPreservesProjectAndPortIsolation(t *testing.T) {
	if err := mapDeploymentCreateError(errors.New("duplicate key violates unique constraint project_deployments_tenant_project_environment_key")); !errors.Is(err, ErrDeploymentExists) {
		t.Fatalf("same project conflict=%v", err)
	}
	if err := mapDeploymentCreateError(errors.New("duplicate key violates unique constraint deployment_services_base_access_port_key")); !errors.Is(err, ErrNodePortConflict) {
		t.Fatalf("same node port conflict=%v", err)
	}
	if err := mapDeploymentCreateError(&pgconn.PgError{ConstraintName: "deployment_services_base_access_port_key"}); !errors.Is(err, ErrNodePortConflict) {
		t.Fatalf("postgres port conflict=%v", err)
	}
}

func TestDeploymentAccessPortRulesAndIPv6URL(t *testing.T) {
	for _, port := range []int{2379, 2380, 6443, 8472, 10250, 10257, 10259} {
		if !isReservedDeploymentPort(port) {
			t.Fatalf("reserved port %d accepted", port)
		}
	}
	if isReservedDeploymentPort(17800) {
		t.Fatal("ordinary user port was reserved")
	}
	if got := deploymentAccessURL("2001:db8::1", 17800); got != "http://[2001:db8::1]:17800" {
		t.Fatalf("IPv6 URL=%s", got)
	}
	if got := deploymentAccessURL("192.0.2.10", 17800); got != "http://192.0.2.10:17800" {
		t.Fatalf("IPv4 URL=%s", got)
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
