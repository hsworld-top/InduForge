-- 将旧 OPC UA、Modbus、S7 建模数据一次性归并到统一工业采集模型。
-- 旧 OPC UA 密码不能在 SQL 迁移中生成应用 AES-GCM 密文，仅记录需重新录入认证的标记。

INSERT INTO data_collector_connections (
    connection_id, project_id, protocol_family, driver_id, driver_version, schema_version, config, metadata, created_at, updated_at
)
SELECT
    c.id, c.project_id, 'opcua', 'opcua.standard', '1.0.0', 1,
    jsonb_build_object(
        'endpointUrl', oc.endpoint,
        'securityMode', oc.security_mode,
        'securityPolicy', oc.security_policy,
        'authenticationType', 'anonymous',
        'timeoutMs', COALESCE((oc.options ->> 'timeoutMs')::integer, 30000)
    ),
    c.metadata || CASE WHEN oc.auth_type <> 'anonymous' THEN '{"legacyAuthRequiresReentry":true}'::jsonb ELSE '{}'::jsonb END,
    c.created_at, c.updated_at
FROM data_connections c
JOIN data_opcua_configs oc ON oc.connection_id = c.id
WHERE c.type = 'opcua'
ON CONFLICT (connection_id) DO NOTHING;

INSERT INTO data_collector_connections (
    connection_id, project_id, protocol_family, driver_id, driver_version, schema_version, config, metadata, created_at, updated_at
)
SELECT
    c.id, c.project_id, 'modbus', CASE WHEN mc.mode = 'rtu' THEN 'modbus.rtu' ELSE 'modbus.tcp' END, '1.0.0', 1,
    CASE WHEN mc.mode = 'rtu' THEN jsonb_build_object(
        'portName', COALESCE(mc.serial_config ->> 'portName', mc.serial_config ->> 'port', ''),
        'baudRate', COALESCE((mc.serial_config ->> 'baudRate')::integer, 9600),
        'dataBits', COALESCE((mc.serial_config ->> 'dataBits')::integer, 8),
        'parity', COALESCE(lower(mc.serial_config ->> 'parity'), 'none'),
        'stopBits', COALESCE(lower(mc.serial_config ->> 'stopBits'), 'one'),
        'timeoutMs', COALESCE((mc.options ->> 'timeoutMs')::integer, 3000)
    ) ELSE jsonb_build_object(
        'host', COALESCE(mc.host, ''),
        'port', COALESCE(mc.port, 502),
        'connectTimeoutMs', COALESCE((mc.options ->> 'connectTimeoutMs')::integer, 5000)
    ) END,
    c.metadata, c.created_at, c.updated_at
FROM data_connections c
JOIN data_modbus_configs mc ON mc.connection_id = c.id
WHERE c.type = 'modbus'
ON CONFLICT (connection_id) DO NOTHING;

INSERT INTO data_collector_connections (
    connection_id, project_id, protocol_family, driver_id, driver_version, schema_version, config, metadata, created_at, updated_at
)
SELECT
    c.id, c.project_id, 's7', 'siemens.s7-tcp', '1.0.0', 1,
    jsonb_strip_nulls(jsonb_build_object(
        'host', COALESCE(sp.host, sc.host),
        'port', COALESCE(sp.port, sc.port, 102),
        'rack', COALESCE(sp.rack, sc.rack, 0),
        'slot', COALESCE(sp.slot, sc.slot, 1),
        'localTsap', CASE WHEN sp.local_tsap ~ '^[0-9]+$' THEN sp.local_tsap::integer END,
        'remoteTsap', CASE WHEN sp.remote_tsap ~ '^[0-9]+$' THEN sp.remote_tsap::integer END,
        'connectTimeoutMs', COALESCE(sp.connect_timeout_ms, 5000)
    )),
    c.metadata, c.created_at, c.updated_at
FROM data_connections c
JOIN data_s7_configs sc ON sc.connection_id = c.id
LEFT JOIN data_s7_plc_profiles sp ON sp.connection_id = c.id AND sp.project_id = c.project_id
WHERE c.type = 's7'
ON CONFLICT (connection_id) DO NOTHING;

