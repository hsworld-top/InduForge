DROP INDEX IF EXISTS data_mqtt_tags_group_idx;

ALTER TABLE data_mqtt_tags
    DROP CONSTRAINT IF EXISTS data_mqtt_tags_group_fkey,
    DROP COLUMN IF EXISTS group_id;

DROP TABLE IF EXISTS data_mqtt_tag_groups;
