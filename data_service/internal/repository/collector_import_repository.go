package repository

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/jackc/pgx/v5"
)

type CollectorImportError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
type CollectorImportCandidate struct {
	Point     CreateCollectorPointParams `json:"point"`
	GroupPath string                     `json:"groupPath"`
}
type CreateCollectorImportSessionParams struct {
	ID, ProjectID, ConnectionID, DriverID, CreatedBy string
	Candidates                                       []CollectorImportCandidate
	Errors                                           []CollectorImportError
	TotalRows                                        int
	ExpiresAt                                        time.Time
}
type CollectorImportSessionRecord struct {
	ID, ProjectID, ConnectionID, DriverID, Status string
	Candidates                                    []CollectorImportCandidate
	Errors                                        []CollectorImportError
	TotalRows, ValidRows                          int
	ExpiresAt                                     time.Time
}

func (r *CollectorRepository) CreateImportSession(ctx context.Context, params CreateCollectorImportSessionParams) (*CollectorImportSessionRecord, error) {
	candidates, err := json.Marshal(params.Candidates)
	if err != nil {
		return nil, badCollectorPayload("序列化导入候选点位失败", err)
	}
	issues, err := json.Marshal(params.Errors)
	if err != nil {
		return nil, badCollectorPayload("序列化导入错误失败", err)
	}
	record, err := scanCollectorImportSession(r.pool.QueryRow(ctx, `INSERT INTO data_collector_import_sessions (id,project_id,connection_id,driver_id,candidates,errors,total_rows,valid_rows,expires_at,created_by) VALUES ($1,$2,$3,$4,$5::jsonb,$6::jsonb,$7,$8,$9,$10) RETURNING id,project_id,connection_id,driver_id,status,candidates,errors,total_rows,valid_rows,expires_at`, params.ID, params.ProjectID, params.ConnectionID, params.DriverID, string(candidates), string(issues), params.TotalRows, len(params.Candidates), params.ExpiresAt, params.CreatedBy))
	if err != nil {
		return nil, translateCollectorWriteError("创建点位导入会话失败", err)
	}
	return &record, nil
}
func (r *CollectorRepository) GetImportSession(ctx context.Context, projectID, connectionID, importID string) (*CollectorImportSessionRecord, error) {
	record, err := scanCollectorImportSession(r.pool.QueryRow(ctx, `SELECT id,project_id,connection_id,driver_id,status,candidates,errors,total_rows,valid_rows,expires_at FROM data_collector_import_sessions WHERE id=$1 AND project_id=$2 AND connection_id=$3`, importID, projectID, connectionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "点位导入会话不存在")
	}
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("查询点位导入会话失败", err)
	}
	return &record, nil
}
func (r *CollectorRepository) CommitImportSession(ctx context.Context, projectID, connectionID, importID string) ([]CollectorPointRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("开启点位导入提交事务失败", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var status string
	var expiresAt time.Time
	var payload []byte
	err = tx.QueryRow(ctx, `SELECT status,expires_at,candidates FROM data_collector_import_sessions WHERE id=$1 AND project_id=$2 AND connection_id=$3 FOR UPDATE`, importID, projectID, connectionID).Scan(&status, &expiresAt, &payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "点位导入会话不存在")
	}
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("锁定点位导入会话失败", err)
	}
	if status != "preview" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "点位导入会话已提交或失效")
	}
	if !expiresAt.After(time.Now()) {
		_, _ = tx.Exec(ctx, `UPDATE data_collector_import_sessions SET status='expired',updated_at=now() WHERE id=$1`, importID)
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusGone, "点位导入会话已过期")
	}
	var candidates []CollectorImportCandidate
	if err := json.Unmarshal(payload, &candidates); err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("解析点位导入候选记录失败", err)
	}
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		point := candidate.Point
		if strings.TrimSpace(candidate.GroupPath) != "" {
			groupID, err := ensureCollectorImportGroup(ctx, tx, projectID, connectionID, candidate.GroupPath)
			if err != nil {
				return nil, err
			}
			point.GroupID = &groupID
		}
		if err := insertCollectorPointAndDataPoint(ctx, tx, point); err != nil {
			return nil, err
		}
		ids = append(ids, point.ID)
	}
	tag, err := tx.Exec(ctx, `UPDATE data_collector_import_sessions SET status='committed',committed_at=now(),updated_at=now() WHERE id=$1 AND status='preview'`, importID)
	if err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("完成点位导入会话失败", err)
	}
	if tag.RowsAffected() != 1 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "点位导入会话状态已变化")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, wrapUnifiedCollectorRepositoryError("提交点位导入事务失败", err)
	}
	return r.listPointsByIDs(ctx, projectID, connectionID, ids)
}

type collectorImportRow interface{ Scan(...any) error }

func scanCollectorImportSession(row collectorImportRow) (CollectorImportSessionRecord, error) {
	var record CollectorImportSessionRecord
	var candidates, issues []byte
	err := row.Scan(&record.ID, &record.ProjectID, &record.ConnectionID, &record.DriverID, &record.Status, &candidates, &issues, &record.TotalRows, &record.ValidRows, &record.ExpiresAt)
	if err != nil {
		return record, err
	}
	if err := json.Unmarshal(candidates, &record.Candidates); err != nil {
		return record, wrapUnifiedCollectorRepositoryError("解析导入候选记录失败", err)
	}
	if err := json.Unmarshal(issues, &record.Errors); err != nil {
		return record, wrapUnifiedCollectorRepositoryError("解析导入错误失败", err)
	}
	return record, nil
}
func ensureCollectorImportGroup(ctx context.Context, tx pgx.Tx, projectID, connectionID, path string) (string, error) {
	var parentID *string
	segments := strings.Split(strings.Trim(path, " /"), "/")
	for index, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		var id string
		err := tx.QueryRow(ctx, `SELECT id FROM data_collector_point_groups WHERE project_id=$1 AND connection_id=$2 AND name=$3 AND (($4::uuid IS NULL AND parent_id IS NULL) OR parent_id=$4::uuid)`, projectID, connectionID, segment, parentID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			id = uuid.NewString()
			_, err = tx.Exec(ctx, `INSERT INTO data_collector_point_groups (id,project_id,connection_id,parent_id,name,sort_order) VALUES ($1,$2,$3,$4,$5,$6)`, id, projectID, connectionID, parentID, segment, index)
		}
		if err != nil {
			return "", translateCollectorWriteError("创建导入点位分组失败", err)
		}
		parentID = &id
	}
	if parentID == nil {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "导入分组路径无效")
	}
	return *parentID, nil
}
