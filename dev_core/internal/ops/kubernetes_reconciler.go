package ops

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/indu-forge/dev_core/internal/dataservice"
	"github.com/nats-io/nats.go"
)

// KubernetesProjectReconciler 只使用 Pod 注入的 ServiceAccount token。构造失败即
// 禁用调和，绝不退回宿主 kubeconfig、kubectl 或节点管理员凭据。
type KubernetesProjectReconciler struct {
	client          *http.Client
	endpoint, token string
	secretManager   *DeploymentSecretManager
	runtimeContext  interface {
		LoadProjectRuntimeContext(context.Context, string) (ProjectRuntimeContext, error)
	}
	collectorBundle interface {
		BuildCollectorBindingBundle(context.Context, dataservice.CollectorBindingBundleRequest) (*dataservice.CollectorBindingBundle, error)
	}
	hostNodes interface {
		LoadHostNodeAddresses(context.Context, []string) (map[string]string, error)
	}
}

const hostNodeIDLabel = "induforge.io/host-node-id"

type KubernetesNode struct {
	Name       string
	InternalIP string
	Ready      bool
	Labels     map[string]string
}

func (r *KubernetesProjectReconciler) SetDeploymentSecretManager(manager *DeploymentSecretManager) {
	r.secretManager = manager
}

func (r *KubernetesProjectReconciler) SetRuntimeContextLoader(loader interface {
	LoadProjectRuntimeContext(context.Context, string) (ProjectRuntimeContext, error)
}) {
	r.runtimeContext = loader
}

func (r *KubernetesProjectReconciler) SetCollectorBindingBundleClient(client interface {
	BuildCollectorBindingBundle(context.Context, dataservice.CollectorBindingBundleRequest) (*dataservice.CollectorBindingBundle, error)
}) {
	r.collectorBundle = client
}

func (r *KubernetesProjectReconciler) SetHostNodeAddressLoader(loader interface {
	LoadHostNodeAddresses(context.Context, []string) (map[string]string, error)
}) {
	r.hostNodes = loader
}

// ListNodes 只读取调度节点的名称、InternalIP、Ready 和标签，禁止使用 hostname 推断身份。
func (r *KubernetesProjectReconciler) ListNodes(ctx context.Context) ([]KubernetesNode, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint+"/api/v1/nodes", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("读取 Kubernetes 节点失败: HTTP %d", resp.StatusCode)
	}
	var body struct {
		Items []struct {
			Metadata struct {
				Name   string            `json:"name"`
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
			Status struct {
				Addresses []struct {
					Type    string `json:"type"`
					Address string `json:"address"`
				} `json:"addresses"`
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	result := make([]KubernetesNode, 0, len(body.Items))
	for _, item := range body.Items {
		node := KubernetesNode{Name: item.Metadata.Name, Labels: item.Metadata.Labels}
		for _, address := range item.Status.Addresses {
			if address.Type == "InternalIP" {
				node.InternalIP = address.Address
				break
			}
		}
		for _, condition := range item.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				node.Ready = true
			}
		}
		result = append(result, node)
	}
	return result, nil
}

// PatchNodeLabel 仅写入平台节点身份标签，调用方须先完成唯一性和冲突校验。
func (r *KubernetesProjectReconciler) PatchNodeLabel(ctx context.Context, name, value string) error {
	body := fmt.Sprintf(`{"metadata":{"labels":{%q:%q}}}`, hostNodeIDLabel, value)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, r.endpoint+"/api/v1/nodes/"+name, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/merge-patch+json")
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("写入 Kubernetes 节点标签失败: HTTP %d", resp.StatusCode)
	}
	return nil
}

