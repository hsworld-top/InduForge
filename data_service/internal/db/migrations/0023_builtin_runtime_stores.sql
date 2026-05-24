-- 扩展 data_connections 支持 IF 内置运行库。内置运行库允许同项目同类型多实例。
ALTER TABLE data_connections DROP CONSTRAINT IF EXISTS data_connections_type_check;
ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_type_check
    CHECK (type IN (
        'relational', 'mqtt', 'websocket', 'opcua', 'modbus', 'http', 's7',
        'kafka', 'redis', 'tdengine',
        'builtin.relation', 'builtin.timeseries', 'builtin.realtime', 'builtin.message'
    ));

ALTER TABLE data_connections DROP CONSTRAINT IF EXISTS data_connections_category_check;
ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_category_check
    CHECK (category IN ('database', 'message', 'protocol', 'api', 'builtin'));

DROP INDEX IF EXISTS data_connections_project_builtin_type_unique;

CREATE TABLE IF NOT EXISTS data_builtin_realtime_keys (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    key_path text NOT NULL,
    value_type text NOT NULL DEFAULT 'json',
    default_ttl_seconds integer NOT NULL DEFAULT 300,
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, connection_id, key_path),
    FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS data_builtin_message_topics (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    topic text NOT NULL,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, connection_id, topic),
    FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS data_builtin_message_variables (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    topic_id uuid NOT NULL,
    name text NOT NULL,
    payload_path text NOT NULL,
    value_type text NOT NULL DEFAULT 'string',
    unit text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    create_datapoint boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, topic_id, name),
    FOREIGN KEY (topic_id) REFERENCES data_builtin_message_topics (id) ON DELETE CASCADE
);
