package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// PreviewSessionRecord 表示 data_preview_sessions 在仓储层的投影。
type PreviewSessionRecord struct {
	ID           string
	ProjectID    string
	UserID       string
	Status       string
	StartedAt    time.Time
	LastActiveAt time.Time
	ExpiredAt    time.Time
	Meta         map[string]any
}

// CreatePreviewSessionParams 描述创建 preview 会话时的落库参数。
type CreatePreviewSessionParams struct {
	ProjectID    string
	UserID       string
	Status       string
	StartedAt    time.Time
	LastActiveAt time.Time
	ExpiredAt    time.Time
	Meta         map[string]any
}

// TouchPreviewSessionParams 描述续期 preview 会话时的更新参数。
type TouchPreviewSessionParams struct {
	ID           string
	LastActiveAt time.Time
	ExpiredAt    time.Time
}

// PreviewSessionRepository 封装 preview 会话的参数化 SQL 访问。
type PreviewSessionRepository struct {
	pool *pgxpool.Pool
}

// NewPreviewSessionRepository 创建 preview 会话仓储。
func NewPreviewSessionRepository(pool *pgxpool.Pool) *PreviewSessionRepository {
	return &PreviewSessionRepository{pool: pool}
}

// Create 新建 preview 会话快照记录。
func (r *PreviewSessionRepository) Create(ctx context.Context, params CreatePreviewSessionParams) (*PreviewSessionRecord, error) {
	metaPayload, err := marshalPreviewMeta(params.Meta)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, `
		INSERT INTO data_preview_sessions (
			project_id,
			user_id,
			status,
			started_at,
			last_active_at,
			expired_at,
			meta
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
		RETURNING id, project_id, user_id, status, started_at, last_active_at, expired_at, meta
	`, params.ProjectID, params.UserID, params.Status, params.StartedAt, params.LastActiveAt, params.ExpiredAt, metaPayload)

	record, scanErr := scanPreviewSession(row)
	if scanErr != nil {
		return nil, scanErr
	}
	return &record, nil
}

// GetByID 按会话 ID 读取快照记录。
func (r *PreviewSessionRepository) GetByID(ctx context.Context, sessionID string) (*PreviewSessionRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, user_id, status, started_at, last_active_at, expired_at, meta
		FROM data_preview_sessions
		WHERE id = $1
	`, sessionID)

	record, err := scanPreviewSession(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// Touch 将会话状态保持在 active，并刷新活动时间与过期时间。
func (r *PreviewSessionRepository) Touch(ctx context.Context, params TouchPreviewSessionParams) (*PreviewSessionRecord, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE data_preview_sessions
		SET status = 'active',
			last_active_at = $2,
			expired_at = $3,
			updated_at = now()
		WHERE id = $1
		RETURNING id, project_id, user_id, status, started_at, last_active_at, expired_at, meta
	`, params.ID, params.LastActiveAt, params.ExpiredAt)

	record, err := scanPreviewSession(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// Close 将会话状态设置为 closed，并更新最近活动时间为关闭时刻。
func (r *PreviewSessionRepository) Close(ctx context.Context, sessionID string, closedAt time.Time) (*PreviewSessionRecord, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE data_preview_sessions
		SET status = 'closed',
			last_active_at = $2,
			updated_at = now()
		WHERE id = $1
		RETURNING id, project_id, user_id, status, started_at, last_active_at, expired_at, meta
	`, sessionID, closedAt)

	record, err := scanPreviewSession(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

type previewSessionScannable interface {
	Scan(dest ...any) error
}

func scanPreviewSession(row previewSessionScannable) (PreviewSessionRecord, error) {
	var (
		record    PreviewSessionRecord
		metaBytes []byte
	)

	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.UserID,
		&record.Status,
		&record.StartedAt,
		&record.LastActiveAt,
		&record.ExpiredAt,
		&metaBytes,
	); err != nil {
		if err == pgx.ErrNoRows {
			return PreviewSessionRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "预览会话不存在")
		}
		return PreviewSessionRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取预览会话失败", err)
	}

	meta, err := unmarshalPreviewMeta(metaBytes)
	if err != nil {
		return PreviewSessionRecord{}, err
	}
	record.Meta = meta
	return record, nil
}

func marshalPreviewMeta(meta map[string]any) (string, error) {
	if meta == nil {
		meta = map[string]any{}
	}
	payload, err := json.Marshal(meta)
	if err != nil {
		return "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "preview meta 格式无效", err)
	}
	return string(payload), nil
}

func unmarshalPreviewMeta(metaBytes []byte) (map[string]any, error) {
	if len(metaBytes) == 0 {
		return map[string]any{}, nil
	}

	var meta map[string]any
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析 preview meta 失败", err)
	}
	if meta == nil {
		meta = map[string]any{}
	}
	return meta, nil
}
