package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// SessionKeyFmt 保存 preview 会话主状态。
	SessionKeyFmt = "preview:session:%s"
	// SessionConnKeyFmt 保存会话内连接态（后续 read/subscribe 会复用）。
	SessionConnKeyFmt = "preview:session:%s:connections"
	// SessionSubKeyFmt 保存会话内订阅态（后续 read/subscribe 会复用）。
	SessionSubKeyFmt = "preview:session:%s:subscriptions"
	// SessionTTL 约定预览会话的滑动过期时长固定为 30 分钟。
	SessionTTL = 30 * time.Minute

	defaultRedisDialTimeout = 3 * time.Second
)

// RedisConfig 描述 Redis 客户端初始化所需参数。
type RedisConfig struct {
	Addr        string
	Password    string
	DB          int
	DialTimeout time.Duration
}

// RedisClient 封装 preview 会话所需的最小 Redis 操作集合。
type RedisClient struct {
	client *redis.Client
}

// NewRedisClient 初始化 Redis 连接并做一次 Ping 探活。
func NewRedisClient(ctx context.Context, cfg RedisConfig) (*RedisClient, error) {
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		return nil, fmt.Errorf("redis addr 不能为空")
	}

	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = defaultRedisDialTimeout
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.DialTimeout,
		WriteTimeout: cfg.DialTimeout,
	})

	pingCtx, cancel := context.WithTimeout(ctx, cfg.DialTimeout)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis 连接失败: %w", err)
	}

	return &RedisClient{client: client}, nil
}

// Close 释放 Redis 连接资源。
func (c *RedisClient) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

// UpsertPreviewSession 覆盖写会话主状态，并刷新三组 key 的 TTL。
// 这里使用 SetNX+Expire 处理连接/订阅 key，避免覆盖后续流程写入的真实内容。
func (c *RedisClient) UpsertPreviewSession(ctx context.Context, sessionID string, payload map[string]any, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("redis client 未初始化")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("sessionID 不能为空")
	}
	if ttl <= 0 {
		ttl = SessionTTL
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化 preview 会话缓存失败: %w", err)
	}

	sessionKey, sessionConnKey, sessionSubKey := SessionKeys(sessionID)

	pipe := c.client.TxPipeline()
	pipe.Set(ctx, sessionKey, payloadBytes, ttl)
	pipe.SetNX(ctx, sessionConnKey, "{}", 0)
	pipe.Expire(ctx, sessionConnKey, ttl)
	pipe.SetNX(ctx, sessionSubKey, "{}", 0)
	pipe.Expire(ctx, sessionSubKey, ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("写入 preview 会话缓存失败: %w", err)
	}
	return nil
}

// DeletePreviewSession 删除 preview 会话相关的全部 Redis key。
func (c *RedisClient) DeletePreviewSession(ctx context.Context, sessionID string) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("redis client 未初始化")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("sessionID 不能为空")
	}

	sessionKey, sessionConnKey, sessionSubKey := SessionKeys(sessionID)
	if err := c.client.Del(ctx, sessionKey, sessionConnKey, sessionSubKey).Err(); err != nil {
		return fmt.Errorf("删除 preview 会话缓存失败: %w", err)
	}
	return nil
}

// SessionKeys 按约定格式返回某个会话的全部 Redis key。
func SessionKeys(sessionID string) (sessionKey string, sessionConnKey string, sessionSubKey string) {
	return fmt.Sprintf(SessionKeyFmt, sessionID), fmt.Sprintf(SessionConnKeyFmt, sessionID), fmt.Sprintf(SessionSubKeyFmt, sessionID)
}
