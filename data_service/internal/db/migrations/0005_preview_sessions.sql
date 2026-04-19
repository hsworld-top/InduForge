-- data_preview_sessions: 记录开发态 preview 会话审计快照。
-- 会话实时状态放在 Redis，这里主要保存创建/续期/关闭的状态切面，便于追溯。
CREATE TABLE IF NOT EXISTS data_preview_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    user_id uuid NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'expired', 'closed', 'error')),
    started_at timestamptz NOT NULL DEFAULT now(),
    last_active_at timestamptz NOT NULL DEFAULT now(),
    expired_at timestamptz NOT NULL,
    meta jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(meta) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- 主要查询路径：项目+用户+状态（用于会话列表与权限边界校验）。
CREATE INDEX IF NOT EXISTS data_preview_sessions_project_user_status_idx
    ON data_preview_sessions (project_id, user_id, status);

-- 过期扫描与清理任务按 last_active_at 拉链。
CREATE INDEX IF NOT EXISTS data_preview_sessions_last_active_at_idx
    ON data_preview_sessions (last_active_at);
