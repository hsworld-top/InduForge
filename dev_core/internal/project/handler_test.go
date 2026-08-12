package project_test

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
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/project"
	"github.com/indu-forge/dev_core/internal/testsupport"
)

func TestProjectHTTPInterfaces(t *testing.T) {
	handler, token := newProjectServer(t)
	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/projects", nil, token))
	tag := call(t, handler, http.MethodPost, "/api/v1/projects/tags", map[string]any{"name": "关键", "description": "说明", "sortOrder": 1}, token)
	assertOK(t, tag)
	tagID := dataID(t, tag)
	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/projects/tags", nil, token))
	assertOK(t, call(t, handler, http.MethodPut, "/api/v1/projects/tags/"+tagID, map[string]any{"name": "关键更新"}, token))
	group := call(t, handler, http.MethodPost, "/api/v1/projects/groups", map[string]any{"name": "产线", "sortOrder": 1}, token)
	assertOK(t, group)
	groupID := dataID(t, group)
	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/projects/groups", nil, token))
	assertOK(t, call(t, handler, http.MethodPut, "/api/v1/projects/groups/"+groupID, map[string]any{"name": "产线更新"}, token))
	created := call(t, handler, http.MethodPost, "/api/v1/projects", map[string]any{"name": "演示工程", "code": "demo", "visibility": "private"}, token)
	assertOK(t, created)
	projectID := dataID(t, created)
	assertOK(t, call(t, handler, http.MethodPut, "/api/v1/projects/"+projectID+"/tags", map[string]any{"tagIds": []string{tagID}}, token))
	assertOK(t, call(t, handler, http.MethodPut, "/api/v1/projects/"+projectID+"/group", map[string]any{"groupId": groupID}, token))
	assertOK(t, call(t, handler, http.MethodPut, "/api/v1/projects/"+projectID, map[string]any{"name": "演示工程更新"}, token))
	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/projects/"+projectID+"/delete-impact", nil, token))
	assertOK(t, call(t, handler, http.MethodPost, "/api/v1/projects/"+projectID+"/operations/archive", nil, token))
	exportRequest := httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+projectID+"/export", nil)
	exportRequest.Header.Set("Authorization", "Bearer "+token)
	exportResponse := httptest.NewRecorder()
	handler.ServeHTTP(exportResponse, exportRequest)
	if exportResponse.Code != http.StatusOK || exportResponse.Header().Get("Content-Disposition") == "" {
		t.Fatalf("导出失败: %d %s", exportResponse.Code, exportResponse.Body.String())
	}
	var exportPayload map[string]any
	if err := json.Unmarshal(exportResponse.Body.Bytes(), &exportPayload); err != nil {
		t.Fatal(err)
	}
	assertOK(t, call(t, handler, http.MethodPost, "/api/v1/projects/import", map[string]any{"name": "导入工程", "payload": exportPayload}, token))
	assertOK(t, call(t, handler, http.MethodDelete, "/api/v1/projects/"+projectID, nil, token))
	assertOK(t, call(t, handler, http.MethodDelete, "/api/v1/projects/tags/"+tagID, nil, token))
	assertOK(t, call(t, handler, http.MethodDelete, "/api/v1/projects/groups/"+groupID, nil, token))
}

func newProjectServer(t *testing.T) (http.Handler, string) {
	t.Helper()
	authService, token, _ := testsupport.NewAuth(t, "SYSTEM_ADMIN")
	repository := &fakeRepository{projects: map[string]project.Project{}, tags: map[string]project.Tag{}, groups: map[string]project.Group{}}
	service := project.NewService(repository, &fakeWorkspace{}, "admin123")
	root := controlplane.NewHandler(auth.NewHandler(authService))
	root.SetProjectHandler(project.NewHandler(service, authService))
	application := app.New(app.Options{RequestID: func() string { return "project-test" }, Mount: func(router chi.Router) { platformapi.HandlerFromMuxWithBaseURL(root, router, "/api/v1") }})
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
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
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
func dataID(t *testing.T, response envelope) string {
	t.Helper()
	var data map[string]any
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatal(err)
	}
	value, _ := data["id"].(string)
	if value == "" {
		t.Fatal("响应缺少 id")
	}
	return value
}

type fakeWorkspace struct{ files map[string]map[string]string }

func (w *fakeWorkspace) Initialize(id string) (string, error) {
	if w.files == nil {
		w.files = map[string]map[string]string{}
	}
	w.files[id] = map[string]string{"package.json": "e30="}
	return id, nil
}
func (w *fakeWorkspace) Remove(path string) error                      { delete(w.files, path); return nil }
func (w *fakeWorkspace) Export(path string) (map[string]string, error) { return w.files[path], nil }
func (w *fakeWorkspace) Import(id string, files map[string]string) (string, error) {
	if w.files == nil {
		w.files = map[string]map[string]string{}
	}
	w.files[id] = files
	return id, nil
}

