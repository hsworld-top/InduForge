package repository

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CollectorDevAgentRecord struct {
	ID             string
	TenantID       string
	Name           string
	OS             string
	Arch           string
	Version        string
	LastIP         string
	CredentialHash string
	Capabilities   json.RawMessage
	LastSeenAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CollectorDevRegistrationCodeRecord struct {
	ID        string
	TenantID  string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type CollectorDevTaskRecord struct {
	ID             string
	TenantID       string
	ProjectID      string
	ConnectionID   string
	AgentID        string
	Operation      string
	Status         string
	RequestPayload json.RawMessage
	ResultPayload  json.RawMessage
	ErrorCode      *string
	ErrorMessage   *string
	DeadlineAt     time.Time
	ClaimedAt      *time.Time
	FinishedAt     *time.Time
	CreatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateCollectorAgentParams struct {
	ID             string
	CodeHash       string
	CredentialHash string
	Name           string
	OS             string
	Arch           string
	Version        string
	LastIP         string
	Capabilities   any
}

type CreateCollectorTaskParams struct {
	ID             string
	TenantID       string
	ProjectID      string
	ConnectionID   string
	AgentID        string
	Operation      string
	RequestPayload any
	DeadlineAt     time.Time
	CreatedBy      string
}

type CompleteCollectorTaskParams struct {
	AgentID      string
	TaskID       string
	Status       string
	Result       any
	ErrorCode    string
	ErrorMessage string
}

type CollectorDevRepository struct {
	pool *pgxpool.Pool
}

func NewCollectorDevRepository(pool *pgxpool.Pool) *CollectorDevRepository {
	return &CollectorDevRepository{pool: pool}
}

func (r *CollectorDevRepository) CreateRegistrationCode(ctx context.Context, id, tenantID, codeHash, createdBy string, expiresAt time.Time) (*CollectorDevRegistrationCodeRecord, error) {
	var record CollectorDevRegistrationCodeRecord
	err := r.pool.QueryRow(ctx, `
		INSERT INTO collector_dev_registration_codes (id, tenant_id, code_hash, expires_at, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, tenant_id, expires_at, created_at
	`, id, tenantID, codeHash, expiresAt, createdBy).Scan(&record.ID, &record.TenantID, &record.ExpiresAt, &record.CreatedAt)
	if err != nil {
		return nil, wrapCollectorRepositoryError("创建采集调试代理注册码失败", err)
	}
	return &record, nil
}

// RegisterAgent 在同一事务中锁定并消费一次性注册码，避免重复注册。
func (r *CollectorDevRepository) RegisterAgent(ctx context.Context, params CreateCollectorAgentParams) (*CollectorDevAgentRecord, error) {
	capabilities, err := json.Marshal(params.Capabilities)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "序列化 Agent 能力失败", err)
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, wrapCollectorRepositoryError("开启 Agent 注册事务失败", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var tenantID string
	err = tx.QueryRow(ctx, `
		SELECT tenant_id
		FROM collector_dev_registration_codes
		WHERE code_hash = $1 AND used_at IS NULL AND expires_at > now()
		FOR UPDATE
	`, params.CodeHash).Scan(&tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "注册码无效、已使用或已过期")
	}
	if err != nil {
		return nil, wrapCollectorRepositoryError("校验 Agent 注册码失败", err)
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO collector_dev_agents (id, tenant_id, name, os, arch, version, last_ip, credential_hash, capabilities, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, now())
		RETURNING id, tenant_id, name, os, arch, version, last_ip, credential_hash, capabilities, last_seen_at, created_at, updated_at
	`, params.ID, tenantID, params.Name, params.OS, params.Arch, params.Version, params.LastIP, params.CredentialHash, string(capabilities))
	record, err := scanCollectorAgent(row)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `
		UPDATE collector_dev_registration_codes
		SET used_at = now(), used_by_agent_id = $2
		WHERE code_hash = $1 AND used_at IS NULL
	`, params.CodeHash, params.ID); err != nil {
		return nil, wrapCollectorRepositoryError("消费 Agent 注册码失败", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, wrapCollectorRepositoryError("提交 Agent 注册事务失败", err)
	}
	return &record, nil
}

func (r *CollectorDevRepository) AuthenticateAgent(ctx context.Context, credentialHash string) (*CollectorDevAgentRecord, error) {
	record, err := scanCollectorAgent(r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, os, arch, version, last_ip, credential_hash, capabilities, last_seen_at, created_at, updated_at
		FROM collector_dev_agents WHERE credential_hash = $1 AND revoked_at IS NULL
	`, credentialHash))
	if errors.Is(err, errCollectorAgentNotFound) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "Agent Token 无效")
	}
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *CollectorDevRepository) GetAgent(ctx context.Context, tenantID, agentID string) (*CollectorDevAgentRecord, error) {
	record, err := scanCollectorAgent(r.pool.QueryRow(ctx, `SELECT id, tenant_id, name, os, arch, version, last_ip, credential_hash, capabilities, last_seen_at, created_at, updated_at FROM collector_dev_agents WHERE id=$1 AND tenant_id=$2 AND revoked_at IS NULL`, agentID, tenantID))
	if errors.Is(err, errCollectorAgentNotFound) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "采集调试代理不存在")
	}
	return &record, err
}