func (r *KubernetesProjectReconciler) EnsureHostNodeLabels(ctx context.Context, hostNodeIDs []string) error {
	if r.hostNodes == nil {
		return fmt.Errorf("Kubernetes 节点地址加载器未配置")
	}
	unique := make([]string, 0, len(hostNodeIDs))
	seen := make(map[string]struct{}, len(hostNodeIDs))
	for _, id := range hostNodeIDs {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			unique = append(unique, id)
		}
	}
	addresses, err := r.hostNodes.LoadHostNodeAddresses(ctx, unique)
	if err != nil {
		return err
	}
	nodes, err := r.ListNodes(ctx)
	if err != nil {
		return err
	}
	matchedHosts := make(map[string]string, len(unique))
	for _, hostNodeID := range unique {
		address := strings.TrimSpace(addresses[hostNodeID])
		if address == "" {
			return fmt.Errorf("物理节点 %s 缺少管理 IP", hostNodeID)
		}
		matches := make([]KubernetesNode, 0, 1)
		for _, node := range nodes {
			if node.InternalIP == address {
				matches = append(matches, node)
			}
		}
		if len(matches) != 1 {
			return fmt.Errorf("物理节点 %s 的管理 IP %s 匹配到 %d 个 Kubernetes 节点", hostNodeID, address, len(matches))
		}
		node := matches[0]
		if !node.Ready {
			return fmt.Errorf("Kubernetes 节点 %s 未就绪", node.Name)
		}
		if owner := matchedHosts[node.Name]; owner != "" && owner != hostNodeID {
			return fmt.Errorf("Kubernetes 节点 %s 同时匹配物理节点 %s 和 %s", node.Name, owner, hostNodeID)
		}
		matchedHosts[node.Name] = hostNodeID
		existing := node.Labels[hostNodeIDLabel]
		if existing != "" && existing != hostNodeID {
			return fmt.Errorf("Kubernetes 节点 %s 已被物理节点 %s 占用", node.Name, existing)
		}
		if existing == hostNodeID {
			continue
		}
		if err := r.PatchNodeLabel(ctx, node.Name, hostNodeID); err != nil {
			return err
		}
	}
	return nil
}

type ProjectWorkloadApplier interface {
	Reconcile(context.Context, ProjectWorkload) error
}
type ProjectWorkloadStatus struct {
	Ready             bool
	Failed            bool
	ReplicasObserved  int
	Message           string
}
type ProjectWorkloadInspector interface {
	Status(context.Context, ProjectWorkload) (ProjectWorkloadStatus, error)
}

// ProjectDeploymentStopper 删除某个工程部署的 K3s 运行资源。停止语义只影响
// 编排资源；部署配置、端口租约和运行消息仍由中心数据库保留。
type ProjectDeploymentStopper interface {
	StopDeployment(context.Context, string, string) error
}

type ProjectDeploymentDeleter interface {
	DeleteDeployment(context.Context, string, string) error
}