INSERT INTO data_collector_point_groups (id, project_id, connection_id, parent_id, name, sort_order, metadata, created_at, updated_at)
SELECT id, project_id, connection_id, parent_id, name, sort_order, jsonb_strip_nulls(jsonb_build_object('description', description)), created_at, updated_at FROM data_opcua_node_groups
UNION ALL
SELECT id, project_id, connection_id, parent_id, name, sort_order, jsonb_strip_nulls(jsonb_build_object('description', description)), created_at, updated_at FROM data_modbus_register_groups
UNION ALL
SELECT id, project_id, connection_id, parent_id, name, sort_order, jsonb_strip_nulls(jsonb_build_object('code', code, 'description', description)), created_at, updated_at FROM data_s7_variable_groups
ON CONFLICT (id) DO NOTHING;

INSERT INTO data_collector_points (id, project_id, connection_id, group_id, code, name, description, address, address_text, address_schema_version, data_type, element_count, read_options, acquisition, enabled, sort_order, metadata, created_at, updated_at)
SELECT id, project_id, connection_id, group_id, code, name, description,
       jsonb_build_object('nodeId', node_id), node_id, 1,
       CASE lower(data_type) WHEN 'boolean' THEN 'bool' WHEN 'bool' THEN 'bool' WHEN 'sbyte' THEN 'int8' WHEN 'byte' THEN 'uint8' WHEN 'int16' THEN 'int16' WHEN 'uint16' THEN 'uint16' WHEN 'int32' THEN 'int32' WHEN 'uint32' THEN 'uint32' WHEN 'int64' THEN 'int64' WHEN 'uint64' THEN 'uint64' WHEN 'single' THEN 'float32' WHEN 'float' THEN 'float32' WHEN 'double' THEN 'float64' WHEN 'datetime' THEN 'datetime' WHEN 'bytestring' THEN 'bytes' ELSE 'string' END,
       1, jsonb_strip_nulls(jsonb_build_object('deadband', deadband)), jsonb_build_object('mode', 'polling', 'intervalMs', sampling_ms), status = 'active', sort_order,
       jsonb_strip_nulls(jsonb_build_object('browseName', browse_name, 'displayName', display_name, 'unit', unit, 'legacyAccessLevel', access_level)), created_at, updated_at
FROM data_opcua_nodes
ON CONFLICT (id) DO NOTHING;

INSERT INTO data_collector_points (id, project_id, connection_id, group_id, code, name, description, address, address_text, address_schema_version, data_type, element_count, read_options, acquisition, enabled, sort_order, metadata, created_at, updated_at)
SELECT id, project_id, connection_id, group_id, code, name, description,
       jsonb_strip_nulls(jsonb_build_object('station', unit_id, 'area', CASE area WHEN 'discrete_input' THEN 'discreteInput' WHEN 'input_register' THEN 'inputRegister' WHEN 'holding_register' THEN 'holdingRegister' ELSE area END, 'address', protocol_address, 'bitIndex', bit_index)),
       concat(unit_id, ':', area, ':', protocol_address, CASE WHEN bit_index IS NULL THEN '' ELSE concat('.', bit_index) END), 1,
       CASE lower(data_type) WHEN 'bool' THEN 'bool' WHEN 'boolean' THEN 'bool' WHEN 'int16' THEN 'int16' WHEN 'uint16' THEN 'uint16' WHEN 'int32' THEN 'int32' WHEN 'uint32' THEN 'uint32' WHEN 'int64' THEN 'int64' WHEN 'uint64' THEN 'uint64' WHEN 'float' THEN 'float32' WHEN 'float32' THEN 'float32' WHEN 'double' THEN 'float64' WHEN 'float64' THEN 'float64' ELSE 'string' END,
       quantity, jsonb_strip_nulls(jsonb_build_object('byteOrder', byte_order, 'wordOrder', word_order, 'scale', scale, 'offset', offset_value, 'timeoutMs', timeout_ms, 'retryCount', retry_count)), jsonb_build_object('mode', 'polling', 'intervalMs', poll_interval_ms), status = 'active', sort_order,
       jsonb_strip_nulls(jsonb_build_object('unit', unit, 'addressBase', address_base, 'legacyAccessLevel', access_level)), created_at, updated_at
