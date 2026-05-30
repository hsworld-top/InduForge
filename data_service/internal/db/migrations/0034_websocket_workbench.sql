CREATE TABLE IF NOT EXISTS data_websocket_session_groups (
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
    CONSTRAINT data_websocket_session_groups_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_websocket_session_groups_parent_fkey
        FOREIGN KEY (parent_id) REFERENCES data_websocket_session_groups (id) ON DELETE SET NULL,
    CONSTRAINT data_websocket_session_groups_name_key
        UNIQUE (project_id, connection_id, parent_id, name)
);

CREATE TABLE IF NOT EXISTS data_websocket_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    url text NOT NULL CHECK (char_length(url) <= 2000),
    headers jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(headers) = 'array'),
    auth jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(auth) = 'object'),
    protocols jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(protocols) = 'array'),
    messages jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(messages) = 'array'),
    settings jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(settings) = 'object'),
    enabled boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    last_message jsonb,
    last_diagnostic text,
    quality text NOT NULL DEFAULT 'unknown' CHECK (quality IN ('good', 'bad', 'unknown')),
    last_message_at timestamptz,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_websocket_sessions_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_websocket_sessions_group_fkey
        FOREIGN KEY (group_id) REFERENCES data_websocket_session_groups (id) ON DELETE SET NULL,
    CONSTRAINT data_websocket_sessions_name_key
        UNIQUE (project_id, connection_id, group_id, name)
);

CREATE INDEX IF NOT EXISTS data_websocket_session_groups_tree_idx
    ON data_websocket_session_groups (project_id, connection_id, parent_id, sort_order, updated_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS data_websocket_session_groups_root_name_key
    ON data_websocket_session_groups (project_id, connection_id, name)
    WHERE parent_id IS NULL;

CREATE INDEX IF NOT EXISTS data_websocket_sessions_group_idx
    ON data_websocket_sessions (project_id, connection_id, group_id, sort_order, updated_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS data_websocket_sessions_root_name_key
    ON data_websocket_sessions (project_id, connection_id, name)
    WHERE group_id IS NULL;

CREATE INDEX IF NOT EXISTS data_websocket_sessions_connection_idx
    ON data_websocket_sessions (project_id, connection_id, updated_at DESC);
