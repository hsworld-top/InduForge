package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/jackc/pgx/v5"
)

type CollectorPointGroupRecord struct {
	ID, ProjectID, ConnectionID string
	ParentID                    *string
	Name                        string
	SortOrder                   int
	Metadata                    map[string]any
	CreatedAt, UpdatedAt        time.Time
}

type CreateCollectorPointGroupParams struct {
	ID, ProjectID, ConnectionID, Name string
	ParentID                          *string
	SortOrder                         int
	Metadata                          map[string]any
}

type UpdateCollectorPointGroupParams struct {
	ID, ProjectID, ConnectionID, Name string
}

type CollectorPointRecord struct {
	ID, ProjectID, ConnectionID, Code, Name, AddressText, DataType string
	GroupID                                                        *string
	Description                                                    *string
	Address, ReadOptions, Acquisition, Metadata                    map[string]any
	AddressSchemaVersion, ElementCount, SortOrder                  int
	Enabled                                                        bool
	CreatedAt, UpdatedAt                                           time.Time
}

type CollectorPointListFilter struct {
	Page, PageSize                      int
	Search, DataType, SortBy, SortOrder string
	GroupID                             *string
	Enabled                             *bool
}

type CreateCollectorPointParams struct {
	ID, ProjectID, ConnectionID, ConnectionCode, UserID, Code, Name, AddressText, DataType string
	GroupID                                                                                *string
	Description                                                                            *string
	Address, ReadOptions, Acquisition, Metadata                                            map[string]any
	AddressSchemaVersion, ElementCount, SortOrder                                          int
	Enabled                                                                                bool
}

type UpdateCollectorPointParams = CreateCollectorPointParams

func (r *CollectorRepository) ListPointGroups(ctx context.Context, projectID, connectionID string, parentID *string) ([]CollectorPointGroupRecord, error) {
	query := `SELECT id, project_id, connection_id, parent_id, name, sort_order, metadata, created_at, updated_at FROM data_collector_point_groups WHERE project_id=$1 AND connection_id=$2`
	args := []any{projectID, connectionID}
	if parentID == nil {
		query += " AND parent_id IS NULL"
	} else {
		query += " AND parent_id=$3"
		args = append(args, *parentID)
	}
	query += " ORDER BY sort_order ASC, created_at ASC, id ASC"
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("查询采集点分组失败", err)
	}
	defer rows.Close()
	result := make([]CollectorPointGroupRecord, 0)
	for rows.Next() {
		record, err := scanCollectorPointGroup(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("遍历采集点分组失败", err)
	}
	return result, nil
}

func (r *CollectorRepository) CreatePointGroup(ctx context.Context, params CreateCollectorPointGroupParams) (*CollectorPointGroupRecord, error) {
	metadata, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, badCollectorPayload("序列化采集点分组元数据失败", err)
	}
	record, err := scanCollectorPointGroup(r.pool.QueryRow(ctx, `INSERT INTO data_collector_point_groups (id,project_id,connection_id,parent_id,name,sort_order,metadata) VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb) RETURNING id,project_id,connection_id,parent_id,name,sort_order,metadata,created_at,updated_at`, params.ID, params.ProjectID, params.ConnectionID, params.ParentID, params.Name, params.SortOrder, string(metadata)))
	if err != nil {
		return nil, translateCollectorWriteError("创建采集点分组失败", err)
	}
	return &record, nil
}

func (r *CollectorRepository) UpdatePointGroup(ctx context.Context, params UpdateCollectorPointGroupParams) (*CollectorPointGroupRecord, error) {
	record, err := scanCollectorPointGroup(r.pool.QueryRow(ctx, `UPDATE data_collector_point_groups SET name=$4,updated_at=now() WHERE id=$1 AND project_id=$2 AND connection_id=$3 RETURNING id,project_id,connection_id,parent_id,name,sort_order,metadata,created_at,updated_at`, params.ID, params.ProjectID, params.ConnectionID, params.Name))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "采集点分组不存在")
		}
		return nil, translateCollectorWriteError("更新采集点分组失败", err)
	}
	return &record, nil
}

