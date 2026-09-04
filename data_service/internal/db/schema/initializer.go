package schema

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const initializationLockKey = int64(4_000_001)

// Initializer 仅负责为空数据库创建当前最终结构，不修改已有数据库。
type Initializer struct {
	pool *pgxpool.Pool
}

// NewInitializer 创建数据库结构初始化器。
func NewInitializer(pool *pgxpool.Pool) (*Initializer, error) {
	if pool == nil {
		return nil, fmt.Errorf("pool 不能为空")
	}
	return &Initializer{pool: pool}, nil
}

// Ensure 在当前 search_path 没有业务表时执行完整建表；已有表时直接跳过。
func (i *Initializer) Ensure(ctx context.Context) error {
	tx, err := i.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("开启数据库结构初始化事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, initializationLockKey); err != nil {
		return fmt.Errorf("获取数据库结构初始化锁失败: %w", err)
	}

	var hasTables bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_catalog.pg_tables
			WHERE schemaname = current_schema()
		)
	`).Scan(&hasTables); err != nil {
		return fmt.Errorf("检查数据库是否为空失败: %w", err)
	}
	if hasTables {
		for table, columns := range map[string][]string{
			"data_project_tenant_bindings": {"authoring_epoch"},
			"data_authoring_fences":        {"project_id", "tenant_id", "owner_id", "mode", "token_hash", "expires_at"},
		} {
			for _, column := range columns {
				var exists bool
				if err := tx.QueryRow(ctx, `
					SELECT EXISTS (
						SELECT 1 FROM information_schema.columns
						WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2
					)
				`, table, column).Scan(&exists); err != nil {
					return fmt.Errorf("校验数据域数据库字段 %s.%s 失败: %w", table, column, err)
				}
				if !exists {
					return fmt.Errorf("数据域数据库结构不兼容，缺少必需字段 %s.%s；请按当前最终基线重建数据库", table, column)
				}
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("提交数据库结构检查事务失败: %w", err)
		}
		return nil
	}

	payload, err := Files.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("读取数据库结构基线失败: %w", err)
	}
	if _, err := tx.Exec(ctx, string(payload)); err != nil {
		return fmt.Errorf("创建数据库结构失败: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交数据库结构初始化事务失败: %w", err)
	}
	return nil
}
