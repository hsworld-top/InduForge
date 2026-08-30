-- RuntimeEngine PostgreSQL V1 的唯一从零基线。禁止通过启动逻辑修表或迁移。
CREATE SCHEMA runtime_engine;

CREATE TABLE runtime_engine.schema_meta (
    version text PRIMARY KEY CHECK (version = 'runtime_engine.v1'),
    applied_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO runtime_engine.schema_meta (version) VALUES ('runtime_engine.v1');

CREATE TABLE runtime_engine.role_fence (
    deployment_id text NOT NULL CHECK (length(deployment_id) > 0),
    role text NOT NULL CHECK (length(role) > 0),
    owner_id text NOT NULL CHECK (length(owner_id) > 0),
    epoch bigint NOT NULL CHECK (epoch >= 1),
    version bigint NOT NULL CHECK (version > 0),
    activated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (deployment_id, role)
);

CREATE TABLE runtime_engine.processed_event (
    deployment_id text NOT NULL CHECK (length(deployment_id) > 0),
    consumer_key text NOT NULL CHECK (length(consumer_key) > 0),
    event_id text NOT NULL CHECK (event_id ~ '^[0-9a-f]{64}$'),
    body_sha256 text NOT NULL CHECK (body_sha256 ~ '^[0-9a-f]{64}$'),
    subject text NOT NULL CHECK (length(subject) > 0 AND length(subject) <= 4096),
    delivery_metadata jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(delivery_metadata) = 'object' AND octet_length(delivery_metadata::text) <= 16384),
    processed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (deployment_id, consumer_key, event_id)
);

-- producer fence 与 consumer role fence 是刻意隔离的两个命名空间，二者不可比较或互相授权。
CREATE TABLE runtime_engine.producer_fence (
    deployment_id text NOT NULL CHECK (length(deployment_id) > 0),
    producer_key text NOT NULL CHECK (length(producer_key) > 0),
    owner_id text NOT NULL CHECK (length(owner_id) > 0),
    epoch bigint NOT NULL CHECK (epoch >= 1),
    version bigint NOT NULL CHECK (version > 0),
    activated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (deployment_id, producer_key)
);

CREATE TABLE runtime_engine.consumer_checkpoint (
    deployment_id text NOT NULL CHECK (length(deployment_id) > 0),
    consumer_key text NOT NULL CHECK (length(consumer_key) > 0),
    position bigint NOT NULL CHECK (position >= 0),
    version bigint NOT NULL CHECK (version > 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (deployment_id, consumer_key)
);

CREATE TABLE runtime_engine.transactional_outbox (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    deployment_id text NOT NULL CHECK (length(deployment_id) > 0),
    dedupe_key text NOT NULL CHECK (length(dedupe_key) > 0 AND length(dedupe_key) <= 512),
    subject text NOT NULL CHECK (length(subject) > 0 AND length(subject) <= 4096),
    headers jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(headers) = 'object' AND octet_length(headers::text) <= 16384),
    payload bytea NOT NULL CHECK (octet_length(payload) <= 2097152),
    payload_sha256 text NOT NULL CHECK (payload_sha256 ~ '^[0-9a-f]{64}$'),
    state text NOT NULL CHECK (state IN ('pending', 'leased', 'published')) DEFAULT 'pending',
    lease_token text,
    lease_owner text,
    lease_until timestamptz,
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    last_error_code text CHECK (last_error_code IS NULL OR last_error_code IN ('publish-error', 'publish-timeout', 'shutdown')),
    published_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (deployment_id, dedupe_key),
    CHECK ((state = 'leased') = (lease_token IS NOT NULL AND lease_owner IS NOT NULL AND lease_until IS NOT NULL)),
    CHECK ((state = 'published') = (published_at IS NOT NULL))
);
CREATE INDEX transactional_outbox_claim_idx ON runtime_engine.transactional_outbox
    (deployment_id, state, next_attempt_at, lease_until, id);

CREATE TABLE runtime_engine.processing_failure (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    deployment_id text NOT NULL CHECK (length(deployment_id) > 0),
    consumer_key text NOT NULL CHECK (length(consumer_key) > 0),
    dlq_id text NOT NULL CHECK (length(dlq_id) > 0 AND length(dlq_id) <= 512),
    event_id text CHECK (event_id IS NULL OR event_id ~ '^[0-9a-f]{64}$'),
    reason_code text NOT NULL CHECK (reason_code IN ('permanent-validation', 'event-id-collision', 'max-deliver', 'handler-failure')),
    delivery_count integer NOT NULL CHECK (delivery_count >= 1),
    jetstream_position bigint NOT NULL CHECK (jetstream_position >= 0),
    subject text NOT NULL CHECK (length(subject) > 0 AND length(subject) <= 4096),
    body_sha256 text NOT NULL CHECK (body_sha256 ~ '^[0-9a-f]{64}$'),
    raw_body bytea CHECK (raw_body IS NULL OR octet_length(raw_body) <= 1048576),
    occurred_at timestamptz NOT NULL,
    quarantined_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (deployment_id, dlq_id)
);
CREATE INDEX processing_failure_event_idx ON runtime_engine.processing_failure (deployment_id, consumer_key, event_id, quarantined_at DESC) WHERE event_id IS NOT NULL;

