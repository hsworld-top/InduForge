ALTER TABLE data_kafka_fields
    DROP COLUMN IF EXISTS last_updated_at,
    DROP COLUMN IF EXISTS quality,
    DROP COLUMN IF EXISTS last_value;
