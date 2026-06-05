ALTER TABLE data_points
    ADD COLUMN IF NOT EXISTS display_order integer NOT NULL DEFAULT 0;

WITH ordered AS (
    SELECT id, row_number() OVER (PARTITION BY project_id ORDER BY created_at ASC, id ASC) - 1 AS next_order
    FROM data_points
)
UPDATE data_points dp
SET display_order = ordered.next_order
FROM ordered
WHERE dp.id = ordered.id;

CREATE INDEX IF NOT EXISTS data_points_project_created_display_idx
    ON data_points (project_id, created_at DESC, display_order ASC, id DESC);
