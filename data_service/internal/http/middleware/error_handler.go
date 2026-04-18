package middleware

import (
	"net/http"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/response"
)

// ErrorHandlerFunc 表示可能返回业务错误的 HTTP 处理函数。
type ErrorHandlerFunc func(http.ResponseWriter, *http.Request) error

// ErrorHandler 将业务错误统一转换为 ApiResponse。
func ErrorHandler(next ErrorHandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := next(w, r); err != nil {
			requestID := RequestID(r.Context())
			appErr := normalizeError(err)
			response.WriteError(w, appErr.StatusCode, requestID, string(appErr.Code), appErr.Message)
		}
	})
}

// normalizeError 将任意错误转换为统一业务错误。
func normalizeError(err error) *apperrors.AppError {
	if err == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeUnknown, http.StatusInternalServerError, "未知错误")
	}

	var appErr *apperrors.AppError
	if AsAppError(err, &appErr) {
		return appErr
	}

	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "系统内部错误", err)
}

// AsAppError 允许在不直接依赖标准库 errors 包的情况下解包业务错误。
func AsAppError(err error, target **apperrors.AppError) bool {
	if err == nil || target == nil {
		return false
	}

	if appErr, ok := err.(*apperrors.AppError); ok {
		*target = appErr
		return true
	}

	type unwrapper interface {
		Unwrap() error
	}

	current := err
	for current != nil {
		if appErr, ok := current.(*apperrors.AppError); ok {
			*target = appErr
			return true
		}

		u, ok := current.(unwrapper)
		if !ok {
			break
		}
		current = u.Unwrap()
	}

	return false
}
