CREATE TABLE IF NOT EXISTS data_opcua_node_groups (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    parent_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    description text,
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_opcua_node_groups_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_opcua_node_groups_parent_fkey
        FOREIGN KEY (parent_id) REFERENCES data_opcua_node_groups (id) ON DELETE CASCADE,
    CONSTRAINT data_opcua_node_groups_identity_key
        UNIQUE (id, project_id, connection_id),
    CONSTRAINT data_opcua_node_groups_parent_connection_fkey
        FOREIGN KEY (parent_id, project_id, connection_id)
        REFERENCES data_opcua_node_groups (id, project_id, connection_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS data_opcua_node_groups_root_name_key
    ON data_opcua_node_groups (project_id, connection_id, name)
    WHERE parent_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS data_opcua_node_groups_parent_name_key
    ON data_opcua_node_groups (project_id, connection_id, parent_id, name)
    WHERE parent_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS data_opcua_node_groups_connection_parent_idx
    ON data_opcua_node_groups (project_id, connection_id, parent_id, sort_order, created_at);

CREATE TABLE IF NOT EXISTS data_opcua_nodes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    code text NOT NULL CHECK (char_length(code) <= 100),
    node_id text NOT NULL CHECK (char_length(node_id) <= 1000),
    browse_name text,
    display_name text,
    data_type text NOT NULL CHECK (char_length(data_type) <= 50),
    unit text CHECK (unit IS NULL OR char_length(unit) <= 20),
    sampling_ms integer NOT NULL DEFAULT 1000 CHECK (sampling_ms > 0),
    deadband numeric(20, 6),
    access_level text NOT NULL DEFAULT 'Read' CHECK (access_level IN ('Read', 'Write', 'ReadWrite')),
    description text,
    sort_order integer NOT NULL DEFAULT 0,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'invalid')),
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_opcua_nodes_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_opcua_nodes_group_fkey
        FOREIGN KEY (group_id) REFERENCES data_opcua_node_groups (id) ON DELETE SET NULL,
    CONSTRAINT data_opcua_nodes_group_connection_fkey
        FOREIGN KEY (group_id, project_id, connection_id)
        REFERENCES data_opcua_node_groups (id, project_id, connection_id) ON DELETE SET NULL,
    CONSTRAINT data_opcua_nodes_connection_code_key
        UNIQUE (project_id, connection_id, code),
    CONSTRAINT data_opcua_nodes_connection_node_key
        UNIQUE (project_id, connection_id, node_id)
);

CREATE INDEX IF NOT EXISTS data_opcua_nodes_group_order_idx
    ON data_opcua_nodes (project_id, connection_id, group_id, sort_order, created_at DESC);

CREATE INDEX IF NOT EXISTS data_opcua_nodes_status_idx
    ON data_opcua_nodes (project_id, connection_id, status);
