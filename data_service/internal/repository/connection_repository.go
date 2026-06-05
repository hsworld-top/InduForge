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

// ConnectionRecord 表示 data_connections 表在仓储层的投影结果。
type ConnectionRecord struct {
	ID           string
	ProjectID    string
	Name         string
	Type         string
	Category     string `json:"-"`
	Status       string
	Config       map[string]any
	DisplayOrder int
	CreatedAt    time.Time
	UpdatedAt    time.Time
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

// UpdateKafkaConnectionParams 描述 Kafka 接入源编辑时需要同步的连接与专用配置表字段。
type UpdateKafkaConnectionParams struct {
	UpdateConnectionParams
	Brokers string
	Options map[string]any
}

// ConnectionRepository 封装 data_connections 的参数化 SQL 访问。
type ConnectionRepository struct {
	pool *pgxpool.Pool
}

type connectionOrderExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// NewConnectionRepository 创建连接仓储。
func NewConnectionRepository(pool *pgxpool.Pool) *ConnectionRepository {
	return &ConnectionRepository{pool: pool}
}

func lockProjectConnectionOrder(ctx context.Context, tx pgx.Tx, projectID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, projectID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "锁定连接排序失败", err)
	}
	return nil
}

func normalizeProjectConnectionDisplayOrder(ctx context.Context, executor connectionOrderExecutor, projectID string) error {
	_, err := executor.Exec(ctx, `
        WITH ordered AS (
            SELECT
                id,
                row_number() OVER (ORDER BY display_order ASC, created_at ASC, id ASC) - 1 AS next_order
            FROM data_connections
            WHERE project_id = $1
        )
        UPDATE data_connections AS conn
        SET display_order = ordered.next_order
        FROM ordered
        WHERE conn.id = ordered.id
          AND conn.display_order <> ordered.next_order
    `, projectID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "归一化连接排序失败", err)
	}
	return nil
}

func translateConnectionWriteError(message string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if pgErr.ConstraintName == "data_connections_project_name_key" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "接入源名称已存在")
		}
	}
	return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, message, err)
}

// ListByProject 按项目读取连接列表。
// 查询路径固定为 project_id；display_order 是用户在接入源面板拖拽后的持久化顺序。
func (r *ConnectionRepository) ListByProject(ctx context.Context, projectID string) ([]ConnectionRecord, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, project_id, name, type, category, status, metadata, display_order, created_at, updated_at
        FROM data_connections
        WHERE project_id = $1
        ORDER BY display_order ASC, created_at ASC
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
        SELECT id, project_id, name, type, category, status, metadata, display_order, created_at, updated_at
        FROM data_connections
        WHERE project_id = $1 AND id = $2
    `, projectID, connectionID)

	record, err := scanConnection(row)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

// GetByProjectAndType 按项目和连接类型读取一条记录。
// 用于校验工程级内置运行库唯一性；调用方只关心是否存在，不依赖排序。
func (r *ConnectionRepository) GetByProjectAndType(ctx context.Context, projectID, connectionType string) (*ConnectionRecord, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, project_id, name, type, category, status, metadata, display_order, created_at, updated_at
        FROM data_connections
        WHERE project_id = $1 AND type = $2
        LIMIT 1
    `, projectID, connectionType)

	record, err := scanConnection(row)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

// Create 写入一条新的连接记录。
// 新连接追加到当前项目排序末尾，避免用户已拖拽好的接入源顺序被新建动作打乱。
func (r *ConnectionRepository) Create(ctx context.Context, params CreateConnectionParams) (*ConnectionRecord, error) {
	configBytes, err := marshalConfig(params.Config)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启连接创建事务失败", err)
	}
	defer tx.Rollback(ctx)

	if err := lockProjectConnectionOrder(ctx, tx, params.ProjectID); err != nil {
		return nil, err
	}
	if err := normalizeProjectConnectionDisplayOrder(ctx, tx, params.ProjectID); err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, `
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
        VALUES ($1, $2, $3, $4, $5, $6::jsonb,
            COALESCE((SELECT MAX(display_order) + 1 FROM data_connections WHERE project_id = $1), 0),
            $7, $7)
        RETURNING id, project_id, name, type, category, status, metadata, display_order, created_at, updated_at
    `, params.ProjectID, params.Name, params.Type, params.Category, params.Status, string(configBytes), params.UserID)

	record, scanErr := scanConnection(row)
	if scanErr != nil {
		return nil, translateConnectionWriteError("写入连接失败", scanErr)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交连接创建事务失败", err)
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
        RETURNING id, project_id, name, type, category, status, metadata, display_order, created_at, updated_at
    `, params.ProjectID, params.ID, params.Name, params.Type, params.Category, params.Status, string(configBytes), params.UserID)

	record, scanErr := scanConnection(row)
	if scanErr != nil {
		return nil, translateConnectionWriteError("更新连接失败", scanErr)
	}

	return &record, nil
}

// UpdateKafka 更新 Kafka 接入源并同步 data_kafka_configs。
// 输入来自通用连接编辑接口；输出为更新后的 data_connections 记录。
// 失败时事务回滚，避免 metadata 与专用 Kafka 配置表不一致。
func (r *ConnectionRepository) UpdateKafka(ctx context.Context, params UpdateKafkaConnectionParams) (*ConnectionRecord, error) {
	configBytes, err := marshalConfig(params.Config)
	if err != nil {
		return nil, err
	}
	optionsBytes, err := marshalConfig(params.Options)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 Kafka 连接更新事务失败", err)
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
        UPDATE data_connections
        SET name = $3,
            type = $4,
            category = $5,
            status = $6,
            metadata = $7::jsonb,
            updated_by = $8,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
        RETURNING id, project_id, name, type, category, status, metadata, display_order, created_at, updated_at
    `, params.ProjectID, params.ID, params.Name, params.Type, params.Category, params.Status, string(configBytes), params.UserID)

	record, scanErr := scanConnection(row)
	if scanErr != nil {
		return nil, translateConnectionWriteError("更新 Kafka 连接失败", scanErr)
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO data_kafka_configs (
            connection_id,
            brokers,
            topic,
            consumer_group,
            start_position,
            options
        )
        VALUES ($1, $2, NULL, NULL, 'latest', $3::jsonb)
        ON CONFLICT (connection_id) DO UPDATE
        SET brokers = EXCLUDED.brokers,
            options = EXCLUDED.options,
            updated_at = now()
    `, params.ID, params.Brokers, string(optionsBytes))
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "同步 Kafka 配置失败", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 Kafka 连接更新事务失败", err)
	}

	return &record, nil
}

// UpdateDisplayOrder 批量更新项目下连接展示顺序。
// 输入必须覆盖调用方传入的每个 id，仓储层逐条带 project_id 更新，避免跨项目改动。
func (r *ConnectionRepository) UpdateDisplayOrder(ctx context.Context, projectID string, connectionIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启连接排序事务失败", err)
	}
	defer tx.Rollback(ctx)

	if err := lockProjectConnectionOrder(ctx, tx, projectID); err != nil {
		return err
	}

	for index, connectionID := range connectionIDs {
		commandTag, execErr := tx.Exec(ctx, `
            UPDATE data_connections
            SET display_order = $3,
                updated_at = now()
            WHERE project_id = $1 AND id = $2
        `, projectID, connectionID, index)
		if execErr != nil {
			return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新连接排序失败", execErr)
		}
		if commandTag.RowsAffected() == 0 {
			return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "连接不存在")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交连接排序失败", err)
	}
	return nil
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
		&record.Category,
		&record.Status,
		&configBytes,
		&record.DisplayOrder,
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
