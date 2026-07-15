-- 修复早期开发库已记录 0060、但缺少后续采集任务字段与导入会话表的问题。
CREATE TABLE IF NOT EXISTS data_collector_import_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    driver_id text NOT NULL,
    status text NOT NULL DEFAULT 'preview' CHECK (status IN ('preview', 'committed', 'expired')),
    candidates jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(candidates) = 'array'),
    errors jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(errors) = 'array'),
    total_rows integer NOT NULL DEFAULT 0 CHECK (total_rows >= 0),
    valid_rows integer NOT NULL DEFAULT 0 CHECK (valid_rows >= 0 AND valid_rows <= total_rows),
    expires_at timestamptz NOT NULL,
    created_by uuid NOT NULL,
    committed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_collector_import_sessions_connection_fkey
        FOREIGN KEY (connection_id, project_id)
        REFERENCES data_collector_connections (id, project_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_collector_import_sessions_connection_created_idx
    ON data_collector_import_sessions (project_id, connection_id, created_at DESC);
CREATE INDEX IF NOT EXISTS data_collector_import_sessions_expiry_idx
    ON data_collector_import_sessions (status, expires_at)
    WHERE status = 'preview';

-- 调试任务均为短期数据，修复字段前清空，避免无法补齐 connection_id 的历史任务阻塞迁移。
DELETE FROM collector_dev_tasks;

ALTER TABLE collector_dev_tasks
    ADD COLUMN IF NOT EXISTS connection_id uuid;

ALTER TABLE collector_dev_tasks
    DROP CONSTRAINT IF EXISTS collector_dev_tasks_operation_check,
    DROP CONSTRAINT IF EXISTS collector_dev_tasks_connection_fkey;

ALTER TABLE collector_dev_tasks
    ALTER COLUMN connection_id SET NOT NULL,
    ADD CONSTRAINT collector_dev_tasks_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_collector_connections (id) ON DELETE CASCADE,
    ADD CONSTRAINT collector_dev_tasks_operation_check
        CHECK (operation IN ('connection.test', 'device.browse', 'point.read', 'point.write', 'point.subscribe.preview'));

CREATE INDEX IF NOT EXISTS collector_dev_tasks_connection_created_idx
    ON collector_dev_tasks (project_id, connection_id, created_at DESC);
