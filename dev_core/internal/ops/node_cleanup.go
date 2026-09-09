package ops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// 清理请求持久化在节点状态中。只有集群清理成功才解除关联，失败由下一轮重试。
// 工程部署可跨节点：仅移除当前节点的服务，其他节点的部署与历史保留。
func (r *PostgreSQLRepository) ReconcileNodeCleanup(ctx context.Context, k *KubernetesProjectReconciler) error {
	rows, err := r.pool.Query(ctx, `SELECT id::text,tenant_id::text FROM host_nodes WHERE resource_summary->>'cleanupRequested'='true' ORDER BY updated_at LIMIT 20`)
	if err != nil {
		return err
	}
	type target struct{ id, tenant string }
	var targets []target
	for rows.Next() {
		var t target
		if err = rows.Scan(&t.id, &t.tenant); err != nil {
			rows.Close()
			return err
		}
		targets = append(targets, t)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	var failures error
	for _, t := range targets {
		if err = r.cleanupNode(ctx, k, t.id, t.tenant); err != nil {
			_, _ = r.pool.Exec(ctx, `UPDATE host_nodes SET resource_summary=resource_summary||jsonb_build_object('cleanupError',$2::text) WHERE id=$1`, t.id, "节点关联清理未完成，正在重试")
			failures = errors.Join(failures, err)
		}
	}
	return failures
}
func (r *PostgreSQLRepository) cleanupNode(ctx context.Context, k *KubernetesProjectReconciler, id, tenant string) error {
	rows, err := r.pool.Query(ctx, `SELECT environment_id::text FROM runtime_environment_nodes WHERE node_id=$1 UNION SELECT d.environment_id::text FROM deployment_services s JOIN project_deployments d ON d.id=s.project_deployment_id WHERE s.node_id=$1 UNION SELECT environment_id::text FROM runtime_environment_services WHERE node_id=$1`, id)
	if err != nil {
		return err
	}
	var environments []string
	for rows.Next() {
		var e string
		if err = rows.Scan(&e); err != nil {
			rows.Close()
			return err
		}
		environments = append(environments, e)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, e := range environments {
		ns, err := projectNamespace(e)
		if err != nil {
			return err
		}
		if err = k.cleanupNodeWorkloads(ctx, ns, id); err != nil {
			return err
		}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var remove bool
	if err = tx.QueryRow(ctx, `SELECT COALESCE(resource_summary->>'removeAfterCleanup','false')='true' FROM host_nodes WHERE id=$1 AND resource_summary->>'cleanupRequested'='true' FOR UPDATE`, id).Scan(&remove); err != nil {
		return err
	}
	// 先结束受影响任务，再移除服务。跨节点部署保留为异常，允许用户重新选择部署节点。
	for _, q := range []string{
		`UPDATE deployment_runs SET observed_status='failed',message='部署节点已卸载或移除，请重新配置部署节点',completed_at=now() WHERE observed_status='pending' AND project_deployment_id IN(SELECT project_deployment_id FROM deployment_services WHERE node_id=$1)`,
		`UPDATE project_deployments SET observed_status='failed',updated_at=now(),deleted_at=CASE WHEN NOT EXISTS(SELECT 1 FROM deployment_services s WHERE s.project_deployment_id=project_deployments.id AND s.node_id<>$1) THEN now() ELSE deleted_at END WHERE id IN(SELECT project_deployment_id FROM deployment_services WHERE node_id=$1)`,
		`DELETE FROM deployment_services WHERE node_id=$1 AND project_deployment_id IN(SELECT id FROM project_deployments WHERE deleted_at IS NOT NULL)`,
		`UPDATE deployment_services SET desired_status='stopped',observed_status='failed',last_message='节点已卸载或移除，请重新选择部署节点',updated_at=now() WHERE node_id=$1`,
		`DELETE FROM runtime_environment_services WHERE node_id=$1`,
		`DELETE FROM runtime_environment_nodes WHERE node_id=$1`,
		`UPDATE host_nodes SET resource_summary=(resource_summary-'cleanupRequested'-'cleanupError')||jsonb_build_object('cleanupCompletedAt',now()),updated_at=now() WHERE id=$1`,
	} {
		if _, err = tx.Exec(ctx, q, id); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO runtime_cluster_events(tenant_id,cluster_id,node_id,event_type,name,target,result,message) SELECT tenant_id,cluster_id,node_id,'worker_removed','节点关联已清理','节点','success','节点卸载或移除后已清理对应运行关联，其他节点不受影响' FROM runtime_cluster_nodes WHERE node_id=$1`, id); err != nil {
		return err
	}
	if remove {
		if _, err = tx.Exec(ctx, `DELETE FROM runtime_cluster_nodes WHERE node_id=$1`, id); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE host_nodes SET desired_status='revoked',observed_status='revoked',agent_token_hash='revoked:'||id::text WHERE id=$1`, id); err != nil {
			return err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	r.publish(tenant, []string{"nodes", "environments", "deployments", "events"}, nil, true)
	return nil
}

func (r *KubernetesProjectReconciler) cleanupNodeWorkloads(ctx context.Context, namespace, node string) error {
	for _, kind := range []string{"deployments", "statefulsets", "jobs"} {
		prefix := "/apis/apps/v1"
		if kind == "jobs" {
			prefix = "/apis/batch/v1"
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint+prefix+"/namespaces/"+namespace+"/"+kind, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+r.token)
		resp, err := r.client.Do(req)
		if err != nil {
			return err
		}
		var list struct {
			Items []struct {
				Metadata struct{ Name string }
				Spec     struct {
					Template struct {
						Spec struct {
							NodeSelector map[string]string `json:"nodeSelector"`
						}
					}
				}
			}
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			return fmt.Errorf("读取节点清理资源失败: HTTP %d", resp.StatusCode)
		}
		err = json.NewDecoder(resp.Body).Decode(&list)
		resp.Body.Close()
		if err != nil {
			return err
		}
		for _, item := range list.Items {
			if item.Spec.Template.Spec.NodeSelector["induforge.io/host-node-id"] != node {
				continue
			}
			name := item.Metadata.Name
			paths := []string{prefix + "/namespaces/" + namespace + "/" + kind + "/" + name}
			if kind != "jobs" {
				paths = append(paths, "/api/v1/namespaces/"+namespace+"/services/"+name)
			}
			if kind == "statefulsets" {
				paths = append(paths, "/api/v1/namespaces/"+namespace+"/persistentvolumeclaims/data-"+name+"-0")
			}
			for _, path := range paths {
				if err = r.deleteNodeResource(ctx, path); err != nil {
					return err
				}
			}
		}
	}
	return r.cleanupRetiredNodePods(ctx, namespace, node)
}

// 节点卸载后 kubelet 已退出，后台删除控制器不会完成 Pod 终止。
// 按旧节点身份匹配，并使用 UID 前置条件，避免误删同名的新实例。
func (r *KubernetesProjectReconciler) cleanupRetiredNodePods(ctx context.Context, namespace, node string) error {
	path := "/api/v1/namespaces/" + namespace + "/pods"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("读取节点残留实例失败: HTTP %d", resp.StatusCode)
	}
	var list struct {
		Items []struct {
			Metadata struct {
				Name string
				UID  string
			}
			Spec struct{ NodeSelector map[string]string }
		}
	}
	if err = json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return err
	}
	for _, pod := range list.Items {
		if pod.Spec.NodeSelector["induforge.io/host-node-id"] != node || pod.Metadata.UID == "" {
			continue
		}
		body, _ := json.Marshal(map[string]any{"gracePeriodSeconds": 0, "preconditions": map[string]string{"uid": pod.Metadata.UID}})
		req, err = http.NewRequestWithContext(ctx, http.MethodDelete, r.endpoint+path+"/"+pod.Metadata.Name, strings.NewReader(string(body)))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+r.token)
		req.Header.Set("Content-Type", "application/json")
		result, err := r.client.Do(req)
		if err != nil {
			return err
		}
		result.Body.Close()
		if result.StatusCode/100 != 2 && result.StatusCode != 404 {
			return fmt.Errorf("清理节点残留实例失败: HTTP %d", result.StatusCode)
		}
	}
	return nil
}

func (r *KubernetesProjectReconciler) deleteNodeResource(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, r.endpoint+path, strings.NewReader(`{"propagationPolicy":"Background"}`))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 && resp.StatusCode != 404 {
		return fmt.Errorf("清理节点资源失败: HTTP %d", resp.StatusCode)
	}
	return nil
}
