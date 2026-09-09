package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	// BuiltinDemoProjectID 是全新环境内置教程工程的稳定标识，供设计器识别并装载示例模板。
	BuiltinDemoProjectID = "00000000-0000-4000-8000-000000000001"
	demoRuntimeRoleID    = "00000000-0000-4000-8000-000000000002"
	demoRuntimeUserID    = "00000000-0000-4000-8000-000000000003"
)

type SeedConfig struct {
	TenantID               string
	TenantName             string
	TenantCode             string
	SuperAdminUserID       string
	SuperAdminUsername     string
	SuperAdminPasswordHash string
}

// EnsureInitialData 只在租户表为空时写入待初始化租户与必须改密的平台账号，不修改任何已有业务数据。
type seedPool interface {
	Begin(context.Context) (pgx.Tx, error)
}

func EnsureInitialData(ctx context.Context, pool seedPool, config SeedConfig) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开始默认数据事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// 安装进程可能并发执行，锁定后重新检查，绝不覆盖已有租户或账号。
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(731904821)`); err != nil {
		return fmt.Errorf("锁定默认数据初始化失败: %w", err)
	}
	var tenantCount int64
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM tenants`).Scan(&tenantCount); err != nil {
		return fmt.Errorf("检查默认数据失败: %w", err)
	}
	if tenantCount > 0 {
		return tx.Commit(ctx)
	}
	if strings.TrimSpace(config.TenantID) == "" || strings.TrimSpace(config.SuperAdminUserID) == "" || strings.TrimSpace(config.SuperAdminPasswordHash) == "" {
		return fmt.Errorf("空库初始化需要默认租户与平台管理员配置")
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO tenants (id, name, code, status, initialized, is_default)
		VALUES ($1, $2, $3, 'active', false, true)
	`, config.TenantID, config.TenantName, config.TenantCode); err != nil {
		return fmt.Errorf("创建默认租户失败: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id, username, password_hash, full_name, role, status, must_change_password)
		VALUES ($1, $2, $3, $4, 'SUPER_ADMIN', 'active', true)
	`, config.SuperAdminUserID, config.SuperAdminUsername, config.SuperAdminPasswordHash, config.SuperAdminUsername); err != nil {
		return fmt.Errorf("创建默认超级管理员失败: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交默认数据事务失败: %w", err)
	}
	return nil
}