// ReconcilePendingProjectWorkloads 是中心控制面周期调用的唯一写集群入口。任何 apply
// 失败都会落库为 failed 并写入 run event；只有 applier 成功返回才标记服务 running。
func (r *PostgreSQLRepository) ReconcilePendingProjectWorkloads(ctx context.Context, applier ProjectWorkloadApplier) (int, error) {
	rows, err := r.pool.Query(ctx, `SELECT s.id::text,s.project_deployment_id::text,d.environment_id::text,s.node_id::text,s.service_type,COALESCE(d.application_version_id::text,d.artifact_descriptor->>'releaseId'),COALESCE(v.artifact_hash,d.artifact_descriptor->>'artifactHash',''),s.desired_generation,s.public_port FROM deployment_services s JOIN project_deployments d ON d.id=s.project_deployment_id LEFT JOIN application_versions v ON v.id=d.application_version_id AND v.status='ready' WHERE s.desired_status='running' AND (s.observed_status<>'running' OR s.observed_generation<>s.desired_generation) ORDER BY s.updated_at LIMIT 100`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var serviceID, deploymentID, environmentID, nodeID, engine, releaseID, releaseDigest string
		var generation int64
		var port *int
		if err = rows.Scan(&serviceID, &deploymentID, &environmentID, &nodeID, &engine, &releaseID, &releaseDigest, &generation, &port); err != nil {
			return count, err
		}
		releaseDigest = projectReleaseDigest(releaseDigest)
		workload := ProjectWorkload{EnvironmentID: environmentID, DeploymentID: deploymentID, ServiceID: serviceID, NodeID: nodeID, Engine: engine, ReleaseID: releaseID, ReleaseDigest: releaseDigest, Generation: generation, HostPort: port}
		if applyErr := applier.Reconcile(ctx, workload); applyErr != nil {
			message := applyErr.Error()
			if len(message) > 1024 {
				message = message[:1024]
			}
			_, _ = r.pool.Exec(ctx, `UPDATE deployment_services SET observed_status='failed',last_message=$1,observed_at=now(),updated_at=now() WHERE id=$2`, message, serviceID)
			_, _ = r.pool.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) SELECT id,$1,$2 FROM deployment_runs WHERE project_deployment_id=$3 AND observed_status='pending'`, workloadFailureStage(message), message, deploymentID)
			_ = r.reconcileDeployment(ctx, deploymentID)
			continue
		}
		status := ProjectWorkloadStatus{Message: "Kubernetes rollout 已提交，等待 readiness"}
		if inspector, ok := applier.(ProjectWorkloadInspector); ok {
			if status, err = inspector.Status(ctx, workload); err != nil {
				return count, err
			}
		}
		if status.Failed {
			_, _ = r.pool.Exec(ctx, `UPDATE deployment_services SET observed_status='failed',last_message=$1,observed_at=now(),updated_at=now() WHERE id=$2`, status.Message, serviceID)
			_, _ = r.pool.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) SELECT id,$1,$2 FROM deployment_runs WHERE project_deployment_id=$3 AND observed_status='pending'`, workloadFailureStage(status.Message), status.Message, deploymentID)
			_ = r.reconcileDeployment(ctx, deploymentID)
			continue
		}
		if status.Ready {
			_, err = r.pool.Exec(ctx, `UPDATE deployment_services SET observed_status='running',replicas_observed=$1,observed_generation=desired_generation,last_message=$2,observed_at=now(),updated_at=now() WHERE id=$3`, status.ReplicasObserved, status.Message, serviceID)
		} else {
			_, err = r.pool.Exec(ctx, `UPDATE deployment_services SET observed_status='pending',last_message=$1,observed_at=now(),updated_at=now() WHERE id=$2`, status.Message, serviceID)
		}
		if err != nil {
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

// ReconcileStoppedProjectDeployments 与运行态调和器成对工作。停止不能只更新
// desired_status，否则 K3s 工作负载会继续运行且 deployment run 永远 pending。
func (r *PostgreSQLRepository) ReconcileStoppedProjectDeployments(ctx context.Context, stopper ProjectDeploymentStopper) (int, error) {
	rows, err := r.pool.Query(ctx, `SELECT d.id::text,d.environment_id::text,d.deletion_requested_at IS NOT NULL FROM project_deployments d WHERE d.desired_status='stopped' AND d.deleted_at IS NULL AND EXISTS (SELECT 1 FROM deployment_services s WHERE s.project_deployment_id=d.id AND (s.observed_status<>'stopped' OR s.observed_generation<>s.desired_generation)) ORDER BY d.updated_at LIMIT 100`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var deploymentID, environmentID string
		var deleting bool
		if err = rows.Scan(&deploymentID, &environmentID, &deleting); err != nil {
			return count, err
		}
		if deleting {
			deleter, ok := stopper.(ProjectDeploymentDeleter)
			if !ok {
				return count, fmt.Errorf("项目删除调和器未配置")
			}
			_, _ = r.pool.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) SELECT id,'dispatched','正在停止 Kubernetes 工作负载' FROM deployment_runs WHERE project_deployment_id=$1 AND observed_status='pending'`, deploymentID)
			err = deleter.DeleteDeployment(ctx, environmentID, deploymentID)
		} else {
			err = stopper.StopDeployment(ctx, environmentID, deploymentID)
		}
		if err != nil {
			message := err.Error()
			if len(message) > 1024 {
				message = message[:1024]
			}
			if _, updateErr := r.pool.Exec(ctx, `UPDATE deployment_services SET observed_status='failed',last_message=$1,observed_at=now(),updated_at=now() WHERE project_deployment_id=$2 AND desired_status='stopped'`, message, deploymentID); updateErr != nil {
				return count, updateErr
			}
			eventMessage := message
			if deleting {
				eventMessage = deletionFailureStage(message)
			}
			_, _ = r.pool.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) SELECT id,'failed',$1 FROM deployment_runs WHERE project_deployment_id=$2 AND observed_status='pending'`, eventMessage, deploymentID)
			if err = r.reconcileDeployment(ctx, deploymentID); err != nil {
				return count, err
			}
			continue
		}
		if deleting {
			_, _ = r.pool.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) SELECT id,'observed','运行态消息已清理，正在释放工程端口' FROM deployment_runs WHERE project_deployment_id=$1 AND observed_status='pending'`, deploymentID)
			if err = r.finalizeDeletedDeployment(ctx, deploymentID); err != nil {
				return count, err
			}
			count++
			continue
		}
		if _, err = r.pool.Exec(ctx, `UPDATE deployment_services SET observed_status='stopped',replicas_observed=0,observed_generation=desired_generation,last_message='Kubernetes 工作负载已停止',observed_at=now(),updated_at=now() WHERE project_deployment_id=$1 AND desired_status='stopped'`, deploymentID); err != nil {
			return count, err
		}
		if err = r.reconcileDeployment(ctx, deploymentID); err != nil {
			return count, err
		}
		count++
	}
	return count, rows.Err()
}

func (r *PostgreSQLRepository) finalizeDeletedDeployment(ctx context.Context, deploymentID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `DELETE FROM deployment_services WHERE project_deployment_id=$1`, deploymentID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE project_deployments SET deleted_at=now(),observed_status='stopped',updated_at=now() WHERE id=$1 AND deletion_requested_at IS NOT NULL`, deploymentID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE deployment_runs SET observed_status='stopped',progress=100,completed_at=now() WHERE project_deployment_id=$1 AND observed_status='pending'`, deploymentID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO deployment_run_events(deployment_run_id,stage,message) SELECT id,'observed','工程端口已释放，部署已删除' FROM deployment_runs WHERE project_deployment_id=$1 AND operation='delete' ORDER BY started_at DESC,id DESC LIMIT 1`, deploymentID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// deletionFailureStage 只向工程使用者展示删除流程阶段，不暴露凭据、连接串或底层响应。
func deletionFailureStage(message string) string {
	if strings.Contains(message, "删除运行态消息") {
		return "运行态消息清理失败"
	}
	return "Kubernetes 工作负载停止失败"
}

func projectReleaseDigest(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "sha256:") {
		return "sha256:" + strings.ToLower(strings.TrimSpace(value[len("sha256:"):]))
	}
	return sha256Value(value)
}

