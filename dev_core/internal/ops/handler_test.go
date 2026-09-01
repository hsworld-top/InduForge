package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/app"
	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"net/http"
	"net/http/httptest"
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
	return []AgentCommand{{NodeID: "node-1", ServiceID: "service-1", ServiceType: ServiceProjectEntry}}, nil
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
	req = req.WithContext(auth.WithUser(req.Context(), auth.User{TenantID: "tenant", Role: "OPERATOR"}))
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
		{ServiceType: ServiceCollector, Endpoint: "https://collector.example.invalid"},
		{ServiceType: ServiceDataRuntime, Endpoint: "https://runtime.example.invalid"},
		{ServiceType: ServiceProjectEntry, Endpoint: "https://gateway.example.com/engineering"},
	}}
	payload := deploymentPayload(deployment)
	if payload["accessUrl"] != "https://gateway.example.com/engineering" {
		t.Fatalf("accessUrl=%v", payload["accessUrl"])
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
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/node-enrollments/id/approve", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/node-enrollments/id/reject", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/project-deployments", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/project-deployments/id/restart", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/project-deployments/id/services/project_entry/start", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/agent/enrollments/claim", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/agent/nodes/id/heartbeat", nil)
}
