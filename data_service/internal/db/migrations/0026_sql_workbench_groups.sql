CREATE TABLE IF NOT EXISTS data_workbench_object_groups (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    scope text NOT NULL CHECK (scope IN ('query', 'table')),
    name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_workbench_object_groups_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS data_workbench_object_groups_name_key
    ON data_workbench_object_groups (project_id, connection_id, scope, name);

CREATE INDEX IF NOT EXISTS data_workbench_object_groups_connection_scope_idx
    ON data_workbench_object_groups (project_id, connection_id, scope, sort_order, created_at);

ALTER TABLE data_queries
    ADD COLUMN IF NOT EXISTS group_id uuid;

ALTER TABLE data_queries DROP CONSTRAINT IF EXISTS data_queries_group_fkey;
ALTER TABLE data_queries
    ADD CONSTRAINT data_queries_group_fkey
    FOREIGN KEY (group_id) REFERENCES data_workbench_object_groups (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS data_queries_project_connection_group_idx
    ON data_queries (project_id, connection_id, group_id, created_at DESC);

CREATE TABLE IF NOT EXISTS data_table_group_members (
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    table_name text NOT NULL CHECK (char_length(table_name) BETWEEN 1 AND 255),
    group_id uuid NOT NULL,
    updated_by uuid,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, connection_id, table_name),
    CONSTRAINT data_table_group_members_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_table_group_members_group_fkey
        FOREIGN KEY (group_id) REFERENCES data_workbench_object_groups (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_table_group_members_group_idx
    ON data_table_group_members (project_id, connection_id, group_id);
