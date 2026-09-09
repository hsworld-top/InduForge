package runtimeaccess_test

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
	"github.com/indu-forge/dev_core/internal/runtimeaccess"
	"github.com/indu-forge/dev_core/internal/testsupport"
)

const (
	projectID = "11111111-1111-4111-8111-111111111111"
	roleID    = "22222222-2222-4222-8222-222222222222"
	userID    = "33333333-3333-4333-8333-333333333333"
)

func TestRuntimeAccessHTTPFlow(t *testing.T) {
	handler, token := newRuntimeAccessServer(t, "SYSTEM_ADMIN")

	createdRole := call(t, handler, http.MethodPost, "/api/v1/projects/"+projectID+"/runtime-roles", map[string]any{
		"code": "operator", "name": "操作员", "capabilities": []string{"point:get"},
	}, token)
	assertOK(t, createdRole)
	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/projects/"+projectID+"/runtime-roles", nil, token))
	assertOK(t, call(t, handler, http.MethodPut, "/api/v1/projects/"+projectID+"/runtime-roles/"+roleID, map[string]any{
		"name": "高级操作员", "capabilities": []string{"point:get", "point:set"},
	}, token))

	createdUser := call(t, handler, http.MethodPost, "/api/v1/projects/"+projectID+"/runtime-users", map[string]any{
		"username": "runtime-user", "initialPassword": "runtime123", "displayName": "运行用户", "roleIds": []string{roleID},
	}, token)
	assertOK(t, createdUser)
	assertOK(t, call(t, handler, http.MethodGet, "/api/v1/projects/"+projectID+"/runtime-users", nil, token))
	assertOK(t, call(t, handler, http.MethodPatch, "/api/v1/projects/"+projectID+"/runtime-users/"+userID+"/status", map[string]any{"status": "disabled"}, token))
	assertOK(t, call(t, handler, http.MethodPut, "/api/v1/projects/"+projectID+"/runtime-users/"+userID+"/roles", map[string]any{"roleIds": []string{roleID}}, token))
	assertOK(t, call(t, handler, http.MethodPost, "/api/v1/projects/"+projectID+"/runtime-users/"+userID+"/reset-password", map[string]any{"newPassword": "changed123"}, token))
	assertOK(t, call(t, handler, http.MethodDelete, "/api/v1/projects/"+projectID+"/runtime-users/"+userID, nil, token))
	assertOK(t, call(t, handler, http.MethodDelete, "/api/v1/projects/"+projectID+"/runtime-roles/"+roleID, nil, token))
}

func TestCreateRuntimeAccessRejectsForeignProject(t *testing.T) {
	handler, token := newRuntimeAccessServer(t, "SYSTEM_ADMIN")
	response := call(t, handler, http.MethodPost, "/api/v1/projects/44444444-4444-4444-8444-444444444444/runtime-roles", map[string]any{
		"code": "operator", "name": "操作员",
	}, token)
	if response.Code != platformapi.ErrorCodeNotFound {
		t.Fatalf("跨租户工程应被拒绝，实际 code=%d msg=%s", response.Code, response.Msg)
	}
}

func TestRuntimeAccessPermissionDenied(t *testing.T) {
	handler, token := newRuntimeAccessServer(t, "VIEWER")

	created := call(t, handler, http.MethodPost, "/api/v1/projects/"+projectID+"/runtime-roles", map[string]any{
		"code": "operator", "name": "操作员",
	}, token)
	if created.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatalf("VIEWER 不应创建运行角色，实际 code=%d msg=%s", created.Code, created.Msg)
	}

	reset := call(t, handler, http.MethodPost, "/api/v1/projects/"+projectID+"/runtime-users/"+userID+"/reset-password", map[string]any{
		"newPassword": "changed123",
	}, token)
	if reset.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatalf("VIEWER 不应重置运行用户密码，实际 code=%d msg=%s", reset.Code, reset.Msg)
	}
}

