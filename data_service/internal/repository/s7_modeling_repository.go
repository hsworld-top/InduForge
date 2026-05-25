package repository

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

// S7ProfileRecord 表示 S7 PLC 档案，运行态按该档案确定连接与读取边界。
type S7ProfileRecord struct {
	ID                   string
	ProjectID            string
	ConnectionID         string
	PlcFamily            string
	CommunicationMode    string
	Host                 string
	Port                 int
	Rack                 int
	Slot                 int
	LocalTSAP            *string
	RemoteTSAP           *string
	PollIntervalMS       int
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
	SupportedAreas       []byte
	Options              []byte
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// S7VariableGroupRecord 表示 S7 变量组。
type S7VariableGroupRecord struct {
	ID           string
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	Code         string
	Description  *string
	SortOrder    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// S7VariableRecord 表示 S7 变量地址建模结果。
type S7VariableRecord struct {
	ID                string
	ProjectID         string
	ConnectionID      string
	GroupID           *string
	Name              string
	Code              string
	Description       *string
	Area              string
	DBNumber          *int
	ByteOffset        int
	BitOffset         *int
	AddressText       string
	NormalizedAddress string
	AddressType       string
	ReadLength        int
	DataType          string
	Length            *int
	ArrayLength       *int
	ByteOrder         string
	WordOrder         string
	Scale             float64
	Offset            float64
	Unit              *string
	PollIntervalMS    int
	QualityRule       []byte
	Metadata          []byte
	LastValue         []byte
	Quality           string
	LastUpdatedAt     *time.Time
	SortOrder         int
	Status            string
	DataPointID       *string
	DataPointPath     *string
	DataPointStatus   *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// UpsertS7ProfileParams 描述 PLC 档案写入参数。
type UpsertS7ProfileParams struct {
	ProjectID            string
	ConnectionID         string
	PlcFamily            string
	CommunicationMode    string
	Host                 string
	Port                 int
	Rack                 int
	Slot                 int
	LocalTSAP            *string
	RemoteTSAP           *string
	PollIntervalMS       int
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
	UserID               string
}

// CreateS7VariableGroupParams 描述创建变量组参数。
type CreateS7VariableGroupParams struct {
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	Code         string
	Description  *string
	SortOrder    int
	UserID       string
}

// UpdateS7VariableGroupParams 描述更新变量组参数。
type UpdateS7VariableGroupParams struct {
	ID           string
	ProjectID    string
	ConnectionID string
	ParentID     *string
	HasParentID  bool
	Name         string
	Code         string
	Description  *string
	SortOrder    int
	UserID       string
}

// CreateS7VariableParams 描述创建 S7 变量参数。
type CreateS7VariableParams struct {
	ProjectID         string
	ConnectionID      string
	GroupID           *string
	Name              string
	Code              string
	Description       *string
	Area              string
	DBNumber          *int
	ByteOffset        int
	BitOffset         *int
	AddressText       string
	NormalizedAddress string
	AddressType       string
	ReadLength        int
	DataType          string
	Length            *int
	ArrayLength       *int
	ByteOrder         string
	WordOrder         string
	Scale             float64
	Offset            float64
	Unit              *string
	PollIntervalMS    int
	QualityRule       map[string]any
	Metadata          map[string]any
	SortOrder         int
	UserID            string
}

// UpdateS7VariableParams 描述更新 S7 变量参数。
type UpdateS7VariableParams struct {
	ID                string
	ProjectID         string
	ConnectionID      string
	GroupID           *string
	HasGroupID        bool
	Name              string
	Code              string
	Description       *string
	Area              string
	DBNumber          *int
	ByteOffset        int
	BitOffset         *int
	AddressText       string
	NormalizedAddress string
	AddressType       string
	ReadLength        int
	DataType          string
	Length            *int
	ArrayLength       *int
	ByteOrder         string
	WordOrder         string
	Scale             float64
	Offset            float64
	Unit              *string
	PollIntervalMS    int
	QualityRule       map[string]any
	Metadata          map[string]any
	SortOrder         int
	Status            string
	UserID            string
}

// UpdateS7VariableLastValueParams 描述开发态读取后写回的最近值快照。
type UpdateS7VariableLastValueParams struct {
	ProjectID    string
	ConnectionID string
	VariableID   string
	LastValue    any
	Quality      string
	UserID       string
}

// S7ModelingRepository 封装 S7 建模参数化 SQL。
type S7ModelingRepository struct {
	pool *pgxpool.Pool
}

// NewS7ModelingRepository 创建 S7 建模仓储。
func NewS7ModelingRepository(pool *pgxpool.Pool) *S7ModelingRepository {
	return &S7ModelingRepository{pool: pool}
}

// GetProfile 返回连接下的 PLC 档案；未配置时返回 nil，便于 service 合成默认档案。
func (r *S7ModelingRepository) GetProfile(ctx context.Context, projectID, connectionID string) (*S7ProfileRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, connection_id, plc_family, communication_mode, host, port,
		       rack, slot, local_tsap, remote_tsap, poll_interval_ms, connect_timeout_ms,
		       read_timeout_ms, pdu_size, max_read_bytes, max_gap_bytes, max_concurrent_reads,
		       byte_order, word_order, optimized_block_access, allow_absolute_address,
		       allow_symbol_address, supported_areas, options, created_at, updated_at
		FROM data_s7_plc_profiles
		WHERE project_id = $1 AND connection_id = $2
	`, projectID, connectionID)

	record, err := scanS7ProfileRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// UpsertProfile 创建或更新 PLC 档案。
func (r *S7ModelingRepository) UpsertProfile(ctx context.Context, params UpsertS7ProfileParams) (*S7ProfileRecord, error) {
	supportedAreasBytes, err := marshalJSONArray(params.SupportedAreas)
	if err != nil {
		return nil, err
	}
	optionsBytes, err := marshalJSONObject(params.Options)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, `
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
		RETURNING id, project_id, connection_id, plc_family, communication_mode, host, port,
		          rack, slot, local_tsap, remote_tsap, poll_interval_ms, connect_timeout_ms,
		          read_timeout_ms, pdu_size, max_read_bytes, max_gap_bytes, max_concurrent_reads,
		          byte_order, word_order, optimized_block_access, allow_absolute_address,
		          allow_symbol_address, supported_areas, options, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.PlcFamily, params.CommunicationMode, params.Host, params.Port,
		params.Rack, params.Slot, params.LocalTSAP, params.RemoteTSAP, params.PollIntervalMS, params.ConnectTimeoutMS,
		params.ReadTimeoutMS, params.PDUSize, params.MaxReadBytes, params.MaxGapBytes, params.MaxConcurrentReads,
		params.ByteOrder, params.WordOrder, params.OptimizedBlockAccess, params.AllowAbsoluteAddress,
		params.AllowSymbolAddress, string(supportedAreasBytes), string(optionsBytes), params.UserID)

	record, scanErr := scanS7ProfileRecord(row)
	if scanErr != nil {
		return nil, translateS7ModelingWriteError(scanErr)
	}
	return &record, nil
}

// ListGroups 返回连接下的变量组。
func (r *S7ModelingRepository) ListGroups(ctx context.Context, projectID, connectionID string) ([]S7VariableGroupRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, connection_id, parent_id, name, code, description, sort_order, created_at, updated_at
		FROM data_s7_variable_groups
		WHERE project_id = $1 AND connection_id = $2
		ORDER BY sort_order ASC, created_at ASC
	`, projectID, connectionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 S7 变量组失败", err)
	}
	defer rows.Close()

	result := make([]S7VariableGroupRecord, 0)
	for rows.Next() {
		record, scanErr := scanS7VariableGroupRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 S7 变量组失败", err)
	}
	return result, nil
}

// CreateGroup 创建变量组。
func (r *S7ModelingRepository) CreateGroup(ctx context.Context, params CreateS7VariableGroupParams) (*S7VariableGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO data_s7_variable_groups (
			project_id, connection_id, parent_id, name, code, description, sort_order, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		RETURNING id, project_id, connection_id, parent_id, name, code, description, sort_order, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.ParentID, params.Name, params.Code, params.Description, params.SortOrder, params.UserID)

	record, err := scanS7VariableGroupRecord(row)
	if err != nil {
		return nil, translateS7ModelingWriteError(err)
	}
	return &record, nil
}

// UpdateGroup 更新变量组。
func (r *S7ModelingRepository) UpdateGroup(ctx context.Context, params UpdateS7VariableGroupParams) (*S7VariableGroupRecord, error) {
	parentSQL := "parent_id"
	args := []any{params.ProjectID, params.ConnectionID, params.ID, params.Name, params.Code, params.Description, params.SortOrder, params.UserID}
	if params.HasParentID {
		parentSQL = "$9"
		args = append(args, params.ParentID)
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE data_s7_variable_groups
		SET name = $4,
		    code = $5,
		    description = $6,
		    sort_order = $7,
		    updated_by = $8,
		    updated_at = now(),
		    parent_id = `+parentSQL+`
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
		RETURNING id, project_id, connection_id, parent_id, name, code, description, sort_order, created_at, updated_at
	`, args...)

	record, err := scanS7VariableGroupRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "S7 变量组不存在")
		}
		return nil, translateS7ModelingWriteError(err)
	}
	return &record, nil
}

