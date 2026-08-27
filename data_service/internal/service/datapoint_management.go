package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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
	SourceID     *string `json:"sourceId,omitempty"`
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

// GetDataPointStatuses 按 id、path 或 sourceId 批量返回数据点状态。
func (s *DataPointService) GetDataPointStatuses(ctx context.Context, projectID string, ids, paths []string, sourceIDs ...[]string) ([]DataPointStatusInfo, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}

	uniqueIDs := uniqueStrings(ids)
	uniquePaths := uniqueStrings(paths)
	uniqueSourceIDs := []string{}
	if len(sourceIDs) > 0 {
		uniqueSourceIDs = uniqueStrings(sourceIDs[0])
	}
	if len(uniqueIDs) == 0 && len(uniquePaths) == 0 && len(uniqueSourceIDs) == 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "ids、paths 或 sourceIds 至少需要提供一个")
	}

	records := make([]repository.DataPointRecord, 0, len(uniqueIDs)+len(uniquePaths)+len(uniqueSourceIDs))
	if len(uniqueIDs) > 0 {
		loaded, err := s.repository.GetByProjectAndIDs(ctx, projectID, uniqueIDs)
		if err != nil {
			return nil, err
		}
		records = append(records, loaded...)
	}
	if len(uniqueSourceIDs) > 0 {
		loaded, _, err := s.repository.ListByProject(ctx, projectID, repository.DataPointListFilter{
			Page:      1,
			PageSize:  5000,
			SourceIDs: uniqueSourceIDs,
		})
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
			SourceID: cloneOptionalString(record.SourceID),
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
func (s *DataPointService) WriteDataPointValue(ctx context.Context, projectID, id, userID, role string, value any) (*WriteDataPointResult, error) {
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

	return s.writeDataPointRecord(ctx, *record, userID, role, value)
}

// WriteDataPointValueByPath 使用跨环境稳定的 path 写入，供场景数据桥调用。
func (s *DataPointService) WriteDataPointValueByPath(ctx context.Context, projectID, path, userID, role string, value any) (*WriteDataPointResult, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "path 不能为空")
	}
	if err := validateUserID(userID); err != nil {
		return nil, err
	}
	record, err := s.repository.GetByProjectAndPath(ctx, projectID, path)
	if err != nil {
		return nil, err
	}
	return s.writeDataPointRecord(ctx, *record, userID, role, value)
}

func (s *DataPointService) writeDataPointRecord(ctx context.Context, record repository.DataPointRecord, userID, role string, value any) (*WriteDataPointResult, error) {
	if record.Status != "active" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点不是 active 状态")
	}
	if !runtimeWriteAllowed(record.RuntimePermissions.Write, role) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodePermissionInsufficient, http.StatusForbidden, "当前角色禁止写入该数据点")
	}
	normalizedValue, err := normalizeRuntimeWriteValue(record, value)
	if err != nil {
		return nil, err
	}
	if record.SourceType == "mqtt.subscription" && record.SourceID != nil && s.mqttPublisher != nil {
		if _, err := s.mqttPublisher.PublishSubscriptionMessage(
			ctx,
			record.ProjectID,
			strings.TrimSpace(*record.SourceID),
			normalizedValue,
		); err != nil {
			return nil, err
		}
		return &WriteDataPointResult{
			ID:        record.ID,
			Path:      record.Path,
			Value:     normalizedValue,
			Timestamp: time.Now().UTC(),
		}, nil
	}
	defaultValue := stringifyDataPointValue(normalizedValue)
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
		Value:     normalizedValue,
		Timestamp: time.Now().UTC(),
	}, nil
}

func runtimeWriteAllowed(grant repository.RuntimePermissionGrant, role string) bool {
	role = strings.TrimSpace(role)
	for _, denied := range grant.DenyRoles {
		if denied == role {
			return false
		}
	}
	for _, allowed := range grant.AllowRoles {
		if allowed == role {
			return true
		}
	}
	return grant.Inherit
}

func normalizeRuntimeWriteValue(record repository.DataPointRecord, value any) (any, error) {
	dataType := strings.ToLower(strings.TrimSpace(record.DataType))
	badType := func() (any, error) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入值与数据点类型不匹配")
	}
	switch dataType {
	case "bool":
		if _, ok := value.(bool); !ok {
			return badType()
		}
	case "string", "bytes", "datetime":
		if _, ok := value.(string); !ok {
			return badType()
		}
	case "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64":
		number, ok := runtimeNumber(value)
		if !ok || math.Trunc(number) != number || (strings.HasPrefix(dataType, "uint") && number < 0) {
			return badType()
		}
		value = number
	case "float32", "float64", "decimal":
		number, ok := runtimeNumber(value)
		if !ok {
			return badType()
		}
		value = number
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return badType()
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return badType()
		}
	default:
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点类型不支持场景写入")
	}
	if number, ok := runtimeNumber(value); ok {
		if record.MinValue != nil && number < *record.MinValue {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入值小于数据点最小值")
		}
		if record.MaxValue != nil && number > *record.MaxValue {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入值大于数据点最大值")
		}
	}
	return value, nil
}

