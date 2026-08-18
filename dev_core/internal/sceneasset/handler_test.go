package sceneasset

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestSceneRoutesRequireAuthentication(t *testing.T) {
	router := chi.NewRouter()
	router.Route("/api/v1", NewHandler(nil, "").MountRoutes)

	assertUnauthorized(t, router, http.MethodGet, "/api/v1/scene-provider")
	assertUnauthorized(t, router, http.MethodPut, "/api/v1/scene-provider")
	assertUnauthorized(t, router, http.MethodGet, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scenes")
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scenes")
	assertUnauthorized(t, router, http.MethodGet, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scenes/main?kind=2d")
	assertUnauthorized(t, router, http.MethodPut, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scenes/main?kind=2d")
	assertUnauthorized(t, router, http.MethodDelete, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scenes/main?kind=2d")
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scenes/main/editor-session?kind=2d")
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scenes/main/commit?kind=2d")

	assertUnauthorized(t, router, http.MethodGet, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scene-assets")
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scene-assets/import")
	assertUnauthorized(t, router, http.MethodGet, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scene-assets/00000000-0000-0000-0000-000000000002")
	assertUnauthorized(t, router, http.MethodPut, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scene-assets/00000000-0000-0000-0000-000000000002")
	assertUnauthorized(t, router, http.MethodDelete, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scene-assets/00000000-0000-0000-0000-000000000002")
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scene-assets/00000000-0000-0000-0000-000000000002/replace")
	assertUnauthorized(t, router, http.MethodGet, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scene-assets/00000000-0000-0000-0000-000000000002/thumbnail")
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scene-assets/00000000-0000-0000-0000-000000000002/editor-session")

	assertUnauthorized(t, router, http.MethodGet, "/api/v1/scene-editor-sessions/00000000-0000-0000-0000-000000000003/files")
	assertUnauthorized(t, router, http.MethodGet, "/api/v1/scene-editor-sessions/00000000-0000-0000-0000-000000000003/files/content?path=displays%2Fmain.json")
	assertUnauthorized(t, router, http.MethodPut, "/api/v1/scene-editor-sessions/00000000-0000-0000-0000-000000000003/files/content?path=displays%2Fmain.json")
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/scene-editor-sessions/00000000-0000-0000-0000-000000000003/import")
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/scene-editor-sessions/00000000-0000-0000-0000-000000000003/export")
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/scene-editor-sessions/00000000-0000-0000-0000-000000000003/commit")
	assertUnauthorized(t, router, http.MethodGet, "/api/v1/scene-editor-sessions/00000000-0000-0000-0000-000000000003/datapoints")
	assertUnauthorized(t, router, http.MethodGet, "/api/v1/scene-editor-sessions/00000000-0000-0000-0000-000000000003/assets")
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/scene-editor-sessions/00000000-0000-0000-0000-000000000003/assets/import")
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/scene-editor-sessions/00000000-0000-0000-0000-000000000003/assets/actions")
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/scene-editor-sessions/00000000-0000-0000-0000-000000000003/assets/00000000-0000-0000-0000-000000000002/replace")
	assertUnauthorized(t, router, http.MethodDelete, "/api/v1/scene-editor-sessions/00000000-0000-0000-0000-000000000003/assets/00000000-0000-0000-0000-000000000002")
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/scene-editor-sessions/00000000-0000-0000-0000-000000000003/assets/00000000-0000-0000-0000-000000000002/editor-session")
	assertUnauthorized(t, router, http.MethodGet, "/api/v1/scene-editor-sessions/00000000-0000-0000-0000-000000000003/dependencies")

	assertUnauthorized(t, router, http.MethodGet, "/api/v1/scene-asset-editor-sessions/00000000-0000-0000-0000-000000000004/files")
	assertUnauthorized(t, router, http.MethodGet, "/api/v1/scene-asset-editor-sessions/00000000-0000-0000-0000-000000000004/files/content?path=symbol.json")
	assertUnauthorized(t, router, http.MethodPut, "/api/v1/scene-asset-editor-sessions/00000000-0000-0000-0000-000000000004/files/content?path=symbol.json")
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/scene-asset-editor-sessions/00000000-0000-0000-0000-000000000004/commit")
}

func assertUnauthorized(t *testing.T, handler http.Handler, method, target string) {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(method, target, nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("%s %s 未拒绝匿名请求: got %d", method, target, response.Code)
	}
}
