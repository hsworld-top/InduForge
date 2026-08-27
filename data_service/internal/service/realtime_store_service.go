package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/redis/go-redis/v9"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

const (
	defaultRealtimeStoreLimit   = 500
	maxRealtimeStoreLimit       = 10000
	maxWritableRealtimeKeyBytes = 256
)

type RealtimeStoreKeySummary struct {
	ID            string         `json:"id,omitempty"`
	Key           string         `json:"key"`
	Type          string         `json:"type"`
	TTL           int64          `json:"ttl"`
	Size          int64          `json:"size"`
	Provider      string         `json:"provider"`
	ValueType     string         `json:"valueType"`
	DataPointID   string         `json:"dataPointId,omitempty"`
	DataPointPath string         `json:"dataPointPath,omitempty"`
	Outputs       []SourceOutput `json:"outputs"`
}

type RealtimeStoreGroup struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type RealtimeStoreKeyList struct {
	List    []RealtimeStoreKeySummary `json:"list"`
	Groups  []RealtimeStoreGroup      `json:"groups"`
	HasMore bool                      `json:"hasMore"`
	Limit   int                       `json:"limit"`
}

type RealtimeStoreValue struct {
	ID            string         `json:"id,omitempty"`
	Key           string         `json:"key"`
	Type          string         `json:"type"`
	TTL           int64          `json:"ttl"`
	Value         any            `json:"value"`
	Size          int64          `json:"size"`
	Provider      string         `json:"provider"`
	ValueType     string         `json:"valueType"`
	DataPointID   string         `json:"dataPointId,omitempty"`
	DataPointPath string         `json:"dataPointPath,omitempty"`
	Outputs       []SourceOutput `json:"outputs"`
}

type SaveRealtimeStoreKeyInput struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	TTLSeconds  int    `json:"ttlSeconds"`
	Value       any    `json:"value"`
	ValueType   string `json:"valueType"`
	Description string `json:"description"`
}

type RenameRealtimeStoreKeyInput struct {
	NewKey string `json:"newKey"`
}

type CreateRealtimeKeyDataPointInput struct {
	DataType string              `json:"dataType"`
	Outputs  []SourceOutputInput `json:"outputs"`
}

type BatchCreateRealtimeKeyDataPointInput struct {
	Keys     []string            `json:"keys"`
	DataType string              `json:"dataType"`
	Outputs  []SourceOutputInput `json:"outputs"`
}

type BatchRealtimeKeyDataPointResult struct {
	Key           string         `json:"key"`
	Status        string         `json:"status"`
	Message       string         `json:"message,omitempty"`
	DataPointID   string         `json:"dataPointId,omitempty"`
	DataPointPath string         `json:"dataPointPath,omitempty"`
	Outputs       []SourceOutput `json:"outputs,omitempty"`
}

type BatchRealtimeKeyDataPointResponse struct {
	List    []BatchRealtimeKeyDataPointResult `json:"list"`
	Created int                               `json:"created"`
	Exists  int                               `json:"exists"`
	Failed  int                               `json:"failed"`
}

type RealtimeStoreService struct {
	repository     *repository.RealtimeStoreRepository
	connections    *repository.ConnectionRepository
	builtinRuntime *BuiltinRuntimeService
}

func NewRealtimeStoreService(repo *repository.RealtimeStoreRepository, connections *repository.ConnectionRepository, builtinRuntime *BuiltinRuntimeService) *RealtimeStoreService {
	return &RealtimeStoreService{repository: repo, connections: connections, builtinRuntime: builtinRuntime}
}

func (s *RealtimeStoreService) ListKeys(ctx context.Context, projectID, connectionID, q, group string, limit int) (*RealtimeStoreKeyList, error) {
	connection, provider, err := s.loadConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultRealtimeStoreLimit
	} else if limit > maxRealtimeStoreLimit {
		limit = maxRealtimeStoreLimit
	}
	metadataByKey, err := s.metadataByKey(ctx, projectID, connectionID, provider)
	if err != nil {
		return nil, err
	}
	var items []RealtimeStoreKeySummary
	if provider == "redis" {
		metadataKeys := make([]string, 0, len(metadataByKey))
		for key := range metadataByKey {
			metadataKeys = append(metadataKeys, key)
		}
		items, err = s.listRedisKeys(ctx, connection, q, limit+1, metadataKeys)
	} else {
		items, err = s.listBuiltinKeys(ctx, connection, q)
	}
	if err != nil {
		return nil, err
	}
	for index := range items {
		if record, ok := metadataByKey[items[index].Key]; ok {
			applyRealtimeMetadata(&items[index], record)
		}
	}
	searchedItems := filterRealtimeKeys(items, q, "")
	groups := buildRealtimeGroups(searchedItems)
	filteredItems := filterRealtimeKeys(searchedItems, "", group)
	hasMore := len(filteredItems) > limit
	if hasMore {
		filteredItems = filteredItems[:limit]
	}
	return &RealtimeStoreKeyList{List: filteredItems, Groups: groups, HasMore: hasMore, Limit: limit}, nil
}

