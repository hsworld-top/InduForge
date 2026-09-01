package ops

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const runtimeEnvironmentProjectionSQL = `
	SELECT e.id,e.tenant_id,e.name,e.code,e.desired_status,e.is_default,
		CASE
			WHEN e.desired_status='deleting' THEN 'deleting'
			WHEN COALESCE(ns.node_count,0)=0 OR COALESCE(fs.service_count,0)=0 THEN 'uninitialized'
			WHEN COALESCE(ns.online_count,0)<COALESCE(ns.node_count,0) OR COALESCE(fs.healthy_count,0)<COALESCE(fs.service_count,0) THEN 'attention'
			ELSE 'available'
		END AS status,
		COALESCE(ns.node_count,0),COALESCE(ns.online_count,0),
		COALESCE(fs.service_count,0),COALESCE(fs.healthy_count,0),0,
		COALESCE(ev.name,'运行环境已创建'),COALESCE(ev.operator_name,'系统'),ev.created_at,
		e.created_at,e.updated_at
	FROM runtime_environments e
	LEFT JOIN LATERAL (
		SELECT count(*)::int AS node_count,
			count(*) FILTER (WHERE n.desired_status='active' AND n.observed_status='online' AND n.last_heartbeat_at>now()-interval '45 seconds')::int AS online_count
		FROM runtime_environment_nodes en
		JOIN host_nodes n ON n.id=en.node_id AND n.tenant_id=e.tenant_id
		WHERE en.environment_id=e.id
	) ns ON true
	LEFT JOIN LATERAL (
		SELECT count(*)::int AS service_count,
			count(*) FILTER (WHERE s.observed_status='running' AND s.observed_at>now()-interval '45 seconds')::int AS healthy_count
		FROM runtime_environment_services s
		WHERE s.environment_id=e.id
	) fs ON true
	LEFT JOIN LATERAL (
		SELECT event.name,COALESCE(u.username,'系统') AS operator_name,event.created_at
		FROM runtime_environment_events event
		LEFT JOIN users u ON u.id=event.created_by
		WHERE event.environment_id=e.id
		ORDER BY event.created_at DESC,event.id DESC LIMIT 1
	) ev ON true
	WHERE e.deleted_at IS NULL`

// 基础服务计划以运行环境为代次边界。调整任意服务分布时，八项服务必须共同进入
// 新代次，Hostd 才能用一次完整状态上报刷新全部服务的观测时间。
const advanceFoundationGenerationSQL = `UPDATE runtime_environment_services SET desired_generation=$1,updated_at=now() WHERE environment_id=$2`

func (r *PostgreSQLRepository) ListRuntimeEnvironments(ctx context.Context, tenant string, f PageFilter) ([]RuntimeEnvironment, int64, error) {
	query := `SELECT * FROM (` + runtimeEnvironmentProjectionSQL + `) environment WHERE environment.tenant_id=$1 AND ($2='' OR environment.name ILIKE '%'||$2||'%' OR environment.code ILIKE '%'||$2||'%') AND ($3='' OR environment.status=$3) ORDER BY environment.is_default DESC,environment.updated_at DESC,environment.id DESC LIMIT $4 OFFSET $5`
	rows, err := r.pool.Query(ctx, query, tenant, f.Search, f.Status, f.PageSize, (f.Page-1)*f.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]RuntimeEnvironment, 0)
	for rows.Next() {
		item, scanErr := scanRuntimeEnvironment(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int64
	countQuery := `SELECT count(*) FROM (` + runtimeEnvironmentProjectionSQL + `) environment WHERE environment.tenant_id=$1 AND ($2='' OR environment.name ILIKE '%'||$2||'%' OR environment.code ILIKE '%'||$2||'%') AND ($3='' OR environment.status=$3)`
	err = r.pool.QueryRow(ctx, countQuery, tenant, f.Search, f.Status).Scan(&total)
	return items, total, err
}

func (r *PostgreSQLRepository) CreateRuntimeEnvironment(ctx context.Context, tenant, user string, input CreateRuntimeEnvironmentInput, code string) (RuntimeEnvironment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return RuntimeEnvironment{}, err
	}
	defer tx.Rollback(ctx)
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO runtime_environments(tenant_id,name,code,created_by) VALUES($1,$2,$3,$4) RETURNING id`, tenant, input.Name, code, user).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return RuntimeEnvironment{}, ErrEnvironmentExists
		}
		return RuntimeEnvironment{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,created_by) VALUES($1,$2,'environment_created','运行环境已创建',$3,'success',$4)`, tenant, id, input.Name, user)
	if err != nil {
		return RuntimeEnvironment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RuntimeEnvironment{}, err
	}
	return r.GetRuntimeEnvironment(ctx, tenant, id)
}

