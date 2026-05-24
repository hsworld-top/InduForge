package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	apperrors "github.com/indu-forge/data_service/internal/errors"
)

const (
	defaultBuiltinSQLLimit = 100
	maxBuiltinSQLLimit     = 500
)

// BuiltinMessagePublisher 抽象消息发布实现，便于测试时注入 fake publisher。
type BuiltinMessagePublisher interface {
	Publish(ctx context.Context, topic string, payload []byte, qos byte) error
}

// BuiltinRuntimeService 承载四类 IF 内置运行库的开发态测试能力。
type BuiltinRuntimeService struct {
	devPool            *pgxpool.Pool
	metaPool           *pgxpool.Pool
	realtimeClient     redis.UniversalClient
	realtimeKeyPrefix  string
	messagePublisher   BuiltinMessagePublisher
	messageTopicPrefix string
	messageHub         *builtinMessageHub
}

type BuiltinRuntimeOptions struct {
	DevPool            *pgxpool.Pool
	MetaPool           *pgxpool.Pool
	RealtimeClient     redis.UniversalClient
	RealtimeKeyPrefix  string
	MessagePublisher   BuiltinMessagePublisher
	MessageTopicPrefix string
}

func NewBuiltinRuntimeService(options BuiltinRuntimeOptions) *BuiltinRuntimeService {
	realtimeKeyPrefix := strings.Trim(strings.TrimSpace(options.RealtimeKeyPrefix), ":")
	if realtimeKeyPrefix == "" {
		realtimeKeyPrefix = "ifdev"
	}
	messageTopicPrefix := strings.Trim(strings.TrimSpace(options.MessageTopicPrefix), "/")
	if messageTopicPrefix == "" {
		messageTopicPrefix = "ifdev"
	}
	return &BuiltinRuntimeService{
		devPool:            options.DevPool,
		metaPool:           options.MetaPool,
		realtimeClient:     options.RealtimeClient,
		realtimeKeyPrefix:  realtimeKeyPrefix,
		messagePublisher:   options.MessagePublisher,
		messageTopicPrefix: messageTopicPrefix,
		messageHub:         newBuiltinMessageHub(),
	}
}

type BuiltinSQLExecuteInput struct {
	Store      string
	SQL        string
	Parameters []any
	Limit      int
}

type BuiltinSQLExecuteResult struct {
	Columns       []string         `json:"columns"`
	Rows          []map[string]any `json:"rows"`
	RowCount      int              `json:"rowCount"`
	ExecutionTime int64            `json:"executionTime"`
}

type BuiltinTimeseriesSampleInput struct {
	Table string
	Point string
	Value any
	TS    *time.Time
}

type BuiltinRealtimeSetInput struct {
	Key        string
	Value      any
	TtlSeconds int
	RuntimeKey string
}

type BuiltinRealtimeValue struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value any    `json:"value"`
	TTL   int64  `json:"ttl"`
}

