package ops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgreSQLRepository struct{ pool *pgxpool.Pool }

const (
	defaultRuntimeClusterName        = "默认运行资源池"
	defaultRuntimeClusterCode        = "default-runtime"
	defaultRuntimeClusterDescription = "由首台 Linux 运行节点接入任务自动创建"
)

type rowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewPostgreSQLRepository(pool *pgxpool.Pool) *PostgreSQLRepository {
	return &PostgreSQLRepository{pool: pool}
}

func (r *PostgreSQLRepository) ListClusters(ctx context.Context, tenant string, f PageFilter) ([]RuntimeCluster, int64, error) {
	rows, err := r.pool.Query(ctx, `SELECT c.id,c.tenant_id,c.name,c.code,COALESCE(c.description,''),c.topology,c.desired_status,c.observed_status,c.controller_status,c.metadata,c.created_at,c.updated_at,(SELECT count(*) FROM host_nodes n WHERE n.runtime_cluster_id=c.id),(SELECT count(*) FROM host_nodes n WHERE n.runtime_cluster_id=c.id AND n.observed_status='online' AND n.last_heartbeat_at>now()-interval '45 seconds'),CASE WHEN (SELECT count(*) FROM host_nodes n WHERE n.runtime_cluster_id=c.id)=0 THEN 'unknown' WHEN (SELECT count(*) FROM host_nodes n WHERE n.runtime_cluster_id=c.id AND n.observed_status='online' AND n.last_heartbeat_at>now()-interval '45 seconds')=(SELECT count(*) FROM host_nodes n WHERE n.runtime_cluster_id=c.id) THEN 'healthy' WHEN (SELECT count(*) FROM host_nodes n WHERE n.runtime_cluster_id=c.id AND n.observed_status='online' AND n.last_heartbeat_at>now()-interval '45 seconds')>0 THEN 'degraded' ELSE 'unavailable' END FROM runtime_clusters c WHERE c.tenant_id=$1 AND ($2='' OR c.name ILIKE '%'||$2||'%' OR c.code ILIKE '%'||$2||'%') ORDER BY c.updated_at DESC LIMIT $3 OFFSET $4`, tenant, f.Search, f.PageSize, (f.Page-1)*f.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []RuntimeCluster{}
	for rows.Next() {
		x, err := scanCluster(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, x)
	}
	var total int64
	err = r.pool.QueryRow(ctx, `SELECT count(*) FROM runtime_clusters WHERE tenant_id=$1 AND ($2='' OR name ILIKE '%'||$2||'%' OR code ILIKE '%'||$2||'%')`, tenant, f.Search).Scan(&total)
	return items, total, err
}
func (r *PostgreSQLRepository) CreateCluster(ctx context.Context, tenant, user string, in CreateClusterInput) (RuntimeCluster, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return RuntimeCluster{}, err
	}
	defer tx.Rollback(ctx)
	// 显式创建与首次接入的自动创建共用租户行锁，避免并发产生两个“首个”集群。
	if err = lockTenantForClusterChange(ctx, tx, tenant); err != nil {
		return RuntimeCluster{}, err
	}
	meta, _ := json.Marshal(in.Metadata)
	row := tx.QueryRow(ctx, `INSERT INTO runtime_clusters(tenant_id,name,code,description,topology,metadata,created_by) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,tenant_id,name,code,COALESCE(description,''),topology,desired_status,observed_status,controller_status,metadata,created_at,updated_at,0,0,'unknown'`, tenant, in.Name, in.Code, in.Description, in.Topology, meta, user)
	cluster, err := scanCluster(row)
	if err != nil {
		return RuntimeCluster{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return RuntimeCluster{}, err
	}
	return cluster, nil
}
func (r *PostgreSQLRepository) GetCluster(ctx context.Context, tenant, id string) (RuntimeCluster, error) {
	x, err := scanCluster(r.pool.QueryRow(ctx, `SELECT c.id,c.tenant_id,c.name,c.code,COALESCE(c.description,''),c.topology,c.desired_status,c.observed_status,c.controller_status,c.metadata,c.created_at,c.updated_at,(SELECT count(*) FROM host_nodes n WHERE n.runtime_cluster_id=c.id),(SELECT count(*) FROM host_nodes n WHERE n.runtime_cluster_id=c.id AND n.observed_status='online' AND n.last_heartbeat_at>now()-interval '45 seconds'),CASE WHEN (SELECT count(*) FROM host_nodes n WHERE n.runtime_cluster_id=c.id)=0 THEN 'unknown' WHEN (SELECT count(*) FROM host_nodes n WHERE n.runtime_cluster_id=c.id AND n.observed_status='online' AND n.last_heartbeat_at>now()-interval '45 seconds')=(SELECT count(*) FROM host_nodes n WHERE n.runtime_cluster_id=c.id) THEN 'healthy' WHEN (SELECT count(*) FROM host_nodes n WHERE n.runtime_cluster_id=c.id AND n.observed_status='online' AND n.last_heartbeat_at>now()-interval '45 seconds')>0 THEN 'degraded' ELSE 'unavailable' END FROM runtime_clusters c WHERE c.tenant_id=$1 AND c.id=$2`, tenant, id))
	return x, mapNotFound(err)
}

func (r *PostgreSQLRepository) ListEnrollments(ctx context.Context, tenant string, f PageFilter) ([]Enrollment, int64, error) {
	_, _ = r.pool.Exec(ctx, `UPDATE node_enrollments SET status='expired',updated_at=now() WHERE tenant_id=$1 AND status='created' AND expires_at<=now()`, tenant)
	rows, err := r.pool.Query(ctx, `SELECT id,tenant_id,COALESCE(runtime_cluster_id::text,''),role,COALESCE(display_name,''),status,expires_at,claimed_at,COALESCE(claimed_by_node_id::text,''),approved_at,rejected_at,created_at,updated_at FROM node_enrollments WHERE tenant_id=$1 AND ($2='' OR role ILIKE '%'||$2||'%' OR COALESCE(display_name,'') ILIKE '%'||$2||'%') ORDER BY created_at DESC LIMIT $3 OFFSET $4`, tenant, f.Search, f.PageSize, (f.Page-1)*f.PageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []Enrollment{}
	for rows.Next() {
		x, e := scanEnrollment(rows)
		if e != nil {
			return nil, 0, e
		}
		r.attachEnrollmentHost(ctx, &x)
		items = append(items, x)
	}
	var total int64
	err = r.pool.QueryRow(ctx, `SELECT count(*) FROM node_enrollments WHERE tenant_id=$1 AND ($2='' OR role ILIKE '%'||$2||'%' OR COALESCE(display_name,'') ILIKE '%'||$2||'%')`, tenant, f.Search).Scan(&total)
	return items, total, err
}
func (r *PostgreSQLRepository) CreateEnrollment(ctx context.Context, tenant, user string, in CreateEnrollmentInput, hash string) (Enrollment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Enrollment{}, err
	}
	defer tx.Rollback(ctx)
	enrollment, err := createEnrollmentRecord(ctx, tx, tenant, user, in, hash)
	if err != nil {
		return Enrollment{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Enrollment{}, err
	}
	return enrollment, nil
}

// createEnrollmentRecord 在同一个事务内解析运行集群并写入接入任务。
// 只有租户完全没有运行集群时才自动创建；已有集群时不猜测目标。
func createEnrollmentRecord(ctx context.Context, query rowQuerier, tenant, user string, in CreateEnrollmentInput, hash string) (Enrollment, error) {
	if in.Role == RoleRuntimeLinux && in.RuntimeClusterID == "" {
		if err := lockTenantForClusterChange(ctx, query, tenant); err != nil {
			return Enrollment{}, err
		}
		var clusterExists bool
		if err := query.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM runtime_clusters WHERE tenant_id=$1)`, tenant).Scan(&clusterExists); err != nil {
			return Enrollment{}, err
		}
		if clusterExists {
			return Enrollment{}, ErrRuntimeClusterSelectionRequired
		}
		metadata, err := json.Marshal(map[string]any{
			"systemManaged": true,
			"isDefault":     true,
			"createdFrom":   "first_runtime_enrollment",
		})
		if err != nil {
			return Enrollment{}, err
		}
		if err = query.QueryRow(ctx, `INSERT INTO runtime_clusters(tenant_id,name,code,description,topology,metadata,created_by) VALUES($1,$2,$3,$4,'single_node',$5,$6) RETURNING id::text`, tenant, defaultRuntimeClusterName, defaultRuntimeClusterCode, defaultRuntimeClusterDescription, metadata, user).Scan(&in.RuntimeClusterID); err != nil {
			return Enrollment{}, err
		}
	}
	enrollment, err := scanEnrollment(query.QueryRow(ctx, `INSERT INTO node_enrollments(tenant_id,runtime_cluster_id,role,display_name,code_hash,expires_at,created_by) VALUES($1,NULLIF($2,'')::uuid,$3,$4,$5,now()+$6::interval,$7) RETURNING id,tenant_id,COALESCE(runtime_cluster_id::text,''),role,COALESCE(display_name,''),status,expires_at,claimed_at,COALESCE(claimed_by_node_id::text,''),approved_at,rejected_at,created_at,updated_at`, tenant, in.RuntimeClusterID, in.Role, in.DisplayName, hash, in.TTL.String(), user))
	return enrollment, mapNotFound(err)
}

func lockTenantForClusterChange(ctx context.Context, query rowQuerier, tenant string) error {
	var lockedTenant string
	err := query.QueryRow(ctx, `SELECT id::text FROM tenants WHERE id=$1 FOR UPDATE`, tenant).Scan(&lockedTenant)
	return mapNotFound(err)
}
func (r *PostgreSQLRepository) GetEnrollment(ctx context.Context, tenant, id string) (Enrollment, error) {
	_, _ = r.pool.Exec(ctx, `UPDATE node_enrollments SET status='expired',updated_at=now() WHERE tenant_id=$1 AND id=$2 AND status='created' AND expires_at<=now()`, tenant, id)
	x, e := scanEnrollment(r.pool.QueryRow(ctx, `SELECT id,tenant_id,COALESCE(runtime_cluster_id::text,''),role,COALESCE(display_name,''),status,expires_at,claimed_at,COALESCE(claimed_by_node_id::text,''),approved_at,rejected_at,created_at,updated_at FROM node_enrollments WHERE tenant_id=$1 AND id=$2`, tenant, id))
	if e != nil {
		return x, mapNotFound(e)
	}
	r.attachEnrollmentHost(ctx, &x)
	return x, nil
}
func (r *PostgreSQLRepository) ApproveEnrollment(ctx context.Context, tenant, id, user string, approve bool) (Enrollment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Enrollment{}, err
	}
	defer tx.Rollback(ctx)

	// 锁住接入任务，避免审批和拒绝同时完成。运行节点还会锁住对应集群，
	// 使 single_node 的“仅一个活动运行节点”检查和写入处于同一临界区。
	var role, clusterID string
	err = tx.QueryRow(ctx, `SELECT role,COALESCE(runtime_cluster_id::text,'') FROM node_enrollments WHERE id=$1 AND tenant_id=$2 AND status='claimed' FOR UPDATE`, id, tenant).Scan(&role, &clusterID)
	if err != nil {
		return Enrollment{}, mapNotFound(err)
	}
	if approve && role == RoleRuntimeLinux {
		var topology string
		err = tx.QueryRow(ctx, `SELECT topology FROM runtime_clusters WHERE id=$1 AND tenant_id=$2 FOR UPDATE`, clusterID, tenant).Scan(&topology)
		if err != nil {
			return Enrollment{}, mapNotFound(err)
		}
		var activeCount int
		err = tx.QueryRow(ctx, `SELECT count(*) FROM host_nodes WHERE runtime_cluster_id=$1 AND role=$2 AND approved_at IS NOT NULL AND desired_status='active'`, clusterID, RoleRuntimeLinux).Scan(&activeCount)
		if err != nil {
			return Enrollment{}, err
		}
		if err = runtimeApprovalAllowed(topology, activeCount); err != nil {
			return Enrollment{}, err
		}
	}

	var x Enrollment
	if approve {
		err = tx.QueryRow(ctx, `UPDATE node_enrollments SET status='approved',approved_at=now(),approved_by=$1,updated_at=now() WHERE tenant_id=$2 AND id=$3 RETURNING id,tenant_id,COALESCE(runtime_cluster_id::text,''),role,COALESCE(display_name,''),status,expires_at,claimed_at,COALESCE(claimed_by_node_id::text,''),approved_at,rejected_at,created_at,updated_at`, user, tenant, id).Scan(&x.ID, &x.TenantID, &x.RuntimeClusterID, &x.Role, &x.DisplayName, &x.Status, &x.ExpiresAt, &x.ClaimedAt, &x.ClaimedByNodeID, &x.ApprovedAt, &x.RejectedAt, &x.CreatedAt, &x.UpdatedAt)
		if err == nil {
			var hostID string
			err = tx.QueryRow(ctx, `UPDATE host_nodes SET observed_status='offline',approved_at=now(),approved_by=$1,updated_at=now() WHERE enrollment_id=$2 RETURNING id`, user, id).Scan(&hostID)
		}
	} else {
		err = tx.QueryRow(ctx, `UPDATE node_enrollments SET status='rejected',rejected_at=now(),rejected_by=$1,updated_at=now() WHERE tenant_id=$2 AND id=$3 RETURNING id,tenant_id,COALESCE(runtime_cluster_id::text,''),role,COALESCE(display_name,''),status,expires_at,claimed_at,COALESCE(claimed_by_node_id::text,''),approved_at,rejected_at,created_at,updated_at`, user, tenant, id).Scan(&x.ID, &x.TenantID, &x.RuntimeClusterID, &x.Role, &x.DisplayName, &x.Status, &x.ExpiresAt, &x.ClaimedAt, &x.ClaimedByNodeID, &x.ApprovedAt, &x.RejectedAt, &x.CreatedAt, &x.UpdatedAt)
		if err == nil {
			var hostID string
			err = tx.QueryRow(ctx, `UPDATE host_nodes SET desired_status='revoked',observed_status='revoked',agent_token_hash='revoked:'||id::text,updated_at=now() WHERE enrollment_id=$1 RETURNING id`, id).Scan(&hostID)
		}
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Enrollment{}, fmt.Errorf("已认领的接入任务缺少主机节点")
		}
		return Enrollment{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Enrollment{}, err
	}
	r.attachEnrollmentHost(ctx, &x)
	return x, nil
}
func (r *PostgreSQLRepository) ClaimEnrollment(ctx context.Context, in ClaimEnrollmentInput, agentHash string) (Enrollment, HostNode, error) {
	tx, e := r.pool.Begin(ctx)
	if e != nil {
		return Enrollment{}, HostNode{}, e
	}
	defer tx.Rollback(ctx)
	var en Enrollment
	e = tx.QueryRow(ctx, `UPDATE node_enrollments SET status='claimed',claimed_at=now(),updated_at=now() WHERE code_hash=$1 AND status='created' AND expires_at>now() RETURNING id,tenant_id,COALESCE(runtime_cluster_id::text,''),role,COALESCE(display_name,''),status,expires_at,claimed_at,COALESCE(claimed_by_node_id::text,''),approved_at,rejected_at,created_at,updated_at`, hashToken(in.Code)).Scan(&en.ID, &en.TenantID, &en.RuntimeClusterID, &en.Role, &en.DisplayName, &en.Status, &en.ExpiresAt, &en.ClaimedAt, &en.ClaimedByNodeID, &en.ApprovedAt, &en.RejectedAt, &en.CreatedAt, &en.UpdatedAt)
	if e != nil {
		return Enrollment{}, HostNode{}, ErrEnrollmentUnavailable
	}
	if e = validateClaimPlatform(en.Role, in.OS, in.Architecture); e != nil {
		return Enrollment{}, HostNode{}, e
	}
	cap, _ := json.Marshal(in.Capabilities)
	// 管理员创建接入任务时给出的名称优先，Agent 仅补充机器自报事实。
	name := en.DisplayName
	if name == "" {
		name = in.DisplayName
	}
	if name == "" {
		name = in.Hostname
	}
	var node HostNode
	e = tx.QueryRow(ctx, `INSERT INTO host_nodes(tenant_id,runtime_cluster_id,enrollment_id,role,display_name,hostname,os,architecture,agent_version,machine_fingerprint,ip_address,agent_token_hash,capabilities) VALUES($1,NULLIF($2,'')::uuid,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id,tenant_id,COALESCE(runtime_cluster_id::text,''),enrollment_id,role,display_name,hostname,os,architecture,COALESCE(agent_version,''),COALESCE(machine_fingerprint,''),COALESCE(ip_address,''),desired_status,observed_status,resource_summary,capabilities,last_heartbeat_at,approved_at,created_at,updated_at`, en.TenantID, en.RuntimeClusterID, en.ID, en.Role, name, in.Hostname, in.OS, in.Architecture, in.AgentVersion, in.MachineFingerprint, in.IPAddress, agentHash, cap).Scan(hostScanArgs(&node)...)
	if e != nil {
		return Enrollment{}, HostNode{}, e
	}
	_, e = tx.Exec(ctx, `UPDATE node_enrollments SET claimed_by_node_id=$1,updated_at=now() WHERE id=$2`, node.ID, en.ID)
	if e != nil {
		return Enrollment{}, HostNode{}, e
	}
	e = tx.Commit(ctx)
	return en, node, e
}

func (r *PostgreSQLRepository) ListHostNodes(ctx context.Context, tenant string, f PageFilter) ([]HostNode, int64, error) {
	rows, e := r.pool.Query(ctx, `SELECT n.id,n.tenant_id,COALESCE(n.runtime_cluster_id::text,''),n.enrollment_id,n.role,n.display_name,n.hostname,n.os,n.architecture,COALESCE(n.agent_version,''),COALESCE(n.machine_fingerprint,''),COALESCE(n.ip_address,''),n.desired_status,n.observed_status,n.resource_summary,n.capabilities,n.last_heartbeat_at,n.approved_at,n.created_at,n.updated_at,COALESCE(c.name,'') FROM host_nodes n LEFT JOIN runtime_clusters c ON c.id=n.runtime_cluster_id AND c.tenant_id=n.tenant_id WHERE n.tenant_id=$1 AND ($2='' OR n.display_name ILIKE '%'||$2||'%' OR n.hostname ILIKE '%'||$2||'%') ORDER BY n.updated_at DESC LIMIT $3 OFFSET $4`, tenant, f.Search, f.PageSize, (f.Page-1)*f.PageSize)
	if e != nil {
		return nil, 0, e
	}
	defer rows.Close()
	items := []HostNode{}
	for rows.Next() {
		x, scanErr := scanManagementHostNode(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, x)
	}
	var total int64
	e = r.pool.QueryRow(ctx, `SELECT count(*) FROM host_nodes WHERE tenant_id=$1 AND ($2='' OR display_name ILIKE '%'||$2||'%' OR hostname ILIKE '%'||$2||'%')`, tenant, f.Search).Scan(&total)
	return items, total, e
}
func (r *PostgreSQLRepository) GetHostNode(ctx context.Context, tenant, id string) (HostNode, error) {
	x, e := scanManagementHostNode(r.pool.QueryRow(ctx, `SELECT n.id,n.tenant_id,COALESCE(n.runtime_cluster_id::text,''),n.enrollment_id,n.role,n.display_name,n.hostname,n.os,n.architecture,COALESCE(n.agent_version,''),COALESCE(n.machine_fingerprint,''),COALESCE(n.ip_address,''),n.desired_status,n.observed_status,n.resource_summary,n.capabilities,n.last_heartbeat_at,n.approved_at,n.created_at,n.updated_at,COALESCE(c.name,'') FROM host_nodes n LEFT JOIN runtime_clusters c ON c.id=n.runtime_cluster_id AND c.tenant_id=n.tenant_id WHERE n.tenant_id=$1 AND n.id=$2`, tenant, id))
	return x, mapNotFound(e)
}

// attachEnrollmentHost 为审批列表补齐 Agent 自报机器事实，失败时保留接入任务本身。
func (r *PostgreSQLRepository) attachEnrollmentHost(ctx context.Context, enrollment *Enrollment) {
	if enrollment.ClaimedByNodeID == "" {
		return
	}
	node, err := r.GetHostNode(ctx, enrollment.TenantID, enrollment.ClaimedByNodeID)
	if err != nil {
		return
	}
	enrollment.ReportedHostName, enrollment.MachineFingerprint, enrollment.IPAddress = node.Hostname, node.MachineFingerprint, node.IPAddress
	enrollment.Node = &node
}
func (r *PostgreSQLRepository) Heartbeat(ctx context.Context, id, hash string, in HeartbeatInput) (HostNode, []Workload, error) {
	summary, _ := json.Marshal(in.ResourceSummary)
	var x HostNode
	e := r.pool.QueryRow(ctx, `UPDATE host_nodes SET observed_status=CASE WHEN desired_status='revoked' THEN 'revoked' WHEN observed_status='pending_approval' THEN 'pending_approval' ELSE 'online' END,resource_summary=$1,agent_version=COALESCE(NULLIF($2,''),agent_version),last_heartbeat_at=now(),updated_at=now() WHERE id=$3 AND agent_token_hash=$4 RETURNING id,tenant_id,COALESCE(runtime_cluster_id::text,''),enrollment_id,role,display_name,hostname,os,architecture,COALESCE(agent_version,''),COALESCE(machine_fingerprint,''),COALESCE(ip_address,''),desired_status,observed_status,resource_summary,capabilities,last_heartbeat_at,approved_at,created_at,updated_at`, summary, in.AgentVersion, id, hash).Scan(hostScanArgs(&x)...)
	if e != nil {
		return HostNode{}, nil, ErrAgentUnauthorized
	}
	// 待审批或已吊销 Agent 只能上报主机资源，不能篡改任何工作负载观测。
	if x.ObservedStatus != "online" || x.DesiredStatus != "active" {
		return x, []Workload{}, nil
	}
	affected := map[string]struct{}{}
	for _, obs := range in.Workloads {
		var deploymentID string
		e = r.pool.QueryRow(ctx, `UPDATE workloads SET observed_status=$1,replicas_observed=$2,observed_generation=$3,last_message=$4,observed_at=now(),updated_at=now() WHERE id=$5 AND desired_generation=$3 AND (host_node_id=$6 OR (host_node_id IS NULL AND $7='runtime_linux' AND project_deployment_id IN (SELECT id FROM project_deployments WHERE runtime_cluster_id=NULLIF($8,'')::uuid))) RETURNING project_deployment_id`, obs.ObservedStatus, obs.ReplicasObserved, obs.ObservedGeneration, obs.Message, obs.WorkloadID, id, x.Role, x.RuntimeClusterID).Scan(&deploymentID)
		if e == nil {
			affected[deploymentID] = struct{}{}
		} else if !errors.Is(e, pgx.ErrNoRows) {
			// 连接在具体采集节点上的 workload 没有 runtime cluster；空 cluster id
			// 不能作为 UUID 参与比较。除此之外的 SQL 错误必须向 Agent 返回，避免
			// 心跳表面成功、工作负载观测却长期停留在 pending。
			return x, nil, e
		}
	}
	for depID := range affected {
		if e = r.reconcileDeployment(ctx, depID); e != nil {
			return x, nil, e
		}
	}
	commands, e := r.pendingWorkloads(ctx, x)
	return x, commands, e
}
func (r *PostgreSQLRepository) AgentCommands(ctx context.Context, id, hash string) ([]AgentCommand, error) {
	var node HostNode
	e := r.pool.QueryRow(ctx, `SELECT id,tenant_id,COALESCE(runtime_cluster_id::text,''),enrollment_id,role,display_name,hostname,os,architecture,COALESCE(agent_version,''),COALESCE(machine_fingerprint,''),COALESCE(ip_address,''),desired_status,observed_status,resource_summary,capabilities,last_heartbeat_at,approved_at,created_at,updated_at FROM host_nodes WHERE id=$1 AND agent_token_hash=$2`, id, hash).Scan(hostScanArgs(&node)...)
	if e != nil {
		return nil, ErrAgentUnauthorized
	}
	if node.ObservedStatus != "online" || node.DesiredStatus != "active" || node.ApprovedAt == nil {
		return []AgentCommand{}, nil
	}
	workloads, e := r.pendingWorkloads(ctx, node)
	if e != nil {
		return nil, e
	}
	items := make([]AgentCommand, 0, len(workloads))
	for _, w := range workloads {
		var runID, version string
		_ = r.pool.QueryRow(ctx, `SELECT r.id,d.version FROM deployment_runs r JOIN project_deployments d ON d.id=r.project_deployment_id WHERE r.project_deployment_id=$1 AND r.observed_status='pending' ORDER BY r.started_at DESC LIMIT 1`, w.ProjectDeploymentID).Scan(&runID, &version)
		if runID != "" {
			_, _ = r.pool.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) SELECT $1,'dispatched','agent fetched desired workload' WHERE NOT EXISTS (SELECT 1 FROM deployment_run_events WHERE deployment_run_id=$1 AND stage='dispatched')`, runID)
		}
		items = append(items, AgentCommand{RunID: runID, DeploymentID: w.ProjectDeploymentID, WorkloadID: w.ID, Role: w.Role, DesiredStatus: w.DesiredStatus, Operation: w.LastOperation, Generation: w.DesiredGeneration, Version: version, ReplicasDesired: w.ReplicasDesired})
	}
	return items, nil
}
func (r *PostgreSQLRepository) pendingWorkloads(ctx context.Context, node HostNode) ([]Workload, error) {
	// Agent 每轮都拉取完整 desired state，而不是只拉数据库看来“未完成”的命令。
	// Agent 重启后本机进程表为空，但中心最后一次 observed 仍可能是 running；完整下发
	// 让 Agent 依靠同 generation 幂等检查自行恢复缺失进程，符合调和模型。
	rows, e := r.pool.Query(ctx, `SELECT w.id,w.tenant_id,w.project_deployment_id,COALESCE(w.host_node_id::text,''),w.role,w.desired_status,w.observed_status,w.replicas_desired,w.replicas_observed,w.desired_generation,w.observed_generation,w.last_operation,COALESCE(w.last_message,''),w.observed_at,w.created_at,w.updated_at FROM workloads w JOIN project_deployments d ON d.id=w.project_deployment_id WHERE (w.host_node_id=$1 OR (w.host_node_id IS NULL AND d.runtime_cluster_id=NULLIF($2,'')::uuid AND $3='runtime_linux')) ORDER BY w.updated_at`, node.ID, node.RuntimeClusterID, node.Role)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []Workload{}
	for rows.Next() {
		var w Workload
		if e = rows.Scan(workloadScanArgs(&w)...); e != nil {
			return nil, e
		}
		items = append(items, w)
	}
	return items, rows.Err()
}

