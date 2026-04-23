package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// QueryHandler 负责承接 queries 领域的 HTTP 请求。
type QueryHandler struct {
	service *service.QueryService
}

// NewQueryHandler 创建 queries 处理器。
func NewQueryHandler(queryService *service.QueryService) *QueryHandler {
	return &QueryHandler{service: queryService}
}

// List 返回项目下的查询列表与分页信息。
func (h *QueryHandler) List(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	filter, err := parseQueryListFilter(r)
	if err != nil {
		return err
	}

	result, err := h.service.ListQueries(r.Context(), r.PathValue("projectId"), filter)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Create 在项目内创建查询定义。
func (h *QueryHandler) Create(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name            string         `json:"name"`
		Description     *string        `json:"description"`
		Category        *string        `json:"category"`
		ConnectionID    string         `json:"connectionId"`
		QueryType       string         `json:"queryType"`
		Config          map[string]any `json:"config"`
		Transformer     *string        `json:"transformer"`
		IsEnabled       *bool          `json:"isEnabled"`
		TimeoutMS       *int           `json:"timeoutMs"`
		CacheEnabled    *bool          `json:"cacheEnabled"`
		CacheTtlSeconds *int           `json:"cacheTtlSeconds"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.CreateQuery(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateQueryInput{
		Name:            request.Name,
		Description:     request.Description,
		Category:        request.Category,
		ConnectionID:    request.ConnectionID,
		QueryType:       request.QueryType,
		Config:          request.Config,
		Transformer:     request.Transformer,
		IsEnabled:       request.IsEnabled,
		TimeoutMS:       request.TimeoutMS,
		CacheEnabled:    request.CacheEnabled,
		CacheTtlSeconds: request.CacheTtlSeconds,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Execute 执行指定查询，并返回 data、executionTime 与 rowCount。
func (h *QueryHandler) Execute(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Parameters map[string]any `json:"parameters"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.ExecuteQuery(r.Context(), claims, r.PathValue("id"), service.ExecuteQueryInput{
		Parameters: request.Parameters,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Update 更新查询定义。
func (h *QueryHandler) Update(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	raw, err := decodeJSONObjectBody(r)
	if err != nil {
		return err
	}
	if err := validateQueryUpdatePayloadKeys(raw); err != nil {
		return err
	}

	input, err := parseQueryUpdateInput(raw)
	if err != nil {
		return err
	}

	result, err := h.service.UpdateQuery(r.Context(), claims, r.PathValue("id"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// Delete 删除查询定义。
func (h *QueryHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	if err := h.service.DeleteQuery(r.Context(), claims, r.PathValue("id")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

func parseQueryListFilter(r *http.Request) (service.QueryListFilter, error) {
	query := r.URL.Query()

	page, err := parseOptionalInt(query.Get("page"), 1, "page")
	if err != nil {
		return service.QueryListFilter{}, err
	}
	pageSize, err := parseOptionalInt(query.Get("pageSize"), 20, "pageSize")
	if err != nil {
		return service.QueryListFilter{}, err
	}

	var isEnabled *bool
	if raw := strings.TrimSpace(query.Get("isEnabled")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return service.QueryListFilter{}, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "isEnabled 参数格式无效")
		}
		isEnabled = &value
	}

	return service.QueryListFilter{
		ConnectionID: strings.TrimSpace(query.Get("connectionId")),
		QueryType:    strings.TrimSpace(query.Get("queryType")),
		Search:       strings.TrimSpace(query.Get("search")),
		IsEnabled:    isEnabled,
		Page:         page,
		PageSize:     pageSize,
	}, nil
}

func validateQueryUpdatePayloadKeys(raw map[string]json.RawMessage) error {
	allowedFields := map[string]struct{}{
		"name":            {},
		"description":     {},
		"category":        {},
		"connectionId":    {},
		"queryType":       {},
		"config":          {},
		"transformer":     {},
		"isEnabled":       {},
		"timeoutMs":       {},
		"cacheEnabled":    {},
		"cacheTtlSeconds": {},
	}

	for field := range raw {
		if _, ok := allowedFields[field]; ok {
			continue
		}
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "更新请求包含不支持的字段: "+field)
	}
	return nil
}

func parseQueryUpdateInput(raw map[string]json.RawMessage) (service.UpdateQueryInput, error) {
	input := service.UpdateQueryInput{}

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
	if value, ok := raw["category"]; ok {
		var next string
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "category 字段格式无效", err)
		}
		input.Category = &next
	}
	if value, ok := raw["connectionId"]; ok {
		var next string
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "connectionId 字段格式无效", err)
		}
		input.ConnectionID = &next
	}
	if value, ok := raw["queryType"]; ok {
		var next string
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "queryType 字段格式无效", err)
		}
		input.QueryType = &next
	}
	if value, ok := raw["config"]; ok {
		var next map[string]any
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "config 字段格式无效", err)
		}
		input.Config = next
		input.HasConfig = true
	}
	if value, ok := raw["transformer"]; ok {
		var next string
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "transformer 字段格式无效", err)
		}
		input.Transformer = &next
	}
	if value, ok := raw["isEnabled"]; ok {
		var next bool
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "isEnabled 字段格式无效", err)
		}
		input.IsEnabled = &next
	}
	if value, ok := raw["timeoutMs"]; ok {
		var next int
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "timeoutMs 字段格式无效", err)
		}
		input.TimeoutMS = &next
	}
	if value, ok := raw["cacheEnabled"]; ok {
		var next bool
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "cacheEnabled 字段格式无效", err)
		}
		input.CacheEnabled = &next
	}
	if value, ok := raw["cacheTtlSeconds"]; ok {
		var next int
		if err := json.Unmarshal(value, &next); err != nil {
			return input, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "cacheTtlSeconds 字段格式无效", err)
		}
		input.CacheTtlSeconds = &next
	}

	return input, nil
}

func decodeJSONObjectBody(r *http.Request) (map[string]json.RawMessage, error) {
	if r.Body == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体不能为空")
	}

	var raw map[string]json.RawMessage
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体 JSON 格式无效", err)
	}
	if decoder.More() {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体只能包含一个 JSON 对象")
	}
	return raw, nil
}

func parseOptionalInt(raw string, defaultValue int, fieldName string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, fieldName+" 参数格式无效")
	}
	return value, nil
}
