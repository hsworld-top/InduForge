ALTER TABLE data_mqtt_subscriptions
    DROP CONSTRAINT IF EXISTS data_mqtt_subscriptions_project_name_key;

ALTER TABLE data_mqtt_subscriptions
    ADD CONSTRAINT data_mqtt_subscriptions_project_connection_name_key
    UNIQUE (project_id, connection_id, name);
