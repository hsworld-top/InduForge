package repository

import (
	"context"
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

type CollectorConnectionRecord struct {
	ID                 string
	ProjectID          string
	Name               string
	Code               string
	Status             string
	IsEnabled          bool
	DefaultAcquisition map[string]any
	DisplayOrder       int
	ProtocolFamily     string
	DriverID           string
	DriverVersion      string
	SchemaVersion      int
	Config             map[string]any
	Metadata           map[string]any
	SecretStatus       map[string]bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
	LastTestStatus     *string
	LastTestedAt       *time.Time
}

type CollectorConnectionListFilter struct {
	Page, PageSize                                      int
	Search, ProtocolFamily, DriverID, SortBy, SortOrder string
}

type EncryptedCollectorSecret struct {
	Key, KeyVersion string
	Value           []byte
}

type CollectorConnectionSecretRecord struct {
	Key        string
	Value      []byte
	KeyVersion string
}

type CreateCollectorConnectionParams struct {
	ID, ProjectID, UserID, Name, Code, ProtocolFamily, DriverID, DriverVersion string
	SchemaVersion                                                              int
	Config, Metadata                                                           map[string]any
	DefaultAcquisition                                                         map[string]any
	IsEnabled                                                                  bool
	Secrets                                                                    []EncryptedCollectorSecret
}

type UpdateCollectorConnectionParams struct {
	ID, ProjectID, UserID, Name string
	Config, Metadata            map[string]any
	DefaultAcquisition          map[string]any
	IsEnabled                   bool
	Secrets                     []EncryptedCollectorSecret
	DeleteSecretKeys            []string
}

type CollectorRepository struct{ pool *pgxpool.Pool }

type CollectorFailedPointDiagnostic struct {
	PointID      string    `json:"pointId"`
	Name         string    `json:"name"`
	AddressText  string    `json:"addressText"`
	ErrorCode    string    `json:"errorCode,omitempty"`
	ErrorMessage string    `json:"errorMessage"`
	AttemptedAt  time.Time `json:"attemptedAt"`
}

type CollectorConnectionDiagnosticRecord struct {
	AgentID, AgentName, AgentVersion string
	AgentLastSeenAt                  *time.Time
	AgentReady                       bool
	AgentReason                      string
	LastTestStatus                   string
	LastTestAt                       *time.Time
	LastTestDurationMS               int64
	LastErrorCode, LastErrorMessage  string
	PointCount, AttemptedPointCount  int
	SucceededPointCount              int
	FailedPoints                     []CollectorFailedPointDiagnostic
}

func NewCollectorRepository(pool *pgxpool.Pool) *CollectorRepository {
	return &CollectorRepository{pool: pool}
}

func (r *CollectorRepository) ListConnections(ctx context.Context, projectID string, filter CollectorConnectionListFilter) ([]CollectorConnectionRecord, int, error) {
	conditions := []string{"collector.project_id = $1"}
	args := []any{projectID}
	add := func(condition string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}
	if value := strings.TrimSpace(filter.Search); value != "" {
		add("(collector.name ILIKE $%[1]d OR collector.driver_id ILIKE $%[1]d OR collector.protocol_family ILIKE $%[1]d)", "%"+value+"%")
	}
	if value := strings.TrimSpace(filter.ProtocolFamily); value != "" {
		add("collector.protocol_family = $%d", value)
	}
	if value := strings.TrimSpace(filter.DriverID); value != "" {
		add("collector.driver_id = $%d", value)
	}

	orderColumn := map[string]string{"name": "collector.name", "driverId": "collector.driver_id", "createdAt": "collector.created_at", "updatedAt": "collector.updated_at"}[filter.SortBy]
	if orderColumn == "" {
		orderColumn = "collector.display_order"
	}
	orderDirection := "ASC"
	if strings.EqualFold(filter.SortOrder, "desc") {
		orderDirection = "DESC"
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	query := fmt.Sprintf(`
		SELECT collector.id, collector.project_id, collector.name, collector.code, 'unknown', collector.is_enabled, collector.default_acquisition, collector.display_order,
		       collector.protocol_family, collector.driver_id, collector.driver_version, collector.schema_version,
		       collector.config, collector.metadata,
		       COALESCE((SELECT jsonb_object_agg(secret_key, true) FROM data_collector_connection_secrets secret WHERE secret.connection_id = collector.id), '{}'::jsonb),
		       (SELECT task.status FROM collector_dev_tasks task WHERE task.project_id = collector.project_id AND task.connection_id = collector.id AND task.operation = 'connection.test' ORDER BY task.created_at DESC, task.id DESC LIMIT 1),
		       (SELECT COALESCE(task.finished_at, task.created_at) FROM collector_dev_tasks task WHERE task.project_id = collector.project_id AND task.connection_id = collector.id AND task.operation = 'connection.test' ORDER BY task.created_at DESC, task.id DESC LIMIT 1),
		       collector.created_at, collector.updated_at, COUNT(*) OVER()::int
		FROM data_collector_connections collector
		WHERE %s
		ORDER BY %s %s, collector.id ASC
		LIMIT $%d OFFSET $%d`, strings.Join(conditions, " AND "), orderColumn, orderDirection, len(args)-1, len(args))
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, wrapUnifiedCollectorRepositoryError("查询工业采集连接失败", err)
	}
	defer rows.Close()
	records := make([]CollectorConnectionRecord, 0)
	total := 0
	for rows.Next() {
		record, count, err := scanCollectorConnectionWithTotal(rows)
		if err != nil {
			return nil, 0, err
		}
		total = count
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, wrapUnifiedCollectorRepositoryError("遍历工业采集连接失败", err)
	}
	return records, total, nil
}

func (r *CollectorRepository) GetConnection(ctx context.Context, projectID, connectionID string) (*CollectorConnectionRecord, error) {
	record, err := scanCollectorConnection(r.pool.QueryRow(ctx, collectorConnectionSelect+` WHERE collector.project_id = $1 AND collector.id = $2`, projectID, connectionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "工业采集连接不存在")
	}
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("查询工业采集连接失败", err)
	}
	return &record, nil
}

