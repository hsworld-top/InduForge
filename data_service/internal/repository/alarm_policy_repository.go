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

// AlarmPolicyGroupRecord 表示报警策略分组的仓储层投影。
type AlarmPolicyGroupRecord struct {
	ID          string
	ProjectID   string
	Name        string
	ParentID    *string
	Description *string
	IsEnabled   bool
	SortOrder   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// AlarmPolicyRecord 表示报警策略的仓储层投影。
type AlarmPolicyRecord struct {
	ID                string
	ProjectID         string
	GroupID           *string
	GroupName         *string
	GroupEnabled      *bool
	Name              string
	Description       *string
	Mode              string
	Targets           []map[string]any
	Inputs            []map[string]any
	DerivedExpression string
	Conditions        []map[string]any
	Suppression       map[string]any
	MessageTemplate   string
	IsEnabled         bool
	Contract          map[string]any
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type AlarmPolicyListFilter struct {
	Search        string
	GroupID       *string
	Enabled       *bool
	Severity      string
	ConditionType string
	TargetPath    string
	Mode          string
	Page          int
	PageSize      int
}

type CreateAlarmPolicyGroupParams struct {
	ProjectID   string
	UserID      string
	Name        string
	ParentID    *string
	Description *string
	IsEnabled   bool
	SortOrder   int
}

type UpdateAlarmPolicyGroupParams struct {
	ID          string
	ProjectID   string
	UserID      string
	Name        string
	ParentID    *string
	Description *string
	IsEnabled   bool
	SortOrder   int
}

type CreateAlarmPolicyParams struct {
	ProjectID         string
	UserID            string
	GroupID           *string
	Name              string
	Description       *string
	Mode              string
	Targets           []map[string]any
	Inputs            []map[string]any
	DerivedExpression string
	Conditions        []map[string]any
	Suppression       map[string]any
	MessageTemplate   string
	IsEnabled         bool
	Contract          map[string]any
}

type UpdateAlarmPolicyParams struct {
	ID                string
	ProjectID         string
	UserID            string
	GroupID           *string
	Name              string
	Description       *string
	Mode              string
	Targets           []map[string]any
	Inputs            []map[string]any
	DerivedExpression string
	Conditions        []map[string]any
	Suppression       map[string]any
	MessageTemplate   string
	IsEnabled         bool
	Contract          map[string]any
}

// AlarmPolicyRepository 封装报警策略的参数化 SQL 访问。
type AlarmPolicyRepository struct {
	pool *pgxpool.Pool
}

func NewAlarmPolicyRepository(pool *pgxpool.Pool) *AlarmPolicyRepository {
	return &AlarmPolicyRepository{pool: pool}
}

func (r *AlarmPolicyRepository) ListGroups(ctx context.Context, projectID string) ([]AlarmPolicyGroupRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, name, parent_id, description, is_enabled, sort_order, created_at, updated_at
        FROM data_alarm_policy_groups
        WHERE project_id = $1
        ORDER BY sort_order ASC, created_at ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询报警分组失败", err)
	}
	defer rows.Close()

	records := make([]AlarmPolicyGroupRecord, 0)
	for rows.Next() {
		record, scanErr := scanAlarmPolicyGroup(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历报警分组失败", err)
	}
	return records, nil
}

func (r *AlarmPolicyRepository) GetGroupByProjectAndID(ctx context.Context, projectID, id string) (*AlarmPolicyGroupRecord, error) {
	record, err := scanAlarmPolicyGroup(r.pool.QueryRow(ctx, `
        SELECT id, project_id, name, parent_id, description, is_enabled, sort_order, created_at, updated_at
        FROM data_alarm_policy_groups
        WHERE project_id = $1 AND id = $2
    `, projectID, id))
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *AlarmPolicyRepository) CreateGroup(ctx context.Context, params CreateAlarmPolicyGroupParams) (*AlarmPolicyGroupRecord, error) {
	record, err := scanAlarmPolicyGroup(r.pool.QueryRow(ctx, `
        INSERT INTO data_alarm_policy_groups (
            project_id, name, parent_id, description, is_enabled, sort_order, created_by, updated_by
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
        RETURNING id, project_id, name, parent_id, description, is_enabled, sort_order, created_at, updated_at
    `, params.ProjectID, params.Name, params.ParentID, params.Description, params.IsEnabled, params.SortOrder, params.UserID))
	if err != nil {
		return nil, translateAlarmPolicyGroupWriteError("创建报警分组失败", err)
	}
	return &record, nil
}

func (r *AlarmPolicyRepository) UpdateGroup(ctx context.Context, params UpdateAlarmPolicyGroupParams) (*AlarmPolicyGroupRecord, error) {
	record, err := scanAlarmPolicyGroup(r.pool.QueryRow(ctx, `
        UPDATE data_alarm_policy_groups
        SET name = $3,
            parent_id = $4,
            description = $5,
            is_enabled = $6,
            sort_order = $7,
            updated_by = $8,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, name, parent_id, description, is_enabled, sort_order, created_at, updated_at
    `, params.ProjectID, params.ID, params.Name, params.ParentID, params.Description, params.IsEnabled, params.SortOrder, params.UserID))
	if err != nil {
		return nil, translateAlarmPolicyGroupWriteError("更新报警分组失败", err)
	}
	return &record, nil
}

func (r *AlarmPolicyRepository) DeleteGroup(ctx context.Context, projectID, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM data_alarm_policy_groups WHERE project_id = $1 AND id = $2`, projectID, id)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除报警分组失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "报警分组不存在")
	}
	return nil
}

func (r *AlarmPolicyRepository) ListPolicyIDsByGroupTree(ctx context.Context, projectID, groupID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
        WITH RECURSIVE target_groups AS (
            SELECT id
            FROM data_alarm_policy_groups
            WHERE project_id = $1 AND id = $2
            UNION ALL
            SELECT child.id
            FROM data_alarm_policy_groups child
            INNER JOIN target_groups parent ON child.parent_id = parent.id
            WHERE child.project_id = $1
        )
        SELECT id
        FROM data_alarm_policies
        WHERE project_id = $1 AND group_id IN (SELECT id FROM target_groups)
    `, projectID, groupID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询报警分组内策略失败", err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取报警分组内策略失败", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历报警分组内策略失败", err)
	}
	return ids, nil
}

func (r *AlarmPolicyRepository) DeletePolicies(ctx context.Context, projectID string, policyIDs []string) error {
	if len(policyIDs) == 0 {
		return nil
	}
	if _, err := r.pool.Exec(ctx, `
        DELETE FROM data_alarm_policies
        WHERE project_id = $1 AND id = ANY($2::uuid[])
    `, projectID, policyIDs); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "批量删除报警策略失败", err)
	}
	return nil
}