// DeletePointGroup 将被删子树中的变量移动到根分组的上级，再级联删除整个分组子树。
func (r *CollectorRepository) DeletePointGroup(ctx context.Context, projectID, connectionID, groupID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return wrapUnifiedCollectorRepositoryError("开启删除采集点分组事务失败", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var parentID *string
	if err := tx.QueryRow(ctx, `SELECT parent_id FROM data_collector_point_groups WHERE id=$1 AND project_id=$2 AND connection_id=$3 FOR UPDATE`, groupID, projectID, connectionID).Scan(&parentID); err != nil {
		if err == pgx.ErrNoRows {
			return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "采集点分组不存在")
		}
		return wrapUnifiedCollectorRepositoryError("查询采集点分组失败", err)
	}

	if _, err := tx.Exec(ctx, `WITH RECURSIVE subtree AS (
		SELECT id FROM data_collector_point_groups WHERE id=$1 AND project_id=$2 AND connection_id=$3
		UNION ALL
		SELECT child.id FROM data_collector_point_groups child JOIN subtree parent ON child.parent_id=parent.id
		WHERE child.project_id=$2 AND child.connection_id=$3
	) UPDATE data_collector_points SET group_id=$4,updated_at=now() WHERE project_id=$2 AND connection_id=$3 AND group_id IN (SELECT id FROM subtree)`, groupID, projectID, connectionID, parentID); err != nil {
		return translateCollectorWriteError("移动被删分组变量失败", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM data_collector_point_groups WHERE id=$1 AND project_id=$2 AND connection_id=$3`, groupID, projectID, connectionID); err != nil {
		return translateCollectorWriteError("删除采集点分组失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return wrapUnifiedCollectorRepositoryError("提交删除采集点分组事务失败", err)
	}
	return nil
}

func (r *CollectorRepository) ListPointCodes(ctx context.Context, projectID, connectionID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT code FROM data_collector_points WHERE project_id=$1 AND connection_id=$2`, projectID, connectionID)
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("查询采集点编码失败", err)
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		result = append(result, code)
	}
	return result, rows.Err()
}