// ListAgentsPage 按租户分页读取未撤销的采集调试代理。
func (r *CollectorDevRepository) ListAgentsPage(ctx context.Context, tenantID string, page, pageSize int) ([]CollectorDevAgentRecord, int, error) {
	page, pageSize = normalizePageAndSize(page, pageSize, 10, 100)
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM collector_dev_agents WHERE tenant_id = $1 AND revoked_at IS NULL`, tenantID).Scan(&total); err != nil {
		return nil, 0, wrapCollectorRepositoryError("统计采集调试代理失败", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, name, os, arch, version, last_ip, credential_hash, capabilities, last_seen_at, created_at, updated_at
		FROM collector_dev_agents
		WHERE tenant_id = $1 AND revoked_at IS NULL
		ORDER BY updated_at DESC, created_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, wrapCollectorRepositoryError("分页查询采集调试代理失败", err)
	}
	defer rows.Close()
	records := make([]CollectorDevAgentRecord, 0, pageSize)
	for rows.Next() {
		record, scanErr := scanCollectorAgent(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, wrapCollectorRepositoryError("遍历采集调试代理分页失败", err)
	}
	return records, total, nil
}

func (r *CollectorDevRepository) DeleteAgent(ctx context.Context, tenantID, agentID string) error {
	result, err := r.pool.Exec(ctx, `UPDATE collector_dev_agents SET revoked_at = now(), updated_at = now() WHERE id = $1 AND tenant_id = $2 AND revoked_at IS NULL`, agentID, tenantID)
	if err != nil {
		return wrapCollectorRepositoryError("删除采集调试代理失败", err)
	}
	if result.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "采集调试代理不存在")
	}
	return nil
}

func (r *CollectorDevRepository) Heartbeat(ctx context.Context, agentID, lastIP string, capabilities any) (*CollectorDevAgentRecord, error) {
	payload, err := json.Marshal(capabilities)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "序列化 Agent 能力失败", err)
	}
	record, err := scanCollectorAgent(r.pool.QueryRow(ctx, `
		UPDATE collector_dev_agents SET capabilities = $2::jsonb, last_ip = $3, last_seen_at = now(), updated_at = now()
		WHERE id = $1 AND revoked_at IS NULL
		RETURNING id, tenant_id, name, os, arch, version, last_ip, credential_hash, capabilities, last_seen_at, created_at, updated_at
	`, agentID, string(payload), lastIP))
	if errors.Is(err, errCollectorAgentNotFound) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeAuthTokenInvalid, http.StatusUnauthorized, "Agent 已被移除")
	}
	return &record, err
}

func (r *CollectorDevRepository) CreateTask(ctx context.Context, params CreateCollectorTaskParams) (*CollectorDevTaskRecord, error) {
	payload, err := json.Marshal(params.RequestPayload)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "序列化调试任务失败", err)
	}
	var agentExists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM collector_dev_agents WHERE id = $1 AND tenant_id = $2 AND revoked_at IS NULL)`, params.AgentID, params.TenantID).Scan(&agentExists); err != nil {
		return nil, wrapCollectorRepositoryError("校验采集调试代理失败", err)
	}
	if !agentExists {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "采集调试代理不存在")
	}
	record, err := scanCollectorTask(r.pool.QueryRow(ctx, `
		INSERT INTO collector_dev_tasks (id, tenant_id, project_id, connection_id, agent_id, operation, status, request_payload, deadline_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, 'queued', $7::jsonb, $8, $9)
		RETURNING id, tenant_id, project_id, connection_id, agent_id, operation, status, request_payload, result_payload, error_code, error_message, deadline_at, claimed_at, finished_at, created_by, created_at, updated_at
	`, params.ID, params.TenantID, params.ProjectID, params.ConnectionID, params.AgentID, params.Operation, string(payload), params.DeadlineAt, params.CreatedBy))
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *CollectorDevRepository) GetTask(ctx context.Context, tenantID, projectID, taskID string) (*CollectorDevTaskRecord, error) {
	if _, err := r.pool.Exec(ctx, `UPDATE collector_dev_tasks SET status = 'expired', finished_at = now(), updated_at = now() WHERE id = $1 AND tenant_id = $2 AND project_id = $3 AND status IN ('queued', 'running') AND deadline_at <= now()`, taskID, tenantID, projectID); err != nil {
		return nil, wrapCollectorRepositoryError("更新调试任务过期状态失败", err)
	}
	record, err := scanCollectorTask(r.pool.QueryRow(ctx, `SELECT id, tenant_id, project_id, connection_id, agent_id, operation, status, request_payload, result_payload, error_code, error_message, deadline_at, claimed_at, finished_at, created_by, created_at, updated_at FROM collector_dev_tasks WHERE id = $1 AND tenant_id = $2 AND project_id = $3`, taskID, tenantID, projectID))
	if errors.Is(err, errCollectorTaskNotFound) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "调试任务不存在")
	}
	return &record, err
}

