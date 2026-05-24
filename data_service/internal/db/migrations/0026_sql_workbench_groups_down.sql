DROP INDEX IF EXISTS data_table_group_members_group_idx;
DROP TABLE IF EXISTS data_table_group_members;

DROP INDEX IF EXISTS data_queries_project_connection_group_idx;
ALTER TABLE data_queries DROP CONSTRAINT IF EXISTS data_queries_group_fkey;
ALTER TABLE data_queries DROP COLUMN IF EXISTS group_id;

DROP INDEX IF EXISTS data_workbench_object_groups_connection_scope_idx;
DROP INDEX IF EXISTS data_workbench_object_groups_name_key;
DROP TABLE IF EXISTS data_workbench_object_groups;
