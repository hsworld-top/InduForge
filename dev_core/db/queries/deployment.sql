-- name: ListApplicationVersions :many
SELECT v.* FROM application_versions v
JOIN projects p ON p.id = v.project_id
WHERE v.project_id = sqlc.arg(project_id) AND v.tenant_id = sqlc.arg(tenant_id)
  AND p.status <> 'deleted' AND v.status <> 'deleted'
ORDER BY v.created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountApplicationVersions :one
SELECT count(*) FROM application_versions v
JOIN projects p ON p.id = v.project_id
WHERE v.project_id = sqlc.arg(project_id) AND v.tenant_id = sqlc.arg(tenant_id)
  AND p.status <> 'deleted' AND v.status <> 'deleted';

-- name: GetApplicationVersion :one
SELECT v.* FROM application_versions v
JOIN projects p ON p.id = v.project_id
WHERE v.id = sqlc.arg(version_id) AND v.tenant_id = sqlc.arg(tenant_id)
  AND p.status <> 'deleted' AND v.status <> 'deleted'
LIMIT 1;

-- name: CreateApplicationVersion :one
INSERT INTO application_versions (
  tenant_id, project_id, version, name, description, status, source_hash,
  artifact_bucket, artifact_key, artifact_hash, artifact_size, manifest,
  created_by, completed_at
)
VALUES (
  sqlc.arg(tenant_id), sqlc.arg(project_id), sqlc.arg(version), sqlc.narg(name), sqlc.narg(description),
  'ready', sqlc.arg(source_hash), sqlc.arg(artifact_bucket), sqlc.arg(artifact_key),
  sqlc.arg(artifact_hash), sqlc.arg(artifact_size), sqlc.arg(manifest), sqlc.arg(user_id), now()
)
RETURNING *;

-- name: DeleteApplicationVersion :execrows
UPDATE application_versions
SET status = 'deleted', deleted_at = now(), updated_at = now()
WHERE application_versions.id = sqlc.arg(version_id) AND application_versions.tenant_id = sqlc.arg(tenant_id)
  AND NOT EXISTS (
    SELECT 1 FROM node_deployments d
    WHERE d.application_version_id = application_versions.id
      AND d.deleted_at IS NULL AND d.status IN ('pending', 'deploying', 'running')
  )
  AND NOT EXISTS (SELECT 1 FROM authoring_restore_tasks t WHERE t.application_version_id = application_versions.id
    AND t.state IN ('queued','staging','restoring_workspace','restoring_scenes','restoring_data','finalizing','compensating'));

-- name: ListProjectNodeDeployments :many
SELECT d.*, n.name AS node_name, n.status AS node_status, n.ip_address,
       p.name AS project_name, p.code AS project_code,
       v.artifact_hash, v.manifest
FROM node_deployments d
JOIN nodes n ON n.id = d.node_id
JOIN projects p ON p.id = d.project_id
LEFT JOIN application_versions v ON v.id = d.application_version_id
WHERE d.project_id = sqlc.arg(project_id) AND d.tenant_id = sqlc.arg(tenant_id)
  AND d.deleted_at IS NULL AND d.status <> 'removed'
ORDER BY d.created_at DESC;

-- name: GetNodeDeployment :one
SELECT d.* FROM node_deployments d
WHERE d.id = sqlc.arg(node_deployment_id) AND d.tenant_id = sqlc.arg(tenant_id)
  AND d.deleted_at IS NULL
LIMIT 1;

-- name: GetApprovedDeploymentNode :one
SELECT * FROM nodes
WHERE id = sqlc.arg(node_id) AND tenant_id = sqlc.arg(tenant_id)
  AND approval_status = 'approved' AND status <> 'disabled'
LIMIT 1;

-- name: RemoveActiveNodeDeployment :exec
UPDATE node_deployments
SET status = 'removed', deleted_at = now(), updated_at = now()
WHERE node_id = sqlc.arg(node_id) AND project_id = sqlc.arg(project_id)
  AND deleted_at IS NULL AND status IN ('pending', 'deploying', 'running', 'stopped', 'failed');

-- name: CreateNodeDeployment :one
INSERT INTO node_deployments (
  tenant_id, node_id, project_id, application_version_id, version, mode,
  status, runtime_config, deployed_by
)
VALUES (
  sqlc.arg(tenant_id), sqlc.arg(node_id), sqlc.arg(project_id), sqlc.narg(application_version_id),
  sqlc.narg(version), sqlc.arg(mode), 'pending', sqlc.arg(runtime_config), sqlc.arg(user_id)
)
RETURNING *;

