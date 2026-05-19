CREATE TABLE IF NOT EXISTS data_compute_folders (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    name text NOT NULL CHECK (char_length(name) <= 100),
    parent_id uuid,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_compute_folders_project_name_parent_key UNIQUE (project_id, parent_id, name),
    CONSTRAINT data_compute_folders_parent_fkey
        FOREIGN KEY (parent_id) REFERENCES data_compute_folders (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_compute_folders_project_parent_idx
    ON data_compute_folders (project_id, parent_id, created_at DESC);

ALTER TABLE data_compute_units
    ADD COLUMN IF NOT EXISTS folder_id uuid;

ALTER TABLE data_compute_units
    ADD CONSTRAINT data_compute_units_folder_fkey
        FOREIGN KEY (folder_id) REFERENCES data_compute_folders (id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS data_compute_units_project_folder_idx
    ON data_compute_units (project_id, folder_id, created_at DESC);
