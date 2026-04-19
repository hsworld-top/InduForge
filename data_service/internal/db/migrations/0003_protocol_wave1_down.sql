DROP TABLE IF EXISTS data_redis_configs;
DROP TABLE IF EXISTS data_websocket_configs;
DROP TABLE IF EXISTS data_http_configs;
DROP TABLE IF EXISTS data_kafka_configs;

DELETE FROM data_connections WHERE type IN ('kafka', 'redis');

ALTER TABLE data_connections DROP CONSTRAINT IF EXISTS data_connections_type_check;
ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_type_check
    CHECK (type IN ('relational', 'mqtt', 'websocket', 'opcua', 'modbus', 'http', 's7'));