type BuiltinRealtimeKeyDefinition struct {
	ID                string    `json:"id"`
	Key               string    `json:"key"`
	ValueType         string    `json:"valueType"`
	DefaultTtlSeconds int       `json:"defaultTtlSeconds"`
	Description       string    `json:"description"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type BuiltinMessageTopic struct {
	ID          string    `json:"id"`
	Topic       string    `json:"topic"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type BuiltinMessageVariable struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	PayloadPath     string    `json:"payloadPath"`
	ValueType       string    `json:"valueType"`
	Unit            string    `json:"unit"`
	Description     string    `json:"description"`
	CreateDatapoint bool      `json:"createDatapoint"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type BuiltinMessagePublishInput struct {
	ConnectionID string
	Topic        string
	Payload      any
	QOS          byte
	RuntimeKey   string
}

type BuiltinMessagePublishResult struct {
	Topic string `json:"topic"`
	QOS   byte   `json:"qos"`
}

type BuiltinMessageEvent struct {
	ConnectionID string    `json:"connectionId"`
	RuntimeKey   string    `json:"runtimeKey"`
	Topic        string    `json:"topic"`
	FullTopic    string    `json:"fullTopic"`
	Payload      any       `json:"payload"`
	QOS          byte      `json:"qos"`
	Timestamp    time.Time `json:"timestamp"`
}

type BuiltinMessagePreviewSession struct {
	SessionID   string    `json:"sessionId"`
	TopicPrefix string    `json:"topicPrefix"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (s *BuiltinRuntimeService) ExecuteSQL(ctx context.Context, projectID string, input BuiltinSQLExecuteInput) (*BuiltinSQLExecuteResult, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if s == nil || s.devPool == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态关系/时序运行库未初始化")
	}
	sqlText := strings.TrimSpace(input.SQL)
	if sqlText == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 不能为空")
	}
	if err := validateBuiltinSQL(sqlText); err != nil {
		return nil, err
	}

	schemaSuffix, err := builtinSQLSchemaSuffix(input.Store)
	if err != nil {
		return nil, err
	}
	schemaName := deriveBuiltinProjectSchema(projectID, schemaSuffix)
	if err := s.ensureSchema(ctx, schemaName); err != nil {
		return nil, err
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultBuiltinSQLLimit
	}
	if limit > maxBuiltinSQLLimit {
		limit = maxBuiltinSQLLimit
	}

	startedAt := time.Now()
	tx, err := s.devPool.Begin(ctx)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启内置运行库 SQL 事务失败", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %s, public", pgx.Identifier{schemaName}.Sanitize())); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "设置内置运行库 schema 失败", err)
	}

	if isBuiltinQuerySQL(sqlText) {
		rows, err := tx.Query(ctx, sqlText, input.Parameters...)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "执行内置运行库 SQL 失败", err)
		}
		defer rows.Close()
		result, err := collectBuiltinSQLRows(rows, limit)
		if err != nil {
			return nil, err
		}
		result.ExecutionTime = time.Since(startedAt).Milliseconds()
		if err := tx.Commit(ctx); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交内置运行库 SQL 事务失败", err)
		}
		return result, nil
	}

	commandTag, err := tx.Exec(ctx, sqlText, input.Parameters...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "执行内置运行库 SQL 失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交内置运行库 SQL 事务失败", err)
	}
	return &BuiltinSQLExecuteResult{
		Columns:       []string{},
		Rows:          []map[string]any{},
		RowCount:      int(commandTag.RowsAffected()),
		ExecutionTime: time.Since(startedAt).Milliseconds(),
	}, nil
}

func (s *BuiltinRuntimeService) ExecuteSQLInSchema(ctx context.Context, schemaName string, sqlText string, parameters []any, limit int) (*BuiltinSQLExecuteResult, error) {
	if s == nil || s.devPool == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态关系/时序运行库未初始化")
	}
	sqlText = strings.TrimSpace(sqlText)
	if sqlText == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 不能为空")
	}
	if err := validateBuiltinSQL(sqlText); err != nil {
		return nil, err
	}
	schemaName = sanitizeBuiltinIdentifier(schemaName, "")
	if schemaName == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库 schema 无效")
	}
	if err := s.ensureSchema(ctx, schemaName); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultBuiltinSQLLimit
	}
	if limit > maxBuiltinSQLLimit {
		limit = maxBuiltinSQLLimit
	}

	startedAt := time.Now()
	tx, err := s.devPool.Begin(ctx)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开启内置运行库 SQL 事务失败", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %s, public", pgx.Identifier{schemaName}.Sanitize())); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "设置内置运行库 schema 失败", err)
	}
	if isBuiltinQuerySQL(sqlText) {
		rows, err := tx.Query(ctx, sqlText, parameters...)
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "执行内置运行库 SQL 失败", err)
		}
		defer rows.Close()
		result, err := collectBuiltinSQLRows(rows, limit)
		if err != nil {
			return nil, err
		}
		result.ExecutionTime = time.Since(startedAt).Milliseconds()
		if err := tx.Commit(ctx); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交内置运行库 SQL 事务失败", err)
		}
		return result, nil
	}
	commandTag, err := tx.Exec(ctx, sqlText, parameters...)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "执行内置运行库 SQL 失败", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "提交内置运行库 SQL 事务失败", err)
	}
	return &BuiltinSQLExecuteResult{
		Columns:       []string{},
		Rows:          []map[string]any{},
		RowCount:      int(commandTag.RowsAffected()),
		ExecutionTime: time.Since(startedAt).Milliseconds(),
	}, nil
}

