package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/indu-forge/dev_core/internal/objectstore"
	platformcache "github.com/indu-forge/dev_core/internal/platform/cache"
	"image"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
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
	avatars     AvatarStore
}

type AssetSigner interface {
	PresignGet(context.Context, string, time.Duration) (string, error)
}

type LoginInput struct {
	Platform          bool
	RememberMe        bool
	Username          string
	Password          string
	TenantCode        string
	SliderChallengeID string
	SliderOffset      int
	LoginIP           string
}

func (s *Service) SetAssetSigner(signer AssetSigner) { s.assets = signer }

type TokenPair struct {
	RememberMe       bool   `json:"-"`
	RefreshExpiresIn int64  `json:"-"`
	AccessToken      string `json:"accessToken"`
	Token            string `json:"token"`
	RefreshToken     string `json:"refreshToken"`
	ExpiresIn        int64  `json:"expiresIn"`
}

type LoginResult struct {
	TokenPair
	User map[string]any `json:"user"`
}

func NewService(repository Repository, cache CaptchaStore, tokens *TokenManager, config ServiceConfig, revocations ...RevocationStore) *Service {
	if config.AppName == "" {
		config.AppName = "InduFrame"
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

// GetActiveUser 为短期派生会话重新读取账号与租户状态，避免长期连接沿用签发时
// 已过期的角色或停用状态。它不签发新 Token，也不接受浏览器提供的身份字段。
func (s *Service) GetActiveUser(ctx context.Context, userID string) (User, error) {
	user, err := s.repository.GetUser(ctx, userID)
	if err != nil {
		return User{}, err
	}
	if !loginAllowed(user) || user.MustChangePassword {
		return User{}, ErrForbidden
	}
	return user, nil
}

func (s *Service) Captcha(ctx context.Context, username, tenantCode, loginIP string, platform bool) (map[string]any, error) {
	contextKey := loginChallengeContextKey(username, tenantCode, loginIP, platform)
	if _, err := s.cache.Get(ctx, contextKey); err != nil {
		return nil, ErrInvalidCaptcha
	}
	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("生成滑块挑战标识失败: %w", err)
	}
	challengeID := hex.EncodeToString(keyBytes)
	var background image.Image
	if !platform {
		background = s.captchaBackground(ctx, tenantCode)
	}
	result, targetX, err := generatePuzzle(background)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(puzzleChallenge{Context: contextKey, TargetX: targetX})
	if err != nil {
		return nil, err
	}
	if err := s.cache.Put(ctx, sliderChallengeCacheKey(challengeID), string(encoded), s.config.CaptchaTTL); err != nil {
		return nil, fmt.Errorf("保存拼图挑战失败: %w", err)
	}
	result["challengeId"], result["expireSeconds"] = challengeID, int64(s.config.CaptchaTTL.Seconds())
	return result, nil
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
	result["logoUrl"] = objectstore.BrandURL(branding.Code, branding.LogoObjectKey)
	result["loginBackgroundUrl"] = objectstore.BrandURL(branding.Code, branding.LoginBackgroundObjectKey)
	return result
}

func (s *Service) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.TenantCode = strings.TrimSpace(input.TenantCode)
	if input.Username == "" || input.Password == "" || (input.Platform && input.TenantCode != "") {
		return LoginResult{}, ErrInvalidCredentials
	}
	if !input.Platform && input.TenantCode == "" {
		return LoginResult{}, ErrTenantRequired
	}
	challengeContextKey := loginChallengeContextKey(input.Username, input.TenantCode, input.LoginIP, input.Platform)
	if _, err := s.cache.Get(ctx, challengeContextKey); err == nil || input.SliderChallengeID != "" {
		if err := s.verifySliderCaptcha(ctx, challengeContextKey, input.SliderChallengeID, input.SliderOffset); err != nil {
			return LoginResult{}, err
		}
	} else if !errors.Is(err, platformcache.ErrMiss) {
		return LoginResult{}, ErrInvalidCaptcha
	}
	user, err := s.repository.FindLoginUser(ctx, input.Username, input.TenantCode, input.Platform)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			s.requireSliderCaptcha(ctx, challengeContextKey)
		}
		return LoginResult{}, err
	}
	if !loginAllowed(user) {
		return LoginResult{}, ErrForbidden
	}
	if !VerifyPassword(input.Password, user.PasswordHash) {
		s.requireSliderCaptcha(ctx, challengeContextKey)
		return LoginResult{}, ErrInvalidCredentials
	}
	// 验证状态仅用于限流式人机校验，清理失败不应影响已经通过的凭据登录。
	_ = s.cache.Delete(ctx, challengeContextKey)
	pair, refresh, err := s.issueTokenPair(user, input.RememberMe)
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
	if err != nil || (!loginAllowed(user) || oldToken.CredentialVersion != user.CredentialVersion) {
		return TokenPair{}, ErrInvalidRefresh
	}
	pair, replacement, err := s.issueTokenPair(user, oldToken.RememberMe)
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
	if err != nil || (!loginAllowed(user) || claims.CredentialVersion != user.CredentialVersion) {
		return User{}, ErrUnauthorized
	}
	return user, nil
}

