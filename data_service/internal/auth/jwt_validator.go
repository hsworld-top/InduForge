package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const hs256Algorithm = "HS256"

// JWTValidator 使用共享密钥校验 Bearer JWT 的签名与结构。
type JWTValidator struct {
	secret []byte
}

// NewJWTValidator 创建一个基于 HMAC-SHA256 的 JWT 校验器。
func NewJWTValidator(secret string) *JWTValidator {
	return &JWTValidator{secret: []byte(secret)}
}

// Validate 校验并解析 JWT，失败时返回错误。
func (v *JWTValidator) Validate(token string) (*Claims, error) {
	if v == nil {
		return nil, errors.New("JWT 校验器未初始化")
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

	mac := hmac.New(sha256.New, v.secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	expected := mac.Sum(nil)
	if !hmac.Equal(signature, expected) {
		return nil, errors.New("JWT 签名校验失败")
	}

	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("JWT payload 解析失败: %w", err)
	}

	return &claims, nil
}

func decodeSegment(segment string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(segment)
}
