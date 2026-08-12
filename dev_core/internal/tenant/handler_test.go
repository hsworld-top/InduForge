package tenant_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/app"
	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/controlplane"
	"github.com/indu-forge/dev_core/internal/objectstore"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/tenant"
	"github.com/indu-forge/dev_core/internal/testsupport"
)

const secondTenantID = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"

func TestTenantHTTPInterfaces(t *testing.T) {
	handler, token := newTenantServer(t)

	assertOK(t, requestJSON(t, handler, http.MethodGet, "/api/v1/tenants", nil, token))
	assertOK(t, requestJSON(t, handler, http.MethodGet, "/api/v1/tenants/current", nil, token))
	assertOK(t, requestJSON(t, handler, http.MethodGet, "/api/v1/tenants/current/dashboard-notes", nil, token))

	createdNote := requestJSON(t, handler, http.MethodPost, "/api/v1/tenants/current/dashboard-notes", map[string]any{"content": "测试便签"}, token)
	assertOK(t, createdNote)
	noteID := dataString(t, createdNote, "id")
	assertOK(t, requestJSON(t, handler, http.MethodPut, "/api/v1/tenants/current/dashboard-notes/"+noteID, map[string]any{"content": "更新便签"}, token))
	assertOK(t, requestJSON(t, handler, http.MethodDelete, "/api/v1/tenants/current/dashboard-notes/"+noteID, nil, token))

	createdTenant := requestJSON(t, handler, http.MethodPost, "/api/v1/tenants", map[string]any{"name": "第二租户", "code": "second"}, token)
	assertOK(t, createdTenant)
	assertOK(t, requestJSON(t, handler, http.MethodGet, "/api/v1/tenants/"+secondTenantID, nil, token))
	assertOK(t, requestJSON(t, handler, http.MethodPut, "/api/v1/tenants/"+secondTenantID, map[string]any{"name": "第二租户更新"}, token))
	assertOK(t, requestJSON(t, handler, http.MethodPost, "/api/v1/tenants/"+secondTenantID+"/suspend", nil, token))
	assertOK(t, requestJSON(t, handler, http.MethodPost, "/api/v1/tenants/"+secondTenantID+"/activate", nil, token))
	assertOK(t, requestUpload(t, handler, http.MethodPost, "/api/v1/tenants/"+secondTenantID+"/upload?type=logo", token))
	assertOK(t, requestJSON(t, handler, http.MethodDelete, "/api/v1/tenants/"+secondTenantID, nil, token))
}

func newTenantServer(t *testing.T) (http.Handler, string) {
	t.Helper()
	authService, token, _ := testsupport.NewAuth(t, "SUPER_ADMIN")
	repository := &fakeRepository{tenants: map[string]tenant.Tenant{
		testsupport.TenantID: {ID: testsupport.TenantID, Name: "默认租户", Code: "default", Status: "active", MaxUsers: 100, MaxProjects: 50, Settings: map[string]any{}, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}}
	service := tenant.NewService(repository, fakeObjectStore{}, tenant.ServiceConfig{})
	root := controlplane.NewHandler(auth.NewHandler(authService), tenant.NewHandler(service, authService))
	application := app.New(app.Options{RequestID: func() string { return "tenant-test" }, Mount: func(router chi.Router) { platformapi.HandlerFromMuxWithBaseURL(root, router, "/api/v1") }})
	return application.Handler(), token
}

type responseEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func requestJSON(t *testing.T, handler http.Handler, method, path string, body any, token string) responseEnvelope {
	t.Helper()
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("%s %s: status=%d body=%s", method, path, response.Code, response.Body.String())
	}
	var result responseEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	return result
}

func requestUpload(t *testing.T, handler http.Handler, method, path, token string) responseEnvelope {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "logo.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("\x89PNG\r\n\x1a\n"))
	_ = writer.Close()
	request := httptest.NewRequest(method, path, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", response.Code, response.Body.String())
	}
	var result responseEnvelope
	_ = json.Unmarshal(response.Body.Bytes(), &result)
	return result
}

func assertOK(t *testing.T, response responseEnvelope) {
	t.Helper()
	if response.Code != 0 {
		t.Fatalf("接口失败: code=%d msg=%s", response.Code, response.Msg)
	}
}
func dataString(t *testing.T, response responseEnvelope, key string) string {
	t.Helper()
	var data map[string]any
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatal(err)
	}
	value, _ := data[key].(string)
	if value == "" {
		t.Fatalf("字段 %s 为空", key)
	}
	return value
}

type fakeRepository struct {
	tenants map[string]tenant.Tenant
	notes   map[string]tenant.Note
}

func (r *fakeRepository) List(context.Context, tenant.ListFilter) ([]tenant.Tenant, int64, error) {
	items := make([]tenant.Tenant, 0, len(r.tenants))
	for _, item := range r.tenants {
		items = append(items, item)
	}
	return items, int64(len(items)), nil
}
func (r *fakeRepository) Get(_ context.Context, identifier string) (tenant.Tenant, error) {
	for _, item := range r.tenants {
		if item.ID == identifier || item.Code == identifier {
			return item, nil
		}
	}
	return tenant.Tenant{}, tenant.ErrNotFound
}
func (r *fakeRepository) Create(_ context.Context, item tenant.Tenant, _, _ string) (tenant.Tenant, error) {
	item.ID = secondTenantID
	item.CreatedAt = time.Now()
	item.UpdatedAt = item.CreatedAt
	r.tenants[item.ID] = item
	return item, nil
}
func (r *fakeRepository) Update(_ context.Context, item tenant.Tenant) (tenant.Tenant, error) {
	item.UpdatedAt = time.Now()
	r.tenants[item.ID] = item
	return item, nil
}
func (r *fakeRepository) Delete(_ context.Context, id string) error {
	delete(r.tenants, id)
	return nil
}
func (r *fakeRepository) SetStatus(_ context.Context, id, status string) (tenant.Tenant, error) {
	item := r.tenants[id]
	item.Status = status
	r.tenants[id] = item
	return item, nil
}
func (r *fakeRepository) ListNotes(context.Context, string) ([]tenant.Note, error) {
	items := []tenant.Note{}
	for _, item := range r.notes {
		items = append(items, item)
	}
	return items, nil
}
func (r *fakeRepository) CreateNote(_ context.Context, tenantID, userID, content string) (tenant.Note, error) {
	if r.notes == nil {
		r.notes = map[string]tenant.Note{}
	}
	item := tenant.Note{ID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd", TenantID: tenantID, CreatedBy: userID, Content: content, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	r.notes[item.ID] = item
	return item, nil
}
func (r *fakeRepository) UpdateNote(_ context.Context, tenantID, userID, noteID, content string) (tenant.Note, error) {
	item := r.notes[noteID]
	item.Content = content
	item.UpdatedBy = userID
	item.UpdatedAt = time.Now()
	r.notes[noteID] = item
	return item, nil
}
func (r *fakeRepository) DeleteNote(_ context.Context, _, noteID string) error {
	delete(r.notes, noteID)
	return nil
}

type fakeObjectStore struct{}

func (fakeObjectStore) Put(_ context.Context, key string, _ io.Reader, size int64, contentType string) (objectstore.ObjectRef, error) {
	return objectstore.ObjectRef{Bucket: "design-assets", Key: key, Size: size, ContentType: contentType}, nil
}

func (fakeObjectStore) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	return "http://objects/" + strings.TrimLeft(key, "/") + "?signed=true", nil
}

func (fakeObjectStore) Delete(context.Context, string) error { return nil }