func (r *KubernetesProjectReconciler) Status(ctx context.Context, workload ProjectWorkload) (ProjectWorkloadStatus, error) {
	namespace, err := projectNamespace(workload.EnvironmentID)
	if err != nil {
		return ProjectWorkloadStatus{}, err
	}
	name, err := projectWorkloadName(workload.DeploymentID, workload.Engine)
	if err != nil {
		return ProjectWorkloadStatus{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint+"/apis/apps/v1/namespaces/"+namespace+"/deployments/"+name, nil)
	if err != nil {
		return ProjectWorkloadStatus{}, err
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	resp, err := r.client.Do(req)
	if err != nil {
		return ProjectWorkloadStatus{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return ProjectWorkloadStatus{}, fmt.Errorf("读取 Kubernetes rollout 状态失败: HTTP %d", resp.StatusCode)
	}
	var value struct {
		Metadata struct {
			Generation int64 `json:"generation"`
		}
		Spec struct {
			Replicas int `json:"replicas"`
		}
		Status struct {
			ObservedGeneration int64                                            `json:"observedGeneration"`
			AvailableReplicas  int                                              `json:"availableReplicas"`
			Conditions         []struct{ Type, Status, Reason, Message string } `json:"conditions"`
		} `json:"status"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&value); err != nil {
		return ProjectWorkloadStatus{}, err
	}
	for _, condition := range value.Status.Conditions {
		if condition.Type == "Progressing" && condition.Status == "False" && condition.Reason == "ProgressDeadlineExceeded" {
			return ProjectWorkloadStatus{Failed: true, Message: condition.Message}, nil
		}
	}
	if value.Status.ObservedGeneration >= value.Metadata.Generation && value.Status.AvailableReplicas >= value.Spec.Replicas {
		return ProjectWorkloadStatus{Ready: true, ReplicasObserved: value.Status.AvailableReplicas, Message: "Kubernetes rollout 已就绪"}, nil
	}
	podStatus, err := r.projectPodFailure(ctx, namespace, name)
	if err != nil || podStatus.Failed {
		return podStatus, err
	}
	return ProjectWorkloadStatus{Message: "等待 Kubernetes rollout readiness"}, nil
}

func (r *KubernetesProjectReconciler) projectPodFailure(ctx context.Context, namespace, name string) (ProjectWorkloadStatus, error) {
	endpoint := r.endpoint + "/api/v1/namespaces/" + namespace + "/pods?labelSelector=" + url.QueryEscape("app.kubernetes.io/name="+name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ProjectWorkloadStatus{}, err
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	resp, err := r.client.Do(req)
	if err != nil {
		return ProjectWorkloadStatus{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return ProjectWorkloadStatus{}, fmt.Errorf("读取 Kubernetes Pod 状态失败: HTTP %d", resp.StatusCode)
	}
	type containerStatus struct {
		Name         string `json:"name"`
		RestartCount int    `json:"restartCount"`
		State        struct {
			Waiting    struct{ Reason string } `json:"waiting"`
			Terminated struct {
				ExitCode int `json:"exitCode"`
			} `json:"terminated"`
		} `json:"state"`
	}
	var pods struct {
		Items []struct {
			Status struct {
				InitContainerStatuses []containerStatus `json:"initContainerStatuses"`
				ContainerStatuses     []containerStatus `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pods); err != nil {
		return ProjectWorkloadStatus{}, err
	}
	for _, pod := range pods.Items {
		for _, status := range pod.Status.InitContainerStatuses {
			if status.State.Terminated.ExitCode != 0 {
				return ProjectWorkloadStatus{Failed: true, Message: "初始化阶段 " + status.Name + " 失败"}, nil
			}
		}
		for _, status := range pod.Status.ContainerStatuses {
			// 探针反复终止、但每次存活时间足以重置 kubelet backoff 时，容器可能
			// 永远处于 Running/Restarting 而不进入 CrashLoopBackOff。连续三次非零
			// 退出同样是不可收敛的 rollout；SIGTERM 可让进程以 0 退出，因此不能用
			// exitCode 过滤。连续重启三次必须结束 pending run 才能正式重试。
			if status.State.Waiting.Reason == "CrashLoopBackOff" || status.RestartCount >= 3 {
				return ProjectWorkloadStatus{Failed: true, Message: "运行容器 " + status.Name + " 反复崩溃"}, nil
			}
		}
	}
	return ProjectWorkloadStatus{}, nil
}

func workloadFailureStage(message string) string {
	if strings.Contains(message, "collector-binding") {
		return "collector-binding"
	}
	for name, stage := range map[string]string{"runtime-provision-nats": "provision-nats", "runtime-provision-state": "provision-state", "runtime-binding-prepare": "prepare"} {
		if strings.Contains(message, name) {
			return stage
		}
	}
	return "rollout"
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
	if r.hostNodes != nil {
		if err := r.EnsureHostNodeLabels(ctx, []string{workload.NodeID}); err != nil {
			return fmt.Errorf("工程节点标签预检失败: %w", err)
		}
	}
	if workload.Engine == ServiceCollector {
		if r.runtimeContext == nil || r.collectorBundle == nil || r.secretManager == nil {
			return fmt.Errorf("collector-binding 阶段依赖未配置")
		}
		runtimeContext, err := r.runtimeContext.LoadProjectRuntimeContext(ctx, workload.DeploymentID)
		if err != nil {
			return fmt.Errorf("collector-binding 加载运行上下文失败")
		}
		if len(runtimeContext.CollectorSourceSnapshot) == 0 {
			return fmt.Errorf("collector-binding 缺少 Release 冻结 sourceSnapshot")
		}
		collectorPath, err := collectorArtifactPath(runtimeContext.Release.Manifest)
		if err != nil {
			return fmt.Errorf("collector-binding Release 工件无效")
		}
		bundle, err := r.collectorBundle.BuildCollectorBindingBundle(ctx, dataservice.CollectorBindingBundleRequest{TenantID: runtimeContext.TenantID, ProjectID: runtimeContext.ProjectID, DeploymentID: workload.DeploymentID, EnvironmentID: workload.EnvironmentID, NodeID: workload.NodeID, ReleaseID: workload.ReleaseID, Revision: int64(runtimeContext.BindingRevision), SourceSnapshot: runtimeContext.CollectorSourceSnapshot, AccountID: runtimeAccountID(runtimeContext.ProjectID, workload.DeploymentID), NATSEndpoint: runtimeContext.Support.NATSEndpoint, NATSResourceRef: runtimeContext.Support.NATSResourceRef, NATSCredentialSecretRef: runtimeContext.Support.NATSCredentialSecretRef})
		if err != nil {
			return fmt.Errorf("collector-binding 构建失败")
		}
		namespace, nsErr := projectNamespace(workload.EnvironmentID)
		if nsErr != nil {
			return nsErr
		}
		secretName, err := r.secretManager.EnsureCollector(ctx, namespace, workload.DeploymentID, runtimeContext.Support, bundle.SecretFiles)
		if err != nil {
			return fmt.Errorf("collector-binding 准备 Secret 失败")
		}
		if err = r.applyCollectorBindingConfigMap(ctx, workload, bundle); err != nil {
			return fmt.Errorf("collector-binding 写入 ConfigMap 失败")
		}
		workload.CollectorBindingChecksum = bundle.BindingSHA256 + ":" + bundle.IndexSHA256
		workload.CollectorSecretName = secretName
		workload.BindingRevision = runtimeContext.BindingRevision
		workload.CollectorArtifactPath = collectorPath
	}
	if workload.Engine == ServiceBase || workload.Engine == ServiceCompute || workload.Engine == ServiceAlarm {
		if r.runtimeContext == nil {
			return fmt.Errorf("运行绑定上下文加载器未配置")
		}
		runtimeContext, err := r.runtimeContext.LoadProjectRuntimeContext(ctx, workload.DeploymentID)
		if err != nil {
			return fmt.Errorf("加载运行绑定上下文失败: %w", err)
		}
		workload.ProjectID = runtimeContext.ProjectID
		input, err := BuildRuntimeBindingInput(workload, runtimeContext)
		if err != nil {
			return fmt.Errorf("构造运行绑定输入失败: %w", err)
		}
		if r.secretManager == nil {
			return fmt.Errorf("部署 Secret 管理器未配置")
		}
		namespace, err := projectNamespace(workload.EnvironmentID)
		if err != nil {
			return err
		}
		if _, err = r.secretManager.Ensure(ctx, namespace, workload.DeploymentID, runtimeContext.ProjectID, runtimeContext.EnvironmentID, workload.Engine, runtimeContext.Support); err != nil {
			return fmt.Errorf("准备部署运行 Secret 失败: %w", err)
		}
		if err := r.applyRuntimeBindingConfigMap(ctx, workload, input); err != nil {
			return err
		}
		digest := sha256.Sum256(input)
		workload.RuntimeBindingChecksum = "sha256:" + hex.EncodeToString(digest[:])
		workload.BindingRevision = runtimeContext.BindingRevision
		workload.RuntimeNATSEndpoint = runtimeContext.Support.NATSEndpoint
	}
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
		if meta.Kind == "Service" {
			resource = "services"
		}
		path := kubernetesApplyPath(meta.Kind, namespace, resource, meta.Metadata.Name)
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
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 8<<10))
		response.Body.Close()
		if response.StatusCode/100 != 2 {
			var status struct {
				Message string `json:"message"`
			}
			_ = json.Unmarshal(responseBody, &status)
			message := strings.TrimSpace(status.Message)
			if len(message) > 512 {
				message = message[:512]
			}
			if message != "" {
				return fmt.Errorf("Kubernetes 调和 %s/%s 失败: HTTP %d: %s", meta.Kind, meta.Metadata.Name, response.StatusCode, message)
			}
			return fmt.Errorf("Kubernetes 调和 %s/%s 失败: HTTP %d", meta.Kind, meta.Metadata.Name, response.StatusCode)
		}
	}
	return nil
}