// AuthenticateSession 供长连接使用已验证的账号与令牌截止时间；不得向浏览器返回令牌本身。
func (s *Service) AuthenticateSession(ctx context.Context, accessToken string) (User, time.Time, error) {
	claims, err := s.tokens.ParseAccessToken(accessToken)
	if err != nil {
		return User{}, time.Time{}, err
	}
	if claims.ExpiresAt == nil {
		return User{}, time.Time{}, ErrUnauthorized
	}
	actor, err := s.Authenticate(ctx, accessToken)
	if err != nil {
		return User{}, time.Time{}, err
	}
	if actor.MustChangePassword {
		return User{}, time.Time{}, ErrForbidden
	}
	return actor, claims.ExpiresAt.Time, nil
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
	user, err := s.repository.FindLoginUser(ctx, username, tenantCode, false)
	if err != nil || !loginAllowed(user) || user.MustChangePassword || !VerifyPassword(password, user.PasswordHash) {
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
	if utf8.RuneCountInString(newPassword) < 8 || newPassword == oldPassword {
		return fmt.Errorf("%w: 新密码至少 8 位且不能与旧密码相同", ErrInvalidCredentials)
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repository.UpdatePassword(ctx, user.ID, hash, user.CredentialVersion); err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}
	return s.repository.RevokeUserRefreshTokens(ctx, user.ID)
}

func (s *Service) issueTokenPair(user User, rememberMe bool) (TokenPair, RefreshToken, error) {
	access, expiresAt, err := s.tokens.IssueAccessToken(user)
	if err != nil {
		return TokenPair{}, RefreshToken{}, err
	}
	rawRefresh, refreshHash, refreshExpiresAt, err := s.tokens.NewRefreshToken()
	if err != nil {
		return TokenPair{}, RefreshToken{}, err
	}
	return TokenPair{RememberMe: rememberMe, RefreshExpiresIn: int64(time.Until(refreshExpiresAt).Seconds()), AccessToken: access, Token: access, RefreshToken: rawRefresh, ExpiresIn: int64(time.Until(expiresAt).Seconds())}, RefreshToken{RememberMe: rememberMe, CredentialVersion: user.CredentialVersion, TenantID: user.TenantID, UserID: user.ID, Hash: refreshHash, ExpiresAt: refreshExpiresAt}, nil
}

func (s *Service) requireSliderCaptcha(ctx context.Context, contextKey string) {
	// 标记只用于控制下一次同一登录上下文是否要求人机验证；缓存失败不应掩盖凭据错误。
	_ = s.cache.Put(ctx, contextKey, "1", s.config.LoginChallengeTTL)
}

func (s *Service) verifySliderCaptcha(ctx context.Context, contextKey, challengeID string, offset int) error {
	if strings.TrimSpace(challengeID) == "" {
		return ErrInvalidCaptcha
	}
	encoded, err := s.cache.Take(ctx, sliderChallengeCacheKey(strings.TrimSpace(challengeID)))
	if err != nil {
		return ErrInvalidCaptcha
	}
	var challenge puzzleChallenge
	if json.Unmarshal([]byte(encoded), &challenge) != nil || challenge.Context != contextKey || offset < 0 || offset > captchaWidth || offset < challenge.TargetX-5 || offset > challenge.TargetX+5 {
		return ErrInvalidCaptcha
	}
	return nil
}
func loginChallengeContextKey(username, tenantCode, loginIP string, platform bool) string {
	value := strings.Join([]string{strings.TrimSpace(username), strings.TrimSpace(tenantCode), strconv.FormatBool(platform), strings.TrimSpace(loginIP)}, "\n")
	sum := sha256.Sum256([]byte(value))
	return "login-challenge:" + hex.EncodeToString(sum[:])
}

func sliderChallengeCacheKey(challengeID string) string { return "slider-challenge:" + challengeID }

func publicUser(user User) map[string]any {
	result := map[string]any{
		"mustChangePassword": user.MustChangePassword, "platform": user.Role == "SUPER_ADMIN",
		"id": user.ID, "tenantId": user.TenantID, "username": user.Username,
		"email": user.Email, "fullName": user.FullName, "role": user.Role, "avatarUrl": avatarURL(user),
		"tenant": map[string]any{"id": user.TenantID, "code": user.TenantCode, "name": user.TenantName, "logoUrl": user.LogoURL},
	}
	if user.Role == "SUPER_ADMIN" {
		result["tenant"], result["tenantId"] = nil, nil
	}
	return result
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

// 平台账号没有租户，租户账号必须属于已启用且已初始化的租户。
func loginAllowed(user User) bool {
	return user.Status == "active" && (user.Role == "SUPER_ADMIN" && user.TenantID == "" || user.Role != "SUPER_ADMIN" && user.TenantID != "" && user.TenantStatus == "active" && user.TenantInitialized)
}
