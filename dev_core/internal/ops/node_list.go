package ops

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type nodeClusterReader interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

const nodePageClustersSQL = `SELECT cn.node_id::text,cn.node_kind,cn.cluster_id::text,CASE WHEN cn.node_kind='center' THEN 'server' ELSE 'agent' END,cn.cluster_status,COALESCE(cn.cluster_message,''),cn.desired_action,cn.desired_generation,cn.observed_generation,cn.cluster_observed_at,COALESCE((SELECT array_agg(e.name ORDER BY en.created_at DESC,e.id DESC) FROM runtime_environment_nodes en JOIN runtime_environments e ON e.id=en.environment_id AND e.tenant_id=n.tenant_id AND e.deleted_at IS NULL WHERE en.node_id=cn.node_id),ARRAY[]::text[]) FROM runtime_cluster_nodes cn JOIN host_nodes n ON n.id=cn.node_id AND n.tenant_id=$1 WHERE cn.node_id=ANY($2::uuid[])`

func (r *PostgreSQLRepository) attachNodeClusters(ctx context.Context, tenant string, nodes []Node) error {
	return attachNodePageClusters(ctx, r.pool, tenant, nodes)
}

// 仅补充本页节点的集群信息，一次参数化批量查询替代逐行查询。
// 调用者必须先关闭分页rows，以免小连接池在嵌套查询时互相等待。
func attachNodePageClusters(ctx context.Context, reader nodeClusterReader, tenant string, nodes []Node) error {
	if len(nodes) == 0 {
		return nil
	}
	ids := make([]string, 0, len(nodes))
	byID := make(map[string]int, len(nodes))
	for i := range nodes {
		ids = append(ids, nodes[i].ID)
		byID[nodes[i].ID] = i
		nodes[i].EnvironmentNames = []string{}
		nodes[i].EnvironmentCount = 0
	}
	rows, err := reader.Query(ctx, nodePageClustersSQL, tenant, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cluster Node
		if err := rows.Scan(&cluster.ID, &cluster.NodeKind, &cluster.ClusterID, &cluster.ClusterRole, &cluster.ClusterStatus, &cluster.ClusterMessage, &cluster.ClusterDesiredAction, &cluster.ClusterDesiredGeneration, &cluster.ClusterObservedGeneration, &cluster.ClusterObservedAt, &cluster.EnvironmentNames); err != nil {
			return err
		}
		if index, ok := byID[cluster.ID]; ok {
			node := &nodes[index]
			node.NodeKind, node.ClusterID, node.ClusterRole = cluster.NodeKind, cluster.ClusterID, cluster.ClusterRole
			node.ClusterStatus, node.ClusterMessage, node.ClusterDesiredAction = cluster.ClusterStatus, cluster.ClusterMessage, cluster.ClusterDesiredAction
			node.ClusterDesiredGeneration, node.ClusterObservedGeneration, node.ClusterObservedAt = cluster.ClusterDesiredGeneration, cluster.ClusterObservedGeneration, cluster.ClusterObservedAt
			node.EnvironmentNames = cluster.EnvironmentNames
			node.EnvironmentCount = len(cluster.EnvironmentNames)
		}
	}
	return rows.Err()
}
