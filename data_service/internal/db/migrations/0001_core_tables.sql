CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- data_connections: 保存连接基础元数据与运行状态。
CREATE TABLE IF NOT EXISTS data_connections (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    name text NOT NULL CHECK (char_length(name) <= 100),
    type text NOT NULL CHECK (type IN ('relational', 'mqtt', 'websocket', 'opcua', 'modbus', 'http', 's7', 'kafka', 'redis', 'tdengine', 'builtin.relation', 'builtin.timeseries', 'builtin.realtime', 'builtin.message')),
    category text NOT NULL DEFAULT 'api' CHECK (category IN ('database', 'message', 'protocol', 'api', 'builtin')),
    status text NOT NULL DEFAULT 'unknown' CHECK (status IN ('connected', 'disconnected', 'error', 'unknown')),
    is_enabled boolean NOT NULL DEFAULT true,
    retry_count integer NOT NULL DEFAULT 3 CHECK (retry_count >= 0),
    retry_interval_ms integer NOT NULL DEFAULT 5000 CHECK (retry_interval_ms >= 0),
    health_check_interval_ms integer NOT NULL DEFAULT 30000 CHECK (health_check_interval_ms >= 0),
    last_connected_at timestamptz,
    last_error_message text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
    display_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS data_connections_project_type_idx
    ON data_connections (project_id, type);

CREATE INDEX IF NOT EXISTS data_connections_project_status_idx
    ON data_connections (project_id, status);

CREATE INDEX IF NOT EXISTS data_connections_project_order_idx
    ON data_connections (project_id, display_order, created_at DESC);

-- data_relational_configs: 保存关系型数据库连接配置。
CREATE TABLE IF NOT EXISTS data_relational_configs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id uuid NOT NULL,
    db_type text NOT NULL CHECK (db_type IN ('mysql', 'postgresql', 'sqlserver', 'oracle', 'sqlite', 'clickhouse')),
    host text NOT NULL CHECK (char_length(host) <= 255),
    port integer NOT NULL CHECK (port > 0 AND port <= 65535),
    database text NOT NULL CHECK (char_length(database) <= 100),
    username text NOT NULL CHECK (char_length(username) <= 100),
    password text NOT NULL,
    schema text,
    charset text NOT NULL DEFAULT 'utf8mb4',
    timezone text,
    ssl boolean NOT NULL DEFAULT false,
    ssl_config jsonb CHECK (ssl_config IS NULL OR jsonb_typeof(ssl_config) = 'object'),
    pool_min integer NOT NULL DEFAULT 2 CHECK (pool_min >= 0),
    pool_max integer NOT NULL DEFAULT 10 CHECK (pool_max > 0 AND pool_max >= pool_min),
    acquire_timeout_ms integer NOT NULL DEFAULT 60000 CHECK (acquire_timeout_ms >= 0),
    idle_timeout_ms integer NOT NULL DEFAULT 30000 CHECK (idle_timeout_ms >= 0),
    query_timeout_ms integer NOT NULL DEFAULT 60000 CHECK (query_timeout_ms >= 0),
    options jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(options) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_relational_configs_connection_id_key UNIQUE (connection_id),
    CONSTRAINT data_relational_configs_connection_id_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_relational_configs_db_type_idx
    ON data_relational_configs (db_type);

CREATE INDEX IF NOT EXISTS data_relational_configs_ssl_config_gin_idx
    ON data_relational_configs USING GIN (ssl_config);

-- data_queries: 保存数据查询定义与执行配置。
CREATE TABLE IF NOT EXISTS data_queries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 200),
    description text,
    category text,
    query_type text NOT NULL CHECK (query_type IN ('sql', 'tags', 'http', 'mqtt_pub', 'mqtt_sub')),
    config jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(config) = 'object'),
    transformer text,
    is_enabled boolean NOT NULL DEFAULT true,
    timeout_ms integer NOT NULL DEFAULT 30000 CHECK (timeout_ms >= 0),
    cache_enabled boolean NOT NULL DEFAULT false,
    cache_ttl_seconds integer NOT NULL DEFAULT 300 CHECK (cache_ttl_seconds >= 0),
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_queries_connection_id_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_queries_project_name_key UNIQUE (project_id, name)
);

CREATE INDEX IF NOT EXISTS data_queries_project_connection_idx
    ON data_queries (project_id, connection_id);

CREATE INDEX IF NOT EXISTS data_queries_connection_project_idx
    ON data_queries (connection_id, project_id);

CREATE INDEX IF NOT EXISTS data_queries_type_enabled_idx
    ON data_queries (query_type, is_enabled);

CREATE INDEX IF NOT EXISTS data_queries_config_gin_idx
    ON data_queries USING GIN (config);

-- data_points: 保存设计器与数据服务可共享的数据点定义。
CREATE TABLE IF NOT EXISTS data_points (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    path text NOT NULL CHECK (char_length(path) <= 255),
    name text NOT NULL CHECK (char_length(name) <= 100),
    description text,
    source_type text NOT NULL CHECK (char_length(source_type) <= 50),
    source_id uuid,
    source_config jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(source_config) = 'object'),
    data_type text NOT NULL CHECK (char_length(data_type) <= 20),
    unit text CHECK (unit IS NULL OR char_length(unit) <= 20),
    precision_num integer CHECK (precision_num IS NULL OR precision_num >= 0),
    default_value text,
    min_value numeric(20, 6),
    max_value numeric(20, 6),
    alarm_low numeric(20, 6),
    alarm_high numeric(20, 6),
    tags jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(tags) = 'array'),
    refresh_mode text NOT NULL DEFAULT 'auto' CHECK (refresh_mode IN ('auto', 'manual', 'subscription')),
    refresh_interval_ms integer CHECK (refresh_interval_ms IS NULL OR refresh_interval_ms >= 0),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'invalid')),
    display_order integer NOT NULL DEFAULT 0,
    runtime_permissions jsonb NOT NULL DEFAULT '{"write":{"allowRoles":[],"denyRoles":[],"inherit":true}}'::jsonb
        CHECK (jsonb_typeof(runtime_permissions) = 'object'),
    created_by uuid,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_points_project_path_key UNIQUE (project_id, path),
    CONSTRAINT data_points_min_max_check CHECK (min_value IS NULL OR max_value IS NULL OR min_value <= max_value),
    CONSTRAINT data_points_alarm_range_check CHECK (
        (alarm_low IS NULL OR alarm_high IS NULL OR alarm_low <= alarm_high)
        AND (min_value IS NULL OR alarm_low IS NULL OR min_value <= alarm_low)
        AND (max_value IS NULL OR alarm_high IS NULL OR alarm_high <= max_value)
    )
);

CREATE INDEX IF NOT EXISTS data_points_project_status_idx
    ON data_points (project_id, status);

CREATE INDEX IF NOT EXISTS data_points_source_idx
    ON data_points (source_type, source_id);

CREATE INDEX IF NOT EXISTS data_points_source_config_gin_idx
    ON data_points USING GIN (source_config);

CREATE INDEX IF NOT EXISTS data_points_tags_gin_idx
    ON data_points USING GIN (tags);

CREATE INDEX IF NOT EXISTS data_points_project_created_display_idx
    ON data_points (project_id, created_at DESC, display_order ASC, id DESC);
