package node

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/auth"
)

var ErrNotFound = errors.New("节点不存在")

type Node struct {
	ID              string
	TenantID        string
	Name            string
	Code            string
	NodeType        string
	Status          string
	ApprovalStatus  string
	AgentVersion    string
	OS              string
	Architecture    string
	Hostname        string
	IPAddress       string
	Capabilities    map[string]any
	Metrics         map[string]any
	Metadata        map[string]any
	Deployments     []map[string]any
	LastHeartbeatAt *time.Time
	ApprovedAt      *time.Time
	RejectedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ListFilter struct {
	Page           int
	Limit          int
	Status         string
	ApprovalStatus string
	Keyword        string
}

type RegisterInput struct {
	NodeID           string
	Name             string
	Description      string
	AgentVersion     string
	IPAddress        string
	Port             int
	Mode             string
	UserAgent        string
	Registrant       auth.User
	RegistrationHash string
	AutoApprove      bool
}

type RegisterResult struct {
	NodeID            string
	NodeName          string
	ApprovalStatus    string
	RegistrationToken string
	AutoApproved      bool
}

type ApprovalStatus struct {
	NodeID         string
	NodeName       string
	Status         string
	ApprovalStatus string
	ApprovedAt     *time.Time
	RejectedAt     *time.Time
	UpdatedAt      time.Time
}

type Command struct {
	Type    string
	Payload map[string]any
}

type HeartbeatInput struct {
	RegistrationToken string
	AgentVersion      string
	Metrics           map[string]any
}

type DeploymentReport struct {
	RegistrationToken string
	DeploymentID      string
	Status            string
	ErrorMessage      string
	Message           string
	StartedAt         *time.Time
	StoppedAt         *time.Time
}

type DeploymentUpdate struct {
	ID           string
	TenantID     string
	NodeID       string
	ProjectID    string
	Status       string
	ErrorMessage string
	StartedAt    *time.Time
	StoppedAt    *time.Time
}

type Repository interface {
	List(context.Context, string, ListFilter) ([]Node, int64, error)
	Approve(context.Context, string, string, string) (Node, error)
	Reject(context.Context, string, string, string) (Node, error)
	Delete(context.Context, string, string) error
	Register(context.Context, RegisterInput) (Node, error)
	GetApprovalStatus(context.Context, string) (ApprovalStatus, error)
	Heartbeat(context.Context, string, string, HeartbeatInput) (Node, []Command, error)
	Offline(context.Context, string, string, string) (Node, error)
	ReportDeployment(context.Context, string, string, DeploymentReport) (DeploymentUpdate, error)
}

type OnlineStore interface {
	TouchNode(context.Context, string, time.Duration) error
}

type Events interface {
	NodePending(string, map[string]any)
	NodeMetrics(string, string, map[string]any)
	NodeStatus(string, string, string)
	DeployStatus(string, map[string]any)
}

type Service struct {
	repository Repository
	online     OnlineStore
	events     Events
}

func (s *Service) SetEvents(events Events) { s.events = events }

func NewService(repository Repository, online ...OnlineStore) *Service {
	service := &Service{repository: repository}
	if len(online) > 0 {
		service.online = online[0]
	}
	return service
}

func (s *Service) List(ctx context.Context, actor auth.User, filter ListFilter) ([]Node, int64, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeRead); err != nil {
		return nil, 0, err
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	filter.ApprovalStatus = strings.ToLower(strings.TrimSpace(filter.ApprovalStatus))
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	return s.repository.List(ctx, actor.TenantID, filter)
}

func (s *Service) Approve(ctx context.Context, actor auth.User, nodeID string) (Node, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeApprove); err != nil {
		return Node{}, err
	}
	if strings.TrimSpace(nodeID) == "" {
		return Node{}, fmt.Errorf("节点 ID 不能为空")
	}
	item, err := s.repository.Approve(ctx, actor.TenantID, nodeID, actor.ID)
	if err == nil && s.events != nil {
		s.events.NodeStatus(item.TenantID, item.ID, item.Status)
	}
	return item, err
}

func (s *Service) Reject(ctx context.Context, actor auth.User, nodeID string) (Node, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeApprove); err != nil {
		return Node{}, err
	}
	if strings.TrimSpace(nodeID) == "" {
		return Node{}, fmt.Errorf("节点 ID 不能为空")
	}
	item, err := s.repository.Reject(ctx, actor.TenantID, nodeID, actor.ID)
	if err == nil && s.events != nil {
		s.events.NodeStatus(item.TenantID, item.ID, "rejected")
	}
	return item, err
}

func (s *Service) Delete(ctx context.Context, actor auth.User, nodeID string) error {
	if err := auth.RequireCapability(actor, auth.CapabilityNodeDelete); err != nil {
		return err
	}
	if strings.TrimSpace(nodeID) == "" {
		return fmt.Errorf("节点 ID 不能为空")
	}
	return s.repository.Delete(ctx, actor.TenantID, nodeID)
}