func (s *BuiltinRuntimeService) ListTablesInSchema(ctx context.Context, schemaName string) ([]RelationalTable, error) {
	if s == nil || s.devPool == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态关系/时序运行库未初始化")
	}
	schemaName = sanitizeBuiltinIdentifier(schemaName, "")
	if schemaName == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库 schema 无效")
	}
	if err := s.ensureSchema(ctx, schemaName); err != nil {
		return nil, err
	}
	rows, err := s.devPool.Query(ctx, `
		SELECT table_schema, table_name, table_type
		FROM information_schema.tables
		WHERE table_schema = $1
		  AND table_type IN ('BASE TABLE', 'VIEW')
		ORDER BY table_name
	`, schemaName)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询内置运行库表列表失败", err)
	}
	defer rows.Close()
	tables := make([]RelationalTable, 0)
	for rows.Next() {
		table := RelationalTable{}
		if err := rows.Scan(&table.Schema, &table.Name, &table.Type); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取内置运行库表列表失败", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历内置运行库表列表失败", err)
	}
	return tables, nil
}

func (s *BuiltinRuntimeService) GetTableStructureInSchema(ctx context.Context, schemaName, tableName string) (*RelationalTableStructure, error) {
	if s == nil || s.devPool == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态关系/时序运行库未初始化")
	}
	schemaName = sanitizeBuiltinIdentifier(schemaName, "")
	tableName = sanitizeBuiltinIdentifier(tableName, "")
	if schemaName == "" || tableName == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置运行库表名无效")
	}
	rows, err := s.devPool.Query(ctx, `
		SELECT column_name, data_type, is_nullable = 'YES', column_default
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`, schemaName, tableName)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询内置运行库表结构失败", err)
	}
	defer rows.Close()
	columns := make([]RelationalTableColumn, 0)
	for rows.Next() {
		var column RelationalTableColumn
		var defaultValue *string
		if err := rows.Scan(&column.Name, &column.Type, &column.Nullable, &defaultValue); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取内置运行库表结构失败", err)
		}
		column.DefaultValue = defaultValue
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历内置运行库表结构失败", err)
	}
	return &RelationalTableStructure{Columns: columns, Indexes: []RelationalIndex{}, ForeignKeys: []RelationalForeignKey{}}, nil
}

