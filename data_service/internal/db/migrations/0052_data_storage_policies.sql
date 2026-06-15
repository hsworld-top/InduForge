CREATE TABLE IF NOT EXISTS data_storage_policies (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    name text NOT NULL CHECK (char_length(name) <= 100),
    description text,
    target_connection_id uuid NOT NULL,
    target_capability text NOT NULL CHECK (target_capability IN ('timeseriesAppend', 'relationalAppend')),
    binding_mode text NOT NULL CHECK (binding_mode IN ('static', 'dynamic')),
    binding_filter jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(binding_filter) = 'object'),
    write_mode text NOT NULL CHECK (write_mode IN ('every_sample', 'on_change', 'periodic_snapshot')),
    min_interval_ms integer CHECK (min_interval_ms IS NULL OR min_interval_ms >= 0),
    deadband numeric(20, 6) CHECK (deadband IS NULL OR deadband >= 0),
    snapshot_interval_ms integer CHECK (snapshot_interval_ms IS NULL OR snapshot_interval_ms > 0),
    include_qualities jsonb NOT NULL DEFAULT '["Good","Uncertain"]'::jsonb CHECK (jsonb_typeof(include_qualities) = 'array'),
    retention_days integer NOT NULL CHECK (retention_days > 0),
    target_table_mode text NOT NULL DEFAULT 'auto_create' CHECK (target_table_mode IN ('auto_create', 'existing_mapping')),
    target_table_config jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(target_table_config) = 'object'),
    status text NOT NULL DEFAULT 'enabled' CHECK (status IN ('enabled', 'disabled', 'error')),
    diagnostics jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(diagnostics) = 'array'),
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_storage_policies_project_name_key UNIQUE (project_id, name),
    CONSTRAINT data_storage_policies_target_connection_fkey
        FOREIGN KEY (target_connection_id) REFERENCES data_connections (id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS data_storage_policies_project_status_idx
    ON data_storage_policies (project_id, status);

CREATE INDEX IF NOT EXISTS data_storage_policies_target_idx
    ON data_storage_policies (project_id, target_connection_id);

CREATE INDEX IF NOT EXISTS data_storage_policies_binding_filter_gin_idx
    ON data_storage_policies USING GIN (binding_filter);

CREATE TABLE IF NOT EXISTS data_storage_policy_bindings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    policy_id uuid NOT NULL,
    datapoint_id uuid NOT NULL,
    datapoint_path text NOT NULL CHECK (char_length(datapoint_path) <= 255),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_storage_policy_bindings_policy_fkey
        FOREIGN KEY (policy_id) REFERENCES data_storage_policies (id) ON DELETE CASCADE,
    CONSTRAINT data_storage_policy_bindings_datapoint_fkey
        FOREIGN KEY (datapoint_id) REFERENCES data_points (id) ON DELETE CASCADE,
    CONSTRAINT data_storage_policy_bindings_policy_datapoint_key UNIQUE (policy_id, datapoint_id)
);

CREATE INDEX IF NOT EXISTS data_storage_policy_bindings_project_policy_idx
    ON data_storage_policy_bindings (project_id, policy_id);

CREATE INDEX IF NOT EXISTS data_storage_policy_bindings_datapoint_idx
    ON data_storage_policy_bindings (project_id, datapoint_id);
