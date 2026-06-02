package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
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
	Order          int            `json:"order"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// SnapshotOpcuaNodeRecord 表示 artifact 使用的 OPC UA 变量投影。
type SnapshotOpcuaNodeRecord struct {
	ID            string   `json:"id"`
	ConnectionID  string   `json:"connectionId"`
	GroupID       *string  `json:"groupId,omitempty"`
	Name          string   `json:"name"`
	Code          string   `json:"code"`
	NodeID        string   `json:"nodeId"`
	DataType      string   `json:"dataType"`
	SamplingMS    int      `json:"samplingMs"`
	Deadband      *float64 `json:"deadband,omitempty"`
	AccessLevel   string   `json:"accessLevel"`
	DataPointPath *string  `json:"datapointPath,omitempty"`
}

// SnapshotModbusRegisterRecord 表示 artifact 使用的 Modbus 寄存器变量投影。
type SnapshotModbusRegisterRecord struct {
	ID              string  `json:"id"`
	ConnectionID    string  `json:"connectionId"`
	GroupID         *string `json:"groupId,omitempty"`
	Name            string  `json:"name"`
	Code            string  `json:"code"`
	UnitID          int     `json:"unitId"`
	Area            string  `json:"area"`
	Address         int     `json:"address"`
	AddressBase     string  `json:"addressBase"`
	ProtocolAddress int     `json:"protocolAddress"`
	Quantity        int     `json:"quantity"`
	DataType        string  `json:"dataType"`
	ByteOrder       string  `json:"byteOrder"`
	WordOrder       string  `json:"wordOrder"`
	BitIndex        *int    `json:"bitIndex,omitempty"`
	Scale           float64 `json:"scale"`
	Offset          float64 `json:"offset"`
	Unit            *string `json:"unit,omitempty"`
	PollIntervalMS  int     `json:"pollIntervalMs"`
	TimeoutMS       *int    `json:"timeoutMs,omitempty"`
	RetryCount      *int    `json:"retryCount,omitempty"`
	AccessLevel     string  `json:"accessLevel"`
	DataPointPath   *string `json:"datapointPath,omitempty"`
}

// SnapshotModbusReadPlanRecord 表示 artifact 使用的 Modbus 运行态读取计划。
type SnapshotModbusReadPlanRecord struct {
	ID             string   `json:"id"`
	ConnectionID   string   `json:"connectionId"`
	UnitID         int      `json:"unitId"`
	Area           string   `json:"area"`
	ProtocolStart  int      `json:"protocolStart"`
	Quantity       int      `json:"quantity"`
	PollIntervalMS int      `json:"pollIntervalMs"`
	RegisterIDs    []string `json:"registerIds"`
	RegisterCount  int      `json:"registerCount"`
}

// SnapshotS7ProfileRecord 表示 artifact 使用的 S7 PLC 档案。
type SnapshotS7ProfileRecord struct {
	ConnectionID         string         `json:"connectionId"`
	PlcFamily            string         `json:"plcFamily"`
	CommunicationMode    string         `json:"communicationMode"`
	Host                 string         `json:"host"`
	Port                 int            `json:"port"`
	Rack                 int            `json:"rack"`
	Slot                 int            `json:"slot"`
	PollIntervalMS       int            `json:"pollIntervalMs"`
	PDUSize              *int           `json:"pduSize,omitempty"`
	MaxReadBytes         *int           `json:"maxReadBytes,omitempty"`
	MaxGapBytes          int            `json:"maxGapBytes"`
	OptimizedBlockAccess bool           `json:"optimizedBlockAccess"`
	SupportedAreas       []string       `json:"supportedAreas"`
	Options              map[string]any `json:"options"`
}

// SnapshotS7VariableRecord 表示 artifact 使用的 S7 变量投影。
type SnapshotS7VariableRecord struct {
	ID                string  `json:"id"`
	ConnectionID      string  `json:"connectionId"`
	GroupID           *string `json:"groupId,omitempty"`
	Name              string  `json:"name"`
	Code              string  `json:"code"`
	AddressText       string  `json:"addressText"`
	NormalizedAddress string  `json:"normalizedAddress"`
	Area              string  `json:"area"`
	DBNumber          *int    `json:"dbNumber,omitempty"`
	ByteOffset        int     `json:"byteOffset"`
	BitOffset         *int    `json:"bitOffset,omitempty"`
	ReadLength        int     `json:"readLength"`
	DataType          string  `json:"dataType"`
	ByteOrder         string  `json:"byteOrder"`
	WordOrder         string  `json:"wordOrder"`
	Scale             float64 `json:"scale"`
	Offset            float64 `json:"offset"`
	Unit              *string `json:"unit,omitempty"`
	PollIntervalMS    int     `json:"pollIntervalMs"`
	DataPointPath     *string `json:"datapointPath,omitempty"`
}

// SnapshotS7ReadPlanRecord 表示 artifact 使用的 S7 运行态读取计划。
type SnapshotS7ReadPlanRecord struct {
	ID             string   `json:"id"`
	ConnectionID   string   `json:"connectionId"`
	Area           string   `json:"area"`
	DBNumber       *int     `json:"dbNumber,omitempty"`
	StartByte      int      `json:"startByte"`
	EndByte        int      `json:"endByte"`
	ReadLength     int      `json:"readLength"`
	PollIntervalMS int      `json:"pollIntervalMs"`
	VariableIDs    []string `json:"variableIds"`
	VariableCount  int      `json:"variableCount"`
	MaxGapBytes    int      `json:"maxGapBytes"`
	ReadMode       string   `json:"readMode"`
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
	OpcuaNodes        []SnapshotOpcuaNodeRecord        `json:"opcuaNodes"`
	ModbusRegisters   []SnapshotModbusRegisterRecord   `json:"modbusRegisters"`
	S7Profiles        []SnapshotS7ProfileRecord        `json:"s7Profiles"`
	S7Variables       []SnapshotS7VariableRecord       `json:"s7Variables"`
	DataPoints        []DataPointRecord                `json:"datapoints"`
	ComputeUnits      []ComputeUnitRecord              `json:"computeUnits"`
	AlarmRules        []AlarmRuleRecord                `json:"alarmRules"`
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

// ArtifactAlarmRuleRecord 表示产物层报警规则对象。
type ArtifactAlarmRuleRecord struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	TargetDataPointID *string        `json:"targetDatapointId,omitempty"`
	TargetPath        string         `json:"targetPath"`
	RuleType          string         `json:"ruleType"`
	Condition         map[string]any `json:"condition"`
	Severity          string         `json:"severity"`
	Hysteresis        *float64       `json:"hysteresis,omitempty"`
	SampleWindowMS    *int           `json:"sampleWindowMs,omitempty"`
	Contract          map[string]any `json:"contract"`
	IsEnabled         bool           `json:"isEnabled"`
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
	ID        string                         `json:"id"`
	Name      string                         `json:"name"`
	Type      string                         `json:"type"`
	Status    string                         `json:"status"`
	Config    map[string]any                 `json:"config"`
	Nodes     []SnapshotOpcuaNodeRecord      `json:"nodes,omitempty"`
	Registers []SnapshotModbusRegisterRecord `json:"registers,omitempty"`
	Profile   *SnapshotS7ProfileRecord       `json:"profile,omitempty"`
	Variables []SnapshotS7VariableRecord     `json:"variables,omitempty"`
	ReadPlans any                            `json:"readPlans,omitempty"`
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
	OPCUA     []ArtifactProtocolRecord `json:"opcua"`
	S7        []ArtifactProtocolRecord `json:"s7"`
	Modbus    []ArtifactProtocolRecord `json:"modbus"`
	TDengine  []ArtifactProtocolRecord `json:"tdengine"`
}

// ArtifactBuiltinStoresPayload 表示产物层 IF 内置运行库契约。
type ArtifactBuiltinStoresPayload struct {
	Relations      []ArtifactBuiltinRelationStore   `json:"relations"`
	Timeseries     []ArtifactBuiltinTimeseriesStore `json:"timeseries"`
	RealtimeSpaces []ArtifactBuiltinRealtimeStore   `json:"realtimeSpaces"`
	MessageSpaces  []ArtifactBuiltinMessageStore    `json:"messageSpaces"`
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

type ArtifactBuiltinMessageStore struct {
	ID          string   `json:"id"`
	RuntimeKey  string   `json:"runtimeKey"`
	Name        string   `json:"name"`
	TopicPrefix string   `json:"topicPrefix"`
	Topics      []string `json:"topics"`
	Variables   []string `json:"variables"`
	Bindings    []string `json:"bindings"`
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
	Alarms        []ArtifactAlarmRuleRecord    `json:"alarms"`
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
	mqttTagGroups, err := r.listMqttTagGroups(ctx, projectID)
	if err != nil {
		return nil, err
	}
	mqttTags, err := r.listMqttTags(ctx, projectID)
	if err != nil {
		return nil, err
	}
	opcuaNodes, err := r.listOpcuaNodes(ctx, projectID)
	if err != nil {
		return nil, err
	}
	modbusRegisters, err := r.listModbusRegisters(ctx, projectID)
	if err != nil {
		return nil, err
	}
	s7Profiles, err := r.listS7Profiles(ctx, projectID)
	if err != nil {
		return nil, err
	}
	s7Variables, err := r.listS7Variables(ctx, projectID)
	if err != nil {
		return nil, err
	}
	computeUnits, err := r.listComputeUnits(ctx, projectID)
	if err != nil {
		return nil, err
	}
	alarmRules, err := r.listAlarmRules(ctx, projectID)
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
		OpcuaNodes:        opcuaNodes,
		ModbusRegisters:   modbusRegisters,
		S7Profiles:        s7Profiles,
		S7Variables:       s7Variables,
		DataPoints:        datapoints,
		ComputeUnits:      computeUnits,
		AlarmRules:        alarmRules,
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
	if err := r.insertComputeUnits(ctx, tx, projectID, actorID, snapshot.ComputeUnits); err != nil {
		return err
	}
	if err := r.insertAlarmRules(ctx, tx, projectID, actorID, snapshot.AlarmRules); err != nil {
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

	alarmRules := make([]ArtifactAlarmRuleRecord, 0, len(snapshot.AlarmRules))
	for _, rule := range snapshot.AlarmRules {
		alarmRules = append(alarmRules, ArtifactAlarmRuleRecord{
			ID:                rule.ID,
			Name:              rule.Name,
			TargetDataPointID: rule.TargetDataPointID,
			TargetPath:        rule.TargetPath,
			RuleType:          rule.RuleType,
			Condition:         cloneSnapshotObject(rule.Condition),
			Severity:          rule.Severity,
			Hysteresis:        rule.Hysteresis,
			SampleWindowMS:    rule.SampleWindowMS,
			Contract:          cloneSnapshotObject(rule.Contract),
			IsEnabled:         rule.IsEnabled,
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
		OPCUA:     make([]ArtifactProtocolRecord, 0),
		S7:        make([]ArtifactProtocolRecord, 0),
		Modbus:    make([]ArtifactProtocolRecord, 0),
		TDengine:  make([]ArtifactProtocolRecord, 0),
	}
	builtinStores := buildBuiltinStores(snapshot.Connections)
	opcuaNodesByConnection := groupOpcuaNodesByConnection(snapshot.OpcuaNodes)
	modbusRegistersByConnection := groupModbusRegistersByConnection(snapshot.ModbusRegisters)
	modbusReadPlansByConnection := buildSnapshotModbusReadPlansByConnection(snapshot.ModbusRegisters)
	s7ProfilesByConnection := groupS7ProfilesByConnection(snapshot.S7Profiles)
	s7VariablesByConnection := groupS7VariablesByConnection(snapshot.S7Variables)
	s7ReadPlansByConnection := buildSnapshotS7ReadPlansByConnection(snapshot.S7Profiles, snapshot.S7Variables)
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
		case "opcua":
			protocolRecord.Nodes = append([]SnapshotOpcuaNodeRecord{}, opcuaNodesByConnection[connection.ID]...)
			protocols.OPCUA = append(protocols.OPCUA, protocolRecord)
		case "s7":
			if profile, ok := s7ProfilesByConnection[connection.ID]; ok {
				protocolRecord.Profile = &profile
			}
			protocolRecord.Variables = append([]SnapshotS7VariableRecord{}, s7VariablesByConnection[connection.ID]...)
			protocolRecord.ReadPlans = append([]SnapshotS7ReadPlanRecord{}, s7ReadPlansByConnection[connection.ID]...)
			protocols.S7 = append(protocols.S7, protocolRecord)
		case "modbus":
			protocolRecord.Registers = append([]SnapshotModbusRegisterRecord{}, modbusRegistersByConnection[connection.ID]...)
			protocolRecord.ReadPlans = append([]SnapshotModbusReadPlanRecord{}, modbusReadPlansByConnection[connection.ID]...)
			protocols.Modbus = append(protocols.Modbus, protocolRecord)
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
		Alarms:      alarmRules,
		Mqtt: ArtifactMqttPayload{
			Connections:   mqttConnections,
			Subscriptions: append([]SnapshotMqttSubscriptionRecord{}, snapshot.MqttSubscriptions...),
			TagGroups:     append([]SnapshotMqttTagGroupRecord{}, snapshot.MqttTagGroups...),
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
		case "builtin.message":
			payload.MessageSpaces = append(payload.MessageSpaces, ArtifactBuiltinMessageStore{
				ID:          connection.ID,
				RuntimeKey:  runtimeKey,
				Name:        connection.Name,
				TopicPrefix: runtimeKey,
				Topics:      []string{},
				Variables:   []string{},
				Bindings:    []string{},
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

func (r *ProjectSnapshotRepository) listAlarmRules(ctx context.Context, projectID string) ([]AlarmRuleRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, name, description, target_datapoint_id, target_path, rule_type, condition,
               severity, hysteresis, sample_window_ms, contract, is_enabled, created_at, updated_at
        FROM data_alarm_rules
        WHERE project_id = $1
        ORDER BY created_at ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询快照报警规则失败", err)
	}
	defer rows.Close()

	result := make([]AlarmRuleRecord, 0)
	for rows.Next() {
		record, scanErr := scanAlarmRule(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照报警规则失败", err)
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
        SELECT id, project_id, connection_id, name, topic, qos, description, message_retention, created_at, updated_at
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

func (r *ProjectSnapshotRepository) listOpcuaNodes(ctx context.Context, projectID string) ([]SnapshotOpcuaNodeRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT n.id, n.connection_id, n.group_id, n.name, n.code, n.node_id, n.data_type,
               n.sampling_ms, n.deadband, n.access_level, dp.path
        FROM data_opcua_nodes n
        LEFT JOIN data_points dp
          ON dp.project_id = n.project_id
         AND dp.source_type = 'opcua.node'
         AND dp.source_id = n.id
        WHERE n.project_id = $1
          AND n.status = 'active'
        ORDER BY n.connection_id ASC, n.sort_order ASC, n.created_at ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询快照 OPC UA 变量失败", err)
	}
	defer rows.Close()

	result := make([]SnapshotOpcuaNodeRecord, 0)
	for rows.Next() {
		record := SnapshotOpcuaNodeRecord{}
		if err := rows.Scan(
			&record.ID,
			&record.ConnectionID,
			&record.GroupID,
			&record.Name,
			&record.Code,
			&record.NodeID,
			&record.DataType,
			&record.SamplingMS,
			&record.Deadband,
			&record.AccessLevel,
			&record.DataPointPath,
		); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照 OPC UA 变量失败", err)
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照 OPC UA 变量失败", err)
	}
	return result, nil
}

func (r *ProjectSnapshotRepository) listModbusRegisters(ctx context.Context, projectID string) ([]SnapshotModbusRegisterRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT r.id, r.connection_id, r.group_id, r.name, r.code, r.unit_id, r.area,
               r.address, r.address_base, r.protocol_address, r.quantity, r.data_type,
               r.byte_order, r.word_order, r.bit_index, r.scale, r.offset_value, r.unit,
               r.poll_interval_ms, r.timeout_ms, r.retry_count, r.access_level, dp.path
        FROM data_modbus_registers r
        LEFT JOIN data_points dp
          ON dp.project_id = r.project_id
         AND dp.source_type = 'modbus.register'
         AND dp.source_id = r.id
        WHERE r.project_id = $1
          AND r.status = 'active'
        ORDER BY r.connection_id ASC, r.unit_id ASC, r.area ASC, r.protocol_address ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询快照 Modbus 变量失败", err)
	}
	defer rows.Close()

	result := make([]SnapshotModbusRegisterRecord, 0)
	for rows.Next() {
		record := SnapshotModbusRegisterRecord{}
		if err := rows.Scan(
			&record.ID,
			&record.ConnectionID,
			&record.GroupID,
			&record.Name,
			&record.Code,
			&record.UnitID,
			&record.Area,
			&record.Address,
			&record.AddressBase,
			&record.ProtocolAddress,
			&record.Quantity,
			&record.DataType,
			&record.ByteOrder,
			&record.WordOrder,
			&record.BitIndex,
			&record.Scale,
			&record.Offset,
			&record.Unit,
			&record.PollIntervalMS,
			&record.TimeoutMS,
			&record.RetryCount,
			&record.AccessLevel,
			&record.DataPointPath,
		); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照 Modbus 变量失败", err)
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照 Modbus 变量失败", err)
	}
	return result, nil
}

func (r *ProjectSnapshotRepository) listS7Profiles(ctx context.Context, projectID string) ([]SnapshotS7ProfileRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT connection_id, plc_family, communication_mode, host, port, rack, slot,
               poll_interval_ms, pdu_size, max_read_bytes, max_gap_bytes,
               optimized_block_access, supported_areas, options
        FROM data_s7_plc_profiles
        WHERE project_id = $1
        ORDER BY connection_id ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询快照 S7 PLC 档案失败", err)
	}
	defer rows.Close()

	result := make([]SnapshotS7ProfileRecord, 0)
	for rows.Next() {
		record := SnapshotS7ProfileRecord{}
		var supportedAreasBytes []byte
		var optionsBytes []byte
		if err := rows.Scan(
			&record.ConnectionID,
			&record.PlcFamily,
			&record.CommunicationMode,
			&record.Host,
			&record.Port,
			&record.Rack,
			&record.Slot,
			&record.PollIntervalMS,
			&record.PDUSize,
			&record.MaxReadBytes,
			&record.MaxGapBytes,
			&record.OptimizedBlockAccess,
			&supportedAreasBytes,
			&optionsBytes,
		); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照 S7 PLC 档案失败", err)
		}
		record.SupportedAreas = mustJSONStringArray(supportedAreasBytes)
		record.Options = mustJSONObject(optionsBytes)
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照 S7 PLC 档案失败", err)
	}
	return result, nil
}

func (r *ProjectSnapshotRepository) listS7Variables(ctx context.Context, projectID string) ([]SnapshotS7VariableRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT v.id, v.connection_id, v.group_id, v.name, v.code, v.address_text,
               v.normalized_address, v.area, v.db_number, v.byte_offset, v.bit_offset,
               v.read_length, v.data_type, v.byte_order, v.word_order, v.scale,
               v.offset_value, v.unit, v.poll_interval_ms, dp.path
        FROM data_s7_variables v
        LEFT JOIN data_points dp
          ON dp.project_id = v.project_id
         AND dp.source_type = 's7.variable'
         AND dp.source_id = v.id
        WHERE v.project_id = $1
          AND v.status = 'active'
        ORDER BY v.connection_id ASC, v.area ASC, v.db_number ASC, v.byte_offset ASC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询快照 S7 变量失败", err)
	}
	defer rows.Close()

	result := make([]SnapshotS7VariableRecord, 0)
	for rows.Next() {
		record := SnapshotS7VariableRecord{}
		if err := rows.Scan(
			&record.ID,
			&record.ConnectionID,
			&record.GroupID,
			&record.Name,
			&record.Code,
			&record.AddressText,
			&record.NormalizedAddress,
			&record.Area,
			&record.DBNumber,
			&record.ByteOffset,
			&record.BitOffset,
			&record.ReadLength,
			&record.DataType,
			&record.ByteOrder,
			&record.WordOrder,
			&record.Scale,
			&record.Offset,
			&record.Unit,
			&record.PollIntervalMS,
			&record.DataPointPath,
		); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取快照 S7 变量失败", err)
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历快照 S7 变量失败", err)
	}
	return result, nil
}

func (r *ProjectSnapshotRepository) deleteProjectSnapshot(ctx context.Context, tx pgx.Tx, projectID string) error {
	for _, sqlText := range []string{
		`DELETE FROM data_alarm_rules WHERE project_id = $1`,
		`DELETE FROM data_compute_units WHERE project_id = $1`,
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
		if _, err := tx.Exec(ctx, `
            INSERT INTO data_mqtt_subscriptions (
                id, project_id, connection_id, name, topic, qos, description,
                message_retention, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        `, subscription.ID, projectID, subscription.ConnectionID, subscription.Name, subscription.Topic, subscription.QOS, subscription.Description, subscription.MessageRetention, actorID, actorID, createdAt, updatedAt); err != nil {
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
                default_value, unit, transform, validation, display_order, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14::jsonb, $15, $16, $17, $18, $19)
        `, tag.ID, projectID, tag.SubscriptionID, tag.GroupID, tag.Name, tag.Code, tag.Description, tag.DataType, tag.ParseType, tag.ParseRule, tag.DefaultValue, tag.Unit, tag.Transform, nullableJSONString(validationBytes), tag.Order, actorID, actorID, createdAt, updatedAt); err != nil {
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

func (r *ProjectSnapshotRepository) insertAlarmRules(ctx context.Context, tx pgx.Tx, projectID, actorID string, rules []AlarmRuleRecord) error {
	for _, rule := range rules {
		conditionBytes, err := marshalSnapshotObject(rule.Condition)
		if err != nil {
			return err
		}
		contractBytes, err := marshalSnapshotObject(rule.Contract)
		if err != nil {
			return err
		}
		createdAt := coalesceTime(rule.CreatedAt)
		updatedAt := coalesceTime(rule.UpdatedAt)
		if _, err := tx.Exec(ctx, `
            INSERT INTO data_alarm_rules (
                id, project_id, name, description, target_datapoint_id, target_path, rule_type, condition,
                severity, hysteresis, sample_window_ms, contract, is_enabled, created_by, updated_by, created_at, updated_at
            )
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11, $12::jsonb, $13, $14, $15, $16, $17)
        `, rule.ID, projectID, rule.Name, rule.Description, rule.TargetDataPointID, rule.TargetPath, rule.RuleType, string(conditionBytes),
			rule.Severity, rule.Hysteresis, rule.SampleWindowMS, string(contractBytes), rule.IsEnabled, actorID, actorID, createdAt, updatedAt); err != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入快照报警规则失败", err)
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

func groupOpcuaNodesByConnection(nodes []SnapshotOpcuaNodeRecord) map[string][]SnapshotOpcuaNodeRecord {
	result := make(map[string][]SnapshotOpcuaNodeRecord)
	for _, node := range nodes {
		result[node.ConnectionID] = append(result[node.ConnectionID], node)
	}
	return result
}

func groupModbusRegistersByConnection(registers []SnapshotModbusRegisterRecord) map[string][]SnapshotModbusRegisterRecord {
	result := make(map[string][]SnapshotModbusRegisterRecord)
	for _, register := range registers {
		result[register.ConnectionID] = append(result[register.ConnectionID], register)
	}
	return result
}

func groupS7ProfilesByConnection(profiles []SnapshotS7ProfileRecord) map[string]SnapshotS7ProfileRecord {
	result := make(map[string]SnapshotS7ProfileRecord, len(profiles))
	for _, profile := range profiles {
		result[profile.ConnectionID] = profile
	}
	return result
}

func groupS7VariablesByConnection(variables []SnapshotS7VariableRecord) map[string][]SnapshotS7VariableRecord {
	result := make(map[string][]SnapshotS7VariableRecord)
	for _, variable := range variables {
		result[variable.ConnectionID] = append(result[variable.ConnectionID], variable)
	}
	return result
}

func buildSnapshotModbusReadPlansByConnection(registers []SnapshotModbusRegisterRecord) map[string][]SnapshotModbusReadPlanRecord {
	grouped := map[string][]SnapshotModbusRegisterRecord{}
	for _, register := range registers {
		key := strings.Join([]string{register.ConnectionID, intToSnapshotString(register.UnitID), register.Area, intToSnapshotString(register.PollIntervalMS)}, "|")
		grouped[key] = append(grouped[key], register)
	}
	result := map[string][]SnapshotModbusReadPlanRecord{}
	for _, items := range grouped {
		sort.Slice(items, func(i, j int) bool {
			return items[i].ProtocolAddress < items[j].ProtocolAddress
		})
		maxQuantity := snapshotModbusMaxReadQuantity(items[0].Area)
		var current *SnapshotModbusReadPlanRecord
		for _, item := range items {
			start := item.ProtocolAddress
			end := item.ProtocolAddress + item.Quantity - 1
			if current == nil || start > current.ProtocolStart+maxQuantity-1 || start > current.ProtocolStart+current.Quantity+1 {
				plan := SnapshotModbusReadPlanRecord{
					ConnectionID:   item.ConnectionID,
					UnitID:         item.UnitID,
					Area:           item.Area,
					ProtocolStart:  start,
					Quantity:       item.Quantity,
					PollIntervalMS: item.PollIntervalMS,
					RegisterIDs:    []string{item.ID},
					RegisterCount:  1,
				}
				current = &plan
				result[item.ConnectionID] = append(result[item.ConnectionID], plan)
				continue
			}
			currentEnd := current.ProtocolStart + current.Quantity - 1
			if end > currentEnd {
				current.Quantity = end - current.ProtocolStart + 1
			}
			current.RegisterIDs = append(current.RegisterIDs, item.ID)
			current.RegisterCount++
			plans := result[item.ConnectionID]
			plans[len(plans)-1] = *current
			result[item.ConnectionID] = plans
		}
	}
	for connectionID, plans := range result {
		sort.Slice(plans, func(i, j int) bool {
			if plans[i].UnitID != plans[j].UnitID {
				return plans[i].UnitID < plans[j].UnitID
			}
			if plans[i].Area != plans[j].Area {
				return plans[i].Area < plans[j].Area
			}
			return plans[i].ProtocolStart < plans[j].ProtocolStart
		})
		for index := range plans {
			plans[index].ID = "modbus-read-plan-" + intToSnapshotString(index+1)
		}
		result[connectionID] = plans
	}
	return result
}

func buildSnapshotS7ReadPlansByConnection(profiles []SnapshotS7ProfileRecord, variables []SnapshotS7VariableRecord) map[string][]SnapshotS7ReadPlanRecord {
	profilesByConnection := groupS7ProfilesByConnection(profiles)
	grouped := map[string][]SnapshotS7VariableRecord{}
	for _, variable := range variables {
		key := strings.Join([]string{variable.ConnectionID, variable.Area, snapshotOptionalIntKey(variable.DBNumber), intToSnapshotString(variable.PollIntervalMS), snapshotS7ReadMode(variable)}, "|")
		grouped[key] = append(grouped[key], variable)
	}
	result := map[string][]SnapshotS7ReadPlanRecord{}
	for _, items := range grouped {
		sort.Slice(items, func(i, j int) bool {
			return items[i].ByteOffset < items[j].ByteOffset
		})
		profile := profilesByConnection[items[0].ConnectionID]
		maxBytes := 240
		if profile.MaxReadBytes != nil && *profile.MaxReadBytes > 0 {
			maxBytes = *profile.MaxReadBytes
		}
		maxGap := profile.MaxGapBytes
		if maxGap < 0 {
			maxGap = 0
		}
		var current *SnapshotS7ReadPlanRecord
		for _, item := range items {
			start := item.ByteOffset
			end := item.ByteOffset + item.ReadLength - 1
			if current == nil || start > current.EndByte+maxGap || end-current.StartByte+1 > maxBytes {
				plan := SnapshotS7ReadPlanRecord{
					ConnectionID:   item.ConnectionID,
					Area:           item.Area,
					DBNumber:       item.DBNumber,
					StartByte:      start,
					EndByte:        end,
					ReadLength:     end - start + 1,
					PollIntervalMS: item.PollIntervalMS,
					VariableIDs:    []string{item.ID},
					VariableCount:  1,
					MaxGapBytes:    maxGap,
					ReadMode:       snapshotS7ReadMode(item),
				}
				current = &plan
				result[item.ConnectionID] = append(result[item.ConnectionID], plan)
				continue
			}
			if end > current.EndByte {
				current.EndByte = end
				current.ReadLength = current.EndByte - current.StartByte + 1
			}
			current.VariableIDs = append(current.VariableIDs, item.ID)
			current.VariableCount++
			plans := result[item.ConnectionID]
			plans[len(plans)-1] = *current
			result[item.ConnectionID] = plans
		}
	}
	for connectionID, plans := range result {
		sort.Slice(plans, func(i, j int) bool {
			if plans[i].Area != plans[j].Area {
				return plans[i].Area < plans[j].Area
			}
			if snapshotOptionalIntKey(plans[i].DBNumber) != snapshotOptionalIntKey(plans[j].DBNumber) {
				return snapshotOptionalIntKey(plans[i].DBNumber) < snapshotOptionalIntKey(plans[j].DBNumber)
			}
			return plans[i].StartByte < plans[j].StartByte
		})
		for index := range plans {
			plans[index].ID = "s7-read-plan-" + intToSnapshotString(index+1)
		}
		result[connectionID] = plans
	}
	return result
}

func snapshotS7ReadMode(variable SnapshotS7VariableRecord) string {
	if variable.Area == "DB" {
		return "db"
	}
	return "area"
}

func snapshotOptionalIntKey(value *int) string {
	if value == nil {
		return ""
	}
	return intToSnapshotString(*value)
}

func snapshotModbusMaxReadQuantity(area string) int {
	if area == "coil" || area == "discrete_input" {
		return 2000
	}
	return 125
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
