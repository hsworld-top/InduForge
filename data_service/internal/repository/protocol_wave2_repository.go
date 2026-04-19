package repository

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// CreateOpcuaConfigParams 描述 OPC UA 配置落库参数。
type CreateOpcuaConfigParams struct {
	ProjectID      string
	UserID         string
	Name           string
	Status         string
	Endpoint       string
	SecurityPolicy string
	SecurityMode   string
	AuthType       string
	Username       *string
	Password       *string
	SamplingMS     int
	Options        map[string]any
}

// CreateS7ConfigParams 描述 S7 配置落库参数。
type CreateS7ConfigParams struct {
	ProjectID      string
	UserID         string
	Name           string
	Status         string
	Host           string
	Port           int
	Rack           int
	Slot           int
	PollIntervalMS int
	Options        map[string]any
}

// CreateModbusConfigParams 描述 Modbus 配置落库参数。
type CreateModbusConfigParams struct {
	ProjectID      string
	UserID         string
	Name           string
	Status         string
	Mode           string
	Host           *string
	Port           *int
	SerialConfig   map[string]any
	HasSerial      bool
	SlaveID        int
	StartAddress   int
	Quantity       int
	PollIntervalMS int
	Options        map[string]any
}

// CreateTdengineConfigParams 描述 TDengine 配置落库参数。
type CreateTdengineConfigParams struct {
	ProjectID    string
	UserID       string
	Name         string
	Status       string
	DSN          string
	DatabaseName string
	Timezone     *string
	Options      map[string]any
}

// ProtocolWave2Repository 负责第二波协议配置（opcua/s7/modbus/tdengine）参数化 SQL。
type ProtocolWave2Repository struct {
	pool *pgxpool.Pool
}

// NewProtocolWave2Repository 创建第二波协议仓储。
func NewProtocolWave2Repository(pool *pgxpool.Pool) *ProtocolWave2Repository {
	return &ProtocolWave2Repository{pool: pool}
}

// CreateOpcuaConfig 创建 OPC UA 配置。
func (r *ProtocolWave2Repository) CreateOpcuaConfig(ctx context.Context, params CreateOpcuaConfigParams) (*ProtocolConnectionRecord, error) {
	optionsPayload, err := marshalWave2JSONObject(params.Options, true)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 OPC UA 配置事务失败", err)
	}
	defer rollbackWave2TxQuietly(ctx, tx)

	record, err := r.createConnectionTx(ctx, tx, createWave2ConnectionTxParams{
		ProjectID: params.ProjectID,
		UserID:    params.UserID,
		Name:      params.Name,
		Type:      "opcua",
		Status:    params.Status,
		Metadata: map[string]any{
			"endpoint": params.Endpoint,
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO data_opcua_configs (
			connection_id,
			endpoint,
			security_policy,
			security_mode,
			auth_type,
			username,
			password,
			sampling_ms,
			options
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)
	`, record.ID, params.Endpoint, params.SecurityPolicy, params.SecurityMode, params.AuthType, params.Username, params.Password, params.SamplingMS, optionsPayload)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 OPC UA 配置失败", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 OPC UA 配置事务失败", err)
	}
	return record, nil
}

// CreateS7Config 创建 S7 配置。
func (r *ProtocolWave2Repository) CreateS7Config(ctx context.Context, params CreateS7ConfigParams) (*ProtocolConnectionRecord, error) {
	optionsPayload, err := marshalWave2JSONObject(params.Options, true)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 S7 配置事务失败", err)
	}
	defer rollbackWave2TxQuietly(ctx, tx)

	record, err := r.createConnectionTx(ctx, tx, createWave2ConnectionTxParams{
		ProjectID: params.ProjectID,
		UserID:    params.UserID,
		Name:      params.Name,
		Type:      "s7",
		Status:    params.Status,
		Metadata: map[string]any{
			"host": params.Host,
			"rack": params.Rack,
			"slot": params.Slot,
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO data_s7_configs (
			connection_id,
			host,
			port,
			rack,
			slot,
			poll_interval_ms,
			options
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
	`, record.ID, params.Host, params.Port, params.Rack, params.Slot, params.PollIntervalMS, optionsPayload)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 S7 配置失败", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 S7 配置事务失败", err)
	}
	return record, nil
}

// CreateModbusConfig 创建 Modbus 配置。
func (r *ProtocolWave2Repository) CreateModbusConfig(ctx context.Context, params CreateModbusConfigParams) (*ProtocolConnectionRecord, error) {
	optionsPayload, err := marshalWave2JSONObject(params.Options, true)
	if err != nil {
		return nil, err
	}
	serialPayload, err := marshalWave2JSONObject(params.SerialConfig, !params.HasSerial)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 Modbus 配置事务失败", err)
	}
	defer rollbackWave2TxQuietly(ctx, tx)

	record, err := r.createConnectionTx(ctx, tx, createWave2ConnectionTxParams{
		ProjectID: params.ProjectID,
		UserID:    params.UserID,
		Name:      params.Name,
		Type:      "modbus",
		Status:    params.Status,
		Metadata: map[string]any{
			"mode": params.Mode,
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO data_modbus_configs (
			connection_id,
			mode,
			host,
			port,
			serial_config,
			slave_id,
			start_address,
			quantity,
			poll_interval_ms,
			options
		)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10::jsonb)
	`, record.ID, params.Mode, params.Host, params.Port, serialPayload, params.SlaveID, params.StartAddress, params.Quantity, params.PollIntervalMS, optionsPayload)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 Modbus 配置失败", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 Modbus 配置事务失败", err)
	}
	return record, nil
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
			"database": params.DatabaseName,
		},
	})
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO data_tdengine_configs (
			connection_id,
			dsn,
			database_name,
			timezone,
			options
		)
		VALUES ($1, $2, $3, $4, $5::jsonb)
	`, record.ID, params.DSN, params.DatabaseName, params.Timezone, optionsPayload)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 TDengine 配置失败", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 TDengine 配置事务失败", err)
	}
	return record, nil
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
	err = tx.QueryRow(ctx, `
		INSERT INTO data_connections (
			project_id,
			name,
			type,
			category,
			status,
			metadata,
			created_by,
			updated_by
		)
		VALUES ($1, $2, $3, 'protocol', $4, $5::jsonb, $6, $6)
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
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 wave2 连接失败", err)
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

func rollbackWave2TxQuietly(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}
