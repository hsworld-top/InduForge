ALTER TABLE data_mqtt_subscriptions
    ADD COLUMN IF NOT EXISTS usage_mode text NOT NULL DEFAULT 'single_variable';

ALTER TABLE data_mqtt_subscriptions
    DROP CONSTRAINT IF EXISTS data_mqtt_subscriptions_usage_mode_check;

ALTER TABLE data_mqtt_subscriptions
    ADD CONSTRAINT data_mqtt_subscriptions_usage_mode_check
        CHECK (usage_mode IN ('raw_datapoint', 'single_variable', 'batch_variable'));
