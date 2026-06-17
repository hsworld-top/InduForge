ALTER TABLE data_modbus_registers
    DROP COLUMN IF EXISTS last_updated_at,
    DROP COLUMN IF EXISTS quality,
    DROP COLUMN IF EXISTS last_value;
