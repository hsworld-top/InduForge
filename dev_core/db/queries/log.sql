-- name: CreateAuditLog :exec
INSERT INTO audit_logs (
  id, tenant_id, user_id, level, action, resource, resource_id, message,
  request_id, method, path, result, ip, user_agent, metadata, created_at
) VALUES (
  sqlc.arg(id), sqlc.arg(tenant_id), sqlc.narg(user_id), sqlc.arg(level), sqlc.arg(action),
  sqlc.arg(resource), sqlc.arg(resource_id), sqlc.arg(message), sqlc.arg(request_id),
  sqlc.arg(method), sqlc.arg(path), sqlc.arg(result), sqlc.arg(ip), sqlc.arg(user_agent),
  sqlc.arg(metadata), sqlc.arg(created_at)
);

-- name: ListAuditLogs :many
SELECT l.*, u.username, u.full_name
FROM audit_logs l
LEFT JOIN users u ON u.id = l.user_id
WHERE (l.method IS DISTINCT FROM 'GET' OR l.action = 'export')
  AND l.tenant_id = sqlc.arg(tenant_id)
  AND (sqlc.arg(level)::text = '' OR l.level = sqlc.arg(level))
  AND (sqlc.arg(action)::text = '' OR COALESCE(l.action, '') = sqlc.arg(action))
  AND (sqlc.arg(resource)::text = '' OR COALESCE(l.resource, '') = sqlc.arg(resource))
  AND (sqlc.arg(user_id)::text = '' OR l.user_id::text = sqlc.arg(user_id))
  AND (sqlc.arg(keyword)::text = '' OR l.message ILIKE '%' || sqlc.arg(keyword) || '%' OR COALESCE(l.action, '') ILIKE '%' || sqlc.arg(keyword) || '%' OR COALESCE(l.resource, '') ILIKE '%' || sqlc.arg(keyword) || '%')
  AND (sqlc.narg(start_time)::timestamptz IS NULL OR l.created_at >= sqlc.narg(start_time))
  AND (sqlc.narg(end_time)::timestamptz IS NULL OR l.created_at <= sqlc.narg(end_time))
ORDER BY l.created_at DESC, l.id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountAuditLogs :one
SELECT count(*) FROM audit_logs l
WHERE (l.method IS DISTINCT FROM 'GET' OR l.action = 'export')
  AND l.tenant_id = sqlc.arg(tenant_id)
  AND (sqlc.arg(level)::text = '' OR l.level = sqlc.arg(level))
  AND (sqlc.arg(action)::text = '' OR COALESCE(l.action, '') = sqlc.arg(action))
  AND (sqlc.arg(resource)::text = '' OR COALESCE(l.resource, '') = sqlc.arg(resource))
  AND (sqlc.arg(user_id)::text = '' OR l.user_id::text = sqlc.arg(user_id))
  AND (sqlc.arg(keyword)::text = '' OR l.message ILIKE '%' || sqlc.arg(keyword) || '%' OR COALESCE(l.action, '') ILIKE '%' || sqlc.arg(keyword) || '%' OR COALESCE(l.resource, '') ILIKE '%' || sqlc.arg(keyword) || '%')
  AND (sqlc.narg(start_time)::timestamptz IS NULL OR l.created_at >= sqlc.narg(start_time))
  AND (sqlc.narg(end_time)::timestamptz IS NULL OR l.created_at <= sqlc.narg(end_time));

-- name: GetAuditLog :one
SELECT l.*, u.username, u.full_name
FROM audit_logs l
LEFT JOIN users u ON u.id = l.user_id
WHERE l.id = sqlc.arg(log_id) AND l.tenant_id = sqlc.arg(tenant_id)
LIMIT 1;

-- name: DeleteAuditLogsBefore :execrows
DELETE FROM audit_logs
WHERE (method IS DISTINCT FROM 'GET' OR action = 'export')
  AND tenant_id = sqlc.arg(tenant_id) AND created_at < sqlc.arg(before_time);