func kubernetesApplyPath(kind, namespace, resource, name string) string {
	prefix := "/apis/apps/v1"
	if kind == "ConfigMap" || kind == "Service" {
		prefix = "/api/v1"
	}
	return prefix + "/namespaces/" + namespace + "/" + resource + "/" + name
}

// StopDeployment 只删除由稳定 deployment ID 派生的工程资源。不得按宽泛标签
// 批量删除，避免同一运行环境内误伤其他工程。
func (r *KubernetesProjectReconciler) StopDeployment(ctx context.Context, environmentID, deploymentID string) error {
	namespace, err := projectNamespace(environmentID)
	if err != nil {
		return err
	}
	roles := []string{ServiceBase, ServiceCompute, ServiceAlarm, ServiceCollector}
	for _, role := range roles {
		name, nameErr := projectWorkloadName(deploymentID, role)
		if nameErr != nil {
			return nameErr
		}
		if err = r.deleteProjectResource(ctx, namespace, "Deployment", "deployments", name); err != nil {
			return err
		}
	}
	for _, role := range roles {
		name, nameErr := projectWorkloadName(deploymentID, role)
		if nameErr != nil {
			return nameErr
		}
		if err = r.waitProjectDeploymentDeleted(ctx, namespace, name); err != nil {
			return err
		}
	}
	for _, role := range roles {
		name, nameErr := projectWorkloadName(deploymentID, role)
		if nameErr != nil {
			return nameErr
		}
		if err = r.deleteProjectResource(ctx, namespace, "Service", "services", name); err != nil {
			return err
		}
		if err = r.deleteProjectResource(ctx, namespace, "ConfigMap", "configmaps", name+"-config"); err != nil {
			return err
		}
		binding := name + "-runtime-binding"
		if role == ServiceCollector {
			binding = name + "-collector-binding"
		}
		if err = r.deleteProjectResource(ctx, namespace, "ConfigMap", "configmaps", binding); err != nil {
			return err
		}
	}
	secretName, err := computeSandboxSecretName(deploymentID)
	if err != nil {
		return err
	}
	if err = r.deleteProjectResource(ctx, namespace, "Secret", "secrets", secretName); err != nil {
		return err
	}
	collectorSecret, err := collectorBundleSecretName(deploymentID)
	if err != nil {
		return err
	}
	return r.deleteProjectResource(ctx, namespace, "Secret", "secrets", collectorSecret)
}

