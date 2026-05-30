package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

const defaultRealtimeStoreLimit = 500

type RealtimeStoreKeySummary struct {
	ID            string `json:"id,omitempty"`
	Key           string `json:"key"`
	Type          string `json:"type"`
	TTL           int64  `json:"ttl"`
	Size          int64  `json:"size"`
	Provider      string `json:"provider"`
	ValueType     string `json:"valueType"`
	DataPointID   string `json:"dataPointId,omitempty"`
	DataPointPath string `json:"dataPointPath,omitempty"`
}

type RealtimeStoreGroup struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type RealtimeStoreKeyList struct {
	List   []RealtimeStoreKeySummary `json:"list"`
	Groups []RealtimeStoreGroup      `json:"groups"`
}

type RealtimeStoreValue struct {
	ID            string `json:"id,omitempty"`
	Key           string `json:"key"`
	Type          string `json:"type"`
	TTL           int64  `json:"ttl"`
	Value         any    `json:"value"`
	Size          int64  `json:"size"`
	Provider      string `json:"provider"`
	ValueType     string `json:"valueType"`
	DataPointID   string `json:"dataPointId,omitempty"`
	DataPointPath string `json:"dataPointPath,omitempty"`
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
	DataType string `json:"dataType"`
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
	if limit <= 0 || limit > 2000 {
		limit = defaultRealtimeStoreLimit
	}
	var items []RealtimeStoreKeySummary
	if provider == "redis" {
		items, err = s.listRedisKeys(ctx, connection, q, limit)
	} else {
		items, err = s.listBuiltinKeys(ctx, connection, q)
	}
	if err != nil {
		return nil, err
	}
	metadataByKey, err := s.metadataByKey(ctx, projectID, connectionID, provider)
	if err != nil {
		return nil, err
	}
	for index := range items {
		if record, ok := metadataByKey[items[index].Key]; ok {
			applyRealtimeMetadata(&items[index], record)
		}
	}
	items = filterRealtimeKeys(items, q, group)
	groups := buildRealtimeGroups(items)
	return &RealtimeStoreKeyList{List: items, Groups: groups}, nil
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

func (s *RealtimeStoreService) SaveKey(ctx context.Context, projectID, connectionID string, input SaveRealtimeStoreKeyInput) (*RealtimeStoreValue, error) {
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
	} else if err := s.saveBuiltinKey(ctx, connection, key, input.Value, ttlSeconds); err != nil {
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
	newKey, err := normalizeRealtimeKey(input.NewKey)
	if err != nil {
		return nil, err
	}
	if oldKey == newKey {
		return s.GetKey(ctx, projectID, connectionID, oldKey)
	}
	if provider == "redis" {
		if err := s.withRedisClient(ctx, connection, func(client redis.UniversalClient) error {
			return client.Rename(ctx, oldKey, newKey).Err()
		}); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "重命名 Redis key 失败", err)
		}
	} else {
		oldFullKey, newFullKey, err := s.builtinFullKeys(projectID, connection, oldKey, newKey)
		if err != nil {
			return nil, err
		}
		if err := s.builtinRuntime.realtimeClient.Rename(ctx, oldFullKey, newFullKey).Err(); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "重命名 IF 实时库 key 失败", err)
		}
	}
	_, _ = s.repository.Rename(ctx, projectID, connectionID, provider, oldKey, newKey)
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

