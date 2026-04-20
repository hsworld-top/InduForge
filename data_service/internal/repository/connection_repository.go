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

// ConnectionRecord 表示 data_connections 表在仓储层的投影结果。
type ConnectionRecord struct {
	ID        string
	ProjectID string
	Name      string
	Type      string
	Status    string
	Config    map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ConnectionStatusRecord 表示连接状态更新后的返回结构。
type ConnectionStatusRecord struct {
	ID        string
	ProjectID string
	Status    string
	UpdatedAt time.Time
}

// CreateConnectionParams 描述创建连接时需要落库的字段。
type CreateConnectionParams struct {
	ProjectID string
	UserID    string
	Name      string
	Type      string
	Category  string
	Status    string
	Config    map[string]any
}

// UpdateConnectionParams 描述更新连接时需要落库的字段。
type UpdateConnectionParams struct {
	ID        string
	ProjectID string
	UserID    string
	Name      string
	Type      string
	Category  string
	Status    string
	Config    map[string]any
}

// ConnectionRepository 封装 data_connections 的参数化 SQL 访问。
type ConnectionRepository struct {
	pool *pgxpool.Pool
}

// NewConnectionRepository 创建连接仓储。
func NewConnectionRepository(pool *pgxpool.Pool) *ConnectionRepository {
	return &ConnectionRepository{pool: pool}
}

// ListByProject 按项目读取连接列表。
// 查询路径固定为 project_id，对应 data_connections_project_type_idx / data_connections_project_status_idx 的前缀列。
// 当前额外按 created_at 排序，连接数量极大时可能触发排序开销，后续可视热点再补复合索引。
func (r *ConnectionRepository) ListByProject(ctx context.Context, projectID string) ([]ConnectionRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, name, type, status, metadata, created_at, updated_at
        FROM data_connections
        WHERE project_id = $1
        ORDER BY created_at DESC
    `, projectID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询连接列表失败", err)
	}
	defer rows.Close()

	records := make([]ConnectionRecord, 0)
	for rows.Next() {
		record, scanErr := scanConnection(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历连接列表失败", err)
	}

	return records, nil
}

// GetByProjectAndID 按项目和连接 ID 读取单条记录。
// 查询路径固定为主键 id 并补 project_id 保护边界，命中主键索引，额外项目条件用于避免跨项目误读。
// 如果后续频繁走 project_id + id 联合过滤，可再评估是否需要复合索引。
func (r *ConnectionRepository) GetByProjectAndID(ctx context.Context, projectID, connectionID string) (*ConnectionRecord, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, project_id, name, type, status, metadata, created_at, updated_at
        FROM data_connections
        WHERE project_id = $1 AND id = $2
    `, projectID, connectionID)

	record, err := scanConnection(row)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

// Create 写入一条新的连接记录。
// 写入时显式绑定 project_id/type/status，便于后续列表查询直接复用已有索引。
// 该语句依赖默认列补齐其余运行参数，若未来写入字段增多，需要同步评估 INSERT RETURNING 体积。
func (r *ConnectionRepository) Create(ctx context.Context, params CreateConnectionParams) (*ConnectionRecord, error) {
	configBytes, err := marshalConfig(params.Config)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, `
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
        VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $7)
        RETURNING id, project_id, name, type, status, metadata, created_at, updated_at
    `, params.ProjectID, params.Name, params.Type, params.Category, params.Status, string(configBytes), params.UserID)

	record, scanErr := scanConnection(row)
	if scanErr != nil {
		return nil, scanErr
	}

	return &record, nil
}

// Update 更新已有连接记录。
// 更新条件固定为 project_id + id，可避免跨项目误改；真正的行定位仍由主键 id 完成。
// 当前直接覆盖 metadata 全量 JSON，若后续出现大对象热更新场景，再考虑局部 jsonb_set。
func (r *ConnectionRepository) Update(ctx context.Context, params UpdateConnectionParams) (*ConnectionRecord, error) {
	configBytes, err := marshalConfig(params.Config)
	if err != nil {
		return nil, err
	}

	row := r.pool.QueryRow(ctx, `
        UPDATE data_connections
        SET name = $3,
            type = $4,
            category = $5,
            status = $6,
            metadata = $7::jsonb,
            updated_by = $8,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, name, type, status, metadata, created_at, updated_at
    `, params.ProjectID, params.ID, params.Name, params.Type, params.Category, params.Status, string(configBytes), params.UserID)

	record, scanErr := scanConnection(row)
	if scanErr != nil {
		return nil, scanErr
	}

	return &record, nil
}

// Delete 按项目删除连接。
// 删除条件同样带 project_id 边界，避免调用方误删其他项目数据。
// 当前为单条删除，性能风险极低；如果未来支持批量删除，需要重新评估 WHERE IN 的批量策略。
func (r *ConnectionRepository) Delete(ctx context.Context, projectID, connectionID string) error {
	commandTag, err := r.pool.Exec(ctx, `
        DELETE FROM data_connections
        WHERE project_id = $1 AND id = $2
    `, projectID, connectionID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除连接失败", err)
	}
	if commandTag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "连接不存在")
	}
	return nil
}

// UpdateStatus 按项目更新连接状态。
func (r *ConnectionRepository) UpdateStatus(ctx context.Context, projectID, connectionID, status string) (*ConnectionStatusRecord, error) {
	row := r.pool.QueryRow(ctx, `
        UPDATE data_connections
        SET status = $3,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, status, updated_at
    `, projectID, connectionID, status)

	record := ConnectionStatusRecord{}
	if err := row.Scan(&record.ID, &record.ProjectID, &record.Status, &record.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "连接不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新连接状态失败", err)
	}

	return &record, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanConnection(row scannable) (ConnectionRecord, error) {
	var (
		record      ConnectionRecord
		configBytes []byte
	)

	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.Name,
		&record.Type,
		&record.Status,
		&configBytes,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return ConnectionRecord{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "连接不存在")
		}
		return ConnectionRecord{}, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取连接数据失败", err)
	}

	config, err := unmarshalConfig(configBytes)
	if err != nil {
		return ConnectionRecord{}, err
	}
	record.Config = config

	return record, nil
}

func marshalConfig(config map[string]any) ([]byte, error) {
	if config == nil {
		config = map[string]any{}
	}

	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "连接配置格式无效", err)
	}

	return configBytes, nil
}

func unmarshalConfig(configBytes []byte) (map[string]any, error) {
	if len(configBytes) == 0 {
		return map[string]any{}, nil
	}

	var config map[string]any
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析连接配置失败", err)
	}
	if config == nil {
		config = map[string]any{}
	}

	return config, nil
}
