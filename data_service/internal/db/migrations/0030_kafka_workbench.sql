CREATE TABLE IF NOT EXISTS data_kafka_topic_groups (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    parent_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_kafka_topic_groups_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_kafka_topic_groups_parent_fkey
        FOREIGN KEY (parent_id) REFERENCES data_kafka_topic_groups (id) ON DELETE SET NULL,
    CONSTRAINT data_kafka_topic_groups_name_key
        UNIQUE (project_id, connection_id, parent_id, name)
);

CREATE TABLE IF NOT EXISTS data_kafka_topic_mappings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    topic text NOT NULL CHECK (char_length(topic) <= 500),
    description text NOT NULL DEFAULT '',
    partition_mode text NOT NULL DEFAULT 'all' CHECK (partition_mode IN ('all', 'single')),
    partition integer CHECK (partition IS NULL OR partition >= 0),
    start_position text NOT NULL DEFAULT 'latest' CHECK (start_position IN ('latest', 'earliest', 'offset')),
    decode text NOT NULL DEFAULT 'json' CHECK (decode IN ('json', 'string', 'binary')),
    sample_limit integer NOT NULL DEFAULT 100 CHECK (sample_limit >= 1 AND sample_limit <= 1000),
    timeout_ms integer NOT NULL DEFAULT 5000 CHECK (timeout_ms >= 1000 AND timeout_ms <= 30000),
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_kafka_topic_mappings_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_kafka_topic_mappings_group_fkey
        FOREIGN KEY (group_id) REFERENCES data_kafka_topic_groups (id) ON DELETE SET NULL,
    CONSTRAINT data_kafka_topic_mappings_single_partition_check
        CHECK (partition_mode <> 'single' OR partition IS NOT NULL),
    CONSTRAINT data_kafka_topic_mappings_topic_key
        UNIQUE (project_id, connection_id, topic)
);

CREATE TABLE IF NOT EXISTS data_kafka_fields (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    topic_mapping_id uuid NOT NULL,
    name text NOT NULL CHECK (char_length(name) <= 100),
    value_path text NOT NULL CHECK (char_length(value_path) <= 500),
    key_path text NOT NULL DEFAULT '' CHECK (char_length(key_path) <= 500),
    data_type text NOT NULL CHECK (char_length(data_type) <= 50),
    enabled boolean NOT NULL DEFAULT true,
    description text NOT NULL DEFAULT '',
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_kafka_fields_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_kafka_fields_mapping_fkey
        FOREIGN KEY (topic_mapping_id) REFERENCES data_kafka_topic_mappings (id) ON DELETE CASCADE,
    CONSTRAINT data_kafka_fields_value_path_key
        UNIQUE (project_id, topic_mapping_id, value_path)
);

CREATE INDEX IF NOT EXISTS data_kafka_topic_groups_tree_idx
    ON data_kafka_topic_groups (project_id, connection_id, parent_id, sort_order, created_at);

CREATE INDEX IF NOT EXISTS data_kafka_topic_mappings_group_idx
    ON data_kafka_topic_mappings (project_id, connection_id, group_id, sort_order, created_at);

CREATE INDEX IF NOT EXISTS data_kafka_fields_connection_idx
    ON data_kafka_fields (project_id, connection_id);
