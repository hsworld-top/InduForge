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

// SnapshotRelationalConfigRecord 表示导入导出使用的关系库配置投影。
type SnapshotRelationalConfigRecord struct {
	ConnectionID string         `json:"connectionId"`
	DBType       string         `json:"dbType"`
	Host         string         `json:"host"`
	Port         int            `json:"port"`
	Database     string         `json:"database"`
	Username     string         `json:"username"`
	Password     string         `json:"password"`
	Schema       *string        `json:"schema"`
	Charset      *string        `json:"charset"`
	Timezone     *string        `json:"timezone"`
	SSL          bool           `json:"ssl"`
	SSLConfig    map[string]any `json:"sslConfig"`
}

// SnapshotMqttConfigRecord 表示导入导出使用的 MQTT 配置投影。
type SnapshotMqttConfigRecord struct {
	ConnectionID      string         `json:"connectionId"`
	BrokerURL         string         `json:"brokerUrl"`
	Protocol          string         `json:"protocol"`
	Port              int            `json:"port"`
	ClientID          *string        `json:"clientId"`
	Username          *string        `json:"username"`
	Password          *string        `json:"password"`
	Keepalive         int            `json:"keepalive"`
	CleanSession      bool           `json:"cleanSession"`
	QOS               int            `json:"qos"`
	ReconnectPeriodMS int            `json:"reconnectPeriod"`
	ConnectTimeoutMS  int            `json:"connectTimeout"`
	Will              map[string]any `json:"will"`
	SSLConfig         map[string]any `json:"sslConfig"`
}

// SnapshotMqttSubscriptionRecord 表示导入导出使用的 MQTT 订阅投影。
type SnapshotMqttSubscriptionRecord struct {
	ID               string    `json:"id"`
	ProjectID        string    `json:"projectId"`
	ConnectionID     string    `json:"connectionId"`
	Name             string    `json:"name"`
	Topic            string    `json:"topic"`
	QOS              int       `json:"qos"`
	Description      *string   `json:"description"`
	IsEnabled        bool      `json:"isEnabled"`
	MessageRetention int       `json:"messageRetention"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// SnapshotMqttTagGroupRecord 表示导入导出使用的 MQTT 变量组投影。
type SnapshotMqttTagGroupRecord struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"projectId"`
	SubscriptionID string    `json:"subscriptionId"`
	Name           string    `json:"name"`
	Code           string    `json:"code"`
	Description    *string   `json:"description"`
	Color          *string   `json:"color"`
	Icon           *string   `json:"icon"`
	Order          int       `json:"order"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// SnapshotMqttTagRecord 表示导入导出使用的 MQTT 变量投影。
type SnapshotMqttTagRecord struct {
	ID             string         `json:"id"`
	ProjectID      string         `json:"projectId"`
	SubscriptionID string         `json:"subscriptionId"`
	GroupID        *string        `json:"groupId"`
	Name           string         `json:"name"`
	Code           string         `json:"code"`
	Description    *string        `json:"description"`
	DataType       string         `json:"dataType"`
	ParseType      string         `json:"parseType"`
	ParseRule      string         `json:"parseRule"`
	DefaultValue   *string        `json:"defaultValue"`
	Unit           *string        `json:"unit"`
	Transform      *string        `json:"transform"`
	Validation     map[string]any `json:"validation"`
	IsEnabled      bool           `json:"isEnabled"`
	Order          int            `json:"order"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// ProjectSnapshot 表示工程级数据域快照。
type ProjectSnapshot struct {
	Connections       []ConnectionRecord               `json:"connections"`
	RelationalConfigs []SnapshotRelationalConfigRecord `json:"relationalConfigs"`
	Queries           []QueryRecord                    `json:"queries"`
	MqttConfigs       []SnapshotMqttConfigRecord       `json:"mqttConfigs"`
	MqttSubscriptions []SnapshotMqttSubscriptionRecord `json:"mqttSubscriptions"`
	MqttTagGroups     []SnapshotMqttTagGroupRecord     `json:"mqttTagGroups"`
	MqttTags          []SnapshotMqttTagRecord          `json:"mqttTags"`
	DataPoints        []DataPointRecord                `json:"datapoints"`
}

// ProjectArtifactVersionV1 表示 Phase 1 产物契约版本号。
const ProjectArtifactVersionV1 = "1.0"

// ArtifactConnectionRecord 表示产物层通用连接对象。
type ArtifactConnectionRecord struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Status string         `json:"status"`
	Config map[string]any `json:"config"`
}