func (r *PostgreSQLRepository) UpdateRuntimeEnvironment(ctx context.Context, tenant, id, user string, input UpdateRuntimeEnvironmentInput) (RuntimeEnvironment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return RuntimeEnvironment{}, err
	}
	defer tx.Rollback(ctx)
	var oldName, desiredStatus string
	if err := tx.QueryRow(ctx, `SELECT name,desired_status FROM runtime_environments WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL FOR UPDATE`, tenant, id).Scan(&oldName, &desiredStatus); errors.Is(err, pgx.ErrNoRows) {
		return RuntimeEnvironment{}, ErrNotFound
	} else if err != nil {
		return RuntimeEnvironment{}, err
	}
	if desiredStatus == "deleting" {
		return RuntimeEnvironment{}, ErrEnvironmentDeleting
	}
	if oldName != input.Name {
		if _, err := tx.Exec(ctx, `UPDATE runtime_environments SET name=$1,updated_at=now() WHERE id=$2`, input.Name, id); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return RuntimeEnvironment{}, ErrEnvironmentExists
			}
			return RuntimeEnvironment{}, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message,created_by) VALUES($1,$2,'environment_updated','运行环境信息已更新',$3,'success',$4,$5)`, tenant, id, input.Name, "名称："+oldName+" → "+input.Name, user); err != nil {
			return RuntimeEnvironment{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return RuntimeEnvironment{}, err
	}
	return r.GetRuntimeEnvironment(ctx, tenant, id)
}

func (r *PostgreSQLRepository) DeleteRuntimeEnvironment(ctx context.Context, tenant, id, user string, input DeleteRuntimeEnvironmentInput) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var name, desiredStatus string
	var isDefault bool
	if err := tx.QueryRow(ctx, `SELECT name,desired_status,is_default FROM runtime_environments WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL FOR UPDATE`, tenant, id).Scan(&name, &desiredStatus, &isDefault); errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	} else if err != nil {
		return "", err
	}
	if isDefault {
		return "", ErrDefaultEnvironmentProtected
	}
	if name != input.ConfirmationName {
		return "", fmt.Errorf("输入的运行环境名称不匹配")
	}
	if desiredStatus == "deleting" {
		return "deleting", nil
	}
	var deploymentCount, serviceCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM project_deployments d JOIN runtime_environment_nodes en ON en.node_id=d.node_id WHERE en.environment_id=$1`, id).Scan(&deploymentCount); err != nil {
		return "", err
	}
	if deploymentCount > 0 {
		return "", ErrEnvironmentHasDeployment
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM runtime_environment_services WHERE environment_id=$1`, id).Scan(&serviceCount); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message,created_by) VALUES($1,$2,'environment_delete_requested','运行环境删除任务已创建',$3,'success',$4,$5)`, tenant, id, name, "将删除该环境的工程资源、基础服务和隔离空间；中心 K3s 集群与物理节点保持不变", user); err != nil {
		return "", err
	}
	if serviceCount == 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM runtime_environment_nodes WHERE environment_id=$1`, id); err != nil {
			return "", err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message) VALUES($1,$2,'environment_deleted','运行环境已删除',$3,'success','环境没有已部署基础服务，逻辑资源已直接清理')`, tenant, id, name); err != nil {
			return "", err
		}
		if _, err := tx.Exec(ctx, `UPDATE runtime_environments SET desired_status='deleting',deleted_at=now(),updated_at=now() WHERE id=$1`, id); err != nil {
			return "", err
		}
		if err := tx.Commit(ctx); err != nil {
			return "", err
		}
		return "deleted", nil
	}
	if _, err := tx.Exec(ctx, `UPDATE runtime_environments SET desired_status='deleting',updated_at=now() WHERE id=$1`, id); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return "deleting", nil
}

