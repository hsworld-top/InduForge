package handler

import (
	"net/http"
	"strconv"
	"strings"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/indu-forge/data_service/internal/service"
)

// ListConnections 返回项目下 MQTT 连接列表。
func (h *MqttHandler) ListConnections(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	page, err := parseOptionalInt(r.URL.Query().Get("page"), 1, "page")
	if err != nil {
		return err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(r.URL.Query().Get("pageSize"), r.URL.Query().Get("limit")), 50, "pageSize")
	if err != nil {
		return err
	}

	connections, total, err := h.service.ListMqttConnections(r.Context(), r.PathValue("projectId"), page, pageSize)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
		"connections": connections,
		"pagination": map[string]int{
			"page":       page,
			"pageSize":   pageSize,
			"total":      total,
			"totalPages": totalPages,
		},
	})
	return nil
}

// GetConnection 返回 MQTT 连接详情。
func (h *MqttHandler) GetConnection(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	result, err := h.service.GetMqttConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// UpdateConnection 更新 MQTT 连接配置。
func (h *MqttHandler) UpdateConnection(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name             string         `json:"name"`
		Status           string         `json:"status"`
		BrokerURL        string         `json:"brokerUrl"`
		Protocol         string         `json:"protocol"`
		Port             *int           `json:"port"`
		ClientID         *string        `json:"clientId"`
		Username         *string        `json:"username"`
		Password         *string        `json:"password"`
		Keepalive        *int           `json:"keepalive"`
		CleanSession     *bool          `json:"cleanSession"`
		QOS              *int           `json:"qos"`
		ReconnectPeriod  *int           `json:"reconnectPeriod"`
		ConnectTimeoutMS *int           `json:"connectTimeout"`
		Will             map[string]any `json:"will"`
		SSLConfig        map[string]any `json:"sslConfig"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.UpdateMqttConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.CreateMqttConnectionInput{
		Name:             request.Name,
		Status:           request.Status,
		BrokerURL:        request.BrokerURL,
		Protocol:         request.Protocol,
		Port:             request.Port,
		ClientID:         request.ClientID,
		Username:         request.Username,
		Password:         request.Password,
		Keepalive:        request.Keepalive,
		CleanSession:     request.CleanSession,
		QOS:              request.QOS,
		ReconnectPeriod:  request.ReconnectPeriod,
		ConnectTimeoutMS: request.ConnectTimeoutMS,
		Will:             request.Will,
		SSLConfig:        request.SSLConfig,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// DeleteConnection 删除 MQTT 连接。
func (h *MqttHandler) DeleteConnection(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if err := h.service.DeleteMqttConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

// TestConnectionConfig 测试 MQTT 连接配置。
func (h *MqttHandler) TestConnectionConfig(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	var request struct {
		Name             string         `json:"name"`
		BrokerURL        string         `json:"brokerUrl"`
		Protocol         string         `json:"protocol"`
		Port             *int           `json:"port"`
		ClientID         *string        `json:"clientId"`
		Username         *string        `json:"username"`
		Password         *string        `json:"password"`
		Keepalive        *int           `json:"keepalive"`
		CleanSession     *bool          `json:"cleanSession"`
		QOS              *int           `json:"qos"`
		ReconnectPeriod  *int           `json:"reconnectPeriod"`
		ConnectTimeoutMS *int           `json:"connectTimeout"`
		Will             map[string]any `json:"will"`
		SSLConfig        map[string]any `json:"sslConfig"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	if err := h.service.TestConnectionConfig(r.Context(), service.CreateMqttConnectionInput{
		Name:             request.Name,
		BrokerURL:        request.BrokerURL,
		Protocol:         request.Protocol,
		Port:             request.Port,
		ClientID:         request.ClientID,
		Username:         request.Username,
		Password:         request.Password,
		Keepalive:        request.Keepalive,
		CleanSession:     request.CleanSession,
		QOS:              request.QOS,
		ReconnectPeriod:  request.ReconnectPeriod,
		ConnectTimeoutMS: request.ConnectTimeoutMS,
		Will:             request.Will,
		SSLConfig:        request.SSLConfig,
	}); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"success": true})
	return nil
}

// StopConnection 停止 MQTT 连接。
func (h *MqttHandler) StopConnection(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	status, err := h.service.StopConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), status)
	return nil
}

