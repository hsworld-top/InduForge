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
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// ComputeUnitRecord 表示 data_compute_units 的仓储层投影。
type ComputeUnitRecord struct {
	ID            string
	ProjectID     string
	Name          string
	Description   *string
	FolderID      *string
	Language      string
	ScriptCode    string
	TriggerType   string
	TriggerConfig map[string]any
	InputBindings map[string]any
	Outputs       []ComputeOutputRecord
	Dependencies  []any
	TimeoutMS     int
	IsEnabled     bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ComputeFolderRecord 表示 data_compute_folders 的仓储层投影。
type ComputeFolderRecord struct {
	ID        string
	ProjectID string
	Name      string
	ParentID  *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ComputeFolderPageRecord 是面向目录懒加载接口的轻量投影。
type ComputeFolderPageRecord struct {
	ComputeFolderRecord
	Path        string
	HasChildren bool
	UnitCount   int
}

// ComputeFolderListFilter 描述目录按父级加载和搜索条件。
type ComputeFolderListFilter struct {
	ParentID *string
	Search   string
	Page     int
	PageSize int
}

// ComputeRunRecord 表示 data_compute_runs 的仓储层投影。
type ComputeRunRecord struct {
	ID            int64
	ProjectID     string
	ComputeUnitID string
	TriggerMode   string
	Status        string
	DurationMS    int
	Output        any
	ErrorMessage  *string
	StartedAt     time.Time
	FinishedAt    time.Time
	CreatedAt     time.Time
}

// ComputeUnitListFilter 表示计算单元列表过滤条件。
type ComputeUnitListFilter struct {
	Language string
	Enabled  *bool
	Search   string
	FolderID *string
	Page     int
	PageSize int
}

// ComputeRunListFilter 表示计算运行记录列表过滤条件。
type ComputeRunListFilter struct {
	UnitID      string
	Status      string
	TriggerMode string
	Page        int
	PageSize    int
}

// CreateComputeUnitParams 描述创建计算单元时的落库参数。
type CreateComputeUnitParams struct {
	ProjectID     string
	UserID        string
	Name          string
	Description   *string
	FolderID      *string
	Language      string
	ScriptCode    string
	TriggerType   string
	TriggerConfig map[string]any
	InputBindings map[string]any
	Dependencies  []any
	TimeoutMS     int
	DatapointRefs []ComputeDatapointRefParam
	Outputs       []ComputeOutputParam
}

// CreateComputeFolderParams 描述创建计算单元文件夹时的落库参数。
type CreateComputeFolderParams struct {
	ProjectID string
	UserID    string
	Name      string
	ParentID  *string
}

// UpdateComputeUnitParams 描述更新计算单元时的落库参数。
type UpdateComputeUnitParams struct {
	ID            string
	ProjectID     string
	UserID        string
	Name          string
	Description   *string
	FolderID      *string
	Language      string
	ScriptCode    string
	TriggerType   string
	TriggerConfig map[string]any
	InputBindings map[string]any
	Dependencies  []any
	TimeoutMS     int
	IsEnabled     bool
	DatapointRefs []ComputeDatapointRefParam
	Outputs       []ComputeOutputParam
}

// ComputeDatapointRefParam 是计算输入或点变触发对数据点的规范化引用。
type ComputeDatapointRefParam struct {
	DatapointID string
	Role        string
	Alias       *string
	SortOrder   int
}

// ComputeOutputParam 描述需要与计算单元定义原子同步的输出数据点。
type ComputeOutputParam struct {
	ID           *string
	OutputKey    string
	Name         string
	Path         string
	DataType     string
	Unit         *string
	PrecisionNum *int
	NullPolicy   string
	Description  *string
}

// ComputeOutputRecord 是规范化计算输出及其稳定生成点身份。
type ComputeOutputRecord struct {
	ID           string
	DatapointID  string
	OutputKey    string
	Name         string
	Description  *string
	Path         string
	DataType     string
	Unit         *string
	PrecisionNum *int
	NullPolicy   string
	SortOrder    int
}

// HasComputeDependencyPath 判断 fromUnit 是否直接或间接依赖 toUnit。
func (r *ComputeRepository) HasComputeDependencyPath(ctx context.Context, projectID, fromUnitID, toUnitID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `WITH RECURSIVE edges AS (
		SELECT ref.compute_unit_id AS dependent_id, point.source_id AS dependency_id
		FROM data_compute_unit_datapoint_refs ref JOIN data_points point ON point.id=ref.datapoint_id
		WHERE ref.project_id=$1 AND point.source_type='calc.output' AND point.source_id IS NOT NULL
	), walk(id) AS (
		SELECT $2::uuid UNION SELECT edge.dependency_id FROM edges edge JOIN walk ON edge.dependent_id=walk.id
	) SELECT EXISTS(SELECT 1 FROM walk WHERE id=$3::uuid)`, projectID, fromUnitID, toUnitID).Scan(&exists)
	if err != nil {
		return false, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "检查计算依赖环失败", err)
	}
	return exists, nil
}

