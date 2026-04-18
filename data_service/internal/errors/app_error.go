package errors

import "fmt"

// AppError 表示 data_service 的业务错误。
type AppError struct {
	Code       ErrorCode
	Message    string
	StatusCode int
	Err        error
}

// NewAppError 创建一个业务错误。
func NewAppError(code ErrorCode, statusCode int, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// WrapAppError 在保留原始错误的同时创建业务错误。
func WrapAppError(code ErrorCode, statusCode int, message string, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
	}
}

// Error 返回错误的文本表示。
func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap 返回被包装的原始错误。
func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