func (s *RealtimeStoreService) GetKey(ctx context.Context, projectID, connectionID, key string) (*RealtimeStoreValue, error) {
	connection, provider, err := s.loadConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	key, err = normalizeRealtimeKey(key)
	if err != nil {
		return nil, err
	}
	var value *RealtimeStoreValue
	if provider == "redis" {
		value, err = s.getRedisValue(ctx, connection, key)
	} else {
		value, err = s.getBuiltinValue(ctx, connection, key)
	}
	if err != nil {
		return nil, err
	}
	if record, metaErr := s.repository.GetByKey(ctx, projectID, connectionID, provider, key); metaErr == nil {
		applyRealtimeValueMetadata(value, record)
	}
	return value, nil
}

func (s *RealtimeStoreService) SaveKey(ctx context.Context, projectID, connectionID, userID string, input SaveRealtimeStoreKeyInput) (*RealtimeStoreValue, error) {
	connection, provider, err := s.loadConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	key, redisType, valueType, ttlSeconds, err := normalizeSaveRealtimeKeyInput(input)
	if err != nil {
		return nil, err
	}
	if provider == "redis" {
		if err := s.saveRedisKey(ctx, connection, key, redisType, input.Value, ttlSeconds); err != nil {
			return nil, err
		}
	} else if err := s.saveBuiltinKey(ctx, connection, key, redisType, input.Value, ttlSeconds); err != nil {
		return nil, err
	}
	if _, err := s.repository.Upsert(ctx, repository.UpsertRealtimeKeyParams{
		ProjectID:         projectID,
		ConnectionID:      connectionID,
		Provider:          provider,
		KeyPath:           key,
		RedisType:         redisType,
		ValueType:         valueType,
		DefaultTtlSeconds: ttlSeconds,
		Description:       strings.TrimSpace(input.Description),
	}); err != nil {
		return nil, err
	}
	// Redis 与内置实时库都遵循“保存即生成数据点”；已有输出配置时保留用户配置。
	record, recordErr := s.repository.GetByKey(ctx, projectID, connectionID, provider, key)
	if recordErr != nil {
		return nil, recordErr
	}
	if len(record.Outputs) == 0 {
		if _, createErr := s.CreateDataPoint(ctx, projectID, connectionID, key, userID, CreateRealtimeKeyDataPointInput{}); createErr != nil {
			return nil, createErr
		}
	}
	return s.GetKey(ctx, projectID, connectionID, key)
}

func (s *RealtimeStoreService) RenameKey(ctx context.Context, projectID, connectionID, key string, input RenameRealtimeStoreKeyInput) (*RealtimeStoreValue, error) {
	connection, provider, err := s.loadConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	oldKey, err := normalizeRealtimeKey(key)
	if err != nil {
		return nil, err
	}
	newKey, err := normalizeWritableRealtimeKey(input.NewKey)
	if err != nil {
		return nil, err
	}
	if oldKey == newKey {
		return s.GetKey(ctx, projectID, connectionID, oldKey)
	}
	current, err := s.GetKey(ctx, projectID, connectionID, oldKey)
	if err != nil {
		return nil, err
	}
	if provider == "redis" {
		if err := s.withRedisClient(ctx, connection, func(client redis.UniversalClient) error {
			renamed, err := client.RenameNX(ctx, oldKey, newKey).Result()
			if err != nil {
				return err
			}
			if !renamed {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "目标 key 已存在")
			}
			return nil
		}); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "重命名 Redis key 失败", err)
		}
	} else {
		oldFullKey, newFullKey, err := s.builtinFullKeys(projectID, connection, oldKey, newKey)
		if err != nil {
			return nil, err
		}
		renamed, err := s.builtinRuntime.realtimeClient.RenameNX(ctx, oldFullKey, newFullKey).Result()
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "重命名 IF 实时库 key 失败", err)
		}
		if !renamed {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "目标 key 已存在")
		}
	}
	if _, err := s.repository.Rename(
		ctx, projectID, connectionID, provider, oldKey, newKey,
		buildRealtimeDataPointPath(provider, connection.Name, newKey),
		realtimeKeyDisplayName(oldKey), realtimeKeyDisplayName(newKey),
	); err != nil {
		_, upsertErr := s.repository.Upsert(ctx, repository.UpsertRealtimeKeyParams{
			ProjectID:         projectID,
			ConnectionID:      connectionID,
			Provider:          provider,
			KeyPath:           newKey,
			RedisType:         current.Type,
			ValueType:         current.ValueType,
			DefaultTtlSeconds: int(current.TTL),
			Description:       "",
		})
		if upsertErr != nil {
			return nil, err
		}
	}
	return s.GetKey(ctx, projectID, connectionID, newKey)
}

