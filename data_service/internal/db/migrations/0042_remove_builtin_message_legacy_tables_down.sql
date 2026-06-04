CREATE TABLE IF NOT EXISTS data_builtin_message_topics (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    topic text NOT NULL,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, connection_id, topic),
    FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS data_builtin_message_variables (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    topic_id uuid NOT NULL,
    name text NOT NULL,
    payload_path text NOT NULL,
    value_type text NOT NULL DEFAULT 'string',
    unit text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    create_datapoint boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, topic_id, name),
    FOREIGN KEY (topic_id) REFERENCES data_builtin_message_topics (id) ON DELETE CASCADE
);
