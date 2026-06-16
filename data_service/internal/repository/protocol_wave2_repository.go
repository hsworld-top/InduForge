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
	Redundancy     map[string]any
}

// UpdateOpcuaConfigParams 描述 OPC UA 配置更新落库参数。
type UpdateOpcuaConfigParams struct {
	ConnectionID string
	CreateOpcuaConfigParams
}

// CreateS7ConfigParams 描述 S7 配置落库参数。
type CreateS7ConfigParams struct {
	ProjectID            string
	UserID               string
	Name                 string
	Status               string
	Host                 string
	Port                 int
	Rack                 int
	Slot                 int
	PollIntervalMS       int
	PlcFamily            string
	CommunicationMode    string
	LocalTSAP            *string
	RemoteTSAP           *string
	ConnectTimeoutMS     int
	ReadTimeoutMS        int
	PDUSize              *int
	MaxReadBytes         *int
	MaxGapBytes          int
	MaxConcurrentReads   int
	ByteOrder            string
	WordOrder            string
	OptimizedBlockAccess bool
	AllowAbsoluteAddress bool
	AllowSymbolAddress   bool
	SupportedAreas       []any
	Options              map[string]any
	Redundancy           map[string]any
}

// UpdateS7ConfigParams 描述 S7 配置更新落库参数。
type UpdateS7ConfigParams struct {
	ConnectionID string
	CreateS7ConfigParams
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
	Redundancy     map[string]any
}

// UpdateModbusConfigParams 描述 Modbus 配置更新落库参数。
type UpdateModbusConfigParams struct {
	ConnectionID string
	CreateModbusConfigParams
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
// 说明：当前只保留 schema 与参数化落库实现，便于后续进入 Phase 2 时复用；
// Phase 1 不再通过这些对象对外宣称正式协议能力。
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
			"endpoint":       params.Endpoint,
			"securityPolicy": params.SecurityPolicy,
			"securityMode":   params.SecurityMode,
			"authType":       params.AuthType,
			"username":       params.Username,
			"password":       params.Password,
			"samplingMs":     params.SamplingMS,
			"options":        cloneWave2Map(params.Options),
			"redundancy":     cloneWave2Map(params.Redundancy),
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

// UpdateOpcuaConfig 更新 OPC UA 配置。
func (r *ProtocolWave2Repository) UpdateOpcuaConfig(ctx context.Context, params UpdateOpcuaConfigParams) (*ProtocolConnectionRecord, error) {
	optionsPayload, err := marshalWave2JSONObject(params.Options, true)
	if err != nil {
		return nil, err
	}
	metadataPayload, err := json.Marshal(map[string]any{
		"endpoint":       params.Endpoint,
		"securityPolicy": params.SecurityPolicy,
		"securityMode":   params.SecurityMode,
		"authType":       params.AuthType,
		"username":       params.Username,
		"password":       params.Password,
		"samplingMs":     params.SamplingMS,
		"options":        cloneWave2Map(params.Options),
		"redundancy":     cloneWave2Map(params.Redundancy),
	})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "序列化 OPC UA 连接元数据失败", err)
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 OPC UA 配置更新事务失败", err)
	}
	defer rollbackWave2TxQuietly(ctx, tx)

	record := ProtocolConnectionRecord{}
	err = tx.QueryRow(ctx, `
		UPDATE data_connections
		SET name = $3,
			type = 'opcua',
			category = 'protocol',
			status = $4,
			metadata = $5::jsonb,
			updated_by = $6,
			updated_at = now()
		WHERE project_id = $1 AND id = $2 AND type = 'opcua'
		RETURNING id, project_id, name, type, status, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.Name, params.Status, string(metadataPayload), params.UserID).Scan(
		&record.ID,
		&record.ProjectID,
		&record.Name,
		&record.Type,
		&record.Status,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "OPC UA 连接不存在")
		}
		return nil, translateConnectionWriteError("更新 OPC UA 连接失败", err)
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
		ON CONFLICT (connection_id) DO UPDATE
		SET endpoint = EXCLUDED.endpoint,
			security_policy = EXCLUDED.security_policy,
			security_mode = EXCLUDED.security_mode,
			auth_type = EXCLUDED.auth_type,
			username = EXCLUDED.username,
			password = EXCLUDED.password,
			sampling_ms = EXCLUDED.sampling_ms,
			options = EXCLUDED.options,
			updated_at = now()
	`, params.ConnectionID, params.Endpoint, params.SecurityPolicy, params.SecurityMode, params.AuthType, params.Username, params.Password, params.SamplingMS, optionsPayload)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "同步 OPC UA 配置失败", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 OPC UA 配置更新事务失败", err)
	}
	return &record, nil
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
			"host":              params.Host,
			"port":              params.Port,
			"rack":              params.Rack,
			"slot":              params.Slot,
			"pollIntervalMs":    params.PollIntervalMS,
			"plcFamily":         params.PlcFamily,
			"communicationMode": params.CommunicationMode,
			"options":           cloneWave2Map(params.Options),
			"redundancy":        cloneWave2Map(params.Redundancy),
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

	if err := upsertS7ProfileTx(ctx, tx, record.ID, params); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 S7 配置事务失败", err)
	}
	return record, nil
}

