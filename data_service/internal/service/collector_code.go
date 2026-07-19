package service

import (
	"fmt"
	"strings"
	"unicode"
)

func normalizeCollectorDisplayName(value string, maxLength int) (string, error) {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" || len([]rune(value)) > maxLength {
		return "", fmt.Errorf("名称长度必须为 1 到 %d 个字符", maxLength)
	}
	for _, char := range value {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != ' ' {
			return "", fmt.Errorf("名称只能包含文字、数字和空格")
		}
	}
	return value, nil
}

func collectorCodeFromName(value, fallback string) string {
	var builder strings.Builder
	pendingSeparator := false
	for _, char := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			if pendingSeparator && builder.Len() > 0 {
				builder.WriteByte('_')
			}
			builder.WriteRune(char)
			pendingSeparator = false
		} else {
			pendingSeparator = true
		}
	}
	code := strings.Trim(builder.String(), "_")
	if code == "" {
		return fallback
	}
	return code
}

func allocateCollectorPointCode(name string, used map[string]struct{}) string {
	baseRunes := []rune(collectorCodeFromName(name, "point"))
	if len(baseRunes) > 90 {
		baseRunes = baseRunes[:90]
	}
	base := string(baseRunes)
	candidate := base
	for suffix := 2; ; suffix++ {
		if _, exists := used[candidate]; !exists {
			used[candidate] = struct{}{}
			return candidate
		}
		candidate = fmt.Sprintf("%s_%d", base, suffix)
	}
}
