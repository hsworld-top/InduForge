package handler

import (
	"net/http"
	"strings"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

type HistoryStorageHandler struct {
	service *service.HistoryStorageService
}

func NewHistoryStorageHandler(historyStorageService *service.HistoryStorageService) *HistoryStorageHandler {
	return &HistoryStorageHandler{service: historyStorageService}
}

func (h *HistoryStorageHandler) ListSources(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	page, err := parseOptionalInt(r.URL.Query().Get("page"), 1, "page")
	if err != nil {
		return err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(r.URL.Query().Get("pageSize"), r.URL.Query().Get("limit")), 20, "pageSize")
	if err != nil {
		return err
	}
	result, err := h.service.ListSources(r.Context(), claims, r.PathValue("projectId"), service.HistoryStorageSourceListFilter{
		Search: strings.TrimSpace(r.URL.Query().Get("search")), ScopeType: strings.TrimSpace(r.URL.Query().Get("scopeType")),
		HistoryState: strings.TrimSpace(r.URL.Query().Get("historyState")), Page: page, PageSize: pageSize,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *HistoryStorageHandler) GetSource(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.GetSource(r.Context(), claims, r.PathValue("projectId"), service.HistoryStorageScopeRef{Type: r.PathValue("scopeType"), ID: r.PathValue("scopeId")})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *HistoryStorageHandler) SaveSource(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	input, err := decodeHistoryStorageSaveInput(r)
	if err != nil {
		return err
	}
	result, err := h.service.SaveSource(r.Context(), claims, r.PathValue("projectId"), service.HistoryStorageScopeRef{Type: r.PathValue("scopeType"), ID: r.PathValue("scopeId")}, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *HistoryStorageHandler) ListTargets(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.ListTargets(r.Context(), claims, r.PathValue("projectId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *HistoryStorageHandler) GetDatapoint(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.GetDatapoint(r.Context(), claims, r.PathValue("projectId"), r.PathValue("datapointId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *HistoryStorageHandler) SaveDatapoint(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	input, err := decodeHistoryStorageSaveInput(r)
	if err != nil {
		return err
	}
	result, err := h.service.SaveDatapoint(r.Context(), claims, r.PathValue("projectId"), r.PathValue("datapointId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *HistoryStorageHandler) BatchConfigure(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Selection struct {
			Mode                string   `json:"mode"`
			DatapointIDs        []string `json:"datapointIds"`
			ExcludeDatapointIDs []string `json:"excludeDatapointIds"`
			Filters             struct {
				Type           string   `json:"type"`
				Status         string   `json:"status"`
				Search         string   `json:"search"`
				AccessSourceID string   `json:"accessSourceId"`
				SourceID       string   `json:"sourceId"`
				SourceIDs      []string `json:"sourceIds"`
				Tags           []string `json:"tags"`
			} `json:"filters"`
		} `json:"selection"`
		Behavior      string                                    `json:"behavior"`
		Configuration *service.HistoryStorageConfigurationInput `json:"configuration"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.BatchConfigure(r.Context(), claims, r.PathValue("projectId"), service.BatchSaveHistoryStorageInput{
		Selection: service.HistoryStorageBulkSelection{
			Mode: request.Selection.Mode, DatapointIDs: request.Selection.DatapointIDs, ExcludeDatapointIDs: request.Selection.ExcludeDatapointIDs,
			Filters: service.DataPointListFilter{Type: request.Selection.Filters.Type, Status: request.Selection.Filters.Status, Search: request.Selection.Filters.Search, AccessSourceID: request.Selection.Filters.AccessSourceID, SourceID: request.Selection.Filters.SourceID, SourceIDs: request.Selection.Filters.SourceIDs, Tags: request.Selection.Filters.Tags},
		},
		Behavior: request.Behavior, Configuration: request.Configuration,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func decodeHistoryStorageSaveInput(r *http.Request) (service.SaveHistoryStorageInput, error) {
	var request struct {
		Behavior      string                                    `json:"behavior"`
		Configuration *service.HistoryStorageConfigurationInput `json:"configuration"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return service.SaveHistoryStorageInput{}, err
	}
	return service.SaveHistoryStorageInput{Behavior: request.Behavior, Configuration: request.Configuration}, nil
}
