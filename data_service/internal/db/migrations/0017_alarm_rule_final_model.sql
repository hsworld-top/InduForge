ALTER TABLE data_alarm_rules
    ADD COLUMN IF NOT EXISTS suppression jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS message_template text NOT NULL DEFAULT '';

ALTER TABLE data_alarm_rules
    DROP CONSTRAINT IF EXISTS data_alarm_rules_suppression_check,
    ADD CONSTRAINT data_alarm_rules_suppression_check
        CHECK (jsonb_typeof(suppression) = 'object');

ALTER TABLE data_alarm_rules
    DROP CONSTRAINT IF EXISTS data_alarm_rules_rule_type_check;

UPDATE data_alarm_rules
SET rule_type = CASE
    WHEN rule_type = 'threshold' THEN 'H'
    WHEN rule_type = 'range' THEN 'H'
    WHEN rule_type = 'expression' THEN 'cel'
    ELSE rule_type
END;

ALTER TABLE data_alarm_rules
    ADD CONSTRAINT data_alarm_rules_rule_type_check
        CHECK (rule_type IN ('H', 'L', 'HH', 'LL', 'deviation_high', 'deviation_low', 'rate_of_change', 'cel'));

ALTER TABLE data_alarm_rules
    DROP CONSTRAINT IF EXISTS data_alarm_rules_severity_check,
    ADD CONSTRAINT data_alarm_rules_severity_check
        CHECK (severity IN ('info', 'warning', 'major', 'critical'));

CREATE INDEX IF NOT EXISTS data_alarm_rules_project_updated_idx
    ON data_alarm_rules (project_id, updated_at DESC);
