package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/indu-forge/dev_core/internal/auth"
	"github.com/jackc/pgx/v5"
)

type EnvironmentOverview struct {
	EnvironmentID string                   `json:"environmentId"`
	SnapshotAt    time.Time                `json:"snapshotAt"`
	Thresholds    OverviewThresholds       `json:"thresholds"`
	Deployments   OverviewDeploymentCounts `json:"deployments"`
	Nodes         OverviewNodeCounts       `json:"nodes"`
	RiskNodes     []OverviewRiskNode       `json:"riskNodes"`
}
type OverviewThresholds struct {
	FreshnessSeconds         int `json:"freshnessSeconds"`
	CapacityAttentionPercent int `json:"capacityAttentionPercent"`
	CapacityCriticalPercent  int `json:"capacityCriticalPercent"`
}
type OverviewDeploymentCounts struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Stopped int `json:"stopped"`
	Failed  int `json:"failed"`
	Pending int `json:"pending"`
	Stale   int `json:"stale"`
	Unknown int `json:"unknown"`
}
type OverviewNodeCounts struct {
	Total               int      `json:"total"`
	Online              int      `json:"online"`
	Offline             int      `json:"offline"`
	Fault               int      `json:"fault"`
	StaleMetrics        int      `json:"staleMetrics"`
	UnknownMetrics      int      `json:"unknownMetrics"`
	CapacityAttention   int      `json:"capacityAttention"`
	CapacityCritical    int      `json:"capacityCritical"`
	MaxDiskUsagePercent *float64 `json:"maxDiskUsagePercent"`
}
type OverviewRiskNode struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	IPAddress       string     `json:"ipAddress"`
	NodeKind        string     `json:"nodeKind"`
	ObservedStatus  string     `json:"observedStatus"`
	ClusterStatus   string     `json:"clusterStatus"`
	LastHeartbeatAt *time.Time `json:"lastHeartbeatAt"`
	MetricsStale    bool       `json:"metricsStale"`
	CPUPercent      *float64   `json:"cpuPercent"`
	MemoryPercent   *float64   `json:"memoryPercent"`
	DiskPercent     *float64   `json:"diskPercent"`
	Health          string     `json:"health"`
}

