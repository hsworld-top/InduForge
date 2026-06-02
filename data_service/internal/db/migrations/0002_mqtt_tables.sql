-- data_mqtt_configs: 保存 MQTT 协议连接配置（与 data_connections 一对一）。
CREATE TABLE IF NOT EXISTS data_mqtt_configs (
    connection_id uuid PRIMARY KEY,
    broker_url text NOT NULL CHECK (char_length(broker_url) <= 500),
    protocol text NOT NULL DEFAULT 'mqtt' CHECK (protocol IN ('mqtt', 'mqtts', 'ws', 'wss')),
    port integer NOT NULL DEFAULT 1883 CHECK (port > 0 AND port <= 65535),
    client_id text CHECK (client_id IS NULL OR char_length(client_id) <= 100),
    username text CHECK (username IS NULL OR char_length(username) <= 100),
    password text,
    keepalive integer NOT NULL DEFAULT 60 CHECK (keepalive >= 0),
    clean_session boolean NOT NULL DEFAULT true,
    qos smallint NOT NULL DEFAULT 0 CHECK (qos IN (0, 1, 2)),
    reconnect_period_ms integer NOT NULL DEFAULT 5000 CHECK (reconnect_period_ms >= 0),
    connect_timeout_ms integer NOT NULL DEFAULT 30000 CHECK (connect_timeout_ms >= 0),
    will jsonb CHECK (will IS NULL OR jsonb_typeof(will) = 'object'),
    ssl_config jsonb CHECK (ssl_config IS NULL OR jsonb_typeof(ssl_config) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_mqtt_configs_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_mqtt_configs_protocol_idx
    ON data_mqtt_configs (protocol);

-- data_mqtt_subscriptions: 保存 MQTT 订阅定义。
CREATE TABLE IF NOT EXISTS data_mqtt_subscriptions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    name text NOT NULL CHECK (char_length(name) <= 100),
    topic text NOT NULL CHECK (char_length(topic) <= 500),
    qos smallint NOT NULL DEFAULT 0 CHECK (qos IN (0, 1, 2)),
    usage_mode text NOT NULL DEFAULT 'single_variable'
        CHECK (usage_mode IN ('raw_datapoint', 'single_variable', 'batch_variable')),
    group_id uuid,
    description text,
    display_order integer NOT NULL DEFAULT 0,
    message_retention integer NOT NULL DEFAULT 5000 CHECK (message_retention > 0),
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_mqtt_subscriptions_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_mqtt_subscriptions_project_connection_name_key UNIQUE (project_id, connection_id, name)
);

CREATE INDEX IF NOT EXISTS data_mqtt_subscriptions_project_connection_idx
    ON data_mqtt_subscriptions (project_id, connection_id);

CREATE INDEX IF NOT EXISTS data_mqtt_subscriptions_connection_order_idx
    ON data_mqtt_subscriptions (connection_id, display_order, created_at DESC);

-- data_mqtt_messages: 保存订阅实时消息缓存。
CREATE TABLE IF NOT EXISTS data_mqtt_messages (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    subscription_id uuid NOT NULL,
    topic text NOT NULL CHECK (char_length(topic) <= 500),
    payload text NOT NULL,
    qos smallint NOT NULL DEFAULT 0 CHECK (qos IN (0, 1, 2)),
    received_at timestamptz NOT NULL DEFAULT now(),
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
    CONSTRAINT data_mqtt_messages_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_mqtt_messages_subscription_fkey
        FOREIGN KEY (subscription_id) REFERENCES data_mqtt_subscriptions (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_mqtt_messages_project_subscription_received_idx
    ON data_mqtt_messages (project_id, subscription_id, received_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS data_mqtt_messages_subscription_received_idx
    ON data_mqtt_messages (subscription_id, received_at DESC, id DESC);
