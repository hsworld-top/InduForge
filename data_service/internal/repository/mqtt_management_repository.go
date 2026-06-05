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

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// MqttConnectionDetailRecord 表示带配置详情的 MQTT 连接投影。
type MqttConnectionDetailRecord struct {
	ID                    string
	ProjectID             string
	Name                  string
	Type                  string
	Status                string
	IsEnabled             bool
	RetryCount            int
	RetryIntervalMS       int
	HealthCheckIntervalMS int
	BrokerURL             string
	Protocol              string
	Port                  int
	ClientID              *string
	Username              *string
	Password              *string
	Keepalive             int
	CleanSession          bool
	QOS                   int
	ReconnectPeriodMS     int
	ConnectTimeoutMS      int
	Will                  map[string]any
	SSLConfig             map[string]any
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// MqttConnectionSummaryRecord 表示 MQTT 连接主表摘要。
type MqttConnectionSummaryRecord struct {
	ID        string
	ProjectID string
	Name      string
	Type      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// MqttSubscriptionRecord 表示 MQTT 订阅投影。
type MqttSubscriptionRecord struct {
	ID               string
	ProjectID        string
	ConnectionID     string
	GroupID          *string
	Name             string
	Topic            string
	QOS              int
	UsageMode        string
	Description      *string
	MessageRetention int
	DefaultBatchRule map[string]any
	Order            int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// MqttSubscriptionGroupRecord 表示 MQTT 订阅分组投影。
type MqttSubscriptionGroupRecord struct {
	ID           string
	ProjectID    string
	ConnectionID string
	Name         string
	ParentID     *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// MqttTagRecord 表示 MQTT 变量投影。
type MqttTagRecord struct {
	ID             string
	ProjectID      string
	SubscriptionID string
	Name           string
	Code           string
	Description    *string
	DataType       string
	ParseType      string
	ParseRule      string
	DefaultValue   *string
	Unit           *string
	Transform      *string
	Validation     map[string]any
	Order          int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// UpdateMqttConnectionParams 表示更新 MQTT 连接时的仓储参数。
type UpdateMqttConnectionParams struct {
	ProjectID             string
	ConnectionID          string
	UserID                string
	Name                  string
	Status                string
	IsEnabled             bool
	RetryCount            int
	RetryIntervalMS       int
	HealthCheckIntervalMS int
	BrokerURL             string
	Protocol              string
	Port                  int
	ClientID              *string
	Username              *string
	Password              *string
	Keepalive             int
	CleanSession          bool
	QOS                   int
	ReconnectPeriodMS     int
	ConnectTimeoutMS      int
	Will                  map[string]any
	SSLConfig             map[string]any
}

// CreateMqttSubscriptionParams 表示新建 MQTT 订阅的仓储参数。
type CreateMqttSubscriptionParams struct {
	ProjectID        string
	ConnectionID     string
	GroupID          *string
	UserID           string
	Name             string
	Topic            string
	QOS              int
	UsageMode        string
	Description      *string
	MessageRetention int
	Order            int
}

// UpdateMqttSubscriptionDefaultBatchRuleParams 表示更新订阅默认批量解析规则的仓储参数。
type UpdateMqttSubscriptionDefaultBatchRuleParams struct {
	ProjectID      string
	SubscriptionID string
	UserID         string
	Rule           map[string]any
}

// UpdateMqttSubscriptionParams 表示更新 MQTT 订阅的仓储参数。
type UpdateMqttSubscriptionParams struct {
	ProjectID        string
	SubscriptionID   string
	GroupID          *string
	HasGroupID       bool
	UserID           string
	Name             string
	Topic            string
	QOS              int
	UsageMode        string
	Description      *string
	MessageRetention int
	Order            int
}

// CreateMqttSubscriptionGroupParams 表示新建 MQTT 订阅分组的仓储参数。
type CreateMqttSubscriptionGroupParams struct {
	ProjectID    string
	ConnectionID string
	UserID       string
	Name         string
	ParentID     *string
}

// UpdateMqttSubscriptionGroupParams 表示更新 MQTT 订阅分组的仓储参数。
type UpdateMqttSubscriptionGroupParams struct {
	ProjectID string
	GroupID   string
	UserID    string
	Name      string
	ParentID  *string
}

// CreateMqttTagParams 表示新建变量的仓储参数。
type CreateMqttTagParams struct {
	ProjectID      string
	SubscriptionID string
	UserID         string
	Name           string
	Code           string
	Description    *string
	DataType       string
	ParseType      string
	ParseRule      string
	DefaultValue   *string
	Unit           *string
	Transform      *string
	Validation     map[string]any
	Order          int
}

// BatchMqttTagDataPointParams 表示批量创建 MQTT 变量时需要同步的数据点信息。
type BatchMqttTagDataPointParams struct {
	Tag          CreateMqttTagParams
	DataPath     string
	DataName     string
	SourceType   string
	SourceConfig map[string]any
	RefreshMode  string
	Status       string
}

// UpdateMqttTagParams 表示更新变量的仓储参数。
type UpdateMqttTagParams struct {
	ProjectID    string
	TagID        string
	UserID       string
	Name         string
	Code         string
	Description  *string
	DataType     string
	ParseType    string
	ParseRule    string
	DefaultValue *string
	Unit         *string
	Transform    *string
	Validation   map[string]any
	Order        int
}

// ListConnectionDetails 按项目分页查询 MQTT 连接详情。
func (r *MqttRepository) ListConnectionDetails(ctx context.Context, projectID string, page, pageSize int) ([]MqttConnectionDetailRecord, int, error) {
	page, pageSize = normalizePageAndSize(page, pageSize, 50, 200)

	var total int
	if err := r.pool.QueryRow(ctx, `
        SELECT COUNT(*)
        FROM data_connections
        WHERE project_id = $1 AND type = 'mqtt'
    `, projectID).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计 MQTT 连接失败", err)
	}

	rows, err := r.pool.Query(ctx, `
        SELECT
            conn.id,
            conn.project_id,
            conn.name,
            conn.type,
            conn.status,
            conn.is_enabled,
            conn.retry_count,
            conn.retry_interval_ms,
            conn.health_check_interval_ms,
            cfg.broker_url,
            cfg.protocol,
            cfg.port,
            cfg.client_id,
            cfg.username,
            cfg.password,
            cfg.keepalive,
            cfg.clean_session,
            cfg.qos,
            cfg.reconnect_period_ms,
            cfg.connect_timeout_ms,
            cfg.will,
            cfg.ssl_config,
            conn.created_at,
            conn.updated_at
        FROM data_connections conn
        JOIN data_mqtt_configs cfg ON cfg.connection_id = conn.id
        WHERE conn.project_id = $1
          AND conn.type = 'mqtt'
        ORDER BY conn.created_at DESC
        LIMIT $2 OFFSET $3
    `, projectID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 MQTT 连接失败", err)
	}
	defer rows.Close()

	result := make([]MqttConnectionDetailRecord, 0)
	for rows.Next() {
		record, scanErr := scanMqttConnectionDetail(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 MQTT 连接失败", err)
	}

	return result, total, nil
}

// GetConnectionSummary 按项目与主键读取 MQTT 连接主表摘要。
func (r *MqttRepository) GetConnectionSummary(ctx context.Context, projectID, connectionID string) (*MqttConnectionSummaryRecord, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, project_id, name, type, status, created_at, updated_at
        FROM data_connections
        WHERE project_id = $1
          AND id = $2
          AND type IN ('mqtt', 'builtin.message')
    `, projectID, connectionID)

	var record MqttConnectionSummaryRecord
	if err := row.Scan(&record.ID, &record.ProjectID, &record.Name, &record.Type, &record.Status, &record.CreatedAt, &record.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 连接不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 MQTT 连接失败", err)
	}
	return &record, nil
}

// GetConnectionDetail 按项目与主键读取 MQTT 连接详情。
// IF消息库没有外置 broker 配置行，这里用默认 MQTT 字段占位，真正连接地址由服务层内置 message-hub 配置覆盖。
func (r *MqttRepository) GetConnectionDetail(ctx context.Context, projectID, connectionID string) (*MqttConnectionDetailRecord, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT
            conn.id,
            conn.project_id,
            conn.name,
            conn.type,
            conn.status,
            conn.is_enabled,
            conn.retry_count,
            conn.retry_interval_ms,
            conn.health_check_interval_ms,
            COALESCE(cfg.broker_url, ''),
            COALESCE(cfg.protocol, 'mqtt'),
            COALESCE(cfg.port, 0),
            cfg.client_id,
            cfg.username,
            cfg.password,
            COALESCE(cfg.keepalive, 30),
            COALESCE(cfg.clean_session, true),
            COALESCE(cfg.qos, 0),
            COALESCE(cfg.reconnect_period_ms, 5000),
            COALESCE(cfg.connect_timeout_ms, 5000),
            COALESCE(cfg.will, '{}'::jsonb),
            COALESCE(cfg.ssl_config, '{}'::jsonb),
            conn.created_at,
            conn.updated_at
        FROM data_connections conn
        LEFT JOIN data_mqtt_configs cfg ON cfg.connection_id = conn.id
        WHERE conn.project_id = $1
          AND conn.id = $2
          AND conn.type IN ('mqtt', 'builtin.message')
    `, projectID, connectionID)

	record, err := scanMqttConnectionDetail(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateConnectionDetail 更新 MQTT 连接与配置。
func (r *MqttRepository) UpdateConnectionDetail(ctx context.Context, params UpdateMqttConnectionParams) (*MqttConnectionDetailRecord, error) {
	willBytes, err := marshalMqttJSONObject(params.Will)
	if err != nil {
		return nil, err
	}
	sslBytes, err := marshalMqttJSONObject(params.SSLConfig)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 MQTT 更新事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)

	metadataBytes, err := json.Marshal(map[string]any{
		"brokerUrl": params.BrokerURL,
		"protocol":  params.Protocol,
		"port":      params.Port,
	})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "序列化 MQTT 元数据失败", err)
	}

	row := tx.QueryRow(ctx, `
        UPDATE data_connections
        SET name = $3,
            status = $4,
            is_enabled = $5,
            retry_count = $6,
            retry_interval_ms = $7,
            health_check_interval_ms = $8,
            metadata = $9::jsonb,
            updated_by = $10,
            updated_at = now()
        WHERE project_id = $1
          AND id = $2
          AND type = 'mqtt'
        RETURNING id, project_id, name, type, status, is_enabled, retry_count, retry_interval_ms, health_check_interval_ms,
                  created_at, updated_at
    `, params.ProjectID, params.ConnectionID, params.Name, params.Status, params.IsEnabled, params.RetryCount, params.RetryIntervalMS, params.HealthCheckIntervalMS, string(metadataBytes), params.UserID)

	var current struct {
		ID                    string
		ProjectID             string
		Name                  string
		Type                  string
		Status                string
		IsEnabled             bool
		RetryCount            int
		RetryIntervalMS       int
		HealthCheckIntervalMS int
		CreatedAt             time.Time
		UpdatedAt             time.Time
	}
	if err := row.Scan(
		&current.ID,
		&current.ProjectID,
		&current.Name,
		&current.Type,
		&current.Status,
		&current.IsEnabled,
		&current.RetryCount,
		&current.RetryIntervalMS,
		&current.HealthCheckIntervalMS,
		&current.CreatedAt,
		&current.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 连接不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新 MQTT 连接失败", err)
	}

	if _, err := tx.Exec(ctx, `
        UPDATE data_mqtt_configs
        SET broker_url = $2,
            protocol = $3,
            port = $4,
            client_id = $5,
            username = $6,
            password = $7,
            keepalive = $8,
            clean_session = $9,
            qos = $10,
            reconnect_period_ms = $11,
            connect_timeout_ms = $12,
            will = $13::jsonb,
            ssl_config = $14::jsonb,
            updated_at = now()
        WHERE connection_id = $1
    `, params.ConnectionID, params.BrokerURL, params.Protocol, params.Port, params.ClientID, params.Username, params.Password, params.Keepalive, params.CleanSession, params.QOS, params.ReconnectPeriodMS, params.ConnectTimeoutMS, nullableMqttJSON(willBytes), nullableMqttJSON(sslBytes)); err != nil {
		return nil, translateMqttWriteError("更新 MQTT 配置失败", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 MQTT 更新事务失败", err)
	}
	return r.GetConnectionDetail(ctx, params.ProjectID, params.ConnectionID)
}

// StopConnection 将 MQTT 连接状态切到 disconnected。
func (r *MqttRepository) StopConnection(ctx context.Context, projectID, connectionID string) (*MqttConnectionStatusRecord, error) {
	var status string
	err := r.pool.QueryRow(ctx, `
        UPDATE data_connections
        SET status = 'disconnected',
            updated_at = now()
        WHERE project_id = $1
          AND id = $2
          AND type IN ('mqtt', 'builtin.message')
        RETURNING status
    `, projectID, connectionID).Scan(&status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 连接不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "停止 MQTT 连接失败", err)
	}
	return &MqttConnectionStatusRecord{Status: status}, nil
}

// ListSubscriptions 按项目分页查询订阅。
func (r *MqttRepository) ListSubscriptions(ctx context.Context, projectID, connectionID string, page, pageSize int) ([]MqttSubscriptionRecord, int, error) {
	page, pageSize = normalizePageAndSize(page, pageSize, 50, 200)
	where := []string{"project_id = $1"}
	args := []any{projectID}
	if strings.TrimSpace(connectionID) != "" {
		args = append(args, strings.TrimSpace(connectionID))
		where = append(where, fmt.Sprintf("connection_id = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_mqtt_subscriptions WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计 MQTT 订阅失败", err)
	}

	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, connection_id, group_id, name, topic, qos, usage_mode, description, message_retention, default_batch_parse_rule, display_order, created_at, updated_at
        FROM data_mqtt_subscriptions
        WHERE `+whereSQL+`
        ORDER BY display_order ASC, created_at ASC
        LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2), listArgs...)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 MQTT 订阅失败", err)
	}
	defer rows.Close()

	result := make([]MqttSubscriptionRecord, 0)
	for rows.Next() {
		record, scanErr := scanMqttSubscription(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 MQTT 订阅失败", err)
	}
	return result, total, nil
}

// GetSubscription 读取单个订阅。
func (r *MqttRepository) GetSubscription(ctx context.Context, projectID, subscriptionID string) (*MqttSubscriptionRecord, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, project_id, connection_id, group_id, name, topic, qos, usage_mode, description, message_retention, default_batch_parse_rule, display_order, created_at, updated_at
        FROM data_mqtt_subscriptions
        WHERE project_id = $1 AND id = $2
    `, projectID, subscriptionID)
	record, err := scanMqttSubscription(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// CreateSubscription 新建订阅。
func (r *MqttRepository) CreateSubscription(ctx context.Context, params CreateMqttSubscriptionParams) (*MqttSubscriptionRecord, error) {
	row := r.pool.QueryRow(ctx, `
        INSERT INTO data_mqtt_subscriptions (
            project_id, connection_id, group_id, name, topic, qos, usage_mode, description, message_retention, display_order, created_by, updated_by
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
        RETURNING id, project_id, connection_id, group_id, name, topic, qos, usage_mode, description, message_retention, default_batch_parse_rule, display_order, created_at, updated_at
    `, params.ProjectID, params.ConnectionID, params.GroupID, params.Name, params.Topic, params.QOS, params.UsageMode, params.Description, params.MessageRetention, params.Order, params.UserID)
	record, err := scanMqttSubscription(row)
	if err != nil {
		return nil, translateMqttWriteError("创建 MQTT 订阅失败", err)
	}
	return &record, nil
}

// UpdateSubscription 更新订阅。
func (r *MqttRepository) UpdateSubscription(ctx context.Context, params UpdateMqttSubscriptionParams) (*MqttSubscriptionRecord, error) {
	groupID := params.GroupID
	if !params.HasGroupID {
		groupID = nil
	}
	row := r.pool.QueryRow(ctx, `
        UPDATE data_mqtt_subscriptions
        SET group_id = $3,
            name = $4,
            topic = $5,
            qos = $6,
            usage_mode = $7,
            description = $8,
            message_retention = $9,
            display_order = $10,
            updated_by = $11,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, connection_id, group_id, name, topic, qos, usage_mode, description, message_retention, default_batch_parse_rule, display_order, created_at, updated_at
    `, params.ProjectID, params.SubscriptionID, groupID, params.Name, params.Topic, params.QOS, params.UsageMode, params.Description, params.MessageRetention, params.Order, params.UserID)
	record, err := scanMqttSubscription(row)
	if err != nil {
		return nil, translateMqttWriteError("更新 MQTT 订阅失败", err)
	}
	return &record, nil
}

// UpdateSubscriptionDefaultBatchRule 更新订阅默认批量解析规则。
func (r *MqttRepository) UpdateSubscriptionDefaultBatchRule(ctx context.Context, params UpdateMqttSubscriptionDefaultBatchRuleParams) (*MqttSubscriptionRecord, error) {
	ruleBytes, err := json.Marshal(params.Rule)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "默认批量解析规则不是合法 JSON", err)
	}
	row := r.pool.QueryRow(ctx, `
        UPDATE data_mqtt_subscriptions
        SET default_batch_parse_rule = $3,
            updated_by = $4,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, connection_id, group_id, name, topic, qos, usage_mode, description, message_retention, default_batch_parse_rule, display_order, created_at, updated_at
    `, params.ProjectID, params.SubscriptionID, nullableMqttJSON(ruleBytes), params.UserID)
	record, err := scanMqttSubscription(row)
	if err != nil {
		return nil, translateMqttWriteError("保存 MQTT 订阅默认批量解析规则失败", err)
	}
	return &record, nil
}

// DeleteSubscription 删除订阅。
func (r *MqttRepository) DeleteSubscription(ctx context.Context, projectID, subscriptionID string) error {
	commandTag, err := r.pool.Exec(ctx, `
        DELETE FROM data_mqtt_subscriptions
        WHERE project_id = $1 AND id = $2
    `, projectID, subscriptionID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 MQTT 订阅失败", err)
	}
	if commandTag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 订阅不存在")
	}
	return nil
}

// ListSubscriptionGroups 查询连接下的订阅分组。
func (r *MqttRepository) ListSubscriptionGroups(ctx context.Context, projectID, connectionID string) ([]MqttSubscriptionGroupRecord, error) {
	where := []string{"project_id = $1"}
	args := []any{projectID}
	if strings.TrimSpace(connectionID) != "" {
		args = append(args, strings.TrimSpace(connectionID))
		where = append(where, fmt.Sprintf("connection_id = $%d", len(args)))
	}
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, connection_id, name, parent_id, created_at, updated_at
        FROM data_mqtt_subscription_groups
        WHERE `+strings.Join(where, " AND ")+`
        ORDER BY created_at ASC
    `, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 MQTT 订阅分组失败", err)
	}
	defer rows.Close()

	result := make([]MqttSubscriptionGroupRecord, 0)
	for rows.Next() {
		record, scanErr := scanMqttSubscriptionGroup(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 MQTT 订阅分组失败", err)
	}
	return result, nil
}

// GetSubscriptionGroup 读取单个订阅分组。
func (r *MqttRepository) GetSubscriptionGroup(ctx context.Context, projectID, groupID string) (*MqttSubscriptionGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, project_id, connection_id, name, parent_id, created_at, updated_at
        FROM data_mqtt_subscription_groups
        WHERE project_id = $1 AND id = $2
    `, projectID, groupID)
	record, err := scanMqttSubscriptionGroup(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// CreateSubscriptionGroup 新建订阅分组。
func (r *MqttRepository) CreateSubscriptionGroup(ctx context.Context, params CreateMqttSubscriptionGroupParams) (*MqttSubscriptionGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
        INSERT INTO data_mqtt_subscription_groups (
            project_id, connection_id, name, parent_id, created_by, updated_by
        )
        VALUES ($1, $2, $3, $4, $5, $5)
        RETURNING id, project_id, connection_id, name, parent_id, created_at, updated_at
    `, params.ProjectID, params.ConnectionID, params.Name, params.ParentID, params.UserID)
	record, err := scanMqttSubscriptionGroup(row)
	if err != nil {
		return nil, translateMqttWriteError("创建 MQTT 订阅分组失败", err)
	}
	return &record, nil
}

// UpdateSubscriptionGroup 更新订阅分组。
func (r *MqttRepository) UpdateSubscriptionGroup(ctx context.Context, params UpdateMqttSubscriptionGroupParams) (*MqttSubscriptionGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
        UPDATE data_mqtt_subscription_groups
        SET name = $3,
            parent_id = $4,
            updated_by = $5,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, connection_id, name, parent_id, created_at, updated_at
    `, params.ProjectID, params.GroupID, params.Name, params.ParentID, params.UserID)
	record, err := scanMqttSubscriptionGroup(row)
	if err != nil {
		return nil, translateMqttWriteError("更新 MQTT 订阅分组失败", err)
	}
	return &record, nil
}

// DeleteSubscriptionGroup 删除订阅分组；组内订阅由外键自动回到根。
func (r *MqttRepository) DeleteSubscriptionGroup(ctx context.Context, projectID, groupID string) error {
	commandTag, err := r.pool.Exec(ctx, `
        DELETE FROM data_mqtt_subscription_groups
        WHERE project_id = $1 AND id = $2
    `, projectID, groupID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 MQTT 订阅分组失败", err)
	}
	if commandTag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 订阅分组不存在")
	}
	return nil
}

// ListTagsBySubscription 查询订阅下变量。
func (r *MqttRepository) ListTagsBySubscription(ctx context.Context, projectID, subscriptionID string, search string, page, pageSize int, sortBy, sortOrder string) ([]MqttTagRecord, int, error) {
	return r.listTags(ctx, projectID, subscriptionID, search, page, pageSize, sortBy, sortOrder)
}

// ListTagIDsBySubscriptionFilter 返回当前订阅和搜索条件命中的变量 ID，用于后端执行“全部筛选结果”批量操作。
func (r *MqttRepository) ListTagIDsBySubscriptionFilter(ctx context.Context, projectID, subscriptionID string, search string) ([]string, error) {
	where := []string{"project_id = $1", "subscription_id = $2"}
	args := []any{projectID, subscriptionID}
	if keyword := strings.TrimSpace(search); keyword != "" {
		args = append(args, "%"+keyword+"%")
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR code ILIKE $%d OR parse_rule ILIKE $%d)", len(args), len(args), len(args)))
	}

	rows, err := r.pool.Query(ctx, `
        SELECT id
        FROM data_mqtt_tags
        WHERE `+strings.Join(where, " AND ")+`
        ORDER BY created_at DESC, id DESC
    `, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 MQTT 变量筛选结果失败", err)
	}
	defer rows.Close()

	result := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 MQTT 变量筛选结果失败", err)
		}
		result = append(result, id)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 MQTT 变量筛选结果失败", err)
	}
	return result, nil
}

// ListTagsByProject 查询项目下变量。
func (r *MqttRepository) ListTagsByProject(ctx context.Context, projectID string, page, pageSize int, subscriptionID string) ([]MqttTagRecord, int, error) {
	return r.listTags(ctx, projectID, subscriptionID, "", page, pageSize, "createdAt", "desc")
}

func (r *MqttRepository) listTags(ctx context.Context, projectID, subscriptionID string, search string, page, pageSize int, sortBy, sortOrder string) ([]MqttTagRecord, int, error) {
	page, pageSize = normalizePageAndSize(page, pageSize, 50, 5000)
	where := []string{"project_id = $1"}
	args := []any{projectID}
	if strings.TrimSpace(subscriptionID) != "" {
		args = append(args, strings.TrimSpace(subscriptionID))
		where = append(where, fmt.Sprintf("subscription_id = $%d", len(args)))
	}
	if keyword := strings.TrimSpace(search); keyword != "" {
		args = append(args, "%"+keyword+"%")
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR code ILIKE $%d OR parse_rule ILIKE $%d)", len(args), len(args), len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	orderSQL := resolveMqttTagOrderSQL(sortBy, sortOrder)

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_mqtt_tags WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计 MQTT 变量失败", err)
	}

	queryArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, subscription_id, name, code, description, data_type, parse_type, parse_rule,
               default_value, unit, transform, validation, display_order, created_at, updated_at
        FROM data_mqtt_tags
        WHERE `+whereSQL+`
        ORDER BY `+orderSQL+`
        LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2), queryArgs...)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 MQTT 变量失败", err)
	}
	defer rows.Close()

	result := make([]MqttTagRecord, 0)
	for rows.Next() {
		record, scanErr := scanMqttTag(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 MQTT 变量失败", err)
	}

	return result, total, nil
}

func resolveMqttTagOrderSQL(sortBy, sortOrder string) string {
	direction := "DESC"
	if strings.EqualFold(strings.TrimSpace(sortOrder), "asc") {
		direction = "ASC"
	}
	switch strings.TrimSpace(sortBy) {
	case "name":
		return "regexp_replace(name, '\\d+$', '') " + direction + ", COALESCE(NULLIF(substring(name FROM '\\d+$'), '')::bigint, 0) " + direction + ", name " + direction + ", created_at DESC, id DESC"
	default:
		return "created_at " + direction + ", display_order " + direction + ", id " + direction
	}
}

// GetTag 读取单个变量。
func (r *MqttRepository) GetTag(ctx context.Context, projectID, tagID string) (*MqttTagRecord, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, project_id, subscription_id, name, code, description, data_type, parse_type, parse_rule,
               default_value, unit, transform, validation, display_order, created_at, updated_at
        FROM data_mqtt_tags
        WHERE project_id = $1 AND id = $2
    `, projectID, tagID)
	record, err := scanMqttTag(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// CreateTag 新建变量。
func (r *MqttRepository) CreateTag(ctx context.Context, params CreateMqttTagParams) (*MqttTagRecord, error) {
	validationBytes, err := marshalMqttJSONObject(params.Validation)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, `
        INSERT INTO data_mqtt_tags (
            project_id, subscription_id, name, code, description, data_type, parse_type, parse_rule,
            default_value, unit, transform, validation, display_order, created_by, updated_by
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb, $13, $14, $14)
        RETURNING id, project_id, subscription_id, name, code, description, data_type, parse_type, parse_rule,
                  default_value, unit, transform, validation, display_order, created_at, updated_at
    `, params.ProjectID, params.SubscriptionID, params.Name, params.Code, params.Description, params.DataType, params.ParseType, params.ParseRule, params.DefaultValue, params.Unit, params.Transform, nullableMqttJSON(validationBytes), params.Order, params.UserID)
	record, err := scanMqttTag(row)
	if err != nil {
		return nil, translateMqttWriteError("创建 MQTT 变量失败", err)
	}
	return &record, nil
}

// CreateTagsBatchWithDataPoints 在一个事务中批量创建 MQTT 变量并同步数据点。
// 这里使用集合 SQL，避免 1000 条变量触发数千次数据库往返；唯一约束仍作为并发冲突的最终防线。
func (r *MqttRepository) CreateTagsBatchWithDataPoints(ctx context.Context, params []BatchMqttTagDataPointParams) ([]MqttTagRecord, error) {
	if len(params) == 0 {
		return []MqttTagRecord{}, nil
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 MQTT 变量批量创建事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)

	subscriptionID := strings.TrimSpace(params[0].Tag.SubscriptionID)
	if err := lockMqttTagOrderAllocation(ctx, tx, subscriptionID); err != nil {
		return nil, err
	}
	baseOrder, err := nextMqttTagDisplayOrder(ctx, tx, params[0].Tag.ProjectID, subscriptionID)
	if err != nil {
		return nil, err
	}

	projectIDs := make([]string, 0, len(params))
	subscriptionIDs := make([]string, 0, len(params))
	names := make([]string, 0, len(params))
	codes := make([]string, 0, len(params))
	descriptions := make([]*string, 0, len(params))
	dataTypes := make([]string, 0, len(params))
	parseTypes := make([]string, 0, len(params))
	parseRules := make([]string, 0, len(params))
	defaultValues := make([]*string, 0, len(params))
	units := make([]*string, 0, len(params))
	transforms := make([]*string, 0, len(params))
	validations := make([]string, 0, len(params))
	orders := make([]int, 0, len(params))
	createdBy := make([]string, 0, len(params))

	for index, item := range params {
		validationBytes, marshalErr := marshalMqttJSONObject(item.Tag.Validation)
		if marshalErr != nil {
			return nil, marshalErr
		}
		projectIDs = append(projectIDs, item.Tag.ProjectID)
		subscriptionIDs = append(subscriptionIDs, item.Tag.SubscriptionID)
		names = append(names, item.Tag.Name)
		codes = append(codes, item.Tag.Code)
		descriptions = append(descriptions, item.Tag.Description)
		dataTypes = append(dataTypes, item.Tag.DataType)
		parseTypes = append(parseTypes, item.Tag.ParseType)
		parseRules = append(parseRules, item.Tag.ParseRule)
		defaultValues = append(defaultValues, item.Tag.DefaultValue)
		units = append(units, item.Tag.Unit)
		transforms = append(transforms, item.Tag.Transform)
		validations = append(validations, string(validationBytes))
		orders = append(orders, baseOrder+index)
		createdBy = append(createdBy, item.Tag.UserID)
	}

	rows, err := tx.Query(ctx, `
        INSERT INTO data_mqtt_tags (
            project_id, subscription_id, name, code, description, data_type, parse_type, parse_rule,
            default_value, unit, transform, validation, display_order, created_by, updated_by
        )
        SELECT project_id, subscription_id, name, code, description, data_type, parse_type, parse_rule,
               default_value, unit, transform,
               NULLIF(validation, '')::jsonb,
               display_order, created_by, created_by
        FROM unnest(
            $1::uuid[], $2::uuid[], $3::text[], $4::text[], $5::text[], $6::text[], $7::text[], $8::text[],
            $9::text[], $10::text[], $11::text[], $12::text[], $13::int[], $14::uuid[]
        ) AS input(project_id, subscription_id, name, code, description, data_type, parse_type, parse_rule,
                   default_value, unit, transform, validation, display_order, created_by)
        RETURNING id, project_id, subscription_id, name, code, description, data_type, parse_type, parse_rule,
                  default_value, unit, transform, validation, display_order, created_at, updated_at
    `, projectIDs, subscriptionIDs, names, codes, descriptions, dataTypes, parseTypes, parseRules, defaultValues, units, transforms, validations, orders, createdBy)
	if err != nil {
		return nil, translateMqttWriteError("批量创建 MQTT 变量失败", err)
	}
	defer rows.Close()

	records := make([]MqttTagRecord, 0, len(params))
	for rows.Next() {
		record, scanErr := scanMqttTag(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 MQTT 变量批量创建结果失败", err)
	}
	rows.Close()

	if len(records) != len(params) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "MQTT 变量批量创建结果数量不一致")
	}

	if err := upsertMqttTagDataPointsInTx(ctx, tx, records, params); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 MQTT 变量批量创建事务失败", err)
	}
	return records, nil
}