func (r *CollectorRepository) GetConnectionSecrets(ctx context.Context, connectionID string) ([]CollectorConnectionSecretRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT secret_key, encrypted_value, encryption_key_version FROM data_collector_connection_secrets WHERE connection_id=$1 ORDER BY secret_key`, connectionID)
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("查询工业采集连接密钥失败", err)
	}
	defer rows.Close()
	result := make([]CollectorConnectionSecretRecord, 0)
	for rows.Next() {
		var record CollectorConnectionSecretRecord
		if err := rows.Scan(&record.Key, &record.Value, &record.KeyVersion); err != nil {
			return nil, wrapUnifiedCollectorRepositoryError("扫描工业采集连接密钥失败", err)
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

// GetConnectionDiagnostic 汇总开发态最近测试、代理就绪状态和点位读取快照。
// 该摘要不承担运行态监控职责，也不保存长期时序诊断数据。
func (r *CollectorRepository) GetConnectionDiagnostic(ctx context.Context, projectID, connectionID string) (*CollectorConnectionDiagnosticRecord, error) {
	if _, err := r.GetConnection(ctx, projectID, connectionID); err != nil {
		return nil, err
	}
	record := CollectorConnectionDiagnosticRecord{FailedPoints: []CollectorFailedPointDiagnostic{}}
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(agent.id::text,''),COALESCE(agent.name,''),COALESCE(agent.version,''),agent.last_seen_at,
		       CASE WHEN agent.id IS NULL THEN false WHEN agent.revoked_at IS NOT NULL THEN false
		            WHEN agent.disconnected_at IS NOT NULL THEN false
		            WHEN agent.last_seen_at IS NULL OR agent.last_seen_at < now()-interval '90 seconds' THEN false ELSE true END,
		       CASE WHEN agent.id IS NULL THEN '未选择调试代理' WHEN agent.revoked_at IS NOT NULL THEN '调试代理已撤销'
		            WHEN agent.disconnected_at IS NOT NULL THEN '调试代理已离线'
		            WHEN agent.last_seen_at IS NULL OR agent.last_seen_at < now()-interval '90 seconds' THEN '调试代理心跳已过期' ELSE '' END,
		       COALESCE(task.status,'not_tested'),COALESCE(task.finished_at,task.created_at),
		       COALESCE((extract(epoch FROM (COALESCE(task.finished_at,now())-COALESCE(task.claimed_at,task.created_at)))*1000)::bigint,0),
		       COALESCE(task.error_code,''),COALESCE(task.error_message,'')
		FROM data_collector_connections connection
		LEFT JOIN LATERAL (
			SELECT * FROM collector_dev_tasks WHERE project_id=connection.project_id AND connection_id=connection.id
			ORDER BY created_at DESC,id DESC LIMIT 1
		) task ON true
		LEFT JOIN collector_dev_agents agent ON agent.id=task.agent_id
		WHERE connection.project_id=$1 AND connection.id=$2
	`, projectID, connectionID).Scan(&record.AgentID, &record.AgentName, &record.AgentVersion, &record.AgentLastSeenAt,
		&record.AgentReady, &record.AgentReason, &record.LastTestStatus, &record.LastTestAt, &record.LastTestDurationMS,
		&record.LastErrorCode, &record.LastErrorMessage)
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("读取工业连接诊断摘要失败", err)
	}
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int,COUNT(snapshot.point_id)::int,
		       COUNT(snapshot.point_id) FILTER (WHERE snapshot.last_attempt_status='succeeded')::int
		FROM data_collector_points point
		LEFT JOIN data_collector_point_debug_snapshots snapshot ON snapshot.point_id=point.id
		WHERE point.project_id=$1 AND point.connection_id=$2
	`, projectID, connectionID).Scan(&record.PointCount, &record.AttemptedPointCount, &record.SucceededPointCount); err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("统计工业连接点位诊断失败", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT point.id::text,point.name,point.address_text,COALESCE(snapshot.last_error_code,''),
		       COALESCE(snapshot.last_error_message,''),snapshot.last_attempt_at
		FROM data_collector_points point
		JOIN data_collector_point_debug_snapshots snapshot ON snapshot.point_id=point.id
		WHERE point.project_id=$1 AND point.connection_id=$2 AND snapshot.last_attempt_status='failed'
		ORDER BY snapshot.last_attempt_at DESC,point.name LIMIT 100
	`, projectID, connectionID)
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("读取失败点位诊断失败", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item CollectorFailedPointDiagnostic
		if err := rows.Scan(&item.PointID, &item.Name, &item.AddressText, &item.ErrorCode, &item.ErrorMessage, &item.AttemptedAt); err != nil {
			return nil, wrapUnifiedCollectorRepositoryError("解析失败点位诊断失败", err)
		}
		record.FailedPoints = append(record.FailedPoints, item)
	}
	return &record, rows.Err()
}

