package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// DataPointRecord 表示 data_points 表在仓储层的投影结果。
type DataPointRecord struct {
	ID                        string
	ProjectID                 string
	Path                      string
	Name                      string
	Description               *string
	SourceType                string
	SourceID                  *string
	SourceConfig              map[string]any
	DataType                  string
	Unit                      *string
	PrecisionNum              *int
	DefaultValue              *string
	MinValue                  *float64
	MaxValue                  *float64
	AlarmLow                  *float64
	AlarmHigh                 *float64
	Tags                      []any
	AttributeDefaults         map[string]string
	RuntimePermissions        DataPointRuntimePermissions
	RuntimePermissionsDefined bool
	RefreshMode               string
	RefreshIntervalMS         *int
	Status                    string
	DisplayOrder              int
	CreatedBy                 *string
	UpdatedBy                 *string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

const dataPointSelectColumns = `id, project_id, path, name, description, source_type, source_id, source_config, data_type,
       unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags, attribute_defaults, runtime_permissions,
       refresh_mode, refresh_interval_ms, status, display_order, created_by, updated_by, created_at, updated_at`

// DataPointListFilter 表示数据点列表的过滤条件。
type DataPointListFilter struct {
	Type           string
	Status         string
	Search         string
	AccessSourceID string
	SourceID       string
	SourceIDs      []string
	Tags           []string
	SortField      string
	SortOrder      string
	Page           int
	PageSize       int
}

// DataPointUsageRecord 表示一个会阻止数据点删除或需要向用户展示的配置引用。
type DataPointUsageRecord struct {
	Module   string
	ObjectID string
	Label    string
}

// DataPointTagSummary 表示工程内标签及其真实引用数据点数量。
type DataPointTagSummary struct {
	Tag   string
	Count int
}

// DataPointRepository 封装 data_points 的参数化 SQL 访问。
type DataPointRepository struct {
	pool *pgxpool.Pool
}

// NewDataPointRepository 创建数据点仓储。
func NewDataPointRepository(pool *pgxpool.Pool) *DataPointRepository {
	return &DataPointRepository{pool: pool}
}

// ListByProject 按项目分页查询 data_points。
// 查询路径：project_id + 可选过滤条件 + created_at/display_order 排序，主要命中 data_points_project_status_idx / data_points_source_idx / data_points_project_path_key。
// 潜在性能风险：search 使用 ILIKE 会导致回表放大；如果数据点规模继续增长，应考虑额外的路径搜索索引或前缀查询策略。
func (r *DataPointRepository) ListByProject(ctx context.Context, projectID string, filter DataPointListFilter) ([]DataPointRecord, int, error) {
	maxPageSize := 500
	if len(filter.SourceIDs) > 0 {
		maxPageSize = 5000
	}
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 50, maxPageSize)
	whereSQL, args, err := buildDataPointWhereClause(projectID, filter)
	if err != nil {
		return nil, 0, err
	}

	countSQL := `SELECT COUNT(*) FROM data_points WHERE ` + whereSQL
	var total int
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计数据点列表失败", err)
	}

	selectArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	orderSQL := resolveDataPointOrderSQL(filter.SortField, filter.SortOrder)
	rows, err := r.pool.Query(ctx, `
		SELECT `+dataPointSelectColumns+`
		FROM data_points
        WHERE `+whereSQL+`
        ORDER BY `+orderSQL+`
        LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2)+`
    `, selectArgs...)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询数据点列表失败", err)
	}
	defer rows.Close()

	records := make([]DataPointRecord, 0)
	for rows.Next() {
		record, scanErr := scanDataPointRecord(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历数据点列表失败", err)
	}

	return records, total, nil
}

// ListUsageCounts 批量统计报警配置和计算单元对当前页数据点的引用数。
func (r *DataPointRepository) ListUsageCounts(ctx context.Context, projectID string, ids []string) (map[string]int, error) {
	result := make(map[string]int, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
		WITH usages AS (
			SELECT item.datapoint_id, 'alarm:' || item.id::text AS usage_key
			FROM data_alarm_items item
			WHERE item.project_id = $1 AND item.datapoint_id = ANY($2::uuid[])
			UNION
			SELECT input.datapoint_id, 'alarm:' || input.alarm_item_id::text AS usage_key
			FROM data_alarm_item_inputs input
			JOIN data_alarm_items item ON item.id = input.alarm_item_id
			WHERE item.project_id = $1 AND input.datapoint_id = ANY($2::uuid[])
			UNION
			SELECT datapoint.id, 'compute:' || unit.id::text AS usage_key
			FROM data_compute_units unit
			JOIN LATERAL jsonb_array_elements(
				CASE WHEN jsonb_typeof(unit.input_bindings->'datapointVariables') = 'array'
					THEN unit.input_bindings->'datapointVariables' ELSE '[]'::jsonb END
			) variable ON true
			JOIN data_points datapoint
			  ON datapoint.project_id = unit.project_id
			 AND datapoint.path = COALESCE(variable->>'path', variable->>'datapointPath')
			WHERE unit.project_id = $1 AND datapoint.id = ANY($2::uuid[])
		)
		SELECT datapoint_id, COUNT(*) FROM usages GROUP BY datapoint_id
	`, projectID, ids)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计数据点引用失败", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取数据点引用统计失败", err)
		}
		result[id] = count
	}
	return result, rows.Err()
}

