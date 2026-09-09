package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/indu-forge/data_service/internal/service"
)

type AuthoringFenceChecker interface {
	GateNormalWrite(context.Context, string, string, string) (func(), error)
	ResolveQueryProject(context.Context, string, string) (string, error)
}

// AuthoringFenceGuard 在统一路由入口阻止所有普通工程配置写入，避免遗漏某个具体 handler。
// 内部恢复接口不经过该前缀，但必须在服务层校验 fence token。
func AuthoringFenceGuard(validator *auth.JWTValidator, checker AuthoringFenceChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if checker == nil || validator == nil || !isAuthoringMutation(r) {
				next.ServeHTTP(w, r)
				return
			}
			token, ok := tokenFromRequest(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			claims, err := validator.ValidateContext(r.Context(), token)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			projectID := projectIDFromDataPath(r.URL.Path)
			if projectID == "" {
				if queryID := queryIDFromMutationPath(r); queryID != "" {
					projectID, err = checker.ResolveQueryProject(r.Context(), queryID, claims.TenantID)
					if err != nil {
						response.WriteAppError(w, http.StatusConflict, RequestID(r.Context()), apperrors.ErrorCodeBadRequest, "无法确定当前工程，请刷新后重试")
						return
					}
				}
			}
			if projectID == "" && len(claims.ProjectIDs) == 1 {
				projectID = claims.ProjectIDs[0]
			}
			if projectID == "" {
				response.WriteAppError(w, http.StatusConflict, RequestID(r.Context()), apperrors.ErrorCodeBadRequest, "无法确定当前工程，请重新进入工程后重试")
				return
			}
			release, err := checker.GateNormalWrite(r.Context(), projectID, claims.TenantID, r.Header.Get("X-InduForge-Authoring-Epoch"))
			if err != nil {
				var epochConflict *repository.AuthoringEpochConflict
				if errors.As(err, &epochConflict) {
					response.WriteAppErrorData(w, http.StatusConflict, RequestID(r.Context()), apperrors.ErrorCodeBadRequest, epochConflict.Error(), map[string]any{"projectId": epochConflict.ProjectID, "currentAuthoringEpoch": service.FormatAuthoringEpoch(epochConflict.Current), "action": "reload"})
					return
				}
				var appErr *apperrors.AppError
				if AsAppError(err, &appErr) {
					response.WriteAppError(w, appErr.StatusCode, RequestID(r.Context()), appErr.Code, appErr.Message)
				} else {
					response.WriteAppError(w, http.StatusInternalServerError, RequestID(r.Context()), apperrors.ErrorCodeInternal, "检查工程开发内容写保护失败")
				}
				return
			}
			defer release()
			next.ServeHTTP(w, r)
		})
	}
}

func isAuthoringMutation(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		if strings.HasPrefix(r.URL.Path, "/api/v1/data/projects/") {
			return true
		}
		return queryIDFromMutationPath(r) != ""
	default:
		return false
	}
}

func queryIDFromMutationPath(r *http.Request) string {
	if r.Method != http.MethodPut && r.Method != http.MethodDelete {
		return ""
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/v1/data/queries/")
	if rest == r.URL.Path || rest == "" || strings.Contains(rest, "/") {
		return ""
	}
	return strings.TrimSpace(rest)
}
func projectIDFromDataPath(path string) string {
	rest := strings.TrimPrefix(path, "/api/v1/data/projects/")
	if rest == path {
		return ""
	}
	id, _, _ := strings.Cut(rest, "/")
	return strings.TrimSpace(id)
}