func (r *ComputeRepository) hydrateComputeOutputs(ctx context.Context, unit *ComputeUnitRecord) error {
	return hydrateComputeOutputsFromPool(ctx, r.pool, unit)
}

func hydrateComputeOutputsFromPool(ctx context.Context, pool *pgxpool.Pool, unit *ComputeUnitRecord) error {
	if unit == nil {
		return nil
	}
	rows, err := pool.Query(ctx, `
		SELECT output.id::text,output.datapoint_id::text,output.output_key,point.name,point.description,
		       output.path,output.data_type,output.unit,output.precision_num,output.null_policy,output.sort_order
		FROM data_compute_unit_outputs output
		JOIN data_points point ON point.id=output.datapoint_id
		WHERE output.project_id=$1 AND output.compute_unit_id=$2
		ORDER BY output.sort_order,output.id
	`, unit.ProjectID, unit.ID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询计算输出失败", err)
	}
	defer rows.Close()
	unit.Outputs = make([]ComputeOutputRecord, 0)
	for rows.Next() {
		var output ComputeOutputRecord
		if err := rows.Scan(&output.ID, &output.DatapointID, &output.OutputKey, &output.Name, &output.Description, &output.Path, &output.DataType, &output.Unit, &output.PrecisionNum, &output.NullPolicy, &output.SortOrder); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取计算输出失败", err)
		}
		unit.Outputs = append(unit.Outputs, output)
	}
	if err := rows.Err(); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历计算输出失败", err)
	}
	return nil
}

func computeOutputRecordsFromParams(outputs []ComputeOutputParam) []ComputeOutputRecord {
	result := make([]ComputeOutputRecord, 0, len(outputs))
	for index, output := range outputs {
		key := strings.TrimSpace(output.OutputKey)
		if key == "" {
			key = strings.TrimSpace(output.Name)
		}
		nullPolicy := strings.TrimSpace(output.NullPolicy)
		if nullPolicy == "" {
			nullPolicy = "error"
		}
		result = append(result, ComputeOutputRecord{OutputKey: key, Name: output.Name, Description: output.Description, Path: output.Path, DataType: output.DataType, Unit: output.Unit, PrecisionNum: output.PrecisionNum, NullPolicy: nullPolicy, SortOrder: index})
	}
	return result
}

// SaveComputeRunParams 描述写入运行记录时的落库参数。
type SaveComputeRunParams struct {
	ProjectID     string
	ComputeUnitID string
	TriggerMode   string
	Status        string
	DurationMS    int
	Output        any
	ErrorMessage  *string
	StartedAt     time.Time
	FinishedAt    time.Time
	OutputValues  []ComputeRunOutputValueParam
}

type ComputeRunOutputValueParam struct {
	Name  string
	Value string
}

// ComputeRepository 封装计算域参数化 SQL 访问。
type ComputeRepository struct {
	pool *pgxpool.Pool
}

// NewComputeRepository 创建计算域仓储。
func NewComputeRepository(pool *pgxpool.Pool) *ComputeRepository {
	return &ComputeRepository{pool: pool}
}

