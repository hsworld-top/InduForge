package middleware

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/http/response"
)

const permissionErrorCodeInsufficient = "PERMISSION_INSUFFICIENT"

// RequireCapability 要求当前请求的 JWT 具备指定能力点。
func RequireCapability(required string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if required == "" {
				next.ServeHTTP(w, r)
				return
			}

			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				response.WriteError(w, http.StatusUnauthorized, RequestID(r.Context()), authErrorCodeTokenRequired, "请先完成认证")
				return
			}

			if !claims.HasCapability(required) {
				response.WriteError(w, http.StatusForbidden, RequestID(r.Context()), permissionErrorCodeInsufficient, "权限不足")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
