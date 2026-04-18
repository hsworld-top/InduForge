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
)
