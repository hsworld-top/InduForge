package ops

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// KubernetesProjectReconciler 只使用 Pod 注入的 ServiceAccount token。构造失败即
// 禁用调和，绝不退回宿主 kubeconfig、kubectl 或节点管理员凭据。
type KubernetesProjectReconciler struct {
	client          *http.Client
	endpoint, token string
}

type ProjectWorkloadApplier interface {
	Reconcile(context.Context, ProjectWorkload) error
}

// ReconcilePendingProjectWorkloads 是中心控制面周期调用的唯一写集群入口。任何 apply
// 失败都会落库为 failed 并写入 run event；只有 applier 成功返回才标记服务 running。
func (r *PostgreSQLRepository) ReconcilePendingProjectWorkloads(ctx context.Context, applier ProjectWorkloadApplier) (int, error) {
	rows, err := r.pool.Query(ctx, `SELECT s.id::text,s.project_deployment_id::text,d.environment_id::text,s.node_id::text,s.service_type,d.application_version_id::text,s.desired_generation,s.public_port FROM deployment_services s JOIN project_deployments d ON d.id=s.project_deployment_id WHERE s.desired_status='running' AND (s.observed_status<>'running' OR s.observed_generation<>s.desired_generation) ORDER BY s.updated_at LIMIT 100`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var serviceID, deploymentID, environmentID, nodeID, engine, releaseID string
		var generation int64
		var port *int
		if err = rows.Scan(&serviceID, &deploymentID, &environmentID, &nodeID, &engine, &releaseID, &generation, &port); err != nil {
			return count, err
		}
		workload := ProjectWorkload{EnvironmentID: environmentID, DeploymentID: deploymentID, ServiceID: serviceID, NodeID: nodeID, Engine: engine, ReleaseID: releaseID, Generation: generation, HostPort: port}
		if applyErr := applier.Reconcile(ctx, workload); applyErr != nil {
			message := applyErr.Error()
			if len(message) > 1024 {
				message = message[:1024]
			}
			_, _ = r.pool.Exec(ctx, `UPDATE deployment_services SET observed_status='failed',last_message=$1,observed_at=now(),updated_at=now() WHERE id=$2`, message, serviceID)
			_, _ = r.pool.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) SELECT id,'failed',$1 FROM deployment_runs WHERE project_deployment_id=$2 AND observed_status='pending'`, message, deploymentID)
			_ = r.reconcileDeployment(ctx, deploymentID)
			continue
		}
		if _, err = r.pool.Exec(ctx, `UPDATE deployment_services SET observed_status='running',observed_generation=desired_generation,last_message='Kubernetes rollout 已提交，等待 readiness',observed_at=now(),updated_at=now() WHERE id=$1`, serviceID); err != nil {
			return count, err
		}
		_, _ = r.pool.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) SELECT id,'dispatched','Kubernetes 工作负载已提交' FROM deployment_runs WHERE project_deployment_id=$1 AND observed_status='pending'`, deploymentID)
		if err = r.reconcileDeployment(ctx, deploymentID); err != nil {
			return count, err
		}
		count++
	}
	return count, rows.Err()
}

func NewInClusterProjectReconciler() (*KubernetesProjectReconciler, error) {
	host, port := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT_HTTPS")
	if host == "" || port == "" {
		return nil, fmt.Errorf("未运行在 Kubernetes 集群内")
	}
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil || len(bytes.TrimSpace(token)) == 0 {
		return nil, fmt.Errorf("读取中心 ServiceAccount token 失败: %w", err)
	}
	ca, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return nil, fmt.Errorf("读取集群 CA 失败: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return nil, fmt.Errorf("集群 CA 无效")
	}
	return &KubernetesProjectReconciler{client: &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}}, endpoint: "https://" + host + ":" + port, token: string(bytes.TrimSpace(token))}, nil
}

// Reconcile 使用稳定名称的 ConfigMap/Deployment server-side apply；同名模板更新由
// Kubernetes RollingUpdate 接管。403 明确暴露为 RBAC 配置错误，不能伪报已运行。
func (r *KubernetesProjectReconciler) Reconcile(ctx context.Context, workload ProjectWorkload) error {
	manifest, err := RenderProjectWorkloadManifest(workload)
	if err != nil {
		return err
	}
	namespace, _ := projectNamespace(workload.EnvironmentID)
	for _, document := range strings.Split(strings.TrimSpace(manifest), "\n---\n") {
		var meta struct {
			Kind     string `json:"kind"`
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
		}
		if err := json.Unmarshal(yamlToJSON(document), &meta); err != nil {
			return fmt.Errorf("项目清单 JSON 转换失败: %w", err)
		}
		resource := "configmaps"
		if meta.Kind == "Deployment" {
			resource = "deployments"
		}
		path := "/apis/apps/v1/namespaces/" + namespace + "/" + resource + "/" + meta.Metadata.Name
		if meta.Kind == "ConfigMap" {
			path = "/api/v1/namespaces/" + namespace + "/" + resource + "/" + meta.Metadata.Name
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodPatch, r.endpoint+path+"?fieldManager=induforge-center&force=true", strings.NewReader(document))
		if err != nil {
			return err
		}
		request.Header.Set("Authorization", "Bearer "+r.token)
		request.Header.Set("Content-Type", "application/apply-patch+yaml")
		response, err := r.client.Do(request)
		if err != nil {
			return err
		}
		response.Body.Close()
		if response.StatusCode/100 != 2 {
			return fmt.Errorf("Kubernetes 调和 %s/%s 失败: HTTP %d", meta.Kind, meta.Metadata.Name, response.StatusCode)
		}
	}
	return nil
}

// yamlToJSON 仅用于结构化提取 kind/name；清单本体仍原样以 apply-patch+yaml 提交。
// 当前固定生成器只含简单标量，转换避免引入通用 YAML 执行入口。
func yamlToJSON(document string) []byte {
	lines := strings.Split(document, "\n")
	kind, name := "", ""
	for index, line := range lines {
		if strings.HasPrefix(line, "kind: ") {
			kind = strings.TrimSpace(strings.TrimPrefix(line, "kind: "))
		}
		if strings.TrimSpace(line) == "metadata:" {
			for _, next := range lines[index+1:] {
				if strings.HasPrefix(next, "  name: ") {
					name = strings.TrimSpace(strings.TrimPrefix(next, "  name: "))
					break
				}
				if next != "" && !strings.HasPrefix(next, "  ") {
					break
				}
			}
		}
	}
	return []byte(fmt.Sprintf(`{"kind":%q,"metadata":{"name":%q}}`, kind, name))
}

func projectArtifactHostPath(deploymentID string) string {
	return filepath.Join("/var/lib/induforge/node-agent/deployments", deploymentID, "release")
}
