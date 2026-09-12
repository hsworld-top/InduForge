package projectfile

import (
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/indu-forge/dev_core/internal/objectstore"
	"github.com/indu-forge/dev_core/internal/project"
)

var (
	ErrNotFound    = fmt.Errorf("工程对象文件不存在")
	ErrConflict    = fmt.Errorf("同目录下已存在同名文件")
	ErrInvalidName = fmt.Errorf("文件名无效")
	ErrInvalidPath = fmt.Errorf("目录路径无效")
	ErrTooLarge    = fmt.Errorf("文件超过对象库上传限制")
)

const MaxFileSize int64 = 2 << 30

type Store interface {
	Put(context.Context, string, io.Reader, int64, string) (objectstore.ObjectRef, error)
	Open(context.Context, string) (objectstore.ObjectReader, error)
	Delete(context.Context, string) error
}

type Service struct {
	repository *Repository
	projects   *project.Service
	objects    Store
}

func NewService(repository *Repository, projects *project.Service, objects Store) *Service {
	return &Service{repository: repository, projects: projects, objects: objects}
}

func (s *Service) List(ctx context.Context, actor auth.User, projectID string, filter ListFilter) ([]File, int64, error) {
	if _, err := s.projects.Get(ctx, actor, projectID); err != nil {
		return nil, 0, err
	}
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	items, total, err := s.repository.List(ctx, actor.TenantID, projectID, filter)
	if err != nil {
		return nil, 0, err
	}
	return publicItems(projectID, items), total, nil
}

// ListPublicMetadata 返回可写入工程上下文的公开元数据，不包含对象存储内部键。
func (s *Service) ListPublicMetadata(ctx context.Context, actor auth.User, projectID string) ([]File, error) {
	items := make([]File, 0)
	for page := 1; ; page++ {
		batch, total, err := s.List(ctx, actor, projectID, ListFilter{Page: page, Limit: 100})
		if err != nil {
			return nil, err
		}
		items = append(items, batch...)
		if int64(len(items)) >= total || len(batch) == 0 {
			return items, nil
		}
	}
}

func (s *Service) Upload(ctx context.Context, actor auth.User, projectID, pathValue, name, contentType string, size int64, reader io.Reader) (File, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectWrite); err != nil {
		return File{}, err
	}
	if _, err := s.projects.Get(ctx, actor, projectID); err != nil {
		return File{}, err
	}
	pathValue, err := validatePath(pathValue)
	if err != nil {
		return File{}, err
	}
	name, err = validateName(name)
	if err != nil {
		return File{}, err
	}
	if size <= 0 || size > MaxFileSize {
		return File{}, ErrTooLarge
	}
	contentType = normalizeContentType(contentType, name)
	key := fmt.Sprintf("projects/%s/files/%s/%s", projectID, uuid.NewString(), name)
	ref, err := s.objects.Put(ctx, key, reader, size, contentType)
	if err != nil {
		return File{}, err
	}
	item, err := s.repository.Create(ctx, record{File: File{ProjectID: projectID, Path: pathValue, Name: name, ContentType: contentType, Size: ref.Size}, TenantID: actor.TenantID, ObjectKey: ref.Key, CreatedBy: actor.ID})
	if err != nil {
		_ = s.objects.Delete(ctx, key)
		return File{}, err
	}
	return publicItem(projectID, item), nil
}

func (s *Service) Get(ctx context.Context, actor auth.User, projectID, id string) (File, objectstore.ObjectReader, error) {
	if _, err := s.projects.Get(ctx, actor, projectID); err != nil {
		return File{}, objectstore.ObjectReader{}, err
	}
	item, err := s.repository.Get(ctx, actor.TenantID, projectID, id)
	if err != nil {
		return File{}, objectstore.ObjectReader{}, err
	}
	reader, err := s.objects.Open(ctx, item.ObjectKey)
	if err != nil {
		return File{}, objectstore.ObjectReader{}, err
	}
	return publicItem(projectID, item), reader, nil
}

func (s *Service) Rename(ctx context.Context, actor auth.User, projectID, id, pathValue, name string) (File, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectWrite); err != nil {
		return File{}, err
	}
	if _, err := s.projects.Get(ctx, actor, projectID); err != nil {
		return File{}, err
	}
	pathValue, err := validatePath(pathValue)
	if err != nil {
		return File{}, err
	}
	name, err = validateName(name)
	if err != nil {
		return File{}, err
	}
	item, err := s.repository.Update(ctx, actor.TenantID, projectID, id, pathValue, name, actor.ID)
	if err != nil {
		return File{}, err
	}
	return publicItem(projectID, item), nil
}

func (s *Service) Replace(ctx context.Context, actor auth.User, projectID, id, pathValue, name, contentType string, size int64, reader io.Reader) (File, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectWrite); err != nil {
		return File{}, err
	}
	if _, err := s.projects.Get(ctx, actor, projectID); err != nil {
		return File{}, err
	}
	pathValue, err := validatePath(pathValue)
	if err != nil {
		return File{}, err
	}
	name, err = validateName(name)
	if err != nil {
		return File{}, err
	}
	if size <= 0 || size > MaxFileSize {
		return File{}, ErrTooLarge
	}
	contentType = normalizeContentType(contentType, name)
	key := fmt.Sprintf("projects/%s/files/%s/%s", projectID, uuid.NewString(), name)
	ref, err := s.objects.Put(ctx, key, reader, size, contentType)
	if err != nil {
		return File{}, err
	}
	item, oldKey, err := s.repository.Replace(ctx, actor.TenantID, projectID, id, pathValue, name, ref.Size, contentType, ref.Key, actor.ID)
	if err != nil {
		_ = s.objects.Delete(ctx, key)
		return File{}, err
	}
	_ = s.objects.Delete(ctx, oldKey)
	return publicItem(projectID, item), nil
}

func (s *Service) Delete(ctx context.Context, actor auth.User, projectID, id string) error {
	if err := auth.RequireCapability(actor, auth.CapabilityProjectWrite); err != nil {
		return err
	}
	if _, err := s.projects.Get(ctx, actor, projectID); err != nil {
		return err
	}
	key, err := s.repository.Delete(ctx, actor.TenantID, projectID, id)
	if err != nil {
		return err
	}
	return s.objects.Delete(ctx, key)
}

func publicItems(projectID string, items []record) []File {
	out := make([]File, 0, len(items))
	for _, item := range items {
		out = append(out, publicItem(projectID, item))
	}
	return out
}
func publicItem(projectID string, item record) File {
	item.File.Entry = fmt.Sprintf("/api/v1/projects/%s/files/%s/content", projectID, item.ID)
	return item.File
}
func validateName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, "/\\\r\n") || len([]rune(value)) > 255 {
		return "", ErrInvalidName
	}
	return value, nil
}
func validatePath(value string) (string, error) {
	value = normalizePath(value)
	if len(value) > 512 || strings.HasPrefix(value, "../") || strings.Contains(value, "/../") || strings.HasSuffix(value, "/..") || strings.Contains(value, "\\") {
		return "", ErrInvalidPath
	}
	return value, nil
}
func normalizeContentType(value, name string) string {
	value = strings.TrimSpace(strings.Split(value, ";")[0])
	if value == "" || value == "application/octet-stream" {
		if detected := mime.TypeByExtension(strings.ToLower(filepath.Ext(name))); detected != "" {
			return detected
		}
	}
	return valueOrDefault(value, "application/octet-stream")
}
func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
func normalizePage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
