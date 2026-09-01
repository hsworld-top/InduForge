package hostd

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (m *Manager) ApplyFoundation(ctx context.Context, plan FoundationPlan) (FoundationState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := plan.Validate(); err != nil {
		return FoundationState{}, err
	}
	cluster, exists, err := m.loadState()
	if err != nil || !exists {
		return FoundationState{}, errors.New("运行底座尚未初始化")
	}
	cluster = m.observe(ctx, cluster)
	if cluster.ObservedState != "ready" || cluster.Operation != OperationInitServer || cluster.NodeID != plan.NodeID {
		return FoundationState{}, errors.New("只有已就绪的中心节点可以部署基础服务")
	}
	if existing, ok, err := m.loadFoundationState(plan.EnvironmentID); err != nil {
		return FoundationState{}, err
	} else if ok {
		if existing.EnvironmentID != plan.EnvironmentID || existing.NodeID != plan.NodeID || existing.Generation > plan.Generation {
			return FoundationState{}, errors.New("拒绝替换或回退基础服务计划")
		}
		if existing.Generation == plan.Generation {
			return m.observeFoundation(ctx, existing), nil
		}
	}
	secret, err := m.foundationSecret(plan.EnvironmentID)
	if err != nil {
		return FoundationState{}, err
	}
	manifestPath := m.foundationManifestPath(plan.EnvironmentID)
	if plan.Operation == "migrate" {
		return m.migrateFoundation(ctx, plan, secret, manifestPath)
	}
	if err := writeAtomic(manifestPath, []byte(plan.RenderManifest(secret)), 0600); err != nil {
		return FoundationState{}, fmt.Errorf("写入固定基础服务清单失败: %w", err)
	}
	if output, err := m.cfg.Runner.Run(ctx, m.cfg.BinaryPath, "kubectl", "apply", "-f", manifestPath); err != nil {
		return FoundationState{}, commandError("应用基础服务清单", output, err)
	}
	state := FoundationState{SchemaVersion: "induforge.foundation-state.v1", Generation: plan.Generation, EnvironmentID: plan.EnvironmentID, NodeID: plan.NodeID, ObservedState: "starting"}
	if err := m.saveFoundationState(state); err != nil {
		return FoundationState{}, err
	}
	return m.observeFoundation(ctx, state), nil
}

func (m *Manager) FoundationStatuses(ctx context.Context) ([]FoundationState, error) {
	// 状态查询只读取原子写入的文件并调用 kubectl。迁移期间不能等待 Manager
	// 的写锁，否则长耗时数据复制会阻塞节点心跳并把中心误判为离线。
	entries, err := os.ReadDir(m.cfg.StateDir)
	if err != nil {
		return nil, err
	}
	states := make([]FoundationState, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "foundation-state-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, readErr := os.ReadFile(filepath.Join(m.cfg.StateDir, entry.Name()))
		if readErr != nil {
			return nil, readErr
		}
		var state FoundationState
		if err := json.Unmarshal(raw, &state); err != nil {
			return nil, err
		}
		observed := m.observeFoundation(ctx, state)
		// 起始时间由状态文件 mtime 表示；部署中不能反复写回，否则永远无法触发超时。
		// 就绪/异常转换才持久化，用于重启后识别“曾经健康、现在丢失实例”。
		if observed.ObservedState != state.ObservedState && observed.ObservedState != "starting" {
			if err := m.saveFoundationState(observed); err != nil {
				return nil, err
			}
		}
		states = append(states, observed)
	}
	return states, nil
}

