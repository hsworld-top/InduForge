package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
	LastValue       []byte
	Quality         string
	LastUpdatedAt   *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// OpcuaNodeListFilter 描述变量列表的服务端筛选、搜索和排序条件。
type OpcuaNodeListFilter struct {
	GroupID     *string
	Search      string
	QuickFilter string
	NodeID      string
	BrowseName  string
	AccessLevel string
	DataType    string
	SortBy      string
	SortOrder   string
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

// UpdateOpcuaNodeLastValueParams 描述开发态读取后写回的最近值快照。
type UpdateOpcuaNodeLastValueParams struct {
	ProjectID    string
	ConnectionID string
	NodeID       string
	LastValue    any
	Quality      string
	UserID       string
}

// BatchCreateOpcuaNodesParams 描述事务化批量创建 OPC UA 变量的参数。
type BatchCreateOpcuaNodesParams struct {
	ProjectID    string
	ConnectionID string
	GroupID      *string
	UserID       string
	Nodes        []CreateOpcuaNodeParams
}

// BuildOpcuaDataPointParamsFunc 根据已插入变量构造数据点同步参数。
type BuildOpcuaDataPointParamsFunc func(context.Context, OpcuaNodeRecord, string) (CreateDataPointParams, error)

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
		SET group_id = NULL, updated_by = $4, updated_at = now()
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
		       n.last_value, n.quality, n.last_updated_at,
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

// ListNodesPage 返回当前条件下的一页 OPC UA 变量，并同时返回匹配总数。
func (r *OpcuaModelingRepository) ListNodesPage(ctx context.Context, projectID, connectionID string, filter OpcuaNodeListFilter, page, pageSize int) ([]OpcuaNodeRecord, int, error) {
	where, args := buildOpcuaNodeListWhere(projectID, connectionID, filter)

	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM data_opcua_nodes n
		LEFT JOIN data_points dp
		  ON dp.project_id = n.project_id
		 AND dp.source_type = 'opcua.node'
		 AND dp.source_id = n.id
		WHERE `+where, args...).Scan(&total); err != nil {
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
		       n.last_value, n.quality, n.last_updated_at,
		       n.created_at, n.updated_at
		FROM data_opcua_nodes n
		LEFT JOIN data_points dp
		  ON dp.project_id = n.project_id
		 AND dp.source_type = 'opcua.node'
		 AND dp.source_id = n.id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, where, opcuaNodeOrderSQL(filter.SortBy, filter.SortOrder), limitIndex, offsetIndex), queryArgs...)
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

func buildOpcuaNodeListWhere(projectID, connectionID string, filter OpcuaNodeListFilter) (string, []any) {
	clauses := []string{"n.project_id = $1", "n.connection_id = $2"}
	args := []any{projectID, connectionID}
	addEqual := func(column, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	if filter.GroupID != nil {
		groupID := strings.TrimSpace(*filter.GroupID)
		if groupID == "__ungrouped__" {
			clauses = append(clauses, "n.group_id IS NULL")
		} else if groupID != "" {
			args = append(args, groupID)
			clauses = append(clauses, fmt.Sprintf("n.group_id = $%d", len(args)))
		}
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+search+"%")
		idx := len(args)
		clauses = append(clauses, fmt.Sprintf("(n.name ILIKE $%d OR n.code ILIKE $%d OR n.node_id ILIKE $%d OR COALESCE(n.browse_name, '') ILIKE $%d OR COALESCE(n.display_name, '') ILIKE $%d OR COALESCE(dp.path, '') ILIKE $%d)", idx, idx, idx, idx, idx, idx))
	}
	if nodeID := strings.TrimSpace(filter.NodeID); nodeID != "" {
		args = append(args, "%"+nodeID+"%")
		clauses = append(clauses, fmt.Sprintf("n.node_id ILIKE $%d", len(args)))
	}
	if browseName := strings.TrimSpace(filter.BrowseName); browseName != "" {
		args = append(args, "%"+browseName+"%")
		clauses = append(clauses, fmt.Sprintf("COALESCE(n.browse_name, '') ILIKE $%d", len(args)))
	}
	addEqual("n.access_level", filter.AccessLevel)
	addEqual("n.data_type", filter.DataType)
	switch strings.TrimSpace(filter.QuickFilter) {
	case "issue":
		clauses = append(clauses, "(n.status <> 'active' OR dp.id IS NULL OR COALESCE(dp.status, '') <> 'active')")
	case "datapoint":
		clauses = append(clauses, "(dp.id IS NULL OR COALESCE(dp.status, '') <> 'active')")
	case "writable":
		clauses = append(clauses, "LOWER(n.access_level) LIKE '%write%'")
	case "disabled":
		clauses = append(clauses, "n.status <> 'active'")
	}
	return strings.Join(clauses, " AND "), args
}

func opcuaNodeOrderSQL(sortBy, sortOrder string) string {
	direction := "ASC"
	if strings.EqualFold(strings.TrimSpace(sortOrder), "desc") || strings.EqualFold(strings.TrimSpace(sortOrder), "descending") {
		direction = "DESC"
	}
	columns := map[string]string{
		"name":        "n.name",
		"code":        "n.code",
		"nodeId":      "n.node_id",
		"dataType":    "n.data_type",
		"samplingMs":  "n.sampling_ms",
		"accessLevel": "n.access_level",
		"status":      "n.status",
		"updatedAt":   "n.updated_at",
		"sortOrder":   "n.sort_order",
	}
	if column, ok := columns[strings.TrimSpace(sortBy)]; ok {
		return column + " " + direction + ", n.sort_order ASC, n.created_at ASC"
	}
	return "n.sort_order ASC, n.created_at ASC"
}

// ListNodeIdentityMap 返回当前连接下已占用的 NodeId 和 Code，用于批量导入前整体预检。
func (r *OpcuaModelingRepository) ListNodeIdentityMap(ctx context.Context, projectID, connectionID string) (map[string]string, map[string]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, node_id, code
		FROM data_opcua_nodes
		WHERE project_id = $1 AND connection_id = $2
	`, projectID, connectionID)
	if err != nil {
		return nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 OPC UA 变量标识失败", err)
	}
	defer rows.Close()

	nodeIDs := map[string]string{}
	codes := map[string]string{}
	for rows.Next() {
		var id, nodeID, code string
		if err := rows.Scan(&id, &nodeID, &code); err != nil {
			return nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 OPC UA 变量标识失败", err)
		}
		nodeIDs[nodeID] = id
		codes[code] = id
	}
	if err := rows.Err(); err != nil {
		return nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 OPC UA 变量标识失败", err)
	}
	return nodeIDs, codes, nil
}