func (r *AlarmPolicyRepository) ListPolicies(ctx context.Context, projectID string, filter AlarmPolicyListFilter) ([]AlarmPolicyRecord, int, error) {
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 20, 100)
	whereSQL, args := buildAlarmPolicyWhereClause(projectID, filter)

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_alarm_policies ap LEFT JOIN data_alarm_policy_groups ag ON ag.id = ap.group_id WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计报警策略失败", err)
	}

	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.pool.Query(ctx, alarmPolicySelectSQL()+`
        WHERE `+whereSQL+`
        ORDER BY ap.updated_at DESC, ap.created_at DESC
        LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2)+`
    `, listArgs...)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询报警策略失败", err)
	}
	defer rows.Close()

	records, err := scanAlarmPolicyRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func (r *AlarmPolicyRepository) ListPoliciesForTree(ctx context.Context, projectID string, filter AlarmPolicyListFilter) ([]AlarmPolicyRecord, error) {
	whereSQL, args := buildAlarmPolicyWhereClause(projectID, filter)
	rows, err := r.pool.Query(ctx, alarmPolicySelectSQL()+`
        WHERE `+whereSQL+`
        ORDER BY ag.sort_order ASC NULLS FIRST, ag.created_at ASC NULLS FIRST, ap.group_id NULLS FIRST, ap.name ASC
    `, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询报警策略树失败", err)
	}
	defer rows.Close()
	return scanAlarmPolicyRows(rows)
}

func (r *AlarmPolicyRepository) GetPolicyByProjectAndID(ctx context.Context, projectID, id string) (*AlarmPolicyRecord, error) {
	record, err := scanAlarmPolicy(r.pool.QueryRow(ctx, alarmPolicySelectSQL()+`
        WHERE ap.project_id = $1 AND ap.id = $2
    `, projectID, id))
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *AlarmPolicyRepository) CreatePolicy(ctx context.Context, params CreateAlarmPolicyParams) (*AlarmPolicyRecord, error) {
	targetsPayload, inputsPayload, conditionsPayload, suppressionPayload, contractPayload, err := marshalAlarmPolicyPayloads(params.Targets, params.Inputs, params.Conditions, params.Suppression, params.Contract)
	if err != nil {
		return nil, err
	}
	record, err := scanAlarmPolicy(r.pool.QueryRow(ctx, `
        WITH inserted AS (
            INSERT INTO data_alarm_policies (
                project_id, group_id, name, description, mode, targets, inputs, derived_expression,
                conditions, suppression, message_template, is_enabled, contract, created_by, updated_by
            )
            VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, $8, $9::jsonb, $10::jsonb, $11, $12, $13::jsonb, $14, $14)
            RETURNING id, project_id, group_id, name, description, mode, targets, inputs, derived_expression,
                      conditions, suppression, message_template, is_enabled, contract, created_at, updated_at
        )
        SELECT inserted.id, inserted.project_id, inserted.group_id, ag.name, ag.is_enabled, inserted.name,
               inserted.description, inserted.mode, inserted.targets, inserted.inputs, inserted.derived_expression,
               inserted.conditions, inserted.suppression, inserted.message_template, inserted.is_enabled,
               inserted.contract, inserted.created_at, inserted.updated_at
        FROM inserted
        LEFT JOIN data_alarm_policy_groups ag ON ag.id = inserted.group_id
    `, params.ProjectID, params.GroupID, params.Name, params.Description, params.Mode, string(targetsPayload), string(inputsPayload),
		params.DerivedExpression, string(conditionsPayload), string(suppressionPayload), params.MessageTemplate,
		params.IsEnabled, string(contractPayload), params.UserID))
	if err != nil {
		return nil, translateAlarmPolicyWriteError("创建报警策略失败", err)
	}
	return &record, nil
}

func (r *AlarmPolicyRepository) UpdatePolicy(ctx context.Context, params UpdateAlarmPolicyParams) (*AlarmPolicyRecord, error) {
	targetsPayload, inputsPayload, conditionsPayload, suppressionPayload, contractPayload, err := marshalAlarmPolicyPayloads(params.Targets, params.Inputs, params.Conditions, params.Suppression, params.Contract)
	if err != nil {
		return nil, err
	}
	record, err := scanAlarmPolicy(r.pool.QueryRow(ctx, `
        WITH updated AS (
            UPDATE data_alarm_policies
            SET group_id = $3,
                name = $4,
                description = $5,
                mode = $6,
                targets = $7::jsonb,
                inputs = $8::jsonb,
                derived_expression = $9,
                conditions = $10::jsonb,
                suppression = $11::jsonb,
                message_template = $12,
                is_enabled = $13,
                contract = $14::jsonb,
                updated_by = $15,
                updated_at = now()
            WHERE project_id = $1 AND id = $2
            RETURNING id, project_id, group_id, name, description, mode, targets, inputs, derived_expression,
                      conditions, suppression, message_template, is_enabled, contract, created_at, updated_at
        )
        SELECT updated.id, updated.project_id, updated.group_id, ag.name, ag.is_enabled, updated.name,
               updated.description, updated.mode, updated.targets, updated.inputs, updated.derived_expression,
               updated.conditions, updated.suppression, updated.message_template, updated.is_enabled,
               updated.contract, updated.created_at, updated.updated_at
        FROM updated
        LEFT JOIN data_alarm_policy_groups ag ON ag.id = updated.group_id
    `, params.ProjectID, params.ID, params.GroupID, params.Name, params.Description, params.Mode, string(targetsPayload),
		string(inputsPayload), params.DerivedExpression, string(conditionsPayload), string(suppressionPayload),
		params.MessageTemplate, params.IsEnabled, string(contractPayload), params.UserID))
	if err != nil {
		return nil, translateAlarmPolicyWriteError("更新报警策略失败", err)
	}
	return &record, nil
}

func (r *AlarmPolicyRepository) DeletePolicy(ctx context.Context, projectID, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM data_alarm_policies WHERE project_id = $1 AND id = $2`, projectID, id)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除报警策略失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "报警策略不存在")
	}
	return nil
}

