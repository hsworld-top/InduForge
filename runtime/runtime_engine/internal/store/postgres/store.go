// Package postgres 提供 RuntimeEngine V1 的 PostgreSQL 状态、fence 与事务 outbox 基座。
package postgres

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/indu-forge/runtime-engine/internal/eventid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const schemaVersion = "runtime_engine.v1"

//go:embed schema.sql
var SchemaSQL string

var (
	ErrFenceRejected     = errors.New("role fence 被拒绝")
	ErrFenceStale        = errors.New("role fence 已过期")
	ErrOutboxConflict    = errors.New("outbox dedupe key 对应的内容不一致")
	ErrLeaseLost         = errors.New("outbox lease 已失效")
	ErrSequenceExhausted = errors.New("producer sequence 已耗尽")
	ErrInvalidInput      = errors.New("postgres store 输入非法")
	// ErrStoreClosed is fatal to a delivery: timeout shutdown deliberately
	// refuses all new database work so callers must not Ack/Nak as success.
	ErrStoreClosed = errors.New("postgres store 已终止")
)

// Store 不保存或记录 DSN；连接池仅由调用者传入的已解析 DSN 构造。
type Store struct {
	pool      *pgxpool.Pool
	closed    atomic.Bool
	abortOnce sync.Once
	closeOnce sync.Once
	closeDone chan struct{}
	doneOnce  sync.Once
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	if dsn == "" {
		return nil, fmt.Errorf("%w: 空 DSN", ErrInvalidInput)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		// pgx/URL 解析错误的文本在不同版本中可能携带原始连接串，绝不能透传。
		return nil, errors.New("创建 PostgreSQL 连接池失败")
	}
	s := &Store{pool: pool, closeDone: make(chan struct{})}
	if err := s.Ping(ctx); err != nil {
		pool.Close()
		return nil, errors.New("PostgreSQL 连通性校验失败")
	}
	if err := s.VerifySchema(ctx); err != nil {
		pool.Close()
		return nil, errors.New("PostgreSQL 状态库基线校验失败")
	}
	return s, nil
}

// Close is the normal, synchronous close used only after workers have drained.
func (s *Store) Close() {
	if s == nil {
		return
	}
	s.closed.Store(true)
	if s.pool != nil {
		s.closeOnce.Do(s.closePool)
	}
}

// Abort is the timeout path. It immediately rejects new work, resets checked
// out connections so they are destroyed when returned, and closes the pool in
// the background because pgxpool.Close waits for a blocked transaction.
func (s *Store) Abort() {
	if s == nil {
		return
	}
	s.closed.Store(true)
	if s.pool == nil {
		return
	}
	s.abortOnce.Do(func() {
		s.pool.Reset()
		go s.closeOnce.Do(s.closePool)
	})
}

func (s *Store) closePool() {
	s.pool.Close()
	if s.closeDone != nil {
		s.doneOnce.Do(func() { close(s.closeDone) })
	}
}

func (s *Store) ensureOpen() error {
	if s == nil || s.pool == nil || s.closed.Load() {
		return ErrStoreClosed
	}
	return nil
}

func (s *Store) Ping(ctx context.Context) error {
	if err := s.ensureOpen(); err != nil {
		return err
	}
	if err := s.pool.Ping(ctx); err != nil {
		return errors.New("PostgreSQL Ping 失败")
	}
	return nil
}

// VerifySchema 只验证基线的精确版本，绝不隐式执行建表、迁移或修复。
func (s *Store) VerifySchema(ctx context.Context) error {
	if err := s.ensureOpen(); err != nil {
		return err
	}
	var count int
	var version *string
	err := s.pool.QueryRow(ctx, `SELECT count(*), min(version) FROM runtime_engine.schema_meta`).Scan(&count, &version)
	if err != nil {
		return fmt.Errorf("读取 RuntimeEngine schema 基线失败: %w", err)
	}
	if count != 1 || version == nil || *version != schemaVersion {
		return fmt.Errorf("RuntimeEngine schema 基线版本不匹配")
	}
	return s.verifySchemaContract(ctx)
}

type schemaColumn struct {
	name, typ string
	notNull   bool
}

