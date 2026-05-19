DROP INDEX IF EXISTS data_compute_units_project_folder_idx;
ALTER TABLE data_compute_units DROP CONSTRAINT IF EXISTS data_compute_units_folder_fkey;
ALTER TABLE data_compute_units DROP COLUMN IF EXISTS folder_id;
DROP INDEX IF EXISTS data_compute_folders_project_parent_idx;
DROP TABLE IF EXISTS data_compute_folders;