func (s *RealtimeStoreService) DeleteKey(ctx context.Context, projectID, connectionID, key, userID string) (map[string]bool, error) {
	connection, provider, err := s.loadConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	key, err = normalizeRealtimeKey(key)
	if err != nil {
		return nil, err
	}
	if provider == "redis" {
		err = s.withRedisClient(ctx, connection, func(client redis.UniversalClient) error {
			return client.Del(ctx, key).Err()
		})
	} else {
		fullKey, keyErr := s.builtinFullKey(projectID, connection, key)
		if keyErr != nil {
			return nil, keyErr
		}
		err = s.builtinRuntime.realtimeClient.Del(ctx, fullKey).Err()
	}
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "删除实时库 key 失败", err)
	}
	var userPtr *string
	if strings.TrimSpace(userID) != "" {
		userPtr = &userID
	}
	if err := s.repository.Delete(ctx, projectID, connectionID, provider, key, userPtr); err != nil {
		return nil, err
	}
	return map[string]bool{"deleted": true}, nil
}

func (s *RealtimeStoreService) CreateDataPoint(ctx context.Context, projectID, connectionID, key, userID string, input CreateRealtimeKeyDataPointInput) (*BatchRealtimeKeyDataPointResult, error) {
	connection, provider, err := s.loadConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	key, err = normalizeRealtimeKey(key)
	if err != nil {
		return nil, err
	}
	value, err := s.GetKey(ctx, projectID, connectionID, key)
	if err != nil {
		return nil, err
	}
	config := map[string]any{
		"provider":     provider,
		"connectionId": connectionID,
		"key":          key,
		"redisType":    value.Type,
		"valueType":    value.ValueType,
	}
	if provider == "redis" {
		config["db"] = intFromAny(connection.Config["database"], intFromAny(connection.Config["db"], 0))
	} else {
		config["runtimeKey"] = s.runtimeKey(connection)
	}
	var userPtr *string
	if strings.TrimSpace(userID) != "" {
		userPtr = &userID
	}
	defaultType := inferRealtimeDataPointType(input.DataType, value.Type, value.ValueType, value.Value)
	displayName := realtimeKeyDisplayName(key)
	pathPrefix := buildRealtimeDataPointPath(provider, connection.Name, key)
	outputs, err := normalizeSourceOutputs(input.Outputs, SourceOutputInput{Key: "value", DisplayName: displayName, Selector: SourceOutputSelector{Kind: "whole"}, DataType: defaultType})
	if err != nil {
		return nil, err
	}
	if err := attachRealtimeOutputValues(value.Value, outputs); err != nil {
		return nil, err
	}
	created, err := s.repository.BatchCreateDataPoints(ctx, []repository.BatchRealtimeKeyDataPointParams{{
		Metadata: repository.UpsertRealtimeKeyParams{ProjectID: projectID, ConnectionID: connectionID,
			Provider: provider, KeyPath: key, RedisType: value.Type, ValueType: defaultType,
			DefaultTtlSeconds: int(value.TTL), Description: ""},
		DataPoint: repository.CreateRealtimeKeyDataPointParams{ProjectID: projectID, ConnectionID: connectionID,
			KeyPath: key, PathPrefix: pathPrefix, DisplayName: displayName, Provider: provider, RedisType: value.Type, DataType: defaultType,
			SourceConfig: config, UserID: userPtr, Outputs: outputs},
	}})
	if err != nil {
		return nil, err
	}
	records := created[0]
	result := &BatchRealtimeKeyDataPointResult{Key: key, Status: "created", Outputs: sourceOutputsFromRecords(records)}
	if len(records) > 0 {
		result.DataPointID = records[0].DataPointID
		result.DataPointPath = records[0].DataPointPath
	}
	return result, nil
}

