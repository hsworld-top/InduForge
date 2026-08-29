package ops

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/indu-forge/dev_core/internal/auth"
)

var (
	ErrNotFound              = errors.New("运维资源不存在")
	ErrEnrollmentUnavailable = errors.New("接入码无效、已使用或已过期")
	ErrAgentUnauthorized     = errors.New("节点代理令牌无效")
	ErrDeploymentBusy        = errors.New("当前部署操作尚未完成，请等待节点返回结果后重试")
	ErrDeploymentExists      = errors.New("该工程在目标运行集群已有部署，请使用现有部署进行操作")
)

type Repository interface {
	ListClusters(context.Context, string, PageFilter) ([]RuntimeCluster, int64, error)
	CreateCluster(context.Context, string, string, CreateClusterInput) (RuntimeCluster, error)
	GetCluster(context.Context, string, string) (RuntimeCluster, error)
	ListEnrollments(context.Context, string, PageFilter) ([]Enrollment, int64, error)
	CreateEnrollment(context.Context, string, string, CreateEnrollmentInput, string) (Enrollment, error)
	GetEnrollment(context.Context, string, string) (Enrollment, error)
	ApproveEnrollment(context.Context, string, string, string, bool) (Enrollment, error)
	ClaimEnrollment(context.Context, ClaimEnrollmentInput, string) (Enrollment, HostNode, error)
	ListHostNodes(context.Context, string, PageFilter) ([]HostNode, int64, error)
	GetHostNode(context.Context, string, string) (HostNode, error)
	Heartbeat(context.Context, string, string, HeartbeatInput) (HostNode, []Workload, error)
	AgentCommands(context.Context, string, string) ([]AgentCommand, error)
	ListDeployments(context.Context, string, PageFilter) ([]ProjectDeployment, int64, error)
	CreateDeployment(context.Context, string, string, CreateDeploymentInput) (ProjectDeployment, DeploymentRun, error)
	ValidateDeploymentTargets(context.Context, string, CreateDeploymentInput) error
	GetDeployment(context.Context, string, string) (ProjectDeployment, error)
	GetRun(context.Context, string, string) (DeploymentRun, error)
	ListRunEvents(context.Context, string, string) ([]DeploymentRunEvent, error)
	OperateWorkload(context.Context, string, string, string, string, string) (ProjectDeployment, DeploymentRun, error)
	CompleteDemoRun(context.Context, string) error
}

type Service struct {
	repository Repository
	packages   PackageStore
}
type PackageStore interface {
	List() []NodePackage
	Open(string) (NodePackage, string, error)
}

func NewService(repository Repository, packages PackageStore) *Service {
	return &Service{repository: repository, packages: packages}
}

