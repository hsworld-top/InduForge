ALTER TABLE data_s7_variables
    ADD COLUMN IF NOT EXISTS access_level text NOT NULL DEFAULT 'Read';

ALTER TABLE data_s7_variables
    DROP CONSTRAINT IF EXISTS data_s7_variables_access_level_check,
    ADD CONSTRAINT data_s7_variables_access_level_check
        CHECK (access_level IN ('Read', 'Write', 'ReadWrite'));
