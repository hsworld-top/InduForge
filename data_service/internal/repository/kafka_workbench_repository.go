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

// KafkaTopicGroupRecord 表示 Kafka 工作台中的项目内 Topic 分组。
type KafkaTopicGroupRecord struct {
	ID           string
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	SortOrder    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// KafkaTopicMappingRecord 表示 Kafka 工作台中的项目内 Topic 映射。
type KafkaTopicMappingRecord struct {
	ID            string
	ProjectID     string
	ConnectionID  string
	GroupID       *string
	Name          string
	Topic         string
	Description   string
	PartitionMode string
	Partition     *int
	StartPosition string
	Decode        string
	SampleLimit   int
	TimeoutMS     int
	SortOrder     int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// KafkaFieldRecord 表示 Kafka 消息字段与数据点之间的映射。
type KafkaFieldRecord struct {
	ID             string
	ProjectID      string
	ConnectionID   string
	TopicMappingID string
	Name           string
	ValuePath      string
	KeyPath        string
	DataType       string
	Enabled        bool
	Description    string
	SortOrder      int
	DataPointID    *string
	DataPointPath  *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateKafkaTopicGroupParams 描述 Kafka Topic 分组创建参数。
type CreateKafkaTopicGroupParams struct {
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	SortOrder    int
	UserID       string
}

// UpdateKafkaTopicGroupParams 描述 Kafka Topic 分组更新参数。
type UpdateKafkaTopicGroupParams struct {
	ProjectID string
	GroupID   string
	ParentID  *string
	HasParent bool
	Name      string
	SortOrder int
	UserID    string
}

// CreateKafkaTopicMappingParams 描述 Kafka Topic 映射创建参数。
type CreateKafkaTopicMappingParams struct {
	ProjectID     string
	ConnectionID  string
	GroupID       *string
	Name          string
	Topic         string
	Description   string
	PartitionMode string
	Partition     *int
	StartPosition string
	Decode        string
	SampleLimit   int
	TimeoutMS     int
	SortOrder     int
	UserID        string
}

// UpdateKafkaTopicMappingParams 描述 Kafka Topic 映射更新参数。
type UpdateKafkaTopicMappingParams struct {
	ID            string
	ProjectID     string
	GroupID       *string
	HasGroupID    bool
	Name          string
	Topic         string
	Description   string
	PartitionMode string
	Partition     *int
	StartPosition string
	Decode        string
	SampleLimit   int
	TimeoutMS     int
	SortOrder     int
	UserID        string
}

// CreateKafkaFieldParams 描述 Kafka 字段映射创建参数。
type CreateKafkaFieldParams struct {
	ProjectID      string
	ConnectionID   string
	TopicMappingID string
	Name           string
	ValuePath      string
	KeyPath        string
	DataType       string
	Enabled        bool
	Description    string
	SortOrder      int
	DataPointPath  string
	SourceConfig   map[string]any
	UserID         string
}

// UpdateKafkaFieldParams 描述 Kafka 字段映射更新参数。
type UpdateKafkaFieldParams struct {
	ID            string
	ProjectID     string
	Name          string
	ValuePath     string
	KeyPath       string
	DataType      string
	Enabled       bool
	Description   string
	SortOrder     int
	DataPointPath string
	SourceConfig  map[string]any
	UserID        string
}

// KafkaWorkbenchRepository 封装 Kafka 工作台参数化 SQL。
type KafkaWorkbenchRepository struct {
	pool *pgxpool.Pool
}

// NewKafkaWorkbenchRepository 创建 Kafka 工作台仓储。
func NewKafkaWorkbenchRepository(pool *pgxpool.Pool) *KafkaWorkbenchRepository {
	return &KafkaWorkbenchRepository{pool: pool}
}

// ListTopicGroups 返回连接下的 Topic 分组。
func (r *KafkaWorkbenchRepository) ListTopicGroups(ctx context.Context, projectID, connectionID string) ([]KafkaTopicGroupRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, connection_id, parent_id, name, sort_order, created_at, updated_at
		FROM data_kafka_topic_groups
		WHERE project_id = $1 AND connection_id = $2
		ORDER BY sort_order ASC, created_at ASC
	`, projectID, connectionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 Kafka Topic 分组失败", err)
	}
	defer rows.Close()

	result := make([]KafkaTopicGroupRecord, 0)
	for rows.Next() {
		record, scanErr := scanKafkaTopicGroupRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 Kafka Topic 分组失败", err)
	}
	return result, nil
}

// CreateTopicGroup 创建 Topic 分组。
func (r *KafkaWorkbenchRepository) CreateTopicGroup(ctx context.Context, params CreateKafkaTopicGroupParams) (*KafkaTopicGroupRecord, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO data_kafka_topic_groups (project_id, connection_id, parent_id, name, sort_order, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		RETURNING id, project_id, connection_id, parent_id, name, sort_order, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.ParentID, params.Name, params.SortOrder, params.UserID)

	record, err := scanKafkaTopicGroupRecord(row)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "创建 Kafka Topic 分组失败", err)
	}
	return &record, nil
}

// UpdateTopicGroup 更新 Topic 分组。
func (r *KafkaWorkbenchRepository) UpdateTopicGroup(ctx context.Context, params UpdateKafkaTopicGroupParams) (*KafkaTopicGroupRecord, error) {
	parentID := params.ParentID
	row := r.pool.QueryRow(ctx, `
		UPDATE data_kafka_topic_groups
		SET parent_id = CASE WHEN $3 THEN $4 ELSE parent_id END,
		    name = $5,
		    sort_order = $6,
		    updated_by = $7,
		    updated_at = now()
		WHERE project_id = $1 AND id = $2
		RETURNING id, project_id, connection_id, parent_id, name, sort_order, created_at, updated_at
	`, params.ProjectID, params.GroupID, params.HasParent, parentID, params.Name, params.SortOrder, params.UserID)

	record, err := scanKafkaTopicGroupRecord(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Kafka Topic 分组不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新 Kafka Topic 分组失败", err)
	}
	return &record, nil
}

// DeleteTopicGroup 删除 Topic 分组，并将子分组与映射移回根目录。
func (r *KafkaWorkbenchRepository) DeleteTopicGroup(ctx context.Context, projectID, groupID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 Kafka Topic 分组删除事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	if _, err := tx.Exec(ctx, `UPDATE data_kafka_topic_mappings SET group_id = NULL WHERE project_id = $1 AND group_id = $2`, projectID, groupID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "移动 Kafka Topic 映射失败", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE data_kafka_topic_groups SET parent_id = NULL WHERE project_id = $1 AND parent_id = $2`, projectID, groupID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "移动 Kafka Topic 子分组失败", err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM data_kafka_topic_groups WHERE project_id = $1 AND id = $2`, projectID, groupID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 Kafka Topic 分组失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Kafka Topic 分组不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 Kafka Topic 分组删除事务失败", err)
	}
	return nil
}

// ListTopicMappings 返回连接下的 Topic 映射。
func (r *KafkaWorkbenchRepository) ListTopicMappings(ctx context.Context, projectID, connectionID string) ([]KafkaTopicMappingRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, connection_id, group_id, name, topic, description,
		       partition_mode, partition, start_position, decode, sample_limit,
		       timeout_ms, sort_order, created_at, updated_at
		FROM data_kafka_topic_mappings
		WHERE project_id = $1 AND connection_id = $2
		ORDER BY sort_order ASC, created_at ASC
	`, projectID, connectionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 Kafka Topic 映射失败", err)
	}
	defer rows.Close()

	result := make([]KafkaTopicMappingRecord, 0)
	for rows.Next() {
		record, scanErr := scanKafkaTopicMappingRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 Kafka Topic 映射失败", err)
	}
	return result, nil
}

// GetTopicMapping 返回单个 Topic 映射。
func (r *KafkaWorkbenchRepository) GetTopicMapping(ctx context.Context, projectID, mappingID string) (*KafkaTopicMappingRecord, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, connection_id, group_id, name, topic, description,
		       partition_mode, partition, start_position, decode, sample_limit,
		       timeout_ms, sort_order, created_at, updated_at
		FROM data_kafka_topic_mappings
		WHERE project_id = $1 AND id = $2
	`, projectID, mappingID)
	record, err := scanKafkaTopicMappingRecord(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Kafka Topic 映射不存在")
		}
		return nil, err
	}
	return &record, nil
}

// CreateTopicMapping 创建 Topic 映射。
func (r *KafkaWorkbenchRepository) CreateTopicMapping(ctx context.Context, params CreateKafkaTopicMappingParams) (*KafkaTopicMappingRecord, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO data_kafka_topic_mappings (
			project_id, connection_id, group_id, name, topic, description,
			partition_mode, partition, start_position, decode, sample_limit,
			timeout_ms, sort_order, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $14)
		RETURNING id, project_id, connection_id, group_id, name, topic, description,
		          partition_mode, partition, start_position, decode, sample_limit,
		          timeout_ms, sort_order, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.GroupID, params.Name, params.Topic, params.Description, params.PartitionMode, params.Partition, params.StartPosition, params.Decode, params.SampleLimit, params.TimeoutMS, params.SortOrder, params.UserID)

	record, err := scanKafkaTopicMappingRecord(row)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "创建 Kafka Topic 映射失败", err)
	}
	return &record, nil
}

// UpdateTopicMapping 更新 Topic 映射。
func (r *KafkaWorkbenchRepository) UpdateTopicMapping(ctx context.Context, params UpdateKafkaTopicMappingParams) (*KafkaTopicMappingRecord, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE data_kafka_topic_mappings
		SET group_id = CASE WHEN $3 THEN $4 ELSE group_id END,
		    name = $5,
		    topic = $6,
		    description = $7,
		    partition_mode = $8,
		    partition = $9,
		    start_position = $10,
		    decode = $11,
		    sample_limit = $12,
		    timeout_ms = $13,
		    sort_order = $14,
		    updated_by = $15,
		    updated_at = now()
		WHERE project_id = $1 AND id = $2
		RETURNING id, project_id, connection_id, group_id, name, topic, description,
		          partition_mode, partition, start_position, decode, sample_limit,
		          timeout_ms, sort_order, created_at, updated_at
	`, params.ProjectID, params.ID, params.HasGroupID, params.GroupID, params.Name, params.Topic, params.Description, params.PartitionMode, params.Partition, params.StartPosition, params.Decode, params.SampleLimit, params.TimeoutMS, params.SortOrder, params.UserID)

	record, err := scanKafkaTopicMappingRecord(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Kafka Topic 映射不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新 Kafka Topic 映射失败", err)
	}
	return &record, nil
}

// DeleteTopicMapping 删除 Topic 映射，并将其生成的数据点标记为失效。
func (r *KafkaWorkbenchRepository) DeleteTopicMapping(ctx context.Context, projectID, mappingID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 Kafka Topic 映射删除事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	if _, err := tx.Exec(ctx, `
		UPDATE data_points
		SET status = 'invalid', updated_at = now()
		WHERE project_id = $1
		  AND source_type = 'kafka.field'
		  AND source_config->>'topicMappingId' = $2
	`, projectID, mappingID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "标记 Kafka 字段数据点失效失败", err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM data_kafka_topic_mappings WHERE project_id = $1 AND id = $2`, projectID, mappingID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 Kafka Topic 映射失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Kafka Topic 映射不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 Kafka Topic 映射删除事务失败", err)
	}
	return nil
}

// ListFields 返回 Topic 映射下的字段映射。
func (r *KafkaWorkbenchRepository) ListFields(ctx context.Context, projectID, mappingID string) ([]KafkaFieldRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT f.id, f.project_id, f.connection_id, f.topic_mapping_id, f.name, f.value_path,
		       f.key_path, f.data_type, f.enabled, f.description, f.sort_order,
		       dp.id, dp.path, f.created_at, f.updated_at
		FROM data_kafka_fields f
		LEFT JOIN data_points dp
		  ON dp.project_id = f.project_id
		 AND dp.source_type = 'kafka.field'
		 AND dp.source_config->>'fieldId' = f.id::text
		WHERE f.project_id = $1 AND f.topic_mapping_id = $2
		ORDER BY f.sort_order ASC, f.created_at ASC
	`, projectID, mappingID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询 Kafka 字段映射失败", err)
	}
	defer rows.Close()

	result := make([]KafkaFieldRecord, 0)
	for rows.Next() {
		record, scanErr := scanKafkaFieldRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历 Kafka 字段映射失败", err)
	}
	return result, nil
}

// CreateFieldWithDataPoint 创建字段映射并同步生成数据点。
func (r *KafkaWorkbenchRepository) CreateFieldWithDataPoint(ctx context.Context, params CreateKafkaFieldParams) (*KafkaFieldRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 Kafka 字段创建事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	record := KafkaFieldRecord{}
	err = tx.QueryRow(ctx, `
		INSERT INTO data_kafka_fields (
			project_id, connection_id, topic_mapping_id, name, value_path, key_path,
			data_type, enabled, description, sort_order, created_by, updated_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)
		RETURNING id, project_id, connection_id, topic_mapping_id, name, value_path,
		          key_path, data_type, enabled, description, sort_order, created_at, updated_at
	`, params.ProjectID, params.ConnectionID, params.TopicMappingID, params.Name, params.ValuePath, params.KeyPath, params.DataType, params.Enabled, params.Description, params.SortOrder, params.UserID).Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.TopicMappingID,
		&record.Name,
		&record.ValuePath,
		&record.KeyPath,
		&record.DataType,
		&record.Enabled,
		&record.Description,
		&record.SortOrder,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "创建 Kafka 字段映射失败", err)
	}

	dataPointID, dataPointPath, err := upsertKafkaFieldDataPoint(ctx, tx, record, params.DataPointPath, params.SourceConfig, params.UserID)
	if err != nil {
		return nil, err
	}
	record.DataPointID = &dataPointID
	record.DataPointPath = &dataPointPath

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 Kafka 字段创建事务失败", err)
	}
	return &record, nil
}

