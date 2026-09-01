package ops

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/objectstore"
)

var (
	ErrNotFound                       = errors.New("运维资源不存在")
	ErrEnrollmentUnavailable          = errors.New("接入码无效、已使用或已过期")
	ErrAgentUnauthorized              = errors.New("节点代理令牌无效")
	ErrDeploymentBusy                 = errors.New("当前部署操作尚未完成，请等待节点返回结果后重试")
	ErrDeploymentExists               = errors.New("该工程已有正式单节点部署，请使用现有部署进行操作")
	ErrNodePortConflict               = errors.New("该物理节点上的工程端口与现有部署冲突")
	ErrReleaseNotDeployable           = errors.New("该版本不是可部署的正式 Release")
	ErrServiceLifecycleDisabled       = errors.New("逐服务操作已停用，请使用工程部署整体操作")
	ErrEnvironmentExists              = errors.New("运行环境名称已存在")
	ErrNodeEnvironmentConflict        = errors.New("物理节点已加入运行环境")
	ErrNodeNotEligible                = errors.New("只有已批准且有效的 Linux 运行节点才能加入运行环境")
	ErrNodeEnvironmentInUse           = errors.New("物理节点仍关联运行环境，不能移除")
	ErrNodeDeploymentInUse            = errors.New("物理节点仍承载工程部署，不能移除")
	ErrEnvironmentNodeServiceInUse    = errors.New("该节点仍承载当前运行环境的基础服务，不能取消分配")
	ErrEnvironmentNodeDeploymentInUse = errors.New("该节点仍承载工程部署，不能取消分配")
	ErrNodeOfflineForRemoval          = errors.New("物理节点当前离线，无法确认运行底座已安全清理")
	ErrCoordinatorInUse               = errors.New("环境协调节点仍有成员节点，必须先移除成员节点")
	ErrCenterNodeProtected            = errors.New("中心节点是平台内置节点，只能随整个平台卸载")
	ErrEnvironmentDeleting            = errors.New("运行环境正在删除，不能执行其他变更")
	ErrDefaultEnvironmentProtected    = errors.New("系统默认运行范围不能删除")
	ErrEnvironmentHasDeployment       = errors.New("运行环境仍有工程部署，必须先停止并删除工程部署")
	ErrFoundationExists               = errors.New("运行环境基础服务已创建，请等待当前期望状态完成")
	ErrFoundationMoveUnsupported      = errors.New("基础服务已部署，当前版本不支持变更服务所在节点；可使用原分配执行重新部署")
	ErrFoundationNotReady             = errors.New("所有目标节点必须在线且运行底座就绪后才能部署基础服务")
)

