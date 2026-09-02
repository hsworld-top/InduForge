// Package model 定义 RuntimeEngine V1 的强类型配置与项目 Artifact 领域模型。
package model

import "encoding/json"

// MaxAlarmItems 是 runtime-project-artifact.v1 冻结的 alarmItems 上限。
// 它同时约束 schema、领域校验和 alarm sweep 的候选集合，避免恢复扫描因
// 配置规模失去确定的内存与事务边界。
const MaxAlarmItems = 4096

type Ownership struct {
	OwnerID string `json:"ownerId"`
	Epoch   int64  `json:"epoch"`
}

type ArtifactRef struct {
	ArtifactID       string `json:"artifactId"`
	ArtifactRevision int64  `json:"artifactRevision"`
	ArtifactDigest   string `json:"artifactDigest"`
}

// WALCapacity 是 Collector Binding 中会影响 Engine 数据缺口处理的容量边界。
type WALCapacity struct {
	MaxBytes               int64  `json:"maxBytes"`
	HighWatermarkBytes     int64  `json:"highWatermarkBytes"`
	DiagnosticReserveBytes int64  `json:"diagnosticReserveBytes"`
	DataGapPolicy          string `json:"dataGapPolicy"`
}

type ArtifactMount struct {
	Source       string `json:"source"`
	MountPath    string `json:"mountPath"`
	ArtifactFile string `json:"artifactFile"`
	ReadOnly     bool   `json:"readOnly"`
}

type RoleAssignment struct {
	Role      string    `json:"role"`
	Ownership Ownership `json:"ownership"`
}

type ProducerAssignment struct {
	ProducerType string `json:"producerType"`
	// ManualID 固定为受控 Runtime API 的 producer 身份；它不是用户可选来源。
	ManualID          string                    `json:"manualId,omitempty"`
	CollectorID       string                    `json:"collectorId,omitempty"`
	CollectorArtifact *CollectorArtifactBinding `json:"collectorArtifact,omitempty"`
	ComputeID         string                    `json:"computeId,omitempty"`
	Role              string                    `json:"role,omitempty"`
	Ownership         Ownership                 `json:"ownership"`
}

// CollectorArtifactBinding 是与 collector owner 绑定的只读逻辑采集 Artifact。
// 文件路径相对 Engine 顶层 release-pvc 挂载根，避免额外网络或可写来源。
type CollectorArtifactBinding struct {
	Artifact     ArtifactRef `json:"artifact"`
	ArtifactFile string      `json:"artifactFile"`
}

type CollectorConnection struct {
	ConnectionID string `json:"connectionId"`
}

type PointMapping struct {
	DatapointID  string `json:"datapointId"`
	ConnectionID string `json:"connectionId"`
	VariableID   string `json:"variableId"`
	DataType     string `json:"dataType"`
	Enabled      bool   `json:"enabled"`
}

// CollectorArtifact 仅保留 Engine ingress 审计所需的受信字段；原始字节仍在 loader 中完成 digest 与 Schema 校验。
type CollectorArtifact struct {
	SchemaVersion    string                `json:"schemaVersion"`
	ArtifactID       string                `json:"artifactId"`
	ArtifactRevision int64                 `json:"artifactRevision"`
	ProjectID        string                `json:"projectId"`
	Connections      []CollectorConnection `json:"connections"`
	PointMappings    []PointMapping        `json:"pointMappings"`
}

type Consumer struct {
	Role                string  `json:"role"`
	ConsumerKey         string  `json:"consumerKey"`
	Stream              string  `json:"stream"`
	DurableName         string  `json:"durableName"`
	FilterSubject       string  `json:"filterSubject"`
	AckPolicy           string  `json:"ackPolicy"`
	AckWaitMS           int64   `json:"ackWaitMs"`
	MaxDeliver          int     `json:"maxDeliver"`
	BackoffMS           []int64 `json:"backoffMs"`
	MaxAckPending       int     `json:"maxAckPending"`
	MaxWaiting          int     `json:"maxWaiting"`
	MaxRequestBatch     int     `json:"maxRequestBatch"`
	MaxRequestExpiresMS int64   `json:"maxRequestExpiresMs"`
	MaxRequestMaxBytes  int     `json:"maxRequestMaxBytes"`
	DeadLetterSubject   string  `json:"deadLetterSubject"`
}

