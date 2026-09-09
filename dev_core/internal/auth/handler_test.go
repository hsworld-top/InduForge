package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/app"
	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/controlplane"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	platformcache "github.com/indu-forge/dev_core/internal/platform/cache"
)

const (
	testUserID   = "11111111-1111-4111-8111-111111111111"
	testTenantID = "22222222-2222-4222-8222-222222222222"
)

func TestAuthHTTPInterfaces(t *testing.T) {
	server, repository := newAuthServer(t)

	config := performJSON(t, server, http.MethodGet, "/api/v1/auth/config?tenantCode=default", nil, "")
	assertSuccess(t, config)
	if multiTenant, ok := config.Data["multiTenant"].(bool); !ok || multiTenant {
		t.Fatalf("单租户配置应关闭多租户登录: %#v", config.Data["multiTenant"])
	}

	login, loginResponse := performJSONResponse(t, server, http.MethodPost, "/api/v1/auth/login", loginBody("admin123"), "", nil)
	assertSuccess(t, login)
	if _, exists := login.Data["accessToken"]; exists {
		t.Fatal("登录响应不应暴露访问令牌")
	}
	accessCookie := responseCookie(t, loginResponse, "if_access")
	refreshCookie := responseCookie(t, loginResponse, "if_refresh")
	if !accessCookie.HttpOnly || !refreshCookie.HttpOnly {
		t.Fatal("会话 Cookie 必须为 HttpOnly")
	}

	currentUser, _ := performJSONResponse(t, server, http.MethodGet, "/api/v1/auth/me", nil, "", []*http.Cookie{accessCookie})
	assertSuccess(t, currentUser)
	if got := nestedString(t, currentUser, "data", "username"); got != "admin" {
		t.Fatalf("当前用户错误: got %q", got)
	}

	refresh, refreshResponse := performJSONResponse(t, server, http.MethodPost, "/api/v1/auth/refresh", nil, "", []*http.Cookie{refreshCookie})
	assertSuccess(t, refresh)
	refreshedAccessCookie := responseCookie(t, refreshResponse, "if_access")
	refreshedRefreshCookie := responseCookie(t, refreshResponse, "if_refresh")
	if refreshedAccessCookie.Value == accessCookie.Value || refreshedRefreshCookie.Value == refreshCookie.Value {
		t.Fatal("刷新后应轮换会话 Cookie")
	}

	changePassword, _ := performJSONResponse(t, server, http.MethodPut, "/api/v1/auth/password", map[string]any{
		"oldPassword": "admin123", "newPassword": "new-admin-123",
	}, "", []*http.Cookie{refreshedAccessCookie})
	assertSuccess(t, changePassword)
	if !auth.VerifyPassword("new-admin-123", repository.user.PasswordHash) {
		t.Fatal("密码未更新为 Argon2id 哈希")
	}

	_, freshLogin := performJSONResponse(t, server, http.MethodPost, "/api/v1/auth/login", loginBody("new-admin-123"), "", nil)
	refreshedAccessCookie = responseCookie(t, freshLogin, "if_access")
	refreshedRefreshCookie = responseCookie(t, freshLogin, "if_refresh")
	logout, logoutResponse := performJSONResponse(t, server, http.MethodPost, "/api/v1/auth/logout", nil, "", []*http.Cookie{refreshedAccessCookie, refreshedRefreshCookie})
	assertSuccess(t, logout)
	if responseCookie(t, logoutResponse, "if_access").MaxAge >= 0 || responseCookie(t, logoutResponse, "if_refresh").MaxAge >= 0 {
		t.Fatal("登出后必须清除会话 Cookie")
	}
}

func TestLoginRejectsInvalidPassword(t *testing.T) {
	server, _ := newAuthServer(t)
	response := performJSON(t, server, http.MethodPost, "/api/v1/auth/login", loginBody("wrong"), "")
	if response.Code != platformapi.ErrorCodeInvalidCredentials {
		t.Fatalf("错误码不正确: got %d", response.Code)
	}
}

