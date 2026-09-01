package project

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
)

var (
	ErrNotFound      = errors.New("工程不存在")
	ErrAlreadyExists = errors.New("工程编码已存在")
	ErrTagNotFound   = errors.New("工程标签不存在")
	ErrGroupNotFound = errors.New("工程分组不存在")
)

type Project struct {
	ID            string
	TenantID      string
	Name          string
	Code          string
	Description   string
	Icon          string
	WorkspacePath string
	Status        string
	Visibility    string
	CreatedBy     string
	CreatedByName string
	UpdatedBy     string
	Group         *Group
	Tags          []Tag
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Tag struct {
	ID           string
	TenantID     string
	Name         string
	Color        string
	Description  string
	SortOrder    int32
	ProjectCount int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
type Group struct {
	ID           string
	TenantID     string
	Name         string
	Description  string
	SortOrder    int32
	ProjectCount int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
type DeleteImpact struct {
	RuntimeUserCount int64
	RuntimeRoleCount int64
	DeploymentCount  int64
	VersionCount     int64
}

type ListFilter struct {
	Keyword         string
	Status          string
	Visibility      string
	GroupID         string
	TagID           string
	ActorID         string
	IsPlatformAdmin bool
	CanReadShared   bool
	Page            int
	Limit           int
}
type ProjectInput struct {
	Name        *string
	Description *string
	Icon        *string
	Visibility  *string
}
type TagInput struct {
	Name        *string
	Color       *string
	Description *string
	SortOrder   *int32
}
type GroupInput struct {
	Name        *string
	Description *string
	SortOrder   *int32
}

type Repository interface {
	List(ctx context.Context, tenantID string, filter ListFilter) ([]Project, int64, error)
	Get(ctx context.Context, tenantID, projectID string) (Project, error)
	Create(ctx context.Context, item Project, actor auth.User, runtimeAdminHash string) (Project, error)
	Update(ctx context.Context, item Project, actor auth.User) (Project, error)
	SetStatus(ctx context.Context, tenantID, projectID, status, userID string) (Project, error)
	Delete(ctx context.Context, tenantID, projectID, userID string) error
	DeleteImpact(ctx context.Context, tenantID, projectID string) (DeleteImpact, error)
	ListTags(ctx context.Context, tenantID, keyword string) ([]Tag, error)
	GetTag(ctx context.Context, tenantID, tagID string) (Tag, error)
	CreateTag(ctx context.Context, tenantID, userID string, item Tag) (Tag, error)
	UpdateTag(ctx context.Context, tenantID string, item Tag) (Tag, error)
	DeleteTag(ctx context.Context, tenantID, tagID string) error
	ReplaceTags(ctx context.Context, tenantID, projectID string, tagIDs []string) error
	ListGroups(ctx context.Context, tenantID, keyword string) ([]Group, error)
	GetGroup(ctx context.Context, tenantID, groupID string) (Group, error)
	CreateGroup(ctx context.Context, tenantID, userID string, item Group) (Group, error)
	UpdateGroup(ctx context.Context, tenantID string, item Group) (Group, error)
	DeleteGroup(ctx context.Context, tenantID, groupID string) error
	SetGroup(ctx context.Context, tenantID, projectID string, groupID *string) error
}

type Workspace interface {
	Initialize(projectID string) (string, error)
	Remove(path string) error
	Export(path string) (map[string]string, error)
	Import(projectID string, files map[string]string) (string, error)
}

// TenantBindingEnsurer 将控制面已持久化的项目归属同步到数据域。
type TenantBindingEnsurer interface {
	EnsureProjectTenantBinding(context.Context, string, string) error
}

type Service struct {
	repository             Repository
	workspace              Workspace
	defaultRuntimePassword string
	tenantBindingEnsurer   TenantBindingEnsurer
}

func (s *Service) SetTenantBindingEnsurer(ensurer TenantBindingEnsurer) {
	s.tenantBindingEnsurer = ensurer
}

func NewService(repository Repository, workspace Workspace, defaultRuntimePassword string) *Service {
	if defaultRuntimePassword == "" {
		defaultRuntimePassword = "admin123"
	}
	return &Service{repository: repository, workspace: workspace, defaultRuntimePassword: defaultRuntimePassword}
}

func (s *Service) List(ctx context.Context, actor auth.User, filter ListFilter) ([]Project, int64, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectRead); err != nil {
		return nil, 0, err
	}
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	filter.ActorID = actor.ID
	filter.IsPlatformAdmin = auth.IsPlatformAdmin(actor.Role)
	filter.CanReadShared = auth.HasCapability(actor.Role, auth.CapabilityProjectRead)
	return s.repository.List(ctx, actor.TenantID, filter)
}
func (s *Service) Get(ctx context.Context, actor auth.User, projectID string) (Project, error) {
	item, err := s.repository.Get(ctx, actor.TenantID, projectID)
	if err != nil {
		return Project{}, err
	}
	if !CanRead(actor, item) {
		return Project{}, auth.ErrPermissionDenied
	}
	return item, nil
}

func (s *Service) Create(ctx context.Context, actor auth.User, input ProjectInput) (Project, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectCreate); err != nil {
		return Project{}, err
	}
	item := Project{TenantID: actor.TenantID, Visibility: "private", Status: "active"}
	applyProjectInput(&item, input)
	item.Visibility = "private"
	if item.Name == "" {
		return Project{}, fmt.Errorf("工程名称不能为空")
	}
	item.ID = newID()
	// 工程编码是系统内部稳定标识，由服务端生成，避免前端输入与重复校验耦合。
	item.Code = generateProjectCode()
	workspacePath, err := s.workspace.Initialize(item.ID)
	if err != nil {
		return Project{}, err
	}
	item.WorkspacePath = workspacePath
	hash, err := auth.HashPassword(s.defaultRuntimePassword)
	if err != nil {
		_ = s.workspace.Remove(workspacePath)
		return Project{}, err
	}
	created, err := s.repository.Create(ctx, item, actor, hash)
	if err != nil {
		_ = s.workspace.Remove(workspacePath)
		return Project{}, err
	}
	if s.tenantBindingEnsurer == nil {
		slog.Default().Error("运维事件: 项目租户绑定同步失败", "projectId", created.ID, "reason", "数据服务内部客户端未配置")
		return Project{}, fmt.Errorf("工程已创建，但项目租户绑定同步失败，可稍后重试")
	}
	if err := s.tenantBindingEnsurer.EnsureProjectTenantBinding(ctx, created.ID, created.TenantID); err != nil {
		slog.Default().Error("运维事件: 项目租户绑定同步失败", "projectId", created.ID, "error", err)
		return Project{}, fmt.Errorf("工程已创建，但项目租户绑定同步失败，可稍后重试: %w", err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, actor auth.User, projectID string, input ProjectInput) (Project, error) {
	item, err := s.repository.Get(ctx, actor.TenantID, projectID)
	if err != nil {
		return Project{}, err
	}
	if err := RequireCapability(actor, item, auth.CapabilityProjectWrite); err != nil {
		return Project{}, err
	}
	if input.Visibility != nil && strings.TrimSpace(*input.Visibility) != item.Visibility {
		if err := RequireOwnerOrAdmin(actor, item, auth.CapabilityProjectShare); err != nil {
			return Project{}, err
		}
	}
	applyProjectInput(&item, input)
	if err := validateVisibility(item.Visibility); err != nil {
		return Project{}, err
	}
	return s.repository.Update(ctx, item, actor)
}
func (s *Service) Delete(ctx context.Context, actor auth.User, projectID string) error {
	item, err := s.repository.Get(ctx, actor.TenantID, projectID)
	if err != nil {
		return err
	}
	if err := RequireOwnerOrAdmin(actor, item, auth.CapabilityProjectDelete); err != nil {
		return err
	}
	if err := s.repository.Delete(ctx, actor.TenantID, projectID, actor.ID); err != nil {
		return err
	}
	return s.workspace.Remove(item.WorkspacePath)
}
func (s *Service) DeleteImpact(ctx context.Context, actor auth.User, projectID string) (DeleteImpact, error) {
	item, err := s.repository.Get(ctx, actor.TenantID, projectID)
	if err != nil {
		return DeleteImpact{}, err
	}
	if err := RequireOwnerOrAdmin(actor, item, auth.CapabilityProjectDelete); err != nil {
		return DeleteImpact{}, err
	}
	return s.repository.DeleteImpact(ctx, actor.TenantID, projectID)
}

func (s *Service) Operate(ctx context.Context, actor auth.User, projectID, operation string) (Project, error) {
	item, err := s.repository.Get(ctx, actor.TenantID, projectID)
	if err != nil {
		return Project{}, err
	}
	if err := RequireCapability(actor, item, auth.CapabilityProjectWrite); err != nil {
		return Project{}, err
	}
	switch operation {
	case "archive", "stop":
		return s.repository.SetStatus(ctx, actor.TenantID, projectID, "archived", actor.ID)
	case "restore", "start", "restart":
		return s.repository.SetStatus(ctx, actor.TenantID, projectID, "active", actor.ID)
	default:
		return Project{}, fmt.Errorf("不支持的工程操作: %s", operation)
	}
}

func (s *Service) Export(ctx context.Context, actor auth.User, projectID string) (map[string]any, error) {
	item, err := s.repository.Get(ctx, actor.TenantID, projectID)
	if err != nil {
		return nil, err
	}
	if err := RequireCapability(actor, item, auth.CapabilityProjectWrite); err != nil {
		return nil, err
	}
	files, err := s.workspace.Export(item.WorkspacePath)
	if err != nil {
		return nil, err
	}
	return map[string]any{"version": 1, "project": projectPayload(item), "workspace": files}, nil
}

func (s *Service) Import(ctx context.Context, actor auth.User, name string, payload map[string]any) (Project, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectCreate); err != nil {
		return Project{}, err
	}
	projectData, _ := payload["project"].(map[string]any)
	filesRaw, _ := payload["workspace"].(map[string]any)
	files := map[string]string{}
	for key, value := range filesRaw {
		if text, ok := value.(string); ok {
			files[key] = text
		}
	}
	item := Project{ID: newID(), TenantID: actor.TenantID, Name: strings.TrimSpace(name), Code: generateProjectCode(), Visibility: "private", Status: "active"}
	if item.Name == "" {
		item.Name, _ = projectData["name"].(string)
	}
	if item.Name == "" {
		return Project{}, fmt.Errorf("导入工程名称不能为空")
	}
	workspacePath, err := s.workspace.Import(item.ID, files)
	if err != nil {
		return Project{}, err
	}
	item.WorkspacePath = workspacePath
	hash, err := auth.HashPassword(s.defaultRuntimePassword)
	if err != nil {
		_ = s.workspace.Remove(workspacePath)
		return Project{}, err
	}
	created, err := s.repository.Create(ctx, item, actor, hash)
	if err != nil {
		_ = s.workspace.Remove(workspacePath)
		return Project{}, err
	}
	return created, nil
}

func (s *Service) ListTags(ctx context.Context, actor auth.User, keyword string) ([]Tag, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectRead); err != nil {
		return nil, err
	}
	return s.repository.ListTags(ctx, actor.TenantID, keyword)
}
func (s *Service) CreateTag(ctx context.Context, actor auth.User, input TagInput) (Tag, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectWrite); err != nil {
		return Tag{}, err
	}
	item := Tag{}
	applyTagInput(&item, input)
	if item.Name == "" {
		return Tag{}, fmt.Errorf("标签名称不能为空")
	}
	return s.repository.CreateTag(ctx, actor.TenantID, actor.ID, item)
}
func (s *Service) UpdateTag(ctx context.Context, actor auth.User, tagID string, input TagInput) (Tag, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectWrite); err != nil {
		return Tag{}, err
	}
	item, err := s.repository.GetTag(ctx, actor.TenantID, tagID)
	if err != nil {
		return Tag{}, err
	}
	applyTagInput(&item, input)
	return s.repository.UpdateTag(ctx, actor.TenantID, item)
}
func (s *Service) DeleteTag(ctx context.Context, actor auth.User, tagID string) error {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectWrite); err != nil {
		return err
	}
	return s.repository.DeleteTag(ctx, actor.TenantID, tagID)
}
func (s *Service) ReplaceTags(ctx context.Context, actor auth.User, projectID string, tagIDs []string) error {
	item, err := s.repository.Get(ctx, actor.TenantID, projectID)
	if err != nil {
		return err
	}
	if err := RequireCapability(actor, item, auth.CapabilityProjectWrite); err != nil {
		return err
	}
	return s.repository.ReplaceTags(ctx, actor.TenantID, projectID, tagIDs)
}
func (s *Service) ListGroups(ctx context.Context, actor auth.User, keyword string) ([]Group, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectRead); err != nil {
		return nil, err
	}
	return s.repository.ListGroups(ctx, actor.TenantID, keyword)
}
func (s *Service) CreateGroup(ctx context.Context, actor auth.User, input GroupInput) (Group, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectWrite); err != nil {
		return Group{}, err
	}
	item := Group{}
	applyGroupInput(&item, input)
	if item.Name == "" {
		return Group{}, fmt.Errorf("分组名称不能为空")
	}
	return s.repository.CreateGroup(ctx, actor.TenantID, actor.ID, item)
}
func (s *Service) UpdateGroup(ctx context.Context, actor auth.User, groupID string, input GroupInput) (Group, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectWrite); err != nil {
		return Group{}, err
	}
	item, err := s.repository.GetGroup(ctx, actor.TenantID, groupID)
	if err != nil {
		return Group{}, err
	}
	applyGroupInput(&item, input)
	return s.repository.UpdateGroup(ctx, actor.TenantID, item)
}
func (s *Service) DeleteGroup(ctx context.Context, actor auth.User, groupID string) error {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectWrite); err != nil {
		return err
	}
	return s.repository.DeleteGroup(ctx, actor.TenantID, groupID)
}
func (s *Service) SetGroup(ctx context.Context, actor auth.User, projectID string, groupID *string) error {
	item, err := s.repository.Get(ctx, actor.TenantID, projectID)
	if err != nil {
		return err
	}
	if err := RequireCapability(actor, item, auth.CapabilityProjectWrite); err != nil {
		return err
	}
	return s.repository.SetGroup(ctx, actor.TenantID, projectID, groupID)
}

func validateVisibility(visibility string) error {
	if visibility != "private" && visibility != "internal" {
		return fmt.Errorf("工程可见性无效")
	}
	return nil
}

func applyProjectInput(item *Project, input ProjectInput) {
	if input.Name != nil {
		item.Name = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		item.Description = *input.Description
	}
	if input.Icon != nil {
		item.Icon = *input.Icon
	}
	if input.Visibility != nil {
		item.Visibility = *input.Visibility
	}
}

func generateProjectCode() string {
	return "PRJ-" + strings.ToUpper(strings.ReplaceAll(newID(), "-", ""))
}
func applyTagInput(item *Tag, input TagInput) {
	if input.Name != nil {
		item.Name = strings.TrimSpace(*input.Name)
	}
	if input.Color != nil {
		item.Color = *input.Color
	}
	if input.Description != nil {
		item.Description = *input.Description
	}
	if input.SortOrder != nil {
		item.SortOrder = *input.SortOrder
	}
}
func applyGroupInput(item *Group, input GroupInput) {
	if input.Name != nil {
		item.Name = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		item.Description = *input.Description
	}
	if input.SortOrder != nil {
		item.SortOrder = *input.SortOrder
	}
}
func normalizePage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
