package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/objectstore"
)

const (
	testProjectID     = "11111111-1111-4111-8111-111111111111"
	testNodeID        = "33333333-3333-4333-8333-333333333333"
	testVersionID     = "44444444-4444-4444-8444-444444444444"
	testEnvironmentID = "66666666-6666-4666-8666-666666666666"
)

type environmentRepository struct {
	Repository
	createdInput CreateRuntimeEnvironmentInput
	updatedInput UpdateRuntimeEnvironmentInput
	deletedInput DeleteRuntimeEnvironmentInput
	addedNodeIDs []string
}

type clusterPlanRepository struct{ Repository }

type nodeMetricsRepository struct {
	Repository
	tenant string
	ids    []string
}

func (r *nodeMetricsRepository) ListNodeMetrics(_ context.Context, tenant string, ids []string) ([]Node, error) {
	r.tenant = tenant
	r.ids = append([]string(nil), ids...)
	return []Node{{ID: ids[0], ObservedStatus: "online"}}, nil
}

func TestListNodeMetricsValidatesScopeAndDeduplicatesIDs(t *testing.T) {
	repository := &nodeMetricsRepository{}
	service := NewService(repository, nil)
	viewer := auth.User{TenantID: "tenant", ID: "user", Role: "VIEWER"}
	ops := auth.User{TenantID: "tenant", ID: "user", Role: "OPS_ADMIN"}
	if _, err := service.ListNodeMetrics(context.Background(), viewer, []string{testNodeID}); !errors.Is(err, auth.ErrPermissionDenied) {
		t.Fatalf("viewer must not read node metrics: %v", err)
	}
	if _, err := service.ListNodeMetrics(context.Background(), ops, []string{"invalid"}); err == nil {
		t.Fatal("invalid node ID must be rejected")
	}
	items, err := service.ListNodeMetrics(context.Background(), ops, []string{testNodeID, testNodeID})
	if err != nil || len(items) != 1 || repository.tenant != "tenant" || len(repository.ids) != 1 || repository.ids[0] != testNodeID {
		t.Fatalf("metrics scope mismatch: items=%#v tenant=%s ids=%v err=%v", items, repository.tenant, repository.ids, err)
	}
}

func (clusterPlanRepository) AgentClusterPlan(context.Context, string, string) (*ClusterPlan, error) {
	return &ClusterPlan{ClusterID: "cluster", NodeID: testNodeID}, nil
}

func TestAgentClusterPlanUsesInstalledK3sJoinToken(t *testing.T) {
	service := NewService(clusterPlanRepository{}, nil)
	if _, err := service.AgentClusterPlan(context.Background(), testNodeID, "agent-token"); err == nil || !strings.Contains(err.Error(), "加入令牌未配置") {
		t.Fatalf("缺少中心 K3s token 时必须失败关闭: %v", err)
	}
	service.SetClusterJoinToken("K10test::server:join-token")
	plan, err := service.AgentClusterPlan(context.Background(), testNodeID, "agent-token")
	if err != nil || plan.Token != "K10test::server:join-token" {
		t.Fatalf("外部节点未取得实际中心 K3s token: plan=%#v err=%v", plan, err)
	}
}

func (r *environmentRepository) UpdateRuntimeEnvironment(_ context.Context, _, _, _ string, input UpdateRuntimeEnvironmentInput) (RuntimeEnvironment, error) {
	r.updatedInput = input
	return RuntimeEnvironment{ID: testEnvironmentID, Name: input.Name}, nil
}

func (r *environmentRepository) DeleteRuntimeEnvironment(_ context.Context, _, _, _ string, input DeleteRuntimeEnvironmentInput) (string, error) {
	r.deletedInput = input
	return "deleting", nil
}

func (r *environmentRepository) CreateRuntimeEnvironment(_ context.Context, _, _ string, input CreateRuntimeEnvironmentInput, code string) (RuntimeEnvironment, error) {
	r.createdInput = input
	return RuntimeEnvironment{ID: testEnvironmentID, Name: input.Name, Code: code}, nil
}

