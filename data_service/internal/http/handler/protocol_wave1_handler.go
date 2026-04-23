package handler

import (
	"net/http"

	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// ProtocolWave1Handler 负责第一波协议（kafka/http/ws/redis）接口。
type ProtocolWave1Handler struct {
	service *service.ProtocolWave1Service
}

// NewProtocolWave1Handler 创建第一波协议处理器。
func NewProtocolWave1Handler(protocolService *service.ProtocolWave1Service) *ProtocolWave1Handler {
	return &ProtocolWave1Handler{service: protocolService}
}

// CreateKafkaConfig 创建 Kafka Source 配置。
func (h *ProtocolWave1Handler) CreateKafkaConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name          string         `json:"name"`
		Status        string         `json:"status"`
		Brokers       string         `json:"brokers"`
		Topic         string         `json:"topic"`
		ConsumerGroup string         `json:"consumerGroup"`
		StartPosition string         `json:"startPosition"`
		Options       map[string]any `json:"options"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateKafkaConfig(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateKafkaConfigInput{
		Name:          request.Name,
		Status:        request.Status,
		Brokers:       request.Brokers,
		Topic:         request.Topic,
		ConsumerGroup: request.ConsumerGroup,
		StartPosition: request.StartPosition,
		Options:       request.Options,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// PreviewKafkaTopic 预览 Kafka Topic 数据。
func (h *ProtocolWave1Handler) PreviewKafkaTopic(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}

	rows, err := h.service.PreviewKafkaTopic(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), rows)
	return nil
}

// CreateHTTPConfig 创建 HTTP Source 配置。
func (h *ProtocolWave1Handler) CreateHTTPConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name         string         `json:"name"`
		Status       string         `json:"status"`
		BaseURL      string         `json:"baseUrl"`
		Method       string         `json:"method"`
		Headers      map[string]any `json:"headers"`
		TimeoutMS    *int           `json:"timeoutMs"`
		BodyTemplate map[string]any `json:"bodyTemplate"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateHTTPConfig(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateHTTPConfigInput{
		Name:         request.Name,
		Status:       request.Status,
		BaseURL:      request.BaseURL,
		Method:       request.Method,
		Headers:      request.Headers,
		TimeoutMS:    request.TimeoutMS,
		BodyTemplate: request.BodyTemplate,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// CreateWebSocketConfig 创建 WebSocket Source 配置。
func (h *ProtocolWave1Handler) CreateWebSocketConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name                string         `json:"name"`
		Status              string         `json:"status"`
		URL                 string         `json:"url"`
		Topic               *string        `json:"topic"`
		Headers             map[string]any `json:"headers"`
		HeartbeatIntervalMS *int           `json:"heartbeatIntervalMs"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateWebSocketConfig(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateWebSocketConfigInput{
		Name:                request.Name,
		Status:              request.Status,
		URL:                 request.URL,
		Topic:               request.Topic,
		Headers:             request.Headers,
		HeartbeatIntervalMS: request.HeartbeatIntervalMS,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// CreateRedisConfig 创建 Redis Source 配置。
func (h *ProtocolWave1Handler) CreateRedisConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name       string         `json:"name"`
		Status     string         `json:"status"`
		Address    string         `json:"address"`
		DB         *int           `json:"db"`
		Username   *string        `json:"username"`
		Password   *string        `json:"password"`
		KeyPattern string         `json:"keyPattern"`
		Mode       string         `json:"mode"`
		Options    map[string]any `json:"options"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateRedisConfig(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateRedisConfigInput{
		Name:       request.Name,
		Status:     request.Status,
		Address:    request.Address,
		DB:         request.DB,
		Username:   request.Username,
		Password:   request.Password,
		KeyPattern: request.KeyPattern,
		Mode:       request.Mode,
		Options:    request.Options,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}
