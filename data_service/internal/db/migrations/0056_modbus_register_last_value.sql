ALTER TABLE data_modbus_registers
    ADD COLUMN IF NOT EXISTS last_value jsonb,
    ADD COLUMN IF NOT EXISTS quality text NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS last_updated_at timestamptz;
