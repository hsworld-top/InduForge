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
	centerDataPath  string
	imagePreparer   interface {
		EnsureNodeImages(context.Context, string, []string) error
	}
	secretManager  *DeploymentSecretManager
	centerReleases *centerReleaseStore
	runtimeContext interface {
		LoadProjectRuntimeContext(context.Context, string) (ProjectRuntimeContext, error)
	}
	collectorBundle interface {
		BuildCollectorBindingBundle(context.Context, dataservice.CollectorBindingBundleRequest) (*dataservice.CollectorBindingBundle, error)
	}
	hostNodes interface {
		LoadHostNodeAddresses(context.Context, []string) (map[string]string, error)
	}
	hostNodeTargets interface {
		LoadHostNodeSchedulingTargets(context.Context, []string) (map[string]HostNodeSchedulingTarget, error)
	}
}

const hostNodeIDLabel = "induforge.io/host-node-id"

type KubernetesNode struct {
	Name            string
	InternalIP      string
	Ready           bool
	Labels          map[string]string
	CPUCapacity     string
	MemoryCapacity  string
	Architecture    string
	OperatingSystem string
	OSImage         string
	KernelVersion   string
	ResourceSummary map[string]any
}

func (r *KubernetesProjectReconciler) SetImagePreparer(p interface {
	EnsureNodeImages(context.Context, string, []string) error
}) {
	r.imagePreparer = p
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
	if targets, ok := loader.(interface {
		LoadHostNodeSchedulingTargets(context.Context, []string) (map[string]HostNodeSchedulingTarget, error)
	}); ok {
		r.hostNodeTargets = targets
	}
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
				Capacity  map[string]string `json:"capacity"`
				Addresses []struct {
					Type    string `json:"type"`
					Address string `json:"address"`
				} `json:"addresses"`
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
				NodeInfo struct {
					Architecture    string `json:"architecture"`
					OperatingSystem string `json:"operatingSystem"`
					OSImage         string `json:"osImage"`
					KernelVersion   string `json:"kernelVersion"`
				} `json:"nodeInfo"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	result := make([]KubernetesNode, 0, len(body.Items))
	for _, item := range body.Items {
		node := KubernetesNode{
			Name:            item.Metadata.Name,
			Labels:          item.Metadata.Labels,
			CPUCapacity:     item.Status.Capacity["cpu"],
			MemoryCapacity:  item.Status.Capacity["memory"],
			Architecture:    item.Status.NodeInfo.Architecture,
			OperatingSystem: item.Status.NodeInfo.OperatingSystem,
			OSImage:         item.Status.NodeInfo.OSImage,
			KernelVersion:   item.Status.NodeInfo.KernelVersion,
		}
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

// CollectNodeResourceSummaries 从 Metrics API 与 Kubelet summary 读取中心节点资源用量。
// 指标读取失败不会影响节点 Ready 状态，调用方仍可用 ListNodes 的结果继续调和。
func (r *KubernetesProjectReconciler) CollectNodeResourceSummaries(ctx context.Context, nodes []KubernetesNode) error {
	return r.collectNodeResourceSummaries(ctx, nodes)
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
	unique := uniqueNodeIDs(hostNodeIDs)
	targets := map[string]HostNodeSchedulingTarget{}
	if r.hostNodeTargets != nil {
		var err error
		targets, err = r.hostNodeTargets.LoadHostNodeSchedulingTargets(ctx, unique)
		if err != nil {
			return err
		}
	}
	agentIDs := make([]string, 0, len(unique))
	for _, id := range unique {
		if targets[id].NodeSource != "built_in" {
			agentIDs = append(agentIDs, id)
		}
	}
	addresses, err := r.hostNodes.LoadHostNodeAddresses(ctx, agentIDs)
	if err != nil {
		return err
	}
	nodes, err := r.ListNodes(ctx)
	if err != nil {
		return err
	}
	matchedHosts := make(map[string]string, len(unique))
	for _, hostNodeID := range unique {
		if target := targets[hostNodeID]; target.NodeSource == "built_in" {
			matched := false
			for _, node := range nodes {
				if node.Name == target.Hostname && node.Ready && node.Labels["induforge.io/center-node"] == "true" {
					matched = true
					break
				}
			}
			if !matched {
				return fmt.Errorf("中心内置节点 %s 未在 Kubernetes 中就绪", target.Hostname)
			}
			continue
		}
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
		// 重装会保留旧 Kubernetes 登记；优先选当前身份，不能仅用共享的历史 IP 判定。
		compactID := strings.ReplaceAll(hostNodeID, "-", "")
		if len(compactID) >= 12 {
			expectedName := "if-" + compactID[:12]
			for _, candidate := range matches {
				if candidate.Name == expectedName {
					matches = []KubernetesNode{candidate}
					break
				}
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
	PreparingResources bool
	Ready              bool
	Failed             bool
	ReplicasObserved   int
	Message            string
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
	rows, err := r.pool.Query(ctx, `SELECT d.tenant_id::text,s.id::text,s.project_deployment_id::text,d.environment_id::text,s.node_id::text,s.service_type,COALESCE(d.application_version_id::text,d.artifact_descriptor->>'releaseId'),COALESCE(v.artifact_hash,d.artifact_descriptor->>'artifactHash',''),s.desired_generation,s.public_port FROM deployment_services s JOIN project_deployments d ON d.id=s.project_deployment_id LEFT JOIN application_versions v ON v.id=d.application_version_id AND v.status='ready' WHERE NOT EXISTS(SELECT 1 FROM host_nodes n WHERE n.id=s.node_id AND n.resource_summary->>'cleanupRequested'='true') AND s.service_type<>'collector' AND s.desired_status='running' AND (s.observed_status<>'running' OR s.observed_generation<>s.desired_generation) ORDER BY s.updated_at LIMIT 100`)
	if err != nil {
		return 0, err
	}
	type pendingWorkload struct {
		tenant   string
		workload ProjectWorkload
	}
	items := []pendingWorkload{}
	for rows.Next() {
		var item pendingWorkload
		w := &item.workload
		if err = rows.Scan(&item.tenant, &w.ServiceID, &w.DeploymentID, &w.EnvironmentID, &w.NodeID, &w.Engine, &w.ReleaseID, &w.ReleaseDigest, &w.Generation, &w.HostPort); err != nil {
			rows.Close()
			return 0, err
		}
		w.ReleaseDigest = projectReleaseDigest(w.ReleaseDigest)
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	// 先释放数据库连接再访问K3s。保持单调和循环、最多100项串行，不按用户启动调和器。
	count := 0
	for _, item := range items {
		workload := item.workload
		status := ProjectWorkloadStatus{Message: "Kubernetes rollout 已提交，等待 readiness"}
		if applyErr := applier.Reconcile(ctx, workload); applyErr != nil {
			var pending *imagePreparationPending
			status = ProjectWorkloadStatus{Failed: !errors.As(applyErr, &pending), PreparingResources: pending != nil, Message: applyErr.Error()}
		} else if inspector, ok := applier.(ProjectWorkloadInspector); ok {
			if status, err = inspector.Status(ctx, workload); err != nil {
				status = ProjectWorkloadStatus{Failed: true, Message: err.Error()}
			}
		}
		tx, err := r.pool.Begin(ctx)
		if err != nil {
			return count, err
		}
		state, err := recordAndReconcileWorkload(ctx, tx, workload, status)
		if err != nil {
			_ = tx.Rollback(ctx)
			return count, err
		}
		if err = tx.Commit(ctx); err != nil {
			return count, err
		}
		if state.changed {
			r.publish(item.tenant, []string{"deployments", "environments", "events"}, []string{workload.DeploymentID, workload.EnvironmentID, workload.ServiceID}, state.observed != "pending")
		}
		count++
	}
	return count, nil
}

// ReconcileStoppedProjectDeployments 与运行态调和器成对工作。停止不能只更新
// desired_status，否则 K3s 工作负载会继续运行且 deployment run 永远 pending。
func (r *PostgreSQLRepository) ReconcileStoppedProjectDeployments(ctx context.Context, stopper ProjectDeploymentStopper) (int, error) {
	rows, err := r.pool.Query(ctx, `SELECT d.id::text,d.environment_id::text,d.tenant_id::text,d.deletion_requested_at IS NOT NULL,COALESCE((SELECT max(desired_generation) FROM deployment_services WHERE project_deployment_id=d.id),0) FROM project_deployments d WHERE d.desired_status='stopped' AND d.deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM deployment_services s JOIN host_nodes n ON n.id=s.node_id WHERE s.project_deployment_id=d.id AND n.resource_summary->>'cleanupRequested'='true') AND NOT EXISTS (SELECT 1 FROM deployment_services c WHERE c.project_deployment_id=d.id AND c.service_type='collector' AND (c.observed_status<>'stopped' OR c.observed_generation<>c.desired_generation)) AND (d.deletion_requested_at IS NOT NULL OR EXISTS (SELECT 1 FROM deployment_services s WHERE s.project_deployment_id=d.id AND (s.observed_status<>'stopped' OR s.observed_generation<>s.desired_generation))) ORDER BY d.updated_at LIMIT 100`)
	if err != nil {
		return 0, err
	}
	type pendingStop struct {
		deployment, environment, tenant string
		deleting                        bool
		generation                      int64
	}
	items := []pendingStop{}
	for rows.Next() {
		var item pendingStop
		if err = rows.Scan(&item.deployment, &item.environment, &item.tenant, &item.deleting, &item.generation); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, item := range items {
		deploymentID, environmentID, deleting := item.deployment, item.environment, item.deleting
		if deleting {
			deleter, ok := stopper.(ProjectDeploymentDeleter)
			if !ok {
				return count, fmt.Errorf("项目删除调和器未配置")
			}
			_, _ = r.pool.Exec(ctx, insertRunEventOnceSQL, "dispatched", "正在停止 Kubernetes 工作负载", deploymentID)
			err = deleter.DeleteDeployment(ctx, environmentID, deploymentID)
		} else {
			err = stopper.StopDeployment(ctx, environmentID, deploymentID)
		}
		if err != nil {
			message := err.Error()
			if deleting {
				message = deletionFailureStage(message)
			}
			if err = r.recordStoppedOutcome(ctx, deploymentID, item.generation, true, message); err != nil {
				return count, err
			}
			continue
		}
		if deleting {
			_, _ = r.pool.Exec(ctx, insertRunEventOnceSQL, "observed", "运行态消息已清理，正在释放工程端口", deploymentID)
			if err = r.finalizeDeletedDeployment(ctx, deploymentID); err != nil {
				return count, err
			}
			r.publish(item.tenant, []string{"deployments", "environments", "events"}, nil, true)
			count++
			continue
		}
		if err = r.recordStoppedOutcome(ctx, deploymentID, item.generation, false, "Kubernetes 工作负载已停止"); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
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
		if workload.Engine == ServiceBase && workload.HostPort != nil {
			ready, serviceErr := r.projectEntryReady(ctx, namespace, name)
			if serviceErr != nil {
				return ProjectWorkloadStatus{}, serviceErr
			}
			if !ready {
				return ProjectWorkloadStatus{ReplicasObserved: value.Status.AvailableReplicas, Message: "等待工程访问入口就绪"}, nil
			}
		}
		return ProjectWorkloadStatus{Ready: true, ReplicasObserved: value.Status.AvailableReplicas, Message: "Kubernetes rollout 已就绪"}, nil
	}
	podStatus, err := r.projectPodFailure(ctx, namespace, name)
	if err != nil || podStatus.Failed {
		return podStatus, err
	}
	return ProjectWorkloadStatus{Message: "等待 Kubernetes rollout readiness"}, nil
}

// projectEntryReady 以 ServiceLB 已公布入口作为基础引擎完成门槛，
// 避免 Pod 已 Ready 但用户端口尚未可达时提前显示“运行中”。
func (r *KubernetesProjectReconciler) projectEntryReady(ctx context.Context, namespace, name string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint+"/api/v1/namespaces/"+namespace+"/services/"+name, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	resp, err := r.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return false, fmt.Errorf("读取 Kubernetes 工程入口状态失败: HTTP %d", resp.StatusCode)
	}
	var service struct {
		Status struct {
			LoadBalancer struct {
				Ingress []struct {
					IP, Hostname string
				} `json:"ingress"`
			} `json:"loadBalancer"`
		} `json:"status"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&service); err != nil {
		return false, err
	}
	for _, ingress := range service.Status.LoadBalancer.Ingress {
		if strings.TrimSpace(ingress.IP) != "" || strings.TrimSpace(ingress.Hostname) != "" {
			return true, nil
		}
	}
	return false, nil
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
		// 初始化和运行容器都只使用节点包镜像，缺失必须结束等待并给出处理方向。
		for _, status := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
			switch status.State.Waiting.Reason {
			case "ErrImageNeverPull", "ImagePullBackOff", "ErrImagePull", "InvalidImageName":
				return ProjectWorkloadStatus{Failed: true, Message: "节点缺少所需运行镜像，请检查节点安装包是否完整并重新导入镜像（" + status.State.Waiting.Reason + "）"}, nil
			}
		}
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
	// deployment_run_events.stage 是稳定生命周期枚举；具体失败阶段留在安全 message，
	// 不得把容器名或调和子阶段写入受约束列而丢失整条运维事件。
	return "failed"
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
	return &KubernetesProjectReconciler{client: &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}}}, endpoint: "https://" + host + ":" + port, token: string(bytes.TrimSpace(token)), centerDataPath: strings.TrimSpace(os.Getenv("IF_OPS_CENTER_DATA_PATH"))}, nil
}

