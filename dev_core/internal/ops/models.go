// Package ops 提供运行集群、Agent 接入和 Demo 工作负载的运维控制面。
package ops

import "time"

const (
	RoleRuntimeLinux      = "runtime_linux"
	RoleCollectorLinux    = "collector_linux"
	RoleCollectorWindows  = "collector_windows"
	WorkloadRoleCompute   = "compute"
	WorkloadRoleAlert     = "alert"
	WorkloadRoleCollector = "collector"
)

type RuntimeCluster struct {
	ID, TenantID, Name, Code, Description, Topology string
	DesiredStatus, ObservedStatus, ControllerStatus string
	Health                                          string
	NodeCount, OnlineNodeCount                      int
	Metadata                                        map[string]any
	CreatedAt, UpdatedAt                            time.Time
}

type Enrollment struct {
	ID, TenantID, RuntimeClusterID, Role, DisplayName, Status string
	ExpiresAt                                                 time.Time
	ClaimedAt, ApprovedAt, RejectedAt                         *time.Time
	ClaimedByNodeID                                           string
	ReportedHostName, MachineFingerprint, IPAddress           string
	Node                                                      *HostNode
	CreatedAt, UpdatedAt                                      time.Time
}

type HostNode struct {
	ID, TenantID, RuntimeClusterID, EnrollmentID, Role, DisplayName         string
	Hostname, OS, Architecture, AgentVersion, MachineFingerprint, IPAddress string
	DesiredStatus, ObservedStatus                                           string
	ResourceSummary, Capabilities                                           map[string]any
	LastHeartbeatAt, ApprovedAt                                             *time.Time
	CreatedAt, UpdatedAt                                                    time.Time
}

type ProjectDeployment struct {
	ID, TenantID, ProjectID, RuntimeClusterID, ProjectName, RuntimeClusterName, LatestRunID, Version, DeploymentMode string
	DesiredStatus, ObservedStatus, Health                                                                            string
	Progress                                                                                                         int
	Workloads                                                                                                        []Workload
	CreatedAt, UpdatedAt                                                                                             time.Time
}

type DeploymentRun struct {
	ID, TenantID, ProjectDeploymentID, Operation, DesiredStatus, ObservedStatus, Message string
	Progress                                                                             int
	StartedAt                                                                            time.Time
	CompletedAt                                                                          *time.Time
}

type Workload struct {
	ID                  string     `json:"id"`
	TenantID            string     `json:"tenantId"`
	ProjectDeploymentID string     `json:"projectDeploymentId"`
	HostNodeID          string     `json:"hostNodeId"`
	Role                string     `json:"role"`
	DesiredStatus       string     `json:"desiredStatus"`
	ObservedStatus      string     `json:"observedStatus"`
	LastMessage         string     `json:"lastMessage"`
	ReplicasDesired     int        `json:"replicasDesired"`
	ReplicasObserved    int        `json:"replicasObserved"`
	DesiredGeneration   int64      `json:"desiredGeneration"`
	ObservedGeneration  int64      `json:"observedGeneration"`
	LastOperation       string     `json:"lastOperation"`
	ObservedAt          *time.Time `json:"observedAt"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
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
	Role         string `json:"role"`
	OS           string `json:"os"`
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
}

type CreateClusterInput struct {
	Name, Code, Description, Topology string
	Metadata                          map[string]any
}
type CreateEnrollmentInput struct {
	RuntimeClusterID, Role, DisplayName string
	TTL                                 time.Duration
}
type ClaimEnrollmentInput struct {
	Code, DisplayName, Hostname, OS, Architecture, AgentVersion, MachineFingerprint, IPAddress string
	Capabilities                                                                               map[string]any
}

// WorkloadObservation 是 Agent 对一个实际 demo 进程的观测结果。workloadId 使重复心跳幂等。
type WorkloadObservation struct {
	WorkloadID         string `json:"workloadId"`
	ObservedStatus     string `json:"observedStatus"`
	Message            string `json:"message"`
	ReplicasObserved   int    `json:"replicasObserved"`
	ObservedGeneration int64  `json:"observedGeneration"`
}
type HeartbeatInput struct {
	ResourceSummary map[string]any
	AgentVersion    string
	Workloads       []WorkloadObservation
}
type AgentCommand struct {
	RunID           string `json:"runId"`
	DeploymentID    string `json:"deploymentId"`
	WorkloadID      string `json:"workloadId"`
	Role            string `json:"role"`
	DesiredStatus   string `json:"desiredStatus"`
	Operation       string `json:"operation"`
	Version         string `json:"version"`
	Generation      int64  `json:"generation"`
	ReplicasDesired int    `json:"replicasDesired"`
}
type CreateDeploymentInput struct {
	ProjectID, RuntimeClusterID, Version, DeploymentMode string
	Workloads                                            []WorkloadInput
}
type WorkloadInput struct {
	Role, HostNodeID string
	Replicas         int
}
