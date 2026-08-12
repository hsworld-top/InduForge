package node

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/auth"
	platformapi "github.com/indu-forge/dev_core/internal/platform/api"
)

const timeFormat = "2006-01-02 15:04:05"

type Handler struct {
	service     *Service
	authService *auth.Service
}

func NewHandler(service *Service, authService *auth.Service) *Handler {
	return &Handler{service: service, authService: authService}
}

func (h *Handler) MountAgentRoutes(router chi.Router) {
	router.Post("/api/v1/node-register/register-with-auth", h.RegisterWithAuth)
	router.Get("/api/v1/node-register/{nodeID}/approval-status", func(w http.ResponseWriter, r *http.Request) {
		h.GetApprovalStatus(w, r, chi.URLParam(r, "nodeID"))
	})
	router.Post("/api/v1/nodes/{nodeID}/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		h.Heartbeat(w, r, chi.URLParam(r, "nodeID"))
	})
	router.Post("/api/v1/nodes/{nodeID}/offline", func(w http.ResponseWriter, r *http.Request) {
		h.Offline(w, r, chi.URLParam(r, "nodeID"))
	})
	router.Post("/api/v1/nodes/{nodeID}/deployment-status", func(w http.ResponseWriter, r *http.Request) {
		h.ReportDeploymentStatus(w, r, chi.URLParam(r, "nodeID"))
	})
}

func (h *Handler) RegisterWithAuth(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username        string `json:"username"`
		Password        string `json:"password"`
		TenantCode      string `json:"tenantCode"`
		NodeID          string `json:"nodeId"`
		NodeName        string `json:"nodeName"`
		NodeDescription string `json:"nodeDescription"`
		AgentVersion    string `json:"agentVersion"`
		IPAddress       string `json:"ipAddress"`
		Port            int    `json:"port"`
		Mode            string `json:"mode"`
	}
	if err := decodeBody(r, &body); err != nil {
		h.invalid(w, r, err)
		return
	}
	actor, err := h.authService.AuthenticatePassword(r.Context(), body.Username, body.Password, body.TenantCode)
	if err != nil {
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidCredentials, "用户名或密码错误")
		return
	}
	result, err := h.service.Register(r.Context(), actor, RegisterInput{
		NodeID: body.NodeID, Name: body.NodeName, Description: body.NodeDescription, AgentVersion: body.AgentVersion,
		IPAddress: body.IPAddress, Port: body.Port, Mode: body.Mode, UserAgent: r.UserAgent(),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{
		"nodeId": result.NodeID, "nodeName": result.NodeName, "approvalStatus": result.ApprovalStatus,
		"registrationToken": result.RegistrationToken, "autoApproved": result.AutoApproved,
	})
}

func (h *Handler) GetApprovalStatus(w http.ResponseWriter, r *http.Request, nodeID string) {
	item, err := h.service.ApprovalStatus(r.Context(), nodeID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{
		"nodeId": item.NodeID, "nodeName": item.NodeName, "status": item.Status, "approvalStatus": item.ApprovalStatus,
		"approvedAt": formattedTime(item.ApprovedAt), "rejectedAt": formattedTime(item.RejectedAt), "updatedAt": item.UpdatedAt.Format(timeFormat),
	})
}

func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request, nodeID string) {
	var body struct {
		RegistrationToken string         `json:"registrationToken"`
		AgentVersion      string         `json:"agentVersion"`
		Metrics           map[string]any `json:"metrics"`
	}
	if err := decodeBody(r, &body); err != nil {
		h.invalid(w, r, err)
		return
	}
	token := strings.TrimSpace(r.Header.Get("X-Registration-Token"))
	if token == "" {
		token = body.RegistrationToken
	}
	item, commands, err := h.service.Heartbeat(r.Context(), nodeID, HeartbeatInput{RegistrationToken: token, AgentVersion: body.AgentVersion, Metrics: body.Metrics})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	pending := make([]map[string]any, 0, len(commands))
	for _, command := range commands {
		pending = append(pending, map[string]any{"type": command.Type, "payload": command.Payload})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true, "code": 0, "msg": "success", "reqId": platformapi.RequestIDFromContext(r.Context()),
		"data": map[string]any{"status": item.Status, "commands": pending},
	})
}

