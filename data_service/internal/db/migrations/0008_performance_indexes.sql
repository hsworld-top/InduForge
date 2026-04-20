-- 为常用项目级快照与列表路径补充复合索引，避免 project_id 过滤后再额外排序。
CREATE INDEX IF NOT EXISTS data_connections_project_created_idx
    ON data_connections (project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS data_queries_project_created_idx
    ON data_queries (project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS data_points_project_path_idx
    ON data_points (project_id, path);

CREATE INDEX IF NOT EXISTS data_points_project_created_idx
    ON data_points (project_id, created_at DESC);

CREATE INDEX IF NOT EXISTS data_mqtt_subscriptions_project_created_idx
    ON data_mqtt_subscriptions (project_id, created_at DESC);