func (s *BuiltinRuntimeService) WriteTimeseriesSample(ctx context.Context, projectID string, input BuiltinTimeseriesSampleInput) (*BuiltinSQLExecuteResult, error) {
	tableName := sanitizeBuiltinIdentifier(input.Table, "point_samples")
	point := strings.TrimSpace(input.Point)
	if point == "" {
		point = "demo.point"
	}
	ts := time.Now().UTC()
	if input.TS != nil && !input.TS.IsZero() {
		ts = input.TS.UTC()
	}
	createSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (ts timestamptz NOT NULL, point text NOT NULL, value jsonb NOT NULL)`, pgx.Identifier{tableName}.Sanitize())
	if _, err := s.ExecuteSQL(ctx, projectID, BuiltinSQLExecuteInput{Store: "timeseries", SQL: createSQL}); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(input.Value)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "样例值必须可序列化为 JSON", err)
	}
	insertSQL := fmt.Sprintf(`INSERT INTO %s (ts, point, value) VALUES ($1, $2, $3::jsonb)`, pgx.Identifier{tableName}.Sanitize())
	return s.ExecuteSQL(ctx, projectID, BuiltinSQLExecuteInput{
		Store:      "timeseries",
		SQL:        insertSQL,
		Parameters: []any{ts, point, string(payload)},
	})
}

func (s *BuiltinRuntimeService) SetRealtimeKey(ctx context.Context, projectID string, input BuiltinRealtimeSetInput) (*BuiltinRealtimeValue, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if s == nil || s.realtimeClient == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态实时运行库未初始化")
	}
	key, err := deriveBuiltinRealtimeKey(s.realtimeKeyPrefix, projectID, input.RuntimeKey, input.Key)
	if err != nil {
		return nil, err
	}
	valueBytes, err := json.Marshal(input.Value)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时库 value 必须可序列化为 JSON", err)
	}
	ttl := time.Duration(input.TtlSeconds) * time.Second
	if input.TtlSeconds < 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TTL 不能小于 0")
	}
	if err := s.realtimeClient.Set(ctx, key, string(valueBytes), ttl).Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "写入实时库 key 失败", err)
	}
	_, _ = s.UpsertRealtimeKeyDefinition(ctx, projectID, input.RuntimeKey, input.Key, "json", input.TtlSeconds, "")
	return s.GetRealtimeKey(ctx, projectID, input.RuntimeKey, input.Key)
}

func (s *BuiltinRuntimeService) ListRealtimeKeyDefinitions(ctx context.Context, projectID, connectionID string) ([]BuiltinRealtimeKeyDefinition, error) {
	metaPool := s.metaDB()
	if metaPool == nil {
		return []BuiltinRealtimeKeyDefinition{}, nil
	}
	rows, err := metaPool.Query(ctx, `
		SELECT id::text, key_path, value_type, default_ttl_seconds, description, created_at, updated_at
		FROM data_builtin_realtime_keys
		WHERE project_id = $1 AND connection_id = $2
		ORDER BY key_path
	`, projectID, connectionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询实时库 key 定义失败", err)
	}
	defer rows.Close()
	keys := make([]BuiltinRealtimeKeyDefinition, 0)
	for rows.Next() {
		item := BuiltinRealtimeKeyDefinition{}
		if err := rows.Scan(&item.ID, &item.Key, &item.ValueType, &item.DefaultTtlSeconds, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取实时库 key 定义失败", err)
		}
		keys = append(keys, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历实时库 key 定义失败", err)
	}
	return keys, nil
}

func (s *BuiltinRuntimeService) UpsertRealtimeKeyDefinition(ctx context.Context, projectID, connectionID, keyPath, valueType string, ttlSeconds int, description string) (*BuiltinRealtimeKeyDefinition, error) {
	metaPool := s.metaDB()
	if metaPool == nil {
		return nil, nil
	}
	keyPath = strings.Trim(strings.TrimSpace(keyPath), "/")
	if keyPath == "" || strings.Contains(keyPath, "..") {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时库 key 不合法")
	}
	valueType = strings.TrimSpace(valueType)
	if valueType == "" {
		valueType = "json"
	}
	if ttlSeconds < 0 {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "TTL 不能小于 0")
	}
	row := metaPool.QueryRow(ctx, `
		INSERT INTO data_builtin_realtime_keys (project_id, connection_id, key_path, value_type, default_ttl_seconds, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (project_id, connection_id, key_path)
		DO UPDATE SET value_type = EXCLUDED.value_type,
		              default_ttl_seconds = EXCLUDED.default_ttl_seconds,
		              description = EXCLUDED.description,
		              updated_at = now()
		RETURNING id::text, key_path, value_type, default_ttl_seconds, description, created_at, updated_at
	`, projectID, connectionID, keyPath, valueType, ttlSeconds, strings.TrimSpace(description))
	item := BuiltinRealtimeKeyDefinition{}
	if err := row.Scan(&item.ID, &item.Key, &item.ValueType, &item.DefaultTtlSeconds, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "保存实时库 key 定义失败", err)
	}
	return &item, nil
}

func (s *BuiltinRuntimeService) GetRealtimeKey(ctx context.Context, projectID, runtimeKey, key string) (*BuiltinRealtimeValue, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if s == nil || s.realtimeClient == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态实时运行库未初始化")
	}
	fullKey, err := deriveBuiltinRealtimeKey(s.realtimeKeyPrefix, projectID, runtimeKey, key)
	if err != nil {
		return nil, err
	}
	raw, err := s.realtimeClient.Get(ctx, fullKey).Result()
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "读取实时库 key 失败", err)
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		value = raw
	}
	ttl, _ := s.realtimeClient.TTL(ctx, fullKey).Result()
	return &BuiltinRealtimeValue{
		Key:   key,
		Type:  "string",
		Value: value,
		TTL:   int64(ttl.Seconds()),
	}, nil
}

func (s *BuiltinRuntimeService) DeleteRealtimeKey(ctx context.Context, projectID, runtimeKey, key string) (map[string]bool, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if s == nil || s.realtimeClient == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态实时运行库未初始化")
	}
	fullKey, err := deriveBuiltinRealtimeKey(s.realtimeKeyPrefix, projectID, runtimeKey, key)
	if err != nil {
		return nil, err
	}
	if err := s.realtimeClient.Del(ctx, fullKey).Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "删除实时库 key 失败", err)
	}
	return map[string]bool{"deleted": true}, nil
}

func (s *BuiltinRuntimeService) PublishMessage(ctx context.Context, projectID string, input BuiltinMessagePublishInput) (*BuiltinMessagePublishResult, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	if s == nil || s.messagePublisher == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态消息运行库未初始化")
	}
	topic, err := deriveBuiltinMessageTopic(s.messageTopicPrefix, projectID, input.RuntimeKey, input.Topic)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(input.Payload)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "消息 payload 必须可序列化为 JSON", err)
	}
	if err := s.messagePublisher.Publish(ctx, topic, payload, input.QOS); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "发布内置消息失败", err)
	}
	s.emitBuiltinMessage(BuiltinMessageEvent{
		ConnectionID: strings.TrimSpace(input.ConnectionID),
		RuntimeKey:   strings.TrimSpace(input.RuntimeKey),
		Topic:        strings.Trim(strings.TrimSpace(input.Topic), "/"),
		FullTopic:    topic,
		Payload:      input.Payload,
		QOS:          input.QOS,
		Timestamp:    time.Now().UTC(),
	})
	return &BuiltinMessagePublishResult{Topic: topic, QOS: input.QOS}, nil
}

func (s *BuiltinRuntimeService) SubscribeMessageTopic(topicID string, handler func(BuiltinMessageEvent)) func() {
	if s == nil || s.messageHub == nil {
		return func() {}
	}
	return s.messageHub.subscribe(topicID, handler)
}

func (s *BuiltinRuntimeService) SubscribeMessageSubscription(subscriptionID string, handler func(BuiltinMessageEvent)) func() {
	return s.SubscribeMessageTopic(subscriptionID, handler)
}

func (s *BuiltinRuntimeService) ListMessageTopics(ctx context.Context, projectID, connectionID string) ([]BuiltinMessageTopic, error) {
	metaPool := s.metaDB()
	if metaPool == nil {
		return []BuiltinMessageTopic{}, nil
	}
	rows, err := metaPool.Query(ctx, `
		SELECT id::text, topic, name, description, created_at, updated_at
		FROM data_builtin_message_topics
		WHERE project_id = $1 AND connection_id = $2
		ORDER BY topic
	`, projectID, connectionID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询消息库 topic 失败", err)
	}
	defer rows.Close()
	topics := make([]BuiltinMessageTopic, 0)
	for rows.Next() {
		item := BuiltinMessageTopic{}
		if err := rows.Scan(&item.ID, &item.Topic, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取消息库 topic 失败", err)
		}
		topics = append(topics, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历消息库 topic 失败", err)
	}
	return topics, nil
}

func (s *BuiltinRuntimeService) CreateMessageTopic(ctx context.Context, projectID, connectionID, topic, name, description string) (*BuiltinMessageTopic, error) {
	metaPool := s.metaDB()
	if metaPool == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态消息运行库未初始化")
	}
	topic = strings.Trim(strings.TrimSpace(topic), "/")
	if topic == "" || strings.Contains(topic, "..") || strings.Contains(topic, "#") || strings.Contains(topic, "+") {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "消息 topic 不合法")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = topic
	}
	row := metaPool.QueryRow(ctx, `
		INSERT INTO data_builtin_message_topics (project_id, connection_id, topic, name, description)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (project_id, connection_id, topic)
		DO UPDATE SET name = EXCLUDED.name,
		              description = EXCLUDED.description,
		              updated_at = now()
		RETURNING id::text, topic, name, description, created_at, updated_at
	`, projectID, connectionID, topic, name, strings.TrimSpace(description))
	item := BuiltinMessageTopic{}
	if err := row.Scan(&item.ID, &item.Topic, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "保存消息库 topic 失败", err)
	}
	return &item, nil
}

func (s *BuiltinRuntimeService) GetMessageTopic(ctx context.Context, projectID, connectionID, topicID string) (*BuiltinMessageTopic, error) {
	metaPool := s.metaDB()
	if metaPool == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态消息运行库未初始化")
	}
	topicID = strings.TrimSpace(topicID)
	if topicID == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "消息 topic ID 不能为空")
	}
	row := metaPool.QueryRow(ctx, `
		SELECT id::text, topic, name, description, created_at, updated_at
		FROM data_builtin_message_topics
		WHERE project_id = $1 AND connection_id = $2 AND id = $3
	`, projectID, connectionID, topicID)
	item := BuiltinMessageTopic{}
	if err := row.Scan(&item.ID, &item.Topic, &item.Name, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "消息 topic 不存在")
		}
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取消息库 topic 失败", err)
	}
	return &item, nil
}

func (s *BuiltinRuntimeService) ListMessageVariables(ctx context.Context, projectID, topicID string) ([]BuiltinMessageVariable, error) {
	metaPool := s.metaDB()
	if metaPool == nil {
		return []BuiltinMessageVariable{}, nil
	}
	rows, err := metaPool.Query(ctx, `
		SELECT id::text, name, payload_path, value_type, unit, description, create_datapoint, created_at, updated_at
		FROM data_builtin_message_variables
		WHERE project_id = $1 AND topic_id = $2
		ORDER BY name
	`, projectID, topicID)
	if err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "查询消息变量失败", err)
	}
	defer rows.Close()
	variables := make([]BuiltinMessageVariable, 0)
	for rows.Next() {
		item := BuiltinMessageVariable{}
		if err := rows.Scan(&item.ID, &item.Name, &item.PayloadPath, &item.ValueType, &item.Unit, &item.Description, &item.CreateDatapoint, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取消息变量失败", err)
		}
		variables = append(variables, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "遍历消息变量失败", err)
	}
	return variables, nil
}

func (s *BuiltinRuntimeService) CreateMessageVariable(ctx context.Context, projectID, topicID, name, payloadPath, valueType, unit, description string, createDatapoint bool) (*BuiltinMessageVariable, error) {
	metaPool := s.metaDB()
	if metaPool == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态消息运行库未初始化")
	}
	name = strings.TrimSpace(name)
	payloadPath = strings.TrimSpace(payloadPath)
	if name == "" || payloadPath == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "变量名称和 payload path 不能为空")
	}
	if valueType == "" {
		valueType = "string"
	}
	row := metaPool.QueryRow(ctx, `
		INSERT INTO data_builtin_message_variables (project_id, topic_id, name, payload_path, value_type, unit, description, create_datapoint)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (project_id, topic_id, name)
		DO UPDATE SET payload_path = EXCLUDED.payload_path,
		              value_type = EXCLUDED.value_type,
		              unit = EXCLUDED.unit,
		              description = EXCLUDED.description,
		              create_datapoint = EXCLUDED.create_datapoint,
		              updated_at = now()
		RETURNING id::text, name, payload_path, value_type, unit, description, create_datapoint, created_at, updated_at
	`, projectID, topicID, name, payloadPath, valueType, strings.TrimSpace(unit), strings.TrimSpace(description), createDatapoint)
	item := BuiltinMessageVariable{}
	if err := row.Scan(&item.ID, &item.Name, &item.PayloadPath, &item.ValueType, &item.Unit, &item.Description, &item.CreateDatapoint, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "保存消息变量失败", err)
	}
	return &item, nil
}

func (s *BuiltinRuntimeService) CreateMessagePreviewSession(ctx context.Context, projectID string) (*BuiltinMessagePreviewSession, error) {
	if err := validateProjectID(projectID); err != nil {
		return nil, err
	}
	return &BuiltinMessagePreviewSession{
		SessionID:   fmt.Sprintf("builtin-msg-%d", time.Now().UnixNano()),
		TopicPrefix: fmt.Sprintf("%s/%s", s.messageTopicPrefix, projectID),
		CreatedAt:   time.Now().UTC(),
	}, nil
}

func (s *BuiltinRuntimeService) metaDB() *pgxpool.Pool {
	if s == nil {
		return nil
	}
	if s.metaPool != nil {
		return s.metaPool
	}
	return s.devPool
}

func (s *BuiltinRuntimeService) emitBuiltinMessage(event BuiltinMessageEvent) {
	if s == nil || s.messageHub == nil {
		return
	}
	for _, subscriptionID := range s.messageSubscriptionIDs(event.ConnectionID, event.Topic) {
		s.messageHub.publish(subscriptionID, event)
	}
}

func (s *BuiltinRuntimeService) messageSubscriptionIDs(connectionID, topic string) []string {
	metaPool := s.metaDB()
	connectionID = strings.TrimSpace(connectionID)
	topic = strings.Trim(strings.TrimSpace(topic), "/")
	if metaPool == nil || connectionID == "" || topic == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	rows, err := metaPool.Query(ctx, `
		SELECT id::text
		FROM data_mqtt_subscriptions
		WHERE connection_id = $1 AND topic = $2
	`, connectionID, topic)
	if err != nil {
		return nil
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil && id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

type builtinMessageHub struct {
	mu          sync.RWMutex
	nextID      int64
	subscribers map[string]map[int64]func(BuiltinMessageEvent)
	lastEvents  map[string]BuiltinMessageEvent
}

func newBuiltinMessageHub() *builtinMessageHub {
	return &builtinMessageHub{
		subscribers: make(map[string]map[int64]func(BuiltinMessageEvent)),
		lastEvents:  make(map[string]BuiltinMessageEvent),
	}
}

func (h *builtinMessageHub) subscribe(topicID string, handler func(BuiltinMessageEvent)) func() {
	topicID = strings.TrimSpace(topicID)
	if h == nil || topicID == "" || handler == nil {
		return func() {}
	}
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	if h.subscribers[topicID] == nil {
		h.subscribers[topicID] = make(map[int64]func(BuiltinMessageEvent))
	}
	h.subscribers[topicID][id] = handler
	last, hasLast := h.lastEvents[topicID]
	h.mu.Unlock()
	if hasLast {
		go handler(last)
	}
	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if subscribers := h.subscribers[topicID]; subscribers != nil {
			delete(subscribers, id)
			if len(subscribers) == 0 {
				delete(h.subscribers, topicID)
			}
		}
	}
}

func (h *builtinMessageHub) publish(topicID string, event BuiltinMessageEvent) {
	topicID = strings.TrimSpace(topicID)
	if h == nil || topicID == "" {
		return
	}
	h.mu.Lock()
	h.lastEvents[topicID] = event
	handlers := make([]func(BuiltinMessageEvent), 0, len(h.subscribers[topicID]))
	for _, handler := range h.subscribers[topicID] {
		handlers = append(handlers, handler)
	}
	h.mu.Unlock()
	for _, handler := range handlers {
		handler(event)
	}
}

func (s *BuiltinRuntimeService) ensureSchema(ctx context.Context, schemaName string) error {
	if s == nil || s.devPool == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态运行库未初始化")
	}
	identifier := pgx.Identifier{schemaName}.Sanitize()
	if _, err := s.devPool.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, identifier)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "创建开发态 schema 失败", err)
	}
	if _, err := s.devPool.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.if_schema_migrations (
			version text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)
	`, identifier)); err != nil {
		return apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "创建开发态迁移表失败", err)
	}
	return nil
}

