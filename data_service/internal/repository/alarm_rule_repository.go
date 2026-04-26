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

// AlarmRuleRecord 表示 data_alarm_rules 的仓储层投影。
type AlarmRuleRecord struct {
	ID                string
	ProjectID         string
	Name              string
	Description       *string
	TargetDataPointID *string
	TargetPath        string
	RuleType          string
	Condition         map[string]any
	Severity          string
	Hysteresis        *float64
	SampleWindowMS    *int
	Contract          map[string]any
	IsEnabled         bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type AlarmRuleListFilter struct {
	TargetPath string
	RuleType   string
	Severity   string
	Enabled    *bool
	Search     string
	Page       int
	PageSize   int
}

type CreateAlarmRuleParams struct {
	ProjectID         string
	UserID            string
	Name              string
	Description       *string
	TargetDataPointID *string
	TargetPath        string
	RuleType          string
	Condition         map[string]any
	Severity          string
	Hysteresis        *float64
	SampleWindowMS    *int
	Contract          map[string]any
	IsEnabled         bool
}

type UpdateAlarmRuleParams struct {
	ID                string
	ProjectID         string
	UserID            string
	Name              string
	Description       *string
	TargetDataPointID *string
	TargetPath        string
	RuleType          string
	Condition         map[string]any
	Severity          string
	Hysteresis        *float64
	SampleWindowMS    *int
	Contract          map[string]any
	IsEnabled         bool
}

// AlarmRuleRepository 封装报警规则的参数化 SQL 访问。
type AlarmRuleRepository struct {
	pool *pgxpool.Pool
}

func NewAlarmRuleRepository(pool *pgxpool.Pool) *AlarmRuleRepository {
	return &AlarmRuleRepository{pool: pool}
}

func (r *AlarmRuleRepository) ListByProject(ctx context.Context, projectID string, filter AlarmRuleListFilter) ([]AlarmRuleRecord, int, error) {
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 20, 100)
	whereSQL, args := buildAlarmRuleWhereClause(projectID, filter)

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_alarm_rules WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计报警规则失败", err)
	}

	selectArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, name, description, target_datapoint_id, target_path, rule_type, condition,
               severity, hysteresis, sample_window_ms, contract, is_enabled, created_at, updated_at
        FROM data_alarm_rules
        WHERE `+whereSQL+`
        ORDER BY updated_at DESC, created_at DESC
        LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2)+`
    `, selectArgs...)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询报警规则失败", err)
	}
	defer rows.Close()

	records := make([]AlarmRuleRecord, 0)
	for rows.Next() {
		record, scanErr := scanAlarmRule(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历报警规则失败", err)
	}
	return records, total, nil
}

func (r *AlarmRuleRepository) GetByProjectAndID(ctx context.Context, projectID, id string) (*AlarmRuleRecord, error) {
	record, err := scanAlarmRule(r.pool.QueryRow(ctx, `
        SELECT id, project_id, name, description, target_datapoint_id, target_path, rule_type, condition,
               severity, hysteresis, sample_window_ms, contract, is_enabled, created_at, updated_at
        FROM data_alarm_rules
        WHERE project_id = $1 AND id = $2
    `, projectID, id))
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *AlarmRuleRepository) Create(ctx context.Context, params CreateAlarmRuleParams) (*AlarmRuleRecord, error) {
	conditionPayload, err := marshalJSONObject(params.Condition)
	if err != nil {
		return nil, err
	}
	contractPayload, err := marshalJSONObject(params.Contract)
	if err != nil {
		return nil, err
	}
	record, err := scanAlarmRule(r.pool.QueryRow(ctx, `
        INSERT INTO data_alarm_rules (
            project_id, name, description, target_datapoint_id, target_path, rule_type, condition,
            severity, hysteresis, sample_window_ms, contract, is_enabled, created_by, updated_by
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11::jsonb, $12, $13, $13)
        RETURNING id, project_id, name, description, target_datapoint_id, target_path, rule_type, condition,
                  severity, hysteresis, sample_window_ms, contract, is_enabled, created_at, updated_at
    `, params.ProjectID, params.Name, params.Description, params.TargetDataPointID, params.TargetPath, params.RuleType,
		string(conditionPayload), params.Severity, params.Hysteresis, params.SampleWindowMS, string(contractPayload), params.IsEnabled, params.UserID))
	if err != nil {
		return nil, translateAlarmRuleWriteError(err)
	}
	return &record, nil
}

