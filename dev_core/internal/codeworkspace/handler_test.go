package codeworkspace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/app"
	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/project"
	"github.com/indu-forge/dev_core/internal/testsupport"
)

func TestWriteErrorReturnsStructuredAuthoringEpochConflict(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects/p/code-workspace/start", nil)
	(&Handler{}).writeError(recorder, request, &project.AuthoringEpochConflict{ProjectID: "p", Current: "epoch-3"})
	if recorder.Code != http.StatusConflict || !strings.Contains(recorder.Body.String(), `"currentAuthoringEpoch":"epoch-3"`) || !strings.Contains(recorder.Body.String(), `"action":"reload"`) {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

type testAPIHandler struct {
	platformapi.Unimplemented
	workspace *Handler
}

func (h testAPIHandler) GetCodeWorkspace(w http.ResponseWriter, r *http.Request, projectID string) {
	h.workspace.GetCodeWorkspace(w, r, projectID)
}

func (h testAPIHandler) StartCodeWorkspace(w http.ResponseWriter, r *http.Request, projectID string) {
	h.workspace.StartCodeWorkspace(w, r, projectID)
}

func (h testAPIHandler) StopCodeWorkspace(w http.ResponseWriter, r *http.Request, projectID string) {
	h.workspace.StopCodeWorkspace(w, r, projectID)
}

func (h testAPIHandler) RebuildCodeWorkspace(w http.ResponseWriter, r *http.Request, projectID string) {
	h.workspace.RebuildCodeWorkspace(w, r, projectID)
}

func (h testAPIHandler) CreateCodeWorkspaceMCPToken(w http.ResponseWriter, r *http.Request, projectID string) {
	h.workspace.CreateCodeWorkspaceMCPToken(w, r, projectID)
}

func TestCodeWorkspaceHTTPRoutes(t *testing.T) {
	application, token := newCodeWorkspaceHTTPTestApp(t)

	assertCodeWorkspaceResponse(t, application, token, http.MethodGet, "/api/v1/projects/"+testProjectID+"/code-workspace")
	assertCodeWorkspaceResponse(t, application, token, http.MethodPost, "/api/v1/projects/"+testProjectID+"/code-workspace/start")
	assertCodeWorkspaceResponse(t, application, token, http.MethodPost, "/api/v1/projects/"+testProjectID+"/code-workspace/stop")
	assertCodeWorkspaceResponse(t, application, token, http.MethodPost, "/api/v1/projects/"+testProjectID+"/code-workspace/rebuild")
	assertMCPTokenResponse(t, application, token)
}

func newCodeWorkspaceHTTPTestApp(t *testing.T) (*app.App, string) {
	t.Helper()
	authService, token, actor := testsupport.NewAuth(t, "PROJECT_ADMIN")
	root := t.TempDir()
	item := project.Project{
		ID: testProjectID, TenantID: actor.TenantID, CreatedBy: actor.ID, Visibility: "private",
		WorkspacePath: filepath.Join(root, testProjectID, "workspace"),
	}
	engine := &fakeEngine{state: ContainerState{
		Name: containerName(testProjectID), Status: "running", Running: true, HostPort: "49152",
		Labels: map[string]string{"com.induforge.managed": "true", "com.induforge.project-id": testProjectID, "com.induforge.role": "code-workspace"},
	}}
	service, err := NewService(fakeProjects{item: item}, engine, Config{Image: "image", VolumeName: "induforge-control-workspaces"})
	if err != nil {
		t.Fatal(err)
	}
	handler := testAPIHandler{workspace: NewHandler(service, authService)}
	application := app.New(app.Options{
		RequestID:   func() string { return "code-workspace-test" },
		Middlewares: []func(http.Handler) http.Handler{auth.ResolveUser(authService)},
		Mount: func(router chi.Router) {
			platformapi.HandlerFromMuxWithBaseURL(handler, router, "/api/v1")
		},
	})
	return application, token
}

func assertCodeWorkspaceResponse(t *testing.T, application *app.App, token, method, path string) {
	t.Helper()
	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	application.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("代码工作区接口状态错误: method=%s path=%s status=%d body=%s", method, path, response.Code, response.Body.String())
	}
	var envelope platformapi.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || envelope.Code != 0 {
		t.Fatalf("代码工作区接口响应错误: method=%s path=%s envelope=%+v err=%v", method, path, envelope, err)
	}
	var payload struct {
		Data struct {
			OnlineUsers []OnlineUser `json:"onlineUsers"`
			Services    map[string]struct {
				URL      *string `json:"url"`
				HostPort *int    `json:"hostPort"`
			} `json:"services"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.OnlineUsers == nil {
		t.Fatal("onlineUsers 必须是数组")
	}
	if code := payload.Data.Services["code"]; code.HostPort == nil || *code.HostPort != 49152 || ((method == http.MethodGet || strings.HasSuffix(path, "/start")) && code.URL == nil) {
		t.Fatalf("code 服务契约错误: %+v", code)
	}
	for _, name := range []string{"ai", "preview", "previewControl"} {
		if _, ok := payload.Data.Services[name]; !ok {
			t.Fatalf("缺少 %s 服务", name)
		}
	}
}

func assertMCPTokenResponse(t *testing.T, application *app.App, token string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+testProjectID+"/code-workspace/mcp-token", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	application.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("MCP 令牌接口状态错误: %d %s", response.Code, response.Body.String())
	}
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			ProjectID string `json:"projectId"`
			Token     string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || envelope.Code != 0 || envelope.Data.ProjectID != testProjectID || envelope.Data.Token == "" {
		t.Fatalf("MCP 令牌接口响应错误: %s err=%v", response.Body.String(), err)
	}
}

func TestMCPTokenBindsUserProjectAndEpoch(t *testing.T) {
	h := &Handler{mcpTokens: map[string]mcpToken{}}
	actor := auth.User{ID: "u1"}
	token, _, err := h.IssueMCPToken(actor, "p1", "e1", time.Hour)
	if err != nil || token == "" {
		t.Fatalf("issue token: %v", err)
	}
	if !h.ValidateMCPToken(token, "u1", "p1", "e1") {
		t.Fatal("token should validate")
	}
	if h.ValidateMCPToken(token, "u2", "p1", "e1") || h.ValidateMCPToken(token, "u1", "p2", "e1") || h.ValidateMCPToken(token, "u1", "p1", "e2") {
		t.Fatal("token crossed binding")
	}
}
