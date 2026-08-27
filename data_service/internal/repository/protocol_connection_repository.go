package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/security"
)

// ProtocolConnectionRecord 表示协议配置创建后返回的连接基础信息。
type ProtocolConnectionRecord struct {
	ID        string
	ProjectID string
	Name      string
	Type      string
	IsEnabled bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateKafkaConfigParams 描述 Kafka 配置落库参数。
type CreateKafkaConfigParams struct {
	ProjectID       string
	UserID          string
	Name            string
	IsEnabled       *bool
	Brokers         string
	Topic           string
	ConsumerGroup   string
	StartPosition   string
	Options         map[string]any
	Secrets         map[string]string
	ClearSecretKeys []string
}

// CreateHTTPConfigParams 描述 HTTP 配置落库参数。
type CreateHTTPConfigParams struct {
	ProjectID   string
	UserID      string
	Name        string
	IsEnabled   *bool
	Description string
}

// CreateWebSocketConfigParams 描述 WebSocket 配置落库参数。
type CreateWebSocketConfigParams struct {
	ProjectID   string
	UserID      string
	Name        string
	IsEnabled   *bool
	Description string
}

// CreateRedisConfigParams 描述 Redis 配置落库参数。
type CreateRedisConfigParams struct {
	ProjectID       string
	UserID          string
	Name            string
	IsEnabled       *bool
	Address         string
	DB              int
	Username        *string
	KeyPattern      string
	Mode            string
	Options         map[string]any
	Secrets         map[string]string
	ClearSecretKeys []string
}

// ProtocolConnectionRepository 负责协议连接配置（kafka/http/ws/redis）的参数化 SQL。
// 说明：开发态协议 对这批协议只冻结配置对象与 artifact 契约，除 Kafka mock preview 外，
// 不把它们扩成完整运行态采集栈。
type ProtocolConnectionRepository struct {
	pool   *pgxpool.Pool
	cipher *security.ConnectionSecretCipher
}

// NewProtocolConnectionRepository 创建协议连接仓储。
func NewProtocolConnectionRepository(pool *pgxpool.Pool, ciphers ...*security.ConnectionSecretCipher) *ProtocolConnectionRepository {
	repository := &ProtocolConnectionRepository{pool: pool}
	if len(ciphers) > 0 {
		repository.cipher = ciphers[0]
	}
	return repository
}

// CreateKafkaConfig 创建 Kafka 配置并写入 data_connections/data_kafka_configs。
func (r *ProtocolConnectionRepository) CreateKafkaConfig(ctx context.Context, params CreateKafkaConfigParams) (*ProtocolConnectionRecord, error) {
	optionsPayload, err := marshalProtocolJSONObject(params.Options, true)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 Kafka 配置事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	record, err := r.createConnectionTx(ctx, tx, createConnectionTxParams{
		ProjectID: params.ProjectID,
		UserID:    params.UserID,
		Name:      params.Name,
		Type:      "kafka",
		IsEnabled: params.IsEnabled,
		Metadata: map[string]any{
			"brokers": params.Brokers,
			"options": cloneProtocolMap(params.Options),
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO data_kafka_configs (
			connection_id,
			brokers,
			topic,
			consumer_group,
			start_position,
			options
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)
	`, record.ID, params.Brokers, nullIfBlank(params.Topic), nullIfBlank(params.ConsumerGroup), params.StartPosition, optionsPayload)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 Kafka 配置失败", err)
	}
	if err := applyPlainConnectionSecretsTx(ctx, tx, r.cipher, record.ID, params.Secrets, params.ClearSecretKeys); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 Kafka 配置事务失败", err)
	}
	return record, nil
}

// CreateHTTPConfig 创建 HTTP 配置。
func (r *ProtocolConnectionRepository) CreateHTTPConfig(ctx context.Context, params CreateHTTPConfigParams) (*ProtocolConnectionRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 HTTP 配置事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	record, err := r.createConnectionTx(ctx, tx, createConnectionTxParams{
		ProjectID: params.ProjectID,
		UserID:    params.UserID,
		Name:      params.Name,
		Type:      "http",
		IsEnabled: params.IsEnabled,
		Metadata: map[string]any{
			"mode":        "workbench",
			"description": params.Description,
		},
	})
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO data_http_configs (connection_id, base_url, method, headers, timeout_ms) VALUES ($1, '', 'GET', '{}'::jsonb, 30000)`, record.ID); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 HTTP 配置失败", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 HTTP 配置事务失败", err)
	}
	return record, nil
}

// CreateWebSocketConfig 创建 WebSocket 配置。
func (r *ProtocolConnectionRepository) CreateWebSocketConfig(ctx context.Context, params CreateWebSocketConfigParams) (*ProtocolConnectionRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 WebSocket 配置事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	record, err := r.createConnectionTx(ctx, tx, createConnectionTxParams{
		ProjectID: params.ProjectID,
		UserID:    params.UserID,
		Name:      params.Name,
		Type:      "websocket",
		IsEnabled: params.IsEnabled,
		Metadata: map[string]any{
			"mode":        "workbench",
			"description": params.Description,
		},
	})
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO data_websocket_configs (connection_id, url, headers, heartbeat_interval_ms) VALUES ($1, NULL, '{}'::jsonb, 30000)`, record.ID); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 WebSocket 配置失败", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 WebSocket 配置事务失败", err)
	}
	return record, nil
}

// CreateRedisConfig 创建 Redis 配置。
func (r *ProtocolConnectionRepository) CreateRedisConfig(ctx context.Context, params CreateRedisConfigParams) (*ProtocolConnectionRecord, error) {
	optionsPayload, err := marshalProtocolJSONObject(params.Options, true)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 Redis 配置事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	record, err := r.createConnectionTx(ctx, tx, createConnectionTxParams{
		ProjectID: params.ProjectID,
		UserID:    params.UserID,
		Name:      params.Name,
		Type:      "redis",
		IsEnabled: params.IsEnabled,
		Metadata: map[string]any{
			"address":    params.Address,
			"db":         params.DB,
			"username":   params.Username,
			"keyPattern": params.KeyPattern,
			"mode":       params.Mode,
			"options":    cloneProtocolMap(params.Options),
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO data_redis_configs (
			connection_id,
			address,
			db,
			username,
			key_pattern,
			mode,
			options
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
	`, record.ID, params.Address, params.DB, params.Username, params.KeyPattern, params.Mode, optionsPayload)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 Redis 配置失败", err)
	}
	if err := applyPlainConnectionSecretsTx(ctx, tx, r.cipher, record.ID, params.Secrets, params.ClearSecretKeys); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 Redis 配置事务失败", err)
	}
	return record, nil
}

