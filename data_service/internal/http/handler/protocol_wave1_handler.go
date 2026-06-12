package handler

import (
	"context"
	"encoding/json"
	"net/http"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// ProtocolWave1Handler 负责第一波协议（kafka/http/ws/redis）接口。
type ProtocolWave1Handler struct {
	service        *service.ProtocolWave1Service
	previewService protocolPreviewService
}

type protocolPreviewService interface {
	Preview(ctx context.Context, projectID, connectionID string, input service.ProtocolPreviewInput) (*service.ProtocolPreviewResult, error)
}

// NewProtocolWave1Handler 创建第一波协议处理器。
func NewProtocolWave1Handler(protocolService *service.ProtocolWave1Service, previewServices ...protocolPreviewService) *ProtocolWave1Handler {
	handler := &ProtocolWave1Handler{service: protocolService}
	if len(previewServices) > 0 {
		handler.previewService = previewServices[0]
	}
	return handler
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

	if h.previewService != nil {
		result, err := h.previewService.Preview(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), service.ProtocolPreviewInput{})
		if err != nil {
			return normalizeRepresentativeHandlerError(err)
		}
		rows := make([]service.KafkaPreview, 0, len(result.Samples))
		for _, sample := range result.Samples {
			rows = append(rows, service.KafkaPreview{
				Topic:   toStringForHandler(result.Diagnostics["topic"]),
				Payload: toStringForHandler(sample),
			})
		}
		response.WriteSuccess(w, middleware.RequestID(r.Context()), rows)
		return nil
	}

	rows, err := h.service.PreviewKafkaTopic(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}

	response.WriteSuccess(w, middleware.RequestID(r.Context()), rows)
	return nil
}

// PreviewProtocol 统一执行 Phase 1 协议短时真实抓样。
func (h *ProtocolWave1Handler) PreviewProtocol(w http.ResponseWriter, r *http.Request) error {
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
func (h *ProtocolWave1Handler) CreateHTTPConfig(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}

	var request struct {
		Name        string `json:"name"`
		Status      string `json:"status"`
		Description string `json:"description"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateHTTPConfig(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateHTTPConfigInput{
		Name:        request.Name,
		Status:      request.Status,
		Description: request.Description,
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
		Name        string `json:"name"`
		Status      string `json:"status"`
		Description string `json:"description"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}

	connection, err := h.service.CreateWebSocketConfig(r.Context(), r.PathValue("projectId"), claims.UserID, service.CreateWebSocketConfigInput{
		Name:        request.Name,
		Status:      request.Status,
		Description: request.Description,
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

func toStringForHandler(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		payload, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(payload)
	}
}
