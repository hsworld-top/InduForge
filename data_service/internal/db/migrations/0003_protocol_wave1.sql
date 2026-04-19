-- 扩展 data_connections 支持 kafka / redis 两类连接类型。
ALTER TABLE data_connections DROP CONSTRAINT IF EXISTS data_connections_type_check;
ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_type_check
    CHECK (type IN ('relational', 'mqtt', 'websocket', 'opcua', 'modbus', 'http', 's7', 'kafka', 'redis'));

-- data_kafka_configs: 保存 Kafka Source 配置（与连接一对一）。
CREATE TABLE IF NOT EXISTS data_kafka_configs (
    connection_id uuid PRIMARY KEY,
    brokers text NOT NULL CHECK (char_length(brokers) <= 500),
    topic text NOT NULL CHECK (char_length(topic) <= 500),
    consumer_group text NOT NULL CHECK (char_length(consumer_group) <= 200),
    start_position text NOT NULL DEFAULT 'latest' CHECK (start_position IN ('latest', 'earliest')),
    options jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(options) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_kafka_configs_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_kafka_configs_topic_idx
    ON data_kafka_configs (topic);

-- data_http_configs: 保存 HTTP Source 配置（与连接一对一）。
CREATE TABLE IF NOT EXISTS data_http_configs (
    connection_id uuid PRIMARY KEY,
    base_url text NOT NULL CHECK (char_length(base_url) <= 1000),
    method text NOT NULL DEFAULT 'GET' CHECK (method IN ('GET', 'POST', 'PUT', 'PATCH', 'DELETE')),
    headers jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(headers) = 'object'),
    timeout_ms integer NOT NULL DEFAULT 30000 CHECK (timeout_ms >= 0),
    body_template jsonb CHECK (body_template IS NULL OR jsonb_typeof(body_template) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_http_configs_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_http_configs_method_idx
    ON data_http_configs (method);

-- data_websocket_configs: 保存 WebSocket Source 配置（与连接一对一）。
CREATE TABLE IF NOT EXISTS data_websocket_configs (
    connection_id uuid PRIMARY KEY,
    url text NOT NULL CHECK (char_length(url) <= 1000),
    topic text CHECK (topic IS NULL OR char_length(topic) <= 500),
    headers jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(headers) = 'object'),
    heartbeat_interval_ms integer NOT NULL DEFAULT 30000 CHECK (heartbeat_interval_ms >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_websocket_configs_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_websocket_configs_url_idx
    ON data_websocket_configs (url);

-- data_redis_configs: 保存 Redis Source 配置（与连接一对一）。
CREATE TABLE IF NOT EXISTS data_redis_configs (
    connection_id uuid PRIMARY KEY,
    address text NOT NULL CHECK (char_length(address) <= 255),
    db integer NOT NULL DEFAULT 0 CHECK (db >= 0),
    username text CHECK (username IS NULL OR char_length(username) <= 100),
    password text,
    key_pattern text NOT NULL DEFAULT '*' CHECK (char_length(key_pattern) <= 255),
    mode text NOT NULL DEFAULT 'standalone' CHECK (mode IN ('standalone', 'sentinel', 'cluster')),
    options jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(options) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_redis_configs_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_redis_configs_mode_idx
    ON data_redis_configs (mode);
