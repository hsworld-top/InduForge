package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
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

	login := performJSON(t, server, http.MethodPost, "/api/v1/auth/login", loginBody(t, server, "admin123"), "")
	assertSuccess(t, login)
	accessToken := nestedString(t, login, "data", "accessToken")
	refreshToken := nestedString(t, login, "data", "refreshToken")

	currentUser := performJSON(t, server, http.MethodGet, "/api/v1/auth/me", nil, accessToken)
	assertSuccess(t, currentUser)
	if got := nestedString(t, currentUser, "data", "username"); got != "admin" {
		t.Fatalf("当前用户错误: got %q", got)
	}

	refresh := performJSON(t, server, http.MethodPost, "/api/v1/auth/refresh", map[string]any{"refreshToken": refreshToken}, "")
	assertSuccess(t, refresh)
	if nestedString(t, refresh, "data", "accessToken") == accessToken {
		t.Fatal("刷新后应签发新的访问令牌")
	}

	changePassword := performJSON(t, server, http.MethodPut, "/api/v1/auth/password", map[string]any{
		"oldPassword": "admin123", "newPassword": "new-admin-123",
	}, accessToken)
	assertSuccess(t, changePassword)
	if !auth.VerifyPassword("new-admin-123", repository.user.PasswordHash) {
		t.Fatal("密码未更新为 Argon2id 哈希")
	}

	logout := performJSON(t, server, http.MethodPost, "/api/v1/auth/logout", nil, accessToken)
	assertSuccess(t, logout)
}

func TestLoginRejectsInvalidPassword(t *testing.T) {
	server, _ := newAuthServer(t)
	response := performJSON(t, server, http.MethodPost, "/api/v1/auth/login", loginBody(t, server, "wrong"), "")
	if response.Code != platformapi.ErrorCodeInvalidCredentials {
		t.Fatalf("错误码不正确: got %d", response.Code)
	}
}

func TestLoginRequiresAndConsumesCaptcha(t *testing.T) {
	server, _ := newAuthServer(t)
	missing := performJSON(t, server, http.MethodPost, "/api/v1/auth/login", map[string]any{"username": "admin", "password": "admin123", "tenantCode": "default"}, "")
	if missing.Code != platformapi.ErrorCodeInvalidCaptcha {
		t.Fatalf("缺少验证码应失败，实际 code=%d", missing.Code)
	}
	body := loginBody(t, server, "admin123")
	assertSuccess(t, performJSON(t, server, http.MethodPost, "/api/v1/auth/login", body, ""))
	reused := performJSON(t, server, http.MethodPost, "/api/v1/auth/login", body, "")
	if reused.Code != platformapi.ErrorCodeInvalidCaptcha {
		t.Fatalf("验证码重复使用应失败，实际 code=%d", reused.Code)
	}
}

func loginBody(t *testing.T, server http.Handler, password string) map[string]any {
	t.Helper()
	captcha := performJSON(t, server, http.MethodGet, "/api/v1/auth/captcha", nil, "")
	assertSuccess(t, captcha)
	key := nestedString(t, captcha, "data", "key")
	image := nestedString(t, captcha, "data", "image")
	match := regexp.MustCompile(`>(\d{4})</text>`).FindStringSubmatch(image)
	if len(match) != 2 {
		t.Fatalf("无法从测试验证码中解析数字: %s", image)
	}
	return map[string]any{"username": "admin", "password": password, "tenantCode": "default", "captchaKey": key, "captchaCode": match[1]}
}

func TestCurrentUserRequiresBearerToken(t *testing.T) {
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
		tokens: make(map[string]auth.RefreshToken),
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
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("%s %s HTTP 状态错误: got %d, body=%s", method, path, response.Code, response.Body.String())
	}
	var result envelope
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, response.Body.String())
	}
	return result
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
	user   auth.User
	tokens map[string]auth.RefreshToken
}

func (r *fakeRepository) FindLoginUser(_ context.Context, username, tenantCode string) (auth.User, error) {
	if username != r.user.Username || tenantCode != r.user.TenantCode {
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
