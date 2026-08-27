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
	"github.com/indu-forge/data_service/internal/security"
)

// ConnectionRecord 表示 data_connections 表在仓储层的投影结果。
type ConnectionRecord struct {
	ID                 string         `json:"id"`
	ProjectID          string         `json:"projectId"`
	Name               string         `json:"name"`
	Type               string         `json:"type"`
	Category           string         `json:"-"`
	IsEnabled          bool           `json:"enabled"`
	Config             map[string]any `json:"config"`
	DisplayOrder       int            `json:"displayOrder"`
	VariableCount      int            `json:"-"`
	LastTestStatus     string         `json:"-"`
	LastTestedAt       *time.Time     `json:"-"`
	LastTestDurationMS *int           `json:"-"`
	LastTestMessage    *string        `json:"-"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}

// ConnectionListFilter 表示接入源管理页的服务端分页与筛选条件。
type ConnectionListFilter struct {
	Page      int
	PageSize  int
	Search    string
	TypeGroup string
}

// CreateConnectionParams 描述创建连接时需要落库的字段。
type CreateConnectionParams struct {
	ProjectID       string
	UserID          string
	Name            string
	Type            string
	Category        string
	IsEnabled       *bool
	Config          map[string]any
	Secrets         map[string]string
	ClearSecretKeys []string
}

// UpdateConnectionParams 描述更新连接时需要落库的字段。
type UpdateConnectionParams struct {
	ID              string
	ProjectID       string
	UserID          string
	Name            string
	Type            string
	Category        string
	IsEnabled       *bool
	Config          map[string]any
	Secrets         map[string]string
	ClearSecretKeys []string
}

// UpdateKafkaConnectionParams 描述 Kafka 接入源编辑时需要同步的连接与专用配置表字段。
type UpdateKafkaConnectionParams struct {
	UpdateConnectionParams
	Brokers string
	Options map[string]any
}

// ConnectionRepository 封装 data_connections 的参数化 SQL 访问。
type ConnectionRepository struct {
	pool   *pgxpool.Pool
	cipher *security.ConnectionSecretCipher
}

type connectionOrderExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// NewConnectionRepository 创建连接仓储。
func NewConnectionRepository(pool *pgxpool.Pool) *ConnectionRepository {
	return &ConnectionRepository{pool: pool}
}

func (r *ConnectionRepository) SetSecretCipher(cipher *security.ConnectionSecretCipher) {
	r.cipher = cipher
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
		SELECT conn.id,
		       conn.project_id,
		       conn.name,
		       conn.type,
		       conn.category,
		       conn.is_enabled,
		       conn.metadata,
		       conn.display_order,
		       (SELECT COUNT(*)::int FROM data_points dp
		        WHERE dp.project_id = conn.project_id AND dp.status <> 'invalid'
		          AND (dp.source_id = conn.id OR dp.source_config->>'connectionId' = conn.id::text)) AS variable_count,
		       COALESCE((SELECT test.status FROM data_connection_test_records test WHERE test.connection_id=conn.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),'not_tested'),
		       (SELECT test.tested_at FROM data_connection_test_records test WHERE test.connection_id=conn.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),
		       (SELECT test.duration_ms FROM data_connection_test_records test WHERE test.connection_id=conn.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),
		       (SELECT test.message FROM data_connection_test_records test WHERE test.connection_id=conn.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),
		       conn.created_at,
		       conn.updated_at
		FROM data_connections conn
		WHERE conn.project_id = $1
		ORDER BY conn.display_order ASC, conn.created_at ASC
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

// ListByProjectPage 按项目、搜索条件和类型分组分页读取连接列表。
func (r *ConnectionRepository) ListByProjectPage(ctx context.Context, projectID string, filter ConnectionListFilter) ([]ConnectionRecord, int, error) {
	page, pageSize := normalizePageAndSize(filter.Page, filter.PageSize, 10, 100)
	conditions := []string{"conn.project_id = $1"}
	args := []any{projectID}

	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+search+"%")
		conditions = append(conditions, fmt.Sprintf("conn.name ILIKE $%d", len(args)))
	}

	switch strings.TrimSpace(filter.TypeGroup) {
	case "builtin":
		conditions = append(conditions, "conn.type LIKE 'builtin.%'")
	case "database":
		conditions = append(conditions, "conn.type IN ('relational', 'mysql', 'postgresql', 'sqlserver', 'tdengine', 'redis')")
	case "stream":
		conditions = append(conditions, "conn.type IN ('mqtt', 'kafka', 'websocket', 'http')")
	}

	whereClause := strings.Join(conditions, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM data_connections conn WHERE "+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "统计连接列表失败", err)
	}

	listArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	listQuery := fmt.Sprintf(`
		SELECT conn.id, conn.project_id, conn.name, conn.type, conn.category,
		       conn.is_enabled, conn.metadata, conn.display_order,
		       (SELECT COUNT(*)::int FROM data_points dp
		        WHERE dp.project_id = conn.project_id AND dp.status <> 'invalid'
		          AND (dp.source_id = conn.id OR dp.source_config->>'connectionId' = conn.id::text)) AS variable_count,
		       COALESCE((SELECT test.status FROM data_connection_test_records test WHERE test.connection_id=conn.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),'not_tested'),
		       (SELECT test.tested_at FROM data_connection_test_records test WHERE test.connection_id=conn.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),
		       (SELECT test.duration_ms FROM data_connection_test_records test WHERE test.connection_id=conn.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),
		       (SELECT test.message FROM data_connection_test_records test WHERE test.connection_id=conn.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),
		       conn.created_at, conn.updated_at
		FROM data_connections conn
		WHERE %s
		ORDER BY conn.display_order ASC, conn.created_at ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, len(args)+1, len(args)+2)

	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "分页查询连接列表失败", err)
	}
	defer rows.Close()

	records := make([]ConnectionRecord, 0, pageSize)
	for rows.Next() {
		record, scanErr := scanConnection(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历分页连接列表失败", err)
	}
	return records, total, nil
}

