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
	RuntimePermissions        DataPointRuntimePermissions
	RuntimePermissionsDefined bool
	RefreshMode               string
	RefreshIntervalMS         *int
	Status                    string
	CreatedBy                 *string
	UpdatedBy                 *string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

// DataPointListFilter 表示数据点列表的过滤条件。
type DataPointListFilter struct {
	Type      string
	Status    string
	Search    string
	SourceID  string
	SourceIDs []string
	Page      int
	PageSize  int
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
// 查询路径：project_id + 可选过滤条件 + created_at 排序，主要命中 data_points_project_status_idx / data_points_source_idx / data_points_project_path_key。
// 潜在性能风险：search 使用 ILIKE 会导致回表放大；如果数据点规模继续增长，应考虑额外的路径搜索索引或前缀查询策略。
func (r *DataPointRepository) ListByProject(ctx context.Context, projectID string, filter DataPointListFilter) ([]DataPointRecord, int, error) {
	maxPageSize := 200
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
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, path, name, description, source_type, source_id, source_config, data_type,
		       unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags, runtime_permissions,
		       refresh_mode, refresh_interval_ms, status, created_by, updated_by, created_at, updated_at
        FROM data_points
        WHERE `+whereSQL+`
        ORDER BY created_at DESC
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

// GetByProjectAndID 按项目与主键读取单条数据点。
// 查询路径：project_id + id，主命中 data_points_pkey；project_id 作为边界约束避免跨项目误读。
// 潜在性能风险：单条主键读取性能稳定，但高频调用 value 接口时应关注上游查询执行成本而非此处扫描成本。
func (r *DataPointRepository) GetByProjectAndID(ctx context.Context, projectID, id string) (*DataPointRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, path, name, description, source_type, source_id, source_config, data_type,
		       unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags, runtime_permissions,
		       refresh_mode, refresh_interval_ms, status, created_by, updated_by, created_at, updated_at
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
		SELECT id, project_id, path, name, description, source_type, source_id, source_config, data_type,
		       unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags, runtime_permissions,
		       refresh_mode, refresh_interval_ms, status, created_by, updated_by, created_at, updated_at
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
		SELECT id, project_id, path, name, description, source_type, source_id, source_config, data_type,
		       unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags, runtime_permissions,
		       refresh_mode, refresh_interval_ms, status, created_by, updated_by, created_at, updated_at
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
        RETURNING id, project_id, path, name, description, source_type, source_id, source_config, data_type,
                  unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags, runtime_permissions,
                  refresh_mode, refresh_interval_ms, status, created_by, updated_by, created_at, updated_at
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
        RETURNING id, project_id, path, name, description, source_type, source_id, source_config, data_type,
                  unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags, runtime_permissions,
                  refresh_mode, refresh_interval_ms, status, created_by, updated_by, created_at, updated_at
    `, params.ProjectID, params.ID, string(runtimePermissionsBytes), params.UserID)

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
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除数据点失败", err)
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
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "批量删除数据点失败", err)
	}
	return int(commandTag.RowsAffected()), nil
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
		&runtimePermissionsBytes,
		&record.RefreshMode,
		&refreshIntervalMS,
		&record.Status,
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
	if err := addStringClause("source_id", filter.SourceID); err != nil {
		return "", nil, err
	}

	if len(filter.SourceIDs) > 0 {
		args = append(args, filter.SourceIDs)
		clauses = append(clauses, fmt.Sprintf("source_id = ANY($%d::uuid[])", len(args)))
	}

	search := strings.TrimSpace(filter.Search)
	if search != "" {
		args = append(args, "%"+search+"%")
		clauseIndex := len(args)
		clauses = append(clauses, fmt.Sprintf("(name ILIKE $%d OR path ILIKE $%d)", clauseIndex, clauseIndex))
	}

	return strings.Join(clauses, " AND "), args, nil
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