func TestRuntimeEnvironmentUpdateAndDeleteRequireConfirmation(t *testing.T) {
	repository := &environmentRepository{}
	service := NewService(repository, nil)
	ops := auth.User{TenantID: "tenant", ID: "user", Role: "OPS_ADMIN"}
	project := auth.User{TenantID: "tenant", ID: "user", Role: "PROJECT_ADMIN"}
	if _, err := service.UpdateRuntimeEnvironment(context.Background(), project, testEnvironmentID, UpdateRuntimeEnvironmentInput{Name: "新名称"}); !errors.Is(err, auth.ErrPermissionDenied) {
		t.Fatalf("project admin must not edit environment: %v", err)
	}
	if _, err := service.UpdateRuntimeEnvironment(context.Background(), ops, testEnvironmentID, UpdateRuntimeEnvironmentInput{Name: "  新名称  "}); err != nil || repository.updatedInput.Name != "新名称" {
		t.Fatalf("environment update failed: input=%#v err=%v", repository.updatedInput, err)
	}
	if _, err := service.DeleteRuntimeEnvironment(context.Background(), ops, testEnvironmentID, DeleteRuntimeEnvironmentInput{}); err == nil {
		t.Fatal("environment deletion must require a confirmation name")
	}
	if status, err := service.DeleteRuntimeEnvironment(context.Background(), ops, testEnvironmentID, DeleteRuntimeEnvironmentInput{ConfirmationName: " 新名称 "}); err != nil || status != "deleting" || repository.deletedInput.ConfirmationName != "新名称" {
		t.Fatalf("environment deletion failed: status=%s input=%#v err=%v", status, repository.deletedInput, err)
	}
}

func (r *environmentRepository) AddRuntimeEnvironmentNodes(_ context.Context, _, _, _ string, nodeIDs []string) ([]Node, error) {
	r.addedNodeIDs = append([]string(nil), nodeIDs...)
	return []Node{{ID: nodeIDs[0], EnvironmentID: testEnvironmentID}}, nil
}

func TestRuntimeEnvironmentManagementRequiresOpsCapability(t *testing.T) {
	repository := &environmentRepository{}
	service := NewService(repository, nil)
	if _, err := service.CreateRuntimeEnvironment(context.Background(), auth.User{TenantID: "tenant", ID: "user", Role: "PROJECT_ADMIN"}, CreateRuntimeEnvironmentInput{Name: "生产环境"}); !errors.Is(err, auth.ErrPermissionDenied) {
		t.Fatalf("project admin must not create environment: %v", err)
	}
	item, err := service.CreateRuntimeEnvironment(context.Background(), auth.User{TenantID: "tenant", ID: "user", Role: "OPS_ADMIN"}, CreateRuntimeEnvironmentInput{Name: " 生产环境 "})
	if err != nil || repository.createdInput.Name != "生产环境" || item.Code == "" {
		t.Fatalf("ops admin create failed: item=%#v input=%#v err=%v", item, repository.createdInput, err)
	}
}

func TestAddRuntimeEnvironmentNodesValidatesStableIDs(t *testing.T) {
	repository := &environmentRepository{}
	service := NewService(repository, nil)
	actor := auth.User{TenantID: "tenant", ID: "user", Role: "OPS_ADMIN"}
	if _, err := service.AddRuntimeEnvironmentNodes(context.Background(), actor, testEnvironmentID, AddRuntimeEnvironmentNodesInput{NodeIDs: []string{"bad"}}); err == nil {
		t.Fatal("malformed node id must be rejected")
	}
	if _, err := service.AddRuntimeEnvironmentNodes(context.Background(), actor, testEnvironmentID, AddRuntimeEnvironmentNodesInput{NodeIDs: []string{testNodeID, testNodeID}}); err == nil {
		t.Fatal("duplicate node id must be rejected")
	}
	if _, err := service.AddRuntimeEnvironmentNodes(context.Background(), actor, testEnvironmentID, AddRuntimeEnvironmentNodesInput{NodeIDs: []string{testNodeID}}); err != nil || len(repository.addedNodeIDs) != 1 {
		t.Fatalf("valid node association failed: ids=%v err=%v", repository.addedNodeIDs, err)
	}
}

func TestRuntimeEnvironmentListRejectsUnknownStatus(t *testing.T) {
	service := NewService(&environmentRepository{}, nil)
	_, _, err := service.ListRuntimeEnvironments(context.Background(), auth.User{TenantID: "tenant", Role: "OPS_ADMIN"}, PageFilter{Status: "healthy"})
	if err == nil || !strings.Contains(err.Error(), "筛选值无效") {
		t.Fatalf("unknown environment status must be rejected: %v", err)
	}
}

