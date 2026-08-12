-- name: ListProjects :many
SELECT p.id, p.tenant_id, p.name, p.code, p.description, p.icon, p.workspace_path,
       p.status, p.visibility, p.created_by, p.updated_by, p.archived_at, p.created_at, p.updated_at,
       creator.username AS created_by_name,
       g.id AS group_id, g.name AS group_name,
       COALESCE(jsonb_agg(DISTINCT jsonb_build_object('id', t.id, 'name', t.name, 'color', t.color, 'description', t.description, 'sortOrder', t.sort_order)) FILTER (WHERE t.id IS NOT NULL), '[]'::jsonb) AS tags
FROM projects p
JOIN users creator ON creator.id = p.created_by
LEFT JOIN project_group_members gm ON gm.project_id = p.id
LEFT JOIN project_groups g ON g.id = gm.group_id
LEFT JOIN project_tag_bindings tb ON tb.project_id = p.id
LEFT JOIN project_tags t ON t.id = tb.tag_id
WHERE p.tenant_id = sqlc.arg(tenant_id)
  AND p.status <> 'deleted'
  AND (
    sqlc.arg(is_platform_admin)::boolean
    OR p.created_by = sqlc.arg(actor_id)
    OR (sqlc.arg(can_read_shared)::boolean AND p.visibility = 'internal')
  )
  AND (sqlc.arg(keyword)::text = '' OR p.name ILIKE '%' || sqlc.arg(keyword) || '%' OR p.code ILIKE '%' || sqlc.arg(keyword) || '%')
  AND (sqlc.arg(status)::text = '' OR p.status = sqlc.arg(status))
  AND (sqlc.arg(visibility)::text = '' OR p.visibility = sqlc.arg(visibility))
  AND (sqlc.arg(group_id)::text = '' OR g.id::text = sqlc.arg(group_id))
  AND (sqlc.arg(tag_id)::text = '' OR EXISTS (SELECT 1 FROM project_tag_bindings filter_tb WHERE filter_tb.project_id = p.id AND filter_tb.tag_id::text = sqlc.arg(tag_id)))
GROUP BY p.id, creator.username, g.id, g.name
ORDER BY p.updated_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountProjects :one
SELECT count(DISTINCT p.id)
FROM projects p
LEFT JOIN project_group_members gm ON gm.project_id = p.id
LEFT JOIN project_groups g ON g.id = gm.group_id
WHERE p.tenant_id = sqlc.arg(tenant_id)
  AND p.status <> 'deleted'
  AND (
    sqlc.arg(is_platform_admin)::boolean
    OR p.created_by = sqlc.arg(actor_id)
    OR (sqlc.arg(can_read_shared)::boolean AND p.visibility = 'internal')
  )
  AND (sqlc.arg(keyword)::text = '' OR p.name ILIKE '%' || sqlc.arg(keyword) || '%' OR p.code ILIKE '%' || sqlc.arg(keyword) || '%')
  AND (sqlc.arg(status)::text = '' OR p.status = sqlc.arg(status))
  AND (sqlc.arg(visibility)::text = '' OR p.visibility = sqlc.arg(visibility))
  AND (sqlc.arg(group_id)::text = '' OR g.id::text = sqlc.arg(group_id))
  AND (sqlc.arg(tag_id)::text = '' OR EXISTS (SELECT 1 FROM project_tag_bindings filter_tb WHERE filter_tb.project_id = p.id AND filter_tb.tag_id::text = sqlc.arg(tag_id)));

-- name: GetProject :one
SELECT * FROM projects WHERE id = sqlc.arg(project_id) AND tenant_id = sqlc.arg(tenant_id) AND status <> 'deleted' LIMIT 1;

-- name: CreateProject :one
INSERT INTO projects (id, tenant_id, name, code, description, icon, workspace_path, status, visibility, created_by)
VALUES (sqlc.arg(project_id), sqlc.arg(tenant_id), sqlc.arg(name), sqlc.narg(code), sqlc.narg(description), sqlc.narg(icon), sqlc.arg(workspace_path), 'active', sqlc.arg(visibility), sqlc.arg(user_id))
RETURNING *;

-- name: UpdateProject :one
UPDATE projects
SET name = sqlc.arg(name), code = sqlc.narg(code), description = sqlc.narg(description),
    icon = sqlc.narg(icon), visibility = sqlc.arg(visibility), updated_by = sqlc.arg(user_id), updated_at = now()
WHERE id = sqlc.arg(project_id) AND tenant_id = sqlc.arg(tenant_id) AND status <> 'deleted'
RETURNING *;

-- name: SetProjectLifecycleStatus :one
UPDATE projects
SET status = sqlc.arg(status), archived_at = CASE WHEN sqlc.arg(status) = 'archived' THEN now() ELSE NULL END,
    updated_by = sqlc.arg(user_id), updated_at = now()