func (s *RealtimeStoreService) CreateDataPoint(ctx context.Context, projectID, connectionID, key, userID string, input CreateRealtimeKeyDataPointInput) (*repository.DataPointRecord, error) {
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
	record, err := s.repository.Upsert(ctx, repository.UpsertRealtimeKeyParams{
		ProjectID:         projectID,
		ConnectionID:      connectionID,
		Provider:          provider,
		KeyPath:           key,
		RedisType:         value.Type,
		ValueType:         normalizeRealtimeValueType(input.DataType),
		DefaultTtlSeconds: int(value.TTL),
		Description:       "",
	})
	if err != nil {
		return nil, err
	}
	config := map[string]any{
		"provider":     provider,
		"connectionId": connectionID,
		"keyId":        record.ID,
		"key":          key,
		"redisType":    value.Type,
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
	return s.repository.CreateDataPoint(ctx, repository.CreateRealtimeKeyDataPointParams{
		ProjectID:    projectID,
		ConnectionID: connectionID,
		KeyID:        record.ID,
		KeyPath:      key,
		Provider:     provider,
		RedisType:    value.Type,
		DataType:     normalizeRealtimeValueType(input.DataType),
		SourceConfig: config,
		UserID:       userPtr,
	})
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

func (s *RealtimeStoreService) listRedisKeys(ctx context.Context, connection *repository.ConnectionRecord, q string, limit int) ([]RealtimeStoreKeySummary, error) {
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
		result = make([]RealtimeStoreKeySummary, 0, len(keys))
		for _, key := range keys {
			keyType, _ := client.Type(ctx, key).Result()
			ttl, _ := client.TTL(ctx, key).Result()
			result = append(result, RealtimeStoreKeySummary{
				Key:       key,
				Type:      normalizeRedisType(keyType),
				TTL:       int64(ttl.Seconds()),
				Size:      redisWorkbenchSize(ctx, client, key, keyType),
				Provider:  "redis",
				ValueType: "object",
			})
		}
		return nil
	})
	return result, err
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
	return s.withRedisClient(ctx, connection, func(client redis.UniversalClient) error {
		if err := writeRealtimeRedisValue(ctx, client, key, redisType, value); err != nil {
			return err
		}
		return applyRealtimeTTL(ctx, client, key, ttlSeconds)
	})
}

func (s *RealtimeStoreService) saveBuiltinKey(ctx context.Context, connection *repository.ConnectionRecord, key string, value any, ttlSeconds int) error {
	fullKey, err := s.builtinFullKey(connection.ProjectID, connection, key)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(value)
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

func writeRealtimeRedisValue(ctx context.Context, client redis.UniversalClient, key, keyType string, value any) error {
	keyType = normalizeRedisType(keyType)
	if keyType == "stream" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "stream 第一版仅支持只读")
	}
	if err := client.Del(ctx, key).Err(); err != nil {
		return err
	}
	switch keyType {
	case "string":
		return client.Set(ctx, key, stringifyRealtimeValue(value), 0).Err()
	case "hash":
		return client.HSet(ctx, key, stringMapFromAny(value)).Err()
	case "list":
		values := stringSliceFromAny(value)
		if len(values) == 0 {
			return client.RPush(ctx, key, "").Err()
		}
		return client.RPush(ctx, key, values).Err()
	case "set":
		values := stringSliceFromAny(value)
		if len(values) == 0 {
			return client.SAdd(ctx, key, "").Err()
		}
		return client.SAdd(ctx, key, values).Err()
	case "zset":
		return client.ZAdd(ctx, key, zsetMembersFromAny(value)...).Err()
	default:
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时库 key 类型不受支持")
	}
}

func applyRealtimeTTL(ctx context.Context, client redis.UniversalClient, key string, ttlSeconds int) error {
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
	key, err := normalizeRealtimeKey(input.Key)
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
	return key, redisType, normalizeRealtimeValueType(input.ValueType), input.TTLSeconds, nil
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

func normalizeRedisType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "none" || value == "unknown" {
		return "string"
	}
	return value
}

func normalizeRealtimeValueType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "string", "number", "boolean", "object", "array":
		return strings.ToLower(strings.TrimSpace(value))
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
	if record.RedisType != "" {
		item.Type = record.RedisType
	}
}

func applyRealtimeValueMetadata(value *RealtimeStoreValue, record *repository.RealtimeKeyRecord) {
	value.ID = record.ID
	value.ValueType = record.ValueType
	value.DataPointID = derefString(record.DataPointID)
	value.DataPointPath = derefString(record.DataPointPath)
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
