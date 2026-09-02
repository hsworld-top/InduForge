package ops

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	projectGatewayImage  = "induforge/project-gateway:1.0.0"
	runtimeAPIImage      = "induforge/project-runtime-api:1.0.0"
	// 运行镜像使用离线基线的不可变版本标签，禁止复用 1.0.0 触发 IfNotPresent 漂移。
	// 运行镜像采用构建基线的不可变版本，避免同标签重导入被 IfNotPresent 缓存。
	runtimeEngineImage   = "induforge/runtime-engine:1.0.3"
	collectorEngineImage = "induforge/collector-engine:1.0.3"
	computeSandboxImage  = "induforge/compute-sandbox:1.0.0"
)

// ProjectWorkload 是中心控制面唯一可调和的固定 K3s 工作负载输入。它不接收
// 任意 YAML、命令或镜像：节点只负责制品准备和状态，中心以环境级 RBAC 写集群。
type ProjectWorkload struct {
	EnvironmentID, ProjectID, DeploymentID, ServiceID, NodeID, Engine, ReleaseID, ReleaseDigest string
	Generation                                                                                  int64
	HostPort                                                                                    *int
	RuntimeBindingChecksum                                                                      string
	BindingRevision                                                                             int
	RuntimeNATSEndpoint                                                                         string
	CollectorBindingChecksum, CollectorSecretName, CollectorArtifactPath                        string
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
	if workload.Generation < 1 || strings.TrimSpace(workload.NodeID) == "" || strings.TrimSpace(workload.ReleaseID) == "" || !validSHA256Checksum(workload.ReleaseDigest) {
		return "", fmt.Errorf("工作负载调和字段不完整")
	}
	artifactRoot := projectArtifactHostPath(workload.DeploymentID, workload.ReleaseDigest)
	if workload.Engine == ServiceCollector {
		if workload.CollectorSecretName == "" {
			workload.CollectorSecretName, err = collectorBundleSecretName(workload.DeploymentID)
			if err != nil {
				return "", err
			}
		}
		return renderCollectorWorkloadManifest(workload, namespace, name, artifactRoot)
	}
	image, role := runtimeEngineImage, workload.Engine
	container := "runtime-engine"
	if workload.Engine == ServiceBase {
		image, role, container = projectGatewayImage, "base", "project-gateway"
	} else if workload.Engine == ServiceCollector {
		image, container = collectorEngineImage, "collector-engine"
	}
	hostPort := ""
	sandbox := ""
	runtimeInit, runtimeArgs, runtimeMounts, runtimeVolumes, apiSidecar := "", "", "", "", ""
	if workload.Engine == ServiceBase {
		secretName, nameErr := computeSandboxSecretName(workload.DeploymentID)
		if nameErr != nil {
			return "", nameErr
		}
		if strings.TrimSpace(workload.RuntimeNATSEndpoint) == "" {
			return "", fmt.Errorf("基础引擎缺少 Runtime API NATS endpoint")
		}
		bindingName := name + "-runtime-binding"
		runtimeInit = fmt.Sprintf(`
      initContainers:
        - name: runtime-provision-nats
          image: %s
          imagePullPolicy: IfNotPresent
          command: ["if-runtime-provisioner"]
          args: ["provision-nats", "--input", "/etc/induforge/runtime-binding/input.json", "--credentials", "/var/run/induforge/bootstrap/nats.json"]
          resources: {requests: {cpu: "50m", memory: "64Mi"}, limits: {cpu: "250m", memory: "256Mi"}}
          securityContext: {allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: ["ALL"]}}
          volumeMounts:
            - {name: runtime-binding, mountPath: /etc/induforge/runtime-binding, readOnly: true}
            - {name: bootstrap-nats, mountPath: /var/run/induforge/bootstrap, readOnly: true}
        - name: runtime-provision-state
          image: %s
          imagePullPolicy: IfNotPresent
          command: ["if-runtime-provisioner"]
          args: ["provision-state", "--input", "/etc/induforge/runtime-binding/input.json", "--credentials", "/var/run/induforge/bootstrap/postgres-bootstrap.json"]
          resources: {requests: {cpu: "50m", memory: "64Mi"}, limits: {cpu: "250m", memory: "256Mi"}}
          securityContext: {allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: ["ALL"]}}
          volumeMounts:
            - {name: runtime-binding, mountPath: /etc/induforge/runtime-binding, readOnly: true}
            - {name: bootstrap-state, mountPath: /var/run/induforge/bootstrap, readOnly: true}
        - name: runtime-api-artifact-prepare
          image: %s
          imagePullPolicy: IfNotPresent
          command: ["if-runtime-provisioner"]
          args: ["unpack-runtime", "--archive", "/opt/induforge/release/runtime-artifact.tar.zst", "--target", "/work/runtime-api-artifact"]
          resources: {requests: {cpu: "50m", memory: "64Mi"}, limits: {cpu: "250m", memory: "256Mi"}}
          securityContext: {allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: ["ALL"]}}
          volumeMounts:
            - {name: release, mountPath: /opt/induforge/release, readOnly: true}
            - {name: work, mountPath: /work}`, runtimeEngineImage, runtimeEngineImage, runtimeEngineImage)
		runtimeVolumes = fmt.Sprintf(`
        - name: runtime-binding
          configMap: {name: %s}
        - name: bootstrap-nats
          secret:
            secretName: %s
            items:
              - {key: nats.json, path: nats.json}
        - name: bootstrap-state
          secret:
            secretName: %s
            items:
              - {key: postgres-bootstrap.json, path: postgres-bootstrap.json}
        - name: runtime-api-secrets
          secret:
            secretName: %s
            defaultMode: 0440
            items:
              - {key: runtime-api-nats.json, path: nats.json}
              - {key: runtime-api-postgres.json, path: postgres.json}
              - {key: runtime-api-tokens.json, path: tokens.json}`, bindingName, secretName, secretName, secretName)
		apiSidecar = fmt.Sprintf(`
        - name: runtime-api
          image: %s
          imagePullPolicy: IfNotPresent
          args: ["--listen", "127.0.0.1:18081", "--artifact", "/work/runtime-api-artifact/runtime-project-artifact.json", "--postgres-secret", "/var/run/induforge/runtime-api/postgres.json", "--token-secret", "/var/run/induforge/runtime-api/tokens.json", "--nats-credentials", "/var/run/induforge/runtime-api/nats.json", "--deployment-id", %q, "--project-id", %q, "--account-id", %q, "--site-id", %q, "--node-id", %q, "--version", %q, "--manual-owner", "runtime-api", "--manual-epoch", %q, "--execution-form", "k3s-workload"]
          env: [{name: IF_RUNTIME_NATS_URL, value: %q}]
          resources: {requests: {cpu: "100m", memory: "128Mi"}, limits: {cpu: "500m", memory: "512Mi"}}
          readinessProbe: {exec: {command: ["/usr/local/bin/runtime-api", "healthcheck", "--url", "http://127.0.0.1:18081/health"]}, initialDelaySeconds: 3, periodSeconds: 3}
          livenessProbe: {exec: {command: ["/usr/local/bin/runtime-api", "healthcheck", "--url", "http://127.0.0.1:18081/health"]}, initialDelaySeconds: 15, periodSeconds: 10}
          securityContext: {allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: ["ALL"]}}
          volumeMounts:
            - {name: work, mountPath: /work, readOnly: true}
            - {name: runtime-api-secrets, mountPath: /var/run/induforge/runtime-api, readOnly: true}`, runtimeAPIImage, workload.DeploymentID, workload.ProjectID, "if-"+stableRuntimeKey(workload.DeploymentID), workload.EnvironmentID, workload.NodeID, workload.ReleaseID, strconv.Itoa(max(1, workload.BindingRevision)), workload.RuntimeNATSEndpoint)
	}
	if workload.Engine == ServiceCompute || workload.Engine == ServiceAlarm {
		secretName, nameErr := computeSandboxSecretName(workload.DeploymentID)
		if nameErr != nil {
			return "", nameErr
		}
		bindingName := name + "-runtime-binding"
		runtimeSecretItems := "\n              - {key: nats.json, path: nats.json}\n              - {key: postgres.json, path: postgres.json}"
		if workload.Engine == ServiceCompute {
			runtimeSecretItems += "\n              - {key: sandbox.json, path: sandbox.json}"
		}
		runtimeInit = fmt.Sprintf(`
      initContainers:
        - name: runtime-provision-nats
          image: %s
          imagePullPolicy: IfNotPresent
          command: ["if-runtime-provisioner"]
          args: ["provision-nats", "--input", "/etc/induforge/runtime-binding/input.json", "--credentials", "/var/run/induforge/bootstrap/nats.json"]
          resources: {requests: {cpu: "50m", memory: "64Mi"}, limits: {cpu: "250m", memory: "256Mi"}}
          securityContext: {allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: ["ALL"]}}
          volumeMounts:
            - {name: runtime-binding, mountPath: /etc/induforge/runtime-binding, readOnly: true}
            - {name: bootstrap-nats, mountPath: /var/run/induforge/bootstrap, readOnly: true}
        - name: runtime-provision-state
          image: %s
          imagePullPolicy: IfNotPresent
          command: ["if-runtime-provisioner"]
          args: ["provision-state", "--input", "/etc/induforge/runtime-binding/input.json", "--credentials", "/var/run/induforge/bootstrap/postgres-bootstrap.json"]
          resources: {requests: {cpu: "50m", memory: "64Mi"}, limits: {cpu: "250m", memory: "256Mi"}}
          securityContext: {allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: ["ALL"]}}
          volumeMounts:
            - {name: runtime-binding, mountPath: /etc/induforge/runtime-binding, readOnly: true}
            - {name: bootstrap-state, mountPath: /var/run/induforge/bootstrap, readOnly: true}
        - name: runtime-binding-prepare
          image: %s
          imagePullPolicy: IfNotPresent
          command: ["if-runtime-provisioner"]
          args: ["prepare", "--input", "/etc/induforge/runtime-binding/input.json"]
          resources: {requests: {cpu: "100m", memory: "128Mi"}, limits: {cpu: "500m", memory: "768Mi"}}
          securityContext: {allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: ["ALL"]}}
          volumeMounts:
            - {name: runtime-binding, mountPath: /etc/induforge/runtime-binding, readOnly: true}
            - {name: release, mountPath: /opt/induforge/release, readOnly: true}
            - {name: work, mountPath: /work}`, runtimeEngineImage, runtimeEngineImage, runtimeEngineImage)
		runtimeArgs = `
          command: ["runtime-engine"]
          args: ["--config", "/work/bundle/runtime-engine-config.json", "--config-root", "/opt/induforge/release/runtime-artifact", "--index", "/work/bundle/site-index.json", "--listen", "0.0.0.0:18080"]`
		runtimeMounts = `
            - {name: runtime-secrets, mountPath: /work/bundle/secrets, readOnly: true}`
		runtimeVolumes = fmt.Sprintf(`
        - name: runtime-binding
          configMap: {name: %s}
        - name: runtime-secrets
          secret:
            secretName: %s
            items:%s
        - name: bootstrap-nats
          secret:
            secretName: %s
            items:
              - {key: nats.json, path: nats.json}
        - name: bootstrap-state
          secret:
            secretName: %s
            items:
              - {key: postgres-bootstrap.json, path: postgres-bootstrap.json}`, bindingName, secretName, runtimeSecretItems, secretName, secretName)
		if workload.Engine == ServiceCompute {
			runtimeVolumes += fmt.Sprintf(`
        - name: sandbox-secret
          secret:
            secretName: %s
            items:
              - {key: sandbox-token, path: sandbox-token}`, secretName)
		}
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
            - {name: COMPUTE_SANDBOX_ARTIFACT_ROOT, value: "/opt/induforge/release"}
            - {name: COMPUTE_SANDBOX_ARTIFACT_FILE, value: "runtime-artifact.tar.zst"}
          ports: [{name: sandbox, containerPort: 18103}]
          readinessProbe: {httpGet: {path: /health, port: sandbox}, initialDelaySeconds: 3, periodSeconds: 3}
          livenessProbe: {httpGet: {path: /health, port: sandbox}, initialDelaySeconds: 15, periodSeconds: 10}
          resources: {requests: {cpu: "100m", memory: "128Mi"}, limits: {cpu: "500m", memory: "512Mi"}}
          securityContext: {runAsUser: 65532, runAsGroup: 65532, allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: ["ALL"]}}
          volumeMounts:
            - {name: release, mountPath: /opt/induforge/release, readOnly: true}
            - {name: sandbox-secret, mountPath: /var/run/induforge/secrets, readOnly: true}`, computeSandboxImage, secretName, workload.EnvironmentID, workload.NodeID, workload.DeploymentID, workload.ProjectID)
	}
	if workload.Engine == ServiceBase {
		if workload.HostPort == nil || *workload.HostPort < 1024 || *workload.HostPort > 65532 || isReservedDeploymentPort(*workload.HostPort) {
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
  # 固定 hostPort 的单节点槽无法并行调度第二个 Pod；先退出旧 Pod，避免更新永久等待端口。
  strategy: {type: RollingUpdate, rollingUpdate: {maxUnavailable: 1, maxSurge: 0}}
  selector: {matchLabels: {app.kubernetes.io/name: %q}}
  template:
    metadata:
      labels: {app.kubernetes.io/name: %q, induforge.io/release-id: %q}
      annotations: {induforge.io/runtime-binding-sha256: %q, induforge.io/release-id: %q, induforge.io/generation: %q, induforge.io/binding-revision: %q}
    spec:
      nodeSelector: {induforge.io/host-node-id: %q}
      securityContext: {runAsNonRoot: true, fsGroup: 65532, fsGroupChangePolicy: OnRootMismatch, seccompProfile: {type: RuntimeDefault}}
%s
      containers:
        - name: %s
          image: %s
          imagePullPolicy: IfNotPresent
%s
          env:
            - {name: IF_ENGINE_ROLE, valueFrom: {configMapKeyRef: {name: %s-config, key: engine-role}}}
            - {name: IF_RELEASE_ID, valueFrom: {configMapKeyRef: {name: %s-config, key: release-id}}}
            - {name: IF_RELEASE_ROOT, value: "/opt/induforge/release"}
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
          emptyDir: {sizeLimit: "768Mi"}
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
`, name, namespace, role, workload.ReleaseID, name, namespace, workload.ServiceID, name, name, workload.ReleaseID, workload.RuntimeBindingChecksum, workload.ReleaseID, fmt.Sprint(workload.Generation), fmt.Sprint(workload.BindingRevision), workload.NodeID, runtimeInit, container, image, runtimeArgs, name, name, workload.DeploymentID, workload.ProjectID, workload.EnvironmentID, workload.NodeID, hostPort, runtimeMounts, sandbox+apiSidecar, artifactRoot, runtimeVolumes, name, namespace, workload.ServiceID, name), nil
}

