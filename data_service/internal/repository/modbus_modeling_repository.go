package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// ModbusRegisterGroupRecord 表示 Modbus 寄存器组。
type ModbusRegisterGroupRecord struct {
	ID           string
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	Description  *string
	SortOrder    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ModbusSlaveDeviceRecord 表示 Modbus 接入源下的真实从站设备。
type ModbusSlaveDeviceRecord struct {
	ID                    string
	ProjectID             string
	ConnectionID          string
	UnitID                int
	Name                  string
	Description           *string
	Enabled               bool
	DefaultPollIntervalMS int
	DefaultByteOrder      string
	DefaultWordOrder      string
	RequestIntervalMS     *int
	TimeoutMS             *int
	RetryCount            *int
	SortOrder             int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ModbusRegisterRecord 表示 Modbus 寄存器变量定义。
type ModbusRegisterRecord struct {
	ID              string
	ProjectID       string
	ConnectionID    string
	GroupID         *string
	Name            string
	Code            string
	UnitID          int
	Area            string
	Address         int
	AddressBase     string
	ProtocolAddress int
	Quantity        int
	DataType        string
	ByteOrder       string
	WordOrder       string
	BitIndex        *int
	Scale           float64
	Offset          float64
	Unit            *string
	PollIntervalMS  int
	TimeoutMS       *int
	RetryCount      *int
	AccessLevel     string
	Description     *string
	SortOrder       int
	Status          string
	DataPointID     *string
	DataPointPath   *string
	DataPointStatus *string
	LastValue       []byte
	Quality         string
	LastUpdatedAt   *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ModbusRegisterListFilter 描述变量列表的服务端筛选、搜索和排序条件。
type ModbusRegisterListFilter struct {
	GroupID      *string
	Search       string
	QuickFilter  string
	UnitID       *int
	Area         string
	AddressStart *int
	AddressEnd   *int
	DataType     string
	SlaveEnabled *bool
	SortBy       string
	SortOrder    string
}

// CreateModbusRegisterGroupParams 描述创建寄存器组参数。
type CreateModbusRegisterGroupParams struct {
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	Description  *string
	SortOrder    int
	UserID       string
}

// UpdateModbusRegisterGroupParams 描述更新寄存器组参数。
type UpdateModbusRegisterGroupParams struct {
	ID           string
	ProjectID    string
	ConnectionID string
	ParentID     *string
	HasParentID  bool
	Name         string
	Description  *string
	SortOrder    int
	UserID       string
}

// CreateModbusSlaveDeviceParams 描述创建 Modbus 从站参数。
type CreateModbusSlaveDeviceParams struct {
	ProjectID             string
	ConnectionID          string
	UnitID                int
	Name                  string
	Description           *string
	Enabled               bool
	DefaultPollIntervalMS int
	DefaultByteOrder      string
	DefaultWordOrder      string
	RequestIntervalMS     *int
	TimeoutMS             *int
	RetryCount            *int
	SortOrder             int
	UserID                string
}

// UpdateModbusSlaveDeviceParams 描述更新 Modbus 从站参数。
type UpdateModbusSlaveDeviceParams struct {
	ID                    string
	ProjectID             string
	ConnectionID          string
	UnitID                int
	Name                  string
	Description           *string
	Enabled               bool
	DefaultPollIntervalMS int
	DefaultByteOrder      string
	DefaultWordOrder      string
	RequestIntervalMS     *int
	TimeoutMS             *int
	RetryCount            *int
	SortOrder             int
	UserID                string
}

// CreateModbusRegisterParams 描述创建寄存器变量参数。
type CreateModbusRegisterParams struct {
	ProjectID       string
	ConnectionID    string
	GroupID         *string
	Name            string
	Code            string
	UnitID          int
	Area            string
	Address         int
	AddressBase     string
	ProtocolAddress int
	Quantity        int
	DataType        string
	ByteOrder       string
	WordOrder       string
	BitIndex        *int
	Scale           float64
	Offset          float64
	Unit            *string
	PollIntervalMS  int
	TimeoutMS       *int
	RetryCount      *int
	AccessLevel     string
	Description     *string
	SortOrder       int
	UserID          string
}

// UpdateModbusRegisterParams 描述更新寄存器变量参数。
type UpdateModbusRegisterParams struct {
	ID              string
	ProjectID       string
	ConnectionID    string
	GroupID         *string
	HasGroupID      bool
	Name            string
	Code            string
	UnitID          int
	Area            string
	Address         int
	AddressBase     string
	ProtocolAddress int
	Quantity        int
	DataType        string
	ByteOrder       string
	WordOrder       string
	BitIndex        *int
	Scale           float64
	Offset          float64
	Unit            *string
	PollIntervalMS  int
	TimeoutMS       *int
	RetryCount      *int
	AccessLevel     string
	Description     *string
	SortOrder       int
	Status          string
	UserID          string
}

// UpdateModbusRegisterLastValueParams 描述开发态读取后写回的最近值快照。
type UpdateModbusRegisterLastValueParams struct {
	ProjectID    string
	ConnectionID string
	RegisterID   string
	LastValue    any
	Quality      string
	UserID       string
}

// BatchUpdateModbusRegistersParams 描述批量更新变量公共属性的参数。
type BatchUpdateModbusRegistersParams struct {
	ProjectID      string
	ConnectionID   string
	IDs            []string
	Filter         ModbusRegisterListFilter
	UseFilter      bool
	GroupID        *string
	HasGroupID     bool
	UnitID         *int
	PollIntervalMS *int
	ByteOrder      string
	WordOrder      string
	Status         string
	UserID         string
}

// ModbusModelingRepository 封装 Modbus 建模参数化 SQL。
type ModbusModelingRepository struct {
	pool *pgxpool.Pool
}

// NewModbusModelingRepository 创建 Modbus 建模仓储。
func NewModbusModelingRepository(pool *pgxpool.Pool) *ModbusModelingRepository {
	return &ModbusModelingRepository{pool: pool}
}

// ListGroups 返回连接下的寄存器组。
func (r *ModbusModelingRepository) ListGroups(ctx context.Context, projectID, connectionID string) ([]ModbusRegisterGroupRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, connection_id, parent_id, name, description, sort_order, created_at, updated_at
		FROM data_modbus_register_groups
		WHERE project_id = $1 AND connection_id = $2
		ORDER BY sort_order ASC, created_at ASC
	`, projectID, connectionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 Modbus 寄存器组失败", err)
	}
	defer rows.Close()

	result := make([]ModbusRegisterGroupRecord, 0)
	for rows.Next() {
		record, scanErr := scanModbusRegisterGroupRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 Modbus 寄存器组失败", err)
	}
	return result, nil
}

// CreateGroup 创建寄存器组。
func (r *ModbusModelingRepository) CreateGroup(ctx context.Context, params CreateModbusRegisterGroupParams) (*ModbusRegisterGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO data_modbus_register_groups (
			project_id, connection_id, parent_id, name, description, sort_order, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id, project_id, connection_id, parent_id, name, description, sort_order, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.ParentID, params.Name, params.Description, params.SortOrder, params.UserID)

	record, err := scanModbusRegisterGroupRecord(row)
	if err != nil {
		return nil, translateModbusModelingWriteError(err)
	}
	return &record, nil
}

// UpdateGroup 更新寄存器组。
func (r *ModbusModelingRepository) UpdateGroup(ctx context.Context, params UpdateModbusRegisterGroupParams) (*ModbusRegisterGroupRecord, error) {
	parentSQL := "parent_id"
	args := []any{params.ProjectID, params.ConnectionID, params.ID, params.Name, params.Description, params.SortOrder, params.UserID}
	if params.HasParentID {
		parentSQL = "$8"
		args = append(args, params.ParentID)
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE data_modbus_register_groups
		SET name = $4,
			description = $5,
			sort_order = $6,
			updated_by = $7,
			updated_at = now(),
			parent_id = `+parentSQL+`
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
		RETURNING id, project_id, connection_id, parent_id, name, description, sort_order, created_at, updated_at
	`, args...)

	record, err := scanModbusRegisterGroupRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Modbus 寄存器组不存在")
		}
		return nil, translateModbusModelingWriteError(err)
	}
	return &record, nil
}