func (s *RealtimeStoreService) BatchCreateDataPoints(ctx context.Context, projectID, connectionID, userID string, input BatchCreateRealtimeKeyDataPointInput) (*BatchRealtimeKeyDataPointResponse, error) {
	keys := normalizeRealtimeBatchKeys(input.Keys)
	if len(keys) == 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "请选择需要创建数据点的 key")
	}
	if len(keys) > 500 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "单次最多创建 500 个实时库 key 数据点")
	}
	connection, provider, err := s.loadConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	metadataByKey, err := s.metadataByKey(ctx, projectID, connectionID, provider)
	if err != nil {
		return nil, err
	}

	result := &BatchRealtimeKeyDataPointResponse{List: make([]BatchRealtimeKeyDataPointResult, len(keys))}
	pending := make([]repository.BatchRealtimeKeyDataPointParams, 0, len(keys))
	pendingIndexes := make([]int, 0, len(keys))
	var userPtr *string
	if strings.TrimSpace(userID) != "" {
		userPtr = &userID
	}
	for _, key := range keys {
		index := len(pendingIndexes) + result.Exists
		if record, ok := metadataByKey[key]; ok && record.DataPointID != nil {
			result.Exists += 1
			result.List[index] = BatchRealtimeKeyDataPointResult{
				Key:           key,
				Status:        "exists",
				DataPointID:   derefString(record.DataPointID),
				DataPointPath: derefString(record.DataPointPath),
			}
			continue
		}
		value, err := s.GetKey(ctx, projectID, connectionID, key)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "批量建点预校验失败: "+key, err)
		}
		dataType := inferRealtimeDataPointType(input.DataType, value.Type, value.ValueType, value.Value)
		displayName := realtimeKeyDisplayName(key)
		pathPrefix := buildRealtimeDataPointPath(provider, connection.Name, key)
		outputs, normalizeErr := normalizeSourceOutputs(input.Outputs, SourceOutputInput{Key: "value", DisplayName: displayName, Selector: SourceOutputSelector{Kind: "whole"}, DataType: dataType})
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		if valueErr := attachRealtimeOutputValues(value.Value, outputs); valueErr != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "批量建点字段校验失败: "+key, valueErr)
		}
		config := map[string]any{
			"provider": provider, "connectionId": connectionID, "key": key,
			"redisType": value.Type, "valueType": value.ValueType,
		}
		if provider == "redis" {
			config["db"] = intFromAny(connection.Config["database"], intFromAny(connection.Config["db"], 0))
		} else {
			config["runtimeKey"] = s.runtimeKey(connection)
		}
		pendingIndexes = append(pendingIndexes, index)
		pending = append(pending, repository.BatchRealtimeKeyDataPointParams{
			Metadata: repository.UpsertRealtimeKeyParams{
				ProjectID: projectID, ConnectionID: connectionID, Provider: provider, KeyPath: key,
				RedisType: value.Type, ValueType: dataType, DefaultTtlSeconds: int(value.TTL), Description: "",
			},
			DataPoint: repository.CreateRealtimeKeyDataPointParams{
				ProjectID: projectID, ConnectionID: connectionID, KeyPath: key, PathPrefix: pathPrefix, DisplayName: displayName, Provider: provider,
				RedisType: value.Type, DataType: dataType, SourceConfig: config, UserID: userPtr, Outputs: outputs,
			},
		})
	}
	created, err := s.repository.BatchCreateDataPoints(ctx, pending)
	if err != nil {
		return nil, err
	}
	for index, records := range created {
		listIndex := pendingIndexes[index]
		item := BatchRealtimeKeyDataPointResult{Key: keys[listIndex], Status: "created", Outputs: sourceOutputsFromRecords(records)}
		if len(records) > 0 {
			item.DataPointID = records[0].DataPointID
			item.DataPointPath = records[0].DataPointPath
		}
		result.List[listIndex] = item
		result.Created++
	}
	return result, nil
}

func (s *RealtimeStoreService) loadConnection(ctx context.Context, projectID, connectionID string) (*repository.ConnectionRecord, string, error) {
	if err := validateProjectAndConnection(projectID, connectionID); err != nil {
		return nil, "", err
	}
	connection, err := s.connections.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return nil, "", err
	}
	switch connection.Type {
	case "redis":
		return connection, "redis", nil
	case "builtin.realtime":
		if s.builtinRuntime == nil || s.builtinRuntime.realtimeClient == nil {
			return nil, "", apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "IF 实时库未初始化")
		}
		return connection, "builtin", nil
	default:
		return nil, "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "接入源类型不匹配")
	}
}

func attachRealtimeOutputValues(root any, outputs []repository.SourceOutputMappingParam) error {
	for index := range outputs {
		value := root
		switch outputs[index].Selector.Kind {
		case "whole":
		case "path":
			selected, ok := extractValueBySegments(root, outputs[index].Selector.Segments)
			if !ok {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时 Key 中不存在输出字段: "+outputs[index].DisplayName)
			}
			value = selected
		case "column":
			if outputs[index].Selector.Column == nil {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时 Key 字段选择器无效")
			}
			object, ok := root.(map[string]any)
			if !ok {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时 Key 不是对象，不能选择字段")
			}
			selected, exists := object[*outputs[index].Selector.Column]
			if !exists {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时 Key 中不存在输出字段: "+outputs[index].DisplayName)
			}
			value = selected
		}
		encoded := stringifyJSONValue(value)
		outputs[index].DefaultValue = &encoded
	}
	return nil
}

func (s *RealtimeStoreService) metadataByKey(ctx context.Context, projectID, connectionID, provider string) (map[string]repository.RealtimeKeyRecord, error) {
	records, err := s.repository.List(ctx, projectID, connectionID, provider)
	if err != nil {
		return nil, err
	}
	result := make(map[string]repository.RealtimeKeyRecord, len(records))
	for _, record := range records {
		result[record.KeyPath] = record
	}
	return result, nil
}

func (s *RealtimeStoreService) listRedisKeys(ctx context.Context, connection *repository.ConnectionRecord, q string, limit int, metadataKeys []string) ([]RealtimeStoreKeySummary, error) {
	var result []RealtimeStoreKeySummary
	err := s.withRedisClient(ctx, connection, func(client redis.UniversalClient) error {
		pattern := strings.TrimSpace(q)
		if pattern == "" {
			pattern = strings.TrimSpace(toString(connection.Config["keyPattern"]))
		}
		if pattern == "" || !strings.ContainsAny(pattern, "*?[]") {
			if pattern == "" {
				pattern = "*"
			} else {
				pattern = "*" + pattern + "*"
			}
		}
		keys, err := scanRedisPreviewKeys(ctx, client, pattern, limit)
		if err != nil {
			return err
		}
		result = make([]RealtimeStoreKeySummary, 0, len(keys)+len(metadataKeys))
		seen := make(map[string]struct{}, len(keys)+len(metadataKeys))
		for _, key := range keys {
			seen[key] = struct{}{}
			result = append(result, redisKeySummary(ctx, client, key))
		}
		query := strings.ToLower(strings.TrimSpace(q))
		for _, key := range metadataKeys {
			if _, ok := seen[key]; ok || (query != "" && !strings.Contains(strings.ToLower(key), query)) {
				continue
			}
			exists, existsErr := client.Exists(ctx, key).Result()
			if existsErr != nil {
				return existsErr
			}
			if exists == 0 {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, redisKeySummary(ctx, client, key))
		}
		return nil
	})
	return result, err
}

