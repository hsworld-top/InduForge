ALTER TABLE data_mqtt_tags
    DROP CONSTRAINT IF EXISTS data_mqtt_tags_project_code_key;

ALTER TABLE data_mqtt_tags
    ADD CONSTRAINT data_mqtt_tags_project_subscription_code_key
    UNIQUE (project_id, subscription_id, code);