// DeleteGroup 删除变量组，组内变量移动到未分组。
func (r *S7ModelingRepository) DeleteGroup(ctx context.Context, projectID, connectionID, groupID, userID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 S7 变量组删除事务失败", err)
	}
	defer rollbackTxQuietly(ctx, tx)

	if _, err := tx.Exec(ctx, `
		UPDATE data_s7_variables
		SET group_id = NULL, updated_by = $4, updated_at = now()
		WHERE project_id = $1 AND connection_id = $2 AND group_id = $3
	`, projectID, connectionID, groupID, userID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "移动 S7 变量失败", err)
	}

	tag, err := tx.Exec(ctx, `
		DELETE FROM data_s7_variable_groups
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
	`, projectID, connectionID, groupID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 S7 变量组失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "S7 变量组不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 S7 变量组删除事务失败", err)
	}
	return nil
}

// ListVariables 返回连接下的 S7 变量，可按 groupID 过滤。
func (r *S7ModelingRepository) ListVariables(ctx context.Context, projectID, connectionID string, groupID *string) ([]S7VariableRecord, error) {
	where := "v.project_id = $1 AND v.connection_id = $2"
	args := []any{projectID, connectionID}
	if groupID != nil {
		where += " AND v.group_id = $3"
		args = append(args, groupID)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT v.id, v.project_id, v.connection_id, v.group_id, v.name, v.code, v.description,
		       v.area, v.db_number, v.byte_offset, v.bit_offset, v.address_text, v.normalized_address,
		       v.address_type, v.read_length, v.data_type, v.length, v.array_length, v.byte_order,
		       v.word_order, v.scale, v.offset_value, v.unit, v.poll_interval_ms, v.quality_rule,
		       v.metadata, v.last_value, v.quality, v.last_updated_at, v.sort_order, v.status,
		       dp.id, dp.path, dp.status, v.created_at, v.updated_at
		FROM data_s7_variables v
		LEFT JOIN data_points dp
		  ON dp.project_id = v.project_id
		 AND dp.source_type = 's7.variable'
		 AND dp.source_id = v.id
		WHERE `+where+`
		ORDER BY v.sort_order ASC, v.created_at ASC
	`, args...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 S7 变量失败", err)
	}
	defer rows.Close()

	result := make([]S7VariableRecord, 0)
	for rows.Next() {
		record, scanErr := scanS7VariableRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 S7 变量失败", err)
	}
	return result, nil
}

// GetVariable 按连接和变量 ID 读取 S7 变量。
func (r *S7ModelingRepository) GetVariable(ctx context.Context, projectID, connectionID, variableID string) (*S7VariableRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT v.id, v.project_id, v.connection_id, v.group_id, v.name, v.code, v.description,
		       v.area, v.db_number, v.byte_offset, v.bit_offset, v.address_text, v.normalized_address,
		       v.address_type, v.read_length, v.data_type, v.length, v.array_length, v.byte_order,
		       v.word_order, v.scale, v.offset_value, v.unit, v.poll_interval_ms, v.quality_rule,
		       v.metadata, v.last_value, v.quality, v.last_updated_at, v.sort_order, v.status,
		       dp.id, dp.path, dp.status, v.created_at, v.updated_at
		FROM data_s7_variables v
		LEFT JOIN data_points dp
		  ON dp.project_id = v.project_id
		 AND dp.source_type = 's7.variable'
		 AND dp.source_id = v.id
		WHERE v.project_id = $1 AND v.connection_id = $2 AND v.id = $3
	`, projectID, connectionID, variableID)

	record, err := scanS7VariableRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "S7 变量不存在")
		}
		return nil, err
	}
	return &record, nil
}

