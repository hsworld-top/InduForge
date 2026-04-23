package errors

import (
	"strconv"
	"strings"
)

// ErrorCode 表示 data_service 内部统一错误码。
type ErrorCode string

const (
	// SuccessCode 是统一成功码，对外响应固定返回 0。
	SuccessCode = 0

	// PublicCodeUnknown 表示未知错误的公开错误码。
	PublicCodeUnknown = 30000
	// PublicCodeInternal 表示系统内部错误的公开错误码。
	PublicCodeInternal = 30001
	// PublicCodeBadRequest 表示参数校验错误的公开错误码。
	PublicCodeBadRequest = 20001
	// PublicCodeNotFound 表示资源不存在的公开错误码。
	PublicCodeNotFound = 21001
	// PublicCodeAuthTokenRequired 表示缺少 JWT 的公开错误码。
	PublicCodeAuthTokenRequired = 10001
	// PublicCodeAuthTokenInvalid 表示 JWT 无效的公开错误码。
	PublicCodeAuthTokenInvalid = 10002
	// PublicCodeAuthSecretRequired 表示服务端 JWT 配置缺失的公开错误码。
	PublicCodeAuthSecretRequired = 30002
	// PublicCodePermissionInsufficient 表示权限不足的公开错误码。
	PublicCodePermissionInsufficient = 11002
	// PublicCodePermissionProjectMismatch 表示项目范围不匹配的公开错误码。
	PublicCodePermissionProjectMismatch = 11004

	// ErrorCodeUnknown 表示未知错误。
	ErrorCodeUnknown ErrorCode = "UNKNOWN_ERROR"
	// ErrorCodeInternal 表示系统内部错误。
	ErrorCodeInternal ErrorCode = "INTERNAL_ERROR"
	// ErrorCodeBadRequest 表示请求参数错误。
	ErrorCodeBadRequest ErrorCode = "BAD_REQUEST"
	// ErrorCodeNotFound 表示资源不存在。
	ErrorCodeNotFound ErrorCode = "NOT_FOUND"
	// ErrorCodeAuthTokenRequired 表示未提供认证 JWT。
	ErrorCodeAuthTokenRequired ErrorCode = "AUTH_TOKEN_REQUIRED"
	// ErrorCodeAuthTokenInvalid 表示 JWT 校验失败。
	ErrorCodeAuthTokenInvalid ErrorCode = "AUTH_TOKEN_INVALID"
	// ErrorCodeAuthSecretRequired 表示 JWT 密钥未配置。
	ErrorCodeAuthSecretRequired ErrorCode = "AUTH_SECRET_REQUIRED"
	// ErrorCodePermissionInsufficient 表示权限不足。
	ErrorCodePermissionInsufficient ErrorCode = "PERMISSION_INSUFFICIENT"
	// ErrorCodePermissionProjectMismatch 表示请求项目不在当前 JWT 允许范围内。
	ErrorCodePermissionProjectMismatch ErrorCode = "PERMISSION_PROJECT_MISMATCH"
)

var publicCodeByErrorCode = map[ErrorCode]int{
	ErrorCodeUnknown:                   PublicCodeUnknown,
	ErrorCodeInternal:                  PublicCodeInternal,
	ErrorCodeBadRequest:                PublicCodeBadRequest,
	ErrorCodeNotFound:                  PublicCodeNotFound,
	ErrorCodeAuthTokenRequired:         PublicCodeAuthTokenRequired,
	ErrorCodeAuthTokenInvalid:          PublicCodeAuthTokenInvalid,
	ErrorCodeAuthSecretRequired:        PublicCodeAuthSecretRequired,
	ErrorCodePermissionInsufficient:    PublicCodePermissionInsufficient,
	ErrorCodePermissionProjectMismatch: PublicCodePermissionProjectMismatch,
}

var publicCodeByLegacyText = map[string]int{
	"SUCCESS":                                  SuccessCode,
	"00000":                                    SuccessCode,
	string(ErrorCodeUnknown):                   PublicCodeUnknown,
	string(ErrorCodeInternal):                  PublicCodeInternal,
	string(ErrorCodeBadRequest):                PublicCodeBadRequest,
	string(ErrorCodeNotFound):                  PublicCodeNotFound,
	string(ErrorCodeAuthTokenRequired):         PublicCodeAuthTokenRequired,
	string(ErrorCodeAuthTokenInvalid):          PublicCodeAuthTokenInvalid,
	string(ErrorCodeAuthSecretRequired):        PublicCodeAuthSecretRequired,
	string(ErrorCodePermissionInsufficient):    PublicCodePermissionInsufficient,
	string(ErrorCodePermissionProjectMismatch): PublicCodePermissionProjectMismatch,
}

// PublicCode 将内部错误码转换为对外整数错误码。
func (c ErrorCode) PublicCode() int {
	if code, ok := publicCodeByErrorCode[c]; ok {
		return code
	}
	return PublicCodeUnknown
}

// ResolvePublicCode 将历史文本错误码转换为对外整数错误码。
// 支持 data_service 旧文本码、以及 A/B/Cxxxx 样式错误码。
func ResolvePublicCode(rawCode string) int {
	normalized := strings.ToUpper(strings.TrimSpace(rawCode))
	if normalized == "" {
		return PublicCodeUnknown
	}

	if code, ok := publicCodeByLegacyText[normalized]; ok {
		return code
	}

	if matched := parseABCLegacyCode(normalized); matched != 0 {
		return matched
	}

	if numeric, err := strconv.Atoi(normalized); err == nil {
		return numeric
	}

	return PublicCodeUnknown
}

func parseABCLegacyCode(code string) int {
	if len(code) != 5 {
		return 0
	}

	module := code[0]
	sequence, err := strconv.Atoi(code[1:])
	if err != nil {
		return 0
	}

	switch module {
	case 'A':
		return 10000 + sequence
	case 'B':
		return 20000 + sequence
	case 'C':
		return 30000 + sequence
	default:
		return 0
	}
}