type Repository interface {
	ListEnrollments(context.Context, string, PageFilter) ([]Enrollment, int64, error)
	CreateEnrollment(context.Context, string, string, CreateEnrollmentInput, string) (Enrollment, error)
	GetEnrollment(context.Context, string, string) (Enrollment, error)
	ApproveEnrollment(context.Context, string, string, string, bool) (Enrollment, error)
	ClaimEnrollment(context.Context, ClaimEnrollmentInput, string) (Enrollment, Node, error)
	ListNodes(context.Context, string, PageFilter) ([]Node, int64, error)
	GetNode(context.Context, string, string) (Node, error)
	RemoveNode(context.Context, string, string, string) error
	ListRuntimeEnvironments(context.Context, string, PageFilter) ([]RuntimeEnvironment, int64, error)
	CreateRuntimeEnvironment(context.Context, string, string, CreateRuntimeEnvironmentInput, string) (RuntimeEnvironment, error)
	UpdateRuntimeEnvironment(context.Context, string, string, string, UpdateRuntimeEnvironmentInput) (RuntimeEnvironment, error)
	DeleteRuntimeEnvironment(context.Context, string, string, string, DeleteRuntimeEnvironmentInput) (string, error)
	GetRuntimeEnvironment(context.Context, string, string) (RuntimeEnvironment, error)
	ListRuntimeEnvironmentNodes(context.Context, string, string, PageFilter) ([]Node, int64, error)
	AddRuntimeEnvironmentNodes(context.Context, string, string, string, []string) ([]Node, error)
	RemoveRuntimeEnvironmentNode(context.Context, string, string, string, string) error
	ListRuntimeEnvironmentEvents(context.Context, string, string, PageFilter) ([]RuntimeEnvironmentEvent, int64, error)
	ListRuntimeEnvironmentServices(context.Context, string, string) ([]RuntimeEnvironmentService, error)
	DeployRuntimeEnvironmentFoundation(context.Context, string, string, string, []FoundationAssignment) ([]RuntimeEnvironmentService, error)
	MigrateRuntimeEnvironmentFoundation(context.Context, string, string, string, []FoundationAssignment) ([]RuntimeEnvironmentService, error)
	Heartbeat(context.Context, string, string, HeartbeatInput) (Node, []DeploymentService, error)
	AgentCommands(context.Context, string, string) ([]AgentCommand, error)
	AgentClusterPlan(context.Context, string, string) (*ClusterPlan, error)
	AgentClusterUninstall(context.Context, string, string) (*ClusterUninstall, error)
	AgentFoundationPlan(context.Context, string, string) (*FoundationPlan, error)
	AgentFoundationDelete(context.Context, string, string) (*FoundationDelete, error)
	AgentTimeSyncPlan(context.Context, string, string) (*TimeSyncPlan, error)
	ListDeployments(context.Context, string, PageFilter) ([]ProjectDeployment, int64, error)
	CreateDeployment(context.Context, string, string, CreateDeploymentInput) (ProjectDeployment, DeploymentRun, error)
	ValidateDeploymentTargets(context.Context, string, CreateDeploymentInput) error
	GetDeployment(context.Context, string, string) (ProjectDeployment, error)
	GetRun(context.Context, string, string) (DeploymentRun, error)
	ListRunEvents(context.Context, string, string) ([]DeploymentRunEvent, error)
	OperateService(context.Context, string, string, string, string, string) (ProjectDeployment, DeploymentRun, error)
	OperateDeployment(context.Context, string, string, string, string) (ProjectDeployment, DeploymentRun, error)
	GetAgentDeploymentBinding(context.Context, string, string, string, string) (DeploymentBinding, error)
	GetAgentRelease(context.Context, string, string, string, string) (AgentRelease, error)
}

// DevelopmentArtifactBuilder 是受控源码构建的窄依赖。ops 只接收已签名描述，
// 不反向依赖发布模块的具体实现或其存储细节。
type DevelopmentArtifactBuilder func(context.Context, auth.User, string, string) (DevelopmentArtifact, error)

var foundationServiceTypes = []string{"if_realtime", "if_history", "if_timeseries", "if_message", "if_object", "nats_jetstream", "nginx", "traefik"}

func validFoundationServiceType(value string) bool {
	for _, serviceType := range foundationServiceTypes {
		if value == serviceType {
			return true
		}
	}
	return false
}

func (s *Service) ListRuntimeEnvironmentServices(ctx context.Context, actor auth.User, environmentID string) ([]RuntimeEnvironmentService, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityEnvironmentRead); err != nil {
		return nil, err
	}
	return s.repository.ListRuntimeEnvironmentServices(ctx, actor.TenantID, environmentID)
}

func (s *Service) DeployRuntimeEnvironmentFoundation(ctx context.Context, actor auth.User, environmentID string, input DeployFoundationInput) ([]RuntimeEnvironmentService, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityEnvironmentManage); err != nil {
		return nil, err
	}
	if !validUUID(environmentID) || len(input.Assignments) != len(foundationServiceTypes) {
		return nil, fmt.Errorf("基础服务部署分配不完整")
	}
	assignments := make(map[string]string, len(input.Assignments))
	for _, assignment := range input.Assignments {
		if !validUUID(assignment.NodeID) || !validFoundationServiceType(assignment.ServiceType) || assignments[assignment.ServiceType] != "" {
			return nil, fmt.Errorf("基础服务类型或目标节点无效")
		}
		assignments[assignment.ServiceType] = assignment.NodeID
	}
	if assignments["if_history"] != assignments["if_timeseries"] {
		return nil, fmt.Errorf("IF 历史库与 IF 时序库共享同一存储实例，必须部署到同一节点")
	}
	if s.foundationNodePreflight != nil {
		ids := make([]string, 0, len(assignments))
		for _, id := range assignments {
			ids = append(ids, id)
		}
		if err := s.foundationNodePreflight.EnsureHostNodeLabels(ctx, ids); err != nil {
			return nil, fmt.Errorf("基础服务节点标签预检失败: %w", err)
		}
	}
	return s.repository.DeployRuntimeEnvironmentFoundation(ctx, actor.TenantID, environmentID, actor.ID, input.Assignments)
}

