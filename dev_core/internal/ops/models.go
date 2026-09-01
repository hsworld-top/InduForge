// Package ops 提供物理节点、NodeAgent 接入和单节点工程部署控制面。
package ops

import "time"

const (
	PlatformLinux   = "linux"
	PlatformWindows = "windows"

	CapabilityProjectEntry = "project_entry"
	CapabilityDataRuntime  = "data_runtime"
	CapabilityCollector    = "collector"

	// ServiceBase 是内部稳定契约值；在 API、页面和运维文案中统一称为“基础引擎”。
	// 它始终包含工程 Gateway 与 Runtime API。计算、报警、采集均由工程内容推导，
	// 不能由前端任意勾选伪造工作负载。
	ServiceBaseName  = "基础引擎"
	ServiceBase      = "base"
	ServiceCompute   = "compute"
	ServiceAlarm     = "alarm"
	ServiceCollector = "collector"
)

type Enrollment struct {
	ID, TenantID, Platform, DisplayName, Status     string
	Capabilities                                    []string
	ExpiresAt                                       time.Time
	ClaimedAt, ApprovedAt, RejectedAt               *time.Time
	ClaimedByNodeID                                 string
	ReportedHostName, MachineFingerprint, IPAddress string
	Node                                            *Node
	CreatedAt, UpdatedAt                            time.Time
}

// Node 表示一台物理主机及其独立 NodeAgent 身份；同机部署中心时也不例外。
type Node struct {
	ID, TenantID, EnrollmentID, DisplayName             string
	Hostname, Platform, Architecture                    string
	AgentVersion, MachineFingerprint, IPAddress         string
	AssignedDeploymentID, AssignedProjectID             string
	AssignedProjectName                                 string
	EnvironmentID, EnvironmentName                      string
	EnvironmentNames                                    []string
	EnvironmentCount                                    int
	NodeKind, ClusterID                                 string
	ClusterRole, ClusterStatus, ClusterMessage          string
	ClusterDesiredAction                                string
	ClusterDesiredGeneration, ClusterObservedGeneration int64
	ClusterObservedAt                                   *time.Time
	DesiredStatus, ObservedStatus                       string
	Capabilities                                        []string
	ResourceSummary                                     map[string]any
	LastHeartbeatAt, ApprovedAt                         *time.Time
	CreatedAt, UpdatedAt                                time.Time
}

// RuntimeEnvironment 是工程运行的逻辑边界。用户只感知环境和物理节点，
// 不感知底层容器编排实现。
type RuntimeEnvironment struct {
	ID, TenantID, Name, Code, Status                 string
	DesiredStatus                                    string
	IsDefault                                        bool
	NodeCount, OnlineNodeCount                       int
	FoundationTotal, FoundationHealthy, ProjectCount int
	RecentChange, RecentBy                           string
	RecentAt                                         *time.Time
	CreatedAt, UpdatedAt                             time.Time
}

type RuntimeEnvironmentEvent struct {
	ID, EnvironmentID, EventType, Name, Target, Result string
	Message, OperatorName                              string
	CreatedAt                                          time.Time
}

type CreateRuntimeEnvironmentInput struct {
	Name string `json:"name"`
}

type UpdateRuntimeEnvironmentInput struct {
	Name string `json:"name"`
}

type DeleteRuntimeEnvironmentInput struct {
	ConfirmationName string `json:"confirmationName"`
}

type AddRuntimeEnvironmentNodesInput struct {
	NodeIDs []string `json:"nodeIds"`
}

type FoundationAssignment struct {
	ServiceType string `json:"serviceType"`
	NodeID      string `json:"nodeId"`
}

type DeployFoundationInput struct {
	Assignments []FoundationAssignment `json:"assignments"`
}

type MigrateFoundationInput struct {
	Assignments []FoundationAssignment `json:"assignments"`
}

type RuntimeEnvironmentService struct {
	ID, EnvironmentID, NodeID, NodeName, ServiceType string
	DesiredStatus, ObservedStatus, LastMessage       string
	DesiredGeneration, ObservedGeneration            int64
	Operation                                        string
	ObservedAt                                       *time.Time
	CreatedAt, UpdatedAt                             time.Time
}

type ProjectDeployment struct {
	ID, TenantID, ProjectID, ProjectName, EnvironmentID, EnvironmentName string
	ApplicationVersionID, Version, LatestRunID                           string
	Mode, DesiredStatus, ObservedStatus, Health                          string
	Progress                                                             int
	AccessPort                                                           int
	Services                                                             []DeploymentService
	CreatedAt, UpdatedAt                                                 time.Time
}

