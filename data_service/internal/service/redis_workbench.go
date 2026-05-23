package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
	"github.com/redis/go-redis/v9"
)

const defaultRedisWorkbenchLimit = 200

// RedisWorkbenchKey 表示 RedisManager 风格 key 浏览器中的一行。
type RedisWorkbenchKey struct {
	Key  string `json:"key"`
	Type string `json:"type"`
	TTL  int64  `json:"ttl"`
	Size int64  `json:"size"`
}

// RedisWorkbenchKeyList 返回 key 浏览结果。
type RedisWorkbenchKeyList struct {
	List []RedisWorkbenchKey `json:"list"`
}

// RedisWorkbenchValue 返回单个 key 的类型和值。
type RedisWorkbenchValue struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value any    `json:"value"`
	TTL   int64  `json:"ttl"`
}

// RedisCommandResult 返回受限命令执行结果。
type RedisCommandResult struct {
	Command string `json:"command"`
	Result  any    `json:"result"`
}

func (s *ConnectionService) loadProtocolConnection(ctx context.Context, projectID, connectionID, expectedType string) (*repository.ConnectionRecord, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateConnectionID(connectionID); err != nil {
		return nil, err
	}
	connection, err := s.repository.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if connection.Type != expectedType {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "接入源类型不匹配")
	}
	return connection, nil
}

// ListRedisKeys 扫描 Redis key，并读取类型、TTL 与基础尺寸。
func (s *ConnectionService) ListRedisKeys(ctx context.Context, projectID, connectionID, pattern string, limit int) (*RedisWorkbenchKeyList, error) {
	connection, err := s.loadProtocolConnection(ctx, projectID, connectionID, "redis")
	if err != nil {
		return nil, err
	}
	client, err := newRedisPreviewClient(connection.Config, mapFromAny(connection.Config["options"]))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		pattern = strings.TrimSpace(toString(connection.Config["keyPattern"]))
	}
	if pattern == "" {
		pattern = "*"
	}
	if limit <= 0 || limit > 1000 {
		limit = defaultRedisWorkbenchLimit
	}

	keys, err := scanRedisPreviewKeys(ctx, client, pattern, limit)
	if err != nil {
		return nil, err
	}
	items := make([]RedisWorkbenchKey, 0, len(keys))
	for _, key := range keys {
		keyType, typeErr := client.Type(ctx, key).Result()
		if typeErr != nil {
			keyType = "unknown"
		}
		ttl, _ := client.TTL(ctx, key).Result()
		items = append(items, RedisWorkbenchKey{
			Key:  key,
			Type: keyType,
			TTL:  int64(ttl.Seconds()),
			Size: redisWorkbenchSize(ctx, client, key, keyType),
		})
	}
	return &RedisWorkbenchKeyList{List: items}, nil
}

// GetRedisValue 读取单个 Redis key 的当前值。
func (s *ConnectionService) GetRedisValue(ctx context.Context, projectID, connectionID, key string) (*RedisWorkbenchValue, error) {
	connection, err := s.loadProtocolConnection(ctx, projectID, connectionID, "redis")
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "key 不能为空")
	}
	client, err := newRedisPreviewClient(connection.Config, mapFromAny(connection.Config["options"]))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	sample, err := readRedisPreviewValue(ctx, client, key)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "读取 Redis key 失败", err)
	}
	ttl, _ := client.TTL(ctx, key).Result()
	return &RedisWorkbenchValue{
		Key:   key,
		Type:  toString(sample["type"]),
		Value: sample["value"],
		TTL:   int64(ttl.Seconds()),
	}, nil
}

// ExecuteRedisCommand 执行 Redis 工作台受限只读命令，避免通过 UI 修改数据。
func (s *ConnectionService) ExecuteRedisCommand(ctx context.Context, projectID, connectionID, command string, args []string) (*RedisCommandResult, error) {
	connection, err := s.loadProtocolConnection(ctx, projectID, connectionID, "redis")
	if err != nil {
		return nil, err
	}
	command = strings.ToUpper(strings.TrimSpace(command))
	if command == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "command 不能为空")
	}
	if !allowedRedisWorkbenchCommand(command) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Redis 工作台仅支持只读命令")
	}
	client, err := newRedisPreviewClient(connection.Config, mapFromAny(connection.Config["options"]))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	cmdArgs := make([]any, 0, len(args)+1)
	cmdArgs = append(cmdArgs, command)
	for _, arg := range args {
		cmdArgs = append(cmdArgs, arg)
	}
	result, err := client.Do(ctx, cmdArgs...).Result()
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "执行 Redis 命令失败", err)
	}
	return &RedisCommandResult{
		Command: strings.TrimSpace(command + " " + strings.Join(args, " ")),
		Result:  normalizeRedisCommandResult(result),
	}, nil
}

func allowedRedisWorkbenchCommand(command string) bool {
	switch command {
	case "PING", "GET", "HGETALL", "LRANGE", "SMEMBERS", "ZRANGE", "TYPE", "TTL", "EXISTS", "SCAN":
		return true
	default:
		return false
	}
}

func redisWorkbenchSize(ctx context.Context, client redis.UniversalClient, key, keyType string) int64 {
	switch keyType {
	case "string":
		size, _ := client.StrLen(ctx, key).Result()
		return size
	case "hash":
		size, _ := client.HLen(ctx, key).Result()
		return size
	case "list":
		size, _ := client.LLen(ctx, key).Result()
		return size
	case "set":
		size, _ := client.SCard(ctx, key).Result()
		return size
	case "zset":
		size, _ := client.ZCard(ctx, key).Result()
		return size
	default:
		return 0
	}
}

func normalizeRedisCommandResult(value any) any {
	switch typed := value.(type) {
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, normalizeRedisCommandResult(item))
		}
		return result
	case []string:
		return typed
	case map[string]string:
		return typed
	case string, int64, bool, nil:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}
