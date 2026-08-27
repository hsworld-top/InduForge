package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// BuiltinDemoProjectID 是全新环境内置教程工程的稳定标识，供设计器识别并装载示例模板。
	BuiltinDemoProjectID = "00000000-0000-4000-8000-000000000001"
	demoRuntimeRoleID    = "00000000-0000-4000-8000-000000000002"
	demoRuntimeUserID    = "00000000-0000-4000-8000-000000000003"
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
	DemoWorkspacePath        string
}

// EnsureInitialData 只在租户表为空时写入默认账号和可删除的教程工程，不修改任何已有业务数据。
func EnsureInitialData(ctx context.Context, pool *pgxpool.Pool, config SeedConfig) error {
	var tenantCount int64
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tenants`).Scan(&tenantCount); err != nil {
		return fmt.Errorf("检查默认数据失败: %w", err)
	}
	if tenantCount > 0 {
		return nil
	}
	if strings.TrimSpace(config.TenantID) == "" || strings.TrimSpace(config.SuperAdminUserID) == "" || strings.TrimSpace(config.SuperAdminPasswordHash) == "" || strings.TrimSpace(config.DefaultAdminUsername) == "" || strings.TrimSpace(config.DefaultAdminPasswordHash) == "" || strings.TrimSpace(config.DemoWorkspacePath) == "" {
		return fmt.Errorf("空库初始化需要默认租户、管理员和教程工程配置")
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
	var defaultAdminUserID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (tenant_id, username, password_hash, full_name, role, status)
		VALUES ($1, $2, $3, $2, 'SYSTEM_ADMIN', 'active')
		RETURNING id::text
	`, config.TenantID, config.DefaultAdminUsername, config.DefaultAdminPasswordHash).Scan(&defaultAdminUserID); err != nil {
		return fmt.Errorf("创建默认租户管理员失败: %w", err)
	}
	if err := createBuiltinDemoProject(ctx, tx, config, defaultAdminUserID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交默认数据事务失败: %w", err)
	}
	return nil
}

// EnsureBuiltinDemoProject 为已有环境只补建一次教程工程。
// 删除工程会保留 status=deleted 墓碑，因此用户删除后不会被重新创建。
func EnsureBuiltinDemoProject(ctx context.Context, pool *pgxpool.Pool, config SeedConfig) error {
	if strings.TrimSpace(config.TenantID) == "" || strings.TrimSpace(config.DefaultAdminUsername) == "" || strings.TrimSpace(config.DefaultAdminPasswordHash) == "" || strings.TrimSpace(config.DemoWorkspacePath) == "" {
		return fmt.Errorf("教程工程初始化配置不完整")
	}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM projects WHERE id=$1 OR (tenant_id=$2 AND code='builtin_data_sdk_demo'))`, BuiltinDemoProjectID, config.TenantID).Scan(&exists); err != nil {
		return fmt.Errorf("检查内置教程工程失败: %w", err)
	}
	if exists {
		return nil
	}
	var defaultAdminUserID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text FROM users
		WHERE tenant_id=$1 AND username=$2 AND role='SYSTEM_ADMIN' AND status='active'
		ORDER BY created_at LIMIT 1
	`, config.TenantID, config.DefaultAdminUsername).Scan(&defaultAdminUserID); err != nil {
		return fmt.Errorf("读取教程工程创建人失败: %w", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开始教程工程事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := createBuiltinDemoProject(ctx, tx, config, defaultAdminUserID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交教程工程事务失败: %w", err)
	}
	return nil
}

func createBuiltinDemoProject(ctx context.Context, tx pgx.Tx, config SeedConfig, defaultAdminUserID string) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO projects (id, tenant_id, name, code, description, workspace_path, status, visibility, created_by)
		VALUES ($1, $2, '数据点与报警 Demo', 'builtin_data_sdk_demo', '内置 IF 数据、计算与页面开发教程；可像普通工程一样编辑和删除。', $3, 'active', 'internal', $4)
	`, BuiltinDemoProjectID, config.TenantID, config.DemoWorkspacePath, defaultAdminUserID); err != nil {
		return fmt.Errorf("创建内置教程工程失败: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO project_roles (id, project_id, code, name, description, status, is_builtin, created_by)
		VALUES ($1, $2, 'ADMIN', '管理员', '工程内置管理员', 'active', true, $3)
	`, demoRuntimeRoleID, BuiltinDemoProjectID, defaultAdminUserID); err != nil {
		return fmt.Errorf("创建教程工程运行角色失败: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO project_role_grants (project_id, role_id, capability, effect)
		VALUES ($1, $2, 'project.*', 'allow')
	`, BuiltinDemoProjectID, demoRuntimeRoleID); err != nil {
		return fmt.Errorf("创建教程工程运行授权失败: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO project_runtime_users (id, project_id, username, password_hash, display_name, status, is_builtin_admin, created_by)
		VALUES ($1, $2, 'admin', $3, '管理员', 'active', true, $4)
	`, demoRuntimeUserID, BuiltinDemoProjectID, config.DefaultAdminPasswordHash, defaultAdminUserID); err != nil {
		return fmt.Errorf("创建教程工程运行用户失败: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO project_user_role_bindings (runtime_user_id, role_id)
		VALUES ($1, $2)
	`, demoRuntimeUserID, demoRuntimeRoleID); err != nil {
		return fmt.Errorf("绑定教程工程运行角色失败: %w", err)
	}
	return nil
}
