package authoringsnapshot

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// Keyring 保留当前写入密钥及仍需读取历史版本/备份的旧密钥。调用方只持久化 ID，
// API、日志和快照元数据均不得包含密钥正文。
type Keyring struct {
	current string
	keys    map[string][]byte
}

func NewKeyring(current string, encoded map[string]string) (*Keyring, error) {
	current = strings.TrimSpace(current)
	if current == "" || len(encoded) == 0 {
		return nil, fmt.Errorf("authoring snapshot keyring 配置不完整")
	}
	keys := make(map[string][]byte, len(encoded))
	for id, value := range encoded {
		id = strings.TrimSpace(id)
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
		if id == "" || err != nil || len(decoded) != 32 {
			return nil, fmt.Errorf("authoring snapshot keyring 包含无效密钥")
		}
		keys[id] = append([]byte(nil), decoded...)
	}
	if _, ok := keys[current]; !ok {
		return nil, fmt.Errorf("authoring snapshot current key 不在 keyring 中")
	}
	return &Keyring{current: current, keys: keys}, nil
}

func (r *Keyring) Current() (string, []byte, error) {
	if r == nil {
		return "", nil, fmt.Errorf("authoring snapshot keyring 未配置")
	}
	key, ok := r.keys[r.current]
	if !ok {
		return "", nil, fmt.Errorf("authoring snapshot current key 不存在")
	}
	return r.current, append([]byte(nil), key...), nil
}

func (r *Keyring) Resolve(id string) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("authoring snapshot keyring 未配置")
	}
	key, ok := r.keys[id]
	if !ok {
		return nil, fmt.Errorf("authoring snapshot keyId 未知")
	}
	return append([]byte(nil), key...), nil
}
