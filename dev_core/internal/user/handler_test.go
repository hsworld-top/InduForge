package user_test

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
	"github.com/indu-forge/dev_core/internal/testsupport"
	"github.com/indu-forge/dev_core/internal/user"
)

const managedUserID = "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"

func TestUserHTTPInterfaces(t *testing.T) {
	handler, token, revoker := newUserServer(t, "SYSTEM_ADMIN")
	assertOK(t, request(t, handler, http.MethodGet, "/api/v1/users", nil, token))
	assertOK(t, request(t, handler, http.MethodPost, "/api/v1/users", map[string]any{"username": "operator", "password": "operator123", "role": "OPS_ADMIN"}, token))
	assertOK(t, request(t, handler, http.MethodPut, "/api/v1/users/"+managedUserID, map[string]any{"fullName": "操作员", "status": "active"}, token))
	assertOK(t, request(t, handler, http.MethodPut, "/api/v1/users/"+managedUserID+"/password", map[string]any{"newPassword": "new-operator-123"}, token))
	if revoker.userID != managedUserID {
		t.Fatalf("修改密码后未撤销目标用户 Refresh Token，实际 userID=%q", revoker.userID)
	}
	assertOK(t, request(t, handler, http.MethodDelete, "/api/v1/users/"+managedUserID, nil, token))
}

func TestDeleteUserRejectsSelf(t *testing.T) {
	handler, token, _ := newUserServer(t, "SYSTEM_ADMIN")
	response := request(t, handler, http.MethodDelete, "/api/v1/users/"+testsupport.UserID, nil, token)
	if response.Code != platformapi.ErrorCodeInvalidRequest {
		t.Fatalf("错误码不正确: %d", response.Code)
	}
}

func TestUserRoleAssignmentBoundaries(t *testing.T) {
	for _, role := range []string{"PROJECT_ADMIN", "SYSTEM_ADMIN", "SUPER_ADMIN"} {
		handler, token, _ := newUserServer(t, "USER_ADMIN")
		response := request(t, handler, http.MethodPost, "/api/v1/users", map[string]any{"username": "target", "password": "target123", "role": role}, token)
		if response.Code != platformapi.ErrorCodePermissionDenied {
			t.Fatalf("USER_ADMIN 不应赋予 %s，实际 code=%d msg=%s", role, response.Code, response.Msg)
		}
	}

	systemHandler, systemToken, _ := newUserServer(t, "SYSTEM_ADMIN")
	created := request(t, systemHandler, http.MethodPost, "/api/v1/users", map[string]any{"username": "target", "password": "target123", "role": "SUPER_ADMIN"}, systemToken)
	if created.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatalf("SYSTEM_ADMIN 不应创建 SUPER_ADMIN，实际 code=%d msg=%s", created.Code, created.Msg)
	}
	updated := request(t, systemHandler, http.MethodPut, "/api/v1/users/"+managedUserID, map[string]any{"role": "SUPER_ADMIN"}, systemToken)
	if updated.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatalf("SYSTEM_ADMIN 不应赋予 SUPER_ADMIN，实际 code=%d msg=%s", updated.Code, updated.Msg)
	}

	superHandler, superToken, _ := newUserServer(t, "SUPER_ADMIN")
	if result := request(t, superHandler, http.MethodPost, "/api/v1/users", map[string]any{"username": "peer-super", "password": "target123", "role": "SUPER_ADMIN"}, superToken); result.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatal("平台管理员不能通过租户用户接口创建账号")
	}
}

func TestUserRejectsSelfRoleAndStatusChanges(t *testing.T) {
	handler, token, _ := newUserServer(t, "SYSTEM_ADMIN")
	for _, body := range []map[string]any{{"role": "VIEWER"}, {"status": "disabled"}} {
		response := request(t, handler, http.MethodPut, "/api/v1/users/"+testsupport.UserID, body, token)
		if response.Code != platformapi.ErrorCodeInvalidRequest {
			t.Fatalf("用户不应修改自己的角色或状态，实际 code=%d msg=%s", response.Code, response.Msg)
		}
	}
}