func (r *PostgreSQLRepository) ListDeployments(ctx context.Context, tenant string, f PageFilter) ([]ProjectDeployment, int64, error) {
	rows, e := r.pool.Query(ctx, `SELECT d.id,d.tenant_id,d.project_id,d.runtime_cluster_id,p.name,c.name,COALESCE((SELECT r.id::text FROM deployment_runs r WHERE r.project_deployment_id=d.id ORDER BY r.started_at DESC LIMIT 1),''),d.version,d.deployment_mode,d.desired_status,d.observed_status,CASE WHEN d.observed_status='running' THEN 'healthy' WHEN d.observed_status='degraded' THEN 'degraded' WHEN d.observed_status='failed' THEN 'unavailable' ELSE 'unknown' END,COALESCE((SELECT r.progress FROM deployment_runs r WHERE r.project_deployment_id=d.id ORDER BY r.started_at DESC LIMIT 1),0),d.created_at,d.updated_at FROM project_deployments d JOIN projects p ON p.id=d.project_id JOIN runtime_clusters c ON c.id=d.runtime_cluster_id WHERE d.tenant_id=$1 AND ($2='' OR d.project_id::text=$2 OR p.name ILIKE '%'||$2||'%') ORDER BY d.updated_at DESC LIMIT $3 OFFSET $4`, tenant, f.Search, f.PageSize, (f.Page-1)*f.PageSize)
	if e != nil {
		return nil, 0, e
	}
	defer rows.Close()
	items := []ProjectDeployment{}
	for rows.Next() {
		var d ProjectDeployment
		if e = rows.Scan(&d.ID, &d.TenantID, &d.ProjectID, &d.RuntimeClusterID, &d.ProjectName, &d.RuntimeClusterName, &d.LatestRunID, &d.Version, &d.DeploymentMode, &d.DesiredStatus, &d.ObservedStatus, &d.Health, &d.Progress, &d.CreatedAt, &d.UpdatedAt); e != nil {
			return nil, 0, e
		}
		d.Workloads, _ = r.listWorkloads(ctx, d.ID)
		items = append(items, d)
	}
	var total int64
	e = r.pool.QueryRow(ctx, `SELECT count(*) FROM project_deployments d JOIN projects p ON p.id=d.project_id WHERE d.tenant_id=$1 AND ($2='' OR d.project_id::text=$2 OR p.name ILIKE '%'||$2||'%')`, tenant, f.Search).Scan(&total)
	return items, total, e
}
func (r *PostgreSQLRepository) CreateDeployment(ctx context.Context, tenant, user string, in CreateDeploymentInput) (ProjectDeployment, DeploymentRun, error) {
	tx, e := r.pool.Begin(ctx)
	if e != nil {
		return ProjectDeployment{}, DeploymentRun{}, e
	}
	defer tx.Rollback(ctx)
	version := in.Version
	if version == "" {
		version = "demo-v1"
	}
	var d ProjectDeployment
	e = tx.QueryRow(ctx, `INSERT INTO project_deployments(tenant_id,project_id,runtime_cluster_id,version,deployment_mode,created_by) VALUES($1,$2,$3,$4,$5,$6) RETURNING id,tenant_id,project_id,runtime_cluster_id,version,deployment_mode,desired_status,observed_status,created_at,updated_at`, tenant, in.ProjectID, in.RuntimeClusterID, version, in.DeploymentMode, user).Scan(&d.ID, &d.TenantID, &d.ProjectID, &d.RuntimeClusterID, &d.Version, &d.DeploymentMode, &d.DesiredStatus, &d.ObservedStatus, &d.CreatedAt, &d.UpdatedAt)
	if e != nil {
		return d, DeploymentRun{}, mapDeploymentCreateError(e)
	}
	for _, w := range in.Workloads {
		rep := w.Replicas
		if rep <= 0 {
			rep = 1
		}
		var x Workload
		e = tx.QueryRow(ctx, `INSERT INTO workloads(tenant_id,project_deployment_id,host_node_id,role,replicas_desired) VALUES($1,$2,NULLIF($3,'')::uuid,$4,$5) RETURNING id,tenant_id,project_deployment_id,COALESCE(host_node_id::text,''),role,desired_status,observed_status,replicas_desired,replicas_observed,desired_generation,observed_generation,last_operation,COALESCE(last_message,''),observed_at,created_at,updated_at`, tenant, d.ID, w.HostNodeID, w.Role, rep).Scan(workloadScanArgs(&x)...)
		if e != nil {
			return d, DeploymentRun{}, e
		}
		d.Workloads = append(d.Workloads, x)
	}
	var run DeploymentRun
	e = tx.QueryRow(ctx, `INSERT INTO deployment_runs(tenant_id,project_deployment_id,operation,desired_status,created_by) VALUES($1,$2,'deploy','running',$3) RETURNING id,tenant_id,project_deployment_id,operation,desired_status,observed_status,progress,COALESCE(message,''),started_at,completed_at`, tenant, d.ID, user).Scan(runScanArgs(&run)...)
	if e != nil {
		return d, run, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) VALUES($1,'queued','deployment queued')`, run.ID)
	if e != nil {
		return d, run, e
	}
	e = tx.Commit(ctx)
	if e != nil {
		return d, run, e
	}
	d, e = r.GetDeployment(ctx, tenant, d.ID)
	return d, run, e
}
func (r *PostgreSQLRepository) ValidateDeploymentTargets(ctx context.Context, tenant string, in CreateDeploymentInput) error {
	var count int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM projects WHERE id=$1 AND tenant_id=$2`, in.ProjectID, tenant).Scan(&count); err != nil || count != 1 {
		return ErrNotFound
	}
	var clusterStatus, topology string
	if err := r.pool.QueryRow(ctx, `SELECT desired_status,topology FROM runtime_clusters WHERE id=$1 AND tenant_id=$2`, in.RuntimeClusterID, tenant).Scan(&clusterStatus, &topology); err != nil {
		return ErrNotFound
	}
	if clusterStatus != "ready" {
		return fmt.Errorf("运行集群当前不能调度：集群未处于就绪状态")
	}
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM project_deployments WHERE tenant_id=$1 AND project_id=$2 AND runtime_cluster_id=$3)`, tenant, in.ProjectID, in.RuntimeClusterID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return ErrDeploymentExists
	}
	if err := deploymentTopologyAllowed(topology); err != nil {
		return err
	}

	requiresRuntime := false
	for _, w := range in.Workloads {
		if w.Role != WorkloadRoleCollector {
			requiresRuntime = true
			continue
		}
		var schedulable bool
		if err := r.pool.QueryRow(ctx, `SELECT approved_at IS NOT NULL AND desired_status='active' AND observed_status='online' AND last_heartbeat_at>now()-interval '45 seconds' FROM host_nodes WHERE id=$1 AND tenant_id=$2 AND role IN ($3,$4)`, w.HostNodeID, tenant, RoleCollectorLinux, RoleCollectorWindows).Scan(&schedulable); err != nil {
			return ErrNotFound
		}
		if !schedulable {
			return fmt.Errorf("采集节点当前不能调度：需已审批、启用且保持在线")
		}
	}
	if requiresRuntime {
		var runtimeNodeCount int
		if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM host_nodes WHERE runtime_cluster_id=$1 AND role=$2 AND approved_at IS NOT NULL AND desired_status='active' AND observed_status='online' AND last_heartbeat_at>now()-interval '45 seconds'`, in.RuntimeClusterID, RoleRuntimeLinux).Scan(&runtimeNodeCount); err != nil {
			return err
		}
		if runtimeNodeCount == 0 {
			return fmt.Errorf("运行集群当前不能调度：没有已审批且在线的运行节点")
		}
	}
	return nil
}
func (r *PostgreSQLRepository) GetDeployment(ctx context.Context, tenant, id string) (ProjectDeployment, error) {
	var d ProjectDeployment
	e := r.pool.QueryRow(ctx, `SELECT d.id,d.tenant_id,d.project_id,d.runtime_cluster_id,p.name,c.name,COALESCE((SELECT r.id::text FROM deployment_runs r WHERE r.project_deployment_id=d.id ORDER BY r.started_at DESC LIMIT 1),''),d.version,d.deployment_mode,d.desired_status,d.observed_status,CASE WHEN d.observed_status='running' THEN 'healthy' WHEN d.observed_status='degraded' THEN 'degraded' WHEN d.observed_status='failed' THEN 'unavailable' ELSE 'unknown' END,COALESCE((SELECT r.progress FROM deployment_runs r WHERE r.project_deployment_id=d.id ORDER BY r.started_at DESC LIMIT 1),0),d.created_at,d.updated_at FROM project_deployments d JOIN projects p ON p.id=d.project_id JOIN runtime_clusters c ON c.id=d.runtime_cluster_id WHERE d.tenant_id=$1 AND d.id=$2`, tenant, id).Scan(&d.ID, &d.TenantID, &d.ProjectID, &d.RuntimeClusterID, &d.ProjectName, &d.RuntimeClusterName, &d.LatestRunID, &d.Version, &d.DeploymentMode, &d.DesiredStatus, &d.ObservedStatus, &d.Health, &d.Progress, &d.CreatedAt, &d.UpdatedAt)
	if e != nil {
		return d, mapNotFound(e)
	}
	d.Workloads, e = r.listWorkloads(ctx, d.ID)
	return d, e
}
func (r *PostgreSQLRepository) GetRun(ctx context.Context, tenant, id string) (DeploymentRun, error) {
	var x DeploymentRun
	e := r.pool.QueryRow(ctx, `SELECT r.id,r.tenant_id,r.project_deployment_id,r.operation,r.desired_status,r.observed_status,r.progress,COALESCE(r.message,''),r.started_at,r.completed_at FROM deployment_runs r WHERE r.tenant_id=$1 AND r.id=$2`, tenant, id).Scan(runScanArgs(&x)...)
	return x, mapNotFound(e)
}
func (r *PostgreSQLRepository) ListRunEvents(ctx context.Context, tenant, id string) ([]DeploymentRunEvent, error) {
	rows, e := r.pool.Query(ctx, `SELECT e.id,e.deployment_run_id,e.stage,e.message,e.created_at FROM deployment_run_events e JOIN deployment_runs r ON r.id=e.deployment_run_id WHERE e.deployment_run_id=$1 AND r.tenant_id=$2 ORDER BY e.created_at`, id, tenant)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []DeploymentRunEvent{}
	for rows.Next() {
		var x DeploymentRunEvent
		if e = rows.Scan(&x.ID, &x.DeploymentRunID, &x.Stage, &x.Message, &x.CreatedAt); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}
func (r *PostgreSQLRepository) OperateWorkload(ctx context.Context, tenant, did, role, operation, user string) (ProjectDeployment, DeploymentRun, error) {
	if !validWorkloadRole(role) {
		return ProjectDeployment{}, DeploymentRun{}, fmt.Errorf("工作负载角色不支持")
	}
	desired := "running"
	if operation == "stop" {
		desired = "stopped"
	}
	tx, e := r.pool.Begin(ctx)
	if e != nil {
		return ProjectDeployment{}, DeploymentRun{}, e
	}
	defer tx.Rollback(ctx)
	// 部署行是同一部署的操作串行锁，避免两次点击同时生成彼此覆盖的 generation。
	var deploymentID string
	if e = tx.QueryRow(ctx, `SELECT id FROM project_deployments WHERE id=$1 AND tenant_id=$2 FOR UPDATE`, did, tenant).Scan(&deploymentID); e != nil {
		return ProjectDeployment{}, DeploymentRun{}, mapNotFound(e)
	}
	var pendingRunID string
	e = tx.QueryRow(ctx, `SELECT id FROM deployment_runs WHERE project_deployment_id=$1 AND observed_status='pending' FOR UPDATE`, did).Scan(&pendingRunID)
	if e == nil {
		return ProjectDeployment{}, DeploymentRun{}, ErrDeploymentBusy
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return ProjectDeployment{}, DeploymentRun{}, e
	}
	tag, e := tx.Exec(ctx, `UPDATE workloads SET desired_status=$1,observed_status='pending',desired_generation=desired_generation+1,last_operation=$2,updated_at=now() WHERE project_deployment_id=$3 AND role=$4 AND tenant_id=$5`, desired, operation, did, role, tenant)
	if e != nil || tag.RowsAffected() == 0 {
		return ProjectDeployment{}, DeploymentRun{}, ErrNotFound
	}
	_, e = tx.Exec(ctx, `UPDATE project_deployments SET desired_status=CASE WHEN EXISTS (SELECT 1 FROM workloads WHERE project_deployment_id=$1 AND desired_status='running') THEN 'running' ELSE 'stopped' END,observed_status='pending',updated_at=now() WHERE id=$1 AND tenant_id=$2`, did, tenant)
	if e != nil {
		return ProjectDeployment{}, DeploymentRun{}, e
	}
	var run DeploymentRun
	e = tx.QueryRow(ctx, `INSERT INTO deployment_runs(tenant_id,project_deployment_id,operation,desired_status,created_by) VALUES($1,$2,$3,$4,$5) RETURNING id,tenant_id,project_deployment_id,operation,desired_status,observed_status,progress,COALESCE(message,''),started_at,completed_at`, tenant, did, operation, desired, user).Scan(runScanArgs(&run)...)
	if e != nil {
		return ProjectDeployment{}, run, e
	}
	_, e = tx.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) VALUES($1,'queued',$2)`, run.ID, operation+" queued")
	if e != nil {
		return ProjectDeployment{}, run, e
	}
	e = tx.Commit(ctx)
	if e != nil {
		return ProjectDeployment{}, run, e
	}
	d, e := r.GetDeployment(ctx, tenant, did)
	return d, run, e
}
func (r *PostgreSQLRepository) CompleteDemoRun(context.Context, string) error { return nil }
func (r *PostgreSQLRepository) reconcileDeployment(ctx context.Context, depID string) error {
	var desired string
	var pending, failed int
	e := r.pool.QueryRow(ctx, `SELECT d.desired_status, count(*) FILTER (WHERE w.observed_generation<>w.desired_generation OR (w.desired_status='running' AND w.observed_status<>'running') OR (w.desired_status='stopped' AND w.observed_status<>'stopped')), count(*) FILTER (WHERE w.observed_status='failed') FROM project_deployments d JOIN workloads w ON w.project_deployment_id=d.id WHERE d.id=$1 GROUP BY d.desired_status`, depID).Scan(&desired, &pending, &failed)
	if e != nil {
		return e
	}
	observed := desired
	if failed > 0 {
		observed = "failed"
	} else if pending > 0 {
		observed = "pending"
	}
	_, e = r.pool.Exec(ctx, `UPDATE project_deployments SET observed_status=$1,updated_at=now() WHERE id=$2`, observed, depID)
	if e != nil {
		return e
	}
	if observed == "pending" {
		return nil
	}
	message := "agent observed desired state"
	stage := "observed"
	if observed == "failed" {
		message = "agent reported workload failure"
		stage = "failed"
	}
	var runID string
	e = r.pool.QueryRow(ctx, `WITH latest_pending AS (SELECT id FROM deployment_runs WHERE project_deployment_id=$1 AND observed_status='pending' ORDER BY started_at DESC LIMIT 1) UPDATE deployment_runs SET observed_status=$2,progress=100,message=$3,completed_at=now() WHERE id=(SELECT id FROM latest_pending) RETURNING id`, depID, observed, message).Scan(&runID)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	_, e = r.pool.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) SELECT $1,$2,$3 WHERE NOT EXISTS (SELECT 1 FROM deployment_run_events WHERE deployment_run_id=$1 AND stage=$2)`, runID, stage, message)
	return e
}

