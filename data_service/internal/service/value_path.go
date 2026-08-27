package service

import (
	"fmt"
	"net/http"
	"strings"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// normalizeValuePathSegments 将字段路径规范为字符串键和非负整数索引，避免点号字段被错误拆分。
func normalizeValuePathSegments(input []any, allowEmpty bool) ([]any, error) {
	if len(input) == 0 {
		if allowEmpty {
			return []any{}, nil
		}
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "字段路径不能为空")
	}
	result := make([]any, 0, len(input))
	for _, segment := range input {
		switch value := segment.(type) {
		case string:
			value = strings.TrimSpace(value)
			if value == "" {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "字段路径不能包含空字段名")
			}
			result = append(result, value)
		case int:
			if value < 0 {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数组索引不能为负数")
			}
			result = append(result, value)
		case float64:
			index := int(value)
			if value != float64(index) || index < 0 {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数组索引必须为非负整数")
			}
			result = append(result, index)
		default:
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fmt.Sprintf("字段路径片段类型不受支持：%T", segment))
		}
	}
	return result, nil
}

// extractValueBySegments 是 HTTP、WebSocket、Redis 和 Kafka 共用的字段提取器。
func extractValueBySegments(root any, segments []any) (any, bool) {
	current := root
	for _, segment := range segments {
		switch value := segment.(type) {
		case string:
			object, ok := current.(map[string]any)
			if !ok {
				return nil, false
			}
			next, exists := object[value]
			if !exists {
				return nil, false
			}
			current = next
		case int:
			array, ok := current.([]any)
			if !ok || value < 0 || value >= len(array) {
				return nil, false
			}
			current = array[value]
		case float64:
			index := int(value)
			array, ok := current.([]any)
			if !ok || value != float64(index) || index < 0 || index >= len(array) {
				return nil, false
			}
			current = array[index]
		default:
			return nil, false
		}
	}
	return current, true
}
