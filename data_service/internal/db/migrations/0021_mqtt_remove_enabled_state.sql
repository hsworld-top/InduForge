DROP INDEX IF EXISTS data_mqtt_subscriptions_connection_enabled_idx;
DROP INDEX IF EXISTS data_mqtt_tags_subscription_enabled_idx;

ALTER TABLE data_mqtt_subscriptions
    DROP COLUMN IF EXISTS is_enabled;

ALTER TABLE data_mqtt_tags
    DROP COLUMN IF EXISTS is_enabled;

CREATE INDEX IF NOT EXISTS data_mqtt_subscriptions_connection_order_idx
    ON data_mqtt_subscriptions (connection_id, display_order, created_at);

CREATE INDEX IF NOT EXISTS data_mqtt_tags_subscription_order_idx
    ON data_mqtt_tags (subscription_id, display_order, created_at DESC);
