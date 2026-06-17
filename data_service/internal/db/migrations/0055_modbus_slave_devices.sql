CREATE TABLE IF NOT EXISTS data_modbus_slave_devices (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    unit_id integer NOT NULL CHECK (unit_id >= 0 AND unit_id <= 247),
    name text NOT NULL CHECK (char_length(name) <= 100),
    description text,
    enabled boolean NOT NULL DEFAULT true,
    default_poll_interval_ms integer NOT NULL DEFAULT 1000 CHECK (default_poll_interval_ms > 0),
    default_byte_order text NOT NULL DEFAULT 'ABCD' CHECK (default_byte_order IN ('ABCD', 'BADC', 'CDAB', 'DCBA')),
    default_word_order text NOT NULL DEFAULT 'high_first' CHECK (default_word_order IN ('high_first', 'low_first')),
    request_interval_ms integer CHECK (request_interval_ms IS NULL OR request_interval_ms >= 0),
    timeout_ms integer CHECK (timeout_ms IS NULL OR timeout_ms > 0),
    retry_count integer CHECK (retry_count IS NULL OR retry_count >= 0),
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_modbus_slave_devices_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_modbus_slave_devices_identity_key
        UNIQUE (id, project_id, connection_id),
    CONSTRAINT data_modbus_slave_devices_unit_key
        UNIQUE (project_id, connection_id, unit_id)
);

CREATE INDEX IF NOT EXISTS data_modbus_slave_devices_connection_order_idx
    ON data_modbus_slave_devices (project_id, connection_id, sort_order, unit_id);

INSERT INTO data_modbus_slave_devices (
    project_id,
    connection_id,
    unit_id,
    name,
    default_poll_interval_ms,
    default_byte_order,
    default_word_order,
    sort_order,
    created_by,
    updated_by
)
SELECT
    r.project_id,
    r.connection_id,
    r.unit_id,
    '从站 ' || r.unit_id::text,
    MIN(r.poll_interval_ms),
    COALESCE((array_agg(r.byte_order ORDER BY r.created_at ASC))[1], 'ABCD'),
    COALESCE((array_agg(r.word_order ORDER BY r.created_at ASC))[1], 'high_first'),
    r.unit_id,
    COALESCE((array_agg(r.created_by ORDER BY r.created_at ASC))[1], c.created_by),
    COALESCE((array_agg(r.updated_by ORDER BY r.updated_at DESC NULLS LAST))[1], (array_agg(r.created_by ORDER BY r.created_at ASC))[1], c.updated_by, c.created_by)
FROM data_modbus_registers r
JOIN data_connections c
  ON c.id = r.connection_id
GROUP BY r.project_id, r.connection_id, r.unit_id, c.created_by, c.updated_by
ON CONFLICT (project_id, connection_id, unit_id) DO NOTHING;

ALTER TABLE data_modbus_registers
    ADD CONSTRAINT data_modbus_registers_slave_device_fkey
    FOREIGN KEY (project_id, connection_id, unit_id)
    REFERENCES data_modbus_slave_devices (project_id, connection_id, unit_id);

CREATE INDEX IF NOT EXISTS data_modbus_registers_slave_idx
    ON data_modbus_registers (project_id, connection_id, unit_id, sort_order, created_at DESC);