// UpdateFieldWithDataPoint 更新字段映射并同步数据点。
func (r *KafkaWorkbenchRepository) UpdateFieldWithDataPoint(ctx context.Context, params UpdateKafkaFieldParams) (*KafkaFieldRecord, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 Kafka 字段更新事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	record := KafkaFieldRecord{}
	err = tx.QueryRow(ctx, `
		UPDATE data_kafka_fields
		SET name = $3,
		    value_path = $4,
		    key_path = $5,
		    data_type = $6,
		    enabled = $7,
		    description = $8,
		    sort_order = $9,
		    updated_by = $10,
		    updated_at = now()
		WHERE project_id = $1 AND id = $2
		RETURNING id, project_id, connection_id, topic_mapping_id, name, value_path,
		          key_path, data_type, enabled, description, sort_order, created_at, updated_at
	`, params.ProjectID, params.ID, params.Name, params.ValuePath, params.KeyPath, params.DataType, params.Enabled, params.Description, params.SortOrder, params.UserID).Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.TopicMappingID,
		&record.Name,
		&record.ValuePath,
		&record.KeyPath,
		&record.DataType,
		&record.Enabled,
		&record.Description,
		&record.SortOrder,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Kafka 字段映射不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "更新 Kafka 字段映射失败", err)
	}

	dataPointID, dataPointPath, err := upsertKafkaFieldDataPoint(ctx, tx, record, params.DataPointPath, params.SourceConfig, params.UserID)
	if err != nil {
		return nil, err
	}
	record.DataPointID = &dataPointID
	record.DataPointPath = &dataPointPath

	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 Kafka 字段更新事务失败", err)
	}
	return &record, nil
}