func (s *Service) MigrateRuntimeEnvironmentFoundation(ctx context.Context, actor auth.User, environmentID string, input MigrateFoundationInput) ([]RuntimeEnvironmentService, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityEnvironmentManage); err != nil {
		return nil, err
	}
	if !validUUID(environmentID) || len(input.Assignments) != len(foundationServiceTypes) {
		return nil, fmt.Errorf("基础服务调整分配不完整")
	}
	assignments := make(map[string]string, len(input.Assignments))
	for _, assignment := range input.Assignments {
		if !validUUID(assignment.NodeID) || !validFoundationServiceType(assignment.ServiceType) || assignments[assignment.ServiceType] != "" {
			return nil, fmt.Errorf("基础服务类型或目标节点无效")
		}
		assignments[assignment.ServiceType] = assignment.NodeID
	}
	if assignments["if_history"] != assignments["if_timeseries"] {
		return nil, fmt.Errorf("IF 历史库与 IF 时序库共享同一存储实例，必须部署到同一节点")
	}
	if s.foundationNodePreflight != nil {
		ids := make([]string, 0, len(assignments))
		for _, id := range assignments {
			ids = append(ids, id)
		}
		if err := s.foundationNodePreflight.EnsureHostNodeLabels(ctx, ids); err != nil {
			return nil, fmt.Errorf("基础服务节点标签预检失败: %w", err)
		}
	}
	return s.repository.MigrateRuntimeEnvironmentFoundation(ctx, actor.TenantID, environmentID, actor.ID, input.Assignments)
}

type PackageStore interface {
	List() []NodePackage
	Open(string) (NodePackage, string, error)
}
type ReleaseStore interface {
	Open(context.Context, string) (objectstore.ObjectReader, error)
}
type Service struct {
	repository              Repository
	packages                PackageStore
	releases                ReleaseStore
	clusterTokenKey         []byte
	developmentBuilder      DevelopmentArtifactBuilder
	foundationNodePreflight interface {
		EnsureHostNodeLabels(context.Context, []string) error
	}
}

func (s *Service) SetFoundationNodePreflight(preflight interface {
	EnsureHostNodeLabels(context.Context, []string) error
}) {
	s.foundationNodePreflight = preflight
}

// SetClusterTokenKey 配置集群令牌派生根密钥。调用方使用已有控制面密钥，避免
// 新增配置导致开发热启动失败；派生结果按环境隔离且不落库。
func (s *Service) SetClusterTokenKey(key []byte) {
	s.clusterTokenKey = append([]byte(nil), key...)
}

func (s *Service) SetDevelopmentArtifactBuilder(builder DevelopmentArtifactBuilder) {
	s.developmentBuilder = builder
}

func NewService(repository Repository, packages PackageStore, releases ...ReleaseStore) *Service {
	service := &Service{repository: repository, packages: packages}
	if len(releases) > 0 {
		service.releases = releases[0]
	}
	return service
}

