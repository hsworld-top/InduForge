package project

import (
	"context"
	"github.com/jackc/pgx/v5"
)

type deploymentSummaryReader interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// 工程可见性先由原分页SQL裁剪，这里仅批量读取该页ID；默认环境也不能代表多部署工程整体。
const deploymentSummarySQL = `SELECT d.project_id::text,d.id::text,d.environment_id::text,e.name,e.is_default,
 CASE WHEN d.mode='release' THEN 'production' ELSE d.mode END,COALESCE(d.application_version_id::text,''),COALESCE(v.version,CASE WHEN d.mode='development' THEN '__DEV__' ELSE '' END),
 d.access_port,d.desired_status,d.observed_status,d.deletion_requested_at IS NOT NULL,d.updated_at,
 COALESCE(run.operation,''),COALESCE(run.observed_status='pending',false),COALESCE(s.service_type,''),COALESCE(s.node_id::text,''),
 COALESCE(n.display_name,n.hostname,''),COALESCE(s.desired_status,''),COALESCE(s.observed_status,''),COALESCE(s.desired_generation,0),COALESCE(s.observed_generation,0)
 FROM project_deployments d JOIN runtime_environments e ON e.id=d.environment_id AND e.tenant_id=d.tenant_id AND e.deleted_at IS NULL
 LEFT JOIN application_versions v ON v.id=d.application_version_id AND v.tenant_id=d.tenant_id
 LEFT JOIN LATERAL (SELECT operation,observed_status FROM deployment_runs WHERE project_deployment_id=d.id ORDER BY started_at DESC,id DESC LIMIT 1) run ON true
 LEFT JOIN deployment_services s ON s.project_deployment_id=d.id AND s.tenant_id=d.tenant_id LEFT JOIN host_nodes n ON n.id=s.node_id AND n.tenant_id=d.tenant_id
 WHERE d.tenant_id=$1 AND d.project_id=ANY($2::uuid[]) AND d.deleted_at IS NULL ORDER BY d.project_id,d.environment_id,d.id,s.service_type`

func (r *PostgreSQLRepository) attachDeploymentSummaries(ctx context.Context, tenant string, items []Project) error {
	return attachDeploymentSummaries(ctx, r.pool, tenant, items)
}
func attachDeploymentSummaries(ctx context.Context, reader deploymentSummaryReader, tenant string, items []Project) error {
	byProject := make(map[string]*Project, len(items))
	ids := make([]string, 0, len(items))
	for i := range items {
		items[i].DeploymentSummary = DeploymentSummary{PrimarySelection: "none"}
		byProject[items[i].ID] = &items[i]
		ids = append(ids, items[i].ID)
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := reader.Query(ctx, deploymentSummarySQL, tenant, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	type candidate struct {
		deployment PrimaryDeployment
		isDefault  bool
	}
	candidates := map[string][]*candidate{}
	current := map[string]*candidate{}
	environments := map[string]map[string]struct{}{}
	for rows.Next() {
		var projectID string
		var c candidate
		var service DeploymentPlacement
		var deleting, pending bool
		var desiredGeneration, observedGeneration int64
		if err = rows.Scan(&projectID, &c.deployment.ID, &c.deployment.EnvironmentID, &c.deployment.EnvironmentName, &c.isDefault, &c.deployment.Mode, &c.deployment.ApplicationVersionID, &c.deployment.Version, &c.deployment.AccessPort, &c.deployment.DesiredStatus, &c.deployment.ObservedStatus, &deleting, &c.deployment.UpdatedAt, &c.deployment.CurrentOperation, &pending, &service.ServiceType, &service.NodeID, &service.NodeName, &service.DesiredStatus, &service.ObservedStatus, &desiredGeneration, &observedGeneration); err != nil {
			return err
		}
		if byProject[projectID] == nil {
			continue
		}
		key := projectID + "/" + c.deployment.ID
		existing := current[key]
		if existing == nil {
			existing = &c
			current[key] = existing
			candidates[projectID] = append(candidates[projectID], existing)
			if environments[projectID] == nil {
				environments[projectID] = map[string]struct{}{}
			}
			environments[projectID][c.deployment.EnvironmentID] = struct{}{}
		}
		if service.ServiceType != "" {
			existing.deployment.Services = append(existing.deployment.Services, service)
			if existing.deployment.Placements == nil {
				existing.deployment.Placements = map[string]string{}
			}
			existing.deployment.Placements[service.ServiceType] = service.NodeID
			// 代次只在尚未失败的正向推进中代表“更新中”；失败/异常态必须允许用户重新处理。
			if existing.deployment.ObservedStatus != "failed" && existing.deployment.ObservedStatus != "degraded" && service.ObservedStatus != "failed" && service.DesiredStatus == "running" && desiredGeneration > observedGeneration {
				existing.deployment.Updating = true
				existing.deployment.OperationInProgress = true
			}
		}
		existing.deployment.OperationInProgress = existing.deployment.OperationInProgress || pending || deleting
		if deleting {
			existing.deployment.CurrentOperation = "delete"
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	for projectID, list := range candidates {
		p := byProject[projectID]
		p.DeploymentSummary.DeploymentCount = len(list)
		p.DeploymentSummary.EnvironmentCount = len(environments[projectID])
		for _, c := range list {
			p.DeploymentSummary.OperationInProgress = p.DeploymentSummary.OperationInProgress || c.deployment.OperationInProgress
			if p.DeploymentSummary.UpdatedAt == nil || c.deployment.UpdatedAt.After(*p.DeploymentSummary.UpdatedAt) {
				updated := c.deployment.UpdatedAt
				p.DeploymentSummary.UpdatedAt = &updated
			}
		}
		if len(list) == 1 {
			p.DeploymentSummary.PrimarySelection = "unique"
			p.DeploymentSummary.PrimaryDeployment = &list[0].deployment
		} else {
			p.DeploymentSummary.PrimarySelection = "multiple"
		}
	}
	return nil
}

func deploymentSummaryResponse(summary DeploymentSummary) map[string]any {
	var primary any
	if d := summary.PrimaryDeployment; d != nil {
		services := make([]map[string]any, 0, len(d.Services))
		for _, s := range d.Services {
			services = append(services, map[string]any{"serviceType": s.ServiceType, "nodeId": s.NodeID, "nodeName": s.NodeName, "desiredStatus": s.DesiredStatus, "observedStatus": s.ObservedStatus})
		}
		placements := d.Placements
		if placements == nil {
			placements = map[string]string{}
		}
		primary = map[string]any{"id": d.ID, "environmentId": d.EnvironmentID, "environmentName": d.EnvironmentName, "mode": d.Mode, "desiredStatus": d.DesiredStatus, "observedStatus": d.ObservedStatus, "version": d.Version, "applicationVersionId": d.ApplicationVersionID, "accessPort": d.AccessPort, "updatedAt": d.UpdatedAt, "operationInProgress": d.OperationInProgress, "currentOperation": d.CurrentOperation, "updating": d.Updating, "placements": placements, "services": services}
	}
	return map[string]any{"deploymentCount": summary.DeploymentCount, "environmentCount": summary.EnvironmentCount, "operationInProgress": summary.OperationInProgress, "updatedAt": summary.UpdatedAt, "primarySelection": summary.PrimarySelection, "primaryDeployment": primary}
}
