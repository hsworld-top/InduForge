CREATE TABLE IF NOT EXISTS data_realtime_keys (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    provider text NOT NULL CHECK (provider IN ('redis', 'builtin')),
    key_path text NOT NULL,
    redis_type text NOT NULL DEFAULT 'string',
    value_type text NOT NULL DEFAULT 'object',
    default_ttl_seconds integer NOT NULL DEFAULT 0,
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, connection_id, key_path),
    FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_realtime_keys_connection_idx
    ON data_realtime_keys (project_id, connection_id, provider, key_path);
