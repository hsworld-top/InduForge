package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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

func TestSingleTenantLoginDoesNotRequireTenantCode(t *testing.T) {
	server, _ := newAuthServer(t)
	body := loginBody("admin123")
	delete(body, "tenantCode")
	assertSuccess(t, performJSON(t, server, http.MethodPost, "/api/v1/auth/login", body, ""))
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
	server, _ := newAuthServer(t)
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
	body["sliderOffset"] = sliderEndOffset(t, challenge)
	assertSuccess(t, performJSON(t, server, http.MethodPost, "/api/v1/auth/login", body, ""))
}

func TestSliderChallengeIsBoundToLoginContextAndConsumed(t *testing.T) {
	server, _ := newAuthServer(t)
	performJSON(t, server, http.MethodPost, "/api/v1/auth/login", loginBody("wrong"), "")
	challenge := getSliderChallenge(t, server)
	body := loginBody("admin123")
	body["sliderChallengeId"] = nestedString(t, challenge, "data", "challengeId")
	body["sliderOffset"] = sliderEndOffset(t, challenge) - 20
	invalid := performJSON(t, server, http.MethodPost, "/api/v1/auth/login", body, "")
	if invalid.Code != platformapi.ErrorCodeInvalidCaptcha {
		t.Fatalf("错误位置应被拒绝，实际 code=%d", invalid.Code)
	}

	body["sliderOffset"] = sliderEndOffset(t, challenge)
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

func sliderEndOffset(t *testing.T, challenge envelope) int {
	t.Helper()
	trackWidth, ok := challenge.Data["trackWidth"].(float64)
	if !ok {
		t.Fatalf("滑块挑战缺少轨道宽度: %#v", challenge.Data["trackWidth"])
	}
	thumbWidth, ok := challenge.Data["thumbWidth"].(float64)
	if !ok {
		t.Fatalf("滑块挑战缺少滑块宽度: %#v", challenge.Data["thumbWidth"])
	}
	return int(trackWidth - thumbWidth)
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

func newAuthServer(t *testing.T) (http.Handler, *fakeRepository) {
	t.Helper()
	passwordHash, err := auth.HashPassword("admin123")
	if err != nil {
		t.Fatalf("生成测试密码失败: %v", err)
	}
	repository := &fakeRepository{
		user: auth.User{
			ID: testUserID, TenantID: testTenantID, Username: "admin", PasswordHash: passwordHash,
			Email: "admin@example.com", FullName: "管理员", Role: "SYSTEM_ADMIN", Status: "active",
			TenantCode: "default", TenantName: "默认租户", TenantStatus: "active",
		},
		tokens:            make(map[string]auth.RefreshToken),
		activeTenantCount: 1,
	}
	tokens, err := auth.NewTokenManager("test-jwt-secret-long-enough", "induforge", "induforge-api", time.Hour, 24*time.Hour)
	if err != nil {
		t.Fatalf("创建 TokenManager 失败: %v", err)
	}
	service := auth.NewService(repository, platformcache.NewMemory(), tokens, auth.ServiceConfig{AppName: "InduForge"})
	rootHandler := controlplane.NewHandler(auth.NewHandler(service))
	application := app.New(app.Options{
		RequestID: func() string { return "auth-test-request" },
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
	if response.Code != http.StatusOK {
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
}

func (r *fakeRepository) CountActiveTenants(context.Context) (int64, error) {
	return r.activeTenantCount, nil
}

func (r *fakeRepository) FindLoginUser(_ context.Context, username, tenantCode string) (auth.User, error) {
	if username != r.user.Username || (tenantCode != "" && tenantCode != r.user.TenantCode) {
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

func (r *fakeRepository) UpdatePassword(_ context.Context, userID, passwordHash string) error {
	if userID != r.user.ID {
		return auth.ErrNotFound
	}
	r.user.PasswordHash = passwordHash
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