// DeleteGroup 删除寄存器组，组内变量移动到未分组。
func (r *ModbusModelingRepository) DeleteGroup(ctx context.Context, projectID, connectionID, groupID, userID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 Modbus 寄存器组删除事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)

	if _, err := tx.Exec(ctx, `
		UPDATE data_modbus_registers
		SET group_id = NULL, updated_by = $4, updated_at = now()
		WHERE project_id = $1 AND connection_id = $2 AND group_id = $3
	`, projectID, connectionID, groupID, userID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "移动 Modbus 变量失败", err)
	}

	tag, err := tx.Exec(ctx, `
		DELETE FROM data_modbus_register_groups
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
	`, projectID, connectionID, groupID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 Modbus 寄存器组失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Modbus 寄存器组不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 Modbus 寄存器组删除事务失败", err)
	}
	return nil
}

// ListSlaveDevices 返回连接下的 Modbus 从站设备。
func (r *ModbusModelingRepository) ListSlaveDevices(ctx context.Context, projectID, connectionID string) ([]ModbusSlaveDeviceRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, connection_id, unit_id, name, description, enabled,
		       default_poll_interval_ms, default_byte_order, default_word_order,
		       request_interval_ms, timeout_ms, retry_count, sort_order, created_at, updated_at
		FROM data_modbus_slave_devices
		WHERE project_id = $1 AND connection_id = $2
		ORDER BY sort_order ASC, unit_id ASC, created_at ASC
	`, projectID, connectionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 Modbus 从站失败", err)
	}
	defer rows.Close()

	result := make([]ModbusSlaveDeviceRecord, 0)
	for rows.Next() {
		record, scanErr := scanModbusSlaveDeviceRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 Modbus 从站失败", err)
	}
	return result, nil
}

// GetSlaveDeviceByUnitID 按 unitId 返回从站设备。
func (r *ModbusModelingRepository) GetSlaveDeviceByUnitID(ctx context.Context, projectID, connectionID string, unitID int) (*ModbusSlaveDeviceRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, connection_id, unit_id, name, description, enabled,
		       default_poll_interval_ms, default_byte_order, default_word_order,
		       request_interval_ms, timeout_ms, retry_count, sort_order, created_at, updated_at
		FROM data_modbus_slave_devices
		WHERE project_id = $1 AND connection_id = $2 AND unit_id = $3
	`, projectID, connectionID, unitID)
	record, err := scanModbusSlaveDeviceRecord(row)
	if err != nil {
		if appErr, ok := err.(*apperrors.AppError); ok && appErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// CreateSlaveDevice 创建 Modbus 从站设备。
func (r *ModbusModelingRepository) CreateSlaveDevice(ctx context.Context, params CreateModbusSlaveDeviceParams) (*ModbusSlaveDeviceRecord, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO data_modbus_slave_devices (
			project_id, connection_id, unit_id, name, description, enabled,
			default_poll_interval_ms, default_byte_order, default_word_order,
			request_interval_ms, timeout_ms, retry_count, sort_order, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14)
		RETURNING id, project_id, connection_id, unit_id, name, description, enabled,
		          default_poll_interval_ms, default_byte_order, default_word_order,
		          request_interval_ms, timeout_ms, retry_count, sort_order, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.UnitID, params.Name, params.Description, params.Enabled,
		params.DefaultPollIntervalMS, params.DefaultByteOrder, params.DefaultWordOrder, params.RequestIntervalMS,
		params.TimeoutMS, params.RetryCount, params.SortOrder, params.UserID)

	record, err := scanModbusSlaveDeviceRecord(row)
	if err != nil {
		return nil, translateModbusModelingWriteError(err)
	}
	return &record, nil
}

