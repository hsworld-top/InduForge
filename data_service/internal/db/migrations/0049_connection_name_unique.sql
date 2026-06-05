WITH duplicate_connections AS (
    SELECT
        id,
        name,
        row_number() OVER (
            PARTITION BY project_id, name
            ORDER BY created_at ASC, id ASC
        ) AS duplicate_index
    FROM data_connections
)
UPDATE data_connections AS conn
SET
    name = left(duplicate_connections.name, 91) || '_' || left(conn.id::text, 8),
    updated_at = now()
FROM duplicate_connections
WHERE conn.id = duplicate_connections.id
  AND duplicate_connections.duplicate_index > 1;

CREATE UNIQUE INDEX IF NOT EXISTS data_connections_project_name_key
    ON data_connections (project_id, name);
