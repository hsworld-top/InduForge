package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

const maxProtocolPreviewPayloadBytes = 64 * 1024

// NewDefaultProtocolPreviewAdapters 返回生产环境默认启用的 Phase 1 短时抓样 adapter。
func NewDefaultProtocolPreviewAdapters() map[string]ProtocolPreviewAdapter {
	return map[string]ProtocolPreviewAdapter{
		"kafka":     KafkaPreviewAdapter{},
		"http":      HTTPPreviewAdapter{Client: http.DefaultClient},
		"websocket": WebSocketPreviewAdapter{},
		"redis":     RedisPreviewAdapter{},
	}
}

// HTTPPreviewAdapter 对 HTTP Source 执行一次真实请求并返回响应样本。
type HTTPPreviewAdapter struct {
	Client *http.Client
}

func (a HTTPPreviewAdapter) Preview(ctx context.Context, input ProtocolPreviewAdapterInput) (*ProtocolPreviewResult, error) {
	startedAt := time.Now()
	config := input.Connection.Config
	baseURL := strings.TrimSpace(toString(config["baseUrl"]))
	if baseURL == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP preview baseUrl 不能为空")
	}
	method := strings.ToUpper(strings.TrimSpace(toString(config["method"])))
	if method == "" {
		method = http.MethodGet
	}

	var body io.Reader
	if rawBody, ok := config["bodyTemplate"]; ok && rawBody != nil {
		payload, err := json.Marshal(rawBody)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP preview bodyTemplate 格式无效", err)
		}
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, baseURL, body)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "创建 HTTP preview 请求失败", err)
	}
	for key, value := range mapFromAny(config["headers"]) {
		if strings.TrimSpace(key) == "" {
			continue
		}
		req.Header.Set(key, toString(value))
	}
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	client := a.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "HTTP preview 请求失败", err)
	}
	defer resp.Body.Close()

	payload, truncated, err := readPreviewPayload(resp.Body)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "读取 HTTP preview 响应失败", err)
	}
	samples, schema, rawPayload := payloadToSamples(payload, input.Limit)
	if len(samples) < input.Limit && truncated {
		truncated = true
	}
	return &ProtocolPreviewResult{
		Protocol:     "http",
		ConnectionID: input.Connection.ID,
		Status:       "ok",
		Schema:       schema,
		Samples:      samples,
		RawPayload:   rawPayload,
		Diagnostics: map[string]any{
			"statusCode":  resp.StatusCode,
			"contentType": resp.Header.Get("Content-Type"),
			"headers":     sanitizePreviewHeaders(resp.Header),
		},
		DurationMS: time.Since(startedAt).Milliseconds(),
		Truncated:  truncated,
	}, nil
}

// WebSocketPreviewAdapter 建立短连接，读取有限条消息后立即释放。
type WebSocketPreviewAdapter struct{}

func (a WebSocketPreviewAdapter) Preview(ctx context.Context, input ProtocolPreviewAdapterInput) (*ProtocolPreviewResult, error) {
	startedAt := time.Now()
	url := strings.TrimSpace(toString(input.Connection.Config["url"]))
	if url == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket preview url 不能为空")
	}
	headers := http.Header{}
	for key, value := range mapFromAny(input.Connection.Config["headers"]) {
		headers.Set(key, toString(value))
	}
	dialer := websocket.Dialer{
		HandshakeTimeout: input.Timeout,
		TLSClientConfig:  &tls.Config{MinVersion: tls.VersionTLS12},
	}
	conn, resp, err := dialer.DialContext(ctx, url, headers)
	if err != nil {
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket preview 连接失败", errWithStatus(err, statusCode))
	}
	defer conn.Close()

	if subscribeMessage, ok := input.Options["subscribeMessage"]; ok && subscribeMessage != nil {
		if err := writeWebSocketPreviewMessage(conn, subscribeMessage); err != nil {
			return nil, err
		}
	}

	samples := make([]any, 0, input.Limit)
	for len(samples) < input.Limit {
		_ = conn.SetReadDeadline(time.Now().Add(input.Timeout))
		_, message, readErr := conn.ReadMessage()
		if readErr != nil {
			if len(samples) > 0 {
				break
			}
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket preview 读取消息失败", readErr)
		}
		sample, _, _ := decodePreviewPayload(message)
		samples = append(samples, sample)
	}
	return &ProtocolPreviewResult{
		Protocol:     "websocket",
		ConnectionID: input.Connection.ID,
		Status:       "ok",
		Schema:       inferPreviewSchema(samples),
		Samples:      samples,
		Diagnostics: map[string]any{
			"messageCount": len(samples),
		},
		DurationMS: time.Since(startedAt).Milliseconds(),
		Truncated:  len(samples) >= input.Limit,
	}, nil
}