// UpdateSlaveDevice 更新 Modbus 从站设备。
func (r *ModbusModelingRepository) UpdateSlaveDevice(ctx context.Context, params UpdateModbusSlaveDeviceParams) (*ModbusSlaveDeviceRecord, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE data_modbus_slave_devices
		SET unit_id = $4,
		    name = $5,
		    description = $6,
		    enabled = $7,
		    default_poll_interval_ms = $8,
		    default_byte_order = $9,
		    default_word_order = $10,
		    request_interval_ms = $11,
		    timeout_ms = $12,
		    retry_count = $13,
		    sort_order = $14,
		    updated_by = $15,
		    updated_at = now()
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
		RETURNING id, project_id, connection_id, unit_id, name, description, enabled,
		          default_poll_interval_ms, default_byte_order, default_word_order,
		          request_interval_ms, timeout_ms, retry_count, sort_order, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.ID, params.UnitID, params.Name, params.Description,
		params.Enabled, params.DefaultPollIntervalMS, params.DefaultByteOrder, params.DefaultWordOrder,
		params.RequestIntervalMS, params.TimeoutMS, params.RetryCount, params.SortOrder, params.UserID)

	record, err := scanModbusSlaveDeviceRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Modbus 从站不存在")
		}
		return nil, translateModbusModelingWriteError(err)
	}
	return &record, nil
}