func runtimeNumber(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, !math.IsNaN(number) && !math.IsInf(number, 0)
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case json.Number:
		parsed, err := number.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

// GetDataPointUsages 返回报警策略与计算单元对数据点的真实引用。
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
	records, err := s.repository.ListUsages(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(records))
	for _, record := range records {
		result = append(result, map[string]any{
			"module":   record.Module,
			"objectId": record.ObjectID,
			"label":    record.Label,
		})
	}
	return result, nil
}

func (s *DataPointService) ListDataPointSourceOptions(ctx context.Context, projectID, search string, page, pageSize int) (map[string]any, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	items, total, err := s.repository.ListSourceOptions(ctx, projectID, search, page, pageSize)
	if err != nil {
		return nil, err
	}
	page, pageSize = normalizePageAndSize(page, pageSize, 1, 100)
	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, map[string]any{"scopeType": item.ScopeType, "id": item.ID, "name": item.Name, "sourceType": item.SourceType})
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	return map[string]any{"list": list, "pagination": map[string]int{"page": page, "pageSize": pageSize, "total": total, "totalPages": totalPages}}, nil
}

// ListDataPointTags 返回工程级标签摘要，GET 不修改任何数据。
func (s *DataPointService) ListDataPointTags(ctx context.Context, projectID string) ([]map[string]any, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	records, err := s.repository.ListTagSummaries(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(records))
	for _, record := range records {
		result = append(result, map[string]any{"tag": record.Tag, "count": record.Count})
	}
	return result, nil
}

// RemoveDataPointTag 删除工程内所有数据点上的指定标签。
func (s *DataPointService) RemoveDataPointTag(ctx context.Context, projectID, userID, tag string) (int, error) {
	if err := validateProjectID(projectID); err != nil {
		return 0, err
	}
	if err := validateUserID(userID); err != nil {
		return 0, err
	}
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "标签不能为空")
	}
	return s.repository.RemoveTagByProject(ctx, projectID, tag, userID)
}

func (s *DataPointService) buildValueFromRecord(ctx context.Context, projectID string, record repository.DataPointRecord) (*DataPointValue, error) {
	return s.buildValueFromRecordWithParameters(ctx, projectID, record, nil)
}

func (s *DataPointService) buildValueFromRecordWithParameters(ctx context.Context, projectID string, record repository.DataPointRecord, runtimeParameters map[string]any) (*DataPointValue, error) {
	now := time.Now().UTC()
	value := DataPointValue{
		Path:        record.Path,
		Value:       defaultValueOrNil(record.DefaultValue),
		Quality:     "unknown",
		Timestamp:   now,
		ValueOrigin: "unavailable",
		OriginLabel: "当前值不可用",
		Status:      record.Status,
	}
	if value.Value != nil {
		value.Quality = "good"
		value.ValueOrigin = "default_value"
		value.OriginLabel = "开发态默认值"
	}

	if record.SourceType == "db.query" && record.SourceID != nil && strings.TrimSpace(*record.SourceID) != "" {
		parameters, err := dataPointQueryParameters(record.SourceConfig)
		if err != nil {
			return nil, err
		}
		for name, runtimeValue := range runtimeParameters {
			parameters[name] = runtimeValue
		}
		result, err := s.queries.ExecuteQueryForProject(ctx, projectID, *record.SourceID, ExecuteQueryInput{
			Parameters: parameters,
		})
		if err != nil {
			return nil, err
		}

		var mapped any
		selector, _ := record.SourceConfig["selector"].(map[string]any)
		selectorKind := strings.TrimSpace(firstString(selector, "kind"))
		if selectorKind == "" || selectorKind == "whole" {
			mapped = queryDatasetValue(result)
		} else {
			mapped, err = selectDataPointOutputValue(result.Data, record.SourceConfig, true)
			if err != nil {
				return nil, err
			}
		}
		value.Value = mapped
		value.Quality = "good"
		value.ObservedAt = &now
		value.ValueOrigin = "realtime_query"
		value.OriginLabel = "实时查询"
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
				value.ObservedAt = &snapshot.Timestamp
				value.SourceTimestamp = &snapshot.Timestamp
				value.ValueOrigin = "realtime_read"
				value.OriginLabel = "MQTT 最近消息"
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
				value.ObservedAt = &snapshot.Timestamp
				value.SourceTimestamp = &snapshot.Timestamp
				value.ValueOrigin = "realtime_read"
				value.OriginLabel = "MQTT 最近消息"
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

// queryDatasetValue 为完整查询结果提供最小稳定对象契约。
// rowCount 只表示本次返回行数，不推断 LIMIT/OFFSET 之前的数据库总量。
func queryDatasetValue(result *QueryExecutionResult) map[string]any {
	if result == nil {
		return map[string]any{
			"fields":   []string{},
			"rows":     []map[string]any{},
			"rowCount": 0,
		}
	}
	return map[string]any{
		"fields":   append([]string{}, result.Columns...),
		"rows":     result.Data,
		"rowCount": result.RowCount,
	}
}

// selectDataPointOutputValue 将工作台原始结果按稳定映射选择为数据点值。
// SQL 的 column/path 标量映射必须恰好一行，避免同一数据点在不同执行中返回数组或不确定行。
func selectDataPointOutputValue(root any, sourceConfig map[string]any, strictSingleRow bool) (any, error) {
	selector, _ := sourceConfig["selector"].(map[string]any)
	kind := strings.TrimSpace(firstString(selector, "kind"))
	if kind == "" || kind == "whole" {
		return root, nil
	}
	if strictSingleRow {
		rows, ok := root.([]map[string]any)
		if !ok {
			if genericRows, genericOK := root.([]any); genericOK {
				rows = make([]map[string]any, 0, len(genericRows))
				for _, row := range genericRows {
					object, objectOK := row.(map[string]any)
					if !objectOK {
						return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "字段映射要求查询返回对象行")
					}
					rows = append(rows, object)
				}
			} else {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "字段映射要求查询返回对象行")
			}
		}
		if len(rows) != 1 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "标量或字段映射要求查询恰好返回一行")
		}
		root = rows[0]
	}
	switch kind {
	case "column":
		column := strings.TrimSpace(firstString(selector, "column"))
		object, ok := root.(map[string]any)
		if !ok || column == "" {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "查询字段映射配置无效")
		}
		value, exists := object[column]
		if !exists {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "查询结果缺少映射字段: "+column)
		}
		return value, nil
	case "path":
		segments, err := sourceConfigSelectorSegments(selector)
		if err != nil {
			return nil, err
		}
		value, ok := extractValueBySegments(root, segments)
		if !ok {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "结果中不存在已配置的字段路径")
		}
		return value, nil
	default:
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点输出选择器无效")
	}
}