// CreateUnit 写入一个新的计算单元定义。
func (r *ComputeRepository) CreateUnit(ctx context.Context, params CreateComputeUnitParams) (*ComputeUnitRecord, error) {
	triggerConfigPayload, err := marshalComputeObject(params.TriggerConfig)
	if err != nil {
		return nil, err
	}
	inputBindingsPayload, err := marshalComputeObject(params.InputBindings)
	if err != nil {
		return nil, err
	}
	dependenciesPayload, err := marshalComputeArray(params.Dependencies)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启计算单元创建事务失败", err)
	}
	defer tx.Rollback(ctx)
	if err := lockComputeProject(ctx, tx, params.ProjectID); err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO data_compute_units (
			project_id,
			name,
			description,
			folder_id,
			language,
			script_code,
			trigger_type,
			trigger_config,
			input_bindings,
			dependencies,
			timeout_ms,
			created_by,
			updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb, $11, $12, $12)
		RETURNING id, project_id, name, description, folder_id, language, script_code, trigger_type, trigger_config, input_bindings, dependencies, timeout_ms, is_enabled, created_at, updated_at
	`, params.ProjectID, params.Name, params.Description, params.FolderID, params.Language, params.ScriptCode, params.TriggerType, triggerConfigPayload, inputBindingsPayload, dependenciesPayload, params.TimeoutMS, params.UserID)

	record, scanErr := scanComputeUnit(row)
	if scanErr != nil {
		return nil, translateComputeWriteError("创建计算单元失败", scanErr)
	}
	if err := replaceComputeDatapointRefsTx(ctx, tx, record.ID, params.ProjectID, params.DatapointRefs); err != nil {
		return nil, err
	}
	if err := syncComputeOutputsTx(ctx, tx, record, params.UserID, params.Outputs); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交计算单元创建事务失败", err)
	}
	if err := r.hydrateComputeOutputs(ctx, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// ListUnits 按项目分页查询计算单元。
func (r *ComputeRepository) ListUnits(ctx context.Context, projectID string, filter ComputeUnitListFilter) ([]ComputeUnitRecord, int, error) {
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 20, 100)
	whereSQL, args := buildComputeUnitWhereClause(projectID, filter)

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_compute_units WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计计算单元列表失败", err)
	}

	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, name, description, folder_id, language, script_code, trigger_type, trigger_config, input_bindings, dependencies, timeout_ms, is_enabled, created_at, updated_at
        FROM data_compute_units
        WHERE `+whereSQL+`
        ORDER BY created_at DESC
        LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2)+`
    `, listArgs...)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询计算单元列表失败", err)
	}
	defer rows.Close()

	records := make([]ComputeUnitRecord, 0)
	for rows.Next() {
		record, scanErr := scanComputeUnit(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		if err := r.hydrateComputeOutputs(ctx, &record); err != nil {
			return nil, 0, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历计算单元列表失败", err)
	}
	return records, total, nil
}

// GetUnitByProjectAndID 按项目与单元 ID 读取单元定义。
func (r *ComputeRepository) GetUnitByProjectAndID(ctx context.Context, projectID, unitID string) (*ComputeUnitRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, name, description, folder_id, language, script_code, trigger_type, trigger_config, input_bindings, dependencies, timeout_ms, is_enabled, created_at, updated_at
		FROM data_compute_units
		WHERE project_id = $1 AND id = $2
	`, projectID, unitID)

	record, err := scanComputeUnit(row)
	if err != nil {
		return nil, err
	}
	if err := r.hydrateComputeOutputs(ctx, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateUnit 更新计算单元定义。
func (r *ComputeRepository) UpdateUnit(ctx context.Context, params UpdateComputeUnitParams) (*ComputeUnitRecord, error) {
	triggerConfigPayload, err := marshalComputeObject(params.TriggerConfig)
	if err != nil {
		return nil, err
	}
	inputBindingsPayload, err := marshalComputeObject(params.InputBindings)
	if err != nil {
		return nil, err
	}
	dependenciesPayload, err := marshalComputeArray(params.Dependencies)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启计算单元更新事务失败", err)
	}
	defer tx.Rollback(ctx)
	if err := lockComputeProject(ctx, tx, params.ProjectID); err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, `
        UPDATE data_compute_units
        SET name = $3,
            description = $4,
            folder_id = $5,
            language = $6,
            script_code = $7,
            trigger_type = $8,
            trigger_config = $9::jsonb,
            input_bindings = $10::jsonb,
            dependencies = $11::jsonb,
            timeout_ms = $12,
            is_enabled = $13,
            updated_by = $14,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, name, description, folder_id, language, script_code, trigger_type, trigger_config, input_bindings, dependencies, timeout_ms, is_enabled, created_at, updated_at
    `, params.ProjectID, params.ID, params.Name, params.Description, params.FolderID, params.Language, params.ScriptCode, params.TriggerType, triggerConfigPayload, inputBindingsPayload, dependenciesPayload, params.TimeoutMS, params.IsEnabled, params.UserID)

	record, err := scanComputeUnit(row)
	if err != nil {
		return nil, translateComputeWriteError("更新计算单元失败", err)
	}
	if err := replaceComputeDatapointRefsTx(ctx, tx, record.ID, params.ProjectID, params.DatapointRefs); err != nil {
		return nil, err
	}
	if err := syncComputeOutputsTx(ctx, tx, record, params.UserID, params.Outputs); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交计算单元更新事务失败", err)
	}
	if err := r.hydrateComputeOutputs(ctx, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func lockComputeProject(ctx context.Context, tx pgx.Tx, projectID string) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 20260825))`, projectID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "锁定工程计算配置失败", err)
	}
	return nil
}

// replaceComputeDatapointRefsTx 将 JSON 中的数据点引用投影为数据库可约束的稳定引用。
func replaceComputeDatapointRefsTx(ctx context.Context, tx pgx.Tx, unitID, projectID string, refs []ComputeDatapointRefParam) error {
	if _, err := tx.Exec(ctx, `DELETE FROM data_compute_unit_datapoint_refs WHERE compute_unit_id=$1`, unitID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "清理计算数据点引用失败", err)
	}
	for _, ref := range refs {
		tag, err := tx.Exec(ctx, `
			INSERT INTO data_compute_unit_datapoint_refs(project_id,compute_unit_id,datapoint_id,role,alias,sort_order)
			SELECT $1,$2,point.id,$4,$5,$6
			FROM data_points point
			WHERE point.project_id=$1 AND point.id=$3 AND point.status <> 'invalid'
		`, projectID, unitID, ref.DatapointID, ref.Role, ref.Alias, ref.SortOrder)
		if err != nil {
			return translateComputeWriteError("写入计算数据点引用失败", err)
		}
		if tag.RowsAffected() == 0 {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "计算引用的数据点不存在或已失效")
		}
	}
	return nil
}

// syncComputeOutputsTx 与计算单元主记录共用事务，避免单元保存成功但输出点同步失败。
func syncComputeOutputsTx(ctx context.Context, tx pgx.Tx, unit ComputeUnitRecord, userID string, outputs []ComputeOutputParam) error {
	type existingOutput struct {
		ID          string
		DatapointID string
	}
	existing := map[string]existingOutput{}
	rows, err := tx.Query(ctx, `
		SELECT output_key,id::text,datapoint_id::text
		FROM data_compute_unit_outputs
		WHERE project_id=$1 AND compute_unit_id=$2
		FOR UPDATE
	`, unit.ProjectID, unit.ID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "锁定计算输出失败", err)
	}
	for rows.Next() {
		var key string
		var item existingOutput
		if err := rows.Scan(&key, &item.ID, &item.DatapointID); err != nil {
			rows.Close()
			return err
		}
		existing[key] = item
	}
	rows.Close()

	kept := make(map[string]struct{}, len(outputs))
	for index, output := range outputs {
		key := strings.TrimSpace(output.OutputKey)
		if key == "" {
			key = strings.TrimSpace(output.Name)
		}
		if key == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "计算输出键不能为空")
		}
		if _, duplicated := kept[key]; duplicated {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "计算输出键不能重复")
		}
		kept[key] = struct{}{}
		current := existing[key]
		path := strings.TrimSpace(output.Path)
		if path == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "计算输出路径不能为空")
		}
		var conflictID string
		conflictErr := tx.QueryRow(ctx, `SELECT id::text FROM data_points WHERE project_id=$1 AND path=$2 LIMIT 1`, unit.ProjectID, path).Scan(&conflictID)
		if conflictErr != nil && !errors.Is(conflictErr, pgx.ErrNoRows) {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "检查计算输出路径失败", conflictErr)
		}
		if conflictErr == nil && conflictID != current.DatapointID {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "计算输出路径已被其他数据点占用")
		}
		nullPolicy := strings.TrimSpace(output.NullPolicy)
		if nullPolicy == "" {
			nullPolicy = "error"
		}
		if nullPolicy != "error" && nullPolicy != "skip" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "计算输出空值策略无效")
		}
		sourceConfig, marshalErr := json.Marshal(map[string]any{"computeUnitId": unit.ID, "outputKey": key})
		if marshalErr != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "编码计算输出来源失败", marshalErr)
		}
		datapointID := current.DatapointID
		if datapointID != "" {
			_, err = tx.Exec(ctx, `
				UPDATE data_points
				SET path=$3,name=$4,description=$5,source_config=$6::jsonb,data_type=$7,unit=$8,precision_num=$9,
				    refresh_mode='manual',status='active',updated_by=$10,updated_at=now()
				WHERE project_id=$1 AND id=$2
			`, unit.ProjectID, datapointID, path, output.Name, output.Description, string(sourceConfig), output.DataType, output.Unit, output.PrecisionNum, userID)
		} else {
			err = tx.QueryRow(ctx, `
				INSERT INTO data_points(project_id,path,name,description,source_type,source_id,source_config,data_type,unit,precision_num,tags,refresh_mode,status,created_by,updated_by)
				VALUES($1,$2,$3,$4,'calc.output',$5,$6::jsonb,$7,$8,$9,'[]'::jsonb,'manual','active',$10,$10)
				RETURNING id::text
			`, unit.ProjectID, path, output.Name, output.Description, unit.ID, string(sourceConfig), output.DataType, output.Unit, output.PrecisionNum, userID).Scan(&datapointID)
		}
		if err != nil {
			return translateComputeWriteError("同步计算输出数据点失败", err)
		}
		if current.ID != "" {
			_, err = tx.Exec(ctx, `
				UPDATE data_compute_unit_outputs
				SET path=$3,data_type=$4,unit=$5,precision_num=$6,null_policy=$7,sort_order=$8,updated_at=now()
				WHERE id=$1 AND datapoint_id=$2
			`, current.ID, datapointID, path, output.DataType, output.Unit, output.PrecisionNum, nullPolicy, index)
		} else {
			_, err = tx.Exec(ctx, `
				INSERT INTO data_compute_unit_outputs(project_id,compute_unit_id,datapoint_id,output_key,path,data_type,unit,precision_num,null_policy,sort_order)
				VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			`, unit.ProjectID, unit.ID, datapointID, key, path, output.DataType, output.Unit, output.PrecisionNum, nullPolicy, index)
		}
		if err != nil {
			return translateComputeWriteError("保存规范化计算输出失败", err)
		}
	}

	removedIDs := make([]string, 0)
	for key, item := range existing {
		if _, ok := kept[key]; !ok {
			removedIDs = append(removedIDs, item.DatapointID)
		}
	}
	if err := ensureNoDatapointBlockingUsagesTx(ctx, tx, unit.ProjectID, removedIDs); err != nil {
		return err
	}
	if len(removedIDs) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM data_compute_unit_outputs WHERE compute_unit_id=$1 AND datapoint_id=ANY($2::uuid[])`, unit.ID, removedIDs); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除计算输出定义失败", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE data_points SET status='invalid',updated_by=$3,updated_at=now() WHERE project_id=$1 AND id=ANY($2::uuid[])`, unit.ProjectID, removedIDs, userID); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "失效已删除计算输出失败", err)
		}
	}
	return nil
}

