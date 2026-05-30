-- data_compute_units: 保存计算单元定义（脚本语言、触发策略、输入输出映射）。
CREATE TABLE IF NOT EXISTS data_compute_units (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    name text NOT NULL CHECK (char_length(name) <= 100),
    language text NOT NULL CHECK (language IN ('js', 'python')),
    description text,
    folder_id uuid,
    script_code text NOT NULL,
    dependencies jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(dependencies) = 'array'),
    trigger_type text NOT NULL DEFAULT 'manual' CHECK (trigger_type IN ('manual', 'timer', 'datapoint_change')),
    trigger_config jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(trigger_config) = 'object'),
    input_bindings jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(input_bindings) = 'object'),
    output_bindings jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(output_bindings) = 'object'),
    timeout_ms integer NOT NULL DEFAULT 3000 CHECK (timeout_ms > 0 AND timeout_ms <= 120000),
    is_enabled boolean NOT NULL DEFAULT true,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_compute_units_project_name_key UNIQUE (project_id, name)
);

CREATE INDEX IF NOT EXISTS data_compute_units_project_enabled_idx
    ON data_compute_units (project_id, is_enabled);

CREATE INDEX IF NOT EXISTS data_compute_units_project_language_idx
    ON data_compute_units (project_id, language);

CREATE INDEX IF NOT EXISTS data_compute_units_project_folder_idx
    ON data_compute_units (project_id, folder_id, created_at DESC);

-- data_compute_runs: 保存运行与调试记录，支持超时/失败审计。
CREATE TABLE IF NOT EXISTS data_compute_runs (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    project_id uuid NOT NULL,
    compute_unit_id uuid NOT NULL,
    trigger_mode text NOT NULL CHECK (trigger_mode IN ('run', 'debug', 'schedule')),
    status text NOT NULL CHECK (status IN ('success', 'timeout', 'failed')),
    duration_ms integer NOT NULL DEFAULT 0 CHECK (duration_ms >= 0),
    output jsonb NOT NULL DEFAULT '{}'::jsonb,
    error_message text,
    started_at timestamptz NOT NULL,
    finished_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_compute_runs_compute_unit_fkey
        FOREIGN KEY (compute_unit_id) REFERENCES data_compute_units (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_compute_runs_unit_created_idx
    ON data_compute_runs (compute_unit_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS data_compute_runs_project_created_idx
    ON data_compute_runs (project_id, created_at DESC, id DESC);
