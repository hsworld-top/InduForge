package db

import (
	"context"
	"github.com/jackc/pgx/v5"
)

const DemoShellName = "工程模板"

// CreateDemoShell 只创建工程元数据；示例页面、数据源、报警和运行用户交由对应模块初始化。
// 每个组织使用独立工程 ID，不再依赖全局固定 ID 触发旧示例装载。
func CreateDemoShell(ctx context.Context, tx pgx.Tx, tenantID, actorID, projectID, workspacePath string) error {
	_, err := tx.Exec(ctx, `INSERT INTO projects(id,tenant_id,name,code,description,workspace_path,status,visibility,created_by) VALUES($1,$2,$3,'organization_project_template','用于了解工程的页面开发、数据接入、报警配置与部署交付能力，可作为新工程的起点。',$4,'active','internal',$5)`, projectID, tenantID, DemoShellName, workspacePath, actorID)
	return err
}