// DeleteSlaveDevice 删除未被变量引用的 Modbus 从站设备。
func (r *ModbusModelingRepository) DeleteSlaveDevice(ctx context.Context, projectID, connectionID, slaveID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 Modbus 从站删除事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)

	var unitID int
	if err := tx.QueryRow(ctx, `
		SELECT unit_id
		FROM data_modbus_slave_devices
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
	`, projectID, connectionID, slaveID).Scan(&unitID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Modbus 从站不存在")
		}
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 Modbus 从站失败", err)
	}
	var registerCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM data_modbus_registers
		WHERE project_id = $1 AND connection_id = $2 AND unit_id = $3
	`, projectID, connectionID, unitID).Scan(&registerCount); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计 Modbus 从站变量失败", err)
	}
	if registerCount > 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "从站下存在变量，请先迁移变量")
	}
	tag, err := tx.Exec(ctx, `
		DELETE FROM data_modbus_slave_devices
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
	`, projectID, connectionID, slaveID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 Modbus 从站失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Modbus 从站不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 Modbus 从站删除事务失败", err)
	}
	return nil
}

// ListRegisters 返回连接下的寄存器变量，可按 groupID 过滤。
func (r *ModbusModelingRepository) ListRegisters(ctx context.Context, projectID, connectionID string, groupID *string) ([]ModbusRegisterRecord, error) {
	where := "r.project_id = $1 AND r.connection_id = $2"
	args := []any{projectID, connectionID}
	if groupID != nil {
		where += " AND r.group_id = $3"
		args = append(args, groupID)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT r.id, r.project_id, r.connection_id, r.group_id, r.name, r.code,
		       r.unit_id, r.area, r.address, r.address_base, r.protocol_address, r.quantity,
		       r.data_type, r.byte_order, r.word_order, r.bit_index, r.scale, r.offset_value,
		       r.unit, r.poll_interval_ms, r.timeout_ms, r.retry_count, r.access_level,
		       r.description, r.sort_order, r.status, dp.id, dp.path, dp.status,
		       r.last_value, r.quality, r.last_updated_at, r.created_at, r.updated_at
		FROM data_modbus_registers r
		LEFT JOIN data_points dp
		  ON dp.project_id = r.project_id
		 AND dp.source_type = 'modbus.register'
		 AND dp.source_id = r.id
		WHERE `+where+`
		ORDER BY r.sort_order ASC, r.created_at ASC
	`, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 Modbus 变量失败", err)
	}
	defer rows.Close()

	result := make([]ModbusRegisterRecord, 0)
	for rows.Next() {
		record, scanErr := scanModbusRegisterRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 Modbus 变量失败", err)
	}
	return result, nil
}

// ListRegistersPage 返回当前条件下的一页 Modbus 变量，并同时返回匹配总数。
func (r *ModbusModelingRepository) ListRegistersPage(ctx context.Context, projectID, connectionID string, filter ModbusRegisterListFilter, page, pageSize int) ([]ModbusRegisterRecord, int, error) {
	where, args := buildModbusRegisterListWhere(projectID, connectionID, filter)

	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM data_modbus_registers r
		LEFT JOIN data_points dp
		  ON dp.project_id = r.project_id
		 AND dp.source_type = 'modbus.register'
		 AND dp.source_id = r.id
		WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计 Modbus 变量失败", err)
	}

	limitIndex := len(args) + 1
	offsetIndex := len(args) + 2
	queryArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT r.id, r.project_id, r.connection_id, r.group_id, r.name, r.code,
		       r.unit_id, r.area, r.address, r.address_base, r.protocol_address, r.quantity,
		       r.data_type, r.byte_order, r.word_order, r.bit_index, r.scale, r.offset_value,
		       r.unit, r.poll_interval_ms, r.timeout_ms, r.retry_count, r.access_level,
		       r.description, r.sort_order, r.status, dp.id, dp.path, dp.status,
		       r.last_value, r.quality, r.last_updated_at, r.created_at, r.updated_at
		FROM data_modbus_registers r
		LEFT JOIN data_points dp
		  ON dp.project_id = r.project_id
		 AND dp.source_type = 'modbus.register'
		 AND dp.source_id = r.id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, where, modbusRegisterOrderSQL(filter.SortBy, filter.SortOrder), limitIndex, offsetIndex), queryArgs...)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 Modbus 变量分页失败", err)
	}
	defer rows.Close()

	result := make([]ModbusRegisterRecord, 0)
	for rows.Next() {
		record, scanErr := scanModbusRegisterRecord(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 Modbus 变量分页失败", err)
	}
	return result, total, nil
}