func (s *Service) ListClusters(ctx context.Context, actor auth.User, filter PageFilter) ([]RuntimeCluster, int64, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeRead); err != nil {
		return nil, 0, err
	}
	return s.repository.ListClusters(ctx, actor.TenantID, normalizePage(filter))
}
func (s *Service) CreateCluster(ctx context.Context, actor auth.User, input CreateClusterInput) (RuntimeCluster, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeApprove); err != nil {
		return RuntimeCluster{}, err
	}
	input.Name, input.Code, input.Topology = strings.TrimSpace(input.Name), strings.TrimSpace(input.Code), strings.TrimSpace(input.Topology)
	if input.Name == "" || input.Code == "" {
		return RuntimeCluster{}, fmt.Errorf("集群名称和编码不能为空")
	}
	if utf8.RuneCountInString(input.Name) > 120 {
		return RuntimeCluster{}, fmt.Errorf("集群名称不能超过 120 个字符")
	}
	if !validClusterCode(input.Code) {
		return RuntimeCluster{}, fmt.Errorf("集群编码仅支持小写字母开头的 2-63 位小写字母、数字和连字符")
	}
	if input.Topology == "" {
		input.Topology = "single_node"
	}
	if input.Topology != "single_node" && input.Topology != "high_availability" {
		return RuntimeCluster{}, fmt.Errorf("集群拓扑不支持")
	}
	return s.repository.CreateCluster(ctx, actor.TenantID, actor.ID, input)
}
func (s *Service) GetCluster(ctx context.Context, actor auth.User, id string) (RuntimeCluster, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeRead); err != nil {
		return RuntimeCluster{}, err
	}
	return s.repository.GetCluster(ctx, actor.TenantID, id)
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
	input.Role = strings.TrimSpace(input.Role)
	input.RuntimeClusterID = strings.TrimSpace(input.RuntimeClusterID)
	if !validNodeRole(input.Role) {
		return Enrollment{}, "", fmt.Errorf("节点角色不支持")
	}
	if input.Role == RoleRuntimeLinux && input.RuntimeClusterID == "" {
		return Enrollment{}, "", fmt.Errorf("运行节点必须选择运行集群")
	}
	if input.RuntimeClusterID != "" && !validUUID(input.RuntimeClusterID) {
		return Enrollment{}, "", fmt.Errorf("运行集群 ID 格式无效")
	}
	if input.RuntimeClusterID != "" {
		if _, err := s.repository.GetCluster(ctx, actor.TenantID, input.RuntimeClusterID); err != nil {
			return Enrollment{}, "", err
		}
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
func (s *Service) ClaimEnrollment(ctx context.Context, input ClaimEnrollmentInput) (Enrollment, HostNode, string, error) {
	if strings.TrimSpace(input.Code) == "" || strings.TrimSpace(input.Hostname) == "" || strings.TrimSpace(input.OS) == "" || strings.TrimSpace(input.Architecture) == "" {
		return Enrollment{}, HostNode{}, "", fmt.Errorf("接入码、主机名、操作系统和架构不能为空")
	}
	token, err := randomToken(32)
	if err != nil {
		return Enrollment{}, HostNode{}, "", err
	}
	e, n, err := s.repository.ClaimEnrollment(ctx, input, hashToken(token))
	return e, n, token, err
}
func (s *Service) ListHostNodes(ctx context.Context, actor auth.User, f PageFilter) ([]HostNode, int64, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeRead); err != nil {
		return nil, 0, err
	}
	return s.repository.ListHostNodes(ctx, actor.TenantID, normalizePage(f))
}
func (s *Service) GetHostNode(ctx context.Context, actor auth.User, id string) (HostNode, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeRead); err != nil {
		return HostNode{}, err
	}
	return s.repository.GetHostNode(ctx, actor.TenantID, id)
}
func (s *Service) Heartbeat(ctx context.Context, nodeID, token string, input HeartbeatInput) (HostNode, []Workload, error) {
	if strings.TrimSpace(token) == "" {
		return HostNode{}, nil, ErrAgentUnauthorized
	}
	return s.repository.Heartbeat(ctx, nodeID, hashToken(token), input)
}
func (s *Service) AgentCommands(ctx context.Context, nodeID, token string) ([]AgentCommand, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrAgentUnauthorized
	}
	return s.repository.AgentCommands(ctx, nodeID, hashToken(token))
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
func (s *Service) OperateWorkload(ctx context.Context, actor auth.User, deploymentID, role, operation string) (ProjectDeployment, DeploymentRun, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityDeploymentOperate); err != nil {
		return ProjectDeployment{}, DeploymentRun{}, err
	}
	if operation != "start" && operation != "stop" && operation != "restart" {
		return ProjectDeployment{}, DeploymentRun{}, fmt.Errorf("工作负载操作不支持")
	}
	if !validWorkloadRole(role) {
		return ProjectDeployment{}, DeploymentRun{}, fmt.Errorf("工作负载角色不支持")
	}
	return s.repository.OperateWorkload(ctx, actor.TenantID, deploymentID, role, operation, actor.ID)
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
	return f
}
func validNodeRole(v string) bool {
	return v == RoleRuntimeLinux || v == RoleCollectorLinux || v == RoleCollectorWindows
}
func validClusterCode(v string) bool {
	if len(v) < 2 || len(v) > 63 || v[0] < 'a' || v[0] > 'z' {
		return false
	}
	for _, ch := range v[1:] {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			continue
		}
		return false
	}
	return true
}
func validWorkloadRole(v string) bool {
	return v == WorkloadRoleCompute || v == WorkloadRoleAlert || v == WorkloadRoleCollector
}
func validateDeployment(in CreateDeploymentInput) error {
	if strings.TrimSpace(in.ProjectID) == "" || strings.TrimSpace(in.RuntimeClusterID) == "" {
		return fmt.Errorf("工程和运行集群不能为空")
	}
	if !validUUID(in.ProjectID) || !validUUID(in.RuntimeClusterID) {
		return fmt.Errorf("工程 ID 和运行集群 ID 格式无效")
	}
	if in.DeploymentMode != "development" && in.DeploymentMode != "production" {
		return fmt.Errorf("部署模式必须是 development 或 production")
	}
	if len(in.Workloads) == 0 {
		return fmt.Errorf("至少需要一个工作负载")
	}
	seen := map[string]bool{}
	for _, w := range in.Workloads {
		if !validWorkloadRole(w.Role) {
			return fmt.Errorf("工作负载角色不支持")
		}
		if seen[w.Role] {
			return fmt.Errorf("工作负载角色不能重复")
		}
		seen[w.Role] = true
		if w.Role == WorkloadRoleCollector && strings.TrimSpace(w.HostNodeID) == "" {
			return fmt.Errorf("采集工作负载必须选择宿主节点")
		}
		if w.Role == WorkloadRoleCollector && !validUUID(w.HostNodeID) {
			return fmt.Errorf("采集宿主节点 ID 格式无效")
		}
		if w.Role != WorkloadRoleCollector && strings.TrimSpace(w.HostNodeID) != "" {
			return fmt.Errorf("计算和报警工作负载不能指定宿主节点")
		}
		if w.Replicas > 1 {
			return fmt.Errorf("Demo 工作负载当前仅支持单副本")
		}
	}
	return nil
}
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