func redisKeySummary(ctx context.Context, client redis.UniversalClient, key string) RealtimeStoreKeySummary {
	keyType, _ := client.Type(ctx, key).Result()
	ttl, _ := client.TTL(ctx, key).Result()
	return RealtimeStoreKeySummary{
		Key: key, Type: normalizeRedisType(keyType), TTL: int64(ttl.Seconds()),
		Size: redisWorkbenchSize(ctx, client, key, keyType), Provider: "redis", ValueType: "object",
	}
}

func (s *RealtimeStoreService) listBuiltinKeys(ctx context.Context, connection *repository.ConnectionRecord, q string) ([]RealtimeStoreKeySummary, error) {
	records, err := s.repository.List(ctx, connection.ProjectID, connection.ID, "builtin")
	if err != nil {
		return nil, err
	}
	items := make([]RealtimeStoreKeySummary, 0, len(records))
	for _, record := range records {
		if q != "" && !strings.Contains(strings.ToLower(record.KeyPath), strings.ToLower(q)) {
			continue
		}
		fullKey, err := s.builtinFullKey(connection.ProjectID, connection, record.KeyPath)
		if err != nil {
			return nil, err
		}
		ttl, _ := s.builtinRuntime.realtimeClient.TTL(ctx, fullKey).Result()
		exists, _ := s.builtinRuntime.realtimeClient.Exists(ctx, fullKey).Result()
		size := int64(0)
		if exists > 0 {
			size, _ = s.builtinRuntime.realtimeClient.StrLen(ctx, fullKey).Result()
		}
		items = append(items, RealtimeStoreKeySummary{
			ID:            record.ID,
			Key:           record.KeyPath,
			Type:          "string",
			TTL:           int64(ttl.Seconds()),
			Size:          size,
			Provider:      "builtin",
			ValueType:     record.ValueType,
			DataPointID:   derefString(record.DataPointID),
			DataPointPath: derefString(record.DataPointPath),
		})
	}
	return items, nil
}

func (s *RealtimeStoreService) getRedisValue(ctx context.Context, connection *repository.ConnectionRecord, key string) (*RealtimeStoreValue, error) {
	var result *RealtimeStoreValue
	err := s.withRedisClient(ctx, connection, func(client redis.UniversalClient) error {
		keyType, err := client.Type(ctx, key).Result()
		if err != nil {
			return err
		}
		value, err := readRealtimeRedisValue(ctx, client, key, keyType)
		if err != nil {
			return err
		}
		ttl, _ := client.TTL(ctx, key).Result()
		result = &RealtimeStoreValue{
			Key:       key,
			Type:      normalizeRedisType(keyType),
			TTL:       int64(ttl.Seconds()),
			Value:     value,
			Size:      redisWorkbenchSize(ctx, client, key, keyType),
			Provider:  "redis",
			ValueType: "object",
		}
		return nil
	})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "读取 Redis key 失败", err)
	}
	return result, nil
}

func (s *RealtimeStoreService) getBuiltinValue(ctx context.Context, connection *repository.ConnectionRecord, key string) (*RealtimeStoreValue, error) {
	fullKey, err := s.builtinFullKey(connection.ProjectID, connection, key)
	if err != nil {
		return nil, err
	}
	raw, err := s.builtinRuntime.realtimeClient.Get(ctx, fullKey).Result()
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "读取 IF 实时库 key 失败", err)
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		value = raw
	}
	ttl, _ := s.builtinRuntime.realtimeClient.TTL(ctx, fullKey).Result()
	return &RealtimeStoreValue{
		Key:       key,
		Type:      "string",
		TTL:       int64(ttl.Seconds()),
		Value:     value,
		Size:      int64(len(raw)),
		Provider:  "builtin",
		ValueType: "object",
	}, nil
}

func (s *RealtimeStoreService) saveRedisKey(ctx context.Context, connection *repository.ConnectionRecord, key, redisType string, value any, ttlSeconds int) error {
	if err := validateRealtimeRedisValue(redisType, value); err != nil {
		return err
	}
	return s.withRedisClient(ctx, connection, func(client redis.UniversalClient) error {
		_, err := client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			// 只操作同一个 key，Redis Cluster 下不会触发跨 slot；MULTI/EXEC 避免删除后写入中途失败留下半截集合。
			pipe.Del(ctx, key)
			if err := writeRealtimeRedisValue(ctx, pipe, key, redisType, value); err != nil {
				return err
			}
			return applyRealtimeTTL(ctx, pipe, key, ttlSeconds)
		})
		return err
	})
}