type JetStream struct {
	ServerResourceRef   string `json:"serverResourceRef"`
	CredentialSecretRef string `json:"credentialSecretRef"`
	DataRawStream       string `json:"dataRawStream"`
	DataDerivedStream   string `json:"dataDerivedStream"`
	EventStream         string `json:"eventStream"`
	// CommandStream 隔离人工计算命令，命令不复用点位数据流的保留与消费语义。
	CommandStream    string     `json:"commandStream"`
	DeadLetterStream string     `json:"deadLetterStream"`
	Consumers        []Consumer `json:"consumers"`
}

// ComputeSandbox 仅包含受信资源和 Secret 引用；resolver 才能得到 endpoint/credential 值。
type ComputeSandbox struct {
	ServerResourceRef   string `json:"serverResourceRef"`
	CredentialSecretRef string `json:"credentialSecretRef"`
}

type StateStore struct {
	Engine                 string `json:"engine"`
	DSNSecretRef           string `json:"dsnSecretRef"`
	CASRequired            bool   `json:"casRequired"`
	OutboxRequired         bool   `json:"outboxRequired"`
	ProcessedEventRequired bool   `json:"processedEventRequired"`
	CheckpointRequired     bool   `json:"checkpointRequired"`
}

type EngineConfig struct {
	SchemaVersion       string               `json:"schemaVersion"`
	SiteID              string               `json:"siteId"`
	NodeID              string               `json:"nodeId,omitempty"`
	ProjectID           string               `json:"projectId"`
	ExecutionForm       string               `json:"executionForm"`
	DeploymentID        string               `json:"deploymentId"`
	AccountID           string               `json:"accountId"`
	ProjectArtifact     ArtifactRef          `json:"projectArtifact"`
	ArtifactMount       ArtifactMount        `json:"artifactMount"`
	Roles               []string             `json:"roles"`
	RoleAssignments     []RoleAssignment     `json:"roleAssignments"`
	ProducerAssignments []ProducerAssignment `json:"producerAssignments"`
	JetStream           JetStream            `json:"jetStream"`
	ComputeSandbox      *ComputeSandbox      `json:"computeSandbox,omitempty"`
	StateStore          StateStore           `json:"stateStore"`
}

type DataPoint struct {
	ID                string             `json:"id"`
	Path              string             `json:"path"`
	Name              string             `json:"name"`
	DataType          string             `json:"dataType"`
	SourceType        string             `json:"sourceType"`
	SourceID          *string            `json:"sourceId"`
	SourceConfig      json.RawMessage    `json:"sourceConfig"`
	RuntimePermission RuntimePermissions `json:"runtimePermissions"`
	RefreshMode       string             `json:"refreshMode"`
	RefreshIntervalMS *int64             `json:"refreshIntervalMs"`
	Status            string             `json:"status"`
	DisplayOrder      int64              `json:"displayOrder"`
	Unit              *string            `json:"unit"`
	PrecisionNum      *int64             `json:"precisionNum"`
	DefaultValue      json.RawMessage    `json:"defaultValue"`
	Tags              []json.RawMessage  `json:"tags"`
	AttributeDefaults map[string]string  `json:"attributeDefaults"`
}

type WriteGrant struct {
	AllowRoles []string `json:"allowRoles"`
	DenyRoles  []string `json:"denyRoles"`
	Inherit    bool     `json:"inherit"`
}

type RuntimePermissions struct {
	Write WriteGrant `json:"write"`
}

type Dependency struct {
	PackageName string `json:"packageName"`
	ImportName  string `json:"importName"`
	Version     string `json:"version"`
	Language    string `json:"language"`
}

type Input struct {
	Alias       string `json:"alias"`
	DatapointID string `json:"datapointId"`
	Path        string `json:"path"`
	DataType    string `json:"dataType"`
}

