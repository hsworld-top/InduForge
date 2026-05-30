package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

// DataPointStatusInfo 表示设计器诊断面板使用的数据点状态摘要。
type DataPointStatusInfo struct {
	ID           string  `json:"id,omitempty"`
	Path         string  `json:"path"`
	Status       string  `json:"status"`
	DataType     string  `json:"dataType"`
	StatusReason *string `json:"statusReason,omitempty"`
}

// WriteDataPointResult 表示写入数据点后的响应结果。
type WriteDataPointResult struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Value     any       `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// GetDataPointStatuses 按 id 或 path 批量返回数据点状态。
func (s *DataPointService) GetDataPointStatuses(ctx context.Context, projectID string, ids, paths []string) ([]DataPointStatusInfo, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}

	uniqueIDs := uniqueStrings(ids)
	uniquePaths := uniqueStrings(paths)
	if len(uniqueIDs) == 0 && len(uniquePaths) == 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "ids 或 paths 至少需要提供一个")
	}

	records := make([]repository.DataPointRecord, 0, len(uniqueIDs)+len(uniquePaths))
	if len(uniqueIDs) > 0 {
		loaded, err := s.repository.GetByProjectAndIDs(ctx, projectID, uniqueIDs)
		if err != nil {
			return nil, err
		}
		records = append(records, loaded...)
	}
	if len(uniquePaths) > 0 {
		loaded, err := s.repository.ListByProjectAndPaths(ctx, projectID, uniquePaths)
		if err != nil {
			return nil, err
		}
		records = append(records, loaded...)
	}

	seen := make(map[string]struct{}, len(records))
	result := make([]DataPointStatusInfo, 0, len(records))
	for _, record := range records {
		if _, ok := seen[record.ID]; ok {
			continue
		}
		seen[record.ID] = struct{}{}

		info := DataPointStatusInfo{
			ID:       record.ID,
			Path:     record.Path,
			Status:   record.Status,
			DataType: record.DataType,
		}
		if record.Status == "invalid" {
			reason := deriveDataPointInvalidReason(record)
			info.StatusReason = &reason
		}
		result = append(result, info)
	}

	return result, nil
}

// GetDataPointValuesByIDs 按数据点主键批量返回当前值。
func (s *DataPointService) GetDataPointValuesByIDs(ctx context.Context, projectID string, ids []string) (map[string]any, error) {
	return s.GetDataPointValues(ctx, projectID, ids, nil)
}

// GetDataPointValues 按 id 或 path 批量返回当前值，返回 key 与调用方传入标识保持一致。
func (s *DataPointService) GetDataPointValues(ctx context.Context, projectID string, ids, paths []string) (map[string]any, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}

	uniqueIDs := uniqueStrings(ids)
	uniquePaths := uniqueStrings(paths)
	if len(uniqueIDs) == 0 && len(uniquePaths) == 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "datapointIds 或 paths 不能为空")
	}

	records := make([]repository.DataPointRecord, 0, len(uniqueIDs)+len(uniquePaths))
	if len(uniqueIDs) > 0 {
		loaded, err := s.repository.GetByProjectAndIDs(ctx, projectID, uniqueIDs)
		if err != nil {
			return nil, err
		}
		records = append(records, loaded...)
	}
	if len(uniquePaths) > 0 {
		loaded, err := s.repository.ListByProjectAndPaths(ctx, projectID, uniquePaths)
		if err != nil {
			return nil, err
		}
		records = append(records, loaded...)
	}

	values := make(map[string]any, len(records))
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		if _, ok := seen[record.ID]; ok {
			continue
		}
		seen[record.ID] = struct{}{}
		value, err := s.buildValueFromRecord(ctx, projectID, record)
		if err != nil {
			return nil, err
		}
		values[record.ID] = value.Value
		values[record.Path] = value.Value
	}
	return values, nil
}

// WriteDataPointValue 以“更新默认值”的方式承接设计态写入能力。
func (s *DataPointService) WriteDataPointValue(ctx context.Context, projectID, id, userID string, value any) (*WriteDataPointResult, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateDataPointID(id); err != nil {
		return nil, err
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}

	record, err := s.repository.GetByProjectAndID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}

	defaultValue := stringifyDataPointValue(value)
	updated, err := s.repository.Update(ctx, repository.UpdateDataPointParams{
		ID:                record.ID,
		ProjectID:         record.ProjectID,
		UserID:            userID,
		Name:              record.Name,
		Description:       cloneOptionalString(record.Description),
		SourceType:        record.SourceType,
		SourceID:          cloneOptionalString(record.SourceID),
		SourceConfig:      cloneMap(record.SourceConfig),
		DataType:          record.DataType,
		Unit:              cloneOptionalString(record.Unit),
		PrecisionNum:      cloneOptionalInt(record.PrecisionNum),
		DefaultValue:      &defaultValue,
		MinValue:          cloneOptionalFloat64(record.MinValue),
		MaxValue:          cloneOptionalFloat64(record.MaxValue),
		AlarmLow:          cloneOptionalFloat64(record.AlarmLow),
		AlarmHigh:         cloneOptionalFloat64(record.AlarmHigh),
		Tags:              cloneJSONArray(record.Tags),
		RefreshMode:       record.RefreshMode,
		RefreshIntervalMS: cloneOptionalInt(record.RefreshIntervalMS),
		Status:            record.Status,
	})
	if err != nil {
		return nil, err
	}

	return &WriteDataPointResult{
		ID:        updated.ID,
		Path:      updated.Path,
		Value:     value,
		Timestamp: time.Now().UTC(),
	}, nil
}

// GetDataPointUsages 返回数据点使用情况，当前先保持兼容返回空列表。
func (s *DataPointService) GetDataPointUsages(ctx context.Context, projectID, id string) ([]map[string]any, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateDataPointID(id); err != nil {
		return nil, err
	}
	if _, err := s.repository.GetByProjectAndID(ctx, projectID, id); err != nil {
		return nil, err
	}
	return []map[string]any{}, nil
}

func (s *DataPointService) buildValueFromRecord(ctx context.Context, projectID string, record repository.DataPointRecord) (*DataPointValue, error) {
	now := time.Now().UTC()
	value := DataPointValue{
		Path:      record.Path,
		Value:     defaultValueOrNil(record.DefaultValue),
		Quality:   "unknown",
		Timestamp: now,
		Status:    record.Status,
	}

	if record.SourceType == "db.query" && record.SourceID != nil && strings.TrimSpace(*record.SourceID) != "" {
		parameters, err := dataPointQueryParameters(record.SourceConfig)
		if err != nil {
			return nil, err
		}
		result, err := s.queries.ExecuteQueryForProject(ctx, projectID, *record.SourceID, ExecuteQueryInput{
			Parameters: parameters,
		})
		if err != nil {
			return nil, err
		}

		value.Value = result.Data
		value.Quality = "good"
		return &value, nil
	}

	if s.mqtt != nil && record.SourceID != nil && strings.TrimSpace(*record.SourceID) != "" {
		switch record.SourceType {
		case "mqtt.tag":
			tag, err := s.mqtt.GetTag(ctx, projectID, *record.SourceID)
			if err != nil {
				return nil, err
			}

			subscription, err := s.mqtt.GetSubscription(ctx, projectID, tag.SubscriptionID)
			if err != nil {
				return nil, err
			}
			messages, err := s.mqtt.ListMessages(ctx, projectID, subscription.ID, 1)
			if err != nil {
				return nil, err
			}
			if len(messages) > 0 {
				snapshot := BuildMqttTagSnapshotFromMessage(*tag, messages[0])
				value.Value = snapshot.Value
				value.Quality = snapshot.Quality
				value.Timestamp = snapshot.Timestamp
				return &value, nil
			}
		case "mqtt.subscription":
			messages, err := s.mqtt.ListMessages(ctx, projectID, *record.SourceID, 1)
			if err != nil {
				return nil, err
			}
			if len(messages) > 0 {
				snapshot := BuildMqttSubscriptionSnapshot(messages[0])
				value.Value = snapshot.Value
				value.Quality = snapshot.Quality
				value.Timestamp = snapshot.Timestamp
				return &value, nil
			}
		}
	}

	if value.Value != nil {
		value.Quality = "good"
	}
	if record.SourceType == "http.request" {
		return s.buildHTTPRequestValue(ctx, projectID, record, value)
	}
	if record.SourceType == "websocket.session" {
		return s.buildWebSocketSessionValue(ctx, projectID, record, value)
	}
	if record.SourceType == "kafka.field" {
		return s.buildKafkaFieldValue(ctx, projectID, record, value)
	}
	if record.SourceType == "realtime.key" {
		return s.buildRealtimeKeyValue(ctx, projectID, record, value)
	}
	return &value, nil
}

func (s *DataPointService) buildHTTPRequestValue(ctx context.Context, projectID string, record repository.DataPointRecord, value DataPointValue) (*DataPointValue, error) {
	requestID := strings.TrimSpace(firstString(record.SourceConfig, "requestId"))
	if s.httpWorkbench != nil && requestID != "" {
		request, err := s.httpWorkbench.GetRequest(ctx, projectID, requestID)
		if err != nil {
			return nil, err
		}
		if request.Quality != "" {
			value.Quality = request.Quality
		}
		if request.LastSentAt != nil {
			value.Timestamp = *request.LastSentAt
		}
		if request.Quality == "bad" {
			value.Value = request.LastResponse
			return &value, nil
		}
	}
	value.Value = parseStoredHTTPDefaultValue(record.DefaultValue)
	if value.Value != nil {
		value.Quality = "good"
	}
	return &value, nil
}

func (s *DataPointService) buildWebSocketSessionValue(ctx context.Context, projectID string, record repository.DataPointRecord, value DataPointValue) (*DataPointValue, error) {
	sessionID := strings.TrimSpace(firstString(record.SourceConfig, "sessionId"))
	if s.websocketWb != nil && sessionID != "" {
		session, err := s.websocketWb.GetSession(ctx, projectID, sessionID)
		if err != nil {
			return nil, err
		}
		if session.Quality != "" {
			value.Quality = session.Quality
		}
		if session.LastMessageAt != nil {
			value.Timestamp = *session.LastMessageAt
		}
		if session.Quality == "bad" {
			value.Value = map[string]any{
				"message":    session.LastMessage,
				"diagnostic": derefString(session.LastDiagnostic),
			}
			return &value, nil
		}
	}
	value.Value = parseStoredHTTPDefaultValue(record.DefaultValue)
	if value.Value != nil {
		value.Quality = "good"
	}
	return &value, nil
}

func (s *DataPointService) buildKafkaFieldValue(ctx context.Context, projectID string, record repository.DataPointRecord, value DataPointValue) (*DataPointValue, error) {
	fieldID := strings.TrimSpace(firstString(record.SourceConfig, "fieldId"))
	if s.kafkaWorkbench == nil || fieldID == "" {
		return &value, nil
	}
	field, err := s.kafkaWorkbench.GetField(ctx, projectID, fieldID)
	if err != nil {
		return nil, err
	}
	value.Value = field.LastValue
	value.Quality = fallbackTrimmed(field.Quality, "unknown")
	if field.LastUpdatedAt != nil {
		value.Timestamp = *field.LastUpdatedAt
	}
	return &value, nil
}

func (s *DataPointService) buildRealtimeKeyValue(ctx context.Context, projectID string, record repository.DataPointRecord, value DataPointValue) (*DataPointValue, error) {
	if s.connections == nil || record.SourceID == nil {
		return &value, nil
	}
	connectionID := strings.TrimSpace(*record.SourceID)
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	key := strings.TrimSpace(firstString(record.SourceConfig, "key"))
	if key == "" {
		key = record.Name
	}
	if connection.Type == "redis" {
		return s.buildRedisKeyValue(ctx, connection, key, value)
	}
	if connection.Type == "builtin.realtime" {
		return s.buildBuiltinRealtimeKeyValue(ctx, projectID, connection, key, value)
	}
	return &value, nil
}

func (s *DataPointService) buildRedisKeyValue(ctx context.Context, connection *repository.ConnectionRecord, key string, value DataPointValue) (*DataPointValue, error) {
	client, err := newRedisPreviewClient(connection.Config, mapFromAny(connection.Config["options"]))
	if err != nil {
		return nil, err
	}
	defer client.Close()
	keyType, err := client.Type(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if keyType == "none" {
		value.Quality = "bad"
		return &value, nil
	}
	current, err := readRealtimeRedisValue(ctx, client, key, keyType)
	if err != nil {
		return nil, err
	}
	value.Value = current
	value.Quality = "good"
	return &value, nil
}

func (s *DataPointService) buildBuiltinRealtimeKeyValue(ctx context.Context, projectID string, connection *repository.ConnectionRecord, key string, value DataPointValue) (*DataPointValue, error) {
	if s.builtinRuntime == nil || s.builtinRuntime.realtimeClient == nil {
		return &value, nil
	}
	fullKey, err := deriveBuiltinRealtimeKey(s.builtinRuntime.realtimeKeyPrefix, projectID, dataPointRuntimeKey(connection), key)
	if err != nil {
		return nil, err
	}
	raw, err := s.builtinRuntime.realtimeClient.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			value.Quality = "bad"
			return &value, nil
		}
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		decoded = raw
	}
	value.Value = decoded
	value.Quality = "good"
	return &value, nil
}

func parseStoredHTTPDefaultValue(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(*value), &decoded); err != nil {
		return *value
	}
	return decoded
}

func dataPointRuntimeKey(connection *repository.ConnectionRecord) string {
	runtimeKey := strings.TrimSpace(toString(connection.Config["runtimeKey"]))
	if runtimeKey == "" {
		runtimeKey = strings.TrimSpace(toString(connection.Config["storeKey"]))
	}
	if runtimeKey == "" {
		runtimeKey = connection.ID
	}
	return runtimeKey
}

func defaultValueOrNil(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func stringifyDataPointValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", typed)
	}
}
