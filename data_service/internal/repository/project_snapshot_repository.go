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
	ConnectionID     string         `json:"connectionId"`
	DBType           string         `json:"dbType"`
	Host             string         `json:"host"`
	Port             int            `json:"port"`
	Database         string         `json:"database"`
	Username         string         `json:"username"`
	Schema           *string        `json:"schema"`
	Charset          *string        `json:"charset"`
	Timezone         *string        `json:"timezone"`
	SSL              bool           `json:"ssl"`
	SSLConfig        map[string]any `json:"sslConfig"`
	PoolMin          int            `json:"poolMin"`
	PoolMax          int            `json:"poolMax"`
	AcquireTimeoutMS int            `json:"acquireTimeoutMs"`
	IdleTimeoutMS    int            `json:"idleTimeoutMs"`
	QueryTimeoutMS   int            `json:"queryTimeoutMs"`
	Options          map[string]any `json:"options"`
}

// SnapshotMqttConfigRecord 表示导入导出使用的 MQTT 配置投影。
type SnapshotMqttConfigRecord struct {
	ConnectionID      string         `json:"connectionId"`
	BrokerURL         string         `json:"brokerUrl"`
	Protocol          string         `json:"protocol"`
	Port              int            `json:"port"`
	ClientID          *string        `json:"clientId"`
	Username          *string        `json:"username"`
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
	GroupID          *string        `json:"groupId"`
	Description      *string        `json:"description"`
	DisplayOrder     int            `json:"displayOrder"`
	MessageRetention int            `json:"messageRetention"`
	DefaultBatchRule map[string]any `json:"defaultBatchParseRule"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

type SnapshotMqttSubscriptionGroupRecord struct {
	ID           string    `json:"id"`
	ConnectionID string    `json:"connectionId"`
	Name         string    `json:"name"`
	ParentID     *string   `json:"parentId"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type SnapshotComputeFolderRecord struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ParentID  *string   `json:"parentId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SnapshotComputeDependencyRecord struct {
	ID          string    `json:"id"`
	Language    string    `json:"language"`
	PackageName string    `json:"packageName"`
	ImportName  string    `json:"importName"`
	Version     string    `json:"version"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type SnapshotKafkaConfigRecord struct {
	ConnectionID  string         `json:"connectionId"`
	Brokers       string         `json:"brokers"`
	Topic         *string        `json:"topic"`
	ConsumerGroup *string        `json:"consumerGroup"`
	StartPosition string         `json:"startPosition"`
	Options       map[string]any `json:"options"`
}

type SnapshotKafkaTopicGroupRecord struct {
	ID           string  `json:"id"`
	ConnectionID string  `json:"connectionId"`
	ParentID     *string `json:"parentId"`
	Name         string  `json:"name"`
	SortOrder    int     `json:"sortOrder"`
}

type SnapshotKafkaTopicMappingRecord struct {
	ID             string  `json:"id"`
	ConnectionID   string  `json:"connectionId"`
	GroupID        *string `json:"groupId"`
	Name           string  `json:"name"`
	Topic          string  `json:"topic"`
	Description    string  `json:"description"`
	ConsumerGroup  string  `json:"consumerGroup"`
	OutputMode     string  `json:"outputMode"`
	RawOutputScope string  `json:"rawOutputScope"`
	PartitionMode  string  `json:"partitionMode"`
	Partition      *int    `json:"partition"`
	StartPosition  string  `json:"startPosition"`
	StartOffset    *int64  `json:"startOffset"`
	Decode         string  `json:"decode"`
	SampleLimit    int     `json:"sampleLimit"`
	TimeoutMS      int     `json:"timeoutMs"`
	SortOrder      int     `json:"sortOrder"`
}

type SnapshotKafkaFieldGroupRecord struct {
	ID             string  `json:"id"`
	ConnectionID   string  `json:"connectionId"`
	TopicMappingID string  `json:"topicMappingId"`
	ParentID       *string `json:"parentId"`
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	SortOrder      int     `json:"sortOrder"`
}

type SnapshotKafkaFieldRecord struct {
	ID             string  `json:"id"`
	ConnectionID   string  `json:"connectionId"`
	TopicMappingID string  `json:"topicMappingId"`
	GroupID        *string `json:"groupId"`
	Name           string  `json:"name"`
	ValuePath      []any   `json:"valuePath"`
	KeyPath        []any   `json:"keyPath"`
	DataType       string  `json:"dataType"`
	Enabled        bool    `json:"enabled"`
	Description    string  `json:"description"`
	SortOrder      int     `json:"sortOrder"`
}

type SnapshotWorkbenchObjectGroupRecord struct {
	ID           string `json:"id"`
	ConnectionID string `json:"connectionId"`
	Scope        string `json:"scope"`
	Name         string `json:"name"`
	SortOrder    int    `json:"sortOrder"`
}

type SnapshotTableGroupMemberRecord struct {
	ConnectionID string    `json:"connectionId"`
	TableName    string    `json:"tableName"`
	GroupID      string    `json:"groupId"`
	UpdatedAt    time.Time `json:"updatedAt"`
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

// SnapshotCollectorConnectionRecord 保存工业采集开发态配置，不包含临时任务和诊断结果。
type SnapshotCollectorConnectionRecord struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	Code               string         `json:"code"`
	ProtocolFamily     string         `json:"protocolFamily"`
	DriverID           string         `json:"driverId"`
	DriverVersion      string         `json:"driverVersion"`
	SchemaVersion      int            `json:"schemaVersion"`
	Config             map[string]any `json:"config"`
	Metadata           map[string]any `json:"metadata"`
	IsEnabled          bool           `json:"isEnabled"`
	DefaultAcquisition map[string]any `json:"defaultAcquisition"`
	DisplayOrder       int            `json:"displayOrder"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}

type SnapshotCollectorSecretRecord struct {
	ConnectionID         string    `json:"connectionId"`
	SecretKey            string    `json:"secretKey"`
	EncryptedValue       []byte    `json:"encryptedValue"`
	EncryptionKeyVersion string    `json:"encryptionKeyVersion"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type SnapshotCollectorPointGroupRecord struct {
	ID           string         `json:"id"`
	ConnectionID string         `json:"connectionId"`
	ParentID     *string        `json:"parentId"`
	Name         string         `json:"name"`
	SortOrder    int            `json:"sortOrder"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

type SnapshotCollectorPointRecord struct {
	ID                   string         `json:"id"`
	ConnectionID         string         `json:"connectionId"`
	GroupID              *string        `json:"groupId"`
	Code                 string         `json:"code"`
	Name                 string         `json:"name"`
	Description          *string        `json:"description"`
	Address              map[string]any `json:"address"`
	AddressText          string         `json:"addressText"`
	AddressSchemaVersion int            `json:"addressSchemaVersion"`
	DataType             string         `json:"dataType"`
	ElementCount         int            `json:"elementCount"`
	ReadOptions          map[string]any `json:"readOptions"`
	AcquisitionMode      string         `json:"acquisitionMode"`
	AcquisitionOverrides map[string]any `json:"acquisitionOverrides"`
	Enabled              bool           `json:"enabled"`
	SortOrder            int            `json:"sortOrder"`
	Metadata             map[string]any `json:"metadata"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
}

type SnapshotHTTPRequestGroupRecord struct {
	ID           string  `json:"id"`
	ConnectionID string  `json:"connectionId"`
	ParentID     *string `json:"parentId"`
	Name         string  `json:"name"`
	SortOrder    int     `json:"sortOrder"`
}

type SnapshotHTTPRequestRecord struct {
	ID           string                      `json:"id"`
	ConnectionID string                      `json:"connectionId"`
	Name         string                      `json:"name"`
	Method       string                      `json:"method"`
	URL          string                      `json:"url"`
	BodyType     string                      `json:"bodyType"`
	GroupID      *string                     `json:"groupId"`
	Params       []any                       `json:"params"`
	Headers      []any                       `json:"headers"`
	Auth         map[string]any              `json:"auth"`
	Body         map[string]any              `json:"body"`
	Settings     map[string]any              `json:"settings"`
	Enabled      bool                        `json:"enabled"`
	SortOrder    int                         `json:"sortOrder"`
	Outputs      []SourceOutputMappingRecord `json:"outputs"`
}

type SnapshotWebSocketGroupRecord struct {
	ID           string  `json:"id"`
	ConnectionID string  `json:"connectionId"`
	ParentID     *string `json:"parentId"`
	Name         string  `json:"name"`
	SortOrder    int     `json:"sortOrder"`
}

type SnapshotWebSocketSessionRecord struct {
	ID           string                      `json:"id"`
	ConnectionID string                      `json:"connectionId"`
	Name         string                      `json:"name"`
	URL          string                      `json:"url"`
	GroupID      *string                     `json:"groupId"`
	Headers      []any                       `json:"headers"`
	Protocols    []any                       `json:"protocols"`
	Messages     []any                       `json:"messages"`
	Auth         map[string]any              `json:"auth"`
	Settings     map[string]any              `json:"settings"`
	Enabled      bool                        `json:"enabled"`
	SortOrder    int                         `json:"sortOrder"`
	Outputs      []SourceOutputMappingRecord `json:"outputs"`
}

type SnapshotRealtimeKeyRecord struct {
	ID                string                      `json:"id"`
	ConnectionID      string                      `json:"connectionId"`
	Provider          string                      `json:"provider"`
	KeyPath           string                      `json:"keyPath"`
	RedisType         string                      `json:"redisType"`
	ValueType         string                      `json:"valueType"`
	DefaultTtlSeconds int                         `json:"defaultTtlSeconds"`
	Description       string                      `json:"description"`
	Outputs           []SourceOutputMappingRecord `json:"outputs"`
}

// ProjectSnapshot 表示工程级数据域快照。
type ProjectSnapshot struct {
	Connections            []ConnectionRecord                    `json:"connections"`
	ConnectionSecrets      []SnapshotConnectionSecretRecord      `json:"connectionSecrets"`
	RelationalConfigs      []SnapshotRelationalConfigRecord      `json:"relationalConfigs"`
	KafkaConfigs           []SnapshotKafkaConfigRecord           `json:"kafkaConfigs"`
	KafkaTopicGroups       []SnapshotKafkaTopicGroupRecord       `json:"kafkaTopicGroups"`
	KafkaTopicMappings     []SnapshotKafkaTopicMappingRecord     `json:"kafkaTopicMappings"`
	KafkaFieldGroups       []SnapshotKafkaFieldGroupRecord       `json:"kafkaFieldGroups"`
	KafkaFields            []SnapshotKafkaFieldRecord            `json:"kafkaFields"`
	Queries                []QueryRecord                         `json:"queries"`
	MqttConfigs            []SnapshotMqttConfigRecord            `json:"mqttConfigs"`
	MqttSubscriptions      []SnapshotMqttSubscriptionRecord      `json:"mqttSubscriptions"`
	MqttSubscriptionGroups []SnapshotMqttSubscriptionGroupRecord `json:"mqttSubscriptionGroups"`
	MqttTags               []SnapshotMqttTagRecord               `json:"mqttTags"`
	CollectorConnections   []SnapshotCollectorConnectionRecord   `json:"collectorConnections"`
	CollectorSecrets       []SnapshotCollectorSecretRecord       `json:"collectorSecrets"`
	CollectorPointGroups   []SnapshotCollectorPointGroupRecord   `json:"collectorPointGroups"`
	CollectorPoints        []SnapshotCollectorPointRecord        `json:"collectorPoints"`
	HTTPRequestGroups      []SnapshotHTTPRequestGroupRecord      `json:"httpRequestGroups"`
	HTTPRequests           []SnapshotHTTPRequestRecord           `json:"httpRequests"`
	WebSocketGroups        []SnapshotWebSocketGroupRecord        `json:"webSocketGroups"`
	WebSocketSessions      []SnapshotWebSocketSessionRecord      `json:"webSocketSessions"`
	RealtimeKeys           []SnapshotRealtimeKeyRecord           `json:"realtimeKeys"`
	DataPoints             []DataPointRecord                     `json:"datapoints"`
	ComputeUnits           []ComputeUnitRecord                   `json:"computeUnits"`
	ComputeFolders         []SnapshotComputeFolderRecord         `json:"computeFolders"`
	ComputeDependencies    []SnapshotComputeDependencyRecord     `json:"computeDependencies"`
	WorkbenchObjectGroups  []SnapshotWorkbenchObjectGroupRecord  `json:"workbenchObjectGroups"`
	TableGroupMembers      []SnapshotTableGroupMemberRecord      `json:"tableGroupMembers"`
	AlarmGroups            []AlarmGroupRecord                    `json:"alarmGroups"`
	AlarmItems             []AlarmItemRecord                     `json:"alarmItems"`
	AlarmSettings          *AlarmProjectSettingsRecord           `json:"alarmSettings"`
	AlarmHistorySettings   *AlarmHistorySettingsRecord           `json:"alarmHistorySettings"`
	AlarmChannels          []AlarmNotificationChannelRecord      `json:"alarmChannels"`
	AlarmChannelSecrets    []SnapshotAlarmChannelSecretRecord    `json:"alarmChannelSecrets"`
	HistoryStorage         []HistoryStorageConfigRecord          `json:"historyStorage"`
}

// SnapshotConnectionSecretRecord 只往返接入源密文和密钥版本，禁止出现明文。
type SnapshotConnectionSecretRecord struct {
	ConnectionID         string    `json:"connectionId"`
	SecretKey            string    `json:"secretKey"`
	EncryptedValue       []byte    `json:"encryptedValue"`
	EncryptionKeyVersion string    `json:"encryptionKeyVersion"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
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

// ProjectArtifactVersionV1 表示 开发态协议 产物契约版本号。
const ProjectArtifactVersionV1 = "1.0"

// ArtifactConnectionRecord 表示产物层通用连接对象。
type ArtifactConnectionRecord struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	Enabled bool           `json:"enabled"`
	Config  map[string]any `json:"config"`
}

// ArtifactQueryRecord 表示产物层查询对象。
type ArtifactQueryRecord struct {
	ID              string                      `json:"id"`
	ConnectionID    string                      `json:"connectionId"`
	Name            string                      `json:"name"`
	QueryType       string                      `json:"queryType"`
	Config          map[string]any              `json:"config"`
	TimeoutMS       int                         `json:"timeoutMs"`
	IsEnabled       bool                        `json:"isEnabled"`
	Transformer     *string                     `json:"transformer,omitempty"`
	CacheEnabled    bool                        `json:"cacheEnabled"`
	CacheTtlSeconds int                         `json:"cacheTtlSeconds"`
	Outputs         []SourceOutputMappingRecord `json:"outputs"`
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
	AttributeDefaults  map[string]string           `json:"attributeDefaults"`
}

// ArtifactComputeUnitRecord 表示产物层计算单元对象。
type ArtifactComputeUnitRecord struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	Description   *string               `json:"description,omitempty"`
	Language      string                `json:"language"`
	ScriptCode    string                `json:"scriptCode"`
	TriggerType   string                `json:"triggerType"`
	TriggerConfig map[string]any        `json:"triggerConfig"`
	InputBindings map[string]any        `json:"inputBindings"`
	Outputs       []ComputeOutputRecord `json:"outputs"`
	Dependencies  []any                 `json:"dependencies"`
	TimeoutMS     int                   `json:"timeoutMs"`
	IsEnabled     bool                  `json:"isEnabled"`
}

type ArtifactAlarmSecretRef struct {
	ChannelID  string `json:"channelId"`
	SecretKey  string `json:"secretKey"`
	KeyVersion string `json:"keyVersion"`
}

type ArtifactConnectionSecretRef struct {
	ConnectionID string `json:"connectionId"`
	SecretKey    string `json:"secretKey"`
	KeyVersion   string `json:"keyVersion"`
}

type ArtifactCollectorSecretRef struct {
	ConnectionID string `json:"connectionId"`
	SecretKey    string `json:"secretKey"`
	KeyVersion   string `json:"keyVersion"`
}

type ArtifactCollectorPayload struct {
	Connections []SnapshotCollectorConnectionRecord `json:"connections"`
	PointGroups []SnapshotCollectorPointGroupRecord `json:"pointGroups"`
	Points      []SnapshotCollectorPointRecord      `json:"points"`
	SecretRefs  []ArtifactCollectorSecretRef        `json:"secretRefs"`
}

type ArtifactAlarmHistoryStorage struct {
	Enabled                     bool `json:"enabled"`
	RetentionDays               *int `json:"retentionDays"`
	StoreNotificationDeliveries bool `json:"storeNotificationDeliveries"`
}

type ArtifactAlarmNotification struct {
	Mode                  string   `json:"mode"`
	NotifyOnRaise         *bool    `json:"notifyOnRaise"`
	NotifyOnClear         *bool    `json:"notifyOnClear"`
	RepeatIntervalSeconds *int     `json:"repeatIntervalSeconds"`
	ChannelIDs            []string `json:"channelIds"`
	MessageTemplate       string   `json:"messageTemplate"`
}

type ArtifactAlarmItem struct {
	AlarmItemID       string                     `json:"alarmItemId"`
	DatapointID       *string                    `json:"datapointId,omitempty"`
	Path              *string                    `json:"path,omitempty"`
	GroupID           *string                    `json:"groupId"`
	DisplayName       string                     `json:"displayName"`
	Description       *string                    `json:"description"`
	Mode              string                     `json:"mode"`
	AlarmType         string                     `json:"alarmType"`
	EvaluationMode    string                     `json:"evaluationMode"`
	Inputs            []AlarmItemInputRecord     `json:"inputs"`
	DerivedExpression string                     `json:"derivedExpression"`
	Conditions        []AlarmItemConditionRecord `json:"conditions"`
	Notification      ArtifactAlarmNotification  `json:"notification"`
	IsEnabled         bool                       `json:"isEnabled"`
	Revision          int64                      `json:"revision"`
}

// ArtifactAlarmsPayload 是节点消费的 alarm.item.v1 完整开发态契约。
type ArtifactAlarmsPayload struct {
	SchemaVersion  string                           `json:"schemaVersion"`
	HistoryStorage ArtifactAlarmHistoryStorage      `json:"historyStorage"`
	Groups         []AlarmGroupRecord               `json:"groups"`
	Items          []ArtifactAlarmItem              `json:"items"`
	Settings       *AlarmProjectSettingsRecord      `json:"settings"`
	Channels       []AlarmNotificationChannelRecord `json:"channels"`
	SecretRefs     []ArtifactAlarmSecretRef         `json:"secretRefs"`
}

// ArtifactMqttConnectionRecord 表示产物层 MQTT 连接对象。
type ArtifactMqttConnectionRecord struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Type              string         `json:"type"`
	Enabled           bool           `json:"enabled"`
	BrokerURL         string         `json:"brokerUrl"`
	Protocol          string         `json:"protocol"`
	Port              int            `json:"port"`
	ClientID          *string        `json:"clientId,omitempty"`
	Username          *string        `json:"username,omitempty"`
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
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	Enabled bool           `json:"enabled"`
	Config  map[string]any `json:"config"`
}

// ArtifactMqttPayload 表示产物层 MQTT 区块。
type ArtifactMqttPayload struct {
	Connections   []ArtifactMqttConnectionRecord   `json:"connections"`
	Subscriptions []SnapshotMqttSubscriptionRecord `json:"subscriptions"`
	Tags          []SnapshotMqttTagRecord          `json:"tags"`
}

// ArtifactProtocolsPayload 表示产物层其他协议区块。
type ArtifactProtocolsPayload struct {
	Kafka             []ArtifactProtocolRecord         `json:"kafka"`
	HTTP              []ArtifactProtocolRecord         `json:"http"`
	HTTPRequests      []SnapshotHTTPRequestRecord      `json:"httpRequests"`
	Websocket         []ArtifactProtocolRecord         `json:"websocket"`
	WebsocketSessions []SnapshotWebSocketSessionRecord `json:"websocketSessions"`
	Redis             []ArtifactProtocolRecord         `json:"redis"`
	RealtimeKeys      []SnapshotRealtimeKeyRecord      `json:"realtimeKeys"`
	TDengine          []ArtifactProtocolRecord         `json:"tdengine"`
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

// ProjectArtifactV1 表示 开发态协议 项目级数据产物。
type ProjectArtifactV1 struct {
	Version        string                        `json:"version"`
	ProjectID      string                        `json:"projectId"`
	GeneratedAt    time.Time                     `json:"generatedAt"`
	Connections    []ArtifactConnectionRecord    `json:"connections"`
	Queries        []ArtifactQueryRecord         `json:"queries"`
	DataPoints     []ArtifactDataPointRecord     `json:"datapoints"`
	Compute        []ArtifactComputeUnitRecord   `json:"compute"`
	HistoryStorage []HistoryStorageConfigRecord  `json:"historyStorage"`
	Alarms         ArtifactAlarmsPayload         `json:"alarms"`
	Mqtt           ArtifactMqttPayload           `json:"mqtt"`
	Protocols      ArtifactProtocolsPayload      `json:"protocols"`
	BuiltinStores  ArtifactBuiltinStoresPayload  `json:"builtinStores"`
	Collectors     ArtifactCollectorPayload      `json:"collectors"`
	SecretRefs     []ArtifactConnectionSecretRef `json:"secretRefs"`
}

// ProjectSnapshotRepository 负责项目级数据域快照读写。
type ProjectSnapshotRepository struct {
	pool projectSnapshotTransactionStarter
}

type projectSnapshotTransactionStarter interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type projectSnapshotQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// NewProjectSnapshotRepository 创建快照仓储。
func NewProjectSnapshotRepository(pool *pgxpool.Pool) *ProjectSnapshotRepository {
	return &ProjectSnapshotRepository{pool: pool}
}

// GetByProject 读取项目级完整快照。
func (r *ProjectSnapshotRepository) GetByProject(ctx context.Context, projectID, tenantID string) (*ProjectSnapshot, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启快照读取事务失败", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	snapshot, err := r.getByProject(ctx, tx, projectID, tenantID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交快照读取事务失败", err)
	}
	return snapshot, nil
}

func (r *ProjectSnapshotRepository) getByProject(ctx context.Context, queryer projectSnapshotQuerier, projectID, tenantID string) (*ProjectSnapshot, error) {
	if err := r.requireProjectTenantBinding(ctx, queryer, projectID, tenantID); err != nil {
		return nil, err
	}
	connections, err := r.listConnections(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	for index := range connections {
		connections[index].Config = scrubArtifactConnectionSecrets(connections[index].Config)
	}
	connectionSecrets, err := r.listConnectionSecrets(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	queries, err := r.listQueries(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	datapoints, err := r.listDataPoints(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	mqttConfigs, err := r.listMqttConfigs(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	mqttSubscriptions, err := r.listMqttSubscriptions(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	mqttTags, err := r.listMqttTags(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	collectorConnections, collectorSecrets, collectorGroups, collectorPoints, err := r.listCollectorSnapshot(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	httpGroups, httpRequests, websocketGroups, websocketSessions, realtimeKeys, err := r.listWorkbenchSnapshot(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	computeUnits, err := r.listComputeUnits(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	alarmGroups, err := listSnapshotAlarmGroups(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	alarmItems, err := listSnapshotAlarmItems(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	alarmSettings, err := getSnapshotAlarmProjectSettings(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	alarmHistorySettings, err := getSnapshotAlarmHistorySettings(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	alarmChannels, err := listSnapshotAlarmChannels(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	alarmChannelSecrets, err := r.listAlarmChannelSecrets(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	historyStorage, err := r.listHistoryStorageConfigs(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}
	additions, err := r.listCompleteAuthoringSnapshot(ctx, queryer, projectID)
	if err != nil {
		return nil, err
	}

	snapshot := &ProjectSnapshot{
		Connections:          connections,
		ConnectionSecrets:    connectionSecrets,
		RelationalConfigs:    deriveRelationalConfigs(connections),
		Queries:              queries,
		MqttConfigs:          mqttConfigs,
		MqttSubscriptions:    mqttSubscriptions,
		MqttTags:             mqttTags,
		CollectorConnections: collectorConnections,
		CollectorSecrets:     collectorSecrets,
		CollectorPointGroups: collectorGroups,
		CollectorPoints:      collectorPoints,
		HTTPRequestGroups:    httpGroups,
		HTTPRequests:         httpRequests,
		WebSocketGroups:      websocketGroups,
		WebSocketSessions:    websocketSessions,
		RealtimeKeys:         realtimeKeys,
		DataPoints:           datapoints,
		ComputeUnits:         computeUnits,
		AlarmGroups:          alarmGroups,
		AlarmItems:           alarmItems,
		AlarmSettings:        alarmSettings,
		AlarmHistorySettings: alarmHistorySettings,
		AlarmChannels:        alarmChannels,
		AlarmChannelSecrets:  alarmChannelSecrets,
		HistoryStorage:       historyStorage,
	}
	additions.apply(snapshot)
	explicitRelational := make(map[string]bool, len(snapshot.RelationalConfigs))
	for _, item := range snapshot.RelationalConfigs {
		explicitRelational[item.ConnectionID] = true
	}
	for _, item := range deriveRelationalConfigs(connections) {
		if !explicitRelational[item.ConnectionID] {
			snapshot.RelationalConfigs = append(snapshot.RelationalConfigs, item)
		}
	}
	return snapshot, nil
}

// requireProjectTenantBinding 是所有正式快照与工件读取的租户边界；未绑定和跨租户均统一为不存在。
func (r *ProjectSnapshotRepository) requireProjectTenantBinding(ctx context.Context, queryer projectSnapshotQuerier, projectID, tenantID string) error {
	var exists bool
	err := queryer.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM data_project_tenant_bindings
			WHERE project_id = $1 AND tenant_id = $2
		)
	`, projectID, tenantID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("校验项目租户归属失败: %w", err)
	}
	if !exists {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "项目快照不存在")
	}
	return nil
}

