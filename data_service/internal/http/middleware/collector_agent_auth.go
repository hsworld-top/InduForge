package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
)

type CollectorAgentAuthenticator interface {
	AuthenticateAgent(context.Context, string) (*auth.CollectorAgentIdentity, error)
}

// AuthenticateCollectorAgent 校验独立 Agent Token，避免与平台 JWT 权限域混用。
func AuthenticateCollectorAgent(authenticator CollectorAgentAuthenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authenticator == nil {
				writeAuthError(w, r, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "Agent 认证器未初始化"))
				return
			}
			token, ok := bearerTokenFromHeader(r.Header.Get("Authorization"))
			if !ok {
				writeAuthError(w, r, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请提供 Agent Bearer Token"))
				return
			}
			identity, err := authenticator.AuthenticateAgent(r.Context(), strings.TrimSpace(token))
			if err != nil {
				var appErr *apperrors.AppError
				if AsAppError(err, &appErr) {
					writeAuthError(w, r, appErr)
				} else {
					writeAuthError(w, r, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "Agent Token 校验失败"))
				}
				return
			}
			next.ServeHTTP(w, r.WithContext(auth.WithCollectorAgent(r.Context(), identity)))
		})
	}
}