func (r *PostgreSQLRepository) GetRuntimeEnvironment(ctx context.Context, tenant, id string) (RuntimeEnvironment, error) {
	item, err := scanRuntimeEnvironment(r.pool.QueryRow(ctx, `SELECT * FROM (`+runtimeEnvironmentProjectionSQL+`) environment WHERE environment.tenant_id=$1 AND environment.id=$2`, tenant, id))
	return item, mapNotFound(err)
}

func (r *PostgreSQLRepository) ListRuntimeEnvironmentNodes(ctx context.Context, tenant, environmentID string, f PageFilter) ([]Node, int64, error) {
	if err := r.requireRuntimeEnvironment(ctx, tenant, environmentID); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `SELECT n.id,n.tenant_id,n.enrollment_id,n.display_name,n.hostname,n.platform,n.architecture,COALESCE(n.agent_version,''),COALESCE(n.machine_fingerprint,''),COALESCE(n.ip_address,''),n.desired_status,n.observed_status,n.capabilities,n.resource_summary,n.last_heartbeat_at,n.approved_at,n.created_at,n.updated_at,COALESCE(d.id::text,''),COALESCE(d.project_id::text,''),COALESCE(p.name,''),e.id::text,e.name FROM runtime_environment_nodes en JOIN runtime_environments e ON e.id=en.environment_id AND e.tenant_id=$1 JOIN host_nodes n ON n.id=en.node_id AND n.tenant_id=e.tenant_id LEFT JOIN LATERAL (SELECT d.id,d.project_id FROM project_deployments d WHERE d.tenant_id=n.tenant_id AND d.node_id=n.id ORDER BY d.created_at DESC,d.id DESC LIMIT 1) d ON true LEFT JOIN projects p ON p.id=d.project_id AND p.tenant_id=n.tenant_id WHERE en.environment_id=$2 AND ($3='' OR n.display_name ILIKE '%'||$3||'%' OR n.hostname ILIKE '%'||$3||'%') ORDER BY en.created_at DESC,n.id DESC LIMIT $4 OFFSET $5`, tenant, environmentID, f.Search, f.PageSize, (f.Page-1)*f.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Node, 0)
	for rows.Next() {
		item, scanErr := scanNodeWithAssignments(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		if scanErr = r.attachNodeCluster(ctx, &item); scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, item)
	}
	var total int64
	err = r.pool.QueryRow(ctx, `SELECT count(*) FROM runtime_environment_nodes en JOIN runtime_environments e ON e.id=en.environment_id AND e.tenant_id=$1 JOIN host_nodes n ON n.id=en.node_id WHERE en.environment_id=$2 AND ($3='' OR n.display_name ILIKE '%'||$3||'%' OR n.hostname ILIKE '%'||$3||'%')`, tenant, environmentID, f.Search).Scan(&total)
	return items, total, err
}

func (r *PostgreSQLRepository) attachNodeCluster(ctx context.Context, node *Node) error {
	err := r.pool.QueryRow(ctx, `SELECT cn.node_kind,cn.cluster_id::text,CASE WHEN cn.node_kind='center' THEN 'server' ELSE 'agent' END,cn.cluster_status,COALESCE(cn.cluster_message,''),cn.desired_action,cn.desired_generation,cn.observed_generation,cn.cluster_observed_at,COALESCE((SELECT array_agg(e.name ORDER BY en.created_at DESC,e.id DESC) FROM runtime_environment_nodes en JOIN runtime_environments e ON e.id=en.environment_id AND e.deleted_at IS NULL WHERE en.node_id=cn.node_id),ARRAY[]::text[]) FROM runtime_cluster_nodes cn WHERE cn.node_id=$1`, node.ID).Scan(&node.NodeKind, &node.ClusterID, &node.ClusterRole, &node.ClusterStatus, &node.ClusterMessage, &node.ClusterDesiredAction, &node.ClusterDesiredGeneration, &node.ClusterObservedGeneration, &node.ClusterObservedAt, &node.EnvironmentNames)
	if errors.Is(err, pgx.ErrNoRows) {
		node.EnvironmentNames = []string{}
		node.EnvironmentCount = 0
		return nil
	}
	node.EnvironmentCount = len(node.EnvironmentNames)
	return err
}