func (r *AlarmRuleRepository) Update(ctx context.Context, params UpdateAlarmRuleParams) (*AlarmRuleRecord, error) {
	conditionPayload, err := marshalJSONObject(params.Condition)
	if err != nil {
		return nil, err
	}
	contractPayload, err := marshalJSONObject(params.Contract)
	if err != nil {
		return nil, err
	}
	record, err := scanAlarmRule(r.pool.QueryRow(ctx, `
        UPDATE data_alarm_rules
        SET name = $3,
            description = $4,
            target_datapoint_id = $5,
            target_path = $6,
            rule_type = $7,
            condition = $8::jsonb,
            severity = $9,
            hysteresis = $10,
            sample_window_ms = $11,
            contract = $12::jsonb,
            is_enabled = $13,
            updated_by = $14,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, name, description, target_datapoint_id, target_path, rule_type, condition,
                  severity, hysteresis, sample_window_ms, contract, is_enabled, created_at, updated_at
    `, params.ProjectID, params.ID, params.Name, params.Description, params.TargetDataPointID, params.TargetPath, params.RuleType,
		string(conditionPayload), params.Severity, params.Hysteresis, params.SampleWindowMS, string(contractPayload), params.IsEnabled, params.UserID))
	if err != nil {
		return nil, translateAlarmRuleWriteError(err)
	}
	return &record, nil
}

func (r *AlarmRuleRepository) Delete(ctx context.Context, projectID, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM data_alarm_rules WHERE project_id = $1 AND id = $2`, projectID, id)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除报警规则失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "报警规则不存在")
	}
	return nil
}

func scanAlarmRule(row pgx.Row) (AlarmRuleRecord, error) {
	var record AlarmRuleRecord
	var description sql.NullString
	var targetDataPointID sql.NullString
	var conditionPayload []byte
	var contractPayload []byte
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.Name,
		&description,
		&targetDataPointID,
		&record.TargetPath,
		&record.RuleType,
		&conditionPayload,
		&record.Severity,
		&record.Hysteresis,
		&record.SampleWindowMS,
		&contractPayload,
		&record.IsEnabled,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AlarmRuleRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "报警规则不存在")
		}
		return AlarmRuleRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取报警规则失败", err)
	}
	record.Description = nullStringToPtr(description)
	record.TargetDataPointID = nullStringToPtr(targetDataPointID)
	condition, err := unmarshalAlarmJSONObject(conditionPayload)
	if err != nil {
		return AlarmRuleRecord{}, err
	}
	contract, err := unmarshalAlarmJSONObject(contractPayload)
	if err != nil {
		return AlarmRuleRecord{}, err
	}
	record.Condition = condition
	record.Contract = contract
	return record, nil
}

func buildAlarmRuleWhereClause(projectID string, filter AlarmRuleListFilter) (string, []any) {
	conditions := []string{"project_id = $1"}
	args := []any{projectID}
	if filter.TargetPath = strings.TrimSpace(filter.TargetPath); filter.TargetPath != "" {
		args = append(args, filter.TargetPath)
		conditions = append(conditions, fmt.Sprintf("target_path = $%d", len(args)))
	}
	if filter.RuleType = strings.TrimSpace(filter.RuleType); filter.RuleType != "" {
		args = append(args, filter.RuleType)
		conditions = append(conditions, fmt.Sprintf("rule_type = $%d", len(args)))
	}
	if filter.Severity = strings.TrimSpace(filter.Severity); filter.Severity != "" {
		args = append(args, filter.Severity)
		conditions = append(conditions, fmt.Sprintf("severity = $%d", len(args)))
	}
	if filter.Enabled != nil {
		args = append(args, *filter.Enabled)
		conditions = append(conditions, fmt.Sprintf("is_enabled = $%d", len(args)))
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+search+"%")
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR target_path ILIKE $%d)", len(args), len(args)))
	}
	return strings.Join(conditions, " AND "), args
}

func unmarshalAlarmJSONObject(payload []byte) (map[string]any, error) {
	if len(payload) == 0 {
		return map[string]any{}, nil
	}
	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析报警规则 JSON 失败", err)
	}
	if result == nil {
		return map[string]any{}, nil
	}
	return result, nil
}

func translateAlarmRuleWriteError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "报警规则名称已存在", err)
		}
		if pgErr.Code == "23503" {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "目标数据点不存在", err)
		}
		if pgErr.Code == "23514" {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "报警规则字段不符合约束", err)
		}
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入报警规则失败", err)
}