// UpdateUnitEnabled 更新计算单元启用状态。
func (r *ComputeRepository) UpdateUnitEnabled(ctx context.Context, projectID, unitID, userID string, enabled bool) (*ComputeUnitRecord, error) {
	row := r.pool.QueryRow(ctx, `
        UPDATE data_compute_units
        SET is_enabled = $3,
            updated_by = $4,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, name, description, folder_id, language, script_code, trigger_type, trigger_config, input_bindings, dependencies, timeout_ms, is_enabled, created_at, updated_at
    `, projectID, unitID, enabled, userID)
	record, err := scanComputeUnit(row)
	if err != nil {
		return nil, translateComputeWriteError("更新计算单元启用状态失败", err)
	}
	if err := r.hydrateComputeOutputs(ctx, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// DeleteUnitWithOutputs 在同一事务内删除计算单元并失效其输出点。
func (r *ComputeRepository) DeleteUnitWithOutputs(ctx context.Context, projectID, unitID, userID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启计算单元删除事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)
	if err := lockComputeProject(ctx, tx, projectID); err != nil {
		return err
	}
	if err := deleteComputeUnitsWithOutputsTx(ctx, tx, projectID, []string{unitID}, userID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交计算单元删除事务失败", err)
	}
	return nil
}

// ListFolders 查询项目下计算单元文件夹。
func (r *ComputeRepository) ListFolders(ctx context.Context, projectID string) ([]ComputeFolderRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, name, parent_id, created_at, updated_at
        FROM data_compute_folders
        WHERE project_id = $1
        ORDER BY created_at ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询计算文件夹失败", err)
	}
	defer rows.Close()

	records := make([]ComputeFolderRecord, 0)
	for rows.Next() {
		record, scanErr := scanComputeFolder(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历计算文件夹失败", err)
	}
	return records, nil
}

// ListFolderPage 按父目录分页读取直接子目录；搜索时在全树匹配并返回完整路径。
func (r *ComputeRepository) ListFolderPage(ctx context.Context, projectID string, filter ComputeFolderListFilter) ([]ComputeFolderPageRecord, int, error) {
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 20, 100)
	parentID := ""
	if filter.ParentID != nil {
		parentID = strings.TrimSpace(*filter.ParentID)
	}
	search := strings.TrimSpace(filter.Search)
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE folder_paths AS (
			SELECT f.id, f.project_id, f.name, f.parent_id, f.created_at, f.updated_at,
			       f.name::text AS full_path, ARRAY[f.id]::uuid[] AS ancestor_ids
			FROM data_compute_folders f
			WHERE f.project_id = $1 AND f.parent_id IS NULL
			UNION ALL
			SELECT child.id, child.project_id, child.name, child.parent_id, child.created_at, child.updated_at,
			       (parent.full_path || ' / ' || child.name)::text,
			       parent.ancestor_ids || child.id
			FROM data_compute_folders child
			JOIN folder_paths parent ON parent.id = child.parent_id
			WHERE child.project_id = $1
		), matched AS (
			SELECT fp.*,
			       EXISTS (
				   SELECT 1 FROM data_compute_folders child
				   WHERE child.project_id = fp.project_id AND child.parent_id = fp.id
			   ) AS has_children,
			   (SELECT COUNT(*)::int
			    FROM data_compute_units unit
			    JOIN folder_paths descendant ON descendant.id = unit.folder_id
			    WHERE unit.project_id = fp.project_id AND fp.id = ANY(descendant.ancestor_ids)) AS unit_count
			FROM folder_paths fp
			WHERE (
				($3::text <> '' AND fp.full_path ILIKE ('%' || $3::text || '%'))
				OR
				($3::text = '' AND (($2::text = '' AND fp.parent_id IS NULL) OR ($2::text <> '' AND fp.parent_id = $2::text::uuid)))
			)
		)
		SELECT id, project_id, name, parent_id, created_at, updated_at,
		       full_path, has_children, unit_count, COUNT(*) OVER()::int
		FROM matched
		ORDER BY lower(name), id
		LIMIT $4 OFFSET $5
	`, projectID, parentID, search, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询计算文件夹分页失败", err)
	}
	defer rows.Close()

	records := make([]ComputeFolderPageRecord, 0)
	total := 0
	for rows.Next() {
		var record ComputeFolderPageRecord
		if err := rows.Scan(
			&record.ID,
			&record.ProjectID,
			&record.Name,
			&record.ParentID,
			&record.CreatedAt,
			&record.UpdatedAt,
			&record.Path,
			&record.HasChildren,
			&record.UnitCount,
			&total,
		); err != nil {
			return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取计算文件夹分页失败", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历计算文件夹分页失败", err)
	}
	return records, total, nil
}

// CreateFolder 创建计算单元文件夹。
func (r *ComputeRepository) CreateFolder(ctx context.Context, params CreateComputeFolderParams) (*ComputeFolderRecord, error) {
	row := r.pool.QueryRow(ctx, `
        INSERT INTO data_compute_folders (
            project_id,
            name,
            parent_id,
            created_by,
            updated_by
        )
        VALUES ($1, $2, $3, $4, $4)
        RETURNING id, project_id, name, parent_id, created_at, updated_at
    `, params.ProjectID, params.Name, params.ParentID, params.UserID)

	record, err := scanComputeFolder(row)
	if err != nil {
		return nil, translateComputeFolderWriteError("创建计算文件夹失败", err)
	}
	return &record, nil
}

// UpdateFolder 更新计算单元文件夹。
func (r *ComputeRepository) UpdateFolder(ctx context.Context, projectID, folderID, userID, name string, parentID *string) (*ComputeFolderRecord, error) {
	row := r.pool.QueryRow(ctx, `
        UPDATE data_compute_folders
        SET name = $3,
            parent_id = $4,
            updated_by = $5,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, name, parent_id, created_at, updated_at
    `, projectID, folderID, name, parentID, userID)

	record, err := scanComputeFolder(row)
	if err != nil {
		return nil, translateComputeFolderWriteError("更新计算文件夹失败", err)
	}
	return &record, nil
}

// ListUnitIDsByFolderTree 查询文件夹及其子文件夹内的计算单元 ID。
func (r *ComputeRepository) ListUnitIDsByFolderTree(ctx context.Context, projectID, folderID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
        WITH RECURSIVE target_folders AS (
            SELECT id
            FROM data_compute_folders
            WHERE project_id = $1 AND id = $2
            UNION ALL
            SELECT child.id
            FROM data_compute_folders child
            INNER JOIN target_folders parent ON child.parent_id = parent.id
            WHERE child.project_id = $1
        )
        SELECT id
        FROM data_compute_units
        WHERE project_id = $1 AND folder_id IN (SELECT id FROM target_folders)
    `, projectID, folderID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询计算文件夹内单元失败", err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取计算文件夹内单元失败", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历计算文件夹内单元失败", err)
	}
	return ids, nil
}

