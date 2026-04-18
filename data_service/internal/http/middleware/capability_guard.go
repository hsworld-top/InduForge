package middleware

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/response"
)

// RequireCapability 要求当前请求的 JWT 具备指定能力点，并在 projectId 存在时校验项目边界。
func RequireCapability(required string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if required == "" {
				writePermissionError(w, r, http.StatusBadRequest, apperrors.ErrorCodeBadRequest, "能力点不能为空")
				return
			}

			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				writePermissionError(w, r, http.StatusUnauthorized, apperrors.ErrorCodeAuthTokenRequired, "请先完成认证")
				return
			}

			if !claims.HasCapability(required) {
				writePermissionError(w, r, http.StatusForbidden, apperrors.ErrorCodePermissionInsufficient, "权限不足")
				return
			}

			projectID := strings.TrimSpace(mux.Vars(r)["projectId"])
			if projectID != "" && !claims.HasProjectAccess(projectID) {
				writePermissionError(w, r, http.StatusForbidden, apperrors.ErrorCodePermissionProjectMismatch, "项目范围不足")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writePermissionError(w http.ResponseWriter, r *http.Request, statusCode int, errorCode apperrors.ErrorCode, message string) {
	response.WriteError(w, statusCode, RequestID(r.Context()), string(errorCode), message)
}
