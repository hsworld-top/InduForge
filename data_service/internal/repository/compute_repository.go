package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// ComputeUnitRecord 表示 data_compute_units 的仓储层投影。
type ComputeUnitRecord struct {
	ID            string
	ProjectID     string
	Name          string
	Language      string
	ScriptCode    string
	TriggerType   string
	TriggerConfig map[string]any
	InputBindings map[string]any
	OutputBinding map[string]any
	TimeoutMS     int
	IsEnabled     bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CreateComputeUnitParams 描述创建计算单元时的落库参数。
type CreateComputeUnitParams struct {
	ProjectID     string
	UserID        string
	Name          string
	Language      string
	ScriptCode    string
	TriggerType   string
	TriggerConfig map[string]any
	InputBindings map[string]any
	OutputBinding map[string]any
	TimeoutMS     int
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

	row := r.pool.QueryRow(ctx, `
		INSERT INTO data_compute_units (
			project_id,
			name,
			language,
			script_code,
			trigger_type,
			trigger_config,
			input_bindings,
			output_bindings,
			timeout_ms,
			created_by,
			updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, $8::jsonb, $9, $10, $10)
		RETURNING id, project_id, name, language, script_code, trigger_type, trigger_config, input_bindings, output_bindings, timeout_ms, is_enabled, created_at, updated_at
	`, params.ProjectID, params.Name, params.Language, params.ScriptCode, params.TriggerType, triggerConfigPayload, inputBindingsPayload, outputBindingsPayload, params.TimeoutMS, params.UserID)

	record, scanErr := scanComputeUnit(row)
	if scanErr != nil {
		return nil, scanErr
	}
	return &record, nil
}

// GetUnitByProjectAndID 按项目与单元 ID 读取单元定义。
func (r *ComputeRepository) GetUnitByProjectAndID(ctx context.Context, projectID, unitID string) (*ComputeUnitRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, name, language, script_code, trigger_type, trigger_config, input_bindings, output_bindings, timeout_ms, is_enabled, created_at, updated_at
		FROM data_compute_units
		WHERE project_id = $1 AND id = $2
	`, projectID, unitID)

	record, err := scanComputeUnit(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
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

type computeScannable interface {
	Scan(dest ...any) error
}

func scanComputeUnit(row computeScannable) (ComputeUnitRecord, error) {
	var (
		record              ComputeUnitRecord
		triggerConfigBytes  []byte
		inputBindingsBytes  []byte
		outputBindingsBytes []byte
	)

	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.Name,
		&record.Language,
		&record.ScriptCode,
		&record.TriggerType,
		&triggerConfigBytes,
		&inputBindingsBytes,
		&outputBindingsBytes,
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

	return record, nil
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
