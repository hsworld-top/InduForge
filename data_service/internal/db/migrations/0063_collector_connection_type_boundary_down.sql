ALTER TABLE data_connections
    DROP CONSTRAINT IF EXISTS data_connections_type_check,
    DROP CONSTRAINT IF EXISTS data_connections_category_check;

ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_type_check
        CHECK (type IN ('relational', 'mqtt', 'websocket', 'opcua', 'modbus', 'http', 's7', 'kafka', 'redis', 'tdengine', 'builtin.relation', 'builtin.timeseries', 'builtin.realtime', 'builtin.message')),
    ADD CONSTRAINT data_connections_category_check
        CHECK (category IN ('database', 'message', 'protocol', 'api', 'builtin'));
