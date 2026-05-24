-- 修正早期 0023 迁移已执行环境中的内置运行库模型：
-- 1. 删除项目内同类型唯一索引，允许一个工程创建多个同类型运行库实例；
-- 2. 为历史内置连接补齐运行态标识，避免连接级工作台无法解析空间。
DROP INDEX IF EXISTS data_connections_project_builtin_type_unique;

CREATE TABLE IF NOT EXISTS data_builtin_realtime_keys (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    key_path text NOT NULL,
    value_type text NOT NULL DEFAULT 'json',
    default_ttl_seconds integer NOT NULL DEFAULT 300,
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, connection_id, key_path),
    FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS data_builtin_message_topics (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    topic text NOT NULL,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, connection_id, topic),
    FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS data_builtin_message_variables (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    topic_id uuid NOT NULL,
    name text NOT NULL,
    payload_path text NOT NULL,
    value_type text NOT NULL DEFAULT 'string',
    unit text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    create_datapoint boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, topic_id, name),
    FOREIGN KEY (topic_id) REFERENCES data_builtin_message_topics (id) ON DELETE CASCADE
);

WITH builtin_connections AS (
    SELECT
        id,
        project_id,
        type,
        CASE type
            WHEN 'builtin.relation' THEN 'rel'
            WHEN 'builtin.timeseries' THEN 'ts'
            WHEN 'builtin.realtime' THEN 'rt'
            WHEN 'builtin.message' THEN 'msg'
            ELSE 'store'
        END AS prefix,
        metadata
    FROM data_connections
    WHERE type IN ('builtin.relation', 'builtin.timeseries', 'builtin.realtime', 'builtin.message')
),
runtime_keys AS (
    SELECT
        id,
        project_id,
        type,
        prefix,
        metadata,
        COALESCE(
            NULLIF(metadata ->> 'runtimeKey', ''),
            NULLIF(metadata ->> 'runtimeSchema', ''),
            NULLIF(metadata ->> 'namespace', ''),
            NULLIF(metadata ->> 'topicPrefix', ''),
            prefix || '_' || left(replace(id::text, '-', ''), 6)
        ) AS runtime_key
    FROM builtin_connections
),
next_metadata AS (
    SELECT
        id,
        CASE type
            WHEN 'builtin.relation' THEN
                metadata
                || jsonb_build_object(
                    'store', 'relation',
                    'runtimeKey', runtime_key,
                    'devSchema', COALESCE(NULLIF(metadata ->> 'devSchema', ''), 'p_' || replace(project_id::text, '-', '_') || '_' || runtime_key),
                    'runtimeSchema', runtime_key,
                    'ddlVersion', COALESCE(NULLIF(metadata ->> 'ddlVersion', ''), '2026-05-24.1')
                )
            WHEN 'builtin.timeseries' THEN
                metadata
                || jsonb_build_object(
                    'store', 'timeseries',
                    'runtimeKey', runtime_key,
                    'devSchema', COALESCE(NULLIF(metadata ->> 'devSchema', ''), 'p_' || replace(project_id::text, '-', '_') || '_' || runtime_key),
                    'runtimeSchema', runtime_key,
                    'retentionDays', COALESCE((metadata ->> 'retentionDays')::integer, 30),
                    'ddlVersion', COALESCE(NULLIF(metadata ->> 'ddlVersion', ''), '2026-05-24.1')
                )
            WHEN 'builtin.realtime' THEN
                metadata
                || jsonb_build_object(
                    'store', 'realtime',
                    'runtimeKey', runtime_key,
                    'namespace', runtime_key,
                    'defaultTtlSeconds', COALESCE((metadata ->> 'defaultTtlSeconds')::integer, 300)
                )
            WHEN 'builtin.message' THEN
                metadata
                || jsonb_build_object(
                    'store', 'message',
                    'runtimeKey', runtime_key,
                    'topicPrefix', runtime_key
                )
            ELSE metadata
        END AS metadata
    FROM runtime_keys
)
UPDATE data_connections AS c
SET metadata = n.metadata,
    updated_at = now()
FROM next_metadata AS n
WHERE c.id = n.id;