// GetByProjectAndID 按项目和连接 ID 读取单条记录。
// 查询路径固定为主键 id 并补 project_id 保护边界，命中主键索引，额外项目条件用于避免跨项目误读。
// 如果后续频繁走 project_id + id 联合过滤，可再评估是否需要复合索引。
func (r *ConnectionRepository) GetByProjectAndID(ctx context.Context, projectID, connectionID string) (*ConnectionRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, name, type, category, is_enabled, metadata, display_order,
               (SELECT COUNT(*)::int FROM data_points dp WHERE dp.project_id=data_connections.project_id AND dp.status <> 'invalid' AND (dp.source_id=data_connections.id OR dp.source_config->>'connectionId'=data_connections.id::text)),
               COALESCE((SELECT test.status FROM data_connection_test_records test WHERE test.connection_id=data_connections.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),'not_tested'),
               (SELECT test.tested_at FROM data_connection_test_records test WHERE test.connection_id=data_connections.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),
               (SELECT test.duration_ms FROM data_connection_test_records test WHERE test.connection_id=data_connections.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),
               (SELECT test.message FROM data_connection_test_records test WHERE test.connection_id=data_connections.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),
               created_at, updated_at
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
		SELECT id, project_id, name, type, category, is_enabled, metadata, display_order,
               (SELECT COUNT(*)::int FROM data_points dp WHERE dp.project_id=data_connections.project_id AND dp.status <> 'invalid' AND (dp.source_id=data_connections.id OR dp.source_config->>'connectionId'=data_connections.id::text)),
               COALESCE((SELECT test.status FROM data_connection_test_records test WHERE test.connection_id=data_connections.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),'not_tested'),
               (SELECT test.tested_at FROM data_connection_test_records test WHERE test.connection_id=data_connections.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),
               (SELECT test.duration_ms FROM data_connection_test_records test WHERE test.connection_id=data_connections.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),
               (SELECT test.message FROM data_connection_test_records test WHERE test.connection_id=data_connections.id ORDER BY test.tested_at DESC,test.id DESC LIMIT 1),
               created_at, updated_at
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
            is_enabled,
            metadata,
            display_order,
            created_by,
            updated_by
        )
        VALUES ($1, $2, $3, $4, $5, $6::jsonb,
            COALESCE((SELECT MAX(display_order) + 1 FROM data_connections WHERE project_id = $1), 0),
            $7, $7)
		RETURNING id, project_id, name, type, category, is_enabled, metadata, display_order, 0 AS variable_count,
                  'not_tested'::text,NULL::timestamptz,NULL::integer,NULL::text,created_at, updated_at
    `, params.ProjectID, params.Name, params.Type, params.Category, connectionEnabledValue(params.IsEnabled), string(configBytes), params.UserID)

	record, scanErr := scanConnection(row)
	if scanErr != nil {
		return nil, translateConnectionWriteError("写入连接失败", scanErr)
	}
	if err := applyPlainConnectionSecretsTx(ctx, tx, r.cipher, record.ID, params.Secrets, params.ClearSecretKeys); err != nil {
		return nil, err
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

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启连接更新事务失败", err)
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `
        UPDATE data_connections
        SET name = $3,
            type = $4,
            category = $5,
            is_enabled = $6,
            metadata = $7::jsonb,
            updated_by = $8,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
		RETURNING id, project_id, name, type, category, is_enabled, metadata, display_order, 0 AS variable_count,
                  'not_tested'::text,NULL::timestamptz,NULL::integer,NULL::text,created_at, updated_at
    `, params.ProjectID, params.ID, params.Name, params.Type, params.Category, connectionEnabledValue(params.IsEnabled), string(configBytes), params.UserID)

	record, scanErr := scanConnection(row)
	if scanErr != nil {
		return nil, translateConnectionWriteError("更新连接失败", scanErr)
	}
	if err := applyPlainConnectionSecretsTx(ctx, tx, r.cipher, record.ID, params.Secrets, params.ClearSecretKeys); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交连接更新事务失败", err)
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
            is_enabled = $6,
            metadata = $7::jsonb,
            updated_by = $8,
            updated_at = now()
        WHERE project_id = $1 AND id = $2
		RETURNING id, project_id, name, type, category, is_enabled, metadata, display_order, 0 AS variable_count,
                  'not_tested'::text,NULL::timestamptz,NULL::integer,NULL::text,created_at, updated_at
    `, params.ProjectID, params.ID, params.Name, params.Type, params.Category, connectionEnabledValue(params.IsEnabled), string(configBytes), params.UserID)

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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == "data_history_storage_targets_connection_fkey" {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusConflict, "该连接正在被历史存储配置使用，请先更换或删除对应存储目标")
		}
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除连接失败", err)
	}
	if commandTag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "连接不存在")
	}
	return nil
}

func connectionEnabledValue(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}

// SaveTestRecord 保存一次已保存连接测试摘要；敏感请求配置不进入测试记录。
func (r *ConnectionRepository) SaveTestRecord(ctx context.Context, projectID, connectionID, userID, status string, durationMS int, message string) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO data_connection_test_records(project_id,connection_id,status,duration_ms,message,tested_by) VALUES($1,$2,$3,$4,$5,$6)`, projectID, connectionID, status, durationMS, message, userID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "保存连接测试摘要失败", err)
	}
	return nil
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
		&record.IsEnabled,
		&configBytes,
		&record.DisplayOrder,
		&record.VariableCount,
		&record.LastTestStatus,
		&record.LastTestedAt,
		&record.LastTestDurationMS,
		&record.LastTestMessage,
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
