DROP INDEX IF EXISTS data_alarm_rules_project_updated_idx;

ALTER TABLE data_alarm_rules
    DROP CONSTRAINT IF EXISTS data_alarm_rules_rule_type_check;

UPDATE data_alarm_rules
SET rule_type = CASE
    WHEN rule_type IN ('H', 'L', 'HH', 'LL', 'deviation_high', 'deviation_low', 'rate_of_change') THEN 'threshold'
    WHEN rule_type = 'cel' THEN 'expression'
    ELSE rule_type
END;

ALTER TABLE data_alarm_rules
    ADD CONSTRAINT data_alarm_rules_rule_type_check
        CHECK (rule_type IN ('threshold', 'range', 'expression'));

ALTER TABLE data_alarm_rules
    DROP CONSTRAINT IF EXISTS data_alarm_rules_severity_check;

UPDATE data_alarm_rules
SET severity = 'warning'
WHERE severity = 'major';

ALTER TABLE data_alarm_rules
    ADD CONSTRAINT data_alarm_rules_severity_check
        CHECK (severity IN ('info', 'warning', 'critical'));

ALTER TABLE data_alarm_rules
    DROP CONSTRAINT IF EXISTS data_alarm_rules_suppression_check,
    DROP COLUMN IF EXISTS message_template,
    DROP COLUMN IF EXISTS suppression;
