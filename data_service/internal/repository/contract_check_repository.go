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

type ContractCheckRunRecord struct {
	ID         int64
	ProjectID  string
	Scope      string
	ObjectType *string
	ObjectID   *string
	Status     string
	Summary    json.RawMessage
	Result     json.RawMessage
	CreatedBy  *string
	CreatedAt  time.Time
}

type SaveContractCheckRunParams struct {
	ProjectID  string
	Scope      string
	ObjectType *string
	ObjectID   *string
	Status     string
	Summary    any
	Result     any
	CreatedBy  *string
}

type ContractCheckRepository struct {
	pool *pgxpool.Pool
}

func NewContractCheckRepository(pool *pgxpool.Pool) *ContractCheckRepository {
	return &ContractCheckRepository{pool: pool}
}

func (r *ContractCheckRepository) SaveRun(ctx context.Context, params SaveContractCheckRunParams) (*ContractCheckRunRecord, error) {
	summaryPayload, err := json.Marshal(params.Summary)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "序列化契约检查摘要失败", err)
	}
	resultPayload, err := json.Marshal(params.Result)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "序列化契约检查结果失败", err)
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO data_contract_check_runs (
			project_id,
			scope,
			object_type,
			object_id,
			status,
			summary,
			result,
			created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, $8)
		RETURNING id, project_id, scope, object_type, object_id, status, summary, result, created_by, created_at
	`, params.ProjectID, params.Scope, params.ObjectType, params.ObjectID, params.Status, string(summaryPayload), string(resultPayload), params.CreatedBy)
	record, scanErr := scanContractCheckRun(row)
	if scanErr != nil {
		return nil, scanErr
	}
	return &record, nil
}

func (r *ContractCheckRepository) GetLatestRun(ctx context.Context, projectID string) (*ContractCheckRunRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, scope, object_type, object_id, status, summary, result, created_by, created_at
		FROM data_contract_check_runs
		WHERE project_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, projectID)
	record, err := scanContractCheckRun(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *ContractCheckRepository) ListRuns(ctx context.Context, projectID string, page, pageSize int) ([]ContractCheckRunRecord, int, error) {
	page, pageSize = normalizePageAndSize(page, pageSize, 20, 100)
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM data_contract_check_runs
		WHERE project_id = $1
	`, projectID).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计契约检查记录失败", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, scope, object_type, object_id, status, summary, result, created_by, created_at
		FROM data_contract_check_runs
		WHERE project_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, projectID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询契约检查记录失败", err)
	}
	defer rows.Close()
	records := make([]ContractCheckRunRecord, 0)
	for rows.Next() {
		record, scanErr := scanContractCheckRun(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历契约检查记录失败", err)
	}
	return records, total, nil
}

func scanContractCheckRun(row pgx.Row) (ContractCheckRunRecord, error) {
	var record ContractCheckRunRecord
	err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.Scope,
		&record.ObjectType,
		&record.ObjectID,
		&record.Status,
		&record.Summary,
		&record.Result,
		&record.CreatedBy,
		&record.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return record, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "契约检查记录不存在")
		}
		return record, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取契约检查记录失败", err)
	}
	return record, nil
}
