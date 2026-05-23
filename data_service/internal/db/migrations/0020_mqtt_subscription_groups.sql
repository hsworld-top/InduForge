CREATE TABLE IF NOT EXISTS data_mqtt_subscription_groups (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    name text NOT NULL CHECK (char_length(name) <= 100),
    parent_id uuid,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_mqtt_subscription_groups_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_mqtt_subscription_groups_parent_fkey
        FOREIGN KEY (parent_id) REFERENCES data_mqtt_subscription_groups (id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS data_mqtt_subscription_groups_root_name_key
    ON data_mqtt_subscription_groups (project_id, connection_id, name)
    WHERE parent_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS data_mqtt_subscription_groups_parent_name_key
    ON data_mqtt_subscription_groups (project_id, connection_id, parent_id, name)
    WHERE parent_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS data_mqtt_subscription_groups_project_connection_parent_idx
    ON data_mqtt_subscription_groups (project_id, connection_id, parent_id, created_at ASC);

ALTER TABLE data_mqtt_subscriptions
    ADD COLUMN IF NOT EXISTS group_id uuid;

ALTER TABLE data_mqtt_subscriptions
    ADD COLUMN IF NOT EXISTS display_order integer NOT NULL DEFAULT 0;

ALTER TABLE data_mqtt_subscriptions
    ADD CONSTRAINT data_mqtt_subscriptions_group_fkey
        FOREIGN KEY (group_id) REFERENCES data_mqtt_subscription_groups (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS data_mqtt_subscriptions_project_connection_group_idx
    ON data_mqtt_subscriptions (project_id, connection_id, group_id, display_order ASC, created_at ASC);
