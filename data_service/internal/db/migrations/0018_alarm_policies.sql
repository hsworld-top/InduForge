CREATE TABLE IF NOT EXISTS data_alarm_policy_groups (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    name varchar(100) NOT NULL,
    description text,
    is_enabled boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_alarm_policy_groups_project_name_key UNIQUE (project_id, name)
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

CREATE INDEX IF NOT EXISTS data_alarm_policies_project_group_idx
    ON data_alarm_policies(project_id, group_id);

CREATE INDEX IF NOT EXISTS data_alarm_policies_project_updated_idx
    ON data_alarm_policies(project_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS data_alarm_policies_project_enabled_idx
    ON data_alarm_policies(project_id, is_enabled);