// UpdateS7Config 更新 S7 配置，并同步 PLC 档案。
func (r *ProtocolWave2Repository) UpdateS7Config(ctx context.Context, params UpdateS7ConfigParams) (*ProtocolConnectionRecord, error) {
	optionsPayload, err := marshalWave2JSONObject(params.Options, true)
	if err != nil {
		return nil, err
	}
	metadataPayload, err := json.Marshal(map[string]any{
		"host":              params.Host,
		"port":              params.Port,
		"rack":              params.Rack,
		"slot":              params.Slot,
		"pollIntervalMs":    params.PollIntervalMS,
		"plcFamily":         params.PlcFamily,
		"communicationMode": params.CommunicationMode,
		"options":           cloneWave2Map(params.Options),
		"redundancy":        cloneWave2Map(params.Redundancy),
	})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "序列化 S7 连接元数据失败", err)
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 S7 配置更新事务失败", err)
	}
	defer rollbackWave2TxQuietly(ctx, tx)

	record := ProtocolConnectionRecord{}
	err = tx.QueryRow(ctx, `
		UPDATE data_connections
		SET name = $3,
			type = 's7',
			category = 'protocol',
			status = $4,
			metadata = $5::jsonb,
			updated_by = $6,
			updated_at = now()
		WHERE project_id = $1 AND id = $2 AND type = 's7'
		RETURNING id, project_id, name, type, status, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.Name, params.Status, string(metadataPayload), params.UserID).Scan(
		&record.ID,
		&record.ProjectID,
		&record.Name,
		&record.Type,
		&record.Status,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "S7 连接不存在")
		}
		return nil, translateConnectionWriteError("更新 S7 连接失败", err)
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
		ON CONFLICT (connection_id) DO UPDATE
		SET host = EXCLUDED.host,
			port = EXCLUDED.port,
			rack = EXCLUDED.rack,
			slot = EXCLUDED.slot,
			poll_interval_ms = EXCLUDED.poll_interval_ms,
			options = EXCLUDED.options,
			updated_at = now()
	`, params.ConnectionID, params.Host, params.Port, params.Rack, params.Slot, params.PollIntervalMS, optionsPayload)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "同步 S7 配置失败", err)
	}

	if err := upsertS7ProfileTx(ctx, tx, params.ConnectionID, params.CreateS7ConfigParams); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 S7 配置更新事务失败", err)
	}
	return &record, nil
}