func (m *Manager) migrateFoundation(ctx context.Context, plan FoundationPlan, secret, manifestPath string) (FoundationState, error) {
	previousManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return FoundationState{}, fmt.Errorf("读取迁移前基础服务清单失败: %w", err)
	}
	state := FoundationState{SchemaVersion: "induforge.foundation-state.v1", Generation: plan.Generation, EnvironmentID: plan.EnvironmentID, NodeID: plan.NodeID, ObservedState: "starting", Message: "正在复制有状态服务数据"}
	if err := m.saveFoundationState(state); err != nil {
		return FoundationState{}, err
	}
	namespace := foundationNamespace(plan.EnvironmentID)
	changed := make([]string, 0)
	for _, workload := range foundationWorkloads {
		sourceNode := plan.SourceAssignments[workload]
		if sourceNode == "" || sourceNode == plan.Assignments[workload] {
			continue
		}
		changed = append(changed, workload)
		if workload == "nginx" {
			continue
		}
		if plan.Claims[workload] == "" {
			return FoundationState{}, fmt.Errorf("%s 迁移缺少目标存储声明", workload)
		}
		if output, runErr := m.cfg.Runner.Run(ctx, m.cfg.BinaryPath, "kubectl", "-n", namespace, "scale", "statefulset/"+workload, "--replicas=0"); runErr != nil {
			return m.rollbackFoundation(ctx, plan, previousManifest, changed, commandError("停止迁移源服务 "+workload, output, runErr))
		}
		if output, runErr := m.cfg.Runner.Run(ctx, m.cfg.BinaryPath, "kubectl", "-n", namespace, "wait", "--for=delete", "pod/"+workload+"-0", "--timeout=120s"); runErr != nil && !strings.Contains(string(output), "not found") {
			return m.rollbackFoundation(ctx, plan, previousManifest, changed, commandError("等待源服务停止 "+workload, output, runErr))
		}
		helperPath := filepath.Join(m.cfg.ConfigDir, "foundation-migration-"+strings.ToLower(plan.EnvironmentID)+"-"+workload+".yaml")
		sourceClaim := plan.SourceClaims[workload]
		if sourceClaim == "" {
			sourceClaim = "data-" + workload + "-0"
		}
		helper := renderFoundationMigration(namespace, workload, plan.Generation, sourceNode, plan.Assignments[workload], sourceClaim, plan.Claims[workload])
		if err := writeAtomic(helperPath, []byte(helper), 0600); err != nil {
			return m.rollbackFoundation(ctx, plan, previousManifest, changed, err)
		}
		if output, runErr := m.cfg.Runner.Run(ctx, m.cfg.BinaryPath, "kubectl", "apply", "-f", helperPath); runErr != nil {
			return m.rollbackFoundation(ctx, plan, previousManifest, changed, commandError("创建数据迁移任务 "+workload, output, runErr))
		}
		prefix := migrationResourcePrefix(workload, plan.Generation)
		if output, runErr := m.cfg.Runner.Run(ctx, m.cfg.BinaryPath, "kubectl", "-n", namespace, "wait", "--for=condition=Ready", "pod/"+prefix+"-source", "--timeout=180s"); runErr != nil {
			return m.rollbackFoundation(ctx, plan, previousManifest, changed, commandError("准备迁移源数据 "+workload, output, runErr))
		}
		if output, runErr := m.cfg.Runner.Run(ctx, m.cfg.BinaryPath, "kubectl", "-n", namespace, "wait", "--for=jsonpath={.status.phase}=Succeeded", "pod/"+prefix+"-target", "--timeout=300s"); runErr != nil {
			return m.rollbackFoundation(ctx, plan, previousManifest, changed, commandError("复制基础服务数据 "+workload, output, runErr))
		}
	}
	for _, workload := range changed {
		if workload == "nginx" {
			continue
		}
		if output, runErr := m.cfg.Runner.Run(ctx, m.cfg.BinaryPath, "kubectl", "-n", namespace, "delete", "statefulset/"+workload, "--ignore-not-found=true", "--wait=true"); runErr != nil {
			return m.rollbackFoundation(ctx, plan, previousManifest, changed, commandError("切换基础服务存储 "+workload, output, runErr))
		}
	}
	if err := writeAtomic(manifestPath, []byte(plan.RenderManifest(secret)), 0600); err != nil {
		return m.rollbackFoundation(ctx, plan, previousManifest, changed, err)
	}
	if output, runErr := m.cfg.Runner.Run(ctx, m.cfg.BinaryPath, "kubectl", "apply", "-f", manifestPath); runErr != nil {
		return m.rollbackFoundation(ctx, plan, previousManifest, changed, commandError("应用迁移后基础服务清单", output, runErr))
	}
	for _, workload := range changed {
		prefix := migrationResourcePrefix(workload, plan.Generation)
		_, _ = m.cfg.Runner.Run(context.Background(), m.cfg.BinaryPath, "kubectl", "-n", namespace, "delete", "pod/"+prefix+"-source", "pod/"+prefix+"-target", "service/"+prefix, "--ignore-not-found=true", "--wait=false")
		_ = os.Remove(filepath.Join(m.cfg.ConfigDir, "foundation-migration-"+strings.ToLower(plan.EnvironmentID)+"-"+workload+".yaml"))
	}
	state.Message = "数据复制完成，等待迁移后实例通过健康检查"
	if err := m.saveFoundationState(state); err != nil {
		return FoundationState{}, err
	}
	return m.observeFoundation(ctx, state), nil
}

