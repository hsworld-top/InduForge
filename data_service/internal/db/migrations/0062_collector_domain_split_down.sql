ALTER TABLE data_connections
    DROP CONSTRAINT IF EXISTS data_connections_type_check,
    DROP CONSTRAINT IF EXISTS data_connections_category_check;

ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_type_check
        CHECK (type IN ('relational', 'mqtt', 'websocket', 'opcua', 'modbus', 'http', 's7', 'kafka', 'redis', 'tdengine', 'builtin.relation', 'builtin.timeseries', 'builtin.realtime', 'builtin.message', 'collector')),
    ADD CONSTRAINT data_connections_category_check
        CHECK (category IN ('database', 'message', 'protocol', 'api', 'builtin', 'industrial')),
    ADD CONSTRAINT data_connections_id_project_key UNIQUE (id, project_id);

INSERT INTO data_connections (
    id, project_id, name, type, category, status, is_enabled, display_order,
    metadata, created_by, updated_by, created_at, updated_at
)
SELECT id, project_id, name, 'collector', 'industrial', status, enabled, display_order,
       metadata, created_by, updated_by, created_at, updated_at
FROM data_collector_connections
ON CONFLICT (id) DO NOTHING;

ALTER TABLE data_collector_connections
    DROP CONSTRAINT data_collector_connections_name_check,
    DROP CONSTRAINT data_collector_connections_status_check,
    DROP CONSTRAINT data_collector_connections_display_order_check;

ALTER TABLE data_collector_connections
    RENAME COLUMN id TO connection_id;

ALTER TABLE data_collector_connections
    DROP COLUMN name,
    DROP COLUMN status,
    DROP COLUMN enabled,
    DROP COLUMN display_order,
    DROP COLUMN created_by,
    DROP COLUMN updated_by,
    ADD CONSTRAINT data_collector_connections_connection_fkey
        FOREIGN KEY (connection_id, project_id) REFERENCES data_connections (id, project_id) ON DELETE CASCADE;