// ArtifactQueryRecord 表示产物层查询对象。
type ArtifactQueryRecord struct {
	ID              string         `json:"id"`
	ConnectionID    string         `json:"connectionId"`
	Name            string         `json:"name"`
	QueryType       string         `json:"queryType"`
	Config          map[string]any `json:"config"`
	TimeoutMS       int            `json:"timeoutMs"`
	IsEnabled       bool           `json:"isEnabled"`
	Transformer     *string        `json:"transformer,omitempty"`
	CacheEnabled    bool           `json:"cacheEnabled"`
	CacheTtlSeconds int            `json:"cacheTtlSeconds"`
}

// ArtifactDataPointRecord 表示产物层数据点对象。
type ArtifactDataPointRecord struct {
	ID                 string                      `json:"id"`
	Path               string                      `json:"path"`
	Name               string                      `json:"name"`
	SourceType         string                      `json:"sourceType"`
	SourceID           *string                     `json:"sourceId,omitempty"`
	SourceConfig       map[string]any              `json:"sourceConfig"`
	DataType           string                      `json:"dataType"`
	RuntimePermissions DataPointRuntimePermissions `json:"runtimePermissions"`
	RefreshMode        string                      `json:"refreshMode"`
	RefreshIntervalMS  *int                        `json:"refreshIntervalMs,omitempty"`
	Status             string                      `json:"status"`
	Unit               *string                     `json:"unit,omitempty"`
	DefaultValue       *string                     `json:"defaultValue,omitempty"`
	Tags               []any                       `json:"tags"`
}

// ArtifactMqttConnectionRecord 表示产物层 MQTT 连接对象。
type ArtifactMqttConnectionRecord struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Type              string         `json:"type"`
	Status            string         `json:"status"`
	BrokerURL         string         `json:"brokerUrl"`
	Protocol          string         `json:"protocol"`
	Port              int            `json:"port"`
	ClientID          *string        `json:"clientId,omitempty"`
	Username          *string        `json:"username,omitempty"`
	Password          *string        `json:"password,omitempty"`
	Keepalive         int            `json:"keepalive"`
	CleanSession      bool           `json:"cleanSession"`
	QOS               int            `json:"qos"`
	ReconnectPeriodMS int            `json:"reconnectPeriod"`
	ConnectTimeoutMS  int            `json:"connectTimeout"`
	Will              map[string]any `json:"will"`
	SSLConfig         map[string]any `json:"sslConfig"`
}

// ArtifactProtocolRecord 表示产物层协议对象（kafka/http/websocket/redis）。
type ArtifactProtocolRecord struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Status string         `json:"status"`
	Config map[string]any `json:"config"`
}

// ArtifactMqttPayload 表示产物层 MQTT 区块。
type ArtifactMqttPayload struct {
	Connections   []ArtifactMqttConnectionRecord   `json:"connections"`
	Subscriptions []SnapshotMqttSubscriptionRecord `json:"subscriptions"`
	TagGroups     []SnapshotMqttTagGroupRecord     `json:"tagGroups"`
	Tags          []SnapshotMqttTagRecord          `json:"tags"`
}

// ArtifactProtocolsPayload 表示产物层其他协议区块。
type ArtifactProtocolsPayload struct {
	Kafka     []ArtifactProtocolRecord `json:"kafka"`
	HTTP      []ArtifactProtocolRecord `json:"http"`
	Websocket []ArtifactProtocolRecord `json:"websocket"`
	Redis     []ArtifactProtocolRecord `json:"redis"`
}

// ProjectArtifactV1 表示 Phase 1 项目级数据产物。
type ProjectArtifactV1 struct {
	Version     string                     `json:"version"`
	ProjectID   string                     `json:"projectId"`
	GeneratedAt time.Time                  `json:"generatedAt"`
	Connections []ArtifactConnectionRecord `json:"connections"`
	Queries     []ArtifactQueryRecord      `json:"queries"`
	DataPoints  []ArtifactDataPointRecord  `json:"datapoints"`
	Mqtt        ArtifactMqttPayload        `json:"mqtt"`
	Protocols   ArtifactProtocolsPayload   `json:"protocols"`
}