// GetNode 按项目和变量 ID 读取 OPC UA 变量。
func (r *OpcuaModelingRepository) GetNode(ctx context.Context, projectID, nodeID string) (*OpcuaNodeRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT n.id, n.project_id, n.connection_id, n.group_id, n.name, n.code, n.node_id,
		       n.browse_name, n.display_name, n.data_type, n.unit, n.sampling_ms, n.deadband,
		       n.access_level, n.description, n.sort_order, n.status,
		       dp.id, dp.path, dp.status,
		       n.last_value, n.quality, n.last_updated_at,
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

// BatchCreateNodesWithDataPoints 在同一事务内创建变量并同步数据点，避免批量导入半成功。
func (r *OpcuaModelingRepository) BatchCreateNodesWithDataPoints(ctx context.Context, params BatchCreateOpcuaNodesParams, buildDataPoint BuildOpcuaDataPointParamsFunc) ([]OpcuaNodeRecord, error) {
	if len(params.Nodes) == 0 {
		return []OpcuaNodeRecord{}, nil
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 OPC UA 批量导入事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)

	createdIDs := make([]string, 0, len(params.Nodes))
	for _, node := range params.Nodes {
		record, err := createOpcuaNodeTx(ctx, tx, node)
		if err != nil {
			return nil, err
		}
		if buildDataPoint != nil {
			dataPointParams, err := buildDataPoint(ctx, *record, params.UserID)
			if err != nil {
				return nil, err
			}
			if err := upsertDataPointBySourceTx(ctx, tx, dataPointParams); err != nil {
				return nil, err
			}
		}
		createdIDs = append(createdIDs, record.ID)
	}

	records, err := listOpcuaNodesByIDsTx(ctx, tx, params.ProjectID, createdIDs)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 OPC UA 批量导入事务失败", err)
	}
	return records, nil
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
		          sort_order, status, NULL::uuid, NULL::text, NULL::text,
		          last_value, quality, last_updated_at, created_at, updated_at
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
		          sort_order, status, NULL::uuid, NULL::text, NULL::text,
		          last_value, quality, last_updated_at, created_at, updated_at
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

// UpdateNodeLastValue 写回开发态读取得到的最近值快照。
func (r *OpcuaModelingRepository) UpdateNodeLastValue(ctx context.Context, params UpdateOpcuaNodeLastValueParams) error {
	valueBytes, err := marshalOpcuaJSONValue(params.LastValue)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE data_opcua_nodes
		SET last_value = $4::jsonb,
		    quality = $5,
		    last_updated_at = now(),
		    updated_by = $6,
		    updated_at = now()
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
	`, params.ProjectID, params.ConnectionID, params.NodeID, string(valueBytes), params.Quality, params.UserID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新 OPC UA 变量最近值失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "OPC UA 变量不存在")
	}
	return nil
}

func createOpcuaNodeTx(ctx context.Context, tx pgx.Tx, params CreateOpcuaNodeParams) (*OpcuaNodeRecord, error) {
	row := tx.QueryRow(ctx, `
		INSERT INTO data_opcua_nodes (
			project_id, connection_id, group_id, name, code, node_id, browse_name, display_name,
			data_type, unit, sampling_ms, deadband, access_level, description, sort_order, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $16)
		RETURNING id, project_id, connection_id, group_id, name, code, node_id, browse_name,
		          display_name, data_type, unit, sampling_ms, deadband, access_level, description,
		          sort_order, status, NULL::uuid, NULL::text, NULL::text,
		          last_value, quality, last_updated_at, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.GroupID, params.Name, params.Code, params.NodeID, params.BrowseName, params.DisplayName, params.DataType, params.Unit, params.SamplingMS, params.Deadband, params.AccessLevel, params.Description, params.SortOrder, params.UserID)

	record, err := scanOpcuaNodeRecord(row)
	if err != nil {
		return nil, translateOpcuaModelingWriteError(err)
	}
	return &record, nil
}