func TestTenantLoginRequiresExplicitTenantCode(t *testing.T) {
	server, _ := newAuthServer(t)
	body := loginBody("admin123")
	delete(body, "tenantCode")
	if got := performJSON(t, server, http.MethodPost, "/api/v1/auth/login", body, ""); got.Code != platformapi.ErrorCodeTenantRequired {
		t.Fatalf("必须明确租户: %#v", got)
	}
}

func TestAuthConfigEnablesTenantCodeForMultipleActiveTenants(t *testing.T) {
	server, repository := newAuthServer(t)
	repository.activeTenantCount = 2

	config := performJSON(t, server, http.MethodGet, "/api/v1/auth/config", nil, "")
	assertSuccess(t, config)
	if multiTenant, ok := config.Data["multiTenant"].(bool); !ok || !multiTenant {
		t.Fatalf("多租户配置应开启多租户登录: %#v", config.Data["multiTenant"])
	}
}

func TestLoginRequiresSliderAfterFirstCredentialFailure(t *testing.T) {
	server, repository := newAuthServer(t)
	firstFailure := performJSON(t, server, http.MethodPost, "/api/v1/auth/login", loginBody("wrong"), "")
	if firstFailure.Code != platformapi.ErrorCodeInvalidCredentials {
		t.Fatalf("首次凭据错误应直接返回凭据错误，实际 code=%d", firstFailure.Code)
	}

	missing := performJSON(t, server, http.MethodPost, "/api/v1/auth/login", loginBody("admin123"), "")
	if missing.Code != platformapi.ErrorCodeInvalidCaptcha {
		t.Fatalf("触发后缺少滑块挑战应失败，实际 code=%d", missing.Code)
	}

	challenge := getSliderChallenge(t, server)
	body := loginBody("admin123")
	body["sliderChallengeId"] = nestedString(t, challenge, "data", "challengeId")
	body["sliderOffset"] = sliderEndOffset(t, repository, challenge)
	assertSuccess(t, performJSON(t, server, http.MethodPost, "/api/v1/auth/login", body, ""))
}

func TestSliderChallengeIsBoundToLoginContextAndConsumed(t *testing.T) {
	server, repository := newAuthServer(t)
	performJSON(t, server, http.MethodPost, "/api/v1/auth/login", loginBody("wrong"), "")
	challenge := getSliderChallenge(t, server)
	body := loginBody("admin123")
	body["sliderChallengeId"] = nestedString(t, challenge, "data", "challengeId")
	target := sliderEndOffset(t, repository, challenge)
	body["sliderOffset"] = target - 20
	invalid := performJSON(t, server, http.MethodPost, "/api/v1/auth/login", body, "")
	if invalid.Code != platformapi.ErrorCodeInvalidCaptcha {
		t.Fatalf("错误位置应被拒绝，实际 code=%d", invalid.Code)
	}

	body["sliderOffset"] = target
	reused := performJSON(t, server, http.MethodPost, "/api/v1/auth/login", body, "")
	if reused.Code != platformapi.ErrorCodeInvalidCaptcha {
		t.Fatalf("滑块挑战重复使用应失败，实际 code=%d", reused.Code)
	}
}

func loginBody(password string) map[string]any {
	return map[string]any{"username": "admin", "password": password, "tenantCode": "default"}
}

func getSliderChallenge(t *testing.T, server http.Handler) envelope {
	t.Helper()
	challenge := performJSON(t, server, http.MethodGet, "/api/v1/auth/captcha?username=admin&tenantCode=default", nil, "")
	assertSuccess(t, challenge)
	if nestedString(t, challenge, "data", "challengeId") == "" {
		t.Fatal("滑块挑战缺少标识")
	}
	return challenge
}

