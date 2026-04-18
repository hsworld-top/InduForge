package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// DataPointHandler 负责承接 datapoints 领域的 HTTP 请求。
type DataPointHandler struct {
	service *service.DataPointService
}

// NewDataPointHandler 创建数据点处理器。
func NewDataPointHandler(dataPointService *service.DataPointService) *DataPointHandler {
	return &DataPointHandler{service: dataPointService}
}

// List 返回项目下的数据点列表与分页信息。
func (h *DataPointHandler) List(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	filter, err := parseDataPointListFilter(r)
	if err != nil {
		return err
	}

	result, err := h.service.ListDataPoints(r.Context(), r.PathValue("projectId"), filter)
	if err != nil {
		return err
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Get 返回单个数据点详情。
func (h *DataPointHandler) Get(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	result, err := h.service.GetDataPoint(r.Context(), r.PathValue("projectId"), r.PathValue("id"))
	if err != nil {
		return err
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// GetValue 按路径读取数据点值。
func (h *DataPointHandler) GetValue(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	result, err := h.service.GetDataPointValue(r.Context(), r.PathValue("projectId"), r.URL.Query().Get("path"))
	if err != nil {
		return err
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Update 更新单个数据点。
func (h *DataPointHandler) Update(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	raw, err := decodeJSONObjectBody(r)
	if err != nil {
		return err
	}
	if err := validateDataPointUpdatePayloadKeys(raw); err != nil {
		return err
	}

	input, err := parseDataPointUpdateInput(raw)
	if err != nil {
		return err
	}

	result, err := h.service.UpdateDataPoint(r.Context(), r.PathValue("projectId"), r.PathValue("id"), claims.UserID, input)
	if err != nil {
		return err
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Delete 删除单个无效数据点。
func (h *DataPointHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	if err := h.service.DeleteDataPoint(r.Context(), r.PathValue("projectId"), r.PathValue("id")); err != nil {
		return err
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

// DeleteBatch 批量删除无效数据点。
func (h *DataPointHandler) DeleteBatch(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	_ = claims

	var request struct {
		IDs []string `json:"ids"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	deletedCount, err := h.service.DeleteDataPointsBatch(r.Context(), r.PathValue("projectId"), request.IDs)
	if err != nil {
		return err
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]int{"deletedCount": deletedCount})
	return nil
}

func parseDataPointListFilter(r *http.Request) (service.DataPointListFilter, error) {
	query := r.URL.Query()

	page, err := parseOptionalInt(query.Get("page"), 1, "page")
	if err != nil {
		return service.DataPointListFilter{}, err
	}
	pageSize, err := parseOptionalInt(query.Get("pageSize"), 50, "pageSize")
	if err != nil {
		return service.DataPointListFilter{}, err
	}

	var sourceIDs []string
	if raw := strings.TrimSpace(query.Get("sourceIds")); raw != "" {
		for _, item := range strings.Split(raw, ",") {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				sourceIDs = append(sourceIDs, trimmed)
			}
		}
	}

	return service.DataPointListFilter{
		Type:      strings.TrimSpace(query.Get("type")),
		Status:    strings.TrimSpace(query.Get("status")),
		Search:    strings.TrimSpace(query.Get("search")),
		SourceID:  strings.TrimSpace(query.Get("sourceId")),
		SourceIDs: sourceIDs,
		Page:      page,
		PageSize:  pageSize,
	}, nil
}

func validateDataPointUpdatePayloadKeys(raw map[string]json.RawMessage) error {
	allowedFields := map[string]struct{}{
		"name":              {},
		"description":       {},
		"sourceType":        {},
		"sourceId":          {},
		"sourceConfig":      {},
		"dataType":          {},
		"unit":              {},
		"precisionNum":      {},
		"defaultValue":      {},
		"minValue":          {},
		"maxValue":          {},
		"alarmLow":          {},
		"alarmHigh":         {},
		"tags":              {},
		"refreshMode":       {},
		"refreshIntervalMs": {},
		"status":            {},
	}

	for field := range raw {
		if _, ok := allowedFields[field]; ok {
			continue
		}
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "更新请求包含不支持的字段: "+field)
	}
	return nil
}

func parseDataPointUpdateInput(raw map[string]json.RawMessage) (service.UpdateDataPointInput, error) {
	input := service.UpdateDataPointInput{}

	if value, ok := raw["name"]; ok {
		var next string
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "name 字段格式无效", err)
		}
		input.Name = &next
	}
	if value, ok := raw["description"]; ok {
		var next string
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "description 字段格式无效", err)
		}
		input.Description = &next
	}
	if value, ok := raw["sourceType"]; ok {
		var next string
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "sourceType 字段格式无效", err)
		}
		input.SourceType = &next
	}
	if value, ok := raw["sourceId"]; ok {
		var next string
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "sourceId 字段格式无效", err)
		}
		input.SourceID = &next
	}
	if value, ok := raw["sourceConfig"]; ok {
		var next map[string]any
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "sourceConfig 字段格式无效", err)
		}
		input.SourceConfig = next
		input.HasSourceConfig = true
	}
	if value, ok := raw["dataType"]; ok {
		var next string
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "dataType 字段格式无效", err)
		}
		input.DataType = &next
	}
	if value, ok := raw["unit"]; ok {
		var next string
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "unit 字段格式无效", err)
		}
		input.Unit = &next
	}
	if value, ok := raw["precisionNum"]; ok {
		var next int
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "precisionNum 字段格式无效", err)
		}
		input.PrecisionNum = &next
	}
	if value, ok := raw["defaultValue"]; ok {
		var next string
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "defaultValue 字段格式无效", err)
		}
		input.DefaultValue = &next
	}
	if value, ok := raw["minValue"]; ok {
		var next float64
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "minValue 字段格式无效", err)
		}
		input.MinValue = &next
	}
	if value, ok := raw["maxValue"]; ok {
		var next float64
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "maxValue 字段格式无效", err)
		}
		input.MaxValue = &next
	}
	if value, ok := raw["alarmLow"]; ok {
		var next float64
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "alarmLow 字段格式无效", err)
		}
		input.AlarmLow = &next
	}
	if value, ok := raw["alarmHigh"]; ok {
		var next float64
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "alarmHigh 字段格式无效", err)
		}
		input.AlarmHigh = &next
	}
	if value, ok := raw["tags"]; ok {
		var next []any
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "tags 字段格式无效", err)
		}
		input.Tags = next
		input.HasTags = true
	}
	if value, ok := raw["refreshMode"]; ok {
		var next string
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "refreshMode 字段格式无效", err)
		}
		input.RefreshMode = &next
	}
	if value, ok := raw["refreshIntervalMs"]; ok {
		var next int
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "refreshIntervalMs 字段格式无效", err)
		}
		input.RefreshIntervalMS = &next
	}
	if value, ok := raw["status"]; ok {
		var next string
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "status 字段格式无效", err)
		}
		input.Status = &next
	}

	return input, nil
}