func (r *ProtocolConnectionRepository) updateConnectionTx(ctx context.Context, tx pgx.Tx, connectionID, expectedType string, params createConnectionTxParams) (*ProtocolConnectionRecord, error) {
	metadataPayload, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "序列化协议连接配置失败", err)
	}
	record := ProtocolConnectionRecord{}
	err = tx.QueryRow(ctx, `UPDATE data_connections SET name=$3,is_enabled=$4,metadata=$5::jsonb,updated_by=$6,updated_at=now() WHERE project_id=$1 AND id=$2 AND type=$7 RETURNING id,project_id,name,type,is_enabled,created_at,updated_at`, params.ProjectID, connectionID, params.Name, connectionEnabledValue(params.IsEnabled), string(metadataPayload), params.UserID, expectedType).Scan(&record.ID, &record.ProjectID, &record.Name, &record.Type, &record.IsEnabled, &record.CreatedAt, &record.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "协议接入源不存在")
	}
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新协议接入源失败", err)
	}
	return &record, nil
}

func (r *ProtocolConnectionRepository) UpdateKafkaConfig(ctx context.Context, connectionID string, params CreateKafkaConfigParams) (*ProtocolConnectionRecord, error) {
	optionsPayload, err := marshalProtocolJSONObject(params.Options, true)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer rollbackProtocolTxQuietly(ctx, tx)
	record, err := r.updateConnectionTx(ctx, tx, connectionID, "kafka", createConnectionTxParams{ProjectID: params.ProjectID, UserID: params.UserID, Name: params.Name, IsEnabled: params.IsEnabled, Metadata: map[string]any{"brokers": params.Brokers, "options": cloneProtocolMap(params.Options)}})
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `UPDATE data_kafka_configs SET brokers=$2,topic=$3,consumer_group=$4,start_position=$5,options=$6::jsonb,updated_at=now() WHERE connection_id=$1`, connectionID, params.Brokers, nullIfBlank(params.Topic), nullIfBlank(params.ConsumerGroup), params.StartPosition, optionsPayload)
	if err != nil {
		return nil, err
	}
	if err := applyPlainConnectionSecretsTx(ctx, tx, r.cipher, connectionID, params.Secrets, params.ClearSecretKeys); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return record, nil
}