// runtimeApprovalAllowed 封装拓扑语义，调用方负责先锁定 runtime_clusters 行，
// 保证 single_node 的检查与审批写入处于同一个事务临界区。
func runtimeApprovalAllowed(topology string, activeCount int) error {
	if topology == "single_node" && activeCount > 0 {
		return fmt.Errorf("单节点运行集群只能审批一个运行节点")
	}
	return nil
}

// deploymentTopologyAllowed 明确当前调度器的能力边界。高可用集群允许提前
// 注册多个运行节点，但在主备选举、工作负载归属和故障转移尚未实现前，禁止部署，
// 避免同一计算或报警进程在每个节点重复执行。
func deploymentTopologyAllowed(topology string) error {
	if topology == "high_availability" {
		return fmt.Errorf("运行集群当前不能调度：高可用部署尚未开放")
	}
	return nil
}

// validateClaimPlatform 只接受与安装包匹配的平台，避免 Agent 误用其他角色的接入码。
func validateClaimPlatform(role, osName, architecture string) error {
	osName = strings.ToLower(strings.TrimSpace(osName))
	architecture = strings.ToLower(strings.TrimSpace(architecture))
	linuxArchitecture := architecture == "amd64" || architecture == "arm64"
	switch role {
	case RoleRuntimeLinux, RoleCollectorLinux:
		if osName == "linux" && linuxArchitecture {
			return nil
		}
		return fmt.Errorf("该接入码仅支持 Linux amd64 或 arm64 节点")
	case RoleCollectorWindows:
		if osName == "windows" && architecture == "amd64" {
			return nil
		}
		return fmt.Errorf("该接入码仅支持 Windows amd64 节点")
	default:
		return fmt.Errorf("节点角色不支持")
	}
}