// Reconcile 使用稳定名称的 ConfigMap/Deployment server-side apply；同名模板更新由
// Kubernetes RollingUpdate 接管。403 明确暴露为 RBAC 配置错误，不能伪报已运行。
func (r *KubernetesProjectReconciler) Reconcile(ctx context.Context, workload ProjectWorkload) error {
	if workload.Engine == ServiceCollector {
		return fmt.Errorf("采集生命周期由 NodeAgent 管理，拒绝创建采集 Pod")
	}
	if all, ok := r.imagePreparer.(interface {
		EnsureDeploymentImages(context.Context, string) error
	}); ok {
		if err := all.EnsureDeploymentImages(ctx, workload.DeploymentID); err != nil {
			return err
		}
	}
	if r.imagePreparer != nil {
		if err := r.imagePreparer.EnsureNodeImages(ctx, workload.NodeID, projectImageReferences(workload.Engine)); err != nil {
			return err
		}
	}
	if r.hostNodes != nil {
		if err := r.EnsureHostNodeLabels(ctx, []string{workload.NodeID}); err != nil {
			return &nodePreflightError{cause: err}
		}
	}
	if r.hostNodeTargets != nil {
		targets, err := r.hostNodeTargets.LoadHostNodeSchedulingTargets(ctx, []string{workload.NodeID})
		if err != nil {
			return fmt.Errorf("加载工程调度节点失败: %w", err)
		}
		if targets[workload.NodeID].NodeSource == "built_in" {
			workload.NodeSelectorKey = "induforge.io/center-node"
			workload.NodeSelectorValue = "true"
		}
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
		if workload.NodeSelectorKey == "induforge.io/center-node" {
			workload.ArtifactHostPath, err = r.centerReleases.prepare(ctx, workload.DeploymentID, runtimeContext.Release)
			if err != nil {
				return fmt.Errorf("准备中心工程制品失败: %w", err)
			}
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
	roles := []string{ServiceBase, ServiceCompute, ServiceAlarm}
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
	return nil
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
