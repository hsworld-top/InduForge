-- name: ListTenants :many
SELECT
  t.*,
  (SELECT count(*) FROM users u WHERE u.tenant_id = t.id AND u.role <> 'SUPER_ADMIN') AS user_count,
  (SELECT count(*) FROM projects p WHERE p.tenant_id = t.id AND p.status <> 'deleted') AS project_count
FROM tenants t
WHERE (sqlc.arg(keyword)::text = '' OR t.name ILIKE '%' || sqlc.arg(keyword) || '%' OR t.code ILIKE '%' || sqlc.arg(keyword) || '%')
  AND (sqlc.arg(status)::text = '' OR t.status = sqlc.arg(status))
ORDER BY t.created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountFilteredTenants :one
SELECT count(*)
FROM tenants t
WHERE (sqlc.arg(keyword)::text = '' OR t.name ILIKE '%' || sqlc.arg(keyword) || '%' OR t.code ILIKE '%' || sqlc.arg(keyword) || '%')
  AND (sqlc.arg(status)::text = '' OR t.status = sqlc.arg(status));

-- name: GetTenantByIdentifier :one
SELECT
  t.*,
  (SELECT count(*) FROM users u WHERE u.tenant_id = t.id AND u.role <> 'SUPER_ADMIN') AS user_count,
  (SELECT count(*) FROM projects p WHERE p.tenant_id = t.id AND p.status <> 'deleted') AS project_count
FROM tenants t
WHERE t.id::text = sqlc.arg(identifier)::text OR lower(t.code) = lower(sqlc.arg(identifier)::text)
LIMIT 1;

-- name: CreateTenant :one
INSERT INTO tenants (
  name, code, description, status, contact_email, contact_phone,
  max_users, max_projects, max_storage, logo_object_key, login_background_object_key,
  company_name, company_address, company_phone, company_website, settings, expires_at
) VALUES (
  sqlc.arg(name), sqlc.arg(code), sqlc.narg(description), sqlc.arg(status),
  sqlc.narg(contact_email), sqlc.narg(contact_phone), sqlc.arg(max_users),
  sqlc.arg(max_projects), sqlc.arg(max_storage), sqlc.narg(logo_object_key),
  sqlc.narg(login_background_object_key), sqlc.narg(company_name), sqlc.narg(company_address),
  sqlc.narg(company_phone), sqlc.narg(company_website), sqlc.arg(settings), sqlc.narg(expires_at)
)
RETURNING *;

-- name: UpdateTenant :one
UPDATE tenants
SET name = sqlc.arg(name),
    code = sqlc.arg(code),
    description = sqlc.narg(description),
    status = sqlc.arg(status),
    contact_email = sqlc.narg(contact_email),
    contact_phone = sqlc.narg(contact_phone),
    max_users = sqlc.arg(max_users),
    max_projects = sqlc.arg(max_projects),
    max_storage = sqlc.arg(max_storage),
    logo_object_key = sqlc.narg(logo_object_key),
    login_background_object_key = sqlc.narg(login_background_object_key),
    company_name = sqlc.narg(company_name),
    company_address = sqlc.narg(company_address),
    company_phone = sqlc.narg(company_phone),
    company_website = sqlc.narg(company_website),
    settings = sqlc.arg(settings),
    expires_at = sqlc.narg(expires_at),
    updated_at = now()
WHERE id = sqlc.arg(tenant_id)
RETURNING *;

-- name: SetTenantStatus :one
UPDATE tenants SET status = sqlc.arg(status), updated_at = now()
WHERE id = sqlc.arg(tenant_id)
RETURNING *;

-- name: DeleteTenant :execrows
DELETE FROM tenants WHERE id = sqlc.arg(tenant_id);

-- name: ListDashboardNotes :many
SELECT n.id, n.tenant_id, n.content, n.created_by, n.updated_by, n.created_at, n.updated_at,
       creator.username AS created_by_username,
       updater.username AS updated_by_username