// ProjectSnapshotRepository 负责项目级数据域快照读写。
type ProjectSnapshotRepository struct {
	pool *pgxpool.Pool
}

// NewProjectSnapshotRepository 创建快照仓储。
func NewProjectSnapshotRepository(pool *pgxpool.Pool) *ProjectSnapshotRepository {
	return &ProjectSnapshotRepository{pool: pool}
}

// GetByProject 读取项目级完整快照。
func (r *ProjectSnapshotRepository) GetByProject(ctx context.Context, projectID string) (*ProjectSnapshot, error) {
	connections, err := r.listConnections(ctx, projectID)
	if err != nil {
		return nil, err
	}
	queries, err := r.listQueries(ctx, projectID)
	if err != nil {
		return nil, err
	}
	datapoints, err := r.listDataPoints(ctx, projectID)
	if err != nil {
		return nil, err
	}
	mqttConfigs, err := r.listMqttConfigs(ctx, projectID)
	if err != nil {
		return nil, err
	}
	mqttSubscriptions, err := r.listMqttSubscriptions(ctx, projectID)
	if err != nil {
		return nil, err
	}
	mqttTagGroups, err := r.listMqttTagGroups(ctx, projectID)
	if err != nil {
		return nil, err
	}
	mqttTags, err := r.listMqttTags(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &ProjectSnapshot{
		Connections:       connections,
		RelationalConfigs: deriveRelationalConfigs(connections),
		Queries:           queries,
		MqttConfigs:       mqttConfigs,
		MqttSubscriptions: mqttSubscriptions,
		MqttTagGroups:     mqttTagGroups,
		MqttTags:          mqttTags,
		DataPoints:        datapoints,
	}, nil
}

// ReplaceProjectData 用快照内容覆盖项目下的数据域数据。
func (r *ProjectSnapshotRepository) ReplaceProjectData(ctx context.Context, projectID, actorID string, snapshot ProjectSnapshot) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启快照写入事务失败", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := r.deleteProjectSnapshot(ctx, tx, projectID); err != nil {
		return err
	}
	if err := r.insertConnections(ctx, tx, projectID, actorID, snapshot.Connections); err != nil {
		return err
	}
	if err := r.insertMqttConfigs(ctx, tx, snapshot.MqttConfigs); err != nil {
		return err
	}
	if err := r.insertQueries(ctx, tx, projectID, actorID, snapshot.Queries); err != nil {
		return err
	}
	if err := r.insertMqttSubscriptions(ctx, tx, projectID, actorID, snapshot.MqttSubscriptions); err != nil {
		return err
	}
	if err := r.insertMqttTagGroups(ctx, tx, projectID, actorID, snapshot.MqttTagGroups); err != nil {
		return err
	}
	if err := r.insertMqttTags(ctx, tx, projectID, actorID, snapshot.MqttTags); err != nil {
		return err
	}
	if err := r.insertDataPoints(ctx, tx, projectID, actorID, snapshot.DataPoints); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交快照写入事务失败", err)
	}
	return nil
}

