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
	Description      *string
	MessageRetention int
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

// MqttTagGroupRecord 表示 MQTT 变量组投影。
type MqttTagGroupRecord struct {
	ID             string
	ProjectID      string
	SubscriptionID string
	Name           string
	Code           string
	Description    *string
	Color          *string
	Icon           *string
	Order          int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// MqttTagRecord 表示 MQTT 变量投影。
type MqttTagRecord struct {
	ID             string
	ProjectID      string
	SubscriptionID string
	GroupID        *string
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
	Description      *string
	MessageRetention int
	Order            int
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

// CreateMqttTagGroupParams 表示新建变量组的仓储参数。
type CreateMqttTagGroupParams struct {
	ProjectID      string
	SubscriptionID string
	UserID         string
	Name           string
	Code           string
	Description    *string
	Color          *string
	Icon           *string
	Order          int
}

// UpdateMqttTagGroupParams 表示更新变量组的仓储参数。
type UpdateMqttTagGroupParams struct {
	ProjectID   string
	GroupID     string
	UserID      string
	Name        string
	Code        string
	Description *string
	Color       *string
	Icon        *string
	Order       int
}

// CreateMqttTagParams 表示新建变量的仓储参数。
type CreateMqttTagParams struct {
	ProjectID      string
	SubscriptionID string
	UserID         string
	GroupID        *string
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

// UpdateMqttTagParams 表示更新变量的仓储参数。
type UpdateMqttTagParams struct {
	ProjectID    string
	TagID        string
	UserID       string
	GroupID      *string
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
          AND type = 'mqtt'
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
          AND conn.id = $2
          AND conn.type = 'mqtt'
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
          AND type = 'mqtt'
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
        SELECT id, project_id, connection_id, group_id, name, topic, qos, description, message_retention, display_order, created_at, updated_at
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
        SELECT id, project_id, connection_id, group_id, name, topic, qos, description, message_retention, display_order, created_at, updated_at
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
            project_id, connection_id, group_id, name, topic, qos, description, message_retention, display_order, created_by, updated_by
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
        RETURNING id, project_id, connection_id, group_id, name, topic, qos, description, message_retention, display_order, created_at, updated_at
    `, params.ProjectID, params.ConnectionID, params.GroupID, params.Name, params.Topic, params.QOS, params.Description, params.MessageRetention, params.Order, params.UserID)
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
            description = $7,
            message_retention = $8,
            display_order = $9,
            updated_by = $10,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, connection_id, group_id, name, topic, qos, description, message_retention, display_order, created_at, updated_at
    `, params.ProjectID, params.SubscriptionID, groupID, params.Name, params.Topic, params.QOS, params.Description, params.MessageRetention, params.Order, params.UserID)
	record, err := scanMqttSubscription(row)
	if err != nil {
		return nil, translateMqttWriteError("更新 MQTT 订阅失败", err)
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

// ListTagGroups 查询订阅下的变量组。
func (r *MqttRepository) ListTagGroups(ctx context.Context, projectID, subscriptionID string) ([]MqttTagGroupRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, subscription_id, name, code, description, color, icon, display_order, created_at, updated_at
        FROM data_mqtt_tag_groups
        WHERE project_id = $1 AND subscription_id = $2
        ORDER BY display_order ASC, created_at ASC
    `, projectID, subscriptionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 MQTT 变量组失败", err)
	}
	defer rows.Close()

	result := make([]MqttTagGroupRecord, 0)
	for rows.Next() {
		record, scanErr := scanMqttTagGroup(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 MQTT 变量组失败", err)
	}
	return result, nil
}

// GetTagGroup 读取单个变量组。
func (r *MqttRepository) GetTagGroup(ctx context.Context, projectID, groupID string) (*MqttTagGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, project_id, subscription_id, name, code, description, color, icon, display_order, created_at, updated_at
        FROM data_mqtt_tag_groups
        WHERE project_id = $1 AND id = $2
    `, projectID, groupID)
	record, err := scanMqttTagGroup(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// CreateTagGroup 新建变量组。
func (r *MqttRepository) CreateTagGroup(ctx context.Context, params CreateMqttTagGroupParams) (*MqttTagGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
        INSERT INTO data_mqtt_tag_groups (
            project_id, subscription_id, name, code, description, color, icon, display_order, created_by, updated_by
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
        RETURNING id, project_id, subscription_id, name, code, description, color, icon, display_order, created_at, updated_at
    `, params.ProjectID, params.SubscriptionID, params.Name, params.Code, params.Description, params.Color, params.Icon, params.Order, params.UserID)
	record, err := scanMqttTagGroup(row)
	if err != nil {
		return nil, translateMqttWriteError("创建 MQTT 变量组失败", err)
	}
	return &record, nil
}

// UpdateTagGroup 更新变量组。
func (r *MqttRepository) UpdateTagGroup(ctx context.Context, params UpdateMqttTagGroupParams) (*MqttTagGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
        UPDATE data_mqtt_tag_groups
        SET name = $3,
            code = $4,
            description = $5,
            color = $6,
            icon = $7,
            display_order = $8,
            updated_by = $9,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, subscription_id, name, code, description, color, icon, display_order, created_at, updated_at
    `, params.ProjectID, params.GroupID, params.Name, params.Code, params.Description, params.Color, params.Icon, params.Order, params.UserID)
	record, err := scanMqttTagGroup(row)
	if err != nil {
		return nil, translateMqttWriteError("更新 MQTT 变量组失败", err)
	}
	return &record, nil
}

// DeleteTagGroup 删除变量组，同时把组内变量解绑。
func (r *MqttRepository) DeleteTagGroup(ctx context.Context, projectID, groupID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启变量组删除事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)

	if _, err := tx.Exec(ctx, `
        UPDATE data_mqtt_tags
        SET group_id = NULL,
            updated_at = now()
        WHERE project_id = $1 AND group_id = $2
    `, projectID, groupID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解绑变量组失败", err)
	}

	commandTag, err := tx.Exec(ctx, `
        DELETE FROM data_mqtt_tag_groups
        WHERE project_id = $1 AND id = $2
    `, projectID, groupID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 MQTT 变量组失败", err)
	}
	if commandTag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 变量组不存在")
	}

	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交变量组删除事务失败", err)
	}
	return nil
}

// UpdateTagGroupsOrder 批量更新变量组顺序。
func (r *MqttRepository) UpdateTagGroupsOrder(ctx context.Context, projectID string, groups []MqttTagGroupRecord, userID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启变量组排序事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)

	for _, group := range groups {
		if _, err := tx.Exec(ctx, `
            UPDATE data_mqtt_tag_groups
            SET display_order = $3,
                updated_by = $4,
                updated_at = now()
            WHERE project_id = $1 AND id = $2
        `, projectID, group.ID, group.Order, userID); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新变量组顺序失败", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交变量组排序事务失败", err)
	}
	return nil
}

