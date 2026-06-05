ALTER TABLE data_kafka_topic_mappings
    DROP CONSTRAINT IF EXISTS data_kafka_topic_mappings_offset_check,
    DROP COLUMN IF EXISTS start_offset,
    DROP COLUMN IF EXISTS consumer_group;
