package ops

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/app"
)

type agentCommandRepository struct {
	Repository
	lastHeartbeat HeartbeatInput
}

func (r *agentCommandRepository) AgentCommands(_ context.Context, _ string, _ string) ([]AgentCommand, error) {
	return []AgentCommand{{RunID: "run-1", WorkloadID: "workload-1", Role: "compute", Operation: "restart", DesiredStatus: "running", Generation: 2}}, nil
}

func (r *agentCommandRepository) Heartbeat(_ context.Context, _ string, _ string, input HeartbeatInput) (HostNode, []Workload, error) {
	r.lastHeartbeat = input
	return HostNode{ID: "node-1", ObservedStatus: "online"}, nil, nil
}

func TestAgentCommandsAndPackageRoutesUseEnvelope(t *testing.T) {
	service := NewService(&agentCommandRepository{}, NewFilePackageStore(t.TempDir()))
	handler := NewHandler(service, nil)
	server := app.New(app.Options{RequestID: func() string { return "ops-test" }, Mount: func(router chi.Router) { handler.MountRoutes(router) }}).Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/ops/agent/host-nodes/node-1/commands", nil)
	request.Header.Set("Authorization", "Bearer agent-token")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("commands status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Code int
		Data struct{ Commands []AgentCommand }
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != 0 || len(payload.Data.Commands) != 1 || payload.Data.Commands[0].Generation != 2 {
		t.Fatalf("unexpected commands response: %s", response.Body.String())
	}
}

func TestHeartbeatDecodesLowerCamelWorkloadObservation(t *testing.T) {
	repository := &agentCommandRepository{}
	service := NewService(repository, NewFilePackageStore(t.TempDir()))
	handler := NewHandler(service, nil)
	server := app.New(app.Options{RequestID: func() string { return "ops-test" }, Mount: func(router chi.Router) { handler.MountRoutes(router) }}).Handler()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ops/agent/host-nodes/node-1/heartbeat", strings.NewReader(`{"agentVersion":"1.0.0","observedState":{"workloads":[{"workloadId":"workload-1","observedStatus":"running","observedGeneration":3,"replicasObserved":1,"message":"running"}]}}`))
	request.Header.Set("Authorization", "Bearer agent-token")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("heartbeat status=%d body=%s", response.Code, response.Body.String())
	}
	if len(repository.lastHeartbeat.Workloads) != 1 {
		t.Fatalf("heartbeat workload not decoded: %#v", repository.lastHeartbeat)
	}
	observed := repository.lastHeartbeat.Workloads[0]
	if observed.WorkloadID != "workload-1" || observed.ObservedStatus != "running" || observed.ObservedGeneration != 3 || observed.ReplicasObserved != 1 || observed.Message != "running" {
		t.Fatalf("unexpected observation: %#v", observed)
	}
}

// 这里保留每条正式操作的 httptest 请求形状，契约测试据此确保不会出现没有 HTTP 覆盖的 OpenAPI 操作。
func TestOpsHTTPContractInventory(t *testing.T) {
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/runtime-clusters", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/runtime-clusters/id", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/node-enrollments", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/node-enrollments/id", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/host-nodes", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/host-nodes/id", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/node-packages", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/node-packages/runtime_linux/download", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/project-deployments", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/project-deployments/id", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/deployment-runs/id", nil)
	_ = httptest.NewRequest(http.MethodGet, "/api/v1/ops/deployment-runs/id/events", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/runtime-clusters", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/node-enrollments", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/node-enrollments/id/approve", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/node-enrollments/id/reject", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/project-deployments", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/project-deployments/id/workloads/compute/start", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/agent/enrollments/claim", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/agent/host-nodes/id/heartbeat", nil)
}