// ListTagsBySubscription 查询订阅下变量。
func (r *MqttRepository) ListTagsBySubscription(ctx context.Context, projectID, subscriptionID string, page, pageSize int) ([]MqttTagRecord, int, error) {
	return r.listTags(ctx, projectID, subscriptionID, page, pageSize)
}

// ListTagsByProject 查询项目下变量。
func (r *MqttRepository) ListTagsByProject(ctx context.Context, projectID string, page, pageSize int, subscriptionID string) ([]MqttTagRecord, int, error) {
	return r.listTags(ctx, projectID, subscriptionID, page, pageSize)
}

func (r *MqttRepository) listTags(ctx context.Context, projectID, subscriptionID string, page, pageSize int) ([]MqttTagRecord, int, error) {
	page, pageSize = normalizePageAndSize(page, pageSize, 50, 200)
	where := []string{"project_id = $1"}
	args := []any{projectID}
	if strings.TrimSpace(subscriptionID) != "" {
		args = append(args, strings.TrimSpace(subscriptionID))
		where = append(where, fmt.Sprintf("subscription_id = $%d", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM data_mqtt_tags WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计 MQTT 变量失败", err)
	}

	queryArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, subscription_id, group_id, name, code, description, data_type, parse_type, parse_rule,
               default_value, unit, transform, validation, display_order, created_at, updated_at
        FROM data_mqtt_tags
        WHERE `+whereSQL+`
        ORDER BY display_order ASC, created_at ASC
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

// GetTag 读取单个变量。
func (r *MqttRepository) GetTag(ctx context.Context, projectID, tagID string) (*MqttTagRecord, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, project_id, subscription_id, group_id, name, code, description, data_type, parse_type, parse_rule,
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
            project_id, subscription_id, group_id, name, code, description, data_type, parse_type, parse_rule,
            default_value, unit, transform, validation, display_order, created_by, updated_by
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::jsonb, $14, $15, $15)
        RETURNING id, project_id, subscription_id, group_id, name, code, description, data_type, parse_type, parse_rule,
                  default_value, unit, transform, validation, display_order, created_at, updated_at
    `, params.ProjectID, params.SubscriptionID, params.GroupID, params.Name, params.Code, params.Description, params.DataType, params.ParseType, params.ParseRule, params.DefaultValue, params.Unit, params.Transform, nullableMqttJSON(validationBytes), params.Order, params.UserID)
	record, err := scanMqttTag(row)
	if err != nil {
		return nil, translateMqttWriteError("创建 MQTT 变量失败", err)
	}
	return &record, nil
}

// UpdateTag 更新变量。
func (r *MqttRepository) UpdateTag(ctx context.Context, params UpdateMqttTagParams) (*MqttTagRecord, error) {
	validationBytes, err := marshalMqttJSONObject(params.Validation)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, `
        UPDATE data_mqtt_tags
        SET group_id = $3,
            name = $4,
            code = $5,
            description = $6,
            data_type = $7,
            parse_type = $8,
            parse_rule = $9,
            default_value = $10,
            unit = $11,
            transform = $12,
            validation = $13::jsonb,
            display_order = $14,
            updated_by = $15,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, subscription_id, group_id, name, code, description, data_type, parse_type, parse_rule,
                  default_value, unit, transform, validation, display_order, created_at, updated_at
    `, params.ProjectID, params.TagID, params.GroupID, params.Name, params.Code, params.Description, params.DataType, params.ParseType, params.ParseRule, params.DefaultValue, params.Unit, params.Transform, nullableMqttJSON(validationBytes), params.Order, params.UserID)
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
		record      MqttSubscriptionRecord
		groupID     sql.NullString
		description sql.NullString
	)
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&groupID,
		&record.Name,
		&record.Topic,
		&record.QOS,
		&description,
		&record.MessageRetention,
		&record.Order,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MqttSubscriptionRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 订阅不存在")
		}
		return MqttSubscriptionRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 MQTT 订阅失败", err)
	}
	record.GroupID = nullStringToPtr(groupID)
	record.Description = nullStringToPtr(description)
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

func scanMqttTagGroup(row scannable) (MqttTagGroupRecord, error) {
	var (
		record      MqttTagGroupRecord
		description sql.NullString
		color       sql.NullString
		icon        sql.NullString
	)
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.SubscriptionID,
		&record.Name,
		&record.Code,
		&description,
		&color,
		&icon,
		&record.Order,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MqttTagGroupRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "MQTT 变量组不存在")
		}
		return MqttTagGroupRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 MQTT 变量组失败", err)
	}
	record.Description = nullStringToPtr(description)
	record.Color = nullStringToPtr(color)
	record.Icon = nullStringToPtr(icon)
	return record, nil
}

func scanMqttTag(row scannable) (MqttTagRecord, error) {
	var (
		record                        MqttTagRecord
		groupID, description          sql.NullString
		defaultValue, unit, transform sql.NullString
		validationBytes               []byte
	)
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.SubscriptionID,
		&groupID,
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
	record.GroupID = nullStringToPtr(groupID)
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
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "MQTT 配置存在重复名称或标识")
		case "23503":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "关联 MQTT 资源不存在")
		}
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}