func (h *Handler) Offline(w http.ResponseWriter, r *http.Request, nodeID string) {
	var body struct {
		RegistrationToken string `json:"registrationToken"`
		Reason            string `json:"reason"`
	}
	if err := decodeBody(r, &body); err != nil && !errors.Is(err, io.EOF) {
		h.invalid(w, r, err)
		return
	}
	token := strings.TrimSpace(r.Header.Get("X-Registration-Token"))
	if token == "" {
		token = body.RegistrationToken
	}
	item, err := h.service.Offline(r.Context(), nodeID, token, body.Reason)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, nodePayload(item))
}

func (h *Handler) ReportDeploymentStatus(w http.ResponseWriter, r *http.Request, nodeID string) {
	var body struct {
		RegistrationToken string         `json:"registrationToken"`
		DeploymentID      string         `json:"deploymentId"`
		Status            string         `json:"status"`
		Error             map[string]any `json:"error"`
		Message           string         `json:"message"`
		StartedAt         *time.Time     `json:"startedAt"`
		StoppedAt         *time.Time     `json:"stoppedAt"`
	}
	if err := decodeBody(r, &body); err != nil {
		h.invalid(w, r, err)
		return
	}
	token := strings.TrimSpace(r.Header.Get("X-Registration-Token"))
	if token == "" {
		token = body.RegistrationToken
	}
	errorMessage, _ := body.Error["message"].(string)
	item, err := h.service.ReportDeployment(r.Context(), nodeID, DeploymentReport{
		RegistrationToken: token, DeploymentID: body.DeploymentID, Status: body.Status,
		ErrorMessage: errorMessage, Message: body.Message, StartedAt: body.StartedAt, StoppedAt: body.StoppedAt,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{"id": item.ID, "status": item.Status})
}

func (h *Handler) ListNodes(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	query := r.URL.Query()
	page := positiveInt(query.Get("page"), 1)
	limit := positiveInt(query.Get("pageSize"), 20)
	items, total, err := h.service.List(r.Context(), actor, ListFilter{
		Page: page, Limit: limit, Status: query.Get("status"), ApprovalStatus: query.Get("approvalStatus"), Keyword: query.Get("search"),
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, nodePayload(item))
	}
	platformapi.WriteSuccess(w, r, map[string]any{"items": list, "list": list, "total": total, "page": page, "pageSize": limit})
}

func (h *Handler) ApproveNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	item, err := h.service.Approve(r.Context(), actor, nodeID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, nodePayload(item))
}

func (h *Handler) RejectNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	item, err := h.service.Reject(r.Context(), actor, nodeID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, nodePayload(item))
}

func (h *Handler) DeleteNode(w http.ResponseWriter, r *http.Request, nodeID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), actor, nodeID); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{})
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
	if errors.Is(err, auth.ErrPermissionDenied) {
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodePermissionDenied, err.Error())
		return
	}
	if errors.Is(err, ErrNotFound) {
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeNotFound, err.Error())
		return
	}
	if strings.Contains(err.Error(), "不能为空") {
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
		return
	}
	platformapi.WriteError(w, r, http.StatusInternalServerError, platformapi.ErrorCodeInternal, "系统内部错误")
}

func (h *Handler) invalid(w http.ResponseWriter, r *http.Request, err error) {
	platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
}

func nodePayload(item Node) map[string]any {
	payload := map[string]any{
		"id": item.ID, "tenantId": item.TenantID, "name": item.Name, "code": item.Code,
		"nodeType": item.NodeType, "status": item.Status, "approvalStatus": item.ApprovalStatus,
		"agentVersion": item.AgentVersion, "os": item.OS, "architecture": item.Architecture,
		"hostname": item.Hostname, "ipAddress": item.IPAddress, "capabilities": item.Capabilities,
		"metrics": item.Metrics, "metadata": item.Metadata, "deployments": item.Deployments,
		"createdAt": item.CreatedAt.Format(timeFormat), "updatedAt": item.UpdatedAt.Format(timeFormat),
	}
	payload["lastHeartbeatAt"] = formattedTime(item.LastHeartbeatAt)
	payload["approvedAt"] = formattedTime(item.ApprovedAt)
	payload["rejectedAt"] = formattedTime(item.RejectedAt)
	if port, ok := item.Metadata["port"]; ok {
		payload["port"] = port
	}
	if registrant, ok := item.Metadata["registrant"]; ok {
		payload["registrant"] = registrant
	}
	return payload
}

func formattedTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.Format(timeFormat)
}

func positiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func decodeBody(r *http.Request, target any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(target)
}