func buildModbusRegisterListWhere(projectID, connectionID string, filter ModbusRegisterListFilter) (string, []any) {
	clauses := []string{"r.project_id = $1", "r.connection_id = $2"}
	args := []any{projectID, connectionID}
	addEqual := func(column, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	if filter.GroupID != nil {
		groupID := strings.TrimSpace(*filter.GroupID)
		if groupID == "__ungrouped__" {
			clauses = append(clauses, "r.group_id IS NULL")
		} else if groupID != "" {
			args = append(args, groupID)
			clauses = append(clauses, fmt.Sprintf("r.group_id = $%d", len(args)))
		}
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+search+"%")
		idx := len(args)
		clauses = append(clauses, fmt.Sprintf("(r.name ILIKE $%d OR r.code ILIKE $%d OR r.area ILIKE $%d OR CAST(r.address AS text) ILIKE $%d OR COALESCE(dp.path, '') ILIKE $%d)", idx, idx, idx, idx, idx))
	}
	if filter.UnitID != nil {
		args = append(args, *filter.UnitID)
		clauses = append(clauses, fmt.Sprintf("r.unit_id = $%d", len(args)))
	}
	if filter.SlaveEnabled != nil {
		args = append(args, *filter.SlaveEnabled)
		clauses = append(clauses, fmt.Sprintf(`EXISTS (
			SELECT 1 FROM data_modbus_slave_devices sd
			WHERE sd.project_id = r.project_id
			  AND sd.connection_id = r.connection_id
			  AND sd.unit_id = r.unit_id
			  AND sd.enabled = $%d
		)`, len(args)))
	}
	addEqual("r.area", filter.Area)
	addEqual("r.data_type", filter.DataType)
	if filter.AddressStart != nil {
		args = append(args, *filter.AddressStart)
		clauses = append(clauses, fmt.Sprintf("r.address >= $%d", len(args)))
	}
	if filter.AddressEnd != nil {
		args = append(args, *filter.AddressEnd)
		clauses = append(clauses, fmt.Sprintf("r.address <= $%d", len(args)))
	}
	switch strings.TrimSpace(filter.QuickFilter) {
	case "issue":
		clauses = append(clauses, "(r.status <> 'active' OR dp.id IS NULL OR COALESCE(dp.status, '') <> 'active')")
	case "datapoint":
		clauses = append(clauses, "(dp.id IS NULL OR COALESCE(dp.status, '') <> 'active')")
	case "writable":
		clauses = append(clauses, "LOWER(r.access_level) LIKE '%write%'")
	case "disabled":
		clauses = append(clauses, "r.status <> 'active'")
	}
	return strings.Join(clauses, " AND "), args
}

func modbusRegisterOrderSQL(sortBy, sortOrder string) string {
	direction := "ASC"
	if strings.EqualFold(strings.TrimSpace(sortOrder), "desc") || strings.EqualFold(strings.TrimSpace(sortOrder), "descending") {
		direction = "DESC"
	}
	columns := map[string]string{
		"name":            "r.name",
		"code":            "r.code",
		"unitId":          "r.unit_id",
		"area":            "r.area",
		"address":         "r.address",
		"protocolAddress": "r.protocol_address",
		"dataType":        "r.data_type",
		"pollIntervalMs":  "r.poll_interval_ms",
		"accessLevel":     "r.access_level",
		"status":          "r.status",
		"updatedAt":       "r.updated_at",
		"sortOrder":       "r.sort_order",
	}
	if column, ok := columns[strings.TrimSpace(sortBy)]; ok {
		return column + " " + direction + ", r.sort_order ASC, r.created_at ASC"
	}
	return "r.sort_order ASC, r.created_at ASC"
}

