package handler

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/indu-forge/data_service/internal/service"
)

type CollectorHandler struct{ service *service.CollectorService }

func NewCollectorHandler(collectorService *service.CollectorService) *CollectorHandler {
	return &CollectorHandler{service: collectorService}
}

func (h *CollectorHandler) ListConnections(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	query := r.URL.Query()
	page, err := parseOptionalInt(query.Get("page"), 1, "page")
	if err != nil {
		return err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(query.Get("pageSize"), query.Get("limit")), 20, "pageSize")
	if err != nil {
		return err
	}

	result, err := h.service.ListConnections(r.Context(), r.PathValue("projectId"), repository.CollectorConnectionListFilter{Page: page, PageSize: pageSize, Search: query.Get("search"), ProtocolFamily: query.Get("protocolFamily"), DriverID: query.Get("driverId"), SortBy: query.Get("sortBy"), SortOrder: query.Get("sortOrder")})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *CollectorHandler) CreateConnection(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.CreateCollectorConnectionInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.CreateConnection(r.Context(), r.PathValue("projectId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *CollectorHandler) GetConnection(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.GetConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *CollectorHandler) UpdateConnection(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := decodeJSONBody(r, &raw); err != nil {
		return err
	}
	allowed := map[string]struct{}{"name": {}, "config": {}, "metadata": {}, "secrets": {}}
	for key := range raw {
		if _, ok := allowed[key]; !ok {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "更新请求包含不支持的字段: "+key)
		}
	}
	input := service.UpdateCollectorConnectionInput{}
	if value, ok := raw["name"]; ok {
		var parsed string
		if err := json.Unmarshal(value, &parsed); err != nil {
			return invalidCollectorField("name", err)
		}
		input.Name = &parsed
	}

	if value, ok := raw["config"]; ok {
		if err := json.Unmarshal(value, &input.Config); err != nil {
			return invalidCollectorField("config", err)
		}
		input.HasConfig = true
	}
	if value, ok := raw["metadata"]; ok {
		if err := json.Unmarshal(value, &input.Metadata); err != nil {
			return invalidCollectorField("metadata", err)
		}
		input.HasMetadata = true
	}
	if value, ok := raw["secrets"]; ok {
		if err := json.Unmarshal(value, &input.Secrets); err != nil {
			return invalidCollectorField("secrets", err)
		}
	}
	result, err := h.service.UpdateConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *CollectorHandler) DeleteConnection(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if err := h.service.DeleteConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

func invalidCollectorField(field string, err error) error {
	return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, field+" 字段格式无效", err)
}