func (r *CollectorDevRepository) ClaimTask(ctx context.Context, agentID string) (*CollectorDevTaskRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, wrapCollectorRepositoryError("开启任务领取事务失败", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `UPDATE collector_dev_tasks SET status = 'expired', finished_at = now(), updated_at = now() WHERE agent_id = $1 AND status IN ('queued', 'running') AND deadline_at <= now()`, agentID); err != nil {
		return nil, wrapCollectorRepositoryError("更新调试任务过期状态失败", err)
	}
	var running bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM collector_dev_tasks WHERE agent_id = $1 AND status = 'running')`, agentID).Scan(&running); err != nil {
		return nil, wrapCollectorRepositoryError("检查 Agent 运行任务失败", err)
	}
	if running {
		if err = tx.Commit(ctx); err != nil {
			return nil, wrapCollectorRepositoryError("提交空任务领取事务失败", err)
		}
		return nil, nil
	}
	record, err := scanCollectorTask(tx.QueryRow(ctx, `
		WITH candidate AS (SELECT id FROM collector_dev_tasks WHERE agent_id = $1 AND status = 'queued' AND deadline_at > now() ORDER BY created_at, id FOR UPDATE SKIP LOCKED LIMIT 1)
		UPDATE collector_dev_tasks task SET status = 'running', claimed_at = now(), updated_at = now() FROM candidate WHERE task.id = candidate.id
		RETURNING task.id, task.tenant_id, task.project_id, task.connection_id, task.agent_id, task.operation, task.status, task.request_payload, task.result_payload, task.error_code, task.error_message, task.deadline_at, task.claimed_at, task.finished_at, task.created_by, task.created_at, task.updated_at
	`, agentID))
	if errors.Is(err, errCollectorTaskNotFound) {
		if err = tx.Commit(ctx); err != nil {
			return nil, wrapCollectorRepositoryError("提交空任务领取事务失败", err)
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, wrapCollectorRepositoryError("提交任务领取事务失败", err)
	}
	return &record, nil
}

func (r *CollectorDevRepository) CompleteTask(ctx context.Context, params CompleteCollectorTaskParams) (*CollectorDevTaskRecord, error) {
	var resultPayload any
	if params.Result != nil {
		payload, err := json.Marshal(params.Result)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "序列化调试任务结果失败", err)
		}
		if len(payload) > 2*1024*1024 {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "调试任务结果不能超过 2 MB")
		}
		resultPayload = string(payload)
	}
	record, err := scanCollectorTask(r.pool.QueryRow(ctx, `
		UPDATE collector_dev_tasks SET status = $3, result_payload = CASE WHEN $4::text IS NULL THEN NULL ELSE $4::jsonb END, error_code = NULLIF($5, ''), error_message = NULLIF($6, ''), finished_at = now(), updated_at = now()
		WHERE id = $1 AND agent_id = $2 AND status = 'running' AND deadline_at > now()
		RETURNING id, tenant_id, project_id, connection_id, agent_id, operation, status, request_payload, result_payload, error_code, error_message, deadline_at, claimed_at, finished_at, created_by, created_at, updated_at
	`, params.TaskID, params.AgentID, params.Status, resultPayload, params.ErrorCode, params.ErrorMessage))
	if errors.Is(err, errCollectorTaskNotFound) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "调试任务不存在、已结束或已过期")
	}
	return &record, err
}

func (r *CollectorDevRepository) CancelTask(ctx context.Context, tenantID, projectID, taskID string) (*CollectorDevTaskRecord, error) {
	record, err := scanCollectorTask(r.pool.QueryRow(ctx, `UPDATE collector_dev_tasks SET status = 'cancelled', finished_at = now(), updated_at = now() WHERE id = $1 AND tenant_id = $2 AND project_id = $3 AND status IN ('queued', 'running') RETURNING id, tenant_id, project_id, connection_id, agent_id, operation, status, request_payload, result_payload, error_code, error_message, deadline_at, claimed_at, finished_at, created_by, created_at, updated_at`, taskID, tenantID, projectID))
	if errors.Is(err, errCollectorTaskNotFound) {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "调试任务不存在或已结束")
	}
	return &record, err
}

var (
	errCollectorAgentNotFound = errors.New("collector agent not found")
	errCollectorTaskNotFound  = errors.New("collector task not found")
)

type collectorRow interface{ Scan(dest ...any) error }

func scanCollectorAgent(row collectorRow) (CollectorDevAgentRecord, error) {
	var record CollectorDevAgentRecord
	err := row.Scan(&record.ID, &record.TenantID, &record.Name, &record.OS, &record.Arch, &record.Version, &record.LastIP, &record.CredentialHash, &record.Capabilities, &record.LastSeenAt, &record.CreatedAt, &record.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return record, errCollectorAgentNotFound
	}
	if err != nil {
		return record, wrapCollectorRepositoryError("读取采集调试代理失败", err)
	}
	return record, nil
}

func scanCollectorTask(row collectorRow) (CollectorDevTaskRecord, error) {
	var record CollectorDevTaskRecord
	err := row.Scan(&record.ID, &record.TenantID, &record.ProjectID, &record.ConnectionID, &record.AgentID, &record.Operation, &record.Status, &record.RequestPayload, &record.ResultPayload, &record.ErrorCode, &record.ErrorMessage, &record.DeadlineAt, &record.ClaimedAt, &record.FinishedAt, &record.CreatedBy, &record.CreatedAt, &record.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return record, errCollectorTaskNotFound
	}
	if err != nil {
		return record, wrapCollectorRepositoryError("读取调试任务失败", err)
	}
	return record, nil
}

func wrapCollectorRepositoryError(message string, err error) error {
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}
