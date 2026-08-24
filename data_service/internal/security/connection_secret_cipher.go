package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"strings"
)

const connectionSecretPayloadVersion byte = 1

// ConnectionSecretCipher 加密外部接入源密钥，并将连接和键名绑定到认证数据，防止密文被跨连接替换。
type ConnectionSecretCipher struct {
	keyVersion string
	aead       cipher.AEAD
}

func NewConnectionSecretCipher(key []byte, keyVersion string) (*ConnectionSecretCipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("接入源密钥主密钥必须为 32 字节")
	}
	keyVersion = strings.TrimSpace(keyVersion)
	if keyVersion == "" {
		return nil, fmt.Errorf("接入源密钥版本不能为空")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("初始化接入源密钥加密器失败")
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("初始化接入源密钥认证加密失败")
	}
	return &ConnectionSecretCipher{keyVersion: keyVersion, aead: aead}, nil
}

func (c *ConnectionSecretCipher) KeyVersion() string { return c.keyVersion }

func (c *ConnectionSecretCipher) Encrypt(connectionID, key string, plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成接入源密钥随机数失败")
	}
	payload := []byte{connectionSecretPayloadVersion}
	payload = append(payload, nonce...)
	return c.aead.Seal(payload, nonce, plaintext, []byte("induforge.connection:"+connectionID+":"+key+":"+c.keyVersion)), nil
}

func (c *ConnectionSecretCipher) Decrypt(connectionID, key string, payload []byte) ([]byte, error) {
	minimum := 1 + c.aead.NonceSize() + c.aead.Overhead()
	if len(payload) < minimum || payload[0] != connectionSecretPayloadVersion {
		return nil, fmt.Errorf("接入源密钥密文格式无效")
	}
	nonceEnd := 1 + c.aead.NonceSize()
	plaintext, err := c.aead.Open(nil, payload[1:nonceEnd], payload[nonceEnd:], []byte("induforge.connection:"+connectionID+":"+key+":"+c.keyVersion))
	if err != nil {
		return nil, fmt.Errorf("接入源密钥解密失败")
	}
	return plaintext, nil
}
