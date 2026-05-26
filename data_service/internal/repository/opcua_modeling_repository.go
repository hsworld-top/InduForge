package repository

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// OpcuaNodeGroupRecord 表示 OPC UA 变量组。
type OpcuaNodeGroupRecord struct {
	ID           string
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	Description  *string
	SortOrder    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// OpcuaNodeRecord 表示 OPC UA 变量定义。
type OpcuaNodeRecord struct {
	ID              string
	ProjectID       string
	ConnectionID    string
	GroupID         *string
	Name            string
	Code            string
	NodeID          string
	BrowseName      *string
	DisplayName     *string
	DataType        string
	Unit            *string
	SamplingMS      int
	Deadband        *float64
	AccessLevel     string
	Description     *string
	SortOrder       int
	Status          string
	DataPointID     *string
	DataPointPath   *string
	DataPointStatus *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CreateOpcuaNodeGroupParams 描述创建 OPC UA 变量组的仓储参数。
type CreateOpcuaNodeGroupParams struct {
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	Description  *string
	SortOrder    int
	UserID       string
}

// UpdateOpcuaNodeGroupParams 描述更新 OPC UA 变量组的仓储参数。
type UpdateOpcuaNodeGroupParams struct {
	ID           string
	ProjectID    string
	ConnectionID string
	ParentID     *string
	HasParentID  bool
	Name         string
	Description  *string
	SortOrder    int
	UserID       string
}

// CreateOpcuaNodeParams 描述创建 OPC UA 变量的仓储参数。
type CreateOpcuaNodeParams struct {
	ProjectID    string
	ConnectionID string
	GroupID      *string
	Name         string
	Code         string
	NodeID       string
	BrowseName   *string
	DisplayName  *string
	DataType     string
	Unit         *string
	SamplingMS   int
	Deadband     *float64
	AccessLevel  string
	Description  *string
	SortOrder    int
	UserID       string
}

// UpdateOpcuaNodeParams 描述更新 OPC UA 变量的仓储参数。
type UpdateOpcuaNodeParams struct {
	ID           string
	ProjectID    string
	ConnectionID string
	GroupID      *string
	HasGroupID   bool
	Name         string
	Code         string
	NodeID       string
	BrowseName   *string
	DisplayName  *string
	DataType     string
	Unit         *string
	SamplingMS   int
	Deadband     *float64
	AccessLevel  string
	Description  *string
	SortOrder    int
	Status       string
	UserID       string
}

// OpcuaModelingRepository 封装 OPC UA 点位建模的参数化 SQL。
type OpcuaModelingRepository struct {
	pool *pgxpool.Pool
}

// NewOpcuaModelingRepository 创建 OPC UA 点位建模仓储。
func NewOpcuaModelingRepository(pool *pgxpool.Pool) *OpcuaModelingRepository {
	return &OpcuaModelingRepository{pool: pool}
}

// ListGroups 返回连接下的 OPC UA 变量组。
func (r *OpcuaModelingRepository) ListGroups(ctx context.Context, projectID, connectionID string) ([]OpcuaNodeGroupRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, connection_id, parent_id, name, description, sort_order, created_at, updated_at
		FROM data_opcua_node_groups
		WHERE project_id = $1 AND connection_id = $2
		ORDER BY sort_order ASC, created_at ASC
	`, projectID, connectionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 OPC UA 变量组失败", err)
	}
	defer rows.Close()

	result := make([]OpcuaNodeGroupRecord, 0)
	for rows.Next() {
		record, scanErr := scanOpcuaNodeGroupRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 OPC UA 变量组失败", err)
	}
	return result, nil
}

// CreateGroup 创建 OPC UA 变量组。
func (r *OpcuaModelingRepository) CreateGroup(ctx context.Context, params CreateOpcuaNodeGroupParams) (*OpcuaNodeGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO data_opcua_node_groups (
			project_id, connection_id, parent_id, name, description, sort_order, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id, project_id, connection_id, parent_id, name, description, sort_order, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.ParentID, params.Name, params.Description, params.SortOrder, params.UserID)

	record, err := scanOpcuaNodeGroupRecord(row)
	if err != nil {
		return nil, translateOpcuaModelingWriteError(err)
	}
	return &record, nil
}

// UpdateGroup 更新 OPC UA 变量组。
func (r *OpcuaModelingRepository) UpdateGroup(ctx context.Context, params UpdateOpcuaNodeGroupParams) (*OpcuaNodeGroupRecord, error) {
	parentSQL := "parent_id"
	args := []any{params.ProjectID, params.ConnectionID, params.ID, params.Name, params.Description, params.SortOrder, params.UserID}
	if params.HasParentID {
		parentSQL = "$8"
		args = append(args, params.ParentID)
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE data_opcua_node_groups
		SET name = $4,
			description = $5,
			sort_order = $6,
			updated_by = $7,
			updated_at = now(),
			parent_id = `+parentSQL+`
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
		RETURNING id, project_id, connection_id, parent_id, name, description, sort_order, created_at, updated_at
	`, args...)

	record, err := scanOpcuaNodeGroupRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "OPC UA 变量组不存在")
		}
		return nil, translateOpcuaModelingWriteError(err)
	}
	return &record, nil
}

