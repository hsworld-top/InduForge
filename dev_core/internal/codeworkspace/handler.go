package codeworkspace

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
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
	dataProxy   func(context.Context, string, string, string, []byte) ([]byte, int, error)
}

func NewHandler(service *Service, authService *auth.Service) *Handler {
	return &Handler{service: service, authService: authService, mcpTokens: map[string]mcpToken{}}
}

func (h *Handler) SetGateway(gateway *Gateway) { h.gateway = gateway }

// SetMCPDataProxy 注入中心到数据服务的受控代理。工作区只持有随机 MCP 令牌，
// 不能自行伪造数据服务 JWT 或工程身份。
func (h *Handler) SetMCPDataProxy(proxy func(context.Context, string, string, string, []byte) ([]byte, int, error)) {
	h.dataProxy = proxy
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
	platformapi.WriteSuccess(w, r, map[string]any{"endpoint": endpoint, "token": token, "expiresAt": expires.UTC().Format(time.RFC3339), "projectId": projectID, "tools": []string{
		"workspace_status", "workspace_list_files", "workspace_read_file", "workspace_file_hash", "workspace_snapshot", "workspace_diff", "workspace_write_file", "workspace_write_batch", "project_context", "datacenter_context", "workspace_exec",
		"source_list", "source_get", "source_health", "source_capabilities", "collector_protocol_list", "collector_list", "collector_get", "collector_point_list", "collector_point_preview",
		"query_list", "query_get", "query_execute", "query_create", "query_update", "query_delete",
		"datapoint_list", "datapoint_get", "datapoint_read", "datapoint_history", "datapoint_subscribe", "datapoint_create", "datapoint_update", "datapoint_delete",
		"compute_list", "compute_get", "compute_validate", "compute_run", "compute_create", "compute_update", "compute_delete", "compute_output_datapoints",
		"alarm_list", "alarm_get", "alarm_datapoint_summary", "alarm_create", "alarm_update", "alarm_delete", "alarm_enable", "alarm_disable", "alarm_test",
	}})
}

// ProxyMCPData 为工作区 MCP 数据工具提供中心侧受控转发。
// 只接受 /api/v1/data 下的白名单路径，并强制令牌工程与路径工程一致。
func (h *Handler) ProxyMCPData(w http.ResponseWriter, r *http.Request) {
	if h.dataProxy == nil {
		platformapi.WriteError(w, r, http.StatusServiceUnavailable, platformapi.ErrorCodeInternal, "数据中心工具代理未配置")
		return
	}
	token := strings.TrimSpace(r.Header.Get("X-InduForge-MCP-Token"))
	if token == "" {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenRequired, "缺少 MCP 连接令牌")
		return
	}
	actor, tokenProject, _, _, ok := h.ResolveMCPToken(r.Context(), token)
	if !ok {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenInvalid, "MCP 连接令牌已失效")
		return
	}
	projectID := chi.URLParam(r, "projectId")
	if projectID == "" || projectID != tokenProject {
		platformapi.WriteError(w, r, http.StatusForbidden, platformapi.ErrorCodePermissionDenied, "MCP 连接令牌与工程不匹配")
		return
	}
	var input struct {
		Method string          `json:"method"`
		Path   string          `json:"path"`
		Body   json.RawMessage `json:"body"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&input); err != nil {
		platformapi.WriteError(w, r, http.StatusBadRequest, platformapi.ErrorCodeInvalidRequest, "数据中心工具请求格式无效")
		return
	}
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	parsed, err := url.Parse(strings.TrimSpace(input.Path))
	if err != nil || parsed.Path == "" || !mcpDataPathAllowed(method, parsed.Path, projectID) {
		platformapi.WriteError(w, r, http.StatusForbidden, platformapi.ErrorCodePermissionDenied, "数据中心工具路径或操作不允许")
		return
	}
	pathWithQuery := parsed.Path
	if parsed.RawQuery != "" {
		pathWithQuery += "?" + parsed.RawQuery
	}
	capabilities := []string{"project:read"}
	if method != http.MethodGet {
		capabilities = append(capabilities, "project:write")
	}
	bearer, _, err := h.authService.IssueScopedAccessToken(actor, projectID, capabilities)
	if err != nil {
		platformapi.WriteError(w, r, http.StatusUnauthorized, platformapi.ErrorCodeTokenInvalid, "无法建立数据中心工具会话")
		return
	}
	raw, status, err := h.dataProxy(r.Context(), method, pathWithQuery, bearer, input.Body)
	if err != nil {
		platformapi.WriteError(w, r, statusOr(status, http.StatusBadGateway), platformapi.ErrorCodeInternal, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func statusOr(value, fallback int) int {
	if value >= 100 && value <= 599 {
		return value
	}
	return fallback
}

func mcpDataPathAllowed(method, path, projectID string) bool {
	if !strings.HasPrefix(path, "/api/v1/data/") || strings.Contains(path, "..") {
		return false
	}
	projectPrefix := "/api/v1/data/projects/" + projectID
	if strings.HasPrefix(path, projectPrefix+"/") {
		relative := strings.TrimPrefix(path, projectPrefix+"/")
		if strings.HasPrefix(relative, "connections") || strings.HasPrefix(relative, "collector/") {
			return method == http.MethodGet || strings.HasSuffix(relative, "/test") || strings.HasSuffix(relative, "/points/check-addresses")
		}
		if strings.HasPrefix(relative, "queries") || strings.HasPrefix(relative, "datapoints") || strings.HasPrefix(relative, "compute-units") || strings.HasPrefix(relative, "alarm-items") || strings.Contains(relative, "/alarms") || strings.HasPrefix(relative, "history-storage/") {
			return method == http.MethodGet || method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
		}
		return false
	}
	return path == "/api/v1/data/collector/drivers" && method == http.MethodGet
}