func sliderEndOffset(t *testing.T, repository *fakeRepository, challenge envelope) int {
	t.Helper()
	encoded, err := repository.cache.Get(context.Background(), "slider-challenge:"+nestedString(t, challenge, "data", "challengeId"))
	if err != nil {
		t.Fatal(err)
	}
	var secret struct {
		TargetX int `json:"targetX"`
	}
	if err := json.Unmarshal([]byte(encoded), &secret); err != nil {
		t.Fatal(err)
	}
	if _, exists := challenge.Data["targetX"]; exists {
		t.Fatal("响应不能泄漏目标坐标")
	}
	return secret.TargetX
}

func TestCurrentUserRequiresSession(t *testing.T) {
	server, _ := newAuthServer(t)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("HTTP 状态错误: got %d", response.Code)
	}
	var payload envelope
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}
	if payload.Code != platformapi.ErrorCodeTokenRequired {
		t.Fatalf("错误码不正确: got %d", payload.Code)
	}
}

type envelope struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data map[string]any `json:"data"`
}

func newAuthServer(t *testing.T, stores ...auth.AvatarStore) (http.Handler, *fakeRepository) {
	t.Helper()
	passwordHash, err := auth.HashPassword("admin123")
	if err != nil {
		t.Fatalf("生成测试密码失败: %v", err)
	}
	repository := &fakeRepository{
		user: auth.User{
			ID: testUserID, TenantID: testTenantID, Username: "admin", PasswordHash: passwordHash,
			Email: "admin@example.com", FullName: "管理员", Role: "SYSTEM_ADMIN", Status: "active",
			TenantCode: "default", TenantName: "默认租户", TenantInitialized: true, TenantStatus: "active",
		},
		tokens:            make(map[string]auth.RefreshToken),
		activeTenantCount: 1,
	}
	tokens, err := auth.NewTokenManager("test-jwt-secret-long-enough", "induforge", "induforge-api", time.Hour, 24*time.Hour)
	if err != nil {
		t.Fatalf("创建 TokenManager 失败: %v", err)
	}
	repository.cache = platformcache.NewMemory()
	service := auth.NewService(repository, repository.cache, tokens, auth.ServiceConfig{AppName: "InduForge"})
	if len(stores) > 0 {
		service.SetAvatarStore(stores[0])
	}
	rootHandler := controlplane.NewHandler(auth.NewHandler(service))
	application := app.New(app.Options{
		RequestID:   func() string { return "auth-test-request" },
		Middlewares: []func(http.Handler) http.Handler{auth.ResolveUser(service)},
		Mount: func(router chi.Router) {
			platformapi.HandlerFromMuxWithBaseURL(rootHandler, router, "/api/v1")
		},
	})
	return application.Handler(), repository
}

func performJSON(t *testing.T, handler http.Handler, method, path string, body any, accessToken string) envelope {
	t.Helper()
	result, _ := performJSONResponse(t, handler, method, path, body, accessToken, nil)
	return result
}

func performJSONResponse(t *testing.T, handler http.Handler, method, path string, body any, accessToken string, cookies []*http.Cookie) (envelope, *httptest.ResponseRecorder) {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("编码请求失败: %v", err)
		}
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		request.Header.Set("Authorization", "Bearer "+accessToken)
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK && response.Code != http.StatusBadRequest && response.Code != http.StatusUnauthorized && response.Code != http.StatusForbidden {
		t.Fatalf("%s %s HTTP 状态错误: got %d, body=%s", method, path, response.Code, response.Body.String())
	}
	var result envelope
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, response.Body.String())
	}
	return result, response
}

func responseCookie(t *testing.T, response *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("响应缺少 Cookie %q", name)
	return nil
}

func assertSuccess(t *testing.T, response envelope) {
	t.Helper()
	if response.Code != 0 {
		t.Fatalf("接口返回失败: code=%d msg=%s", response.Code, response.Msg)
	}
}

func nestedString(t *testing.T, response envelope, keys ...string) string {
	t.Helper()
	var current any = response.Data
	for _, key := range keys[1:] {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("响应字段 %v 不是对象", keys)
		}
		current = object[key]
	}
	value, ok := current.(string)
	if !ok || value == "" {
		t.Fatalf("响应字段 %v 不是非空字符串: %#v", keys, current)
	}
	return value
}