func (s *RealtimeStoreService) saveBuiltinKey(ctx context.Context, connection *repository.ConnectionRecord, key, redisType string, value any, ttlSeconds int) error {
	fullKey, err := s.builtinFullKey(connection.ProjectID, connection, key)
	if err != nil {
		return err
	}
	normalizedValue, err := normalizeBuiltinRealtimeValue(redisType, value)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(normalizedValue)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "IF 实时库 value 必须可序列化为 JSON", err)
	}
	if err := s.builtinRuntime.realtimeClient.Set(ctx, fullKey, string(payload), ttlDuration(ttlSeconds)).Err(); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入 IF 实时库 key 失败", err)
	}
	return nil
}

func (s *RealtimeStoreService) withRedisClient(ctx context.Context, connection *repository.ConnectionRecord, fn func(redis.UniversalClient) error) error {
	client, err := newRedisPreviewClient(connection.Config, mapFromAny(connection.Config["options"]))
	if err != nil {
		return err
	}
	defer client.Close()
	return fn(client)
}

func (s *RealtimeStoreService) builtinFullKey(projectID string, connection *repository.ConnectionRecord, key string) (string, error) {
	return deriveBuiltinRealtimeKey(s.builtinRuntime.realtimeKeyPrefix, projectID, s.runtimeKey(connection), key)
}

func (s *RealtimeStoreService) builtinFullKeys(projectID string, connection *repository.ConnectionRecord, keys ...string) (string, string, error) {
	if len(keys) != 2 {
		return "", "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "重命名 key 参数不完整")
	}
	oldKey, err := s.builtinFullKey(projectID, connection, keys[0])
	if err != nil {
		return "", "", err
	}
	newKey, err := s.builtinFullKey(projectID, connection, keys[1])
	if err != nil {
		return "", "", err
	}
	return oldKey, newKey, nil
}

func (s *RealtimeStoreService) runtimeKey(connection *repository.ConnectionRecord) string {
	runtimeKey := strings.TrimSpace(toString(connection.Config["runtimeKey"]))
	if runtimeKey == "" {
		runtimeKey = strings.TrimSpace(toString(connection.Config["storeKey"]))
	}
	if runtimeKey == "" {
		runtimeKey = connection.ID
	}
	return runtimeKey
}

func readRealtimeRedisValue(ctx context.Context, client redis.UniversalClient, key, keyType string) (any, error) {
	switch normalizeRedisType(keyType) {
	case "string":
		return client.Get(ctx, key).Result()
	case "hash":
		return client.HGetAll(ctx, key).Result()
	case "list":
		return client.LRange(ctx, key, 0, -1).Result()
	case "set":
		return client.SMembers(ctx, key).Result()
	case "zset":
		values, err := client.ZRangeWithScores(ctx, key, 0, -1).Result()
		if err != nil {
			return nil, err
		}
		result := make([]map[string]any, 0, len(values))
		for _, item := range values {
			result = append(result, map[string]any{"member": fmt.Sprintf("%v", item.Member), "score": item.Score})
		}
		return result, nil
	case "stream":
		values, err := client.XRangeN(ctx, key, "-", "+", 100).Result()
		if err != nil {
			return nil, err
		}
		result := make([]map[string]any, 0, len(values))
		for _, item := range values {
			result = append(result, map[string]any{"id": item.ID, "values": item.Values})
		}
		return result, nil
	default:
		return nil, nil
	}
}

func writeRealtimeRedisValue(ctx context.Context, client redis.Cmdable, key, keyType string, value any) error {
	keyType = normalizeRedisType(keyType)
	if keyType == "stream" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "stream 第一版仅支持只读")
	}
	switch keyType {
	case "string":
		return client.Set(ctx, key, stringifyRealtimeValue(value), 0).Err()
	case "hash":
		return client.HSet(ctx, key, stringMapFromAny(value)).Err()
	case "list":
		values := stringSliceFromAny(value)
		return client.RPush(ctx, key, values).Err()
	case "set":
		values := stringSliceFromAny(value)
		return client.SAdd(ctx, key, values).Err()
	case "zset":
		return client.ZAdd(ctx, key, zsetMembersFromAny(value)...).Err()
	default:
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时库 key 类型不受支持")
	}
}

func validateRealtimeRedisValue(keyType string, value any) error {
	switch normalizeRedisType(keyType) {
	case "string":
		return nil
	case "hash":
		if len(stringMapFromAny(value)) == 0 {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "hash 至少需要一行有效字段")
		}
	case "list", "set":
		if len(stringSliceFromAny(value)) == 0 {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "集合至少需要一行有效值")
		}
	case "zset":
		if len(zsetMembersFromAny(value)) == 0 {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "zset 至少需要一行有效 member")
		}
	}
	return nil
}

func applyRealtimeTTL(ctx context.Context, client redis.Cmdable, key string, ttlSeconds int) error {
	if ttlSeconds < 0 {
		return client.Persist(ctx, key).Err()
	}
	if ttlSeconds == 0 {
		return client.Persist(ctx, key).Err()
	}
	return client.Expire(ctx, key, ttlDuration(ttlSeconds)).Err()
}