func (s *Service) ListEnrollments(ctx context.Context, actor auth.User, f PageFilter) ([]Enrollment, int64, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeRead); err != nil {
		return nil, 0, err
	}
	return s.repository.ListEnrollments(ctx, actor.TenantID, normalizePage(f))
}
func (s *Service) CreateEnrollment(ctx context.Context, actor auth.User, input CreateEnrollmentInput) (Enrollment, string, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeApprove); err != nil {
		return Enrollment{}, "", err
	}
	input.Platform, input.DisplayName = strings.ToLower(strings.TrimSpace(input.Platform)), strings.TrimSpace(input.DisplayName)
	if !validPlatform(input.Platform) {
		return Enrollment{}, "", fmt.Errorf("节点平台仅支持 linux 或 windows")
	}
	if err := validateCapabilities(input.Capabilities); err != nil {
		return Enrollment{}, "", err
	}
	if input.TTL <= 0 {
		input.TTL = 30 * time.Minute
	}
	if input.TTL > 24*time.Hour {
		return Enrollment{}, "", fmt.Errorf("接入码有效期不能超过 24 小时")
	}
	code, err := randomToken(18)
	if err != nil {
		return Enrollment{}, "", err
	}
	item, err := s.repository.CreateEnrollment(ctx, actor.TenantID, actor.ID, input, hashToken(code))
	return item, code, err
}
func (s *Service) GetEnrollment(ctx context.Context, actor auth.User, id string) (Enrollment, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeRead); err != nil {
		return Enrollment{}, err
	}
	return s.repository.GetEnrollment(ctx, actor.TenantID, id)
}
func (s *Service) ApproveEnrollment(ctx context.Context, actor auth.User, id string, approve bool) (Enrollment, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeApprove); err != nil {
		return Enrollment{}, err
	}
	return s.repository.ApproveEnrollment(ctx, actor.TenantID, id, actor.ID, approve)
}
func (s *Service) ClaimEnrollment(ctx context.Context, input ClaimEnrollmentInput) (Enrollment, Node, string, error) {
	input.Platform = strings.ToLower(strings.TrimSpace(input.Platform))
	input.Hostname = strings.TrimSpace(input.Hostname)
	input.Architecture = strings.TrimSpace(input.Architecture)
	input.AgentVersion = strings.TrimSpace(input.AgentVersion)
	input.MachineFingerprint = strings.TrimSpace(input.MachineFingerprint)
	if strings.TrimSpace(input.Code) == "" || input.Hostname == "" || !validPlatform(input.Platform) || input.Architecture == "" || input.AgentVersion == "" || input.MachineFingerprint == "" {
		return Enrollment{}, Node{}, "", fmt.Errorf("接入码、主机名、平台、架构、Agent 版本和机器指纹不能为空或无效")
	}
	if len(input.Hostname) > 255 || len(input.Architecture) > 64 || len(input.AgentVersion) > 128 || len(input.MachineFingerprint) > 512 {
		return Enrollment{}, Node{}, "", fmt.Errorf("NodeAgent 主机身份字段长度无效")
	}
	if err := validateCapabilities(input.Capabilities); err != nil {
		return Enrollment{}, Node{}, "", err
	}
	token, err := randomToken(32)
	if err != nil {
		return Enrollment{}, Node{}, "", err
	}
	e, n, err := s.repository.ClaimEnrollment(ctx, input, hashToken(token))
	return e, n, token, err
}
func (s *Service) ListNodes(ctx context.Context, actor auth.User, f PageFilter) ([]Node, int64, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeRead); err != nil {
		return nil, 0, err
	}
	return s.repository.ListNodes(ctx, actor.TenantID, normalizePage(f))
}
func (s *Service) GetNode(ctx context.Context, actor auth.User, id string) (Node, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeRead); err != nil {
		return Node{}, err
	}
	return s.repository.GetNode(ctx, actor.TenantID, id)
}
func (s *Service) RemoveNode(ctx context.Context, actor auth.User, id string) error {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeDelete); err != nil {
		return err
	}
	if !validUUID(id) {
		return ErrNotFound
	}
	return s.repository.RemoveNode(ctx, actor.TenantID, id, actor.ID)
}

func (s *Service) ListRuntimeEnvironments(ctx context.Context, actor auth.User, f PageFilter) ([]RuntimeEnvironment, int64, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityEnvironmentRead); err != nil {
		return nil, 0, err
	}
	f = normalizePage(f)
	if f.Status != "" && f.Status != "available" && f.Status != "attention" && f.Status != "uninitialized" && f.Status != "deleting" {
		return nil, 0, fmt.Errorf("运行环境状态筛选值无效")
	}
	return s.repository.ListRuntimeEnvironments(ctx, actor.TenantID, f)
}

func (s *Service) CreateRuntimeEnvironment(ctx context.Context, actor auth.User, input CreateRuntimeEnvironmentInput) (RuntimeEnvironment, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityEnvironmentManage); err != nil {
		return RuntimeEnvironment{}, err
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len([]rune(input.Name)) > 80 {
		return RuntimeEnvironment{}, fmt.Errorf("运行环境名称不能为空且不能超过 80 个字符")
	}
	token, err := randomToken(6)
	if err != nil {
		return RuntimeEnvironment{}, err
	}
	return s.repository.CreateRuntimeEnvironment(ctx, actor.TenantID, actor.ID, input, "runtime-"+token)
}

func (s *Service) UpdateRuntimeEnvironment(ctx context.Context, actor auth.User, id string, input UpdateRuntimeEnvironmentInput) (RuntimeEnvironment, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityEnvironmentManage); err != nil {
		return RuntimeEnvironment{}, err
	}
	input.Name = strings.TrimSpace(input.Name)
	if !validUUID(id) || input.Name == "" || len([]rune(input.Name)) > 80 {
		return RuntimeEnvironment{}, fmt.Errorf("运行环境名称不能为空且不能超过 80 个字符")
	}
	return s.repository.UpdateRuntimeEnvironment(ctx, actor.TenantID, id, actor.ID, input)
}

