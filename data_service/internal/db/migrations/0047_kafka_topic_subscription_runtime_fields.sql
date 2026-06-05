ALTER TABLE data_kafka_topic_mappings
    ADD COLUMN IF NOT EXISTS consumer_group text NOT NULL DEFAULT '' CHECK (char_length(consumer_group) <= 200),
    ADD COLUMN IF NOT EXISTS start_offset bigint;

ALTER TABLE data_kafka_topic_mappings
    DROP CONSTRAINT IF EXISTS data_kafka_topic_mappings_offset_check,
    ADD CONSTRAINT data_kafka_topic_mappings_offset_check
        CHECK (start_position <> 'offset' OR (partition_mode = 'single' AND partition IS NOT NULL AND start_offset IS NOT NULL));
