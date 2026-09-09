package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
)

var (
	ErrNotFound      = errors.New("用户不存在")
	ErrAlreadyExists = errors.New("用户名已存在")
	ErrDeleteSelf    = errors.New("不能删除当前登录用户")
	ErrInvalidInput  = errors.New("用户资料无效")
	ErrModifySelf    = errors.New("不能修改当前登录用户的角色或状态")
)

type User struct {
	ID                string
	TenantID          string
	Username          string
	Email             string
	Phone             string
	FullName          string
	Avatar            string
	Role              string
	Status            string
	Preferences       map[string]any
	LastLoginAt       *time.Time
	LastLoginIP       string
	PasswordChangedAt *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Input struct {
	TenantID   *string
	Username   *string
	Password   *string
	Email      *string
	Phone      *string
	FullName   *string
	Role       *string
	Status     *string
	Gender     *string
	Attributes map[string]string
}

type ListFilter struct {
	Keyword string
	Role    string
	Status  string
	Page    int
	Limit   int
}

type Repository interface {
	List(ctx context.Context, tenantID string, filter ListFilter) ([]User, int64, error)
	Get(ctx context.Context, tenantID, userID string) (User, string, error)
	Create(ctx context.Context, item User, passwordHash string) (User, error)
	Update(ctx context.Context, item User) (User, error)
	UpdatePassword(ctx context.Context, tenantID, userID, passwordHash string) error
	Delete(ctx context.Context, tenantID, userID string) error
}

type TokenRevoker interface {
	RevokeUserRefreshTokens(context.Context, string) error
}

type Service struct {
	repository Repository
	revoker    TokenRevoker
}

func NewService(repository Repository, revoker TokenRevoker) *Service {
	return &Service{repository: repository, revoker: revoker}
}

func (s *Service) List(ctx context.Context, tenantID string, filter ListFilter) ([]User, int64, error) {
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	return s.repository.List(ctx, tenantID, filter)
}

func (s *Service) Create(ctx context.Context, actor auth.User, input Input) (User, error) {
	tenantID := actor.TenantID
	if input.TenantID != nil && actor.Role == "SUPER_ADMIN" && strings.TrimSpace(*input.TenantID) != "" {
		tenantID = strings.TrimSpace(*input.TenantID)
	}
	item := User{TenantID: tenantID, Status: "active", Preferences: map[string]any{}}
	applyInput(&item, input)
	if input.Role == nil || strings.TrimSpace(*input.Role) == "" {
		return User{}, fmt.Errorf("%w: 请选择角色", ErrInvalidInput)
	}
	if err := validateUser(item); err != nil {
		return User{}, err
	}
	item.Role = strings.ToUpper(strings.TrimSpace(item.Role))
	if !auth.CanAssignRole(actor.Role, item.Role) {
		return User{}, auth.ErrPermissionDenied
	}
	if input.Password == nil || strings.TrimSpace(*input.Password) == "" {
		return User{}, fmt.Errorf("%w: 初始密码不能为空", ErrInvalidInput)
	}
	hash, err := auth.HashPassword(*input.Password)
	if err != nil {
		return User{}, err
	}
	return s.repository.Create(ctx, item, hash)
}

func (s *Service) Update(ctx context.Context, actor auth.User, userID string, input Input) (User, error) {
	item, _, err := s.repository.Get(ctx, actor.TenantID, userID)
	if err != nil {
		return User{}, err
	}
	if actor.ID == userID && (input.Role != nil || input.Status != nil) {
		return User{}, ErrModifySelf
	}
	if input.Role != nil {
		targetRole := strings.ToUpper(strings.TrimSpace(*input.Role))
		if !auth.CanAssignRole(actor.Role, targetRole) {
			return User{}, auth.ErrPermissionDenied
		}
		input.Role = &targetRole
	}
	if input.Username != nil && strings.TrimSpace(*input.Username) != item.Username {
		return User{}, fmt.Errorf("%w: 用户名不能修改", ErrInvalidInput)
	}
	if input.Password != nil || input.TenantID != nil {
		return User{}, fmt.Errorf("%w: 请使用专门的密码重置入口", ErrInvalidInput)
	}
	applyInput(&item, input)
	if err := validateUser(item); err != nil {
		return User{}, err
	}
	return s.repository.Update(ctx, item)
}

func (s *Service) UpdatePassword(ctx context.Context, actor auth.User, userID, password string) error {
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("%w: 临时密码不能为空", ErrInvalidInput)
	}
	if _, _, err := s.repository.Get(ctx, actor.TenantID, userID); err != nil {
		return err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	if err := s.repository.UpdatePassword(ctx, actor.TenantID, userID, hash); err != nil {
		return err
	}
	return s.revoker.RevokeUserRefreshTokens(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, actor auth.User, userID string) error {
	if actor.ID == userID {
		return ErrDeleteSelf
	}
	return s.repository.Delete(ctx, actor.TenantID, userID)
}

func applyInput(item *User, input Input) {
	if input.Username != nil {
		item.Username = strings.TrimSpace(*input.Username)
	}
	if input.Email != nil {
		item.Email = strings.TrimSpace(*input.Email)
	}
	if input.Phone != nil {
		item.Phone = strings.TrimSpace(*input.Phone)
	}
	if input.FullName != nil {
		item.FullName = strings.TrimSpace(*input.FullName)
	}
	if input.Role != nil {
		item.Role = strings.TrimSpace(*input.Role)
	}
	if input.Status != nil {
		item.Status = strings.TrimSpace(*input.Status)
	}

	if input.Gender != nil || input.Attributes != nil {
		preferences := make(map[string]any, len(item.Preferences)+1)
		for key, value := range item.Preferences {
			preferences[key] = value
		}
		profile := extendedProfile(*item)
		if input.Gender != nil {
			profile["gender"] = strings.TrimSpace(*input.Gender)
		}
		if input.Attributes != nil {
			profile["attributes"] = input.Attributes
		}
		preferences["profile"] = profile
		item.Preferences = preferences
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
