package service

import (
	"fmt"
	"strings"
)

// toString 统一把开发态配置中的弱类型 JSON 值转换为文本。
// 该边界转换只用于外部配置解析，不改变数据库和公开响应的规范类型。
func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	}
}

func isSensitiveConfigKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	for _, part := range []string{"password", "passwd", "secret", "token", "credential", "privatekey", "private_key", "certificate"} {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}