// CreateVariable 创建 S7 变量。
func (r *S7ModelingRepository) CreateVariable(ctx context.Context, params CreateS7VariableParams) (*S7VariableRecord, error) {
	qualityRuleBytes, err := marshalJSONObject(params.QualityRule)
	if err != nil {
		return nil, err
	}
	metadataBytes, err := marshalJSONObject(params.Metadata)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO data_s7_variables (
			project_id, connection_id, group_id, name, code, description, area, db_number,
			byte_offset, bit_offset, address_text, normalized_address, address_type, read_length,
			data_type, length, array_length, byte_order, word_order, scale, offset_value, unit,
			poll_interval_ms, quality_rule, metadata, sort_order, created_by, updated_by
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23, $24::jsonb, $25::jsonb, $26, $27, $27
		)
		RETURNING id, project_id, connection_id, group_id, name, code, description, area,
		          db_number, byte_offset, bit_offset, address_text, normalized_address,
		          address_type, read_length, data_type, length, array_length, byte_order,
		          word_order, scale, offset_value, unit, poll_interval_ms, quality_rule,
		          metadata, last_value, quality, last_updated_at, sort_order, status,
		          NULL::uuid, NULL::text, NULL::text, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.GroupID, params.Name, params.Code, params.Description, params.Area,
		params.DBNumber, params.ByteOffset, params.BitOffset, params.AddressText, params.NormalizedAddress,
		params.AddressType, params.ReadLength, params.DataType, params.Length, params.ArrayLength, params.ByteOrder,
		params.WordOrder, params.Scale, params.Offset, params.Unit, params.PollIntervalMS, string(qualityRuleBytes),
		string(metadataBytes), params.SortOrder, params.UserID)

	record, scanErr := scanS7VariableRecord(row)
	if scanErr != nil {
		return nil, translateS7ModelingWriteError(scanErr)
	}
	return &record, nil
}

