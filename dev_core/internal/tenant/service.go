package tenant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/objectstore"
)

var (
	ErrNotFound      = errors.New("租户不存在")
	ErrAlreadyExists = errors.New("租户编码已存在")
	ErrNoteNotFound  = errors.New("便签不存在")
)

type Tenant struct {
	Initialized              bool
	IsDefault                bool
	AdminUsername            string
	AdminUserID              string
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
	AdminUsername            *string
	AdminPassword            *string
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

type Service struct {
	repository Repository
	objects    ObjectStore
}

func NewService(repository Repository, objects ObjectStore) *Service {
	return &Service{repository: repository, objects: objects}
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
	username, passwordHash, err := administratorCredentials(input.AdminUsername, input.AdminPassword)
	if err != nil {
		return Tenant{}, err
	}
	created, err := s.repository.Create(ctx, item, username, passwordHash)
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
	if err := validateBrandKeys(item); err != nil {
		return Tenant{}, err
	}
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
	if item.IsDefault {
		return fmt.Errorf("默认租户不可删除")
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
	if assetType != "logo" && assetType != "background" && assetType != "captcha" {
		return UploadedAsset{}, fmt.Errorf("文件类型仅支持 logo、background 或 captcha")
	}
	if size <= 0 || size > 16<<20 {
		return UploadedAsset{}, fmt.Errorf("上传文件不能超过 16MB")
	}
	extension := strings.ToLower(filepath.Ext(fileName))
	allowed := map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".webp": "image/webp", ".svg": "image/svg+xml"}
	expectedType, ok := allowed[extension]
	if !ok || contentType != expectedType || (assetType != "logo" && extension == ".svg") {
		return UploadedAsset{}, fmt.Errorf("上传文件扩展名或内容类型不受支持")
	}
	if assetType == "captcha" {
		raw, err := io.ReadAll(io.LimitReader(reader, 5<<20+1))
		if err != nil || len(raw) > 5<<20 {
			return UploadedAsset{}, fmt.Errorf("验证码背景不能超过 5MB")
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil || cfg.Width < 300 || cfg.Height < 180 || int64(cfg.Width)*int64(cfg.Height) > 16_000_000 {
			return UploadedAsset{}, fmt.Errorf("验证码背景至少 300×180，且不能超过 1600 万像素")
		}
		if _, _, err = image.Decode(bytes.NewReader(raw)); err != nil {
			return UploadedAsset{}, fmt.Errorf("验证码背景图片无法读取")
		}
		reader = bytes.NewReader(raw)
		size = int64(len(raw))
	}
	key := fmt.Sprintf("tenants/%s/%s/%d-%s", tenantID, assetType, time.Now().UnixNano(), sanitizeFileName(fileName))
	ref, err := s.objects.Put(ctx, key, reader, size, contentType)
	if err != nil {
		return UploadedAsset{}, err
	}
	return UploadedAsset{ObjectKey: ref.Key}, nil
}

func (s *Service) hydrateAssets(ctx context.Context, item *Tenant) error {
	item.LogoURL = objectstore.BrandURL(item.Code, item.LogoObjectKey)
	item.LoginBackgroundURL = objectstore.BrandURL(item.Code, item.LoginBackgroundObjectKey)
	return nil
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

func administratorCredentials(username, password *string) (string, string, error) {
	if username == nil || password == nil || strings.TrimSpace(*username) == "" || utf8.RuneCountInString(*password) < 8 {
		return "", "", fmt.Errorf("管理员账号不能为空，密码至少8位")
	}
	name := strings.TrimSpace(*username)
	if utf8.RuneCountInString(name) > 50 {
		return "", "", fmt.Errorf("管理员账号不能超过50个字符")
	}
	hash, err := auth.HashPassword(*password)
	return name, hash, err
}
func (s *Service) Initialize(ctx context.Context, identifier string, input Input) (Tenant, error) {
	name, hash, err := administratorCredentials(input.AdminUsername, input.AdminPassword)
	if err != nil {
		return Tenant{}, err
	}
	repo, ok := s.repository.(interface {
		Initialize(context.Context, string, string, string) (Tenant, error)
	})
	if !ok {
		return Tenant{}, fmt.Errorf("租户初始化不可用")
	}
	return repo.Initialize(ctx, identifier, name, hash)
}
func (s *Service) ResetAdminPassword(ctx context.Context, identifier, password string) (Tenant, error) {
	if utf8.RuneCountInString(password) < 8 {
		return Tenant{}, fmt.Errorf("管理员密码至少8位")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return Tenant{}, err
	}
	repo, ok := s.repository.(interface {
		ResetAdminPassword(context.Context, string, string) (Tenant, error)
	})
	if !ok {
		return Tenant{}, fmt.Errorf("租户管理员重置不可用")
	}
	return repo.ResetAdminPassword(ctx, identifier, hash)
}

func validateBrandKeys(item Tenant) error {
	valid := func(key, kind string) bool {
		return key == "" || strings.HasPrefix(key, "tenants/"+item.ID+"/"+kind+"/") && !strings.Contains(key, "..")
	}
	if !valid(item.LogoObjectKey, "logo") || !valid(item.LoginBackgroundObjectKey, "background") {
		return fmt.Errorf("品牌图片不属于当前组织")
	}
	if raw, ok := item.Settings["captchaBackgrounds"]; ok {
		data, err := json.Marshal(raw)
		if err != nil {
			return err
		}
		var keys []string
		if json.Unmarshal(data, &keys) != nil || len(keys) > 10 {
			return fmt.Errorf("验证码背景最多 10 张")
		}
		for _, key := range keys {
			if key == "" || !valid(key, "captcha") {
				return fmt.Errorf("验证码背景不属于当前组织")
			}
		}
	}
	return nil
}
