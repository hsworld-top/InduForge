DROP INDEX IF EXISTS data_modbus_registers_slave_idx;

ALTER TABLE data_modbus_registers
    DROP CONSTRAINT IF EXISTS data_modbus_registers_slave_device_fkey;

DROP INDEX IF EXISTS data_modbus_slave_devices_connection_order_idx;
DROP TABLE IF EXISTS data_modbus_slave_devices;
