-- name: FindAuthUsersByUsername :many
SELECT
  u.id,
  u.tenant_id,
  u.username,
  u.password_hash,
  u.email,
  u.full_name,
  u.role,
  u.status,
  t.code AS tenant_code,
  t.name AS tenant_name,
  t.status AS tenant_status,
  t.logo_object_key,
  t.login_background_object_key
FROM users u
JOIN tenants t ON t.id = u.tenant_id
WHERE lower(u.username) = lower(sqlc.arg(username))
ORDER BY u.created_at
LIMIT 2;

-- name: FindAuthUserByTenantCode :one
SELECT
  u.id,
  u.tenant_id,
  u.username,
  u.password_hash,
  u.email,
  u.full_name,
  u.role,
  u.status,
  t.code AS tenant_code,
  t.name AS tenant_name,
  t.status AS tenant_status,
  t.logo_object_key,
  t.login_background_object_key
FROM users u
JOIN tenants t ON t.id = u.tenant_id
WHERE lower(u.username) = lower(sqlc.arg(username))
  AND lower(t.code) = lower(sqlc.arg(tenant_code))
LIMIT 1;

-- name: GetAuthUserByID :one
SELECT
  u.id,
  u.tenant_id,
  u.username,
  u.password_hash,
  u.email,
  u.full_name,
  u.role,
  u.status,
  t.code AS tenant_code,
  t.name AS tenant_name,
  t.status AS tenant_status,
  t.logo_object_key,
  t.login_background_object_key
FROM users u
JOIN tenants t ON t.id = u.tenant_id
WHERE u.id = sqlc.arg(user_id)
LIMIT 1;

-- name: GetTenantBrandingByCode :one
SELECT id, code, name, logo_object_key, login_background_object_key
FROM tenants
WHERE lower(code) = lower(sqlc.arg(tenant_code)) AND status = 'active'
LIMIT 1;

-- name: UpdateAuthUserLogin :exec
UPDATE users
SET last_login_at = now(),
    last_login_ip = sqlc.arg(login_ip),
    updated_at = now()
WHERE id = sqlc.arg(user_id);

-- name: UpdateAuthUserPassword :exec
UPDATE users
SET password_hash = sqlc.arg(password_hash),
    password_changed_at = now(),
    updated_at = now()
WHERE id = sqlc.arg(user_id);

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
  tenant_id,
  user_id,
  token_hash,
  expires_at
) VALUES (
  sqlc.arg(tenant_id),
  sqlc.arg(user_id),
  sqlc.arg(token_hash),
  sqlc.arg(expires_at)
)
RETURNING id;

-- name: GetRefreshTokenByHash :one
SELECT id, tenant_id, user_id, token_hash, expires_at, revoked_at
FROM refresh_tokens
WHERE token_hash = sqlc.arg(token_hash)
LIMIT 1;

-- name: ConsumeRefreshToken :one
UPDATE refresh_tokens
SET revoked_at = now(), last_used_at = now()
WHERE token_hash = sqlc.arg(token_hash)
  AND revoked_at IS NULL
  AND expires_at > now()
RETURNING id, tenant_id, user_id;

-- name: SetRefreshTokenReplacement :exec
UPDATE refresh_tokens
SET replaced_by_token_id = sqlc.arg(replaced_by_token_id)
WHERE id = sqlc.arg(token_id)
  AND revoked_at IS NOT NULL;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = COALESCE(revoked_at, now()),
    last_used_at = now()
WHERE token_hash = sqlc.arg(token_hash);

-- name: RevokeUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked_at = COALESCE(revoked_at, now())
WHERE user_id = sqlc.arg(user_id)
  AND revoked_at IS NULL;
