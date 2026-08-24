package repository

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/security"
)

// CreateTdengineConfigParams 描述 TDengine 配置落库参数。
type CreateTdengineConfigParams struct {
	ProjectID       string
	UserID          string
	Name            string
	Status          string
	Protocol        string
	Host            string
	Port            int
	Username        string
	DatabaseName    string
	Timezone        *string
	TLSSkipVerify   bool
	Options         map[string]any
	Secrets         map[string]string
	ClearSecretKeys []string
}

// ProtocolWave2Repository 负责 TDengine 配置的参数化 SQL。
type ProtocolWave2Repository struct {
	pool   *pgxpool.Pool
	cipher *security.ConnectionSecretCipher
}

// NewProtocolWave2Repository 创建第二波协议仓储。
func NewProtocolWave2Repository(pool *pgxpool.Pool, ciphers ...*security.ConnectionSecretCipher) *ProtocolWave2Repository {
	repository := &ProtocolWave2Repository{pool: pool}
	if len(ciphers) > 0 {
		repository.cipher = ciphers[0]
	}
	return repository
}

// CreateTdengineConfig 创建 TDengine 配置。
func (r *ProtocolWave2Repository) CreateTdengineConfig(ctx context.Context, params CreateTdengineConfigParams) (*ProtocolConnectionRecord, error) {
	optionsPayload, err := marshalWave2JSONObject(params.Options, true)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 TDengine 配置事务失败", err)
	}
	defer rollbackWave2TxQuietly(ctx, tx)

	record, err := r.createConnectionTx(ctx, tx, createWave2ConnectionTxParams{
		ProjectID: params.ProjectID,
		UserID:    params.UserID,
		Name:      params.Name,
		Type:      "tdengine",
		Status:    params.Status,
		Metadata: map[string]any{
			"protocol": params.Protocol, "host": params.Host, "port": params.Port,
			"username": params.Username, "databaseName": params.DatabaseName,
			"timezone": params.Timezone, "tlsSkipVerify": params.TLSSkipVerify,
			"options": cloneWave2Map(params.Options),
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO data_tdengine_configs (
			connection_id,
			protocol, host, port, username,
			database_name,
			timezone,
			tls_skip_verify,
			options
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)
	`, record.ID, params.Protocol, params.Host, params.Port, params.Username, params.DatabaseName, params.Timezone, params.TLSSkipVerify, optionsPayload)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 TDengine 配置失败", err)
	}
	if err := applyPlainConnectionSecretsTx(ctx, tx, r.cipher, record.ID, params.Secrets, params.ClearSecretKeys); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 TDengine 配置事务失败", err)
	}
	return record, nil
}

// UpdateTdengineConfig 原子更新主连接、结构化 TDengine 配置和密钥。
func (r *ProtocolWave2Repository) UpdateTdengineConfig(ctx context.Context, connectionID string, params CreateTdengineConfigParams) (*ProtocolConnectionRecord, error) {
	optionsPayload, err := marshalWave2JSONObject(params.Options, true)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 TDengine 更新事务失败", err)
	}
	defer rollbackWave2TxQuietly(ctx, tx)
	metadataPayload, _ := json.Marshal(map[string]any{"protocol": params.Protocol, "host": params.Host, "port": params.Port, "username": params.Username, "databaseName": params.DatabaseName, "timezone": params.Timezone, "tlsSkipVerify": params.TLSSkipVerify, "options": cloneWave2Map(params.Options)})
	record := ProtocolConnectionRecord{}
	err = tx.QueryRow(ctx, `UPDATE data_connections SET name=$3,status=$4,metadata=$5::jsonb,updated_by=$6,updated_at=now() WHERE project_id=$1 AND id=$2 AND type='tdengine' RETURNING id,project_id,name,type,status,created_at,updated_at`, params.ProjectID, connectionID, params.Name, params.Status, metadataPayload, params.UserID).Scan(&record.ID, &record.ProjectID, &record.Name, &record.Type, &record.Status, &record.CreatedAt, &record.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "TDengine 配置不存在")
	}
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新 TDengine 主连接失败", err)
	}
	_, err = tx.Exec(ctx, `UPDATE data_tdengine_configs SET protocol=$2,host=$3,port=$4,username=$5,database_name=$6,timezone=$7,tls_skip_verify=$8,options=$9::jsonb,updated_at=now() WHERE connection_id=$1`, connectionID, params.Protocol, params.Host, params.Port, params.Username, params.DatabaseName, params.Timezone, params.TLSSkipVerify, optionsPayload)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新 TDengine 配置失败", err)
	}
	if err := applyPlainConnectionSecretsTx(ctx, tx, r.cipher, connectionID, params.Secrets, params.ClearSecretKeys); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 TDengine 更新事务失败", err)
	}
	return &record, nil
}

type createWave2ConnectionTxParams struct {
	ProjectID string
	UserID    string
	Name      string
	Type      string
	Status    string
	Metadata  map[string]any
}

// createConnectionTx 统一写入 data_connections 主表。
func (r *ProtocolWave2Repository) createConnectionTx(ctx context.Context, tx pgx.Tx, params createWave2ConnectionTxParams) (*ProtocolConnectionRecord, error) {
	metadataPayload, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "序列化 wave2 连接元数据失败", err)
	}

	record := ProtocolConnectionRecord{}
	if err := lockProjectConnectionOrder(ctx, tx, params.ProjectID); err != nil {
		return nil, err
	}
	if err := normalizeProjectConnectionDisplayOrder(ctx, tx, params.ProjectID); err != nil {
		return nil, err
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO data_connections (
			project_id,
			name,
			type,
			category,
			status,
			metadata,
			display_order,
			created_by,
			updated_by
		)
		VALUES ($1, $2, $3, 'protocol', $4, $5::jsonb,
			COALESCE((SELECT MAX(display_order) + 1 FROM data_connections WHERE project_id = $1), 0),
			$6, $6)
		RETURNING id, project_id, name, type, status, created_at, updated_at
	`, params.ProjectID, params.Name, params.Type, params.Status, string(metadataPayload), params.UserID).Scan(
		&record.ID,
		&record.ProjectID,
		&record.Name,
		&record.Type,
		&record.Status,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, translateConnectionWriteError("写入 wave2 连接失败", err)
	}

	return &record, nil
}

func marshalWave2JSONObject(input map[string]any, emptyAsObject bool) (any, error) {
	if input == nil {
		if emptyAsObject {
			return "{}", nil
		}
		return nil, nil
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "wave2 JSON 配置格式无效", err)
	}
	return string(payload), nil
}

func cloneWave2Map(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func rollbackWave2TxQuietly(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}