func ttlDuration(ttlSeconds int) time.Duration {
	if ttlSeconds <= 0 {
		return 0
	}
	return time.Duration(ttlSeconds) * time.Second
}

func normalizeSaveRealtimeKeyInput(input SaveRealtimeStoreKeyInput) (string, string, string, int, error) {
	key, err := normalizeWritableRealtimeKey(input.Key)
	if err != nil {
		return "", "", "", 0, err
	}
	redisType := normalizeRedisType(input.Type)
	if redisType == "" {
		redisType = "string"
	}
	switch redisType {
	case "string", "hash", "list", "set", "zset":
	default:
		return "", "", "", 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时库 key 类型不受支持")
	}
	if input.TTLSeconds < -1 {
		return "", "", "", 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TTL 不能小于 -1")
	}
	return key, redisType, normalizeRealtimeStoredValueType(redisType, input.ValueType, input.Value), input.TTLSeconds, nil
}

func normalizeRealtimeStoredValueType(redisType, valueType string, value any) string {
	if normalizeRedisType(redisType) != "string" {
		return "object"
	}
	mode := strings.ToLower(strings.TrimSpace(valueType))
	if mode == "json" || mode == "text" {
		return mode
	}
	if text, ok := value.(string); ok {
		var decoded any
		if json.Unmarshal([]byte(strings.TrimSpace(text)), &decoded) == nil {
			switch decoded.(type) {
			case map[string]any, []any:
				return "json"
			}
		}
	}
	return "text"
}

func normalizeRealtimeKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "key 不能为空")
	}
	if strings.Contains(key, "\x00") {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "key 不合法")
	}
	return key, nil
}

// normalizeWritableRealtimeKey 约束工作台新建和重命名的 Key。
// 外部 Redis 中既有的特殊 Key 仍可读取；这里只限制平台主动写入的名称，避免树结构和数据点路径产生歧义。
func normalizeWritableRealtimeKey(key string) (string, error) {
	trimmed := strings.TrimSpace(key)
	if _, err := normalizeRealtimeKey(trimmed); err != nil {
		return "", err
	}
	if key != trimmed {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Key 不能包含首尾空格")
	}
	if len([]byte(trimmed)) > maxWritableRealtimeKeyBytes {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Key 长度不能超过 256 字节")
	}
	if strings.HasPrefix(trimmed, ":") || strings.HasSuffix(trimmed, ":") || strings.Contains(trimmed, "::") {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Key 的层级分隔符 : 之间不能为空")
	}
	for _, char := range trimmed {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			continue
		}
		switch char {
		case ':', '_', '-':
			continue
		default:
			return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Key 只能包含中文、字母、数字、冒号、下划线和短横线")
		}
	}
	return trimmed, nil
}

// buildRealtimeDataPointPath 将接入源和 Key 命名空间组合为稳定、可读的数据点路径。
// Redis 常见的冒号分层和用户输入的路径分隔符都转换为数据点层级。
func buildRealtimeDataPointPath(provider, connectionName, key string) string {
	prefix := "realtime"
	if strings.EqualFold(strings.TrimSpace(provider), "redis") {
		prefix = "redis"
	}
	segments := []string{prefix, normalizeDatapointSegment(connectionName)}
	for _, segment := range realtimeKeySegments(key) {
		segments = append(segments, normalizeDatapointSegment(segment))
	}
	if len(segments) == 2 {
		segments = append(segments, "unnamed")
	}
	return strings.Join(segments, ".")
}

func realtimeKeyDisplayName(key string) string {
	segments := realtimeKeySegments(key)
	if len(segments) == 0 {
		return strings.TrimSpace(key)
	}
	return strings.TrimSpace(segments[len(segments)-1])
}

func realtimeKeySegments(key string) []string {
	return strings.FieldsFunc(strings.TrimSpace(key), func(char rune) bool {
		switch char {
		case ':', '.', '/', '\\':
			return true
		default:
			return false
		}
	})
}

func normalizeRealtimeBatchKeys(keys []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, key)
	}
	return result
}

func normalizeRedisType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "none" || value == "unknown" {
		return "string"
	}
	return value
}

func normalizeRealtimeValueType(value string) string {
	if normalized, ok := externalDataPointType(value); ok {
		return normalized
	}
	return "object"
}

// inferRealtimeDataPointType 在未显式指定类型时，根据 Key 类型和值推断数据点契约。
// Redis String 的 JSON 编辑模式会继续解析内容，避免对象或数组被错误标记为 string。
func inferRealtimeDataPointType(requested, redisType, valueType string, value any) string {
	if strings.TrimSpace(requested) != "" {
		return normalizeRealtimeValueType(requested)
	}
	switch normalizeRedisType(redisType) {
	case "hash":
		return "object"
	case "list", "set", "zset", "stream":
		return "array"
	case "string":
		if strings.EqualFold(strings.TrimSpace(valueType), "json") {
			if text, ok := value.(string); ok {
				var decoded any
				if json.Unmarshal([]byte(text), &decoded) == nil {
					value = decoded
				}
			}
		}
		return inferRealtimeValueDataType(value)
	default:
		return inferRealtimeValueDataType(value)
	}
}

