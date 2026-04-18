package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultMaxConns          int32 = 10
	defaultMinConns          int32 = 1
	defaultMaxConnLifetime         = time.Hour
	defaultMaxConnIdleTime         = 30 * time.Minute
	defaultHealthCheckPeriod       = time.Minute
	defaultConnectionTimeout       = 5 * time.Second
	defaultPingRetries             = 3
	defaultPingRetryDelay          = 200 * time.Millisecond
)

// PoolConfig 定义 PostgreSQL 连接池所需的最小配置。
type PoolConfig struct {
	DatabaseURL       string
	SearchPath        string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	ConnectionTimeout time.Duration
	PingRetries       int
	PingRetryDelay    time.Duration
}

// NewPool 使用统一默认值创建 PostgreSQL 连接池。
func NewPool(ctx context.Context, cfg PoolConfig) (*pgxpool.Pool, error) {
	databaseURL := strings.TrimSpace(cfg.DatabaseURL)
	if databaseURL == "" {
		return nil, fmt.Errorf("database url 不能为空")
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("解析 PostgreSQL 连接串失败: %w", err)
	}

	if cfg.MaxConns <= 0 {
		cfg.MaxConns = defaultMaxConns
	}
	if cfg.MinConns < 0 {
		cfg.MinConns = 0
	}
	if cfg.MinConns == 0 {
		cfg.MinConns = defaultMinConns
	}
	if cfg.ConnectionTimeout <= 0 {
		cfg.ConnectionTimeout = defaultConnectionTimeout
	}
	if cfg.MaxConnLifetime <= 0 {
		cfg.MaxConnLifetime = defaultMaxConnLifetime
	}
	if cfg.MaxConnIdleTime <= 0 {
		cfg.MaxConnIdleTime = defaultMaxConnIdleTime
	}
	if cfg.HealthCheckPeriod <= 0 {
		cfg.HealthCheckPeriod = defaultHealthCheckPeriod
	}
	if cfg.PingRetries <= 0 {
		cfg.PingRetries = defaultPingRetries
	}
	if cfg.PingRetryDelay <= 0 {
		cfg.PingRetryDelay = defaultPingRetryDelay
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod
	if poolConfig.ConnConfig.RuntimeParams == nil {
		poolConfig.ConnConfig.RuntimeParams = make(map[string]string)
	}
	if searchPath := strings.TrimSpace(cfg.SearchPath); searchPath != "" {
		poolConfig.ConnConfig.RuntimeParams["search_path"] = searchPath
	}

	// 迁移会直接执行多条 SQL 语句，这里统一切到 simple protocol。
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	connectCtx, cancel := context.WithTimeout(ctx, cfg.ConnectionTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("创建 PostgreSQL 连接池失败: %w", err)
	}

	if err := pingWithRetry(ctx, pool, cfg.ConnectionTimeout, cfg.PingRetries, cfg.PingRetryDelay); err != nil {
		pool.Close()
		return nil, fmt.Errorf("连接 PostgreSQL 失败: %w", err)
	}

	return pool, nil
}

// NewPoolFromURL 允许调用方直接使用数据库连接串创建连接池。
func NewPoolFromURL(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	return NewPool(ctx, PoolConfig{
		DatabaseURL: databaseURL,
	})
}

func pingWithRetry(ctx context.Context, pool *pgxpool.Pool, timeout time.Duration, retries int, retryDelay time.Duration) error {
	var lastErr error

	for attempt := 1; attempt <= retries; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, timeout)
		lastErr = pool.Ping(pingCtx)
		cancel()
		if lastErr == nil {
			return nil
		}

		if attempt == retries {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay):
		}
	}

	return lastErr
}
