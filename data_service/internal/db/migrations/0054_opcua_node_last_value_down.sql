ALTER TABLE data_opcua_nodes
    DROP COLUMN IF EXISTS last_updated_at,
    DROP COLUMN IF EXISTS quality,
    DROP COLUMN IF EXISTS last_value;

