package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrAccessTokenExpired 仅供长连接区分可续租到期与账号撤权；REST仍统一返回未授权。
var ErrAccessTokenExpired = errors.New("访问令牌已到期")

type Claims struct {
	CredentialVersion int64  `json:"credentialVersion"`
	UserID            string `json:"userId"`
	TenantID          string `json:"tenantId"`
	Username          string `json:"username"`
	Role              string `json:"role"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret     []byte
	issuer     string
	audience   string
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

func NewTokenManager(secret, issuer, audience string, accessTTL, refreshTTL time.Duration) (*TokenManager, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("JWT 密钥长度不能少于 16 字节")
	}
	if accessTTL <= 0 || refreshTTL <= 0 {
		return nil, fmt.Errorf("Token 有效期必须大于零")
	}
	return &TokenManager{
		secret:     []byte(secret),
		issuer:     issuer,
		audience:   audience,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		now:        time.Now,
	}, nil
}

func (m *TokenManager) IssueAccessToken(user User) (string, time.Time, error) {
	now := m.now().UTC()
	expiresAt := now.Add(m.accessTTL)
	tokenID, err := randomTokenValue(16)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("生成访问令牌标识失败: %w", err)
	}
	claims := Claims{
		CredentialVersion: user.CredentialVersion,
		UserID:            user.ID,
		TenantID:          user.TenantID,
		Username:          user.Username,
		Role:              user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("签发访问令牌失败: %w", err)
	}
	return value, expiresAt, nil
}

func (m *TokenManager) ParseAccessToken(value string) (Claims, error) {
	claims := Claims{}
	token, err := jwt.ParseWithClaims(value, &claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("不支持的 JWT 签名算法")
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithAudience(m.audience), jwt.WithExpirationRequired())
	if errors.Is(err, jwt.ErrTokenExpired) {
		return Claims{}, ErrAccessTokenExpired
	}
	if err != nil || !token.Valid {
		return Claims{}, ErrUnauthorized
	}
	return claims, nil
}

func (m *TokenManager) NewRefreshToken() (raw, hash string, expiresAt time.Time, err error) {
	raw, err = randomTokenValue(32)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("生成刷新令牌失败: %w", err)
	}
	hash = HashRefreshToken(raw)
	expiresAt = m.now().UTC().Add(m.refreshTTL)
	return raw, hash, expiresAt, nil
}

func HashRefreshToken(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(digest[:])
}

func randomTokenValue(length int) (string, error) {
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
