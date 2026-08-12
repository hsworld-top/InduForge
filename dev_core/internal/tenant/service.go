package tenant

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/objectstore"
)

var (
	ErrNotFound      = errors.New("租户不存在")
	ErrAlreadyExists = errors.New("租户编码已存在")
	ErrNoteNotFound  = errors.New("便签不存在")
)

type Tenant struct {
	ID                       string
	Name                     string
	Code                     string
	Description              string
	Status                   string
	ContactEmail             string
	ContactPhone             string
	MaxUsers                 int32
	MaxProjects              int32
	MaxStorage               int64
	UsedStorage              int64
	LogoObjectKey            string
	LoginBackgroundObjectKey string
	LogoURL                  string
	LoginBackgroundURL       string
	CompanyName              string
	CompanyAddress           string
	CompanyPhone             string
	CompanyWebsite           string
	Settings                 map[string]any
	ExpiresAt                *time.Time
	CreatedAt                time.Time
	UpdatedAt                time.Time
	UserCount                int64
	ProjectCount             int64
}

type Input struct {
	Name                     *string
	Code                     *string
	Description              *string
	Status                   *string
	ContactEmail             *string
	ContactPhone             *string
	MaxUsers                 *int32
	MaxProjects              *int32
	MaxStorage               *int64
	LogoObjectKey            *string
	LoginBackgroundObjectKey *string
	CompanyName              *string
	CompanyAddress           *string
	CompanyPhone             *string
	CompanyWebsite           *string
	Settings                 map[string]any
	ExpiresAt                *time.Time
}