type deploymentRepository struct {
	Repository
	validated           bool
	input               CreateDeploymentInput
	deploymentID        string
	deploymentOperation string
	deploymentErr       error
}

func (r *deploymentRepository) ValidateDeploymentTargets(_ context.Context, _ string, in CreateDeploymentInput) error {
	r.validated = true
	r.input = in
	return nil
}
func (r *deploymentRepository) CreateDeployment(_ context.Context, _, _ string, in CreateDeploymentInput) (ProjectDeployment, DeploymentRun, error) {
	return ProjectDeployment{ProjectID: in.ProjectID, EnvironmentID: in.EnvironmentID, ApplicationVersionID: in.ApplicationVersionID}, DeploymentRun{}, nil
}
func (r *deploymentRepository) OperateService(context.Context, string, string, string, string, string) (ProjectDeployment, DeploymentRun, error) {
	return ProjectDeployment{}, DeploymentRun{}, nil
}
func (r *deploymentRepository) OperateDeployment(_ context.Context, _, deploymentID, operation, _ string) (ProjectDeployment, DeploymentRun, error) {
	r.deploymentID, r.deploymentOperation = deploymentID, operation
	if r.deploymentErr != nil {
		return ProjectDeployment{}, DeploymentRun{}, r.deploymentErr
	}
	return ProjectDeployment{ID: deploymentID, Services: []DeploymentService{{ServiceType: ServiceBase}, {ServiceType: ServiceCompute}, {ServiceType: ServiceCollector}}}, DeploymentRun{ProjectDeploymentID: deploymentID, Operation: operation}, nil
}

func TestCreateDeploymentUsesEnvironmentModeAndPlacements(t *testing.T) {
	r := &deploymentRepository{}
	input := CreateDeploymentInput{ProjectID: testProjectID, EnvironmentID: testEnvironmentID, ApplicationVersionID: testVersionID, Mode: "production", AccessPort: 17800, Placements: map[string]string{ServiceBase: testNodeID}}
	_, _, e := NewService(r, nil).CreateDeployment(context.Background(), auth.User{TenantID: "tenant", Role: "OPS_ADMIN"}, input)
	if e != nil || !r.validated || r.input.EnvironmentID != testEnvironmentID || r.input.Mode != "release" || r.input.Placements[ServiceBase] != testNodeID {
		t.Fatalf("environment deployment not validated: %#v %v", r.input, e)
	}
}
func TestCreateDevelopmentDeploymentDoesNotAcceptUserVersion(t *testing.T) {
	base := CreateDeploymentInput{ProjectID: testProjectID, EnvironmentID: testEnvironmentID, Mode: "development", AccessPort: 17800, Placements: map[string]string{ServiceBase: testNodeID}}
	r := &deploymentRepository{}
	service := NewService(r, nil)
	service.SetDevelopmentArtifactBuilder(func(context.Context, auth.User, string, string) (DevelopmentArtifact, error) {
		return DevelopmentArtifact{ReleaseID: testVersionID, Version: "__DEV__", ArtifactKey: "development/test", ArtifactSize: 1}, nil
	})
	if _, _, err := service.CreateDeployment(context.Background(), auth.User{TenantID: "tenant", Role: "OPS_ADMIN"}, base); err != nil || !r.validated || r.input.DevelopmentArtifact == nil {
		t.Fatalf("development slot request was rejected: %v", err)
	}
	base.ApplicationVersionID = testVersionID
	r = &deploymentRepository{}
	if _, _, err := NewService(r, nil).CreateDeployment(context.Background(), auth.User{TenantID: "tenant", Role: "OPS_ADMIN"}, base); err == nil || r.validated {
		t.Fatalf("development request accepted a user selected version: %v", err)
	}
}

