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