// ListRegisterIDsByFilter 返回当前筛选条件匹配的 Modbus 变量 ID，用于“全部结果”批量操作。
func (r *ModbusModelingRepository) ListRegisterIDsByFilter(ctx context.Context, projectID, connectionID string, filter ModbusRegisterListFilter) ([]string, error) {
	where, args := buildModbusRegisterListWhere(projectID, connectionID, filter)
	rows, err := r.pool.Query(ctx, `
		SELECT r.id
		FROM data_modbus_registers r
		LEFT JOIN data_points dp
		  ON dp.project_id = r.project_id
		 AND dp.source_type = 'modbus.register'
		 AND dp.source_id = r.id
		WHERE `+where+`
		ORDER BY `+modbusRegisterOrderSQL(filter.SortBy, filter.SortOrder), args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 Modbus 变量 ID 失败", err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 Modbus 变量 ID 失败", scanErr)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 Modbus 变量 ID 失败", err)
	}
	return ids, nil
}

// GetRegister 按项目和变量 ID 读取 Modbus 变量。
func (r *ModbusModelingRepository) GetRegister(ctx context.Context, projectID, registerID string) (*ModbusRegisterRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT r.id, r.project_id, r.connection_id, r.group_id, r.name, r.code,
		       r.unit_id, r.area, r.address, r.address_base, r.protocol_address, r.quantity,
		       r.data_type, r.byte_order, r.word_order, r.bit_index, r.scale, r.offset_value,
		       r.unit, r.poll_interval_ms, r.timeout_ms, r.retry_count, r.access_level,
		       r.description, r.sort_order, r.status, dp.id, dp.path, dp.status,
		       r.last_value, r.quality, r.last_updated_at, r.created_at, r.updated_at
		FROM data_modbus_registers r
		LEFT JOIN data_points dp
		  ON dp.project_id = r.project_id
		 AND dp.source_type = 'modbus.register'
		 AND dp.source_id = r.id
		WHERE r.project_id = $1 AND r.id = $2
	`, projectID, registerID)

	record, err := scanModbusRegisterRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Modbus 变量不存在")
		}
		return nil, err
	}
	return &record, nil
}

// CreateRegister 创建 Modbus 变量。
func (r *ModbusModelingRepository) CreateRegister(ctx context.Context, params CreateModbusRegisterParams) (*ModbusRegisterRecord, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO data_modbus_registers (
			project_id, connection_id, group_id, name, code, unit_id, area, address,
			address_base, protocol_address, quantity, data_type, byte_order, word_order,
			bit_index, scale, offset_value, unit, poll_interval_ms, timeout_ms, retry_count,
			access_level, description, sort_order, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
		        $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $25)
		RETURNING id, project_id, connection_id, group_id, name, code, unit_id, area,
		          address, address_base, protocol_address, quantity, data_type, byte_order,
		          word_order, bit_index, scale, offset_value, unit, poll_interval_ms,
		          timeout_ms, retry_count, access_level, description, sort_order, status,
		          NULL::uuid, NULL::text, NULL::text, NULL::jsonb, 'unknown'::text,
		          NULL::timestamptz, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.GroupID, params.Name, params.Code, params.UnitID, params.Area,
		params.Address, params.AddressBase, params.ProtocolAddress, params.Quantity, params.DataType, params.ByteOrder,
		params.WordOrder, params.BitIndex, params.Scale, params.Offset, params.Unit, params.PollIntervalMS,
		params.TimeoutMS, params.RetryCount, params.AccessLevel, params.Description, params.SortOrder, params.UserID)

	record, err := scanModbusRegisterRecord(row)
	if err != nil {
		return nil, translateModbusModelingWriteError(err)
	}
	return &record, nil
}

// UpdateRegister 更新 Modbus 变量。
func (r *ModbusModelingRepository) UpdateRegister(ctx context.Context, params UpdateModbusRegisterParams) (*ModbusRegisterRecord, error) {
	groupSQL := "group_id"
	args := []any{
		params.ProjectID, params.ConnectionID, params.ID, params.Name, params.Code, params.UnitID, params.Area,
		params.Address, params.AddressBase, params.ProtocolAddress, params.Quantity, params.DataType, params.ByteOrder,
		params.WordOrder, params.BitIndex, params.Scale, params.Offset, params.Unit, params.PollIntervalMS,
		params.TimeoutMS, params.RetryCount, params.AccessLevel, params.Description, params.SortOrder, params.Status,
		params.UserID,
	}
	if params.HasGroupID {
		groupSQL = "$27"
		args = append(args, params.GroupID)
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE data_modbus_registers
		SET name = $4,
			code = $5,
			unit_id = $6,
			area = $7,
			address = $8,
			address_base = $9,
			protocol_address = $10,
			quantity = $11,
			data_type = $12,
			byte_order = $13,
			word_order = $14,
			bit_index = $15,
			scale = $16,
			offset_value = $17,
			unit = $18,
			poll_interval_ms = $19,
			timeout_ms = $20,
			retry_count = $21,
			access_level = $22,
			description = $23,
			sort_order = $24,
			status = $25,
			updated_by = $26,
			updated_at = now(),
			group_id = `+groupSQL+`
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
		RETURNING id, project_id, connection_id, group_id, name, code, unit_id, area,
		          address, address_base, protocol_address, quantity, data_type, byte_order,
		          word_order, bit_index, scale, offset_value, unit, poll_interval_ms,
		          timeout_ms, retry_count, access_level, description, sort_order, status,
		          NULL::uuid, NULL::text, NULL::text, last_value, quality, last_updated_at,
		          created_at, updated_at
	`, args...)

	record, err := scanModbusRegisterRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Modbus 变量不存在")
		}
		return nil, translateModbusModelingWriteError(err)
	}
	return &record, nil
}