func (s *Service) DeleteRuntimeEnvironment(ctx context.Context, actor auth.User, id string, input DeleteRuntimeEnvironmentInput) (string, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityEnvironmentManage); err != nil {
		return "", err
	}
	input.ConfirmationName = strings.TrimSpace(input.ConfirmationName)
	if !validUUID(id) || input.ConfirmationName == "" {
		return "", fmt.Errorf("必须输入运行环境名称确认删除")
	}
	return s.repository.DeleteRuntimeEnvironment(ctx, actor.TenantID, id, actor.ID, input)
}

func (s *Service) GetRuntimeEnvironment(ctx context.Context, actor auth.User, id string) (RuntimeEnvironment, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityEnvironmentRead); err != nil {
		return RuntimeEnvironment{}, err
	}
	if !validUUID(id) {
		return RuntimeEnvironment{}, fmt.Errorf("运行环境 ID 格式无效")
	}
	return s.repository.GetRuntimeEnvironment(ctx, actor.TenantID, id)
}

func (s *Service) ListRuntimeEnvironmentNodes(ctx context.Context, actor auth.User, id string, f PageFilter) ([]Node, int64, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityEnvironmentRead); err != nil {
		return nil, 0, err
	}
	if !validUUID(id) {
		return nil, 0, fmt.Errorf("运行环境 ID 格式无效")
	}
	return s.repository.ListRuntimeEnvironmentNodes(ctx, actor.TenantID, id, normalizePage(f))
}

func (s *Service) AddRuntimeEnvironmentNodes(ctx context.Context, actor auth.User, id string, input AddRuntimeEnvironmentNodesInput) ([]Node, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityEnvironmentManage); err != nil {
		return nil, err
	}
	if !validUUID(id) || len(input.NodeIDs) == 0 || len(input.NodeIDs) > 100 {
		return nil, fmt.Errorf("运行环境或物理节点 ID 格式无效")
	}
	seen := make(map[string]struct{}, len(input.NodeIDs))
	for _, nodeID := range input.NodeIDs {
		if !validUUID(nodeID) {
			return nil, fmt.Errorf("运行环境或物理节点 ID 格式无效")
		}
		if _, exists := seen[nodeID]; exists {
			return nil, fmt.Errorf("物理节点不能重复")
		}
		seen[nodeID] = struct{}{}
	}
	return s.repository.AddRuntimeEnvironmentNodes(ctx, actor.TenantID, id, actor.ID, input.NodeIDs)
}

func (s *Service) RemoveRuntimeEnvironmentNode(ctx context.Context, actor auth.User, id, nodeID string) error {
	if err := auth.RequireCapability(actor, auth.CapabilityEnvironmentManage); err != nil {
		return err
	}
	if !validUUID(id) || !validUUID(nodeID) {
		return fmt.Errorf("运行环境或物理节点 ID 格式无效")
	}
	return s.repository.RemoveRuntimeEnvironmentNode(ctx, actor.TenantID, id, nodeID, actor.ID)
}

