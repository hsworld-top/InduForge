package ops

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/jackc/pgx/v5/pgconn"
)

// embedded Repository 让测试只实现当前场景会调用的方法，其他调用会立即暴露为 nil。
type deploymentRepository struct {
	Repository
	validated       bool
	operated        bool
	validationError error
}

func TestOpsPublicValuesUseLowerCamelJSON(t *testing.T) {
	raw, err := json.Marshal(map[string]any{"command": AgentCommand{RunID: "run", WorkloadID: "workload", DesiredStatus: "running"}, "package": NodePackage{ID: RoleRuntimeLinux, FileName: "agent.tgz", Available: true}, "workload": Workload{ID: "workload", DesiredGeneration: 2}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, field := range []string{`"runId"`, `"workloadId"`, `"desiredStatus"`, `"fileName"`, `"desiredGeneration"`} {
		if !strings.Contains(text, field) {
			t.Fatalf("missing lower camel field %s: %s", field, text)
		}
	}
	for _, field := range []string{`"RunID"`, `"FileName"`, `"DesiredGeneration"`} {
		if strings.Contains(text, field) {
			t.Fatalf("leaked Go field %s: %s", field, text)
		}
	}
}

func TestManagementPayloadsExposeApprovalAndListPresentationFields(t *testing.T) {
	node := HostNode{
		ID: "node-1", Hostname: "edge-a", MachineFingerprint: "fingerprint", IPAddress: "10.0.0.8", ObservedStatus: "online",
		ResourceSummary: map[string]any{
			"cpu":    map[string]any{"usedPercent": 12.5},
			"memory": map[string]any{"usedPercent": 34.5},
			"disk":   map[string]any{"usedPercent": 56.5},
		},
	}
	enrollment := enrollmentPayload(Enrollment{ID: "enrollment-1", ClaimedByNodeID: node.ID, ReportedHostName: node.Hostname, MachineFingerprint: node.MachineFingerprint, IPAddress: node.IPAddress, Node: &node})
	if enrollment["reportedHostName"] != "edge-a" || enrollment["machineFingerprint"] != "fingerprint" || enrollment["node"] == nil {
		t.Fatalf("审批 payload 缺少已 claim 主机事实: %#v", enrollment)
	}
	cluster := clusterPayload(RuntimeCluster{ID: "cluster-1", NodeCount: 2, OnlineNodeCount: 1, Health: "healthy"})
	if cluster["nodeCount"] != 2 || cluster["onlineNodeCount"] != 1 || cluster["health"] != "healthy" {
		t.Fatalf("集群展示字段缺失: %#v", cluster)
	}
	deployment := deploymentPayload(ProjectDeployment{ID: "deployment-1", ProjectName: "演示工程", RuntimeClusterName: "默认运行集群", LatestRunID: "run-latest", Progress: 60, Health: "pending"})
	if deployment["projectName"] != "演示工程" || deployment["runtimeClusterName"] != "默认运行集群" || deployment["latestRunId"] != "run-latest" || deployment["progress"] != 60 {
		t.Fatalf("部署展示字段缺失: %#v", deployment)
	}
	nodeResult := nodePayload(node)
	metrics := nodeResult["metrics"].(map[string]any)
	if nodeResult["health"] != "healthy" || metrics["cpuPercent"] != 12.5 || metrics["memoryPercent"] != 34.5 || metrics["diskPercent"] != 56.5 {
		t.Fatalf("节点健康度或资源指标映射错误: %#v", nodeResult)
	}
}

func TestHealthPresentationUsesOnlySupportedValues(t *testing.T) {
	for input, want := range map[string]string{"pending": "unknown", "offline": "unavailable", "failed": "unavailable", "running": "unknown", "healthy": "healthy", "degraded": "degraded", "unavailable": "unavailable", "unknown": "unknown"} {
		if got := normalizeHealth(input); got != want {
			t.Fatalf("%s: got %s want %s", input, got, want)
		}
	}
}

func (r *deploymentRepository) ValidateDeploymentTargets(context.Context, string, CreateDeploymentInput) error {
	r.validated = true
	return r.validationError
}
func (r *deploymentRepository) CreateDeployment(_ context.Context, tenant, user string, in CreateDeploymentInput) (ProjectDeployment, DeploymentRun, error) {
	return ProjectDeployment{ID: "deployment-1", TenantID: tenant, ProjectID: in.ProjectID, RuntimeClusterID: in.RuntimeClusterID, DeploymentMode: in.DeploymentMode}, DeploymentRun{ID: "run-1"}, nil
}
func (r *deploymentRepository) OperateWorkload(context.Context, string, string, string, string, string) (ProjectDeployment, DeploymentRun, error) {
	r.operated = true
	return ProjectDeployment{}, DeploymentRun{}, nil
}

func TestCreateDeploymentValidatesTargetsAfterSemanticValidation(t *testing.T) {
	repo := &deploymentRepository{}
	service := NewService(repo, nil)
	actor := auth.User{ID: "user-1", TenantID: "tenant-1", Role: "OPS_ADMIN"}
	_, _, err := service.CreateDeployment(context.Background(), actor, CreateDeploymentInput{ProjectID: testProjectID, RuntimeClusterID: testClusterID, DeploymentMode: "production", Workloads: []WorkloadInput{{Role: "compute"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.validated {
		t.Fatal("创建部署前未校验租户内工程、集群和采集节点")
	}
	_, _, err = service.CreateDeployment(context.Background(), actor, CreateDeploymentInput{ProjectID: testProjectID, RuntimeClusterID: testClusterID, DeploymentMode: "development", Workloads: []WorkloadInput{{Role: "collector"}}})
	if err == nil {
		t.Fatal("collector 未指定 HostNode 时应拒绝")
	}
}

const (
	testProjectID = "11111111-1111-4111-8111-111111111111"
	testClusterID = "22222222-2222-4222-8222-222222222222"
	testNodeID    = "33333333-3333-4333-8333-333333333333"
)

func TestCreateDeploymentRejectsMalformedExternalUUIDBeforeRepository(t *testing.T) {
	actor := auth.User{ID: "user-1", TenantID: "tenant-1", Role: "OPS_ADMIN"}
	for _, input := range []CreateDeploymentInput{
		{ProjectID: "not-a-uuid", RuntimeClusterID: testClusterID, DeploymentMode: "development", Workloads: []WorkloadInput{{Role: WorkloadRoleCompute}}},
		{ProjectID: testProjectID, RuntimeClusterID: "not-a-uuid", DeploymentMode: "development", Workloads: []WorkloadInput{{Role: WorkloadRoleCompute}}},
		{ProjectID: testProjectID, RuntimeClusterID: testClusterID, DeploymentMode: "development", Workloads: []WorkloadInput{{Role: WorkloadRoleCollector, HostNodeID: "not-a-uuid"}}},
	} {
		repo := &deploymentRepository{}
		if _, _, err := NewService(repo, nil).CreateDeployment(context.Background(), actor, input); err == nil || !strings.Contains(err.Error(), "格式无效") {
			t.Fatalf("input %#v should fail UUID validation, got %v", input, err)
		}
		if repo.validated {
			t.Fatalf("malformed UUID reached repository: %#v", input)
		}
	}
}

func TestCreateEnrollmentRejectsMalformedRuntimeClusterIDBeforeRepository(t *testing.T) {
	actor := auth.User{ID: "user-1", TenantID: "tenant-1", Role: "OPS_ADMIN"}
	repo := &clusterRepository{}
	_, _, err := NewService(repo, nil).CreateEnrollment(context.Background(), actor, CreateEnrollmentInput{Role: RoleRuntimeLinux, RuntimeClusterID: "not-a-uuid"})
	if err == nil || !strings.Contains(err.Error(), "格式无效") {
		t.Fatalf("want malformed runtime cluster ID error, got %v", err)
	}
}

func TestCreateDeploymentSurfacesExistingTargetConflict(t *testing.T) {
	repo := &deploymentRepository{validationError: ErrDeploymentExists}
	actor := auth.User{ID: "user-1", TenantID: "tenant-1", Role: "OPS_ADMIN"}
	_, _, err := NewService(repo, nil).CreateDeployment(context.Background(), actor, CreateDeploymentInput{ProjectID: testProjectID, RuntimeClusterID: testClusterID, DeploymentMode: "development", Workloads: []WorkloadInput{{Role: WorkloadRoleCompute}}})
	if !errors.Is(err, ErrDeploymentExists) {
		t.Fatalf("want deployment conflict, got %v", err)
	}
}

func TestMapDeploymentCreateErrorMapsUniqueConstraintToConflict(t *testing.T) {
	err := mapDeploymentCreateError(&pgconn.PgError{Code: "23505", ConstraintName: "project_deployments_tenant_project_cluster_key"})
	if !errors.Is(err, ErrDeploymentExists) {
		t.Fatalf("concurrent insert conflict should map to deployment exists, got %v", err)
	}
}

type clusterRepository struct {
	Repository
	created bool
}

func (r *clusterRepository) CreateCluster(_ context.Context, tenant, user string, input CreateClusterInput) (RuntimeCluster, error) {
	r.created = true
	return RuntimeCluster{ID: "cluster-1", TenantID: tenant, Name: input.Name, Code: input.Code}, nil
}

func TestCreateClusterValidatesSchemaBoundariesBeforeRepository(t *testing.T) {
	actor := auth.User{ID: "user-1", TenantID: "tenant-1", Role: "OPS_ADMIN"}
	for _, input := range []CreateClusterInput{
		{Name: "集群", Code: "a"},
		{Name: "集群", Code: "A1"},
		{Name: "集群", Code: "a_1"},
		{Name: strings.Repeat("名", 121), Code: "cluster-1"},
	} {
		repo := &clusterRepository{}
		if _, err := NewService(repo, nil).CreateCluster(context.Background(), actor, input); err == nil {
			t.Fatalf("input %#v should fail before repository", input)
		}
		if repo.created {
			t.Fatalf("invalid input reached repository: %#v", input)
		}
	}
	repo := &clusterRepository{}
	if _, err := NewService(repo, nil).CreateCluster(context.Background(), actor, CreateClusterInput{Name: "边缘运行集群", Code: "edge-cluster-1"}); err != nil {
		t.Fatal(err)
	}
	if !repo.created {
		t.Fatal("valid cluster did not reach repository")
	}
}

func TestOperateWorkloadRejectsUnknownRoleBeforeRepository(t *testing.T) {
	repo := &deploymentRepository{}
	actor := auth.User{ID: "user-1", TenantID: "tenant-1", Role: "OPS_ADMIN"}
	_, _, err := NewService(repo, nil).OperateWorkload(context.Background(), actor, "deployment-1", "unknown", "start")
	if err == nil || !strings.Contains(err.Error(), "角色不支持") {
		t.Fatalf("want invalid workload role error, got %v", err)
	}
	if repo.operated {
		t.Fatal("invalid role reached repository")
	}
}

func TestRuntimeApprovalAllowedHonorsTopology(t *testing.T) {
	if err := runtimeApprovalAllowed("single_node", 1); err == nil {
		t.Fatal("single node cluster must reject its second active runtime node")
	}
	for _, tc := range []struct {
		topology string
		count    int
	}{{"single_node", 0}, {"high_availability", 1}, {"high_availability", 3}} {
		if err := runtimeApprovalAllowed(tc.topology, tc.count); err != nil {
			t.Fatalf("%s/%d should allow approval: %v", tc.topology, tc.count, err)
		}
	}
}

func TestDeploymentTopologyAllowedRejectsHighAvailability(t *testing.T) {
	if err := deploymentTopologyAllowed("single_node"); err != nil {
		t.Fatalf("single node deployment should be allowed: %v", err)
	}
	if err := deploymentTopologyAllowed("high_availability"); err == nil || !strings.Contains(err.Error(), "尚未开放") {
		t.Fatalf("high availability deployment should be rejected explicitly, got %v", err)
	}
}

func TestValidateClaimPlatformMatchesPackageRole(t *testing.T) {
	for _, tc := range []struct {
		role, osName, arch string
		valid              bool
	}{
		{RoleRuntimeLinux, "linux", "amd64", true},
		{RoleCollectorLinux, "Linux", "arm64", true},
		{RoleCollectorWindows, "windows", "amd64", true},
		{RoleRuntimeLinux, "windows", "amd64", false},
		{RoleCollectorWindows, "windows", "arm64", false},
		{RoleCollectorLinux, "linux", "386", false},
	} {
		err := validateClaimPlatform(tc.role, tc.osName, tc.arch)
		if (err == nil) != tc.valid {
			t.Fatalf("%s/%s/%s valid=%t got err=%v", tc.role, tc.osName, tc.arch, tc.valid, err)
		}
	}
}

func TestFilePackageStoreListsMissingAndOpensOnlyFixedNames(t *testing.T) {
	dir := t.TempDir()
	store := NewFilePackageStore(dir)
	items := store.List()
	if len(items) != 3 {
		t.Fatalf("want 3 packages, got %d", len(items))
	}
	for _, item := range items {
		if item.Available {
			t.Fatalf("%s should be unavailable", item.ID)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "induforge-node-runtime-linux.tar.gz"), []byte("demo"), 0600); err != nil {
		t.Fatal(err)
	}
	p, path, err := store.Open(RoleRuntimeLinux)
	if err != nil || !p.Available || filepath.Base(path) != p.FileName {
		t.Fatalf("open fixed package failed: %v", err)
	}
	if _, _, err := store.Open("../../etc/passwd"); err == nil {
		t.Fatal("path traversal id must not open a file")
	}
}