// RedisPreviewAdapter 用 SCAN + TYPE 读取有限 key 的当前样本。
type RedisPreviewAdapter struct{}

func (a RedisPreviewAdapter) Preview(ctx context.Context, input ProtocolPreviewAdapterInput) (*ProtocolPreviewResult, error) {
	startedAt := time.Now()
	client, err := newRedisPreviewClient(input.Connection.Config, input.Options)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	pattern := strings.TrimSpace(toString(input.Connection.Config["keyPattern"]))
	if pattern == "" {
		pattern = "*"
	}
	keys, err := scanRedisPreviewKeys(ctx, client, pattern, input.Limit)
	if err != nil {
		return nil, err
	}
	samples := make([]any, 0, len(keys))
	for _, key := range keys {
		sample, sampleErr := readRedisPreviewValue(ctx, client, key)
		if sampleErr != nil {
			sample = map[string]any{"key": key, "error": sampleErr.Error()}
		}
		samples = append(samples, sample)
	}
	return &ProtocolPreviewResult{
		Protocol:     "redis",
		ConnectionID: input.Connection.ID,
		Status:       "ok",
		Schema:       inferPreviewSchema(samples),
		Samples:      samples,
		Diagnostics: map[string]any{
			"keyPattern":  pattern,
			"sampleCount": len(samples),
		},
		DurationMS: time.Since(startedAt).Milliseconds(),
		Truncated:  len(keys) >= input.Limit,
	}, nil
}

type kafkaPreviewReader interface {
	ReadMessage(ctx context.Context) (kafka.Message, error)
	Close() error
}

// KafkaPreviewAdapter 使用 kafka-go 临时 reader 抓取有限条消息，不复用运行态 consumerGroup。
type KafkaPreviewAdapter struct {
	ReaderFactory func(config kafka.ReaderConfig) kafkaPreviewReader
}

