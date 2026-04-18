package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

const hs256Algorithm = "HS256"

// JWTValidator 使用共享密钥校验 Bearer JWT 的签名、结构与时间声明。
type JWTValidator struct {
	secret []byte
}

// NewJWTValidator 创建一个基于 HMAC-SHA256 的 JWT 校验器。
func NewJWTValidator(secret string) (*JWTValidator, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("%s: JWT secret 未配置", apperrors.ErrorCodeAuthSecretRequired)
	}

	return &JWTValidator{secret: []byte(secret)}, nil
}

// Validate 校验并解析 JWT，失败时返回错误。
func (v *JWTValidator) Validate(token string) (*Claims, error) {
	if v == nil {
		return nil, errors.New("JWT 校验器未初始化")
	}
	if len(v.secret) == 0 {
		return nil, fmt.Errorf("%s: JWT secret 未配置", apperrors.ErrorCodeAuthSecretRequired)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("JWT 结构无效")
	}

	headerBytes, err := decodeSegment(parts[0])
	if err != nil {
		return nil, fmt.Errorf("JWT header 解码失败: %w", err)
	}

	payloadBytes, err := decodeSegment(parts[1])
	if err != nil {
		return nil, fmt.Errorf("JWT payload 解码失败: %w", err)
	}

	signature, err := decodeSegment(parts[2])
	if err != nil {
		return nil, fmt.Errorf("JWT signature 解码失败: %w", err)
	}

	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("JWT header 解析失败: %w", err)
	}
	if header.Alg != hs256Algorithm {
		return nil, fmt.Errorf("不支持的 JWT 算法: %s", header.Alg)
	}

	var payload jwtPayload
	decoder := json.NewDecoder(bytes.NewReader(payloadBytes))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("JWT payload 解析失败: %w", err)
	}

	now := time.Now().UTC()
	if err := payload.validateTimeClaims(now); err != nil {
		return nil, err
	}

	mac := hmac.New(sha256.New, v.secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	expected := mac.Sum(nil)
	if !hmac.Equal(signature, expected) {
		return nil, errors.New("JWT 签名校验失败")
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
			return fmt.Errorf("JWT exp 声明无效: %w", err)
		}
		if nowUnix >= exp {
			return errors.New("JWT 已过期")
		}
	}

	if p.Nbf != nil {
		nbf, err := jsonNumberToUnix(*p.Nbf)
		if err != nil {
			return fmt.Errorf("JWT nbf 声明无效: %w", err)
		}
		if nowUnix < nbf {
			return errors.New("JWT 尚未生效")
		}
	}

	if p.Iat != nil {
		iat, err := jsonNumberToUnix(*p.Iat)
		if err != nil {
			return fmt.Errorf("JWT iat 声明无效: %w", err)
		}
		if nowUnix < iat {
			return errors.New("JWT 签发时间晚于当前时间")
		}
	}

	return nil
}

func jsonNumberToUnix(number json.Number) (int64, error) {
	return number.Int64()
}
