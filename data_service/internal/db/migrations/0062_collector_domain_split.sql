-- 工业采集已升级为数据中心独立业务域，连接主记录不再依赖通用接入源表。
ALTER TABLE data_collector_connections
    ADD COLUMN name text,
    ADD COLUMN status text,
    ADD COLUMN enabled boolean,
    ADD COLUMN display_order integer,
    ADD COLUMN created_by uuid,
    ADD COLUMN updated_by uuid;

UPDATE data_collector_connections collector
SET name = conn.name,
    status = CASE WHEN conn.status IN ('unknown', 'connected', 'disconnected', 'error') THEN conn.status ELSE 'unknown' END,
    enabled = conn.is_enabled,
    display_order = conn.display_order,
    created_by = conn.created_by,
    updated_by = conn.updated_by
FROM data_connections conn
WHERE conn.id = collector.connection_id
  AND conn.project_id = collector.project_id;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM data_collector_connections
        WHERE name IS NULL OR status IS NULL OR enabled IS NULL OR display_order IS NULL OR created_by IS NULL
    ) THEN
        RAISE EXCEPTION '存在无法关联通用接入源主记录的工业采集连接';
    END IF;
END $$;

ALTER TABLE data_collector_connections
    DROP CONSTRAINT data_collector_connections_connection_fkey;

ALTER TABLE data_collector_connections
    RENAME COLUMN connection_id TO id;

ALTER TABLE data_collector_connections
    ALTER COLUMN name SET NOT NULL,
    ALTER COLUMN status SET NOT NULL,
    ALTER COLUMN status SET DEFAULT 'unknown',
    ALTER COLUMN enabled SET NOT NULL,
    ALTER COLUMN enabled SET DEFAULT true,
    ALTER COLUMN display_order SET NOT NULL,
    ALTER COLUMN display_order SET DEFAULT 0,
    ALTER COLUMN created_by SET NOT NULL,
    ADD CONSTRAINT data_collector_connections_name_check CHECK (char_length(trim(name)) BETWEEN 1 AND 100),
    ADD CONSTRAINT data_collector_connections_status_check CHECK (status IN ('unknown', 'connected', 'disconnected', 'error')),
    ADD CONSTRAINT data_collector_connections_display_order_check CHECK (display_order >= 0);

DELETE FROM data_connections
WHERE type = 'collector' OR category = 'industrial';

ALTER TABLE data_connections
    DROP CONSTRAINT IF EXISTS data_connections_id_project_key,
    DROP CONSTRAINT IF EXISTS data_connections_type_check,
    DROP CONSTRAINT IF EXISTS data_connections_category_check;

ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_type_check
        CHECK (type IN ('relational', 'mqtt', 'websocket', 'opcua', 'modbus', 'http', 's7', 'kafka', 'redis', 'tdengine', 'builtin.relation', 'builtin.timeseries', 'builtin.realtime', 'builtin.message')),
    ADD CONSTRAINT data_connections_category_check
        CHECK (category IN ('database', 'message', 'protocol', 'api', 'builtin'));