func TestDevelopmentBuildFailureDoesNotReachDeploymentReplacement(t *testing.T) {
	r := &deploymentRepository{}
	service := NewService(r, nil)
	service.SetDevelopmentArtifactBuilder(func(context.Context, auth.User, string, string) (DevelopmentArtifact, error) {
		return DevelopmentArtifact{}, errors.New("build failed")
	})
	input := CreateDeploymentInput{ProjectID: testProjectID, EnvironmentID: testEnvironmentID, Mode: "development", AccessPort: 17800, Placements: map[string]string{ServiceBase: testNodeID}}
	if _, _, err := service.CreateDeployment(context.Background(), auth.User{TenantID: "tenant", Role: "OPS_ADMIN"}, input); err == nil || r.validated {
		t.Fatalf("构建失败不得替换已有部署: err=%v validated=%v", err, r.validated)
	}
}
func TestCreateDeploymentRejectsMalformedNodeOrVersion(t *testing.T) {
	for _, in := range []CreateDeploymentInput{{ProjectID: testProjectID, EnvironmentID: "bad", Mode: "development"}, {ProjectID: testProjectID, EnvironmentID: testEnvironmentID, ApplicationVersionID: "bad", Mode: "production"}} {
		r := &deploymentRepository{}
		_, _, e := NewService(r, nil).CreateDeployment(context.Background(), auth.User{Role: "OPS_ADMIN"}, in)
		if e == nil || !strings.Contains(e.Error(), "格式无效") || r.validated {
			t.Fatalf("invalid input reached repository: %#v %v", in, e)
		}
	}
}

func TestCreateDeploymentRejectsInvalidCustomerPort(t *testing.T) {
	input := CreateDeploymentInput{ProjectID: testProjectID, EnvironmentID: testEnvironmentID, ApplicationVersionID: testVersionID, Mode: "production", AccessPort: 65533}
	r := &deploymentRepository{}
	if _, _, err := NewService(r, nil).CreateDeployment(context.Background(), auth.User{Role: "OPS_ADMIN"}, input); err == nil || !strings.Contains(err.Error(), "工程访问端口") || r.validated {
		t.Fatalf("invalid customer port reached repository: %v", err)
	}
}
func TestOperateDeploymentUsesOneLifecycleOperationForAllServices(t *testing.T) {
	repository := &deploymentRepository{}
	deployment, run, err := NewService(repository, nil).OperateDeployment(context.Background(), auth.User{TenantID: "tenant", Role: "OPS_ADMIN"}, "deployment-1", "restart")
	if err != nil || repository.deploymentID != "deployment-1" || repository.deploymentOperation != "restart" || run.Operation != "restart" || len(deployment.Services) != 3 {
		t.Fatalf("deployment lifecycle operation was not delegated atomically: deployment=%#v run=%#v repository=%#v err=%v", deployment, run, repository, err)
	}
}

func TestOperateDeploymentPreservesBusyAndNotFoundFailures(t *testing.T) {
	for _, expected := range []error{ErrDeploymentBusy, ErrNotFound} {
		repository := &deploymentRepository{deploymentErr: expected}
		_, _, err := NewService(repository, nil).OperateDeployment(context.Background(), auth.User{TenantID: "tenant", Role: "OPS_ADMIN"}, "deployment-1", "stop")
		if !errors.Is(err, expected) {
			t.Fatalf("expected %v, got %v", expected, err)
		}
	}
}

func TestOperateDeploymentRequiresOperateCapability(t *testing.T) {
	repository := &deploymentRepository{}
	_, _, err := NewService(repository, nil).OperateDeployment(context.Background(), auth.User{TenantID: "tenant", Role: "VIEWER"}, "deployment-1", "start")
	if !errors.Is(err, auth.ErrPermissionDenied) || repository.deploymentOperation != "" {
		t.Fatalf("permission denial must prevent deployment operation: repository=%#v err=%v", repository, err)
	}
}

func TestRedeployDoesNotRebuildAndRequiresOperateCapability(t *testing.T) {
	for _, role := range []string{"OPS_ADMIN", "VIEWER"} {
		r := &deploymentRepository{}
		s := NewService(r, nil)
		s.SetDevelopmentArtifactBuilder(func(context.Context, auth.User, string, string) (DevelopmentArtifact, error) {
			t.Fatal("重新部署不得调用开发构建器")
			return DevelopmentArtifact{}, nil
		})
		_, _, err := s.OperateDeployment(context.Background(), auth.User{TenantID: "tenant", Role: role}, "deployment", "redeploy")
		if role == "VIEWER" {
			if err == nil || r.deploymentOperation != "" {
				t.Fatal("无运维权限不能重新部署")
			}
		} else if err != nil || r.deploymentOperation != "redeploy" || r.validated {
			t.Fatalf("重新部署必须直接使用已有制品而不是创建路径: %v", err)
		}
	}
}