// ListUsages 返回单个数据点的报警和计算引用明细。
func (r *DataPointRepository) ListUsages(ctx context.Context, projectID, datapointID string) ([]DataPointUsageRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT module, object_id, label FROM (
			SELECT 'alarm'::text AS module, item.id::text AS object_id, item.display_name AS label
			FROM data_alarm_items item
			WHERE item.project_id = $1 AND item.datapoint_id = $2
			UNION
			SELECT 'alarm'::text AS module, item.id::text AS object_id, item.display_name AS label
			FROM data_alarm_item_inputs input
			JOIN data_alarm_items item ON item.id = input.alarm_item_id
			WHERE item.project_id = $1 AND input.datapoint_id = $2
			UNION
			SELECT 'compute'::text AS module, unit.id::text AS object_id, unit.name AS label
			FROM data_compute_units unit
			JOIN LATERAL jsonb_array_elements(
				CASE WHEN jsonb_typeof(unit.input_bindings->'datapointVariables') = 'array'
					THEN unit.input_bindings->'datapointVariables' ELSE '[]'::jsonb END
			) variable ON true
			JOIN data_points datapoint
			  ON datapoint.project_id = unit.project_id
			 AND datapoint.path = COALESCE(variable->>'path', variable->>'datapointPath')
			WHERE unit.project_id = $1 AND datapoint.id = $2
		) usages
		ORDER BY module, label, object_id
	`, projectID, datapointID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询数据点引用失败", err)
	}
	defer rows.Close()
	result := make([]DataPointUsageRecord, 0)
	for rows.Next() {
		var item DataPointUsageRecord
		if err := rows.Scan(&item.Module, &item.ObjectID, &item.Label); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取数据点引用失败", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// ListTagSummaries 按工程统计全部标签，避免前端只看到当前页标签。
func (r *DataPointRepository) ListTagSummaries(ctx context.Context, projectID string) ([]DataPointTagSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT tag.value, COUNT(*)
		FROM data_points datapoint
		JOIN LATERAL jsonb_array_elements_text(
			CASE WHEN jsonb_typeof(datapoint.tags) = 'array' THEN datapoint.tags ELSE '[]'::jsonb END
		) tag(value) ON true
		WHERE datapoint.project_id = $1
		GROUP BY tag.value
		ORDER BY tag.value
	`, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询数据点标签失败", err)
	}
	defer rows.Close()
	result := make([]DataPointTagSummary, 0)
	for rows.Next() {
		var item DataPointTagSummary
		if err := rows.Scan(&item.Tag, &item.Count); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取数据点标签失败", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// RemoveTagByProject 在单个事务中移除工程内所有数据点上的指定标签。
func (r *DataPointRepository) RemoveTagByProject(ctx context.Context, projectID, tag, userID string) (int, error) {
	payload, err := json.Marshal([]string{tag})
	if err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "标签格式无效", err)
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开始删除数据点标签事务失败", err)
	}
	defer rollbackDataPointTxQuietly(ctx, tx)
	commandTag, err := tx.Exec(ctx, `
		UPDATE data_points
		SET tags = tags - $2,
			updated_by = NULLIF($3, '')::uuid,
			updated_at = now()
		WHERE project_id = $1 AND tags @> $4::jsonb
	`, projectID, tag, userID, string(payload))
	if err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除数据点标签失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交删除数据点标签事务失败", err)
	}
	return int(commandTag.RowsAffected()), nil
}