func (m *Manager) rollbackFoundation(ctx context.Context, plan FoundationPlan, manifest []byte, changed []string, cause error) (FoundationState, error) {
	namespace := foundationNamespace(plan.EnvironmentID)
	for _, workload := range changed {
		if workload != "nginx" {
			_, _ = m.cfg.Runner.Run(ctx, m.cfg.BinaryPath, "kubectl", "-n", namespace, "delete", "statefulset/"+workload, "--ignore-not-found=true", "--wait=true")
		}
	}
	path := m.foundationManifestPath(plan.EnvironmentID)
	if err := writeAtomic(path, manifest, 0600); err == nil {
		_, _ = m.cfg.Runner.Run(ctx, m.cfg.BinaryPath, "kubectl", "apply", "-f", path)
	}
	state := FoundationState{SchemaVersion: "induforge.foundation-state.v1", Generation: plan.Generation, EnvironmentID: plan.EnvironmentID, NodeID: plan.NodeID, ObservedState: "failed", Message: "基础服务迁移失败，已尝试恢复原节点：" + cause.Error()}
	_ = m.saveFoundationState(state)
	return state, cause
}

func migrationResourcePrefix(workload string, generation int64) string {
	return fmt.Sprintf("if-migrate-%s-%d", workload, generation)
}

func renderFoundationMigration(namespace, workload string, generation int64, sourceNode, targetNode, sourceClaim, targetClaim string) string {
	prefix := migrationResourcePrefix(workload, generation)
	return fmt.Sprintf(`apiVersion: v1
kind: PersistentVolumeClaim
metadata: {name: %s, namespace: %s}
spec:
  accessModes: [ReadWriteOnce]
  resources: {requests: {storage: 10Gi}}
---
apiVersion: v1
kind: Pod
metadata: {name: %s-source, namespace: %s, labels: {app: %s}}
spec:
  restartPolicy: Never
  nodeSelector: {kubernetes.io/hostname: %s}
  containers:
    - name: exporter
      image: nginx:1.28-alpine
      imagePullPolicy: Never
      command: ["/bin/sh", "-c"]
      args: ["tar -C /source -czf /usr/share/nginx/html/data.tgz . && nginx -g 'daemon off;'"]
      readinessProbe: {httpGet: {path: /data.tgz, port: 80}, periodSeconds: 2, failureThreshold: 90}
      volumeMounts: [{name: source, mountPath: /source, readOnly: true}]
  volumes: [{name: source, persistentVolumeClaim: {claimName: %s}}]
---
apiVersion: v1
kind: Service
metadata: {name: %s, namespace: %s}
spec: {selector: {app: %s}, ports: [{port: 80, targetPort: 80}]}
---
apiVersion: v1
kind: Pod
metadata: {name: %s-target, namespace: %s}
spec:
  restartPolicy: Never
  nodeSelector: {kubernetes.io/hostname: %s}
  containers:
    - name: importer
      image: nginx:1.28-alpine
      imagePullPolicy: Never
      command: ["/bin/sh", "-c"]
      args: ["until wget -q -O /tmp/data.tgz http://%s/data.tgz; do sleep 2; done; tar -C /target -xzf /tmp/data.tgz"]
      volumeMounts: [{name: target, mountPath: /target}]
  volumes: [{name: target, persistentVolumeClaim: {claimName: %s}}]
`, targetClaim, namespace, prefix, namespace, prefix, foundationNodeName(sourceNode), sourceClaim, prefix, namespace, prefix, prefix, namespace, foundationNodeName(targetNode), prefix, targetClaim)
}

