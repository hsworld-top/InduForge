package handler

import (
	"net/http"

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
	result, err := h.service.ListAgents(r.Context(), claims)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
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
	result, err := h.service.Register(r.Context(), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
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
	result, err := h.service.Heartbeat(r.Context(), identity, input.Capabilities)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
func (h *CollectorDevHandler) ClaimTask(w http.ResponseWriter, r *http.Request) error {
	identity, err := requireCollectorAgent(r)
	if err != nil {
		return err
	}
	result, err := h.service.ClaimTask(r.Context(), identity)
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
