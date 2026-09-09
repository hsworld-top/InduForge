package ops

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type foundationPlacement struct {
	Service, Node, Source, Claim, Operation, ObservedStatus string
	Generation, ObservedGeneration                          int64
}

// ReconcileFoundationServices 由中心统一调和各环境，不再依赖某个租户的中心 Agent。
// 每轮重新观察健康状态；写回时比较 generation，避免覆盖并发的新部署意图。
func (r *PostgreSQLRepository) ReconcileFoundationServices(ctx context.Context, kube *KubernetesProjectReconciler) error {
	rows, err := r.pool.Query(ctx, `SELECT e.id::text,e.tenant_id::text,e.desired_status FROM runtime_environments e WHERE e.deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM runtime_environment_services s JOIN host_nodes n ON n.id=s.node_id WHERE s.environment_id=e.id AND n.resource_summary->>'cleanupRequested'='true') AND (e.desired_status='deleting' OR EXISTS(SELECT 1 FROM runtime_environment_services s WHERE s.environment_id=e.id)) ORDER BY e.id`)
	if err != nil {
		return err
	}
	type environment struct{ id, tenant, desired string }
	var environments []environment
	for rows.Next() {
		var e environment
		if err = rows.Scan(&e.id, &e.tenant, &e.desired); err != nil {
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
	var failures []error
	for _, e := range environments {
		ns := "if-env-" + strings.ReplaceAll(e.id, "-", "")[:12]
		if e.desired == "deleting" {
			code, deleteErr := kube.foundationRequest(ctx, http.MethodDelete, "/api/v1/namespaces/"+ns, nil, nil)
			if deleteErr != nil {
				failures = append(failures, deleteErr)
				continue
			}
			if code == http.StatusNotFound {
				if err = r.finalizeEnvironmentDeletion(ctx, e.tenant, e.id); err != nil {
					failures = append(failures, err)
				} else {
					r.publish(e.tenant, []string{"environments", "events"}, []string{e.id}, true)
				}
			}
			continue
		}
		if err = r.reconcileEnvironmentFoundation(ctx, kube, e.id, e.tenant, ns); err != nil {
			failures = append(failures, err)
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("基础服务调和失败: %v", failures)
	}
	return nil
}

func (r *PostgreSQLRepository) reconcileEnvironmentFoundation(ctx context.Context, kube *KubernetesProjectReconciler, id, tenant, ns string) error {
	rows, err := r.pool.Query(ctx, `SELECT service_type,node_id::text,COALESCE(previous_node_id::text,''),storage_claim,operation,desired_generation,observed_generation,observed_status FROM runtime_environment_services WHERE environment_id=$1 ORDER BY service_type`, id)
	if err != nil {
		return err
	}
	var placements []foundationPlacement
	for rows.Next() {
		var p foundationPlacement
		if err = rows.Scan(&p.Service, &p.Node, &p.Source, &p.Claim, &p.Operation, &p.Generation, &p.ObservedGeneration, &p.ObservedStatus); err != nil {
			rows.Close()
			return err
		}
		placements = append(placements, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if len(placements) == 0 {
		return nil
	}
	nodes := make([]string, 0, len(placements))
	for _, p := range placements {
		nodes = append(nodes, p.Node)
	}
	nodeErrors := make(map[string]error)
	for _, node := range nodes {
		if _, checked := nodeErrors[node]; !checked {
			nodeErrors[node] = kube.EnsureHostNodeLabels(ctx, []string{node})
		}
	}
	var applyErr error
	var targets map[string]HostNodeSchedulingTarget
	if applyErr == nil {
		targets, applyErr = r.LoadHostNodeSchedulingTargets(ctx, nodes)
	}
	if applyErr == nil {
		_, applyErr = kube.foundationRequest(ctx, http.MethodPatch, "/api/v1/namespaces/"+ns, map[string]any{"apiVersion": "v1", "kind": "Namespace", "metadata": map[string]any{"name": ns, "labels": map[string]string{"induforge.io/environment-id": id}}}, nil)
	}
	if applyErr == nil {
		applyErr = kube.ensureFoundationCredentials(ctx, ns)
	}
	changed, running, failed := false, 0, 0
	var pendingNames, failedNames []string
	for _, p := range placements {
		workload := foundationWorkloadForService(p.Service)
		status, message := "pending", "等待基础服务就绪"
		serviceErr := applyErr
		if serviceErr == nil {
			serviceErr = nodeErrors[p.Node]
		}
		if serviceErr == nil && p.Operation == "migrate" {
			serviceErr = fmt.Errorf("基础服务数据迁移尚未由中心控制面支持，原服务与数据保持不变")
		}
		// 历史与时序数据共用 PostgreSQL；Traefik 是集群内置入口，只观察。
		if serviceErr == nil && workload != "traefik" && (p.Generation != p.ObservedGeneration || p.ObservedStatus != "running") {
			selector := map[string]string{"induforge.io/host-node-id": p.Node}
			if targets[p.Node].NodeSource == "built_in" {
				selector = map[string]string{"induforge.io/center-node": "true"}
			}
			if kube.imagePreparer != nil {
				serviceErr = kube.imagePreparer.EnsureNodeImages(ctx, p.Node, foundationImageReferences(p.Service))
			}
			if serviceErr == nil {
				serviceErr = kube.applyFoundationWorkload(ctx, ns, workload, p.Claim, selector)
			}
		}
		if serviceErr == nil {
			status, message, serviceErr = kube.foundationWorkloadStatus(ctx, ns, workload)
		}
		if serviceErr != nil {
			status, message = "failed", serviceErr.Error()
			var pending *imagePreparationPending
			if errors.As(serviceErr, &pending) {
				status = "pending"
			}
		}
		if status == "running" {
			running++
		} else if status == "failed" {
			failed++
			failedNames = append(failedNames, foundationDisplayName(p.Service))
		} else {
			pendingNames = append(pendingNames, foundationDisplayName(p.Service))
		}
		refs, _ := foundationResourceRefs(id, p.Service)
		raw, _ := json.Marshal(refs)
		var wasChanged bool
		err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM runtime_environment_services WHERE environment_id=$1 AND service_type=$2 AND desired_generation=$3 AND (observed_generation<>$3 OR observed_status<>$4 OR COALESCE(last_message,'')<>$5))`, id, p.Service, p.Generation, status, message).Scan(&wasChanged)
		if err != nil {
			return err
		}
		_, err = r.pool.Exec(ctx, `UPDATE runtime_environment_services SET observed_generation=$3,observed_status=$4,last_message=$5,observed_at=now(),resource_refs=CASE WHEN $4='running' AND $6::jsonb<>'null'::jsonb THEN $6::jsonb ELSE resource_refs END,updated_at=now() WHERE environment_id=$1 AND service_type=$2 AND desired_generation=$3`, id, p.Service, p.Generation, status, message, raw)
		if err != nil {
			return err
		}
		changed = changed || wasChanged
	}
	if changed {
		name, result := "正在部署："+strings.Join(pendingNames, "、"), "success"
		message := fmt.Sprintf("%d/%d 项基础服务已就绪", running, len(placements))
		if failed > 0 {
			name, result = "部署异常："+strings.Join(failedNames, "、"), "failed"
			message = fmt.Sprintf("%d 项异常，%d/%d 项已就绪", failed, running, len(placements))
		} else if running == len(placements) {
			name, message = "基础服务部署完成", "全部基础服务已就绪"
		}
		if _, err = r.pool.Exec(ctx, `INSERT INTO runtime_environment_events(tenant_id,environment_id,event_type,name,target,result,message) VALUES($1,$2,'foundation_state_changed',$3,'全部基础服务',$4,$5)`, tenant, id, name, result, message); err != nil {
			return err
		}
		r.publish(tenant, []string{"environments", "events"}, []string{id}, true)
	}
	return nil
}

func foundationWorkloadForService(service string) string {
	for _, supported := range foundationServiceTypes {
		if service == supported {
			return foundationWorkloadForLogicalType(service)
		}
	}
	return ""
}

// foundationRequest 只暴露固定 Kubernetes 资源路径，错误不回显 Secret 请求体。
func (r *KubernetesProjectReconciler) foundationRequest(ctx context.Context, method, path string, value, out any) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var raw []byte
	var err error
	if value != nil {
		raw, err = json.Marshal(value)
		if err != nil {
			return 0, err
		}
	}
	if method == http.MethodPatch {
		path += "?fieldManager=induforge-center&force=true"
	}
	req, err := http.NewRequestWithContext(ctx, method, r.endpoint+path, bytes.NewReader(raw))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Content-Type", "application/json")
	if method == http.MethodPatch {
		req.Header.Set("Content-Type", "application/apply-patch+yaml")
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 && (method == http.MethodGet || method == http.MethodDelete) {
		return resp.StatusCode, nil
	}
	if resp.StatusCode/100 != 2 {
		return resp.StatusCode, fmt.Errorf("Kubernetes 基础服务 %s %s: HTTP %d", method, strings.Split(path, "?")[0], resp.StatusCode)
	}
	if out != nil {
		err = json.NewDecoder(resp.Body).Decode(out)
	}
	return resp.StatusCode, err
}

func (r *KubernetesProjectReconciler) ensureFoundationCredentials(ctx context.Context, ns string) error {
	data, exists, err := r.GetSecret(ctx, ns, "foundation-credentials")
	if err != nil {
		return err
	}
	if exists {
		for _, key := range []string{"postgres-password", "nats-token", "redis.conf", "nats.conf", "object-access-key", "object-secret-key", "s3.json", "shared-password"} {
			if data[key] == "" {
				return fmt.Errorf("环境基础服务凭据缺少 %s，拒绝覆盖已有凭据", key)
			}
		}
		return nil
	}
	secretBytes := make([]byte, 32)
	if _, err = rand.Read(secretBytes); err != nil {
		return err
	}
	secret := hex.EncodeToString(secretBytes)
	access := "if" + secret[:20]
	objectConfig, _ := json.Marshal(map[string]any{"identities": []any{map[string]any{"name": "induforge-runtime", "credentials": []any{map[string]string{"accessKey": access, "secretKey": secret}}, "actions": []string{"Admin", "Read", "List", "Tagging", "Write"}}}})
	data = map[string]string{"postgres-password": secret, "nats-token": secret, "shared-password": secret, "redis.conf": "requirepass " + secret + "\nappendonly yes\n", "nats.conf": "authorization { token: " + secret + " }\nmax_payload: 2097152\njetstream { store_dir: /data }\n", "object-access-key": access, "object-secret-key": secret, "s3.json": string(objectConfig)}
	// 并发中心实例只允许首次创建，冲突沿用胜出者的 Secret，不能轮换已启动服务的密码。
	code, err := r.foundationRequest(ctx, http.MethodPost, "/api/v1/namespaces/"+ns+"/secrets", map[string]any{"apiVersion": "v1", "kind": "Secret", "metadata": map[string]string{"name": "foundation-credentials", "namespace": ns}, "type": "Opaque", "stringData": data}, nil)
	if code == http.StatusConflict {
		return nil
	}
	return err
}

func (r *KubernetesProjectReconciler) applyFoundationWorkload(ctx context.Context, ns, name, claim string, selector map[string]string) error {
	images := map[string]string{"postgres": "timescale/timescaledb:2.26.4-pg16", "redis": "redis:7.2-alpine", "emqx": "emqx/emqx:5.6.1", "nats": "nats:2.12.8-alpine", "object": "chrislusf/seaweedfs:3.85", "nginx": "nginx:1.28-alpine"}
	ports := map[string]int{"postgres": 5432, "redis": 6379, "emqx": 1883, "nats": 4222, "object": 8333, "nginx": 80}
	paths := map[string]string{"postgres": "/var/lib/postgresql/data", "redis": "/data", "emqx": "/opt/emqx/data", "nats": "/data", "object": "/data"}
	if images[name] == "" {
		return fmt.Errorf("未知基础服务 %s", name)
	}
	labels := map[string]string{"app": name, "induforge.io/foundation": "true", "induforge.io/service": name}
	container := map[string]any{"name": name, "image": images[name], "imagePullPolicy": "Never", "ports": []any{map[string]int{"containerPort": ports[name]}}, "readinessProbe": map[string]any{"tcpSocket": map[string]int{"port": ports[name]}, "initialDelaySeconds": 5, "periodSeconds": 5}, "livenessProbe": map[string]any{"tcpSocket": map[string]int{"port": ports[name]}, "initialDelaySeconds": 30, "periodSeconds": 10}}
	secretEnv := func(n, key string) any {
		return map[string]any{"name": n, "valueFrom": map[string]any{"secretKeyRef": map[string]string{"name": "foundation-credentials", "key": key}}}
	}
	if name == "postgres" {
		container["env"] = []any{secretEnv("POSTGRES_PASSWORD", "postgres-password"), map[string]string{"name": "POSTGRES_DB", "value": "induforge"}}
	}
	if name == "emqx" {
		container["env"] = []any{secretEnv("EMQX_DASHBOARD__DEFAULT_PASSWORD", "shared-password"), map[string]string{"name": "EMQX_AUTHENTICATION__1__MECHANISM", "value": "password_based"}, map[string]string{"name": "EMQX_AUTHENTICATION__1__BACKEND", "value": "built_in_database"}, map[string]string{"name": "EMQX_AUTHENTICATION__1__USER_ID_TYPE", "value": "username"}, map[string]string{"name": "EMQX_AUTHENTICATION__1__PASSWORD_HASH_ALGORITHM__NAME", "value": "sha256"}, map[string]string{"name": "EMQX_AUTHENTICATION__1__PASSWORD_HASH_ALGORITHM__SALT_POSITION", "value": "disable"}}
	}
	pod := map[string]any{"nodeSelector": selector, "containers": []any{container}}
	kind, resource := "StatefulSet", "statefulsets"
	if name == "nginx" {
		kind, resource = "Deployment", "deployments"
	} else {
		if claim == "" {
			claim = "data-" + name + "-0"
		}
		pvc := map[string]any{"apiVersion": "v1", "kind": "PersistentVolumeClaim", "metadata": map[string]string{"name": claim, "namespace": ns}, "spec": map[string]any{"accessModes": []string{"ReadWriteOnce"}, "resources": map[string]any{"requests": map[string]string{"storage": "10Gi"}}}}
		if _, err := r.foundationRequest(ctx, http.MethodPatch, "/api/v1/namespaces/"+ns+"/persistentvolumeclaims/"+claim, pvc, nil); err != nil {
			return err
		}
		mounts := []any{map[string]string{"name": "data", "mountPath": paths[name]}}
		volumes := []any{map[string]any{"name": "data", "persistentVolumeClaim": map[string]string{"claimName": claim}}}
		key, configPath := "", ""
		switch name {
		case "redis":
			key, configPath = "redis.conf", "/etc/redis/redis.conf"
			container["command"] = []string{"redis-server", configPath}
		case "nats":
			key, configPath = "nats.conf", "/etc/nats/nats.conf"
			container["args"] = []string{"-c", configPath}
		case "object":
			key, configPath = "s3.json", "/etc/seaweedfs/s3.json"
			container["args"] = []string{"server", "-dir=/data", "-s3", "-s3.port=8333", "-s3.config=" + configPath}
		}
		if key != "" {
			mounts = append(mounts, map[string]string{"name": "config", "mountPath": configPath, "subPath": key})
			volumes = append(volumes, map[string]any{"name": "config", "secret": map[string]any{"secretName": "foundation-credentials", "items": []any{map[string]string{"key": key, "path": key}}}})
		}
		container["volumeMounts"] = mounts
		pod["volumes"] = volumes
	}
	spec := map[string]any{"replicas": 1, "selector": map[string]any{"matchLabels": map[string]string{"app": name}}, "template": map[string]any{"metadata": map[string]any{"labels": labels}, "spec": pod}}
	if kind == "StatefulSet" {
		spec["serviceName"] = name
	}
	obj := map[string]any{"apiVersion": "apps/v1", "kind": kind, "metadata": map[string]any{"name": name, "namespace": ns, "labels": labels}, "spec": spec}
	if _, err := r.foundationRequest(ctx, http.MethodPatch, "/apis/apps/v1/namespaces/"+ns+"/"+resource+"/"+name, obj, nil); err != nil {
		return err
	}
	servicePorts := []any{map[string]any{"name": "main", "port": ports[name], "targetPort": ports[name]}}
	if name == "emqx" {
		servicePorts = append(servicePorts, map[string]any{"name": "dashboard", "port": 18083, "targetPort": 18083})
	}
	service := map[string]any{"apiVersion": "v1", "kind": "Service", "metadata": map[string]string{"name": name, "namespace": ns}, "spec": map[string]any{"selector": map[string]string{"app": name}, "ports": servicePorts}}
	_, err := r.foundationRequest(ctx, http.MethodPatch, "/api/v1/namespaces/"+ns+"/services/"+name, service, nil)
	if err == nil && name == "emqx" {
		err = r.ensureFoundationMQTTUser(ctx, ns, selector)
	}
	return err
}

func (r *KubernetesProjectReconciler) foundationWorkloadStatus(ctx context.Context, ns, name string) (string, string, error) {
	resource := "statefulsets"
	if name == "nginx" || name == "traefik" {
		resource = "deployments"
	}
	if name == "traefik" {
		ns = "kube-system"
	}
	var result struct {
		Metadata struct{ Generation int64 }
		Spec     struct {
			Replicas int
			Template struct {
				Spec struct{ NodeSelector map[string]string }
			}
		}
		Status struct {
			ObservedGeneration int64
			ReadyReplicas      int
			UpdatedReplicas    int
			Replicas           int
			Conditions         []struct{ Type, Status, Reason, Message string }
		}
	}
	code, err := r.foundationRequest(ctx, http.MethodGet, "/apis/apps/v1/namespaces/"+ns+"/"+resource+"/"+name, nil, &result)
	if err != nil {
		return "", "", err
	}
	if code == 404 {
		return "pending", "等待创建基础服务", nil
	}
	for _, c := range result.Status.Conditions {
		if c.Type == "ReplicaFailure" && c.Status == "True" {
			return "failed", c.Message, nil
		}
	}
	// Deployment 必须完成当前版本替换，旧节点仍健康不代表新节点切换成功。
	rolloutReady := resource != "deployments" || (result.Status.UpdatedReplicas >= result.Spec.Replicas && result.Status.Replicas == result.Status.UpdatedReplicas)
	if result.Status.ObservedGeneration >= result.Metadata.Generation && result.Status.ReadyReplicas > 0 && rolloutReady {
		if name == "emqx" {
			var job struct {
				Status struct {
					Succeeded, Failed int
					Conditions        []struct{ Type, Status, Message string }
				}
			}
			_, err = r.foundationRequest(ctx, http.MethodGet, "/apis/batch/v1/namespaces/"+ns+"/jobs/emqx-runtime-credential", nil, &job)
			if err != nil {
				return "", "", err
			}
			for _, condition := range job.Status.Conditions {
				if condition.Type == "Failed" && condition.Status == "True" {
					return "failed", "消息服务账户初始化失败: " + condition.Message, nil
				}
			}
			if job.Status.Succeeded < 1 {
				return "pending", "等待消息服务账户初始化", nil
			}
		}
		return "running", "", nil
	}
	var pods struct {
		Items []struct {
			Metadata struct{ DeletionTimestamp *string }
			Spec     struct{ NodeSelector map[string]string }
			Status   struct {
				ContainerStatuses []struct {
					State struct {
						Waiting *struct{ Reason, Message string }
					}
				}
				Conditions []struct{ Type, Status, Reason, Message string }
			}
		}
	}
	_, err = r.foundationRequest(ctx, http.MethodGet, "/api/v1/namespaces/"+ns+"/pods?labelSelector=app%3D"+name, nil, &pods)
	if err != nil {
		return "", "", err
	}
	for _, pod := range pods.Items {
		// 切换期间旧节点的退出中实例及旧调度目标不参与本次诊断。
		if pod.Metadata.DeletionTimestamp != nil {
			continue
		}
		matches := true
		for key, value := range result.Spec.Template.Spec.NodeSelector {
			if pod.Spec.NodeSelector[key] != value {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		for _, container := range pod.Status.ContainerStatuses {
			if waiting := container.State.Waiting; waiting != nil {
				switch waiting.Reason {
				case "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull", "ErrImageNeverPull", "CreateContainerConfigError":
					return "failed", waiting.Reason + ": " + waiting.Message, nil
				}
			}
		}
		for _, condition := range pod.Status.Conditions {
			if condition.Type == "PodScheduled" && condition.Status == "False" {
				return "pending", condition.Message, nil
			}
		}
	}
	if len(pods.Items) == 0 {
		return "pending", "等待执行", nil
	}
	return "pending", "启动中", nil
}

func (r *KubernetesProjectReconciler) ensureFoundationMQTTUser(ctx context.Context, ns string, selector map[string]string) error {
	path := "/apis/batch/v1/namespaces/" + ns + "/jobs/emqx-runtime-credential"
	code, err := r.foundationRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil || code != 404 {
		return err
	}
	// EMQX 的内置认证器需要 API 引导用户；固定脚本只使用 Secret 注入的凭据。
	script := `set -eu
until response="$(wget -qO- --header='Content-Type: application/json' --post-data="{\"username\":\"admin\",\"password\":\"${EMQX_PASSWORD}\"}" http://emqx:18083/api/v5/login 2>/dev/null)"; do sleep 2; done
token="$(printf '%s' "$response" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')"
test -n "$token"
if wget -qO- --header="Authorization: Bearer $token" --header='Content-Type: application/json' --post-data="{\"user_id\":\"induforge\",\"password\":\"${EMQX_PASSWORD}\",\"is_superuser\":true}" http://emqx:18083/api/v5/authentication/password_based%3Abuilt_in_database/users >/dev/null 2>&1; then exit 0; fi
wget -qO- --header="Authorization: Bearer $token" http://emqx:18083/api/v5/authentication/password_based%3Abuilt_in_database/users/induforge >/dev/null`
	job := map[string]any{"apiVersion": "batch/v1", "kind": "Job", "metadata": map[string]string{"name": "emqx-runtime-credential", "namespace": ns}, "spec": map[string]any{"backoffLimit": 3, "activeDeadlineSeconds": 300, "template": map[string]any{"spec": map[string]any{"nodeSelector": selector, "restartPolicy": "OnFailure", "containers": []any{map[string]any{"name": "bootstrap", "image": "nginx:1.28-alpine", "imagePullPolicy": "Never", "command": []string{"/bin/sh", "-c", script}, "env": []any{map[string]any{"name": "EMQX_PASSWORD", "valueFrom": map[string]any{"secretKeyRef": map[string]string{"name": "foundation-credentials", "key": "shared-password"}}}}}}}}}}
	_, err = r.foundationRequest(ctx, http.MethodPatch, path, job, nil)
	return err
}

func foundationDisplayName(service string) string {
	names := map[string]string{"if_realtime": "实时库", "if_history": "历史库", "if_timeseries": "时序库", "if_message": "消息库", "if_object": "对象存储", "nats_jetstream": "消息总线", "nginx": "页面服务", "traefik": "访问路由"}
	if name := names[service]; name != "" {
		return name
	}
	return "基础服务"
}