// verifySchemaContract 防止有人只伪造 schema_meta：同时核验 V1 全部表的精确列布局、关键约束和查询索引。
func (s *Store) verifySchemaContract(ctx context.Context) error {
	expected := map[string][]schemaColumn{
		"schema_meta":            {{"version", "text", true}, {"applied_at", "timestamp with time zone", true}},
		"role_fence":             {{"deployment_id", "text", true}, {"role", "text", true}, {"owner_id", "text", true}, {"epoch", "bigint", true}, {"version", "bigint", true}, {"activated_at", "timestamp with time zone", true}},
		"producer_fence":         {{"deployment_id", "text", true}, {"producer_key", "text", true}, {"owner_id", "text", true}, {"epoch", "bigint", true}, {"version", "bigint", true}, {"activated_at", "timestamp with time zone", true}},
		"processed_event":        {{"deployment_id", "text", true}, {"consumer_key", "text", true}, {"event_id", "text", true}, {"body_sha256", "text", true}, {"subject", "text", true}, {"delivery_metadata", "jsonb", true}, {"processed_at", "timestamp with time zone", true}},
		"consumer_checkpoint":    {{"deployment_id", "text", true}, {"consumer_key", "text", true}, {"position", "bigint", true}, {"version", "bigint", true}, {"updated_at", "timestamp with time zone", true}},
		"transactional_outbox":   {{"id", "bigint", true}, {"deployment_id", "text", true}, {"dedupe_key", "text", true}, {"subject", "text", true}, {"headers", "jsonb", true}, {"payload", "bytea", true}, {"payload_sha256", "text", true}, {"state", "text", true}, {"lease_token", "text", false}, {"lease_owner", "text", false}, {"lease_until", "timestamp with time zone", false}, {"attempt_count", "integer", true}, {"next_attempt_at", "timestamp with time zone", true}, {"last_error_code", "text", false}, {"published_at", "timestamp with time zone", false}, {"created_at", "timestamp with time zone", true}, {"updated_at", "timestamp with time zone", true}},
		"processing_failure":     {{"id", "bigint", true}, {"deployment_id", "text", true}, {"consumer_key", "text", true}, {"dlq_id", "text", true}, {"event_id", "text", false}, {"reason_code", "text", true}, {"delivery_count", "integer", true}, {"jetstream_position", "bigint", true}, {"subject", "text", true}, {"body_sha256", "text", true}, {"raw_body", "bytea", false}, {"occurred_at", "timestamp with time zone", true}, {"quarantined_at", "timestamp with time zone", true}},
		"producer_sequence":      {{"deployment_id", "text", true}, {"producer_key", "text", true}, {"owner_id", "text", true}, {"epoch", "bigint", true}, {"last_sequence", "bigint", true}, {"updated_at", "timestamp with time zone", true}},
		"point_history":          {{"deployment_id", "text", true}, {"point_id", "uuid", true}, {"event_id", "text", true}, {"owner_id", "text", true}, {"epoch", "bigint", true}, {"sequence", "bigint", true}, {"source_timestamp", "timestamp with time zone", true}, {"server_timestamp", "timestamp with time zone", true}, {"value", "jsonb", false}, {"quality", "text", true}, {"received_at", "timestamp with time zone", true}},
		"point_current":          {{"deployment_id", "text", true}, {"point_id", "uuid", true}, {"owner_id", "text", true}, {"epoch", "bigint", true}, {"source_timestamp", "timestamp with time zone", true}, {"server_timestamp", "timestamp with time zone", true}, {"sequence", "bigint", true}, {"event_id", "text", true}, {"value", "jsonb", false}, {"quality", "text", true}, {"version", "bigint", true}, {"updated_at", "timestamp with time zone", true}},
		"compute_input_snapshot": {{"deployment_id", "text", true}, {"compute_id", "uuid", true}, {"datapoint_id", "uuid", true}, {"event_id", "text", true}, {"owner_id", "text", true}, {"epoch", "bigint", true}, {"sequence", "bigint", true}, {"source_timestamp", "timestamp with time zone", true}, {"server_timestamp", "timestamp with time zone", true}, {"received_at", "timestamp with time zone", true}, {"value", "jsonb", false}, {"quality", "text", true}, {"version", "bigint", true}, {"updated_at", "timestamp with time zone", true}},
		"compute_trigger_state":  {{"deployment_id", "text", true}, {"compute_id", "uuid", true}, {"compute_revision", "bigint", true}, {"previous_value", "jsonb", false}, {"previous_quality", "text", false}, {"previous_seen", "boolean", true}, {"condition_active", "boolean", true}, {"pending_phase", "text", false}, {"pending_since", "timestamp with time zone", false}, {"updated_at", "timestamp with time zone", true}},
		"compute_schedule_state": {{"deployment_id", "text", true}, {"compute_id", "uuid", true}, {"compute_revision", "bigint", true}, {"run_count", "bigint", true}, {"next_run_at", "timestamp with time zone", false}, {"last_run_at", "timestamp with time zone", false}, {"last_local_key", "text", false}, {"updated_at", "timestamp with time zone", true}},
		"alarm_item_state":       {{"deployment_id", "text", true}, {"alarm_item_id", "uuid", true}, {"alarm_revision", "bigint", true}, {"state", "jsonb", true}, {"version", "bigint", true}, {"next_evaluation_at", "timestamp with time zone", false}, {"updated_at", "timestamp with time zone", true}},
	}
	defaults := map[string]map[string]string{"schema_meta": {"applied_at": "now()"}, "role_fence": {"activated_at": "now()"}, "producer_fence": {"activated_at": "now()"}, "processed_event": {"delivery_metadata": "'{}'::jsonb", "processed_at": "now()"}, "consumer_checkpoint": {"updated_at": "now()"}, "transactional_outbox": {"headers": "'{}'::jsonb", "state": "'pending'::text", "attempt_count": "0", "next_attempt_at": "now()", "created_at": "now()", "updated_at": "now()"}, "processing_failure": {"quarantined_at": "now()"}, "producer_sequence": {"updated_at": "now()"}, "point_current": {"updated_at": "now()"}, "compute_input_snapshot": {"updated_at": "now()"}, "compute_trigger_state": {"previous_seen": "false", "condition_active": "false", "updated_at": "now()"}, "compute_schedule_state": {"run_count": "0", "updated_at": "now()"}, "alarm_item_state": {"updated_at": "now()"}}
	identities := map[string]string{"transactional_outbox.id": "a", "processing_failure.id": "a"}
	rows, err := s.pool.Query(ctx, `SELECT c.relname,c.relpersistence::text,a.attname,pg_catalog.format_type(a.atttypid,a.atttypmod),a.attnotnull,COALESCE(pg_get_expr(d.adbin,d.adrelid),''),a.attidentity::text,a.attgenerated::text
FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
JOIN pg_catalog.pg_attribute a ON a.attrelid=c.oid
LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum
WHERE n.nspname='runtime_engine' AND c.relkind='r' AND a.attnum>0 AND NOT a.attisdropped ORDER BY c.relname,a.attnum`)
	if err != nil {
		return fmt.Errorf("读取 RuntimeEngine schema 目录失败: %w", err)
	}
	defer rows.Close()
	actual := map[string][]schemaColumn{}
	actualDefaults, actualIdentities, actualGenerated := map[string]map[string]string{}, map[string]string{}, map[string]string{}
	for rows.Next() {
		var table, persistence string
		var column schemaColumn
		var defaultExpr, identity, generated string
		if err := rows.Scan(&table, &persistence, &column.name, &column.typ, &column.notNull, &defaultExpr, &identity, &generated); err != nil {
			return fmt.Errorf("读取 RuntimeEngine schema 列失败: %w", err)
		}
		if persistence != "p" {
			return fmt.Errorf("RuntimeEngine schema 表 %s 不是永久表", table)
		}
		actual[table] = append(actual[table], column)
		if actualDefaults[table] == nil {
			actualDefaults[table] = map[string]string{}
		}
		actualDefaults[table][column.name] = normalizeCatalog(defaultExpr)
		actualIdentities[table+"."+column.name] = identity
		actualGenerated[table+"."+column.name] = generated
	}
	if err := rows.Err(); err != nil {
		return errors.New("读取 RuntimeEngine schema 列失败")
	}
	if len(actual) != len(expected) {
		return errors.New("RuntimeEngine schema 表集合不匹配")
	}
	for table, columns := range expected {
		got, ok := actual[table]
		if !ok || len(got) != len(columns) {
			return fmt.Errorf("RuntimeEngine schema 表 %s 的列布局不匹配", table)
		}
		for i := range columns {
			if got[i] != columns[i] {
				return fmt.Errorf("RuntimeEngine schema 表 %s 的列定义不匹配", table)
			}
		}
		for _, column := range columns {
			expectedDefault := ""
			if defaults[table] != nil {
				expectedDefault = defaults[table][column.name]
			}
			if actualDefaults[table][column.name] != expectedDefault {
				return fmt.Errorf("RuntimeEngine schema 表 %s 的默认值不匹配", table)
			}
			key := table + "." + column.name
			expectedIdentity := identities[key]
			if actualIdentities[key] != expectedIdentity || actualGenerated[key] != "" {
				return fmt.Errorf("RuntimeEngine schema 表 %s 的 identity/generated 不匹配", table)
			}
		}
	}
	if err := s.verifySchemaDefinitions(ctx); err != nil {
		return err
	}
	return nil
}