func newUserServer(t *testing.T, role string) (http.Handler, string, *fakeRevoker) {
	t.Helper()
	authService, token, actor := testsupport.NewAuth(t, role)
	now := time.Now()
	repository := &fakeRepository{items: map[string]user.User{
		actor.ID:      {ID: actor.ID, TenantID: actor.TenantID, Username: actor.Username, Role: actor.Role, Status: "active", CreatedAt: now, UpdatedAt: now},
		managedUserID: {ID: managedUserID, TenantID: actor.TenantID, Username: "managed", Role: "OPS_ADMIN", Status: "active", CreatedAt: now, UpdatedAt: now},
	}, hashes: map[string]string{}}
	revoker := &fakeRevoker{}
	root := controlplane.NewHandler(auth.NewHandler(authService))
	root.SetUserHandler(user.NewHandler(user.NewService(repository, revoker), authService))
	application := app.New(app.Options{RequestID: func() string { return "user-test" }, Mount: func(router chi.Router) { platformapi.HandlerFromMuxWithBaseURL(root, router, "/api/v1") }})
	return application.Handler(), token, revoker
}

type fakeRevoker struct{ userID string }

func (r *fakeRevoker) RevokeUserRefreshTokens(_ context.Context, userID string) error {
	r.userID = userID
	return nil
}

type responseEnvelope struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data map[string]any `json:"data"`
}

func request(t *testing.T, handler http.Handler, method, path string, body any, token string) responseEnvelope {
	t.Helper()
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s %s status=%d body=%s", method, path, rec.Code, rec.Body.String())
	}
	var result responseEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}
func assertOK(t *testing.T, response responseEnvelope) {
	t.Helper()
	if response.Code != 0 {
		t.Fatalf("接口失败: code=%d msg=%s", response.Code, response.Msg)
	}
}

type fakeRepository struct {
	items  map[string]user.User
	hashes map[string]string
}

