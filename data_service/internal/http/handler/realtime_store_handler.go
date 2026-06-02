package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

type RealtimeStoreHandler struct {
	service *service.RealtimeStoreService
}

func NewRealtimeStoreHandler(service *service.RealtimeStoreService) *RealtimeStoreHandler {
	return &RealtimeStoreHandler{service: service}
}

func (h *RealtimeStoreHandler) ListKeys(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	result, err := h.service.ListKeys(
		r.Context(),
		r.PathValue("projectId"),
		r.PathValue("connectionId"),
		r.URL.Query().Get("q"),
		r.URL.Query().Get("group"),
		limit,
	)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *RealtimeStoreHandler) GetKey(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.GetKey(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), realtimeKeyFromRequest(r))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *RealtimeStoreHandler) SaveKey(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var request service.SaveRealtimeStoreKeyInput
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	if request.Key == "" {
		request.Key = realtimeKeyFromRequest(r)
	}
	result, err := h.service.SaveKey(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), request)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *RealtimeStoreHandler) RenameKey(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var request service.RenameRealtimeStoreKeyInput
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.RenameKey(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), realtimeKeyFromRequest(r), request)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *RealtimeStoreHandler) DeleteKey(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.DeleteKey(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), realtimeKeyFromRequest(r), claims.UserID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *RealtimeStoreHandler) CreateDataPoint(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request service.CreateRealtimeKeyDataPointInput
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.CreateDataPoint(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), realtimeKeyFromRequest(r), claims.UserID, request)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *RealtimeStoreHandler) BatchCreateDataPoints(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request service.BatchCreateRealtimeKeyDataPointInput
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.BatchCreateDataPoints(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, request)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func realtimeKeyFromRequest(r *http.Request) string {
	return strings.TrimSpace(r.URL.Query().Get("key"))
}
