package repository

import (
	"context"
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
	ID               string         `json:"id"`
	ProjectID        string         `json:"projectId"`
	ConnectionID     string         `json:"connectionId"`
	Name             string         `json:"name"`
	Topic            string         `json:"topic"`
	QOS              int            `json:"qos"`
	UsageMode        string         `json:"usageMode"`
	Description      *string        `json:"description"`
	MessageRetention int            `json:"messageRetention"`
	DefaultBatchRule map[string]any `json:"defaultBatchParseRule"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

// SnapshotMqttTagRecord 表示导入导出使用的 MQTT 变量投影。
type SnapshotMqttTagRecord struct {
	ID             string         `json:"id"`
	ProjectID      string         `json:"projectId"`
	SubscriptionID string         `json:"subscriptionId"`
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
	Order          int            `json:"order"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// ProjectSnapshot 表示工程级数据域快照。
type ProjectSnapshot struct {
	Connections         []ConnectionRecord                 `json:"connections"`
	RelationalConfigs   []SnapshotRelationalConfigRecord   `json:"relationalConfigs"`
	Queries             []QueryRecord                      `json:"queries"`
	MqttConfigs         []SnapshotMqttConfigRecord         `json:"mqttConfigs"`
	MqttSubscriptions   []SnapshotMqttSubscriptionRecord   `json:"mqttSubscriptions"`
	MqttTags            []SnapshotMqttTagRecord            `json:"mqttTags"`
	DataPoints          []DataPointRecord                  `json:"datapoints"`
	ComputeUnits        []ComputeUnitRecord                `json:"computeUnits"`
	AlarmPolicyGroups   []AlarmPolicyGroupRecord           `json:"alarmPolicyGroups"`
	AlarmPolicies       []AlarmPolicyRecord                `json:"alarmPolicies"`
	AlarmSettings       *AlarmProjectSettingsRecord        `json:"alarmSettings"`
	AlarmChannels       []AlarmNotificationChannelRecord   `json:"alarmChannels"`
	AlarmChannelSecrets []SnapshotAlarmChannelSecretRecord `json:"alarmChannelSecrets"`
	HistoryStorage      []HistoryStorageConfigRecord       `json:"historyStorage"`
}

// SnapshotAlarmChannelSecretRecord 仅保存密文，快照与 API 均不包含渠道密钥明文。
type SnapshotAlarmChannelSecretRecord struct {
	ChannelID            string    `json:"channelId"`
	SecretKey            string    `json:"secretKey"`
	EncryptedValue       []byte    `json:"encryptedValue"`
	EncryptionKeyVersion string    `json:"encryptionKeyVersion"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
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
	DisplayOrder       int                         `json:"displayOrder"`
	Unit               *string                     `json:"unit,omitempty"`
	DefaultValue       *string                     `json:"defaultValue,omitempty"`
	Tags               []any                       `json:"tags"`
}

// ArtifactComputeUnitRecord 表示产物层计算单元对象。
type ArtifactComputeUnitRecord struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    *string        `json:"description,omitempty"`
	Language       string         `json:"language"`
	ScriptCode     string         `json:"scriptCode"`
	TriggerType    string         `json:"triggerType"`
	TriggerConfig  map[string]any `json:"triggerConfig"`
	InputBindings  map[string]any `json:"inputBindings"`
	OutputBindings map[string]any `json:"outputBindings"`
	Dependencies   []any          `json:"dependencies"`
	TimeoutMS      int            `json:"timeoutMs"`
	IsEnabled      bool           `json:"isEnabled"`
}

type ArtifactAlarmSecretRef struct {
	ChannelID  string `json:"channelId"`
	SecretKey  string `json:"secretKey"`
	KeyVersion string `json:"keyVersion"`
}

// ArtifactAlarmsPayload 是节点消费的 alarm.policy.v1 完整开发态契约。
type ArtifactAlarmsPayload struct {
	SchemaVersion string                           `json:"schemaVersion"`
	Groups        []AlarmPolicyGroupRecord         `json:"groups"`
	Policies      []AlarmPolicyRecord              `json:"policies"`
	Settings      *AlarmProjectSettingsRecord      `json:"settings"`
	Channels      []AlarmNotificationChannelRecord `json:"channels"`
	SecretRefs    []ArtifactAlarmSecretRef         `json:"secretRefs"`
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
	Tags          []SnapshotMqttTagRecord          `json:"tags"`
}

// ArtifactProtocolsPayload 表示产物层其他协议区块。
type ArtifactProtocolsPayload struct {
	Kafka     []ArtifactProtocolRecord `json:"kafka"`
	HTTP      []ArtifactProtocolRecord `json:"http"`
	Websocket []ArtifactProtocolRecord `json:"websocket"`
	Redis     []ArtifactProtocolRecord `json:"redis"`
	TDengine  []ArtifactProtocolRecord `json:"tdengine"`
}

// ArtifactBuiltinStoresPayload 表示产物层 IF 内置运行库契约。
type ArtifactBuiltinStoresPayload struct {
	Relations      []ArtifactBuiltinRelationStore   `json:"relations"`
	Timeseries     []ArtifactBuiltinTimeseriesStore `json:"timeseries"`
	RealtimeSpaces []ArtifactBuiltinRealtimeStore   `json:"realtimeSpaces"`
}

type ArtifactBuiltinRelationStore struct {
	ID         string   `json:"id"`
	RuntimeKey string   `json:"runtimeKey"`
	Name       string   `json:"name"`
	Schema     string   `json:"schema"`
	DDLVersion string   `json:"ddlVersion"`
	Queries    []string `json:"queries"`
}

type ArtifactBuiltinTimeseriesStore struct {
	ID         string   `json:"id"`
	RuntimeKey string   `json:"runtimeKey"`
	Name       string   `json:"name"`
	Schema     string   `json:"schema"`
	DDLVersion string   `json:"ddlVersion"`
	Tables     []string `json:"tables"`
}

type ArtifactBuiltinRealtimeStore struct {
	ID                string   `json:"id"`
	RuntimeKey        string   `json:"runtimeKey"`
	Name              string   `json:"name"`
	Enabled           bool     `json:"enabled"`
	Namespace         string   `json:"namespace"`
	DefaultTtlSeconds int      `json:"defaultTtlSeconds"`
	Keys              []string `json:"keys"`
}

// ProjectArtifactV1 表示 Phase 1 项目级数据产物。
type ProjectArtifactV1 struct {
	Version       string                       `json:"version"`
	ProjectID     string                       `json:"projectId"`
	GeneratedAt   time.Time                    `json:"generatedAt"`
	Connections   []ArtifactConnectionRecord   `json:"connections"`
	Queries       []ArtifactQueryRecord        `json:"queries"`
	DataPoints    []ArtifactDataPointRecord    `json:"datapoints"`
	Compute       []ArtifactComputeUnitRecord  `json:"compute"`
	Alarms        ArtifactAlarmsPayload        `json:"alarms"`
	Mqtt          ArtifactMqttPayload          `json:"mqtt"`
	Protocols     ArtifactProtocolsPayload     `json:"protocols"`
	BuiltinStores ArtifactBuiltinStoresPayload `json:"builtinStores"`
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
	mqttTags, err := r.listMqttTags(ctx, projectID)
	if err != nil {
		return nil, err
	}
	computeUnits, err := r.listComputeUnits(ctx, projectID)
	if err != nil {
		return nil, err
	}
	alarmRepository := NewAlarmPolicyRepository(r.pool)
	alarmGroups, err := alarmRepository.ListAllGroups(ctx, projectID)
	if err != nil {
		return nil, err
	}
	alarmPolicies, err := alarmRepository.ListAllPolicies(ctx, projectID)
	if err != nil {
		return nil, err
	}
	alarmSettings, err := alarmRepository.GetProjectSettings(ctx, projectID)
	if err != nil {
		return nil, err
	}
	alarmChannels, err := alarmRepository.ListChannels(ctx, projectID)
	if err != nil {
		return nil, err
	}
	alarmChannelSecrets, err := r.listAlarmChannelSecrets(ctx, projectID)
	if err != nil {
		return nil, err
	}
	historyStorage, err := r.listHistoryStorageConfigs(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return &ProjectSnapshot{
		Connections:         connections,
		RelationalConfigs:   deriveRelationalConfigs(connections),
		Queries:             queries,
		MqttConfigs:         mqttConfigs,
		MqttSubscriptions:   mqttSubscriptions,
		MqttTags:            mqttTags,
		DataPoints:          datapoints,
		ComputeUnits:        computeUnits,
		AlarmPolicyGroups:   alarmGroups,
		AlarmPolicies:       alarmPolicies,
		AlarmSettings:       alarmSettings,
		AlarmChannels:       alarmChannels,
		AlarmChannelSecrets: alarmChannelSecrets,
		HistoryStorage:      historyStorage,
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
	if err := r.insertMqttTags(ctx, tx, projectID, actorID, snapshot.MqttTags); err != nil {
		return err
	}
	if err := r.insertDataPoints(ctx, tx, projectID, actorID, snapshot.DataPoints); err != nil {
		return err
	}
	if err := r.insertHistoryStorageConfigs(ctx, tx, projectID, actorID, snapshot.HistoryStorage); err != nil {
		return err
	}
	if err := r.insertComputeUnits(ctx, tx, projectID, actorID, snapshot.ComputeUnits); err != nil {
		return err
	}
	if err := r.insertAlarmPolicyGroups(ctx, tx, projectID, actorID, snapshot.AlarmPolicyGroups); err != nil {
		return err
	}
	if err := r.insertAlarmChannels(ctx, tx, projectID, actorID, snapshot.AlarmChannels, snapshot.AlarmChannelSecrets); err != nil {
		return err
	}
	if err := r.insertAlarmSettings(ctx, tx, projectID, actorID, snapshot.AlarmSettings); err != nil {
		return err
	}
	if err := r.insertAlarmPolicies(ctx, tx, projectID, actorID, snapshot.AlarmPolicies); err != nil {
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
			Config: buildArtifactConnectionConfig(connection),
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
			DisplayOrder:       dataPoint.DisplayOrder,
			Unit:               dataPoint.Unit,
			DefaultValue:       dataPoint.DefaultValue,
			Tags:               cloneSnapshotAnyArray(dataPoint.Tags),
		})
	}

	computeUnits := make([]ArtifactComputeUnitRecord, 0, len(snapshot.ComputeUnits))
	for _, unit := range snapshot.ComputeUnits {
		computeUnits = append(computeUnits, ArtifactComputeUnitRecord{
			ID:             unit.ID,
			Name:           unit.Name,
			Description:    unit.Description,
			Language:       unit.Language,
			ScriptCode:     unit.ScriptCode,
			TriggerType:    unit.TriggerType,
			TriggerConfig:  cloneSnapshotObject(unit.TriggerConfig),
			InputBindings:  cloneSnapshotObject(unit.InputBindings),
			OutputBindings: cloneSnapshotObject(unit.OutputBinding),
			Dependencies:   cloneSnapshotAnyArray(unit.Dependencies),
			TimeoutMS:      unit.TimeoutMS,
			IsEnabled:      unit.IsEnabled,
		})
	}

	secretRefs := make([]ArtifactAlarmSecretRef, 0, len(snapshot.AlarmChannelSecrets))
	for _, secret := range snapshot.AlarmChannelSecrets {
		secretRefs = append(secretRefs, ArtifactAlarmSecretRef{
			ChannelID:  secret.ChannelID,
			SecretKey:  secret.SecretKey,
			KeyVersion: secret.EncryptionKeyVersion,
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
		TDengine:  make([]ArtifactProtocolRecord, 0),
	}
	builtinStores := buildBuiltinStores(snapshot.Connections)
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
		case "tdengine":
			protocols.TDengine = append(protocols.TDengine, protocolRecord)
		}
	}

	return &ProjectArtifactV1{
		Version:     ProjectArtifactVersionV1,
		ProjectID:   projectID,
		GeneratedAt: generatedAt.UTC(),
		Connections: connections,
		Queries:     queries,
		DataPoints:  dataPoints,
		Compute:     computeUnits,
		Alarms: ArtifactAlarmsPayload{
			SchemaVersion: "alarm.policy.v1",
			Groups:        append([]AlarmPolicyGroupRecord{}, snapshot.AlarmPolicyGroups...),
			Policies:      append([]AlarmPolicyRecord{}, snapshot.AlarmPolicies...),
			Settings:      snapshot.AlarmSettings,
			Channels:      append([]AlarmNotificationChannelRecord{}, snapshot.AlarmChannels...),
			SecretRefs:    secretRefs,
		},
		Mqtt: ArtifactMqttPayload{
			Connections:   mqttConnections,
			Subscriptions: append([]SnapshotMqttSubscriptionRecord{}, snapshot.MqttSubscriptions...),
			Tags:          append([]SnapshotMqttTagRecord{}, snapshot.MqttTags...),
		},
		Protocols:     protocols,
		BuiltinStores: builtinStores,
	}
}

func buildArtifactConnectionConfig(connection ConnectionRecord) map[string]any {
	if !strings.HasPrefix(connection.Type, "builtin.") {
		return cloneSnapshotObject(connection.Config)
	}
	result := map[string]any{}
	allowedKeys := map[string]struct{}{
		"store":             {},
		"runtimeKey":        {},
		"runtimeSchema":     {},
		"ddlVersion":        {},
		"namespace":         {},
		"defaultTtlSeconds": {},
		"topicPrefix":       {},
		"description":       {},
		"allowDdlTest":      {},
	}
	for key, value := range connection.Config {
		if _, ok := allowedKeys[key]; ok {
			result[key] = value
		}
	}
	return result
}

func buildBuiltinStores(connections []ConnectionRecord) ArtifactBuiltinStoresPayload {
	payload := ArtifactBuiltinStoresPayload{}
	for _, connection := range connections {
		runtimeKey := artifactConfigString(connection.Config, "runtimeKey", "")
		if runtimeKey == "" {
			continue
		}
		switch connection.Type {
		case "builtin.relation":
			payload.Relations = append(payload.Relations, ArtifactBuiltinRelationStore{
				ID:         connection.ID,
				RuntimeKey: runtimeKey,
				Name:       connection.Name,
				Schema:     runtimeKey,
				DDLVersion: artifactConfigString(connection.Config, "ddlVersion", "2026-05-24.1"),
				Queries:    []string{},
			})
		case "builtin.timeseries":
			payload.Timeseries = append(payload.Timeseries, ArtifactBuiltinTimeseriesStore{
				ID:         connection.ID,
				RuntimeKey: runtimeKey,
				Name:       connection.Name,
				Schema:     runtimeKey,
				DDLVersion: artifactConfigString(connection.Config, "ddlVersion", "2026-05-24.1"),
				Tables:     []string{},
			})
		case "builtin.realtime":
			payload.RealtimeSpaces = append(payload.RealtimeSpaces, ArtifactBuiltinRealtimeStore{
				ID:                connection.ID,
				RuntimeKey:        runtimeKey,
				Name:              connection.Name,
				Enabled:           true,
				Namespace:         runtimeKey,
				DefaultTtlSeconds: artifactConfigInt(connection.Config, "defaultTtlSeconds", 300),
				Keys:              []string{},
			})
		}
	}
	return payload
}

func artifactConfigString(config map[string]any, key string, defaultValue string) string {
	value, ok := config[key].(string)
	if !ok || value == "" {
		return defaultValue
	}
	return value
}

func artifactConfigInt(config map[string]any, key string, defaultValue int) int {
	switch typed := config[key].(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case int64:
		if typed > 0 {
			return int(typed)
		}
	case float64:
		if typed > 0 {
			return int(typed)
		}
	}
	return defaultValue
}

func (r *ProjectSnapshotRepository) listConnections(ctx context.Context, projectID string) ([]ConnectionRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, name, type, category, status, metadata, display_order, created_at, updated_at
        FROM data_connections
        WHERE project_id = $1
        ORDER BY display_order ASC, created_at ASC
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
        SELECT `+dataPointSelectColumns+`
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

func (r *ProjectSnapshotRepository) listComputeUnits(ctx context.Context, projectID string) ([]ComputeUnitRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, name, description, folder_id, language, script_code, trigger_type, trigger_config,
               input_bindings, output_bindings, dependencies, timeout_ms, is_enabled, created_at, updated_at
        FROM data_compute_units
        WHERE project_id = $1
        ORDER BY created_at ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询快照计算单元失败", err)
	}
	defer rows.Close()

	result := make([]ComputeUnitRecord, 0)
	for rows.Next() {
		record, scanErr := scanComputeUnit(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照计算单元失败", err)
	}
	return result, nil
}

func (r *ProjectSnapshotRepository) listAlarmChannelSecrets(ctx context.Context, projectID string) ([]SnapshotAlarmChannelSecretRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT secret.channel_id, secret.secret_key, secret.encrypted_value,
               secret.encryption_key_version, secret.created_at, secret.updated_at
        FROM data_alarm_notification_channel_secrets secret
        JOIN data_alarm_notification_channels channel ON channel.id = secret.channel_id
        WHERE channel.project_id = $1
        ORDER BY channel.created_at, secret.secret_key
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询快照报警渠道密钥失败", err)
	}
	defer rows.Close()

	result := make([]SnapshotAlarmChannelSecretRecord, 0)
	for rows.Next() {
		var record SnapshotAlarmChannelSecretRecord
		if scanErr := rows.Scan(&record.ChannelID, &record.SecretKey, &record.EncryptedValue, &record.EncryptionKeyVersion, &record.CreatedAt, &record.UpdatedAt); scanErr != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照报警渠道密钥失败", scanErr)
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照报警渠道密钥失败", err)
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
        SELECT id, project_id, connection_id, name, topic, qos, usage_mode, description, message_retention, default_batch_parse_rule, created_at, updated_at
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
		var defaultBatchRuleBytes []byte
		if err := rows.Scan(
			&record.ID,
			&record.ProjectID,
			&record.ConnectionID,
			&record.Name,
			&record.Topic,
			&record.QOS,
			&record.UsageMode,
			&record.Description,
			&record.MessageRetention,
			&defaultBatchRuleBytes,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照 MQTT 订阅失败", err)
		}
		record.DefaultBatchRule = mustJSONObject(defaultBatchRuleBytes)
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照 MQTT 订阅失败", err)
	}
	return result, nil
}

func (r *ProjectSnapshotRepository) listMqttTags(ctx context.Context, projectID string) ([]SnapshotMqttTagRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, subscription_id, name, code, description, data_type, parse_type, parse_rule,
               default_value, unit, transform, validation, display_order, created_at, updated_at
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

func (r *ProjectSnapshotRepository) listHistoryStorageConfigs(ctx context.Context, projectID string) ([]HistoryStorageConfigRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+historyStorageConfigColumns+`
        FROM data_history_storage_configs WHERE project_id=$1 ORDER BY created_at,id`, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照历史存储配置失败", err)
	}
	defer rows.Close()
	result := make([]HistoryStorageConfigRecord, 0)
	for rows.Next() {
		config, err := scanHistoryStorageConfig(rows)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析快照历史存储配置失败", err)
		}
		config.Targets, err = r.listHistoryStorageTargets(ctx, projectID, config.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, config)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照历史存储配置失败", err)
	}
	return result, nil
}

func (r *ProjectSnapshotRepository) listHistoryStorageTargets(ctx context.Context, projectID, configID string) ([]HistoryStorageTargetRecord, error) {
	rows, err := r.pool.Query(ctx, `SELECT target.id,target.project_id,target.config_id,target.connection_id,
        connection.name,connection.type,connection.status,target.is_primary,target.sort_order,target.retention_days,
        target.created_at,target.updated_at
      FROM data_history_storage_targets target
      JOIN data_connections connection ON connection.id=target.connection_id
      WHERE target.project_id=$1 AND target.config_id=$2
      ORDER BY target.is_primary DESC,target.sort_order,target.id`, projectID, configID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照历史存储目标失败", err)
	}
	defer rows.Close()
	result := make([]HistoryStorageTargetRecord, 0)
	for rows.Next() {
		var target HistoryStorageTargetRecord
		if err := rows.Scan(&target.ID, &target.ProjectID, &target.ConfigID, &target.ConnectionID, &target.ConnectionName, &target.ConnectionType, &target.ConnectionStatus, &target.IsPrimary, &target.SortOrder, &target.RetentionDays, &target.CreatedAt, &target.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, target)
	}
	return result, rows.Err()
}

func (r *ProjectSnapshotRepository) insertHistoryStorageConfigs(ctx context.Context, tx pgx.Tx, projectID, actorID string, configs []HistoryStorageConfigRecord) error {
	for _, config := range configs {
		createdAt := coalesceTime(config.CreatedAt)
		updatedAt := coalesceTime(config.UpdatedAt)
		if config.IsEnabled && len(config.Targets) == 0 {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照中已开启的历史存储配置缺少目标")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO data_history_storage_configs
            (id,project_id,access_source_id,collector_connection_id,datapoint_id,is_enabled,write_mode,
             interval_ms,deadband,max_silence_ms,offline_behavior,created_by,updated_by,created_at,updated_at)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$12,$13,$14)`,
			config.ID, projectID, config.AccessSourceID, config.CollectorConnectionID, config.DatapointID,
			config.IsEnabled, config.WriteMode, config.IntervalMS, config.Deadband, config.MaxSilenceMS,
			config.OfflineBehavior, actorID, createdAt, updatedAt); err != nil {
			return translateHistoryStorageSnapshotError("写入快照历史存储配置失败", err)
		}
		primaryCount := 0
		for _, target := range config.Targets {
			if target.IsPrimary {
				primaryCount++
			}
			targetCreatedAt := coalesceTime(target.CreatedAt)
			targetUpdatedAt := coalesceTime(target.UpdatedAt)
			tag, err := tx.Exec(ctx, `INSERT INTO data_history_storage_targets
                (id,project_id,config_id,connection_id,is_primary,sort_order,retention_days,created_at,updated_at)
                SELECT $1,$2,$3,connection.id,$4,$5,$6,$7,$8
                FROM data_connections connection
                WHERE connection.project_id=$2 AND connection.id=$9 AND connection.type IN ('builtin.timeseries','tdengine')`,
				target.ID, projectID, config.ID, target.IsPrimary, target.SortOrder, target.RetentionDays,
				targetCreatedAt, targetUpdatedAt, target.ConnectionID)
			if err != nil {
				return translateHistoryStorageSnapshotError("写入快照历史存储目标失败", err)
			}
			if tag.RowsAffected() == 0 {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照历史存储目标不存在或类型不受支持")
			}
		}
		if config.IsEnabled && primaryCount != 1 {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照历史存储配置必须且只能有一个主目标")
		}
	}
	return nil
}

