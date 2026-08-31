package ops

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/auth"
)

var (
	ErrNotFound              = errors.New("运维资源不存在")
	ErrEnrollmentUnavailable = errors.New("接入码无效、已使用或已过期")
	ErrAgentUnauthorized     = errors.New("节点代理令牌无效")
	ErrDeploymentBusy        = errors.New("当前部署操作尚未完成，请等待节点返回结果后重试")
	ErrDeploymentExists      = errors.New("该工程已有正式单节点部署，请使用现有部署进行操作")
	ErrNodeProjectConflict   = errors.New("该物理节点已承载另一个工程；在 DeploymentBinding 和端口隔离交付前，每个节点仅能部署一个工程")
	ErrReleaseNotDeployable  = errors.New("该版本不是可部署的正式 Release")
)

type Repository interface {
	ListEnrollments(context.Context, string, PageFilter) ([]Enrollment, int64, error)
	CreateEnrollment(context.Context, string, string, CreateEnrollmentInput, string) (Enrollment, error)
	GetEnrollment(context.Context, string, string) (Enrollment, error)
	ApproveEnrollment(context.Context, string, string, string, bool) (Enrollment, error)
	ClaimEnrollment(context.Context, ClaimEnrollmentInput, string) (Enrollment, Node, error)
	ListNodes(context.Context, string, PageFilter) ([]Node, int64, error)
	GetNode(context.Context, string, string) (Node, error)
	Heartbeat(context.Context, string, string, HeartbeatInput) (Node, []DeploymentService, error)
	AgentCommands(context.Context, string, string) ([]AgentCommand, error)
	ListDeployments(context.Context, string, PageFilter) ([]ProjectDeployment, int64, error)
	CreateDeployment(context.Context, string, string, CreateDeploymentInput) (ProjectDeployment, DeploymentRun, error)
	ValidateDeploymentTargets(context.Context, string, CreateDeploymentInput) error
	GetDeployment(context.Context, string, string) (ProjectDeployment, error)
	GetRun(context.Context, string, string) (DeploymentRun, error)
	ListRunEvents(context.Context, string, string) ([]DeploymentRunEvent, error)
	OperateService(context.Context, string, string, string, string, string) (ProjectDeployment, DeploymentRun, error)
}

type PackageStore interface {
	List() []NodePackage
	Open(string) (NodePackage, string, error)
}
type Service struct {
	repository Repository
	packages   PackageStore
}

func NewService(repository Repository, packages PackageStore) *Service {
	return &Service{repository: repository, packages: packages}
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
	return s.repository.Heartbeat(ctx, nodeID, hashToken(token), input)
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
func (s *Service) OperateService(ctx context.Context, actor auth.User, deploymentID, serviceType, operation string) (ProjectDeployment, DeploymentRun, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityDeploymentOperate); err != nil {
		return ProjectDeployment{}, DeploymentRun{}, err
	}
	if operation != "start" && operation != "stop" && operation != "restart" {
		return ProjectDeployment{}, DeploymentRun{}, fmt.Errorf("工程服务操作不支持")
	}
	if !validServiceType(serviceType) {
		return ProjectDeployment{}, DeploymentRun{}, fmt.Errorf("工程服务类型不支持")
	}
	return s.repository.OperateService(ctx, actor.TenantID, deploymentID, serviceType, operation, actor.ID)
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
	return f
}
func validPlatform(v string) bool { return v == PlatformLinux || v == PlatformWindows }
func validCapability(v string) bool {
	return v == CapabilityProjectEntry || v == CapabilityDataRuntime || v == CapabilityCollector
}
func validServiceType(v string) bool {
	return v == ServiceProjectEntry || v == ServiceDataRuntime || v == ServiceCollector
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
	if !validUUID(in.ProjectID) || !validUUID(in.NodeID) || !validUUID(in.ApplicationVersionID) {
		return fmt.Errorf("工程、节点和正式版本 ID 格式无效")
	}
	return nil
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
