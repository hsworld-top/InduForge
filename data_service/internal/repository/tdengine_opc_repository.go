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
	IsEnabled       *bool
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

// TDengineOPCRepository 负责 TDengine 配置的参数化 SQL。
type TDengineOPCRepository struct {
	pool   *pgxpool.Pool
	cipher *security.ConnectionSecretCipher
}

// NewTDengineOPCRepository 创建TDengine 与 OPC DA仓储。
func NewTDengineOPCRepository(pool *pgxpool.Pool, ciphers ...*security.ConnectionSecretCipher) *TDengineOPCRepository {
	repository := &TDengineOPCRepository{pool: pool}
	if len(ciphers) > 0 {
		repository.cipher = ciphers[0]
	}
	return repository
}

// CreateTdengineConfig 创建 TDengine 配置。
func (r *TDengineOPCRepository) CreateTdengineConfig(ctx context.Context, params CreateTdengineConfigParams) (*ProtocolConnectionRecord, error) {
	optionsPayload, err := marshalTDengineOPCJSONObject(params.Options, true)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 TDengine 配置事务失败", err)
	}
	defer rollbackTDengineOPCTxQuietly(ctx, tx)

	record, err := r.createConnectionTx(ctx, tx, createTDengineOPCConnectionTxParams{
		ProjectID: params.ProjectID,
		UserID:    params.UserID,
		Name:      params.Name,
		Type:      "tdengine",
		IsEnabled: params.IsEnabled,
		Metadata: map[string]any{
			"protocol": params.Protocol, "host": params.Host, "port": params.Port,
			"username": params.Username, "databaseName": params.DatabaseName,
			"timezone": params.Timezone, "tlsSkipVerify": params.TLSSkipVerify,
			"options": cloneTDengineOPCMap(params.Options),
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
func (r *TDengineOPCRepository) UpdateTdengineConfig(ctx context.Context, connectionID string, params CreateTdengineConfigParams) (*ProtocolConnectionRecord, error) {
	optionsPayload, err := marshalTDengineOPCJSONObject(params.Options, true)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 TDengine 更新事务失败", err)
	}
	defer rollbackTDengineOPCTxQuietly(ctx, tx)
	metadataPayload, err := json.Marshal(map[string]any{"protocol": params.Protocol, "host": params.Host, "port": params.Port, "username": params.Username, "databaseName": params.DatabaseName, "timezone": params.Timezone, "tlsSkipVerify": params.TLSSkipVerify, "options": cloneTDengineOPCMap(params.Options)})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "序列化 TDengine 连接配置失败", err)
	}
	record := ProtocolConnectionRecord{}
	err = tx.QueryRow(ctx, `UPDATE data_connections SET name=$3,is_enabled=$4,metadata=$5::jsonb,updated_by=$6,updated_at=now() WHERE project_id=$1 AND id=$2 AND type='tdengine' RETURNING id,project_id,name,type,is_enabled,created_at,updated_at`, params.ProjectID, connectionID, params.Name, connectionEnabledValue(params.IsEnabled), string(metadataPayload), params.UserID).Scan(&record.ID, &record.ProjectID, &record.Name, &record.Type, &record.IsEnabled, &record.CreatedAt, &record.UpdatedAt)
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

type createTDengineOPCConnectionTxParams struct {
	ProjectID string
	UserID    string
	Name      string
	Type      string
	IsEnabled *bool
	Metadata  map[string]any
}

// createConnectionTx 统一写入 data_connections 主表。
func (r *TDengineOPCRepository) createConnectionTx(ctx context.Context, tx pgx.Tx, params createTDengineOPCConnectionTxParams) (*ProtocolConnectionRecord, error) {
	metadataPayload, err := json.Marshal(params.Metadata)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "序列化 TDengine 连接元数据失败", err)
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
			is_enabled,
			metadata,
			display_order,
			created_by,
			updated_by
		)
		VALUES ($1, $2, $3, 'protocol', $4, $5::jsonb,
			COALESCE((SELECT MAX(display_order) + 1 FROM data_connections WHERE project_id = $1), 0),
			$6, $6)
		RETURNING id, project_id, name, type, is_enabled, created_at, updated_at
	`, params.ProjectID, params.Name, params.Type, connectionEnabledValue(params.IsEnabled), string(metadataPayload), params.UserID).Scan(
		&record.ID,
		&record.ProjectID,
		&record.Name,
		&record.Type,
		&record.IsEnabled,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, translateConnectionWriteError("写入 TDengine 连接失败", err)
	}
	return &record, nil
}

func marshalTDengineOPCJSONObject(input map[string]any, emptyAsObject bool) (any, error) {
	if input == nil {
		if emptyAsObject {
			return "{}", nil
		}
		return nil, nil
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TDengine JSON 配置格式无效", err)
	}
	return string(payload), nil
}

func cloneTDengineOPCMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func rollbackTDengineOPCTxQuietly(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}
