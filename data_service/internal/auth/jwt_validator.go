package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

const hs256Algorithm = "HS256"

// JWTValidator 使用共享密钥校验 Bearer JWT 的签名、结构与时间声明。
type JWTValidator struct {
	secret    []byte
	centerURL string
	client    *http.Client
}

// NewJWTValidator 创建一个基于 HMAC-SHA256 的 JWT 校验器。
func NewJWTValidator(secret, centerURL string) (*JWTValidator, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthSecretRequired, http.StatusInternalServerError, "JWT secret 未配置")
	}

	endpoint, err := identityEndpoint(centerURL)
	if err != nil {
		return nil, err
	}
	return &JWTValidator{secret: []byte(secret), centerURL: endpoint, client: &http.Client{
		Timeout:       3 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}, nil
}

// Validate 校验并解析 JWT，失败时返回错误。
func (v *JWTValidator) Validate(token string) (*Claims, error) {
	return v.ValidateContext(context.Background(), token)
}

// ValidateContext 每次验签后向中心复查当前身份；撤销、改密状态和中心故障均不缓存放行。
func (v *JWTValidator) ValidateContext(ctx context.Context, token string) (*Claims, error) {
	if v == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthSecretRequired, http.StatusInternalServerError, "JWT 校验器未初始化")
	}
	if len(v.secret) == 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthSecretRequired, http.StatusInternalServerError, "JWT secret 未配置")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT 结构无效")
	}

	headerBytes, err := decodeSegment(parts[0])
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT header 解码失败", err)
	}

	signature, err := decodeSegment(parts[2])
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT signature 解码失败", err)
	}

	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT header 解析失败", err)
	}
	if header.Alg != hs256Algorithm {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, fmt.Sprintf("不支持的 JWT 算法: %s", header.Alg))
	}

	mac := hmac.New(sha256.New, v.secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	expected := mac.Sum(nil)
	if !hmac.Equal(signature, expected) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT 签名校验失败")
	}

	payloadBytes, err := decodeSegment(parts[1])
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT payload 解码失败", err)
	}

	var payload jwtPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT payload 解析失败", err)
	}

	if err := payload.validateTimeClaims(time.Now().UTC()); err != nil {
		return nil, err
	}

	if err := v.verifyIdentity(ctx, token, &payload.Claims); err != nil {
		return nil, err
	}
	return &payload.Claims, nil
}

func decodeSegment(segment string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(segment)
}

type jwtPayload struct {
	Claims
	Exp *json.Number `json:"exp"`
	Nbf *json.Number `json:"nbf"`
	Iat *json.Number `json:"iat"`
}

func (p *jwtPayload) validateTimeClaims(now time.Time) error {
	nowUnix := now.Unix()

	if p.Exp != nil {
		exp, err := jsonNumberToUnix(*p.Exp)
		if err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT exp 声明无效", err)
		}
		if nowUnix >= exp {
			return apperrors.NewAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT 已过期")
		}
	}

	if p.Nbf != nil {
		nbf, err := jsonNumberToUnix(*p.Nbf)
		if err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT nbf 声明无效", err)
		}
		if nowUnix < nbf {
			return apperrors.NewAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT 尚未生效")
		}
	}

	if p.Iat != nil {
		iat, err := jsonNumberToUnix(*p.Iat)
		if err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT iat 声明无效", err)
		}
		if nowUnix < iat {
			return apperrors.NewAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "JWT 签发时间晚于当前时间")
		}
	}

	return nil
}

func jsonNumberToUnix(number json.Number) (int64, error) {
	return number.Int64()
}
