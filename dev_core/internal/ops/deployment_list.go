package ops

import (
	"context"
	"github.com/jackc/pgx/v5"
)

type deploymentPageReader interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

const deploymentPageServicesSQL = `SELECT s.id,s.tenant_id,s.project_deployment_id,s.node_id,COALESCE(n.display_name,n.hostname,''),s.service_type,s.public_port,s.desired_status,s.observed_status,COALESCE(s.last_message,''),COALESCE(s.endpoint,''),s.replicas_desired,s.replicas_observed,s.desired_generation,s.observed_generation,s.last_operation,s.observed_at,s.created_at,s.updated_at FROM deployment_services s JOIN host_nodes n ON n.id=s.node_id AND n.tenant_id=s.tenant_id WHERE s.tenant_id=$1 AND s.project_deployment_id=ANY($2::uuid[]) ORDER BY s.project_deployment_id,s.service_type`

// loadDeploymentPage 恒定三次查询，任何下一次查询前均释放rows，避免N+1及连接池嵌套等待。
func loadDeploymentPage(ctx context.Context, reader deploymentPageReader, tenant string, f PageFilter) ([]ProjectDeployment, int64, error) {
	rows, err := reader.Query(ctx, deploymentSelect+` WHERE `+deploymentListFilter+` ORDER BY d.updated_at DESC,d.id DESC LIMIT $3 OFFSET $4`, tenant, f.Search, f.PageSize, (f.Page-1)*f.PageSize, f.ProjectID, f.EnvironmentID)
	if err != nil {
		return nil, 0, err
	}
	items := []ProjectDeployment{}
	ids := []string{}
	for rows.Next() {
		item, err := scanDeployment(rows)
		if err != nil {
			rows.Close()
			return nil, 0, err
		}
		item.Services = []DeploymentService{}
		items = append(items, item)
		ids = append(ids, item.ID)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, 0, err
	}
	services, err := reader.Query(ctx, deploymentPageServicesSQL, tenant, ids)
	if err != nil {
		return nil, 0, err
	}
	byDeployment := map[string][]DeploymentService{}
	for services.Next() {
		var service DeploymentService
		if err := services.Scan(serviceScanArgs(&service)...); err != nil {
			services.Close()
			return nil, 0, err
		}
		byDeployment[service.ProjectDeploymentID] = append(byDeployment[service.ProjectDeploymentID], service)
	}
	err = services.Err()
	services.Close()
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		if value := byDeployment[items[i].ID]; value != nil {
			items[i].Services = value
		}
	}
	var total int64
	err = reader.QueryRow(ctx, `SELECT count(*) FROM project_deployments d JOIN projects p ON p.id=d.project_id AND p.tenant_id=d.tenant_id JOIN runtime_environments e ON e.id=d.environment_id AND e.tenant_id=d.tenant_id WHERE `+deploymentCountFilter, tenant, f.Search, f.ProjectID, f.EnvironmentID).Scan(&total)
	return items, total, err
}
