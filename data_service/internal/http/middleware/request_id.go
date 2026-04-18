package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const requestIDHeader = "X-Request-ID"
const maxRequestIDLength = 64

var requestIDAllowedPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type requestIDKey struct{}

// RequestID 从请求上下文中读取请求 ID。
func RequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if value, ok := ctx.Value(requestIDKey{}).(string); ok {
		return value
	}
	return ""
}

// WithRequestID 将请求 ID 写入上下文。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestIDMiddleware 为每个请求补齐并回写请求 ID。
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := normalizeRequestID(r.Header.Get(requestIDHeader))

		w.Header().Set(requestIDHeader, requestID)
		ctx := WithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// normalizeRequestID 校验请求 ID，不合法时生成新的值。
func normalizeRequestID(requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if len(requestID) == 0 || len(requestID) > maxRequestIDLength {
		return generateRequestID()
	}
	if !requestIDAllowedPattern.MatchString(requestID) {
		return generateRequestID()
	}
	return requestID
}

// generateRequestID 生成一个轻量级请求 ID。
func generateRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
}
