-- 通用接入源表只保留数据库、消息、接口和内置运行库类型，工业协议统一进入采集域。
ALTER TABLE data_connections
    DROP CONSTRAINT IF EXISTS data_connections_type_check,
    DROP CONSTRAINT IF EXISTS data_connections_category_check;

ALTER TABLE data_connections
    ADD CONSTRAINT data_connections_type_check
        CHECK (type IN ('relational', 'mqtt', 'websocket', 'http', 'kafka', 'redis', 'tdengine', 'builtin.relation', 'builtin.timeseries', 'builtin.realtime', 'builtin.message')),
    ADD CONSTRAINT data_connections_category_check
        CHECK (category IN ('database', 'message', 'protocol', 'api', 'builtin'));