// DeleteDeployment 仅在 K3s 资源确认停止后清理该 deployment 专属 JetStream
// 拓扑。stream 名由 project+deployment 的稳定键派生，不能接受调用方输入。
func (r *KubernetesProjectReconciler) DeleteDeployment(ctx context.Context, environmentID, deploymentID string) error {
	if err := r.StopDeployment(ctx, environmentID, deploymentID); err != nil {
		return err
	}
	if r.runtimeContext == nil {
		return fmt.Errorf("删除运行态消息缺少运行上下文加载器")
	}
	runtimeContext, err := r.runtimeContext.LoadProjectRuntimeContext(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("删除运行态消息加载运行上下文失败: %w", err)
	}
	secret, exists, err := r.GetSecret(ctx, runtimeContext.Support.NATSCredentialSource.Namespace, runtimeContext.Support.NATSCredentialSource.Name)
	credentialKey := runtimeContext.Support.NATSCredentialSource.Keys["credential"]
	if err != nil || !exists || strings.TrimSpace(credentialKey) == "" || strings.TrimSpace(secret[credentialKey]) == "" {
		return fmt.Errorf("删除运行态消息读取 NATS 凭据失败")
	}
	// 工程 Pod 与基础服务同 namespace，短 Service 名可正常解析；控制面位于
	// induforge-system，删除时必须把短名定位到环境基础服务 namespace。
	natsEndpoint := natsControlEndpoint(runtimeContext.Support.NATSEndpoint, runtimeContext.Support.NATSCredentialSource.Namespace)
	nc, err := nats.Connect(natsEndpoint, nats.Token(secret[credentialKey]), nats.Timeout(15*time.Second))
	if err != nil {
		return fmt.Errorf("删除运行态消息连接 NATS 失败")
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		return fmt.Errorf("删除运行态消息初始化 JetStream 失败")
	}
	key := strings.ToUpper(stableRuntimeKey(runtimeContext.ProjectID, deploymentID))
	streams := map[string][]string{
		"IF_" + key + "_RAW":     {"compute-raw-v1", "alarm-raw-v1"},
		"IF_" + key + "_DERIVED": {"compute-derived-v1", "alarm-derived-v1"},
		"IF_" + key + "_COMMAND": {"compute-command-v1"},
		"IF_" + key + "_EVENT":   nil,
		"IF_" + key + "_DLQ":     nil,
	}
	for stream, consumers := range streams {
		for _, consumer := range consumers {
			if err = js.DeleteConsumer(stream, consumer); err != nil && !errors.Is(err, nats.ErrConsumerNotFound) && !errors.Is(err, nats.ErrStreamNotFound) {
				return fmt.Errorf("删除运行态消息 consumer 失败: %s", consumer)
			}
		}
	}
	for _, stream := range []string{"IF_" + key + "_DLQ", "IF_" + key + "_COMMAND", "IF_" + key + "_EVENT", "IF_" + key + "_DERIVED", "IF_" + key + "_RAW"} {
		if err = js.DeleteStream(stream); err != nil && !errors.Is(err, nats.ErrStreamNotFound) {
			return fmt.Errorf("删除运行态消息 stream 失败: %s", stream)
		}
	}
	return nil
}

