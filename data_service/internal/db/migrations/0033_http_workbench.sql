CREATE TABLE IF NOT EXISTS data_http_request_groups (
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
    CONSTRAINT data_http_request_groups_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_http_request_groups_parent_fkey
        FOREIGN KEY (parent_id) REFERENCES data_http_request_groups (id) ON DELETE SET NULL,
    CONSTRAINT data_http_request_groups_name_key
        UNIQUE (project_id, connection_id, parent_id, name)
);

CREATE TABLE IF NOT EXISTS data_http_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    method text NOT NULL DEFAULT 'GET' CHECK (method IN ('GET', 'POST', 'PUT', 'PATCH', 'DELETE')),
    url text NOT NULL CHECK (char_length(url) <= 2000),
    params jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(params) = 'array'),
    headers jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(headers) = 'array'),
    auth jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(auth) = 'object'),
    body_type text NOT NULL DEFAULT 'none' CHECK (body_type IN ('none', 'json', 'raw', 'form-data', 'x-www-form-urlencoded')),
    body jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(body) = 'object'),
    settings jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(settings) = 'object'),
    enabled boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    last_response jsonb,
    quality text NOT NULL DEFAULT 'unknown' CHECK (quality IN ('good', 'bad', 'unknown')),
    last_sent_at timestamptz,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_http_requests_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_http_requests_group_fkey
        FOREIGN KEY (group_id) REFERENCES data_http_request_groups (id) ON DELETE SET NULL,
    CONSTRAINT data_http_requests_name_key
        UNIQUE (project_id, connection_id, group_id, name)
);

CREATE INDEX IF NOT EXISTS data_http_request_groups_tree_idx
    ON data_http_request_groups (project_id, connection_id, parent_id, sort_order, updated_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS data_http_request_groups_root_name_key
    ON data_http_request_groups (project_id, connection_id, name)
    WHERE parent_id IS NULL;

CREATE INDEX IF NOT EXISTS data_http_requests_group_idx
    ON data_http_requests (project_id, connection_id, group_id, sort_order, updated_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS data_http_requests_root_name_key
    ON data_http_requests (project_id, connection_id, name)
    WHERE group_id IS NULL;

CREATE INDEX IF NOT EXISTS data_http_requests_connection_idx
    ON data_http_requests (project_id, connection_id, updated_at DESC);
