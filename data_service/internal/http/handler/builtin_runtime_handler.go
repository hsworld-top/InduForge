package handler

import (
	"net/http"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

// BuiltinRuntimeHandler 负责承接 IF 内置运行库开发态测试请求。
type BuiltinRuntimeHandler struct {
	service     *service.BuiltinRuntimeService
	connections *service.ConnectionService
}

func NewBuiltinRuntimeHandler(runtimeService *service.BuiltinRuntimeService, connectionService ...*service.ConnectionService) *BuiltinRuntimeHandler {
	handler := &BuiltinRuntimeHandler{service: runtimeService}
	if len(connectionService) > 0 {
		handler.connections = connectionService[0]
	}
	return handler
}

func (h *BuiltinRuntimeHandler) ExecuteRelationSQL(w http.ResponseWriter, r *http.Request) error {
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请使用连接级 IF关系库 SQL 工作台")
}

func (h *BuiltinRuntimeHandler) QueryTimeseries(w http.ResponseWriter, r *http.Request) error {
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请使用连接级 IF时序库 SQL 工作台")
}

func (h *BuiltinRuntimeHandler) SampleTimeseries(w http.ResponseWriter, r *http.Request) error {
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请使用连接级 IF时序库 SQL 工作台")
}

func (h *BuiltinRuntimeHandler) SetRealtimeKey(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var request struct {
		Key         string `json:"key"`
		Value       any    `json:"value"`
		TtlSeconds  int    `json:"ttlSeconds"`
		ValueType   string `json:"valueType"`
		Description string `json:"description"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	runtimeKey, err := h.runtimeKeyForConnection(r, "builtin.realtime")
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	result, err := h.service.SetRealtimeKey(r.Context(), r.PathValue("projectId"), service.BuiltinRealtimeSetInput{
		Key:        request.Key,
		Value:      request.Value,
		TtlSeconds: request.TtlSeconds,
		RuntimeKey: runtimeKey,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	if _, err := h.service.UpsertRealtimeKeyDefinition(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), request.Key, request.ValueType, request.TtlSeconds, request.Description); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *BuiltinRuntimeHandler) GetRealtimeKey(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	runtimeKey, err := h.runtimeKeyForConnection(r, "builtin.realtime")
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	result, err := h.service.GetRealtimeKey(r.Context(), r.PathValue("projectId"), runtimeKey, r.PathValue("key"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *BuiltinRuntimeHandler) DeleteRealtimeKey(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	runtimeKey, err := h.runtimeKeyForConnection(r, "builtin.realtime")
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	result, err := h.service.DeleteRealtimeKey(r.Context(), r.PathValue("projectId"), runtimeKey, r.PathValue("key"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *BuiltinRuntimeHandler) PublishMessage(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var request struct {
		Topic   string `json:"topic"`
		Payload any    `json:"payload"`
		QOS     byte   `json:"qos"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	runtimeKey, err := h.runtimeKeyForConnection(r, "builtin.message")
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	result, err := h.service.PublishMessage(r.Context(), r.PathValue("projectId"), service.BuiltinMessagePublishInput{
		ConnectionID: r.PathValue("connectionId"),
		Topic:        request.Topic,
		Payload:      request.Payload,
		QOS:          request.QOS,
		RuntimeKey:   runtimeKey,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *BuiltinRuntimeHandler) ListRealtimeKeys(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if _, err := h.runtimeKeyForConnection(r, "builtin.realtime"); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	keys, err := h.service.ListRealtimeKeyDefinitions(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": keys})
	return nil
}

func (h *BuiltinRuntimeHandler) ListMessageTopics(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if _, err := h.runtimeKeyForConnection(r, "builtin.message"); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	topics, err := h.service.ListMessageTopics(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": topics})
	return nil
}

func (h *BuiltinRuntimeHandler) CreateMessageTopic(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if _, err := h.runtimeKeyForConnection(r, "builtin.message"); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	var request struct {
		Topic       string `json:"topic"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	topic, err := h.service.CreateMessageTopic(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), request.Topic, request.Name, request.Description)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), topic)
	return nil
}

func (h *BuiltinRuntimeHandler) ListMessageVariables(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if _, err := h.runtimeKeyForConnection(r, "builtin.message"); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	variables, err := h.service.ListMessageVariables(r.Context(), r.PathValue("projectId"), r.PathValue("topicId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), map[string]any{"list": variables})
	return nil
}

func (h *BuiltinRuntimeHandler) CreateMessageVariable(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	if _, err := h.runtimeKeyForConnection(r, "builtin.message"); err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	var request struct {
		Name            string `json:"name"`
		PayloadPath     string `json:"payloadPath"`
		ValueType       string `json:"valueType"`
		Unit            string `json:"unit"`
		Description     string `json:"description"`
		CreateDatapoint bool   `json:"createDatapoint"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	variable, err := h.service.CreateMessageVariable(r.Context(), r.PathValue("projectId"), r.PathValue("topicId"), request.Name, request.PayloadPath, request.ValueType, request.Unit, request.Description, request.CreateDatapoint)
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), variable)
	return nil
}

func (h *BuiltinRuntimeHandler) CreateMessagePreviewSession(w http.ResponseWriter, r *http.Request) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	result, err := h.service.CreateMessagePreviewSession(r.Context(), r.PathValue("projectId"))
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *BuiltinRuntimeHandler) executeSQL(w http.ResponseWriter, r *http.Request, store string) error {
	if _, err := requireClaims(r); err != nil {
		return err
	}
	var request struct {
		SQL        string `json:"sql"`
		Parameters []any  `json:"parameters"`
		Limit      int    `json:"limit"`
	}
	if err := decodeJSONBody(r, &request); err != nil {
		return err
	}
	result, err := h.service.ExecuteSQL(r.Context(), r.PathValue("projectId"), service.BuiltinSQLExecuteInput{
		Store:      store,
		SQL:        request.SQL,
		Parameters: request.Parameters,
		Limit:      request.Limit,
	})
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}

func (h *BuiltinRuntimeHandler) runtimeKeyForConnection(r *http.Request, expectedType string) (string, error) {
	if h.connections == nil {
		return "", apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "内置运行库连接服务未初始化")
	}
	connection, err := h.connections.GetConnection(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), "")
	if err != nil {
		return "", err
	}
	if connection.Type != expectedType {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库类型不匹配")
	}
	runtimeKey, _ := connection.Config["runtimeKey"].(string)
	if runtimeKey == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库缺少系统标识")
	}
	return runtimeKey, nil
}