// BuildProjectArtifactV1 基于项目快照构建 Phase 1 产物契约对象。
func BuildProjectArtifactV1(projectID string, snapshot *ProjectSnapshot, generatedAt time.Time) *ProjectArtifactV1 {
	if snapshot == nil {
		snapshot = &ProjectSnapshot{}
	}
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}

	connections := make([]ArtifactConnectionRecord, 0, len(snapshot.Connections))
	connectionIndex := make(map[string]ConnectionRecord, len(snapshot.Connections))
	for _, connection := range snapshot.Connections {
		connectionIndex[connection.ID] = connection
		connections = append(connections, ArtifactConnectionRecord{
			ID:     connection.ID,
			Name:   connection.Name,
			Type:   connection.Type,
			Status: connection.Status,
			Config: cloneSnapshotObject(connection.Config),
		})
	}

	queries := make([]ArtifactQueryRecord, 0, len(snapshot.Queries))
	for _, query := range snapshot.Queries {
		queries = append(queries, ArtifactQueryRecord{
			ID:              query.ID,
			ConnectionID:    query.ConnectionID,
			Name:            query.Name,
			QueryType:       query.QueryType,
			Config:          cloneSnapshotObject(query.Config),
			TimeoutMS:       query.TimeoutMS,
			IsEnabled:       query.IsEnabled,
			Transformer:     query.Transformer,
			CacheEnabled:    query.CacheEnabled,
			CacheTtlSeconds: query.CacheTtlSeconds,
		})
	}

	dataPoints := make([]ArtifactDataPointRecord, 0, len(snapshot.DataPoints))
	for _, dataPoint := range snapshot.DataPoints {
		dataPoints = append(dataPoints, ArtifactDataPointRecord{
			ID:                 dataPoint.ID,
			Path:               dataPoint.Path,
			Name:               dataPoint.Name,
			SourceType:         dataPoint.SourceType,
			SourceID:           dataPoint.SourceID,
			SourceConfig:       cloneSnapshotObject(dataPoint.SourceConfig),
			DataType:           dataPoint.DataType,
			RuntimePermissions: normalizeDataPointRuntimePermissions(dataPoint.RuntimePermissions),
			RefreshMode:        dataPoint.RefreshMode,
			RefreshIntervalMS:  dataPoint.RefreshIntervalMS,
			Status:             dataPoint.Status,
			Unit:               dataPoint.Unit,
			DefaultValue:       dataPoint.DefaultValue,
			Tags:               cloneSnapshotAnyArray(dataPoint.Tags),
		})
	}

	mqttConnections := make([]ArtifactMqttConnectionRecord, 0, len(snapshot.MqttConfigs))
	for _, mqttConfig := range snapshot.MqttConfigs {
		connection := connectionIndex[mqttConfig.ConnectionID]
		mqttConnections = append(mqttConnections, ArtifactMqttConnectionRecord{
			ID:                mqttConfig.ConnectionID,
			Name:              connection.Name,
			Type:              coalesceProtocolType(connection.Type, "mqtt"),
			Status:            connection.Status,
			BrokerURL:         mqttConfig.BrokerURL,
			Protocol:          mqttConfig.Protocol,
			Port:              mqttConfig.Port,
			ClientID:          mqttConfig.ClientID,
			Username:          mqttConfig.Username,
			Password:          mqttConfig.Password,
			Keepalive:         mqttConfig.Keepalive,
			CleanSession:      mqttConfig.CleanSession,
			QOS:               mqttConfig.QOS,
			ReconnectPeriodMS: mqttConfig.ReconnectPeriodMS,
			ConnectTimeoutMS:  mqttConfig.ConnectTimeoutMS,
			Will:              cloneSnapshotObject(mqttConfig.Will),
			SSLConfig:         cloneSnapshotObject(mqttConfig.SSLConfig),
		})
	}

	protocols := ArtifactProtocolsPayload{
		Kafka:     make([]ArtifactProtocolRecord, 0),
		HTTP:      make([]ArtifactProtocolRecord, 0),
		Websocket: make([]ArtifactProtocolRecord, 0),
		Redis:     make([]ArtifactProtocolRecord, 0),
	}
	for _, connection := range snapshot.Connections {
		protocolRecord := ArtifactProtocolRecord{
			ID:     connection.ID,
			Name:   connection.Name,
			Type:   connection.Type,
			Status: connection.Status,
			Config: cloneSnapshotObject(connection.Config),
		}
		switch connection.Type {
		case "kafka":
			protocols.Kafka = append(protocols.Kafka, protocolRecord)
		case "http":
			protocols.HTTP = append(protocols.HTTP, protocolRecord)
		case "websocket":
			protocols.Websocket = append(protocols.Websocket, protocolRecord)
		case "redis":
			protocols.Redis = append(protocols.Redis, protocolRecord)
		}
	}

	return &ProjectArtifactV1{
		Version:     ProjectArtifactVersionV1,
		ProjectID:   projectID,
		GeneratedAt: generatedAt.UTC(),
		Connections: connections,
		Queries:     queries,
		DataPoints:  dataPoints,
		Mqtt: ArtifactMqttPayload{
			Connections:   mqttConnections,
			Subscriptions: append([]SnapshotMqttSubscriptionRecord{}, snapshot.MqttSubscriptions...),
			TagGroups:     append([]SnapshotMqttTagGroupRecord{}, snapshot.MqttTagGroups...),
			Tags:          append([]SnapshotMqttTagRecord{}, snapshot.MqttTags...),
		},
		Protocols: protocols,
	}
}

