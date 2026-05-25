CREATE TABLE IF NOT EXISTS data_modbus_register_groups (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    parent_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    description text,
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_modbus_register_groups_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_modbus_register_groups_identity_key
        UNIQUE (id, project_id, connection_id),
    CONSTRAINT data_modbus_register_groups_parent_connection_fkey
        FOREIGN KEY (parent_id, project_id, connection_id)
        REFERENCES data_modbus_register_groups (id, project_id, connection_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS data_modbus_register_groups_root_name_key
    ON data_modbus_register_groups (project_id, connection_id, name)
    WHERE parent_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS data_modbus_register_groups_parent_name_key
    ON data_modbus_register_groups (project_id, connection_id, parent_id, name)
    WHERE parent_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS data_modbus_register_groups_connection_parent_idx
    ON data_modbus_register_groups (project_id, connection_id, parent_id, sort_order, created_at);

CREATE TABLE IF NOT EXISTS data_modbus_registers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    code text NOT NULL CHECK (char_length(code) <= 100),
    unit_id integer NOT NULL CHECK (unit_id >= 0 AND unit_id <= 247),
    area text NOT NULL CHECK (area IN ('coil', 'discrete_input', 'input_register', 'holding_register')),
    address integer NOT NULL CHECK (address >= 0),
    address_base text NOT NULL DEFAULT 'modicon' CHECK (address_base IN ('modicon', 'one_based', 'zero_based')),
    protocol_address integer NOT NULL CHECK (protocol_address >= 0),
    quantity integer NOT NULL DEFAULT 1 CHECK (quantity > 0),
    data_type text NOT NULL CHECK (char_length(data_type) <= 50),
    byte_order text NOT NULL DEFAULT 'ABCD' CHECK (byte_order IN ('ABCD', 'BADC', 'CDAB', 'DCBA')),
    word_order text NOT NULL DEFAULT 'high_first' CHECK (word_order IN ('high_first', 'low_first')),
    bit_index integer CHECK (bit_index IS NULL OR (bit_index >= 0 AND bit_index <= 15)),
    scale numeric(20, 6) NOT NULL DEFAULT 1,
    offset_value numeric(20, 6) NOT NULL DEFAULT 0,
    unit text CHECK (unit IS NULL OR char_length(unit) <= 20),
    poll_interval_ms integer NOT NULL DEFAULT 1000 CHECK (poll_interval_ms > 0),
    timeout_ms integer CHECK (timeout_ms IS NULL OR timeout_ms > 0),
    retry_count integer CHECK (retry_count IS NULL OR retry_count >= 0),
    access_level text NOT NULL DEFAULT 'Read' CHECK (access_level IN ('Read', 'Write', 'ReadWrite')),
    description text,
    sort_order integer NOT NULL DEFAULT 0,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'invalid')),
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_modbus_registers_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_modbus_registers_group_fkey
        FOREIGN KEY (group_id) REFERENCES data_modbus_register_groups (id) ON DELETE SET NULL,
    CONSTRAINT data_modbus_registers_group_connection_fkey
        FOREIGN KEY (group_id, project_id, connection_id)
        REFERENCES data_modbus_register_groups (id, project_id, connection_id) ON DELETE SET NULL,
    CONSTRAINT data_modbus_registers_connection_code_key
        UNIQUE (project_id, connection_id, code)
);

CREATE INDEX IF NOT EXISTS data_modbus_registers_group_order_idx
    ON data_modbus_registers (project_id, connection_id, group_id, sort_order, created_at DESC);

CREATE INDEX IF NOT EXISTS data_modbus_registers_plan_idx
    ON data_modbus_registers (project_id, connection_id, unit_id, area, poll_interval_ms, protocol_address);

CREATE INDEX IF NOT EXISTS data_modbus_registers_status_idx
    ON data_modbus_registers (project_id, connection_id, status);