func (r *CollectorRepository) ListExistingPointAddressTexts(ctx context.Context, projectID, connectionID string, addressTexts []string) ([]string, error) {
	if len(addressTexts) == 0 {
		return []string{}, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT address_text FROM data_collector_points WHERE project_id=$1 AND connection_id=$2 AND address_text=ANY($3::text[])`, projectID, connectionID, addressTexts)
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("查询已存在采集点地址失败", err)
	}
	defer rows.Close()
	result := make([]string, 0, len(addressTexts))
	for rows.Next() {
		var addressText string
		if err := rows.Scan(&addressText); err != nil {
			return nil, err
		}
		result = append(result, addressText)
	}
	return result, rows.Err()
}

func (r *CollectorRepository) ListPoints(ctx context.Context, projectID, connectionID string, filter CollectorPointListFilter) ([]CollectorPointRecord, int, error) {
	conditions := []string{"project_id=$1", "connection_id=$2"}
	args := []any{projectID, connectionID}
	add := func(condition string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}
	if value := strings.TrimSpace(filter.Search); value != "" {
		add("(code ILIKE $%[1]d OR name ILIKE $%[1]d OR address_text ILIKE $%[1]d)", "%"+value+"%")
	}
	if filter.GroupID != nil {
		add("group_id=$%d", *filter.GroupID)
	}
	if filter.Enabled != nil {
		add("enabled=$%d", *filter.Enabled)
	}
	if value := strings.TrimSpace(filter.DataType); value != "" {
		add("data_type=$%d", value)
	}
	order := map[string]string{"code": "code", "name": "name", "dataType": "data_type", "createdAt": "created_at", "updatedAt": "updated_at"}[filter.SortBy]
	if order == "" {
		order = "sort_order"
	}
	direction := "ASC"
	if strings.EqualFold(filter.SortOrder, "desc") {
		direction = "DESC"
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	query := fmt.Sprintf(`SELECT id,project_id,connection_id,group_id,code,name,description,address,address_text,address_schema_version,data_type,element_count,read_options,acquisition,enabled,sort_order,metadata,created_at,updated_at,COUNT(*) OVER()::int FROM data_collector_points WHERE %s ORDER BY %s %s,id ASC LIMIT $%d OFFSET $%d`, strings.Join(conditions, " AND "), order, direction, len(args)-1, len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, wrapUnifiedCollectorRepositoryError("查询采集点失败", err)
	}
	defer rows.Close()
	result := make([]CollectorPointRecord, 0)
	total := 0
	for rows.Next() {
		record, count, err := scanCollectorPointWithTotal(rows)
		if err != nil {
			return nil, 0, err
		}
		total = count
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, wrapUnifiedCollectorRepositoryError("遍历采集点失败", err)
	}
	return result, total, nil
}

func (r *CollectorRepository) CreatePointsBatch(ctx context.Context, params []CreateCollectorPointParams) ([]CollectorPointRecord, error) {
	if len(params) == 0 {
		return []CollectorPointRecord{}, nil
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("开启批量创建采集点事务失败", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ids := make([]string, 0, len(params))
	for _, item := range params {
		if err := insertCollectorPointAndDataPoint(ctx, tx, item); err != nil {
			return nil, err
		}
		ids = append(ids, item.ID)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("提交批量创建采集点事务失败", err)
	}
	return r.listPointsByIDs(ctx, params[0].ProjectID, params[0].ConnectionID, ids)
}

func (r *CollectorRepository) UpdatePointsBatch(ctx context.Context, params []UpdateCollectorPointParams) ([]CollectorPointRecord, error) {
	if len(params) == 0 {
		return []CollectorPointRecord{}, nil
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("开启批量更新采集点事务失败", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ids := make([]string, 0, len(params))
	for _, item := range params {
		address, _ := json.Marshal(item.Address)
		readOptions, _ := json.Marshal(item.ReadOptions)
		acquisition, _ := json.Marshal(item.Acquisition)
		metadata, _ := json.Marshal(item.Metadata)
		tag, err := tx.Exec(ctx, `UPDATE data_collector_points SET group_id=$4,code=$5,name=$6,description=$7,address=$8::jsonb,address_text=$9,address_schema_version=$10,data_type=$11,element_count=$12,read_options=$13::jsonb,acquisition=$14::jsonb,enabled=$15,sort_order=$16,metadata=$17::jsonb,updated_at=now() WHERE id=$1 AND project_id=$2 AND connection_id=$3`, item.ID, item.ProjectID, item.ConnectionID, item.GroupID, item.Code, item.Name, item.Description, string(address), item.AddressText, item.AddressSchemaVersion, item.DataType, item.ElementCount, string(readOptions), string(acquisition), item.Enabled, item.SortOrder, string(metadata))
		if err != nil {
			return nil, translateCollectorWriteError("更新采集点失败", err)
		}
		if tag.RowsAffected() == 0 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "采集点不存在")
		}
		interval := collectorRefreshInterval(item.Acquisition)
		_, err = tx.Exec(ctx, `UPDATE data_points SET path=$2,name=$3,description=$4,data_type=$5,refresh_mode='auto',refresh_interval_ms=$6,status=$7,display_order=$8,updated_by=$9,updated_at=now() WHERE project_id=$1 AND source_type='collector.point' AND source_id=$10`, item.ProjectID, collectorPointPath(item.ConnectionCode, item.Code), item.Name, item.Description, item.DataType, interval, collectorDataPointStatus(item.Enabled), item.SortOrder, item.UserID, item.ID)
		if err != nil {
			return nil, translateCollectorWriteError("更新采集点映射数据点失败", err)
		}
		ids = append(ids, item.ID)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("提交批量更新采集点事务失败", err)
	}
	return r.listPointsByIDs(ctx, params[0].ProjectID, params[0].ConnectionID, ids)
}

func (r *CollectorRepository) DeletePointsBatch(ctx context.Context, projectID, connectionID string, pointIDs []string) error {
	if len(pointIDs) == 0 {
		return nil
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return wrapUnifiedCollectorRepositoryError("开启批量删除采集点事务失败", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `DELETE FROM data_points WHERE project_id=$1 AND source_type='collector.point' AND source_id=ANY($2::uuid[])`, projectID, pointIDs); err != nil {
		return translateCollectorWriteError("删除采集点映射数据点失败", err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM data_collector_points WHERE project_id=$1 AND connection_id=$2 AND id=ANY($3::uuid[])`, projectID, connectionID, pointIDs)
	if err != nil {
		return translateCollectorWriteError("删除采集点失败", err)
	}
	if tag.RowsAffected() != int64(len(pointIDs)) {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "部分采集点不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return wrapUnifiedCollectorRepositoryError("提交批量删除采集点事务失败", err)
	}
	return nil
}

