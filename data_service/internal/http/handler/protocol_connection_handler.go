package handler

import (
	"context"
	"net/http"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// ProtocolConnectionHandler 负责协议连接（kafka/http/ws/redis）接口。
type ProtocolConnectionHandler struct {
	service        *service.ProtocolConnectionService
	previewService protocolPreviewService
}

type protocolPreviewService interface {
	Preview(ctx context.Context, projectID, connectionID string, input service.ProtocolPreviewInput) (*service.ProtocolPreviewResult, error)
}

// NewProtocolConnectionHandler 创建协议连接处理器。
func NewProtocolConnectionHandler(protocolService *service.ProtocolConnectionService, previewService protocolPreviewService) *ProtocolConnectionHandler {
	return &ProtocolConnectionHandler{service: protocolService, previewService: previewService}
}

// CreateKafkaConfig 创建 Kafka Source 配置。
func (h *ProtocolConnectionHandler) CreateKafkaConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name            string            `json:"name"`
		Enabled         *bool             `json:"enabled"`
		Brokers         string            `json:"brokers"`
		Topic           string            `json:"topic"`
		ConsumerGroup   string            `json:"consumerGroup"`
		StartPosition   string            `json:"startPosition"`
		Options         map[string]any    `json:"options"`
		Secrets         map[string]string `json:"secrets"`
		ClearSecretKeys []string          `json:"clearSecretKeys"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateKafkaConfig(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateKafkaConfigInput{
		Name:          request.Name,
		Enabled:       request.Enabled,
		Brokers:       request.Brokers,
		Topic:         request.Topic,
		ConsumerGroup: request.ConsumerGroup,
		StartPosition: request.StartPosition,
		Options:       request.Options,
		Secrets:       request.Secrets, ClearSecretKeys: request.ClearSecretKeys,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// PreviewProtocol 统一执行开发态协议短时真实抓样。
func (h *ProtocolConnectionHandler) PreviewProtocol(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if h.previewService == nil {
		return normalizeRepresentativeHandlerError(apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "协议预览服务未初始化"))
	}

	var request service.ProtocolPreviewInput
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.previewService.Preview(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), request)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

// CreateHTTPConfig 创建 HTTP Source 配置。
func (h *ProtocolConnectionHandler) CreateHTTPConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name        string `json:"name"`
		Enabled     *bool  `json:"enabled"`
		Description string `json:"description"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateHTTPConfig(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateHTTPConfigInput{
		Name:        request.Name,
		Enabled:     request.Enabled,
		Description: request.Description,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// CreateWebSocketConfig 创建 WebSocket Source 配置。
func (h *ProtocolConnectionHandler) CreateWebSocketConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name        string `json:"name"`
		Enabled     *bool  `json:"enabled"`
		Description string `json:"description"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateWebSocketConfig(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateWebSocketConfigInput{
		Name:        request.Name,
		Enabled:     request.Enabled,
		Description: request.Description,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

// CreateRedisConfig 创建 Redis Source 配置。
func (h *ProtocolConnectionHandler) CreateRedisConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name            string            `json:"name"`
		Enabled         *bool             `json:"enabled"`
		Address         string            `json:"address"`
		DB              *int              `json:"db"`
		Username        *string           `json:"username"`
		Password        *string           `json:"password"`
		KeyPattern      string            `json:"keyPattern"`
		Mode            string            `json:"mode"`
		Options         map[string]any    `json:"options"`
		Secrets         map[string]string `json:"secrets"`
		ClearSecretKeys []string          `json:"clearSecretKeys"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateRedisConfig(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateRedisConfigInput{
		Name:       request.Name,
		Enabled:    request.Enabled,
		Address:    request.Address,
		DB:         request.DB,
		Username:   request.Username,
		Password:   request.Password,
		KeyPattern: request.KeyPattern,
		Mode:       request.Mode,
		Options:    request.Options,
		Secrets:    request.Secrets, ClearSecretKeys: request.ClearSecretKeys,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), connection)
	return nil
}

func (h *ProtocolConnectionHandler) UpdateKafkaConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Name            string            `json:"name"`
		Enabled         *bool             `json:"enabled"`
		Brokers         string            `json:"brokers"`
		Topic           string            `json:"topic"`
		ConsumerGroup   string            `json:"consumerGroup"`
		StartPosition   string            `json:"startPosition"`
		Options         map[string]any    `json:"options"`
		Secrets         map[string]string `json:"secrets"`
		ClearSecretKeys []string          `json:"clearSecretKeys"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateKafkaConfig(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.CreateKafkaConfigInput{Name: request.Name, Enabled: request.Enabled, Brokers: request.Brokers, Topic: request.Topic, ConsumerGroup: request.ConsumerGroup, StartPosition: request.StartPosition, Options: request.Options, Secrets: request.Secrets, ClearSecretKeys: request.ClearSecretKeys})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
func (h *ProtocolConnectionHandler) UpdateHTTPConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Name        string `json:"name"`
		Enabled     *bool  `json:"enabled"`
		Description string `json:"description"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateHTTPConfig(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.CreateHTTPConfigInput{Name: request.Name, Enabled: request.Enabled, Description: request.Description})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
func (h *ProtocolConnectionHandler) UpdateWebSocketConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Name        string `json:"name"`
		Enabled     *bool  `json:"enabled"`
		Description string `json:"description"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateWebSocketConfig(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.CreateWebSocketConfigInput{Name: request.Name, Enabled: request.Enabled, Description: request.Description})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
func (h *ProtocolConnectionHandler) UpdateRedisConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	var request struct {
		Name            string            `json:"name"`
		Enabled         *bool             `json:"enabled"`
		Address         string            `json:"address"`
		DB              *int              `json:"db"`
		Username        *string           `json:"username"`
		Password        *string           `json:"password"`
		KeyPattern      string            `json:"keyPattern"`
		Mode            string            `json:"mode"`
		Options         map[string]any    `json:"options"`
		Secrets         map[string]string `json:"secrets"`
		ClearSecretKeys []string          `json:"clearSecretKeys"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.UpdateRedisConfig(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, service.CreateRedisConfigInput{Name: request.Name, Enabled: request.Enabled, Address: request.Address, DB: request.DB, Username: request.Username, Password: request.Password, KeyPattern: request.KeyPattern, Mode: request.Mode, Options: request.Options, Secrets: request.Secrets, ClearSecretKeys: request.ClearSecretKeys})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