func newRuntimeAccessServer(t *testing.T, role string) (http.Handler, string) {
	t.Helper()
	authService, token, _ := testsupport.NewAuth(t, role)
	repository := &fakeRepository{roles: map[string]runtimeaccess.Role{}, users: map[string]runtimeaccess.RuntimeUser{}}
	root := controlplane.NewHandler(auth.NewHandler(authService))
	root.SetRuntimeAccessHandler(runtimeaccess.NewHandler(runtimeaccess.NewService(repository), authService))
	application := app.New(app.Options{
		RequestID: func() string { return "runtime-access-test" },
		Mount: func(router chi.Router) {
			platformapi.HandlerFromMuxWithBaseURL(root, router, "/api/v1")
		},
	})
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

type fakeRepository struct {
	roles map[string]runtimeaccess.Role
	users map[string]runtimeaccess.RuntimeUser
}

func (r *fakeRepository) GetProjectAccess(_ context.Context, tenantID, id string) (runtimeaccess.ProjectAccess, error) {
	if tenantID != testsupport.TenantID || id != projectID {
		return runtimeaccess.ProjectAccess{}, runtimeaccess.ErrProjectNotFound
	}
	return runtimeaccess.ProjectAccess{ID: id, TenantID: tenantID, CreatedBy: testsupport.UserID, Visibility: "private"}, nil
}

func (r *fakeRepository) ListRoles(context.Context, string, string) ([]runtimeaccess.Role, error) {
	items := make([]runtimeaccess.Role, 0, len(r.roles))
	for _, item := range r.roles {
		items = append(items, item)
	}
	return items, nil
}

func (r *fakeRepository) GetRole(_ context.Context, _, _, id string) (runtimeaccess.Role, error) {
	item, ok := r.roles[id]
	if !ok {
		return runtimeaccess.Role{}, runtimeaccess.ErrRoleNotFound
	}
	return item, nil
}

func (r *fakeRepository) CreateRole(_ context.Context, id, _ string, input runtimeaccess.RoleInput) (runtimeaccess.Role, error) {
	now := time.Now()
	item := runtimeaccess.Role{ID: roleID, ProjectID: id, Code: input.Code, Name: input.Name, Description: input.Description, Status: input.Status, Capabilities: input.Capabilities, CreatedAt: now, UpdatedAt: now}
	r.roles[item.ID] = item
	return item, nil
}

func (r *fakeRepository) UpdateRole(_ context.Context, _, id, _ string, input runtimeaccess.RoleInput) (runtimeaccess.Role, error) {
	item, ok := r.roles[id]
	if !ok {
		return runtimeaccess.Role{}, runtimeaccess.ErrRoleNotFound
	}
	item.Code = input.Code
	item.Name = input.Name
	item.Description = input.Description
	item.Status = input.Status
	item.Capabilities = input.Capabilities
	item.UpdatedAt = time.Now()
	r.roles[id] = item
	return item, nil
}

func (r *fakeRepository) DeleteRole(_ context.Context, _, _, id string) error {
	delete(r.roles, id)
	return nil
}

func (r *fakeRepository) ListUsers(context.Context, string, string) ([]runtimeaccess.RuntimeUser, error) {
	items := make([]runtimeaccess.RuntimeUser, 0, len(r.users))
	for _, item := range r.users {
		items = append(items, item)
	}
	return items, nil
}

func (r *fakeRepository) GetUser(_ context.Context, _, _, id string) (runtimeaccess.RuntimeUser, error) {
	item, ok := r.users[id]
	if !ok {
		return runtimeaccess.RuntimeUser{}, runtimeaccess.ErrUserNotFound
	}
	return item, nil
}

func (r *fakeRepository) CreateUser(_ context.Context, id, _ string, input runtimeaccess.UserInput, _ string) (runtimeaccess.RuntimeUser, error) {
	now := time.Now()
	item := runtimeaccess.RuntimeUser{ID: userID, ProjectID: id, Username: input.Username, DisplayName: input.DisplayName, Email: input.Email, Status: input.Status, CreatedAt: now, UpdatedAt: now}
	r.users[item.ID] = item
	if err := r.ReplaceUserRoles(context.Background(), testsupport.TenantID, id, item.ID, input.RoleIDs); err != nil {
		return runtimeaccess.RuntimeUser{}, err
	}
	return r.users[item.ID], nil
}

func (r *fakeRepository) UpdateUserStatus(_ context.Context, _, _, id, status, _ string) (runtimeaccess.RuntimeUser, error) {
	item, ok := r.users[id]
	if !ok {
		return runtimeaccess.RuntimeUser{}, runtimeaccess.ErrUserNotFound
	}
	item.Status = status
	item.UpdatedAt = time.Now()
	r.users[id] = item
	return item, nil
}

func (r *fakeRepository) DeleteUser(_ context.Context, _, _, id string) error {
	delete(r.users, id)
	return nil
}

func (r *fakeRepository) ReplaceUserRoles(_ context.Context, _, _, id string, roleIDs []string) error {
	item, ok := r.users[id]
	if !ok {
		return runtimeaccess.ErrUserNotFound
	}
	item.Roles = make([]runtimeaccess.Role, 0, len(roleIDs))
	for _, id := range roleIDs {
		role, ok := r.roles[id]
		if !ok {
			return runtimeaccess.ErrRoleNotFound
		}
		item.Roles = append(item.Roles, role)
	}
	r.users[item.ID] = item
	return nil
}

func (r *fakeRepository) UpdateUserPassword(_ context.Context, _, _, id, _ string, _ string) error {
	item, ok := r.users[id]
	if !ok {
		return runtimeaccess.ErrUserNotFound
	}
	now := time.Now()
	item.PasswordChangedAt = &now
	item.UpdatedAt = now
	r.users[id] = item
	return nil
}

func (r *fakeRepository) PageRoles(ctx context.Context, tenant, project string, q runtimeaccess.ListQuery) ([]runtimeaccess.Role, int64, error) {
	items, err := r.ListRoles(ctx, tenant, project)
	return items, int64(len(items)), err
}
func (r *fakeRepository) PageUsers(ctx context.Context, tenant, project string, q runtimeaccess.ListQuery) ([]runtimeaccess.RuntimeUser, int64, error) {
	items, err := r.ListUsers(ctx, tenant, project)
	return items, int64(len(items)), err
}

func TestBuiltinAdminCannotBeDisabled(t *testing.T) {
	repo := &fakeRepository{roles: map[string]runtimeaccess.Role{}, users: map[string]runtimeaccess.RuntimeUser{userID: {ID: userID, IsBuiltinAdmin: true, Status: "active"}}}
	service := runtimeaccess.NewService(repo)
	_, err := service.UpdateUserStatus(context.Background(), auth.User{ID: testsupport.UserID, TenantID: testsupport.TenantID, Role: "SYSTEM_ADMIN"}, projectID, userID, "disabled")
	if err == nil {
		t.Fatal("builtin admin was disabled")
	}
	if repo.users[userID].Status != "active" {
		t.Fatal("builtin admin state changed")
	}
}
