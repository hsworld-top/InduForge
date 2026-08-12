package api

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
)

type requestIDContextKey struct{}

func RequestID(generator func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
			if requestID == "" {
				requestID = generator()
			}
			w.Header().Set("X-Request-ID", requestID)
			ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(r.Context(), "HTTP 请求发生未处理异常",
						"requestId", RequestIDFromContext(r.Context()),
						"panic", recovered,
						"stack", string(debug.Stack()),
					)
					WriteError(w, r, http.StatusInternalServerError, ErrorCodeInternal, "系统内部错误")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
