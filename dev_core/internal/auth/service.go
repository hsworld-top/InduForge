package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html"
	"math/big"
	"strings"
	"time"
)

type ServiceConfig struct {
	AppName    string
	CaptchaTTL time.Duration
}

type Service struct {
	repository  Repository
	cache       CaptchaStore
	revocations RevocationStore
	tokens      *TokenManager
	config      ServiceConfig
	assets      AssetSigner
}

type AssetSigner interface {
	PresignGet(context.Context, string, time.Duration) (string, error)
}

type LoginInput struct {
	Username    string
	Password    string
	TenantCode  string
	CaptchaKey  string
	CaptchaCode string
	LoginIP     string
}

func (s *Service) SetAssetSigner(signer AssetSigner) { s.assets = signer }

type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

type LoginResult struct {
	TokenPair
	User map[string]any `json:"user"`
}

func NewService(repository Repository, cache CaptchaStore, tokens *TokenManager, config ServiceConfig, revocations ...RevocationStore) *Service {
	if config.AppName == "" {
		config.AppName = "InduForge"
	}
	if config.CaptchaTTL <= 0 {
		config.CaptchaTTL = 5 * time.Minute
	}
	service := &Service{repository: repository, cache: cache, tokens: tokens, config: config}
	if len(revocations) > 0 {
		service.revocations = revocations[0]
	}
	return service
}