// natsControlEndpoint 仅修正环境内短 Service 名，外部或已完整限定的 NATS 地址保持原样。
// 这样项目工作负载仍可使用短名，中心控制面也能在跨 namespace 删除时连接同一 NATS。
func natsControlEndpoint(endpoint, namespace string) string {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Scheme != "nats" || parsed.Hostname() != "nats" || strings.TrimSpace(namespace) == "" {
		return endpoint
	}
	host := "nats." + namespace + ".svc"
	if port := parsed.Port(); port != "" {
		host += ":" + port
	}
	parsed.Host = host
	return parsed.String()
}

func (r *KubernetesProjectReconciler) deleteProjectResource(ctx context.Context, namespace, kind, resource, name string) error {
	prefix := "/api/v1"
	if kind == "Deployment" {
		prefix = "/apis/apps/v1"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, r.endpoint+prefix+"/namespaces/"+namespace+"/"+resource+"/"+name, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound && resp.StatusCode/100 != 2 {
		return fmt.Errorf("删除 Kubernetes %s %s 失败: HTTP %d", kind, name, resp.StatusCode)
	}
	return nil
}

func (r *KubernetesProjectReconciler) waitProjectDeploymentDeleted(ctx context.Context, namespace, name string) error {
	deadline := time.NewTimer(90 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint+"/apis/apps/v1/namespaces/"+namespace+"/deployments/"+name, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+r.token)
		resp, err := r.client.Do(req)
		if err != nil {
			return err
		}
		status := resp.StatusCode
		resp.Body.Close()
		if status == http.StatusNotFound {
			return nil
		}
		if status/100 != 2 {
			return fmt.Errorf("读取 Kubernetes Deployment %s 状态失败: HTTP %d", name, status)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("等待 Kubernetes Deployment %s 停止超时", name)
		case <-ticker.C:
		}
	}
}