func normalizeCatalog(value string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(value, `"`, "")), " ")
}

func (s *Store) verifySchemaDefinitions(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `SELECT c.relname,pg_get_constraintdef(k.oid,true),k.convalidated,k.condeferrable,k.condeferred FROM pg_catalog.pg_constraint k JOIN pg_catalog.pg_class c ON c.oid=k.conrelid JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='runtime_engine'`)
	if err != nil {
		return errors.New("读取 RuntimeEngine schema 约束失败")
	}
	defer rows.Close()
	definitions := map[string][]string{}
	for rows.Next() {
		var table, definition string
		var validated, deferrable, deferred bool
		if err := rows.Scan(&table, &definition, &validated, &deferrable, &deferred); err != nil {
			return errors.New("读取 RuntimeEngine schema 约束失败")
		}
		if !validated || deferrable || deferred {
			return errors.New("RuntimeEngine schema 约束属性不匹配")
		}
		definitions[table] = append(definitions[table], definition)
	}
	if err := rows.Err(); err != nil {
		return errors.New("读取 RuntimeEngine schema 约束失败")
	}
	required := map[string][]string{
		"schema_meta":            {"PRIMARY KEY (version)", "version = 'runtime_engine.v1'"},
		"role_fence":             {"PRIMARY KEY (deployment_id, role)", "epoch >= 1", "version > 0"},
		"producer_fence":         {"PRIMARY KEY (deployment_id, producer_key)", "epoch >= 1", "version > 0"},
		"processed_event":        {"PRIMARY KEY (deployment_id, consumer_key, event_id)", "event_id ~", "body_sha256 ~"},
		"consumer_checkpoint":    {"PRIMARY KEY (deployment_id, consumer_key)", "position >= 0", "version > 0"},
		"transactional_outbox":   {"UNIQUE (deployment_id, dedupe_key)", "state = ANY", "payload_sha256 ~", "last_error_code = ANY", "lease_token IS NOT NULL", "published_at IS NOT NULL"},
		"processing_failure":     {"UNIQUE (deployment_id, dlq_id)", "reason_code = ANY", "event_id IS NULL", "body_sha256 ~"},
		"producer_sequence":      {"PRIMARY KEY (deployment_id, producer_key)", "last_sequence >= 0"},
		"point_history":          {"PRIMARY KEY (deployment_id, point_id, event_id)", "quality = ANY"},
		"point_current":          {"PRIMARY KEY (deployment_id, point_id)", "quality = ANY", "version > 0"},
		"compute_input_snapshot": {"PRIMARY KEY (deployment_id, compute_id, datapoint_id)", "quality = ANY", "version > 0"},
		"compute_trigger_state":  {"PRIMARY KEY (deployment_id, compute_id)", "compute_revision > 0", "pending_phase = ANY"},
		"compute_schedule_state": {"PRIMARY KEY (deployment_id, compute_id)", "compute_revision > 0", "run_count >= 0", "last_run_at IS NULL"},
		"alarm_item_state":       {"PRIMARY KEY (deployment_id, alarm_item_id)", "alarm_revision > 0", "jsonb_typeof(state)", "version > 0"},
	}
	expectedConstraintCount := map[string]int{"schema_meta": 2, "role_fence": 6, "producer_fence": 6, "processed_event": 7, "consumer_checkpoint": 5, "transactional_outbox": 13, "processing_failure": 12, "producer_sequence": 6, "point_history": 8, "point_current": 9, "compute_input_snapshot": 9, "compute_trigger_state": 7, "compute_schedule_state": 6, "alarm_item_state": 5}
	for table, count := range expectedConstraintCount {
		if len(definitions[table]) != count {
			return fmt.Errorf("RuntimeEngine schema 表 %s 的约束集合不匹配", table)
		}
	}
	for table, fragments := range required {
		for _, fragment := range fragments {
			if !containsDefinition(definitions[table], fragment) {
				return fmt.Errorf("RuntimeEngine schema 表 %s 缺少关键约束", table)
			}
		}
	}
	expectedDefinitions := map[string][]string{
		"schema_meta":            {"CHECK (version = 'runtime_engine.v1'::text)", "PRIMARY KEY (version)"},
		"role_fence":             {"CHECK (epoch >= 1)", "CHECK (length(deployment_id) > 0)", "CHECK (length(owner_id) > 0)", "CHECK (length(role) > 0)", "CHECK (version > 0)", "PRIMARY KEY (deployment_id, role)"},
		"producer_fence":         {"CHECK (epoch >= 1)", "CHECK (length(deployment_id) > 0)", "CHECK (length(owner_id) > 0)", "CHECK (length(producer_key) > 0)", "CHECK (version > 0)", "PRIMARY KEY (deployment_id, producer_key)"},
		"processed_event":        {"CHECK (body_sha256 ~ '^[0-9a-f]{64}$'::text)", "CHECK (event_id ~ '^[0-9a-f]{64}$'::text)", "CHECK (jsonb_typeof(delivery_metadata) = 'object'::text AND octet_length(delivery_metadata::text) <= 16384)", "CHECK (length(consumer_key) > 0)", "CHECK (length(deployment_id) > 0)", "CHECK (length(subject) > 0 AND length(subject) <= 4096)", "PRIMARY KEY (deployment_id, consumer_key, event_id)"},
		"consumer_checkpoint":    {"CHECK (position >= 0)", "CHECK (length(consumer_key) > 0)", "CHECK (length(deployment_id) > 0)", "CHECK (version > 0)", "PRIMARY KEY (deployment_id, consumer_key)"},
		"transactional_outbox":   {"CHECK ((state = 'leased'::text) = (lease_token IS NOT NULL AND lease_owner IS NOT NULL AND lease_until IS NOT NULL))", "CHECK ((state = 'published'::text) = (published_at IS NOT NULL))", "CHECK (attempt_count >= 0)", "CHECK (jsonb_typeof(headers) = 'object'::text AND octet_length(headers::text) <= 16384)", "CHECK (last_error_code IS NULL OR (last_error_code = ANY (ARRAY['publish-error'::text, 'publish-timeout'::text, 'shutdown'::text])))", "CHECK (length(dedupe_key) > 0 AND length(dedupe_key) <= 512)", "CHECK (length(deployment_id) > 0)", "CHECK (length(subject) > 0 AND length(subject) <= 4096)", "CHECK (octet_length(payload) <= 2097152)", "CHECK (payload_sha256 ~ '^[0-9a-f]{64}$'::text)", "CHECK (state = ANY (ARRAY['pending'::text, 'leased'::text, 'published'::text]))", "PRIMARY KEY (id)", "UNIQUE (deployment_id, dedupe_key)"},
		"processing_failure":     {"CHECK (body_sha256 ~ '^[0-9a-f]{64}$'::text)", "CHECK (delivery_count >= 1)", "CHECK (event_id IS NULL OR event_id ~ '^[0-9a-f]{64}$'::text)", "CHECK (jetstream_position >= 0)", "CHECK (length(consumer_key) > 0)", "CHECK (length(deployment_id) > 0)", "CHECK (length(dlq_id) > 0 AND length(dlq_id) <= 512)", "CHECK (length(subject) > 0 AND length(subject) <= 4096)", "CHECK (raw_body IS NULL OR octet_length(raw_body) <= 1048576)", "CHECK (reason_code = ANY (ARRAY['permanent-validation'::text, 'event-id-collision'::text, 'max-deliver'::text, 'handler-failure'::text]))", "PRIMARY KEY (id)", "UNIQUE (deployment_id, dlq_id)"},
		"producer_sequence":      {"CHECK (epoch >= 1)", "CHECK (last_sequence >= 0)", "CHECK (length(deployment_id) > 0)", "CHECK (length(owner_id) > 0)", "CHECK (length(producer_key) > 0)", "PRIMARY KEY (deployment_id, producer_key)"},
		"point_history":          {"CHECK (deployment_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'::text)", "CHECK (epoch >= 1)", "CHECK (event_id ~ '^[0-9a-f]{64}$'::text)", "CHECK (owner_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'::text)", "CHECK (quality = ANY (ARRAY['good'::text, 'bad'::text, 'unknown'::text]))", "CHECK (sequence >= 0)", "CHECK (value IS NULL OR octet_length(value::text) <= 1048576)", "PRIMARY KEY (deployment_id, point_id, event_id)"},
		"point_current":          {"CHECK (deployment_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'::text)", "CHECK (epoch >= 1)", "CHECK (event_id ~ '^[0-9a-f]{64}$'::text)", "CHECK (owner_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'::text)", "CHECK (quality = ANY (ARRAY['good'::text, 'bad'::text, 'unknown'::text]))", "CHECK (sequence >= 0)", "CHECK (value IS NULL OR octet_length(value::text) <= 1048576)", "CHECK (version > 0)", "PRIMARY KEY (deployment_id, point_id)"},
		"compute_input_snapshot": {"CHECK (deployment_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'::text)", "CHECK (epoch >= 1)", "CHECK (event_id ~ '^[0-9a-f]{64}$'::text)", "CHECK (owner_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'::text)", "CHECK (quality = ANY (ARRAY['good'::text, 'bad'::text, 'unknown'::text]))", "CHECK (sequence >= 0)", "CHECK (value IS NULL OR octet_length(value::text) <= 1048576)", "CHECK (version > 0)", "PRIMARY KEY (deployment_id, compute_id, datapoint_id)"},
		"compute_trigger_state":  {"CHECK ((pending_phase IS NULL) = (pending_since IS NULL))", "CHECK (compute_revision > 0)", "CHECK (deployment_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'::text)", "CHECK (previous_quality IS NULL OR (previous_quality = ANY (ARRAY['good'::text, 'bad'::text, 'unknown'::text])))", "CHECK (previous_value IS NULL OR octet_length(previous_value::text) <= 1048576)", "CHECK (pending_phase IS NULL OR (pending_phase = ANY (ARRAY['entered'::text, 'exited'::text])))", "PRIMARY KEY (deployment_id, compute_id)"},
		"compute_schedule_state": {"CHECK (compute_revision > 0)", "CHECK (last_local_key IS NULL OR length(last_local_key) >= 1 AND length(last_local_key) <= 256)", "CHECK (last_run_at IS NULL OR last_local_key IS NOT NULL)", "CHECK (deployment_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'::text)", "CHECK (run_count >= 0)", "PRIMARY KEY (deployment_id, compute_id)"},
		"alarm_item_state":       {"CHECK (alarm_revision > 0)", "CHECK (deployment_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'::text)", "CHECK (jsonb_typeof(state) = 'object'::text AND octet_length(state::text) <= 1048576)", "CHECK (version > 0)", "PRIMARY KEY (deployment_id, alarm_item_id)"},
	}
	if len(definitions) != len(expectedDefinitions) {
		return errors.New("RuntimeEngine schema 约束集合不匹配")
	}
	for table, expected := range expectedDefinitions {
		actual := normalizeDefinitions(definitions[table])
		expected = normalizeDefinitions(expected)
		if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
			return fmt.Errorf("RuntimeEngine schema 表 %s 的约束定义不匹配", table)
		}
	}
	indexRows, err := s.pool.Query(ctx, `SELECT tablename,indexname,indexdef FROM pg_catalog.pg_indexes WHERE schemaname='runtime_engine'`)
	if err != nil {
		return errors.New("读取 RuntimeEngine schema 索引失败")
	}
	defer indexRows.Close()
	indexes := map[string][]string{}
	indexNames := map[string]string{}
	for indexRows.Next() {
		var table, name, definition string
		if err := indexRows.Scan(&table, &name, &definition); err != nil {
			return errors.New("读取 RuntimeEngine schema 索引失败")
		}
		indexes[table] = append(indexes[table], definition)
		indexNames[name] = definition
	}
	if err := indexRows.Err(); err != nil {
		return errors.New("读取 RuntimeEngine schema 索引失败")
	}
	expectedIndexNames := []string{"schema_meta_pkey", "role_fence_pkey", "processed_event_pkey", "producer_fence_pkey", "consumer_checkpoint_pkey", "transactional_outbox_pkey", "transactional_outbox_deployment_id_dedupe_key_key", "transactional_outbox_claim_idx", "processing_failure_pkey", "processing_failure_deployment_id_dlq_id_key", "processing_failure_event_idx", "producer_sequence_pkey", "point_history_pkey", "point_history_order_idx", "point_current_pkey", "compute_input_snapshot_pkey", "compute_input_snapshot_event_idx", "compute_trigger_state_pkey", "compute_schedule_state_pkey", "compute_schedule_due_idx", "alarm_item_state_pkey", "alarm_item_state_due_idx"}
	if len(indexNames) != len(expectedIndexNames) {
		return errors.New("RuntimeEngine schema 索引集合不匹配")
	}
	for _, name := range expectedIndexNames {
		if _, ok := indexNames[name]; !ok {
			return errors.New("RuntimeEngine schema 索引集合不匹配")
		}
	}
	for table, fragment := range map[string]string{"transactional_outbox": "USING btree (deployment_id, state, next_attempt_at, lease_until, id)", "processing_failure": "USING btree (deployment_id, consumer_key, event_id, quarantined_at DESC) WHERE (event_id IS NOT NULL)", "point_history": "USING btree (deployment_id, point_id, epoch DESC, source_timestamp DESC, sequence DESC)", "compute_input_snapshot": "USING btree (deployment_id, compute_id, event_id)", "compute_schedule_state": "USING btree (deployment_id, next_run_at, compute_id)", "alarm_item_state": "USING btree (deployment_id, next_evaluation_at, alarm_item_id)"} {
		if !containsDefinition(indexes[table], fragment) {
			return fmt.Errorf("RuntimeEngine schema 表 %s 缺少关键索引", table)
		}
	}
	if strings.Contains(indexNames["transactional_outbox_claim_idx"], " WHERE ") {
		return errors.New("RuntimeEngine schema outbox claim 索引谓词不匹配")
	}
	propertyRows, err := s.pool.Query(ctx, `SELECT i.relname,am.amname,x.indisvalid,x.indisready,x.indnkeyatts,x.indnatts,COALESCE(pg_get_expr(x.indpred,x.indrelid),'')
FROM pg_catalog.pg_index x JOIN pg_catalog.pg_class i ON i.oid=x.indexrelid JOIN pg_catalog.pg_class t ON t.oid=x.indrelid JOIN pg_catalog.pg_namespace n ON n.oid=t.relnamespace JOIN pg_catalog.pg_am am ON am.oid=i.relam WHERE n.nspname='runtime_engine'`)
	if err != nil {
		return errors.New("读取 RuntimeEngine schema 索引属性失败")
	}
	defer propertyRows.Close()
	for propertyRows.Next() {
		var name, method, predicate string
		var valid, ready bool
		var keyAttrs, allAttrs int16
		if err := propertyRows.Scan(&name, &method, &valid, &ready, &keyAttrs, &allAttrs, &predicate); err != nil {
			return errors.New("读取 RuntimeEngine schema 索引属性失败")
		}
		if method != "btree" || !valid || !ready || keyAttrs != allAttrs {
			return errors.New("RuntimeEngine schema 索引属性不匹配")
		}
		predicate = normalizeCatalog(predicate)
		if name == "processing_failure_event_idx" {
			if predicate != "(event_id IS NOT NULL)" {
				return errors.New("RuntimeEngine schema failure 索引谓词不匹配")
			}
		} else if predicate != "" {
			return errors.New("RuntimeEngine schema 索引谓词不匹配")
		}
	}
	if err := propertyRows.Err(); err != nil {
		return errors.New("读取 RuntimeEngine schema 索引属性失败")
	}
	return nil
}