func (r *CollectorRepository) MovePointsBatch(ctx context.Context, projectID, connectionID string, groupID *string, pointIDs []string) error {
	if len(pointIDs) == 0 {
		return nil
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return wrapUnifiedCollectorRepositoryError("开启批量移动采集点事务失败", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for index, id := range pointIDs {
		tag, err := tx.Exec(ctx, `UPDATE data_collector_points SET group_id=$4,sort_order=$5,updated_at=now() WHERE id=$1 AND project_id=$2 AND connection_id=$3`, id, projectID, connectionID, groupID, index)
		if err != nil {
			return translateCollectorWriteError("移动采集点失败", err)
		}
		if tag.RowsAffected() == 0 {
			return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "采集点不存在")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return wrapUnifiedCollectorRepositoryError("提交批量移动采集点事务失败", err)
	}
	return nil
}

func insertCollectorPointAndDataPoint(ctx context.Context, tx pgx.Tx, item CreateCollectorPointParams) error {
	address, err := json.Marshal(item.Address)
	if err != nil {
		return badCollectorPayload("序列化采集点地址失败", err)
	}
	readOptions, err := json.Marshal(item.ReadOptions)
	if err != nil {
		return badCollectorPayload("序列化采集点读取参数失败", err)
	}
	acquisition, err := json.Marshal(item.Acquisition)
	if err != nil {
		return badCollectorPayload("序列化采集策略失败", err)
	}
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return badCollectorPayload("序列化采集点元数据失败", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO data_collector_points (id,project_id,connection_id,group_id,code,name,description,address,address_text,address_schema_version,data_type,element_count,read_options,acquisition,enabled,sort_order,metadata) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10,$11,$12,$13::jsonb,$14::jsonb,$15,$16,$17::jsonb)`, item.ID, item.ProjectID, item.ConnectionID, item.GroupID, item.Code, item.Name, item.Description, string(address), item.AddressText, item.AddressSchemaVersion, item.DataType, item.ElementCount, string(readOptions), string(acquisition), item.Enabled, item.SortOrder, string(metadata))
	if err != nil {
		return translateCollectorWriteError("创建采集点失败", err)
	}
	sourceConfig := `{}`
	interval := collectorRefreshInterval(item.Acquisition)
	_, err = tx.Exec(ctx, `INSERT INTO data_points (project_id,path,name,description,source_type,source_id,source_config,data_type,refresh_mode,refresh_interval_ms,status,display_order,created_by,updated_by) VALUES ($1,$2,$3,$4,'collector.point',$5,$6::jsonb,$7,'auto',$8,$9,$10,$11,$11)`, item.ProjectID, collectorPointPath(item.ConnectionCode, item.Code), item.Name, item.Description, item.ID, sourceConfig, item.DataType, interval, collectorDataPointStatus(item.Enabled), item.SortOrder, item.UserID)
	if err != nil {
		return translateCollectorWriteError("创建采集点映射数据点失败", err)
	}
	return nil
}

func (r *CollectorRepository) listPointsByIDs(ctx context.Context, projectID, connectionID string, ids []string) ([]CollectorPointRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT id,project_id,connection_id,group_id,code,name,description,address,address_text,address_schema_version,data_type,element_count,read_options,acquisition,enabled,sort_order,metadata,created_at,updated_at FROM data_collector_points WHERE project_id=$1 AND connection_id=$2 AND id=ANY($3::uuid[]) ORDER BY sort_order,id`, projectID, connectionID, ids)
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("读取批量采集点结果失败", err)
	}
	defer rows.Close()
	result := make([]CollectorPointRecord, 0, len(ids))
	for rows.Next() {
		record, err := scanCollectorPoint(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (r *CollectorRepository) GetPointsByIDs(ctx context.Context, projectID, connectionID string, ids []string) ([]CollectorPointRecord, error) {
	if len(ids) == 0 {
		return []CollectorPointRecord{}, nil
	}
	return r.listPointsByIDs(ctx, projectID, connectionID, ids)
}

type unifiedCollectorPointRow interface{ Scan(...any) error }

func scanCollectorPointGroup(row unifiedCollectorPointRow) (CollectorPointGroupRecord, error) {
	var record CollectorPointGroupRecord
	var metadata []byte
	err := row.Scan(&record.ID, &record.ProjectID, &record.ConnectionID, &record.ParentID, &record.Name, &record.SortOrder, &metadata, &record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		return record, err
	}
	if err := json.Unmarshal(metadata, &record.Metadata); err != nil {
		return record, wrapUnifiedCollectorRepositoryError("解析采集点分组元数据失败", err)
	}
	return record, nil
}
func scanCollectorPoint(row unifiedCollectorPointRow) (CollectorPointRecord, error) {
	var record CollectorPointRecord
	var address, readOptions, acquisition, metadata []byte
	err := row.Scan(&record.ID, &record.ProjectID, &record.ConnectionID, &record.GroupID, &record.Code, &record.Name, &record.Description, &address, &record.AddressText, &record.AddressSchemaVersion, &record.DataType, &record.ElementCount, &readOptions, &acquisition, &record.Enabled, &record.SortOrder, &metadata, &record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		return record, err
	}
	if err := decodeCollectorPointJSON(address, &record.Address); err != nil {
		return record, err
	}
	if err := decodeCollectorPointJSON(readOptions, &record.ReadOptions); err != nil {
		return record, err
	}
	if err := decodeCollectorPointJSON(acquisition, &record.Acquisition); err != nil {
		return record, err
	}
	if err := decodeCollectorPointJSON(metadata, &record.Metadata); err != nil {
		return record, err
	}
	return record, nil
}
func scanCollectorPointWithTotal(row unifiedCollectorPointRow) (CollectorPointRecord, int, error) {
	var record CollectorPointRecord
	var address, readOptions, acquisition, metadata []byte
	var total int
	err := row.Scan(&record.ID, &record.ProjectID, &record.ConnectionID, &record.GroupID, &record.Code, &record.Name, &record.Description, &address, &record.AddressText, &record.AddressSchemaVersion, &record.DataType, &record.ElementCount, &readOptions, &acquisition, &record.Enabled, &record.SortOrder, &metadata, &record.CreatedAt, &record.UpdatedAt, &total)
	if err != nil {
		return record, 0, wrapUnifiedCollectorRepositoryError("扫描采集点失败", err)
	}
	if err := decodeCollectorPointJSON(address, &record.Address); err != nil {
		return record, 0, err
	}
	if err := decodeCollectorPointJSON(readOptions, &record.ReadOptions); err != nil {
		return record, 0, err
	}
	if err := decodeCollectorPointJSON(acquisition, &record.Acquisition); err != nil {
		return record, 0, err
	}
	if err := decodeCollectorPointJSON(metadata, &record.Metadata); err != nil {
		return record, 0, err
	}
	return record, total, nil
}

func decodeCollectorPointJSON(payload []byte, target *map[string]any) error {
	if err := json.Unmarshal(payload, target); err != nil {
		return wrapUnifiedCollectorRepositoryError("解析采集点 JSON 失败", err)
	}
	return nil
}

func collectorPointPath(connectionCode, code string) string {
	prefix := strings.TrimSpace(connectionCode)
	var builder strings.Builder
	for _, char := range strings.TrimSpace(code) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '.' || char == '_' || char == '-' {
			builder.WriteRune(char)
		} else {
			builder.WriteByte('_')
		}
	}
	value := builder.String()
	if len(value) > 100 {
		value = value[:100]
	}
	return "collector." + prefix + "." + value
}
func collectorRefreshInterval(acquisition map[string]any) *int {
	value, ok := acquisition["intervalMs"].(float64)
	if !ok {
		if integer, ok := acquisition["intervalMs"].(int); ok {
			result := integer
			return &result
		}
		return nil
	}
	result := int(value)
	return &result
}
func collectorDataPointStatus(enabled bool) string {
	if enabled {
		return "active"
	}
	return "inactive"
}