type DeploymentRun struct {
	ID, TenantID, ProjectDeploymentID, Operation, DesiredStatus, ObservedStatus, Message string
	Progress                                                                             int
	StartedAt                                                                            time.Time
	CompletedAt                                                                          *time.Time
}

type DeploymentService struct {
	ID, TenantID, ProjectDeploymentID, NodeID string     `json:"-"`
	ServiceType                               string     `json:"serviceType"`
	PublicPort                                *int       `json:"publicPort,omitempty"`
	DesiredStatus                             string     `json:"desiredStatus"`
	ObservedStatus                            string     `json:"observedStatus"`
	LastMessage                               string     `json:"lastMessage"`
	ReplicasDesired                           int        `json:"replicasDesired"`
	ReplicasObserved                          int        `json:"replicasObserved"`
	DesiredGeneration                         int64      `json:"desiredGeneration"`
	ObservedGeneration                        int64      `json:"observedGeneration"`
	LastOperation                             string     `json:"lastOperation"`
	Endpoint                                  string     `json:"endpoint"`
	ObservedAt                                *time.Time `json:"observedAt"`
	CreatedAt                                 time.Time  `json:"createdAt"`
	UpdatedAt                                 time.Time  `json:"updatedAt"`
}

type DeploymentRunEvent struct {
	ID              string    `json:"id"`
	DeploymentRunID string    `json:"deploymentRunId"`
	Stage           string    `json:"stage"`
	Message         string    `json:"message"`
	CreatedAt       time.Time `json:"createdAt"`
}

type NodePackage struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	Version      string `json:"version"`
	FileName     string `json:"fileName"`
	Available    bool   `json:"available"`
	Size         int64  `json:"size"`
}

type PageFilter struct {
	Page, PageSize int
	Search         string
	ProjectID      string
	Status         string
}

type CreateEnrollmentInput struct {
	Platform, DisplayName string
	Capabilities          []string
	TTL                   time.Duration
}
type ClaimEnrollmentInput struct {
	Code, DisplayName, Hostname, Platform, Architecture, AgentVersion, MachineFingerprint, IPAddress string
	Capabilities                                                                                     []string
}

// ServiceObservation 是 Agent 对单节点部署服务的观测。serviceId 使重试幂等。
type ServiceObservation struct {
	ServiceID          string `json:"serviceId"`
	ObservedStatus     string `json:"observedStatus"`
	Message            string `json:"message"`
	Endpoint           string `json:"endpoint"`
	ReplicasObserved   int    `json:"replicasObserved"`
	ObservedGeneration int64  `json:"observedGeneration"`
}
type HeartbeatInput struct {
	ResourceSummary  map[string]any       `json:"resourceSummary"`
	AgentVersion     string               `json:"agentVersion"`
	IPAddress        string               `json:"ipAddress"`
	Services         []ServiceObservation `json:"services"`
	ClusterState     *ClusterState        `json:"clusterState"`
	FoundationStates []FoundationState    `json:"foundationStates"`
}

// ClusterPlan 是中心生成的受限 K3s 期望状态。Token 仅在已认证的节点拉取时
// 临时派生并返回，不持久化到数据库，也不会通过管理端接口暴露。
type ClusterPlan struct {
	SchemaVersion string `json:"schemaVersion"`
	Generation    int64  `json:"generation"`
	ClusterID     string `json:"clusterId"`
	NodeID        string `json:"nodeId"`
	Operation     string `json:"operation"`
	K3sVersion    string `json:"k3sVersion"`
	ServerURL     string `json:"serverUrl,omitempty"`
	Token         string `json:"token"`
	NodeIP        string `json:"nodeIp"`
	DataDir       string `json:"dataDir"`
	APIPort       int    `json:"apiPort"`
	VXLANPort     int    `json:"vxlanPort"`
	KubeletPort   int    `json:"kubeletPort"`
}

type ClusterState struct {
	SchemaVersion string `json:"schemaVersion"`
	Generation    int64  `json:"generation"`
	ClusterID     string `json:"clusterId"`
	NodeID        string `json:"nodeId"`
	Operation     string `json:"operation"`
	K3sVersion    string `json:"k3sVersion"`
	NodeName      string `json:"nodeName"`
	NodeIP        string `json:"nodeIp"`
	ObservedState string `json:"observedState"`
	Message       string `json:"message"`
}

