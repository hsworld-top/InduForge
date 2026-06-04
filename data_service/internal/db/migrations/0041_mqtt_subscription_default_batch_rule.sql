ALTER TABLE data_mqtt_subscriptions
    ADD COLUMN IF NOT EXISTS default_batch_parse_rule jsonb;
