-- data_access_source_records: 保存开发态接入源操作摘要。
-- 说明：协议短时抓样只落摘要，不保存原始样本或 payload，避免敏感数据持久化。
CREATE TABLE IF NOT EXISTS data_access_source_records (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id text NOT NULL,
    connection_id uuid NOT NULL,
    record_type text NOT NULL CHECK (char_length(record_type) <= 100),
    title text NOT NULL CHECK (char_length(title) <= 200),
    status text NOT NULL CHECK (char_length(status) <= 50),
    protocol text NOT NULL CHECK (char_length(protocol) <= 50),
    duration_ms integer NOT NULL DEFAULT 0 CHECK (duration_ms >= 0),
    sample_count integer NOT NULL DEFAULT 0 CHECK (sample_count >= 0),
    truncated boolean NOT NULL DEFAULT false,
    error_summary text CHECK (error_summary IS NULL OR char_length(error_summary) <= 1000),
    detail jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(detail) = 'object'),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_access_source_records_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS data_access_source_records_connection_created_idx
    ON data_access_source_records (connection_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS data_access_source_records_project_created_idx
    ON data_access_source_records (project_id, created_at DESC, id DESC);