func collectorArtifactPath(raw []byte) (string, error) {
	var manifest deployableReleaseManifest
	if json.Unmarshal(raw, &manifest) != nil || manifest.Artifacts.Collector == nil || manifest.Artifacts.Collector.File != collectorArtifactFile {
		return "", fmt.Errorf("collector artifact 缺失")
	}
	return manifest.Artifacts.Collector.File, nil
}

func (r *KubernetesProjectReconciler) applyCollectorBindingConfigMap(ctx context.Context, workload ProjectWorkload, bundle *dataservice.CollectorBindingBundle) error {
	namespace, err := projectNamespace(workload.EnvironmentID)
	if err != nil {
		return err
	}
	name, err := projectWorkloadName(workload.DeploymentID, ServiceCollector)
	if err != nil {
		return err
	}
	manifest := fmt.Sprintf("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: %s-collector-binding\n  namespace: %s\n  annotations: {induforge.io/binding-sha256: %q, induforge.io/index-sha256: %q}\ndata:\n  binding.json: %q\n  index.json: %q\n", name, namespace, bundle.BindingSHA256, bundle.IndexSHA256, string(bundle.Binding), string(bundle.Index))
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, r.endpoint+"/api/v1/namespaces/"+namespace+"/configmaps/"+name+"-collector-binding?fieldManager=induforge-center&force=true", strings.NewReader(manifest))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/apply-patch+yaml")
	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func (r *KubernetesProjectReconciler) applyRuntimeBindingConfigMap(ctx context.Context, workload ProjectWorkload, input []byte) error {
	namespace, err := projectNamespace(workload.EnvironmentID)
	if err != nil {
		return err
	}
	name, err := projectWorkloadName(workload.DeploymentID, workload.Engine)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(input)
	manifest := fmt.Sprintf("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: %s-runtime-binding\n  namespace: %s\n  annotations: {induforge.io/input-sha256: %q}\ndata:\n  input.json: %q\n", name, namespace, "sha256:"+hex.EncodeToString(digest[:]), string(input))
	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, r.endpoint+"/api/v1/namespaces/"+namespace+"/configmaps/"+name+"-runtime-binding?fieldManager=induforge-center&force=true", strings.NewReader(manifest))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+r.token)
	request.Header.Set("Content-Type", "application/apply-patch+yaml")
	response, err := r.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return fmt.Errorf("Kubernetes 调和 ConfigMap/%s-runtime-binding 失败: HTTP %d", name, response.StatusCode)
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

func projectArtifactHostPath(deploymentID, digest string) string {
	return filepath.Join("/var/lib/induforge/node-agent/deployments", deploymentID, "release", "releases", "sha256-"+strings.TrimPrefix(digest, "sha256:"))
}