func (r *PostgreSQLRepository) AddRuntimeEnvironmentNodes(ctx context.Context, tenant, environmentID, user string, nodeIDs []string) ([]Node, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var environmentName, desiredStatus string
	if err := tx.QueryRow(ctx, `SELECT name,desired_status FROM runtime_environments WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL FOR UPDATE`, tenant, environmentID).Scan(&environmentName, &desiredStatus); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	if desiredStatus == "deleting" {
		return nil, ErrEnvironmentDeleting
	}
	rows, err := tx.Query(ctx, `SELECT n.id::text,n.display_name FROM host_nodes n JOIN runtime_cluster_nodes cn ON cn.node_id=n.id WHERE n.tenant_id=$1 AND n.id=ANY($2::uuid[]) AND n.platform='linux' AND n.capabilities @> '["project_entry","data_runtime"]'::jsonb AND n.approved_at IS NOT NULL AND n.desired_status='active' AND cn.desired_action='active' AND cn.cluster_status='ready' FOR UPDATE OF n,cn`, tenant, nodeIDs)
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(nodeIDs))
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			rows.Close()
			return nil, err
		}
		names[id] = name
	}
	rows.Close()
	if len(names) != len(nodeIDs) {
		return nil, ErrNodeNotEligible
	}
	for _, nodeID := range nodeIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO runtime_environment_nodes(tenant_id,environment_id,node_id,created_by) VALUES($1,$2,$3,$4)`, tenant, environmentID, nodeID, user); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return nil, ErrNodeEnvironmentConflict
			}
			return nil, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,created_by) VALUES($1,$2,'node_added','物理节点已关联',$3,'success',$4)`, tenant, environmentID, names[nodeID], user)
		if err != nil {
			return nil, err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE runtime_environments SET updated_at=now() WHERE id=$1`, environmentID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	items, _, err := r.ListRuntimeEnvironmentNodes(ctx, tenant, environmentID, PageFilter{Page: 1, PageSize: 200})
	return items, err
}

func (r *PostgreSQLRepository) RemoveRuntimeEnvironmentNode(ctx context.Context, tenant, environmentID, nodeID, user string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var nodeName, environmentStatus string
	var serviceInUse, deploymentInUse bool
	err = tx.QueryRow(ctx, `SELECT n.display_name,e.desired_status,
		EXISTS(SELECT 1 FROM runtime_environment_services s WHERE s.environment_id=en.environment_id AND s.node_id=en.node_id),
		EXISTS(SELECT 1 FROM project_deployments d WHERE d.node_id=en.node_id)
		FROM runtime_environment_nodes en JOIN runtime_environments e ON e.id=en.environment_id JOIN host_nodes n ON n.id=en.node_id
		WHERE e.tenant_id=$1 AND e.id=$2 AND n.id=$3 AND e.deleted_at IS NULL FOR UPDATE OF en,e,n`, tenant, environmentID, nodeID).Scan(&nodeName, &environmentStatus, &serviceInUse, &deploymentInUse)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if environmentStatus == "deleting" {
		return ErrEnvironmentDeleting
	}
	if serviceInUse {
		return ErrEnvironmentNodeServiceInUse
	}
	if deploymentInUse {
		return ErrEnvironmentNodeDeploymentInUse
	}
	if _, err = tx.Exec(ctx, `DELETE FROM runtime_environment_nodes WHERE environment_id=$1 AND node_id=$2`, environmentID, nodeID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message,created_by) VALUES($1,$2,'node_unassigned','运行节点已取消分配',$3,'success','仅移除环境调度范围，节点仍保留在中心运行集群中',$4)`, tenant, environmentID, nodeName, user)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE runtime_environments SET updated_at=now() WHERE id=$1`, environmentID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgreSQLRepository) finalizeEnvironmentDeletion(ctx context.Context, tenant, environmentID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var name, desiredStatus string
	err = tx.QueryRow(ctx, `SELECT name,desired_status FROM runtime_environments WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL FOR UPDATE`, tenant, environmentID).Scan(&name, &desiredStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if desiredStatus != "deleting" {
		return nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM runtime_environment_services WHERE environment_id=$1`, environmentID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM runtime_environment_nodes WHERE environment_id=$1`, environmentID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message) VALUES($1,$2,'environment_deleted','运行环境已删除',$3,'success','基础服务与环境隔离空间已清理，中心运行集群保持运行')`, tenant, environmentID, name); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE runtime_environments SET deleted_at=now(),updated_at=now() WHERE id=$1`, environmentID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgreSQLRepository) ListRuntimeEnvironmentEvents(ctx context.Context, tenant, environmentID string, f PageFilter) ([]RuntimeEnvironmentEvent, int64, error) {
	if err := r.requireRuntimeEnvironment(ctx, tenant, environmentID); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `SELECT event.id,event.environment_id,event.event_type,event.name,event.target,event.result,COALESCE(event.message,''),COALESCE(u.username,'系统'),event.created_at FROM runtime_environment_events event JOIN runtime_environments e ON e.id=event.environment_id AND e.tenant_id=$1 LEFT JOIN users u ON u.id=event.created_by WHERE event.environment_id=$2 ORDER BY event.created_at DESC,event.id DESC LIMIT $3 OFFSET $4`, tenant, environmentID, f.PageSize, (f.Page-1)*f.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]RuntimeEnvironmentEvent, 0)
	for rows.Next() {
		var item RuntimeEnvironmentEvent
		if err := rows.Scan(&item.ID, &item.EnvironmentID, &item.EventType, &item.Name, &item.Target, &item.Result, &item.Message, &item.OperatorName, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	var total int64
	err = r.pool.QueryRow(ctx, `SELECT count(*) FROM runtime_environment_events event JOIN runtime_environments e ON e.id=event.environment_id AND e.tenant_id=$1 WHERE event.environment_id=$2`, tenant, environmentID).Scan(&total)
	return items, total, err
}

func (r *PostgreSQLRepository) ListRuntimeEnvironmentServices(ctx context.Context, tenant, environmentID string) ([]RuntimeEnvironmentService, error) {
	if err := r.requireRuntimeEnvironment(ctx, tenant, environmentID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT s.id::text,s.environment_id::text,s.node_id::text,n.display_name,s.service_type,s.desired_status,s.observed_status,COALESCE(s.last_message,''),s.desired_generation,s.observed_generation,s.operation,s.observed_at,s.created_at,s.updated_at FROM runtime_environment_services s JOIN host_nodes n ON n.id=s.node_id WHERE s.tenant_id=$1 AND s.environment_id=$2 ORDER BY s.created_at,s.service_type`, tenant, environmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]RuntimeEnvironmentService, 0)
	for rows.Next() {
		var item RuntimeEnvironmentService
		if err := rows.Scan(&item.ID, &item.EnvironmentID, &item.NodeID, &item.NodeName, &item.ServiceType, &item.DesiredStatus, &item.ObservedStatus, &item.LastMessage, &item.DesiredGeneration, &item.ObservedGeneration, &item.Operation, &item.ObservedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgreSQLRepository) MigrateRuntimeEnvironmentFoundation(ctx context.Context, tenant, environmentID, user string, assignments []FoundationAssignment) ([]RuntimeEnvironmentService, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var desiredStatus string
	if err := tx.QueryRow(ctx, `SELECT desired_status FROM runtime_environments WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL FOR UPDATE`, tenant, environmentID).Scan(&desiredStatus); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	if desiredStatus == "deleting" {
		return nil, ErrEnvironmentDeleting
	}
	var activeOperations int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM runtime_environment_services WHERE environment_id=$1 AND (desired_generation<>observed_generation OR observed_status<>'running')`, environmentID).Scan(&activeOperations); err != nil {
		return nil, err
	}
	if activeOperations > 0 {
		return nil, ErrDeploymentBusy
	}
	current := make(map[string]string, len(foundationServiceTypes))
	rows, err := tx.Query(ctx, `SELECT service_type,node_id::text FROM runtime_environment_services WHERE environment_id=$1 FOR UPDATE`, environmentID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var serviceType, nodeID string
		if err := rows.Scan(&serviceType, &nodeID); err != nil {
			rows.Close()
			return nil, err
		}
		current[serviceType] = nodeID
	}
	rows.Close()
	if len(current) != len(foundationServiceTypes) {
		return nil, fmt.Errorf("基础服务尚未完整部署")
	}
	nodeSet, changed := map[string]struct{}{}, 0
	newClaims := map[string]string{}
	for _, assignment := range assignments {
		nodeSet[assignment.NodeID] = struct{}{}
		if current[assignment.ServiceType] != assignment.NodeID {
			changed++
		}
	}
	if changed == 0 {
		return nil, fmt.Errorf("基础服务分布没有变化")
	}
	nodeIDs := make([]string, 0, len(nodeSet))
	for nodeID := range nodeSet {
		nodeIDs = append(nodeIDs, nodeID)
	}
	var readyCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM runtime_environment_nodes en JOIN host_nodes n ON n.id=en.node_id JOIN runtime_cluster_nodes cn ON cn.node_id=n.id WHERE en.environment_id=$1 AND en.node_id=ANY($2::uuid[]) AND cn.desired_action='active' AND cn.cluster_status='ready' AND n.observed_status='online' AND n.last_heartbeat_at>now()-interval '45 seconds'`, environmentID, nodeIDs).Scan(&readyCount); err != nil {
		return nil, err
	}
	if readyCount != len(nodeIDs) {
		return nil, ErrFoundationNotReady
	}
	var nextGeneration int64
	if err := tx.QueryRow(ctx, `SELECT max(desired_generation)+1 FROM runtime_environment_services WHERE environment_id=$1`, environmentID).Scan(&nextGeneration); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, advanceFoundationGenerationSQL, nextGeneration, environmentID); err != nil {
		return nil, err
	}
	// 八项服务共享新代次，但只把实际移动项置为 pending。未移动服务继续显示
	// 当前观测状态，Hostd 下一次完整上报会一起确认新代次并刷新观测时间。
	for _, assignment := range assignments {
		if current[assignment.ServiceType] == assignment.NodeID {
			continue
		}
		claim := ""
		if assignment.ServiceType != "nginx" && assignment.ServiceType != "traefik" {
			workload := foundationWorkloadForLogicalType(assignment.ServiceType)
			claim = newClaims[workload]
			if claim == "" {
				claim = "data-" + workload + "-m" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
				newClaims[workload] = claim
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE runtime_environment_services SET previous_node_id=node_id,previous_storage_claim=storage_claim,node_id=$1,storage_claim=$2,observed_status='pending',last_message='等待从原节点迁移数据',operation='migrate',updated_at=now() WHERE environment_id=$3 AND service_type=$4`, assignment.NodeID, claim, environmentID, assignment.ServiceType); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message,created_by) VALUES($1,$2,'foundation_migration_requested','基础服务分布调整任务已创建','基础服务','success',$3,$4)`, tenant, environmentID, fmt.Sprintf("已提交 %d 项节点调整；有状态服务将先复制数据并校验，再切换运行节点", changed), user); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE runtime_environments SET updated_at=now() WHERE id=$1`, environmentID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.ListRuntimeEnvironmentServices(ctx, tenant, environmentID)
}

func foundationWorkloadForLogicalType(serviceType string) string {
	switch serviceType {
	case "if_history", "if_timeseries":
		return "postgres"
	case "if_realtime":
		return "redis"
	case "if_message":
		return "emqx"
	case "if_object":
		return "object"
	case "nats_jetstream":
		return "nats"
	default:
		return serviceType
	}
}

func (r *PostgreSQLRepository) DeployRuntimeEnvironmentFoundation(ctx context.Context, tenant, environmentID, user string, assignments []FoundationAssignment) ([]RuntimeEnvironmentService, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var lockedEnvironmentID, desiredStatus string
	if err := tx.QueryRow(ctx, `SELECT id::text,desired_status FROM runtime_environments WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL FOR UPDATE`, tenant, environmentID).Scan(&lockedEnvironmentID, &desiredStatus); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	if desiredStatus == "deleting" {
		return nil, ErrEnvironmentDeleting
	}
	existingAssignments := make(map[string]string, len(foundationServiceTypes))
	rows, err := tx.Query(ctx, `SELECT service_type,node_id::text FROM runtime_environment_services WHERE environment_id=$1 FOR UPDATE`, environmentID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var serviceType, nodeID string
		if err := rows.Scan(&serviceType, &nodeID); err != nil {
			rows.Close()
			return nil, err
		}
		existingAssignments[serviceType] = nodeID
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	nodeIDs := make([]string, 0, len(assignments))
	seen := make(map[string]struct{})
	for _, assignment := range assignments {
		if _, ok := seen[assignment.NodeID]; !ok {
			seen[assignment.NodeID] = struct{}{}
			nodeIDs = append(nodeIDs, assignment.NodeID)
		}
	}
	var readyCount int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM runtime_environment_nodes en JOIN host_nodes n ON n.id=en.node_id JOIN runtime_cluster_nodes cn ON cn.node_id=n.id WHERE en.environment_id=$1 AND en.node_id=ANY($2::uuid[]) AND cn.desired_action='active' AND cn.cluster_status='ready' AND n.observed_status='online' AND n.last_heartbeat_at>now()-interval '45 seconds'`, environmentID, nodeIDs).Scan(&readyCount); err != nil {
		return nil, err
	}
	if readyCount != len(nodeIDs) {
		return nil, ErrFoundationNotReady
	}
	if len(existingAssignments) == 0 {
		for _, assignment := range assignments {
			if _, err := tx.Exec(ctx, `INSERT INTO runtime_environment_services(tenant_id,environment_id,node_id,service_type) VALUES($1,$2,$3,$4)`, tenant, environmentID, assignment.NodeID, assignment.ServiceType); err != nil {
				return nil, err
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message,created_by) VALUES($1,$2,'foundation_deploy_requested','基础服务部署任务已创建',$3,'success','已记录 8 项基础服务的单实例多节点分配，等待节点执行',$4)`, tenant, environmentID, "运行环境", user); err != nil {
			return nil, err
		}
	} else {
		// 当前基线只允许按原位置重新应用固定清单，避免在没有数据迁移协议时把
		// 有状态服务直接搬到其他节点，造成 PVC 与数据库数据不可恢复地分离。
		if len(existingAssignments) != len(assignments) {
			return nil, ErrFoundationMoveUnsupported
		}
		for _, assignment := range assignments {
			if existingAssignments[assignment.ServiceType] != assignment.NodeID {
				return nil, ErrFoundationMoveUnsupported
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE runtime_environment_services SET desired_generation=(SELECT max(desired_generation)+1 FROM runtime_environment_services WHERE environment_id=$1),observed_status='pending',last_message='等待节点重新应用基础服务清单',updated_at=now() WHERE environment_id=$1`, environmentID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message,created_by) VALUES($1,$2,'foundation_redeploy_requested','基础服务重新部署任务已创建',$3,'success','保持现有节点分配并重新应用固定清单，用于修复异常或升级内置配置',$4)`, tenant, environmentID, "运行环境", user); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE runtime_environments SET updated_at=now() WHERE id=$1`, environmentID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.ListRuntimeEnvironmentServices(ctx, tenant, environmentID)
}

func (r *PostgreSQLRepository) requireRuntimeEnvironment(ctx context.Context, tenant, environmentID string) error {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT true FROM runtime_environments WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenant, environmentID).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func scanRuntimeEnvironment(row scanner) (RuntimeEnvironment, error) {
	var item RuntimeEnvironment
	err := row.Scan(&item.ID, &item.TenantID, &item.Name, &item.Code, &item.DesiredStatus, &item.IsDefault, &item.Status, &item.NodeCount, &item.OnlineNodeCount, &item.FoundationTotal, &item.FoundationHealthy, &item.ProjectCount, &item.RecentChange, &item.RecentBy, &item.RecentAt, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func scanNodeWithAssignments(row scanner) (Node, error) {
	var item Node
	var caps, summary []byte
	args := nodeScanArgs(&item, &caps, &summary)
	args = append(args, &item.AssignedDeploymentID, &item.AssignedProjectID, &item.AssignedProjectName, &item.EnvironmentID, &item.EnvironmentName)
	err := row.Scan(args...)
	hydrateNode(&item, caps, summary)
	return item, err
}
