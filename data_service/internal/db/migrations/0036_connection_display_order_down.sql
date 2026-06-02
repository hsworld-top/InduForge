DROP INDEX IF EXISTS data_connections_project_order_idx;

ALTER TABLE data_connections
    DROP COLUMN IF EXISTS display_order;
