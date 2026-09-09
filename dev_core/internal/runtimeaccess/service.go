package runtimeaccess

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/project"
)

var (
	ErrUserNotFound    = errors.New("运行用户不存在")
	ErrRoleNotFound    = errors.New("运行角色不存在")
	ErrProjectNotFound = errors.New("工程不存在")
	ErrAlreadyExists   = errors.New("运行用户或角色已存在")
	ErrBuiltin         = errors.New("内置用户或角色不能删除")
)

type Role struct {
	ID           string
	ProjectID    string
	Code         string
	Name         string
	Description  string
	Status       string
	IsBuiltin    bool
	Capabilities []string
	UserCount    int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
type RuntimeUser struct {
	ID                string
	ProjectID         string
	Username          string
	DisplayName       string
	Email             string
	Status            string
	IsBuiltinAdmin    bool
	Roles             []Role
	LastLoginAt       *time.Time
	PasswordChangedAt *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
type RoleInput struct {
	Code         string
	Name         string
	Description  string
	Status       string
	Capabilities []string
}
type UserInput struct {
	Username    string
	Password    string
	DisplayName string
	Email       string
	Status      string
	RoleIDs     []string
}

type ProjectAccess struct {
	ID         string
	TenantID   string
	CreatedBy  string
	Visibility string
}

type ListQuery struct {
	Keyword     string
	Page, Limit int
}
type Repository interface {
	PageRoles(context.Context, string, string, ListQuery) ([]Role, int64, error)
	PageUsers(context.Context, string, string, ListQuery) ([]RuntimeUser, int64, error)
	GetProjectAccess(context.Context, string, string) (ProjectAccess, error)
	ListRoles(context.Context, string, string) ([]Role, error)
	GetRole(context.Context, string, string, string) (Role, error)
	CreateRole(context.Context, string, string, RoleInput) (Role, error)
	UpdateRole(context.Context, string, string, string, RoleInput) (Role, error)
	DeleteRole(context.Context, string, string, string) error
	ListUsers(context.Context, string, string) ([]RuntimeUser, error)
	GetUser(context.Context, string, string, string) (RuntimeUser, error)
	CreateUser(context.Context, string, string, UserInput, string) (RuntimeUser, error)
	UpdateUserStatus(context.Context, string, string, string, string, string) (RuntimeUser, error)
	DeleteUser(context.Context, string, string, string) error
	ReplaceUserRoles(context.Context, string, string, string, []string) error
	UpdateUserPassword(context.Context, string, string, string, string, string) error
}
type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }
func (s *Service) PageRoles(ctx context.Context, actor auth.User, projectID string, query ListQuery) ([]Role, int64, error) {
	if err := s.requireProject(ctx, actor, projectID, auth.CapabilityProjectRead); err != nil {
		return nil, 0, err
	}
	return s.repository.PageRoles(ctx, actor.TenantID, projectID, query)
}
func (s *Service) PageUsers(ctx context.Context, actor auth.User, projectID string, query ListQuery) ([]RuntimeUser, int64, error) {
	if err := s.requireProject(ctx, actor, projectID, auth.CapabilityProjectRead); err != nil {
		return nil, 0, err
	}
	return s.repository.PageUsers(ctx, actor.TenantID, projectID, query)
}
func (s *Service) ListRoles(ctx context.Context, actor auth.User, projectID string) ([]Role, error) {
	if err := s.requireProject(ctx, actor, projectID, auth.CapabilityProjectRead); err != nil {
		return nil, err
	}
	return s.repository.ListRoles(ctx, actor.TenantID, projectID)
}
func (s *Service) CreateRole(ctx context.Context, actor auth.User, projectID string, input RoleInput) (Role, error) {
	if err := s.requireProject(ctx, actor, projectID, auth.CapabilityRuntimeAccessManage); err != nil {
		return Role{}, err
	}
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	if input.Code == "" || input.Name == "" {
		return Role{}, fmt.Errorf("角色代码和名称不能为空")
	}
	if input.Status == "" {
		input.Status = "active"
	}
	return s.repository.CreateRole(ctx, projectID, actor.ID, input)
}
func (s *Service) UpdateRole(ctx context.Context, actor auth.User, projectID, roleID string, input RoleInput) (Role, error) {
	if err := s.requireProject(ctx, actor, projectID, auth.CapabilityRuntimeAccessManage); err != nil {
		return Role{}, err
	}
	current, err := s.repository.GetRole(ctx, actor.TenantID, projectID, roleID)
	if err != nil {
		return Role{}, err
	}
	if current.IsBuiltin {
		return Role{}, ErrBuiltin
	}
	if input.Code == "" {
		input.Code = current.Code
	}
	if input.Name == "" {
		input.Name = current.Name
	}
	if input.Status == "" {
		input.Status = current.Status
	}
	return s.repository.UpdateRole(ctx, projectID, roleID, actor.ID, input)
}
func (s *Service) DeleteRole(ctx context.Context, actor auth.User, projectID, roleID string) error {
	if err := s.requireProject(ctx, actor, projectID, auth.CapabilityRuntimeAccessManage); err != nil {
		return err
	}
	role, err := s.repository.GetRole(ctx, actor.TenantID, projectID, roleID)
	if err != nil {
		return err
	}
	if role.IsBuiltin {
		return ErrBuiltin
	}
	return s.repository.DeleteRole(ctx, actor.TenantID, projectID, roleID)
}
func (s *Service) ListUsers(ctx context.Context, actor auth.User, projectID string) ([]RuntimeUser, error) {
	if err := s.requireProject(ctx, actor, projectID, auth.CapabilityProjectRead); err != nil {
		return nil, err
	}
	return s.repository.ListUsers(ctx, actor.TenantID, projectID)
}
func (s *Service) CreateUser(ctx context.Context, actor auth.User, projectID string, input UserInput) (RuntimeUser, error) {
	if err := s.requireProject(ctx, actor, projectID, auth.CapabilityRuntimeAccessManage); err != nil {
		return RuntimeUser{}, err
	}
	input.Username = strings.TrimSpace(input.Username)
	if input.Username == "" || len(input.Password) < 8 {
		return RuntimeUser{}, fmt.Errorf("用户名不能为空且密码至少 8 位")
	}
	if input.Status == "" {
		input.Status = "active"
	}
	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		return RuntimeUser{}, err
	}
	return s.repository.CreateUser(ctx, projectID, actor.ID, input, hash)
}
func (s *Service) UpdateUserStatus(ctx context.Context, actor auth.User, projectID, userID, status string) (RuntimeUser, error) {
	if err := s.requireProject(ctx, actor, projectID, auth.CapabilityRuntimeAccessManage); err != nil {
		return RuntimeUser{}, err
	}
	if status != "active" && status != "disabled" {
		return RuntimeUser{}, fmt.Errorf("运行用户状态无效")
	}
	user, err := s.repository.GetUser(ctx, actor.TenantID, projectID, userID)
	if err != nil {
		return RuntimeUser{}, err
	}
	if user.IsBuiltinAdmin && status != "active" {
		return RuntimeUser{}, fmt.Errorf("%w：不能停用内置管理员", auth.ErrPermissionDenied)
	}
	return s.repository.UpdateUserStatus(ctx, actor.TenantID, projectID, userID, status, actor.ID)
}
func (s *Service) DeleteUser(ctx context.Context, actor auth.User, projectID, userID string) error {
	if err := s.requireProject(ctx, actor, projectID, auth.CapabilityRuntimeAccessManage); err != nil {
		return err
	}
	item, err := s.repository.GetUser(ctx, actor.TenantID, projectID, userID)
	if err != nil {
		return err
	}
	if item.IsBuiltinAdmin {
		return ErrBuiltin
	}
	return s.repository.DeleteUser(ctx, actor.TenantID, projectID, userID)
}
func (s *Service) ReplaceUserRoles(ctx context.Context, actor auth.User, projectID, userID string, roleIDs []string) error {
	if err := s.requireProject(ctx, actor, projectID, auth.CapabilityRuntimeAccessManage); err != nil {
		return err
	}
	return s.repository.ReplaceUserRoles(ctx, actor.TenantID, projectID, userID, roleIDs)
}
func (s *Service) ResetPassword(ctx context.Context, actor auth.User, projectID, userID, password string) error {
	if err := s.requireProject(ctx, actor, projectID, auth.CapabilityRuntimeAccessManage); err != nil {
		return err
	}
	if len(password) < 8 {
		return fmt.Errorf("新密码至少 8 位")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	return s.repository.UpdateUserPassword(ctx, actor.TenantID, projectID, userID, actor.ID, hash)
}

func (s *Service) requireProject(ctx context.Context, actor auth.User, projectID string, capability auth.Capability) error {
	access, err := s.repository.GetProjectAccess(ctx, actor.TenantID, projectID)
	if err != nil {
		return err
	}
	return project.RequireCapability(actor, project.Project{
		ID: access.ID, TenantID: access.TenantID, CreatedBy: access.CreatedBy, Visibility: access.Visibility,
	}, capability)
}