func (s *Service) ListRuntimeEnvironmentEvents(ctx context.Context, actor auth.User, id string, f PageFilter) ([]RuntimeEnvironmentEvent, int64, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityEnvironmentRead); err != nil {
		return nil, 0, err
	}
	if !validUUID(id) {
		return nil, 0, fmt.Errorf("运行环境 ID 格式无效")
	}
	return s.repository.ListRuntimeEnvironmentEvents(ctx, actor.TenantID, id, normalizePage(f))
}
func (s *Service) Heartbeat(ctx context.Context, nodeID, token string, input HeartbeatInput) (Node, []DeploymentService, error) {
	if strings.TrimSpace(token) == "" {
		return Node{}, nil, ErrAgentUnauthorized
	}
	for _, observation := range input.Services {
		if !validUUID(observation.ServiceID) || (observation.ObservedStatus != "running" && observation.ObservedStatus != "stopped" && observation.ObservedStatus != "failed") || observation.ObservedGeneration < 1 || observation.ReplicasObserved < 0 || observation.ReplicasObserved > 1 || len(observation.Message) > 1024 || len(observation.Endpoint) > 2048 {
			return Node{}, nil, fmt.Errorf("节点服务观测字段无效")
		}
		if err := validateReportedEndpoint(observation.Endpoint); err != nil {
			return Node{}, nil, err
		}
	}
	if input.IPAddress != "" && !validNodeIPAddress(input.IPAddress) {
		return Node{}, nil, fmt.Errorf("节点上报的通信地址无效")
	}
	if input.ClusterState != nil {
		state := input.ClusterState
		if state.SchemaVersion != "induforge.cluster-state.v1" || state.Generation < 0 || len(state.Message) > 1024 ||
			(state.ObservedState != "starting" && state.ObservedState != "ready" && state.ObservedState != "failed" && state.ObservedState != "not-installed") {
			return Node{}, nil, fmt.Errorf("节点集群观测字段无效")
		}
	}
	for index := range input.FoundationStates {
		state := &input.FoundationStates[index]
		if state.SchemaVersion != "induforge.foundation-state.v1" || state.Generation < 0 || len(state.Message) > 1024 ||
			(state.ObservedState != "starting" && state.ObservedState != "ready" && state.ObservedState != "failed" && state.ObservedState != "not-installed") {
			return Node{}, nil, fmt.Errorf("节点基础服务观测字段无效")
		}
		seen := make(map[string]struct{}, len(state.Services))
		for _, service := range state.Services {
			if (service.Status != "starting" && service.Status != "ready" && service.Status != "failed") || len(service.Message) > 1024 {
				return Node{}, nil, fmt.Errorf("节点基础服务明细观测无效")
			}
			if service.Workload != "postgres" && service.Workload != "redis" && service.Workload != "emqx" && service.Workload != "nats" && service.Workload != "object" && service.Workload != "nginx" && service.Workload != "traefik" {
				return Node{}, nil, fmt.Errorf("节点基础服务明细类型无效")
			}
			if _, exists := seen[service.Workload]; exists {
				return Node{}, nil, fmt.Errorf("节点基础服务明细重复")
			}
			seen[service.Workload] = struct{}{}
		}
	}
	return s.repository.Heartbeat(ctx, nodeID, hashToken(token), input)
}

func validNodeIPAddress(value string) bool {
	ip := net.ParseIP(strings.TrimSpace(value))
	return ip != nil && !ip.IsUnspecified() && !ip.IsLoopback() && !ip.IsMulticast()
}

// validateReportedEndpoint 只校验 Agent 上报地址的安全边界；服务类型归属由仓储层
// 依据 serviceId 的数据库记录判定，避免 Agent 伪造 collector/runtime 的访问入口。
func validateReportedEndpoint(endpoint string) error {
	if endpoint == "" {
		return nil
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("工程入口地址必须为不含 userinfo 或 fragment 的 http/https URL")
	}
	return nil
}
func (s *Service) AgentCommands(ctx context.Context, nodeID, token string) ([]AgentCommand, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrAgentUnauthorized
	}
	return s.repository.AgentCommands(ctx, nodeID, hashToken(token))
}

func (s *Service) AgentClusterPlan(ctx context.Context, nodeID, token string) (*ClusterPlan, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrAgentUnauthorized
	}
	plan, err := s.repository.AgentClusterPlan(ctx, nodeID, hashToken(token))
	if err != nil || plan == nil {
		return plan, err
	}
	if len(s.clusterTokenKey) < 16 {
		return nil, fmt.Errorf("控制面集群令牌密钥未配置")
	}
	mac := hmac.New(sha256.New, s.clusterTokenKey)
	_, _ = mac.Write([]byte("induforge:k3s:v1:" + plan.ClusterID))
	plan.Token = base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return plan, nil
}

func (s *Service) AgentClusterUninstall(ctx context.Context, nodeID, token string) (*ClusterUninstall, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrAgentUnauthorized
	}
	return s.repository.AgentClusterUninstall(ctx, nodeID, hashToken(token))
}

func (s *Service) AgentFoundationPlan(ctx context.Context, nodeID, token string) (*FoundationPlan, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrAgentUnauthorized
	}
	plan, err := s.repository.AgentFoundationPlan(ctx, nodeID, hashToken(token))
	if err != nil || plan == nil || s.foundationNodePreflight == nil {
		return plan, err
	}
	ids := make([]string, 0, len(plan.Assignments))
	for _, id := range plan.Assignments {
		ids = append(ids, id)
	}
	if err := s.foundationNodePreflight.EnsureHostNodeLabels(ctx, ids); err != nil {
		return nil, fmt.Errorf("基础服务节点标签预检失败: %w", err)
	}
	return plan, nil
}