func (s *Service) Captcha(ctx context.Context) (map[string]any, error) {
	code, err := randomDigits(4)
	if err != nil {
		return nil, err
	}
	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("生成验证码标识失败: %w", err)
	}
	key := hex.EncodeToString(keyBytes)
	if err := s.cache.Put(ctx, key, code, s.config.CaptchaTTL); err != nil {
		return nil, fmt.Errorf("保存验证码失败: %w", err)
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="128" height="40"><rect width="100%%" height="100%%" fill="#f4f6f8"/><text x="64" y="27" text-anchor="middle" font-family="monospace" font-size="24" letter-spacing="6" fill="#1f2937">%s</text></svg>`, html.EscapeString(code))
	return map[string]any{
		"key":           key,
		"image":         "data:image/svg+xml;utf8," + svg,
		"expireSeconds": int64(s.config.CaptchaTTL.Seconds()),
	}, nil
}

func (s *Service) Config(ctx context.Context, tenantCode string) map[string]any {
	result := map[string]any{
		"name":        s.config.AppName,
		"title":       s.config.AppName,
		"appName":     s.config.AppName,
		"tenantCode":  tenantCode,
		"multiTenant": true,
	}
	repository, ok := s.repository.(interface {
		GetTenantBranding(context.Context, string) (TenantBranding, error)
	})
	if !ok || strings.TrimSpace(tenantCode) == "" {
		return result
	}
	branding, err := repository.GetTenantBranding(ctx, tenantCode)
	if err != nil {
		return result
	}
	result["tenantName"] = branding.Name
	result["logoUrl"] = s.signAsset(ctx, branding.LogoObjectKey)
	result["loginBackgroundUrl"] = s.signAsset(ctx, branding.LoginBackgroundObjectKey)
	return result
}

func (s *Service) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.TenantCode = strings.TrimSpace(input.TenantCode)
	if input.Username == "" || input.Password == "" {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err := s.verifyCaptcha(ctx, input.CaptchaKey, input.CaptchaCode); err != nil {
		return LoginResult{}, err
	}
	user, err := s.repository.FindLoginUser(ctx, input.Username, input.TenantCode)
	if err != nil {
		return LoginResult{}, err
	}
	if user.Status != "active" || user.TenantStatus != "active" {
		return LoginResult{}, ErrForbidden
	}
	if !VerifyPassword(input.Password, user.PasswordHash) {
		return LoginResult{}, ErrInvalidCredentials
	}
	pair, refresh, err := s.issueTokenPair(user)
	if err != nil {
		return LoginResult{}, err
	}
	if err := s.repository.CreateRefreshToken(ctx, refresh); err != nil {
		return LoginResult{}, fmt.Errorf("保存刷新令牌失败: %w", err)
	}
	if err := s.repository.UpdateLogin(ctx, user.ID, input.LoginIP); err != nil {
		return LoginResult{}, fmt.Errorf("更新登录信息失败: %w", err)
	}
	return LoginResult{TokenPair: pair, User: publicUser(user)}, nil
}

func (s *Service) Refresh(ctx context.Context, rawToken string) (TokenPair, error) {
	if strings.TrimSpace(rawToken) == "" {
		return TokenPair{}, ErrInvalidRefresh
	}
	oldToken, err := s.repository.GetRefreshToken(ctx, HashRefreshToken(rawToken))
	if err != nil || oldToken.Revoked || !time.Now().UTC().Before(oldToken.ExpiresAt) {
		return TokenPair{}, ErrInvalidRefresh
	}
	user, err := s.repository.GetUser(ctx, oldToken.UserID)
	if err != nil || user.Status != "active" || user.TenantStatus != "active" {
		return TokenPair{}, ErrInvalidRefresh
	}
	pair, replacement, err := s.issueTokenPair(user)
	if err != nil {
		return TokenPair{}, err
	}
	if err := s.repository.RotateRefreshToken(ctx, oldToken.Hash, replacement); err != nil {
		return TokenPair{}, err
	}
	return pair, nil
}

func (s *Service) Authenticate(ctx context.Context, accessToken string) (User, error) {
	claims, err := s.tokens.ParseAccessToken(accessToken)
	if err != nil {
		return User{}, ErrUnauthorized
	}
	if s.revocations != nil {
		revoked, err := s.revocations.IsRevoked(ctx, claims.ID)
		if err != nil || revoked {
			return User{}, ErrUnauthorized
		}
	}
	user, err := s.repository.GetUser(ctx, claims.UserID)
	if err != nil || user.Status != "active" || user.TenantStatus != "active" {
		return User{}, ErrUnauthorized
	}
	return user, nil
}

func (s *Service) RevokeAccessToken(ctx context.Context, accessToken string) error {
	if s.revocations == nil || strings.TrimSpace(accessToken) == "" {
		return nil
	}
	claims, err := s.tokens.ParseAccessToken(accessToken)
	if err != nil || claims.ExpiresAt == nil {
		return ErrUnauthorized
	}
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		return nil
	}
	return s.revocations.Revoke(ctx, claims.ID, ttl)
}

// AuthenticatePassword 只校验账号归属与密码，不签发 Token，供节点注册等一次性身份确认使用。
func (s *Service) AuthenticatePassword(ctx context.Context, username, password, tenantCode string) (User, error) {
	username = strings.TrimSpace(username)
	tenantCode = strings.TrimSpace(tenantCode)
	if username == "" || password == "" {
		return User{}, ErrInvalidCredentials
	}
	user, err := s.repository.FindLoginUser(ctx, username, tenantCode)
	if err != nil || user.Status != "active" || user.TenantStatus != "active" || !VerifyPassword(password, user.PasswordHash) {
		return User{}, ErrInvalidCredentials
	}
	return user, nil
}

func (s *Service) Logout(ctx context.Context, userID, refreshToken string) error {
	if refreshToken != "" {
		return s.repository.RevokeRefreshToken(ctx, HashRefreshToken(refreshToken))
	}
	return s.RevokeUserRefreshTokens(ctx, userID)
}

func (s *Service) RevokeUserRefreshTokens(ctx context.Context, userID string) error {
	return s.repository.RevokeUserRefreshTokens(ctx, userID)
}

func (s *Service) ChangePassword(ctx context.Context, user User, oldPassword, newPassword string) error {
	if !VerifyPassword(oldPassword, user.PasswordHash) {
		return ErrInvalidCredentials
	}
	if len(newPassword) < 8 {
		return fmt.Errorf("%w: 新密码至少 8 位", ErrInvalidCredentials)
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repository.UpdatePassword(ctx, user.ID, hash); err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}
	return s.repository.RevokeUserRefreshTokens(ctx, user.ID)
}

func (s *Service) issueTokenPair(user User) (TokenPair, RefreshToken, error) {
	access, expiresAt, err := s.tokens.IssueAccessToken(user)
	if err != nil {
		return TokenPair{}, RefreshToken{}, err
	}
	rawRefresh, refreshHash, refreshExpiresAt, err := s.tokens.NewRefreshToken()
	if err != nil {
		return TokenPair{}, RefreshToken{}, err
	}
	return TokenPair{AccessToken: access, Token: access, RefreshToken: rawRefresh, ExpiresIn: int64(time.Until(expiresAt).Seconds())}, RefreshToken{TenantID: user.TenantID, UserID: user.ID, Hash: refreshHash, ExpiresAt: refreshExpiresAt}, nil
}

func (s *Service) verifyCaptcha(ctx context.Context, key, code string) error {
	if strings.TrimSpace(key) == "" || strings.TrimSpace(code) == "" {
		return ErrInvalidCaptcha
	}
	expected, err := s.cache.Take(ctx, strings.TrimSpace(key))
	if err != nil || !strings.EqualFold(expected, strings.TrimSpace(code)) {
		return ErrInvalidCaptcha
	}
	return nil
}

func publicUser(user User) map[string]any {
	return map[string]any{
		"id": user.ID, "tenantId": user.TenantID, "username": user.Username,
		"email": user.Email, "fullName": user.FullName, "role": user.Role,
		"tenant": map[string]any{"id": user.TenantID, "code": user.TenantCode, "name": user.TenantName, "logoUrl": user.LogoURL},
	}
}

func (s *Service) signAsset(ctx context.Context, objectKey string) string {
	if s.assets == nil || strings.TrimSpace(objectKey) == "" {
		return ""
	}
	url, err := s.assets.PresignGet(ctx, objectKey, time.Hour)
	if err != nil {
		return ""
	}
	return url
}

func randomDigits(length int) (string, error) {
	result := make([]byte, length)
	for index := range result {
		value, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("生成验证码失败: %w", err)
		}
		result[index] = byte('0' + value.Int64())
	}
	return string(result), nil
}
