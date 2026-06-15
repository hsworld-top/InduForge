ALTER TABLE data_s7_variables
    DROP CONSTRAINT IF EXISTS data_s7_variables_access_level_check,
    DROP COLUMN IF EXISTS access_level;
