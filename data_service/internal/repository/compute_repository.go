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
	OutputBinding map[string]any
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
	OutputBinding map[string]any
	Dependencies  []any
	TimeoutMS     int
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
	OutputBinding map[string]any
	Dependencies  []any
	TimeoutMS     int
	IsEnabled     bool
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
	outputBindingsPayload, err := marshalComputeObject(params.OutputBinding)
	if err != nil {
		return nil, err
	}
	dependenciesPayload, err := marshalComputeArray(params.Dependencies)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, `
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
			output_bindings,
			dependencies,
			timeout_ms,
			created_by,
			updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb, $11::jsonb, $12, $13, $13)
		RETURNING id, project_id, name, description, folder_id, language, script_code, trigger_type, trigger_config, input_bindings, output_bindings, dependencies, timeout_ms, is_enabled, created_at, updated_at
	`, params.ProjectID, params.Name, params.Description, params.FolderID, params.Language, params.ScriptCode, params.TriggerType, triggerConfigPayload, inputBindingsPayload, outputBindingsPayload, dependenciesPayload, params.TimeoutMS, params.UserID)

	record, scanErr := scanComputeUnit(row)
	if scanErr != nil {
		return nil, translateComputeWriteError("创建计算单元失败", scanErr)
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
        SELECT id, project_id, name, description, folder_id, language, script_code, trigger_type, trigger_config, input_bindings, output_bindings, dependencies, timeout_ms, is_enabled, created_at, updated_at
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
		SELECT id, project_id, name, description, folder_id, language, script_code, trigger_type, trigger_config, input_bindings, output_bindings, dependencies, timeout_ms, is_enabled, created_at, updated_at
		FROM data_compute_units
		WHERE project_id = $1 AND id = $2
	`, projectID, unitID)

	record, err := scanComputeUnit(row)
	if err != nil {
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
	outputBindingsPayload, err := marshalComputeObject(params.OutputBinding)
	if err != nil {
		return nil, err
	}
	dependenciesPayload, err := marshalComputeArray(params.Dependencies)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, `
        UPDATE data_compute_units
        SET name = $3,
            description = $4,
            folder_id = $5,
            language = $6,
            script_code = $7,
            trigger_type = $8,
            trigger_config = $9::jsonb,
            input_bindings = $10::jsonb,
            output_bindings = $11::jsonb,
            dependencies = $12::jsonb,
            timeout_ms = $13,
            is_enabled = $14,
            updated_by = $15,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, name, description, folder_id, language, script_code, trigger_type, trigger_config, input_bindings, output_bindings, dependencies, timeout_ms, is_enabled, created_at, updated_at
    `, params.ProjectID, params.ID, params.Name, params.Description, params.FolderID, params.Language, params.ScriptCode, params.TriggerType, triggerConfigPayload, inputBindingsPayload, outputBindingsPayload, dependenciesPayload, params.TimeoutMS, params.IsEnabled, params.UserID)

	record, err := scanComputeUnit(row)
	if err != nil {
		return nil, translateComputeWriteError("更新计算单元失败", err)
	}
	return &record, nil
}

// UpdateUnitEnabled 更新计算单元启用状态。
func (r *ComputeRepository) UpdateUnitEnabled(ctx context.Context, projectID, unitID, userID string, enabled bool) (*ComputeUnitRecord, error) {
	row := r.pool.QueryRow(ctx, `
        UPDATE data_compute_units
        SET is_enabled = $3,
            updated_by = $4,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, name, description, folder_id, language, script_code, trigger_type, trigger_config, input_bindings, output_bindings, dependencies, timeout_ms, is_enabled, created_at, updated_at
    `, projectID, unitID, enabled, userID)
	record, err := scanComputeUnit(row)
	if err != nil {
		return nil, translateComputeWriteError("更新计算单元启用状态失败", err)
	}
	return &record, nil
}

// DeleteUnit 按项目删除计算单元。
func (r *ComputeRepository) DeleteUnit(ctx context.Context, projectID, unitID string) error {
	commandTag, err := r.pool.Exec(ctx, `
        DELETE FROM data_compute_units
        WHERE project_id = $1 AND id = $2
    `, projectID, unitID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除计算单元失败", err)
	}
	if commandTag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "计算单元不存在")
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

// DeleteUnits 按项目批量删除计算单元。
func (r *ComputeRepository) DeleteUnits(ctx context.Context, projectID string, unitIDs []string) error {
	if len(unitIDs) == 0 {
		return nil
	}
	if _, err := r.pool.Exec(ctx, `
        DELETE FROM data_compute_units
        WHERE project_id = $1 AND id = ANY($2::uuid[])
    `, projectID, unitIDs); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "批量删除计算单元失败", err)
	}
	return nil
}

// DeleteFolder 删除计算单元文件夹，子文件夹级联删除。
func (r *ComputeRepository) DeleteFolder(ctx context.Context, projectID, folderID string) error {
	commandTag, err := r.pool.Exec(ctx, `
        DELETE FROM data_compute_folders
        WHERE project_id = $1 AND id = $2
    `, projectID, folderID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除计算文件夹失败", err)
	}
	if commandTag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "计算文件夹不存在")
	}
	return nil
}

// SaveRun 写入计算执行记录（run/debug）。
func (r *ComputeRepository) SaveRun(ctx context.Context, params SaveComputeRunParams) error {
	outputPayload, err := marshalComputeAny(params.Output)
	if err != nil {
		return err
	}

	_, execErr := r.pool.Exec(ctx, `
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
		record              ComputeUnitRecord
		folderID            sql.NullString
		triggerConfigBytes  []byte
		inputBindingsBytes  []byte
		outputBindingsBytes []byte
		dependenciesBytes   []byte
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
		&outputBindingsBytes,
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
	record.OutputBinding, decodeErr = unmarshalComputeObject(outputBindingsBytes)
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