WHERE id = sqlc.arg(project_id) AND tenant_id = sqlc.arg(tenant_id) AND status <> 'deleted'
RETURNING *;

-- name: SoftDeleteProject :execrows
UPDATE projects SET status = 'deleted', updated_by = sqlc.arg(user_id), updated_at = now()
WHERE id = sqlc.arg(project_id) AND tenant_id = sqlc.arg(tenant_id) AND status <> 'deleted';

-- name: ListProjectTags :many
SELECT t.*, (SELECT count(*) FROM project_tag_bindings b WHERE b.tag_id = t.id) AS project_count
FROM project_tags t
WHERE t.tenant_id = sqlc.arg(tenant_id)
  AND (sqlc.arg(keyword)::text = '' OR t.name ILIKE '%' || sqlc.arg(keyword) || '%')
ORDER BY t.sort_order, t.created_at;

-- name: GetProjectTag :one
SELECT * FROM project_tags WHERE id = sqlc.arg(tag_id) AND tenant_id = sqlc.arg(tenant_id) LIMIT 1;

-- name: CreateProjectTag :one
INSERT INTO project_tags (tenant_id, name, color, description, sort_order, created_by)
VALUES (sqlc.arg(tenant_id), sqlc.arg(name), sqlc.narg(color), sqlc.narg(description), sqlc.arg(sort_order), sqlc.arg(user_id))
RETURNING *;

-- name: UpdateProjectTag :one
UPDATE project_tags
SET name = sqlc.arg(name), color = sqlc.narg(color), description = sqlc.narg(description),
    sort_order = sqlc.arg(sort_order), updated_at = now()
WHERE id = sqlc.arg(tag_id) AND tenant_id = sqlc.arg(tenant_id)
RETURNING *;

-- name: DeleteProjectTag :execrows
DELETE FROM project_tags WHERE id = sqlc.arg(tag_id) AND tenant_id = sqlc.arg(tenant_id);

-- name: ListProjectGroups :many
SELECT g.*, (SELECT count(*) FROM project_group_members m WHERE m.group_id = g.id) AS project_count
FROM project_groups g
WHERE g.tenant_id = sqlc.arg(tenant_id)
  AND (sqlc.arg(keyword)::text = '' OR g.name ILIKE '%' || sqlc.arg(keyword) || '%')
ORDER BY g.sort_order, g.created_at;

-- name: GetProjectGroup :one
SELECT * FROM project_groups WHERE id = sqlc.arg(group_id) AND tenant_id = sqlc.arg(tenant_id) LIMIT 1;

-- name: CreateProjectGroup :one
INSERT INTO project_groups (tenant_id, name, description, sort_order, created_by)
VALUES (sqlc.arg(tenant_id), sqlc.arg(name), sqlc.narg(description), sqlc.arg(sort_order), sqlc.arg(user_id))
RETURNING *;

-- name: UpdateProjectGroup :one
UPDATE project_groups
SET name = sqlc.arg(name), description = sqlc.narg(description), sort_order = sqlc.arg(sort_order), updated_at = now()
WHERE id = sqlc.arg(group_id) AND tenant_id = sqlc.arg(tenant_id)
RETURNING *;

-- name: DeleteProjectGroup :execrows
DELETE FROM project_groups WHERE id = sqlc.arg(group_id) AND tenant_id = sqlc.arg(tenant_id);

-- name: DeleteProjectTagBindings :exec
DELETE FROM project_tag_bindings WHERE project_id = sqlc.arg(project_id);

-- name: CreateProjectTagBinding :exec
INSERT INTO project_tag_bindings (project_id, tag_id) VALUES (sqlc.arg(project_id), sqlc.arg(tag_id));

-- name: DeleteProjectGroupBinding :exec
DELETE FROM project_group_members WHERE project_id = sqlc.arg(project_id);

-- name: CreateProjectGroupBinding :exec
INSERT INTO project_group_members (project_id, group_id) VALUES (sqlc.arg(project_id), sqlc.arg(group_id));

-- name: GetProjectDeleteImpact :one
WITH input AS (SELECT sqlc.arg(project_id)::uuid AS id)
SELECT
  (SELECT count(*) FROM project_runtime_users, input WHERE project_runtime_users.project_id = input.id) AS runtime_user_count,
  (SELECT count(*) FROM project_roles, input WHERE project_roles.project_id = input.id) AS runtime_role_count,
  (SELECT count(*) FROM node_deployments, input WHERE node_deployments.project_id = input.id AND deleted_at IS NULL) AS deployment_count,
  (SELECT count(*) FROM application_versions, input WHERE application_versions.project_id = input.id AND deleted_at IS NULL) AS version_count;
