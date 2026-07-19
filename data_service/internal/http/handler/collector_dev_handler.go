package handler

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

type CollectorDevHandler struct{ service *service.CollectorDevService }

func NewCollectorDevHandler(collectorService *service.CollectorDevService) *CollectorDevHandler {
	return &CollectorDevHandler{service: collectorService}
}

func (h *CollectorDevHandler) CreateRegistrationCode(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.CreateRegistrationCode(r.Context(), claims)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
func (h *CollectorDevHandler) ListAgents(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	page, err := parseOptionalInt(r.URL.Query().Get("page"), 1, "page")
	if err != nil {
		return err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(r.URL.Query().Get("pageSize"), r.URL.Query().Get("limit")), 10, "pageSize")
	if err != nil {
		return err
	}
	if pageSize > 100 {
		pageSize = 100
	}
	result, total, err := h.service.ListAgentsPage(r.Context(), claims, page, pageSize)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
		"list":       result,
		"pagination": map[string]int{"page": page, "pageSize": pageSize, "total": total, "totalPages": totalPages},
	})
	return nil
}
func (h *CollectorDevHandler) DeleteAgent(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err = h.service.DeleteAgent(r.Context(), claims, r.PathValue("agentId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}
func (h *CollectorDevHandler) RegisterAgent(w http.ResponseWriter, r *http.Request) error {
	var input service.CollectorRegisterInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.Register(r.Context(), input, requestClientIP(r))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
func (h *CollectorDevHandler) Disconnect(w http.ResponseWriter, r *http.Request) error {
	identity, err := requireCollectorAgent(r)
	if err != nil {
		return err
	}
	if err = h.service.Disconnect(r.Context(), identity); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"disconnected": true})
	return nil
}

func (h *CollectorDevHandler) Revoke(w http.ResponseWriter, r *http.Request) error {
	identity, err := requireCollectorAgent(r)
	if err != nil {
		return err
	}
	if err = h.service.Revoke(r.Context(), identity); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"revoked": true})
	return nil
}

func (h *CollectorDevHandler) Heartbeat(w http.ResponseWriter, r *http.Request) error {
	identity, err := requireCollectorAgent(r)
	if err != nil {
		return err
	}
	var input struct {
		Capabilities []service.CollectorProtocolCapability `json:"capabilities"`
	}
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.Heartbeat(r.Context(), identity, input.Capabilities, requestClientIP(r))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func requestClientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
func (h *CollectorDevHandler) ClaimTask(w http.ResponseWriter, r *http.Request) error {
	identity, err := requireCollectorAgent(r)
	if err != nil {
		return err
	}
	waitSeconds, err := parseOptionalInt(r.URL.Query().Get("waitSeconds"), 0, "waitSeconds")
	if err != nil {
		return err
	}
	if waitSeconds < 0 || waitSeconds > 25 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "waitSeconds 必须在 0 到 25 之间")
	}
	result, err := h.service.ClaimTask(r.Context(), identity, time.Duration(waitSeconds)*time.Second)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
func (h *CollectorDevHandler) CompleteTask(w http.ResponseWriter, r *http.Request) error {
	identity, err := requireCollectorAgent(r)
	if err != nil {
		return err
	}
	var input service.CollectorTaskCompletion
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.CompleteTask(r.Context(), identity, r.PathValue("taskId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
func (h *CollectorDevHandler) CreateTask(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.CollectorTaskInput
	if err = decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.CreateTask(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
func (h *CollectorDevHandler) GetTask(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.GetTask(r.Context(), claims, r.PathValue("projectId"), r.PathValue("taskId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
func (h *CollectorDevHandler) CancelTask(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.CancelTask(r.Context(), claims, r.PathValue("projectId"), r.PathValue("taskId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
func requireCollectorAgent(r *http.Request) (*auth.CollectorAgentIdentity, error) {
	identity, ok := auth.CollectorAgentFromContext(r.Context())
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成 Agent 认证")
	}
	return identity, nil
}