func (s *Service) Register(ctx context.Context, actor auth.User, input RegisterInput) (RegisterResult, error) {
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]{3,50}$`).MatchString(strings.TrimSpace(input.Name)) {
		return RegisterResult{}, fmt.Errorf("节点名称仅允许字母、数字、中划线和下划线，长度 3-50")
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return RegisterResult{}, fmt.Errorf("生成节点注册令牌失败: %w", err)
	}
	rawToken := hex.EncodeToString(tokenBytes)
	input.NodeID = normalizeNodeID(input.NodeID)
	input.Name = strings.TrimSpace(input.Name)
	input.Registrant = actor
	input.RegistrationHash = hashToken(rawToken)
	input.AutoApprove = auth.HasCapability(actor.Role, auth.CapabilityNodeApprove)
	item, err := s.repository.Register(ctx, input)
	if err != nil {
		return RegisterResult{}, err
	}
	if s.events != nil && item.ApprovalStatus == "pending" {
		s.events.NodePending(item.TenantID, map[string]any{"nodeId": item.ID, "nodeName": item.Name, "approvalStatus": item.ApprovalStatus})
	}
	return RegisterResult{NodeID: item.ID, NodeName: item.Name, ApprovalStatus: item.ApprovalStatus, RegistrationToken: rawToken, AutoApproved: input.AutoApprove}, nil
}

func (s *Service) ApprovalStatus(ctx context.Context, nodeID string) (ApprovalStatus, error) {
	return s.repository.GetApprovalStatus(ctx, normalizeNodeID(nodeID))
}

func (s *Service) Heartbeat(ctx context.Context, nodeID string, input HeartbeatInput) (Node, []Command, error) {
	if strings.TrimSpace(input.RegistrationToken) == "" {
		return Node{}, nil, fmt.Errorf("节点注册令牌不能为空")
	}
	normalizedID := normalizeNodeID(nodeID)
	item, commands, err := s.repository.Heartbeat(ctx, normalizedID, hashToken(input.RegistrationToken), input)
	if err != nil {
		return Node{}, nil, err
	}
	if s.online != nil {
		if err := s.online.TouchNode(ctx, normalizedID, 90*time.Second); err != nil {
			return Node{}, nil, err
		}
	}
	if s.events != nil {
		s.events.NodeMetrics(item.TenantID, item.ID, item.Metrics)
		s.events.NodeStatus(item.TenantID, item.ID, item.Status)
	}
	return item, commands, nil
}

func (s *Service) Offline(ctx context.Context, nodeID, registrationToken, reason string) (Node, error) {
	if strings.TrimSpace(registrationToken) == "" {
		return Node{}, fmt.Errorf("节点注册令牌不能为空")
	}
	item, err := s.repository.Offline(ctx, normalizeNodeID(nodeID), hashToken(registrationToken), strings.TrimSpace(reason))
	if err == nil && s.events != nil {
		s.events.NodeStatus(item.TenantID, item.ID, item.Status)
	}
	return item, err
}

func (s *Service) ReportDeployment(ctx context.Context, nodeID string, input DeploymentReport) (DeploymentUpdate, error) {
	if strings.TrimSpace(input.RegistrationToken) == "" || strings.TrimSpace(input.DeploymentID) == "" || strings.TrimSpace(input.Status) == "" {
		return DeploymentUpdate{}, fmt.Errorf("节点注册令牌、deploymentId 和 status 不能为空")
	}
	status := strings.ToLower(strings.TrimSpace(input.Status))
	status = map[string]string{"error": "failed", "rolling_back": "deploying"}[status]
	if status == "" {
		status = strings.ToLower(strings.TrimSpace(input.Status))
	}
	if status != "pending" && status != "deploying" && status != "running" && status != "stopped" && status != "failed" && status != "removed" {
		return DeploymentUpdate{}, fmt.Errorf("部署状态无效")
	}
	input.Status = status
	item, err := s.repository.ReportDeployment(ctx, normalizeNodeID(nodeID), hashToken(input.RegistrationToken), input)
	if err == nil && s.events != nil {
		s.events.DeployStatus(item.TenantID, map[string]any{
			"deploymentId": item.ID, "nodeId": item.NodeID, "projectId": item.ProjectID, "status": item.Status,
			"errorMessage": item.ErrorMessage, "startedAt": item.StartedAt, "stoppedAt": item.StoppedAt, "removed": item.Status == "removed",
		})
	}
	return item, err
}

func normalizeNodeID(raw string) string {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "node-"))
	if parsed, err := uuid.Parse(raw); err == nil {
		return parsed.String()
	}
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(raw)).String()
}

func hashToken(raw string) string {
	hash := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(hash[:])
}