func TestOperateServiceIsFailClosed(t *testing.T) {
	repository := &deploymentRepository{}
	_, _, err := NewService(repository, nil).OperateService(context.Background(), auth.User{TenantID: "tenant", Role: "OPS_ADMIN"}, "deployment-1", ServiceBase, "stop")
	if !errors.Is(err, ErrServiceLifecycleDisabled) || repository.deploymentOperation != "" {
		t.Fatalf("service-level lifecycle route must not reach repository: repository=%#v err=%v", repository, err)
	}
}
func TestCapabilityValidationRejectsUnsupportedAndDuplicates(t *testing.T) {
	for _, v := range [][]string{{}, {"unknown"}, {CapabilityCollector, CapabilityCollector}} {
		if validateCapabilities(v) == nil {
			t.Fatalf("must reject %#v", v)
		}
	}
	if validateCapabilities([]string{CapabilityProjectEntry, CapabilityDataRuntime, CapabilityCollector}) != nil {
		t.Fatal("valid capability set rejected")
	}
}

func TestEnrollmentCapabilitiesMustMatchExactly(t *testing.T) {
	requested := []string{CapabilityProjectEntry, CapabilityDataRuntime}
	if !sameStrings([]string{CapabilityDataRuntime, CapabilityProjectEntry}, requested) {
		t.Fatal("capability order must not affect enrollment matching")
	}
	if sameStrings([]string{CapabilityProjectEntry, CapabilityDataRuntime, CapabilityCollector}, requested) {
		t.Fatal("NodeAgent must not escalate beyond the approved capability set")
	}
}

func TestClaimRequiresAuditableAgentIdentity(t *testing.T) {
	service := NewService(&deploymentRepository{}, nil)
	base := ClaimEnrollmentInput{
		Code: "code", Hostname: "edge-01", Platform: PlatformLinux, Architecture: "amd64",
		AgentVersion: "1.0.0", MachineFingerprint: "sha256:fingerprint",
		Capabilities: []string{CapabilityProjectEntry, CapabilityDataRuntime},
	}
	base.MachineFingerprint = ""
	if _, _, _, err := service.ClaimEnrollment(context.Background(), base); err == nil || !strings.Contains(err.Error(), "机器指纹") {
		t.Fatalf("missing fingerprint must be rejected: %v", err)
	}
	base.MachineFingerprint = "sha256:fingerprint"
	base.AgentVersion = ""
	if _, _, _, err := service.ClaimEnrollment(context.Background(), base); err == nil || !strings.Contains(err.Error(), "Agent 版本") {
		t.Fatalf("missing agent version must be rejected: %v", err)
	}
}

func TestHeartbeatRejectsMalformedServiceObservation(t *testing.T) {
	service := NewService(&deploymentRepository{}, nil)
	for _, observation := range []ServiceObservation{
		{ServiceID: "not-a-uuid", ObservedStatus: "running", ObservedGeneration: 1, ReplicasObserved: 1},
		{ServiceID: testVersionID, ObservedStatus: "starting", ObservedGeneration: 1, ReplicasObserved: 1},
		{ServiceID: testVersionID, ObservedStatus: "running", ObservedGeneration: 0, ReplicasObserved: 1},
		{ServiceID: testVersionID, ObservedStatus: "running", ObservedGeneration: 1, ReplicasObserved: 2},
	} {
		if _, _, err := service.Heartbeat(context.Background(), testNodeID, "agent-token", HeartbeatInput{Services: []ServiceObservation{observation}}); err == nil {
			t.Fatalf("malformed service observation accepted: %#v", observation)
		}
	}
}
func TestDefaultServicesAreExplicit(t *testing.T) {
	if !validServiceType(ServiceBase) || !validServiceType(ServiceCompute) || !validServiceType(ServiceAlarm) || !validServiceType(ServiceCollector) || validServiceType("project_entry") {
		t.Fatal("unexpected service vocabulary")
	}
}

func TestReportedEndpointRejectsUnsafeValues(t *testing.T) {
	if err := validateReportedEndpoint("https://gateway.example.com/engineering"); err != nil {
		t.Fatalf("valid endpoint rejected: %v", err)
	}
	for _, endpoint := range []string{
		"ftp://gateway.example.com",
		"https://user:pass@gateway.example.com",
		"https://gateway.example.com/#fragment",
	} {
		if err := validateReportedEndpoint(endpoint); err == nil {
			t.Fatalf("unsafe endpoint accepted: %s", endpoint)
		}
	}
}

