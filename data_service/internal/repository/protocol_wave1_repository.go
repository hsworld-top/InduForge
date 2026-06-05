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

// ProtocolConnectionRecord 表示协议配置创建后返回的连接基础信息。
type ProtocolConnectionRecord struct {
	ID        string
	ProjectID string
	Name      string
	Type      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// KafkaPreviewRecord 表示 Kafka 预览接口返回的一行消息。
type KafkaPreviewRecord struct {
	Topic   string
	Payload string
}

// CreateKafkaConfigParams 描述 Kafka 配置落库参数。
type CreateKafkaConfigParams struct {
	ProjectID     string
	UserID        string
	Name          string
	Status        string
	Brokers       string
	Topic         string
	ConsumerGroup string
	StartPosition string
	Options       map[string]any
}

// CreateHTTPConfigParams 描述 HTTP 配置落库参数。
type CreateHTTPConfigParams struct {
	ProjectID   string
	UserID      string
	Name        string
	Status      string
	Description string
}

// CreateWebSocketConfigParams 描述 WebSocket 配置落库参数。
type CreateWebSocketConfigParams struct {
	ProjectID           string
	UserID              string
	Name                string
	Status              string
	URL                 string
	Topic               *string
	Headers             map[string]any
	HeartbeatIntervalMS int
}

// CreateRedisConfigParams 描述 Redis 配置落库参数。
type CreateRedisConfigParams struct {
	ProjectID  string
	UserID     string
	Name       string
	Status     string
	Address    string
	DB         int
	Username   *string
	Password   *string
	KeyPattern string
	Mode       string
	Options    map[string]any
}

// ProtocolWave1Repository 负责第一波协议配置（kafka/http/ws/redis）的参数化 SQL。
// 说明：Phase 1 对这批协议只冻结配置对象与 artifact 契约，除 Kafka mock preview 外，
// 不把它们扩成完整运行态采集栈。
type ProtocolWave1Repository struct {
	pool *pgxpool.Pool
}

// NewProtocolWave1Repository 创建第一波协议仓储。
func NewProtocolWave1Repository(pool *pgxpool.Pool) *ProtocolWave1Repository {
	return &ProtocolWave1Repository{pool: pool}
}

// CreateKafkaConfig 创建 Kafka 配置并写入 data_connections/data_kafka_configs。
func (r *ProtocolWave1Repository) CreateKafkaConfig(ctx context.Context, params CreateKafkaConfigParams) (*ProtocolConnectionRecord, error) {
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
		Status:    params.Status,
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

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 Kafka 配置事务失败", err)
	}
	return record, nil
}

// PreviewKafkaTopic 返回 Kafka 预览消息。
// 说明：当前版本先返回 mock 预览数据，后续接入真实 consumer 后可替换此实现。
func (r *ProtocolWave1Repository) PreviewKafkaTopic(ctx context.Context, projectID, connectionID string) ([]KafkaPreviewRecord, error) {
	var topic string
	err := r.pool.QueryRow(ctx, `
		SELECT cfg.topic
		FROM data_kafka_configs cfg
		JOIN data_connections conn ON conn.id = cfg.connection_id
		WHERE conn.project_id = $1
		  AND conn.id = $2
		  AND conn.type = 'kafka'
	`, projectID, connectionID).Scan(&topic)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Kafka 配置不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 Kafka 配置失败", err)
	}

	return []KafkaPreviewRecord{
		{
			Topic:   topic,
			Payload: `{"source":"kafka-preview-mock","status":"ok"}`,
		},
	}, nil
}

// CreateHTTPConfig 创建 HTTP 配置。
func (r *ProtocolWave1Repository) CreateHTTPConfig(ctx context.Context, params CreateHTTPConfigParams) (*ProtocolConnectionRecord, error) {
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
		Status:    params.Status,
		Metadata: map[string]any{
			"mode":        "workbench",
			"description": params.Description,
		},
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 HTTP 配置事务失败", err)
	}
	return record, nil
}

// CreateWebSocketConfig 创建 WebSocket 配置。
func (r *ProtocolWave1Repository) CreateWebSocketConfig(ctx context.Context, params CreateWebSocketConfigParams) (*ProtocolConnectionRecord, error) {
	headersPayload, err := marshalProtocolJSONObject(params.Headers, true)
	if err != nil {
		return nil, err
	}

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
		Status:    params.Status,
		Metadata: map[string]any{
			"url":                 params.URL,
			"topic":               params.Topic,
			"headers":             cloneProtocolMap(params.Headers),
			"heartbeatIntervalMs": params.HeartbeatIntervalMS,
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO data_websocket_configs (
			connection_id,
			url,
			topic,
			headers,
			heartbeat_interval_ms
		)
		VALUES ($1, $2, $3, $4::jsonb, $5)
	`, record.ID, params.URL, params.Topic, headersPayload, params.HeartbeatIntervalMS)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 WebSocket 配置失败", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 WebSocket 配置事务失败", err)
	}
	return record, nil
}

// CreateRedisConfig 创建 Redis 配置。
func (r *ProtocolWave1Repository) CreateRedisConfig(ctx context.Context, params CreateRedisConfigParams) (*ProtocolConnectionRecord, error) {
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
		Status:    params.Status,
		Metadata: map[string]any{
			"address":    params.Address,
			"db":         params.DB,
			"username":   params.Username,
			"password":   params.Password,
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
			password,
			key_pattern,
			mode,
			options
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
	`, record.ID, params.Address, params.DB, params.Username, params.Password, params.KeyPattern, params.Mode, optionsPayload)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 Redis 配置失败", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 Redis 配置事务失败", err)
	}
	return record, nil
}

type createConnectionTxParams struct {
	ProjectID string
	UserID    string
	Name      string
	Type      string
	Status    string
	Metadata  map[string]any
}

// createConnectionTx 统一写入 data_connections。
// 说明：所有 wave1 协议都先落连接主表，再写各自配置表，方便复用项目级连接治理能力。
func (r *ProtocolWave1Repository) createConnectionTx(ctx context.Context, tx pgx.Tx, params createConnectionTxParams) (*ProtocolConnectionRecord, error) {
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
			status,
			metadata,
			display_order,
			created_by,
			updated_by
		)
		VALUES ($1, $2, $3, 'protocol', $4, $5::jsonb,
			COALESCE((SELECT MAX(display_order) + 1 FROM data_connections WHERE project_id = $1), 0),
			$6, $6)
		RETURNING id, project_id, name, type, status, created_at, updated_at
	`, params.ProjectID, params.Name, params.Type, params.Status, string(metadataPayload), params.UserID).Scan(
		&record.ID,
		&record.ProjectID,
		&record.Name,
		&record.Type,
		&record.Status,
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
