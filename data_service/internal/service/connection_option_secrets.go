package service

import (
	"fmt"
	"net/http"
	"strings"
	"unicode"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// validateConnectionOptionsContainNoSecrets 防止扩展 options 绕过专用密钥表把敏感值写入 JSON 配置。
// 密钥只能通过 secrets/clearSecretKeys 更新；这里递归检查嵌套对象和数组。
func validateConnectionOptionsContainNoSecrets(options map[string]any) error {
	if key, ok := findSensitiveOptionKey(options); ok {
		return apperrors.NewAppError(
			apperrors.ErrorCodeBadRequest,
			http.StatusBadRequest,
			fmt.Sprintf("options.%s 属于敏感配置，请通过 secrets 提交", key),
		)
	}
	return nil
}

func findSensitiveOptionKey(value any) (string, bool) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if isSensitiveOptionKey(key) {
				return key, true
			}
			if nested, ok := findSensitiveOptionKey(child); ok {
				return key + "." + nested, true
			}
		}
	case []any:
		for _, child := range typed {
			if nested, ok := findSensitiveOptionKey(child); ok {
				return nested, true
			}
		}
	}
	return "", false
}

func isSensitiveOptionKey(key string) bool {
	normalized := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, key)
	for _, fragment := range []string{"password", "passwd", "token", "secret", "credential", "privatekey", "clientkey", "authorization"} {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	switch normalized {
	case "dsn", "apikey", "accesskey", "saslpassword", "ca", "cert", "certificate", "key":
		return true
	default:
		return false
	}
}
