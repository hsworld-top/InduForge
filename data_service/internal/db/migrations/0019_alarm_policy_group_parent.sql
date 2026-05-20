ALTER TABLE data_alarm_policy_groups
    ADD COLUMN IF NOT EXISTS parent_id uuid;

ALTER TABLE data_alarm_policy_groups
    DROP CONSTRAINT IF EXISTS data_alarm_policy_groups_project_name_key;

ALTER TABLE data_alarm_policy_groups
    ADD CONSTRAINT data_alarm_policy_groups_parent_fkey
        FOREIGN KEY (parent_id) REFERENCES data_alarm_policy_groups(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS data_alarm_policy_groups_project_parent_idx
    ON data_alarm_policy_groups(project_id, parent_id, sort_order, created_at);

CREATE UNIQUE INDEX IF NOT EXISTS data_alarm_policy_groups_project_root_name_key
    ON data_alarm_policy_groups(project_id, name)
    WHERE parent_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS data_alarm_policy_groups_project_parent_name_key
    ON data_alarm_policy_groups(project_id, parent_id, name)
    WHERE parent_id IS NOT NULL;