func (s *Service) AgentTimeSyncPlan(ctx context.Context, nodeID, token string) (*TimeSyncPlan, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrAgentUnauthorized
	}
	return s.repository.AgentTimeSyncPlan(ctx, nodeID, hashToken(token))
}
func (s *Service) AgentFoundationDelete(ctx context.Context, nodeID, token string) (*FoundationDelete, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrAgentUnauthorized
	}
	return s.repository.AgentFoundationDelete(ctx, nodeID, hashToken(token))
}
func (s *Service) AgentDeploymentBinding(ctx context.Context, nodeID, token, deploymentID, serviceID string) (DeploymentBinding, error) {
	if strings.TrimSpace(token) == "" {
		return DeploymentBinding{}, ErrAgentUnauthorized
	}
	return s.repository.GetAgentDeploymentBinding(ctx, nodeID, hashToken(token), deploymentID, serviceID)
}
func (s *Service) AgentRelease(ctx context.Context, nodeID, token, deploymentID, serviceID string) (AgentRelease, objectstore.ObjectReader, error) {
	if strings.TrimSpace(token) == "" {
		return AgentRelease{}, objectstore.ObjectReader{}, ErrAgentUnauthorized
	}
	if s.releases == nil {
		return AgentRelease{}, objectstore.ObjectReader{}, fmt.Errorf("%w: Release 下载存储未配置", ErrReleaseNotDeployable)
	}
	release, err := s.repository.GetAgentRelease(ctx, nodeID, hashToken(token), deploymentID, serviceID)
	if err != nil {
		return AgentRelease{}, objectstore.ObjectReader{}, err
	}
	content, err := s.releases.Open(ctx, release.ArtifactKey)
	if err != nil {
		return AgentRelease{}, objectstore.ObjectReader{}, fmt.Errorf("%w: Release 对象不可读取", ErrReleaseNotDeployable)
	}
	if release.ArtifactSize <= 0 || content.Size != release.ArtifactSize {
		_ = content.Reader.Close()
		return AgentRelease{}, objectstore.ObjectReader{}, fmt.Errorf("%w: Release 对象大小不匹配", ErrReleaseNotDeployable)
	}
	return release, content, nil
}
func (s *Service) ListDeployments(ctx context.Context, actor auth.User, f PageFilter) ([]ProjectDeployment, int64, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeRead); err != nil {
		return nil, 0, err
	}
	return s.repository.ListDeployments(ctx, actor.TenantID, normalizePage(f))
}
func (s *Service) CreateDeployment(ctx context.Context, actor auth.User, input CreateDeploymentInput) (ProjectDeployment, DeploymentRun, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityDeploymentExecute); err != nil {
		return ProjectDeployment{}, DeploymentRun{}, err
	}
	if err := validateDeployment(input); err != nil {
		return ProjectDeployment{}, DeploymentRun{}, err
	}
	// API 以 production 表达用户意图，持久层沿用既有 release 枚举，避免把
	// 编排内部术语暴露给调用方或引入并行模式。
	if input.Mode == "production" {
		input.Mode = "release"
	}
	if input.Mode == "development" {
		if s.developmentBuilder == nil {
			return ProjectDeployment{}, DeploymentRun{}, fmt.Errorf("开发制品构建服务未配置")
		}
		artifact, err := s.developmentBuilder(ctx, actor, input.ProjectID, input.Authorization)
		if err != nil {
			return ProjectDeployment{}, DeploymentRun{}, fmt.Errorf("构建开发制品失败: %w", err)
		}
		input.DevelopmentArtifact = &artifact
	}
	if err := s.repository.ValidateDeploymentTargets(ctx, actor.TenantID, input); err != nil {
		return ProjectDeployment{}, DeploymentRun{}, err
	}
	return s.repository.CreateDeployment(ctx, actor.TenantID, actor.ID, input)
}
func (s *Service) GetDeployment(ctx context.Context, actor auth.User, id string) (ProjectDeployment, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeRead); err != nil {
		return ProjectDeployment{}, err
	}
	return s.repository.GetDeployment(ctx, actor.TenantID, id)
}
func (s *Service) GetRun(ctx context.Context, actor auth.User, id string) (DeploymentRun, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeRead); err != nil {
		return DeploymentRun{}, err
	}
	return s.repository.GetRun(ctx, actor.TenantID, id)
}
func (s *Service) ListRunEvents(ctx context.Context, actor auth.User, id string) ([]DeploymentRunEvent, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeRead); err != nil {
		return nil, err
	}
	return s.repository.ListRunEvents(ctx, actor.TenantID, id)
}
func (s *Service) OperateService(ctx context.Context, actor auth.User, deploymentID, serviceType, operation string) (ProjectDeployment, DeploymentRun, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityDeploymentOperate); err != nil {
		return ProjectDeployment{}, DeploymentRun{}, err
	}
	// 旧服务级路由暂时保留以返回明确错误，但不得再进入仓储层形成部分运行状态。
	// 正式运维只允许以工程部署为一个原子生命周期单元。
	return ProjectDeployment{}, DeploymentRun{}, ErrServiceLifecycleDisabled
}

