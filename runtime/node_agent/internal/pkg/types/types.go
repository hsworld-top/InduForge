package types

import (
	"context"
	"time"
)

// NodeInfo 节点信息
type NodeInfo struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Version      string    `json:"version"`
	Mode         string    `json:"mode"`
	ExecutorType string    `json:"executorType"`
	WorkDir      string    `json:"workDir"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// RuntimeStatus 运行时状态
type RuntimeStatus struct {
	ProjectID   string                 `json:"projectId"`
	Version     string                 `json:"version"`
	State       string                 `json:"state"`
	PID         int                    `json:"pid"`
	StartedAt   *time.Time             `json:"startedAt"`
	Health      string                 `json:"health"`
	Metrics     map[string]interface{}  `json:"metrics"`
	LastCheck   time.Time              `json:"lastCheck"`
}

// DeployRequest 部署请求
type DeployRequest struct {
	ProjectID         string            `json:"projectId"`
	Version           string            `json:"version"`
	IFPPackage        string            `json:"ifpPackage"`
	ExecutorType      string            `json:"executorType,omitempty"`
	ConnectionProfile ConnectionProfile `json:"connectionProfile"`
	EnvVars           map[string]string `json:"envVars,omitempty"`
	AutoStart         bool              `json:"autoStart,omitempty"`
}

// ConnectionProfile 连接配置
type ConnectionProfile struct {
	Name        string            `json:"name"`
	Endpoint   string            `json:"endpoint"`
	AuthType   string            `json:"authType"`
	AuthData   map[string]string `json:"authData"`
	Metadata   map[string]string `json:"metadata"`
	Secrets    map[string]string `json:"secrets,omitempty"`
}

// RuntimeOperation 运行时操作
type RuntimeOperation struct {
	ProjectID   string                 `json:"projectId"`
	Operation   string                 `json:"operation"`
	Version     string                 `json:"version"`
	Params      map[string]interface{} `json:"params"`
	RequestedAt time.Time              `json:"requestedAt"`
}

// ProjectInfo 项目信息
type ProjectInfo struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	CurrentVersion   string          `json:"currentVersion"`
	Status           string          `json:"status"`
	ConnectionProfile ConnectionProfile `json:"connectionProfile"`
	DeployedAt       *time.Time      `json:"deployedAt"`
	LastStartedAt    *time.Time      `json:"lastStartedAt"`
	RuntimeStatus    *RuntimeStatus  `json:"runtimeStatus,omitempty"`
}

// HeartbeatRequest 心跳请求
type HeartbeatRequest struct {
	NodeID            string                 `json:"nodeId"`
	RegistrationToken string                 `json:"registrationToken"`
	AgentVersion      string                 `json:"agentVersion"`
	Timestamp         time.Time              `json:"timestamp"`
	Status            string                 `json:"status"`
	Metrics           map[string]interface{} `json:"metrics"`
	Projects          []string               `json:"projects"`
}

// HeartbeatResponse 心跳响应
type HeartbeatResponse struct {
	Success bool                   `json:"success"`
	Data    HeartbeatResponseData  `json:"data"`
	Message string                 `json:"message,omitempty"`
}

// HeartbeatResponseData 心跳响应数据
type HeartbeatResponseData struct {
	Status   string               `json:"status"`
	Commands []PendingCommand     `json:"commands"`
}

// PendingCommand 待执行指令
type PendingCommand struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Status    string                 `json:"status"`
	Checks    map[string]interface{} `json:"checks"`
	Timestamp time.Time              `json:"timestamp"`
}

// Executor 执行器接口
type Executor interface {
	// 部署
	Deploy(ctx context.Context, req DeployRequest) error

	// 启动
	Start(ctx context.Context, projectID string) error

	// 停止
	Stop(ctx context.Context, projectID string) error

	// 重启
	Restart(ctx context.Context, projectID string) error

	// 回滚
	Rollback(ctx context.Context, projectID string, version string) error

	// 获取状态
	GetStatus(ctx context.Context, projectID string) (*RuntimeStatus, error)

	// 健康检查
	HealthCheck(ctx context.Context) error
}

// Store 存储接口
type Store interface {
	// 保存项目
	SaveProject(project *ProjectInfo) error

	// 获取项目
	GetProject(projectID string) (*ProjectInfo, error)

	// 列出项目
	ListProjects() ([]*ProjectInfo, error)

	// 删除项目
	DeleteProject(projectID string) error

	// 保存连接配置
	SaveConnectionProfile(projectID string, profile ConnectionProfile) error

	// 获取连接配置
	GetConnectionProfile(projectID string) (*ConnectionProfile, error)

	// 保存版本
	SaveVersion(projectID string, version string, files map[string][]byte) error

	// 获取版本目录
	GetVersionDir(projectID string, version string) (string, error)

	// 列出版本
	ListVersions(projectID string) ([]string, error)

	// 切换当前版本
	SwitchVersion(projectID string, version string) error

	// 获取当前版本
	GetCurrentVersion(projectID string) (string, error)
}

// Orchestrator 编排器接口
type Orchestrator interface {
	// 部署项目
	Deploy(ctx context.Context, req DeployRequest) error

	// 启动项目
	Start(ctx context.Context, projectID string) error

	// 停止项目
	Stop(ctx context.Context, projectID string) error

	// 重启项目
	Restart(ctx context.Context, projectID string) error

	// 回滚项目
	Rollback(ctx context.Context, projectID string, version string) error

	// 获取项目状态
	GetProjectStatus(ctx context.Context, projectID string) (*ProjectInfo, error)

	// 导入 IFP 包
	ImportIFP(ctx context.Context, projectID string, filePath string) error

	// 健康检查
	HealthCheck(ctx context.Context) error
}
