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
func (r *ProjectTenantBindingRepository) BindIfUnbound(ctx context.Context, projectID, tenantID string, authoringEpoch int64) (boundTenantID string, boundEpoch int64, created bool, err error) {
	err = r.pool.QueryRow(ctx, `
		INSERT INTO data_project_tenant_bindings (project_id, tenant_id, authoring_epoch)
		VALUES ($1, $2, $3)
		ON CONFLICT (project_id) DO NOTHING
		RETURNING tenant_id, authoring_epoch
	`, projectID, tenantID, authoringEpoch).Scan(&boundTenantID, &boundEpoch)
	if err == nil {
		return boundTenantID, boundEpoch, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", 0, false, fmt.Errorf("写入项目租户绑定失败: %w", err)
	}

	err = r.pool.QueryRow(ctx, `
		SELECT tenant_id, authoring_epoch
		FROM data_project_tenant_bindings
		WHERE project_id = $1
	`, projectID).Scan(&boundTenantID, &boundEpoch)
	if err != nil {
		return "", 0, false, fmt.Errorf("读取项目租户绑定失败: %w", err)
	}
	return boundTenantID, boundEpoch, false, nil
}

func (r *ProjectTenantBindingRepository) Get(ctx context.Context, projectID, tenantID string) (int64, error) {
	var epoch int64
	err := r.pool.QueryRow(ctx, `SELECT authoring_epoch FROM data_project_tenant_bindings WHERE project_id=$1 AND tenant_id=$2`, projectID, tenantID).Scan(&epoch)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("项目租户绑定不存在")
	}
	return epoch, err
}