func inferRealtimeValueDataType(value any) string {
	switch typed := value.(type) {
	case bool:
		return "bool"
	case string:
		return "string"
	case float32:
		return "float32"
	case float64:
		if math.Trunc(typed) == typed {
			return "int64"
		}
		return "float64"
	case int, int8, int16, int32, int64:
		return "int64"
	case uint, uint8, uint16, uint32, uint64:
		return "uint64"
	case []any, []string:
		return "array"
	case map[string]any, map[string]string:
		return "object"
	default:
		return "object"
	}
}

func filterRealtimeKeys(items []RealtimeStoreKeySummary, q, group string) []RealtimeStoreKeySummary {
	q = strings.ToLower(strings.TrimSpace(q))
	group = strings.TrimSpace(group)
	result := make([]RealtimeStoreKeySummary, 0, len(items))
	for _, item := range items {
		if q != "" && !strings.Contains(strings.ToLower(item.Key), q) {
			continue
		}
		if group != "" && realtimeKeyGroup(item.Key) != group {
			continue
		}
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
}

func buildRealtimeGroups(items []RealtimeStoreKeySummary) []RealtimeStoreGroup {
	counts := map[string]int{}
	for _, item := range items {
		counts[realtimeKeyGroup(item.Key)]++
	}
	groups := make([]RealtimeStoreGroup, 0, len(counts))
	for name, count := range counts {
		groups = append(groups, RealtimeStoreGroup{Name: name, Count: count})
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].Name == "未分组" {
			return false
		}
		if groups[j].Name == "未分组" {
			return true
		}
		return groups[i].Name < groups[j].Name
	})
	return groups
}

func realtimeKeyGroup(key string) string {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" {
		return strings.TrimSpace(parts[0])
	}
	return "未分组"
}

func applyRealtimeMetadata(item *RealtimeStoreKeySummary, record repository.RealtimeKeyRecord) {
	item.ID = record.ID
	item.ValueType = record.ValueType
	item.DataPointID = derefString(record.DataPointID)
	item.DataPointPath = derefString(record.DataPointPath)
	item.Outputs = sourceOutputsFromRecords(record.Outputs)
	if record.RedisType != "" {
		item.Type = record.RedisType
	}
}

func applyRealtimeValueMetadata(value *RealtimeStoreValue, record *repository.RealtimeKeyRecord) {
	value.ID = record.ID
	if record.RedisType != "" {
		value.Type = record.RedisType
	}
	value.ValueType = record.ValueType
	value.DataPointID = derefString(record.DataPointID)
	value.DataPointPath = derefString(record.DataPointPath)
	value.Outputs = sourceOutputsFromRecords(record.Outputs)
}

// normalizeBuiltinRealtimeValue 保持 IF 实时库的逻辑类型与 Redis 工作台一致。
// 内置运行库物理上使用 String 保存 JSON，因此 Hash 必须在落库前转换为字段对象，
// 避免再次读取时退化为无法编辑的行数组。
func normalizeBuiltinRealtimeValue(redisType string, value any) (any, error) {
	if err := validateRealtimeRedisValue(redisType, value); err != nil {
		return nil, err
	}
	if normalizeRedisType(redisType) == "hash" {
		return stringMapFromAny(value), nil
	}
	return value, nil
}

func stringMapFromAny(value any) map[string]string {
	result := map[string]string{}
	switch typed := value.(type) {
	case map[string]string:
		return typed
	case map[string]any:
		for key, item := range typed {
			result[key] = stringifyRealtimeValue(item)
		}
	case []any:
		for _, item := range typed {
			row, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := strings.TrimSpace(toString(row["key"]))
			if key == "" {
				continue
			}
			result[key] = stringifyRealtimeValue(row["value"])
		}
	}
	return result
}

func stringSliceFromAny(value any) []any {
	switch typed := value.(type) {
	case []string:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, item)
		}
		return result
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			if row, ok := item.(map[string]any); ok {
				result = append(result, stringifyRealtimeValue(row["value"]))
			} else {
				result = append(result, stringifyRealtimeValue(item))
			}
		}
		return result
	default:
		if value == nil {
			return []any{}
		}
		return []any{stringifyRealtimeValue(value)}
	}
}

func zsetMembersFromAny(value any) []redis.Z {
	rows, ok := value.([]any)
	if !ok {
		return []redis.Z{}
	}
	result := make([]redis.Z, 0, len(rows))
	for _, item := range rows {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		member := stringifyRealtimeValue(row["member"])
		score := floatFromAny(row["score"])
		result = append(result, redis.Z{Member: member, Score: score})
	}
	return result
}

func stringifyRealtimeValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		payload, err := json.Marshal(typed)
		if err == nil {
			return string(payload)
		}
		return fmt.Sprintf("%v", typed)
	}
}

func floatFromAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		parsed, _ := typed.Float64()
		return parsed
	default:
		var parsed float64
		_, _ = fmt.Sscanf(fmt.Sprintf("%v", typed), "%f", &parsed)
		return parsed
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