// OperateDeployment 是正式工程部署的生命周期入口。它把同一工程部署下的
// 全部服务作为一个原子工作负载处理，避免用户在服务级别拼装运行状态。
func (s *Service) OperateDeployment(ctx context.Context, actor auth.User, deploymentID, operation string) (ProjectDeployment, DeploymentRun, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityDeploymentOperate); err != nil {
		return ProjectDeployment{}, DeploymentRun{}, err
	}
	if operation != "start" && operation != "stop" && operation != "restart" {
		return ProjectDeployment{}, DeploymentRun{}, fmt.Errorf("工程部署操作不支持")
	}
	return s.repository.OperateDeployment(ctx, actor.TenantID, deploymentID, operation, actor.ID)
}
func (s *Service) ListPackages() []NodePackage {
	if s.packages == nil {
		return []NodePackage{}
	}
	return s.packages.List()
}
func (s *Service) OpenPackage(id string) (NodePackage, string, error) {
	if s.packages == nil {
		return NodePackage{}, "", ErrNotFound
	}
	return s.packages.Open(id)
}

func normalizePage(f PageFilter) PageFilter {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > 200 {
		f.PageSize = 200
	}
	f.Search = strings.TrimSpace(f.Search)
	f.ProjectID = strings.TrimSpace(f.ProjectID)
	f.Status = strings.TrimSpace(f.Status)
	return f
}
func validPlatform(v string) bool { return v == PlatformLinux || v == PlatformWindows }
func validCapability(v string) bool {
	return v == CapabilityProjectEntry || v == CapabilityDataRuntime || v == CapabilityCollector
}
func validServiceType(v string) bool {
	return v == ServiceBase || v == ServiceCompute || v == ServiceAlarm || v == ServiceCollector
}
func validateCapabilities(values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("节点至少需要一项能力")
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if !validCapability(value) {
			return fmt.Errorf("节点能力不支持")
		}
		if _, ok := seen[value]; ok {
			return fmt.Errorf("节点能力不能重复")
		}
		seen[value] = struct{}{}
	}
	return nil
}
func validateDeployment(in CreateDeploymentInput) error {
	if !validUUID(in.ProjectID) || !validUUID(in.EnvironmentID) {
		return fmt.Errorf("工程或运行环境 ID 格式无效")
	}
	if in.Mode != "development" && in.Mode != "production" {
		return fmt.Errorf("部署模式无效")
	}
	if in.Mode == "production" && !validUUID(in.ApplicationVersionID) {
		return fmt.Errorf("正式版本 ID 格式无效")
	}
	if in.Mode == "development" && in.ApplicationVersionID != "" {
		return fmt.Errorf("开发部署不能绑定正式版本")
	}
	if in.DevelopmentArtifact != nil {
		return fmt.Errorf("开发制品只能由服务端生成")
	}
	// 0 表示由服务端在安全动态范围内分配；浏览器不得猜测端口。显式端口
	// 继续禁止特权端口，并为节点原生运行时保留相邻内部端口空间。
	if in.AccessPort != 0 && (in.AccessPort < 1024 || in.AccessPort > 65532) {
		return fmt.Errorf("工程访问端口必须在 1024 到 65532 之间")
	}
	return validateEnginePlacements([]string{ServiceBase}, in.Placements)
}
func validUUID(value string) bool { _, err := uuid.Parse(value); return err == nil }
func randomToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("生成令牌失败: %w", err)
	}
	return hex.EncodeToString(b), nil
}
func hashToken(v string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(v)))
	return hex.EncodeToString(h[:])
}
