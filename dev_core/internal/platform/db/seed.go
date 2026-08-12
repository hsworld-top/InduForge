package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SeedConfig struct {
	TenantID                 string
	TenantName               string
	TenantCode               string
	SuperAdminUserID         string
	SuperAdminUsername       string
	SuperAdminPasswordHash   string
	DefaultAdminUsername     string
	DefaultAdminPasswordHash string
}

// EnsureInitialData 只在租户表为空时写入默认租户、超级管理员和租户管理员，不修改任何已有业务数据。
func EnsureInitialData(ctx context.Context, pool *pgxpool.Pool, config SeedConfig) error {
	var tenantCount int64
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tenants`).Scan(&tenantCount); err != nil {
		return fmt.Errorf("检查默认数据失败: %w", err)
	}
	if tenantCount > 0 {
		return nil
	}
	if strings.TrimSpace(config.TenantID) == "" || strings.TrimSpace(config.SuperAdminUserID) == "" || strings.TrimSpace(config.SuperAdminPasswordHash) == "" || strings.TrimSpace(config.DefaultAdminUsername) == "" || strings.TrimSpace(config.DefaultAdminPasswordHash) == "" {
		return fmt.Errorf("空库初始化需要默认租户、超级管理员和默认租户管理员配置")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开始默认数据事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO tenants (id, name, code, status)
		VALUES ($1, $2, $3, 'active')
	`, config.TenantID, config.TenantName, config.TenantCode); err != nil {
		return fmt.Errorf("创建默认租户失败: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id, tenant_id, username, password_hash, full_name, role, status)
		VALUES ($1, $2, $3, $4, $5, 'SUPER_ADMIN', 'active')
	`, config.SuperAdminUserID, config.TenantID, config.SuperAdminUsername, config.SuperAdminPasswordHash, config.SuperAdminUsername); err != nil {
		return fmt.Errorf("创建默认超级管理员失败: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (tenant_id, username, password_hash, full_name, role, status)
		VALUES ($1, $2, $3, $2, 'SYSTEM_ADMIN', 'active')
	`, config.TenantID, config.DefaultAdminUsername, config.DefaultAdminPasswordHash); err != nil {
		return fmt.Errorf("创建默认租户管理员失败: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交默认数据事务失败: %w", err)
	}
	return nil
}
