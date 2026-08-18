package deployment

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

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

func (h *Handler) ListProjectVersions(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	page := positiveInt(r.URL.Query().Get("page"), 1)
	limit := positiveInt(r.URL.Query().Get("pageSize"), 20)
	items, total, err := h.service.ListVersions(r.Context(), actor, projectID, page, limit)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, versionPayload(item))
	}
	platformapi.WriteSuccess(w, r, map[string]any{"items": list, "list": list, "total": total, "page": page, "pageSize": limit})
}

func (h *Handler) PublishProjectVersion(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var input PublishInput
	if err := decodeJSON(r, &input); err != nil {
		h.invalid(w, r, err)
		return
	}
	item, err := h.service.Publish(r.Context(), actor, projectID, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, versionPayload(item))
}

func (h *Handler) DeletePublishedVersion(w http.ResponseWriter, r *http.Request, versionID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteVersion(r.Context(), actor, versionID); err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, map[string]any{})
}

func (h *Handler) ListProjectDeploymentNodes(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListProjectDeployments(r.Context(), actor, projectID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, deploymentPayload(item))
	}
	platformapi.WriteSuccess(w, r, list)
}

func (h *Handler) DeployPublishedVersion(w http.ResponseWriter, r *http.Request, versionID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		NodeIDs       []string       `json:"nodeIds"`
		RuntimeConfig map[string]any `json:"runtimeConfig"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.invalid(w, r, err)
		return
	}
	items, err := h.service.DeployVersion(r.Context(), actor, versionID, body.NodeIDs, body.RuntimeConfig)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeDeploymentBatch(w, r, items)
}

func (h *Handler) DeployProjectDevelopment(w http.ResponseWriter, r *http.Request, projectID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		NodeIDs       []string       `json:"nodeIds"`
		RuntimeConfig map[string]any `json:"runtimeConfig"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.invalid(w, r, err)
		return
	}
	items, err := h.service.DeployDevelopment(r.Context(), actor, projectID, body.NodeIDs, body.RuntimeConfig)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeDeploymentBatch(w, r, items)
}

func (h *Handler) StartNodeDeployment(w http.ResponseWriter, r *http.Request, deploymentID string) {
	h.operate(w, r, deploymentID, "start")
}

func (h *Handler) StopNodeDeployment(w http.ResponseWriter, r *http.Request, deploymentID string) {
	h.operate(w, r, deploymentID, "stop")
}

func (h *Handler) RestartNodeDeployment(w http.ResponseWriter, r *http.Request, deploymentID string) {
	h.operate(w, r, deploymentID, "restart")
}

func (h *Handler) DeleteNodeDeployment(w http.ResponseWriter, r *http.Request, deploymentID string) {
	h.operate(w, r, deploymentID, "remove")
}

func (h *Handler) RollbackDeployment(w http.ResponseWriter, r *http.Request, versionID string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		NodeID string `json:"nodeId"`
	}
	if err := decodeJSON(r, &body); err != nil {
		h.invalid(w, r, err)
		return
	}
	items, err := h.service.Rollback(r.Context(), actor, versionID, body.NodeID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeDeploymentBatch(w, r, items)
}

func (h *Handler) operate(w http.ResponseWriter, r *http.Request, deploymentID, operation string) {
	actor, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	item, err := h.service.Operate(r.Context(), actor, deploymentID, operation)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	platformapi.WriteSuccess(w, r, deploymentPayload(item))
}

func (h *Handler) writeDeploymentBatch(w http.ResponseWriter, r *http.Request, items []Deployment) {
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, deploymentPayload(item))
	}
	platformapi.WriteSuccess(w, r, map[string]any{"items": list, "summary": map[string]any{"success": len(list), "failed": 0, "total": len(list)}})
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
	case errors.Is(err, auth.ErrPermissionDenied):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodePermissionDenied, err.Error())
	case errors.Is(err, ErrNotFound):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeNotFound, err.Error())
	case errors.Is(err, ErrAlreadyExists):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeAlreadyExists, err.Error())
	case errors.Is(err, ErrVersionInUse):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
	case errors.Is(err, ErrScenesNotReady):
		platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
	default:
		if strings.Contains(err.Error(), "不能为空") || strings.Contains(err.Error(), "至少选择") || strings.Contains(err.Error(), "无效") || strings.Contains(err.Error(), "缺少") {
			h.invalid(w, r, err)
			return
		}
		platformapi.WriteError(w, r, http.StatusInternalServerError, platformapi.ErrorCodeInternal, "系统内部错误")
	}
}

func (h *Handler) invalid(w http.ResponseWriter, r *http.Request, err error) {
	platformapi.WriteError(w, r, http.StatusOK, platformapi.ErrorCodeInvalidRequest, err.Error())
}

func decodeJSON(r *http.Request, target any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(target)
}

func versionPayload(item Version) map[string]any {
	status := item.Status
	if status == "ready" {
		status = "success"
	}
	payload := map[string]any{
		"id": item.ID, "projectId": item.ProjectID, "version": item.Version, "name": item.Name,
		"description": item.Description, "status": status, "mode": "RELEASE", "sourceHash": item.SourceHash,
		"artifactHash": item.ArtifactHash, "artifactSize": item.ArtifactSize, "manifest": item.Manifest,
		"errorMessage": item.ErrorMessage, "createdAt": item.CreatedAt.Format(timeFormat), "updatedAt": item.UpdatedAt.Format(timeFormat),
	}
	if item.CompletedAt != nil {
		payload["completedAt"] = item.CompletedAt.Format(timeFormat)
	} else {
		payload["completedAt"] = nil
	}
	return payload
}

func deploymentPayload(item Deployment) map[string]any {
	mode := "RELEASE"
	if item.Mode == "development" {
		mode = "DEV"
	}
	return map[string]any{
		"id": item.ID, "nodeId": item.NodeID, "projectId": item.ProjectID,
		"deploymentId": item.ApplicationVersionID, "version": item.Version, "mode": mode, "status": item.Status,
		"runtimeConfig": item.RuntimeConfig, "runtimeMetrics": item.RuntimeMetrics, "errorMessage": item.ErrorMessage,
		"node":         map[string]any{"id": item.NodeID, "name": item.NodeName, "status": item.NodeStatus, "ipAddress": item.IPAddress},
		"project":      map[string]any{"id": item.ProjectID, "name": item.ProjectName, "code": item.ProjectCode},
		"artifactHash": item.ArtifactHash, "manifest": item.Manifest,
		"createdAt": item.CreatedAt.Format(timeFormat), "updatedAt": item.UpdatedAt.Format(timeFormat),
	}
}

func positiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