func listSnapshotAlarmGroups(ctx context.Context, queryer projectSnapshotQuerier, projectID string) ([]AlarmGroupRecord, error) {
	rows, err := queryer.Query(ctx, `WITH RECURSIVE paths AS (
        SELECT id,project_id,name,parent_id,description,sort_order,name::text AS full_path,created_at,updated_at FROM data_alarm_groups WHERE project_id=$1 AND parent_id IS NULL
        UNION ALL SELECT c.id,c.project_id,c.name,c.parent_id,c.description,c.sort_order,(p.full_path || ' / ' || c.name),c.created_at,c.updated_at FROM data_alarm_groups c JOIN paths p ON p.id=c.parent_id WHERE c.project_id=$1
    ) SELECT id,project_id,name,parent_id,description,sort_order,full_path,EXISTS(SELECT 1 FROM data_alarm_groups c WHERE c.parent_id=paths.id),created_at,updated_at FROM paths ORDER BY full_path`, projectID)
	if err != nil {
		return nil, wrapAlarmRepo("查询报警目录失败", err)
	}
	defer rows.Close()
	items := make([]AlarmGroupRecord, 0)
	for rows.Next() {
		var item AlarmGroupRecord
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Name, &item.ParentID, &item.Description, &item.SortOrder, &item.FullPath, &item.HasChildren, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, wrapAlarmRepo("读取报警目录失败", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func listSnapshotAlarmItems(ctx context.Context, queryer projectSnapshotQuerier, projectID string) ([]AlarmItemRecord, error) {
	rows, err := queryer.Query(ctx, alarmItemBaseSelect()+` WHERE ai.project_id=$1 ORDER BY ai.created_at`, projectID)
	if err != nil {
		return nil, wrapAlarmRepo("查询全部报警项失败", err)
	}
	defer rows.Close()
	items, err := scanAlarmItemRows(rows)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index].Inputs, err = listSnapshotAlarmItemInputs(ctx, queryer, items[index].ID)
		if err != nil {
			return nil, err
		}
		items[index].Conditions, err = listSnapshotAlarmItemConditions(ctx, queryer, items[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return items, nil
}

func listSnapshotAlarmItemInputs(ctx context.Context, queryer projectSnapshotQuerier, id string) ([]AlarmItemInputRecord, error) {
	rows, err := queryer.Query(ctx, `SELECT i.id,i.datapoint_id,p.path,p.name,p.data_type,i.input_key,i.sort_order FROM data_alarm_item_inputs i JOIN data_points p ON p.id=i.datapoint_id WHERE i.alarm_item_id=$1 ORDER BY i.sort_order,i.created_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AlarmItemInputRecord, 0)
	for rows.Next() {
		var item AlarmItemInputRecord
		if err := rows.Scan(&item.ID, &item.DatapointID, &item.Path, &item.Name, &item.DataType, &item.InputKey, &item.SortOrder); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func listSnapshotAlarmItemConditions(ctx context.Context, queryer projectSnapshotQuerier, id string) ([]AlarmItemConditionRecord, error) {
	rows, err := queryer.Query(ctx, `SELECT id,kind,operator,label,severity,params,trigger_delay_ms,clear_delay_ms,deadband,sort_order FROM data_alarm_item_conditions WHERE alarm_item_id=$1 ORDER BY sort_order,created_at`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AlarmItemConditionRecord, 0)
	for rows.Next() {
		var item AlarmItemConditionRecord
		var raw []byte
		if err := rows.Scan(&item.ID, &item.Kind, &item.Operator, &item.Label, &item.Severity, &raw, &item.TriggerDelayMS, &item.ClearDelayMS, &item.Deadband, &item.SortOrder); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &item.Params); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func getSnapshotAlarmProjectSettings(ctx context.Context, queryer projectSnapshotQuerier, projectID string) (*AlarmProjectSettingsRecord, error) {
	item, err := scanAlarmProjectSettings(queryer.QueryRow(ctx, `SELECT project_id,notify_on_raise,notify_on_clear,repeat_interval_seconds,default_message_template,default_channel_ids,severity_definitions,escalation_rules,created_at,updated_at FROM data_alarm_project_settings WHERE project_id=$1`, projectID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func getSnapshotAlarmHistorySettings(ctx context.Context, queryer projectSnapshotQuerier, projectID string) (*AlarmHistorySettingsRecord, error) {
	item, err := scanAlarmHistorySettings(queryer.QueryRow(ctx, `SELECT project_id,is_enabled,retention_days,store_notification_deliveries,created_at,updated_at FROM data_alarm_history_settings WHERE project_id=$1`, projectID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func listSnapshotAlarmChannels(ctx context.Context, queryer projectSnapshotQuerier, projectID string) ([]AlarmNotificationChannelRecord, error) {
	rows, err := queryer.Query(ctx, `SELECT id,project_id,name,channel_type,config,secret_status,is_enabled,last_test_status,last_tested_at,last_test_duration_ms,last_test_message,created_at,updated_at FROM data_alarm_notification_channels WHERE project_id=$1 ORDER BY updated_at DESC`, projectID)
	if err != nil {
		return nil, wrapAlarmRepo("查询通知渠道失败", err)
	}
	defer rows.Close()
	items := make([]AlarmNotificationChannelRecord, 0)
	for rows.Next() {
		item, scanErr := scanAlarmChannel(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ReplaceProjectData 用快照内容覆盖项目下的数据域数据。
func (r *ProjectSnapshotRepository) ReplaceProjectData(ctx context.Context, projectID, actorID string, snapshot ProjectSnapshot) error {
	for _, connection := range snapshot.Connections {
		if connectionConfigContainsPlaintextSecret(connection.Config, "") {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "工程快照连接配置包含明文密钥，请使用 connectionSecrets 密文区块")
		}
	}
	if err := validateSnapshotComputeOutputTargets(snapshot); err != nil {
		return err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启快照写入事务失败", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	if err := r.replaceProjectDataTx(ctx, tx, projectID, actorID, snapshot); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交快照写入事务失败", err)
	}
	return nil
}

// replaceProjectDataTx 允许恢复路径在同一数据库会话的独占 advisory lock 下执行完整覆盖事务。
func (r *ProjectSnapshotRepository) replaceProjectDataTx(ctx context.Context, tx pgx.Tx, projectID, actorID string, snapshot ProjectSnapshot) error {
	if err := lockComputeProject(ctx, tx, projectID); err != nil {
		return err
	}
	// 快照 Replace 不能信任调用方携带的 revision：以同一事务中已锁定的可执行投影为事实来源。
	if err := r.assignSnapshotComputeRevisions(ctx, tx, projectID, &snapshot); err != nil {
		return err
	}

	if err := r.deleteProjectSnapshot(ctx, tx, projectID); err != nil {
		return err
	}
	if err := r.insertConnections(ctx, tx, projectID, actorID, snapshot.Connections); err != nil {
		return err
	}
	if err := r.insertProtocolConnectionConfigs(ctx, tx, snapshot.Connections); err != nil {
		return err
	}
	if err := r.insertExplicitRelationalAndKafkaConfigs(ctx, tx, snapshot); err != nil {
		return err
	}
	if err := r.insertConnectionSecrets(ctx, tx, snapshot.ConnectionSecrets); err != nil {
		return err
	}
	if err := r.insertMqttConfigs(ctx, tx, snapshot.MqttConfigs); err != nil {
		return err
	}
	if err := r.insertCollectorSnapshot(ctx, tx, projectID, actorID, snapshot); err != nil {
		return err
	}
	if err := r.insertCompleteAuthoringGroups(ctx, tx, projectID, actorID, snapshot); err != nil {
		return err
	}
	if err := r.insertQueries(ctx, tx, projectID, actorID, snapshot.Queries); err != nil {
		return err
	}
	if err := r.insertWorkbenchOwners(ctx, tx, projectID, actorID, snapshot); err != nil {
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
	if err := r.insertQueryOutputMappings(ctx, tx, projectID, snapshot.Queries); err != nil {
		return err
	}
	if err := r.insertWorkbenchOutputMappings(ctx, tx, projectID, snapshot); err != nil {
		return err
	}
	if err := r.insertComputeUnits(ctx, tx, projectID, actorID, snapshot.ComputeUnits); err != nil {
		return err
	}
	if err := r.insertHistoryStorageConfigs(ctx, tx, projectID, actorID, snapshot.HistoryStorage); err != nil {
		return err
	}
	if err := r.insertAlarmGroups(ctx, tx, projectID, actorID, snapshot.AlarmGroups); err != nil {
		return err
	}
	if err := r.insertAlarmChannels(ctx, tx, projectID, actorID, snapshot.AlarmChannels, snapshot.AlarmChannelSecrets); err != nil {
		return err
	}
	if err := r.insertAlarmSettings(ctx, tx, projectID, actorID, snapshot.AlarmSettings); err != nil {
		return err
	}
	if err := r.insertAlarmHistorySettings(ctx, tx, projectID, actorID, snapshot.AlarmHistorySettings); err != nil {
		return err
	}
	if err := r.insertAlarmItems(ctx, tx, projectID, actorID, snapshot.AlarmItems); err != nil {
		return err
	}

	return nil
}

func (r *ProjectSnapshotRepository) assignSnapshotComputeRevisions(ctx context.Context, tx pgx.Tx, projectID string, snapshot *ProjectSnapshot) error {
	for index := range snapshot.ComputeUnits {
		unit := &snapshot.ComputeUnits[index]
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM data_compute_units WHERE project_id=$1 AND id=$2)`, projectID, unit.ID).Scan(&exists); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照计算单元版本失败", err)
		}
		if !exists {
			unit.Revision = 1
			continue
		}
		refs, err := snapshotComputeDatapointRefs(*unit)
		if err != nil {
			return err
		}
		outputs := make([]ComputeOutputParam, 0, len(unit.Outputs))
		for _, output := range unit.Outputs {
			outputs = append(outputs, ComputeOutputParam{ID: &output.ID, OutputKey: output.OutputKey, Name: output.Name, Path: output.Path, DataType: output.DataType, Unit: output.Unit, PrecisionNum: output.PrecisionNum, NullPolicy: output.NullPolicy, Description: output.Description, DefaultValue: output.DefaultValue})
		}
		trigger, err := marshalComputeObject(unit.TriggerConfig)
		if err != nil {
			return err
		}
		inputs, err := marshalComputeObject(unit.InputBindings)
		if err != nil {
			return err
		}
		dependencies, err := marshalComputeArray(unit.Dependencies)
		if err != nil {
			return err
		}
		changed, err := computeExecutableDefinitionChangedTx(ctx, tx, UpdateComputeUnitParams{ID: unit.ID, ProjectID: projectID, Language: unit.Language, ScriptCode: unit.ScriptCode, TriggerType: unit.TriggerType, TriggerConfig: unit.TriggerConfig, InputBindings: unit.InputBindings, Dependencies: unit.Dependencies, TimeoutMS: unit.TimeoutMS, IsEnabled: unit.IsEnabled, DatapointRefs: refs, Outputs: outputs}, []byte(trigger), []byte(inputs), []byte(dependencies))
		if err != nil {
			return err
		}
		if !changed {
			changed, err = snapshotComputeOutputTargetsChangedTx(ctx, tx, projectID, *unit)
			if err != nil {
				return err
			}
		}
		var current int64
		if err := tx.QueryRow(ctx, `SELECT revision FROM data_compute_units WHERE project_id=$1 AND id=$2`, projectID, unit.ID).Scan(&current); err != nil {
			return err
		}
		unit.Revision = current
		if changed {
			unit.Revision++
		}
	}
	return nil
}

// validateSnapshotComputeOutputTargets 在删除旧数据前校验每个 compute output 的完整可发布投影。
// 它和运行制品构建共用同一纯 helper，避免快照可写入但节点无法装载的定义漂移。
func validateSnapshotComputeOutputTargets(snapshot ProjectSnapshot) error {
	points := make(map[string]DataPointRecord, len(snapshot.DataPoints))
	for _, point := range snapshot.DataPoints {
		points[point.ID] = point
	}
	for _, unit := range snapshot.ComputeUnits {
		for _, output := range unit.Outputs {
			point, exists := points[output.DatapointID]
			if !exists {
				return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照计算输出目标数据点不存在")
			}
			if _, _, err := validateComputeOutputTarget(unit.ID, output, point); err != nil {
				return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照计算输出投影无效", err)
			}
		}
	}
	return nil
}

func snapshotComputeOutputTargetsChangedTx(ctx context.Context, tx pgx.Tx, projectID string, unit ComputeUnitRecord) (bool, error) {
	rows, err := tx.Query(ctx, `SELECT output_key,datapoint_id::text FROM data_compute_unit_outputs WHERE project_id=$1 AND compute_unit_id=$2`, projectID, unit.ID)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	current := map[string]string{}
	for rows.Next() {
		var key, datapointID string
		if err := rows.Scan(&key, &datapointID); err != nil {
			return false, err
		}
		current[key] = datapointID
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return snapshotComputeOutputTargetsChanged(current, unit.Outputs), nil
}

func snapshotComputeOutputTargetsChanged(current map[string]string, outputs []ComputeOutputRecord) bool {
	if len(current) != len(outputs) {
		return true
	}
	for _, output := range outputs {
		if current[output.OutputKey] != output.DatapointID {
			return true
		}
	}
	return false
}

// BuildProjectArtifactV1 基于项目快照构建 开发态协议 产物契约对象。
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
			ID:      connection.ID,
			Name:    connection.Name,
			Type:    connection.Type,
			Enabled: connection.IsEnabled,
			Config:  buildArtifactConnectionConfig(connection),
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
			Outputs:         append([]SourceOutputMappingRecord(nil), query.Outputs...),
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
			AttributeDefaults:  cloneSnapshotStringMap(dataPoint.AttributeDefaults),
		})
	}

	computeUnits := make([]ArtifactComputeUnitRecord, 0, len(snapshot.ComputeUnits))
	for _, unit := range snapshot.ComputeUnits {
		computeUnits = append(computeUnits, ArtifactComputeUnitRecord{
			ID:            unit.ID,
			Name:          unit.Name,
			Description:   unit.Description,
			Language:      unit.Language,
			ScriptCode:    unit.ScriptCode,
			TriggerType:   unit.TriggerType,
			TriggerConfig: cloneSnapshotObject(unit.TriggerConfig),
			InputBindings: cloneSnapshotObject(unit.InputBindings),
			Outputs:       append([]ComputeOutputRecord(nil), unit.Outputs...),
			Dependencies:  cloneSnapshotAnyArray(unit.Dependencies),
			TimeoutMS:     unit.TimeoutMS,
			IsEnabled:     unit.IsEnabled,
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
	connectionSecretRefs := make([]ArtifactConnectionSecretRef, 0, len(snapshot.ConnectionSecrets))
	for _, secret := range snapshot.ConnectionSecrets {
		connectionSecretRefs = append(connectionSecretRefs, ArtifactConnectionSecretRef{
			ConnectionID: secret.ConnectionID,
			SecretKey:    secret.SecretKey,
			KeyVersion:   secret.EncryptionKeyVersion,
		})
	}
	collectorSecretRefs := make([]ArtifactCollectorSecretRef, 0, len(snapshot.CollectorSecrets))
	for _, secret := range snapshot.CollectorSecrets {
		collectorSecretRefs = append(collectorSecretRefs, ArtifactCollectorSecretRef{
			ConnectionID: secret.ConnectionID,
			SecretKey:    secret.SecretKey,
			KeyVersion:   secret.EncryptionKeyVersion,
		})
	}

	alarmHistoryStorage := ArtifactAlarmHistoryStorage{
		Enabled:                     true,
		RetentionDays:               alarmHistoryDefaultRetentionDays(),
		StoreNotificationDeliveries: true,
	}
	if snapshot.AlarmHistorySettings != nil {
		alarmHistoryStorage.Enabled = snapshot.AlarmHistorySettings.IsEnabled
		alarmHistoryStorage.RetentionDays = snapshot.AlarmHistorySettings.RetentionDays
		alarmHistoryStorage.StoreNotificationDeliveries = snapshot.AlarmHistorySettings.StoreNotificationDeliveries
	}

	mqttConnections := make([]ArtifactMqttConnectionRecord, 0, len(snapshot.MqttConfigs))
	for _, mqttConfig := range snapshot.MqttConfigs {
		connection := connectionIndex[mqttConfig.ConnectionID]
		mqttConnections = append(mqttConnections, ArtifactMqttConnectionRecord{
			ID:                mqttConfig.ConnectionID,
			Name:              connection.Name,
			Type:              coalesceProtocolType(connection.Type, "mqtt"),
			Enabled:           connection.IsEnabled,
			BrokerURL:         mqttConfig.BrokerURL,
			Protocol:          mqttConfig.Protocol,
			Port:              mqttConfig.Port,
			ClientID:          mqttConfig.ClientID,
			Username:          mqttConfig.Username,
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
		Kafka:             make([]ArtifactProtocolRecord, 0),
		HTTP:              make([]ArtifactProtocolRecord, 0),
		HTTPRequests:      append([]SnapshotHTTPRequestRecord(nil), snapshot.HTTPRequests...),
		Websocket:         make([]ArtifactProtocolRecord, 0),
		WebsocketSessions: append([]SnapshotWebSocketSessionRecord(nil), snapshot.WebSocketSessions...),
		Redis:             make([]ArtifactProtocolRecord, 0),
		RealtimeKeys:      append([]SnapshotRealtimeKeyRecord(nil), snapshot.RealtimeKeys...),
		TDengine:          make([]ArtifactProtocolRecord, 0),
	}
	builtinStores := buildBuiltinStores(snapshot.Connections)
	for _, connection := range snapshot.Connections {
		protocolRecord := ArtifactProtocolRecord{
			ID:      connection.ID,
			Name:    connection.Name,
			Type:    connection.Type,
			Enabled: connection.IsEnabled,
			Config:  scrubArtifactConnectionSecrets(connection.Config),
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
		Version:        ProjectArtifactVersionV1,
		ProjectID:      projectID,
		GeneratedAt:    generatedAt.UTC(),
		Connections:    connections,
		Queries:        queries,
		DataPoints:     dataPoints,
		Compute:        computeUnits,
		HistoryStorage: append([]HistoryStorageConfigRecord(nil), snapshot.HistoryStorage...),
		Alarms: ArtifactAlarmsPayload{
			SchemaVersion:  "alarm.item.v1",
			HistoryStorage: alarmHistoryStorage,
			Groups:         append([]AlarmGroupRecord{}, snapshot.AlarmGroups...),
			Items:          buildArtifactAlarmItems(snapshot.AlarmItems),
			Settings:       snapshot.AlarmSettings,
			Channels:       append([]AlarmNotificationChannelRecord{}, snapshot.AlarmChannels...),
			SecretRefs:     secretRefs,
		},
		Mqtt: ArtifactMqttPayload{
			Connections:   mqttConnections,
			Subscriptions: append([]SnapshotMqttSubscriptionRecord{}, snapshot.MqttSubscriptions...),
			Tags:          append([]SnapshotMqttTagRecord{}, snapshot.MqttTags...),
		},
		Protocols:     protocols,
		BuiltinStores: builtinStores,
		Collectors: ArtifactCollectorPayload{
			Connections: append([]SnapshotCollectorConnectionRecord(nil), snapshot.CollectorConnections...),
			PointGroups: append([]SnapshotCollectorPointGroupRecord(nil), snapshot.CollectorPointGroups...),
			Points:      append([]SnapshotCollectorPointRecord(nil), snapshot.CollectorPoints...),
			SecretRefs:  collectorSecretRefs,
		},
		SecretRefs: connectionSecretRefs,
	}
}

func buildArtifactAlarmItems(records []AlarmItemRecord) []ArtifactAlarmItem {
	items := make([]ArtifactAlarmItem, 0, len(records))
	for _, record := range records {
		items = append(items, ArtifactAlarmItem{
			AlarmItemID:       record.ID,
			DatapointID:       record.DatapointID,
			Path:              record.Path,
			GroupID:           record.GroupID,
			DisplayName:       record.DisplayName,
			Description:       record.Description,
			Mode:              record.Mode,
			AlarmType:         record.AlarmType,
			EvaluationMode:    record.EvaluationMode,
			Inputs:            append([]AlarmItemInputRecord{}, record.Inputs...),
			DerivedExpression: record.DerivedExpression,
			Conditions:        append([]AlarmItemConditionRecord{}, record.Conditions...),
			Notification: ArtifactAlarmNotification{
				Mode:                  record.NotificationMode,
				NotifyOnRaise:         record.NotifyOnRaise,
				NotifyOnClear:         record.NotifyOnClear,
				RepeatIntervalSeconds: record.RepeatIntervalSeconds,
				ChannelIDs:            append([]string{}, record.NotificationChannelIDs...),
				MessageTemplate:       record.MessageTemplate,
			},
			IsEnabled: record.IsEnabled,
			Revision:  record.Revision,
		})
	}
	return items
}

func buildArtifactConnectionConfig(connection ConnectionRecord) map[string]any {
	if !strings.HasPrefix(connection.Type, "builtin.") {
		return scrubArtifactConnectionSecrets(connection.Config)
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

func scrubArtifactConnectionSecrets(config map[string]any) map[string]any {
	result := map[string]any{}
	for key, value := range config {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized == "password" || normalized == "token" || normalized == "dsn" || normalized == "privatekey" || normalized == "private_key" {
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			result[key] = scrubArtifactConnectionSecrets(typed)
		default:
			result[key] = typed
		}
	}
	return result
}

func connectionConfigContainsPlaintextSecret(config map[string]any, parent string) bool {
	for key, value := range config {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized == "password" || normalized == "token" || normalized == "dsn" || normalized == "privatekey" || normalized == "private_key" {
			return true
		}
		if strings.EqualFold(parent, "sslConfig") && (normalized == "ca" || normalized == "cert" || normalized == "key") {
			return true
		}
		if nested, ok := value.(map[string]any); ok && connectionConfigContainsPlaintextSecret(nested, key) {
			return true
		}
	}
	return false
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

func (r *ProjectSnapshotRepository) listConnections(ctx context.Context, queryer projectSnapshotQuerier, projectID string) ([]ConnectionRecord, error) {
	rows, err := queryer.Query(ctx, `
		SELECT id, project_id, name, type, category, is_enabled, metadata, display_order, 0,
		       'not_tested'::text, NULL::timestamptz, NULL::integer, NULL::text, created_at, updated_at
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

func (r *ProjectSnapshotRepository) listQueries(ctx context.Context, queryer projectSnapshotQuerier, projectID string) ([]QueryRecord, error) {
	rows, err := queryer.Query(ctx, `
        SELECT id, project_id, connection_id, name, description, category, group_id, query_type, config, transformer,
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
	for index := range result {
		outputs, outputErr := listSourceOutputMappings(ctx, queryer, "query", result[index].ID)
		if outputErr != nil {
			return nil, outputErr
		}
		result[index].Outputs = outputs
	}
	return result, nil
}

func (r *ProjectSnapshotRepository) listCollectorSnapshot(ctx context.Context, queryer projectSnapshotQuerier, projectID string) ([]SnapshotCollectorConnectionRecord, []SnapshotCollectorSecretRecord, []SnapshotCollectorPointGroupRecord, []SnapshotCollectorPointRecord, error) {
	connections := make([]SnapshotCollectorConnectionRecord, 0)
	rows, err := queryer.Query(ctx, `SELECT id::text,name,code,protocol_family,driver_id,driver_version,schema_version,
		config,metadata,is_enabled,default_acquisition,display_order,created_at,updated_at
		FROM data_collector_connections WHERE project_id=$1 ORDER BY display_order,id`, projectID)
	if err != nil {
		return nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照工业连接失败", err)
	}
	for rows.Next() {
		var item SnapshotCollectorConnectionRecord
		if err := rows.Scan(&item.ID, &item.Name, &item.Code, &item.ProtocolFamily, &item.DriverID, &item.DriverVersion,
			&item.SchemaVersion, &item.Config, &item.Metadata, &item.IsEnabled, &item.DefaultAcquisition,
			&item.DisplayOrder, &item.CreatedAt, &item.UpdatedAt); err != nil {
			rows.Close()
			return nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析快照工业连接失败", err)
		}
		connections = append(connections, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照工业连接失败", err)
	}
	rows.Close()

	secrets := make([]SnapshotCollectorSecretRecord, 0)
	rows, err = queryer.Query(ctx, `SELECT secret.connection_id::text,secret.secret_key,secret.encrypted_value,
		secret.encryption_key_version,secret.created_at,secret.updated_at
		FROM data_collector_connection_secrets secret
		JOIN data_collector_connections connection ON connection.id=secret.connection_id
		WHERE connection.project_id=$1 ORDER BY secret.connection_id,secret.secret_key`, projectID)
	if err != nil {
		return nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照工业密钥失败", err)
	}
	for rows.Next() {
		var item SnapshotCollectorSecretRecord
		if err := rows.Scan(&item.ConnectionID, &item.SecretKey, &item.EncryptedValue, &item.EncryptionKeyVersion, &item.CreatedAt, &item.UpdatedAt); err != nil {
			rows.Close()
			return nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析快照工业密钥失败", err)
		}
		secrets = append(secrets, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照工业密钥失败", err)
	}
	rows.Close()

	groups := make([]SnapshotCollectorPointGroupRecord, 0)
	rows, err = queryer.Query(ctx, `SELECT id::text,connection_id::text,parent_id::text,name,sort_order,metadata,created_at,updated_at
		FROM data_collector_point_groups WHERE project_id=$1 ORDER BY connection_id,sort_order,id`, projectID)
	if err != nil {
		return nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照工业点目录失败", err)
	}
	for rows.Next() {
		var item SnapshotCollectorPointGroupRecord
		if err := rows.Scan(&item.ID, &item.ConnectionID, &item.ParentID, &item.Name, &item.SortOrder, &item.Metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
			rows.Close()
			return nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析快照工业点目录失败", err)
		}
		groups = append(groups, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照工业点目录失败", err)
	}
	rows.Close()

	points := make([]SnapshotCollectorPointRecord, 0)
	rows, err = queryer.Query(ctx, `SELECT id::text,connection_id::text,group_id::text,code,name,description,address,address_text,
		address_schema_version,data_type,element_count,read_options,acquisition_mode,acquisition_overrides,enabled,
		sort_order,metadata,created_at,updated_at FROM data_collector_points WHERE project_id=$1 ORDER BY connection_id,sort_order,id`, projectID)
	if err != nil {
		return nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照工业点失败", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item SnapshotCollectorPointRecord
		if err := rows.Scan(&item.ID, &item.ConnectionID, &item.GroupID, &item.Code, &item.Name, &item.Description,
			&item.Address, &item.AddressText, &item.AddressSchemaVersion, &item.DataType, &item.ElementCount,
			&item.ReadOptions, &item.AcquisitionMode, &item.AcquisitionOverrides, &item.Enabled, &item.SortOrder,
			&item.Metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析快照工业点失败", err)
		}
		points = append(points, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照工业点失败", err)
	}
	return connections, secrets, groups, points, nil
}

func (r *ProjectSnapshotRepository) listWorkbenchSnapshot(ctx context.Context, queryer projectSnapshotQuerier, projectID string) ([]SnapshotHTTPRequestGroupRecord, []SnapshotHTTPRequestRecord, []SnapshotWebSocketGroupRecord, []SnapshotWebSocketSessionRecord, []SnapshotRealtimeKeyRecord, error) {
	httpGroups := make([]SnapshotHTTPRequestGroupRecord, 0)
	rows, err := queryer.Query(ctx, `SELECT id::text,connection_id::text,parent_id::text,name,sort_order FROM data_http_request_groups WHERE project_id=$1 ORDER BY connection_id,sort_order,id`, projectID)
	if err != nil {
		return nil, nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照 HTTP 分组失败", err)
	}
	for rows.Next() {
		var item SnapshotHTTPRequestGroupRecord
		if err := rows.Scan(&item.ID, &item.ConnectionID, &item.ParentID, &item.Name, &item.SortOrder); err != nil {
			rows.Close()
			return nil, nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析快照 HTTP 分组失败", err)
		}
		httpGroups = append(httpGroups, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, nil, nil, nil, err
	}
	rows.Close()

	httpRequests := make([]SnapshotHTTPRequestRecord, 0)
	rows, err = queryer.Query(ctx, `SELECT id::text,connection_id::text,group_id::text,name,method,url,params,headers,auth,body_type,body,settings,enabled,sort_order FROM data_http_requests WHERE project_id=$1 ORDER BY connection_id,sort_order,id`, projectID)
	if err != nil {
		return nil, nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照 HTTP 请求失败", err)
	}
	for rows.Next() {
		var item SnapshotHTTPRequestRecord
		if err := rows.Scan(&item.ID, &item.ConnectionID, &item.GroupID, &item.Name, &item.Method, &item.URL,
			&item.Params, &item.Headers, &item.Auth, &item.BodyType, &item.Body, &item.Settings, &item.Enabled, &item.SortOrder); err != nil {
			rows.Close()
			return nil, nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析快照 HTTP 请求失败", err)
		}
		httpRequests = append(httpRequests, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, nil, nil, nil, err
	}
	rows.Close()
	for index := range httpRequests {
		httpRequests[index].Outputs, err = listSourceOutputMappings(ctx, queryer, "http", httpRequests[index].ID)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
	}

	websocketGroups := make([]SnapshotWebSocketGroupRecord, 0)
	rows, err = queryer.Query(ctx, `SELECT id::text,connection_id::text,parent_id::text,name,sort_order FROM data_websocket_session_groups WHERE project_id=$1 ORDER BY connection_id,sort_order,id`, projectID)
	if err != nil {
		return nil, nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照 WebSocket 分组失败", err)
	}
	for rows.Next() {
		var item SnapshotWebSocketGroupRecord
		if err := rows.Scan(&item.ID, &item.ConnectionID, &item.ParentID, &item.Name, &item.SortOrder); err != nil {
			rows.Close()
			return nil, nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析快照 WebSocket 分组失败", err)
		}
		websocketGroups = append(websocketGroups, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, nil, nil, nil, err
	}
	rows.Close()

	websocketSessions := make([]SnapshotWebSocketSessionRecord, 0)
	rows, err = queryer.Query(ctx, `SELECT id::text,connection_id::text,group_id::text,name,url,headers,auth,protocols,messages,settings,enabled,sort_order FROM data_websocket_sessions WHERE project_id=$1 ORDER BY connection_id,sort_order,id`, projectID)
	if err != nil {
		return nil, nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照 WebSocket 会话失败", err)
	}
	for rows.Next() {
		var item SnapshotWebSocketSessionRecord
		if err := rows.Scan(&item.ID, &item.ConnectionID, &item.GroupID, &item.Name, &item.URL, &item.Headers,
			&item.Auth, &item.Protocols, &item.Messages, &item.Settings, &item.Enabled, &item.SortOrder); err != nil {
			rows.Close()
			return nil, nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析快照 WebSocket 会话失败", err)
		}
		websocketSessions = append(websocketSessions, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, nil, nil, nil, err
	}
	rows.Close()
	for index := range websocketSessions {
		websocketSessions[index].Outputs, err = listSourceOutputMappings(ctx, queryer, "websocket", websocketSessions[index].ID)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
	}

	realtimeKeys := make([]SnapshotRealtimeKeyRecord, 0)
	rows, err = queryer.Query(ctx, `SELECT id::text,connection_id::text,provider,key_path,redis_type,value_type,default_ttl_seconds,description FROM data_realtime_keys WHERE project_id=$1 ORDER BY connection_id,key_path,id`, projectID)
	if err != nil {
		return nil, nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照实时 Key 失败", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item SnapshotRealtimeKeyRecord
		if err := rows.Scan(&item.ID, &item.ConnectionID, &item.Provider, &item.KeyPath, &item.RedisType, &item.ValueType, &item.DefaultTtlSeconds, &item.Description); err != nil {
			return nil, nil, nil, nil, nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析快照实时 Key 失败", err)
		}
		realtimeKeys = append(realtimeKeys, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, nil, nil, err
	}
	rows.Close()
	for index := range realtimeKeys {
		realtimeKeys[index].Outputs, err = listSourceOutputMappings(ctx, queryer, "realtime", realtimeKeys[index].ID)
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
	}
	return httpGroups, httpRequests, websocketGroups, websocketSessions, realtimeKeys, nil
}

func (r *ProjectSnapshotRepository) listDataPoints(ctx context.Context, queryer projectSnapshotQuerier, projectID string) ([]DataPointRecord, error) {
	rows, err := queryer.Query(ctx, `
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

func (r *ProjectSnapshotRepository) listComputeUnits(ctx context.Context, queryer projectSnapshotQuerier, projectID string) ([]ComputeUnitRecord, error) {
	rows, err := queryer.Query(ctx, `
		SELECT id, project_id, name, description, folder_id, language, script_code, trigger_type, trigger_config,
		       input_bindings, dependencies, timeout_ms, is_enabled, revision, created_at, updated_at
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
	rows.Close()
	for index := range result {
		if err := hydrateSnapshotComputeOutputs(ctx, queryer, &result[index]); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func hydrateSnapshotComputeOutputs(ctx context.Context, queryer projectSnapshotQuerier, unit *ComputeUnitRecord) error {
	if unit == nil {
		return nil
	}
	rows, err := queryer.Query(ctx, `
		SELECT output.id::text,output.datapoint_id::text,output.output_key,point.name,point.description,point.default_value,
		       output.path,output.data_type,output.unit,output.precision_num,output.null_policy,output.sort_order
		FROM data_compute_unit_outputs output
		JOIN data_points point ON point.id=output.datapoint_id
		WHERE output.project_id=$1 AND output.compute_unit_id=$2
		ORDER BY output.sort_order,output.id
	`, unit.ProjectID, unit.ID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询计算输出失败", err)
	}
	defer rows.Close()
	unit.Outputs = make([]ComputeOutputRecord, 0)
	for rows.Next() {
		var output ComputeOutputRecord
		if err := rows.Scan(&output.ID, &output.DatapointID, &output.OutputKey, &output.Name, &output.Description, &output.DefaultValue, &output.Path, &output.DataType, &output.Unit, &output.PrecisionNum, &output.NullPolicy, &output.SortOrder); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取计算输出失败", err)
		}
		unit.Outputs = append(unit.Outputs, output)
	}
	if err := rows.Err(); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历计算输出失败", err)
	}
	return nil
}

func (r *ProjectSnapshotRepository) listAlarmChannelSecrets(ctx context.Context, queryer projectSnapshotQuerier, projectID string) ([]SnapshotAlarmChannelSecretRecord, error) {
	rows, err := queryer.Query(ctx, `
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

func (r *ProjectSnapshotRepository) listConnectionSecrets(ctx context.Context, queryer projectSnapshotQuerier, projectID string) ([]SnapshotConnectionSecretRecord, error) {
	rows, err := queryer.Query(ctx, `
		SELECT secret.connection_id, secret.secret_key, secret.encrypted_value,
		       secret.encryption_key_version, secret.created_at, secret.updated_at
		FROM data_connection_secrets secret
		JOIN data_connections connection ON connection.id = secret.connection_id
		WHERE connection.project_id = $1
		ORDER BY connection.display_order, secret.secret_key
	`, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询快照接入源密钥失败", err)
	}
	defer rows.Close()
	result := make([]SnapshotConnectionSecretRecord, 0)
	for rows.Next() {
		var record SnapshotConnectionSecretRecord
		if err := rows.Scan(&record.ConnectionID, &record.SecretKey, &record.EncryptedValue, &record.EncryptionKeyVersion, &record.CreatedAt, &record.UpdatedAt); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照接入源密钥失败", err)
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (r *ProjectSnapshotRepository) listMqttConfigs(ctx context.Context, queryer projectSnapshotQuerier, projectID string) ([]SnapshotMqttConfigRecord, error) {
	rows, err := queryer.Query(ctx, `
        SELECT
            cfg.connection_id,
            cfg.broker_url,
            cfg.protocol,
            cfg.port,
			cfg.client_id,
			cfg.username,
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

func (r *ProjectSnapshotRepository) listMqttSubscriptions(ctx context.Context, queryer projectSnapshotQuerier, projectID string) ([]SnapshotMqttSubscriptionRecord, error) {
	rows, err := queryer.Query(ctx, `
		SELECT id, project_id, connection_id, name, topic, qos, usage_mode, group_id, description, display_order, message_retention, default_batch_parse_rule, created_at, updated_at
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
			&record.GroupID,
			&record.Description,
			&record.DisplayOrder,
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

func (r *ProjectSnapshotRepository) listMqttTags(ctx context.Context, queryer projectSnapshotQuerier, projectID string) ([]SnapshotMqttTagRecord, error) {
	rows, err := queryer.Query(ctx, `
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

func (r *ProjectSnapshotRepository) listHistoryStorageConfigs(ctx context.Context, queryer projectSnapshotQuerier, projectID string) ([]HistoryStorageConfigRecord, error) {
	rows, err := queryer.Query(ctx, `SELECT `+historyStorageConfigColumns+`
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
		config.Targets, err = r.listHistoryStorageTargets(ctx, queryer, projectID, config.ID)
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

func (r *ProjectSnapshotRepository) listHistoryStorageTargets(ctx context.Context, queryer projectSnapshotQuerier, projectID, configID string) ([]HistoryStorageTargetRecord, error) {
	rows, err := queryer.Query(ctx, `SELECT target.id,target.project_id,target.config_id,target.connection_id,
        connection.name,connection.type,'unknown',target.is_primary,target.sort_order,target.retention_days,
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
		if err := rows.Scan(&target.ID, &target.ProjectID, &target.ConfigID, &target.ConnectionID, &target.ConnectionName, &target.ConnectionType, &target.LastTestStatus, &target.IsPrimary, &target.SortOrder, &target.RetentionDays, &target.CreatedAt, &target.UpdatedAt); err != nil {
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
            (id,project_id,access_source_id,collector_connection_id,compute_unit_id,datapoint_id,is_enabled,write_mode,
             interval_ms,deadband,max_silence_ms,offline_behavior,created_by,updated_by,created_at,updated_at)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13,$14,$15)`,
			config.ID, projectID, config.AccessSourceID, config.CollectorConnectionID, config.ComputeUnitID, config.DatapointID,
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
		`DELETE FROM data_source_output_mappings WHERE project_id = $1`,
		`DELETE FROM data_alarm_items WHERE project_id = $1`,
		`DELETE FROM data_alarm_history_settings WHERE project_id = $1`,
		`DELETE FROM data_alarm_project_settings WHERE project_id = $1`,
		`DELETE FROM data_alarm_notification_channels WHERE project_id = $1`,
		`DELETE FROM data_alarm_groups WHERE project_id = $1`,
		`DELETE FROM data_alarm_config_sync_requests WHERE project_id = $1`,
		`DELETE FROM data_alarm_config_sync_state WHERE project_id = $1`,
		`DELETE FROM data_compute_units WHERE project_id = $1`,
		`DELETE FROM data_compute_dependencies WHERE project_id = $1`,
		`DELETE FROM data_compute_folders WHERE project_id = $1`,
		`DELETE FROM data_mqtt_tags WHERE project_id = $1`,
		`DELETE FROM data_mqtt_subscriptions WHERE project_id = $1`,
		`DELETE FROM data_points WHERE project_id = $1`,
		`DELETE FROM data_collector_connections WHERE project_id = $1`,
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
				id, project_id, name, type, category, is_enabled, metadata, display_order, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11, $12)
		`, connection.ID, projectID, connection.Name, connection.Type, category, connection.IsEnabled, string(configBytes), connection.DisplayOrder, actorID, actorID, createdAt, updatedAt); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照连接失败", err)
		}
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertConnectionSecrets(ctx context.Context, tx pgx.Tx, secrets []SnapshotConnectionSecretRecord) error {
	for _, secret := range secrets {
		if strings.TrimSpace(secret.SecretKey) == "" || len(secret.EncryptedValue) == 0 || strings.TrimSpace(secret.EncryptionKeyVersion) == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照接入源密钥记录不完整")
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO data_connection_secrets
				(connection_id, secret_key, encrypted_value, encryption_key_version, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, secret.ConnectionID, secret.SecretKey, secret.EncryptedValue, secret.EncryptionKeyVersion, coalesceTime(secret.CreatedAt), coalesceTime(secret.UpdatedAt)); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入快照接入源密钥失败", err)
		}
	}
	return nil
}

// insertProtocolConnectionConfigs 从快照中的非敏感结构化 metadata 恢复专用配置表。
// 密钥由 insertConnectionSecrets 单独恢复，二者都处在同一个覆盖事务中。
func (r *ProjectSnapshotRepository) insertProtocolConnectionConfigs(ctx context.Context, tx pgx.Tx, connections []ConnectionRecord) error {
	for _, connection := range connections {
		config := connection.Config
		optionsBytes, err := marshalSnapshotObject(parseSnapshotObject(config["options"]))
		if err != nil {
			return err
		}
		switch connection.Type {
		case "kafka":
			_, err = tx.Exec(ctx, `INSERT INTO data_kafka_configs(connection_id,brokers,topic,consumer_group,start_position,options) VALUES($1,$2,$3,$4,$5,$6::jsonb)`, connection.ID, parseSnapshotString(config["brokers"], ""), parseSnapshotOptionalString(config["topic"]), parseSnapshotOptionalString(config["consumerGroup"]), parseSnapshotString(config["startPosition"], "latest"), string(optionsBytes))
		case "http":
			_, err = tx.Exec(ctx, `INSERT INTO data_http_configs(connection_id,base_url,method,headers,timeout_ms) VALUES($1,'','GET','{}'::jsonb,30000)`, connection.ID)
		case "websocket":
			_, err = tx.Exec(ctx, `INSERT INTO data_websocket_configs(connection_id,url,headers,heartbeat_interval_ms) VALUES($1,NULL,'{}'::jsonb,30000)`, connection.ID)
		case "redis":
			_, err = tx.Exec(ctx, `INSERT INTO data_redis_configs(connection_id,address,db,username,key_pattern,mode,options) VALUES($1,$2,$3,$4,$5,$6,$7::jsonb)`, connection.ID, parseSnapshotString(config["address"], ""), parseSnapshotInt(config["db"]), parseSnapshotOptionalString(config["username"]), parseSnapshotString(config["keyPattern"], "*"), parseSnapshotString(config["mode"], "standalone"), string(optionsBytes))
		case "tdengine":
			_, err = tx.Exec(ctx, `INSERT INTO data_tdengine_configs(connection_id,protocol,host,port,username,database_name,timezone,tls_skip_verify,options) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb)`, connection.ID, parseSnapshotString(config["protocol"], "ws"), parseSnapshotString(config["host"], ""), parseSnapshotInt(config["port"]), parseSnapshotString(config["username"], ""), parseSnapshotString(config["databaseName"], ""), parseSnapshotOptionalString(config["timezone"]), parseSnapshotBool(config["tlsSkipVerify"]), string(optionsBytes))
		case "relational":
			sslBytes, marshalErr := marshalSnapshotObject(parseSnapshotObject(config["sslConfig"]))
			if marshalErr != nil {
				return marshalErr
			}
			_, err = tx.Exec(ctx, `INSERT INTO data_relational_configs(connection_id,db_type,host,port,database,username,schema,charset,timezone,ssl,ssl_config,options) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12::jsonb)`, connection.ID, parseSnapshotString(config["dbType"], "postgresql"), parseSnapshotString(config["host"], ""), parseSnapshotInt(config["port"]), parseSnapshotString(config["database"], ""), parseSnapshotString(config["username"], ""), parseSnapshotOptionalString(config["schema"]), parseSnapshotString(config["charset"], "utf8mb4"), parseSnapshotOptionalString(config["timezone"]), parseSnapshotBool(config["ssl"]), nullableJSONString(sslBytes), string(optionsBytes))
		}
		if err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "恢复快照专用接入源配置失败", err)
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
				connection_id, broker_url, protocol, port, client_id, username,
				keepalive, clean_session, qos, reconnect_period_ms, connect_timeout_ms, will, ssl_config
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb, $13::jsonb)
		`, config.ConnectionID, config.BrokerURL, config.Protocol, config.Port, config.ClientID, config.Username, config.Keepalive, config.CleanSession, config.QOS, config.ReconnectPeriodMS, config.ConnectTimeoutMS, nullableJSONString(willBytes), nullableJSONString(sslConfigBytes)); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照 MQTT 配置失败", err)
		}
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertCollectorSnapshot(ctx context.Context, tx pgx.Tx, projectID, actorID string, snapshot ProjectSnapshot) error {
	for _, item := range snapshot.CollectorConnections {
		config, err := marshalSnapshotObject(item.Config)
		if err != nil {
			return err
		}
		metadata, err := marshalSnapshotObject(item.Metadata)
		if err != nil {
			return err
		}
		defaults, err := marshalSnapshotObject(item.DefaultAcquisition)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO data_collector_connections(
			id,project_id,name,code,protocol_family,driver_id,driver_version,schema_version,config,metadata,
			is_enabled,default_acquisition,display_order,created_by,updated_by,created_at,updated_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10::jsonb,$11,$12::jsonb,$13,$14,$14,$15,$16)`,
			item.ID, projectID, item.Name, item.Code, item.ProtocolFamily, item.DriverID, item.DriverVersion,
			item.SchemaVersion, string(config), string(metadata), item.IsEnabled, string(defaults), item.DisplayOrder,
			actorID, coalesceTime(item.CreatedAt), coalesceTime(item.UpdatedAt)); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入快照工业连接失败", err)
		}
	}
	for _, secret := range snapshot.CollectorSecrets {
		if strings.TrimSpace(secret.SecretKey) == "" || len(secret.EncryptedValue) == 0 || strings.TrimSpace(secret.EncryptionKeyVersion) == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照工业连接密钥记录不完整")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO data_collector_connection_secrets(
			connection_id,secret_key,encrypted_value,encryption_key_version,created_at,updated_at
		) VALUES($1,$2,$3,$4,$5,$6)`, secret.ConnectionID, secret.SecretKey, secret.EncryptedValue,
			secret.EncryptionKeyVersion, coalesceTime(secret.CreatedAt), coalesceTime(secret.UpdatedAt)); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入快照工业连接密钥失败", err)
		}
	}

	pending := append([]SnapshotCollectorPointGroupRecord(nil), snapshot.CollectorPointGroups...)
	inserted := make(map[string]bool, len(pending))
	for len(pending) > 0 {
		next := make([]SnapshotCollectorPointGroupRecord, 0)
		progressed := false
		for _, item := range pending {
			if item.ParentID != nil && !inserted[*item.ParentID] {
				next = append(next, item)
				continue
			}
			metadata, err := marshalSnapshotObject(item.Metadata)
			if err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO data_collector_point_groups(
				id,project_id,connection_id,parent_id,name,sort_order,metadata,created_at,updated_at
			) VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9)`, item.ID, projectID, item.ConnectionID,
				item.ParentID, item.Name, item.SortOrder, string(metadata), coalesceTime(item.CreatedAt), coalesceTime(item.UpdatedAt)); err != nil {
				return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入快照工业点目录失败", err)
			}
			inserted[item.ID] = true
			progressed = true
		}
		if !progressed {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照工业点目录存在缺失父目录或循环")
		}
		pending = next
	}

	for _, item := range snapshot.CollectorPoints {
		address, err := marshalSnapshotObject(item.Address)
		if err != nil {
			return err
		}
		readOptions, err := marshalSnapshotObject(item.ReadOptions)
		if err != nil {
			return err
		}
		overrides, err := marshalSnapshotObject(item.AcquisitionOverrides)
		if err != nil {
			return err
		}
		metadata, err := marshalSnapshotObject(item.Metadata)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO data_collector_points(
			id,project_id,connection_id,group_id,code,name,description,address,address_text,address_schema_version,
			data_type,element_count,read_options,acquisition_mode,acquisition_overrides,enabled,sort_order,metadata,created_at,updated_at
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10,$11,$12,$13::jsonb,$14,$15::jsonb,$16,$17,$18::jsonb,$19,$20)`,
			item.ID, projectID, item.ConnectionID, item.GroupID, item.Code, item.Name, item.Description, string(address),
			item.AddressText, item.AddressSchemaVersion, item.DataType, item.ElementCount, string(readOptions),
			item.AcquisitionMode, string(overrides), item.Enabled, item.SortOrder, string(metadata),
			coalesceTime(item.CreatedAt), coalesceTime(item.UpdatedAt)); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入快照工业点失败", err)
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
                id, project_id, connection_id, group_id, name, description, category, query_type, config, transformer,
                is_enabled, timeout_ms, cache_enabled, cache_ttl_seconds, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11, $12, $13, $14, $15, $16, $17, $18)
        `, query.ID, projectID, query.ConnectionID, query.GroupID, query.Name, query.Description, query.Category, query.QueryType, string(configBytes), query.Transformer, query.IsEnabled, query.TimeoutMS, query.CacheEnabled, query.CacheTtlSeconds, actorID, actorID, createdAt, updatedAt); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照查询失败", err)
		}
	}
	return nil
}

// insertQueryOutputMappings 在查询和数据点都恢复后重建稳定映射关系，保持映射 ID 与生成点 ID 不变。
func (r *ProjectSnapshotRepository) insertQueryOutputMappings(ctx context.Context, tx pgx.Tx, projectID string, queries []QueryRecord) error {
	for _, query := range queries {
		for _, output := range query.Outputs {
			segments, err := json.Marshal(output.Selector.Segments)
			if err != nil {
				return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照查询输出路径无效", err)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO data_source_output_mappings(
					id,project_id,query_id,datapoint_id,output_key,display_name,selector_kind,
					selector_column,selector_path,data_type,unit,precision_num,sort_order
				) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,$13)
			`, output.ID, projectID, query.ID, output.DataPointID, output.Key, output.DisplayName,
				output.Selector.Kind, output.Selector.Column, string(segments), output.DataType,
				output.Unit, output.PrecisionNum, output.SortOrder); err != nil {
				return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入快照查询输出映射失败", err)
			}
		}
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertWorkbenchOwners(ctx context.Context, tx pgx.Tx, projectID, actorID string, snapshot ProjectSnapshot) error {
	marshal := func(label string, value any) (string, error) {
		payload, err := json.Marshal(value)
		if err != nil {
			return "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照"+label+"格式无效", err)
		}
		return string(payload), nil
	}
	if err := insertSnapshotGroups(ctx, tx, projectID, actorID, "data_http_request_groups", snapshot.HTTPRequestGroups); err != nil {
		return err
	}
	for _, item := range snapshot.HTTPRequests {
		params, err := marshal(" HTTP 参数", item.Params)
		if err != nil {
			return err
		}
		headers, err := marshal(" HTTP 请求头", item.Headers)
		if err != nil {
			return err
		}
		auth, err := marshal(" HTTP 认证", item.Auth)
		if err != nil {
			return err
		}
		body, err := marshal(" HTTP 请求体", item.Body)
		if err != nil {
			return err
		}
		settings, err := marshal(" HTTP 设置", item.Settings)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO data_http_requests(
			id,project_id,connection_id,group_id,name,method,url,params,headers,auth,body_type,body,settings,
			enabled,sort_order,quality,created_by,updated_by
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9::jsonb,$10::jsonb,$11,$12::jsonb,$13::jsonb,$14,$15,'unknown',$16,$16)`,
			item.ID, projectID, item.ConnectionID, item.GroupID, item.Name, item.Method, item.URL, params,
			headers, auth, item.BodyType, body, settings, item.Enabled, item.SortOrder, actorID); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入快照 HTTP 请求失败", err)
		}
	}
	if err := insertSnapshotGroups(ctx, tx, projectID, actorID, "data_websocket_session_groups", snapshot.WebSocketGroups); err != nil {
		return err
	}
	for _, item := range snapshot.WebSocketSessions {
		headers, err := marshal(" WebSocket 请求头", item.Headers)
		if err != nil {
			return err
		}
		auth, err := marshal(" WebSocket 认证", item.Auth)
		if err != nil {
			return err
		}
		protocols, err := marshal(" WebSocket 子协议", item.Protocols)
		if err != nil {
			return err
		}
		messages, err := marshal(" WebSocket 消息", item.Messages)
		if err != nil {
			return err
		}
		settings, err := marshal(" WebSocket 设置", item.Settings)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO data_websocket_sessions(
			id,project_id,connection_id,group_id,name,url,headers,auth,protocols,messages,settings,
			enabled,sort_order,quality,created_by,updated_by
		) VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9::jsonb,$10::jsonb,$11::jsonb,$12,$13,'unknown',$14,$14)`,
			item.ID, projectID, item.ConnectionID, item.GroupID, item.Name, item.URL, headers, auth,
			protocols, messages, settings, item.Enabled, item.SortOrder, actorID); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入快照 WebSocket 会话失败", err)
		}
	}
	for _, item := range snapshot.RealtimeKeys {
		if _, err := tx.Exec(ctx, `INSERT INTO data_realtime_keys(
			id,project_id,connection_id,provider,key_path,redis_type,value_type,default_ttl_seconds,description
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, item.ID, projectID, item.ConnectionID, item.Provider,
			item.KeyPath, item.RedisType, item.ValueType, item.DefaultTtlSeconds, item.Description); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入快照实时 Key 失败", err)
		}
	}
	return nil
}

type snapshotGroup interface {
	SnapshotHTTPRequestGroupRecord | SnapshotWebSocketGroupRecord
}

func insertSnapshotGroups[T snapshotGroup](ctx context.Context, tx pgx.Tx, projectID, actorID, table string, groups []T) error {
	type normalized struct {
		id, connectionID string
		parentID         *string
		name             string
		sortOrder        int
	}
	pending := make([]normalized, 0, len(groups))
	for _, raw := range groups {
		switch item := any(raw).(type) {
		case SnapshotHTTPRequestGroupRecord:
			pending = append(pending, normalized{item.ID, item.ConnectionID, item.ParentID, item.Name, item.SortOrder})
		case SnapshotWebSocketGroupRecord:
			pending = append(pending, normalized{item.ID, item.ConnectionID, item.ParentID, item.Name, item.SortOrder})
		}
	}
	inserted := make(map[string]bool, len(pending))
	for len(pending) > 0 {
		next := make([]normalized, 0)
		progressed := false
		for _, item := range pending {
			if item.parentID != nil && !inserted[*item.parentID] {
				next = append(next, item)
				continue
			}
			query := `INSERT INTO ` + table + `(id,project_id,connection_id,parent_id,name,sort_order,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$7)`
			if _, err := tx.Exec(ctx, query, item.id, projectID, item.connectionID, item.parentID, item.name, item.sortOrder, actorID); err != nil {
				return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入快照工作台分组失败", err)
			}
			inserted[item.id] = true
			progressed = true
		}
		if !progressed {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照工作台分组存在缺失父目录或循环")
		}
		pending = next
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertWorkbenchOutputMappings(ctx context.Context, tx pgx.Tx, projectID string, snapshot ProjectSnapshot) error {
	insert := func(ownerColumn, ownerID string, outputs []SourceOutputMappingRecord) error {
		allowed := map[string]bool{"http_request_id": true, "websocket_session_id": true, "realtime_key_id": true}
		if !allowed[ownerColumn] {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照输出映射所属对象无效")
		}
		for _, output := range outputs {
			segments, err := json.Marshal(output.Selector.Segments)
			if err != nil {
				return err
			}
			query := `INSERT INTO data_source_output_mappings(id,project_id,` + ownerColumn + `,datapoint_id,
				output_key,display_name,selector_kind,selector_column,selector_path,data_type,unit,precision_num,sort_order)
				VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,$13)`
			if _, err := tx.Exec(ctx, query, output.ID, projectID, ownerID, output.DataPointID, output.Key,
				output.DisplayName, output.Selector.Kind, output.Selector.Column, string(segments), output.DataType,
				output.Unit, output.PrecisionNum, output.SortOrder); err != nil {
				return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入快照工作台输出映射失败", err)
			}
		}
		return nil
	}
	for _, item := range snapshot.HTTPRequests {
		if err := insert("http_request_id", item.ID, item.Outputs); err != nil {
			return err
		}
	}
	for _, item := range snapshot.WebSocketSessions {
		if err := insert("websocket_session_id", item.ID, item.Outputs); err != nil {
			return err
		}
	}
	for _, item := range snapshot.RealtimeKeys {
		if err := insert("realtime_key_id", item.ID, item.Outputs); err != nil {
			return err
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
                id, project_id, connection_id, name, topic, qos, usage_mode, group_id, description, display_order,
                message_retention, default_batch_parse_rule, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
        `, subscription.ID, projectID, subscription.ConnectionID, subscription.Name, subscription.Topic, subscription.QOS, usageMode, subscription.GroupID, subscription.Description, subscription.DisplayOrder, subscription.MessageRetention, nullableJSONString(defaultBatchRuleBytes), actorID, actorID, createdAt, updatedAt); err != nil {
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
		attributeDefaultsBytes, err := json.Marshal(cloneSnapshotStringMap(datapoint.AttributeDefaults))
		if err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "数据点自定义属性格式无效", err)
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
                unit, precision_num, default_value, min_value, max_value, tags, attribute_defaults, runtime_permissions,
                refresh_mode, refresh_interval_ms, status, display_order, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11, $12, $13, $14, $15::jsonb, $16::jsonb, $17::jsonb, $18, $19, $20, $21, $22, $23, $24, $25)
        `, datapoint.ID, projectID, datapoint.Path, datapoint.Name, datapoint.Description, datapoint.SourceType, datapoint.SourceID, string(sourceConfigBytes), datapoint.DataType, datapoint.Unit, datapoint.PrecisionNum, datapoint.DefaultValue, datapoint.MinValue, datapoint.MaxValue, string(tagsBytes), string(attributeDefaultsBytes), string(runtimePermissionsBytes), datapoint.RefreshMode, datapoint.RefreshIntervalMS, datapoint.Status, datapoint.DisplayOrder, actorID, actorID, createdAt, updatedAt); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照数据点失败", err)
		}
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertComputeUnits(ctx context.Context, tx pgx.Tx, projectID, actorID string, units []ComputeUnitRecord) error {
	for _, unit := range units {
		if unit.Revision < 1 {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照计算单元 revision 必须大于等于 1")
		}
		triggerConfigBytes, err := marshalSnapshotObject(unit.TriggerConfig)
		if err != nil {
			return err
		}
		inputBindingsBytes, err := marshalSnapshotObject(unit.InputBindings)
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
				input_bindings, dependencies, timeout_ms, is_enabled, revision, created_by, updated_by, created_at, updated_at
            )
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10::jsonb, $11::jsonb, $12, $13, $14, $15, $16, $17, $18)
        `, unit.ID, projectID, unit.Name, unit.Description, unit.FolderID, unit.Language, unit.ScriptCode, unit.TriggerType, string(triggerConfigBytes),
			string(inputBindingsBytes), string(dependenciesBytes), unit.TimeoutMS, unit.IsEnabled, unit.Revision, actorID, actorID, createdAt, updatedAt); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照计算单元失败", err)
		}
		refs, err := snapshotComputeDatapointRefs(unit)
		if err != nil {
			return err
		}
		if err := replaceComputeDatapointRefsTx(ctx, tx, unit.ID, projectID, refs); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照计算单元引用的数据点不存在或已失效", err)
		}
		for _, output := range unit.Outputs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO data_compute_unit_outputs(id,project_id,compute_unit_id,datapoint_id,output_key,path,data_type,unit,precision_num,null_policy,sort_order,created_at,updated_at)
				VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
			`, output.ID, projectID, unit.ID, output.DatapointID, output.OutputKey, output.Path, output.DataType, output.Unit, output.PrecisionNum, output.NullPolicy, output.SortOrder, createdAt, updatedAt); err != nil {
				return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入快照计算输出失败", err)
			}
		}
	}
	return nil
}

func snapshotComputeDatapointRefs(unit ComputeUnitRecord) ([]ComputeDatapointRefParam, error) {
	refs := make([]ComputeDatapointRefParam, 0)
	if raw, ok := unit.InputBindings["datapointVariables"].([]any); ok {
		for index, item := range raw {
			mapped, ok := item.(map[string]any)
			if !ok {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照计算数据点变量格式无效")
			}
			id, _ := mapped["datapointId"].(string)
			alias, _ := mapped["alias"].(string)
			id, alias = strings.TrimSpace(id), strings.TrimSpace(alias)
			if id == "" || alias == "" || alias == "true" || alias == "false" {
				return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照计算数据点变量缺少稳定 ID 或别名")
			}
			aliasCopy := alias
			refs = append(refs, ComputeDatapointRefParam{DatapointID: id, Role: "input", Alias: &aliasCopy, SortOrder: index})
		}
	}
	if unit.TriggerType == "datapoint_change" {
		id, _ := unit.TriggerConfig["datapointId"].(string)
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照点变触发缺少稳定数据点 ID")
		}
		refs = append(refs, ComputeDatapointRefParam{DatapointID: id, Role: "trigger"})
	}
	return refs, nil
}

func (r *ProjectSnapshotRepository) insertAlarmGroups(ctx context.Context, tx pgx.Tx, projectID, actorID string, groups []AlarmGroupRecord) error {
	pending := append([]AlarmGroupRecord{}, groups...)
	inserted := make(map[string]struct{}, len(groups))
	for len(pending) > 0 {
		remaining := make([]AlarmGroupRecord, 0, len(pending))
		progress := false
		for _, group := range pending {
			if group.ParentID != nil {
				if _, ok := inserted[*group.ParentID]; !ok {
					remaining = append(remaining, group)
					continue
				}
			}
			createdAt, updatedAt := coalesceTime(group.CreatedAt), coalesceTime(group.UpdatedAt)
			if _, err := tx.Exec(ctx, `INSERT INTO data_alarm_groups
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
	severityDefinitions, err := json.Marshal(settings.SeverityDefinitions)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照报警级别格式无效", err)
	}
	escalationRules, err := json.Marshal(settings.EscalationRules)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照报警升级规则格式无效", err)
	}
	createdAt, updatedAt := coalesceTime(settings.CreatedAt), coalesceTime(settings.UpdatedAt)
	_, err = tx.Exec(ctx, `INSERT INTO data_alarm_project_settings
	    (project_id,notify_on_raise,notify_on_clear,repeat_interval_seconds,default_message_template,default_channel_ids,severity_definitions,escalation_rules,created_by,updated_by,created_at,updated_at)
	    VALUES($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8::jsonb,$9,$9,$10,$11)`, projectID, settings.NotifyOnRaise, settings.NotifyOnClear,
		settings.RepeatIntervalSeconds, settings.DefaultMessageTemplate, string(channelIDs), string(severityDefinitions), string(escalationRules), actorID, createdAt, updatedAt)
	if err != nil {
		return translateAlarmSnapshotError("写入快照报警默认设置失败", err)
	}
	return nil
}

func (r *ProjectSnapshotRepository) insertAlarmHistorySettings(ctx context.Context, tx pgx.Tx, projectID, actorID string, settings *AlarmHistorySettingsRecord) error {
	if settings == nil {
		return nil
	}
	if settings.RetentionDays != nil && *settings.RetentionDays <= 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照报警历史保留天数必须为正整数或永久")
	}
	createdAt, updatedAt := coalesceTime(settings.CreatedAt), coalesceTime(settings.UpdatedAt)
	_, err := tx.Exec(ctx, `INSERT INTO data_alarm_history_settings
	    (project_id,is_enabled,retention_days,store_notification_deliveries,created_by,updated_by,created_at,updated_at)
	    VALUES($1,$2,$3,$4,$5,$5,$6,$7)`, projectID, settings.IsEnabled, settings.RetentionDays,
		settings.StoreNotificationDeliveries, actorID, createdAt, updatedAt)
	if err != nil {
		return translateAlarmSnapshotError("写入快照报警历史设置失败", err)
	}
	return nil
}

func alarmHistoryDefaultRetentionDays() *int {
	value := 30
	return &value
}

func (r *ProjectSnapshotRepository) insertAlarmItems(ctx context.Context, tx pgx.Tx, projectID, actorID string, items []AlarmItemRecord) error {
	for _, item := range items {
		channelIDs, contractBytes, err := marshalAlarmJSON(item.NotificationChannelIDs, item.Contract)
		if err != nil {
			return err
		}
		revision := item.Revision
		if revision < 1 {
			revision = 1
		}
		createdAt, updatedAt := coalesceTime(item.CreatedAt), coalesceTime(item.UpdatedAt)
		if _, err = tx.Exec(ctx, `INSERT INTO data_alarm_items
            (id,project_id,datapoint_id,group_id,display_name,name_key,description,mode,alarm_type,evaluation_mode,derived_expression,trigger_fingerprint,notification_mode,notify_on_raise,notify_on_clear,
             repeat_interval_seconds,notification_channel_ids,message_template,is_enabled,revision,contract,created_by,updated_by,created_at,updated_at)
            VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17::jsonb,$18,$19,$20,$21::jsonb,$22,$22,$23,$24)`,
			item.ID, projectID, item.DatapointID, item.GroupID, item.DisplayName, item.NameKey, item.Description, item.Mode, item.AlarmType, item.EvaluationMode, item.DerivedExpression, item.TriggerFingerprint,
			item.NotificationMode, item.NotifyOnRaise, item.NotifyOnClear, item.RepeatIntervalSeconds, string(channelIDs),
			item.MessageTemplate, item.IsEnabled, revision, string(contractBytes), actorID, createdAt, updatedAt); err != nil {
			return translateAlarmSnapshotError("写入快照报警项失败", err)
		}
		if err = insertSnapshotAlarmItemDetails(ctx, tx, item); err != nil {
			return err
		}
	}
	return nil
}

func insertSnapshotAlarmItemDetails(ctx context.Context, tx pgx.Tx, item AlarmItemRecord) error {
	for index, input := range item.Inputs {
		if input.ID == "" || input.InputKey == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照组合报警输入缺少稳定 ID 或别名")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO data_alarm_item_inputs(id,alarm_item_id,datapoint_id,input_key,sort_order) VALUES($1,$2,$3,$4,$5)`, input.ID, item.ID, input.DatapointID, input.InputKey, index); err != nil {
			return translateAlarmSnapshotError("写入快照组合报警输入失败", err)
		}
	}
	for index, condition := range item.Conditions {
		params, err := json.Marshal(condition.Params)
		if err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照报警条件参数格式无效", err)
		}
		if condition.ID == "" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "快照报警条件缺少稳定 ID")
		}
		_, err = tx.Exec(ctx, `INSERT INTO data_alarm_item_conditions(id,alarm_item_id,kind,operator,label,severity,params,trigger_delay_ms,clear_delay_ms,deadband,sort_order) VALUES($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10,$11)`, condition.ID, item.ID, condition.Kind, condition.Operator, condition.Label, condition.Severity, string(params), condition.TriggerDelayMS, condition.ClearDelayMS, condition.Deadband, index)
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
	defaultInt := func(value any, fallback int) int {
		if parsed := parseSnapshotInt(value); parsed > 0 {
			return parsed
		}
		return fallback
	}
	result := make([]SnapshotRelationalConfigRecord, 0)
	for _, connection := range connections {
		if connection.Type != "relational" {
			continue
		}
		port := 5432
		defaultCharset := "utf8mb4"
		if parsed := parseSnapshotInt(connection.Config["port"]); parsed > 0 {
			port = parsed
		}
		record := SnapshotRelationalConfigRecord{
			ConnectionID:     connection.ID,
			DBType:           parseSnapshotString(connection.Config["dbType"], "postgresql"),
			Host:             parseSnapshotString(connection.Config["host"], ""),
			Port:             port,
			Database:         parseSnapshotString(connection.Config["database"], ""),
			Username:         parseSnapshotString(connection.Config["username"], ""),
			Schema:           parseSnapshotOptionalString(connection.Config["schema"]),
			Charset:          &defaultCharset,
			Timezone:         parseSnapshotOptionalString(connection.Config["timezone"]),
			SSL:              parseSnapshotBool(connection.Config["ssl"]),
			SSLConfig:        parseSnapshotObject(connection.Config["sslConfig"]),
			PoolMin:          defaultInt(connection.Config["poolMin"], 2),
			PoolMax:          defaultInt(connection.Config["poolMax"], 10),
			AcquireTimeoutMS: defaultInt(connection.Config["acquireTimeoutMs"], 60000),
			IdleTimeoutMS:    defaultInt(connection.Config["idleTimeoutMs"], 30000),
			QueryTimeoutMS:   defaultInt(connection.Config["queryTimeoutMs"], 60000),
			Options:          parseSnapshotObject(connection.Config["options"]),
		}
		if charset := parseSnapshotOptionalString(connection.Config["charset"]); charset != nil {
			record.Charset = charset
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

func cloneSnapshotStringMap(value map[string]string) map[string]string {
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
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
