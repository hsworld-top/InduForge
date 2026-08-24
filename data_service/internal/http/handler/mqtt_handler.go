package handler

import (
	"net/http"
	"strconv"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// MqttHandler 负责承接 MQTT 领域相关 HTTP 请求。
type MqttHandler struct {
	service *service.MqttService
}

// NewMqttHandler 创建 MQTT 处理器。
func NewMqttHandler(mqttService *service.MqttService) *MqttHandler {
	return &MqttHandler{service: mqttService}
}

// CreateConnection 创建项目下的 MQTT 连接。
func (h *MqttHandler) CreateConnection(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name             string            `json:"name"`
		Status           string            `json:"status"`
		BrokerURL        string            `json:"brokerUrl"`
		Protocol         string            `json:"protocol"`
		Port             *int              `json:"port"`
		ClientID         *string           `json:"clientId"`
		Username         *string           `json:"username"`
		Password         *string           `json:"password"`
		Secrets          map[string]string `json:"secrets"`
		ClearSecretKeys  []string          `json:"clearSecretKeys"`
		Keepalive        *int              `json:"keepalive"`
		CleanSession     *bool             `json:"cleanSession"`
		QOS              *int              `json:"qos"`
		ReconnectPeriod  *int              `json:"reconnectPeriod"`
		ConnectTimeoutMS *int              `json:"connectTimeout"`
		Will             map[string]any    `json:"will"`
		SSLConfig        map[string]any    `json:"sslConfig"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateConnection(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateMqttConnectionInput{
		Name:             request.Name,
		Status:           request.Status,
		BrokerURL:        request.BrokerURL,
		Protocol:         request.Protocol,
		Port:             request.Port,
		ClientID:         request.ClientID,
		Username:         request.Username,
		Password:         request.Password,
		Secrets:          request.Secrets,
		ClearSecretKeys:  request.ClearSecretKeys,
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

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// StartConnection 将 MQTT 连接状态切到 connected。
func (h *MqttHandler) StartConnection(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	status, err := h.service.StartConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), status)
	return nil
}

// GetConnectionStatus 读取 MQTT 连接状态。
func (h *MqttHandler) GetConnectionStatus(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	status, err := h.service.GetConnectionStatus(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), status)
	return nil
}

// PublishMessage 使用已保存的 MQTT 连接发布一条测试消息。
func (h *MqttHandler) PublishMessage(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	var request struct {
		Topic   string `json:"topic"`
		Payload any    `json:"payload"`
		QOS     *int   `json:"qos"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	result, err := h.service.PublishMessage(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), service.MqttPublishInput{
		Topic:   request.Topic,
		Payload: request.Payload,
		QOS:     request.QOS,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// ListMessages 读取指定订阅的消息缓存列表。
func (h *MqttHandler) ListMessages(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "limit 参数格式无效", err)
		}
		limit = parsed
	}

	result, err := h.service.ListMessages(r.Context(), r.PathValue("projectId"), r.PathValue("subscriptionId"), limit)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{
		"list": result.Messages,
		"pagination": map[string]int{
			"page":       1,
			"pageSize":   result.Limit,
			"total":      len(result.Messages),
			"totalPages": 1,
		},
	})
	return nil
}

// ClearMessages 清空指定订阅的消息缓存。
func (h *MqttHandler) ClearMessages(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	if err := h.service.ClearMessages(r.Context(), r.PathValue("projectId"), r.PathValue("subscriptionId")); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]bool{"cleared": true})
	return nil
}
