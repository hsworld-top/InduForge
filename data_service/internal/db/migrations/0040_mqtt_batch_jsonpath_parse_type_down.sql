ALTER TABLE data_mqtt_tags
    DROP CONSTRAINT IF EXISTS data_mqtt_tags_parse_type_check,
    ADD CONSTRAINT data_mqtt_tags_parse_type_check
        CHECK (parse_type IN ('jsonpath', 'regex', 'script', 'fixed'));
