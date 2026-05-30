ALTER TABLE data_compute_units DROP CONSTRAINT IF EXISTS data_compute_units_folder_fkey;
DROP INDEX IF EXISTS data_compute_folders_project_parent_name_key;
DROP INDEX IF EXISTS data_compute_folders_project_root_name_key;
DROP INDEX IF EXISTS data_compute_folders_project_parent_idx;
DROP TABLE IF EXISTS data_compute_folders;