func (r *CollectorRepository) CreateConnection(ctx context.Context, params CreateCollectorConnectionParams) (*CollectorConnectionRecord, error) {
	configPayload, err := json.Marshal(params.Config)
	if err != nil {
		return nil, badCollectorPayload("序列化连接配置失败", err)
	}
	metadataPayload, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, badCollectorPayload("序列化连接元数据失败", err)
	}
	defaultAcquisitionPayload, err := json.Marshal(params.DefaultAcquisition)
	if err != nil {
		return nil, badCollectorPayload("序列化默认采集参数失败", err)
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("开启连接创建事务失败", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO data_collector_connections (id, project_id, name, code, protocol_family, driver_id, driver_version, schema_version, config, metadata, is_enabled, default_acquisition, created_by, updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10::jsonb,$11,$12::jsonb,$13,$13)`, params.ID, params.ProjectID, params.Name, params.Code, params.ProtocolFamily, params.DriverID, params.DriverVersion, params.SchemaVersion, string(configPayload), string(metadataPayload), params.IsEnabled, string(defaultAcquisitionPayload), params.UserID)
	if err != nil {
		return nil, translateCollectorWriteError("创建工业采集连接失败", err)
	}
	if err := writeCollectorSecrets(ctx, tx, params.ID, params.Secrets, nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("提交工业采集连接创建事务失败", err)
	}
	return r.GetConnection(ctx, params.ProjectID, params.ID)
}

func (r *CollectorRepository) UpdateConnection(ctx context.Context, params UpdateCollectorConnectionParams) (*CollectorConnectionRecord, error) {
	configPayload, err := json.Marshal(params.Config)
	if err != nil {
		return nil, badCollectorPayload("序列化连接配置失败", err)
	}
	metadataPayload, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, badCollectorPayload("序列化连接元数据失败", err)
	}
	defaultAcquisitionPayload, err := json.Marshal(params.DefaultAcquisition)
	if err != nil {
		return nil, badCollectorPayload("序列化默认采集参数失败", err)
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("开启连接更新事务失败", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE data_collector_connections SET name=$3, config=$4::jsonb, metadata=$5::jsonb, is_enabled=$6, default_acquisition=$7::jsonb, updated_by=$8, updated_at=now() WHERE id=$1 AND project_id=$2`, params.ID, params.ProjectID, params.Name, string(configPayload), string(metadataPayload), params.IsEnabled, string(defaultAcquisitionPayload), params.UserID)
	if err != nil {
		return nil, translateCollectorWriteError("更新工业采集连接失败", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "工业采集连接不存在")
	}
	if err := writeCollectorSecrets(ctx, tx, params.ID, params.Secrets, params.DeleteSecretKeys); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("提交工业采集连接更新事务失败", err)
	}
	return r.GetConnection(ctx, params.ProjectID, params.ID)
}

