package repository

import (
	"context"
	"errors"
	"net/http"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/jackc/pgx/v5"
)

type ComputeDependencyRecord struct {
	ID          string
	ProjectID   string
	Language    string
	PackageName string
	ImportName  string
	Version     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SaveComputeDependencyParams struct {
	ProjectID   string
	UserID      string
	Language    string
	PackageName string
	ImportName  string
	Version     string
}

func (r *ComputeRepository) ListDependencies(ctx context.Context, projectID string) ([]ComputeDependencyRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT id::text,project_id::text,language,package_name,import_name,version,created_at,updated_at
		FROM data_compute_dependencies WHERE project_id=$1 ORDER BY language,package_name`, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询工程依赖失败", err)
	}
	defer rows.Close()
	result := make([]ComputeDependencyRecord, 0)
	for rows.Next() {
		var item ComputeDependencyRecord
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Language, &item.PackageName, &item.ImportName, &item.Version, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取工程依赖失败", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *ComputeRepository) SaveDependency(ctx context.Context, params SaveComputeDependencyParams) (*ComputeDependencyRecord, error) {
	var item ComputeDependencyRecord
	err := r.pool.QueryRow(ctx, `INSERT INTO data_compute_dependencies(project_id,language,package_name,import_name,version,created_by)
		VALUES($1,$2,$3,$4,$5,$6)
		ON CONFLICT(project_id,language,package_name) DO UPDATE SET import_name=EXCLUDED.import_name,version=EXCLUDED.version,updated_at=now()
		RETURNING id::text,project_id::text,language,package_name,import_name,version,created_at,updated_at`,
		params.ProjectID, params.Language, params.PackageName, params.ImportName, params.Version, params.UserID,
	).Scan(&item.ID, &item.ProjectID, &item.Language, &item.PackageName, &item.ImportName, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "保存工程依赖失败", err)
	}
	return &item, nil
}

func (r *ComputeRepository) GetDependency(ctx context.Context, projectID, dependencyID string) (*ComputeDependencyRecord, error) {
	var item ComputeDependencyRecord
	err := r.pool.QueryRow(ctx, `SELECT id::text,project_id::text,language,package_name,import_name,version,created_at,updated_at
		FROM data_compute_dependencies WHERE project_id=$1 AND id=$2`, projectID, dependencyID,
	).Scan(&item.ID, &item.ProjectID, &item.Language, &item.PackageName, &item.ImportName, &item.Version, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "工程依赖不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询工程依赖失败", err)
	}
	return &item, nil
}

func (r *ComputeRepository) CountDependencyReferences(ctx context.Context, projectID, dependencyID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM data_compute_units
		WHERE project_id=$1 AND dependencies @> jsonb_build_array(jsonb_build_object('id',$2::text))`, projectID, dependencyID).Scan(&count)
	if err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "检查依赖引用失败", err)
	}
	return count, nil
}

func (r *ComputeRepository) DeleteDependency(ctx context.Context, projectID, dependencyID string) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM data_compute_dependencies WHERE project_id=$1 AND id=$2`, projectID, dependencyID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除工程依赖失败", err)
	}
	if result.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "工程依赖不存在")
	}
	return nil
}
