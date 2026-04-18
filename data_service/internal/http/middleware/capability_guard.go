package middleware

import (
	"net/http"
	"strings"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/response"
)

// RequireCapability 要求当前请求的 JWT 具备指定能力点，并在 projectId 存在时校验项目边界。
func RequireCapability(required string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.TrimSpace(required) == "" {
				writePermissionError(w, r, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "能力点不能为空"))
				return
			}

			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				writePermissionError(w, r, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成认证"))
				return
			}

			if !claims.HasCapability(required) {
				writePermissionError(w, r, apperrors.NewAppError(apperrors.ErrorCodePermissionInsufficient, http.StatusForbidden, "权限不足"))
				return
			}

			projectID, isProjectRoute := projectIDFromPath(r.URL.Path)
			if isProjectRoute {
				if projectID == "" {
					writePermissionError(w, r, apperrors.NewAppError(apperrors.ErrorCodePermissionProjectMismatch, http.StatusForbidden, "项目范围不足"))
					return
				}
				if !claims.HasProjectAccess(projectID) {
					writePermissionError(w, r, apperrors.NewAppError(apperrors.ErrorCodePermissionProjectMismatch, http.StatusForbidden, "项目范围不足"))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func projectIDFromPath(path string) (string, bool) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i, segment := range segments {
		if segment != "projects" {
			continue
		}
		if i+1 >= len(segments) {
			return "", true
		}

		projectID := strings.TrimSpace(segments[i+1])
		return projectID, true
	}

	return "", false
}

func writePermissionError(w http.ResponseWriter, r *http.Request, appErr *apperrors.AppError) {
	statusCode := http.StatusInternalServerError
	errorCode := apperrors.ErrorCodeInternal
	message := "系统内部错误"
	if appErr != nil {
		statusCode = appErr.StatusCode
		errorCode = appErr.Code
		message = appErr.Message
	}

	response.WriteError(w, statusCode, RequestID(r.Context()), string(errorCode), message)
}