func (r *fakeRepository) List(context.Context, string, user.ListFilter) ([]user.User, int64, error) {
	items := make([]user.User, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	return items, int64(len(items)), nil
}
func (r *fakeRepository) Get(_ context.Context, tenantID, userID string) (user.User, string, error) {
	item, ok := r.items[userID]
	if !ok || item.TenantID != tenantID {
		return user.User{}, "", user.ErrNotFound
	}
	return item, r.hashes[userID], nil
}
func (r *fakeRepository) Create(_ context.Context, item user.User, hash string) (user.User, error) {
	item.ID = managedUserID
	item.CreatedAt = time.Now()
	item.UpdatedAt = item.CreatedAt
	r.items[item.ID] = item
	r.hashes[item.ID] = hash
	return item, nil
}
func (r *fakeRepository) Update(_ context.Context, item user.User) (user.User, error) {
	item.UpdatedAt = time.Now()
	r.items[item.ID] = item
	return item, nil
}
func (r *fakeRepository) UpdatePassword(_ context.Context, tenantID, userID, hash string) error {
	item, ok := r.items[userID]
	if !ok || item.TenantID != tenantID {
		return user.ErrNotFound
	}
	r.hashes[userID] = hash
	return nil
}
func (r *fakeRepository) Delete(_ context.Context, tenantID, userID string) error {
	item, ok := r.items[userID]
	if !ok || item.TenantID != tenantID {
		return user.ErrNotFound
	}
	delete(r.items, userID)
	return nil
}

func TestUserAdvancedProfileRoundTrip(t *testing.T) {
	handler, token, _ := newUserServer(t, "SYSTEM_ADMIN")
	for _, role := range []string{"SYSTEM_ADMIN", "PROJECT_ADMIN", "OPS_ADMIN"} {
		result := request(t, handler, http.MethodPost, "/api/v1/users", map[string]any{"username": "engineer", "password": "example123", "role": role}, token)
		assertOK(t, result)
		if result.Data["role"] != role {
			t.Fatalf("role: %+v", result.Data)
		}
	}
	profile := map[string]any{"fullName": "张工", "email": "engineer@example.com", "phone": "13800000000", "gender": "female", "attributes": map[string]string{"部门": "工程部", "工号": "A001"}}
	result := request(t, handler, http.MethodPut, "/api/v1/users/"+managedUserID, profile, token)
	assertOK(t, result)
	if result.Data["gender"] != "female" || result.Data["attributes"].(map[string]any)["部门"] != "工程部" {
		t.Fatalf("profile: %+v", result.Data)
	}
	// 编辑部分资料不应丢失其他高级字段。
	result = request(t, handler, http.MethodPut, "/api/v1/users/"+managedUserID, map[string]any{"fullName": "李工"}, token)
	assertOK(t, result)
	if result.Data["attributes"].(map[string]any)["工号"] != "A001" {
		t.Fatal("attributes lost")
	}
	result = request(t, handler, http.MethodPut, "/api/v1/users/"+managedUserID, map[string]any{"gender": "", "attributes": map[string]string{}, "email": ""}, token)
	assertOK(t, result)
	if len(result.Data["attributes"].(map[string]any)) != 0 || result.Data["email"] != "" {
		t.Fatalf("not cleared: %+v", result.Data)
	}
}

func TestUserValidationAndManagementBoundary(t *testing.T) {
	handler, token, _ := newUserServer(t, "SYSTEM_ADMIN")
	for _, field := range []map[string]any{
		{"role": ""}, {"password": " "}, {"email": "bad"}, {"gender": "invalid"},
		{"attributes": map[string]string{"role": "SYSTEM_ADMIN"}},
		{"attributes": map[string]string{"Dept": "a", "dept": "b"}},
		{"attributes": map[string]any{"部门": 123}}, {"preferences": map[string]any{"theme": "dark"}},
	} {
		body := map[string]any{"username": "engineer", "password": "example123", "role": "PROJECT_ADMIN"}
		for key, value := range field {
			body[key] = value
		}
		result := request(t, handler, http.MethodPost, "/api/v1/users", body, token)
		if result.Code != platformapi.ErrorCodeInvalidRequest {
			t.Fatalf("invalid field %v: %+v", field, result)
		}
	}
	for _, role := range []string{"USER_ADMIN", "DEVELOPER", "OPERATOR", "VIEWER", "SUPER_ADMIN"} {
		result := request(t, handler, http.MethodPost, "/api/v1/users", map[string]any{"username": "engineer", "password": "example123", "role": role}, token)
		if result.Code != platformapi.ErrorCodePermissionDenied {
			t.Fatalf("assign role %s: %+v", role, result)
		}
	}
	for _, role := range []string{"PROJECT_ADMIN", "OPS_ADMIN", "USER_ADMIN"} {
		other, otherToken, _ := newUserServer(t, role)
		result := request(t, other, http.MethodGet, "/api/v1/users", nil, otherToken)
		if result.Code != platformapi.ErrorCodePermissionDenied {
			t.Fatalf("manage users as %s: %+v", role, result)
		}
	}
}

func TestTemporaryPasswordAndUsernameRules(t *testing.T) {
	handler, token, _ := newUserServer(t, "SYSTEM_ADMIN")
	for _, password := range []string{"1", "abc", "简单密码!@#"} {
		assertOK(t, request(t, handler, http.MethodPost, "/api/v1/users", map[string]any{"username": "user.test-1", "password": password, "role": "PROJECT_ADMIN"}, token))
	}
	for _, name := range []string{"user/name", "user@name", "user<name", "user name"} {
		result := request(t, handler, http.MethodPost, "/api/v1/users", map[string]any{"username": name, "password": "123", "role": "PROJECT_ADMIN"}, token)
		if result.Code != platformapi.ErrorCodeInvalidRequest {
			t.Fatalf("username %s accepted: %+v", name, result)
		}
	}
}
