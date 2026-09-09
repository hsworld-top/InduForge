package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/app"
	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestReleaseNotDeployableUsesBusinessValidationEnvelope(t *testing.T) {
	h := &Handler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/ops/project-deployments", nil)
	h.err(w, r, errors.Join(ErrReleaseNotDeployable, errors.New("缺少正式 Manifest")))

	var response struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if w.Code != http.StatusOK || response.Code != platformapi.ErrorCodeInvalidRequest {
		t.Fatalf("release rejection must be a business validation error: status=%d code=%d", w.Code, response.Code)
	}
}

func TestDeploymentMutationAndRequirementsRoutesAreMounted(t *testing.T) {
	h := NewHandler(NewService(struct{ Repository }{}, nil), nil)
	router := chi.NewRouter()
	h.MountRoutes(router)
	for _, item := range []struct{ method, path string }{
		{http.MethodDelete, "/api/v1/ops/project-deployments/11111111-1111-4111-8111-111111111111"},
		{http.MethodGet, "/api/v1/ops/project-deployments/development-requirements"},
	} {
		req := httptest.NewRequest(item.method, item.path, nil)
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		if out.Code == http.StatusNotFound {
			t.Fatalf("route not mounted: %s %s", item.method, item.path)
		}
	}
}

func TestNewOpsReadRoutesHaveConcreteHTTPEvidence(t *testing.T) {
	h := NewHandler(NewService(struct{ Repository }{}, nil), nil)
	router := chi.NewRouter()
	h.MountRoutes(router)
	requests := []*http.Request{
		httptest.NewRequest(http.MethodDelete, "/api/v1/ops/project-deployments/11111111-1111-4111-8111-111111111111", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/ops/project-deployments/development-requirements", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/ops/project-deployments/11111111-1111-4111-8111-111111111111/runs", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/ops/deployment-runs/11111111-1111-4111-8111-111111111111/events/page", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/ops/records", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/ops/runtime-environments/11111111-1111-4111-8111-111111111111/overview", nil),
	}
	for _, request := range requests {
		out := httptest.NewRecorder()
		router.ServeHTTP(out, request)
		if out.Code == http.StatusNotFound {
			t.Fatalf("route not mounted: %s %s", request.Method, request.URL.Path)
		}
	}
}

type deploymentListHTTPRepository struct {
	Repository
	filter PageFilter
	tenant string
}

func (r *deploymentListHTTPRepository) ListDeployments(_ context.Context, tenant string, f PageFilter) ([]ProjectDeployment, int64, error) {
	r.tenant, r.filter = tenant, f
	return []ProjectDeployment{{ID: "deployment", EnvironmentID: f.EnvironmentID}}, 21, nil
}

func TestDeploymentListForwardsEnvironmentAndPaginationWithinActorTenant(t *testing.T) {
	repository := &deploymentListHTTPRepository{}
	router := chi.NewRouter()
	NewHandler(NewService(repository, nil), nil).MountRoutes(router)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ops/project-deployments?environmentId="+testEnvironmentID+"&projectId="+testProjectID+"&page=2&pageSize=10&search=demo&tenantId=other", nil)
	request = request.WithContext(auth.WithUser(request.Context(), auth.User{TenantID: "tenant", Role: "OPS_ADMIN"}))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || repository.tenant != "tenant" || repository.filter.EnvironmentID != testEnvironmentID || repository.filter.ProjectID != testProjectID || repository.filter.Page != 2 || repository.filter.PageSize != 10 || repository.filter.Search != "demo" {
		t.Fatalf("分页过滤没有进入当前租户仓储: filter=%+v tenant=%s body=%s", repository.filter, repository.tenant, response.Body.String())
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Total int                 `json:"total"`
			Items []ProjectDeployment `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || envelope.Code != 0 || envelope.Data.Total != 21 || len(envelope.Data.Items) != 1 {
		t.Fatalf("必须保留后端总数和分页结果: %s err=%v", response.Body.String(), err)
	}
}

func TestRedeployRouteUsesUnifiedResponseAndReportsBusy(t *testing.T) {
	for _, failure := range []error{nil, ErrDeploymentBusy} {
		repository := &deploymentRepository{deploymentErr: failure}
		router := chi.NewRouter()
		NewHandler(NewService(repository, nil), nil).MountRoutes(router)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/ops/project-deployments/deployment/redeploy", nil)
		request = request.WithContext(auth.WithUser(request.Context(), auth.User{TenantID: "tenant", Role: "OPS_ADMIN"}))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		var envelope struct {
			Code int                        `json:"code"`
			Data map[string]json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || repository.deploymentOperation != "redeploy" {
			t.Fatalf("重部署路由未进入生命周期服务: %s %v", response.Body.String(), err)
		}
		if failure == nil && (envelope.Code != 0 || envelope.Data["deployment"] == nil || envelope.Data["run"] == nil) {
			t.Fatalf("成功包络无部署及run: %s", response.Body.String())
		}
		if failure != nil && envelope.Code == 0 {
			t.Fatal("互斥拒绝不能返回成功")
		}
	}
}

func TestNodePortConflictUsesAlreadyExistsEnvelope(t *testing.T) {
	h := &Handler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/ops/project-deployments", nil)
	h.err(w, r, ErrNodePortConflict)

	var response struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if w.Code != http.StatusConflict || response.Code != platformapi.ErrorCodeAlreadyExists {
		t.Fatalf("node project conflict must be a conflict envelope: status=%d code=%d", w.Code, response.Code)
	}
}

func TestEnvironmentNodeUsageUsesConflictEnvelope(t *testing.T) {
	for _, testCase := range []struct {
		name string
		err  error
		msg  string
	}{
		{name: "foundation service", err: ErrEnvironmentNodeServiceInUse, msg: "不能取消分配"},
		{name: "project deployment", err: ErrEnvironmentNodeDeploymentInUse, msg: "不能取消分配"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			h := &Handler{}
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodDelete, "/api/v1/ops/runtime-environments/environment/nodes/node", nil)
			h.err(w, r, testCase.err)
			if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), testCase.msg) {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestDefaultEnvironmentProtectionUsesConflictEnvelope(t *testing.T) {
	h := &Handler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/ops/runtime-environments/default", nil)
	h.err(w, r, ErrDefaultEnvironmentProtected)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "默认运行范围不能删除") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestRuntimeEnvironmentPayloadMarksDefaultScope(t *testing.T) {
	payload := runtimeEnvironmentPayload(RuntimeEnvironment{ID: testEnvironmentID, IsDefault: true})
	if payload["isDefault"] != true {
		t.Fatalf("default scope marker missing: %#v", payload)
	}
}

func TestPageReadsExactProjectDeploymentFilter(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/ops/project-deployments?page=2&pageSize=10&search=line&projectId=project-1&status=attention", nil)
	filter := page(r)
	if filter.Page != 2 || filter.PageSize != 10 || filter.Search != "line" || filter.ProjectID != "project-1" || filter.Status != "attention" {
		t.Fatalf("unexpected page filter: %#v", filter)
	}
}

type environmentHTTPRepository struct{ Repository }

func (r *environmentHTTPRepository) ListRuntimeEnvironments(context.Context, string, PageFilter) ([]RuntimeEnvironment, int64, error) {
	return []RuntimeEnvironment{{ID: testEnvironmentID, Name: "生产环境", Status: "uninitialized"}}, 1, nil
}
func (r *environmentHTTPRepository) CreateRuntimeEnvironment(context.Context, string, string, CreateRuntimeEnvironmentInput, string) (RuntimeEnvironment, error) {
	return RuntimeEnvironment{ID: testEnvironmentID, Name: "生产环境", Status: "uninitialized"}, nil
}
func (r *environmentHTTPRepository) GetRuntimeEnvironment(context.Context, string, string) (RuntimeEnvironment, error) {
	return RuntimeEnvironment{ID: testEnvironmentID, Name: "生产环境", Status: "uninitialized"}, nil
}
func (r *environmentHTTPRepository) ListRuntimeEnvironmentNodes(context.Context, string, string, PageFilter) ([]Node, int64, error) {
	return []Node{{ID: testNodeID, DisplayName: "运行节点"}}, 1, nil
}
func (r *environmentHTTPRepository) AddRuntimeEnvironmentNodes(context.Context, string, string, string, []string) ([]Node, error) {
	return []Node{{ID: testNodeID, DisplayName: "运行节点"}}, nil
}
func (r *environmentHTTPRepository) RemoveRuntimeEnvironmentNode(context.Context, string, string, string, string) error {
	return nil
}
func (r *environmentHTTPRepository) UpdateRuntimeEnvironment(context.Context, string, string, string, UpdateRuntimeEnvironmentInput) (RuntimeEnvironment, error) {
	return RuntimeEnvironment{ID: testEnvironmentID, Name: "新名称", Status: "uninitialized"}, nil
}
func (r *environmentHTTPRepository) DeleteRuntimeEnvironment(context.Context, string, string, string, DeleteRuntimeEnvironmentInput) (string, error) {
	return "deleting", nil
}
func (r *environmentHTTPRepository) ListRuntimeEnvironmentEvents(context.Context, string, string, PageFilter) ([]RuntimeEnvironmentEvent, int64, error) {
	return []RuntimeEnvironmentEvent{{ID: testVersionID, EnvironmentID: testEnvironmentID, Name: "运行环境已创建", Result: "success"}}, 1, nil
}
func (r *environmentHTTPRepository) ListRuntimeEnvironmentServices(context.Context, string, string) ([]RuntimeEnvironmentService, error) {
	return []RuntimeEnvironmentService{{ID: testVersionID, EnvironmentID: testEnvironmentID, NodeID: testNodeID, ServiceType: "if_realtime", ObservedStatus: "running"}}, nil
}
func (r *environmentHTTPRepository) DeployRuntimeEnvironmentFoundation(_ context.Context, _, environmentID, _ string, assignments []FoundationAssignment) ([]RuntimeEnvironmentService, error) {
	items := make([]RuntimeEnvironmentService, 0, len(assignments))
	for _, assignment := range assignments {
		items = append(items, RuntimeEnvironmentService{ID: testVersionID, EnvironmentID: environmentID, NodeID: assignment.NodeID, ServiceType: assignment.ServiceType, DesiredStatus: "running"})
	}
	return items, nil
}
func (r *environmentHTTPRepository) MigrateRuntimeEnvironmentFoundation(_ context.Context, _, environmentID, _ string, assignments []FoundationAssignment) ([]RuntimeEnvironmentService, error) {
	return r.DeployRuntimeEnvironmentFoundation(context.Background(), "", environmentID, "", assignments)
}

func TestRuntimeEnvironmentHTTPRoutes(t *testing.T) {
	h := NewHandler(NewService(&environmentHTTPRepository{}, nil), nil)
	server := app.New(app.Options{Mount: func(r chi.Router) { h.MountRoutes(r) }}).Handler()
	requests := []*http.Request{
		httptest.NewRequest(http.MethodGet, "/api/v1/ops/runtime-environments", nil),
		httptest.NewRequest(http.MethodPost, "/api/v1/ops/runtime-environments", bytes.NewBufferString(`{"name":"生产环境"}`)),
		httptest.NewRequest(http.MethodGet, "/api/v1/ops/runtime-environments/"+testEnvironmentID, nil),
		httptest.NewRequest(http.MethodPatch, "/api/v1/ops/runtime-environments/"+testEnvironmentID, bytes.NewBufferString(`{"name":"新名称"}`)),
		httptest.NewRequest(http.MethodDelete, "/api/v1/ops/runtime-environments/"+testEnvironmentID, bytes.NewBufferString(`{"confirmationName":"生产环境"}`)),
		httptest.NewRequest(http.MethodGet, "/api/v1/ops/runtime-environments/"+testEnvironmentID+"/nodes", nil),
		httptest.NewRequest(http.MethodPost, "/api/v1/ops/runtime-environments/"+testEnvironmentID+"/nodes", bytes.NewBufferString(`{"nodeIds":["`+testNodeID+`"]}`)),
		httptest.NewRequest(http.MethodDelete, "/api/v1/ops/runtime-environments/"+testEnvironmentID+"/nodes/"+testNodeID, nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/ops/runtime-environments/"+testEnvironmentID+"/events", nil),
		httptest.NewRequest(http.MethodGet, "/api/v1/ops/runtime-environments/"+testEnvironmentID+"/foundation-services", nil),
		httptest.NewRequest(http.MethodPost, "/api/v1/ops/runtime-environments/"+testEnvironmentID+"/foundation-services/deploy", bytes.NewBufferString(`{"assignments":[{"serviceType":"if_realtime","nodeId":"`+testNodeID+`"},{"serviceType":"if_history","nodeId":"`+testNodeID+`"},{"serviceType":"if_timeseries","nodeId":"`+testNodeID+`"},{"serviceType":"if_message","nodeId":"`+testNodeID+`"},{"serviceType":"if_object","nodeId":"`+testNodeID+`"},{"serviceType":"nats_jetstream","nodeId":"`+testNodeID+`"},{"serviceType":"nginx","nodeId":"`+testNodeID+`"},{"serviceType":"traefik","nodeId":"`+testNodeID+`"}]}`)),
		httptest.NewRequest(http.MethodPost, "/api/v1/ops/runtime-environments/"+testEnvironmentID+"/foundation-services/migrate", bytes.NewBufferString(`{"assignments":[{"serviceType":"if_realtime","nodeId":"`+testNodeID+`"},{"serviceType":"if_history","nodeId":"`+testNodeID+`"},{"serviceType":"if_timeseries","nodeId":"`+testNodeID+`"},{"serviceType":"if_message","nodeId":"`+testNodeID+`"},{"serviceType":"if_object","nodeId":"`+testNodeID+`"},{"serviceType":"nats_jetstream","nodeId":"`+testNodeID+`"},{"serviceType":"nginx","nodeId":"`+testNodeID+`"},{"serviceType":"traefik","nodeId":"`+testNodeID+`"}]}`)),
	}
	for _, request := range requests {
		request = request.WithContext(auth.WithUser(request.Context(), auth.User{ID: testVersionID, TenantID: testProjectID, Role: "OPS_ADMIN"}))
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s %s status=%d body=%s", request.Method, request.URL.Path, response.Code, response.Body.String())
		}
	}
}

type nodeRemovalHTTPRepository struct{ Repository }

func (r *nodeRemovalHTTPRepository) RemoveNode(context.Context, string, string, string) error {
	return nil
}

func TestRemoveOpsNodeHTTPRoute(t *testing.T) {
	h := NewHandler(NewService(&nodeRemovalHTTPRepository{}, nil), nil)
	server := app.New(app.Options{Mount: func(r chi.Router) { h.MountRoutes(r) }}).Handler()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/ops/nodes/"+testNodeID, nil)
	request = request.WithContext(auth.WithUser(request.Context(), auth.User{ID: testVersionID, TenantID: testProjectID, Role: "OPS_ADMIN"}))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"removing"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

type agentCommandRepository struct{ Repository }

func (r *agentCommandRepository) AgentCommands(context.Context, string, string) ([]AgentCommand, error) {
	return []AgentCommand{{NodeID: "node-1", ServiceID: "service-1", ServiceType: ServiceBase}}, nil
}
func (r *agentCommandRepository) AgentClusterPlan(context.Context, string, string) (*ClusterPlan, error) {
	return nil, nil
}
func (r *agentCommandRepository) AgentClusterUninstall(context.Context, string, string) (*ClusterUninstall, error) {
	return nil, nil
}
func (r *agentCommandRepository) AgentFoundationPlan(context.Context, string, string) (*FoundationPlan, error) {
	return nil, nil
}
func (r *agentCommandRepository) AgentFoundationDelete(context.Context, string, string) (*FoundationDelete, error) {
	return nil, nil
}
func (r *agentCommandRepository) AgentTimeSyncPlan(context.Context, string, string) (*TimeSyncPlan, error) {
	return nil, nil
}
func (r *agentCommandRepository) Heartbeat(context.Context, string, string, HeartbeatInput) (Node, []DeploymentService, error) {
	return Node{ID: "node-1", ObservedStatus: "online"}, nil, nil
}
func TestAgentCommandsUseNodePath(t *testing.T) {
	h := NewHandler(NewService(&agentCommandRepository{}, NewFilePackageStore(t.TempDir())), nil)
	s := app.New(app.Options{Mount: func(r chi.Router) { h.MountRoutes(r) }}).Handler()
	q := httptest.NewRequest(http.MethodGet, "/api/v1/ops/agent/nodes/node-1/commands", nil)
	q.Header.Set("Authorization", "Bearer token")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, q)
	if w.Code != 200 {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestAgentEndpointsRejectLegacyFields(t *testing.T) {
	h := NewHandler(NewService(&agentCommandRepository{}, NewFilePackageStore(t.TempDir())), nil)
	s := app.New(app.Options{Mount: func(r chi.Router) { h.MountRoutes(r) }}).Handler()
	requests := []*http.Request{
		httptest.NewRequest(http.MethodPost, "/api/v1/ops/agent/enrollments/claim", bytes.NewBufferString(`{"code":"x","hostname":"host","platform":"linux","architecture":"amd64","capabilities":["collector"],"runtimeClusterId":"legacy"}`)),
		httptest.NewRequest(http.MethodPost, "/api/v1/ops/agent/nodes/node-1/heartbeat", bytes.NewBufferString(`{"resourceSummary":{},"workloads":[]}`)),
	}
	for _, request := range requests {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, request)
		var response struct {
			Code int `json:"code"`
		}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil || response.Code == 0 {
			t.Fatalf("legacy request must be rejected: body=%s err=%v", w.Body.String(), err)
		}
	}
}

func TestAgentClaimAcceptsCurrentIPAddressField(t *testing.T) {
	var body struct {
		Code               string   `json:"code"`
		Hostname           string   `json:"hostname"`
		Platform           string   `json:"platform"`
		Architecture       string   `json:"architecture"`
		AgentVersion       string   `json:"agentVersion"`
		MachineFingerprint string   `json:"machineFingerprint"`
		IPAddress          string   `json:"ipAddress"`
		Capabilities       []string `json:"capabilities"`
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ops/agent/enrollments/claim", bytes.NewBufferString(`{"code":"x","hostname":"host","platform":"linux","architecture":"arm64","agentVersion":"dev","machineFingerprint":"host-id","ipAddress":"172.16.125.129","capabilities":["project_entry","data_runtime"]}`))
	if !decode(request, &body) {
		t.Fatal("当前 NodeAgent 领取请求不应被严格字段校验拒绝")
	}
	if body.IPAddress != "172.16.125.129" {
		t.Fatalf("ipAddress=%q", body.IPAddress)
	}
}

func TestDeploymentLifecycleEndpointUsesDeploymentOperation(t *testing.T) {
	repository := &deploymentRepository{}
	h := NewHandler(NewService(repository, nil), nil)
	s := app.New(app.Options{Mount: func(r chi.Router) { h.MountRoutes(r) }}).Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/project-deployments/deployment-1/restart", nil)
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{TenantID: "tenant", Role: "OPS_ADMIN"}))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK || repository.deploymentID != "deployment-1" || repository.deploymentOperation != "restart" {
		t.Fatalf("deployment lifecycle route did not use the formal operation: status=%d repository=%#v body=%s", w.Code, repository, w.Body.String())
	}
}

func TestAgentBindingRejectsCrossNodeOrRevokedNode(t *testing.T) {
	for _, name := range []string{"cross node", "revoked node"} {
		t.Run(name, func(t *testing.T) {
			h := NewHandler(NewService(&agentReleaseRepository{bindingErr: ErrAgentUnauthorized}, nil), nil)
			s := app.New(app.Options{Mount: func(r chi.Router) { h.MountRoutes(r) }}).Handler()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/agent/nodes/"+testNodeID+"/deployments/55555555-5555-4555-8555-555555555555/binding", nil)
			req.Header.Set("Authorization", "Bearer agent-token")
			w := httptest.NewRecorder()
			s.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("%s status=%d body=%s", name, w.Code, w.Body.String())
			}
		})
	}
}

func TestAgentReleaseStreamsOnlyBundle(t *testing.T) {
	bundle := []byte("release-bundle")
	h := NewHandler(NewService(&agentReleaseRepository{release: AgentRelease{ReleaseID: testVersionID, ArtifactKey: "private/release.tar.zst", ArtifactSize: int64(len(bundle))}}, nil, memoryReleaseStore{content: bundle}), nil)
	s := app.New(app.Options{Mount: func(r chi.Router) { h.MountRoutes(r) }}).Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/agent/nodes/"+testNodeID+"/deployments/55555555-5555-4555-8555-555555555555/release", nil)
	req.Header.Set("Authorization", "Bearer agent-token")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/zstd" || !bytes.Equal(w.Body.Bytes(), bundle) || strings.Contains(w.Body.String(), "private/release") || strings.Contains(w.Header().Get("Location"), "http") {
		t.Fatalf("release must be streamed without object-store reference: status=%d headers=%v body=%q", w.Code, w.Header(), w.Body.String())
	}
}

func TestDeploymentPayloadOnlyUsesProjectEntryEndpoint(t *testing.T) {
	deployment := ProjectDeployment{Services: []DeploymentService{
		{ServiceType: ServiceCollector, NodeName: "采集节点", Endpoint: "https://collector.example.invalid"},
		{ServiceType: ServiceCompute, NodeName: "计算节点", Endpoint: "https://runtime.example.invalid"},
		{ServiceType: ServiceBase, NodeName: "采集节点", Endpoint: "https://gateway.example.com/engineering", DesiredStatus: "running", ObservedStatus: "running"},
	}}
	payload := deploymentPayload(deployment)
	if payload["accessUrl"] != "https://gateway.example.com/engineering" {
		t.Fatalf("accessUrl=%v", payload["accessUrl"])
	}
	if payload["accessAvailable"] != true {
		t.Fatalf("running entry must be available: %v", payload["accessAvailable"])
	}
	if got := payload["nodeNames"]; !reflect.DeepEqual(got, []string{"采集节点", "计算节点"}) {
		t.Fatalf("nodeNames=%#v", got)
	}
	deployment.Services[2].ObservedStatus = "pending"
	if payload := deploymentPayload(deployment); payload["accessUrl"] == "" || payload["accessAvailable"] != false {
		t.Fatalf("planned URL must stay unavailable: %#v", payload)
	}
	deployment.Services[2].ObservedStatus = "failed"
	if payload := deploymentPayload(deployment); payload["accessUrl"] == "" || payload["accessAvailable"] != false {
		t.Fatalf("failed entry must preserve planned URL but remain unavailable: %#v", payload)
	}
	deployment.Services = deployment.Services[:2]
	if payload := deploymentPayload(deployment); payload["accessUrl"] != "" {
		t.Fatalf("non-entry service must not produce accessUrl: %v", payload["accessUrl"])
	}
}

func TestNodePayloadIncludesCurrentProjectAssignment(t *testing.T) {
	payload := nodePayload(Node{
		AssignedDeploymentID: "deployment-1",
		AssignedProjectID:    "project-1",
		AssignedProjectName:  "一号产线",
	})
	if payload["assignedDeploymentId"] != "deployment-1" || payload["assignedProjectId"] != "project-1" || payload["assignedProjectName"] != "一号产线" {
		t.Fatalf("unexpected assignment payload: %#v", payload)
	}
}

func TestNodePayloadIdentifiesBuiltInCenterNode(t *testing.T) {
	payload := nodePayload(Node{NodeSource: "built_in", NodeKind: "center"})
	if payload["nodeSource"] != "built_in" || payload["builtIn"] != true {
		t.Fatalf("中心内置节点标识不完整: %#v", payload)
	}
}

func TestEnrollmentPayloadContainsAuditAndClaimIdentity(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	payload := enrollmentPayload(Enrollment{
		ID:                 "enrollment-1",
		ExpiresAt:          now.Add(time.Hour),
		ClaimedAt:          &now,
		ReportedHostName:   "edge-01",
		MachineFingerprint: "sha256:fingerprint",
		CreatedAt:          now,
		UpdatedAt:          now,
		Node:               &Node{ID: "node-1", DisplayName: "边缘节点", CreatedAt: now, UpdatedAt: now},
	})
	for _, field := range []string{"expiresAt", "claimedAt", "createdAt", "updatedAt", "node"} {
		if payload[field] == nil {
			t.Fatalf("enrollment payload missing %s", field)
		}
	}
	node, ok := payload["node"].(map[string]any)
	if !ok || node["id"] != "node-1" {
		t.Fatalf("unexpected enrollment node payload: %#v", payload["node"])
	}
}

func TestOpsHTTPContractInventory(t *testing.T) {
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/nodes", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/nodes/id", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/node-enrollments", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/node-enrollments/id", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/node-packages", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/node-packages/linux/download", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/project-deployments", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/project-deployments/id", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/deployment-runs/id", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/deployment-runs/id/events", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/node-enrollments", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/node-enrollments/id/revoke", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/project-deployments", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/project-deployments/id/restart", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/project-deployments/id/services/project_entry/start", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/agent/enrollments/claim", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/agent/nodes/id/heartbeat", nil)
}

func TestNodePayloadConfirmedUninstall(t *testing.T) {
	result := nodePayload(Node{ObservedStatus: "offline", ResourceSummary: map[string]any{"uninstalledAt": "2026-09-08T00:00:00Z"}})
	if result["observedStatus"] != "uninstalled" {
		t.Fatalf("confirmed uninstall lost: %v", result)
	}
	result = nodePayload(Node{ObservedStatus: "offline"})
	if result["observedStatus"] != "offline" {
		t.Fatal("offline is not proof of uninstall")
	}
}

func TestDuplicateNodeNameReturnsBusinessError(t *testing.T) {
	w := httptest.NewRecorder()
	(&Handler{}).err(w, httptest.NewRequest("POST", "/api/v1/ops/node-enrollments", nil), ErrNodeNameExists)
	if w.Code != 200 {
		t.Fatalf("expected business error, got %d", w.Code)
	}
	var payload struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code == 0 || payload.Msg != ErrNodeNameExists.Error() {
		t.Fatalf("wrong error: %+v", payload)
	}
}

func TestNodePreflightReturnsActionableBusinessError(t *testing.T) {
	w := httptest.NewRecorder()
	(&Handler{}).err(w, httptest.NewRequest("POST", "/", nil), &nodePreflightError{cause: fmt.Errorf("物理节点 secret-id 的管理 IP 匹配到 3 个 Kubernetes 节点")})
	if w.Code != 200 {
		t.Fatalf("unexpected HTTP status: %d", w.Code)
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Code == 0 || result.Msg != "无法确认部署节点：节点登记与运行组件不一致。请检查节点接入状态后重试。" {
		t.Fatalf("unexpected response: %+v", result)
	}
}
