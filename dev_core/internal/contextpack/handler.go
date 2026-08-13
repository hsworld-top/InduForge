package contextpack

import (
	"context"
	"net/http"

	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
)

type ContextSynchronizer interface {
	Sync(context.Context, auth.User, string, string) (map[string]any, error)
}

type Handler struct{ service ContextSynchronizer }

func NewHandler(service ContextSynchronizer) *Handler { return &Handler{service: service} }

func (h *Handler) RefreshWorkspaceContext(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := auth.UserFromContext(r.Context())
	if !ok {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenRequired, "缺少访问令牌")
		return
	}
	result, err := h.service.Sync(r.Context(), actor, projectID, auth.ForwardAuthorization(r))
	if err != nil {
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
		return
	}
	platformapi.WriteSuccess(w, r, result)
}
