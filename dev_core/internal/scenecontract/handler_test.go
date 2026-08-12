package scenecontract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/app"
	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/project"
	"github.com/indu-forge/dev_core/internal/testsupport"
)

type sceneContractAPIHandler struct {
	platformapi.Unimplemented
	scenes *Handler
}

func (h sceneContractAPIHandler) ListSceneContracts(w http.ResponseWriter, r *http.Request, projectID string) {
	h.scenes.ListSceneContracts(w, r, projectID)
}
func (h sceneContractAPIHandler) GetSceneContract(w http.ResponseWriter, r *http.Request, projectID string, kind platformapi.GetSceneContractParamsKind, sceneID string) {
	h.scenes.GetSceneContract(w, r, projectID, string(kind), sceneID)
}
func (h sceneContractAPIHandler) PutSceneContract(w http.ResponseWriter, r *http.Request, projectID string, kind platformapi.PutSceneContractParamsKind, sceneID string) {
	h.scenes.PutSceneContract(w, r, projectID, string(kind), sceneID)
}
func (h sceneContractAPIHandler) DeleteSceneContract(w http.ResponseWriter, r *http.Request, projectID string, kind platformapi.DeleteSceneContractParamsKind, sceneID string) {
	h.scenes.DeleteSceneContract(w, r, projectID, string(kind), sceneID)
}

func TestSceneContractHTTP(t *testing.T) {
	workspace := t.TempDir()
	authService, token, actor := testsupport.NewAuth(t, "DEVELOPER")
	projects := fakeProjects{item: project.Project{ID: "project-1", TenantID: actor.TenantID, CreatedBy: actor.ID, Visibility: "private", WorkspacePath: workspace}}
	handler := sceneContractAPIHandler{scenes: NewHandler(NewService(projects), authService)}
	application := app.New(app.Options{
		RequestID:   func() string { return "scene-contract-test" },
		Middlewares: []func(http.Handler) http.Handler{auth.ResolveUser(authService)},
		Mount: func(router chi.Router) {
			platformapi.HandlerFromMuxWithBaseURL(handler, router, "/api/v1")
		},
	})

	assertSceneContractResponse(t, application.Handler(), token, http.MethodPut, "/api/v1/projects/project-1/scene-contracts/2d/line-overview", `{"name":"产线总览","embedMode":"both"}`)
	assertSceneContractResponse(t, application.Handler(), token, http.MethodGet, "/api/v1/projects/project-1/scene-contracts", "")
	assertSceneContractResponse(t, application.Handler(), token, http.MethodGet, "/api/v1/projects/project-1/scene-contracts/2d/line-overview", "")
	assertSceneContractResponse(t, application.Handler(), token, http.MethodDelete, "/api/v1/projects/project-1/scene-contracts/2d/line-overview", "")
}

func assertSceneContractResponse(t *testing.T, handler http.Handler, token, method, target, body string) {
	t.Helper()
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	request.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("%s %s returned HTTP %d: %s", method, target, response.Code, response.Body.String())
	}
	var envelope platformapi.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || envelope.Code != 0 {
		t.Fatalf("%s %s returned invalid envelope: err=%v body=%s", method, target, err, response.Body.String())
	}
}
