package ops

import (
	"fmt"
	"strings"
)

const (
	projectGatewayImage = "induforge/project-gateway:1.0.0"
	runtimeEngineImage  = "induforge/runtime-engine:1.0.0"
	computeSandboxImage = "induforge/compute-sandbox:1.0.0"
)

// ProjectWorkload 是中心控制面唯一可调和的固定 K3s 工作负载输入。它不接收
// 任意 YAML、命令或镜像：节点只负责制品准备和状态，中心以环境级 RBAC 写集群。
type ProjectWorkload struct {
	EnvironmentID, DeploymentID, ServiceID, NodeID, Engine, ReleaseID string
	Generation                                                        int64
	HostPort                                                          *int
	RuntimeBindingChecksum                                            string
	BindingRevision                                                   int
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

func computeSandboxSecretName(deploymentID string) (string, error) {
	name, err := projectWorkloadName(deploymentID, ServiceCompute)
	if err != nil {
		return "", err
	}
	return name + "-sandbox", nil
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
	artifactRoot := projectArtifactHostPath(workload.DeploymentID)
	image, role := runtimeEngineImage, workload.Engine
	container := "runtime-engine"
	if workload.Engine == ServiceBase {
		image, role, container = projectGatewayImage, "base", "project-gateway"
	}
	hostPort := ""
	sandbox := ""
	runtimeInit, runtimeArgs, runtimeMounts, runtimeVolumes := "", "", "", ""
	if workload.Engine == ServiceCompute || workload.Engine == ServiceAlarm {
		secretName, nameErr := computeSandboxSecretName(workload.DeploymentID)
		if nameErr != nil {
			return "", nameErr
		}
		bindingName := name + "-runtime-binding"
		runtimeInit = fmt.Sprintf(`
      initContainers:
        - name: runtime-binding-prepare
          image: %s
          imagePullPolicy: IfNotPresent
          command: ["if-runtime-provisioner", "prepare", "--input", "/etc/induforge/runtime-binding/input.json"]
          securityContext: {allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: ["ALL"]}}
          volumeMounts:
            - {name: runtime-binding, mountPath: /etc/induforge/runtime-binding, readOnly: true}
            - {name: release, mountPath: /opt/induforge/release, readOnly: true}
            - {name: work, mountPath: /work}`, runtimeEngineImage)
		runtimeArgs = `
          command: ["runtime-engine"]
          args: ["--config", "/work/bundle/runtime-engine-config.json", "--config-root", "/work/artifact", "--index", "/work/bundle/site-index.json", "--listen", "0.0.0.0:18080"]`
		runtimeMounts = `
            - {name: runtime-binding, mountPath: /etc/induforge/runtime-binding, readOnly: true}
            - {name: deployment-secrets, mountPath: /work/bundle/secrets, readOnly: true}`
		runtimeVolumes = fmt.Sprintf(`
        - name: runtime-binding
          configMap: {name: %s}
        - name: deployment-secrets
          secret: {secretName: %s}`, bindingName, secretName)
	}
	if workload.Engine == ServiceCompute {
		secretName, nameErr := computeSandboxSecretName(workload.DeploymentID)
		if nameErr != nil {
			return "", nameErr
		}
		sandbox = fmt.Sprintf(`
        - name: compute-sandbox
          image: %s
          imagePullPolicy: IfNotPresent
          env:
            - {name: COMPUTE_SANDBOX_TOKEN, valueFrom: {secretKeyRef: {name: %s, key: sandbox-token}}}
            - {name: COMPUTE_SANDBOX_SITE_ID, value: %q}
            - {name: COMPUTE_SANDBOX_NODE_ID, value: %q}
            - {name: COMPUTE_SANDBOX_EXECUTION_FORM, value: "native-linux"}
            - {name: COMPUTE_SANDBOX_DEPLOYMENT_ID, value: %q}
            - {name: COMPUTE_SANDBOX_PROJECT_ID, value: %q}
            - {name: COMPUTE_SANDBOX_ARTIFACT_ROOT, value: "/opt/induforge/release/current"}
            - {name: COMPUTE_SANDBOX_ARTIFACT_FILE, value: "runtime-artifact.tar.zst"}
          ports: [{name: sandbox, containerPort: 18103}]
          readinessProbe: {httpGet: {path: /health, port: sandbox}, initialDelaySeconds: 3, periodSeconds: 3}
          livenessProbe: {httpGet: {path: /health, port: sandbox}, initialDelaySeconds: 15, periodSeconds: 10}
          resources: {requests: {cpu: "100m", memory: "128Mi"}, limits: {cpu: "500m", memory: "512Mi"}}
          securityContext: {allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: ["ALL"]}}
          volumeMounts:
            - {name: release, mountPath: /opt/induforge/release, readOnly: true}
            - {name: deployment-secrets, mountPath: /var/run/induforge/secrets, readOnly: true}`, computeSandboxImage, secretName, workload.EnvironmentID, workload.NodeID, workload.DeploymentID, workload.DeploymentID)
	}
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
    metadata:
      labels: {app.kubernetes.io/name: %q, induforge.io/release-id: %q}
      annotations: {induforge.io/runtime-binding-sha256: %q, induforge.io/release-id: %q, induforge.io/generation: %q, induforge.io/binding-revision: %q}
    spec:
      nodeSelector: {induforge.io/node-id: %q}
      securityContext: {runAsNonRoot: true, seccompProfile: {type: RuntimeDefault}}
%s
      containers:
        - name: %s
          image: %s
          imagePullPolicy: IfNotPresent
%s
          env:
            - {name: IF_ENGINE_ROLE, valueFrom: {configMapKeyRef: {name: %s-config, key: engine-role}}}
            - {name: IF_RELEASE_ID, valueFrom: {configMapKeyRef: {name: %s-config, key: release-id}}}
            - {name: IF_RELEASE_ROOT, value: "/opt/induforge/release/current"}
            - {name: IF_WORK_ROOT, value: "/work"}
            - {name: IF_DEPLOYMENT_ID, value: %q}
            - {name: IF_PROJECT_ID, value: %q}
            - {name: IF_ENVIRONMENT_ID, value: %q}
            - {name: IF_NODE_ID, value: %q}
          ports:
            - name: http
              containerPort: 18080%s
          readinessProbe: {httpGet: {path: /health, port: http}, initialDelaySeconds: 3, periodSeconds: 3}
          livenessProbe: {httpGet: {path: /health, port: http}, initialDelaySeconds: 15, periodSeconds: 10}
          securityContext: {allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: ["ALL"]}}
          volumeMounts:
            - {name: release, mountPath: /opt/induforge/release, readOnly: true}
            - {name: work, mountPath: /work}
%s
%s
      volumes:
        - name: release
          hostPath: {path: %q, type: Directory}
        - name: work
          emptyDir: {}
%s
---
apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
  labels: {induforge.io/project-workload: "true", induforge.io/service-id: %q}
spec:
  selector: {app.kubernetes.io/name: %q}
  ports: [{name: http, port: 80, targetPort: http}]
`, name, namespace, role, workload.ReleaseID, name, namespace, workload.ServiceID, name, name, workload.ReleaseID, workload.RuntimeBindingChecksum, workload.ReleaseID, fmt.Sprint(workload.Generation), fmt.Sprint(workload.BindingRevision), workload.NodeID, runtimeInit, container, image, runtimeArgs, name, name, workload.DeploymentID, workload.DeploymentID, workload.EnvironmentID, workload.NodeID, hostPort, runtimeMounts, sandbox, artifactRoot, runtimeVolumes, name, namespace, workload.ServiceID, name), nil
}
