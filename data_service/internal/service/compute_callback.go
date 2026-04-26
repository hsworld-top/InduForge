package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

type computeSDKCallbackServer struct {
	URL    string
	Token  string
	server *http.Server
}

type computeSQLCallbackRequest struct {
	Key        string         `json:"key"`
	Params     map[string]any `json:"params"`
	Parameters map[string]any `json:"parameters"`
}

type computeMQTTPublishRequest struct {
	Source  string `json:"source"`
	Topic   string `json:"topic"`
	Payload any    `json:"payload"`
}

func (s *ComputeService) startComputeSDKCallback(unit repository.ComputeUnitRecord) (*computeSDKCallbackServer, error) {
	if s == nil || (s.queries == nil && s.mqtt == nil) {
		return nil, nil
	}
	token, err := generateComputeCallbackToken()
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	callback := &computeSDKCallbackServer{Token: token}
	mux.HandleFunc("/sql/query", s.computeCallbackAuth(token, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeComputeCallbackError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var request computeSQLCallbackRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeComputeCallbackError(w, http.StatusBadRequest, "invalid sql query callback payload")
			return
		}
		params := cloneMap(request.Parameters)
		if len(params) == 0 {
			params = cloneMap(request.Params)
		}
		result, err := s.executeComputeSDKDynamicQuery(r.Context(), unit, request.Key, params)
		if err != nil {
			writeComputeCallbackServiceError(w, err)
			return
		}
		writeComputeCallbackJSON(w, http.StatusOK, map[string]any{"result": result})
	}))
	mux.HandleFunc("/mqtt/publish", s.computeCallbackAuth(token, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeComputeCallbackError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var request computeMQTTPublishRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			writeComputeCallbackError(w, http.StatusBadRequest, "invalid mqtt publish callback payload")
			return
		}
		effect, err := s.publishComputeSDKMQTT(r.Context(), unit, request)
		if err != nil {
			writeComputeCallbackServiceError(w, err)
			return
		}
		writeComputeCallbackJSON(w, http.StatusOK, effect)
	}))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "启动 compute SDK 回调失败", err)
	}
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}
	callback.server = server
	callback.URL = "http://" + listener.Addr().String()
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			// 本地回调服务只服务当前脚本生命周期，启动后的异常会通过脚本侧 HTTP 调用体现。
		}
	}()
	return callback, nil
}

func (c *computeSDKCallbackServer) Close() {
	if c == nil || c.server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = c.server.Shutdown(ctx)
}

func (s *ComputeService) computeCallbackAuth(token string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expected := "Bearer " + token
		if strings.TrimSpace(r.Header.Get("Authorization")) != expected {
			writeComputeCallbackError(w, http.StatusUnauthorized, "invalid compute callback token")
			return
		}
		next(w, r)
	}
}

func (s *ComputeService) executeComputeSDKDynamicQuery(ctx context.Context, unit repository.ComputeUnitRecord, key string, params map[string]any) (any, error) {
	if s == nil || s.queries == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "compute SDK 查询服务未初始化")
	}
	queryID := strings.TrimSpace(key)
	parameters := cloneMap(params)
	if binding, ok := extractComputeSQLBindings(unit.InputBindings)[queryID]; ok {
		queryID = binding.QueryID
		parameters = mergeComputeSDKQueryParameters(binding.Parameters, params)
	} else if _, err := uuid.Parse(queryID); err != nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "compute SDK 查询必须使用已声明别名或项目内查询 ID")
	}
	result, err := s.executeComputeSDKQuery(ctx, unit.ProjectID, computeSQLBinding{QueryID: queryID, Parameters: parameters})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func mergeComputeSDKQueryParameters(base map[string]any, override map[string]any) map[string]any {
	result := cloneMap(base)
	for key, value := range override {
		result[key] = value
	}
	return result
}

func (s *ComputeService) publishComputeSDKMQTT(ctx context.Context, unit repository.ComputeUnitRecord, request computeMQTTPublishRequest) (map[string]any, error) {
	source := strings.TrimSpace(request.Source)
	topic := strings.TrimSpace(request.Topic)
	effect := map[string]any{
		"type":      "mqtt.publish",
		"source":    source,
		"topic":     topic,
		"payload":   request.Payload,
		"accepted":  false,
		"published": false,
	}
	if s == nil || s.mqtt == nil {
		effect["reason"] = "compute MQTT 发布能力未初始化"
		return effect, nil
	}
	if source == "" || topic == "" {
		effect["reason"] = "source 和 topic 不能为空"
		return effect, nil
	}
	if !computeMqttPublishAllowed(unit, source, topic) {
		effect["reason"] = "当前计算单元未授权发布该 MQTT 主题"
		return effect, nil
	}

	connection, err := s.mqtt.GetConnectionDetail(ctx, unit.ProjectID, source)
	if err != nil {
		return nil, err
	}
	if !connection.IsEnabled || connection.Status == "disabled" {
		effect["reason"] = "MQTT 连接未启用"
		return effect, nil
	}
	if err := publishMQTTMessage(connection, topic, request.Payload); err != nil {
		return nil, err
	}
	effect["accepted"] = true
	effect["published"] = true
	effect["qos"] = connection.QOS
	return effect, nil
}