// UpdateVariable 更新 S7 变量。
func (r *S7ModelingRepository) UpdateVariable(ctx context.Context, params UpdateS7VariableParams) (*S7VariableRecord, error) {
	qualityRuleBytes, err := marshalJSONObject(params.QualityRule)
	if err != nil {
		return nil, err
	}
	metadataBytes, err := marshalJSONObject(params.Metadata)
	if err != nil {
		return nil, err
	}
	groupSQL := "group_id"
	args := []any{
		params.ProjectID, params.ConnectionID, params.ID, params.Name, params.Code, params.Description,
		params.Area, params.DBNumber, params.ByteOffset, params.BitOffset, params.AddressText,
		params.NormalizedAddress, params.AddressType, params.ReadLength, params.DataType, params.Length,
		params.ArrayLength, params.ByteOrder, params.WordOrder, params.Scale, params.Offset, params.Unit,
		params.PollIntervalMS, string(qualityRuleBytes), string(metadataBytes), params.SortOrder, params.Status,
		params.UserID,
	}
	if params.HasGroupID {
		groupSQL = "$29"
		args = append(args, params.GroupID)
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE data_s7_variables
		SET name = $4,
		    code = $5,
		    description = $6,
		    area = $7,
		    db_number = $8,
		    byte_offset = $9,
		    bit_offset = $10,
		    address_text = $11,
		    normalized_address = $12,
		    address_type = $13,
		    read_length = $14,
		    data_type = $15,
		    length = $16,
		    array_length = $17,
		    byte_order = $18,
		    word_order = $19,
		    scale = $20,
		    offset_value = $21,
		    unit = $22,
		    poll_interval_ms = $23,
		    quality_rule = $24::jsonb,
		    metadata = $25::jsonb,
		    sort_order = $26,
		    status = $27,
		    updated_by = $28,
		    updated_at = now(),
		    group_id = `+groupSQL+`
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
		RETURNING id, project_id, connection_id, group_id, name, code, description, area,
		          db_number, byte_offset, bit_offset, address_text, normalized_address,
		          address_type, read_length, data_type, length, array_length, byte_order,
		          word_order, scale, offset_value, unit, poll_interval_ms, quality_rule,
		          metadata, last_value, quality, last_updated_at, sort_order, status,
		          NULL::uuid, NULL::text, NULL::text, created_at, updated_at
	`, args...)

	record, scanErr := scanS7VariableRecord(row)
	if scanErr != nil {
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "S7 变量不存在")
		}
		return nil, translateS7ModelingWriteError(scanErr)
	}
	return &record, nil
}

// DeleteVariable 删除 S7 变量定义。
func (r *S7ModelingRepository) DeleteVariable(ctx context.Context, projectID, connectionID, variableID string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM data_s7_variables
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
	`, projectID, connectionID, variableID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 S7 变量失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "S7 变量不存在")
	}
	return nil
}