type fakeRepository struct {
	user              auth.User
	tokens            map[string]auth.RefreshToken
	activeTenantCount int64
	cache             *platformcache.Memory
}

func (r *fakeRepository) CountActiveTenants(context.Context) (int64, error) {
	return r.activeTenantCount, nil
}

func (r *fakeRepository) FindLoginUser(_ context.Context, username, tenantCode string, platform bool) (auth.User, error) {
	if platform != (r.user.Role == "SUPER_ADMIN") || username != r.user.Username || (tenantCode != "" && tenantCode != r.user.TenantCode) {
		return auth.User{}, auth.ErrInvalidCredentials
	}
	return r.user, nil
}

func (r *fakeRepository) GetUser(_ context.Context, userID string) (auth.User, error) {
	if userID != r.user.ID {
		return auth.User{}, auth.ErrNotFound
	}
	return r.user, nil
}

func (r *fakeRepository) UpdateLogin(context.Context, string, string) error { return nil }

func (r *fakeRepository) UpdatePassword(_ context.Context, userID, passwordHash string, credentialVersion int64) error {
	if userID != r.user.ID {
		return auth.ErrNotFound
	}
	if r.user.CredentialVersion != credentialVersion {
		return auth.ErrUnauthorized
	}
	r.user.PasswordHash = passwordHash
	r.user.CredentialVersion++
	r.user.MustChangePassword = false
	return nil
}

func (r *fakeRepository) CreateRefreshToken(_ context.Context, token auth.RefreshToken) error {
	r.tokens[token.Hash] = token
	return nil
}

func (r *fakeRepository) GetRefreshToken(_ context.Context, tokenHash string) (auth.RefreshToken, error) {
	token, exists := r.tokens[tokenHash]
	if !exists {
		return auth.RefreshToken{}, auth.ErrInvalidRefresh
	}
	return token, nil
}

func (r *fakeRepository) RotateRefreshToken(_ context.Context, oldTokenHash string, replacement auth.RefreshToken) error {
	oldToken, exists := r.tokens[oldTokenHash]
	if !exists || oldToken.Revoked {
		return auth.ErrInvalidRefresh
	}
	oldToken.Revoked = true
	r.tokens[oldTokenHash] = oldToken
	r.tokens[replacement.Hash] = replacement
	return nil
}

func (r *fakeRepository) RevokeRefreshToken(_ context.Context, tokenHash string) error {
	token, exists := r.tokens[tokenHash]
	if !exists {
		return nil
	}
	token.Revoked = true
	r.tokens[tokenHash] = token
	return nil
}

func (r *fakeRepository) RevokeUserRefreshTokens(_ context.Context, userID string) error {
	if userID != r.user.ID {
		return errors.New("用户不存在")
	}
	for hash, token := range r.tokens {
		if token.UserID == userID {
			token.Revoked = true
			r.tokens[hash] = token
		}
	}
	return nil
}