func translateHistoryStorageSnapshotError(message string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23505" || pgErr.Code == "23514") {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照历史存储配置引用的来源、数据点或目标无效")
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}

func (r *ProjectSnapshotRepository) deleteProjectSnapshot(ctx context.Context, tx pgx.Tx, projectID string) error {
	for _, sqlText := range []string{
		`DELETE FROM data_history_storage_configs WHERE project_id = $1`,
		`DELETE FROM data_alarm_policies WHERE project_id = $1`,
		`DELETE FROM data_alarm_project_settings WHERE project_id = $1`,
		`DELETE FROM data_alarm_notification_channels WHERE project_id = $1`,
		`DELETE FROM data_alarm_policy_groups WHERE project_id = $1`,
		`DELETE FROM data_alarm_config_sync_requests WHERE project_id = $1`,
		`DELETE FROM data_alarm_config_sync_state WHERE project_id = $1`,
		`DELETE FROM data_compute_units WHERE project_id = $1`,
		`DELETE FROM data_mqtt_tags WHERE project_id = $1`,
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
                id, project_id, name, type, category, status, metadata, display_order, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11, $12)
        `, connection.ID, projectID, connection.Name, connection.Type, category, connection.Status, string(configBytes), connection.DisplayOrder, actorID, actorID, createdAt, updatedAt); err != nil {
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
		usageMode := normalizeSnapshotMqttSubscriptionUsageMode(subscription.UsageMode)
		defaultBatchRuleBytes, err := marshalSnapshotObject(subscription.DefaultBatchRule)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
            INSERT INTO data_mqtt_subscriptions (
                id, project_id, connection_id, name, topic, qos, usage_mode, description,
                message_retention, default_batch_parse_rule, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
        `, subscription.ID, projectID, subscription.ConnectionID, subscription.Name, subscription.Topic, subscription.QOS, usageMode, subscription.Description, subscription.MessageRetention, nullableJSONString(defaultBatchRuleBytes), actorID, actorID, createdAt, updatedAt); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照 MQTT 订阅失败", err)
		}
	}
	return nil
}

