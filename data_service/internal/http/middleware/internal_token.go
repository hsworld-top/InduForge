package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/response"
)

const internalTokenHeader = "X-InduForge-Internal-Token"

// RequireInternalToken 仅允许持有 data_service 内部令牌的调用进入内部路由。
// 令牌只参与常量时间比较，且该中间件不会记录请求头或令牌内容。
func RequireInternalToken(expectedToken string) func(http.Handler) http.Handler {
	expectedToken = strings.TrimSpace(expectedToken)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			providedToken := strings.TrimSpace(r.Header.Get(internalTokenHeader))
			if expectedToken == "" || providedToken == "" || subtle.ConstantTimeCompare([]byte(providedToken), []byte(expectedToken)) != 1 {
				response.WriteAppError(w, http.StatusUnauthorized, RequestID(r.Context()), apperrors.ErrorCodeAuthTokenInvalid, "内部认证失败")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