func TestForcedPasswordChangeRevokesAccessAndRefresh(t *testing.T) {
	server, repository := newAuthServer(t)
	repository.user.MustChangePassword = true
	login, res := performJSONResponse(t, server, http.MethodPost, "/api/v1/auth/login", loginBody("admin123"), "", nil)
	assertSuccess(t, login)
	if login.Data["user"].(map[string]any)["mustChangePassword"] != true {
		t.Fatal("登录缺少强制改密状态")
	}
	access, refresh := responseCookie(t, res, "if_access"), responseCookie(t, res, "if_refresh")
	denied, _ := performJSONResponse(t, server, http.MethodGet, "/api/v1/tenants", nil, "", []*http.Cookie{access})
	if denied.Code != platformapi.ErrorCodePermissionDenied {
		t.Fatalf("未改密不能访问管理: %#v", denied)
	}
	me, _ := performJSONResponse(t, server, http.MethodGet, "/api/v1/auth/me", nil, "", []*http.Cookie{access})
	assertSuccess(t, me)
	same, _ := performJSONResponse(t, server, http.MethodPut, "/api/v1/auth/password", map[string]any{"oldPassword": "admin123", "newPassword": "admin123"}, "", []*http.Cookie{access})
	if same.Code == 0 {
		t.Fatal("不能用初始密码解除强制改密")
	}
	changed, cleared := performJSONResponse(t, server, http.MethodPut, "/api/v1/auth/password", map[string]any{"oldPassword": "admin123", "newPassword": "12345678"}, "", []*http.Cookie{access})
	assertSuccess(t, changed)
	if responseCookie(t, cleared, "if_access").MaxAge != -1 || repository.user.MustChangePassword {
		t.Fatal("改密必须清会话和强制标志")
	}
	stale := performJSON(t, server, http.MethodGet, "/api/v1/auth/me", nil, access.Value)
	if stale.Code == 0 {
		t.Fatal("旧Bearer必须立即失效")
	}
	staleRefresh, _ := performJSONResponse(t, server, http.MethodPost, "/api/v1/auth/refresh", nil, "", []*http.Cookie{refresh})
	if staleRefresh.Code == 0 {
		t.Fatal("旧refresh必须失效")
	}
}
func TestRememberMeSurvivesRefresh(t *testing.T) {
	for _, remember := range []bool{false, true} {
		t.Run(fmt.Sprint(remember), func(t *testing.T) {
			server, _ := newAuthServer(t)
			body := loginBody("admin123")
			body["rememberMe"] = remember
			login, res := performJSONResponse(t, server, http.MethodPost, "/api/v1/auth/login", body, "", nil)
			assertSuccess(t, login)
			for _, name := range []string{"if_access", "if_refresh"} {
				if (responseCookie(t, res, name).MaxAge > 0) != remember {
					t.Fatal("Cookie持久性不匹配")
				}
			}
			refresh, next := performJSONResponse(t, server, http.MethodPost, "/api/v1/auth/refresh", nil, "", []*http.Cookie{responseCookie(t, res, "if_refresh")})
			assertSuccess(t, refresh)
			for _, name := range []string{"if_access", "if_refresh"} {
				if (responseCookie(t, next, name).MaxAge > 0) != remember {
					t.Fatal("刷新丢失remember选择")
				}
			}
		})
	}
}
func TestPlatformScopeAndUninitializedTenantIsolation(t *testing.T) {
	server, repository := newAuthServer(t)
	body := loginBody("admin123")
	body["platform"] = true
	delete(body, "tenantCode")
	delete(body, "tenantCode")
	if got := performJSON(t, server, http.MethodPost, "/api/v1/auth/login", body, ""); got.Code == 0 {
		t.Fatal("平台入口不能登录同名租户账号")
	}
	repository.user.Role = "SUPER_ADMIN"
	repository.user.TenantID = ""
	repository.user.TenantCode = ""
	repository.user.MustChangePassword = true
	server, repository = newAuthServer(t)
	repository.user.Role = "SUPER_ADMIN"
	repository.user.TenantID = ""
	repository.user.TenantCode = ""
	repository.user.MustChangePassword = true
	assertSuccess(t, performJSON(t, server, http.MethodPost, "/api/v1/auth/login", body, ""))
	if got := performJSON(t, server, http.MethodPost, "/api/v1/auth/login", loginBody("admin123"), ""); got.Code == 0 {
		t.Fatal("租户入口不能登录平台账号")
	}
	server, repository = newAuthServer(t)
	repository.user.TenantInitialized = false
	if got := performJSON(t, server, http.MethodPost, "/api/v1/auth/login", loginBody("admin123"), ""); got.Code == 0 {
		t.Fatal("待初始化租户不能登录")
	}
}
func TestPuzzleImagesAndExpiredChallenge(t *testing.T) {
	server, repository := newAuthServer(t)
	performJSON(t, server, http.MethodPost, "/api/v1/auth/login", loginBody("wrong"), "")
	challenge := getSliderChallenge(t, server)
	for _, name := range []string{"image", "thumb"} {
		raw, ok := challenge.Data[name].(string)
		if !ok || !strings.HasPrefix(raw, "data:image/") {
			t.Fatalf("缺少拼图%s", name)
		}
	}
	body := loginBody("admin123")
	body["sliderOffset"] = sliderEndOffset(t, repository, challenge)
	body["sliderChallengeId"] = nestedString(t, challenge, "data", "challengeId")
	key := "slider-challenge:" + body["sliderChallengeId"].(string)
	if err := repository.cache.Put(context.Background(), key, "{}", -time.Second); err != nil {
		t.Fatal(err)
	}
	if got := performJSON(t, server, http.MethodPost, "/api/v1/auth/login", body, ""); got.Code != platformapi.ErrorCodeInvalidCaptcha {
		t.Fatalf("过期挑战被接受: %#v", got)
	}
}