func (r *ProjectSnapshotRepository) listConnections(ctx context.Context, projectID string) ([]ConnectionRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, name, type, status, metadata, created_at, updated_at
        FROM data_connections
        WHERE project_id = $1
        ORDER BY created_at ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询快照连接失败", err)
	}
	defer rows.Close()

	result := make([]ConnectionRecord, 0)
	for rows.Next() {
		record, scanErr := scanConnection(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照连接失败", err)
	}
	return result, nil
}

func (r *ProjectSnapshotRepository) listQueries(ctx context.Context, projectID string) ([]QueryRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, connection_id, name, description, category, query_type, config, transformer,
               is_enabled, timeout_ms, cache_enabled, cache_ttl_seconds, created_by, updated_by, created_at, updated_at
        FROM data_queries
        WHERE project_id = $1
        ORDER BY created_at ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询快照查询失败", err)
	}
	defer rows.Close()

	result := make([]QueryRecord, 0)
	for rows.Next() {
		record, scanErr := scanQueryRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照查询失败", err)
	}
	return result, nil
}

func (r *ProjectSnapshotRepository) listDataPoints(ctx context.Context, projectID string) ([]DataPointRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, path, name, description, source_type, source_id, source_config, data_type,
               unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags, runtime_permissions,
               refresh_mode, refresh_interval_ms, status, created_by, updated_by, created_at, updated_at
        FROM data_points
        WHERE project_id = $1
        ORDER BY created_at ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询快照数据点失败", err)
	}
	defer rows.Close()

	result := make([]DataPointRecord, 0)
	for rows.Next() {
		record, scanErr := scanDataPointRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照数据点失败", err)
	}
	return result, nil
}

func (r *ProjectSnapshotRepository) listMqttConfigs(ctx context.Context, projectID string) ([]SnapshotMqttConfigRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT
            cfg.connection_id,
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
            cfg.ssl_config
        FROM data_mqtt_configs cfg
        JOIN data_connections conn ON conn.id = cfg.connection_id
        WHERE conn.project_id = $1
        ORDER BY cfg.connection_id
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询快照 MQTT 配置失败", err)
	}
	defer rows.Close()

	result := make([]SnapshotMqttConfigRecord, 0)
	for rows.Next() {
		record := SnapshotMqttConfigRecord{}
		var willBytes, sslConfigBytes []byte
		if err := rows.Scan(
			&record.ConnectionID,
			&record.BrokerURL,
			&record.Protocol,
			&record.Port,
			&record.ClientID,
			&record.Username,
			&record.Password,
			&record.Keepalive,
			&record.CleanSession,
			&record.QOS,
			&record.ReconnectPeriodMS,
			&record.ConnectTimeoutMS,
			&willBytes,
			&sslConfigBytes,
		); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照 MQTT 配置失败", err)
		}
		record.Will = mustJSONObject(willBytes)
		record.SSLConfig = mustJSONObject(sslConfigBytes)
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照 MQTT 配置失败", err)
	}
	return result, nil
}

func (r *ProjectSnapshotRepository) listMqttSubscriptions(ctx context.Context, projectID string) ([]SnapshotMqttSubscriptionRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, connection_id, name, topic, qos, description, is_enabled, message_retention, created_at, updated_at
        FROM data_mqtt_subscriptions
        WHERE project_id = $1
        ORDER BY created_at ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询快照 MQTT 订阅失败", err)
	}
	defer rows.Close()

	result := make([]SnapshotMqttSubscriptionRecord, 0)
	for rows.Next() {
		record := SnapshotMqttSubscriptionRecord{}
		if err := rows.Scan(
			&record.ID,
			&record.ProjectID,
			&record.ConnectionID,
			&record.Name,
			&record.Topic,
			&record.QOS,
			&record.Description,
			&record.IsEnabled,
			&record.MessageRetention,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照 MQTT 订阅失败", err)
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照 MQTT 订阅失败", err)
	}
	return result, nil
}