// 一条SQL提供同一MVCC/数据库时间快照；先限定环境与租户，再聚合全量部署和节点。
// 容量阈值仅影响概览展示，绝不改变节点/环境运行状态或生成告警。
const environmentOverviewSQL = `WITH env AS (
 SELECT id,tenant_id FROM runtime_environments WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL
), raw_nodes AS (
 SELECT n.id,COALESCE(NULLIF(n.display_name,''),NULLIF(n.hostname,''),n.id::text) AS display_name,n.ip_address,n.desired_status,n.observed_status,n.last_heartbeat_at,
 COALESCE(cn.node_kind,'') AS node_kind,COALESCE(cn.cluster_status,'') AS cluster_status,
 (n.platform='linux' AND n.capabilities @> '["project_entry","data_runtime"]'::jsonb) AS requires_cluster,
 COALESCE(cn.desired_action='removing' OR cn.observed_generation<>cn.desired_generation,false) AS cluster_pending,
 COALESCE(n.last_heartbeat_at>now()-interval '45 seconds',false) AS fresh,
 CASE WHEN jsonb_typeof(n.resource_summary->'cpu'->'usedPercent')='number' THEN (n.resource_summary->'cpu'->>'usedPercent')::numeric END AS raw_cpu,
 CASE WHEN jsonb_typeof(n.resource_summary->'memory'->'usedPercent')='number' THEN (n.resource_summary->'memory'->>'usedPercent')::numeric END AS raw_memory,
 CASE WHEN jsonb_typeof(n.resource_summary->'disk'->'usedPercent')='number' THEN (n.resource_summary->'disk'->>'usedPercent')::numeric END AS raw_disk
 FROM env JOIN runtime_environment_nodes en ON en.environment_id=env.id AND en.tenant_id=env.tenant_id
 JOIN host_nodes n ON n.id=en.node_id AND n.tenant_id=env.tenant_id
 LEFT JOIN runtime_cluster_nodes cn ON cn.node_id=n.id AND cn.tenant_id=env.tenant_id
), node_metrics AS (
 SELECT *,NOT fresh AS metrics_stale,
 (raw_cpu IS NULL OR raw_cpu NOT BETWEEN 0 AND 100 OR raw_memory IS NULL OR raw_memory NOT BETWEEN 0 AND 100 OR raw_disk IS NULL OR raw_disk NOT BETWEEN 0 AND 100) AS metrics_unknown,
 CASE WHEN fresh AND desired_status='active' AND observed_status='online' AND raw_cpu BETWEEN 0 AND 100 THEN raw_cpu END AS cpu,
 CASE WHEN fresh AND desired_status='active' AND observed_status='online' AND raw_memory BETWEEN 0 AND 100 THEN raw_memory END AS memory,
 CASE WHEN fresh AND desired_status='active' AND observed_status='online' AND raw_disk BETWEEN 0 AND 100 THEN raw_disk END AS disk
 FROM raw_nodes
), ranked_nodes AS (
 SELECT *,CASE WHEN desired_status<>'active' OR observed_status<>'online' THEN 'offline'
 WHEN cluster_status='failed' THEN 'failed' WHEN metrics_stale THEN 'stale'
 WHEN requires_cluster AND (cluster_pending OR cluster_status IN ('pending','starting','removing')) THEN 'pending'
 WHEN requires_cluster AND cluster_status<>'ready' THEN 'unknown'
 WHEN disk>=95 THEN 'capacity_critical' WHEN disk>=85 THEN 'capacity_attention'
 WHEN metrics_unknown THEN 'unknown' ELSE 'healthy' END AS health,
 CASE WHEN desired_status<>'active' OR observed_status<>'online' THEN 0
 WHEN cluster_status='failed' THEN 1 WHEN metrics_stale THEN 2
 WHEN requires_cluster AND (cluster_pending OR cluster_status IN ('pending','starting','removing')) THEN 3
 WHEN requires_cluster AND cluster_status<>'ready' THEN 4
 WHEN disk>=95 THEN 5 WHEN disk>=85 THEN 6 WHEN metrics_unknown THEN 7 ELSE 8 END AS priority
 FROM node_metrics
), deployment_observation AS (
 SELECT d.id,d.desired_status,d.observed_status,count(s.id) AS service_count,
 bool_or(s.desired_status='running') AS has_running,
 bool_or(s.observed_status='failed') AS service_failed,
 bool_or(s.id IS NOT NULL AND (s.desired_generation IS NULL OR s.observed_generation IS NULL)) AS generation_unknown,
 bool_and(COALESCE(s.observed_status=s.desired_status AND s.observed_generation=s.desired_generation,false)) AS converged,
 bool_or(s.desired_status='running' AND (n.id IS NULL OR n.last_heartbeat_at IS NULL)) AS node_unknown,
 bool_and(CASE WHEN s.desired_status='running' THEN COALESCE(n.desired_status='active' AND n.observed_status='online' AND n.last_heartbeat_at>now()-interval '45 seconds',false) ELSE true END) AS nodes_fresh
 FROM env JOIN project_deployments d ON d.environment_id=env.id AND d.tenant_id=env.tenant_id AND d.deleted_at IS NULL
 LEFT JOIN deployment_services s ON s.project_deployment_id=d.id AND s.tenant_id=d.tenant_id
 LEFT JOIN host_nodes n ON n.id=s.node_id AND n.tenant_id=d.tenant_id
 GROUP BY d.id,d.desired_status,d.observed_status
), deployment_states AS (
 SELECT CASE WHEN observed_status='failed' OR service_failed THEN 'failed'
 WHEN service_count=0 OR generation_unknown OR (desired_status='running' AND NOT has_running) THEN 'unknown'
 WHEN observed_status='pending' OR NOT converged THEN 'pending'
 WHEN desired_status='stopped' AND observed_status='stopped' THEN 'stopped'
 WHEN desired_status='running' AND observed_status='running' AND node_unknown THEN 'unknown'
 WHEN desired_status='running' AND observed_status='running' AND NOT nodes_fresh THEN 'stale'
 WHEN desired_status='running' AND observed_status='running' AND nodes_fresh THEN 'running'
 ELSE 'unknown' END AS state FROM deployment_observation
)
SELECT jsonb_build_object('environmentId',env.id,'snapshotAt',now(),
 'thresholds',jsonb_build_object('freshnessSeconds',45,'capacityAttentionPercent',85,'capacityCriticalPercent',95),
 'deployments',(SELECT jsonb_build_object('total',count(*),'running',count(*) FILTER(WHERE state='running'),
 'stopped',count(*) FILTER(WHERE state='stopped'),'failed',count(*) FILTER(WHERE state='failed'),
 'pending',count(*) FILTER(WHERE state='pending'),'stale',count(*) FILTER(WHERE state='stale'),'unknown',count(*) FILTER(WHERE state='unknown')) FROM deployment_states),
 'nodes',(SELECT jsonb_build_object('total',count(*),'online',count(*) FILTER(WHERE desired_status='active' AND observed_status='online' AND fresh),
 'offline',count(*) FILTER(WHERE desired_status<>'active' OR observed_status<>'online' OR NOT fresh),
 'fault',count(*) FILTER(WHERE cluster_status='failed'),'staleMetrics',count(*) FILTER(WHERE metrics_stale),
 'unknownMetrics',count(*) FILTER(WHERE metrics_unknown),'capacityAttention',count(*) FILTER(WHERE disk>=85 AND disk<95),
 'capacityCritical',count(*) FILTER(WHERE disk>=95),'maxDiskUsagePercent',max(disk)) FROM ranked_nodes),
 'riskNodes',COALESCE((SELECT jsonb_agg(jsonb_build_object('id',id,'name',display_name,'ipAddress',COALESCE(ip_address,''),
 'nodeKind',node_kind,'observedStatus',observed_status,'clusterStatus',cluster_status,'lastHeartbeatAt',last_heartbeat_at,
 'metricsStale',metrics_stale,'cpuPercent',cpu,'memoryPercent',memory,'diskPercent',disk,'health',health) ORDER BY priority,disk DESC NULLS LAST,id)
 FROM (SELECT * FROM ranked_nodes ORDER BY priority,disk DESC NULLS LAST,id LIMIT 5) top_nodes),'[]'::jsonb)) FROM env`

