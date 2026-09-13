package codeworkspace

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
	"github.com/indu-forge/dev_core/internal/project"
)

type Handler struct {
	service     *Service
	authService *auth.Service
	gateway     *Gateway
	mcpMu       sync.Mutex
	mcpTokens   map[string]mcpToken
}

func NewHandler(service *Service, authService *auth.Service) *Handler {
	return &Handler{service: service, authService: authService, mcpTokens: map[string]mcpToken{}}
}

func (h *Handler) SetGateway(gateway *Gateway) { h.gateway = gateway }

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
	onlineUsers := status.OnlineUsers
	if onlineUsers == nil {
		onlineUsers = []OnlineUser{}
	}
	services, err := h.workspaceServices(r, actor, projectID, status)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	payload := map[string]any{"containerName": status.ContainerName, "status": status.Status, "hostPort": hostPort, "onlineUsers": onlineUsers, "services": services}
	if h.gateway != nil {
		w.Header().Set("Cache-Control", "no-store")
	}
	if h.gateway == nil && status.Status == "running" && status.HostPort != "" {
		payload["url"] = codeServerURL(r, status.HostPort)
	}
	platformapi.WriteSuccess(w, r, payload)
}

// workspaceServices 始终返回前端所需的服务，避免工作区因缺字段被当作断连。
func (h *Handler) workspaceServices(r *http.Request, actor auth.User, projectID string, status Status) (map[string]any, error) {
	if h.gateway != nil {
		result := map[string]any{}
		epoch, err := h.service.AuthoringEpoch(r.Context(), actor, projectID)
		if err != nil {
			return nil, err
		}
		for responseName, serviceName := range map[string]string{"ai": "ai", "code": "code", "preview": "preview", "previewControl": "preview-control"} {
			var endpoint any
			if status.Status == "running" {
				value, issueErr := h.gateway.PublicURL(actor, projectID, epoch, serviceName)
				if issueErr != nil {
					return nil, issueErr
				}
				endpoint = value
			}
			result[responseName] = map[string]any{"url": endpoint, "hostPort": nil}
		}
		var mcpURL any
		if status.Status == "running" {
			value, mcpErr := h.gateway.PublicURLWithoutTicket(projectID, "mcp")
			if mcpErr != nil {
				return nil, mcpErr
			}
			mcpURL = value
		}
		result["mcp"] = map[string]any{"url": mcpURL, "hostPort": nil}
		return result, nil
	}
	port := func(name string) string {
		if status.ServicePorts != nil && status.ServicePorts[name] != "" {
			return status.ServicePorts[name]
		}
		if name == "code" {
			return status.HostPort
		}
		return ""
	}
	service := func(name string) map[string]any {
		value := port(name)
		var hostPort any
		if parsed, err := strconv.Atoi(value); err == nil {
			hostPort = parsed
		}
		var endpoint any
		if status.Status == "running" && value != "" {
			endpoint = codeServerURL(r, value)
		}
		return map[string]any{"url": endpoint, "hostPort": hostPort}
	}
	return map[string]any{"ai": service("ai"), "code": service("code"), "preview": service("preview"), "previewControl": service("preview-control"), "mcp": service("mcp")}, nil
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
	if data, ok := project.AuthoringEpochConflictData(err); ok {
		platformapi.WriteErrorData(w, r, http.StatusConflict, platformapi.ErrorCodeAlreadyExists, err.Error(), data)
		return
	}
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

type mcpToken struct {
	User             auth.User
	ProjectID, Epoch string
	ExpiresAt        time.Time
}

// IssueMCPToken 为当前用户和工作区签发开发态 MCP 令牌。令牌只绑定当前工程和 authoring epoch。
func (h *Handler) IssueMCPToken(user auth.User, projectID, epoch string, ttl time.Duration) (string, time.Time, error) {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", time.Time{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	exp := time.Now().Add(ttl)
	h.mcpMu.Lock()
	h.mcpTokens[token] = mcpToken{User: user, ProjectID: projectID, Epoch: epoch, ExpiresAt: exp}
	h.mcpMu.Unlock()
	return token, exp, nil
}

func (h *Handler) ValidateMCPToken(token string, userID, projectID, epoch string) bool {
	h.mcpMu.Lock()
	defer h.mcpMu.Unlock()
	v, ok := h.mcpTokens[token]
	if !ok || time.Now().After(v.ExpiresAt) {
		delete(h.mcpTokens, token)
		return false
	}
	return v.User.ID == userID && v.ProjectID == projectID && v.Epoch == epoch
}

func (h *Handler) ResolveMCPToken(_ context.Context, token string) (auth.User, string, string, time.Time, bool) {
	h.mcpMu.Lock()
	defer h.mcpMu.Unlock()
	v, ok := h.mcpTokens[token]
	if !ok || time.Now().After(v.ExpiresAt) {
		delete(h.mcpTokens, token)
		return auth.User{}, "", "", time.Time{}, false
	}
	return v.User, v.ProjectID, v.Epoch, v.ExpiresAt, true
}

// CreateCodeWorkspaceMCPToken creates a development token bound to the current user, project and authoring epoch.
func (h *Handler) CreateCodeWorkspaceMCPToken(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	epoch, err := h.service.AuthoringEpoch(r.Context(), actor, projectID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	token, expires, err := h.IssueMCPToken(actor, projectID, epoch, 24*time.Hour)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	endpoint := any(nil)
	if h.gateway != nil {
		endpoint, err = h.gateway.PublicURLWithoutTicket(projectID, "mcp")
		if err != nil {
			h.writeError(w, r, err)
			return
		}
	}
	platformapi.WriteSuccess(w, r, map[string]any{"endpoint": endpoint, "token": token, "expiresAt": expires.UTC().Format(time.RFC3339), "projectId": projectID, "tools": []string{"workspace_status", "workspace_list_files", "workspace_read_file", "project_context", "workspace_exec"}})
}
