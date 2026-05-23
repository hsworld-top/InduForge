DROP INDEX IF EXISTS data_mqtt_tags_subscription_order_idx;
DROP INDEX IF EXISTS data_mqtt_subscriptions_connection_order_idx;

ALTER TABLE data_mqtt_subscriptions
    ADD COLUMN IF NOT EXISTS is_enabled boolean NOT NULL DEFAULT true;

ALTER TABLE data_mqtt_tags
    ADD COLUMN IF NOT EXISTS is_enabled boolean NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS data_mqtt_subscriptions_connection_enabled_idx
    ON data_mqtt_subscriptions (connection_id, is_enabled);

CREATE INDEX IF NOT EXISTS data_mqtt_tags_subscription_enabled_idx
    ON data_mqtt_tags (subscription_id, is_enabled, display_order, created_at DESC);
