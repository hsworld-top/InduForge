package repository

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// WorkbenchObjectGroupRecord 是 SQL 工作台查询/表展示分组的持久化记录。
type WorkbenchObjectGroupRecord struct {
	ID           string
	ProjectID    string
	ConnectionID string
	Scope        string
	Name         string
	SortOrder    int
	CreatedBy    string
	UpdatedBy    *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TableGroupMemberRecord 记录真实表名在工作台中的展示分组。
type TableGroupMemberRecord struct {
	ProjectID    string
	ConnectionID string
	TableName    string
	GroupID      string
	UpdatedBy    *string
	UpdatedAt    time.Time
}

// WorkbenchGroupRepository 封装 SQL 工作台分组的参数化 SQL。
type WorkbenchGroupRepository struct {
	pool *pgxpool.Pool
}

func NewWorkbenchGroupRepository(pool *pgxpool.Pool) *WorkbenchGroupRepository {
	return &WorkbenchGroupRepository{pool: pool}
}

func (r *WorkbenchGroupRepository) ListGroups(ctx context.Context, projectID, connectionID, scope string) ([]WorkbenchObjectGroupRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, connection_id, scope, name, sort_order, created_by, updated_by, created_at, updated_at
        FROM data_workbench_object_groups
        WHERE project_id = $1 AND connection_id = $2 AND scope = $3
        ORDER BY sort_order ASC, created_at ASC
    `, projectID, connectionID, scope)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询工作台分组失败", err)
	}
	defer rows.Close()

	records := make([]WorkbenchObjectGroupRecord, 0)
	for rows.Next() {
		record, scanErr := scanWorkbenchGroup(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历工作台分组失败", err)
	}
	return records, nil
}

func (r *WorkbenchGroupRepository) GetGroup(ctx context.Context, projectID, groupID string) (*WorkbenchObjectGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, project_id, connection_id, scope, name, sort_order, created_by, updated_by, created_at, updated_at
        FROM data_workbench_object_groups
        WHERE project_id = $1 AND id = $2
    `, projectID, groupID)
	record, err := scanWorkbenchGroup(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *WorkbenchGroupRepository) CreateGroup(ctx context.Context, params CreateWorkbenchGroupParams) (*WorkbenchObjectGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
        INSERT INTO data_workbench_object_groups (
            project_id, connection_id, scope, name, sort_order, created_by, updated_by
        )
        VALUES ($1, $2, $3, $4, $5, $6, $6)
        RETURNING id, project_id, connection_id, scope, name, sort_order, created_by, updated_by, created_at, updated_at
    `, params.ProjectID, params.ConnectionID, params.Scope, params.Name, params.SortOrder, params.UserID)
	record, err := scanWorkbenchGroup(row)
	if err != nil {
		return nil, translateWorkbenchGroupWriteError(err)
	}
	return &record, nil
}

func (r *WorkbenchGroupRepository) UpdateGroup(ctx context.Context, params UpdateWorkbenchGroupParams) (*WorkbenchObjectGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
        UPDATE data_workbench_object_groups
        SET name = $3,
            updated_by = $4,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, connection_id, scope, name, sort_order, created_by, updated_by, created_at, updated_at
    `, params.ProjectID, params.ID, params.Name, params.UserID)
	record, err := scanWorkbenchGroup(row)
	if err != nil {
		return nil, translateWorkbenchGroupWriteError(err)
	}
	return &record, nil
}

func (r *WorkbenchGroupRepository) DeleteGroup(ctx context.Context, projectID, groupID string) error {
	commandTag, err := r.pool.Exec(ctx, `
        DELETE FROM data_workbench_object_groups
        WHERE project_id = $1 AND id = $2
    `, projectID, groupID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除工作台分组失败", err)
	}
	if commandTag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "工作台分组不存在")
	}
	return nil
}

func (r *WorkbenchGroupRepository) MoveQuery(ctx context.Context, projectID, queryID string, groupID *string, userID string) error {
	commandTag, err := r.pool.Exec(ctx, `
        UPDATE data_queries
        SET group_id = $3,
            updated_by = $4,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
    `, projectID, queryID, groupID, userID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "移动查询分组失败", err)
	}
	if commandTag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "查询不存在")
	}
	return nil
}

