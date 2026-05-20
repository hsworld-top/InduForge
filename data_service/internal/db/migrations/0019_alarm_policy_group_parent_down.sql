DROP INDEX IF EXISTS data_alarm_policy_groups_project_parent_name_key;
DROP INDEX IF EXISTS data_alarm_policy_groups_project_root_name_key;
DROP INDEX IF EXISTS data_alarm_policy_groups_project_parent_idx;

ALTER TABLE data_alarm_policy_groups
    DROP CONSTRAINT IF EXISTS data_alarm_policy_groups_parent_fkey;

ALTER TABLE data_alarm_policy_groups
    ADD CONSTRAINT data_alarm_policy_groups_project_name_key UNIQUE (project_id, name);

ALTER TABLE data_alarm_policy_groups
    DROP COLUMN IF EXISTS parent_id;