// DeleteGroup 删除 OPC UA 变量组，组内变量移动到未分组。
func (r *OpcuaModelingRepository) DeleteGroup(ctx context.Context, projectID, connectionID, groupID, userID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 OPC UA 变量组删除事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)

	if _, err := tx.Exec(ctx, `
		UPDATE data_opcua_nodes
		SET group_id = NULL, updated_by = $3, updated_at = now()
		WHERE project_id = $1 AND connection_id = $2 AND group_id = $3
	`, projectID, connectionID, groupID, userID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "移动 OPC UA 变量失败", err)
	}

	tag, err := tx.Exec(ctx, `
		DELETE FROM data_opcua_node_groups
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
	`, projectID, connectionID, groupID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 OPC UA 变量组失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "OPC UA 变量组不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 OPC UA 变量组删除事务失败", err)
	}
	return nil
}

// ListNodes 返回连接下的 OPC UA 变量，可按 groupID 过滤。
func (r *OpcuaModelingRepository) ListNodes(ctx context.Context, projectID, connectionID string, groupID *string) ([]OpcuaNodeRecord, error) {
	where := "n.project_id = $1 AND n.connection_id = $2"
	args := []any{projectID, connectionID}
	if groupID != nil {
		where += " AND n.group_id = $3"
		args = append(args, groupID)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.project_id, n.connection_id, n.group_id, n.name, n.code, n.node_id,
		       n.browse_name, n.display_name, n.data_type, n.unit, n.sampling_ms, n.deadband,
		       n.access_level, n.description, n.sort_order, n.status,
		       dp.id, dp.path, dp.status,
		       n.created_at, n.updated_at
		FROM data_opcua_nodes n
		LEFT JOIN data_points dp
		  ON dp.project_id = n.project_id
		 AND dp.source_type = 'opcua.node'
		 AND dp.source_id = n.id
		WHERE `+where+`
		ORDER BY n.sort_order ASC, n.created_at ASC
	`, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 OPC UA 变量失败", err)
	}
	defer rows.Close()

	result := make([]OpcuaNodeRecord, 0)
	for rows.Next() {
		record, scanErr := scanOpcuaNodeRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 OPC UA 变量失败", err)
	}
	return result, nil
}

// ListNodesPage 返回当前分组下的一页 OPC UA 变量，并同时返回匹配总数。
func (r *OpcuaModelingRepository) ListNodesPage(ctx context.Context, projectID, connectionID string, groupID *string, page, pageSize int) ([]OpcuaNodeRecord, int, error) {
	where := "n.project_id = $1 AND n.connection_id = $2"
	args := []any{projectID, connectionID}
	if groupID != nil {
		where += " AND n.group_id = $3"
		args = append(args, groupID)
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM data_opcua_nodes n WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计 OPC UA 变量失败", err)
	}

	limitIndex := len(args) + 1
	offsetIndex := len(args) + 2
	queryArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT n.id, n.project_id, n.connection_id, n.group_id, n.name, n.code, n.node_id,
		       n.browse_name, n.display_name, n.data_type, n.unit, n.sampling_ms, n.deadband,
		       n.access_level, n.description, n.sort_order, n.status,
		       dp.id, dp.path, dp.status,
		       n.created_at, n.updated_at
		FROM data_opcua_nodes n
		LEFT JOIN data_points dp
		  ON dp.project_id = n.project_id
		 AND dp.source_type = 'opcua.node'
		 AND dp.source_id = n.id
		WHERE %s
		ORDER BY n.sort_order ASC, n.created_at ASC
		LIMIT $%d OFFSET $%d
	`, where, limitIndex, offsetIndex), queryArgs...)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 OPC UA 变量分页失败", err)
	}
	defer rows.Close()

	result := make([]OpcuaNodeRecord, 0)
	for rows.Next() {
		record, scanErr := scanOpcuaNodeRecord(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 OPC UA 变量分页失败", err)
	}
	return result, total, nil
}