func (r *CollectorRepository) DeleteConnection(ctx context.Context, projectID, connectionID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM data_collector_connections WHERE id=$1 AND project_id=$2`, connectionID, projectID)
	if err != nil {
		return translateCollectorWriteError("删除工业采集连接失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "工业采集连接不存在")
	}
	return nil
}

const collectorConnectionSelect = `
	SELECT collector.id, collector.project_id, collector.name, collector.code, 'unknown', collector.is_enabled, collector.default_acquisition, collector.display_order,
	       collector.protocol_family, collector.driver_id, collector.driver_version, collector.schema_version,
	       collector.config, collector.metadata,
	       COALESCE((SELECT jsonb_object_agg(secret_key, true) FROM data_collector_connection_secrets secret WHERE secret.connection_id = collector.id), '{}'::jsonb),
	       (SELECT task.status FROM collector_dev_tasks task WHERE task.project_id = collector.project_id AND task.connection_id = collector.id AND task.operation = 'connection.test' ORDER BY task.created_at DESC, task.id DESC LIMIT 1),
	       (SELECT COALESCE(task.finished_at, task.created_at) FROM collector_dev_tasks task WHERE task.project_id = collector.project_id AND task.connection_id = collector.id AND task.operation = 'connection.test' ORDER BY task.created_at DESC, task.id DESC LIMIT 1),
	       collector.created_at, collector.updated_at
	FROM data_collector_connections collector`

type unifiedCollectorRow interface{ Scan(...any) error }

func scanCollectorConnection(row unifiedCollectorRow) (CollectorConnectionRecord, error) {
	var record CollectorConnectionRecord
	var configPayload, metadataPayload, defaultAcquisitionPayload, secretStatusPayload []byte
	err := row.Scan(&record.ID, &record.ProjectID, &record.Name, &record.Code, &record.Status, &record.IsEnabled, &defaultAcquisitionPayload, &record.DisplayOrder, &record.ProtocolFamily, &record.DriverID, &record.DriverVersion, &record.SchemaVersion, &configPayload, &metadataPayload, &secretStatusPayload, &record.LastTestStatus, &record.LastTestedAt, &record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		return CollectorConnectionRecord{}, err
	}
	if err := json.Unmarshal(configPayload, &record.Config); err != nil {
		return CollectorConnectionRecord{}, wrapUnifiedCollectorRepositoryError("解析连接配置失败", err)
	}
	if err := json.Unmarshal(metadataPayload, &record.Metadata); err != nil {
		return CollectorConnectionRecord{}, wrapUnifiedCollectorRepositoryError("解析连接元数据失败", err)
	}
	if err := json.Unmarshal(defaultAcquisitionPayload, &record.DefaultAcquisition); err != nil {
		return CollectorConnectionRecord{}, wrapUnifiedCollectorRepositoryError("解析默认采集参数失败", err)
	}
	if err := json.Unmarshal(secretStatusPayload, &record.SecretStatus); err != nil {
		return CollectorConnectionRecord{}, wrapUnifiedCollectorRepositoryError("解析连接密钥状态失败", err)
	}
	return record, nil
}

func scanCollectorConnectionWithTotal(row unifiedCollectorRow) (CollectorConnectionRecord, int, error) {
	var record CollectorConnectionRecord
	var configPayload, metadataPayload, defaultAcquisitionPayload, secretStatusPayload []byte
	var total int
	err := row.Scan(&record.ID, &record.ProjectID, &record.Name, &record.Code, &record.Status, &record.IsEnabled, &defaultAcquisitionPayload, &record.DisplayOrder, &record.ProtocolFamily, &record.DriverID, &record.DriverVersion, &record.SchemaVersion, &configPayload, &metadataPayload, &secretStatusPayload, &record.LastTestStatus, &record.LastTestedAt, &record.CreatedAt, &record.UpdatedAt, &total)
	if err != nil {
		return CollectorConnectionRecord{}, 0, wrapUnifiedCollectorRepositoryError("扫描工业采集连接失败", err)
	}
	if err := json.Unmarshal(configPayload, &record.Config); err != nil {
		return CollectorConnectionRecord{}, 0, wrapUnifiedCollectorRepositoryError("解析连接配置失败", err)
	}
	if err := json.Unmarshal(metadataPayload, &record.Metadata); err != nil {
		return CollectorConnectionRecord{}, 0, wrapUnifiedCollectorRepositoryError("解析连接元数据失败", err)
	}
	if err := json.Unmarshal(defaultAcquisitionPayload, &record.DefaultAcquisition); err != nil {
		return CollectorConnectionRecord{}, 0, wrapUnifiedCollectorRepositoryError("解析默认采集参数失败", err)
	}
	if err := json.Unmarshal(secretStatusPayload, &record.SecretStatus); err != nil {
		return CollectorConnectionRecord{}, 0, wrapUnifiedCollectorRepositoryError("解析连接密钥状态失败", err)
	}
	return record, total, nil
}

func writeCollectorSecrets(ctx context.Context, tx pgx.Tx, connectionID string, secrets []EncryptedCollectorSecret, deleteKeys []string) error {
	for _, key := range deleteKeys {
		if _, err := tx.Exec(ctx, `DELETE FROM data_collector_connection_secrets WHERE connection_id=$1 AND secret_key=$2`, connectionID, key); err != nil {
			return wrapUnifiedCollectorRepositoryError("删除工业采集连接密钥失败", err)
		}
	}
	for _, secret := range secrets {
		_, err := tx.Exec(ctx, `INSERT INTO data_collector_connection_secrets (connection_id, secret_key, encrypted_value, encryption_key_version) VALUES ($1,$2,$3,$4) ON CONFLICT (connection_id, secret_key) DO UPDATE SET encrypted_value=EXCLUDED.encrypted_value, encryption_key_version=EXCLUDED.encryption_key_version, updated_at=now()`, connectionID, secret.Key, secret.Value, secret.KeyVersion)
		if err != nil {
			return translateCollectorWriteError("保存工业采集连接密钥失败", err)
		}
	}
	return nil
}

func translateCollectorWriteError(message string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "data_collector_connections_project_name_key":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "工业采集连接名称已存在")
		case "data_collector_connections_project_code_key":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "工业采集连接编码已存在")
		case "data_collector_connections_driver_id_check":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "工业采集驱动标识无效")
		case "data_collector_points_connection_name_key":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "变量名称已存在")
		case "data_collector_points_connection_address_key":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "变量地址已存在")
		case "data_collector_points_connection_code_key":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "变量编码已存在")
		}
	}
	return wrapUnifiedCollectorRepositoryError(message, err)
}

func badCollectorPayload(message string, err error) error {
	return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, message, err)
}
func wrapUnifiedCollectorRepositoryError(message string, err error) error {
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}
