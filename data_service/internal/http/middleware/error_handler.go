package middleware

import (
	"errors"
	"net/http"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/response"
)

// ErrorHandlerFunc 表示可能返回业务错误的 HTTP 处理函数。
type ErrorHandlerFunc func(http.ResponseWriter, *http.Request) error

// ErrorHandler 将业务错误统一转换为 ApiResponse。
func ErrorHandler(next ErrorHandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &capturingResponseWriter{ResponseWriter: w}
		if err := next(rw, r); err != nil {
			if rw.committed {
				return
			}

			requestID := RequestID(r.Context())
			appErr := normalizeError(err)
			statusCode := normalizeStatusCode(appErr.StatusCode)
			response.WriteError(rw, statusCode, requestID, string(appErr.Code), appErr.Message)
		}
	})
}

// normalizeError 将任意错误转换为统一业务错误。
func normalizeError(err error) *apperrors.AppError {
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

	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		*target = appErr
		return true
	}
	return false
}

// normalizeStatusCode 将非法状态码回退为 500。
func normalizeStatusCode(statusCode int) int {
	if statusCode < http.StatusBadRequest || statusCode > 599 {
		return http.StatusInternalServerError
	}
	return statusCode
}

type capturingResponseWriter struct {
	http.ResponseWriter
	committed bool
}

func (w *capturingResponseWriter) WriteHeader(statusCode int) {
	w.committed = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *capturingResponseWriter) Write(p []byte) (int, error) {
	w.committed = true
	return w.ResponseWriter.Write(p)
}
