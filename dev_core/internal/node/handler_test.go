package node_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/app"
	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/controlplane"
	"github.com/indu-forge/dev_core/internal/node"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/testsupport"
)

const testNodeID = "55555555-5555-4555-8555-555555555555"

func TestNodeManagementHTTPFlow(t *testing.T) {
	handler, token := newNodeServer(t, "OPS_ADMIN")

	list := call(t, handler, http.MethodGet, "/api/v1/nodes?page=1&pageSize=20&approvalStatus=pending", nil, token)
	assertOK(t, list)
	assertOK(t, call(t, handler, http.MethodPut, "/api/v1/nodes/"+testNodeID+"/approve", nil, token))
	assertOK(t, call(t, handler, http.MethodPut, "/api/v1/nodes/"+testNodeID+"/reject", nil, token))
	assertOK(t, call(t, handler, http.MethodDelete, "/api/v1/nodes/"+testNodeID, nil, token))
}

func TestNodeAgentHTTPFlow(t *testing.T) {
	handler, _ := newNodeServer(t, "OPS_ADMIN")
	register := callPublic(t, handler, http.MethodPost, "/api/v1/node-register/register-with-auth", map[string]any{
		"username": "tester", "password": "admin123", "nodeId": "machine-code-not-uuid", "nodeName": "edge-node-01",
	})
	assertOK(t, register)
	var registerData struct {
		NodeID            string `json:"nodeId"`
		RegistrationToken string `json:"registrationToken"`
	}
	if err := json.Unmarshal(register.Data, &registerData); err != nil {
		t.Fatal(err)
	}
	assertOK(t, callPublic(t, handler, http.MethodGet, "/api/v1/node-register/"+registerData.NodeID+"/approval-status", nil))

	heartbeat := agentRequest(t, handler, "/api/v1/nodes/"+registerData.NodeID+"/heartbeat", registerData.RegistrationToken, map[string]any{"metrics": map[string]any{"cpu": 0.1}})
	if !bytes.Contains(heartbeat, []byte(`"success":true`)) {
		t.Fatalf("心跳响应缺少 success=true: %s", heartbeat)
	}
	agentRequest(t, handler, "/api/v1/nodes/"+registerData.NodeID+"/deployment-status", registerData.RegistrationToken, map[string]any{"deploymentId": testNodeID, "status": "running"})
	agentRequest(t, handler, "/api/v1/nodes/"+registerData.NodeID+"/offline", registerData.RegistrationToken, map[string]any{"reason": "shutdown"})
}

func TestNodePermissionDenied(t *testing.T) {
	developerHandler, developerToken := newNodeServer(t, "DEVELOPER")
	approved := call(t, developerHandler, http.MethodPut, "/api/v1/nodes/"+testNodeID+"/approve", nil, developerToken)
	if approved.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatalf("DEVELOPER 不应审批节点，实际 code=%d msg=%s", approved.Code, approved.Msg)
	}
	deleted := call(t, developerHandler, http.MethodDelete, "/api/v1/nodes/"+testNodeID, nil, developerToken)
	if deleted.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatalf("DEVELOPER 不应删除节点，实际 code=%d msg=%s", deleted.Code, deleted.Msg)
	}

	userAdminHandler, userAdminToken := newNodeServer(t, "USER_ADMIN")
	listed := call(t, userAdminHandler, http.MethodGet, "/api/v1/nodes", nil, userAdminToken)
	if listed.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatalf("USER_ADMIN 不应读取节点，实际 code=%d msg=%s", listed.Code, listed.Msg)
	}
}

func newNodeServer(t *testing.T, role string) (http.Handler, string) {
	t.Helper()
	authService, token, _ := testsupport.NewAuth(t, role)
	now := time.Now()
	repository := &fakeRepository{items: map[string]node.Node{
		testNodeID: {ID: testNodeID, TenantID: testsupport.TenantID, Name: "边缘节点", Status: "pending", ApprovalStatus: "pending", Metrics: map[string]any{}, Metadata: map[string]any{}, Deployments: []map[string]any{}, CreatedAt: now, UpdatedAt: now},
	}}
	nodeHandler := node.NewHandler(node.NewService(repository), authService)
	root := controlplane.NewHandler(auth.NewHandler(authService))
	root.SetNodeHandler(nodeHandler)
	application := app.New(app.Options{RequestID: func() string { return "node-test" }, Mount: func(router chi.Router) {
		nodeHandler.MountAgentRoutes(router)
		platformapi.HandlerFromMuxWithBaseURL(root, router, "/api/v1")
	}})
	return application.Handler(), token
}

type envelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func call(t *testing.T, handler http.Handler, method, path string, body any, token string) envelope {
	t.Helper()
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(raw))
	request.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("%s %s status=%d body=%s", method, path, response.Code, response.Body.String())
	}
	var result envelope
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func callPublic(t *testing.T, handler http.Handler, method, path string, body any) envelope {
	t.Helper()
	raw, _ := json.Marshal(body)
	request := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("%s %s status=%d body=%s", method, path, response.Code, response.Body.String())
	}
	var result envelope
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func agentRequest(t *testing.T, handler http.Handler, path, token string, body any) []byte {
	t.Helper()
	raw, _ := json.Marshal(body)
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Registration-Token", token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("POST %s status=%d body=%s", path, response.Code, response.Body.String())
	}
	return response.Body.Bytes()
}

func assertOK(t *testing.T, response envelope) {
	t.Helper()
	if response.Code != 0 {
		t.Fatalf("接口失败: %d %s", response.Code, response.Msg)
	}
}

type fakeRepository struct {
	items     map[string]node.Node
	tokenHash string
}

func (r *fakeRepository) List(_ context.Context, tenantID string, filter node.ListFilter) ([]node.Node, int64, error) {
	items := make([]node.Node, 0, len(r.items))
	for _, item := range r.items {
		if item.TenantID == tenantID && (filter.ApprovalStatus == "" || item.ApprovalStatus == filter.ApprovalStatus) {
			items = append(items, item)
		}
	}
	return items, int64(len(items)), nil
}

func (r *fakeRepository) Approve(_ context.Context, tenantID, id, _ string) (node.Node, error) {
	item, ok := r.items[id]
	if !ok || item.TenantID != tenantID {
		return node.Node{}, node.ErrNotFound
	}
	now := time.Now()
	item.Status = "offline"
	item.ApprovalStatus = "approved"
	item.ApprovedAt = &now
	item.UpdatedAt = now
	r.items[id] = item
	return item, nil
}

func (r *fakeRepository) Reject(_ context.Context, tenantID, id, _ string) (node.Node, error) {
	item, ok := r.items[id]
	if !ok || item.TenantID != tenantID {
		return node.Node{}, node.ErrNotFound
	}
	now := time.Now()
	item.Status = "rejected"
	item.ApprovalStatus = "rejected"
	item.RejectedAt = &now
	item.UpdatedAt = now
	r.items[id] = item
	return item, nil
}

func (r *fakeRepository) Delete(_ context.Context, tenantID, id string) error {
	item, ok := r.items[id]
	if !ok || item.TenantID != tenantID {
		return node.ErrNotFound
	}
	delete(r.items, id)
	return nil
}

func (r *fakeRepository) Register(_ context.Context, input node.RegisterInput) (node.Node, error) {
	now := time.Now()
	approval := "pending"
	if input.AutoApprove {
		approval = "approved"
	}
	item := node.Node{ID: input.NodeID, TenantID: input.Registrant.TenantID, Name: input.Name, Status: "offline", ApprovalStatus: approval, Metrics: map[string]any{}, Metadata: map[string]any{}, Deployments: []map[string]any{}, CreatedAt: now, UpdatedAt: now}
	r.items[item.ID] = item
	r.tokenHash = input.RegistrationHash
	return item, nil
}

func (r *fakeRepository) GetApprovalStatus(_ context.Context, id string) (node.ApprovalStatus, error) {
	item, ok := r.items[id]
	if !ok {
		return node.ApprovalStatus{}, node.ErrNotFound
	}
	return node.ApprovalStatus{NodeID: item.ID, NodeName: item.Name, Status: item.Status, ApprovalStatus: item.ApprovalStatus, UpdatedAt: item.UpdatedAt}, nil
}

func (r *fakeRepository) Heartbeat(_ context.Context, id, tokenHash string, input node.HeartbeatInput) (node.Node, []node.Command, error) {
	item, ok := r.items[id]
	if !ok || tokenHash != r.tokenHash {
		return node.Node{}, nil, node.ErrNotFound
	}
	item.Status = "online"
	item.Metrics = input.Metrics
	r.items[id] = item
	return item, []node.Command{{Type: "start", Payload: map[string]any{"projectId": testNodeID}}}, nil
}

func (r *fakeRepository) Offline(_ context.Context, id, tokenHash, _ string) (node.Node, error) {
	item, ok := r.items[id]
	if !ok || tokenHash != r.tokenHash {
		return node.Node{}, node.ErrNotFound
	}
	item.Status = "offline"
	r.items[id] = item
	return item, nil
}

func (r *fakeRepository) ReportDeployment(_ context.Context, id, tokenHash string, input node.DeploymentReport) (node.DeploymentUpdate, error) {
	if _, ok := r.items[id]; !ok || tokenHash != r.tokenHash {
		return node.DeploymentUpdate{}, node.ErrNotFound
	}
	return node.DeploymentUpdate{ID: input.DeploymentID, TenantID: testsupport.TenantID, NodeID: id, ProjectID: "33333333-3333-4333-8333-333333333333", Status: input.Status}, nil
}
