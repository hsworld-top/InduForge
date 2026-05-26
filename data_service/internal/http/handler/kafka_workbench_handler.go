package handler

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// KafkaWorkbenchHandler 暴露 Kafka 工作台配置、预览和字段映射接口。
type KafkaWorkbenchHandler struct {
	service *service.KafkaWorkbenchService
}

// NewKafkaWorkbenchHandler 创建 Kafka 工作台处理器。
func NewKafkaWorkbenchHandler(kafkaService *service.KafkaWorkbenchService) *KafkaWorkbenchHandler {
	return &KafkaWorkbenchHandler{service: kafkaService}
}

// ListTopicGroups 查询 Kafka Topic 分组。
func (h *KafkaWorkbenchHandler) ListTopicGroups(w http.ResponseWriter, r *http.Request) error {
	groups, err := h.service.ListTopicGroups(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": groups})
	return nil
}

// CreateTopicGroup 创建 Kafka Topic 分组。
func (h *KafkaWorkbenchHandler) CreateTopicGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.CreateKafkaTopicGroupInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	group, err := h.service.CreateTopicGroup(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), group)
	return nil
}

// UpdateTopicGroup 更新 Kafka Topic 分组。
func (h *KafkaWorkbenchHandler) UpdateTopicGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.UpdateKafkaTopicGroupInput
	raw, err := decodeKafkaWorkbenchBody(r, &input)
	if err != nil {
		return err
	}
	_, input.HasParent = raw["parentId"]
	group, err := h.service.UpdateTopicGroup(r.Context(), r.PathValue("projectId"), r.PathValue("groupId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), group)
	return nil
}

// DeleteTopicGroup 删除 Kafka Topic 分组。
func (h *KafkaWorkbenchHandler) DeleteTopicGroup(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if err := h.service.DeleteTopicGroup(r.Context(), r.PathValue("projectId"), r.PathValue("groupId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"deleted": true})
	return nil
}

// ListTopicMappings 查询 Kafka Topic 映射。
func (h *KafkaWorkbenchHandler) ListTopicMappings(w http.ResponseWriter, r *http.Request) error {
	mappings, err := h.service.ListTopicMappings(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": mappings})
	return nil
}

// GetTopicMapping 读取单个 Kafka Topic 映射。
func (h *KafkaWorkbenchHandler) GetTopicMapping(w http.ResponseWriter, r *http.Request) error {
	mapping, err := h.service.GetTopicMapping(r.Context(), r.PathValue("projectId"), r.PathValue("mappingId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), mapping)
	return nil
}

// CreateTopicMapping 创建 Kafka Topic 映射。
func (h *KafkaWorkbenchHandler) CreateTopicMapping(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.CreateKafkaTopicMappingInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	mapping, err := h.service.CreateTopicMapping(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), mapping)
	return nil
}

// UpdateTopicMapping 更新 Kafka Topic 映射。
func (h *KafkaWorkbenchHandler) UpdateTopicMapping(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.UpdateKafkaTopicMappingInput
	raw, err := decodeKafkaWorkbenchBody(r, &input)
	if err != nil {
		return err
	}
	_, input.HasGroupID = raw["groupId"]
	mapping, err := h.service.UpdateTopicMapping(r.Context(), r.PathValue("projectId"), r.PathValue("mappingId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), mapping)
	return nil
}

// DeleteTopicMapping 删除 Kafka Topic 映射。
func (h *KafkaWorkbenchHandler) DeleteTopicMapping(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if err := h.service.DeleteTopicMapping(r.Context(), r.PathValue("projectId"), r.PathValue("mappingId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"deleted": true})
	return nil
}

// PreviewConnection 对 Kafka 接入源默认 Topic 做一次短时预览。
func (h *KafkaWorkbenchHandler) PreviewConnection(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var input service.KafkaPreviewInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.PreviewConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// PreviewTopicMapping 对指定 Kafka Topic 映射做一次短时预览。
func (h *KafkaWorkbenchHandler) PreviewTopicMapping(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var input service.KafkaPreviewInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	result, err := h.service.PreviewTopicMapping(r.Context(), r.PathValue("projectId"), r.PathValue("mappingId"), input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// ListFieldGroups 查询 Kafka Topic 下变量分组。
func (h *KafkaWorkbenchHandler) ListFieldGroups(w http.ResponseWriter, r *http.Request) error {
	groups, err := h.service.ListFieldGroups(r.Context(), r.PathValue("projectId"), r.PathValue("mappingId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": groups})
	return nil
}

// CreateFieldGroup 创建 Kafka Topic 下变量分组。
func (h *KafkaWorkbenchHandler) CreateFieldGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.CreateKafkaFieldGroupInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	group, err := h.service.CreateFieldGroup(r.Context(), r.PathValue("projectId"), r.PathValue("mappingId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), group)
	return nil
}

// UpdateFieldGroup 更新 Kafka 变量分组。
func (h *KafkaWorkbenchHandler) UpdateFieldGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.UpdateKafkaFieldGroupInput
	raw, err := decodeKafkaWorkbenchBody(r, &input)
	if err != nil {
		return err
	}
	_, input.HasParentID = raw["parentId"]
	group, err := h.service.UpdateFieldGroup(r.Context(), r.PathValue("projectId"), r.PathValue("groupId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), group)
	return nil
}

// DeleteFieldGroup 删除 Kafka 变量分组。
func (h *KafkaWorkbenchHandler) DeleteFieldGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err := h.service.DeleteFieldGroup(r.Context(), r.PathValue("projectId"), r.PathValue("groupId"), claims.UserID); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"deleted": true})
	return nil
}

// ListFields 查询 Kafka 字段映射。
func (h *KafkaWorkbenchHandler) ListFields(w http.ResponseWriter, r *http.Request) error {
	query := r.URL.Query()
	var groupID *string
	if value := query.Get("groupId"); value != "" {
		groupID = &value
	}
	page, err := parseOptionalInt(query.Get("page"), 1, "page")
	if err != nil {
		return err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(query.Get("pageSize"), query.Get("limit")), 20, "pageSize")
	if err != nil {
		return err
	}
	result, err := h.service.ListFieldsPage(r.Context(), r.PathValue("projectId"), r.PathValue("mappingId"), groupID, query.Get("q"), page, pageSize)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// CreateField 创建 Kafka 字段映射。
func (h *KafkaWorkbenchHandler) CreateField(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.CreateKafkaFieldInput
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	field, err := h.service.CreateField(r.Context(), r.PathValue("projectId"), r.PathValue("mappingId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), field)
	return nil
}

// CreateFieldsBatch 批量创建 Kafka 字段映射。
func (h *KafkaWorkbenchHandler) CreateFieldsBatch(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input struct {
		Fields []service.CreateKafkaFieldInput `json:"fields"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	fields, err := h.service.CreateFieldsBatch(r.Context(), r.PathValue("projectId"), r.PathValue("mappingId"), claims.UserID, input.Fields)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": fields})
	return nil
}

// UpdateField 更新 Kafka 字段映射。
func (h *KafkaWorkbenchHandler) UpdateField(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input service.UpdateKafkaFieldInput
	raw, err := decodeKafkaWorkbenchBody(r, &input)
	if err != nil {
		return err
	}
	_, input.HasGroupID = raw["groupId"]
	field, err := h.service.UpdateField(r.Context(), r.PathValue("projectId"), r.PathValue("fieldId"), claims.UserID, input)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), field)
	return nil
}

// DeleteField 删除 Kafka 字段映射。
func (h *KafkaWorkbenchHandler) DeleteField(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if err := h.service.DeleteField(r.Context(), r.PathValue("projectId"), r.PathValue("fieldId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"deleted": true})
	return nil
}

// ToggleField 切换 Kafka 字段映射启停状态。
func (h *KafkaWorkbenchHandler) ToggleField(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var input struct {
		Enabled bool `json:"enabled"`
	}
	if err := decodeJSONBody(r, &input); err != nil {
		return err
	}
	field, err := h.service.ToggleField(r.Context(), r.PathValue("projectId"), r.PathValue("fieldId"), claims.UserID, input.Enabled)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), field)
	return nil
}

func decodeKafkaWorkbenchBody(r *http.Request, target any) (map[string]json.RawMessage, error) {
	if r.Body == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体不能为空")
	}
	var raw map[string]json.RawMessage
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体 JSON 格式无效", err)
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体 JSON 格式无效", err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请求体 JSON 格式无效", err)
	}
	return raw, nil
}
