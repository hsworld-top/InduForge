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

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StoragePolicyRecord 表示历史归档策略的仓储层投影。
type StoragePolicyRecord struct {
	ID                 string
	ProjectID          string
	Name               string
	Description        *string
	TargetConnectionID string
	TargetName         string
	TargetType         string
	TargetStatus       string
	TargetCapability   string
	BindingMode        string
	BindingFilter      map[string]any
	WriteMode          string
	MinIntervalMS      *int
	Deadband           *float64
	SnapshotIntervalMS *int
	IncludeQualities   []any
	RetentionDays      int
	TargetTableMode    string
	TargetTableConfig  map[string]any
	Status             string
	Diagnostics        []any
	BindingCount       int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// StoragePolicyBindingRecord 表示策略与数据点静态绑定关系。
type StoragePolicyBindingRecord struct {
	ID            string
	ProjectID     string
	PolicyID      string
	DatapointID   string
	DatapointPath string
	DatapointName string
	DataType      string
	Status        string
	CreatedAt     time.Time
}

// StorageTargetRecord 表示可作为历史归档目标的接入源摘要。
type StorageTargetRecord struct {
	ID           string
	ProjectID    string
	Name         string
	Type         string
	Category     string
	Status       string
	Capabilities []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type StoragePolicyListFilter struct {
	Search             string
	Status             string
	TargetConnectionID string
	WriteMode          string
	BindingMode        string
	Page               int
	PageSize           int
}

type StoragePolicyDataPointFilter struct {
	Type           string
	Status         string
	Search         string
	AccessSourceID string
	SourceID       string
	SourceIDs      []string
	Tags           []string
}

type CreateStoragePolicyParams struct {
	ProjectID          string
	UserID             string
	Name               string
	Description        *string
	TargetConnectionID string
	TargetCapability   string
	BindingMode        string
	BindingFilter      map[string]any
	WriteMode          string
	MinIntervalMS      *int
	Deadband           *float64
	SnapshotIntervalMS *int
	IncludeQualities   []any
	RetentionDays      int
	TargetTableMode    string
	TargetTableConfig  map[string]any
	Status             string
	Diagnostics        []any
	DatapointIDs       []string
}

type UpdateStoragePolicyParams struct {
	ID                 string
	ProjectID          string
	UserID             string
	Name               string
	Description        *string
	TargetConnectionID string
	TargetCapability   string
	BindingMode        string
	BindingFilter      map[string]any
	WriteMode          string
	MinIntervalMS      *int
	Deadband           *float64
	SnapshotIntervalMS *int
	IncludeQualities   []any
	RetentionDays      int
	TargetTableMode    string
	TargetTableConfig  map[string]any
	Status             string
	Diagnostics        []any
	DatapointIDs       []string
}

// StoragePolicyRepository 封装存储策略、绑定和目标能力查询。
type StoragePolicyRepository struct {
	pool *pgxpool.Pool
}

func NewStoragePolicyRepository(pool *pgxpool.Pool) *StoragePolicyRepository {
	return &StoragePolicyRepository{pool: pool}
}

func (r *StoragePolicyRepository) ListPolicies(ctx context.Context, projectID string, filter StoragePolicyListFilter) ([]StoragePolicyRecord, int, error) {
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 20, 100)
	whereSQL, args := buildStoragePolicyWhereClause(projectID, filter)

	var total int
	if err := r.pool.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM data_storage_policies sp
        INNER JOIN data_connections conn ON conn.id = sp.target_connection_id
        WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计存储策略失败", err)
	}

	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.pool.Query(ctx, storagePolicySelectSQL()+`
        WHERE `+whereSQL+`
        ORDER BY sp.updated_at DESC, sp.created_at DESC
        LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2)+`
    `, listArgs...)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询存储策略失败", err)
	}
	defer rows.Close()

	records, err := scanStoragePolicyRows(rows)
	if err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func (r *StoragePolicyRepository) GetPolicyByProjectAndID(ctx context.Context, projectID, id string) (*StoragePolicyRecord, error) {
	record, err := scanStoragePolicy(r.pool.QueryRow(ctx, storagePolicySelectSQL()+`
        WHERE sp.project_id = $1 AND sp.id = $2
    `, projectID, id))
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *StoragePolicyRepository) CreatePolicy(ctx context.Context, params CreateStoragePolicyParams) (*StoragePolicyRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开始创建存储策略事务失败", err)
	}
	defer tx.Rollback(ctx)

	record, err := insertStoragePolicy(ctx, tx, params)
	if err != nil {
		return nil, err
	}
	if err := replaceStoragePolicyBindings(ctx, tx, params.ProjectID, record.ID, params.DatapointIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交创建存储策略事务失败", err)
	}
	return r.GetPolicyByProjectAndID(ctx, params.ProjectID, record.ID)
}

func (r *StoragePolicyRepository) UpdatePolicy(ctx context.Context, params UpdateStoragePolicyParams) (*StoragePolicyRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开始更新存储策略事务失败", err)
	}
	defer tx.Rollback(ctx)

	if _, err := updateStoragePolicy(ctx, tx, params); err != nil {
		return nil, err
	}
	if err := replaceStoragePolicyBindings(ctx, tx, params.ProjectID, params.ID, params.DatapointIDs); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交更新存储策略事务失败", err)
	}
	return r.GetPolicyByProjectAndID(ctx, params.ProjectID, params.ID)
}

func (r *StoragePolicyRepository) DeletePolicy(ctx context.Context, projectID, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM data_storage_policies WHERE project_id = $1 AND id = $2`, projectID, id)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除存储策略失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "存储策略不存在")
	}
	return nil
}

