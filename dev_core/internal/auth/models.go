package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrTenantRequired     = errors.New("该用户名存在于多个租户，请填写租户编码")
	ErrUnauthorized       = errors.New("未登录或登录已失效")
	ErrForbidden          = errors.New("当前账号不可用")
	ErrInvalidRefresh     = errors.New("刷新令牌无效或已过期")
	ErrInvalidCaptcha     = errors.New("滑块验证错误或已过期")
	ErrPermissionDenied   = errors.New("无权执行当前操作")
	ErrNotFound           = errors.New("资源不存在")
)

type User struct {
	MustChangePassword       bool
	CredentialVersion        int64
	TenantInitialized        bool
	ID                       string
	TenantID                 string
	Username                 string
	PasswordHash             string
	Email                    string
	Avatar                   string
	FullName                 string
	Role                     string
	Status                   string
	TenantCode               string
	TenantName               string
	TenantStatus             string
	LogoObjectKey            string
	LoginBackgroundObjectKey string
	LogoURL                  string
}

type TenantBranding struct {
	CaptchaBackgrounds       []string
	ID                       string
	Code                     string
	Name                     string
	LogoObjectKey            string
	LoginBackgroundObjectKey string
}

type RefreshToken struct {
	RememberMe        bool
	CredentialVersion int64
	ID                string
	TenantID          string
	UserID            string
	Hash              string
	ExpiresAt         time.Time
	Revoked           bool
}

type Repository interface {
	CountActiveTenants(ctx context.Context) (int64, error)
	FindLoginUser(ctx context.Context, username, tenantCode string, platform bool) (User, error)
	GetUser(ctx context.Context, userID string) (User, error)
	UpdateProfile(ctx context.Context, userID, fullName, email string) error
	UpdateAvatar(ctx context.Context, userID, objectKey string) (string, error)
	UpdateLogin(ctx context.Context, userID, loginIP string) error
	UpdatePassword(ctx context.Context, userID, passwordHash string, credentialVersion int64) error
	CreateRefreshToken(ctx context.Context, token RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (RefreshToken, error)
	RotateRefreshToken(ctx context.Context, oldTokenHash string, replacement RefreshToken) error
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeUserRefreshTokens(ctx context.Context, userID string) error
}

type CaptchaStore interface {
	Put(ctx context.Context, key, value string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Take(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}

type RevocationStore interface {
	Revoke(ctx context.Context, tokenID string, ttl time.Duration) error
	IsRevoked(ctx context.Context, tokenID string) (bool, error)
}
