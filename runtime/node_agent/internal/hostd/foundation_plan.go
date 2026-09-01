package hostd

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const FoundationSchemaVersion = "induforge.foundation-plan.v1"

var foundationWorkloads = []string{"postgres", "redis", "emqx", "nats", "object", "nginx"}

// FoundationPlan 只允许中心选择固定工作负载落在哪个已登记节点。镜像、命令、
// 挂载和权限均由 Hostd 内置，控制面不能把该接口变成通用容器执行能力。
type FoundationPlan struct {
	SchemaVersion     string            `json:"schemaVersion"`
	Generation        int64             `json:"generation"`
	EnvironmentID     string            `json:"environmentId"`
	NodeID            string            `json:"nodeId"`
	Assignments       map[string]string `json:"assignments"`
	SourceAssignments map[string]string `json:"sourceAssignments,omitempty"`
	Claims            map[string]string `json:"claims,omitempty"`
	SourceClaims      map[string]string `json:"sourceClaims,omitempty"`
	Operation         string            `json:"operation,omitempty"`
}

type FoundationState struct {
	SchemaVersion string                   `json:"schemaVersion"`
	Generation    int64                    `json:"generation"`
	EnvironmentID string                   `json:"environmentId"`
	NodeID        string                   `json:"nodeId"`
	ObservedState string                   `json:"observedState"`
	Message       string                   `json:"message,omitempty"`
	Services      []FoundationServiceState `json:"services,omitempty"`
}

type FoundationDeleteRequest struct {
	EnvironmentID string `json:"environmentId"`
	NodeID        string `json:"nodeId"`
}

func (request FoundationDeleteRequest) Validate() error {
	if !uuidPattern.MatchString(strings.ToLower(request.EnvironmentID)) || !uuidPattern.MatchString(strings.ToLower(request.NodeID)) {
		return errors.New("运行环境或中心节点 ID 无效")
	}
	return nil
}