func (r *StoragePolicyRepository) ListBindings(ctx context.Context, projectID, policyID string) ([]StoragePolicyBindingRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT b.id, b.project_id, b.policy_id, b.datapoint_id, b.datapoint_path,
               dp.name, dp.data_type, dp.status, b.created_at
        FROM data_storage_policy_bindings b
        INNER JOIN data_points dp ON dp.id = b.datapoint_id
        WHERE b.project_id = $1 AND b.policy_id = $2
        ORDER BY b.datapoint_path ASC
    `, projectID, policyID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询存储策略绑定失败", err)
	}
	defer rows.Close()

	records := make([]StoragePolicyBindingRecord, 0)
	for rows.Next() {
		record, scanErr := scanStoragePolicyBinding(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历存储策略绑定失败", err)
	}
	return records, nil
}

func (r *StoragePolicyRepository) ListTargets(ctx context.Context, projectID string) ([]StorageTargetRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, name, type, category, status, created_at, updated_at
        FROM data_connections
        WHERE project_id = $1
          AND type IN ('builtin.timeseries', 'tdengine', 'relational', 'builtin.relation')
        ORDER BY display_order ASC, created_at ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询存储目标失败", err)
	}
	defer rows.Close()

	records := make([]StorageTargetRecord, 0)
	for rows.Next() {
		var record StorageTargetRecord
		if err := rows.Scan(&record.ID, &record.ProjectID, &record.Name, &record.Type, &record.Category, &record.Status, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取存储目标失败", err)
		}
		record.Capabilities = storageCapabilitiesForConnectionType(record.Type)
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历存储目标失败", err)
	}
	return records, nil
}

func (r *StoragePolicyRepository) GetTarget(ctx context.Context, projectID, connectionID string) (*StorageTargetRecord, error) {
	var record StorageTargetRecord
	err := r.pool.QueryRow(ctx, `
        SELECT id, project_id, name, type, category, status, created_at, updated_at
        FROM data_connections
        WHERE project_id = $1 AND id = $2
    `, projectID, connectionID).Scan(&record.ID, &record.ProjectID, &record.Name, &record.Type, &record.Category, &record.Status, &record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "存储目标不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取存储目标失败", err)
	}
	record.Capabilities = storageCapabilitiesForConnectionType(record.Type)
	return &record, nil
}

func (r *StoragePolicyRepository) CountDataPointsByFilter(ctx context.Context, projectID string, filter StoragePolicyDataPointFilter) (int, error) {
	whereSQL, args, err := buildStoragePolicyDataPointWhereClause(projectID, filter)
	if err != nil {
		return 0, err
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_points WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计策略命中数据点失败", err)
	}
	return total, nil
}

func (r *StoragePolicyRepository) CountPoliciesForDataPoint(ctx context.Context, projectID, datapointID, datapointPath string) (int, error) {
	conditions := []string{"sp.project_id = $1", "sp.status = 'enabled'"}
	args := []any{projectID}
	if strings.TrimSpace(datapointID) != "" {
		args = append(args, strings.TrimSpace(datapointID))
		conditions = append(conditions, fmt.Sprintf(`(
            EXISTS (
                SELECT 1 FROM data_storage_policy_bindings b
                WHERE b.policy_id = sp.id AND b.datapoint_id = $%d::uuid
            )
            OR sp.binding_mode = 'dynamic'
        )`, len(args)))
	} else if strings.TrimSpace(datapointPath) != "" {
		args = append(args, strings.TrimSpace(datapointPath))
		conditions = append(conditions, fmt.Sprintf(`(
            EXISTS (
                SELECT 1 FROM data_storage_policy_bindings b
                WHERE b.policy_id = sp.id AND b.datapoint_path = $%d
            )
            OR sp.binding_mode = 'dynamic'
        )`, len(args)))
	}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_storage_policies sp WHERE `+strings.Join(conditions, " AND "), args...).Scan(&total); err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计数据点命中存储策略失败", err)
	}
	return total, nil
}

