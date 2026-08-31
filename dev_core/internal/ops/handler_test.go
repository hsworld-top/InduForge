package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/app"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"net/http"
	"net/http/httptest"
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

func TestNodeProjectConflictUsesAlreadyExistsEnvelope(t *testing.T) {
	h := &Handler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/ops/project-deployments", nil)
	h.err(w, r, ErrNodeProjectConflict)

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

func TestPageReadsExactProjectDeploymentFilter(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/ops/project-deployments?page=2&pageSize=10&search=line&projectId=project-1", nil)
	filter := page(r)
	if filter.Page != 2 || filter.PageSize != 10 || filter.Search != "line" || filter.ProjectID != "project-1" {
		t.Fatalf("unexpected page filter: %#v", filter)
	}
}

type agentCommandRepository struct{ Repository }

func (r *agentCommandRepository) AgentCommands(context.Context, string, string) ([]AgentCommand, error) {
	return []AgentCommand{{NodeID: "node-1", ServiceID: "service-1", ServiceType: ServiceProjectEntry}}, nil
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
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/project-deployments/id/services/project_entry/start", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/agent/enrollments/claim", nil)
	_ = httptest.NewRequest(http.MethodPost, "/api/v1/ops/agent/nodes/id/heartbeat", nil)
}
