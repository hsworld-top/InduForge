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
// UpdateOpcuaConfigParams 描述 OPC UA 配置更新落库参数。
// CreateS7ConfigParams 描述 S7 配置落库参数。
// UpdateS7ConfigParams 描述 S7 配置更新落库参数。
// CreateModbusConfigParams 描述 Modbus 配置落库参数。
// UpdateModbusConfigParams 描述 Modbus 配置更新落库参数。
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
// UpdateOpcuaConfig 更新 OPC UA 配置。
// CreateS7Config 创建 S7 配置。
// UpdateS7Config 更新 S7 配置，并同步 PLC 档案。
// CreateModbusConfig 创建 Modbus 配置。
// UpdateModbusConfig 更新 Modbus 配置。
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