func containsDefinition(definitions []string, fragment string) bool {
	sort.Strings(definitions)
	for _, definition := range definitions {
		if strings.Contains(strings.ReplaceAll(definition, `"`, ""), fragment) {
			return true
		}
	}
	return false
}

func normalizeDefinitions(definitions []string) []string {
	result := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		result = append(result, normalizeCatalog(definition))
	}
	sort.Strings(result)
	return result
}

// ApplySchema 是显式部署/测试初始化动作。重复执行应因 CREATE SCHEMA 失败，不能被当作迁移机制。
func (s *Store) ApplySchema(ctx context.Context) error {
	if err := s.ensureOpen(); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, SchemaSQL); err != nil {
		return fmt.Errorf("应用 RuntimeEngine schema 基线失败: %w", err)
	}
	return tx.Commit(ctx)
}

type ConsumerRoleToken struct {
	OwnerID string
	Epoch   int64
}

// ProducerToken 是生产者 fence，刻意不同于 ConsumerRoleToken，禁止跨用途比较。
type ProducerToken struct {
	OwnerID string
	Epoch   int64
}

type Fence struct {
	DeploymentID string
	Role         string
	OwnerID      string
	Epoch        int64
	Version      int64
}

// ActivateRole 显式激活已提交的 assignment。提升 epoch 时必须提交当前 version；同 token 重试幂等。
func (s *Store) ActivateRole(ctx context.Context, deploymentID, role string, token ConsumerRoleToken, expectedVersion int64) (Fence, error) {
	if err := validToken(deploymentID, role, token.OwnerID, token.Epoch); err != nil || expectedVersion < 0 {
		return Fence{}, fmt.Errorf("%w: role assignment", ErrInvalidInput)
	}
	if err := s.ensureOpen(); err != nil {
		return Fence{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Fence{}, err
	}
	defer tx.Rollback(ctx)
	var current Fence
	err = tx.QueryRow(ctx, `SELECT deployment_id, role, owner_id, epoch, version FROM runtime_engine.role_fence WHERE deployment_id=$1 AND role=$2 FOR UPDATE`, deploymentID, role).
		Scan(&current.DeploymentID, &current.Role, &current.OwnerID, &current.Epoch, &current.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		if expectedVersion != 0 {
			return Fence{}, fmt.Errorf("%w: 初始 assignment 的 expectedVersion 必须为 0", ErrFenceRejected)
		}
		current = Fence{DeploymentID: deploymentID, Role: role, OwnerID: token.OwnerID, Epoch: token.Epoch, Version: 1}
		if _, err = tx.Exec(ctx, `INSERT INTO runtime_engine.role_fence (deployment_id, role, owner_id, epoch, version) VALUES ($1,$2,$3,$4,$5)`, deploymentID, role, token.OwnerID, token.Epoch, current.Version); err != nil {
			return Fence{}, fmt.Errorf("写入 role fence 失败: %w", err)
		}
	} else if err != nil {
		return Fence{}, err
	} else if current.OwnerID == token.OwnerID && current.Epoch == token.Epoch {
		// 同 owner+epoch 的重复配置提交不改 version，调用方可安全重试。
	} else if token.Epoch <= current.Epoch {
		return Fence{}, fmt.Errorf("%w: owner/epoch 只能严格提升", ErrFenceRejected)
	} else if current.Version != expectedVersion {
		return Fence{}, fmt.Errorf("%w: expectedVersion 不匹配", ErrFenceRejected)
	} else {
		current.OwnerID, current.Epoch, current.Version = token.OwnerID, token.Epoch, current.Version+1
		if _, err = tx.Exec(ctx, `UPDATE runtime_engine.role_fence SET owner_id=$3, epoch=$4, version=$5, activated_at=now() WHERE deployment_id=$1 AND role=$2`, deploymentID, role, current.OwnerID, current.Epoch, current.Version); err != nil {
			return Fence{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Fence{}, err
	}
	return current, nil
}

type Message struct {
	DeploymentID string
	AccountID    string
	ConsumerKey  string
	Role         string
	Token        ConsumerRoleToken
	// ProducerKey/ProducerToken come from strict ingress validation.  They are
	// deliberately distinct from role ownership: a writer consumes facts owned
	// by collectors or compute producers, never by itself.
	ProducerKey        string
	ProducerToken      ProducerToken
	EventID            string
	RawBody            []byte
	Subject            string
	DeliveryMetadata   json.RawMessage
	CheckpointPosition int64
	DeliveryCount      int
	OccurredAt         time.Time
}

type Disposition string

const (
	Processed Disposition = "processed"
	Duplicate Disposition = "duplicate"
	Collision Disposition = "collision"
)

type Handler func(context.Context, *BusinessTx) error
type CollisionInfo struct {
	Message Message
	DLQID   string
}
type CollisionHandler func(context.Context, *BusinessTx, CollisionInfo) error

type ProcessOptions struct{ OnCollision CollisionHandler }

// ProcessMessage 完成去重、纯数据库副作用、checkpoint 和 outbox 的同事务提交；它绝不 Ack 外部消息。
func (s *Store) ProcessMessage(ctx context.Context, msg Message, handler Handler, options ProcessOptions) (Disposition, error) {
	if err := validMessage(msg); err != nil {
		return "", err
	}
	if err := s.ensureOpen(); err != nil {
		return "", err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if err := assertConsumerFence(ctx, tx, msg); err != nil {
		return "", err
	}
	if err := assertProducerFence(ctx, tx, msg.DeploymentID, msg.ProducerKey, msg.ProducerToken); err != nil {
		return "", err
	}
	digest := eventid.BodySHA256(msg.RawBody)
	var inserted int
	err = tx.QueryRow(ctx, `INSERT INTO runtime_engine.processed_event (deployment_id, consumer_key, event_id, body_sha256, subject, delivery_metadata)
VALUES ($1,$2,$3,$4,$5,$6::jsonb) ON CONFLICT DO NOTHING RETURNING 1`, msg.DeploymentID, msg.ConsumerKey, msg.EventID, digest, msg.Subject, metadataOrEmpty(msg.DeliveryMetadata)).Scan(&inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		var previous string
		if err = tx.QueryRow(ctx, `SELECT body_sha256 FROM runtime_engine.processed_event WHERE deployment_id=$1 AND consumer_key=$2 AND event_id=$3`, msg.DeploymentID, msg.ConsumerKey, msg.EventID).Scan(&previous); err != nil {
			return "", err
		}
		if previous == digest {
			// 重投可能携带更高的 JetStream delivery position；不重放副作用但安全水位可前移。
			if err := advanceCheckpointTx(ctx, tx, msg.DeploymentID, msg.ConsumerKey, msg.CheckpointPosition); err != nil {
				return "", err
			}
			if err := tx.Commit(ctx); err != nil {
				return "", err
			}
			return Duplicate, nil
		}
		if options.OnCollision == nil {
			return "", fmt.Errorf("%w: eventId collision 必须写入 DLQ outbox", ErrInvalidInput)
		}
		// 原始 body 仅保存于受限表或 DLQ outbox；错误文本绝不包含 body 内容。
		dlqID, dlqErr := runtimeDLQID(msg.DeploymentID, msg.ConsumerKey, msg.Subject, msg.EventID, FailureEventIDCollision, digest)
		if dlqErr != nil {
			return "", dlqErr
		}
		var failureInserted int
		err = tx.QueryRow(ctx, `INSERT INTO runtime_engine.processing_failure (deployment_id,consumer_key,dlq_id,event_id,reason_code,delivery_count,jetstream_position,subject,body_sha256,raw_body,occurred_at)
VALUES ($1,$2,$3,$4,'event-id-collision',$5,$6,$7,$8,$9,$10) ON CONFLICT (deployment_id,dlq_id) DO NOTHING RETURNING 1`, msg.DeploymentID, msg.ConsumerKey, dlqID, msg.EventID, msg.DeliveryCount, msg.CheckpointPosition, msg.Subject, digest, msg.RawBody, msg.OccurredAt.UTC()).Scan(&failureInserted)
		if err != nil && err != pgx.ErrNoRows {
			return "", err
		}
		if failureInserted == 1 {
			if err = options.OnCollision(ctx, &BusinessTx{store: s, tx: tx, deploymentID: msg.DeploymentID}, CollisionInfo{Message: msg, DLQID: dlqID}); err != nil {
				return "", err
			}
		}
		if err = ensureOutboxTx(ctx, tx, msg.DeploymentID, dlqID); err != nil {
			return "", err
		}
		if err := advanceCheckpointTx(ctx, tx, msg.DeploymentID, msg.ConsumerKey, msg.CheckpointPosition); err != nil {
			return "", err
		}
		if err := tx.Commit(ctx); err != nil {
			return "", err
		}
		return Collision, nil
	}
	if err != nil {
		return "", err
	}
	if handler != nil {
		if err := handler(ctx, &BusinessTx{store: s, tx: tx, deploymentID: msg.DeploymentID}); err != nil {
			return "", fmt.Errorf("消息 DB handler 失败: %w", err)
		}
	}
	if err := advanceCheckpointTx(ctx, tx, msg.DeploymentID, msg.ConsumerKey, msg.CheckpointPosition); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return Processed, nil
}

type checkpoint struct{ Position, Version int64 }

// advanceCheckpointTx 只能由已持有 role_fence 锁的消息/永久失败事务调用，外部没有管理捷径。
func advanceCheckpointTx(ctx context.Context, tx pgx.Tx, deploymentID, consumerKey string, position int64) error {
	var current checkpoint
	err := tx.QueryRow(ctx, `SELECT position, version FROM runtime_engine.consumer_checkpoint WHERE deployment_id=$1 AND consumer_key=$2 FOR UPDATE`, deploymentID, consumerKey).Scan(&current.Position, &current.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `INSERT INTO runtime_engine.consumer_checkpoint (deployment_id,consumer_key,position,version) VALUES ($1,$2,$3,1)`, deploymentID, consumerKey, position)
		return err
	}
	if err != nil {
		return err
	}
	if position <= current.Position {
		return nil
	}
	_, err = tx.Exec(ctx, `UPDATE runtime_engine.consumer_checkpoint SET position=$3, version=version+1, updated_at=now() WHERE deployment_id=$1 AND consumer_key=$2 AND version=$4`, deploymentID, consumerKey, position, current.Version)
	return err
}

func assertConsumerFence(ctx context.Context, tx pgx.Tx, msg Message) error {
	var owner string
	var epoch int64
	err := tx.QueryRow(ctx, `SELECT owner_id, epoch FROM runtime_engine.role_fence WHERE deployment_id=$1 AND role=$2 FOR UPDATE`, msg.DeploymentID, msg.Role).Scan(&owner, &epoch)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrFenceStale
	}
	if err != nil {
		return err
	}
	if owner != msg.Token.OwnerID || epoch != msg.Token.Epoch {
		return ErrFenceStale
	}
	return nil
}

func assertProducerFence(ctx context.Context, tx pgx.Tx, deploymentID, producerKey string, token ProducerToken) error {
	if err := validToken(deploymentID, producerKey, token.OwnerID, token.Epoch); err != nil {
		return err
	}
	var owner string
	var epoch int64
	err := tx.QueryRow(ctx, `SELECT owner_id,epoch FROM runtime_engine.producer_fence WHERE deployment_id=$1 AND producer_key=$2 FOR UPDATE`, deploymentID, producerKey).Scan(&owner, &epoch)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrFenceStale
	}
	if err != nil {
		return err
	}
	if owner != token.OwnerID || epoch != token.Epoch {
		return ErrFenceStale
	}
	return nil
}

func validToken(deployment, role, owner string, epoch int64) error {
	if deployment == "" || role == "" || owner == "" || epoch < 1 {
		return ErrInvalidInput
	}
	return nil
}

func validStableID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' || r == ':') {
			return false
		}
	}
	return true
}

