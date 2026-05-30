-- data_alarm_rules: 保存开发态报警规则草稿与运行态契约预览所需的稳定字段。
CREATE TABLE IF NOT EXISTS data_alarm_rules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    name text NOT NULL CHECK (char_length(name) <= 100),
    description text,
    target_datapoint_id uuid,
    target_path text NOT NULL CHECK (char_length(target_path) <= 255),
    rule_type text NOT NULL DEFAULT 'H' CHECK (rule_type IN ('H', 'L', 'HH', 'LL', 'deviation_high', 'deviation_low', 'rate_of_change', 'cel')),
    condition jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(condition) = 'object'),
    severity text NOT NULL DEFAULT 'warning' CHECK (severity IN ('info', 'warning', 'major', 'critical')),
    hysteresis double precision CHECK (hysteresis IS NULL OR hysteresis >= 0),
    sample_window_ms integer CHECK (sample_window_ms IS NULL OR sample_window_ms >= 0),
    suppression jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(suppression) = 'object'),
    message_template text NOT NULL DEFAULT '',
    contract jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(contract) = 'object'),
    is_enabled boolean NOT NULL DEFAULT true,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_alarm_rules_project_name_key UNIQUE (project_id, name),
    CONSTRAINT data_alarm_rules_target_datapoint_fkey
        FOREIGN KEY (target_datapoint_id) REFERENCES data_points (id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS data_alarm_rules_project_enabled_idx
    ON data_alarm_rules (project_id, is_enabled);

CREATE INDEX IF NOT EXISTS data_alarm_rules_project_target_path_idx
    ON data_alarm_rules (project_id, target_path);

CREATE INDEX IF NOT EXISTS data_alarm_rules_project_updated_idx
    ON data_alarm_rules (project_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS data_alarm_rules_target_datapoint_idx
    ON data_alarm_rules (target_datapoint_id);

CREATE INDEX IF NOT EXISTS data_alarm_rules_condition_gin_idx
    ON data_alarm_rules USING GIN (condition);

CREATE TABLE IF NOT EXISTS data_alarm_policy_groups (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    parent_id uuid,
    name varchar(100) NOT NULL,
    description text,
    is_enabled boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_alarm_policy_groups_parent_fkey
        FOREIGN KEY (parent_id) REFERENCES data_alarm_policy_groups(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS data_alarm_policies (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    group_id uuid REFERENCES data_alarm_policy_groups(id) ON DELETE SET NULL,
    name varchar(100) NOT NULL,
    description text,
    mode varchar(20) NOT NULL DEFAULT 'per_target',
    targets jsonb NOT NULL DEFAULT '[]'::jsonb,
    inputs jsonb NOT NULL DEFAULT '[]'::jsonb,
    derived_expression text NOT NULL DEFAULT '',
    conditions jsonb NOT NULL DEFAULT '[]'::jsonb,
    suppression jsonb NOT NULL DEFAULT '{}'::jsonb,
    message_template text NOT NULL DEFAULT '',
    is_enabled boolean NOT NULL DEFAULT true,
    contract jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_by uuid,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_alarm_policies_project_name_key UNIQUE (project_id, name),
    CONSTRAINT data_alarm_policies_mode_check CHECK (mode IN ('per_target', 'derived')),
    CONSTRAINT data_alarm_policies_targets_array_check CHECK (jsonb_typeof(targets) = 'array'),
    CONSTRAINT data_alarm_policies_inputs_array_check CHECK (jsonb_typeof(inputs) = 'array'),
    CONSTRAINT data_alarm_policies_conditions_array_check CHECK (jsonb_typeof(conditions) = 'array'),
    CONSTRAINT data_alarm_policies_suppression_object_check CHECK (jsonb_typeof(suppression) = 'object'),
    CONSTRAINT data_alarm_policies_contract_object_check CHECK (jsonb_typeof(contract) = 'object')
);

CREATE INDEX IF NOT EXISTS data_alarm_policy_groups_project_sort_idx
    ON data_alarm_policy_groups(project_id, sort_order, created_at);

CREATE INDEX IF NOT EXISTS data_alarm_policy_groups_project_parent_idx
    ON data_alarm_policy_groups(project_id, parent_id, sort_order, created_at);

CREATE UNIQUE INDEX IF NOT EXISTS data_alarm_policy_groups_project_root_name_key
    ON data_alarm_policy_groups(project_id, name)
    WHERE parent_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS data_alarm_policy_groups_project_parent_name_key
    ON data_alarm_policy_groups(project_id, parent_id, name)
    WHERE parent_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS data_alarm_policies_project_group_idx
    ON data_alarm_policies(project_id, group_id);

CREATE INDEX IF NOT EXISTS data_alarm_policies_project_updated_idx
    ON data_alarm_policies(project_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS data_alarm_policies_project_enabled_idx
    ON data_alarm_policies(project_id, is_enabled);
