CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE data_connections
    DROP CONSTRAINT IF EXISTS data_connections_type_check;

ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_type_check
        CHECK (type IN ('relational', 'mqtt', 'websocket', 'opcua', 'modbus', 'http', 's7', 'kafka', 'redis', 'tdengine', 'builtin.relation', 'builtin.timeseries', 'builtin.realtime', 'builtin.message', 'collector'));

ALTER TABLE data_connections
    DROP CONSTRAINT IF EXISTS data_connections_category_check;

ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_category_check
        CHECK (category IN ('database', 'message', 'protocol', 'api', 'builtin', 'industrial'));

ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_id_project_key UNIQUE (id, project_id);

CREATE TABLE IF NOT EXISTS data_collector_connections (
    connection_id uuid PRIMARY KEY,
    project_id uuid NOT NULL,
    protocol_family text NOT NULL,
    driver_id text NOT NULL,
    driver_version text NOT NULL,
    schema_version integer NOT NULL,
    config jsonb NOT NULL DEFAULT '{}'::jsonb,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_collector_connections_identity_key UNIQUE (connection_id, project_id),
    CONSTRAINT data_collector_connections_connection_fkey
        FOREIGN KEY (connection_id, project_id) REFERENCES data_connections (id, project_id) ON DELETE CASCADE,
    CONSTRAINT data_collector_connections_protocol_family_check
        CHECK (protocol_family = lower(protocol_family) AND protocol_family ~ '^[a-z0-9][a-z0-9-]*$'),
    CONSTRAINT data_collector_connections_driver_id_check
        CHECK (driver_id = lower(driver_id) AND driver_id ~ '^[a-z0-9][a-z0-9.-]*$'),
    CONSTRAINT data_collector_connections_driver_version_check CHECK (char_length(driver_version) BETWEEN 1 AND 50),
    CONSTRAINT data_collector_connections_schema_version_check CHECK (schema_version > 0),
    CONSTRAINT data_collector_connections_config_check CHECK (jsonb_typeof(config) = 'object'),
    CONSTRAINT data_collector_connections_metadata_check CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE INDEX IF NOT EXISTS data_collector_connections_project_driver_idx
    ON data_collector_connections (project_id, protocol_family, driver_id, created_at DESC);

CREATE INDEX IF NOT EXISTS data_collector_connections_config_gin_idx
    ON data_collector_connections USING GIN (config);

CREATE TABLE IF NOT EXISTS data_collector_connection_secrets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id uuid NOT NULL,
    secret_key text NOT NULL,
    encrypted_value bytea NOT NULL,
    encryption_key_version text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_collector_connection_secrets_identity_key UNIQUE (connection_id, secret_key),
    CONSTRAINT data_collector_connection_secrets_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_collector_connections (connection_id) ON DELETE CASCADE,
    CONSTRAINT data_collector_connection_secrets_key_check
        CHECK (secret_key ~ '^[A-Za-z][A-Za-z0-9_.-]*$' AND char_length(secret_key) <= 100),
    CONSTRAINT data_collector_connection_secrets_value_check CHECK (octet_length(encrypted_value) > 0),
    CONSTRAINT data_collector_connection_secrets_version_check CHECK (char_length(encryption_key_version) BETWEEN 1 AND 50)
);

CREATE INDEX IF NOT EXISTS data_collector_connection_secrets_connection_idx
    ON data_collector_connection_secrets (connection_id);