func (r *fakeRepository) ListLoginTenants(ctx context.Context, keyword, code string, page, limit int) ([]auth.TenantBranding, int64, error) {
	if code != "" && (code != "default" || !r.user.TenantInitialized || r.user.TenantStatus != "active") {
		return []auth.TenantBranding{}, 0, nil
	}
	return []auth.TenantBranding{{ID: testTenantID, Code: "default", Name: "InduFrame"}}, 1, nil
}
func TestPublicLoginTenantsExposeOnlyBranding(t *testing.T) {
	server, _ := newAuthServer(t)
	result := performJSON(t, server, http.MethodGet, "/api/v1/auth/tenants", nil, "")
	assertSuccess(t, result)
	list, ok := result.Data["list"].([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("list=%#v", result.Data)
	}
	item := list[0].(map[string]any)
	for key := range item {
		if key != "id" && key != "code" && key != "name" && key != "logoUrl" && key != "loginBackgroundUrl" {
			t.Fatalf("private field %s", key)
		}
	}
}

func TestPasswordChangeCannotOverwriteConcurrentReset(t *testing.T) {
	_, repo := newAuthServer(t)
	stale := repo.user
	resetHash, _ := auth.HashPassword("reset-password")
	repo.user.PasswordHash = resetHash
	repo.user.CredentialVersion++
	repo.user.MustChangePassword = true
	service := auth.NewService(repo, repo.cache, nil, auth.ServiceConfig{})
	if err := service.ChangePassword(context.Background(), stale, "admin123", "replacement-password"); !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("stale change err=%v", err)
	}
	if !repo.user.MustChangePassword || !auth.VerifyPassword("reset-password", repo.user.PasswordHash) {
		t.Fatal("stale request overwrote reset")
	}
}
func TestPasswordMinimumCountsUnicodeCharacters(t *testing.T) {
	_, repo := newAuthServer(t)
	service := auth.NewService(repo, repo.cache, nil, auth.ServiceConfig{})
	if err := service.ChangePassword(context.Background(), repo.user, "admin123", "密码短短"); err == nil {
		t.Fatal("four multibyte characters must not count as eight")
	}
	if err := service.ChangePassword(context.Background(), repo.user, "admin123", "密码长度恰好八字"); err != nil {
		t.Fatal(err)
	}
}

func TestPublicLoginTenantExactCode(t *testing.T) {
	server, repo := newAuthServer(t)
	for _, tc := range []struct {
		code        string
		initialized bool
		status      string
		count       int
	}{{"default", true, "active", 1}, {"def", true, "active", 0}, {"default", false, "active", 0}, {"default", true, "suspended", 0}} {
		repo.user.TenantInitialized, repo.user.TenantStatus = tc.initialized, tc.status
		result := performJSON(t, server, http.MethodGet, "/api/v1/auth/tenants?code="+tc.code+"&page=1&limit=20", nil, "")
		assertSuccess(t, result)
		if list := result.Data["list"].([]any); len(list) != tc.count {
			t.Fatalf("%+v list=%v", tc, list)
		}
		pagination := result.Data["pagination"].(map[string]any)
		if pagination["total"] != float64(tc.count) || pagination["limit"] != float64(1) {
			t.Fatalf("pagination=%v", pagination)
		}
	}
}