func (r *ProjectSnapshotRepository) listMqttTagGroups(ctx context.Context, projectID string) ([]SnapshotMqttTagGroupRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, subscription_id, name, code, description, color, icon, display_order, created_at, updated_at
        FROM data_mqtt_tag_groups
        WHERE project_id = $1
        ORDER BY display_order ASC, created_at ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询快照 MQTT 变量组失败", err)
	}
	defer rows.Close()

	result := make([]SnapshotMqttTagGroupRecord, 0)
	for rows.Next() {
		record := SnapshotMqttTagGroupRecord{}
		if err := rows.Scan(
			&record.ID,
			&record.ProjectID,
			&record.SubscriptionID,
			&record.Name,
			&record.Code,
			&record.Description,
			&record.Color,
			&record.Icon,
			&record.Order,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照 MQTT 变量组失败", err)
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照 MQTT 变量组失败", err)
	}
	return result, nil
}

func (r *ProjectSnapshotRepository) listMqttTags(ctx context.Context, projectID string) ([]SnapshotMqttTagRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, subscription_id, group_id, name, code, description, data_type, parse_type, parse_rule,
               default_value, unit, transform, validation, is_enabled, display_order, created_at, updated_at
        FROM data_mqtt_tags
        WHERE project_id = $1
        ORDER BY display_order ASC, created_at ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询快照 MQTT 变量失败", err)
	}
	defer rows.Close()

	result := make([]SnapshotMqttTagRecord, 0)
	for rows.Next() {
		record := SnapshotMqttTagRecord{}
		var validationBytes []byte
		if err := rows.Scan(
			&record.ID,
			&record.ProjectID,
			&record.SubscriptionID,
			&record.GroupID,
			&record.Name,
			&record.Code,
			&record.Description,
			&record.DataType,
			&record.ParseType,
			&record.ParseRule,
			&record.DefaultValue,
			&record.Unit,
			&record.Transform,
			&validationBytes,
			&record.IsEnabled,
			&record.Order,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照 MQTT 变量失败", err)
		}
		record.Validation = mustJSONObject(validationBytes)
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照 MQTT 变量失败", err)
	}
	return result, nil
}

