DROP INDEX IF EXISTS data_connections_project_builtin_type_unique;
DROP TABLE IF EXISTS data_builtin_message_variables;
DROP TABLE IF EXISTS data_builtin_message_topics;
DROP TABLE IF EXISTS data_builtin_realtime_keys;

DELETE FROM data_connections
WHERE type IN ('builtin.relation', 'builtin.timeseries', 'builtin.realtime', 'builtin.message');

ALTER TABLE data_connections DROP CONSTRAINT IF EXISTS data_connections_category_check;
ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_category_check
    CHECK (category IN ('database', 'message', 'protocol', 'api'));

ALTER TABLE data_connections DROP CONSTRAINT IF EXISTS data_connections_type_check;
ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_type_check
    CHECK (type IN (
        'relational', 'mqtt', 'websocket', 'opcua', 'modbus', 'http', 's7',
        'kafka', 'redis', 'tdengine'
    ));
