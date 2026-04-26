package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/cache"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
)

// PreviewSession 表示返回给 HTTP 层和 preview socket 的会话快照。
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

// PreviewDiagnostic 表示开发态预览联调使用的诊断结果。
type PreviewDiagnostic struct {
	ProjectID        string          `json:"projectId"`
	UserID           string          `json:"userId"`
	TenantID         string          `json:"tenantId"`
	AuthOK           bool            `json:"authOk"`
	ProjectAccessOK  bool            `json:"projectAccessOk"`
	RedisReady       bool            `json:"redisReady"`
	SocketEnabled    bool            `json:"socketEnabled"`
	SessionID        string          `json:"sessionId,omitempty"`
	Session          *PreviewSession `json:"session,omitempty"`
	SessionActiveOK  bool            `json:"sessionActiveOk"`
	FailureReason    string          `json:"failureReason,omitempty"`
	RecommendedEvent string          `json:"recommendedEvent,omitempty"`
}

// CreatePreviewSessionInput 描述创建 preview 会话时允许外部传入的附加数据。
type CreatePreviewSessionInput struct {
	Meta map[string]any
}

// PreviewSessionService 负责 preview 会话的权限边界、状态流转与 Redis 协同。
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

// HeartbeatSession 对会话执行一次续期，刷新数据库快照与 Redis TTL。
func (s *PreviewSessionService) HeartbeatSession(ctx context.Context, claims *auth.Claims, sessionID string) (*PreviewSession, error) {
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}
	if claims == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成认证")
	}

	current, err := s.loadPreviewSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := ensurePreviewSessionAccess(claims, *current); err != nil {
		return nil, err
	}
	if err := ensurePreviewSessionActive(*current); err != nil {
		return nil, err
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

	current, err := s.loadPreviewSession(ctx, sessionID)
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

// AuthorizeSession 为 preview socket 等短时预览入口校验会话是否仍然有效。
func (s *PreviewSessionService) AuthorizeSession(ctx context.Context, claims *auth.Claims, projectID, sessionID string) (*PreviewSession, error) {
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}
	if claims == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenRequired, http.StatusUnauthorized, "请先完成认证")
	}
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}

	current, err := s.loadPreviewSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(current.ProjectID) != strings.TrimSpace(projectID) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodePermissionProjectMismatch, http.StatusForbidden, "项目范围不足")
	}
	if err := ensurePreviewSessionAccess(claims, *current); err != nil {
		return nil, err
	}
	if err := ensurePreviewSessionActive(*current); err != nil {
		return nil, err
	}

	session := toPreviewSession(*current)
	return &session, nil
}

// DiagnosePreview 对开发态预览链路做只读诊断，方便 debug 页面定位 403/会话过期等问题。
func (s *PreviewSessionService) DiagnosePreview(ctx context.Context, claims *auth.Claims, projectID, sessionID string) (*PreviewDiagnostic, error) {
	result := &PreviewDiagnostic{
		ProjectID:     strings.TrimSpace(projectID),
		RedisReady:    s != nil && s.redis != nil,
		SocketEnabled: s != nil && s.redis != nil && s.repository != nil,
	}
	if claims != nil {
		result.UserID = strings.TrimSpace(claims.UserID)
		result.TenantID = strings.TrimSpace(claims.TenantID)
		result.AuthOK = true
		result.ProjectAccessOK = claims.HasProjectAccess(projectID)
	}

	if err := s.validateDependencies(); err != nil {
		result.FailureReason = "preview 会话依赖未就绪"
		result.RecommendedEvent = "检查 data_service Redis 配置与启动日志"
		return result, nil
	}
	if claims == nil {
		result.FailureReason = "缺少认证令牌"
		result.RecommendedEvent = "传入 Authorization Bearer token"
		return result, nil
	}
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if !claims.HasProjectAccess(projectID) {
		result.FailureReason = "当前令牌没有项目访问权限"
		result.RecommendedEvent = "确认 token 内的项目范围与 projectId 一致"
		return result, nil
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		result.SessionActiveOK = false
		result.RecommendedEvent = "先创建 preview session，再建立 Socket.IO 连接"
		return result, nil
	}

	result.SessionID = sessionID
	session, err := s.AuthorizeSession(ctx, claims, projectID, sessionID)
	if err != nil {
		result.FailureReason = socketFriendlyPreviewFailure(err)
		result.RecommendedEvent = "重新创建 preview session 或检查 Socket.IO 握手参数"
		return result, nil
	}
	result.Session = session
	result.SessionActiveOK = true
	return result, nil
}

// GetSession 返回当前 preview 会话快照，供 socket 清理协程探测状态。
func (s *PreviewSessionService) GetSession(ctx context.Context, sessionID string) (*PreviewSession, error) {
	if err := s.validateDependencies(); err != nil {
		return nil, err
	}

	current, err := s.loadPreviewSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	session := toPreviewSession(*current)
	return &session, nil
}

func socketFriendlyPreviewFailure(err error) string {
	if err == nil {
		return ""
	}
	var appErr *apperrors.AppError
	if ok := errors.As(err, &appErr); ok && appErr.Message != "" {
		return appErr.Message
	}
	return err.Error()
}

func (s *PreviewSessionService) validateDependencies() error {
	if s == nil || s.repository == nil || s.redis == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "preview 会话依赖未初始化")
	}
	return nil
}

func (s *PreviewSessionService) loadPreviewSession(ctx context.Context, sessionID string) (*repository.PreviewSessionRecord, error) {
	if _, err := uuid.Parse(strings.TrimSpace(sessionID)); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "sessionId 格式无效", err)
	}
	return s.repository.GetByID(ctx, sessionID)
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

func ensurePreviewSessionActive(session repository.PreviewSessionRecord) error {
	if session.Status != "active" {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "当前会话状态不允许预览")
	}
	if !session.ExpiredAt.IsZero() && time.Now().UTC().After(session.ExpiredAt.UTC()) {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "preview 会话已过期")
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
