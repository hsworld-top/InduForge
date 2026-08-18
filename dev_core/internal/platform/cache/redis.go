package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Address  string
	Password string
	DB       int
}

type Redis struct{ client *redis.Client }

type ProjectMember struct {
	ID   string
	Name string
}

func NewRedis(ctx context.Context, config RedisConfig) (*Redis, error) {
	client := redis.NewClient(&redis.Options{Addr: config.Address, Password: config.Password, DB: config.DB})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("连接 Redis 失败: %w", err)
	}
	return &Redis{client: client}, nil
}

func (r *Redis) Close() error { return r.client.Close() }

func (r *Redis) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, "captcha:"+key, value, ttl).Err()
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	value, err := r.client.Get(ctx, "captcha:"+key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrMiss
	}
	return value, err
}

func (r *Redis) Take(ctx context.Context, key string) (string, error) {
	value, err := r.client.GetDel(ctx, "captcha:"+key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrMiss
	}
	return value, err
}

func (r *Redis) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, "captcha:"+key).Err()
}

func (r *Redis) Revoke(ctx context.Context, tokenID string, ttl time.Duration) error {
	return r.client.Set(ctx, "auth:revoked:"+tokenID, "1", ttl).Err()
}

func (r *Redis) IsRevoked(ctx context.Context, tokenID string) (bool, error) {
	count, err := r.client.Exists(ctx, "auth:revoked:"+tokenID).Result()
	return count > 0, err
}

func (r *Redis) PutSceneSession(ctx context.Context, sessionID, value string, ttl time.Duration) error {
	return r.client.Set(ctx, "scene:session:"+sessionID, value, ttl).Err()
}

func (r *Redis) GetSceneSession(ctx context.Context, sessionID string) (string, error) {
	value, err := r.client.Get(ctx, "scene:session:"+sessionID).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrMiss
	}
	return value, err
}

func (r *Redis) DeleteSceneSession(ctx context.Context, sessionID string) error {
	return r.client.Del(ctx, "scene:session:"+sessionID).Err()
}

func (r *Redis) TouchNode(ctx context.Context, nodeID string, ttl time.Duration) error {
	return r.client.Set(ctx, "node:online:"+nodeID, "1", ttl).Err()
}

// TouchProjectMember 使用短时有序集合记录在线提示，不参与工程锁定或保存冲突处理。
func (r *Redis) TouchProjectMember(ctx context.Context, projectID, userID, name string, ttl time.Duration) error {
	presenceKey := "code:presence:" + projectID
	nameKey := presenceKey + ":names"
	pipeline := r.client.TxPipeline()
	pipeline.ZAdd(ctx, presenceKey, redis.Z{Score: float64(time.Now().Add(ttl).UnixMilli()), Member: userID})
	pipeline.HSet(ctx, nameKey, userID, name)
	pipeline.Expire(ctx, presenceKey, ttl*2)
	pipeline.Expire(ctx, nameKey, ttl*2)
	_, err := pipeline.Exec(ctx)
	return err
}

func (r *Redis) ListProjectMembers(ctx context.Context, projectID string, now time.Time) ([]ProjectMember, error) {
	presenceKey := "code:presence:" + projectID
	nameKey := presenceKey + ":names"
	if err := r.client.ZRemRangeByScore(ctx, presenceKey, "-inf", strconv.FormatInt(now.UnixMilli(), 10)).Err(); err != nil {
		return nil, err
	}
	ids, err := r.client.ZRange(ctx, presenceKey, 0, -1).Result()
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	names, err := r.client.HMGet(ctx, nameKey, ids...).Result()
	if err != nil {
		return nil, err
	}
	members := make([]ProjectMember, 0, len(ids))
	for index, id := range ids {
		name := id
		if index < len(names) && names[index] != nil {
			name = fmt.Sprint(names[index])
		}
		members = append(members, ProjectMember{ID: id, Name: name})
	}
	return members, nil
}