// DeleteFolderTreeWithOutputs 在同一事务内删除目录树、计算单元并失效全部输出点。
func (r *ComputeRepository) DeleteFolderTreeWithOutputs(ctx context.Context, projectID, folderID, userID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启计算文件夹删除事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)
	if err := lockComputeProject(ctx, tx, projectID); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `
		WITH RECURSIVE target_folders AS (
			SELECT id FROM data_compute_folders WHERE project_id=$1 AND id=$2
			UNION ALL
			SELECT child.id FROM data_compute_folders child
			JOIN target_folders parent ON child.parent_id=parent.id
			WHERE child.project_id=$1
		)
		SELECT id::text FROM data_compute_units
		WHERE project_id=$1 AND folder_id IN (SELECT id FROM target_folders)
		FOR UPDATE
	`, projectID, folderID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "锁定计算文件夹内单元失败", err)
	}
	unitIDs := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取计算文件夹内单元失败", err)
		}
		unitIDs = append(unitIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历计算文件夹内单元失败", err)
	}
	if err := deleteComputeUnitsWithOutputsTx(ctx, tx, projectID, unitIDs, userID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM data_compute_folders WHERE project_id=$1 AND id=$2`, projectID, folderID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除计算文件夹失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "计算文件夹不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交计算文件夹删除事务失败", err)
	}
	return nil
}

func deleteComputeUnitsWithOutputsTx(ctx context.Context, tx pgx.Tx, projectID string, unitIDs []string, userID string) error {
	if len(unitIDs) == 0 {
		return nil
	}
	rows, err := tx.Query(ctx, `
		SELECT id::text FROM data_points
		WHERE project_id=$1 AND source_type='calc.output' AND source_id=ANY($2::uuid[])
		FOR UPDATE
	`, projectID, unitIDs)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "锁定计算输出点失败", err)
	}
	pointIDs := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取计算输出点失败", err)
		}
		pointIDs = append(pointIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历计算输出点失败", err)
	}
	if len(pointIDs) > 0 {
		var blockers int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM (
				SELECT item.id FROM data_alarm_items item WHERE item.project_id=$1 AND item.datapoint_id=ANY($2::uuid[])
				UNION ALL
				SELECT input.alarm_item_id FROM data_alarm_item_inputs input JOIN data_alarm_items item ON item.id=input.alarm_item_id WHERE item.project_id=$1 AND input.datapoint_id=ANY($2::uuid[])
				UNION ALL
				SELECT ref.compute_unit_id FROM data_compute_unit_datapoint_refs ref WHERE ref.project_id=$1 AND ref.datapoint_id=ANY($2::uuid[]) AND NOT (ref.compute_unit_id=ANY($3::uuid[]))
			) usage
		`, projectID, pointIDs, unitIDs).Scan(&blockers); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "检查计算输出引用失败", err)
		}
		if blockers > 0 {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, fmt.Sprintf("计算输出仍存在 %d 项报警或其他计算引用，请先解除引用", blockers))
		}
		if _, err := tx.Exec(ctx, `UPDATE data_points SET status='invalid',updated_by=$3,updated_at=now() WHERE project_id=$1 AND id=ANY($2::uuid[])`, projectID, pointIDs, userID); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "标记计算输出点失效失败", err)
		}
	}
	tag, err := tx.Exec(ctx, `DELETE FROM data_compute_units WHERE project_id=$1 AND id=ANY($2::uuid[])`, projectID, unitIDs)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除计算单元失败", err)
	}
	if tag.RowsAffected() != int64(len(unitIDs)) {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "部分计算单元不存在或不属于当前工程")
	}
	return nil
}

