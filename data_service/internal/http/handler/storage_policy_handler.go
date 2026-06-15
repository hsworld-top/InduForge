package handler

import (
	"net/http"
	"strings"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// StoragePolicyHandler 承接统一数据点历史归档策略请求。
type StoragePolicyHandler struct {
	service *service.StoragePolicyService
}

func NewStoragePolicyHandler(storagePolicyService *service.StoragePolicyService) *StoragePolicyHandler {
	return &StoragePolicyHandler{service: storagePolicyService}
}

func (h *StoragePolicyHandler) List(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	filter, err := parseStoragePolicyListFilter(r)
	if err != nil {
		return err
	}
	result, err := h.service.List(r.Context(), claims, r.PathValue("projectId"), filter)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *StoragePolicyHandler) Get(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.Get(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *StoragePolicyHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	input, err := decodeStoragePolicyInput(r)
	if err != nil {
		return err
	}
	result, err := h.service.Create(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *StoragePolicyHandler) Update(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	input, err := decodeStoragePolicyInput(r)
	if err != nil {
		return err
	}
	result, err := h.service.Update(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id"), service.UpdateStoragePolicyInput{CreateStoragePolicyInput: input})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *StoragePolicyHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err := h.service.Delete(r.Context(), claims, r.PathValue("projectId"), r.PathValue("id")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

func (h *StoragePolicyHandler) Targets(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.Targets(r.Context(), claims, r.PathValue("projectId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *StoragePolicyHandler) Estimate(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	input, err := decodeStoragePolicyInput(r)
	if err != nil {
		return err
	}
	result, err := h.service.Estimate(r.Context(), claims, r.PathValue("projectId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *StoragePolicyHandler) Coverage(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	query := r.URL.Query()
	result, err := h.service.Coverage(r.Context(), claims, r.PathValue("projectId"), query.Get("datapointId"), query.Get("path"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func parseStoragePolicyListFilter(r *http.Request) (service.StoragePolicyListFilter, error) {
	query := r.URL.Query()
	page, err := parseOptionalInt(query.Get("page"), 1, "page")
	if err != nil {
		return service.StoragePolicyListFilter{}, err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(query.Get("pageSize"), query.Get("limit")), 20, "pageSize")
	if err != nil {
		return service.StoragePolicyListFilter{}, err
	}
	return service.StoragePolicyListFilter{
		Search: strings.TrimSpace(query.Get("search")), Status: strings.TrimSpace(query.Get("status")),
		TargetConnectionID: strings.TrimSpace(query.Get("targetConnectionId")), WriteMode: strings.TrimSpace(query.Get("writeMode")),
		BindingMode: strings.TrimSpace(query.Get("bindingMode")), Page: page, PageSize: pageSize,
	}, nil
}

func decodeStoragePolicyInput(r *http.Request) (service.CreateStoragePolicyInput, error) {
	var request struct {
		Name               string                         `json:"name"`
		Description        *string                        `json:"description"`
		TargetConnectionID string                         `json:"targetConnectionId"`
		TargetCapability   string                         `json:"targetCapability"`
		BindingMode        string                         `json:"bindingMode"`
		BindingFilter      service.StorageDataPointFilter `json:"bindingFilter"`
		DatapointIDs       []string                       `json:"datapointIds"`
		WriteMode          string                         `json:"writeMode"`
		MinIntervalMS      *int                           `json:"minIntervalMs"`
		Deadband           *float64                       `json:"deadband"`
		SnapshotIntervalMS *int                           `json:"snapshotIntervalMs"`
		IncludeQualities   []string                       `json:"includeQualities"`
		RetentionDays      int                            `json:"retentionDays"`
		TargetTableMode    string                         `json:"targetTableMode"`
		TargetTableConfig  map[string]any                 `json:"targetTableConfig"`
		Status             *string                        `json:"status"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return service.CreateStoragePolicyInput{}, err
	}
	return service.CreateStoragePolicyInput{
		Name: request.Name, Description: request.Description, TargetConnectionID: request.TargetConnectionID,
		TargetCapability: request.TargetCapability, BindingMode: request.BindingMode, BindingFilter: request.BindingFilter,
		DatapointIDs: request.DatapointIDs, WriteMode: request.WriteMode, MinIntervalMS: request.MinIntervalMS,
		Deadband: request.Deadband, SnapshotIntervalMS: request.SnapshotIntervalMS, IncludeQualities: request.IncludeQualities,
		RetentionDays: request.RetentionDays, TargetTableMode: request.TargetTableMode, TargetTableConfig: request.TargetTableConfig,
		Status: request.Status,
	}, nil
}
