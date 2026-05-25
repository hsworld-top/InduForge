DROP INDEX IF EXISTS data_modbus_registers_status_idx;
DROP INDEX IF EXISTS data_modbus_registers_plan_idx;
DROP INDEX IF EXISTS data_modbus_registers_group_order_idx;
DROP TABLE IF EXISTS data_modbus_registers;
DROP INDEX IF EXISTS data_modbus_register_groups_connection_parent_idx;
DROP INDEX IF EXISTS data_modbus_register_groups_parent_name_key;
DROP INDEX IF EXISTS data_modbus_register_groups_root_name_key;
DROP TABLE IF EXISTS data_modbus_register_groups;
