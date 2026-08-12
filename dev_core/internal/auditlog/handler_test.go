package auditlog_test

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
	"github.com/indu-forge/dev_core/internal/auditlog"
	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/controlplane"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/testsupport"
)

const testLogID = "aaaaaaaa-1111-4111-8111-aaaaaaaaaaaa"

func TestAuditLogHTTPFlow(t *testing.T) {
	handler, token := newAuditLogServer(t, "OPS_ADMIN")

	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/logs?page=1&limit=20", token))
	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/logs/"+testLogID, token))
	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/logs/stats", token))
	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/logs/recent-activities?limit=5", token))
	adminHandler, adminToken := newAuditLogServer(t, "SYSTEM_ADMIN")
	assertOK(t, call(t, adminHandler, http.MethodDelete, "/api/v1/logs?beforeDate=2026-08-04%2000:00:00", adminToken))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/logs/export", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Content-Disposition") == "" || response.Body.Len() == 0 {
		t.Fatalf("日志导出失败: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAuditLogPermissionDenied(t *testing.T) {
	handler, token := newAuditLogServer(t, "USER_ADMIN")
	response := call(t, handler, http.MethodGet, "/api/v1/logs", token)
	if response.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatalf("USER_ADMIN 不应读取审计日志，实际 code=%d msg=%s", response.Code, response.Msg)
	}
}

func TestAuditLogDeleteRequiresDeleteCapability(t *testing.T) {
	handler, token := newAuditLogServer(t, "PROJECT_ADMIN")
	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/logs", token))
	response := call(t, handler, http.MethodDelete, "/api/v1/logs?beforeDate=2026-08-04%2000:00:00", token)
	if response.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatalf("PROJECT_ADMIN 不应清理审计日志，实际 code=%d msg=%s", response.Code, response.Msg)
	}
}

func newAuditLogServer(t *testing.T, role string) (http.Handler, string) {
	t.Helper()
	authService, token, _ := testsupport.NewAuth(t, role)
	now := time.Now()
	repository := &fakeRepository{items: []auditlog.Log{{ID: testLogID, TenantID: testsupport.TenantID, UserID: testsupport.UserID, Username: "tester", Level: "INFO", Action: "login", Resource: "auth", Message: "登录成功", Metadata: map[string]any{}, CreatedAt: now}}}
	root := controlplane.NewHandler(auth.NewHandler(authService))
	root.SetAuditLogHandler(auditlog.NewHandler(auditlog.NewService(repository), authService))
	application := app.New(app.Options{RequestID: func() string { return "audit-log-test" }, Mount: func(router chi.Router) {
		platformapi.HandlerFromMuxWithBaseURL(root, router, "/api/v1")
	}})
	return application.Handler(), token
}

type envelope struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func call(t *testing.T, handler http.Handler, method, path, token string) envelope {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewReader(nil))
	request.Header.Set("Authorization", "Bearer "+token)
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

func assertOK(t *testing.T, response envelope) {
	t.Helper()
	if response.Code != 0 {
		t.Fatalf("接口失败: %d %s", response.Code, response.Msg)
	}
}

type fakeRepository struct{ items []auditlog.Log }

func (r *fakeRepository) List(context.Context, string, auditlog.Filter) ([]auditlog.Log, int64, error) {
	return r.items, int64(len(r.items)), nil
}

func (r *fakeRepository) Get(_ context.Context, _, id string) (auditlog.Log, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return auditlog.Log{}, auditlog.ErrNotFound
}

func (r *fakeRepository) DeleteBefore(_ context.Context, _ string, before time.Time) (int64, error) {
	remaining := r.items[:0]
	var deleted int64
	for _, item := range r.items {
		if item.CreatedAt.Before(before) {
			deleted++
			continue
		}
		remaining = append(remaining, item)
	}
	r.items = remaining
	return deleted, nil
}

func (r *fakeRepository) Export(context.Context, string, auditlog.Filter) ([]auditlog.Log, error) {
	return r.items, nil
}

func (r *fakeRepository) Stats(context.Context, string, *time.Time, *time.Time) (auditlog.Stats, error) {
	return auditlog.Stats{LevelStats: map[string]int64{"INFO": 1}, TrendStats: []map[string]any{}, ActionStats: []map[string]any{}}, nil
}

func (r *fakeRepository) Recent(context.Context, string, int) ([]auditlog.Log, error) {
	return r.items, nil
}
