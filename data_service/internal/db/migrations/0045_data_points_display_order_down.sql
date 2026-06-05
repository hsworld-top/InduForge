DROP INDEX IF EXISTS data_points_project_created_display_idx;

ALTER TABLE data_points
    DROP COLUMN IF EXISTS display_order;