func (r *fakeRepository) UpdateProfile(_ context.Context, userID, fullName, email string) error {
	if userID != r.user.ID {
		return auth.ErrUnauthorized
	}
	r.user.FullName, r.user.Email = fullName, email
	return nil
}

func TestUpdateCurrentUserProfile(t *testing.T) {
	for _, platform := range []bool{false, true} {
		t.Run(fmt.Sprintf("platform=%v", platform), func(t *testing.T) {
			server, repository := newAuthServer(t)
			body := loginBody("admin123")
			if platform {
				repository.user.Role, repository.user.TenantID = "SUPER_ADMIN", ""
				body["platform"] = true
				delete(body, "tenantCode")
			}
			login, response := performJSONResponse(t, server, http.MethodPost, "/api/v1/auth/login", body, "", nil)
			assertSuccess(t, login)
			cookies := []*http.Cookie{responseCookie(t, response, "if_access")}
			original := repository.user
			updated, _ := performJSONResponse(t, server, http.MethodPut, "/api/v1/auth/me", map[string]any{"fullName": " 新姓名 ", "email": " contact@example.com "}, "", cookies)
			assertSuccess(t, updated)
			if updated.Data["fullName"] != "新姓名" || updated.Data["email"] != "contact@example.com" || repository.user.FullName != "新姓名" {
				t.Fatalf("未保存资料: %#v", updated)
			}
			if repository.user.ID != original.ID || repository.user.Role != original.Role || repository.user.TenantID != original.TenantID || repository.user.Username != original.Username {
				t.Fatal("不应修改身份字段")
			}
			current, _ := performJSONResponse(t, server, http.MethodGet, "/api/v1/auth/me", nil, "", cookies)
			assertSuccess(t, current)
			if current.Data["fullName"] != "新姓名" {
				t.Fatal("重新读取未返回新资料")
			}
			for _, invalid := range []map[string]any{
				{"fullName": "测试", "email": "invalid"}, {"fullName": "测试", "email": "Name <name@example.com>"},
				{"fullName": strings.Repeat("中", 101), "email": ""}, {"fullName": "测试", "email": strings.Repeat("a", 255) + "@example.com"},
				{"fullName": "测试", "email": "", "id": "another-user"}, {"fullName": "测试", "email": "", "tenantId": "another-tenant"},
				{"fullName": "测试", "email": "", "role": "SUPER_ADMIN"}, {"fullName": "测试", "email": "", "username": "other"},
				{"fullName": 42, "email": ""}, {"fullName": "测试"},
			} {
				rejected, _ := performJSONResponse(t, server, http.MethodPut, "/api/v1/auth/me", invalid, "", cookies)
				if rejected.Code != platformapi.ErrorCodeInvalidRequest {
					t.Fatalf("应拒绝非法资料: %#v => %#v", invalid, rejected)
				}
				if repository.user.FullName != "新姓名" || repository.user.Email != "contact@example.com" {
					t.Fatal("拒绝请求不应写入")
				}
			}
			cleared, _ := performJSONResponse(t, server, http.MethodPut, "/api/v1/auth/me", map[string]any{"fullName": nil, "email": ""}, "", cookies)
			assertSuccess(t, cleared)
			if repository.user.FullName != "" || repository.user.Email != "" {
				t.Fatal("空值应清除资料")
			}
		})
	}
	server, _ := newAuthServer(t)
	result := performJSON(t, server, http.MethodPut, "/api/v1/auth/me", map[string]any{"fullName": "测试", "email": ""}, "")
	if result.Code != platformapi.ErrorCodeTokenRequired {
		t.Fatalf("未认证应拒绝: %#v", result)
	}
}

func (r *fakeRepository) UpdateAvatar(_ context.Context, userID, objectKey string) (string, error) {
	if userID != r.user.ID {
		return "", auth.ErrUnauthorized
	}
	old := r.user.Avatar
	r.user.Avatar = objectKey
	return old, nil
}
