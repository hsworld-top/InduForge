ALTER TABLE data_kafka_topic_mappings
    DROP CONSTRAINT IF EXISTS data_kafka_topic_mappings_name_key,
    ADD CONSTRAINT data_kafka_topic_mappings_topic_key
        UNIQUE (project_id, connection_id, topic);

ALTER TABLE data_kafka_topic_mappings
    DROP COLUMN IF EXISTS raw_output_scope,
    DROP COLUMN IF EXISTS output_mode;
