ALTER TABLE data_compute_folders
    DROP CONSTRAINT IF EXISTS data_compute_folders_project_name_parent_key;

CREATE UNIQUE INDEX IF NOT EXISTS data_compute_folders_project_root_name_key
    ON data_compute_folders (project_id, name)
    WHERE parent_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS data_compute_folders_project_parent_name_key
    ON data_compute_folders (project_id, parent_id, name)
    WHERE parent_id IS NOT NULL;
