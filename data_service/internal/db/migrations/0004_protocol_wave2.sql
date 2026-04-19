-- 扩展 data_connections 支持 tdengine 连接类型。
ALTER TABLE data_connections DROP CONSTRAINT IF EXISTS data_connections_type_check;
ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_type_check
    CHECK (type IN ('relational', 'mqtt', 'websocket', 'opcua', 'modbus', 'http', 's7', 'kafka', 'redis', 'tdengine'));

-- data_opcua_configs: 保存 OPC UA Source 配置。
CREATE TABLE IF NOT EXISTS data_opcua_configs (
    connection_id uuid PRIMARY KEY,
    endpoint text NOT NULL CHECK (char_length(endpoint) <= 1000),
    security_policy text NOT NULL DEFAULT 'None',
    security_mode text NOT NULL DEFAULT 'None' CHECK (security_mode IN ('None', 'Sign', 'SignAndEncrypt')),
    auth_type text NOT NULL DEFAULT 'anonymous' CHECK (auth_type IN ('anonymous', 'username_password')),
    username text CHECK (username IS NULL OR char_length(username) <= 100),
    password text,
    sampling_ms integer NOT NULL DEFAULT 1000 CHECK (sampling_ms > 0),
    options jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(options) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_opcua_configs_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_opcua_configs_endpoint_idx
    ON data_opcua_configs (endpoint);

-- data_s7_configs: 保存西门子 S7 Source 配置。
CREATE TABLE IF NOT EXISTS data_s7_configs (
    connection_id uuid PRIMARY KEY,
    host text NOT NULL CHECK (char_length(host) <= 255),
    port integer NOT NULL DEFAULT 102 CHECK (port > 0 AND port <= 65535),
    rack integer NOT NULL DEFAULT 0 CHECK (rack >= 0),
    slot integer NOT NULL DEFAULT 1 CHECK (slot >= 0),
    poll_interval_ms integer NOT NULL DEFAULT 1000 CHECK (poll_interval_ms > 0),
    options jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(options) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_s7_configs_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_s7_configs_host_idx
    ON data_s7_configs (host);

-- data_modbus_configs: 保存 Modbus Source 配置。
CREATE TABLE IF NOT EXISTS data_modbus_configs (
    connection_id uuid PRIMARY KEY,
    mode text NOT NULL DEFAULT 'tcp' CHECK (mode IN ('tcp', 'rtu')),
    host text CHECK (host IS NULL OR char_length(host) <= 255),
    port integer CHECK (port IS NULL OR (port > 0 AND port <= 65535)),
    serial_config jsonb CHECK (serial_config IS NULL OR jsonb_typeof(serial_config) = 'object'),
    slave_id integer NOT NULL DEFAULT 1 CHECK (slave_id >= 0),
    start_address integer NOT NULL DEFAULT 0 CHECK (start_address >= 0),
    quantity integer NOT NULL DEFAULT 1 CHECK (quantity > 0),
    poll_interval_ms integer NOT NULL DEFAULT 1000 CHECK (poll_interval_ms > 0),
    options jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(options) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_modbus_configs_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_modbus_configs_mode_idx
    ON data_modbus_configs (mode);

-- data_tdengine_configs: 保存 TDengine Source 配置。
CREATE TABLE IF NOT EXISTS data_tdengine_configs (
    connection_id uuid PRIMARY KEY,
    dsn text NOT NULL CHECK (char_length(dsn) <= 1000),
    database_name text NOT NULL CHECK (char_length(database_name) <= 128),
    timezone text,
    options jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(options) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_tdengine_configs_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_tdengine_configs_database_idx
    ON data_tdengine_configs (database_name);
