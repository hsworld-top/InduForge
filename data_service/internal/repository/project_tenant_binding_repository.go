package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProjectTenantBindingRepository 持久化 control 面写入的项目租户归属。
type ProjectTenantBindingRepository struct {
	pool *pgxpool.Pool
}

// NewProjectTenantBindingRepository 创建项目租户绑定仓储。
func NewProjectTenantBindingRepository(pool *pgxpool.Pool) *ProjectTenantBindingRepository {
	return &ProjectTenantBindingRepository{pool: pool}
}

// BindIfUnbound 首次绑定项目租户；已绑定同一租户时保持幂等，不同租户由调用方拒绝改绑。
func (r *ProjectTenantBindingRepository) BindIfUnbound(ctx context.Context, projectID, tenantID string) (boundTenantID string, created bool, err error) {
	err = r.pool.QueryRow(ctx, `
		INSERT INTO data_project_tenant_bindings (project_id, tenant_id)
		VALUES ($1, $2)
		ON CONFLICT (project_id) DO NOTHING
		RETURNING tenant_id
	`, projectID, tenantID).Scan(&boundTenantID)
	if err == nil {
		return boundTenantID, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", false, fmt.Errorf("写入项目租户绑定失败: %w", err)
	}

	err = r.pool.QueryRow(ctx, `
		SELECT tenant_id
		FROM data_project_tenant_bindings
		WHERE project_id = $1
	`, projectID).Scan(&boundTenantID)
	if err != nil {
		return "", false, fmt.Errorf("读取项目租户绑定失败: %w", err)
	}
	return boundTenantID, false, nil
}