func (r *AlarmPolicyRepository) SetPoliciesEnabled(ctx context.Context, projectID, userID string, ids []string, enabled bool) error {
	if len(ids) == 0 {
		return nil
	}
	tag, err := r.pool.Exec(ctx, `
        UPDATE data_alarm_policies
        SET is_enabled = $3, updated_by = $4, updated_at = now()
        WHERE project_id = $1 AND id = ANY($2::uuid[])
    `, projectID, ids, enabled, userID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "批量更新报警策略启停失败", err)
	}
	if int(tag.RowsAffected()) != len(ids) {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "部分报警策略不存在")
	}
	return nil
}

func (r *AlarmPolicyRepository) MovePolicies(ctx context.Context, projectID, userID string, ids []string, groupID *string) error {
	if len(ids) == 0 {
		return nil
	}
	tag, err := r.pool.Exec(ctx, `
        UPDATE data_alarm_policies
        SET group_id = $3, updated_by = $4, updated_at = now()
        WHERE project_id = $1 AND id = ANY($2::uuid[])
    `, projectID, ids, groupID, userID)
	if err != nil {
		return translateAlarmPolicyWriteError("批量移动报警策略失败", err)
	}
	if int(tag.RowsAffected()) != len(ids) {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "部分报警策略不存在")
	}
	return nil
}