func (a KafkaPreviewAdapter) Preview(ctx context.Context, input ProtocolPreviewAdapterInput) (*ProtocolPreviewResult, error) {
	startedAt := time.Now()
	config := input.Connection.Config
	brokers := splitCSV(strings.TrimSpace(toString(config["brokers"])))
	if len(brokers) == 0 || strings.TrimSpace(brokers[0]) == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka preview brokers 不能为空")
	}
	topic := strings.TrimSpace(toString(input.Options["topic"]))
	if topic == "" {
		topic = strings.TrimSpace(toString(config["topic"]))
	}
	if topic == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka preview topic 不能为空")
	}
	if boolFromAny(input.Options["probe"]) {
		conn, err := (&kafka.Dialer{}).DialContext(ctx, "tcp", brokers[0])
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka preview 连接失败", err)
		}
		defer conn.Close()
		if _, err := conn.ReadPartitions(topic); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka preview Topic 探测失败", err)
		}
		return &ProtocolPreviewResult{
			Protocol:     "kafka",
			ConnectionID: input.Connection.ID,
			Status:       "ok",
			Schema:       map[string]any{},
			Samples:      []any{},
			Diagnostics: map[string]any{
				"topic":       topic,
				"brokerCount": len(brokers),
				"probe":       true,
			},
			DurationMS: time.Since(startedAt).Milliseconds(),
			Truncated:  false,
		}, nil
	}
	startPosition := strings.ToLower(strings.TrimSpace(toString(input.Options["startPosition"])))
	if startPosition == "" {
		startPosition = strings.ToLower(strings.TrimSpace(toString(config["startPosition"])))
	}
	startOffset := kafka.LastOffset
	if startPosition == "earliest" {
		startOffset = kafka.FirstOffset
	} else if startPosition == "offset" {
		if _, ok := input.Options["offset"]; !ok {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka offset 预览必须指定 offset")
		}
		startOffset = int64FromAny(input.Options["offset"], kafka.LastOffset)
	}
	if startPosition == "" {
		startPosition = "latest"
	}
	partitionMode := strings.ToLower(strings.TrimSpace(toString(input.Options["partitionMode"])))
	if partitionMode == "" {
		partitionMode = "all"
	}
	partition := intFromAny(input.Options["partition"], -1)
	if startPosition == "offset" && partitionMode != "single" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka offset 预览必须指定单分区")
	}

	factory := a.ReaderFactory
	if factory == nil {
		factory = func(config kafka.ReaderConfig) kafkaPreviewReader {
			return kafka.NewReader(config)
		}
	}
	readerConfig := kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		StartOffset: startOffset,
		MinBytes:    1,
		MaxBytes:    maxProtocolPreviewPayloadBytes,
		MaxWait:     input.Timeout,
	}
	if partitionMode == "single" {
		if partition < 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka 单分区预览 partition 不能为空")
		}
		readerConfig.Partition = partition
	} else {
		readerConfig.GroupID = fmt.Sprintf("data-service-preview-%s-%d", input.Connection.ID, time.Now().UnixNano())
	}
	reader := factory(readerConfig)
	defer reader.Close()

	samples := make([]any, 0, input.Limit)
	decode := normalizeKafkaDecode(toString(input.Options["decode"]))
	for len(samples) < input.Limit {
		message, err := reader.ReadMessage(ctx)
		if err != nil {
			if len(samples) > 0 && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) {
				break
			}
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka preview 读取消息失败", err)
		}
		samples = append(samples, kafkaMessagePreviewSample(message, decode))
	}

	return &ProtocolPreviewResult{
		Protocol:     "kafka",
		ConnectionID: input.Connection.ID,
		Status:       "ok",
		Schema:       inferPreviewSchema(samples),
		Samples:      samples,
		Diagnostics: map[string]any{
			"topic":         topic,
			"brokerCount":   len(brokers),
			"startPosition": startPosition,
			"partitionMode": partitionMode,
			"partition":     partition,
			"sampleCount":   len(samples),
		},
		DurationMS: time.Since(startedAt).Milliseconds(),
		Truncated:  len(samples) >= input.Limit,
	}, nil
}

func kafkaMessagePreviewSample(message kafka.Message, decode string) map[string]any {
	value, rawPayload := decodeKafkaPreviewPayload(message.Value, decode)
	return map[string]any{
		"topic":      message.Topic,
		"partition":  message.Partition,
		"offset":     message.Offset,
		"key":        string(message.Key),
		"headers":    sanitizeKafkaPreviewHeaders(message.Headers),
		"value":      value,
		"rawPayload": rawPayload,
		"timestamp":  message.Time,
	}
}

func decodeKafkaPreviewPayload(payload []byte, decode string) (any, string) {
	switch normalizeKafkaDecode(decode) {
	case "string":
		text := string(payload)
		return text, text
	case "binary":
		encoded := base64.StdEncoding.EncodeToString(payload)
		return encoded, encoded
	default:
		value, _, rawPayload := decodePreviewPayload(payload)
		return value, rawPayload
	}
}

func sanitizeKafkaPreviewHeaders(headers []kafka.Header) map[string]string {
	result := map[string]string{}
	for _, header := range headers {
		key := strings.TrimSpace(header.Key)
		if key == "" {
			continue
		}
		if isSensitiveConfigKey(key) {
			result[key] = "******"
			continue
		}
		result[key] = string(header.Value)
	}
	return result
}

func readPreviewPayload(reader io.Reader) ([]byte, bool, error) {
	limited := io.LimitReader(reader, maxProtocolPreviewPayloadBytes+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return nil, false, err
	}
	if len(payload) > maxProtocolPreviewPayloadBytes {
		return payload[:maxProtocolPreviewPayloadBytes], true, nil
	}
	return payload, false, nil
}

func payloadToSamples(payload []byte, limit int) ([]any, map[string]any, any) {
	decoded, _, textPayload := decodePreviewPayload(payload)
	var samples []any
	if list, ok := decoded.([]any); ok {
		if len(list) > limit {
			list = list[:limit]
		}
		samples = list
	} else {
		samples = []any{decoded}
	}
	return samples, inferPreviewSchema(samples), textPayload
}