type Output struct {
	DatapointID  string          `json:"datapointId"`
	OutputKey    string          `json:"outputKey"`
	Path         string          `json:"path"`
	Name         string          `json:"name"`
	Description  *string         `json:"description"`
	DataType     string          `json:"dataType"`
	Unit         *string         `json:"unit"`
	Precision    *int64          `json:"precisionNum"`
	SortOrder    int64           `json:"sortOrder"`
	NullPolicy   string          `json:"nullPolicy"`
	DefaultValue json.RawMessage `json:"defaultValue"`
}

type Schedule struct {
	Kind        string  `json:"kind"`
	Every       *int64  `json:"every"`
	Unit        *string `json:"unit"`
	Timezone    *string `json:"timezone"`
	Time        *string `json:"time"`
	Weekdays    []int   `json:"weekdays"`
	DayRule     *string `json:"dayRule"`
	DayOfMonth  *int    `json:"dayOfMonth"`
	WeekOfMonth *int    `json:"weekOfMonth"`
	Weekday     *int    `json:"weekday"`
	Month       *int    `json:"month"`
	StartAt     *string `json:"startAt"`
	EndAt       *string `json:"endAt"`
	MaxRuns     *int    `json:"maxRuns"`
}

type Trigger struct {
	Kind        string    `json:"kind"`
	Schedule    *Schedule `json:"schedule,omitempty"`
	DatapointID string    `json:"datapointId,omitempty"`
	Path        string    `json:"path,omitempty"`
	DataType    string    `json:"dataType,omitempty"`
	Mode        string    `json:"mode,omitempty"`
	// Deadband keeps the original JSON number, avoiding float64 loss for
	// uint64-scale points and high-precision decimal values.
	Deadband   json.RawMessage `json:"deadband,omitempty"`
	DebounceMS *int64          `json:"debounceMs,omitempty"`
	Expression string          `json:"expression,omitempty"`
	Phases     []string        `json:"phases,omitempty"`
	Variables  []Input         `json:"variables,omitempty"`
}

type ComputeUnit struct {
	ID           string       `json:"id"`
	Revision     int64        `json:"revision"`
	Name         string       `json:"name"`
	Description  *string      `json:"description"`
	Language     string       `json:"language"`
	ScriptCode   string       `json:"scriptCode"`
	Enabled      bool         `json:"enabled"`
	TimeoutMS    int64        `json:"timeoutMs"`
	Dependencies []Dependency `json:"dependencies"`
	Inputs       []Input      `json:"inputs"`
	Outputs      []Output     `json:"outputs"`
	Trigger      Trigger      `json:"trigger"`
}

type AlarmCondition struct {
	ID             string          `json:"id"`
	Kind           string          `json:"kind"`
	Operator       string          `json:"operator"`
	Params         json.RawMessage `json:"params"`
	Severity       string          `json:"severity"`
	TriggerDelayMS int64           `json:"triggerDelayMs"`
	ClearDelayMS   int64           `json:"clearDelayMs"`
	// Deadband 保留 Artifact JSON decimal 原文；报警侧以 big.Rat 比较，不能经 float64。
	Deadband json.RawMessage `json:"deadband"`
}

type AlarmItem struct {
	ID                string           `json:"id"`
	Revision          int64            `json:"revision"`
	DisplayName       string           `json:"displayName"`
	Enabled           bool             `json:"enabled"`
	Mode              string           `json:"mode"`
	EvaluationMode    string           `json:"evaluationMode"`
	Inputs            []Input          `json:"inputs"`
	DerivedExpression *string          `json:"derivedExpression"`
	Conditions        []AlarmCondition `json:"conditions"`
}

type ProjectArtifact struct {
	SchemaVersion          string        `json:"schemaVersion"`
	ProjectArtifactVersion string        `json:"projectArtifactVersion"`
	ProjectID              string        `json:"projectId"`
	GeneratedAt            string        `json:"generatedAt"`
	DataPoints             []DataPoint   `json:"dataPoints"`
	ComputeUnits           []ComputeUnit `json:"computeUnits"`
	AlarmItems             []AlarmItem   `json:"alarmItems"`
}