func (r *AlarmPolicyRepository) ListPolicyIDsByFilter(ctx context.Context, projectID string, filter AlarmPolicyListFilter, excludeIDs []string) ([]string, error) {
	whereSQL, args := buildAlarmPolicyWhereClause(projectID, filter)
	if len(excludeIDs) > 0 {
		args = append(args, excludeIDs)
		whereSQL += fmt.Sprintf(" AND NOT (ap.id = ANY($%d::uuid[]))", len(args))
	}
	rows, err := r.pool.Query(ctx, `
        SELECT ap.id
        FROM data_alarm_policies ap
        LEFT JOIN data_alarm_policy_groups ag ON ag.id = ap.group_id
        WHERE `+whereSQL+`
        ORDER BY ap.id
    `, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询报警策略 ID 失败", err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取报警策略 ID 失败", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历报警策略 ID 失败", err)
	}
	return ids, nil
}

func scanAlarmPolicyGroup(row pgx.Row) (AlarmPolicyGroupRecord, error) {
	var record AlarmPolicyGroupRecord
	var parentID sql.NullString
	var description sql.NullString
	if err := row.Scan(&record.ID, &record.ProjectID, &record.Name, &parentID, &description, &record.IsEnabled, &record.SortOrder, &record.CreatedAt, &record.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AlarmPolicyGroupRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "报警分组不存在")
		}
		return AlarmPolicyGroupRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取报警分组失败", err)
	}
	record.ParentID = nullStringToPtr(parentID)
	record.Description = nullStringToPtr(description)
	return record, nil
}

func scanAlarmPolicyRows(rows pgx.Rows) ([]AlarmPolicyRecord, error) {
	records := make([]AlarmPolicyRecord, 0)
	for rows.Next() {
		record, scanErr := scanAlarmPolicy(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历报警策略失败", err)
	}
	return records, nil
}

func scanAlarmPolicy(row pgx.Row) (AlarmPolicyRecord, error) {
	var record AlarmPolicyRecord
	var groupID sql.NullString
	var groupName sql.NullString
	var groupEnabled sql.NullBool
	var description sql.NullString
	var targetsPayload []byte
	var inputsPayload []byte
	var conditionsPayload []byte
	var suppressionPayload []byte
	var contractPayload []byte
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&groupID,
		&groupName,
		&groupEnabled,
		&record.Name,
		&description,
		&record.Mode,
		&targetsPayload,
		&inputsPayload,
		&record.DerivedExpression,
		&conditionsPayload,
		&suppressionPayload,
		&record.MessageTemplate,
		&record.IsEnabled,
		&contractPayload,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AlarmPolicyRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "报警策略不存在")
		}
		return AlarmPolicyRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取报警策略失败", err)
	}
	record.GroupID = nullStringToPtr(groupID)
	record.GroupName = nullStringToPtr(groupName)
	record.GroupEnabled = nullBoolToPtr(groupEnabled)
	record.Description = nullStringToPtr(description)

	var err error
	if record.Targets, err = unmarshalAlarmJSONArray(targetsPayload); err != nil {
		return AlarmPolicyRecord{}, err
	}
	if record.Inputs, err = unmarshalAlarmJSONArray(inputsPayload); err != nil {
		return AlarmPolicyRecord{}, err
	}
	if record.Conditions, err = unmarshalAlarmJSONArray(conditionsPayload); err != nil {
		return AlarmPolicyRecord{}, err
	}
	if record.Suppression, err = unmarshalAlarmJSONObject(suppressionPayload); err != nil {
		return AlarmPolicyRecord{}, err
	}
	if record.Contract, err = unmarshalAlarmJSONObject(contractPayload); err != nil {
		return AlarmPolicyRecord{}, err
	}
	return record, nil
}