type agentReleaseRepository struct {
	Repository
	bindingErr error
	releaseErr error
	release    AgentRelease
}

func (r *agentReleaseRepository) GetAgentDeploymentBinding(context.Context, string, string, string, string) (DeploymentBinding, error) {
	return DeploymentBinding{}, r.bindingErr
}
func (r *agentReleaseRepository) GetAgentRelease(context.Context, string, string, string, string) (AgentRelease, error) {
	return r.release, r.releaseErr
}

type memoryReleaseStore struct{ content []byte }

func (s memoryReleaseStore) Open(context.Context, string) (objectstore.ObjectReader, error) {
	return objectstore.ObjectReader{Reader: io.NopCloser(bytes.NewReader(s.content)), Size: int64(len(s.content)), ContentType: "text/plain"}, nil
}

func TestAgentReleaseRejectsCrossNodeAndRevokedNode(t *testing.T) {
	for _, name := range []string{"cross node", "revoked node"} {
		t.Run(name, func(t *testing.T) {
			service := NewService(&agentReleaseRepository{releaseErr: ErrAgentUnauthorized}, nil, memoryReleaseStore{})
			if _, _, err := service.AgentRelease(context.Background(), testNodeID, "agent-token", "deployment-id", "service-id"); !errors.Is(err, ErrAgentUnauthorized) {
				t.Fatalf("%s must be denied, got %v", name, err)
			}
		})
	}
}

func TestAgentReleaseRejectsMissingSigningMetadata(t *testing.T) {
	metadata := releaseMetadata{ID: testVersionID, ArtifactKey: "releases/project/release.tar.zst", ArtifactHash: strings.Repeat("a", 64), ManifestHash: strings.Repeat("b", 64), ChecksumsHash: strings.Repeat("c", 64), ArtifactSize: 1, Manifest: []byte(validReleaseManifest(testProjectID))}
	if err := validateReleaseMetadata(metadata, testProjectID, false); !errors.Is(err, ErrReleaseNotDeployable) {
		t.Fatalf("missing signing metadata must be rejected, got %v", err)
	}
}

func TestInitialDeploymentBindingUsesCustomerPortAndEmptySecrets(t *testing.T) {
	manifest := strings.Replace(validReleaseManifest(testProjectID), `"requiredNodeCapabilities":["project_entry","data_runtime"]`, `"requiredNodeCapabilities":["project_entry","data_runtime","collector"]`, 1)
	release := releaseMetadata{ID: testVersionID, ArtifactKey: "releases/project/release.tar.zst", ArtifactHash: strings.Repeat("a", 64), ManifestHash: strings.Repeat("b", 64), ChecksumsHash: strings.Repeat("c", 64), SigningKeyID: "induforge-release-2026-01", ArtifactSize: 1, Manifest: []byte(manifest)}
	deploymentID := "55555555-5555-4555-8555-555555555555"
	bindingID, raw, err := newInitialDeploymentBinding(ProjectDeployment{ID: deploymentID, ProjectID: testProjectID, AccessPort: 17800}, release, testNodeID, []string{ServiceBase, ServiceCompute, ServiceCollector})
	if err != nil {
		t.Fatal(err)
	}
	var binding struct {
		Revision int            `json:"revision"`
		Ports    map[string]int `json:"ports"`
		Secrets  []any          `json:"secrets"`
	}
	if err := json.Unmarshal(raw, &binding); err != nil {
		t.Fatal(err)
	}
	if binding.Revision != 1 || binding.Ports["gatewayPublic"] != 17800 || binding.Ports["runtimeApiLoopback"] != 17801 || binding.Ports["engineLoopback"] != 17802 || binding.Ports["collectorHealthLoopback"] != 17803 || len(binding.Secrets) != 0 {
		t.Fatalf("unexpected fail-closed binding: %s", raw)
	}
	if err := validateInitialDeploymentBinding(raw, bindingMetadata{ID: bindingID, ProjectID: testProjectID, Revision: 1}, release, testNodeID, deploymentID); err != nil {
		t.Fatalf("valid generated binding rejected: %v", err)
	}
	var tampered map[string]any
	if err := json.Unmarshal(raw, &tampered); err != nil {
		t.Fatal(err)
	}
	tampered["secrets"] = []any{map[string]any{"name": "forbidden"}}
	tamperedRaw, _ := json.Marshal(tampered)
	if err := validateInitialDeploymentBinding(tamperedRaw, bindingMetadata{ID: bindingID, ProjectID: testProjectID, Revision: 1}, release, testNodeID, deploymentID); !errors.Is(err, ErrReleaseNotDeployable) {
		t.Fatalf("binding with secrets must be rejected, got %v", err)
	}
}

