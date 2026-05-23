CREATE TABLE IF NOT EXISTS data_alarm_project_settings (
    project_id uuid PRIMARY KEY,
    escalation_interval_seconds integer NOT NULL DEFAULT 300,
    repeat_notification_interval_seconds integer NOT NULL DEFAULT 60,
    created_by uuid,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_alarm_project_settings_escalation_interval_check
        CHECK (escalation_interval_seconds >= 30),
    CONSTRAINT data_alarm_project_settings_repeat_interval_check
        CHECK (repeat_notification_interval_seconds >= 10)
);
