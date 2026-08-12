-- name: ListNodes :many
SELECT n.id, n.tenant_id, n.name, n.code, n.node_type, n.status, n.approval_status,
       n.agent_version, n.os, n.architecture, n.hostname, n.ip_address,
       n.capabilities, n.metrics, n.metadata, n.last_heartbeat_at,
       n.approved_at, n.rejected_at, n.created_at, n.updated_at,
       COALESCE(
         jsonb_agg(
           DISTINCT jsonb_build_object(
             'id', d.id,
             'deploymentId', d.application_version_id,
             'nodeId', d.node_id,
             'projectId', d.project_id,
             'version', d.version,
             'mode', upper(CASE WHEN d.mode = 'development' THEN 'DEV' ELSE 'RELEASE' END),
             'status', d.status,
             'runtimeConfig', d.runtime_config,
             'runtimeMetrics', d.runtime_metrics,
             'errorMessage', d.error_message,
             'createdAt', d.created_at,
             'updatedAt', d.updated_at,
             'project', jsonb_build_object('id', p.id, 'name', p.name, 'code', p.code)
           )
         ) FILTER (WHERE d.id IS NOT NULL),
         '[]'::jsonb
       ) AS deployments
FROM nodes n
LEFT JOIN node_deployments d ON d.node_id = n.id AND d.deleted_at IS NULL AND d.status <> 'removed'
LEFT JOIN projects p ON p.id = d.project_id
WHERE n.tenant_id = sqlc.arg(tenant_id)
  AND (sqlc.arg(status)::text = '' OR n.status = sqlc.arg(status))
  AND (sqlc.arg(approval_status)::text = '' OR n.approval_status = sqlc.arg(approval_status))
  AND (sqlc.arg(keyword)::text = '' OR n.name ILIKE '%' || sqlc.arg(keyword) || '%' OR COALESCE(n.hostname, '') ILIKE '%' || sqlc.arg(keyword) || '%' OR COALESCE(n.ip_address, '') ILIKE '%' || sqlc.arg(keyword) || '%')
GROUP BY n.id
ORDER BY n.created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountNodes :one
SELECT count(*) FROM nodes n
WHERE n.tenant_id = sqlc.arg(tenant_id)
  AND (sqlc.arg(status)::text = '' OR n.status = sqlc.arg(status))
  AND (sqlc.arg(approval_status)::text = '' OR n.approval_status = sqlc.arg(approval_status))
  AND (sqlc.arg(keyword)::text = '' OR n.name ILIKE '%' || sqlc.arg(keyword) || '%' OR COALESCE(n.hostname, '') ILIKE '%' || sqlc.arg(keyword) || '%' OR COALESCE(n.ip_address, '') ILIKE '%' || sqlc.arg(keyword) || '%');

-- name: GetNode :one
SELECT * FROM nodes WHERE id = sqlc.arg(node_id) AND tenant_id = sqlc.arg(tenant_id) LIMIT 1;

-- name: ApproveNode :one
UPDATE nodes
SET approval_status = 'approved', status = 'offline', approved_at = now(), approved_by = sqlc.arg(user_id),
    rejected_at = NULL, rejected_by = NULL, updated_at = now()
WHERE id = sqlc.arg(node_id) AND tenant_id = sqlc.arg(tenant_id)
RETURNING *;

-- name: RejectNode :one
UPDATE nodes
SET approval_status = 'rejected', status = 'rejected', rejected_at = now(), rejected_by = sqlc.arg(user_id),
    approved_at = NULL, approved_by = NULL, updated_at = now()
WHERE id = sqlc.arg(node_id) AND tenant_id = sqlc.arg(tenant_id)
RETURNING *;

-- name: DeleteNode :execrows
DELETE FROM nodes WHERE id = sqlc.arg(node_id) AND tenant_id = sqlc.arg(tenant_id);

