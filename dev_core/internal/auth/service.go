package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ServiceConfig struct {
	AppName           string
	CaptchaTTL        time.Duration
	LoginChallengeTTL time.Duration
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
	Username          string
	Password          string
	TenantCode        string
	SliderChallengeID string
	SliderOffset      int
	LoginIP           string
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
	if config.LoginChallengeTTL <= 0 {
		config.LoginChallengeTTL = 10 * time.Minute
	}
	service := &Service{repository: repository, cache: cache, tokens: tokens, config: config}
	if len(revocations) > 0 {
		service.revocations = revocations[0]
	}
	return service
}

const (
	sliderTrackWidth = 280
	sliderThumbWidth = 44
	sliderTolerance  = 5
)

func (s *Service) Captcha(ctx context.Context, username, tenantCode, loginIP string) (map[string]any, error) {
	contextKey := loginChallengeContextKey(username, tenantCode, loginIP)
	if _, err := s.cache.Get(ctx, contextKey); err != nil {
		return nil, ErrInvalidCaptcha
	}
	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("生成滑块挑战标识失败: %w", err)
	}
	challengeID := hex.EncodeToString(keyBytes)
	if err := s.cache.Put(ctx, sliderChallengeCacheKey(challengeID), contextKey, s.config.CaptchaTTL); err != nil {
		return nil, fmt.Errorf("保存滑块挑战失败: %w", err)
	}
	return map[string]any{
		"challengeId":   challengeID,
		"trackWidth":    sliderTrackWidth,
		"thumbWidth":    sliderThumbWidth,
		"expireSeconds": int64(s.config.CaptchaTTL.Seconds()),
	}, nil
}

func (s *Service) Config(ctx context.Context, tenantCode string) map[string]any {
	multiTenant := true
	if activeTenantCount, err := s.repository.CountActiveTenants(ctx); err == nil {
		multiTenant = activeTenantCount > 1
	}
	result := map[string]any{
		"name":        s.config.AppName,
		"title":       s.config.AppName,
		"appName":     s.config.AppName,
		"tenantCode":  tenantCode,
		"multiTenant": multiTenant,
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
	challengeContextKey := loginChallengeContextKey(input.Username, input.TenantCode, input.LoginIP)
	if _, err := s.cache.Get(ctx, challengeContextKey); err == nil {
		if err := s.verifySliderCaptcha(ctx, challengeContextKey, input.SliderChallengeID, input.SliderOffset); err != nil {
			return LoginResult{}, err
		}
	}
	user, err := s.repository.FindLoginUser(ctx, input.Username, input.TenantCode)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			s.requireSliderCaptcha(ctx, challengeContextKey)
		}
		return LoginResult{}, err
	}
	if user.Status != "active" || user.TenantStatus != "active" {
		return LoginResult{}, ErrForbidden
	}
	if !VerifyPassword(input.Password, user.PasswordHash) {
		s.requireSliderCaptcha(ctx, challengeContextKey)
		return LoginResult{}, ErrInvalidCredentials
	}
	// 验证状态仅用于限流式人机校验，清理失败不应影响已经通过的凭据登录。
	_ = s.cache.Delete(ctx, challengeContextKey)
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

func (s *Service) requireSliderCaptcha(ctx context.Context, contextKey string) {
	// 标记只用于控制下一次同一登录上下文是否要求人机验证；缓存失败不应掩盖凭据错误。
	_ = s.cache.Put(ctx, contextKey, "1", s.config.LoginChallengeTTL)
}

func (s *Service) verifySliderCaptcha(ctx context.Context, contextKey, challengeID string, offset int) error {
	if strings.TrimSpace(challengeID) == "" || offset < 0 || offset > sliderTrackWidth-sliderThumbWidth {
		return ErrInvalidCaptcha
	}
	challengeContextKey, err := s.cache.Take(ctx, sliderChallengeCacheKey(strings.TrimSpace(challengeID)))
	if err != nil {
		return ErrInvalidCaptcha
	}
	if challengeContextKey != contextKey || offset < sliderTrackWidth-sliderThumbWidth-sliderTolerance {
		return ErrInvalidCaptcha
	}
	return nil
}

func loginChallengeContextKey(username, tenantCode, loginIP string) string {
	value := strings.Join([]string{strings.TrimSpace(username), strings.TrimSpace(tenantCode), strings.TrimSpace(loginIP)}, "\n")
	sum := sha256.Sum256([]byte(value))
	return "login-challenge:" + hex.EncodeToString(sum[:])
}

func sliderChallengeCacheKey(challengeID string) string { return "slider-challenge:" + challengeID }

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