func insertStoragePolicy(ctx context.Context, tx pgx.Tx, params CreateStoragePolicyParams) (*StoragePolicyRecord, error) {
	payloads, err := marshalStoragePolicyPayloads(params.BindingFilter, params.IncludeQualities, params.TargetTableConfig, params.Diagnostics)
	if err != nil {
		return nil, err
	}
	record, err := scanStoragePolicy(tx.QueryRow(ctx, `
        WITH inserted AS (
            INSERT INTO data_storage_policies (
                project_id, name, description, target_connection_id, target_capability,
                binding_mode, binding_filter, write_mode, min_interval_ms, deadband,
                snapshot_interval_ms, include_qualities, retention_days, target_table_mode,
                target_table_config, status, diagnostics, created_by, updated_by
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11, $12::jsonb, $13, $14, $15::jsonb, $16, $17::jsonb, $18, $18)
            RETURNING *
        )
        SELECT inserted.id, inserted.project_id, inserted.name, inserted.description,
               inserted.target_connection_id, conn.name, conn.type, conn.status,
               inserted.target_capability, inserted.binding_mode, inserted.binding_filter,
               inserted.write_mode, inserted.min_interval_ms, inserted.deadband,
               inserted.snapshot_interval_ms, inserted.include_qualities, inserted.retention_days,
               inserted.target_table_mode, inserted.target_table_config, inserted.status,
               inserted.diagnostics, 0, inserted.created_at, inserted.updated_at
        FROM inserted
        INNER JOIN data_connections conn ON conn.id = inserted.target_connection_id
    `, params.ProjectID, params.Name, params.Description, params.TargetConnectionID, params.TargetCapability,
		params.BindingMode, string(payloads.bindingFilter), params.WriteMode, params.MinIntervalMS, params.Deadband,
		params.SnapshotIntervalMS, string(payloads.includeQualities), params.RetentionDays, params.TargetTableMode,
		string(payloads.targetTableConfig), params.Status, string(payloads.diagnostics), params.UserID))
	if err != nil {
		return nil, translateStoragePolicyWriteError("创建存储策略失败", err)
	}
	return &record, nil
}

func updateStoragePolicy(ctx context.Context, tx pgx.Tx, params UpdateStoragePolicyParams) (*StoragePolicyRecord, error) {
	payloads, err := marshalStoragePolicyPayloads(params.BindingFilter, params.IncludeQualities, params.TargetTableConfig, params.Diagnostics)
	if err != nil {
		return nil, err
	}
	record, err := scanStoragePolicy(tx.QueryRow(ctx, `
        WITH updated AS (
            UPDATE data_storage_policies
            SET name = $3,
                description = $4,
                target_connection_id = $5,
                target_capability = $6,
                binding_mode = $7,
                binding_filter = $8::jsonb,
                write_mode = $9,
                min_interval_ms = $10,
                deadband = $11,
                snapshot_interval_ms = $12,
                include_qualities = $13::jsonb,
                retention_days = $14,
                target_table_mode = $15,
                target_table_config = $16::jsonb,
                status = $17,
                diagnostics = $18::jsonb,
                updated_by = $19,
                updated_at = now()
            WHERE project_id = $1 AND id = $2
            RETURNING *
        )
        SELECT updated.id, updated.project_id, updated.name, updated.description,
               updated.target_connection_id, conn.name, conn.type, conn.status,
               updated.target_capability, updated.binding_mode, updated.binding_filter,
               updated.write_mode, updated.min_interval_ms, updated.deadband,
               updated.snapshot_interval_ms, updated.include_qualities, updated.retention_days,
               updated.target_table_mode, updated.target_table_config, updated.status,
               updated.diagnostics, 0, updated.created_at, updated.updated_at
        FROM updated
        INNER JOIN data_connections conn ON conn.id = updated.target_connection_id
    `, params.ProjectID, params.ID, params.Name, params.Description, params.TargetConnectionID, params.TargetCapability,
		params.BindingMode, string(payloads.bindingFilter), params.WriteMode, params.MinIntervalMS, params.Deadband,
		params.SnapshotIntervalMS, string(payloads.includeQualities), params.RetentionDays, params.TargetTableMode,
		string(payloads.targetTableConfig), params.Status, string(payloads.diagnostics), params.UserID))
	if err != nil {
		return nil, translateStoragePolicyWriteError("更新存储策略失败", err)
	}
	return &record, nil
}