func validateBuiltinSQL(sqlText string) error {
	normalized := strings.ToLower(strings.TrimSpace(sqlText))
	blocked := []string{
		"drop database",
		"create database",
		"alter system",
		"copy ",
		" program ",
		"pg_catalog",
		"information_schema",
		"pg_toast",
		"if_core",
		"if_data",
		"if_dev_data",
		"postgres.",
		"template0",
		"template1",
	}
	for _, token := range blocked {
		if strings.Contains(normalized, token) {
			return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 包含不允许访问的系统资源或危险语句")
		}
	}
	if strings.Contains(normalized, "p_") && strings.Contains(normalized, "_app.") {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 不允许显式访问工程 schema")
	}
	if strings.Contains(normalized, "p_") && strings.Contains(normalized, "_ts.") {
		return apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "SQL 不允许显式访问工程 schema")
	}
	return nil
}

func builtinSQLSchemaSuffix(store string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(store)) {
	case "relation":
		return "app", nil
	case "timeseries":
		return "ts", nil
	default:
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "内置 SQL 运行库类型不受支持")
	}
}

func isBuiltinQuerySQL(sqlText string) bool {
	normalized := strings.TrimSpace(strings.ToLower(sqlText))
	return strings.HasPrefix(normalized, "select") || strings.HasPrefix(normalized, "with")
}

