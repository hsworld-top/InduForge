ALTER TABLE data_compute_units
    ADD COLUMN IF NOT EXISTS dependencies jsonb NOT NULL DEFAULT '[]'::jsonb
    CHECK (jsonb_typeof(dependencies) = 'array');
