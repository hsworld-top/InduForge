DELETE FROM collector_dev_tasks;

ALTER TABLE collector_dev_tasks
    DROP CONSTRAINT IF EXISTS collector_dev_tasks_operation_check,
    DROP CONSTRAINT IF EXISTS collector_dev_tasks_connection_fkey,
    DROP COLUMN IF EXISTS connection_id;

ALTER TABLE collector_dev_tasks
    ADD CONSTRAINT collector_dev_tasks_operation_check
        CHECK (operation IN ('connection.test', 'opcua.browse', 'opcua.read'));

DROP TABLE IF EXISTS data_collector_import_sessions;
DROP TABLE IF EXISTS data_collector_points;
DROP TABLE IF EXISTS data_collector_point_groups;
DROP TABLE IF EXISTS data_collector_connection_secrets;
DROP TABLE IF EXISTS data_collector_connections;

DELETE FROM data_connections
WHERE type = 'collector' OR category = 'industrial';

ALTER TABLE data_connections
    DROP CONSTRAINT IF EXISTS data_connections_id_project_key;

ALTER TABLE data_connections
    DROP CONSTRAINT IF EXISTS data_connections_type_check;

ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_type_check
        CHECK (type IN ('relational', 'mqtt', 'websocket', 'opcua', 'modbus', 'http', 's7', 'kafka', 'redis', 'tdengine', 'builtin.relation', 'builtin.timeseries', 'builtin.realtime', 'builtin.message'));

ALTER TABLE data_connections
    DROP CONSTRAINT IF EXISTS data_connections_category_check;

ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_category_check
        CHECK (category IN ('database', 'message', 'protocol', 'api', 'builtin'));