func (r *ProjectSnapshotRepository) deleteProjectSnapshot(ctx context.Context, tx pgx.Tx, projectID string) error {
	for _, sqlText := range []string{
		`DELETE FROM data_mqtt_tags WHERE project_id = $1`,
		`DELETE FROM data_mqtt_tag_groups WHERE project_id = $1`,
		`DELETE FROM data_mqtt_subscriptions WHERE project_id = $1`,
		`DELETE FROM data_points WHERE project_id = $1`,
		`DELETE FROM data_queries WHERE project_id = $1`,
		`DELETE FROM data_connections WHERE project_id = $1`,
	} {
		if _, err := tx.Exec(ctx, sqlText, projectID); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "清理旧快照失败", err)
		}
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertConnections(ctx context.Context, tx pgx.Tx, projectID, actorID string, connections []ConnectionRecord) error {
	for _, connection := range connections {
		configBytes, err := marshalConfig(connection.Config)
		if err != nil {
			return err
		}
		category := deriveConnectionCategory(connection.Type)
		createdAt := coalesceTime(connection.CreatedAt)
		updatedAt := coalesceTime(connection.UpdatedAt)
		if _, err := tx.Exec(ctx, `
            INSERT INTO data_connections (
                id, project_id, name, type, category, status, metadata, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11)
        `, connection.ID, projectID, connection.Name, connection.Type, category, connection.Status, string(configBytes), actorID, actorID, createdAt, updatedAt); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照连接失败", err)
		}
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertMqttConfigs(ctx context.Context, tx pgx.Tx, mqttConfigs []SnapshotMqttConfigRecord) error {
	for _, config := range mqttConfigs {
		willBytes, err := marshalSnapshotObject(config.Will)
		if err != nil {
			return err
		}
		sslConfigBytes, err := marshalSnapshotObject(config.SSLConfig)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
            INSERT INTO data_mqtt_configs (
                connection_id, broker_url, protocol, port, client_id, username, password,
                keepalive, clean_session, qos, reconnect_period_ms, connect_timeout_ms, will, ssl_config
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::jsonb, $14::jsonb)
        `, config.ConnectionID, config.BrokerURL, config.Protocol, config.Port, config.ClientID, config.Username, config.Password, config.Keepalive, config.CleanSession, config.QOS, config.ReconnectPeriodMS, config.ConnectTimeoutMS, nullableJSONString(willBytes), nullableJSONString(sslConfigBytes)); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照 MQTT 配置失败", err)
		}
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertQueries(ctx context.Context, tx pgx.Tx, projectID, actorID string, queries []QueryRecord) error {
	for _, query := range queries {
		configBytes, err := marshalSnapshotObject(query.Config)
		if err != nil {
			return err
		}
		createdAt := coalesceTime(query.CreatedAt)
		updatedAt := coalesceTime(query.UpdatedAt)
		if _, err := tx.Exec(ctx, `
            INSERT INTO data_queries (
                id, project_id, connection_id, name, description, category, query_type, config, transformer,
                is_enabled, timeout_ms, cache_enabled, cache_ttl_seconds, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11, $12, $13, $14, $15, $16, $17)
        `, query.ID, projectID, query.ConnectionID, query.Name, query.Description, query.Category, query.QueryType, string(configBytes), query.Transformer, query.IsEnabled, query.TimeoutMS, query.CacheEnabled, query.CacheTtlSeconds, actorID, actorID, createdAt, updatedAt); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照查询失败", err)
		}
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertMqttSubscriptions(ctx context.Context, tx pgx.Tx, projectID, actorID string, subscriptions []SnapshotMqttSubscriptionRecord) error {
	for _, subscription := range subscriptions {
		createdAt := coalesceTime(subscription.CreatedAt)
		updatedAt := coalesceTime(subscription.UpdatedAt)
		if _, err := tx.Exec(ctx, `
            INSERT INTO data_mqtt_subscriptions (
                id, project_id, connection_id, name, topic, qos, description, is_enabled,
                message_retention, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
        `, subscription.ID, projectID, subscription.ConnectionID, subscription.Name, subscription.Topic, subscription.QOS, subscription.Description, subscription.IsEnabled, subscription.MessageRetention, actorID, actorID, createdAt, updatedAt); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照 MQTT 订阅失败", err)
		}
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertMqttTagGroups(ctx context.Context, tx pgx.Tx, projectID, actorID string, groups []SnapshotMqttTagGroupRecord) error {
	for _, group := range groups {
		createdAt := coalesceTime(group.CreatedAt)
		updatedAt := coalesceTime(group.UpdatedAt)
		if _, err := tx.Exec(ctx, `
            INSERT INTO data_mqtt_tag_groups (
                id, project_id, subscription_id, name, code, description, color, icon, display_order,
                created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
        `, group.ID, projectID, group.SubscriptionID, group.Name, group.Code, group.Description, group.Color, group.Icon, group.Order, actorID, actorID, createdAt, updatedAt); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照 MQTT 变量组失败", err)
		}
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertMqttTags(ctx context.Context, tx pgx.Tx, projectID, actorID string, tags []SnapshotMqttTagRecord) error {
	for _, tag := range tags {
		validationBytes, err := marshalSnapshotObject(tag.Validation)
		if err != nil {
			return err
		}
		createdAt := coalesceTime(tag.CreatedAt)
		updatedAt := coalesceTime(tag.UpdatedAt)
		if _, err := tx.Exec(ctx, `
            INSERT INTO data_mqtt_tags (
                id, project_id, subscription_id, group_id, name, code, description, data_type, parse_type, parse_rule,
                default_value, unit, transform, validation, is_enabled, display_order, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14::jsonb, $15, $16, $17, $18, $19, $20)
        `, tag.ID, projectID, tag.SubscriptionID, tag.GroupID, tag.Name, tag.Code, tag.Description, tag.DataType, tag.ParseType, tag.ParseRule, tag.DefaultValue, tag.Unit, tag.Transform, nullableJSONString(validationBytes), tag.IsEnabled, tag.Order, actorID, actorID, createdAt, updatedAt); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照 MQTT 变量失败", err)
		}
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertDataPoints(ctx context.Context, tx pgx.Tx, projectID, actorID string, datapoints []DataPointRecord) error {
	for _, datapoint := range datapoints {
		sourceConfigBytes, err := marshalSnapshotObject(datapoint.SourceConfig)
		if err != nil {
			return err
		}
		tagsBytes, err := marshalSnapshotArray(datapoint.Tags)
		if err != nil {
			return err
		}
		runtimePermissionsBytes, err := marshalDataPointRuntimePermissions(datapoint.RuntimePermissions)
		if err != nil {
			return err
		}
		createdAt := coalesceTime(datapoint.CreatedAt)
		updatedAt := coalesceTime(datapoint.UpdatedAt)
		if _, err := tx.Exec(ctx, `
            INSERT INTO data_points (
                id, project_id, path, name, description, source_type, source_id, source_config, data_type,
                unit, precision_num, default_value, min_value, max_value, alarm_low, alarm_high, tags, runtime_permissions,
                refresh_mode, refresh_interval_ms, status, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11, $12, $13, $14, $15, $16, $17::jsonb, $18::jsonb, $19, $20, $21, $22, $23, $24, $25)
        `, datapoint.ID, projectID, datapoint.Path, datapoint.Name, datapoint.Description, datapoint.SourceType, datapoint.SourceID, string(sourceConfigBytes), datapoint.DataType, datapoint.Unit, datapoint.PrecisionNum, datapoint.DefaultValue, datapoint.MinValue, datapoint.MaxValue, datapoint.AlarmLow, datapoint.AlarmHigh, string(tagsBytes), string(runtimePermissionsBytes), datapoint.RefreshMode, datapoint.RefreshIntervalMS, datapoint.Status, actorID, actorID, createdAt, updatedAt); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照数据点失败", err)
		}
	}
	return nil
}

func deriveRelationalConfigs(connections []ConnectionRecord) []SnapshotRelationalConfigRecord {
	result := make([]SnapshotRelationalConfigRecord, 0)
	for _, connection := range connections {
		if connection.Type != "relational" {
			continue
		}
		port := 5432
		if parsed := parseSnapshotInt(connection.Config["port"]); parsed > 0 {
			port = parsed
		}
		record := SnapshotRelationalConfigRecord{
			ConnectionID: connection.ID,
			DBType:       parseSnapshotString(connection.Config["dbType"], "postgresql"),
			Host:         parseSnapshotString(connection.Config["host"], ""),
			Port:         port,
			Database:     parseSnapshotString(connection.Config["database"], ""),
			Username:     parseSnapshotString(connection.Config["username"], ""),
			Password:     parseSnapshotString(connection.Config["password"], ""),
			Schema:       parseSnapshotOptionalString(connection.Config["schema"]),
			Charset:      parseSnapshotOptionalString(connection.Config["charset"]),
			Timezone:     parseSnapshotOptionalString(connection.Config["timezone"]),
			SSL:          parseSnapshotBool(connection.Config["ssl"]),
			SSLConfig:    parseSnapshotObject(connection.Config["sslConfig"]),
		}
		result = append(result, record)
	}
	return result
}

func cloneSnapshotObject(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}

func cloneSnapshotAnyArray(value []any) []any {
	if len(value) == 0 {
		return []any{}
	}
	return append([]any{}, value...)
}

func coalesceProtocolType(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func deriveConnectionCategory(connectionType string) string {
	switch connectionType {
	case "relational":
		return "database"
	case "mqtt":
		return "message"
	case "kafka", "http", "websocket", "redis", "opcua", "modbus", "s7", "tdengine":
		return "protocol"
	default:
		return "api"
	}
}

func marshalSnapshotObject(value map[string]any) ([]byte, error) {
	if value == nil {
		value = map[string]any{}
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照对象 JSON 序列化失败", err)
	}
	return payload, nil
}

func marshalSnapshotArray(value []any) ([]byte, error) {
	if value == nil {
		value = []any{}
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照数组 JSON 序列化失败", err)
	}
	return payload, nil
}

func nullableJSONString(payload []byte) any {
	if len(payload) == 0 || string(payload) == "{}" {
		return nil
	}
	return string(payload)
}

func mustJSONObject(payload []byte) map[string]any {
	if len(payload) == 0 {
		return map[string]any{}
	}
	var result map[string]any
	if err := json.Unmarshal(payload, &result); err != nil || result == nil {
		return map[string]any{}
	}
	return result
}

func parseSnapshotString(value any, fallback string) string {
	typed, ok := value.(string)
	if !ok {
		return fallback
	}
	if typed == "" {
		return fallback
	}
	return typed
}

func parseSnapshotOptionalString(value any) *string {
	typed, ok := value.(string)
	if !ok || typed == "" {
		return nil
	}
	return &typed
}

func parseSnapshotBool(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}

func parseSnapshotInt(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	default:
		return 0
	}
}

func parseSnapshotObject(value any) map[string]any {
	typed, ok := value.(map[string]any)
	if !ok || typed == nil {
		return map[string]any{}
	}
	return typed
}

func coalesceTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value
}