CREATE TABLE IF NOT EXISTS data_collector_point_groups (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    parent_id uuid,
    name text NOT NULL,
    sort_order integer NOT NULL DEFAULT 0,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_collector_point_groups_identity_key UNIQUE (id, project_id, connection_id),
    CONSTRAINT data_collector_point_groups_connection_fkey
        FOREIGN KEY (connection_id, project_id) REFERENCES data_collector_connections (connection_id, project_id) ON DELETE CASCADE,
    CONSTRAINT data_collector_point_groups_parent_fkey
        FOREIGN KEY (parent_id, project_id, connection_id)
        REFERENCES data_collector_point_groups (id, project_id, connection_id) ON DELETE CASCADE,
    CONSTRAINT data_collector_point_groups_name_check CHECK (char_length(trim(name)) BETWEEN 1 AND 100),
    CONSTRAINT data_collector_point_groups_metadata_check CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE UNIQUE INDEX IF NOT EXISTS data_collector_point_groups_root_name_key
    ON data_collector_point_groups (project_id, connection_id, name)
    WHERE parent_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS data_collector_point_groups_parent_name_key
    ON data_collector_point_groups (project_id, connection_id, parent_id, name)
    WHERE parent_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS data_collector_point_groups_parent_order_idx
    ON data_collector_point_groups (project_id, connection_id, parent_id, sort_order, created_at, id);

CREATE TABLE IF NOT EXISTS data_collector_points (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    code text NOT NULL,
    name text NOT NULL,
    description text,
    address jsonb NOT NULL,
    address_text text NOT NULL,
    address_schema_version integer NOT NULL,
    data_type text NOT NULL,
    element_count integer NOT NULL DEFAULT 1,
    read_options jsonb NOT NULL DEFAULT '{}'::jsonb,
    acquisition jsonb NOT NULL DEFAULT '{}'::jsonb,
    enabled boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_collector_points_connection_code_key UNIQUE (project_id, connection_id, code),
    CONSTRAINT data_collector_points_connection_fkey
        FOREIGN KEY (connection_id, project_id) REFERENCES data_collector_connections (connection_id, project_id) ON DELETE CASCADE,
    CONSTRAINT data_collector_points_group_fkey
        FOREIGN KEY (group_id, project_id, connection_id)
        REFERENCES data_collector_point_groups (id, project_id, connection_id) ON DELETE RESTRICT,
    CONSTRAINT data_collector_points_code_check CHECK (char_length(trim(code)) BETWEEN 1 AND 100),
    CONSTRAINT data_collector_points_name_check CHECK (char_length(trim(name)) BETWEEN 1 AND 200),
    CONSTRAINT data_collector_points_address_check CHECK (jsonb_typeof(address) = 'object'),
    CONSTRAINT data_collector_points_address_text_check CHECK (char_length(trim(address_text)) BETWEEN 1 AND 500),
    CONSTRAINT data_collector_points_address_schema_version_check CHECK (address_schema_version > 0),
    CONSTRAINT data_collector_points_data_type_check
        CHECK (data_type IN ('bool', 'int8', 'uint8', 'int16', 'uint16', 'int32', 'uint32', 'int64', 'uint64', 'float32', 'float64', 'decimal', 'string', 'bytes', 'datetime')),
    CONSTRAINT data_collector_points_element_count_check CHECK (element_count >= 1),
    CONSTRAINT data_collector_points_read_options_check CHECK (jsonb_typeof(read_options) = 'object'),
    CONSTRAINT data_collector_points_acquisition_check CHECK (jsonb_typeof(acquisition) = 'object'),
    CONSTRAINT data_collector_points_metadata_check CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE INDEX IF NOT EXISTS data_collector_points_list_idx
    ON data_collector_points (project_id, connection_id, enabled, sort_order, created_at, id);

CREATE INDEX IF NOT EXISTS data_collector_points_group_idx
    ON data_collector_points (project_id, connection_id, group_id, sort_order, created_at, id);

CREATE INDEX IF NOT EXISTS data_collector_points_data_type_idx
    ON data_collector_points (project_id, connection_id, data_type);

CREATE INDEX IF NOT EXISTS data_collector_points_address_text_trgm_idx
    ON data_collector_points USING GIN (address_text gin_trgm_ops);

CREATE INDEX IF NOT EXISTS data_collector_points_address_gin_idx
    ON data_collector_points USING GIN (address);

CREATE TABLE IF NOT EXISTS data_collector_import_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    driver_id text NOT NULL,
    status text NOT NULL DEFAULT 'preview' CHECK (status IN ('preview', 'committed', 'expired')),
    candidates jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(candidates) = 'array'),
    errors jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(errors) = 'array'),
    total_rows integer NOT NULL DEFAULT 0 CHECK (total_rows >= 0),
    valid_rows integer NOT NULL DEFAULT 0 CHECK (valid_rows >= 0 AND valid_rows <= total_rows),
    expires_at timestamptz NOT NULL,
    created_by uuid NOT NULL,
    committed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_collector_import_sessions_connection_fkey
        FOREIGN KEY (connection_id, project_id)
        REFERENCES data_collector_connections (connection_id, project_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_collector_import_sessions_connection_created_idx
    ON data_collector_import_sessions (project_id, connection_id, created_at DESC);

CREATE INDEX IF NOT EXISTS data_collector_import_sessions_expiry_idx
    ON data_collector_import_sessions (status, expires_at)
    WHERE status = 'preview';

DELETE FROM collector_dev_tasks;

ALTER TABLE collector_dev_tasks
    DROP CONSTRAINT IF EXISTS collector_dev_tasks_operation_check;

ALTER TABLE collector_dev_tasks
    ADD COLUMN connection_id uuid NOT NULL,
    ADD CONSTRAINT collector_dev_tasks_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_collector_connections (connection_id) ON DELETE CASCADE,
    ADD CONSTRAINT collector_dev_tasks_operation_check
        CHECK (operation IN ('connection.test', 'device.browse', 'point.read', 'point.write', 'point.subscribe.preview'));

CREATE INDEX IF NOT EXISTS collector_dev_tasks_connection_created_idx
    ON collector_dev_tasks (project_id, connection_id, created_at DESC);
