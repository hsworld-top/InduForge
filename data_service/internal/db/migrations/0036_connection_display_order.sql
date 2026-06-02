ALTER TABLE data_connections
    ADD COLUMN IF NOT EXISTS display_order integer NOT NULL DEFAULT 0;

WITH ordered AS (
    SELECT
        id,
        row_number() OVER (PARTITION BY project_id ORDER BY created_at ASC, id ASC) - 1 AS next_order
    FROM data_connections
)
UPDATE data_connections AS conn
SET display_order = ordered.next_order
FROM ordered
WHERE conn.id = ordered.id;

CREATE INDEX IF NOT EXISTS data_connections_project_order_idx
    ON data_connections (project_id, display_order, created_at DESC);
