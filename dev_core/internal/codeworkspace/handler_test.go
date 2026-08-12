package codeworkspace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/app"
	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/project"
	"github.com/indu-forge/dev_core/internal/testsupport"
)

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

func TestCodeWorkspaceHTTPRoutes(t *testing.T) {
	application, token := newCodeWorkspaceHTTPTestApp(t)

	assertCodeWorkspaceResponse(t, application, token, http.MethodGet, "/api/v1/projects/"+testProjectID+"/code-workspace")
	assertCodeWorkspaceResponse(t, application, token, http.MethodPost, "/api/v1/projects/"+testProjectID+"/code-workspace/start")
	assertCodeWorkspaceResponse(t, application, token, http.MethodPost, "/api/v1/projects/"+testProjectID+"/code-workspace/stop")
	assertCodeWorkspaceResponse(t, application, token, http.MethodPost, "/api/v1/projects/"+testProjectID+"/code-workspace/rebuild")
}

func newCodeWorkspaceHTTPTestApp(t *testing.T) (*app.App, string) {
	t.Helper()
	authService, token, actor := testsupport.NewAuth(t, "DEVELOPER")
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
}
