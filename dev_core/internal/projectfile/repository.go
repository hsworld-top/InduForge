package projectfile

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type record struct {
	File
	TenantID  string
	ObjectKey string
	CreatedBy string
}

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// EnsureSchema 为已有控制面数据库补齐工程对象库表；空库初始化仍以 core-schema.sql 为准。
func EnsureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS project_files (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  path text NOT NULL DEFAULT '' CHECK (path !~ '(^|/)\.\.(/|$)' AND left(path, 1) <> '/'),
  name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 255 AND name !~ '(^|/)\.\.(/|$)' AND position('/' IN name) = 0),
  content_type text NOT NULL DEFAULT 'application/octet-stream',
  size bigint NOT NULL CHECK (size >= 0),
  object_key text NOT NULL UNIQUE,
  created_by uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  updated_by uuid REFERENCES users (id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (id, project_id),
  UNIQUE (project_id, path, name),
  FOREIGN KEY (project_id, tenant_id) REFERENCES projects (id, tenant_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS project_files_project_path_idx ON project_files (project_id, path, updated_at DESC);
CREATE INDEX IF NOT EXISTS project_files_project_name_idx ON project_files (project_id, lower(name));`)
	if err != nil {
		return fmt.Errorf("初始化工程对象库表失败: %w", err)
	}
	return nil
}

func (r *Repository) List(ctx context.Context, tenantID, projectID string, filter ListFilter) ([]record, int64, error) {
	pathValue := normalizePath(filter.Path)
	keyword := strings.TrimSpace(filter.Keyword)
	offset := (filter.Page - 1) * filter.Limit
	args := []any{tenantID, projectID, pathValue, keyword, filter.Limit, offset}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, project_id, path, name, content_type, size, object_key, created_by, created_at, updated_at
		FROM project_files
		WHERE tenant_id=$1 AND project_id=$2 AND path=$3 AND ($4='' OR name ILIKE '%' || $4 || '%')
		ORDER BY updated_at DESC, name ASC LIMIT $5 OFFSET $6`, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询工程对象文件失败: %w", err)
	}
	defer rows.Close()
	items := make([]record, 0)
	for rows.Next() {
		var item record
		if err := rows.Scan(&item.ID, &item.TenantID, &item.ProjectID, &item.Path, &item.Name, &item.ContentType, &item.Size, &item.ObjectKey, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM project_files WHERE tenant_id=$1 AND project_id=$2 AND path=$3 AND ($4='' OR name ILIKE '%' || $4 || '%')`, tenantID, projectID, pathValue, keyword).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *Repository) Get(ctx context.Context, tenantID, projectID, id string) (record, error) {
	var item record
	err := r.pool.QueryRow(ctx, `SELECT id, tenant_id, project_id, path, name, content_type, size, object_key, created_by, created_at, updated_at FROM project_files WHERE id=$1 AND tenant_id=$2 AND project_id=$3`, id, tenantID, projectID).
		Scan(&item.ID, &item.TenantID, &item.ProjectID, &item.Path, &item.Name, &item.ContentType, &item.Size, &item.ObjectKey, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return record{}, ErrNotFound
	}
	if err != nil {
		return record{}, err
	}
	return item, nil
}

func (r *Repository) Create(ctx context.Context, item record) (record, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO project_files (id, tenant_id, project_id, path, name, content_type, size, object_key, created_by, updated_by)
		VALUES (gen_random_uuid(), $1,$2,$3,$4,$5,$6,$7,$8,$8)
		RETURNING id, tenant_id, project_id, path, name, content_type, size, object_key, created_by, created_at, updated_at`,
		item.TenantID, item.ProjectID, item.Path, item.Name, item.ContentType, item.Size, item.ObjectKey, item.CreatedBy).
		Scan(&item.ID, &item.TenantID, &item.ProjectID, &item.Path, &item.Name, &item.ContentType, &item.Size, &item.ObjectKey, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return record{}, mapError(err)
	}
	return item, nil
}

func (r *Repository) Update(ctx context.Context, tenantID, projectID, id, pathValue, name, userID string) (record, error) {
	var item record
	err := r.pool.QueryRow(ctx, `
		UPDATE project_files SET path=$1, name=$2, updated_by=$3, updated_at=now()
		WHERE id=$4 AND tenant_id=$5 AND project_id=$6
		RETURNING id, tenant_id, project_id, path, name, content_type, size, object_key, created_by, created_at, updated_at`,
		normalizePath(pathValue), name, userID, id, tenantID, projectID).
		Scan(&item.ID, &item.TenantID, &item.ProjectID, &item.Path, &item.Name, &item.ContentType, &item.Size, &item.ObjectKey, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return record{}, ErrNotFound
	}
	if err != nil {
		return record{}, mapError(err)
	}
	return item, nil
}

func (r *Repository) Replace(ctx context.Context, tenantID, projectID, id, pathValue, name string, size int64, contentType, objectKey, userID string) (record, string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return record{}, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var oldKey string
	if err := tx.QueryRow(ctx, `SELECT object_key FROM project_files WHERE id=$1 AND tenant_id=$2 AND project_id=$3 FOR UPDATE`, id, tenantID, projectID).Scan(&oldKey); errors.Is(err, pgx.ErrNoRows) {
		return record{}, "", ErrNotFound
	} else if err != nil {
		return record{}, "", err
	}
	var item record
	err = tx.QueryRow(ctx, `UPDATE project_files SET path=$1,name=$2,size=$3,content_type=$4,object_key=$5,updated_by=$6,updated_at=now() WHERE id=$7 AND tenant_id=$8 AND project_id=$9 RETURNING id,tenant_id,project_id,path,name,content_type,size,object_key,created_by,created_at,updated_at`, normalizePath(pathValue), name, size, contentType, objectKey, userID, id, tenantID, projectID).
		Scan(&item.ID, &item.TenantID, &item.ProjectID, &item.Path, &item.Name, &item.ContentType, &item.Size, &item.ObjectKey, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return record{}, "", mapError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return record{}, "", err
	}
	return item, oldKey, nil
}

func (r *Repository) Delete(ctx context.Context, tenantID, projectID, id string) (string, error) {
	var key string
	err := r.pool.QueryRow(ctx, `DELETE FROM project_files WHERE id=$1 AND tenant_id=$2 AND project_id=$3 RETURNING object_key`, id, tenantID, projectID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return key, err
}

func mapError(err error) error {
	var constraint *pgconn.PgError
	if errors.As(err, &constraint) && constraint.Code == "23505" {
		return ErrConflict
	}
	return err
}

func normalizePath(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "/")
	if value == "." {
		return ""
	}
	return value
}
