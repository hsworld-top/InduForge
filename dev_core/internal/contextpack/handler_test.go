package contextpack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/app"
	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/testsupport"
)

type fakeContextSynchronizer struct{}

func (fakeContextSynchronizer) Sync(_ context.Context, actor auth.User, projectID, authorization string) (map[string]any, error) {
	return map[string]any{"projectId": projectID, "actorId": actor.ID, "hasAuthorization": authorization != ""}, nil
}

type contextPackAPIHandler struct {
	platformapi.Unimplemented
	context *Handler
}

func (h contextPackAPIHandler) RefreshWorkspaceContext(w http.ResponseWriter, r *http.Request, projectID string) {
	h.context.RefreshWorkspaceContext(w, r, projectID)
}

func TestRefreshWorkspaceContextHTTP(t *testing.T) {
	authService, token, _ := testsupport.NewAuth(t, "DEVELOPER")
	handler := contextPackAPIHandler{context: NewHandler(fakeContextSynchronizer{})}
	application := app.New(app.Options{
		RequestID:   func() string { return "context-pack-test" },
		Middlewares: []func(http.Handler) http.Handler{auth.ResolveUser(authService)},
		Mount: func(router chi.Router) {
			platformapi.HandlerFromMuxWithBaseURL(handler, router, "/api/v1")
		},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects/project-1/workspace-context/refresh", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	application.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP 状态错误: %d %s", response.Code, response.Body.String())
	}
	var envelope platformapi.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || envelope.Code != 0 {
		t.Fatalf("业务响应失败: err=%v body=%s", err, response.Body.String())
	}
}
