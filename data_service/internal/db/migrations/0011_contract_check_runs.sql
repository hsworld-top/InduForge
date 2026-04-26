-- data_contract_check_runs: 保存开发态契约检查历史，便于 latest 接口返回可审计的最近一次结果。
CREATE TABLE IF NOT EXISTS data_contract_check_runs (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    project_id uuid NOT NULL,
    scope text NOT NULL DEFAULT 'project',
    object_type text,
    object_id text,
    status text NOT NULL CHECK (status IN ('passed', 'warning', 'pending', 'failed')),
    summary jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(summary) = 'object'),
    result jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(result) = 'object'),
    created_by uuid,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS data_contract_check_runs_project_created_idx
    ON data_contract_check_runs (project_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS data_contract_check_runs_project_status_idx
    ON data_contract_check_runs (project_id, status, created_at DESC);
