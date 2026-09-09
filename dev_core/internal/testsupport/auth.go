package testsupport

import (
	"context"
	"testing"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
	platformcache "github.com/indu-forge/dev_core/internal/platform/cache"
)

const (
	UserID   = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	TenantID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
)

func NewAuth(t *testing.T, role string) (*auth.Service, string, auth.User) {
	t.Helper()
	hash, err := auth.HashPassword("admin123")
	if err != nil {
		t.Fatalf("生成测试密码失败: %v", err)
	}
	user := auth.User{ID: UserID, TenantID: TenantID, Username: "tester", PasswordHash: hash, Role: role, Status: "active", TenantCode: "default", TenantName: "默认租户", TenantInitialized: true, TenantStatus: "active"}
	if role == "SUPER_ADMIN" {
		user.TenantID = ""
	}
	repository := &authRepository{user: user, tokens: make(map[string]auth.RefreshToken)}
	tokens, err := auth.NewTokenManager("test-jwt-secret-long-enough", "induforge", "induforge-api", time.Hour, 24*time.Hour)
	if err != nil {
		t.Fatalf("创建 TokenManager 失败: %v", err)
	}
	service := auth.NewService(repository, platformcache.NewMemory(), tokens, auth.ServiceConfig{AppName: "InduForge"})
	accessToken, _, err := tokens.IssueAccessToken(user)
	if err != nil {
		t.Fatalf("签发测试令牌失败: %v", err)
	}
	return service, accessToken, user
}

type authRepository struct {
	user   auth.User
	tokens map[string]auth.RefreshToken
}

func (r *authRepository) CountActiveTenants(context.Context) (int64, error) {
	return 1, nil
}

func (r *authRepository) FindLoginUser(context.Context, string, string, bool) (auth.User, error) {
	return r.user, nil
}
func (r *authRepository) GetUser(_ context.Context, userID string) (auth.User, error) {
	if userID != r.user.ID {
		return auth.User{}, auth.ErrNotFound
	}
	return r.user, nil
}
func (r *authRepository) UpdateLogin(context.Context, string, string) error           { return nil }
func (r *authRepository) UpdatePassword(context.Context, string, string, int64) error { return nil }
func (r *authRepository) CreateRefreshToken(_ context.Context, token auth.RefreshToken) error {
	r.tokens[token.Hash] = token
	return nil
}
func (r *authRepository) GetRefreshToken(_ context.Context, hash string) (auth.RefreshToken, error) {
	token, ok := r.tokens[hash]
	if !ok {
		return auth.RefreshToken{}, auth.ErrInvalidRefresh
	}
	return token, nil
}
func (r *authRepository) RotateRefreshToken(_ context.Context, oldHash string, replacement auth.RefreshToken) error {
	delete(r.tokens, oldHash)
	r.tokens[replacement.Hash] = replacement
	return nil
}
func (r *authRepository) RevokeRefreshToken(_ context.Context, hash string) error {
	delete(r.tokens, hash)
	return nil
}
func (r *authRepository) RevokeUserRefreshTokens(context.Context, string) error { return nil }

func (r *authRepository) UpdateProfile(_ context.Context, userID, fullName, email string) error {
	if userID != r.user.ID {
		return auth.ErrUnauthorized
	}
	r.user.FullName, r.user.Email = fullName, email
	return nil
}

func (r *authRepository) UpdateAvatar(_ context.Context, userID, objectKey string) (string, error) {
	if userID != r.user.ID {
		return "", auth.ErrUnauthorized
	}
	old := r.user.Avatar
	r.user.Avatar = objectKey
	return old, nil
}
