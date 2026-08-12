package scenecontract

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/project"
)

type Handler struct {
	service     *Service
	authService *auth.Service
	contextSync ContextSynchronizer
}

type ContextSynchronizer interface {
	Sync(context.Context, auth.User, string, string) (map[string]any, error)
}

func NewHandler(service *Service, authService *auth.Service) *Handler {
	return &Handler{service: service, authService: authService}
}

func (h *Handler) SetContextSynchronizer(sync ContextSynchronizer) { h.contextSync = sync }

func (h *Handler) ListSceneContracts(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	result, err := h.service.List(r.Context(), actor, projectID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, result)
}

func (h *Handler) GetSceneContract(w http.ResponseWriter, r *http.Request, projectID, kind, sceneID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	result, err := h.service.Get(r.Context(), actor, projectID, kind, sceneID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, result)
}

func (h *Handler) PutSceneContract(w http.ResponseWriter, r *http.Request, projectID, kind, sceneID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var input Contract
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&input); err != nil {
		h.writeError(w, r, ErrInvalidContract)
		return
	}
	result, err := h.service.Put(r.Context(), actor, projectID, kind, sceneID, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"contract": result, "contextSync": h.syncContext(r, actor, projectID)})
}

func (h *Handler) DeleteSceneContract(w http.ResponseWriter, r *http.Request, projectID, kind, sceneID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), actor, projectID, kind, sceneID); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"deleted": true, "contextSync": h.syncContext(r, actor, projectID)})
}

// syncContext 不回滚已持久化的契约。上下文是可再生只读镜像，失败时返回过期状态供前端提示。
func (h *Handler) syncContext(r *http.Request, actor auth.User, projectID string) map[string]any {
	if h.contextSync == nil {
		return map[string]any{"status": "not-configured"}
	}
	result, err := h.contextSync.Sync(r.Context(), actor, projectID, r.Header.Get("Authorization"))
	if err != nil {
		return map[string]any{"status": "stale", "message": err.Error()}
	}
	return map[string]any{"status": "updated", "result": result}
}

func (h *Handler) requireUser(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	if actor, ok := auth.UserFromContext(r.Context()); ok {
		return actor, true
	}
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenRequired, "缺少访问令牌")
		return auth.User{}, false
	}
	actor, err := h.authService.Authenticate(r.Context(), parts[1])
	if err != nil {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenInvalid, "访问令牌无效或已过期")
		return auth.User{}, false
	}
	return actor, true
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, project.ErrNotFound), errors.Is(err, ErrNotFound):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeNotFound, err.Error())
	case errors.Is(err, auth.ErrPermissionDenied):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodePermissionDenied, err.Error())
	case errors.Is(err, ErrInvalidContract):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
	default:
		platformapi.WriteError(w, r, http.StatusInternalServerError, platformapi.ErrorCodeInternal, "系统内部错误")
	}
}
