package repository

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	CreatedAt       time.Time
	UpdatedAt       time.Time
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
		       r.created_at, r.updated_at
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

// ListRegistersPage 返回当前分组下的一页 Modbus 变量，并同时返回匹配总数。
func (r *ModbusModelingRepository) ListRegistersPage(ctx context.Context, projectID, connectionID string, groupID *string, page, pageSize int) ([]ModbusRegisterRecord, int, error) {
	where := "r.project_id = $1 AND r.connection_id = $2"
	args := []any{projectID, connectionID}
	if groupID != nil {
		where += " AND r.group_id = $3"
		args = append(args, groupID)
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM data_modbus_registers r WHERE "+where, args...).Scan(&total); err != nil {
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
		       r.created_at, r.updated_at
		FROM data_modbus_registers r
		LEFT JOIN data_points dp
		  ON dp.project_id = r.project_id
		 AND dp.source_type = 'modbus.register'
		 AND dp.source_id = r.id
		WHERE %s
		ORDER BY r.sort_order ASC, r.created_at ASC
		LIMIT $%d OFFSET $%d
	`, where, limitIndex, offsetIndex), queryArgs...)
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

// GetRegister 按项目和变量 ID 读取 Modbus 变量。
func (r *ModbusModelingRepository) GetRegister(ctx context.Context, projectID, registerID string) (*ModbusRegisterRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT r.id, r.project_id, r.connection_id, r.group_id, r.name, r.code,
		       r.unit_id, r.area, r.address, r.address_base, r.protocol_address, r.quantity,
		       r.data_type, r.byte_order, r.word_order, r.bit_index, r.scale, r.offset_value,
		       r.unit, r.poll_interval_ms, r.timeout_ms, r.retry_count, r.access_level,
		       r.description, r.sort_order, r.status, dp.id, dp.path, dp.status,
		       r.created_at, r.updated_at
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
		          NULL::uuid, NULL::text, NULL::text, created_at, updated_at
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
		          NULL::uuid, NULL::text, NULL::text, created_at, updated_at
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