func (r *WorkbenchGroupRepository) ListTableMembers(ctx context.Context, projectID, connectionID string) ([]TableGroupMemberRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT project_id, connection_id, table_name, group_id, updated_by, updated_at
        FROM data_table_group_members
        WHERE project_id = $1 AND connection_id = $2
    `, projectID, connectionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询表分组映射失败", err)
	}
	defer rows.Close()

	records := make([]TableGroupMemberRecord, 0)
	for rows.Next() {
		record, scanErr := scanTableGroupMember(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历表分组映射失败", err)
	}
	return records, nil
}

func (r *WorkbenchGroupRepository) MoveTable(ctx context.Context, params MoveTableGroupParams) error {
	if params.GroupID == nil {
		_, err := r.pool.Exec(ctx, `
            DELETE FROM data_table_group_members
            WHERE project_id = $1 AND connection_id = $2 AND table_name = $3
        `, params.ProjectID, params.ConnectionID, params.TableName)
		if err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "移除表分组映射失败", err)
		}
		return nil
	}

	_, err := r.pool.Exec(ctx, `
        INSERT INTO data_table_group_members (
            project_id, connection_id, table_name, group_id, updated_by, updated_at
        )
        VALUES ($1, $2, $3, $4, $5, now())
        ON CONFLICT (project_id, connection_id, table_name)
        DO UPDATE SET group_id = EXCLUDED.group_id,
                      updated_by = EXCLUDED.updated_by,
                      updated_at = now()
    `, params.ProjectID, params.ConnectionID, params.TableName, *params.GroupID, params.UserID)
	if err != nil {
		return translateWorkbenchGroupWriteError(err)
	}
	return nil
}

func (r *WorkbenchGroupRepository) RenameTableMember(ctx context.Context, projectID, connectionID, oldName, newName, userID string) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE data_table_group_members
        SET table_name = $4,
            updated_by = $5,
            updated_at = now()
        WHERE project_id = $1 AND connection_id = $2 AND table_name = $3
    `, projectID, connectionID, oldName, newName, userID)
	if err != nil {
		return translateWorkbenchGroupWriteError(err)
	}
	return nil
}

func (r *WorkbenchGroupRepository) DeleteTableMember(ctx context.Context, projectID, connectionID, tableName string) error {
	_, err := r.pool.Exec(ctx, `
        DELETE FROM data_table_group_members
        WHERE project_id = $1 AND connection_id = $2 AND table_name = $3
    `, projectID, connectionID, tableName)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除表分组映射失败", err)
	}
	return nil
}

type CreateWorkbenchGroupParams struct {
	ProjectID    string
	ConnectionID string
	Scope        string
	Name         string
	SortOrder    int
	UserID       string
}

type UpdateWorkbenchGroupParams struct {
	ProjectID string
	ID        string
	Name      string
	UserID    string
}

type MoveTableGroupParams struct {
	ProjectID    string
	ConnectionID string
	TableName    string
	GroupID      *string
	UserID       string
}

type workbenchGroupScannable interface {
	Scan(dest ...any) error
}

func scanWorkbenchGroup(row workbenchGroupScannable) (WorkbenchObjectGroupRecord, error) {
	var (
		record    WorkbenchObjectGroupRecord
		updatedBy sql.NullString
	)
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.Scope,
		&record.Name,
		&record.SortOrder,
		&record.CreatedBy,
		&updatedBy,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WorkbenchObjectGroupRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "工作台分组不存在")
		}
		return WorkbenchObjectGroupRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取工作台分组失败", err)
	}
	record.UpdatedBy = nullStringToPtr(updatedBy)
	return record, nil
}

func scanTableGroupMember(row workbenchGroupScannable) (TableGroupMemberRecord, error) {
	var (
		record    TableGroupMemberRecord
		updatedBy sql.NullString
	)
	if err := row.Scan(
		&record.ProjectID,
		&record.ConnectionID,
		&record.TableName,
		&record.GroupID,
		&updatedBy,
		&record.UpdatedAt,
	); err != nil {
		return TableGroupMemberRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取表分组映射失败", err)
	}
	record.UpdatedBy = nullStringToPtr(updatedBy)
	return record, nil
}

func translateWorkbenchGroupWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "同名分组已存在")
		case "23503":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "分组引用不存在")
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "工作台分组不存在")
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "保存工作台分组失败", err)
}
