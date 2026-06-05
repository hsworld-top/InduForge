ALTER TABLE data_kafka_topic_mappings
    ADD COLUMN IF NOT EXISTS output_mode text NOT NULL DEFAULT 'field_mapping' CHECK (output_mode IN ('raw_message', 'field_mapping')),
    ADD COLUMN IF NOT EXISTS raw_output_scope text NOT NULL DEFAULT 'value' CHECK (raw_output_scope IN ('value', 'full_message'));

ALTER TABLE data_kafka_topic_mappings
    DROP CONSTRAINT IF EXISTS data_kafka_topic_mappings_topic_key,
    DROP CONSTRAINT IF EXISTS data_kafka_topic_mappings_name_key,
    ADD CONSTRAINT data_kafka_topic_mappings_name_key
        UNIQUE (project_id, connection_id, name);