-- name: RegisterNode :one
INSERT INTO nodes (
  id, tenant_id, name, node_type, status, approval_status, registration_token_hash,
  agent_version, ip_address, metadata, approved_at, approved_by
)
VALUES (
  sqlc.arg(node_id), sqlc.arg(tenant_id), sqlc.arg(name), 'edge', sqlc.arg(status),
  sqlc.arg(approval_status), sqlc.arg(registration_token_hash), sqlc.narg(agent_version),
  sqlc.narg(ip_address), sqlc.arg(metadata), sqlc.narg(approved_at), sqlc.narg(approved_by)
)
ON CONFLICT (id) DO UPDATE SET
  tenant_id = EXCLUDED.tenant_id,
  name = EXCLUDED.name,
  status = EXCLUDED.status,
  approval_status = EXCLUDED.approval_status,
  registration_token_hash = EXCLUDED.registration_token_hash,
  agent_version = EXCLUDED.agent_version,
  ip_address = EXCLUDED.ip_address,
  metadata = EXCLUDED.metadata,
  approved_at = EXCLUDED.approved_at,
  approved_by = EXCLUDED.approved_by,
  rejected_at = NULL,
  rejected_by = NULL,
  updated_at = now()
RETURNING *;

-- name: GetNodeApprovalStatus :one
SELECT id, name, status, approval_status, approved_at, rejected_at, updated_at
FROM nodes WHERE id = sqlc.arg(node_id) LIMIT 1;

-- name: GetNodeByToken :one
SELECT * FROM nodes
WHERE id = sqlc.arg(node_id) AND registration_token_hash = sqlc.arg(registration_token_hash)
LIMIT 1;

-- name: UpdateNodeHeartbeat :one
UPDATE nodes
SET status = 'online', metrics = sqlc.arg(metrics), agent_version = sqlc.narg(agent_version),
    last_heartbeat_at = now(), updated_at = now()
WHERE id = sqlc.arg(node_id) AND registration_token_hash = sqlc.arg(registration_token_hash)
  AND approval_status = 'approved'
RETURNING *;

-- name: UpdateNodeOffline :one
UPDATE nodes
SET status = 'offline', metadata = metadata || sqlc.arg(metadata)::jsonb, updated_at = now()
WHERE id = sqlc.arg(node_id) AND registration_token_hash = sqlc.arg(registration_token_hash)
RETURNING *;

-- name: ListPendingNodeCommands :many
SELECT * FROM node_commands
WHERE node_id = sqlc.arg(node_id) AND status = 'pending'
ORDER BY requested_at
LIMIT 20;

-- name: MarkNodeCommandIssued :exec
UPDATE node_commands
SET status = 'issued', issued_at = now(), attempts = attempts + 1, updated_at = now()
WHERE id = sqlc.arg(command_id) AND status = 'pending';

-- name: UpdateNodeReportedDeployment :one
UPDATE node_deployments
SET status = sqlc.arg(status), error_message = sqlc.narg(error_message), deploy_log = sqlc.narg(message),
    deployed_at = CASE WHEN sqlc.arg(status)::text = 'running' THEN COALESCE(deployed_at, now()) ELSE deployed_at END,
    started_at = CASE WHEN sqlc.arg(status)::text = 'running' THEN COALESCE(sqlc.narg(started_at), now()) ELSE started_at END,
    stopped_at = CASE WHEN sqlc.arg(status)::text = 'stopped' THEN COALESCE(sqlc.narg(stopped_at), now()) ELSE stopped_at END,
    updated_at = now()
WHERE id = sqlc.arg(node_deployment_id) AND node_id = sqlc.arg(node_id)
RETURNING *;

-- name: CompleteNodeDeploymentCommands :exec
UPDATE node_commands
SET status = sqlc.arg(status), completed_at = now(), last_error = sqlc.narg(last_error), updated_at = now()
WHERE node_deployment_id = sqlc.arg(node_deployment_id)
  AND status IN ('pending', 'issued', 'acknowledged');
