package sceneasset

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	platformcache "github.com/indu-forge/dev_core/internal/platform/cache"
	"github.com/indu-forge/dev_core/internal/project"
)

func TestWriteErrorReturnsStructuredAuthoringEpochConflict(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/scenes/s/files", nil)
	(&Handler{}).writeError(recorder, request, &project.AuthoringEpochConflict{ProjectID: "p", Current: "epoch-7"})
	if recorder.Code != http.StatusConflict || !strings.Contains(recorder.Body.String(), `"currentAuthoringEpoch":"epoch-7"`) || !strings.Contains(recorder.Body.String(), `"action":"reload"`) {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

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
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/projects/00000000-0000-0000-0000-000000000001/scenes/main/viewer-session?kind=2d")
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
	assertUnauthorized(t, router, http.MethodPost, "/api/v1/scene-editor-sessions/00000000-0000-0000-0000-000000000003/assets/from-selection")
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

func TestViewerRoutesRejectExpiredCapabilitySession(t *testing.T) {
	service := NewService(nil, nil, nil, missingSessionStore{}, nil)
	router := chi.NewRouter()
	router.Route("/api/v1", NewHandler(service, "").MountRoutes)

	assertViewerSessionExpired(t, router, http.MethodGet, "/api/v1/scene-viewer-sessions/00000000-0000-0000-0000-000000000005")
	assertViewerSessionExpired(t, router, http.MethodPost, "/api/v1/scene-viewer-sessions/00000000-0000-0000-0000-000000000005/heartbeat")
	assertViewerSessionExpired(t, router, http.MethodGet, "/api/v1/scene-viewer-sessions/00000000-0000-0000-0000-000000000005/files/content?path=displays%2Fmain.json")
}

func assertViewerSessionExpired(t *testing.T, handler http.Handler, method, target string) {
	t.Helper()
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(method, target, nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("%s %s 未拒绝过期 Viewer 会话: got %d", method, target, response.Code)
	}
	var envelope struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("解析 Viewer 会话错误响应失败: %v", err)
	}
	if envelope.Code != 10003 {
		t.Fatalf("Viewer 会话过期错误码错误: got %d, want 10003", envelope.Code)
	}
}

type missingSessionStore struct{}

func (missingSessionStore) PutSceneSession(context.Context, string, string, time.Duration) error {
	return nil
}

func (missingSessionStore) GetSceneSession(context.Context, string) (string, error) {
	return "", platformcache.ErrMiss
}

func (missingSessionStore) DeleteSceneSession(context.Context, string) error {
	return nil
}