FROM data_modbus_registers
ON CONFLICT (id) DO NOTHING;

INSERT INTO data_collector_points (id, project_id, connection_id, group_id, code, name, description, address, address_text, address_schema_version, data_type, element_count, read_options, acquisition, enabled, sort_order, metadata, created_at, updated_at)
SELECT id, project_id, connection_id, group_id, code, name, description,
       jsonb_strip_nulls(jsonb_build_object('area', CASE area WHEN 'DB' THEN 'dataBlock' WHEN 'M' THEN 'marker' WHEN 'I' THEN 'input' WHEN 'Q' THEN 'output' WHEN 'T' THEN 'timer' WHEN 'C' THEN 'counter' END, 'dbNumber', db_number, 'byteOffset', byte_offset, 'bitOffset', bit_offset)),
       address_text, 1,
       CASE lower(data_type) WHEN 'bool' THEN 'bool' WHEN 'byte' THEN 'uint8' WHEN 'word' THEN 'uint16' WHEN 'dword' THEN 'uint32' WHEN 'int' THEN 'int16' WHEN 'dint' THEN 'int32' WHEN 'lint' THEN 'int64' WHEN 'real' THEN 'float32' WHEN 'lreal' THEN 'float64' WHEN 'string' THEN 'string' WHEN 'date_time' THEN 'datetime' ELSE 'string' END,
       COALESCE(array_length, length, 1), jsonb_build_object('byteOrder', byte_order, 'wordOrder', word_order, 'scale', scale, 'offset', offset_value), jsonb_build_object('mode', 'polling', 'intervalMs', poll_interval_ms), status = 'active', sort_order,
       metadata || jsonb_strip_nulls(jsonb_build_object('unit', unit, 'qualityRule', quality_rule, 'legacyAddressType', address_type)), created_at, updated_at
FROM data_s7_variables
ON CONFLICT (id) DO NOTHING;

UPDATE data_points dp
SET source_type = 'collector.point', source_config = jsonb_build_object('connectionId', cp.connection_id, 'address', cp.address), updated_at = now()
FROM data_collector_points cp
WHERE dp.source_id = cp.id AND dp.source_type IN ('opcua.node', 'modbus.register', 's7.variable');

INSERT INTO data_points (project_id, path, name, description, source_type, source_id, source_config, data_type, unit, refresh_mode, refresh_interval_ms, status, display_order, created_at, updated_at)
SELECT cp.project_id, concat('collector/', cp.connection_id, '/', cp.code), cp.name, cp.description, 'collector.point', cp.id,
       jsonb_build_object('connectionId', cp.connection_id, 'address', cp.address), cp.data_type, cp.metadata ->> 'unit', 'auto', COALESCE((cp.acquisition ->> 'intervalMs')::integer, 1000), CASE WHEN cp.enabled THEN 'active' ELSE 'inactive' END, cp.sort_order, cp.created_at, cp.updated_at
FROM data_collector_points cp
WHERE NOT EXISTS (SELECT 1 FROM data_points dp WHERE dp.source_type = 'collector.point' AND dp.source_id = cp.id)
ON CONFLICT (project_id, path) DO NOTHING;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM data_collector_points cp LEFT JOIN data_points dp ON dp.source_type = 'collector.point' AND dp.source_id = cp.id WHERE dp.id IS NULL) THEN
        RAISE EXCEPTION '统一采集点存在未映射的数据点';
    END IF;
END $$;

UPDATE data_connections SET type = 'collector', category = 'industrial', updated_at = now() WHERE type IN ('opcua', 'modbus', 's7');

DROP TABLE data_s7_variables, data_s7_variable_groups, data_s7_plc_profiles;
DROP TABLE data_modbus_registers, data_modbus_slave_devices, data_modbus_register_groups;
DROP TABLE data_opcua_nodes, data_opcua_node_groups;
DROP TABLE data_s7_configs, data_modbus_configs, data_opcua_configs;