func (m *Manager) DeleteFoundation(ctx context.Context, request FoundationDeleteRequest) (FoundationState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := request.Validate(); err != nil {
		return FoundationState{}, err
	}
	cluster, exists, err := m.loadState()
	if err != nil || !exists {
		return FoundationState{}, errors.New("运行底座尚未初始化")
	}
	cluster = m.observe(ctx, cluster)
	if cluster.ObservedState != "ready" || cluster.Operation != OperationInitServer || cluster.NodeID != request.NodeID {
		return FoundationState{}, errors.New("只有已就绪的中心节点可以删除运行环境资源")
	}
	// Namespace 删除由 Kubernetes 异步收敛。这里不能同步等待所有 Pod、PVC 退出，
	// 否则长时间终止的工作负载会占住 Agent 调度循环并造成中心节点心跳误离线。
	if output, err := m.cfg.Runner.Run(ctx, m.cfg.BinaryPath, "kubectl", "delete", "namespace", foundationNamespace(request.EnvironmentID), "--ignore-not-found=true", "--wait=false"); err != nil {
		return FoundationState{}, commandError("删除运行环境隔离空间", output, err)
	}
	for _, path := range []string{m.foundationStatePath(request.EnvironmentID), m.foundationManifestPath(request.EnvironmentID), m.foundationCredentialPath(request.EnvironmentID)} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return FoundationState{}, err
		}
	}
	return FoundationState{SchemaVersion: "induforge.foundation-state.v1", EnvironmentID: request.EnvironmentID, NodeID: request.NodeID, ObservedState: "not-installed"}, nil
}