// UpdateVariableLastValue 写回开发态读取得到的最近值快照。
func (r *S7ModelingRepository) UpdateVariableLastValue(ctx context.Context, params UpdateS7VariableLastValueParams) error {
	valueBytes, err := marshalS7JSONValue(params.LastValue)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE data_s7_variables
		SET last_value = $4::jsonb,
		    quality = $5,
		    last_updated_at = now(),
		    updated_by = $6,
		    updated_at = now()
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
	`, params.ProjectID, params.ConnectionID, params.VariableID, string(valueBytes), params.Quality, params.UserID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新 S7 变量最近值失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "S7 变量不存在")
	}
	return nil
}

func scanS7ProfileRecord(row pgx.Row) (S7ProfileRecord, error) {
	record := S7ProfileRecord{}
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.PlcFamily,
		&record.CommunicationMode,
		&record.Host,
		&record.Port,
		&record.Rack,
		&record.Slot,
		&record.LocalTSAP,
		&record.RemoteTSAP,
		&record.PollIntervalMS,
		&record.ConnectTimeoutMS,
		&record.ReadTimeoutMS,
		&record.PDUSize,
		&record.MaxReadBytes,
		&record.MaxGapBytes,
		&record.MaxConcurrentReads,
		&record.ByteOrder,
		&record.WordOrder,
		&record.OptimizedBlockAccess,
		&record.AllowAbsoluteAddress,
		&record.AllowSymbolAddress,
		&record.SupportedAreas,
		&record.Options,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return record, err
	}
	return record, nil
}

func scanS7VariableGroupRecord(row pgx.Row) (S7VariableGroupRecord, error) {
	record := S7VariableGroupRecord{}
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.ParentID,
		&record.Name,
		&record.Code,
		&record.Description,
		&record.SortOrder,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return record, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "S7 变量组不存在")
		}
		return record, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 S7 变量组失败", err)
	}
	return record, nil
}

func scanS7VariableRecord(row pgx.Row) (S7VariableRecord, error) {
	record := S7VariableRecord{}
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.GroupID,
		&record.Name,
		&record.Code,
		&record.Description,
		&record.Area,
		&record.DBNumber,
		&record.ByteOffset,
		&record.BitOffset,
		&record.AddressText,
		&record.NormalizedAddress,
		&record.AddressType,
		&record.ReadLength,
		&record.DataType,
		&record.Length,
		&record.ArrayLength,
		&record.ByteOrder,
		&record.WordOrder,
		&record.Scale,
		&record.Offset,
		&record.Unit,
		&record.PollIntervalMS,
		&record.QualityRule,
		&record.Metadata,
		&record.LastValue,
		&record.Quality,
		&record.LastUpdatedAt,
		&record.SortOrder,
		&record.Status,
		&record.DataPointID,
		&record.DataPointPath,
		&record.DataPointStatus,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return record, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "S7 变量不存在")
		}
		return record, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 S7 变量失败", err)
	}
	return record, nil
}

func translateS7ModelingWriteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "S7 建模对象已存在")
		case "23503":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "S7 建模引用不存在")
		case "23514":
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "S7 建模参数不合法")
		}
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 S7 建模数据失败", err)
}

func marshalS7JSONValue(value any) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "S7 最近值 JSON 格式无效", err)
	}
	return payload, nil
}
