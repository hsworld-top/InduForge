ALTER TABLE data_mqtt_subscriptions
    DROP CONSTRAINT IF EXISTS data_mqtt_subscriptions_usage_mode_check;

ALTER TABLE data_mqtt_subscriptions
    DROP COLUMN IF EXISTS usage_mode;