type Note struct {
	ID                string
	TenantID          string
	Content           string
	CreatedBy         string
	UpdatedBy         string
	CreatedByUsername string
	UpdatedByUsername string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ListFilter struct {
	Keyword string
	Status  string
	Page    int
	Limit   int
}

type Repository interface {
	List(ctx context.Context, filter ListFilter) ([]Tenant, int64, error)
	Get(ctx context.Context, identifier string) (Tenant, error)
	Create(ctx context.Context, item Tenant, adminUsername, adminPasswordHash string) (Tenant, error)
	Update(ctx context.Context, item Tenant) (Tenant, error)
	Delete(ctx context.Context, tenantID string) error
	SetStatus(ctx context.Context, tenantID, status string) (Tenant, error)
	ListNotes(ctx context.Context, tenantID string) ([]Note, error)
	CreateNote(ctx context.Context, tenantID, userID, content string) (Note, error)
	UpdateNote(ctx context.Context, tenantID, userID, noteID, content string) (Note, error)
	DeleteNote(ctx context.Context, tenantID, noteID string) error
}

type ObjectStore interface {
	Put(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) (objectstore.ObjectRef, error)
	PresignGet(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
	Delete(ctx context.Context, objectKey string) error
}

type UploadedAsset struct {
	ObjectKey string
	URL       string
}

type ServiceConfig struct {
	DefaultAdminUsername string
	DefaultAdminPassword string
}

type Service struct {
	repository Repository
	objects    ObjectStore
	config     ServiceConfig
}

func NewService(repository Repository, objects ObjectStore, config ServiceConfig) *Service {
	if config.DefaultAdminUsername == "" {
		config.DefaultAdminUsername = "admin"
	}
	if config.DefaultAdminPassword == "" {
		config.DefaultAdminPassword = "admin123"
	}
	return &Service{repository: repository, objects: objects, config: config}
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Tenant, int64, error) {
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	items, total, err := s.repository.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	for index := range items {
		if err := s.hydrateAssets(ctx, &items[index]); err != nil {
			return nil, 0, err
		}
	}
	return items, total, nil
}

func (s *Service) Get(ctx context.Context, identifier string) (Tenant, error) {
	item, err := s.repository.Get(ctx, strings.TrimSpace(identifier))
	if err != nil {
		return Tenant{}, err
	}
	if err := s.hydrateAssets(ctx, &item); err != nil {
		return Tenant{}, err
	}
	return item, nil
}

func (s *Service) Create(ctx context.Context, input Input) (Tenant, error) {
	item := Tenant{Status: "active", MaxUsers: 100, MaxProjects: 50, MaxStorage: 10 * 1024 * 1024 * 1024, Settings: map[string]any{}}
	applyInput(&item, input)
	if strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.Code) == "" {
		return Tenant{}, fmt.Errorf("租户名称和编码不能为空")
	}
	passwordHash, err := auth.HashPassword(s.config.DefaultAdminPassword)
	if err != nil {
		return Tenant{}, err
	}
	created, err := s.repository.Create(ctx, item, s.config.DefaultAdminUsername, passwordHash)
	if err != nil {
		return Tenant{}, err
	}
	if err := s.hydrateAssets(ctx, &created); err != nil {
		return Tenant{}, err
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, identifier string, input Input) (Tenant, error) {
	item, err := s.repository.Get(ctx, identifier)
	if err != nil {
		return Tenant{}, err
	}
	applyInput(&item, input)
	updated, err := s.repository.Update(ctx, item)
	if err != nil {
		return Tenant{}, err
	}
	if err := s.hydrateAssets(ctx, &updated); err != nil {
		return Tenant{}, err
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, identifier string) error {
	item, err := s.repository.Get(ctx, identifier)
	if err != nil {
		return err
	}
	return s.repository.Delete(ctx, item.ID)
}

func (s *Service) SetStatus(ctx context.Context, identifier, status string) (Tenant, error) {
	item, err := s.repository.Get(ctx, identifier)
	if err != nil {
		return Tenant{}, err
	}
	updated, err := s.repository.SetStatus(ctx, item.ID, status)
	if err != nil {
		return Tenant{}, err
	}
	if err := s.hydrateAssets(ctx, &updated); err != nil {
		return Tenant{}, err
	}
	return updated, nil
}

func (s *Service) ListNotes(ctx context.Context, tenantID string) ([]Note, error) {
	return s.repository.ListNotes(ctx, tenantID)
}

func (s *Service) CreateNote(ctx context.Context, tenantID, userID, content string) (Note, error) {
	content = strings.TrimSpace(content)
	if content == "" || len([]rune(content)) > 2000 {
		return Note{}, fmt.Errorf("便签内容不能为空且不能超过 2000 个字符")
	}
	return s.repository.CreateNote(ctx, tenantID, userID, content)
}

func (s *Service) UpdateNote(ctx context.Context, tenantID, userID, noteID, content string) (Note, error) {
	content = strings.TrimSpace(content)
	if content == "" || len([]rune(content)) > 2000 {
		return Note{}, fmt.Errorf("便签内容不能为空且不能超过 2000 个字符")
	}
	return s.repository.UpdateNote(ctx, tenantID, userID, noteID, content)
}

func (s *Service) DeleteNote(ctx context.Context, tenantID, noteID string) error {
	return s.repository.DeleteNote(ctx, tenantID, noteID)
}

func (s *Service) Upload(ctx context.Context, tenantID, assetType, fileName, contentType string, reader io.Reader, size int64) (UploadedAsset, error) {
	if s.objects == nil {
		return UploadedAsset{}, fmt.Errorf("对象存储未配置")
	}
	if assetType != "logo" && assetType != "background" {
		return UploadedAsset{}, fmt.Errorf("文件类型仅支持 logo 或 background")
	}
	if size <= 0 || size > 16<<20 {
		return UploadedAsset{}, fmt.Errorf("上传文件不能超过 16MB")
	}
	extension := strings.ToLower(filepath.Ext(fileName))
	allowed := map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".webp": "image/webp", ".svg": "image/svg+xml"}
	expectedType, ok := allowed[extension]
	if !ok || contentType != expectedType || (assetType == "background" && extension == ".svg") {
		return UploadedAsset{}, fmt.Errorf("上传文件扩展名或内容类型不受支持")
	}
	key := fmt.Sprintf("tenants/%s/%s/%d-%s", tenantID, assetType, time.Now().UnixNano(), sanitizeFileName(fileName))
	ref, err := s.objects.Put(ctx, key, reader, size, contentType)
	if err != nil {
		return UploadedAsset{}, err
	}
	url, err := s.objects.PresignGet(ctx, ref.Key, time.Hour)
	if err != nil {
		_ = s.objects.Delete(ctx, ref.Key)
		return UploadedAsset{}, err
	}
	return UploadedAsset{ObjectKey: ref.Key, URL: url}, nil
}

func (s *Service) hydrateAssets(ctx context.Context, item *Tenant) error {
	if s.objects == nil {
		return nil
	}
	var err error
	if item.LogoObjectKey != "" {
		item.LogoURL, err = s.objects.PresignGet(ctx, item.LogoObjectKey, time.Hour)
		if err != nil {
			return err
		}
	}
	if item.LoginBackgroundObjectKey != "" {
		item.LoginBackgroundURL, err = s.objects.PresignGet(ctx, item.LoginBackgroundObjectKey, time.Hour)
	}
	return err
}

func applyInput(item *Tenant, input Input) {
	if input.Name != nil {
		item.Name = strings.TrimSpace(*input.Name)
	}
	if input.Code != nil {
		item.Code = strings.TrimSpace(*input.Code)
	}
	if input.Description != nil {
		item.Description = *input.Description
	}
	if input.Status != nil {
		item.Status = *input.Status
	}
	if input.ContactEmail != nil {
		item.ContactEmail = *input.ContactEmail
	}
	if input.ContactPhone != nil {
		item.ContactPhone = *input.ContactPhone
	}
	if input.MaxUsers != nil {
		item.MaxUsers = *input.MaxUsers
	}
	if input.MaxProjects != nil {
		item.MaxProjects = *input.MaxProjects
	}
	if input.MaxStorage != nil {
		item.MaxStorage = *input.MaxStorage
	}
	if input.LogoObjectKey != nil {
		item.LogoObjectKey = *input.LogoObjectKey
	}
	if input.LoginBackgroundObjectKey != nil {
		item.LoginBackgroundObjectKey = *input.LoginBackgroundObjectKey
	}
	if input.CompanyName != nil {
		item.CompanyName = *input.CompanyName
	}
	if input.CompanyAddress != nil {
		item.CompanyAddress = *input.CompanyAddress
	}
	if input.CompanyPhone != nil {
		item.CompanyPhone = *input.CompanyPhone
	}
	if input.CompanyWebsite != nil {
		item.CompanyWebsite = *input.CompanyWebsite
	}
	if input.Settings != nil {
		item.Settings = input.Settings
	}
	if input.ExpiresAt != nil {
		item.ExpiresAt = input.ExpiresAt
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

func sanitizeFileName(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\", "-")
	value = strings.ReplaceAll(value, "/", "-")
	if value == "" {
		return "upload.bin"
	}
	return value
}
