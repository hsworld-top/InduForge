-- data_mqtt_tags: 保存 MQTT 变量定义。
CREATE TABLE IF NOT EXISTS data_mqtt_tags (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    subscription_id uuid NOT NULL,
    name text NOT NULL CHECK (char_length(name) <= 100),
    code text NOT NULL CHECK (char_length(code) <= 100),
    description text,
    data_type text NOT NULL DEFAULT 'string' CHECK (data_type IN ('string', 'number', 'boolean', 'object', 'array')),
    parse_type text NOT NULL DEFAULT 'jsonpath' CHECK (parse_type IN ('jsonpath', 'regex', 'script', 'fixed')),
    parse_rule text NOT NULL,
    default_value text,
    unit text CHECK (unit IS NULL OR char_length(unit) <= 50),
    transform text,
    validation jsonb CHECK (validation IS NULL OR jsonb_typeof(validation) = 'object'),
    display_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_mqtt_tags_project_code_key UNIQUE (project_id, code),
    CONSTRAINT data_mqtt_tags_subscription_fkey
        FOREIGN KEY (subscription_id) REFERENCES data_mqtt_subscriptions (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_mqtt_tags_subscription_order_idx
    ON data_mqtt_tags (subscription_id, display_order, created_at DESC);

CREATE INDEX IF NOT EXISTS data_mqtt_tags_project_idx
    ON data_mqtt_tags (project_id, display_order, created_at DESC);

CREATE INDEX IF NOT EXISTS data_mqtt_tags_validation_gin_idx
    ON data_mqtt_tags USING GIN (validation);