FROM tenant_dashboard_notes n
JOIN users creator ON creator.id = n.created_by
LEFT JOIN users updater ON updater.id = n.updated_by
WHERE n.tenant_id = sqlc.arg(tenant_id)
ORDER BY n.updated_at DESC;

-- name: CreateDashboardNote :one
INSERT INTO tenant_dashboard_notes (tenant_id, content, created_by)
VALUES (sqlc.arg(tenant_id), sqlc.arg(content), sqlc.arg(user_id))
RETURNING *;

-- name: UpdateDashboardNote :one
UPDATE tenant_dashboard_notes
SET content = sqlc.arg(content), updated_by = sqlc.arg(user_id), updated_at = now()
WHERE id = sqlc.arg(note_id) AND tenant_id = sqlc.arg(tenant_id)
RETURNING *;

-- name: DeleteDashboardNote :execrows
DELETE FROM tenant_dashboard_notes
WHERE id = sqlc.arg(note_id) AND tenant_id = sqlc.arg(tenant_id);

-- name: ListUsers :many
SELECT u.id, u.tenant_id, u.username, u.email, u.phone, u.full_name, u.avatar,
       u.role, u.status, u.preferences, u.last_login_at, u.last_login_ip,
       u.password_changed_at, u.created_at, u.updated_at
FROM users u
WHERE u.tenant_id = sqlc.arg(tenant_id)
  AND u.role <> 'SUPER_ADMIN'
  AND (sqlc.arg(keyword)::text = '' OR u.username ILIKE '%' || sqlc.arg(keyword) || '%' OR u.full_name ILIKE '%' || sqlc.arg(keyword) || '%' OR u.email ILIKE '%' || sqlc.arg(keyword) || '%')
  AND (sqlc.arg(role)::text = '' OR u.role = sqlc.arg(role))
  AND (sqlc.arg(status)::text = '' OR u.status = sqlc.arg(status))
ORDER BY u.created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountUsers :one
SELECT count(*)
FROM users u
WHERE u.tenant_id = sqlc.arg(tenant_id)
  AND u.role <> 'SUPER_ADMIN'
  AND (sqlc.arg(keyword)::text = '' OR u.username ILIKE '%' || sqlc.arg(keyword) || '%' OR u.full_name ILIKE '%' || sqlc.arg(keyword) || '%' OR u.email ILIKE '%' || sqlc.arg(keyword) || '%')
  AND (sqlc.arg(role)::text = '' OR u.role = sqlc.arg(role))
  AND (sqlc.arg(status)::text = '' OR u.status = sqlc.arg(status));

-- name: GetManagedUser :one
SELECT * FROM users WHERE id = sqlc.arg(user_id) AND tenant_id = sqlc.arg(tenant_id) LIMIT 1;

-- name: CreateManagedUser :one
INSERT INTO users (tenant_id, username, password_hash, email, phone, full_name, avatar, role, status, preferences)
VALUES (sqlc.arg(tenant_id), sqlc.arg(username), sqlc.arg(password_hash), sqlc.narg(email),
        sqlc.narg(phone), sqlc.narg(full_name), sqlc.narg(avatar), sqlc.arg(role),
        sqlc.arg(status), sqlc.arg(preferences))
RETURNING *;

-- name: UpdateManagedUser :one
UPDATE users
SET email = sqlc.narg(email), phone = sqlc.narg(phone), full_name = sqlc.narg(full_name),
    avatar = sqlc.narg(avatar), role = sqlc.arg(role), status = sqlc.arg(status),
    preferences = sqlc.arg(preferences), updated_at = now()
WHERE id = sqlc.arg(user_id) AND tenant_id = sqlc.arg(tenant_id)
RETURNING *;

-- name: DeleteManagedUser :execrows
DELETE FROM users WHERE id = sqlc.arg(user_id) AND tenant_id = sqlc.arg(tenant_id);

-- name: CountTenantUsers :one
SELECT count(*) FROM users WHERE tenant_id = sqlc.arg(tenant_id) AND role <> 'SUPER_ADMIN';