// SaveRun 写入计算执行记录（run/debug）。
func (r *ComputeRepository) SaveRun(ctx context.Context, params SaveComputeRunParams) error {
	outputPayload, err := marshalComputeAny(params.Output)
	if err != nil {
		return err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启计算运行结果事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)
	for _, output := range params.OutputValues {
		tag, updateErr := tx.Exec(ctx, `
			UPDATE data_points point SET default_value=$4,updated_at=now()
			FROM data_compute_unit_outputs output
			WHERE point.project_id=$1 AND output.compute_unit_id=$2 AND output.output_key=$3
			  AND output.datapoint_id=point.id AND point.status='active'
		`, params.ProjectID, params.ComputeUnitID, output.Name, output.Value)
		if updateErr != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入计算输出数据点失败", updateErr)
		}
		if tag.RowsAffected() != 1 {
			return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "计算输出数据点不存在、失效或配置重复")
		}
	}
	_, execErr := tx.Exec(ctx, `
		INSERT INTO data_compute_runs (
			project_id,
			compute_unit_id,
			trigger_mode,
			status,
			duration_ms,
			output,
			error_message,
			started_at,
			finished_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9)
	`, params.ProjectID, params.ComputeUnitID, params.TriggerMode, params.Status, params.DurationMS, outputPayload, params.ErrorMessage, params.StartedAt, params.FinishedAt)
	if execErr != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入计算运行记录失败", execErr)
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交计算运行结果事务失败", err)
	}
	return nil
}