-- name: ExportAuditLogs :many
SELECT l.*, u.username, u.full_name
FROM audit_logs l
LEFT JOIN users u ON u.id = l.user_id
WHERE (l.method IS DISTINCT FROM 'GET' OR l.action = 'export')
  AND l.tenant_id = sqlc.arg(tenant_id)
  AND (sqlc.arg(level)::text = '' OR l.level = sqlc.arg(level))
  AND (sqlc.arg(action)::text = '' OR COALESCE(l.action, '') = sqlc.arg(action))
  AND (sqlc.arg(resource)::text = '' OR COALESCE(l.resource, '') = sqlc.arg(resource))
  AND (sqlc.arg(user_id)::text = '' OR l.user_id::text = sqlc.arg(user_id))
  AND (sqlc.arg(keyword)::text = '' OR l.message ILIKE '%' || sqlc.arg(keyword) || '%' OR COALESCE(l.action, '') ILIKE '%' || sqlc.arg(keyword) || '%' OR COALESCE(l.resource, '') ILIKE '%' || sqlc.arg(keyword) || '%')
  AND (sqlc.narg(start_time)::timestamptz IS NULL OR l.created_at >= sqlc.narg(start_time))
  AND (sqlc.narg(end_time)::timestamptz IS NULL OR l.created_at <= sqlc.narg(end_time))
ORDER BY l.created_at DESC, l.id DESC
LIMIT 10000;

-- name: GetAuditLogLevelStats :many
SELECT level, count(*) AS count
FROM audit_logs
WHERE (method IS DISTINCT FROM 'GET' OR action = 'export')
  AND tenant_id = sqlc.arg(tenant_id)
  AND (sqlc.narg(start_time)::timestamptz IS NULL OR created_at >= sqlc.narg(start_time))
  AND (sqlc.narg(end_time)::timestamptz IS NULL OR created_at <= sqlc.narg(end_time))
GROUP BY level;

-- name: GetAuditLogTrendStats :many
SELECT to_char(date_trunc('day', created_at), 'YYYY-MM-DD') AS date, count(*) AS count
FROM audit_logs
WHERE (method IS DISTINCT FROM 'GET' OR action = 'export')
  AND tenant_id = sqlc.arg(tenant_id)
  AND created_at >= COALESCE(sqlc.narg(start_time)::timestamptz, now() - interval '7 days')
  AND (sqlc.narg(end_time)::timestamptz IS NULL OR created_at <= sqlc.narg(end_time))
GROUP BY date_trunc('day', created_at)
ORDER BY date_trunc('day', created_at);

-- name: GetAuditLogActionStats :many
SELECT COALESCE(action, '') AS action, count(*) AS count
FROM audit_logs
WHERE (method IS DISTINCT FROM 'GET' OR action = 'export')
  AND tenant_id = sqlc.arg(tenant_id)
  AND (sqlc.narg(start_time)::timestamptz IS NULL OR created_at >= sqlc.narg(start_time))
  AND (sqlc.narg(end_time)::timestamptz IS NULL OR created_at <= sqlc.narg(end_time))
GROUP BY action
ORDER BY count(*) DESC
LIMIT 10;

-- name: ListRecentAuditActivities :many
SELECT l.id, l.action, l.resource, l.path, l.created_at, u.id AS user_id, u.username, u.full_name
FROM audit_logs l
LEFT JOIN users u ON u.id = l.user_id
WHERE (l.method IS DISTINCT FROM 'GET' OR l.action = 'export')
  AND l.tenant_id = sqlc.arg(tenant_id)
  AND l.result = 'success'
  AND l.action IN ('create', 'update', 'delete')
  AND l.resource NOT IN ('auth', 'logs')
ORDER BY l.created_at DESC, l.id DESC
LIMIT sqlc.arg(item_limit);
