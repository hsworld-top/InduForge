CREATE TABLE IF NOT EXISTS data_s7_plc_profiles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    plc_family text NOT NULL DEFAULT 'S7 Compatible',
    communication_mode text NOT NULL DEFAULT 'rack_slot',
    host text NOT NULL CHECK (char_length(host) <= 255),
    port integer NOT NULL DEFAULT 102 CHECK (port > 0 AND port <= 65535),
    rack integer NOT NULL DEFAULT 0 CHECK (rack >= 0),
    slot integer NOT NULL DEFAULT 1 CHECK (slot >= 0),
    local_tsap text,
    remote_tsap text,
    poll_interval_ms integer NOT NULL DEFAULT 1000 CHECK (poll_interval_ms > 0),
    connect_timeout_ms integer NOT NULL DEFAULT 3000 CHECK (connect_timeout_ms > 0),
    read_timeout_ms integer NOT NULL DEFAULT 3000 CHECK (read_timeout_ms > 0),
    pdu_size integer CHECK (pdu_size IS NULL OR pdu_size > 0),
    max_read_bytes integer CHECK (max_read_bytes IS NULL OR max_read_bytes > 0),
    max_gap_bytes integer NOT NULL DEFAULT 8 CHECK (max_gap_bytes >= 0),
    max_concurrent_reads integer NOT NULL DEFAULT 1 CHECK (max_concurrent_reads > 0),
    byte_order text NOT NULL DEFAULT 'big_endian',
    word_order text NOT NULL DEFAULT 'big_endian',
    optimized_block_access boolean NOT NULL DEFAULT false,
    allow_absolute_address boolean NOT NULL DEFAULT true,
    allow_symbol_address boolean NOT NULL DEFAULT false,
    supported_areas jsonb NOT NULL DEFAULT '["DB","M","I","Q"]'::jsonb CHECK (jsonb_typeof(supported_areas) = 'array'),
    options jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(options) = 'object'),
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_s7_plc_profiles_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_s7_plc_profiles_connection_key
        UNIQUE (project_id, connection_id)
);

CREATE TABLE IF NOT EXISTS data_s7_variable_groups (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    parent_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    code text NOT NULL CHECK (char_length(code) <= 100),
    description text,
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_s7_variable_groups_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_s7_variable_groups_identity_key
        UNIQUE (id, project_id, connection_id),
    CONSTRAINT data_s7_variable_groups_parent_connection_fkey
        FOREIGN KEY (parent_id, project_id, connection_id)
        REFERENCES data_s7_variable_groups (id, project_id, connection_id) ON DELETE CASCADE,
    CONSTRAINT data_s7_variable_groups_connection_code_key
        UNIQUE (project_id, connection_id, code)
);

CREATE INDEX IF NOT EXISTS data_s7_variable_groups_connection_parent_idx
    ON data_s7_variable_groups (project_id, connection_id, parent_id, sort_order, created_at);

CREATE TABLE IF NOT EXISTS data_s7_variables (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    code text NOT NULL CHECK (char_length(code) <= 100),
    description text,
    area text NOT NULL CHECK (area IN ('DB', 'M', 'I', 'Q', 'T', 'C')),
    db_number integer CHECK (db_number IS NULL OR db_number >= 0),
    byte_offset integer NOT NULL CHECK (byte_offset >= 0),
    bit_offset integer CHECK (bit_offset IS NULL OR (bit_offset >= 0 AND bit_offset <= 7)),
    address_text text NOT NULL CHECK (char_length(address_text) <= 120),
    normalized_address text NOT NULL CHECK (char_length(normalized_address) <= 120),
    address_type text NOT NULL CHECK (char_length(address_type) <= 20),
    read_length integer NOT NULL DEFAULT 1 CHECK (read_length > 0),
    data_type text NOT NULL CHECK (char_length(data_type) <= 50),
    length integer CHECK (length IS NULL OR length > 0),
    array_length integer CHECK (array_length IS NULL OR array_length > 0),
    byte_order text NOT NULL DEFAULT 'big_endian',
    word_order text NOT NULL DEFAULT 'big_endian',
    scale numeric(20, 6) NOT NULL DEFAULT 1,
    offset_value numeric(20, 6) NOT NULL DEFAULT 0,
    unit text CHECK (unit IS NULL OR char_length(unit) <= 20),
    poll_interval_ms integer NOT NULL DEFAULT 1000 CHECK (poll_interval_ms > 0),
    quality_rule jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(quality_rule) = 'object'),
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
    last_value jsonb,
    quality text NOT NULL DEFAULT 'unknown',
    last_updated_at timestamptz,
    sort_order integer NOT NULL DEFAULT 0,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'invalid')),
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_s7_variables_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_s7_variables_group_connection_fkey
        FOREIGN KEY (group_id, project_id, connection_id)
        REFERENCES data_s7_variable_groups (id, project_id, connection_id) ON DELETE SET NULL,
    CONSTRAINT data_s7_variables_connection_code_key
        UNIQUE (project_id, connection_id, code)
);

CREATE INDEX IF NOT EXISTS data_s7_variables_group_order_idx
    ON data_s7_variables (project_id, connection_id, group_id, sort_order, created_at DESC);

CREATE INDEX IF NOT EXISTS data_s7_variables_plan_idx
    ON data_s7_variables (project_id, connection_id, area, db_number, poll_interval_ms, byte_offset);

CREATE INDEX IF NOT EXISTS data_s7_variables_status_idx
    ON data_s7_variables (project_id, connection_id, status);