// ListRuns 按项目分页查询计算运行记录。
func (r *ComputeRepository) ListRuns(ctx context.Context, projectID string, filter ComputeRunListFilter) ([]ComputeRunRecord, int, error) {
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 20, 100)
	whereSQL, args := buildComputeRunWhereClause(projectID, filter)

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_compute_runs WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计计算运行记录失败", err)
	}

	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, compute_unit_id, trigger_mode, status, duration_ms, output, error_message, started_at, finished_at, created_at
        FROM data_compute_runs
        WHERE `+whereSQL+`
        ORDER BY created_at DESC, id DESC
        LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2)+`
    `, listArgs...)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询计算运行记录失败", err)
	}
	defer rows.Close()

	records := make([]ComputeRunRecord, 0)
	for rows.Next() {
		record, scanErr := scanComputeRun(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历计算运行记录失败", err)
	}
	return records, total, nil
}

type computeScannable interface {
	Scan(dest ...any) error
}

func scanComputeUnit(row computeScannable) (ComputeUnitRecord, error) {
	var (
		record             ComputeUnitRecord
		folderID           sql.NullString
		triggerConfigBytes []byte
		inputBindingsBytes []byte
		dependenciesBytes  []byte
	)

	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.Name,
		&record.Description,
		&folderID,
		&record.Language,
		&record.ScriptCode,
		&record.TriggerType,
		&triggerConfigBytes,
		&inputBindingsBytes,
		&dependenciesBytes,
		&record.TimeoutMS,
		&record.IsEnabled,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return ComputeUnitRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "计算单元不存在")
		}
		return ComputeUnitRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取计算单元失败", err)
	}
	record.FolderID = nullStringToPtr(folderID)

	var decodeErr error
	record.TriggerConfig, decodeErr = unmarshalComputeObject(triggerConfigBytes)
	if decodeErr != nil {
		return ComputeUnitRecord{}, decodeErr
	}
	record.InputBindings, decodeErr = unmarshalComputeObject(inputBindingsBytes)
	if decodeErr != nil {
		return ComputeUnitRecord{}, decodeErr
	}
	record.Dependencies, decodeErr = unmarshalComputeArray(dependenciesBytes)
	if decodeErr != nil {
		return ComputeUnitRecord{}, decodeErr
	}

	return record, nil
}

