package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
)

// BatchStatus 批量读取数据点状态，兼容 ids 与 paths 两种入参。
func (h *DataPointHandler) BatchStatus(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	var request struct {
		IDs          []string `json:"ids"`
		DatapointIDs []string `json:"datapointIds"`
		SourceIDs    []string `json:"sourceIds"`
		Paths        []string `json:"paths"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	ids := request.IDs
	if len(ids) == 0 {
		ids = request.DatapointIDs
	}

	result, err := h.service.GetDataPointStatuses(r.Context(), r.PathValue("projectId"), ids, request.Paths, request.SourceIDs)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
		"datapoints": result,
	})
	return nil
}

// BatchValues 按数据点主键批量读取当前值。
func (h *DataPointHandler) BatchValues(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	var request struct {
		IDs          []string `json:"ids"`
		DatapointIDs []string `json:"datapointIds"`
		Paths        []string `json:"paths"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	ids := request.IDs
	if len(ids) == 0 {
		ids = request.DatapointIDs
	}

	values, err := h.service.GetDataPointValues(r.Context(), r.PathValue("projectId"), ids, request.Paths)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
		"values": values,
	})
	return nil
}

// WriteValue 写入设计态数据点值。
func (h *DataPointHandler) WriteValue(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Value any `json:"value"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.WriteDataPointValue(r.Context(), r.PathValue("projectId"), r.PathValue("id"), claims.UserID, request.Value)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Usages 返回数据点使用情况。
func (h *DataPointHandler) Usages(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	usages, err := h.service.GetDataPointUsages(r.Context(), r.PathValue("projectId"), r.PathValue("id"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
		"usages": usages,
	})
	return nil
}