func (r *ProtocolConnectionRepository) UpdateSimpleConfig(ctx context.Context, connectionID, protocolType, projectID, userID, name string, isEnabled *bool, metadata map[string]any) (*ProtocolConnectionRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer rollbackProtocolTxQuietly(ctx, tx)
	record, err := r.updateConnectionTx(ctx, tx, connectionID, protocolType, createConnectionTxParams{ProjectID: projectID, UserID: userID, Name: name, IsEnabled: isEnabled, Metadata: metadata})
	if err != nil {
		return nil, err
	}
	if protocolType == "http" {
		if _, err := tx.Exec(ctx, `INSERT INTO data_http_configs (connection_id, base_url, method, headers, timeout_ms) VALUES ($1, '', 'GET', '{}'::jsonb, 30000) ON CONFLICT (connection_id) DO UPDATE SET updated_at=now()`, connectionID); err != nil {
			return nil, err
		}
	} else if protocolType == "websocket" {
		if _, err := tx.Exec(ctx, `INSERT INTO data_websocket_configs (connection_id, url, headers, heartbeat_interval_ms) VALUES ($1, NULL, '{}'::jsonb, 30000) ON CONFLICT (connection_id) DO UPDATE SET updated_at=now()`, connectionID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return record, nil
}

func (r *ProtocolConnectionRepository) UpdateRedisConfig(ctx context.Context, connectionID string, params CreateRedisConfigParams) (*ProtocolConnectionRecord, error) {
	optionsPayload, err := marshalProtocolJSONObject(params.Options, true)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer rollbackProtocolTxQuietly(ctx, tx)
	record, err := r.updateConnectionTx(ctx, tx, connectionID, "redis", createConnectionTxParams{ProjectID: params.ProjectID, UserID: params.UserID, Name: params.Name, IsEnabled: params.IsEnabled, Metadata: map[string]any{"address": params.Address, "db": params.DB, "username": params.Username, "keyPattern": params.KeyPattern, "mode": params.Mode, "options": cloneProtocolMap(params.Options)}})
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `UPDATE data_redis_configs SET address=$2,db=$3,username=$4,key_pattern=$5,mode=$6,options=$7::jsonb,updated_at=now() WHERE connection_id=$1`, connectionID, params.Address, params.DB, params.Username, params.KeyPattern, params.Mode, optionsPayload)
	if err != nil {
		return nil, err
	}
	if err := applyPlainConnectionSecretsTx(ctx, tx, r.cipher, connectionID, params.Secrets, params.ClearSecretKeys); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return record, nil
}

type createConnectionTxParams struct {
	ProjectID string
	UserID    string
	Name      string
	Type      string
	IsEnabled *bool
	Metadata  map[string]any
}

// createConnectionTx 统一写入 data_connections。
// 说明：所有 协议连接 协议都先落连接主表，再写各自配置表，方便复用项目级连接治理能力。
func (r *ProtocolConnectionRepository) createConnectionTx(ctx context.Context, tx pgx.Tx, params createConnectionTxParams) (*ProtocolConnectionRecord, error) {
	metadataPayload, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "序列化协议连接元数据失败", err)
	}

	record := ProtocolConnectionRecord{}
	if err := lockProjectConnectionOrder(ctx, tx, params.ProjectID); err != nil {
		return nil, err
	}
	if err := normalizeProjectConnectionDisplayOrder(ctx, tx, params.ProjectID); err != nil {
		return nil, err
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO data_connections (
			project_id,
			name,
			type,
			category,
			is_enabled,
			metadata,
			display_order,
			created_by,
			updated_by
		)
		VALUES ($1, $2, $3, 'protocol', $4, $5::jsonb,
			COALESCE((SELECT MAX(display_order) + 1 FROM data_connections WHERE project_id = $1), 0),
			$6, $6)
		RETURNING id, project_id, name, type, is_enabled, created_at, updated_at
	`, params.ProjectID, params.Name, params.Type, connectionEnabledValue(params.IsEnabled), string(metadataPayload), params.UserID).Scan(
		&record.ID,
		&record.ProjectID,
		&record.Name,
		&record.Type,
		&record.IsEnabled,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, translateConnectionWriteError("写入协议连接失败", err)
	}
	return &record, nil
}

func marshalProtocolJSONObject(input map[string]any, emptyAsObject bool) (any, error) {
	if input == nil {
		if emptyAsObject {
			return "{}", nil
		}
		return nil, nil
	}

	payload, err := json.Marshal(input)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "协议 JSON 配置格式无效", err)
	}
	return string(payload), nil
}

func cloneProtocolMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func nullIfBlank(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func rollbackProtocolTxQuietly(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}
