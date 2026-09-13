package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/repository"
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
		return normalizeRepresentativeHandlerError(err)
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
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Create 创建手工或引用已有来源的数据点。
func (h *DataPointHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.CreateDataPointInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.CreateDataPoint(r.Context(), r.PathValue("projectId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// GetValue 按路径读取数据点值。
func (h *DataPointHandler) GetValue(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	parameters := map[string]any{}
	if rawParameters := strings.TrimSpace(r.URL.Query().Get("parameters")); rawParameters != "" {
		if err := json.Unmarshal([]byte(rawParameters), &parameters); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "parameters 必须是 JSON 对象", err)
		}
	}
	result, err := h.service.GetDataPointValueWithParameters(r.Context(), r.PathValue("projectId"), r.URL.Query().Get("path"), parameters)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
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
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// UpdateRuntimePermissions 更新数据点运行态写权限。
func (h *DataPointHandler) UpdateRuntimePermissions(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	raw, err := decodeJSONObjectBody(r)
	if err != nil {
		return err
	}

	input, err := parseDataPointRuntimePermissionsInput(raw)
	if err != nil {
		return err
	}

	result, err := h.service.UpdateDataPointRuntimePermissions(r.Context(), r.PathValue("projectId"), r.PathValue("id"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// GetCustomAttributes 返回数据点开发态自定义属性默认值。
func (h *DataPointHandler) GetCustomAttributes(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.GetDataPointCustomAttributes(r.Context(), r.PathValue("projectId"), r.PathValue("id"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// UpdateCustomAttributes 原子替换数据点开发态自定义属性默认值。
func (h *DataPointHandler) UpdateCustomAttributes(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Attributes map[string]string `json:"attributes"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateDataPointCustomAttributes(r.Context(), r.PathValue("projectId"), r.PathValue("id"), claims.UserID, request.Attributes)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
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
		return normalizeRepresentativeHandlerError(err)
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
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]int{"deletedCount": deletedCount})
	return nil
}

// DeleteBatchByFilter 按筛选条件批量删除无效数据点。
func (h *DataPointHandler) DeleteBatchByFilter(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	var request struct {
		Filter dataPointFilterPayload `json:"filter"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	deletedCount, err := h.service.DeleteInvalidDataPointsByFilter(r.Context(), r.PathValue("projectId"), request.Filter.toServiceFilter())
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]int{"deletedCount": deletedCount})
	return nil
}

// AppendTagsByFilter 按筛选条件批量追加数据点标签。
func (h *DataPointHandler) AppendTagsByFilter(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Filter dataPointFilterPayload `json:"filter"`
		Tags   []string               `json:"tags"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	updatedCount, err := h.service.AppendDataPointTagsByFilter(r.Context(), r.PathValue("projectId"), claims.UserID, request.Filter.toServiceFilter(), request.Tags)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]int{"updatedCount": updatedCount})
	return nil
}

type dataPointFilterPayload struct {
	Type           string   `json:"type"`
	DataType       string   `json:"dataType"`
	Status         string   `json:"status"`
	Search         string   `json:"search"`
	AccessSourceID string   `json:"accessSourceId"`
	SourceID       string   `json:"sourceId"`
	SourceIDs      []string `json:"sourceIds"`
	Tags           []string `json:"tags"`
}

func (p dataPointFilterPayload) toServiceFilter() service.DataPointListFilter {
	return service.DataPointListFilter{
		Type:           strings.TrimSpace(p.Type),
		DataType:       strings.TrimSpace(p.DataType),
		Status:         strings.TrimSpace(p.Status),
		Search:         strings.TrimSpace(p.Search),
		AccessSourceID: strings.TrimSpace(p.AccessSourceID),
		SourceID:       strings.TrimSpace(p.SourceID),
		SourceIDs:      p.SourceIDs,
		Tags:           p.Tags,
	}
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

	var tags []string
	if raw := strings.TrimSpace(query.Get("tags")); raw != "" {
		for _, item := range strings.Split(raw, ",") {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
	}

	return service.DataPointListFilter{
		Type:           strings.TrimSpace(query.Get("type")),
		DataType:       strings.TrimSpace(query.Get("dataType")),
		Status:         strings.TrimSpace(query.Get("status")),
		Search:         strings.TrimSpace(query.Get("search")),
		AccessSourceID: strings.TrimSpace(query.Get("accessSourceId")),
		SourceID:       strings.TrimSpace(query.Get("sourceId")),
		SourceIDs:      sourceIDs,
		Tags:           tags,
		SortField:      strings.TrimSpace(firstNonEmpty(query.Get("sortField"), query.Get("sort"))),
		SortOrder:      strings.TrimSpace(firstNonEmpty(query.Get("sortOrder"), query.Get("order"))),
		Page:           page,
		PageSize:       pageSize,
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
		input.HasDefaultValue = true
		if !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			var next string
			if err := json.Unmarshal(value, &next); err != nil {
				return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "defaultValue 字段格式无效", err)
			}
			input.DefaultValue = &next
		}
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

func parseDataPointRuntimePermissionsInput(raw map[string]json.RawMessage) (service.UpdateDataPointRuntimePermissionsInput, error) {
	for field := range raw {
		if field == "write" {
			continue
		}
		return service.UpdateDataPointRuntimePermissionsInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "更新请求包含不支持的字段: "+field)
	}

	writePayload, ok := raw["write"]
	if !ok {
		return service.UpdateDataPointRuntimePermissionsInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "运行态权限请求缺少 write 字段")
	}
	if bytes.Equal(bytes.TrimSpace(writePayload), []byte("null")) {
		return service.UpdateDataPointRuntimePermissionsInput{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "write 权限字段不能为空")
	}

	payload, err := json.Marshal(map[string]json.RawMessage{
		"write": writePayload,
	})
	if err != nil {
		return service.UpdateDataPointRuntimePermissionsInput{}, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "write 权限字段格式无效", err)
	}

	var runtimePermissions repository.DataPointRuntimePermissions
	if err := json.Unmarshal(payload, &runtimePermissions); err != nil {
		return service.UpdateDataPointRuntimePermissionsInput{}, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "write 权限字段格式无效", err)
	}

	return service.UpdateDataPointRuntimePermissionsInput{
		Write: runtimePermissions.Write,
	}, nil
}

func normalizeRoleNames(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return []string{}
	}
	return result
}