func (r *PostgreSQLRepository) GetEnvironmentOverview(ctx context.Context, tenant, id string) (EnvironmentOverview, error) {
	return loadEnvironmentOverview(ctx, r.pool, tenant, id)
}

type overviewReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadEnvironmentOverview(ctx context.Context, reader overviewReader, tenant, id string) (EnvironmentOverview, error) {
	var raw []byte
	if err := reader.QueryRow(ctx, environmentOverviewSQL, tenant, id).Scan(&raw); err != nil {
		return EnvironmentOverview{}, mapNotFound(err)
	}
	var out EnvironmentOverview
	err := json.Unmarshal(raw, &out)
	return out, err
}
func (s *Service) GetEnvironmentOverview(ctx context.Context, actor auth.User, id string) (EnvironmentOverview, error) {
	if err := auth.RequireCapability(actor, auth.CapabilityEnvironmentRead); err != nil {
		return EnvironmentOverview{}, err
	}
	if !validUUID(id) {
		return EnvironmentOverview{}, fmt.Errorf("运行环境 ID 无效")
	}
	return s.repository.GetEnvironmentOverview(ctx, actor.TenantID, id)
}
func (h *Handler) getEnvironmentOverview(w http.ResponseWriter, r *http.Request) {
	h.user(w, r, func(actor auth.User) {
		out, err := h.service.GetEnvironmentOverview(r.Context(), actor, chi.URLParam(r, "id"))
		if err != nil {
			h.err(w, r, err)
			return
		}
		h.writeSuccess(w, r, out)
	})
}