func (m *Manager) observeFoundation(ctx context.Context, state FoundationState) FoundationState {
	output, err := m.cfg.Runner.Run(ctx, m.cfg.BinaryPath, "kubectl", "-n", foundationNamespace(state.EnvironmentID), "get", "pods", "-l", "induforge.io/foundation=true", "-o", "json")
	if err != nil {
		state.ObservedState, state.Message = "failed", sanitizedCommandMessage(output, err)
		return state
	}
	var pods struct {
		Items []struct {
			Metadata struct {
				Name   string            `json:"name"`
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
			Status struct {
				Conditions        []struct{ Type, Status string } `json:"conditions"`
				ContainerStatuses []struct {
					State struct {
						Waiting *struct{ Reason, Message string } `json:"waiting"`
					} `json:"state"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(output, &pods); err != nil {
		state.ObservedState, state.Message = "failed", "无法解析基础服务健康状态"
		return state
	}
	states := make(map[string]FoundationServiceState, len(foundationWorkloads)+1)
	for _, workload := range foundationWorkloads {
		states[workload] = FoundationServiceState{Workload: workload, Status: "starting", Message: "等待实例就绪"}
	}
	failed := false
	for _, pod := range pods.Items {
		workload := pod.Metadata.Labels["induforge.io/service"]
		if workload == "" {
			for _, candidate := range foundationWorkloads {
				if strings.HasPrefix(pod.Metadata.Name, candidate+"-") {
					workload = candidate
					break
				}
			}
		}
		if _, ok := states[workload]; !ok {
			continue
		}
		service := states[workload]
		for _, condition := range pod.Status.Conditions {
			if condition.Type == "Ready" && condition.Status == "True" {
				service.Status, service.Message = "ready", ""
			}
		}
		for _, container := range pod.Status.ContainerStatuses {
			if container.State.Waiting == nil {
				continue
			}
			reason := container.State.Waiting.Reason
			if reason == "CrashLoopBackOff" || reason == "ImagePullBackOff" || reason == "ErrImagePull" || reason == "ErrImageNeverPull" || reason == "CreateContainerConfigError" {
				service.Status, service.Message, failed = "failed", reason, true
			}
		}
		states[workload] = service
	}
	traefik := FoundationServiceState{Workload: "traefik", Status: "starting", Message: "等待实例就绪"}
	if _, err := m.cfg.Runner.Run(ctx, m.cfg.BinaryPath, "kubectl", "-n", "kube-system", "wait", "--for=condition=Available", "deployment/traefik", "--timeout=1s"); err == nil {
		traefik.Status, traefik.Message = "ready", ""
	}
	states["traefik"] = traefik
	stableBefore := state.ObservedState == "ready" || state.ObservedState == "failed"
	deployTimedOut := false
	if info, err := os.Stat(m.foundationStatePath(state.EnvironmentID)); err == nil {
		deployTimedOut = time.Since(info.ModTime()) > 2*time.Minute
	}
	if stableBefore || deployTimedOut {
		for workload, service := range states {
			if service.Status == "starting" {
				service.Status, service.Message, failed = "failed", "基础服务实例未就绪", true
				states[workload] = service
			}
		}
	}
	allReady := true
	state.Services = state.Services[:0]
	for _, workload := range append(append([]string(nil), foundationWorkloads...), "traefik") {
		service := states[workload]
		state.Services = append(state.Services, service)
		if service.Status != "ready" {
			allReady = false
		}
	}
	if failed {
		state.ObservedState, state.Message = "failed", "一个或多个基础服务实例异常"
	} else if allReady {
		state.ObservedState, state.Message = "ready", ""
	} else {
		state.ObservedState, state.Message = "starting", "基础服务正在启动或等待健康检查"
	}
	return state
}

func (m *Manager) foundationStatePath(environmentID string) string {
	return filepath.Join(m.cfg.StateDir, "foundation-state-"+strings.ToLower(environmentID)+".json")
}

func (m *Manager) loadFoundationState(environmentID string) (FoundationState, bool, error) {
	raw, err := os.ReadFile(m.foundationStatePath(environmentID))
	if os.IsNotExist(err) {
		return FoundationState{}, false, nil
	}
	if err != nil {
		return FoundationState{}, false, err
	}
	var state FoundationState
	if err := json.Unmarshal(raw, &state); err != nil {
		return FoundationState{}, false, err
	}
	return state, true, nil
}

func (m *Manager) saveFoundationState(state FoundationState) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return writeAtomic(m.foundationStatePath(state.EnvironmentID), raw, 0600)
}

func (m *Manager) foundationManifestPath(environmentID string) string {
	return filepath.Join(m.cfg.ConfigDir, "foundation-"+strings.ToLower(environmentID)+".yaml")
}

func (m *Manager) foundationCredentialPath(environmentID string) string {
	return filepath.Join(m.cfg.StateDir, "foundation-credentials-"+strings.ToLower(environmentID))
}

func (m *Manager) foundationSecret(environmentID string) (string, error) {
	path := m.foundationCredentialPath(environmentID)
	if raw, err := os.ReadFile(path); err == nil {
		value := strings.TrimSpace(string(raw))
		if len(value) >= 32 {
			return value, nil
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	value := base64.RawURLEncoding.EncodeToString(raw)
	if err := writeAtomic(path, []byte(value+"\n"), 0600); err != nil {
		return "", err
	}
	return value, nil
}