func TestEngineDeploymentBindingAcceptsSignedDevelopmentArtifact(t *testing.T) {
	development := &DevelopmentArtifact{
		ReleaseID:     testVersionID,
		Version:       "__DEV__",
		ArtifactKey:   "development/tenant/project/dev.tar.zst",
		ArtifactHash:  strings.Repeat("a", 64),
		ManifestHash:  strings.Repeat("b", 64),
		ChecksumsHash: strings.Repeat("c", 64),
		SigningKeyID:  "induforge-release-2026-01",
		ArtifactSize:  1,
		Manifest:      []byte(validReleaseManifest(testProjectID)),
	}
	release, err := developmentReleaseMetadata(development, testProjectID)
	if err != nil {
		t.Fatalf("开发制品元数据无效: %v", err)
	}
	deploymentID := "55555555-5555-4555-8555-555555555555"
	deployment := ProjectDeployment{ID: deploymentID, ProjectID: testProjectID, EnvironmentID: testEnvironmentID, Mode: "development", AccessPort: 20000}
	bindingID, raw, err := newEngineDeploymentBinding(deployment, release, "66666666-6666-4666-8666-666666666666", testNodeID, ServiceBase, 1, 20000)
	if err != nil {
		t.Fatal(err)
	}
	if err = validateInitialDeploymentBinding(raw, bindingMetadata{ID: bindingID, ProjectID: testProjectID, Revision: 1}, release, testNodeID, deploymentID); err != nil {
		t.Fatalf("已签开发制品的引擎绑定不应回退到生产 Release 校验: %v", err)
	}
}

func TestEngineDeploymentBindingAcceptsCollectorReleaseForNonCollectorEngine(t *testing.T) {
	manifest := strings.Replace(validReleaseManifest(testProjectID), `"capabilities":["runtime.auth","runtime.datapoint"]`, `"capabilities":["base","compute","alarm","collector"]`, 1)
	manifest = strings.Replace(manifest, `"requiredNodeCapabilities":["project_entry","data_runtime"]`, `"requiredNodeCapabilities":["collector","data_runtime","project_entry"]`, 1)
	release := releaseMetadata{ID: testVersionID, Version: "1.0.24", ArtifactKey: "releases/tenant/project/release.tar.zst", ArtifactHash: strings.Repeat("a", 64), ManifestHash: strings.Repeat("b", 64), ChecksumsHash: strings.Repeat("c", 64), SigningKeyID: "induforge-release-2026-01", ArtifactSize: 1, Manifest: []byte(manifest)}
	deploymentID := "55555555-5555-4555-8555-555555555555"
	deployment := ProjectDeployment{ID: deploymentID, ProjectID: testProjectID, EnvironmentID: testEnvironmentID, Mode: "release", AccessPort: 20000}
	bindingID, raw, err := newEngineDeploymentBinding(deployment, release, "66666666-6666-4666-8666-666666666666", testNodeID, ServiceCompute, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = validateInitialDeploymentBinding(raw, bindingMetadata{ID: bindingID, ProjectID: testProjectID, Revision: 1}, release, testNodeID, deploymentID); err != nil {
		t.Fatalf("非 collector 引擎不得把整包 Release 的 collector 能力误判为不匹配: %v", err)
	}
}

func TestAgentCommandCarriesReleaseAndBindingIdentifiers(t *testing.T) {
	raw, err := json.Marshal(AgentCommand{DeploymentID: "deployment", ReleaseID: testVersionID, ArchiveSHA256: "sha256:" + strings.Repeat("a", 64), ManifestSHA256: "sha256:" + strings.Repeat("b", 64), ChecksumsSHA256: "sha256:" + strings.Repeat("c", 64), SigningKeyID: "induforge-release-2026-01", BindingRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"releaseId", "archiveSha256", "manifestSha256", "checksumsSha256", "signingKeyId", "bindingRevision"} {
		if !strings.Contains(string(raw), `"`+field+`"`) {
			t.Fatalf("agent command missing %s: %s", field, raw)
		}
	}
}
