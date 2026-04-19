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

	connection, err := h.service.CreateConnection(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateMqttConnectionInput{
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
		return err
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
		return err
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
		return err
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), status)
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

	messages, err := h.service.ListMessages(r.Context(), r.PathValue("projectId"), r.PathValue("subscriptionId"), limit)
	if err != nil {
		return err
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), messages)
	return nil
}
