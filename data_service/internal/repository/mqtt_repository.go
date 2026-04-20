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

// MqttConnectionRecord 表示 MQTT 连接在仓储层的返回结构。
type MqttConnectionRecord struct {
	ID        string
	ProjectID string
	Name      string
	Type      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// MqttConnectionStatusRecord 表示连接状态返回结构。
type MqttConnectionStatusRecord struct {
	Status string
}

// MqttMessageRecord 表示 MQTT 消息列表返回结构。
type MqttMessageRecord struct {
	ID             int64
	SubscriptionID string
	Topic          string
	Payload        string
	QOS            int
	ReceivedAt     time.Time
}

// CreateMqttConnectionParams 描述创建 MQTT 连接时的数据库入参。
type CreateMqttConnectionParams struct {
	ProjectID        string
	UserID           string
	Name             string
	Status           string
	BrokerURL        string
	Protocol         string
	Port             int
	ClientID         *string
	Username         *string
	Password         *string
	Keepalive        int
	CleanSession     bool
	QOS              int
	ReconnectPeriod  int
	ConnectTimeoutMS int
	Will             map[string]any
	SSLConfig        map[string]any
}

// MqttRepository 负责 data_mqtt_* 相关参数化 SQL。
// CreateMqttMessageParams 描述 MQTT 预览运行时落库最近消息时的写入参数。
type CreateMqttMessageParams struct {
	ProjectID      string
	ConnectionID   string
	SubscriptionID string
	Topic          string
	Payload        string
	QOS            int
	ReceivedAt     time.Time
	Metadata       map[string]any
	RetentionLimit int
}

type MqttRepository struct {
	pool *pgxpool.Pool
}

// NewMqttRepository 创建 MQTT 仓储。
func NewMqttRepository(pool *pgxpool.Pool) *MqttRepository {
	return &MqttRepository{pool: pool}
}

// CreateConnection 创建 MQTT 连接并同步写入 data_connections / data_mqtt_configs。
// 查询路径说明：
// 1. data_connections 主写表，后续启动与状态查询都以 (project_id, id) 为入口。
// 2. data_mqtt_configs 通过 connection_id 一对一关联，依赖主键索引快速命中。
func (r *MqttRepository) CreateConnection(ctx context.Context, params CreateMqttConnectionParams) (*MqttConnectionRecord, error) {
	metadataBytes, err := json.Marshal(map[string]any{
		"brokerUrl": params.BrokerURL,
		"protocol":  params.Protocol,
		"port":      params.Port,
	})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "序列化 MQTT 连接元数据失败", err)
	}

	willPayload, err := marshalMqttJSONObject(params.Will)
	if err != nil {
		return nil, err
	}
	sslPayload, err := marshalMqttJSONObject(params.SSLConfig)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 MQTT 连接创建事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)

	record := MqttConnectionRecord{}
	err = tx.QueryRow(ctx, `
		INSERT INTO data_connections (
			project_id,
			name,
			type,
			category,
			status,
			metadata,
			created_by,
			updated_by
		)
		VALUES ($1, $2, 'mqtt', 'message', $3, $4::jsonb, $5, $5)
		RETURNING id, project_id, name, type, status, created_at, updated_at
	`, params.ProjectID, params.Name, params.Status, string(metadataBytes), params.UserID).Scan(
		&record.ID,
		&record.ProjectID,
		&record.Name,
		&record.Type,
		&record.Status,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, wrapMqttWriteError("写入 MQTT 连接失败", err)
	}

	var willSQL any
	if len(willPayload) > 0 {
		willSQL = string(willPayload)
	}
	var sslSQL any
	if len(sslPayload) > 0 {
		sslSQL = string(sslPayload)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO data_mqtt_configs (
			connection_id,
			broker_url,
			protocol,
			port,
			client_id,
			username,
			password,
			keepalive,
			clean_session,
			qos,
			reconnect_period_ms,
			connect_timeout_ms,
			will,
			ssl_config
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13::jsonb, $14::jsonb
		)
	`, record.ID, params.BrokerURL, params.Protocol, params.Port, params.ClientID, params.Username, params.Password, params.Keepalive, params.CleanSession, params.QOS, params.ReconnectPeriod, params.ConnectTimeoutMS, willSQL, sslSQL)
	if err != nil {
		return nil, wrapMqttWriteError("写入 MQTT 连接配置失败", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 MQTT 连接创建事务失败", err)
	}

	return &record, nil
}

// StartConnection 将 MQTT 连接状态切换为 connected。
// 查询路径说明：按 (project_id, id, type='mqtt') 更新，避免跨项目误写。
func (r *MqttRepository) StartConnection(ctx context.Context, projectID, connectionID string) (*MqttConnectionStatusRecord, error) {
	var status string
	err := r.pool.QueryRow(ctx, `
		UPDATE data_connections
		SET status = 'connected',
			last_connected_at = now(),
			last_error_message = NULL,
			updated_at = now()
		WHERE project_id = $1
		  AND id = $2
		  AND type = 'mqtt'
		RETURNING status
	`, projectID, connectionID).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 连接不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "启动 MQTT 连接失败", err)
	}

	return &MqttConnectionStatusRecord{Status: status}, nil
}

// GetConnectionStatus 读取 MQTT 连接当前状态。
// 查询路径说明：按 (project_id, id, type='mqtt') 精确读取，复用 data_connections 主键路径。
func (r *MqttRepository) GetConnectionStatus(ctx context.Context, projectID, connectionID string) (*MqttConnectionStatusRecord, error) {
	var status string
	err := r.pool.QueryRow(ctx, `
		SELECT status
		FROM data_connections
		WHERE project_id = $1
		  AND id = $2
		  AND type = 'mqtt'
	`, projectID, connectionID).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 连接不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 MQTT 连接状态失败", err)
	}

	return &MqttConnectionStatusRecord{Status: status}, nil
}

// ListMessages 按 subscription 读取消息缓存。
// 查询路径说明：先校验 subscription 在当前项目内存在，再走 (project_id, subscription_id, received_at desc) 索引读取。
func (r *MqttRepository) ListMessages(ctx context.Context, projectID, subscriptionID string, limit int) ([]MqttMessageRecord, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM data_mqtt_subscriptions
			WHERE project_id = $1
			  AND id = $2
		)
	`, projectID, subscriptionID).Scan(&exists); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "校验 MQTT 订阅失败", err)
	}
	if !exists {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 订阅不存在")
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, subscription_id, topic, payload, qos, received_at
		FROM data_mqtt_messages
		WHERE project_id = $1
		  AND subscription_id = $2
		ORDER BY received_at DESC, id DESC
		LIMIT $3
	`, projectID, subscriptionID, limit)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 MQTT 消息失败", err)
	}
	defer rows.Close()

	messages := make([]MqttMessageRecord, 0)
	for rows.Next() {
		record := MqttMessageRecord{}
		if err := rows.Scan(
			&record.ID,
			&record.SubscriptionID,
			&record.Topic,
			&record.Payload,
			&record.QOS,
			&record.ReceivedAt,
		); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 MQTT 消息结果失败", err)
		}
		messages = append(messages, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 MQTT 消息结果失败", err)
	}

	return messages, nil
}

// CreateMessage 持久化一条 MQTT 预览消息，并按订阅保留策略裁剪旧消息。
func (r *MqttRepository) CreateMessage(ctx context.Context, params CreateMqttMessageParams) (*MqttMessageRecord, error) {
	metadataPayload, err := marshalMqttJSONObject(params.Metadata)
	if err != nil {
		return nil, err
	}

	receivedAt := params.ReceivedAt.UTC()
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}

	record := MqttMessageRecord{}
	err = r.pool.QueryRow(ctx, `
		INSERT INTO data_mqtt_messages (
			project_id,
			connection_id,
			subscription_id,
			topic,
			payload,
			qos,
			received_at,
			metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
		RETURNING id, subscription_id, topic, payload, qos, received_at
	`, params.ProjectID, params.ConnectionID, params.SubscriptionID, params.Topic, params.Payload, params.QOS, receivedAt, nullableMqttJSON(metadataPayload)).Scan(
		&record.ID,
		&record.SubscriptionID,
		&record.Topic,
		&record.Payload,
		&record.QOS,
		&record.ReceivedAt,
	)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "鍐欏叆 MQTT 娑堟伅澶辫触", err)
	}

	if params.RetentionLimit > 0 {
		if _, err := r.pool.Exec(ctx, `
			DELETE FROM data_mqtt_messages
			WHERE id IN (
				SELECT id
				FROM data_mqtt_messages
				WHERE project_id = $1
				  AND subscription_id = $2
				ORDER BY received_at DESC, id DESC
				OFFSET $3
			)
		`, params.ProjectID, params.SubscriptionID, params.RetentionLimit); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "瑁佸壀 MQTT 娑堟伅缂撳瓨澶辫触", err)
		}
	}

	return &record, nil
}

func marshalMqttJSONObject(input map[string]any) ([]byte, error) {
	if input == nil {
		return nil, nil
	}

	payload, err := json.Marshal(input)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT JSON 配置格式无效", err)
	}
	return payload, nil
}

func wrapMqttWriteError(message string, err error) error {
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}

func rollbackTxQuietly(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}