// GetNode 按项目和变量 ID 读取 OPC UA 变量。
func (r *OpcuaModelingRepository) GetNode(ctx context.Context, projectID, nodeID string) (*OpcuaNodeRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT n.id, n.project_id, n.connection_id, n.group_id, n.name, n.code, n.node_id,
		       n.browse_name, n.display_name, n.data_type, n.unit, n.sampling_ms, n.deadband,
		       n.access_level, n.description, n.sort_order, n.status,
		       dp.id, dp.path, dp.status,
		       n.created_at, n.updated_at
		FROM data_opcua_nodes n
		LEFT JOIN data_points dp
		  ON dp.project_id = n.project_id
		 AND dp.source_type = 'opcua.node'
		 AND dp.source_id = n.id
		WHERE n.project_id = $1 AND n.id = $2
	`, projectID, nodeID)

	record, err := scanOpcuaNodeRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "OPC UA 变量不存在")
		}
		return nil, err
	}
	return &record, nil
}

// CreateNode 创建 OPC UA 变量。
func (r *OpcuaModelingRepository) CreateNode(ctx context.Context, params CreateOpcuaNodeParams) (*OpcuaNodeRecord, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO data_opcua_nodes (
			project_id, connection_id, group_id, name, code, node_id, browse_name, display_name,
			data_type, unit, sampling_ms, deadband, access_level, description, sort_order, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $16)
		RETURNING id, project_id, connection_id, group_id, name, code, node_id, browse_name,
		          display_name, data_type, unit, sampling_ms, deadband, access_level, description,
		          sort_order, status, NULL::uuid, NULL::text, NULL::text, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.GroupID, params.Name, params.Code, params.NodeID, params.BrowseName, params.DisplayName, params.DataType, params.Unit, params.SamplingMS, params.Deadband, params.AccessLevel, params.Description, params.SortOrder, params.UserID)

	record, err := scanOpcuaNodeRecord(row)
	if err != nil {
		return nil, translateOpcuaModelingWriteError(err)
	}
	return &record, nil
}

// UpdateNode 更新 OPC UA 变量。
func (r *OpcuaModelingRepository) UpdateNode(ctx context.Context, params UpdateOpcuaNodeParams) (*OpcuaNodeRecord, error) {
	groupSQL := "group_id"
	args := []any{
		params.ProjectID, params.ConnectionID, params.ID, params.Name, params.Code, params.NodeID, params.BrowseName,
		params.DisplayName, params.DataType, params.Unit, params.SamplingMS, params.Deadband,
		params.AccessLevel, params.Description, params.SortOrder, params.Status, params.UserID,
	}
	if params.HasGroupID {
		groupSQL = "$18"
		args = append(args, params.GroupID)
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE data_opcua_nodes
		SET name = $4,
			code = $5,
			node_id = $6,
			browse_name = $7,
			display_name = $8,
			data_type = $9,
			unit = $10,
			sampling_ms = $11,
			deadband = $12,
			access_level = $13,
			description = $14,
			sort_order = $15,
			status = $16,
			updated_by = $17,
			updated_at = now(),
			group_id = `+groupSQL+`
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
		RETURNING id, project_id, connection_id, group_id, name, code, node_id, browse_name,
		          display_name, data_type, unit, sampling_ms, deadband, access_level, description,
		          sort_order, status, NULL::uuid, NULL::text, NULL::text, created_at, updated_at
	`, args...)

	record, err := scanOpcuaNodeRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "OPC UA 变量不存在")
		}
		return nil, translateOpcuaModelingWriteError(err)
	}
	return &record, nil
}

// DeleteNode 删除 OPC UA 变量定义。
func (r *OpcuaModelingRepository) DeleteNode(ctx context.Context, projectID, connectionID, nodeID string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM data_opcua_nodes
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
	`, projectID, connectionID, nodeID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 OPC UA 变量失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "OPC UA 变量不存在")
	}
	return nil
}

func scanOpcuaNodeGroupRecord(row pgx.Row) (OpcuaNodeGroupRecord, error) {
	record := OpcuaNodeGroupRecord{}
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.ParentID,
		&record.Name,
		&record.Description,
		&record.SortOrder,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return record, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "OPC UA 变量组不存在")
		}
		return record, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 OPC UA 变量组失败", err)
	}
	return record, nil
}

func scanOpcuaNodeRecord(row pgx.Row) (OpcuaNodeRecord, error) {
	record := OpcuaNodeRecord{}
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.GroupID,
		&record.Name,
		&record.Code,
		&record.NodeID,
		&record.BrowseName,
		&record.DisplayName,
		&record.DataType,
		&record.Unit,
		&record.SamplingMS,
		&record.Deadband,
		&record.AccessLevel,
		&record.Description,
		&record.SortOrder,
		&record.Status,
		&record.DataPointID,
		&record.DataPointPath,
		&record.DataPointStatus,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return record, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "OPC UA 变量不存在")
		}
		return record, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 OPC UA 变量失败", err)
	}
	return record, nil
}

func translateOpcuaModelingWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "OPC UA 建模对象已存在")
		case "23503":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "OPC UA 建模引用不存在")
		case "23514":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "OPC UA 建模参数不合法")
		}
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 OPC UA 建模数据失败", err)
}