func lockMqttTagOrderAllocation(ctx context.Context, tx pgx.Tx, subscriptionID string) error {
	if subscriptionID == "" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "订阅 ID 不能为空")
	}
	// 同一订阅的批量建点需要串行分配 display_order，避免多用户同时保存时出现重复顺序。
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`, "mqtt-tags-order", subscriptionID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "锁定 MQTT 变量排序失败", err)
	}
	return nil
}

func nextMqttTagDisplayOrder(ctx context.Context, tx pgx.Tx, projectID, subscriptionID string) (int, error) {
	var next int
	if err := tx.QueryRow(ctx, `
        SELECT COALESCE(MAX(display_order) + 1, 0)
        FROM data_mqtt_tags
        WHERE project_id = $1 AND subscription_id = $2
    `, projectID, subscriptionID).Scan(&next); err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "分配 MQTT 变量排序失败", err)
	}
	return next, nil
}

func upsertMqttTagDataPointsInTx(ctx context.Context, tx pgx.Tx, tags []MqttTagRecord, params []BatchMqttTagDataPointParams) error {
	paramsByCode := make(map[string]BatchMqttTagDataPointParams, len(params))
	for _, item := range params {
		paramsByCode[item.Tag.Code] = item
	}

	projectIDs := make([]string, 0, len(tags))
	paths := make([]string, 0, len(tags))
	names := make([]string, 0, len(tags))
	sourceTypes := make([]string, 0, len(tags))
	sourceIDs := make([]string, 0, len(tags))
	sourceConfigs := make([]string, 0, len(tags))
	dataTypes := make([]string, 0, len(tags))
	units := make([]*string, 0, len(tags))
	defaultValues := make([]*string, 0, len(tags))
	tagsPayload := make([]string, 0, len(tags))
	refreshModes := make([]string, 0, len(tags))
	statuses := make([]string, 0, len(tags))
	displayOrders := make([]int, 0, len(tags))
	userIDs := make([]string, 0, len(tags))

	for _, tag := range tags {
		item, ok := paramsByCode[tag.Code]
		if !ok {
			return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "MQTT 变量批量创建结果无法匹配输入")
		}
		projectIDs = append(projectIDs, tag.ProjectID)
		paths = append(paths, item.DataPath)
		names = append(names, item.DataName)
		sourceTypes = append(sourceTypes, item.SourceType)
		sourceIDs = append(sourceIDs, tag.ID)
		sourceConfigBytes, marshalErr := marshalMqttJSONObject(item.SourceConfig)
		if marshalErr != nil {
			return marshalErr
		}
		sourceConfigs = append(sourceConfigs, string(sourceConfigBytes))
		dataTypes = append(dataTypes, tag.DataType)
		units = append(units, tag.Unit)
		defaultValues = append(defaultValues, tag.DefaultValue)
		tagsPayload = append(tagsPayload, "[]")
		refreshModes = append(refreshModes, item.RefreshMode)
		statuses = append(statuses, item.Status)
		displayOrders = append(displayOrders, tag.Order)
		userIDs = append(userIDs, item.Tag.UserID)
	}

	rows, err := tx.Query(ctx, `
        INSERT INTO data_points (
            project_id, path, name, source_type, source_id, source_config, data_type,
            unit, default_value, tags, refresh_mode, status, display_order, created_by, updated_by
        )
        SELECT project_id, path, name, source_type, source_id, source_config::jsonb, data_type,
               unit, default_value, tags::jsonb, refresh_mode, status, display_order, user_id, user_id
        FROM unnest(
            $1::uuid[], $2::text[], $3::text[], $4::text[], $5::uuid[], $6::text[], $7::text[],
            $8::text[], $9::text[], $10::text[], $11::text[], $12::text[], $13::int[], $14::uuid[]
        ) AS input(project_id, path, name, source_type, source_id, source_config, data_type,
                   unit, default_value, tags, refresh_mode, status, display_order, user_id)
        ON CONFLICT (project_id, path) DO UPDATE
        SET name = EXCLUDED.name,
            source_type = EXCLUDED.source_type,
            source_id = EXCLUDED.source_id,
            source_config = EXCLUDED.source_config,
            data_type = EXCLUDED.data_type,
            unit = EXCLUDED.unit,
            default_value = EXCLUDED.default_value,
            tags = EXCLUDED.tags,
            refresh_mode = EXCLUDED.refresh_mode,
            status = EXCLUDED.status,
            display_order = EXCLUDED.display_order,
            updated_by = EXCLUDED.updated_by,
            updated_at = now()
        WHERE data_points.status = 'invalid'
          AND data_points.source_type = EXCLUDED.source_type
        RETURNING id
    `, projectIDs, paths, names, sourceTypes, sourceIDs, sourceConfigs, dataTypes, units, defaultValues, tagsPayload, refreshModes, statuses, displayOrders, userIDs)
	if err != nil {
		return translateDataPointWriteError(err)
	}
	defer rows.Close()

	affected := 0
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 MQTT 变量数据点批量同步结果失败", err)
		}
		affected += 1
	}
	if err := rows.Err(); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 MQTT 变量数据点批量同步结果失败", err)
	}
	if affected != len(tags) {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "部分 MQTT 变量数据点路径已被占用")
	}
	return nil
}

// UpdateTag 更新变量。
func (r *MqttRepository) UpdateTag(ctx context.Context, params UpdateMqttTagParams) (*MqttTagRecord, error) {
	validationBytes, err := marshalMqttJSONObject(params.Validation)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, `
        UPDATE data_mqtt_tags
        SET name = $3,
            code = $4,
            description = $5,
            data_type = $6,
            parse_type = $7,
            parse_rule = $8,
            default_value = $9,
            unit = $10,
            transform = $11,
            validation = $12::jsonb,
            display_order = $13,
            updated_by = $14,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, subscription_id, name, code, description, data_type, parse_type, parse_rule,
                  default_value, unit, transform, validation, display_order, created_at, updated_at
    `, params.ProjectID, params.TagID, params.Name, params.Code, params.Description, params.DataType, params.ParseType, params.ParseRule, params.DefaultValue, params.Unit, params.Transform, nullableMqttJSON(validationBytes), params.Order, params.UserID)
	record, err := scanMqttTag(row)
	if err != nil {
		return nil, translateMqttWriteError("更新 MQTT 变量失败", err)
	}
	return &record, nil
}

// DeleteTag 删除变量。
func (r *MqttRepository) DeleteTag(ctx context.Context, projectID, tagID string) error {
	commandTag, err := r.pool.Exec(ctx, `
        DELETE FROM data_mqtt_tags
        WHERE project_id = $1 AND id = $2
    `, projectID, tagID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 MQTT 变量失败", err)
	}
	if commandTag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 变量不存在")
	}
	return nil
}

// DeleteTagsBatchWithDataPoints 在一个事务中批量删除变量并标记对应数据点失效。
func (r *MqttRepository) DeleteTagsBatchWithDataPoints(ctx context.Context, projectID string, tagIDs []string, userID string) (int, error) {
	if len(tagIDs) == 0 {
		return 0, nil
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 MQTT 变量批量删除事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)

	commandTag, err := tx.Exec(ctx, `
        UPDATE data_points
        SET status = 'invalid',
            updated_by = COALESCE($4, updated_by),
            updated_at = now()
        WHERE project_id = $1
          AND source_type = $2
          AND source_id = ANY($3::uuid[])
          AND status <> 'invalid'
    `, projectID, "mqtt.tag", tagIDs, userID)
	if err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "批量标记 MQTT 变量数据点失效失败", err)
	}
	_ = commandTag

	deleted, err := tx.Exec(ctx, `
        DELETE FROM data_mqtt_tags
        WHERE project_id = $1 AND id = ANY($2::uuid[])
    `, projectID, tagIDs)
	if err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "批量删除 MQTT 变量失败", err)
	}
	deletedCount := int(deleted.RowsAffected())
	if deletedCount != len(tagIDs) {
		return 0, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "部分 MQTT 变量不存在或不属于当前项目")
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 MQTT 变量批量删除事务失败", err)
	}
	return deletedCount, nil
}

// UpdateTagsOrder 批量更新变量顺序。
func (r *MqttRepository) UpdateTagsOrder(ctx context.Context, projectID string, tagIDs []string, userID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启变量排序事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)

	for index, tagID := range tagIDs {
		if _, err := tx.Exec(ctx, `
            UPDATE data_mqtt_tags
            SET display_order = $3,
                updated_by = $4,
                updated_at = now()
            WHERE project_id = $1 AND id = $2
        `, projectID, tagID, index, userID); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新变量顺序失败", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交变量排序事务失败", err)
	}
	return nil
}

func scanMqttConnectionDetail(row scannable) (MqttConnectionDetailRecord, error) {
	var (
		record              MqttConnectionDetailRecord
		clientID            sql.NullString
		username            sql.NullString
		password            sql.NullString
		willBytes, sslBytes []byte
	)
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.Name,
		&record.Type,
		&record.Status,
		&record.IsEnabled,
		&record.RetryCount,
		&record.RetryIntervalMS,
		&record.HealthCheckIntervalMS,
		&record.BrokerURL,
		&record.Protocol,
		&record.Port,
		&clientID,
		&username,
		&password,
		&record.Keepalive,
		&record.CleanSession,
		&record.QOS,
		&record.ReconnectPeriodMS,
		&record.ConnectTimeoutMS,
		&willBytes,
		&sslBytes,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MqttConnectionDetailRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 连接不存在")
		}
		return MqttConnectionDetailRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 MQTT 连接失败", err)
	}
	record.ClientID = nullStringToPtr(clientID)
	record.Username = nullStringToPtr(username)
	record.Password = nullStringToPtr(password)
	record.Will = mustJSONObject(willBytes)
	record.SSLConfig = mustJSONObject(sslBytes)
	return record, nil
}

func scanMqttSubscription(row scannable) (MqttSubscriptionRecord, error) {
	var (
		record                MqttSubscriptionRecord
		groupID               sql.NullString
		description           sql.NullString
		defaultBatchRuleBytes []byte
	)
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&groupID,
		&record.Name,
		&record.Topic,
		&record.QOS,
		&record.UsageMode,
		&description,
		&record.MessageRetention,
		&defaultBatchRuleBytes,
		&record.Order,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MqttSubscriptionRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 订阅不存在")
		}
		return MqttSubscriptionRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 MQTT 订阅失败", err)
	}
	if strings.TrimSpace(record.UsageMode) == "" {
		record.UsageMode = "single_variable"
	}
	record.GroupID = nullStringToPtr(groupID)
	record.Description = nullStringToPtr(description)
	record.DefaultBatchRule = mustJSONObject(defaultBatchRuleBytes)
	return record, nil
}

func scanMqttSubscriptionGroup(row scannable) (MqttSubscriptionGroupRecord, error) {
	var (
		record   MqttSubscriptionGroupRecord
		parentID sql.NullString
	)
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.Name,
		&parentID,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MqttSubscriptionGroupRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 订阅分组不存在")
		}
		return MqttSubscriptionGroupRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 MQTT 订阅分组失败", err)
	}
	record.ParentID = nullStringToPtr(parentID)
	return record, nil
}

func scanMqttTag(row scannable) (MqttTagRecord, error) {
	var (
		record                        MqttTagRecord
		description                   sql.NullString
		defaultValue, unit, transform sql.NullString
		validationBytes               []byte
	)
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.SubscriptionID,
		&record.Name,
		&record.Code,
		&description,
		&record.DataType,
		&record.ParseType,
		&record.ParseRule,
		&defaultValue,
		&unit,
		&transform,
		&validationBytes,
		&record.Order,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MqttTagRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 变量不存在")
		}
		return MqttTagRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 MQTT 变量失败", err)
	}
	record.Description = nullStringToPtr(description)
	record.DefaultValue = nullStringToPtr(defaultValue)
	record.Unit = nullStringToPtr(unit)
	record.Transform = nullStringToPtr(transform)
	record.Validation = mustJSONObject(validationBytes)
	return record, nil
}

func nullableMqttJSON(payload []byte) any {
	if len(payload) == 0 || string(payload) == "{}" {
		return nil
	}
	return string(payload)
}

func translateMqttWriteError(message string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			switch pgErr.ConstraintName {
			case "data_mqtt_subscriptions_project_connection_name_key":
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前接入源内已存在同名订阅")
			case "data_mqtt_tags_project_subscription_code_key":
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前订阅内已存在同标识变量")
			case "data_mqtt_tags_project_code_key":
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前工程内已存在同标识变量")
			default:
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT 配置存在重复名称或标识")
			}
		case "23503":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "关联 MQTT 资源不存在")
		}
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}
