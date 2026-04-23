package middleware

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/response"
)

// ErrorHandlerFunc 表示一个可能返回业务错误的 HTTP 处理函数。
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
			statusCode := normalizeStatusCode(appErr.StatusCode, true)
			response.WriteAppError(rw, statusCode, requestID, appErr.Code, appErr.Message)
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

// normalizeStatusCode 将非法状态码回退为 500，并按场景决定是否允许业务 2xx。
func normalizeStatusCode(statusCode int, allowBusinessStatus bool) int {
	if allowBusinessStatus && statusCode >= http.StatusOK && statusCode <= 299 {
		return statusCode
	}
	if statusCode < http.StatusBadRequest || statusCode > 599 {
		return http.StatusInternalServerError
	}
	return statusCode
}

// normalizeTechnicalStatusCode 仅允许技术异常状态码（4xx/5xx）。
func normalizeTechnicalStatusCode(statusCode int) int {
	return normalizeStatusCode(statusCode, false)
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

// Hijack 透传底层连接劫持能力，不支持时返回标准错误。
func (w *capturingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}

	w.committed = true
	return hijacker.Hijack()
}

// Flush 透传底层刷新能力，不支持时保持空操作。
func (w *capturingResponseWriter) Flush() {
	flusher, ok := w.ResponseWriter.(http.Flusher)
	if !ok {
		return
	}

	flusher.Flush()
	w.committed = true
}

// Push 透传底层 HTTP/2 推送能力，不支持时返回标准错误。
func (w *capturingResponseWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}

	return pusher.Push(target, opts)
}

// ReadFrom 透传底层零拷贝能力，不支持时回退到通用拷贝逻辑。
func (w *capturingResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	w.committed = true

	if readerFrom, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		return readerFrom.ReadFrom(r)
	}

	return io.Copy(w.ResponseWriter, r)
}