func upsertS7ProfileTx(ctx context.Context, tx pgx.Tx, connectionID string, params CreateS7ConfigParams) error {
	supportedAreasPayload, err := json.Marshal(params.SupportedAreas)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "S7 支持地址区格式无效", err)
	}
	optionsPayload, err := marshalWave2JSONObject(params.Options, true)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO data_s7_plc_profiles (
			project_id, connection_id, plc_family, communication_mode, host, port,
			rack, slot, local_tsap, remote_tsap, poll_interval_ms, connect_timeout_ms,
			read_timeout_ms, pdu_size, max_read_bytes, max_gap_bytes, max_concurrent_reads,
			byte_order, word_order, optimized_block_access, allow_absolute_address,
			allow_symbol_address, supported_areas, options, created_by, updated_by
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23::jsonb, $24::jsonb, $25, $25
		)
		ON CONFLICT (project_id, connection_id) DO UPDATE
		SET plc_family = EXCLUDED.plc_family,
		    communication_mode = EXCLUDED.communication_mode,
		    host = EXCLUDED.host,
		    port = EXCLUDED.port,
		    rack = EXCLUDED.rack,
		    slot = EXCLUDED.slot,
		    local_tsap = EXCLUDED.local_tsap,
		    remote_tsap = EXCLUDED.remote_tsap,
		    poll_interval_ms = EXCLUDED.poll_interval_ms,
		    connect_timeout_ms = EXCLUDED.connect_timeout_ms,
		    read_timeout_ms = EXCLUDED.read_timeout_ms,
		    pdu_size = EXCLUDED.pdu_size,
		    max_read_bytes = EXCLUDED.max_read_bytes,
		    max_gap_bytes = EXCLUDED.max_gap_bytes,
		    max_concurrent_reads = EXCLUDED.max_concurrent_reads,
		    byte_order = EXCLUDED.byte_order,
		    word_order = EXCLUDED.word_order,
		    optimized_block_access = EXCLUDED.optimized_block_access,
		    allow_absolute_address = EXCLUDED.allow_absolute_address,
		    allow_symbol_address = EXCLUDED.allow_symbol_address,
		    supported_areas = EXCLUDED.supported_areas,
		    options = EXCLUDED.options,
		    updated_by = EXCLUDED.updated_by,
		    updated_at = now()
	`, params.ProjectID, connectionID, params.PlcFamily, params.CommunicationMode, params.Host, params.Port,
		params.Rack, params.Slot, params.LocalTSAP, params.RemoteTSAP, params.PollIntervalMS, params.ConnectTimeoutMS,
		params.ReadTimeoutMS, params.PDUSize, params.MaxReadBytes, params.MaxGapBytes, params.MaxConcurrentReads,
		params.ByteOrder, params.WordOrder, params.OptimizedBlockAccess, params.AllowAbsoluteAddress,
		params.AllowSymbolAddress, string(supportedAreasPayload), optionsPayload, params.UserID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "同步 S7 PLC 档案失败", err)
	}
	return nil
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
			"mode":           params.Mode,
			"host":           params.Host,
			"port":           params.Port,
			"serialConfig":   cloneWave2Map(params.SerialConfig),
			"slaveId":        params.SlaveID,
			"startAddress":   params.StartAddress,
			"quantity":       params.Quantity,
			"pollIntervalMs": params.PollIntervalMS,
			"options":        cloneWave2Map(params.Options),
			"redundancy":     cloneWave2Map(params.Redundancy),
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

// UpdateModbusConfig 更新 Modbus 配置。
func (r *ProtocolWave2Repository) UpdateModbusConfig(ctx context.Context, params UpdateModbusConfigParams) (*ProtocolConnectionRecord, error) {
	optionsPayload, err := marshalWave2JSONObject(params.Options, true)
	if err != nil {
		return nil, err
	}
	serialPayload, err := marshalWave2JSONObject(params.SerialConfig, !params.HasSerial)
	if err != nil {
		return nil, err
	}
	metadataPayload, err := json.Marshal(map[string]any{
		"mode":           params.Mode,
		"host":           params.Host,
		"port":           params.Port,
		"serialConfig":   cloneWave2Map(params.SerialConfig),
		"slaveId":        params.SlaveID,
		"startAddress":   params.StartAddress,
		"quantity":       params.Quantity,
		"pollIntervalMs": params.PollIntervalMS,
		"options":        cloneWave2Map(params.Options),
		"redundancy":     cloneWave2Map(params.Redundancy),
	})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "序列化 Modbus 连接元数据失败", err)
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 Modbus 配置更新事务失败", err)
	}
	defer rollbackWave2TxQuietly(ctx, tx)

	record := ProtocolConnectionRecord{}
	err = tx.QueryRow(ctx, `
		UPDATE data_connections
		SET name = $3,
			type = 'modbus',
			category = 'protocol',
			status = $4,
			metadata = $5::jsonb,
			updated_by = $6,
			updated_at = now()
		WHERE project_id = $1 AND id = $2 AND type = 'modbus'
		RETURNING id, project_id, name, type, status, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.Name, params.Status, string(metadataPayload), params.UserID).Scan(
		&record.ID,
		&record.ProjectID,
		&record.Name,
		&record.Type,
		&record.Status,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Modbus 连接不存在")
		}
		return nil, translateConnectionWriteError("更新 Modbus 连接失败", err)
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
		ON CONFLICT (connection_id) DO UPDATE
		SET mode = EXCLUDED.mode,
			host = EXCLUDED.host,
			port = EXCLUDED.port,
			serial_config = EXCLUDED.serial_config,
			slave_id = EXCLUDED.slave_id,
			start_address = EXCLUDED.start_address,
			quantity = EXCLUDED.quantity,
			poll_interval_ms = EXCLUDED.poll_interval_ms,
			options = EXCLUDED.options,
			updated_at = now()
	`, params.ConnectionID, params.Mode, params.Host, params.Port, serialPayload, params.SlaveID, params.StartAddress, params.Quantity, params.PollIntervalMS, optionsPayload)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "同步 Modbus 配置失败", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 Modbus 配置更新事务失败", err)
	}
	return &record, nil
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
			"dsn":      params.DSN,
			"database": params.DatabaseName,
			"timezone": params.Timezone,
			"options":  cloneWave2Map(params.Options),
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