type FoundationServiceState struct {
	Workload string `json:"workload"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

func (plan FoundationPlan) Validate() error {
	if plan.SchemaVersion != FoundationSchemaVersion || plan.Generation < 1 || !uuidPattern.MatchString(strings.ToLower(plan.EnvironmentID)) || !uuidPattern.MatchString(strings.ToLower(plan.NodeID)) {
		return errors.New("基础服务计划元数据无效")
	}
	if len(plan.Assignments) != len(foundationWorkloads) {
		return errors.New("基础服务工作负载分配不完整")
	}
	for _, workload := range foundationWorkloads {
		if !uuidPattern.MatchString(strings.ToLower(plan.Assignments[workload])) {
			return fmt.Errorf("基础服务 %s 的目标节点无效", workload)
		}
	}
	if plan.Operation != "" && plan.Operation != "apply" && plan.Operation != "migrate" {
		return errors.New("基础服务计划操作无效")
	}
	if plan.Operation == "migrate" && len(plan.SourceAssignments) == 0 {
		return errors.New("基础服务迁移缺少原节点信息")
	}
	return nil
}

func foundationNamespace(environmentID string) string {
	return "if-env-" + strings.ReplaceAll(strings.ToLower(environmentID), "-", "")[:12]
}

func foundationNodeName(nodeID string) string {
	return "if-" + strings.ReplaceAll(strings.ToLower(nodeID), "-", "")[:12]
}

func (plan FoundationPlan) RenderManifest(secret string) string {
	namespace := foundationNamespace(plan.EnvironmentID)
	objectAccessKey := "if" + secret[:20]
	objectConfig := fmt.Sprintf(`{"identities":[{"name":"induforge-runtime","credentials":[{"accessKey":"%s","secretKey":"%s"}],"actions":["Admin","Read","List","Tagging","Write"]}]}`, objectAccessKey, secret)
	parts := []string{
		"apiVersion: v1\nkind: Namespace\nmetadata:\n  name: " + namespace,
		// Center 控制面仅被授予本环境命名空间的项目工作负载权限；不能取得
		// ClusterRole 或 K3s 管理员凭据，因此无法越过已登记的运行环境。
		"apiVersion: rbac.authorization.k8s.io/v1\nkind: Role\nmetadata:\n  name: induforge-project-reconciler\n  namespace: " + namespace + "\nrules:\n  - apiGroups: [\"apps\"]\n    resources: [\"deployments\"]\n    verbs: [\"get\", \"list\", \"watch\", \"create\", \"update\", \"patch\", \"delete\"]\n  - apiGroups: [\"\"]\n    resources: [\"services\", \"configmaps\", \"secrets\"]\n    verbs: [\"get\", \"list\", \"watch\", \"create\", \"update\", \"patch\", \"delete\"]\n  - apiGroups: [\"\"]\n    resources: [\"pods\", \"events\"]\n    verbs: [\"get\", \"list\", \"watch\"]",
		"apiVersion: rbac.authorization.k8s.io/v1\nkind: RoleBinding\nmetadata:\n  name: induforge-project-reconciler\n  namespace: " + namespace + "\nsubjects:\n  - kind: ServiceAccount\n    name: center-control\n    namespace: induforge-system\nroleRef:\n  apiGroup: rbac.authorization.k8s.io\n  kind: Role\n  name: induforge-project-reconciler",
		"apiVersion: v1\nkind: Secret\nmetadata:\n  name: foundation-credentials\n  namespace: " + namespace + "\ntype: Opaque\nstringData:\n  postgres-password: " + quoteYAML(secret) + "\n  nats-token: " + quoteYAML(secret) + "\n  redis.conf: " + quoteYAML("requirepass "+secret+"\nappendonly yes\n") + "\n  nats.conf: " + quoteYAML("authorization { token: "+secret+" }\njetstream { store_dir: /data }\n") + "\n  object-access-key: " + quoteYAML(objectAccessKey) + "\n  object-secret-key: " + quoteYAML(secret) + "\n  s3.json: " + quoteYAML(objectConfig) + "\n  shared-password: " + quoteYAML(secret),
		renderStatefulSet(namespace, "postgres", "timescale/timescaledb:2.26.4-pg16", plan.Assignments["postgres"], plan.Claims["postgres"], []string{"containerPort: 5432"}, []string{"name: POSTGRES_PASSWORD\n              valueFrom:\n                secretKeyRef:\n                  name: foundation-credentials\n                  key: postgres-password", "name: POSTGRES_DB\n              value: induforge"}, "/var/lib/postgresql/data"),
		renderStatefulSet(namespace, "redis", "redis:7.2-alpine", plan.Assignments["redis"], plan.Claims["redis"], []string{"containerPort: 6379"}, nil, "/data"),
		renderStatefulSet(namespace, "emqx", "emqx/emqx:5.6.1", plan.Assignments["emqx"], plan.Claims["emqx"], []string{"containerPort: 1883"}, []string{
			"name: EMQX_DASHBOARD__DEFAULT_PASSWORD\n              valueFrom:\n                secretKeyRef:\n                  name: foundation-credentials\n                  key: shared-password",
			"name: EMQX_AUTHENTICATION__1__MECHANISM\n              value: password_based",
			"name: EMQX_AUTHENTICATION__1__BACKEND\n              value: built_in_database",
			"name: EMQX_AUTHENTICATION__1__USER_ID_TYPE\n              value: username",
			"name: EMQX_AUTHENTICATION__1__PASSWORD_HASH_ALGORITHM__NAME\n              value: sha256",
			"name: EMQX_AUTHENTICATION__1__PASSWORD_HASH_ALGORITHM__SALT_POSITION\n              value: disable",
		}, "/opt/emqx/data"),
		renderStatefulSet(namespace, "nats", "nats:2.12.8-alpine", plan.Assignments["nats"], plan.Claims["nats"], []string{"containerPort: 4222"}, nil, "/data"),
		renderStatefulSet(namespace, "object", "chrislusf/seaweedfs:3.85", plan.Assignments["object"], plan.Claims["object"], []string{"containerPort: 8333"}, nil, "/data"),
		renderDeployment(namespace, "nginx", "nginx:1.28-alpine", plan.Assignments["nginx"], "80"),
		renderEMQXCredentialJob(namespace, plan.Assignments["emqx"]),
	}
	sort.Strings(parts[4:])
	return strings.Join(parts, "\n---\n") + "\n"
}

func renderStatefulSet(namespace, name, image, nodeID, claim string, ports, env []string, mountPath string) string {
	portBlock := ""
	if len(ports) > 0 {
		portBlock = "\n          ports:\n            - " + strings.Join(ports, "\n            - ")
	}
	envBlock := ""
	if len(env) > 0 {
		envBlock = "\n          env:\n            - " + strings.Join(env, "\n            - ")
	}
	commandBlock := ""
	volumeExtra := ""
	if name == "redis" {
		commandBlock = "\n          command: [\"redis-server\", \"/etc/redis/redis.conf\"]"
		volumeExtra = "\n            - name: config\n              mountPath: /etc/redis/redis.conf\n              subPath: redis.conf"
	} else if name == "nats" {
		commandBlock = "\n          args: [\"-c\", \"/etc/nats/nats.conf\"]"
		volumeExtra = "\n            - name: config\n              mountPath: /etc/nats/nats.conf\n              subPath: nats.conf"
	} else if name == "object" {
		commandBlock = "\n          args: [\"server\", \"-dir=/data\", \"-s3\", \"-s3.port=8333\", \"-s3.config=/etc/seaweedfs/s3.json\"]"
		volumeExtra = "\n            - name: config\n              mountPath: /etc/seaweedfs/s3.json\n              subPath: s3.json"
	}
	volumes := ""
	if name == "redis" || name == "nats" || name == "object" {
		configKey, configPath := "redis.conf", "redis.conf"
		if name == "nats" {
			configKey, configPath = "nats.conf", "nats.conf"
		} else if name == "object" {
			configKey, configPath = "s3.json", "s3.json"
		}
		volumes = "\n      volumes:\n        - name: config\n          secret:\n            secretName: foundation-credentials\n            items:\n              - key: redis.conf\n                path: redis.conf"
		volumes = strings.ReplaceAll(strings.ReplaceAll(volumes, "key: redis.conf", "key: "+configKey), "path: redis.conf", "path: "+configPath)
	}
	servicePorts := "    - {port: " + servicePort(name) + ", targetPort: " + servicePort(name) + "}"
	if name == "emqx" {
		servicePorts = "    - {name: mqtt, port: 1883, targetPort: 1883}\n    - {name: dashboard, port: 18083, targetPort: 18083}"
	}
	storage := "\n  volumeClaimTemplates:\n    - metadata: {name: data}\n      spec:\n        accessModes: [ReadWriteOnce]\n        resources:\n          requests: {storage: 10Gi}"
	if claim != "" {
		storage = ""
		claimVolume := "\n        - name: data\n          persistentVolumeClaim:\n            claimName: " + claim
		if volumes == "" {
			volumes = "\n      volumes:" + claimVolume
		} else {
			volumes += claimVolume
		}
	}
	return fmt.Sprintf("apiVersion: apps/v1\nkind: StatefulSet\nmetadata:\n  name: %s\n  namespace: %s\n  labels: {induforge.io/foundation: \"true\", induforge.io/service: %s}\nspec:\n  serviceName: %s\n  replicas: 1\n  selector:\n    matchLabels: {app: %s}\n  template:\n    metadata:\n      labels: {app: %s, induforge.io/foundation: \"true\", induforge.io/service: %s}\n    spec:\n      nodeSelector:\n        kubernetes.io/hostname: %s\n      containers:\n        - name: %s\n          image: %s\n          imagePullPolicy: Never%s%s%s\n          readinessProbe:\n            tcpSocket: {port: %s}\n            initialDelaySeconds: 5\n            periodSeconds: 5\n            failureThreshold: 6\n          livenessProbe:\n            tcpSocket: {port: %s}\n            initialDelaySeconds: 30\n            periodSeconds: 10\n            failureThreshold: 6\n          volumeMounts:\n            - name: data\n              mountPath: %s%s%s%s\n---\napiVersion: v1\nkind: Service\nmetadata: {name: %s, namespace: %s}\nspec:\n  selector: {app: %s}\n  ports:\n%s\n", name, namespace, name, name, name, name, name, foundationNodeName(nodeID), name, image, commandBlock, portBlock, envBlock, servicePort(name), servicePort(name), mountPath, volumeExtra, volumes, storage, name, namespace, name, servicePorts)
}

// EMQX 5.6 可以用静态配置创建内置数据库认证器，但该版本不支持在配置中
// 引导首个 MQTT 用户。这个幂等 Job 只通过集群内 Dashboard API 写入固定运行
// 账户；重复应用时会确认账户已经存在，既不开放匿名连接，也不暴露凭据。
func renderEMQXCredentialJob(namespace, nodeID string) string {
	script := `set -eu
until response="$(wget -qO- --header='Content-Type: application/json' --post-data="{\"username\":\"admin\",\"password\":\"${EMQX_PASSWORD}\"}" http://emqx:18083/api/v5/login 2>/dev/null)"; do sleep 2; done
token="$(printf '%s' "$response" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')"
test -n "$token"
if wget -qO- --header="Authorization: Bearer $token" --header='Content-Type: application/json' --post-data="{\"user_id\":\"induforge\",\"password\":\"${EMQX_PASSWORD}\",\"is_superuser\":true}" http://emqx:18083/api/v5/authentication/password_based%3Abuilt_in_database/users >/dev/null 2>&1; then exit 0; fi
wget -qO- --header="Authorization: Bearer $token" http://emqx:18083/api/v5/authentication/password_based%3Abuilt_in_database/users/induforge >/dev/null`
	return fmt.Sprintf("apiVersion: batch/v1\nkind: Job\nmetadata:\n  name: emqx-runtime-credential\n  namespace: %s\n  labels: {induforge.io/component: credential-bootstrap}\nspec:\n  backoffLimit: 10\n  template:\n    metadata:\n      labels: {induforge.io/component: credential-bootstrap}\n    spec:\n      restartPolicy: OnFailure\n      nodeSelector:\n        kubernetes.io/hostname: %s\n      containers:\n        - name: bootstrap\n          image: nginx:1.28-alpine\n          imagePullPolicy: Never\n          command: [/bin/sh, -c]\n          args:\n            - %s\n          env:\n            - name: EMQX_PASSWORD\n              valueFrom:\n                secretKeyRef:\n                  name: foundation-credentials\n                  key: shared-password\n", namespace, foundationNodeName(nodeID), quoteYAML(script))
}

func renderDeployment(namespace, name, image, nodeID, port string) string {
	args := ""
	if name == "nats" {
		args = "\n          args: [\"-c\", \"/etc/nats/nats.conf\"]"
	} else if name == "object" {
		args = "\n          args: [\"server\", \"-dir=/data\", \"-s3\", \"-s3.port=8333\"]"
	}
	return fmt.Sprintf("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: %s\n  namespace: %s\n  labels: {induforge.io/foundation: \"true\", induforge.io/service: %s}\nspec:\n  replicas: 1\n  selector:\n    matchLabels: {app: %s}\n  template:\n    metadata:\n      labels: {app: %s, induforge.io/foundation: \"true\", induforge.io/service: %s}\n    spec:\n      nodeSelector:\n        kubernetes.io/hostname: %s\n      containers:\n        - name: %s\n          image: %s\n          imagePullPolicy: Never%s\n          ports: [{containerPort: %s}]\n          readinessProbe:\n            httpGet: {path: /, port: %s}\n            initialDelaySeconds: 3\n            periodSeconds: 5\n          livenessProbe:\n            httpGet: {path: /, port: %s}\n            initialDelaySeconds: 20\n            periodSeconds: 10\n---\napiVersion: v1\nkind: Service\nmetadata: {name: %s, namespace: %s}\nspec:\n  selector: {app: %s}\n  ports: [{port: %s, targetPort: %s}]\n", name, namespace, name, name, name, name, foundationNodeName(nodeID), name, image, args, port, port, port, name, namespace, name, port, port)
}

func servicePort(name string) string {
	switch name {
	case "postgres":
		return "5432"
	case "redis":
		return "6379"
	case "emqx":
		return "1883"
	case "nats":
		return "4222"
	case "object":
		return "8333"
	default:
		return "80"
	}
}