// DeleteFieldWithDataPoint 删除字段映射，并将对应数据点标记为失效。
func (r *KafkaWorkbenchRepository) DeleteFieldWithDataPoint(ctx context.Context, projectID, fieldID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启 Kafka 字段删除事务失败", err)
	}
	defer rollbackProtocolTxQuietly(ctx, tx)

	if _, err := tx.Exec(ctx, `
		UPDATE data_points
		SET status = 'invalid', updated_at = now()
		WHERE project_id = $1
		  AND source_type = 'kafka.field'
		  AND source_config->>'fieldId' = $2
	`, projectID, fieldID); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "标记 Kafka 字段数据点失效失败", err)
	}
	tag, err := tx.Exec(ctx, `DELETE FROM data_kafka_fields WHERE project_id = $1 AND id = $2`, projectID, fieldID)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "删除 Kafka 字段映射失败", err)
	}
	if tag.RowsAffected() == 0 {
		return apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Kafka 字段映射不存在")
	}
	if err := tx.Commit(ctx); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交 Kafka 字段删除事务失败", err)
	}
	return nil
}

// GetPreviewConnection 按 Kafka 连接读取短时抓样配置。
func (r *KafkaWorkbenchRepository) GetPreviewConnection(ctx context.Context, projectID, connectionID string) (*ProtocolPreviewConnectionRecord, error) {
	var (
		record        ProtocolPreviewConnectionRecord
		configPayload []byte
	)
	err := r.pool.QueryRow(ctx, `
		SELECT conn.id::text, conn.project_id, conn.name, conn.type, conn.status,
		       jsonb_build_object(
		         'brokers', kafka.brokers,
		         'topic', kafka.topic,
		         'consumerGroup', kafka.consumer_group,
		         'startPosition', kafka.start_position,
		         'options', kafka.options
		       ) AS config
		FROM data_connections conn
		JOIN data_kafka_configs kafka ON kafka.connection_id = conn.id
		WHERE conn.project_id = $1
		  AND conn.id = $2
		  AND conn.type = 'kafka'
	`, projectID, connectionID).Scan(&record.ID, &record.ProjectID, &record.Name, &record.Type, &record.Status, &configPayload)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "Kafka 预览配置不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取 Kafka 预览配置失败", err)
	}
	if err := json.Unmarshal(configPayload, &record.Config); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "解析 Kafka 预览配置失败", err)
	}
	if record.Config == nil {
		record.Config = map[string]any{}
	}
	return &record, nil
}