type fakeRepository struct {
	projects map[string]project.Project
	tags     map[string]project.Tag
	groups   map[string]project.Group
}

func (r *fakeRepository) List(_ context.Context, tenantID string, filter project.ListFilter) ([]project.Project, int64, error) {
	items := []project.Project{}
	for _, item := range r.projects {
		if item.TenantID != tenantID {
			continue
		}
		if filter.IsPlatformAdmin || item.CreatedBy == filter.ActorID || (filter.CanReadShared && item.Visibility == "internal") {
			items = append(items, item)
		}
	}
	return items, int64(len(items)), nil
}
func (r *fakeRepository) Get(_ context.Context, tenantID, id string) (project.Project, error) {
	item, ok := r.projects[id]
	if !ok || item.TenantID != tenantID {
		return project.Project{}, project.ErrNotFound
	}
	return item, nil
}
func (r *fakeRepository) Create(_ context.Context, item project.Project, actor auth.User, _ string) (project.Project, error) {
	item.CreatedBy = actor.ID
	item.CreatedByName = actor.Username
	item.CreatedAt = time.Now()
	item.UpdatedAt = item.CreatedAt
	r.projects[item.ID] = item
	return item, nil
}
func (r *fakeRepository) Update(_ context.Context, item project.Project, _ auth.User) (project.Project, error) {
	item.UpdatedAt = time.Now()
	r.projects[item.ID] = item
	return item, nil
}
func (r *fakeRepository) SetStatus(_ context.Context, _, id, status, _ string) (project.Project, error) {
	item := r.projects[id]
	item.Status = status
	r.projects[id] = item
	return item, nil
}
func (r *fakeRepository) Delete(_ context.Context, _, id, _ string) error {
	delete(r.projects, id)
	return nil
}
func (r *fakeRepository) DeleteImpact(context.Context, string, string) (project.DeleteImpact, error) {
	return project.DeleteImpact{}, nil
}
func (r *fakeRepository) ListTags(context.Context, string, string) ([]project.Tag, error) {
	items := []project.Tag{}
	for _, item := range r.tags {
		items = append(items, item)
	}
	return items, nil
}
func (r *fakeRepository) GetTag(_ context.Context, _, id string) (project.Tag, error) {
	item, ok := r.tags[id]
	if !ok {
		return project.Tag{}, project.ErrTagNotFound
	}
	return item, nil
}
func (r *fakeRepository) CreateTag(_ context.Context, tenantID, _ string, item project.Tag) (project.Tag, error) {
	item.ID = "11111111-aaaa-4aaa-8aaa-111111111111"
	item.TenantID = tenantID
	item.CreatedAt = time.Now()
	item.UpdatedAt = item.CreatedAt
	r.tags[item.ID] = item
	return item, nil
}
func (r *fakeRepository) UpdateTag(_ context.Context, _ string, item project.Tag) (project.Tag, error) {
	item.UpdatedAt = time.Now()
	r.tags[item.ID] = item
	return item, nil
}
func (r *fakeRepository) DeleteTag(_ context.Context, _, id string) error {
	delete(r.tags, id)
	return nil
}
func (r *fakeRepository) ReplaceTags(context.Context, string, string, []string) error { return nil }
func (r *fakeRepository) ListGroups(context.Context, string, string) ([]project.Group, error) {
	items := []project.Group{}
	for _, item := range r.groups {
		items = append(items, item)
	}
	return items, nil
}
func (r *fakeRepository) GetGroup(_ context.Context, _, id string) (project.Group, error) {
	item, ok := r.groups[id]
	if !ok {
		return project.Group{}, project.ErrGroupNotFound
	}
	return item, nil
}
func (r *fakeRepository) CreateGroup(_ context.Context, tenantID, _ string, item project.Group) (project.Group, error) {
	item.ID = "22222222-aaaa-4aaa-8aaa-222222222222"
	item.TenantID = tenantID
	item.CreatedAt = time.Now()
	item.UpdatedAt = item.CreatedAt
	r.groups[item.ID] = item
	return item, nil
}
func (r *fakeRepository) UpdateGroup(_ context.Context, _ string, item project.Group) (project.Group, error) {
	item.UpdatedAt = time.Now()
	r.groups[item.ID] = item
	return item, nil
}
func (r *fakeRepository) DeleteGroup(_ context.Context, _, id string) error {
	delete(r.groups, id)
	return nil
}
func (r *fakeRepository) SetGroup(context.Context, string, string, *string) error { return nil }
