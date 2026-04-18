package middleware

import (
	"net/http"
	"strings"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/http/response"
)

const (
	authErrorCodeTokenRequired = "AUTH_TOKEN_REQUIRED"
	authErrorCodeTokenInvalid  = "AUTH_TOKEN_INVALID"
)

// Authenticate 校验 Bearer JWT，并将解析后的 claims 注入上下文。
func Authenticate(validator *auth.JWTValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerTokenFromHeader(r.Header.Get("Authorization"))
			if !ok {
				writeAuthError(w, r, http.StatusUnauthorized, authErrorCodeTokenRequired, "请提供 Bearer JWT")
				return
			}

			claims, err := validator.Validate(token)
			if err != nil {
				writeAuthError(w, r, http.StatusUnauthorized, authErrorCodeTokenInvalid, "JWT 校验失败")
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), claims)))
		})
	}
}

func bearerTokenFromHeader(header string) (string, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", false
	}

	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}

	return token, true
}

func writeAuthError(w http.ResponseWriter, r *http.Request, statusCode int, errorCode, message string) {
	response.WriteError(w, statusCode, RequestID(r.Context()), errorCode, message)
}
