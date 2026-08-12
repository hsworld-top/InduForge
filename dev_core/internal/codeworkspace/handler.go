package codeworkspace

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/project"
)

type Handler struct {
	service     *Service
	authService *auth.Service
}

func NewHandler(service *Service, authService *auth.Service) *Handler {
	return &Handler{service: service, authService: authService}
}

func (h *Handler) GetCodeWorkspace(w http.ResponseWriter, r *http.Request, projectID string) {
	h.execute(w, r, projectID, h.service.Status)
}

func (h *Handler) StartCodeWorkspace(w http.ResponseWriter, r *http.Request, projectID string) {
	h.execute(w, r, projectID, h.service.Start)
}

func (h *Handler) StopCodeWorkspace(w http.ResponseWriter, r *http.Request, projectID string) {
	h.execute(w, r, projectID, h.service.Stop)
}

func (h *Handler) RebuildCodeWorkspace(w http.ResponseWriter, r *http.Request, projectID string) {
	h.execute(w, r, projectID, h.service.Rebuild)
}

func (h *Handler) execute(w http.ResponseWriter, r *http.Request, projectID string, operation func(context.Context, auth.User, string) (Status, error)) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	status, err := operation(r.Context(), actor, projectID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	var hostPort any
	if parsedPort, parseErr := strconv.Atoi(status.HostPort); parseErr == nil {
		hostPort = parsedPort
	}
	payload := map[string]any{"containerName": status.ContainerName, "status": status.Status, "hostPort": hostPort, "onlineUsers": status.OnlineUsers}
	if status.Status == "running" && status.HostPort != "" {
		payload["url"] = codeServerURL(r, status.HostPort)
	}
	platformapi.WriteSuccess(w, r, payload)
}

func codeServerURL(r *http.Request, port string) string {
	host := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0])
	if host == "" {
		host = r.Host
	}
	hostname := host
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		hostname = parsed
	}
	scheme := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])
	if scheme == "" {
		scheme = "http"
	}
	return (&url.URL{Scheme: scheme, Host: net.JoinHostPort(strings.Trim(hostname, "[]"), port), Path: "/"}).String()
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
	case errors.Is(err, project.ErrNotFound):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeProjectNotFound, err.Error())
	case errors.Is(err, auth.ErrPermissionDenied):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodePermissionDenied, err.Error())
	case errors.Is(err, ErrContainerConflict):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
	default:
		platformapi.WriteError(w, r, http.StatusInternalServerError, platformapi.ErrorCodeInternal, "代码工作区操作失败")
	}
}
