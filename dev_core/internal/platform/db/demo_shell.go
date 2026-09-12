package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const DemoShellName = "工程模板"

// CreateDemoShell 只创建工程元数据；示例页面、数据源、报警和运行用户交由对应模块初始化。
// 每个组织使用独立工程 ID，不再依赖全局固定 ID 触发旧示例装载。
func CreateDemoShell(ctx context.Context, tx pgx.Tx, tenantID, actorID, projectID, workspacePath string) error {
	_, err := tx.Exec(ctx, `INSERT INTO projects(id,tenant_id,name,code,description,workspace_path,status,visibility,created_by) VALUES($1,$2,$3,'organization_project_template','用于了解工程的页面开发、数据接入、报警配置与部署交付能力，可作为新工程的起点。',$4,'active','internal',$5)`, projectID, tenantID, DemoShellName, workspacePath, actorID)
	return err
}

// CreateDemoShellWithRuntimeAccess 创建模板工程时同步建立与普通工程一致的
// 内置 ADMIN/admin 运行身份。模板是可直接打开和发布的工程，不能因为走示例
// 初始化路径而缺少默认登录身份。
func CreateDemoShellWithRuntimeAccess(ctx context.Context, tx pgx.Tx, tenantID, actorID, projectID, workspacePath, runtimeAdminHash string) error {
	if err := CreateDemoShell(ctx, tx, tenantID, actorID, projectID, workspacePath); err != nil {
		return err
	}
	roleID, runtimeUserID := uuid.NewString(), uuid.NewString()
	if _, err := tx.Exec(ctx, `INSERT INTO project_roles (id, project_id, code, name, description, status, is_builtin, created_by) VALUES ($1,$2,'ADMIN','管理员','工程内置管理员','active',true,$3)`, roleID, projectID, actorID); err != nil {
		return fmt.Errorf("创建模板默认运行角色失败: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO project_role_grants (project_id, role_id, capability, effect) VALUES ($1,$2,'project.*','allow')`, projectID, roleID); err != nil {
		return fmt.Errorf("创建模板默认运行授权失败: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO project_runtime_users (id, project_id, username, password_hash, display_name, status, is_builtin_admin, created_by) VALUES ($1,$2,'admin',$3,'管理员','active',true,$4)`, runtimeUserID, projectID, runtimeAdminHash, actorID); err != nil {
		return fmt.Errorf("创建模板默认运行用户失败: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO project_user_role_bindings (runtime_user_id, role_id) VALUES ($1,$2)`, runtimeUserID, roleID); err != nil {
		return fmt.Errorf("绑定模板默认运行角色失败: %w", err)
	}
	return nil
}
