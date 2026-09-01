package deployment

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

const maxSigningKeyBytes = 16 << 10

// LoadEd25519SigningKey 只读取受控 PKCS#8 PEM 私钥。调用方只能保留内存中的
// SigningConfig，禁止把密钥内容写入日志、数据库或 Release 清单。
func LoadEd25519SigningKey(path, keyID string) (SigningConfig, error) {
	path, keyID = strings.TrimSpace(path), strings.TrimSpace(keyID)
	if path == "" || !validSigningKeyID(keyID) {
		return SigningConfig{}, fmt.Errorf("签名密钥路径或 keyId 无效")
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return SigningConfig{}, fmt.Errorf("签名密钥必须是非链接普通文件")
	}
	if info.Size() <= 0 || info.Size() > maxSigningKeyBytes || info.Mode().Perm()&0o022 != 0 {
		return SigningConfig{}, fmt.Errorf("签名密钥文件大小或权限不安全")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return SigningConfig{}, fmt.Errorf("读取签名密钥失败")
	}
	block, rest := pem.Decode(raw)
	if block == nil || len(strings.TrimSpace(string(rest))) != 0 || block.Type != "PRIVATE KEY" {
		return SigningConfig{}, fmt.Errorf("签名密钥必须是 PKCS#8 PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return SigningConfig{}, fmt.Errorf("解析签名密钥失败")
	}
	key, ok := parsed.(ed25519.PrivateKey)
	if !ok || len(key) != ed25519.PrivateKeySize {
		return SigningConfig{}, fmt.Errorf("签名密钥必须是 Ed25519")
	}
	return SigningConfig{Key: key, KeyID: keyID}, nil
}
func validSigningKeyID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for i, c := range value {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || i > 0 && (c == '.' || c == '-' || c == '_') {
			continue
		}
		return false
	}
	return true
}