func alarmPolicySelectSQL() string {
	return `
        SELECT ap.id, ap.project_id, ap.group_id, ag.name, ag.is_enabled, ap.name, ap.description,
               ap.mode, ap.targets, ap.inputs, ap.derived_expression, ap.conditions, ap.suppression,
               ap.message_template, ap.is_enabled, ap.contract, ap.created_at, ap.updated_at
        FROM data_alarm_policies ap
        LEFT JOIN data_alarm_policy_groups ag ON ag.id = ap.group_id
    `
}

func buildAlarmPolicyWhereClause(projectID string, filter AlarmPolicyListFilter) (string, []any) {
	conditions := []string{"ap.project_id = $1"}
	args := []any{projectID}

	if filter.GroupID != nil {
		groupID := strings.TrimSpace(*filter.GroupID)
		if groupID == "" {
			conditions = append(conditions, "ap.group_id IS NULL")
		} else {
			args = append(args, groupID)
			conditions = append(conditions, fmt.Sprintf("ap.group_id = $%d", len(args)))
		}
	}
	if filter.Enabled != nil {
		args = append(args, *filter.Enabled)
		conditions = append(conditions, fmt.Sprintf("ap.is_enabled = $%d", len(args)))
	}
	if mode := strings.TrimSpace(filter.Mode); mode != "" {
		args = append(args, mode)
		conditions = append(conditions, fmt.Sprintf("ap.mode = $%d", len(args)))
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+search+"%")
		conditions = append(conditions, fmt.Sprintf("(ap.name ILIKE $%d OR ap.description ILIKE $%d)", len(args), len(args)))
	}
	if severity := strings.TrimSpace(filter.Severity); severity != "" {
		args = append(args, severity)
		conditions = append(conditions, fmt.Sprintf("ap.conditions @> jsonb_build_array(jsonb_build_object('severity', $%d::text))", len(args)))
	}
	if conditionType := strings.TrimSpace(filter.ConditionType); conditionType != "" {
		args = append(args, conditionType)
		conditions = append(conditions, fmt.Sprintf("ap.conditions @> jsonb_build_array(jsonb_build_object('type', $%d::text))", len(args)))
	}
	if targetPath := strings.TrimSpace(filter.TargetPath); targetPath != "" {
		args = append(args, targetPath)
		conditions = append(conditions, fmt.Sprintf("(ap.targets @> jsonb_build_array(jsonb_build_object('path', $%d::text)) OR ap.inputs @> jsonb_build_array(jsonb_build_object('path', $%d::text)))", len(args), len(args)))
	}
	return strings.Join(conditions, " AND "), args
}

func marshalAlarmPolicyPayloads(targets, inputs, conditions []map[string]any, suppression, contract map[string]any) ([]byte, []byte, []byte, []byte, []byte, error) {
	targetsPayload, err := marshalAlarmJSONArray(targets)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	inputsPayload, err := marshalAlarmJSONArray(inputs)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	conditionsPayload, err := marshalAlarmJSONArray(conditions)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	suppressionPayload, err := marshalJSONObject(suppression)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	contractPayload, err := marshalJSONObject(contract)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return targetsPayload, inputsPayload, conditionsPayload, suppressionPayload, contractPayload, nil
}

func marshalAlarmJSONArray(value []map[string]any) ([]byte, error) {
	if value == nil {
		value = []map[string]any{}
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "报警策略数组 JSON 无效", err)
	}
	return payload, nil
}

func unmarshalAlarmJSONArray(payload []byte) ([]map[string]any, error) {
	if len(payload) == 0 {
		return []map[string]any{}, nil
	}
	var result []map[string]any
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析报警策略数组失败", err)
	}
	if result == nil {
		result = []map[string]any{}
	}
	return result, nil
}

func translateAlarmPolicyGroupWriteError(message string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "报警分组名称已存在", err)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "报警分组不存在")
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}

func translateAlarmPolicyWriteError(message string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "报警策略名称已存在", err)
		case "23503":
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "报警分组不存在", err)
		case "23514":
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "报警策略字段不符合约束", err)
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "报警策略不存在")
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}

func nullBoolToPtr(value sql.NullBool) *bool {
	if !value.Valid {
		return nil
	}
	result := value.Bool
	return &result
}
