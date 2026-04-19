DROP TABLE IF EXISTS data_tdengine_configs;
DROP TABLE IF EXISTS data_modbus_configs;
DROP TABLE IF EXISTS data_s7_configs;
DROP TABLE IF EXISTS data_opcua_configs;

DELETE FROM data_connections WHERE type IN ('tdengine');

ALTER TABLE data_connections DROP CONSTRAINT IF EXISTS data_connections_type_check;
ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_type_check
    CHECK (type IN ('relational', 'mqtt', 'websocket', 'opcua', 'modbus', 'http', 's7', 'kafka', 'redis'));