func collectBuiltinSQLRows(rows pgx.Rows, limit int) (*BuiltinSQLExecuteResult, error) {
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, 0, len(fieldDescriptions))
	for _, field := range fieldDescriptions {
		columns = append(columns, field.Name)
	}

	resultRows := make([]map[string]any, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, apperrors.WrapAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "读取内置运行库 SQL 结果失败", err)
		}
		if len(resultRows) >= limit {
			continue
		}
		item := map[string]any{}
		for index, column := range columns {
			if index < len(values) {
				item[column] = normalizeBuiltinSQLValue(values[index])
			}
		}
		resultRows = append(resultRows, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.WrapAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "遍历内置运行库 SQL 结果失败", err)
	}
	return &BuiltinSQLExecuteResult{
		Columns:  columns,
		Rows:     resultRows,
		RowCount: len(resultRows),
	}, nil
}

func normalizeBuiltinSQLValue(value any) any {
	switch typed := value.(type) {
	case []byte:
		return string(typed)
	default:
		return typed
	}
}

func deriveBuiltinRealtimeKey(prefix, projectID, runtimeKey, key string) (string, error) {
	runtimeKey = strings.Trim(strings.TrimSpace(runtimeKey), ":")
	if runtimeKey == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时库缺少系统标识")
	}
	key = strings.Trim(strings.TrimSpace(key), "/")
	if key == "" || strings.Contains(key, "..") {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "实时库 key 不合法")
	}
	return fmt.Sprintf("%s:%s:%s:%s", prefix, projectID, runtimeKey, key), nil
}

func deriveBuiltinMessageTopic(prefix, projectID, runtimeKey, topic string) (string, error) {
	runtimeKey = strings.Trim(strings.TrimSpace(runtimeKey), "/")
	if runtimeKey == "" {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "消息库缺少系统标识")
	}
	topic = strings.Trim(strings.TrimSpace(topic), "/")
	if topic == "" || strings.Contains(topic, "..") || strings.Contains(topic, "#") || strings.Contains(topic, "+") {
		return "", apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "消息 topic 不合法")
	}
	return fmt.Sprintf("%s/%s/%s/%s", strings.Trim(prefix, "/"), projectID, runtimeKey, topic), nil
}

func sanitizeBuiltinIdentifier(value string, defaultValue string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultValue
	}
	normalized := schemaUnsafePattern.ReplaceAllString(strings.ToLower(value), "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return defaultValue
	}
	return normalized
}

type pgxCommandTag interface {
	RowsAffected() int64
}

var _ pgxCommandTag = pgconn.CommandTag{}
