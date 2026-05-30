CREATE TABLE IF NOT EXISTS data_kafka_field_groups (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    topic_mapping_id uuid NOT NULL,
    parent_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    description text NOT NULL DEFAULT '',
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_kafka_field_groups_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_kafka_field_groups_mapping_fkey
        FOREIGN KEY (topic_mapping_id) REFERENCES data_kafka_topic_mappings (id) ON DELETE CASCADE,
    CONSTRAINT data_kafka_field_groups_parent_fkey
        FOREIGN KEY (parent_id) REFERENCES data_kafka_field_groups (id) ON DELETE SET NULL,
    CONSTRAINT data_kafka_field_groups_name_key
        UNIQUE (project_id, topic_mapping_id, parent_id, name)
);

ALTER TABLE data_kafka_fields
    ADD COLUMN IF NOT EXISTS group_id uuid;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'data_kafka_fields_group_fkey'
    ) THEN
        ALTER TABLE data_kafka_fields
            ADD CONSTRAINT data_kafka_fields_group_fkey
            FOREIGN KEY (group_id) REFERENCES data_kafka_field_groups (id) ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS data_kafka_field_groups_tree_idx
    ON data_kafka_field_groups (project_id, topic_mapping_id, parent_id, sort_order, created_at);

CREATE UNIQUE INDEX IF NOT EXISTS data_kafka_field_groups_root_name_key
    ON data_kafka_field_groups (project_id, topic_mapping_id, name)
    WHERE parent_id IS NULL;

CREATE INDEX IF NOT EXISTS data_kafka_fields_group_idx
    ON data_kafka_fields (project_id, topic_mapping_id, group_id, sort_order, created_at);