func normalizeSnapshotMqttSubscriptionUsageMode(value string) string {
	switch strings.TrimSpace(value) {
	case "raw_datapoint":
		return "raw_datapoint"
	case "batch_variable":
		return "batch_variable"
	default:
		return "single_variable"
	}
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
                id, project_id, subscription_id, name, code, description, data_type, parse_type, parse_rule,
                default_value, unit, transform, validation, display_order, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::jsonb, $14, $15, $16, $17, $18)
        `, tag.ID, projectID, tag.SubscriptionID, tag.Name, tag.Code, tag.Description, tag.DataType, tag.ParseType, tag.ParseRule, tag.DefaultValue, tag.Unit, tag.Transform, nullableJSONString(validationBytes), tag.Order, actorID, actorID, createdAt, updatedAt); err != nil {
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
                refresh_mode, refresh_interval_ms, status, display_order, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11, $12, $13, $14, $15, $16, $17::jsonb, $18::jsonb, $19, $20, $21, $22, $23, $24, $25, $26)
        `, datapoint.ID, projectID, datapoint.Path, datapoint.Name, datapoint.Description, datapoint.SourceType, datapoint.SourceID, string(sourceConfigBytes), datapoint.DataType, datapoint.Unit, datapoint.PrecisionNum, datapoint.DefaultValue, datapoint.MinValue, datapoint.MaxValue, datapoint.AlarmLow, datapoint.AlarmHigh, string(tagsBytes), string(runtimePermissionsBytes), datapoint.RefreshMode, datapoint.RefreshIntervalMS, datapoint.Status, datapoint.DisplayOrder, actorID, actorID, createdAt, updatedAt); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照数据点失败", err)
		}
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertComputeUnits(ctx context.Context, tx pgx.Tx, projectID, actorID string, units []ComputeUnitRecord) error {
	for _, unit := range units {
		triggerConfigBytes, err := marshalSnapshotObject(unit.TriggerConfig)
		if err != nil {
			return err
		}
		inputBindingsBytes, err := marshalSnapshotObject(unit.InputBindings)
		if err != nil {
			return err
		}
		outputBindingsBytes, err := marshalSnapshotObject(unit.OutputBinding)
		if err != nil {
			return err
		}
		dependenciesBytes, err := json.Marshal(unit.Dependencies)
		if err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照计算单元依赖格式无效", err)
		}
		createdAt := coalesceTime(unit.CreatedAt)
		updatedAt := coalesceTime(unit.UpdatedAt)
		if _, err := tx.Exec(ctx, `
            INSERT INTO data_compute_units (
                id, project_id, name, description, folder_id, language, script_code, trigger_type, trigger_config,
                input_bindings, output_bindings, dependencies, timeout_ms, is_enabled, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, NULL, $5, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb, $11::jsonb, $12, $13, $14, $15, $16, $17)
        `, unit.ID, projectID, unit.Name, unit.Description, unit.Language, unit.ScriptCode, unit.TriggerType, string(triggerConfigBytes),
			string(inputBindingsBytes), string(outputBindingsBytes), string(dependenciesBytes), unit.TimeoutMS, unit.IsEnabled, actorID, actorID, createdAt, updatedAt); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照计算单元失败", err)
		}
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertAlarmPolicyGroups(ctx context.Context, tx pgx.Tx, projectID, actorID string, groups []AlarmPolicyGroupRecord) error {
	pending := append([]AlarmPolicyGroupRecord{}, groups...)
	inserted := make(map[string]struct{}, len(groups))
	for len(pending) > 0 {
		remaining := make([]AlarmPolicyGroupRecord, 0, len(pending))
		progress := false
		for _, group := range pending {
			if group.ParentID != nil {
				if _, ok := inserted[*group.ParentID]; !ok {
					remaining = append(remaining, group)
					continue
				}
			}
			createdAt, updatedAt := coalesceTime(group.CreatedAt), coalesceTime(group.UpdatedAt)
			if _, err := tx.Exec(ctx, `INSERT INTO data_alarm_policy_groups
                (id,project_id,parent_id,name,description,sort_order,created_by,updated_by,created_at,updated_at)
                VALUES($1,$2,$3,$4,$5,$6,$7,$7,$8,$9)`,
				group.ID, projectID, group.ParentID, group.Name, group.Description, group.SortOrder, actorID, createdAt, updatedAt); err != nil {
				return translateAlarmSnapshotError("写入快照报警目录失败", err)
			}
			inserted[group.ID] = struct{}{}
			progress = true
		}
		if !progress {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照报警目录存在缺失父目录或循环引用")
		}
		pending = remaining
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertAlarmChannels(ctx context.Context, tx pgx.Tx, projectID, actorID string, channels []AlarmNotificationChannelRecord, secrets []SnapshotAlarmChannelSecretRecord) error {
	channelIDs := make(map[string]struct{}, len(channels))
	for _, channel := range channels {
		configBytes, err := marshalSnapshotObject(channel.Config)
		if err != nil {
			return err
		}
		statusBytes, err := marshalSnapshotObject(channel.SecretStatus)
		if err != nil {
			return err
		}
		createdAt, updatedAt := coalesceTime(channel.CreatedAt), coalesceTime(channel.UpdatedAt)
		if _, err = tx.Exec(ctx, `INSERT INTO data_alarm_notification_channels
            (id,project_id,name,channel_type,config,secret_status,is_enabled,created_by,updated_by,created_at,updated_at)
            VALUES($1,$2,$3,$4,$5::jsonb,$6::jsonb,$7,$8,$8,$9,$10)`,
			channel.ID, projectID, channel.Name, channel.ChannelType, string(configBytes), string(statusBytes), channel.IsEnabled, actorID, createdAt, updatedAt); err != nil {
			return translateAlarmSnapshotError("写入快照报警通知渠道失败", err)
		}
		channelIDs[channel.ID] = struct{}{}
	}
	for _, secret := range secrets {
		if _, ok := channelIDs[secret.ChannelID]; !ok || secret.SecretKey == "" || len(secret.EncryptedValue) == 0 || secret.EncryptionKeyVersion == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照报警渠道密钥引用无效")
		}
		createdAt, updatedAt := coalesceTime(secret.CreatedAt), coalesceTime(secret.UpdatedAt)
		if _, err := tx.Exec(ctx, `INSERT INTO data_alarm_notification_channel_secrets
            (channel_id,secret_key,encrypted_value,encryption_key_version,created_at,updated_at)
            VALUES($1,$2,$3,$4,$5,$6)`, secret.ChannelID, secret.SecretKey, secret.EncryptedValue, secret.EncryptionKeyVersion, createdAt, updatedAt); err != nil {
			return translateAlarmSnapshotError("写入快照报警渠道密钥失败", err)
		}
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertAlarmSettings(ctx context.Context, tx pgx.Tx, projectID, actorID string, settings *AlarmProjectSettingsRecord) error {
	if settings == nil {
		return nil
	}
	channelIDs, err := json.Marshal(settings.DefaultChannelIDs)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照报警默认渠道格式无效", err)
	}
	createdAt, updatedAt := coalesceTime(settings.CreatedAt), coalesceTime(settings.UpdatedAt)
	_, err = tx.Exec(ctx, `INSERT INTO data_alarm_project_settings
        (project_id,notify_on_raise,notify_on_clear,repeat_interval_seconds,default_message_template,default_channel_ids,created_by,updated_by,created_at,updated_at)
        VALUES($1,$2,$3,$4,$5,$6::jsonb,$7,$7,$8,$9)`, projectID, settings.NotifyOnRaise, settings.NotifyOnClear,
		settings.RepeatIntervalSeconds, settings.DefaultMessageTemplate, string(channelIDs), actorID, createdAt, updatedAt)
	if err != nil {
		return translateAlarmSnapshotError("写入快照报警默认设置失败", err)
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertAlarmPolicies(ctx context.Context, tx pgx.Tx, projectID, actorID string, policies []AlarmPolicyRecord) error {
	for _, policy := range policies {
		channelIDs, contractBytes, err := marshalAlarmJSON(policy.NotificationChannelIDs, policy.Contract)
		if err != nil {
			return err
		}
		revision := policy.Revision
		if revision < 1 {
			revision = 1
		}
		createdAt, updatedAt := coalesceTime(policy.CreatedAt), coalesceTime(policy.UpdatedAt)
		if _, err = tx.Exec(ctx, `INSERT INTO data_alarm_policies
            (id,project_id,group_id,name,description,mode,derived_expression,notification_mode,notify_on_raise,notify_on_clear,
             repeat_interval_seconds,notification_channel_ids,message_template,is_enabled,revision,contract,created_by,updated_by,created_at,updated_at)
            VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13,$14,$15,$16::jsonb,$17,$17,$18,$19)`,
			policy.ID, projectID, policy.GroupID, policy.Name, policy.Description, policy.Mode, policy.DerivedExpression,
			policy.NotificationMode, policy.NotifyOnRaise, policy.NotifyOnClear, policy.RepeatIntervalSeconds, string(channelIDs),
			policy.MessageTemplate, policy.IsEnabled, revision, string(contractBytes), actorID, createdAt, updatedAt); err != nil {
			return translateAlarmSnapshotError("写入快照报警策略失败", err)
		}
		if err = insertSnapshotAlarmPolicyDetails(ctx, tx, policy.ID, policy.Bindings, policy.Conditions); err != nil {
			return err
		}
	}
	return nil
}

func insertSnapshotAlarmPolicyDetails(ctx context.Context, tx pgx.Tx, policyID string, bindings []AlarmPolicyBindingRecord, conditions []AlarmPolicyConditionRecord) error {
	for index, binding := range bindings {
		if binding.ID == "" {
			_, err := tx.Exec(ctx, `INSERT INTO data_alarm_policy_bindings(policy_id,datapoint_id,role,input_key,sort_order) VALUES($1,$2,$3,$4,$5)`, policyID, binding.DatapointID, binding.Role, binding.InputKey, index)
			if err != nil {
				return translateAlarmSnapshotError("写入快照报警点位绑定失败", err)
			}
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO data_alarm_policy_bindings(id,policy_id,datapoint_id,role,input_key,sort_order) VALUES($1,$2,$3,$4,$5,$6)`, binding.ID, policyID, binding.DatapointID, binding.Role, binding.InputKey, index); err != nil {
			return translateAlarmSnapshotError("写入快照报警点位绑定失败", err)
		}
	}
	for index, condition := range conditions {
		params, err := json.Marshal(condition.Params)
		if err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照报警条件参数格式无效", err)
		}
		if condition.ID == "" {
			_, err = tx.Exec(ctx, `INSERT INTO data_alarm_policy_conditions(policy_id,kind,operator,label,severity,params,trigger_delay_ms,clear_delay_ms,deadband,sort_order) VALUES($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9,$10)`, policyID, condition.Kind, condition.Operator, condition.Label, condition.Severity, string(params), condition.TriggerDelayMS, condition.ClearDelayMS, condition.Deadband, index)
		} else {
			_, err = tx.Exec(ctx, `INSERT INTO data_alarm_policy_conditions(id,policy_id,kind,operator,label,severity,params,trigger_delay_ms,clear_delay_ms,deadband,sort_order) VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10,$11)`, condition.ID, policyID, condition.Kind, condition.Operator, condition.Label, condition.Severity, string(params), condition.TriggerDelayMS, condition.ClearDelayMS, condition.Deadband, index)
		}
		if err != nil {
			return translateAlarmSnapshotError("写入快照报警条件失败", err)
		}
	}
	return nil
}

func translateAlarmSnapshotError(message string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "23505" || pgErr.Code == "23514" || pgErr.Code == "22P02") {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照报警配置引用无效、名称重复或字段不合法")
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
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

func intToSnapshotString(value int) string {
	return fmt.Sprintf("%d", value)
}

func deriveConnectionCategory(connectionType string) string {
	switch connectionType {
	case "relational":
		return "database"
	case "mqtt":
		return "message"
	case "kafka", "http", "websocket", "redis", "tdengine":
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

func mustJSONStringArray(payload []byte) []string {
	if len(payload) == 0 {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal(payload, &result); err != nil || result == nil {
		return []string{}
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