func validMessage(msg Message) error {
	if err := validToken(msg.DeploymentID, msg.Role, msg.Token.OwnerID, msg.Token.Epoch); err != nil || validToken(msg.DeploymentID, msg.ProducerKey, msg.ProducerToken.OwnerID, msg.ProducerToken.Epoch) != nil || msg.AccountID == "" || msg.ConsumerKey == "" || msg.EventID == "" || msg.Subject == "" || len(msg.Subject) > 4096 || len(msg.RawBody) > 1<<20 || len(msg.DeliveryMetadata) > 16<<10 || msg.CheckpointPosition < 0 || msg.DeliveryCount < 1 || msg.OccurredAt.IsZero() {
		return fmt.Errorf("%w: message", ErrInvalidInput)
	}
	if len(msg.EventID) != 64 || msg.EventID != strings.ToLower(msg.EventID) {
		return fmt.Errorf("%w: eventId", ErrInvalidInput)
	}
	if _, err := hex.DecodeString(msg.EventID); err != nil {
		return fmt.Errorf("%w: eventId", ErrInvalidInput)
	}
	var metadata map[string]json.RawMessage
	if len(msg.DeliveryMetadata) > 0 && (json.Unmarshal(msg.DeliveryMetadata, &metadata) != nil || metadata == nil) {
		return fmt.Errorf("%w: delivery metadata", ErrInvalidInput)
	}
	return nil
}

func metadataOrEmpty(value json.RawMessage) []byte {
	if len(value) == 0 {
		return []byte("{}")
	}
	return value
}

func randomLeaseToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
func payloadSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// runtimeDLQID 使用冻结的原始 body 摘要；deliveryCount/occurredAt 有审计意义但不得改变重投身份。
func runtimeDLQID(deploymentID, consumerKey, subject, eventID string, reason FailureReasonCode, bodySHA256 string) (string, error) {
	return eventid.HashFields("runtime.dlq.event.v1", deploymentID, consumerKey, subject, eventID, string(reason), "sha256:"+bodySHA256)
}
