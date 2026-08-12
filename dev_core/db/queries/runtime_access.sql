-- name: ListRuntimeRoles :many
SELECT r.id, r.project_id, r.code, r.name, r.description, r.status, r.is_builtin,
       r.created_at, r.updated_at,
       COALESCE(jsonb_agg(DISTINCT g.capability) FILTER (WHERE g.id IS NOT NULL AND g.effect = 'allow'), '[]'::jsonb) AS capabilities,
       (SELECT count(*) FROM project_user_role_bindings b WHERE b.role_id = r.id) AS user_count
FROM project_roles r
JOIN projects p ON p.id = r.project_id
LEFT JOIN project_role_grants g ON g.role_id = r.id
WHERE r.project_id = sqlc.arg(project_id) AND p.tenant_id = sqlc.arg(tenant_id) AND p.status <> 'deleted'
GROUP BY r.id
ORDER BY r.is_builtin DESC, r.created_at;

-- name: GetRuntimeRole :one
SELECT r.* FROM project_roles r JOIN projects p ON p.id = r.project_id
WHERE r.id = sqlc.arg(role_id) AND r.project_id = sqlc.arg(project_id) AND p.tenant_id = sqlc.arg(tenant_id) AND p.status <> 'deleted'
LIMIT 1;

-- name: CheckRuntimeRoleInProject :one
SELECT EXISTS (SELECT 1 FROM project_roles WHERE id = sqlc.arg(role_id) AND project_id = sqlc.arg(project_id));

-- name: CreateRuntimeRole :one
INSERT INTO project_roles (project_id, code, name, description, status, is_builtin, created_by)
VALUES (sqlc.arg(project_id), sqlc.arg(code), sqlc.arg(name), sqlc.narg(description), sqlc.arg(status), false, sqlc.arg(user_id))
RETURNING *;

-- name: UpdateRuntimeRole :one
UPDATE project_roles SET code = sqlc.arg(code), name = sqlc.arg(name), description = sqlc.narg(description),
  status = sqlc.arg(status), updated_by = sqlc.arg(user_id), updated_at = now()
WHERE id = sqlc.arg(role_id) AND project_id = sqlc.arg(project_id) AND is_builtin = false
RETURNING *;

-- name: DeleteRuntimeRole :execrows
DELETE FROM project_roles WHERE id = sqlc.arg(role_id) AND project_id = sqlc.arg(project_id) AND is_builtin = false;

-- name: DeleteRuntimeRoleGrants :exec
DELETE FROM project_role_grants WHERE role_id = sqlc.arg(role_id);

-- name: CreateRuntimeRoleGrant :exec
INSERT INTO project_role_grants (project_id, role_id, capability, effect)
VALUES (sqlc.arg(project_id), sqlc.arg(role_id), sqlc.arg(capability), 'allow');

-- name: ListRuntimeUsers :many
SELECT u.id, u.project_id, u.username, u.display_name, u.email, u.status, u.is_builtin_admin,
       u.last_login_at, u.password_changed_at, u.created_at, u.updated_at,
       COALESCE(jsonb_agg(DISTINCT jsonb_build_object('id', r.id, 'code', r.code, 'name', r.name, 'status', r.status, 'isBuiltin', r.is_builtin)) FILTER (WHERE r.id IS NOT NULL), '[]'::jsonb) AS roles
FROM project_runtime_users u
JOIN projects p ON p.id = u.project_id
LEFT JOIN project_user_role_bindings b ON b.runtime_user_id = u.id
LEFT JOIN project_roles r ON r.id = b.role_id
WHERE u.project_id = sqlc.arg(project_id) AND p.tenant_id = sqlc.arg(tenant_id) AND p.status <> 'deleted'
GROUP BY u.id
ORDER BY u.is_builtin_admin DESC, u.created_at;

-- name: GetRuntimeUser :one
SELECT u.* FROM project_runtime_users u JOIN projects p ON p.id = u.project_id
WHERE u.id = sqlc.arg(runtime_user_id) AND u.project_id = sqlc.arg(project_id) AND p.tenant_id = sqlc.arg(tenant_id) AND p.status <> 'deleted'
LIMIT 1;

-- name: CreateRuntimeUser :one
INSERT INTO project_runtime_users (project_id, username, password_hash, display_name, email, status, is_builtin_admin, created_by)
VALUES (sqlc.arg(project_id), sqlc.arg(username), sqlc.arg(password_hash), sqlc.narg(display_name), sqlc.narg(email), sqlc.arg(status), false, sqlc.arg(user_id))
RETURNING *;

-- name: UpdateRuntimeUserStatus :one
UPDATE project_runtime_users SET status = sqlc.arg(status), updated_by = sqlc.arg(user_id), updated_at = now()
WHERE id = sqlc.arg(runtime_user_id) AND project_id = sqlc.arg(project_id)
RETURNING *;

-- name: UpdateRuntimeUserPassword :exec
UPDATE project_runtime_users SET password_hash = sqlc.arg(password_hash), password_changed_at = now(), updated_by = sqlc.arg(user_id), updated_at = now()
WHERE id = sqlc.arg(runtime_user_id) AND project_id = sqlc.arg(project_id);

-- name: DeleteRuntimeUser :execrows
DELETE FROM project_runtime_users WHERE id = sqlc.arg(runtime_user_id) AND project_id = sqlc.arg(project_id) AND is_builtin_admin = false;

-- name: DeleteRuntimeUserRoleBindings :exec
DELETE FROM project_user_role_bindings WHERE runtime_user_id = sqlc.arg(runtime_user_id);

-- name: CreateRuntimeUserRoleBinding :exec
INSERT INTO project_user_role_bindings (runtime_user_id, role_id) VALUES (sqlc.arg(runtime_user_id), sqlc.arg(role_id));