// ListAllByProject 仅供生成开发契约快照使用，按路径稳定排序返回完整内部记录。
// 上层必须负责过滤私有字段，仓储层不直接面向 HTTP 暴露结果。
func (r *DataPointRepository) ListAllByProject(ctx context.Context, projectID string) ([]DataPointRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+dataPointSelectColumns+`
		FROM data_points
		WHERE project_id = $1
		ORDER BY path ASC, id ASC
	`, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询数据点开发契约失败", err)
	}
	defer rows.Close()

	records := make([]DataPointRecord, 0)
	for rows.Next() {
		record, scanErr := scanDataPointRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历数据点开发契约失败", err)
	}
	return records, nil
}

// GetByProjectAndID 按项目与主键读取单条数据点。
// 查询路径：project_id + id，主命中 data_points_pkey；project_id 作为边界约束避免跨项目误读。
// 潜在性能风险：单条主键读取性能稳定，但高频调用 value 接口时应关注上游查询执行成本而非此处扫描成本。
func (r *DataPointRepository) GetByProjectAndID(ctx context.Context, projectID, id string) (*DataPointRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+dataPointSelectColumns+`
        FROM data_points
        WHERE project_id = $1 AND id = $2
    `, projectID, id)

	record, err := scanDataPointRecord(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByProjectAndPath 按项目与路径读取单条数据点。
// 查询路径：project_id + path，主命中 data_points_project_path_key；这是 value 接口的主路径，边界清晰且具备唯一性。
// 潜在性能风险：若 path 频繁被模糊搜索，不要把该唯一索引误当作搜索索引使用。
func (r *DataPointRepository) GetByProjectAndPath(ctx context.Context, projectID, path string) (*DataPointRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+dataPointSelectColumns+`
        FROM data_points
        WHERE project_id = $1 AND path = $2
    `, projectID, path)

	record, err := scanDataPointRecord(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByProjectAndIDs 按项目与 ID 列表读取数据点。
// 查询路径：project_id + id 列表，主命中 data_points_pkey；适合批量删除前做存在性与状态校验。
// 潜在性能风险：当批量 ID 数量过大时，ANY(...uuid[]) 的参数体积会增加，仍需控制批量大小。
func (r *DataPointRepository) GetByProjectAndIDs(ctx context.Context, projectID string, ids []string) ([]DataPointRecord, error) {
	if len(ids) == 0 {
		return []DataPointRecord{}, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT `+dataPointSelectColumns+`
        FROM data_points
        WHERE project_id = $1 AND id = ANY($2::uuid[])
        ORDER BY id
    `, projectID, ids)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "批量读取数据点失败", err)
	}
	defer rows.Close()

	records := make([]DataPointRecord, 0, len(ids))
	for rows.Next() {
		record, scanErr := scanDataPointRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历批量数据点失败", err)
	}

	return records, nil
}

// Update 更新单条数据点记录。
// 查询路径：project_id + id，主命中 data_points_pkey；更新保留项目边界，不允许跨项目写入。
// 潜在性能风险：该方法会整体重写 tags/source_config 等 JSON 字段，若这些字段后续膨胀较大，应考虑局部更新策略。
func (r *DataPointRepository) Update(ctx context.Context, params UpdateDataPointParams) (*DataPointRecord, error) {
	sourceConfigBytes, err := marshalJSONObject(params.SourceConfig)
	if err != nil {
		return nil, err
	}
	tagsBytes, err := marshalJSONArray(params.Tags)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, `
        UPDATE data_points
        SET name = $3,
            description = $4,
            source_type = $5,
            source_id = $6,
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
            updated_by = $20,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING `+dataPointSelectColumns+`
    `, params.ProjectID, params.ID, params.Name, params.Description, params.SourceType, params.SourceID, string(sourceConfigBytes), params.DataType, params.Unit, params.PrecisionNum, params.DefaultValue, params.MinValue, params.MaxValue, params.AlarmLow, params.AlarmHigh, string(tagsBytes), params.RefreshMode, params.RefreshIntervalMS, params.Status, params.UserID)

	record, err := scanDataPointRecord(row)
	if err != nil {
		return nil, translateDataPointWriteError(err)
	}

	return &record, nil
}

// UpdateRuntimePermissions 只更新运行态权限 JSON 字段。
// 说明：本轮仅开放 write 权限，因此这里整体覆盖 runtime_permissions，避免与其他业务字段耦合。
func (r *DataPointRepository) UpdateRuntimePermissions(ctx context.Context, params UpdateDataPointRuntimePermissionsParams) (*DataPointRecord, error) {
	runtimePermissionsBytes, err := marshalDataPointRuntimePermissions(params.RuntimePermissions)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, `
        UPDATE data_points
        SET runtime_permissions = $3::jsonb,
            updated_by = $4,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING `+dataPointSelectColumns+`
    `, params.ProjectID, params.ID, string(runtimePermissionsBytes), params.UserID)

	record, err := scanDataPointRecord(row)
	if err != nil {
		return nil, translateDataPointWriteError(err)
	}
	return &record, nil
}

// UpdateAttributeDefaults 原子替换单个数据点的开发态自定义属性默认值。
func (r *DataPointRepository) UpdateAttributeDefaults(ctx context.Context, params UpdateDataPointAttributeDefaultsParams) (*DataPointRecord, error) {
	attributeDefaultsBytes, err := json.Marshal(params.AttributeDefaults)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "自定义属性格式无效", err)
	}

	row := r.pool.QueryRow(ctx, `
        UPDATE data_points
        SET attribute_defaults = $3::jsonb,
            updated_by = $4,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING `+dataPointSelectColumns+`
    `, params.ProjectID, params.ID, string(attributeDefaultsBytes), params.UserID)

	record, err := scanDataPointRecord(row)
	if err != nil {
		return nil, translateDataPointWriteError(err)
	}
	return &record, nil
}

// Delete 按项目与主键删除单条数据点。
// 查询路径：project_id + id，主命中 data_points_pkey；项目边界与主键组合可避免误删其他项目记录。
// 潜在性能风险：单条删除通常很轻量，但如果未来引入批量级联同步，应注意不要在这里堆叠额外循环。
func (r *DataPointRepository) Delete(ctx context.Context, projectID, id string) error {
	commandTag, err := r.pool.Exec(ctx, `
        DELETE FROM data_points
        WHERE project_id = $1 AND id = $2
	`, projectID, id)
	if err != nil {
		return translateDataPointDeleteError("删除数据点失败", err)
	}
	if commandTag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "数据点不存在")
	}
	return nil
}

// DeleteInvalidBatch 删除一批已确认无效的数据点。
// 查询路径：project_id + id 列表 + status = invalid，主命中 data_points_pkey；先校验再删除可避免误删有效数据点。
// 潜在性能风险：当批量 ID 很大时，ANY(...) 与前置校验会产生两次扫描，调用方应保持批次适中。
func (r *DataPointRepository) DeleteInvalidBatch(ctx context.Context, projectID string, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "删除列表不能为空")
	}

	commandTag, err := r.pool.Exec(ctx, `
        DELETE FROM data_points
        WHERE project_id = $1 AND id = ANY($2::uuid[]) AND status = 'invalid'
	`, projectID, ids)
	if err != nil {
		return 0, translateDataPointDeleteError("批量删除数据点失败", err)
	}
	return int(commandTag.RowsAffected()), nil
}

// DeleteInvalidByFilter 按筛选条件批量删除无效数据点。
// 说明：用于前端“选择全部筛选结果”场景，where 条件与列表查询共用，避免前端拉取全量 ID。
func (r *DataPointRepository) DeleteInvalidByFilter(ctx context.Context, projectID string, filter DataPointListFilter) (int, error) {
	filter.Status = "invalid"
	whereSQL, args, err := buildDataPointWhereClause(projectID, filter)
	if err != nil {
		return 0, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开始批量删除数据点事务失败", err)
	}
	defer rollbackDataPointTxQuietly(ctx, tx)

	commandTag, err := tx.Exec(ctx, `DELETE FROM data_points WHERE `+whereSQL, args...)
	if err != nil {
		return 0, translateDataPointDeleteError("按筛选条件批量删除数据点失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交批量删除数据点事务失败", err)
	}
	return int(commandTag.RowsAffected()), nil
}

func translateDataPointDeleteError(message string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "数据点正在被报警策略或其他配置引用，请先解除引用")
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}

// AppendTagsByFilter 按筛选条件批量追加标签。
// 说明：JSONB 数组在 SQL 内合并去重，保证跨页批量操作只发起一次更新并保持事务一致。
func (r *DataPointRepository) AppendTagsByFilter(ctx context.Context, projectID string, filter DataPointListFilter, tags []string, userID string) (int, error) {
	whereSQL, args, err := buildDataPointWhereClause(projectID, filter)
	if err != nil {
		return 0, err
	}
	tagBytes, err := json.Marshal(tags)
	if err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "标签格式无效", err)
	}
	args = append(args, string(tagBytes), userID)
	tagsArgIndex := len(args) - 1
	userArgIndex := len(args)

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开始批量更新标签事务失败", err)
	}
	defer rollbackDataPointTxQuietly(ctx, tx)

	commandTag, err := tx.Exec(ctx, `
        UPDATE data_points
        SET tags = (
                SELECT COALESCE(jsonb_agg(DISTINCT tag_item.value), '[]'::jsonb)
                FROM jsonb_array_elements(tags || $`+fmt.Sprint(tagsArgIndex)+`::jsonb) AS tag_item(value)
            ),
            updated_by = NULLIF($`+fmt.Sprint(userArgIndex)+`, '')::uuid,
            updated_at = now()
        WHERE `+whereSQL, args...)
	if err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "按筛选条件批量更新标签失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交批量更新标签事务失败", err)
	}
	return int(commandTag.RowsAffected()), nil
}

func rollbackDataPointTxQuietly(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

// DataPointUpdateParams 表示数据点更新时的仓储参数。
type UpdateDataPointParams struct {
	ID                string
	ProjectID         string
	UserID            string
	Name              string
	Description       *string
	SourceType        string
	SourceID          *string
	SourceConfig      map[string]any
	DataType          string
	Unit              *string
	PrecisionNum      *int
	DefaultValue      *string
	MinValue          *float64
	MaxValue          *float64
	AlarmLow          *float64
	AlarmHigh         *float64
	Tags              []any
	RefreshMode       string
	RefreshIntervalMS *int
	Status            string
}

// UpdateDataPointRuntimePermissionsParams 表示运行态权限更新参数。
type UpdateDataPointRuntimePermissionsParams struct {
	ID                 string
	ProjectID          string
	UserID             string
	RuntimePermissions DataPointRuntimePermissions
}

// UpdateDataPointAttributeDefaultsParams 表示自定义属性默认值更新参数。
type UpdateDataPointAttributeDefaultsParams struct {
	ID                string
	ProjectID         string
	UserID            string
	AttributeDefaults map[string]string
}

type dataPointScannable interface {
	Scan(dest ...any) error
}

func scanDataPointRecord(row dataPointScannable) (DataPointRecord, error) {
	var (
		record                  DataPointRecord
		description             sql.NullString
		sourceID                pgtype.UUID
		unit                    sql.NullString
		precisionNum            sql.NullInt32
		defaultValue            sql.NullString
		minValue                sql.NullFloat64
		maxValue                sql.NullFloat64
		alarmLow                sql.NullFloat64
		alarmHigh               sql.NullFloat64
		refreshIntervalMS       sql.NullInt32
		createdBy               pgtype.UUID
		updatedBy               pgtype.UUID
		sourceConfigBytes       []byte
		tagsBytes               []byte
		attributeDefaultsBytes  []byte
		runtimePermissionsBytes []byte
	)

	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.Path,
		&record.Name,
		&description,
		&record.SourceType,
		&sourceID,
		&sourceConfigBytes,
		&record.DataType,
		&unit,
		&precisionNum,
		&defaultValue,
		&minValue,
		&maxValue,
		&alarmLow,
		&alarmHigh,
		&tagsBytes,
		&attributeDefaultsBytes,
		&runtimePermissionsBytes,
		&record.RefreshMode,
		&refreshIntervalMS,
		&record.Status,
		&record.DisplayOrder,
		&createdBy,
		&updatedBy,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DataPointRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "数据点不存在")
		}
		return DataPointRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取数据点失败", err)
	}

	record.Description = nullStringToPtr(description)
	record.SourceID = uuidToPtr(sourceID)
	record.Unit = nullStringToPtr(unit)
	record.PrecisionNum = nullInt32ToPtr(precisionNum)
	record.DefaultValue = nullStringToPtr(defaultValue)
	record.MinValue = nullFloat64ToPtr(minValue)
	record.MaxValue = nullFloat64ToPtr(maxValue)
	record.AlarmLow = nullFloat64ToPtr(alarmLow)
	record.AlarmHigh = nullFloat64ToPtr(alarmHigh)
	record.RefreshIntervalMS = nullInt32ToPtr(refreshIntervalMS)
	record.CreatedBy = uuidToPtr(createdBy)
	record.UpdatedBy = uuidToPtr(updatedBy)
	record.SourceConfig = map[string]any{}
	if len(sourceConfigBytes) > 0 {
		if err := json.Unmarshal(sourceConfigBytes, &record.SourceConfig); err != nil {
			return DataPointRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析数据点 source_config 失败", err)
		}
		if record.SourceConfig == nil {
			record.SourceConfig = map[string]any{}
		}
	}
	record.Tags = make([]any, 0)
	if len(tagsBytes) > 0 {
		if err := json.Unmarshal(tagsBytes, &record.Tags); err != nil {
			return DataPointRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析数据点 tags 失败", err)
		}
		if record.Tags == nil {
			record.Tags = make([]any, 0)
		}
	}
	record.AttributeDefaults = map[string]string{}
	if len(attributeDefaultsBytes) > 0 {
		if err := json.Unmarshal(attributeDefaultsBytes, &record.AttributeDefaults); err != nil {
			return DataPointRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析数据点 attribute_defaults 失败", err)
		}
		if record.AttributeDefaults == nil {
			record.AttributeDefaults = map[string]string{}
		}
	}
	runtimePermissions, err := unmarshalDataPointRuntimePermissions(runtimePermissionsBytes)
	if err != nil {
		return DataPointRecord{}, err
	}
	record.RuntimePermissions = runtimePermissions
	record.RuntimePermissionsDefined = true

	return record, nil
}

func buildDataPointWhereClause(projectID string, filter DataPointListFilter) (string, []any, error) {
	clauses := []string{"project_id = $1"}
	args := []any{projectID}

	addStringClause := func(column, value string) error {
		if strings.TrimSpace(value) == "" {
			return nil
		}
		args = append(args, strings.TrimSpace(value))
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
		return nil
	}

	if err := addStringClause("source_type", filter.Type); err != nil {
		return "", nil, err
	}
	if err := addStringClause("status", filter.Status); err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(filter.AccessSourceID) != "" {
		args = append(args, strings.TrimSpace(filter.AccessSourceID))
		argIndex := len(args)
		// 接入源筛选是用户可见维度；连接 ID 可能存于 source_config、结构化来源表或 source_id。
		clauses = append(clauses, fmt.Sprintf(`COALESCE(
            NULLIF(BTRIM(source_config->>'connectionId'), ''),
            NULLIF(BTRIM(source_config->>'sourceConnectionId'), ''),
            (SELECT source_query.connection_id::text FROM data_queries source_query
             WHERE data_points.source_type='db.query' AND source_query.project_id=data_points.project_id AND source_query.id=data_points.source_id),
            (SELECT subscription.connection_id::text FROM data_mqtt_subscriptions subscription
             WHERE data_points.source_type='mqtt.subscription' AND subscription.project_id=data_points.project_id AND subscription.id=data_points.source_id),
			(SELECT subscription.connection_id::text FROM data_mqtt_tags tag
             JOIN data_mqtt_subscriptions subscription
               ON subscription.project_id=tag.project_id AND subscription.id=tag.subscription_id
			 WHERE data_points.source_type='mqtt.tag' AND tag.project_id=data_points.project_id AND tag.id=data_points.source_id),
			(SELECT collector_point.connection_id::text FROM data_collector_points collector_point
			 WHERE data_points.source_type='collector.point' AND collector_point.project_id=data_points.project_id AND collector_point.id=data_points.source_id),
			source_id::text
        ) = $%d`, argIndex))
	}
	if err := addStringClause("source_id", filter.SourceID); err != nil {
		return "", nil, err
	}

	if len(filter.SourceIDs) > 0 {
		args = append(args, filter.SourceIDs)
		clauses = append(clauses, fmt.Sprintf("source_id = ANY($%d::uuid[])", len(args)))
	}

	for _, tag := range filter.Tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		payload, err := json.Marshal([]string{trimmed})
		if err != nil {
			return "", nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "标签筛选格式无效", err)
		}
		args = append(args, string(payload))
		clauses = append(clauses, fmt.Sprintf("tags @> $%d::jsonb", len(args)))
	}

	search := strings.TrimSpace(filter.Search)
	if search != "" {
		args = append(args, "%"+search+"%")
		clauseIndex := len(args)
		clauses = append(clauses, fmt.Sprintf("(name ILIKE $%d OR path ILIKE $%d)", clauseIndex, clauseIndex))
	}

	return strings.Join(clauses, " AND "), args, nil
}

func resolveDataPointOrderSQL(sortField, sortOrder string) string {
	direction := "DESC"
	if strings.EqualFold(strings.TrimSpace(sortOrder), "asc") {
		direction = "ASC"
	}

	switch strings.TrimSpace(sortField) {
	case "name":
		return "name " + direction + ", path " + direction + ", id " + direction
	case "path":
		return "path " + direction + ", id " + direction
	default:
		// 前端只暴露“创建时间”排序；display_order 仅作为同创建时间下的内部稳定顺序，方向跟随创建时间。
		return "created_at " + direction + ", display_order " + direction + ", id " + direction
	}
}

func marshalJSONArray(value []any) ([]byte, error) {
	if value == nil {
		value = []any{}
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "JSON 数组格式无效", err)
	}
	return bytes, nil
}

func uuidToPtr(value pgtype.UUID) *string {
	if !value.Valid {
		return nil
	}
	result := value.String()
	return &result
}

func nullInt32ToPtr(value sql.NullInt32) *int {
	if !value.Valid {
		return nil
	}
	result := int(value.Int32)
	return &result
}

func nullFloat64ToPtr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	result := value.Float64
	return &result
}

func translateDataPointWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点路径已存在")
		case "23503":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "关联资源不存在")
		}
	}
	return err
}