func computeMqttPublishAllowed(unit repository.ComputeUnitRecord, source, topic string) bool {
	for _, section := range []map[string]any{unit.TriggerConfig, unit.InputBindings} {
		if section == nil {
			continue
		}
		sideEffects, _ := section["sideEffects"].(map[string]any)
		if sideEffects == nil {
			sideEffects, _ = section["effects"].(map[string]any)
		}
		mqttPublish, _ := sideEffects["mqttPublish"].(map[string]any)
		if mqttPublish == nil {
			mqttPublish, _ = sideEffects["mqtt"].(map[string]any)
		}
		if !boolFromMap(mqttPublish, "enabled", "allow", "allowed") {
			continue
		}
		if !matchesComputeMQTTSource(source, mqttPublish) {
			continue
		}
		if !matchesOptionalTopic(topic, mqttPublish["topics"]) {
			continue
		}
		return true
	}
	return false
}

func matchesComputeMQTTSource(source string, config map[string]any) bool {
	_, hasSources := config["sources"]
	_, hasSourceIDs := config["sourceIds"]
	if !hasSources && !hasSourceIDs {
		return true
	}
	return (hasSources && matchesOptionalList(source, config["sources"])) ||
		(hasSourceIDs && matchesOptionalList(source, config["sourceIds"]))
}

func publishMQTTMessage(connection *repository.MqttConnectionDetailRecord, topic string, payload any) error {
	if connection == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT 连接不存在")
	}
	timeout := time.Duration(connection.ConnectTimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	options := mqtt.NewClientOptions()
	options.AddBroker(computeMQTTBrokerAddress(connection))
	clientID := strings.TrimSpace(optionalComputeStringValue(connection.ClientID))
	if clientID == "" {
		clientID = "compute-" + connection.ID
	}
	options.SetClientID(clientID)
	options.SetCleanSession(connection.CleanSession)
	if connection.Keepalive > 0 {
		options.SetKeepAlive(time.Duration(connection.Keepalive) * time.Second)
	}
	options.SetConnectTimeout(timeout)
	if username := optionalComputeStringValue(connection.Username); username != "" {
		options.SetUsername(username)
	}
	if password := optionalComputeStringValue(connection.Password); password != "" {
		options.SetPassword(password)
	}

	client := mqtt.NewClient(options)
	if token := client.Connect(); !token.WaitTimeout(timeout) {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT 连接超时")
	} else if err := token.Error(); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT 连接失败", err)
	}
	defer client.Disconnect(250)

	publishPayload, err := computeMQTTPayload(payload)
	if err != nil {
		return err
	}
	if token := client.Publish(topic, byte(connection.QOS), false, publishPayload); !token.WaitTimeout(timeout) {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT 发布超时")
	} else if err := token.Error(); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT 发布失败", err)
	}
	return nil
}

func computeMQTTPayload(payload any) (any, error) {
	switch typed := payload.(type) {
	case string:
		return typed, nil
	case []byte:
		return typed, nil
	default:
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT payload 序列化失败", err)
		}
		return payloadBytes, nil
	}
}

func computeMQTTBrokerAddress(connection *repository.MqttConnectionDetailRecord) string {
	broker := strings.TrimSpace(connection.BrokerURL)
	if strings.Contains(broker, "://") {
		return broker
	}
	protocol := strings.TrimSpace(connection.Protocol)
	if protocol == "" {
		protocol = "tcp"
	}
	if connection.Port > 0 {
		return fmt.Sprintf("%s://%s:%d", protocol, broker, connection.Port)
	}
	return protocol + "://" + broker
}

func boolFromMap(input map[string]any, keys ...string) bool {
	for _, key := range keys {
		switch value := input[key].(type) {
		case bool:
			return value
		case string:
			return strings.EqualFold(strings.TrimSpace(value), "true") || strings.TrimSpace(value) == "1"
		}
	}
	return false
}

func matchesOptionalList(value string, raw any) bool {
	values := stringListFromAny(raw)
	if len(values) == 0 {
		return true
	}
	for _, candidate := range values {
		if candidate == "*" || candidate == value {
			return true
		}
	}
	return false
}

func matchesOptionalTopic(topic string, raw any) bool {
	patterns := stringListFromAny(raw)
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		if mqttTopicMatches(pattern, topic) {
			return true
		}
	}
	return false
}

func mqttTopicMatches(pattern, topic string) bool {
	pattern = strings.TrimSpace(pattern)
	topic = strings.TrimSpace(topic)
	if pattern == "" {
		return false
	}
	if pattern == "*" || pattern == topic {
		return true
	}
	patternParts := strings.Split(pattern, "/")
	topicParts := strings.Split(topic, "/")
	for index, part := range patternParts {
		if part == "#" {
			return index == len(patternParts)-1
		}
		if index >= len(topicParts) {
			return false
		}
		if part != "+" && part != topicParts[index] {
			return false
		}
	}
	return len(patternParts) == len(topicParts)
}

func stringListFromAny(raw any) []string {
	switch typed := raw.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{strings.TrimSpace(typed)}
	case []string:
		return typed
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(fmt.Sprintf("%v", item)); text != "" {
				values = append(values, text)
			}
		}
		return values
	default:
		return nil
	}
}

func optionalComputeStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func generateComputeCallbackToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "生成 compute SDK 回调令牌失败", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func writeComputeCallbackJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeComputeCallbackError(w http.ResponseWriter, status int, message string) {
	writeComputeCallbackJSON(w, status, map[string]any{"error": message})
}

func writeComputeCallbackServiceError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) && appErr.StatusCode > 0 {
		writeComputeCallbackError(w, appErr.StatusCode, appErr.Message)
		return
	}
	writeComputeCallbackError(w, http.StatusInternalServerError, err.Error())
}