func sourceConfigSelectorSegments(selector map[string]any) ([]any, error) {
	raw, ok := selector["segments"]
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "字段路径配置缺失")
	}
	segments, ok := raw.([]any)
	if !ok {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "字段路径配置无效")
	}
	return normalizeValuePathSegments(segments, false)
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
			value.ObservedAt = request.LastSentAt
		}
		if request.Quality == "bad" {
			// 旧版本会写入 LastResponse 作为数据点 value；新版不再持久化响应，
			// quality 已经是"bad"时让 value 保持空，避免数据点误用错误响应作为值。
			value.Value = nil
			return &value, nil
		}
	}
	value.Value = parseStoredHTTPDefaultValue(record.DefaultValue)
	if value.Value != nil {
		value.Quality = "good"
		value.ValueOrigin = "recent_preview"
		value.OriginLabel = "HTTP 最近预览"
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
			value.ObservedAt = session.LastMessageAt
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
		value.ValueOrigin = "recent_preview"
		value.OriginLabel = "WebSocket 最近预览"
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
		value.ObservedAt = field.LastUpdatedAt
		value.SourceTimestamp = field.LastUpdatedAt
	}
	value.ValueOrigin = "recent_preview"
	value.OriginLabel = "Kafka 最近预览"
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
		resolved, err := s.buildRedisKeyValue(ctx, connection, key, record.SourceConfig, value)
		return selectRealtimeDataPointOutput(resolved, record.SourceConfig, err)
	}
	if connection.Type == "builtin.realtime" {
		resolved, err := s.buildBuiltinRealtimeKeyValue(ctx, projectID, connection, key, value)
		return selectRealtimeDataPointOutput(resolved, record.SourceConfig, err)
	}
	return &value, nil
}

func selectRealtimeDataPointOutput(value *DataPointValue, sourceConfig map[string]any, readErr error) (*DataPointValue, error) {
	if readErr != nil || value == nil || value.Value == nil {
		return value, readErr
	}
	selected, err := selectDataPointOutputValue(value.Value, sourceConfig, false)
	if err != nil {
		return nil, err
	}
	value.Value = selected
	return value, nil
}

func (s *DataPointService) buildRedisKeyValue(ctx context.Context, connection *repository.ConnectionRecord, key string, sourceConfig map[string]any, value DataPointValue) (*DataPointValue, error) {
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
	if normalizeRedisType(keyType) == "string" && strings.EqualFold(firstString(sourceConfig, "valueType"), "json") {
		if text, ok := current.(string); ok {
			var decoded any
			if json.Unmarshal([]byte(text), &decoded) == nil {
				current = decoded
			}
		}
	}
	value.Value = current
	value.Quality = "good"
	now := time.Now().UTC()
	value.Timestamp = now
	value.ObservedAt = &now
	value.ValueOrigin = "realtime_read"
	value.OriginLabel = "Redis 实时读取"
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
	now := time.Now().UTC()
	value.Timestamp = now
	value.ObservedAt = &now
	value.ValueOrigin = "realtime_read"
	value.OriginLabel = "IF 实时库读取"
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
		encoded, err := json.Marshal(typed)
		if err == nil {
			return string(encoded)
		}
		return fmt.Sprintf("%v", typed)
	}
}