func upsertDataPointBySourceTx(ctx context.Context, tx pgx.Tx, params CreateDataPointParams) error {
	if params.SourceID == nil || *params.SourceID == "" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "sourceId 不能为空")
	}
	sourceConfigBytes, err := marshalJSONObject(params.SourceConfig)
	if err != nil {
		return err
	}
	tagsBytes, err := marshalJSONArray(params.Tags)
	if err != nil {
		return err
	}
	commandTag, err := tx.Exec(ctx, `
		UPDATE data_points
		SET path = $4,
		    name = $5,
		    description = $6,
		    source_config = $7::jsonb,
		    data_type = $8,
		    unit = $9,
		    precision_num = $10,
		    default_value = $11,
		    min_value = $12,
		    max_value = $13,
		    alarm_low = $14,
		    alarm_high = $15,
		    tags = $16::jsonb,
		    refresh_mode = $17,
		    refresh_interval_ms = $18,
		    status = $19,
		    display_order = COALESCE($21, display_order),
		    updated_by = COALESCE($20, updated_by),
		    updated_at = now()
		WHERE project_id = $1
		  AND source_type = $2
		  AND source_id = $3
	`, params.ProjectID, params.SourceType, *params.SourceID, params.Path, params.Name, params.Description, string(sourceConfigBytes), params.DataType, params.Unit, params.PrecisionNum, params.DefaultValue, params.MinValue, params.MaxValue, params.AlarmLow, params.AlarmHigh, string(tagsBytes), params.RefreshMode, params.RefreshIntervalMS, params.Status, params.UserID, params.DisplayOrder)
	if err != nil {
		return translateDataPointWriteError(err)
	}
	if commandTag.RowsAffected() > 0 {
		return nil
	}

	runtimePermissionsBytes, err := marshalDataPointRuntimePermissions(DefaultDataPointRuntimePermissions())
	if err != nil {
		return err
	}
	displayOrder := 0
	if params.DisplayOrder != nil {
		displayOrder = *params.DisplayOrder
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO data_points (
			project_id, path, name, description, source_type, source_id, source_config, data_type,
			unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags, runtime_permissions,
			refresh_mode, refresh_interval_ms, status, display_order, created_by, updated_by
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11, $12, $13, $14, $15,
			$16::jsonb, $17::jsonb, $18, $19, $20, $21, $22, $22
		)
	`, params.ProjectID, params.Path, params.Name, params.Description, params.SourceType, params.SourceID, string(sourceConfigBytes), params.DataType, params.Unit, params.PrecisionNum, params.DefaultValue, params.MinValue, params.MaxValue, params.AlarmLow, params.AlarmHigh, string(tagsBytes), string(runtimePermissionsBytes), params.RefreshMode, params.RefreshIntervalMS, params.Status, displayOrder, params.UserID)
	if err != nil {
		return translateDataPointWriteError(err)
	}
	return nil
}

func listOpcuaNodesByIDsTx(ctx context.Context, tx pgx.Tx, projectID string, ids []string) ([]OpcuaNodeRecord, error) {
	if len(ids) == 0 {
		return []OpcuaNodeRecord{}, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT n.id, n.project_id, n.connection_id, n.group_id, n.name, n.code, n.node_id,
		       n.browse_name, n.display_name, n.data_type, n.unit, n.sampling_ms, n.deadband,
		       n.access_level, n.description, n.sort_order, n.status,
		       dp.id, dp.path, dp.status,
		       n.last_value, n.quality, n.last_updated_at,
		       n.created_at, n.updated_at
		FROM data_opcua_nodes n
		LEFT JOIN data_points dp
		  ON dp.project_id = n.project_id
		 AND dp.source_type = 'opcua.node'
		 AND dp.source_id = n.id
		WHERE n.project_id = $1 AND n.id = ANY($2::uuid[])
		ORDER BY n.sort_order ASC, n.created_at ASC
	`, projectID, ids)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 OPC UA 批量导入结果失败", err)
	}
	defer rows.Close()

	result := make([]OpcuaNodeRecord, 0, len(ids))
	for rows.Next() {
		record, scanErr := scanOpcuaNodeRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 OPC UA 批量导入结果失败", err)
	}
	return result, nil
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
		&record.LastValue,
		&record.Quality,
		&record.LastUpdatedAt,
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

func marshalOpcuaJSONValue(value any) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "OPC UA 最近值 JSON 格式无效", err)
	}
	return payload, nil
}
