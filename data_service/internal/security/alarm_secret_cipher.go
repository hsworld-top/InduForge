package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

const alarmSecretPayloadVersion byte = 1

// AlarmSecretCipher 使用独立密文格式保护报警通知渠道密钥。
type AlarmSecretCipher struct {
	keyVersion string
	aead       cipher.AEAD
}

func NewAlarmSecretCipher(key []byte, keyVersion string) (*AlarmSecretCipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("报警渠道密钥主密钥必须为 32 字节")
	}
	if keyVersion == "" {
		return nil, fmt.Errorf("报警渠道密钥版本不能为空")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("初始化报警渠道密钥加密器失败")
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("初始化报警渠道密钥认证加密失败")
	}
	return &AlarmSecretCipher{keyVersion: keyVersion, aead: aead}, nil
}

func (c *AlarmSecretCipher) KeyVersion() string { return c.keyVersion }

func (c *AlarmSecretCipher) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成报警渠道密钥随机数失败")
	}
	payload := make([]byte, 1, 1+len(nonce)+len(plaintext)+c.aead.Overhead())
	payload[0] = alarmSecretPayloadVersion
	payload = append(payload, nonce...)
	payload = c.aead.Seal(payload, nonce, plaintext, []byte("induforge.alarm-channel"))
	return payload, nil
}

func (c *AlarmSecretCipher) Decrypt(payload []byte) ([]byte, error) {
	minimum := 1 + c.aead.NonceSize() + c.aead.Overhead()
	if len(payload) < minimum || payload[0] != alarmSecretPayloadVersion {
		return nil, fmt.Errorf("报警渠道密钥密文格式无效")
	}
	nonce := payload[1 : 1+c.aead.NonceSize()]
	plaintext, err := c.aead.Open(nil, nonce, payload[1+c.aead.NonceSize():], []byte("induforge.alarm-channel"))
	if err != nil {
		return nil, fmt.Errorf("报警渠道密钥解密失败")
	}
	return plaintext, nil
}
