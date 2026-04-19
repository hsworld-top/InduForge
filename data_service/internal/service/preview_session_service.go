package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/cache"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

// PreviewSession 表示返回给 HTTP 层的 preview 会话对象。
type PreviewSession struct {
	ID           string         `json:"id"`
	ProjectID    string         `json:"projectId"`
	UserID       string         `json:"userId"`
	Status       string         `json:"status"`
	StartedAt    time.Time      `json:"startedAt"`
	LastActiveAt time.Time      `json:"lastActiveAt"`
	ExpiredAt    time.Time      `json:"expiredAt"`
	Meta         map[string]any `json:"meta"`
}

// CreatePreviewSessionInput 描述创建会话时允许外部传入的数据。
type CreatePreviewSessionInput struct {
	Meta map[string]any
}

// PreviewSessionService 负责 preview 会话的鉴权边界、状态流转与缓存协同。
type PreviewSessionService struct {
	repository *repository.PreviewSessionRepository
	redis      *cache.RedisClient
}

// NewPreviewSessionService 创建 preview 会话服务。
func NewPreviewSessionService(repo *repository.PreviewSessionRepository, redisClient *cache.RedisClient) *PreviewSessionService {
	return &PreviewSessionService{
		repository: repo,
		redis:      redisClient,
	}
}

// CreateSession 创建一个新的 active 会话，并写入 Redis 滑动过期 key。
func (s *PreviewSessionService) CreateSession(ctx context.Context, claims *auth.Claims, projectID string, input CreatePreviewSessionInput) (*PreviewSession, error) {
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}
	if claims == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成认证")
	}
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if err := validateUserID(claims.UserID); err != nil {
		return nil, err
	}
	if !claims.HasProjectAccess(projectID) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodePermissionProjectMismatch, http.StatusForbidden, "项目范围不足")
	}

	now := time.Now().UTC()
	expiredAt := now.Add(cache.SessionTTL)

	record, err := s.repository.Create(ctx, repository.CreatePreviewSessionParams{
		ProjectID:    projectID,
		UserID:       strings.TrimSpace(claims.UserID),
		Status:       "active",
		StartedAt:    now,
		LastActiveAt: now,
		ExpiredAt:    expiredAt,
		Meta:         cloneMap(input.Meta),
	})
	if err != nil {
		return nil, err
	}

	if err := s.redis.UpsertPreviewSession(ctx, record.ID, buildPreviewCachePayload(*record), cache.SessionTTL); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 preview 会话缓存失败", err)
	}

	session := toPreviewSession(*record)
	return &session, nil
}

// HeartbeatSession 对会话做一次续期，刷新数据库快照与 Redis TTL。
func (s *PreviewSessionService) HeartbeatSession(ctx context.Context, claims *auth.Claims, sessionID string) (*PreviewSession, error) {
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}
	if claims == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成认证")
	}
	if _, err := uuid.Parse(strings.TrimSpace(sessionID)); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "sessionId 格式无效", err)
	}

	current, err := s.repository.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := ensurePreviewSessionAccess(claims, *current); err != nil {
		return nil, err
	}
	if current.Status != "active" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前会话状态不允许续期")
	}

	now := time.Now().UTC()
	expiredAt := now.Add(cache.SessionTTL)

	updated, err := s.repository.Touch(ctx, repository.TouchPreviewSessionParams{
		ID:           current.ID,
		LastActiveAt: now,
		ExpiredAt:    expiredAt,
	})
	if err != nil {
		return nil, err
	}

	if err := s.redis.UpsertPreviewSession(ctx, updated.ID, buildPreviewCachePayload(*updated), cache.SessionTTL); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "刷新 preview 会话缓存失败", err)
	}

	session := toPreviewSession(*updated)
	return &session, nil
}

// CloseSession 关闭会话并清理 Redis 侧的临时状态。
func (s *PreviewSessionService) CloseSession(ctx context.Context, claims *auth.Claims, sessionID string) error {
	if err := s.validateDependencies(); err != nil {
		return err
	}
	if claims == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成认证")
	}
	if _, err := uuid.Parse(strings.TrimSpace(sessionID)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "sessionId 格式无效", err)
	}

	current, err := s.repository.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if err := ensurePreviewSessionAccess(claims, *current); err != nil {
		return err
	}

	if _, err := s.repository.Close(ctx, current.ID, time.Now().UTC()); err != nil {
		return err
	}
	if err := s.redis.DeletePreviewSession(ctx, current.ID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "清理 preview 会话缓存失败", err)
	}
	return nil
}

func (s *PreviewSessionService) validateDependencies() error {
	if s == nil || s.repository == nil || s.redis == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "preview 会话依赖未初始化")
	}
	return nil
}

func ensurePreviewSessionAccess(claims *auth.Claims, session repository.PreviewSessionRecord) error {
	if claims == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成认证")
	}
	if !claims.HasProjectAccess(session.ProjectID) {
		return apperrors.NewAppError(apperrors.ErrorCodePermissionProjectMismatch, http.StatusForbidden, "项目范围不足")
	}
	if strings.TrimSpace(claims.UserID) != strings.TrimSpace(session.UserID) {
		return apperrors.NewAppError(apperrors.ErrorCodePermissionInsufficient, http.StatusForbidden, "仅支持操作自己创建的预览会话")
	}
	return nil
}

func toPreviewSession(record repository.PreviewSessionRecord) PreviewSession {
	return PreviewSession{
		ID:           record.ID,
		ProjectID:    record.ProjectID,
		UserID:       record.UserID,
		Status:       record.Status,
		StartedAt:    record.StartedAt,
		LastActiveAt: record.LastActiveAt,
		ExpiredAt:    record.ExpiredAt,
		Meta:         cloneMap(record.Meta),
	}
}

func buildPreviewCachePayload(record repository.PreviewSessionRecord) map[string]any {
	return map[string]any{
		"id":           record.ID,
		"projectId":    record.ProjectID,
		"userId":       record.UserID,
		"status":       record.Status,
		"startedAt":    record.StartedAt.UTC().Format(time.RFC3339Nano),
		"lastActiveAt": record.LastActiveAt.UTC().Format(time.RFC3339Nano),
		"expiredAt":    record.ExpiredAt.UTC().Format(time.RFC3339Nano),
		"meta":         cloneMap(record.Meta),
	}
}
