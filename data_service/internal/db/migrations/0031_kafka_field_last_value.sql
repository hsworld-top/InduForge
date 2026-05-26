ALTER TABLE data_kafka_fields
    ADD COLUMN IF NOT EXISTS last_value jsonb,
    ADD COLUMN IF NOT EXISTS quality text NOT NULL DEFAULT 'unknown' CHECK (quality IN ('good', 'bad', 'unknown')),
    ADD COLUMN IF NOT EXISTS last_updated_at timestamptz;