// DeleteRegister 删除 Modbus 变量定义。
func (r *ModbusModelingRepository) DeleteRegister(ctx context.Context, projectID, connectionID, registerID string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM data_modbus_registers
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
	`, projectID, connectionID, registerID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 Modbus 变量失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Modbus 变量不存在")
	}
	return nil
}

// DeleteRegistersBatch 批量删除变量，并在同一事务中标记数据点失效。
func (r *ModbusModelingRepository) DeleteRegistersBatch(ctx context.Context, projectID, connectionID string, ids []string, userID string) ([]string, error) {
	ids = uniqueTrimmedModbusIDs(ids)
	if len(ids) == 0 {
		return []string{}, nil
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 Modbus 批量删除事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)

	rows, err := tx.Query(ctx, `
		DELETE FROM data_modbus_registers
		WHERE project_id = $1 AND connection_id = $2 AND id = ANY($3::uuid[])
		RETURNING id
	`, projectID, connectionID, ids)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "批量删除 Modbus 变量失败", err)
	}
	deleted := make([]string, 0, len(ids))
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			rows.Close()
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 Modbus 批量删除结果失败", scanErr)
		}
		deleted = append(deleted, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 Modbus 批量删除结果失败", err)
	}
	rows.Close()
	if len(deleted) > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE data_points
			SET status = 'invalid',
			    updated_by = COALESCE($3, updated_by),
			    updated_at = now()
			WHERE project_id = $1
			  AND source_type = 'modbus.register'
			  AND source_id = ANY($2::uuid[])
		`, projectID, deleted, userID); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "批量标记 Modbus 数据点失效失败", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 Modbus 批量删除事务失败", err)
	}
	return deleted, nil
}