// ListSubscriptions 返回 MQTT 订阅列表。
func (h *MqttHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	page, err := parseOptionalInt(r.URL.Query().Get("page"), 1, "page")
	if err != nil {
		return err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(r.URL.Query().Get("pageSize"), r.URL.Query().Get("limit")), 50, "pageSize")
	if err != nil {
		return err
	}

	subscriptions, total, err := h.service.ListSubscriptions(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), page, pageSize)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
		"list": subscriptions,
		"pagination": map[string]int{
			"page":       page,
			"pageSize":   pageSize,
			"total":      total,
			"totalPages": totalPages,
		},
	})
	return nil
}

// GetSubscription 返回单个订阅。
func (h *MqttHandler) GetSubscription(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.GetSubscription(r.Context(), r.PathValue("projectId"), r.PathValue("subscriptionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// CreateSubscription 创建订阅。
func (h *MqttHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Name             string  `json:"name"`
		Topic            string  `json:"topic"`
		QOS              int     `json:"qos"`
		Description      *string `json:"description"`
		IsEnabled        bool    `json:"isEnabled"`
		MessageRetention int     `json:"messageRetention"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.CreateSubscription(r.Context(), r.PathValue("projectId"), claims.UserID, repository.CreateMqttSubscriptionParams{
		ConnectionID:     r.PathValue("connectionId"),
		Name:             request.Name,
		Topic:            request.Topic,
		QOS:              request.QOS,
		Description:      request.Description,
		IsEnabled:        request.IsEnabled,
		MessageRetention: request.MessageRetention,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// UpdateSubscription 更新订阅。
func (h *MqttHandler) UpdateSubscription(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Name             string  `json:"name"`
		Topic            string  `json:"topic"`
		QOS              int     `json:"qos"`
		Description      *string `json:"description"`
		IsEnabled        bool    `json:"isEnabled"`
		MessageRetention int     `json:"messageRetention"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateSubscription(r.Context(), r.PathValue("projectId"), r.PathValue("subscriptionId"), claims.UserID, repository.UpdateMqttSubscriptionParams{
		Name:             request.Name,
		Topic:            request.Topic,
		QOS:              request.QOS,
		Description:      request.Description,
		IsEnabled:        request.IsEnabled,
		MessageRetention: request.MessageRetention,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// DeleteSubscription 删除订阅。
func (h *MqttHandler) DeleteSubscription(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err := h.service.DeleteSubscription(r.Context(), r.PathValue("projectId"), r.PathValue("subscriptionId"), claims.UserID); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

// ToggleSubscription 切换订阅启用状态。
func (h *MqttHandler) ToggleSubscription(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.ToggleSubscription(r.Context(), r.PathValue("projectId"), r.PathValue("subscriptionId"), claims.UserID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// ListTagGroups 返回变量组列表。
func (h *MqttHandler) ListTagGroups(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	groups, err := h.service.ListTagGroups(r.Context(), r.PathValue("projectId"), r.PathValue("subscriptionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
		"list": groups,
		"pagination": map[string]int{
			"page":       1,
			"pageSize":   len(groups),
			"total":      len(groups),
			"totalPages": 1,
		},
	})
	return nil
}

// GetTagGroup 返回单个变量组。
func (h *MqttHandler) GetTagGroup(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.GetTagGroup(r.Context(), r.PathValue("projectId"), r.PathValue("groupId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"group": result})
	return nil
}

// CreateTagGroup 创建变量组。
func (h *MqttHandler) CreateTagGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Name        string  `json:"name"`
		Code        string  `json:"code"`
		Description *string `json:"description"`
		Color       *string `json:"color"`
		Icon        *string `json:"icon"`
		Order       int     `json:"order"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.CreateTagGroup(r.Context(), r.PathValue("projectId"), r.PathValue("subscriptionId"), claims.UserID, repository.CreateMqttTagGroupParams{
		Name:        request.Name,
		Code:        request.Code,
		Description: request.Description,
		Color:       request.Color,
		Icon:        request.Icon,
		Order:       request.Order,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"group": result})
	return nil
}

// UpdateTagGroup 更新变量组。
func (h *MqttHandler) UpdateTagGroup(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Name        string  `json:"name"`
		Code        string  `json:"code"`
		Description *string `json:"description"`
		Color       *string `json:"color"`
		Icon        *string `json:"icon"`
		Order       int     `json:"order"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateTagGroup(r.Context(), r.PathValue("projectId"), r.PathValue("groupId"), claims.UserID, repository.UpdateMqttTagGroupParams{
		Name:        request.Name,
		Code:        request.Code,
		Description: request.Description,
		Color:       request.Color,
		Icon:        request.Icon,
		Order:       request.Order,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"group": result})
	return nil
}

// DeleteTagGroup 删除变量组。
func (h *MqttHandler) DeleteTagGroup(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if err := h.service.DeleteTagGroup(r.Context(), r.PathValue("projectId"), r.PathValue("groupId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

// UpdateTagGroupsOrder 批量更新变量组顺序。
func (h *MqttHandler) UpdateTagGroupsOrder(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Groups []struct {
			ID    string `json:"id"`
			Order int    `json:"order"`
		} `json:"groups"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	groups := make([]service.MqttTagGroup, 0, len(request.Groups))
	for _, group := range request.Groups {
		groups = append(groups, service.MqttTagGroup{ID: group.ID, Order: group.Order})
	}
	if len(groups) == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "groups 不能为空")
	}
	if err := h.service.UpdateTagGroupsOrder(r.Context(), r.PathValue("projectId"), claims.UserID, groups); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"updated": true})
	return nil
}

// ListTagsBySubscription 返回订阅下变量列表。
func (h *MqttHandler) ListTagsBySubscription(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	page, err := parseOptionalInt(r.URL.Query().Get("page"), 1, "page")
	if err != nil {
		return err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(r.URL.Query().Get("pageSize"), r.URL.Query().Get("limit")), 50, "pageSize")
	if err != nil {
		return err
	}
	isEnabled, err := parseOptionalBool(r.URL.Query().Get("isEnabled"))
	if err != nil {
		return err
	}
	tags, total, err := h.service.ListTagsBySubscription(r.Context(), r.PathValue("projectId"), r.PathValue("subscriptionId"), page, pageSize, isEnabled)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
		"list": tags,
		"pagination": map[string]int{
			"page":       page,
			"pageSize":   pageSize,
			"total":      total,
			"totalPages": totalPages,
		},
	})
	return nil
}

// ListTagsByProject 返回项目下变量列表。
func (h *MqttHandler) ListTagsByProject(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	page, err := parseOptionalInt(r.URL.Query().Get("page"), 1, "page")
	if err != nil {
		return err
	}
	pageSize, err := parseOptionalInt(firstNonEmpty(r.URL.Query().Get("pageSize"), r.URL.Query().Get("limit")), 50, "pageSize")
	if err != nil {
		return err
	}
	isEnabled, err := parseOptionalBool(r.URL.Query().Get("isEnabled"))
	if err != nil {
		return err
	}
	tags, total, err := h.service.ListTagsByProject(r.Context(), r.PathValue("projectId"), page, pageSize, r.URL.Query().Get("subscriptionId"), isEnabled)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
		"list": tags,
		"pagination": map[string]int{
			"page":       page,
			"pageSize":   pageSize,
			"total":      total,
			"totalPages": totalPages,
		},
	})
	return nil
}

// GetTag 返回单个变量。
func (h *MqttHandler) GetTag(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.GetTag(r.Context(), r.PathValue("projectId"), r.PathValue("tagId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// CreateTag 创建变量。
func (h *MqttHandler) CreateTag(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	tagInput, err := decodeMqttTagPayload(r)
	if err != nil {
		return err
	}
	result, err := h.service.CreateTag(r.Context(), r.PathValue("projectId"), r.PathValue("subscriptionId"), claims.UserID, tagInput)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// CreateTagsBatch 批量创建变量。
func (h *MqttHandler) CreateTagsBatch(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Tags []map[string]any `json:"tags"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	inputs := make([]repository.CreateMqttTagParams, 0, len(request.Tags))
	for _, raw := range request.Tags {
		payload, convertErr := convertMqttTagPayload(raw)
		if convertErr != nil {
			return convertErr
		}
		inputs = append(inputs, payload)
	}
	result, err := h.service.CreateTagsBatch(r.Context(), r.PathValue("projectId"), r.PathValue("subscriptionId"), claims.UserID, inputs)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"tags": result})
	return nil
}

// UpdateTag 更新变量。
func (h *MqttHandler) UpdateTag(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	tagInput, err := decodeMqttTagUpdatePayload(r)
	if err != nil {
		return err
	}
	result, err := h.service.UpdateTag(r.Context(), r.PathValue("projectId"), r.PathValue("tagId"), claims.UserID, tagInput)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// DeleteTag 删除变量。
func (h *MqttHandler) DeleteTag(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	if err := h.service.DeleteTag(r.Context(), r.PathValue("projectId"), r.PathValue("tagId"), claims.UserID); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"deleted": true})
	return nil
}

// ToggleTag 切换变量启用状态。
func (h *MqttHandler) ToggleTag(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.ToggleTag(r.Context(), r.PathValue("projectId"), r.PathValue("tagId"), claims.UserID)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// UpdateTagsOrder 批量更新变量顺序。
func (h *MqttHandler) UpdateTagsOrder(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		TagIDs []string `json:"tagIds"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	if len(request.TagIDs) == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "tagIds 不能为空")
	}
	if err := h.service.UpdateTagsOrder(r.Context(), r.PathValue("projectId"), request.TagIDs, claims.UserID); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"updated": true})
	return nil
}

// GetTagValue 返回单个变量当前值。
func (h *MqttHandler) GetTagValue(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.GetTagValue(r.Context(), r.PathValue("projectId"), r.PathValue("tagId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// GetTagValues 返回多个变量当前值。
func (h *MqttHandler) GetTagValues(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var request struct {
		TagIDs []string `json:"tagIds"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.GetTagValues(r.Context(), r.PathValue("projectId"), request.TagIDs)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func parseOptionalBool(raw string) (*bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "布尔参数格式无效")
	}
	return &value, nil
}

func decodeMqttTagPayload(r *http.Request) (repository.CreateMqttTagParams, error) {
	var raw map[string]any
	if err := decodeJSONBody(r, &raw); err != nil {
		return repository.CreateMqttTagParams{}, err
	}
	return convertMqttTagPayload(raw)
}

func decodeMqttTagUpdatePayload(r *http.Request) (repository.UpdateMqttTagParams, error) {
	var raw map[string]any
	if err := decodeJSONBody(r, &raw); err != nil {
		return repository.UpdateMqttTagParams{}, err
	}
	payload, err := convertMqttTagPayload(raw)
	if err != nil {
		return repository.UpdateMqttTagParams{}, err
	}
	return repository.UpdateMqttTagParams{
		GroupID:      payload.GroupID,
		Name:         payload.Name,
		Code:         payload.Code,
		Description:  payload.Description,
		DataType:     payload.DataType,
		ParseType:    payload.ParseType,
		ParseRule:    payload.ParseRule,
		DefaultValue: payload.DefaultValue,
		Unit:         payload.Unit,
		Transform:    payload.Transform,
		Validation:   payload.Validation,
		IsEnabled:    payload.IsEnabled,
		Order:        payload.Order,
	}, nil
}

func convertMqttTagPayload(raw map[string]any) (repository.CreateMqttTagParams, error) {
	result := repository.CreateMqttTagParams{
		Validation: map[string]any{},
		IsEnabled:  true,
	}
	if value, ok := raw["groupId"].(string); ok && strings.TrimSpace(value) != "" {
		trimmed := strings.TrimSpace(value)
		result.GroupID = &trimmed
	}
	if value, ok := raw["name"].(string); ok {
		result.Name = value
	}
	if value, ok := raw["code"].(string); ok {
		result.Code = value
	}
	if value, ok := raw["description"].(string); ok {
		result.Description = &value
	}
	if value, ok := raw["dataType"].(string); ok {
		result.DataType = value
	}
	if value, ok := raw["parseType"].(string); ok {
		result.ParseType = value
	}
	if value, ok := raw["parseRule"].(string); ok {
		result.ParseRule = value
	}
	if value, ok := raw["defaultValue"].(string); ok {
		result.DefaultValue = &value
	}
	if value, ok := raw["unit"].(string); ok {
		result.Unit = &value
	}
	if value, ok := raw["transform"].(string); ok {
		result.Transform = &value
	}
	if value, ok := raw["validation"].(map[string]any); ok {
		result.Validation = value
	}
	if value, ok := raw["isEnabled"].(bool); ok {
		result.IsEnabled = value
	}
	switch value := raw["order"].(type) {
	case float64:
		result.Order = int(value)
	case int:
		result.Order = value
	}
	return result, nil
}