CREATE TABLE runtime_engine.producer_sequence (
    deployment_id text NOT NULL CHECK (length(deployment_id) > 0),
    producer_key text NOT NULL CHECK (length(producer_key) > 0),
    owner_id text NOT NULL CHECK (length(owner_id) > 0),
    epoch bigint NOT NULL CHECK (epoch >= 1),
    last_sequence bigint NOT NULL CHECK (last_sequence >= 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (deployment_id, producer_key)
);

-- 乱序历史可完整审计；当前值由后续 writer 以 owner/epoch/version CAS 决定。
CREATE TABLE runtime_engine.point_history (
    deployment_id text NOT NULL CHECK (deployment_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    point_id uuid NOT NULL,
    event_id text NOT NULL CHECK (event_id ~ '^[0-9a-f]{64}$'),
    owner_id text NOT NULL CHECK (owner_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    epoch bigint NOT NULL CHECK (epoch >= 1),
    sequence bigint NOT NULL CHECK (sequence >= 0),
    source_timestamp timestamptz NOT NULL,
    server_timestamp timestamptz NOT NULL,
    value jsonb CHECK (value IS NULL OR octet_length(value::text) <= 1048576),
    quality text NOT NULL CHECK (quality IN ('good', 'bad', 'unknown')),
    received_at timestamptz NOT NULL,
    PRIMARY KEY (deployment_id, point_id, event_id)
);
CREATE INDEX point_history_order_idx ON runtime_engine.point_history (deployment_id, point_id, epoch DESC, source_timestamp DESC, sequence DESC);

CREATE TABLE runtime_engine.point_current (
    deployment_id text NOT NULL CHECK (deployment_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    point_id uuid NOT NULL,
    owner_id text NOT NULL CHECK (owner_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    epoch bigint NOT NULL CHECK (epoch >= 1),
    source_timestamp timestamptz NOT NULL,
    server_timestamp timestamptz NOT NULL,
    sequence bigint NOT NULL CHECK (sequence >= 0),
    event_id text NOT NULL CHECK (event_id ~ '^[0-9a-f]{64}$'),
    value jsonb CHECK (value IS NULL OR octet_length(value::text) <= 1048576),
    quality text NOT NULL CHECK (quality IN ('good', 'bad', 'unknown')),
    version bigint NOT NULL CHECK (version > 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (deployment_id, point_id)
);

-- Compute role owns its own input projection.  It deliberately does not read
-- writer's projection: raw and derived consumers may arrive in either order.
CREATE TABLE runtime_engine.compute_input_snapshot (
    deployment_id text NOT NULL CHECK (deployment_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    compute_id uuid NOT NULL,
    datapoint_id uuid NOT NULL,
    event_id text NOT NULL CHECK (event_id ~ '^[0-9a-f]{64}$'),
    owner_id text NOT NULL CHECK (owner_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    epoch bigint NOT NULL CHECK (epoch >= 1),
    sequence bigint NOT NULL CHECK (sequence >= 0),
    source_timestamp timestamptz NOT NULL,
    server_timestamp timestamptz NOT NULL,
    received_at timestamptz NOT NULL,
    value jsonb CHECK (value IS NULL OR octet_length(value::text) <= 1048576),
    quality text NOT NULL CHECK (quality IN ('good', 'bad', 'unknown')),
    version bigint NOT NULL CHECK (version > 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (deployment_id, compute_id, datapoint_id)
);
CREATE INDEX compute_input_snapshot_event_idx ON runtime_engine.compute_input_snapshot
    (deployment_id, compute_id, event_id);

-- Change/condition debounce is durable.  Keeping last_value here means a
-- restart cannot turn an old value into a synthetic change edge.
CREATE TABLE runtime_engine.compute_trigger_state (
    deployment_id text NOT NULL CHECK (deployment_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    compute_id uuid NOT NULL,
	compute_revision bigint NOT NULL CHECK (compute_revision > 0),
    previous_value jsonb CHECK (previous_value IS NULL OR octet_length(previous_value::text) <= 1048576),
    previous_quality text CHECK (previous_quality IS NULL OR previous_quality IN ('good', 'bad', 'unknown')),
    previous_seen boolean NOT NULL DEFAULT false,
    condition_active boolean NOT NULL DEFAULT false,
    pending_phase text CHECK (pending_phase IS NULL OR pending_phase IN ('entered', 'exited')),
    pending_since timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (deployment_id, compute_id),
    CHECK ((pending_phase IS NULL) = (pending_since IS NULL))
);

-- Local wall-clock key freezes the DST policy: ambiguous fall-back times run
-- once (the earlier UTC instant); nonexistent spring-forward times are skipped.
CREATE TABLE runtime_engine.compute_schedule_state (
    deployment_id text NOT NULL CHECK (deployment_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    compute_id uuid NOT NULL,
	compute_revision bigint NOT NULL CHECK (compute_revision > 0),
    run_count bigint NOT NULL DEFAULT 0 CHECK (run_count >= 0),
    next_run_at timestamptz,
    last_run_at timestamptz,
    last_local_key text CHECK (last_local_key IS NULL OR length(last_local_key) BETWEEN 1 AND 256),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (deployment_id, compute_id),
    CHECK (last_run_at IS NULL OR last_local_key IS NOT NULL)
);
CREATE INDEX compute_schedule_due_idx ON runtime_engine.compute_schedule_state
    (deployment_id, next_run_at, compute_id);

-- Alarm role 维护自己的完整领域状态，不读取 writer 的 point_current 投影。
-- revision 变化时由领域处理器决定是否可安全重置；活动实例绝不在数据库层被静默覆盖。
CREATE TABLE runtime_engine.alarm_item_state (
    deployment_id text NOT NULL CHECK (deployment_id ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'),
    alarm_item_id uuid NOT NULL,
    alarm_revision bigint NOT NULL CHECK (alarm_revision > 0),
    state jsonb NOT NULL CHECK (jsonb_typeof(state) = 'object' AND octet_length(state::text) <= 1048576),
    version bigint NOT NULL CHECK (version > 0),
    next_evaluation_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (deployment_id, alarm_item_id)
);
CREATE INDEX alarm_item_state_due_idx ON runtime_engine.alarm_item_state
    (deployment_id, next_evaluation_at, alarm_item_id);