func scanComputeFolder(row computeScannable) (ComputeFolderRecord, error) {
	var (
		record   ComputeFolderRecord
		parentID sql.NullString
	)
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.Name,
		&parentID,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ComputeFolderRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "计算文件夹不存在")
		}
		return ComputeFolderRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取计算文件夹失败", err)
	}
	record.ParentID = nullStringToPtr(parentID)
	return record, nil
}

func scanComputeRun(row computeScannable) (ComputeRunRecord, error) {
	var (
		record       ComputeRunRecord
		outputBytes  []byte
		errorMessage sql.NullString
	)
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ComputeUnitID,
		&record.TriggerMode,
		&record.Status,
		&record.DurationMS,
		&outputBytes,
		&errorMessage,
		&record.StartedAt,
		&record.FinishedAt,
		&record.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ComputeRunRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "计算运行记录不存在")
		}
		return ComputeRunRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取计算运行记录失败", err)
	}
	record.ErrorMessage = nullStringToPtr(errorMessage)
	if len(outputBytes) > 0 {
		if err := json.Unmarshal(outputBytes, &record.Output); err != nil {
			return ComputeRunRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析计算输出失败", err)
		}
	}
	return record, nil
}

func buildComputeUnitWhereClause(projectID string, filter ComputeUnitListFilter) (string, []any) {
	clauses := []string{"project_id = $1"}
	args := []any{projectID}

	if strings.TrimSpace(filter.Language) != "" {
		args = append(args, strings.TrimSpace(filter.Language))
		clauses = append(clauses, fmt.Sprintf("language = $%d", len(args)))
	}
	if filter.Enabled != nil {
		args = append(args, *filter.Enabled)
		clauses = append(clauses, fmt.Sprintf("is_enabled = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Search) != "" {
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		index := len(args)
		clauses = append(clauses, fmt.Sprintf("name ILIKE $%d", index))
	}
	if filter.FolderID != nil {
		args = append(args, strings.TrimSpace(*filter.FolderID))
		clauses = append(clauses, fmt.Sprintf("folder_id = $%d::uuid", len(args)))
	}
	return strings.Join(clauses, " AND "), args
}

func buildComputeRunWhereClause(projectID string, filter ComputeRunListFilter) (string, []any) {
	clauses := []string{"project_id = $1"}
	args := []any{projectID}
	if strings.TrimSpace(filter.UnitID) != "" {
		args = append(args, strings.TrimSpace(filter.UnitID))
		clauses = append(clauses, fmt.Sprintf("compute_unit_id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Status) != "" {
		args = append(args, strings.TrimSpace(filter.Status))
		clauses = append(clauses, fmt.Sprintf("status = $%d", len(args)))
	}
	if strings.TrimSpace(filter.TriggerMode) != "" {
		args = append(args, strings.TrimSpace(filter.TriggerMode))
		clauses = append(clauses, fmt.Sprintf("trigger_mode = $%d", len(args)))
	}
	return strings.Join(clauses, " AND "), args
}

func marshalComputeObject(input map[string]any) (string, error) {
	if input == nil {
		input = map[string]any{}
	}

	payload, err := json.Marshal(input)
	if err != nil {
		return "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "compute JSON 对象格式无效", err)
	}
	return string(payload), nil
}

func marshalComputeArray(input []any) (string, error) {
	if input == nil {
		input = []any{}
	}

	payload, err := json.Marshal(input)
	if err != nil {
		return "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "compute JSON 数组格式无效", err)
	}
	return string(payload), nil
}

func unmarshalComputeObject(payload []byte) (map[string]any, error) {
	if len(payload) == 0 {
		return map[string]any{}, nil
	}

	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析 compute JSON 对象失败", err)
	}
	if result == nil {
		result = map[string]any{}
	}
	return result, nil
}

func unmarshalComputeArray(payload []byte) ([]any, error) {
	if len(payload) == 0 {
		return []any{}, nil
	}

	var result []any
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析 compute JSON 数组失败", err)
	}
	if result == nil {
		result = []any{}
	}
	return result, nil
}

func marshalComputeAny(input any) (string, error) {
	if input == nil {
		return "null", nil
	}

	payload, err := json.Marshal(input)
	if err != nil {
		return "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "compute 输出格式无效", err)
	}
	return string(payload), nil
}

func translateComputeWriteError(message string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "计算单元名称已存在")
		case "23503":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "关联计算资源不存在")
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "计算单元不存在")
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}

func translateComputeFolderWriteError(message string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "同级计算文件夹名称已存在")
		case "23503":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "上级计算文件夹不存在")
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "计算文件夹不存在")
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}
