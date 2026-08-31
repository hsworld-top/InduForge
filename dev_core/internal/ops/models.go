// Package ops 提供物理节点、NodeAgent 接入和单节点工程部署控制面。
package ops

import "time"

const (
	PlatformLinux   = "linux"
	PlatformWindows = "windows"

	CapabilityProjectEntry = "project_entry"
	CapabilityDataRuntime  = "data_runtime"
	CapabilityCollector    = "collector"

	ServiceProjectEntry = "project_entry"
	ServiceDataRuntime  = "data_runtime"
	ServiceCollector    = "collector"
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
	ID, TenantID, EnrollmentID, DisplayName     string
	Hostname, Platform, Architecture            string
	AgentVersion, MachineFingerprint, IPAddress string
	AssignedDeploymentID, AssignedProjectID     string
	AssignedProjectName                         string
	DesiredStatus, ObservedStatus               string
	Capabilities                                []string
	ResourceSummary                             map[string]any
	LastHeartbeatAt, ApprovedAt                 *time.Time
	CreatedAt, UpdatedAt                        time.Time
}

type ProjectDeployment struct {
	ID, TenantID, ProjectID, ProjectName, NodeID, NodeName string
	ApplicationVersionID, Version, LatestRunID             string
	DesiredStatus, ObservedStatus, Health                  string
	Progress                                               int
	Services                                               []DeploymentService
	CreatedAt, UpdatedAt                                   time.Time
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
	ResourceSummary map[string]any
	AgentVersion    string
	Services        []ServiceObservation
}
type AgentCommand struct {
	NodeID          string `json:"nodeId"`
	RunID           string `json:"runId"`
	DeploymentID    string `json:"deploymentId"`
	ServiceID       string `json:"serviceId"`
	ServiceType     string `json:"serviceType"`
	DesiredStatus   string `json:"desiredStatus"`
	Operation       string `json:"operation"`
	Version         string `json:"version"`
	Generation      int64  `json:"generation"`
	ReplicasDesired int    `json:"replicasDesired"`
}
type CreateDeploymentInput struct {
	ProjectID, NodeID, ApplicationVersionID string
	EnableCollector                         bool
}
