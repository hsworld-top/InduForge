package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"strings"
)

const collectorSecretPayloadVersion byte = 1

// CollectorSecretCipher 使用带认证加密保护采集连接密钥。
// 输入输出均为字节切片，密文自带随机 nonce；任何格式或认证失败只返回脱敏错误。
type CollectorSecretCipher struct {
	keyVersion string
	aead       cipher.AEAD
}

func NewCollectorSecretCipher(key []byte, keyVersion string) (*CollectorSecretCipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("采集连接密钥主密钥必须为 32 字节")
	}
	keyVersion = strings.TrimSpace(keyVersion)
	if keyVersion == "" {
		return nil, fmt.Errorf("采集连接密钥版本不能为空")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("初始化采集连接密钥加密器失败")
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("初始化采集连接密钥认证加密失败")
	}
	return &CollectorSecretCipher{keyVersion: keyVersion, aead: aead}, nil
}

func (c *CollectorSecretCipher) KeyVersion() string {
	return c.keyVersion
}

func (c *CollectorSecretCipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成采集连接密钥随机数失败")
	}
	payload := make([]byte, 1, 1+len(nonce)+len(plaintext)+c.aead.Overhead())
	payload[0] = collectorSecretPayloadVersion
	payload = append(payload, nonce...)
	payload = c.aead.Seal(payload, nonce, plaintext, []byte(c.keyVersion))
	return payload, nil
}

func (c *CollectorSecretCipher) Decrypt(payload []byte) ([]byte, error) {
	minimumLength := 1 + c.aead.NonceSize() + c.aead.Overhead()
	if len(payload) < minimumLength || payload[0] != collectorSecretPayloadVersion {
		return nil, fmt.Errorf("采集连接密钥密文格式无效")
	}
	nonceEnd := 1 + c.aead.NonceSize()
	nonce := payload[1:nonceEnd]
	plaintext, err := c.aead.Open(nil, nonce, payload[nonceEnd:], []byte(c.keyVersion))
	if err != nil {
		return nil, fmt.Errorf("采集连接密钥解密失败")
	}
	return plaintext, nil
}