func replaceStoragePolicyBindings(ctx context.Context, tx pgx.Tx, projectID, policyID string, datapointIDs []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM data_storage_policy_bindings WHERE project_id = $1 AND policy_id = $2`, projectID, policyID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "清理存储策略绑定失败", err)
	}
	if len(datapointIDs) == 0 {
		return nil
	}
	tag, err := tx.Exec(ctx, `
        INSERT INTO data_storage_policy_bindings (project_id, policy_id, datapoint_id, datapoint_path)
        SELECT $1, $2, dp.id, dp.path
        FROM data_points dp
        WHERE dp.project_id = $1 AND dp.id = ANY($3::uuid[])
        ON CONFLICT (policy_id, datapoint_id) DO NOTHING
    `, projectID, policyID, datapointIDs)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入存储策略绑定失败", err)
	}
	if int(tag.RowsAffected()) != len(uniqueStrings(datapointIDs)) {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "部分数据点不存在，无法绑定存储策略")
	}
	return nil
}

type storagePolicyPayloads struct {
	bindingFilter     []byte
	includeQualities  []byte
	targetTableConfig []byte
	diagnostics       []byte
}

func marshalStoragePolicyPayloads(bindingFilter map[string]any, includeQualities []any, targetTableConfig map[string]any, diagnostics []any) (storagePolicyPayloads, error) {
	bindingFilterPayload, err := marshalJSONObject(bindingFilter)
	if err != nil {
		return storagePolicyPayloads{}, err
	}
	targetTableConfigPayload, err := marshalJSONObject(targetTableConfig)
	if err != nil {
		return storagePolicyPayloads{}, err
	}
	includeQualitiesPayload, err := marshalJSONArray(includeQualities)
	if err != nil {
		return storagePolicyPayloads{}, err
	}
	diagnosticsPayload, err := marshalJSONArray(diagnostics)
	if err != nil {
		return storagePolicyPayloads{}, err
	}
	return storagePolicyPayloads{
		bindingFilter: bindingFilterPayload, includeQualities: includeQualitiesPayload,
		targetTableConfig: targetTableConfigPayload, diagnostics: diagnosticsPayload,
	}, nil
}

func storagePolicySelectSQL() string {
	return `
        SELECT sp.id, sp.project_id, sp.name, sp.description,
               sp.target_connection_id, conn.name, conn.type, conn.status,
               sp.target_capability, sp.binding_mode, sp.binding_filter,
               sp.write_mode, sp.min_interval_ms, sp.deadband, sp.snapshot_interval_ms,
               sp.include_qualities, sp.retention_days, sp.target_table_mode,
               sp.target_table_config, sp.status, sp.diagnostics,
               COALESCE(binding_stats.binding_count, 0), sp.created_at, sp.updated_at
        FROM data_storage_policies sp
        INNER JOIN data_connections conn ON conn.id = sp.target_connection_id
        LEFT JOIN (
            SELECT policy_id, COUNT(*)::integer AS binding_count
            FROM data_storage_policy_bindings
            GROUP BY policy_id
        ) binding_stats ON binding_stats.policy_id = sp.id
    `
}

func buildStoragePolicyWhereClause(projectID string, filter StoragePolicyListFilter) (string, []any) {
	clauses := []string{"sp.project_id = $1"}
	args := []any{projectID}
	add := func(column, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		args = append(args, strings.TrimSpace(value))
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	add("sp.status", filter.Status)
	add("sp.target_connection_id", filter.TargetConnectionID)
	add("sp.write_mode", filter.WriteMode)
	add("sp.binding_mode", filter.BindingMode)
	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+search+"%")
		index := len(args)
		clauses = append(clauses, fmt.Sprintf("(sp.name ILIKE $%d OR COALESCE(sp.description, '') ILIKE $%d OR conn.name ILIKE $%d)", index, index, index))
	}
	return strings.Join(clauses, " AND "), args
}

func buildStoragePolicyDataPointWhereClause(projectID string, filter StoragePolicyDataPointFilter) (string, []any, error) {
	return buildDataPointWhereClause(projectID, DataPointListFilter{
		Type: filter.Type, Status: filter.Status, Search: filter.Search, AccessSourceID: filter.AccessSourceID,
		SourceID: filter.SourceID, SourceIDs: filter.SourceIDs, Tags: filter.Tags,
	})
}

func scanStoragePolicyRows(rows pgx.Rows) ([]StoragePolicyRecord, error) {
	records := make([]StoragePolicyRecord, 0)
	for rows.Next() {
		record, scanErr := scanStoragePolicy(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历存储策略失败", err)
	}
	return records, nil
}

type storagePolicyScannable interface {
	Scan(dest ...any) error
}

func scanStoragePolicy(row storagePolicyScannable) (StoragePolicyRecord, error) {
	var record StoragePolicyRecord
	var description sql.NullString
	var minInterval sql.NullInt32
	var deadband sql.NullFloat64
	var snapshotInterval sql.NullInt32
	var bindingFilterPayload []byte
	var includeQualitiesPayload []byte
	var targetTableConfigPayload []byte
	var diagnosticsPayload []byte
	if err := row.Scan(
		&record.ID, &record.ProjectID, &record.Name, &description,
		&record.TargetConnectionID, &record.TargetName, &record.TargetType, &record.TargetStatus,
		&record.TargetCapability, &record.BindingMode, &bindingFilterPayload,
		&record.WriteMode, &minInterval, &deadband, &snapshotInterval,
		&includeQualitiesPayload, &record.RetentionDays, &record.TargetTableMode,
		&targetTableConfigPayload, &record.Status, &diagnosticsPayload,
		&record.BindingCount, &record.CreatedAt, &record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoragePolicyRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "存储策略不存在")
		}
		return StoragePolicyRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取存储策略失败", err)
	}
	record.Description = nullStringToPtr(description)
	record.MinIntervalMS = nullInt32ToPtr(minInterval)
	record.Deadband = nullFloat64ToPtr(deadband)
	record.SnapshotIntervalMS = nullInt32ToPtr(snapshotInterval)
	if err := json.Unmarshal(bindingFilterPayload, &record.BindingFilter); err != nil {
		return StoragePolicyRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析存储策略筛选失败", err)
	}
	if record.BindingFilter == nil {
		record.BindingFilter = map[string]any{}
	}
	if err := json.Unmarshal(includeQualitiesPayload, &record.IncludeQualities); err != nil {
		return StoragePolicyRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析存储策略质量配置失败", err)
	}
	if record.IncludeQualities == nil {
		record.IncludeQualities = []any{}
	}
	if err := json.Unmarshal(targetTableConfigPayload, &record.TargetTableConfig); err != nil {
		return StoragePolicyRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析存储目标表配置失败", err)
	}
	if record.TargetTableConfig == nil {
		record.TargetTableConfig = map[string]any{}
	}
	if err := json.Unmarshal(diagnosticsPayload, &record.Diagnostics); err != nil {
		return StoragePolicyRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析存储策略诊断失败", err)
	}
	if record.Diagnostics == nil {
		record.Diagnostics = []any{}
	}
	return record, nil
}

func scanStoragePolicyBinding(row pgx.Row) (StoragePolicyBindingRecord, error) {
	var record StoragePolicyBindingRecord
	if err := row.Scan(&record.ID, &record.ProjectID, &record.PolicyID, &record.DatapointID, &record.DatapointPath, &record.DatapointName, &record.DataType, &record.Status, &record.CreatedAt); err != nil {
		return StoragePolicyBindingRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取存储策略绑定失败", err)
	}
	return record, nil
}

func storageCapabilitiesForConnectionType(connectionType string) []string {
	switch strings.TrimSpace(connectionType) {
	case "builtin.timeseries", "tdengine":
		return []string{"timeseriesAppend"}
	case "relational", "builtin.relation":
		return []string{"relationalAppend"}
	default:
		return []string{}
	}
}

func translateStoragePolicyWriteError(message string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "存储策略名称已存在", err)
		case "23503":
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "存储策略关联资源不存在", err)
		case "23514":
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "存储策略字段不符合约束", err)
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "存储策略不存在")
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
