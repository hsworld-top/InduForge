WITH ordered AS (
    SELECT
        id,
        row_number() OVER (
            PARTITION BY project_id
            ORDER BY display_order ASC, created_at ASC, id ASC
        ) - 1 AS next_order
    FROM data_connections
)
UPDATE data_connections AS conn
SET display_order = ordered.next_order
FROM ordered
WHERE conn.id = ordered.id
  AND conn.display_order <> ordered.next_order;
