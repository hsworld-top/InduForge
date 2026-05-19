DROP INDEX IF EXISTS data_compute_folders_project_parent_name_key;
DROP INDEX IF EXISTS data_compute_folders_project_root_name_key;

ALTER TABLE data_compute_folders
    ADD CONSTRAINT data_compute_folders_project_name_parent_key UNIQUE (project_id, parent_id, name);