func (r *PostgreSQLRepository) listWorkloads(ctx context.Context, did string) ([]Workload, error) {
	rows, e := r.pool.Query(ctx, `SELECT id,tenant_id,project_deployment_id,COALESCE(host_node_id::text,''),role,desired_status,observed_status,replicas_desired,replicas_observed,desired_generation,observed_generation,last_operation,COALESCE(last_message,''),observed_at,created_at,updated_at FROM workloads WHERE project_deployment_id=$1 ORDER BY role`, did)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	items := []Workload{}
	for rows.Next() {
		var x Workload
		if e = rows.Scan(workloadScanArgs(&x)...); e != nil {
			return nil, e
		}
		items = append(items, x)
	}
	return items, rows.Err()
}

type scanner interface{ Scan(...any) error }

func scanCluster(s scanner) (RuntimeCluster, error) {
	var x RuntimeCluster
	var meta []byte
	e := s.Scan(&x.ID, &x.TenantID, &x.Name, &x.Code, &x.Description, &x.Topology, &x.DesiredStatus, &x.ObservedStatus, &x.ControllerStatus, &meta, &x.CreatedAt, &x.UpdatedAt, &x.NodeCount, &x.OnlineNodeCount, &x.Health)
	_ = json.Unmarshal(meta, &x.Metadata)
	if x.Metadata == nil {
		x.Metadata = map[string]any{}
	}
	return x, e
}
func scanEnrollment(s scanner) (Enrollment, error) {
	var x Enrollment
	e := s.Scan(&x.ID, &x.TenantID, &x.RuntimeClusterID, &x.Role, &x.DisplayName, &x.Status, &x.ExpiresAt, &x.ClaimedAt, &x.ClaimedByNodeID, &x.ApprovedAt, &x.RejectedAt, &x.CreatedAt, &x.UpdatedAt)
	return x, e
}
func hostScanArgs(x *HostNode) []any {
	return []any{&x.ID, &x.TenantID, &x.RuntimeClusterID, &x.EnrollmentID, &x.Role, &x.DisplayName, &x.Hostname, &x.OS, &x.Architecture, &x.AgentVersion, &x.MachineFingerprint, &x.IPAddress, &x.DesiredStatus, &x.ObservedStatus, jsonTarget(&x.ResourceSummary), jsonTarget(&x.Capabilities), &x.LastHeartbeatAt, &x.ApprovedAt, &x.CreatedAt, &x.UpdatedAt}
}
func scanManagementHostNode(s scanner) (HostNode, error) {
	var x HostNode
	args := append(hostScanArgs(&x), &x.RuntimeClusterName)
	err := s.Scan(args...)
	return x, err
}
func workloadScanArgs(x *Workload) []any {
	return []any{&x.ID, &x.TenantID, &x.ProjectDeploymentID, &x.HostNodeID, &x.Role, &x.DesiredStatus, &x.ObservedStatus, &x.ReplicasDesired, &x.ReplicasObserved, &x.DesiredGeneration, &x.ObservedGeneration, &x.LastOperation, &x.LastMessage, &x.ObservedAt, &x.CreatedAt, &x.UpdatedAt}
}
func runScanArgs(x *DeploymentRun) []any {
	return []any{&x.ID, &x.TenantID, &x.ProjectDeploymentID, &x.Operation, &x.DesiredStatus, &x.ObservedStatus, &x.Progress, &x.Message, &x.StartedAt, &x.CompletedAt}
}

type jsonDestination struct{ target *map[string]any }

func (d jsonDestination) Scan(src any) error {
	raw, ok := src.([]byte)
	if !ok {
		raw = []byte("{}")
	}
	if err := json.Unmarshal(raw, d.target); err != nil {
		return err
	}
	if *d.target == nil {
		*d.target = map[string]any{}
	}
	return nil
}
func jsonTarget(t *map[string]any) jsonDestination { return jsonDestination{t} }
func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
func mapDeploymentCreateError(err error) error {
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) && pgError.Code == "23505" && pgError.ConstraintName == "project_deployments_tenant_project_cluster_key" {
		return ErrDeploymentExists
	}
	return err
}
func validUUID(value string) bool { _, e := uuid.Parse(value); return e == nil }
