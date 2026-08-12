package auditlog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/indu-forge/dev_core/internal/auth"
)

var ErrNotFound = errors.New("日志不存在")

type Log struct {
	ID         string
	TenantID   string
	UserID     string
	Username   string
	FullName   string
	Level      string
	Action     string
	Resource   string
	ResourceID string
	Message    string
	RequestID  string
	Method     string
	Path       string
	Result     string
	IP         string
	UserAgent  string
	Metadata   map[string]any
	CreatedAt  time.Time
}

type Filter struct {
	Page      int
	Limit     int
	Level     string
	Action    string
	Resource  string
	UserID    string
	Keyword   string
	StartTime *time.Time
	EndTime   *time.Time
}

type Stats struct {
	LevelStats  map[string]int64
	TrendStats  []map[string]any
	ActionStats []map[string]any
}

type Repository interface {
	List(context.Context, string, Filter) ([]Log, int64, error)
	Get(context.Context, string, string) (Log, error)
	DeleteBefore(context.Context, string, time.Time) (int64, error)
	Export(context.Context, string, Filter) ([]Log, error)
	Stats(context.Context, string, *time.Time, *time.Time) (Stats, error)
	Recent(context.Context, string, int) ([]Log, error)
}

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) List(ctx context.Context, actor auth.User, filter Filter) ([]Log, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}
	filter.Level = strings.ToUpper(strings.TrimSpace(filter.Level))
	return s.repository.List(ctx, actor.TenantID, filter)
}

func (s *Service) Get(ctx context.Context, actor auth.User, id string) (Log, error) {
	return s.repository.Get(ctx, actor.TenantID, id)
}

func (s *Service) DeleteBefore(ctx context.Context, actor auth.User, before time.Time) (int64, error) {
	if before.IsZero() {
		return 0, fmt.Errorf("删除截止时间不能为空")
	}
	return s.repository.DeleteBefore(ctx, actor.TenantID, before)
}

func (s *Service) Export(ctx context.Context, actor auth.User, filter Filter) ([]Log, error) {
	filter.Level = strings.ToUpper(strings.TrimSpace(filter.Level))
	return s.repository.Export(ctx, actor.TenantID, filter)
}

func (s *Service) Stats(ctx context.Context, actor auth.User, start, end *time.Time) (Stats, error) {
	return s.repository.Stats(ctx, actor.TenantID, start, end)
}

func (s *Service) Recent(ctx context.Context, actor auth.User, limit int) ([]Log, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}
	return s.repository.Recent(ctx, actor.TenantID, limit)
}
