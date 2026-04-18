package errors

// ErrorCode 表示 data_service 内部统一错误码。
type ErrorCode string

const (
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