// UpdateRegistersBatch 批量更新变量公共属性。
func (r *ModbusModelingRepository) UpdateRegistersBatch(ctx context.Context, params BatchUpdateModbusRegistersParams) ([]ModbusRegisterRecord, error) {
	ids := uniqueTrimmedModbusIDs(params.IDs)
	if params.UseFilter {
		matched, err := r.ListRegisterIDsByFilter(ctx, params.ProjectID, params.ConnectionID, params.Filter)
		if err != nil {
			return nil, err
		}
		ids = matched
	}
	if len(ids) == 0 {
		return []ModbusRegisterRecord{}, nil
	}
	setParts := []string{"updated_by = $4", "updated_at = now()"}
	args := []any{params.ProjectID, params.ConnectionID, ids, params.UserID}
	if params.HasGroupID {
		args = append(args, params.GroupID)
		setParts = append(setParts, fmt.Sprintf("group_id = $%d", len(args)))
	}
	if params.UnitID != nil {
		args = append(args, *params.UnitID)
		setParts = append(setParts, fmt.Sprintf("unit_id = $%d", len(args)))
	}
	if params.PollIntervalMS != nil {
		args = append(args, *params.PollIntervalMS)
		setParts = append(setParts, fmt.Sprintf("poll_interval_ms = $%d", len(args)))
	}
	if strings.TrimSpace(params.ByteOrder) != "" {
		args = append(args, strings.TrimSpace(params.ByteOrder))
		setParts = append(setParts, fmt.Sprintf("byte_order = $%d", len(args)))
	}
	if strings.TrimSpace(params.WordOrder) != "" {
		args = append(args, strings.TrimSpace(params.WordOrder))
		setParts = append(setParts, fmt.Sprintf("word_order = $%d", len(args)))
	}
	if strings.TrimSpace(params.Status) != "" {
		args = append(args, strings.TrimSpace(params.Status))
		setParts = append(setParts, fmt.Sprintf("status = $%d", len(args)))
	}

	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		UPDATE data_modbus_registers
		SET %s
		WHERE project_id = $1 AND connection_id = $2 AND id = ANY($3::uuid[])
		RETURNING id, project_id, connection_id, group_id, name, code, unit_id, area,
		          address, address_base, protocol_address, quantity, data_type, byte_order,
		          word_order, bit_index, scale, offset_value, unit, poll_interval_ms,
		          timeout_ms, retry_count, access_level, description, sort_order, status,
		          NULL::uuid, NULL::text, NULL::text, last_value, quality, last_updated_at,
		          created_at, updated_at
	`, strings.Join(setParts, ",\n\t\t\t")), args...)
	if err != nil {
		return nil, translateModbusModelingWriteError(err)
	}
	defer rows.Close()

	result := make([]ModbusRegisterRecord, 0, len(ids))
	for rows.Next() {
		record, scanErr := scanModbusRegisterRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 Modbus 批量更新结果失败", err)
	}
	return result, nil
}

// UpdateRegisterLastValue 写回开发态读取得到的最近值快照。
func (r *ModbusModelingRepository) UpdateRegisterLastValue(ctx context.Context, params UpdateModbusRegisterLastValueParams) error {
	valueBytes, err := marshalModbusJSONValue(params.LastValue)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE data_modbus_registers
		SET last_value = $4::jsonb,
		    quality = $5,
		    last_updated_at = now(),
		    updated_by = $6,
		    updated_at = now()
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
	`, params.ProjectID, params.ConnectionID, params.RegisterID, string(valueBytes), params.Quality, params.UserID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新 Modbus 变量最近值失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Modbus 变量不存在")
	}
	return nil
}

func scanModbusRegisterGroupRecord(row pgx.Row) (ModbusRegisterGroupRecord, error) {
	record := ModbusRegisterGroupRecord{}
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.ParentID,
		&record.Name,
		&record.Description,
		&record.SortOrder,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return record, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Modbus 寄存器组不存在")
		}
		return record, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 Modbus 寄存器组失败", err)
	}
	return record, nil
}

func scanModbusSlaveDeviceRecord(row pgx.Row) (ModbusSlaveDeviceRecord, error) {
	record := ModbusSlaveDeviceRecord{}
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.UnitID,
		&record.Name,
		&record.Description,
		&record.Enabled,
		&record.DefaultPollIntervalMS,
		&record.DefaultByteOrder,
		&record.DefaultWordOrder,
		&record.RequestIntervalMS,
		&record.TimeoutMS,
		&record.RetryCount,
		&record.SortOrder,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return record, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Modbus 从站不存在")
		}
		return record, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 Modbus 从站失败", err)
	}
	return record, nil
}

func scanModbusRegisterRecord(row pgx.Row) (ModbusRegisterRecord, error) {
	record := ModbusRegisterRecord{}
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.GroupID,
		&record.Name,
		&record.Code,
		&record.UnitID,
		&record.Area,
		&record.Address,
		&record.AddressBase,
		&record.ProtocolAddress,
		&record.Quantity,
		&record.DataType,
		&record.ByteOrder,
		&record.WordOrder,
		&record.BitIndex,
		&record.Scale,
		&record.Offset,
		&record.Unit,
		&record.PollIntervalMS,
		&record.TimeoutMS,
		&record.RetryCount,
		&record.AccessLevel,
		&record.Description,
		&record.SortOrder,
		&record.Status,
		&record.DataPointID,
		&record.DataPointPath,
		&record.DataPointStatus,
		&record.LastValue,
		&record.Quality,
		&record.LastUpdatedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return record, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Modbus 变量不存在")
		}
		return record, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 Modbus 变量失败", err)
	}
	return record, nil
}

func uniqueTrimmedModbusIDs(ids []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func marshalModbusJSONValue(value any) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Modbus 最近值不是有效 JSON", err)
	}
	return payload, nil
}

func translateModbusModelingWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "Modbus 建模对象已存在")
		case "23503":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Modbus 建模引用不存在")
		case "23514":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Modbus 建模参数不合法")
		}
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 Modbus 建模数据失败", err)
}