-- name: UpdateNodeDeploymentState :one
UPDATE node_deployments
SET status = sqlc.arg(status), updated_at = now(),
    started_at = CASE WHEN sqlc.arg(status)::text = 'running' THEN now() ELSE started_at END,
    stopped_at = CASE WHEN sqlc.arg(status)::text = 'stopped' THEN now() ELSE stopped_at END,
    deleted_at = CASE WHEN sqlc.arg(status)::text = 'removed' THEN now() ELSE deleted_at END
WHERE id = sqlc.arg(node_deployment_id) AND tenant_id = sqlc.arg(tenant_id)
RETURNING *;

-- name: CreateNodeCommand :one
INSERT INTO node_commands (
  tenant_id, node_id, node_deployment_id, project_id, command_type, status, payload
)
VALUES (
  sqlc.arg(tenant_id), sqlc.arg(node_id), sqlc.arg(node_deployment_id), sqlc.arg(project_id),
  sqlc.arg(command_type), 'pending', sqlc.arg(payload)
)
RETURNING *;

-- name: RecoverStaleNodeCommands :execrows
UPDATE node_commands
SET status = CASE WHEN attempts >= max_attempts THEN 'dead_letter' ELSE 'pending' END,
    issued_at = CASE WHEN attempts >= max_attempts THEN issued_at ELSE NULL END,
    completed_at = CASE WHEN attempts >= max_attempts THEN now() ELSE completed_at END,
    last_error = '命令执行超时，已由控制面恢复',
    updated_at = now()
WHERE status IN ('issued', 'acknowledged')
  AND COALESCE(issued_at, requested_at) < now() - timeout_seconds * interval '1 second';

-- name: ListStaleNodeCommands :many
SELECT id, payload, attempts, max_attempts
FROM node_commands
WHERE status IN ('issued', 'acknowledged')
  AND COALESCE(issued_at, requested_at) < now() - timeout_seconds * interval '1 second'
ORDER BY requested_at
LIMIT 100;

-- name: RecoverStaleNodeCommand :execrows
UPDATE node_commands
SET status = CASE WHEN attempts >= max_attempts THEN 'dead_letter' ELSE 'pending' END,
    payload = sqlc.arg(payload),
    issued_at = CASE WHEN attempts >= max_attempts THEN issued_at ELSE NULL END,
    completed_at = CASE WHEN attempts >= max_attempts THEN now() ELSE completed_at END,
    last_error = '命令执行超时，已由控制面恢复',
    updated_at = now()
WHERE id = sqlc.arg(command_id)
  AND status IN ('issued', 'acknowledged')
  AND COALESCE(issued_at, requested_at) < now() - timeout_seconds * interval '1 second';

-- name: MarkApplicationVersionReady :one
UPDATE application_versions
SET status='ready', artifact_bucket=sqlc.arg(artifact_bucket), artifact_key=sqlc.arg(artifact_key),
    artifact_hash=sqlc.arg(artifact_hash), artifact_size=sqlc.arg(artifact_size), manifest=sqlc.arg(manifest),
    manifest_hash=sqlc.arg(manifest_hash), checksums_hash=sqlc.arg(checksums_hash), signing_key_id=sqlc.arg(signing_key_id),
    authoring_snapshot_schema=sqlc.arg(authoring_snapshot_schema), authoring_snapshot_bucket=sqlc.arg(authoring_snapshot_bucket),
    authoring_snapshot_key=sqlc.arg(authoring_snapshot_key), authoring_snapshot_hash=sqlc.arg(authoring_snapshot_hash),
    authoring_snapshot_cipher_hash=sqlc.arg(authoring_snapshot_cipher_hash), authoring_snapshot_size=sqlc.arg(authoring_snapshot_size),
    authoring_snapshot_key_id=sqlc.arg(authoring_snapshot_key_id), authoring_project_revision=sqlc.arg(authoring_project_revision),
    restorable=sqlc.arg(restorable),
    completed_at=now(), error_message=NULL, updated_at=now()
WHERE id=sqlc.arg(version_id) AND tenant_id=sqlc.arg(tenant_id) AND status='building'
RETURNING *;

-- name: MarkApplicationVersionFailed :one
UPDATE application_versions SET status='failed', error_message=sqlc.arg(error_message), completed_at=now(), updated_at=now()
WHERE id=sqlc.arg(version_id) AND tenant_id=sqlc.arg(tenant_id) AND status='building'
RETURNING *;