func decodePreviewPayload(payload []byte) (any, bool, string) {
	text := string(payload)
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err == nil {
		return decoded, true, text
	}
	return text, false, text
}

func inferPreviewSchema(samples []any) map[string]any {
	if len(samples) == 0 {
		return map[string]any{"type": "empty"}
	}
	return map[string]any{"type": previewValueType(samples[0])}
}

func previewValueType(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		properties := map[string]string{}
		for key, field := range typed {
			properties[key] = previewValueType(field)
		}
		payload, _ := json.Marshal(properties)
		return "object:" + string(payload)
	case []any:
		return "array"
	case string:
		return "string"
	case float64, float32, int, int64, int32:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

func sanitizePreviewHeaders(headers http.Header) map[string]string {
	result := map[string]string{}
	for key, values := range headers {
		canonical := textproto.CanonicalMIMEHeaderKey(key)
		if isSensitiveConfigKey(canonical) {
			result[canonical] = "******"
			continue
		}
		if len(values) > 0 {
			result[canonical] = values[0]
		}
	}
	return result
}

func writeWebSocketPreviewMessage(conn *websocket.Conn, value any) error {
	if text, ok := value.(string); ok {
		return conn.WriteMessage(websocket.TextMessage, []byte(text))
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "WebSocket subscribeMessage 格式无效", err)
	}
	return conn.WriteMessage(websocket.TextMessage, payload)
}

func newRedisPreviewClient(config map[string]any, options map[string]any) (redis.UniversalClient, error) {
	address := strings.TrimSpace(toString(config["address"]))
	if address == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Redis preview address 不能为空")
	}
	mode := strings.ToLower(strings.TrimSpace(toString(config["mode"])))
	addrs := splitCSV(address)
	db := intFromAny(config["db"], 0)
	username := strings.TrimSpace(toString(config["username"]))
	password := strings.TrimSpace(toString(config["password"]))
	switch mode {
	case "cluster":
		return redis.NewClusterClient(&redis.ClusterOptions{Addrs: addrs, Username: username, Password: password}), nil
	case "sentinel":
		masterName := strings.TrimSpace(toString(options["masterName"]))
		if masterName == "" {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Redis sentinel preview 需要 options.masterName")
		}
		return redis.NewFailoverClient(&redis.FailoverOptions{MasterName: masterName, SentinelAddrs: addrs, DB: db, Username: username, Password: password}), nil
	default:
		return redis.NewClient(&redis.Options{Addr: address, DB: db, Username: username, Password: password}), nil
	}
}

func scanRedisPreviewKeys(ctx context.Context, client redis.UniversalClient, pattern string, limit int) ([]string, error) {
	keys := make([]string, 0, limit)
	var cursor uint64
	for len(keys) < limit {
		batch, nextCursor, err := client.Scan(ctx, cursor, pattern, int64(limit-len(keys))).Result()
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Redis preview 扫描 key 失败", err)
		}
		keys = append(keys, batch...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	if len(keys) > limit {
		keys = keys[:limit]
	}
	return keys, nil
}

func readRedisPreviewValue(ctx context.Context, client redis.UniversalClient, key string) (map[string]any, error) {
	keyType, err := client.Type(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	sample := map[string]any{"key": key, "type": keyType}
	switch keyType {
	case "string":
		sample["value"], err = client.Get(ctx, key).Result()
	case "hash":
		sample["value"], err = client.HGetAll(ctx, key).Result()
	case "list":
		sample["value"], err = client.LRange(ctx, key, 0, 9).Result()
	case "set":
		sample["value"], err = client.SMembers(ctx, key).Result()
	case "zset":
		sample["value"], err = client.ZRangeWithScores(ctx, key, 0, 9).Result()
	default:
		sample["value"] = nil
	}
	return sample, err
}

func mapFromAny(value any) map[string]any {
	if mapped, ok := value.(map[string]any); ok {
		return mapped
	}
	return map[string]any{}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		result = []string{strings.TrimSpace(value)}
	}
	return result
}

func intFromAny(value any, defaultValue int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return int(parsed)
		}
	}
	return defaultValue
}

func int64FromAny(value any, defaultValue int64) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}

func errWithStatus(err error, statusCode int) error {
	if statusCode == 0 {
		return err
	}
	return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, err.Error())
}
