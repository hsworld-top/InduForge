package ops

import (
	"fmt"
	"strings"
)

// ProjectWorkload 是中心控制面唯一可调和的固定 K3s 工作负载输入。它不接收
// 任意 YAML、命令或镜像：节点只负责制品准备和状态，中心以环境级 RBAC 写集群。
type ProjectWorkload struct {
	EnvironmentID, DeploymentID, ServiceID, NodeID, Engine, ReleaseID string
	Generation                                                        int64
	HostPort                                                          *int
}

func projectNamespace(environmentID string) (string, error) {
	value := strings.ReplaceAll(strings.ToLower(environmentID), "-", "")
	if len(value) != 32 {
		return "", fmt.Errorf("运行环境 ID 无效")
	}
	return "if-env-" + value[:12], nil
}

func projectWorkloadName(deploymentID, engine string) (string, error) {
	value := strings.ReplaceAll(strings.ToLower(deploymentID), "-", "")
	if len(value) != 32 {
		return "", fmt.Errorf("部署 ID 无效")
	}
	if engine != ServiceBase && engine != ServiceCompute && engine != ServiceAlarm && engine != ServiceCollector {
		return "", fmt.Errorf("引擎类型无效")
	}
	return "if-project-" + value[:12] + "-" + engine, nil
}

// RenderProjectWorkloadManifest 生成稳定资源名。base 暴露 hostPort；compute/alarm
// 使用同一 runtime 镜像但强制不同 IF_ENGINE_ROLE，避免一个 Pod 同时执行两类任务。
func RenderProjectWorkloadManifest(workload ProjectWorkload) (string, error) {
	namespace, err := projectNamespace(workload.EnvironmentID)
	if err != nil {
		return "", err
	}
	name, err := projectWorkloadName(workload.DeploymentID, workload.Engine)
	if err != nil {
		return "", err
	}
	if workload.Generation < 1 || strings.TrimSpace(workload.NodeID) == "" || strings.TrimSpace(workload.ReleaseID) == "" {
		return "", fmt.Errorf("工作负载调和字段不完整")
	}
	image, role := "induforge/runtime-engine:latest", workload.Engine
	container := "runtime-engine"
	if workload.Engine == ServiceBase {
		image, role, container = "induforge/project-gateway:latest", "base", "project-gateway"
	}
	hostPort := ""
	if workload.Engine == ServiceBase {
		if workload.HostPort == nil || *workload.HostPort < 1024 || *workload.HostPort > 65535 {
			return "", fmt.Errorf("基础引擎 hostPort 无效")
		}
		hostPort = fmt.Sprintf("\n              hostPort: %d", *workload.HostPort)
	} else if workload.HostPort != nil {
		return "", fmt.Errorf("仅基础引擎允许 hostPort")
	}
	return fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-config
  namespace: %s
  labels: {induforge.io/project-workload: "true"}
data:
  engine-role: %q
  release-id: %q
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels: {induforge.io/project-workload: "true", induforge.io/service-id: %q}
spec:
  replicas: 1
  strategy: {type: RollingUpdate, rollingUpdate: {maxUnavailable: 0, maxSurge: 1}}
  selector: {matchLabels: {app.kubernetes.io/name: %q}}
  template:
    metadata: {labels: {app.kubernetes.io/name: %q, induforge.io/release-id: %q}}
    spec:
      nodeSelector: {induforge.io/node-id: %q}
      securityContext: {runAsNonRoot: true, seccompProfile: {type: RuntimeDefault}}
      containers:
        - name: %s
          image: %s
          imagePullPolicy: IfNotPresent
          env:
            - {name: IF_ENGINE_ROLE, valueFrom: {configMapKeyRef: {name: %s-config, key: engine-role}}}
            - {name: IF_RELEASE_ID, valueFrom: {configMapKeyRef: {name: %s-config, key: release-id}}}
          ports:
            - name: http
              containerPort: 18080%s
          readinessProbe: {httpGet: {path: /health, port: http}, initialDelaySeconds: 3, periodSeconds: 3}
          livenessProbe: {httpGet: {path: /health, port: http}, initialDelaySeconds: 15, periodSeconds: 10}
          securityContext: {allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: ["ALL"]}}
`, name, namespace, role, workload.ReleaseID, name, namespace, workload.ServiceID, name, name, workload.ReleaseID, workload.NodeID, container, image, name, name, hostPort), nil
}
