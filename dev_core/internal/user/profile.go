package user

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode"
	"unicode/utf8"
)

// 高级资料归入既有用户扩展 JSON 的 profile 命名空间，不覆盖界面偏好，也不参与鉴权。
func extendedProfile(item User) map[string]any {
	result := map[string]any{"gender": "", "attributes": map[string]string{}}
	if profile, ok := item.Preferences["profile"].(map[string]any); ok {
		if gender, ok := profile["gender"].(string); ok {
			result["gender"] = gender
		}
		if attrs, ok := profile["attributes"].(map[string]string); ok {
			result["attributes"] = attrs
		}
		if attrs, ok := profile["attributes"].(map[string]any); ok {
			values := map[string]string{}
			for key, value := range attrs {
				if text, ok := value.(string); ok {
					values[key] = text
				}
			}
			result["attributes"] = values
		}
	}
	return result
}
func validateUser(item User) error {
	invalid := func(message string) error { return fmt.Errorf("%w: %s", ErrInvalidInput, message) }
	if utf8.RuneCountInString(item.Username) < 3 || utf8.RuneCountInString(item.Username) > 50 || strings.ContainsFunc(item.Username, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '_' && r != '-' && r != '.'
	}) {
		return invalid("用户名需为 3–50 个字符，仅允许文字、数字、下划线、短横线和点")
	}
	if utf8.RuneCountInString(item.FullName) > 100 || utf8.RuneCountInString(item.Phone) > 40 || utf8.RuneCountInString(item.Email) > 254 {
		return invalid("姓名、电话或邮箱过长")
	}
	if item.Email != "" {
		address, err := mail.ParseAddress(item.Email)
		if err != nil || address.Address != item.Email || !strings.Contains(item.Email, "@") {
			return invalid("邮箱格式不正确")
		}
	}
	if item.Status != "active" && item.Status != "inactive" && item.Status != "suspended" {
		return invalid("用户状态无效")
	}
	profile := extendedProfile(item)
	switch profile["gender"] {
	case "", "male", "female", "other", "undisclosed":
	default:
		return invalid("性别选项无效")
	}
	attrs := profile["attributes"].(map[string]string)
	if len(attrs) > 20 {
		return invalid("自定义属性最多 20 项")
	}
	seen := map[string]bool{}
	for key, value := range attrs {
		name := strings.TrimSpace(key)
		normalized := strings.ToLower(name)
		if name == "" || name != key || utf8.RuneCountInString(name) > 50 || utf8.RuneCountInString(value) > 500 || strings.ContainsFunc(name, unicode.IsControl) {
			return invalid("属性名称最多 50 字符，属性值最多 500 字符")
		}
		switch normalized {
		case "__proto__", "prototype", "constructor", "id", "userid", "tenantid", "username", "password", "role", "status", "permissions", "avatar", "avatarurl":
			return invalid("自定义属性不能使用账号或权限字段名称")
		}
		if seen[normalized] {
			return invalid("属性名称不能重复")
		}
		seen[normalized] = true
	}
	return nil
}