// CreatePreviewRecord 写入 Kafka 工作台预览摘要。
func (r *KafkaWorkbenchRepository) CreatePreviewRecord(ctx context.Context, params CreateAccessSourceRecordParams) error {
	detailPayload, err := json.Marshal(params.Detail)
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka 预览记录 detail 格式无效", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO data_access_source_records (
			project_id, connection_id, record_type, title, status, protocol,
			duration_ms, sample_count, truncated, error_summary, detail
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10, ''), $11::jsonb)
	`, params.ProjectID, params.ConnectionID, params.RecordType, params.Title, params.Status, params.Protocol, params.DurationMS, params.SampleCount, params.Truncated, params.ErrorSummary, string(detailPayload))
	if err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "写入 Kafka 预览记录失败", err)
	}
	return nil
}

func upsertKafkaFieldDataPoint(ctx context.Context, tx pgx.Tx, field KafkaFieldRecord, path string, sourceConfig map[string]any, userID string) (string, string, error) {
	sourceConfigWithField := cloneProtocolMap(sourceConfig)
	sourceConfigWithField["fieldId"] = field.ID
	sourceConfigPayload, err := json.Marshal(sourceConfigWithField)
	if err != nil {
		return "", "", apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "Kafka 字段数据点 sourceConfig 格式无效", err)
	}
	status := "active"
	if !field.Enabled {
		status = "inactive"
	}

	var dataPointID, dataPointPath string
	err = tx.QueryRow(ctx, `
		INSERT INTO data_points (
			project_id, path, name, source_type, source_id, source_config,
			data_type, refresh_mode, status, created_by, updated_by
		)
		VALUES ($1, $2, $3, 'kafka.field', $4, $5::jsonb, $6, 'subscription', $7, $8, $8)
		ON CONFLICT (project_id, path)
		DO UPDATE SET
			name = EXCLUDED.name,
			source_type = EXCLUDED.source_type,
			source_id = EXCLUDED.source_id,
			source_config = EXCLUDED.source_config,
			data_type = EXCLUDED.data_type,
			refresh_mode = EXCLUDED.refresh_mode,
			status = EXCLUDED.status,
			updated_by = EXCLUDED.updated_by,
			updated_at = now()
		RETURNING id, path
	`, field.ProjectID, path, field.Name, field.ConnectionID, string(sourceConfigPayload), field.DataType, status, userID).Scan(&dataPointID, &dataPointPath)
	if err != nil {
		return "", "", apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "同步 Kafka 字段数据点失败", err)
	}
	return dataPointID, dataPointPath, nil
}

func scanKafkaTopicGroupRecord(row pgx.Row) (KafkaTopicGroupRecord, error) {
	record := KafkaTopicGroupRecord{}
	if err := row.Scan(&record.ID, &record.ProjectID, &record.ConnectionID, &record.ParentID, &record.Name, &record.SortOrder, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return record, err
	}
	return record, nil
}

func scanKafkaTopicMappingRecord(row pgx.Row) (KafkaTopicMappingRecord, error) {
	record := KafkaTopicMappingRecord{}
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.GroupID,
		&record.Name,
		&record.Topic,
		&record.Description,
		&record.PartitionMode,
		&record.Partition,
		&record.StartPosition,
		&record.Decode,
		&record.SampleLimit,
		&record.TimeoutMS,
		&record.SortOrder,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return record, err
	}
	return record, nil
}

func scanKafkaFieldRecord(row pgx.Row) (KafkaFieldRecord, error) {
	record := KafkaFieldRecord{}
	if err := row.Scan(
		&record.ID,
		&record.ProjectID,
		&record.ConnectionID,
		&record.TopicMappingID,
		&record.Name,
		&record.ValuePath,
		&record.KeyPath,
		&record.DataType,
		&record.Enabled,
		&record.Description,
		&record.SortOrder,
		&record.DataPointID,
		&record.DataPointPath,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return record, err
	}
	return record, nil
}