// ClusterUninstall 是中心下发给工作节点的固定卸载意图。中心节点永远不会收到
// 此命令；节点侧仍会再次校验本机落盘身份，不能借此清理任意目录。
type ClusterUninstall struct {
	ClusterID string `json:"clusterId"`
	NodeID    string `json:"nodeId"`
	PurgeData bool   `json:"purgeData"`
}

type FoundationPlan struct {
	SchemaVersion     string            `json:"schemaVersion"`
	Generation        int64             `json:"generation"`
	EnvironmentID     string            `json:"environmentId"`
	NodeID            string            `json:"nodeId"`
	Assignments       map[string]string `json:"assignments"`
	SourceAssignments map[string]string `json:"sourceAssignments,omitempty"`
	Claims            map[string]string `json:"claims,omitempty"`
	SourceClaims      map[string]string `json:"sourceClaims,omitempty"`
	Operation         string            `json:"operation,omitempty"`
}

type TimeSyncPlan struct {
	SchemaVersion  string   `json:"schemaVersion"`
	Generation     int64    `json:"generation"`
	NodeID         string   `json:"nodeId"`
	Role           string   `json:"role"`
	CenterIP       string   `json:"centerIp"`
	AllowedClients []string `json:"allowedClients,omitempty"`
}

type FoundationDelete struct {
	EnvironmentID string `json:"environmentId"`
	NodeID        string `json:"nodeId"`
}

type FoundationState struct {
	SchemaVersion string                   `json:"schemaVersion"`
	Generation    int64                    `json:"generation"`
	EnvironmentID string                   `json:"environmentId"`
	NodeID        string                   `json:"nodeId"`
	ObservedState string                   `json:"observedState"`
	Message       string                   `json:"message"`
	Services      []FoundationServiceState `json:"services"`
}

type FoundationServiceState struct {
	Workload string `json:"workload"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}
type AgentCommand struct {
	NodeID          string `json:"nodeId"`
	RunID           string `json:"runId"`
	DeploymentID    string `json:"deploymentId"`
	ReleaseID       string `json:"releaseId"`
	ServiceID       string `json:"serviceId"`
	ServiceType     string `json:"serviceType"`
	DesiredStatus   string `json:"desiredStatus"`
	Operation       string `json:"operation"`
	Version         string `json:"version"`
	ArchiveSHA256   string `json:"archiveSha256"`
	ManifestSHA256  string `json:"manifestSha256"`
	ChecksumsSHA256 string `json:"checksumsSha256"`
	SigningKeyID    string `json:"signingKeyId"`
	BindingRevision int    `json:"bindingRevision"`
	Generation      int64  `json:"generation"`
	ReplicasDesired int    `json:"replicasDesired"`
}

// DeploymentBinding 是节点专用、版本化的部署快照。Release 保持不可变，
// 节点端只接受与自身 Deployment 关联的 Binding。
type DeploymentBinding struct {
	ID, TenantID, ProjectDeploymentID, ProjectID, NodeID, ApplicationVersionID string
	Revision                                                                   int
	Content                                                                    []byte
}

// AgentRelease 只在服务端使用；对象键绝不能出现在 Agent 响应体或命令中。
type AgentRelease struct {
	ReleaseID, ArtifactKey, ArtifactHash, ManifestHash, ChecksumsHash, SigningKeyID string
	ArtifactSize                                                                    int64
}

// DevelopmentArtifact 是服务端构建的内部 __DEV__ 制品描述。它不属于版本管理，
// 但必须与正式 Release 使用同一下载、摘要和签名校验链路。
type DevelopmentArtifact struct {
	ReleaseID, Version, Bucket, ArtifactKey, ArtifactHash, ManifestHash, ChecksumsHash, SigningKeyID string
	ArtifactSize                                                                                     int64
	Manifest                                                                                         []byte
}
type CreateDeploymentInput struct {
	ProjectID            string            `json:"projectId"`
	EnvironmentID        string            `json:"environmentId"`
	ApplicationVersionID string            `json:"applicationVersionId"`
	Mode                 string            `json:"mode"`
	AccessPort           int               `json:"accessPort"`
	Placements           map[string]string `json:"placements"`
	// DevelopmentArtifact 仅由服务端注入，绝不能从公开请求体接收。
	DevelopmentArtifact *DevelopmentArtifact `json:"-"`
}