// renderCollectorWorkloadManifest 明确以 collector CLI 消费受信 Release 子工件、
// binding/index 和只读 Secret。WAL 暂用 Pod 生命周期 emptyDir；Pod 重建会丢失未上游
// 确认记录，后续节点持久目录交付前不承诺跨 Pod WAL 恢复。
func renderCollectorWorkloadManifest(w ProjectWorkload, namespace, name, artifactRoot string) (string, error) {
	if w.CollectorArtifactPath == "" {
		w.CollectorArtifactPath = collectorArtifactFile
	}
	return fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-config
  namespace: %s
data:
  engine-role: "collector"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels: {induforge.io/project-workload: "true", induforge.io/service-id: %q}
spec:
  replicas: 1
  strategy: {type: RollingUpdate, rollingUpdate: {maxUnavailable: 1, maxSurge: 0}}
  selector: {matchLabels: {app.kubernetes.io/name: %q}}
  template:
    metadata:
      labels: {app.kubernetes.io/name: %q, induforge.io/release-id: %q}
      annotations: {induforge.io/collector-binding-sha256: %q, induforge.io/release-id: %q, induforge.io/generation: %q}
    spec:
      nodeSelector: {induforge.io/host-node-id: %q}
      # 节点 Agent 以 induforge(1000) 创建 WAL；collector 仅以补充组访问该 0770 目录。
      securityContext: {runAsNonRoot: true, runAsUser: 65532, runAsGroup: 65532, fsGroup: 1000, fsGroupChangePolicy: OnRootMismatch, seccompProfile: {type: RuntimeDefault}}
      initContainers:
        - name: collector-artifact-prepare
          image: %s
          imagePullPolicy: IfNotPresent
          command: ["/collector_artifact_unpack"]
          args: ["--archive", "/opt/induforge/release/%s", "--target", "/work/artifact"]
          resources: {requests: {cpu: "50m", memory: "64Mi"}, limits: {cpu: "250m", memory: "256Mi"}}
          securityContext: {allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: ["ALL"]}}
          volumeMounts:
            - {name: release, mountPath: /opt/induforge/release, readOnly: true}
            - {name: work, mountPath: /work}
      containers:
        - name: collector-engine
          image: %s
          imagePullPolicy: IfNotPresent
          command: ["/industrial_collector"]
          args: ["--artifact", "/work/artifact/collector-runtime-artifact.json", "--binding", "/etc/induforge/collector/binding.json", "--index", "/etc/induforge/collector/index.json", "--listen", "0.0.0.0:18080"]
          ports: [{name: http, containerPort: 18080}]
          readinessProbe: {httpGet: {path: /health, port: http}, initialDelaySeconds: 3, periodSeconds: 3}
          livenessProbe: {httpGet: {path: /health, port: http}, initialDelaySeconds: 15, periodSeconds: 10}
          resources: {requests: {cpu: "100m", memory: "128Mi"}, limits: {cpu: "500m", memory: "512Mi"}}
          securityContext: {allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {drop: ["ALL"]}}
          volumeMounts:
            - {name: release, mountPath: /opt/induforge/release, readOnly: true}
            - {name: collector-binding, mountPath: /etc/induforge/collector, readOnly: true}
            - {name: collector-secrets, mountPath: /etc/induforge/collector/secrets, readOnly: true}
            - {name: work, mountPath: /work, readOnly: true}
            - {name: wal, mountPath: /var/lib/induforge/collector/wal}
      volumes:
        - name: release
          hostPath: {path: %q, type: Directory}
        - name: collector-binding
          configMap: {name: %s-collector-binding}
        - name: collector-secrets
          secret:
            secretName: %s
        - name: work
          emptyDir: {sizeLimit: "256Mi"}
        - name: wal
          hostPath: {path: %q, type: Directory}
---
apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  selector: {app.kubernetes.io/name: %q}
  ports: [{name: http, port: 80, targetPort: http}]
`, name, namespace, name, namespace, w.ServiceID, name, name, w.ReleaseID, w.CollectorBindingChecksum, w.ReleaseID, fmt.Sprint(w.Generation), w.NodeID, collectorEngineImage, w.CollectorArtifactPath, collectorEngineImage, artifactRoot, name, w.CollectorSecretName, filepath.Join("/var/lib/induforge/node-agent/deployments", w.DeploymentID, "state", "collector-wal"), name, namespace, name), nil
}
