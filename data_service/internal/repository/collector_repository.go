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
	ID             string
	ProjectID      string
	Name           string
	Status         string
	Enabled        bool
	DisplayOrder   int
	ProtocolFamily string
	DriverID       string
	DriverVersion  string
	SchemaVersion  int
	Config         map[string]any
	Metadata       map[string]any
	SecretStatus   map[string]bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CollectorConnectionListFilter struct {
	Page, PageSize                                      int
	Search, ProtocolFamily, DriverID, SortBy, SortOrder string
	Enabled                                             *bool
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
	ID, ProjectID, UserID, Name, ProtocolFamily, DriverID, DriverVersion string
	SchemaVersion                                                        int
	Config, Metadata                                                     map[string]any
	Secrets                                                              []EncryptedCollectorSecret
}

type UpdateCollectorConnectionParams struct {
	ID, ProjectID, UserID, Name string
	Enabled                     bool
	Config, Metadata            map[string]any
	Secrets                     []EncryptedCollectorSecret
	DeleteSecretKeys            []string
}

type CollectorRepository struct{ pool *pgxpool.Pool }

func NewCollectorRepository(pool *pgxpool.Pool) *CollectorRepository {
	return &CollectorRepository{pool: pool}
}

func (r *CollectorRepository) ListConnections(ctx context.Context, projectID string, filter CollectorConnectionListFilter) ([]CollectorConnectionRecord, int, error) {
	conditions := []string{"conn.project_id = $1", "conn.type = 'collector'", "conn.category = 'industrial'"}
	args := []any{projectID}
	add := func(condition string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(condition, len(args)))
	}
	if value := strings.TrimSpace(filter.Search); value != "" {
		add("(conn.name ILIKE $%[1]d OR collector.driver_id ILIKE $%[1]d OR collector.protocol_family ILIKE $%[1]d)", "%"+value+"%")
	}
	if value := strings.TrimSpace(filter.ProtocolFamily); value != "" {
		add("collector.protocol_family = $%d", value)
	}
	if value := strings.TrimSpace(filter.DriverID); value != "" {
		add("collector.driver_id = $%d", value)
	}
	if filter.Enabled != nil {
		add("conn.is_enabled = $%d", *filter.Enabled)
	}
	orderColumn := map[string]string{"name": "conn.name", "driverId": "collector.driver_id", "createdAt": "conn.created_at", "updatedAt": "collector.updated_at"}[filter.SortBy]
	if orderColumn == "" {
		orderColumn = "conn.display_order"
	}
	orderDirection := "ASC"
	if strings.EqualFold(filter.SortOrder, "desc") {
		orderDirection = "DESC"
	}
	args = append(args, filter.PageSize, (filter.Page-1)*filter.PageSize)
	query := fmt.Sprintf(`
		SELECT conn.id, conn.project_id, conn.name, conn.status, conn.is_enabled, conn.display_order,
		       collector.protocol_family, collector.driver_id, collector.driver_version, collector.schema_version,
		       collector.config, collector.metadata,
		       COALESCE((SELECT jsonb_object_agg(secret_key, true) FROM data_collector_connection_secrets secret WHERE secret.connection_id = conn.id), '{}'::jsonb),
		       conn.created_at, collector.updated_at, COUNT(*) OVER()::int
		FROM data_connections conn
		JOIN data_collector_connections collector ON collector.connection_id = conn.id AND collector.project_id = conn.project_id
		WHERE %s
		ORDER BY %s %s, conn.id ASC
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
	record, err := scanCollectorConnection(r.pool.QueryRow(ctx, collectorConnectionSelect+` WHERE conn.project_id = $1 AND conn.id = $2 AND conn.type = 'collector' AND conn.category = 'industrial'`, projectID, connectionID))
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

func (r *CollectorRepository) CreateConnection(ctx context.Context, params CreateCollectorConnectionParams) (*CollectorConnectionRecord, error) {
	configPayload, err := json.Marshal(params.Config)
	if err != nil {
		return nil, badCollectorPayload("序列化连接配置失败", err)
	}
	metadataPayload, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, badCollectorPayload("序列化连接元数据失败", err)
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("开启连接创建事务失败", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO data_connections (id, project_id, name, type, category, status, created_by, updated_by) VALUES ($1,$2,$3,'collector','industrial','unknown',$4,$4)`, params.ID, params.ProjectID, params.Name, params.UserID)
	if err != nil {
		return nil, translateCollectorWriteError("创建工业采集连接主表失败", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO data_collector_connections (connection_id, project_id, protocol_family, driver_id, driver_version, schema_version, config, metadata) VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb)`, params.ID, params.ProjectID, params.ProtocolFamily, params.DriverID, params.DriverVersion, params.SchemaVersion, string(configPayload), string(metadataPayload))
	if err != nil {
		return nil, translateCollectorWriteError("创建工业采集连接扩展失败", err)
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
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("开启连接更新事务失败", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE data_connections SET name=$3, is_enabled=$4, updated_by=$5, updated_at=now() WHERE id=$1 AND project_id=$2 AND type='collector' AND category='industrial'`, params.ID, params.ProjectID, params.Name, params.Enabled, params.UserID)
	if err != nil {
		return nil, translateCollectorWriteError("更新工业采集连接主表失败", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "工业采集连接不存在")
	}
	_, err = tx.Exec(ctx, `UPDATE data_collector_connections SET config=$3::jsonb, metadata=$4::jsonb, updated_at=now() WHERE connection_id=$1 AND project_id=$2`, params.ID, params.ProjectID, string(configPayload), string(metadataPayload))
	if err != nil {
		return nil, translateCollectorWriteError("更新工业采集连接扩展失败", err)
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
	tag, err := r.pool.Exec(ctx, `DELETE FROM data_connections WHERE id=$1 AND project_id=$2 AND type='collector' AND category='industrial'`, connectionID, projectID)
	if err != nil {
		return translateCollectorWriteError("删除工业采集连接失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "工业采集连接不存在")
	}
	return nil
}

const collectorConnectionSelect = `
	SELECT conn.id, conn.project_id, conn.name, conn.status, conn.is_enabled, conn.display_order,
	       collector.protocol_family, collector.driver_id, collector.driver_version, collector.schema_version,
	       collector.config, collector.metadata,
	       COALESCE((SELECT jsonb_object_agg(secret_key, true) FROM data_collector_connection_secrets secret WHERE secret.connection_id = conn.id), '{}'::jsonb),
	       conn.created_at, collector.updated_at
	FROM data_connections conn
	JOIN data_collector_connections collector ON collector.connection_id = conn.id AND collector.project_id = conn.project_id`

type unifiedCollectorRow interface{ Scan(...any) error }

func scanCollectorConnection(row unifiedCollectorRow) (CollectorConnectionRecord, error) {
	var record CollectorConnectionRecord
	var configPayload, metadataPayload, secretStatusPayload []byte
	err := row.Scan(&record.ID, &record.ProjectID, &record.Name, &record.Status, &record.Enabled, &record.DisplayOrder, &record.ProtocolFamily, &record.DriverID, &record.DriverVersion, &record.SchemaVersion, &configPayload, &metadataPayload, &secretStatusPayload, &record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		return CollectorConnectionRecord{}, err
	}
	if err := json.Unmarshal(configPayload, &record.Config); err != nil {
		return CollectorConnectionRecord{}, wrapUnifiedCollectorRepositoryError("解析连接配置失败", err)
	}
	if err := json.Unmarshal(metadataPayload, &record.Metadata); err != nil {
		return CollectorConnectionRecord{}, wrapUnifiedCollectorRepositoryError("解析连接元数据失败", err)
	}
	if err := json.Unmarshal(secretStatusPayload, &record.SecretStatus); err != nil {
		return CollectorConnectionRecord{}, wrapUnifiedCollectorRepositoryError("解析连接密钥状态失败", err)
	}
	return record, nil
}

func scanCollectorConnectionWithTotal(row unifiedCollectorRow) (CollectorConnectionRecord, int, error) {
	var record CollectorConnectionRecord
	var configPayload, metadataPayload, secretStatusPayload []byte
	var total int
	err := row.Scan(&record.ID, &record.ProjectID, &record.Name, &record.Status, &record.Enabled, &record.DisplayOrder, &record.ProtocolFamily, &record.DriverID, &record.DriverVersion, &record.SchemaVersion, &configPayload, &metadataPayload, &secretStatusPayload, &record.CreatedAt, &record.UpdatedAt, &total)
	if err != nil {
		return CollectorConnectionRecord{}, 0, wrapUnifiedCollectorRepositoryError("扫描工业采集连接失败", err)
	}
	if err := json.Unmarshal(configPayload, &record.Config); err != nil {
		return CollectorConnectionRecord{}, 0, wrapUnifiedCollectorRepositoryError("解析连接配置失败", err)
	}
	if err := json.Unmarshal(metadataPayload, &record.Metadata); err != nil {
		return CollectorConnectionRecord{}, 0, wrapUnifiedCollectorRepositoryError("解析连接元数据失败", err)
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
		case "data_connections_project_name_key":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "工业采集连接名称已存在")
		case "data_collector_connections_driver_id_check":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "工业采集驱动标识无效")
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
