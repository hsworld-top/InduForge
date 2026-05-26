DROP INDEX IF EXISTS data_kafka_fields_group_idx;
DROP INDEX IF EXISTS data_kafka_field_groups_tree_idx;

ALTER TABLE data_kafka_fields
    DROP CONSTRAINT IF EXISTS data_kafka_fields_group_fkey,
    DROP COLUMN IF EXISTS group_id;

DROP TABLE IF EXISTS data_kafka_field_groups;
