package ops

import (
	"strings"
	"testing"
)

func TestRenderProjectWorkloadManifestSeparatesRolesAndHostPort(t *testing.T) {
	port := 17800
	digest := "sha256:" + strings.Repeat("a", 64)
	base, err := RenderProjectWorkloadManifest(ProjectWorkload{EnvironmentID: testEnvironmentID, ProjectID: testProjectID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", NodeID: testNodeID, Engine: ServiceBase, ReleaseID: testVersionID, ReleaseDigest: digest, Generation: 2, HostPort: &port, RuntimeNATSEndpoint: "nats://nats:4222"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(base, "\t") {
		t.Fatalf("base manifest must not contain YAML tab indentation: %q", base)
	}
	for _, expected := range []string{"if-env-666666666666", "if-project-999999999999-base", "type: LoadBalancer", "port: 17800", "induforge.io/host-node-id", "maxUnavailable: 0, maxSurge: 1", "allowPrivilegeEscalation: false", "IF_RELEASE_ROOT", "IF_WORK_ROOT", `IF_PROJECT_ID, value: "` + testProjectID + `"`, "mountPath: /opt/induforge/release", "kind: Service", "image: induforge/project-gateway:1.0.2", "image: induforge/project-runtime-api:1.0.1", "image: induforge/runtime-engine:1.0.31", "name: runtime-writer", `"--project-id", "` + testProjectID + `"`, `"--account-id", "` + runtimeAccountID(testProjectID, "99999999-9999-4999-8999-999999999999") + `"`, `exec: {command: ["/usr/local/bin/runtime-api", "healthcheck", "--url", "http://127.0.0.1:18081/health"]}`, "name: runtime-provision-nats", "name: runtime-provision-state", "name: runtime-api-artifact-prepare", "name: runtime-binding-prepare", "name: runtime-api-secrets", "runtime-api-tokens.json", "name: runtime-viewer-token", "{key: runtime-api-token, path: token}", "mountPath: /var/run/induforge/runtime-viewer", "imagePullPolicy: IfNotPresent", `path: "/var/lib/induforge/node-agent/deployments/99999999-9999-4999-8999-999999999999/release/releases/sha256-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`} {
		if !strings.Contains(base, expected) {
			t.Fatalf("base manifest missing %q: %s", expected, base)
		}
	}
	if strings.Contains(base, "hostPort:") {
		t.Fatalf("base 入口必须由稳定 ServiceLB 承载，不得绑定 Pod hostPort:\n%s", base)
	}
	if !strings.Contains(base, `"--listen", "0.0.0.0:18082"`) || strings.Contains(base, `"--listen", "127.0.0.1:18082"`) {
		t.Fatalf("writer 健康端口必须可由 kubelet 通过 Pod IP 探测:\n%s", base)
	}
	if strings.Count(base, "mountPath: /var/run/induforge/runtime-viewer") != 1 {
		t.Fatalf("viewer token must only be mounted once:\n%s", base)
	}
	apiStart := strings.Index(base, "- name: runtime-api\n")
	if apiStart < 0 {
		t.Fatalf("runtime-api sidecar missing:\n%s", base)
	}
	apiEnd := strings.Index(base[apiStart:], "\n      volumes:")
	if apiEnd < 0 || strings.Contains(base[apiStart:apiStart+apiEnd], "runtime-viewer-token") {
		t.Fatalf("viewer token must not be mounted by Runtime API:\n%s", base)
	}
	compute, err := RenderProjectWorkloadManifest(ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", NodeID: testNodeID, Engine: ServiceCompute, ReleaseID: testVersionID, ReleaseDigest: digest, Generation: 2})
	if err != nil || !strings.Contains(compute, "IF_ENGINE_ROLE") || !strings.Contains(compute, `"compute"`) || strings.Contains(compute, "hostPort:") {
		t.Fatalf("compute role/port isolation failed: %v\n%s", err, compute)
	}
	if !strings.Contains(compute, "- {name: work, mountPath: /work, readOnly: true}") {
		t.Fatalf("compute 主容器必须以只读方式消费 init 生成的运行 bundle:\n%s", compute)
	}
	for _, expected := range []string{"initContainers:", "name: runtime-provision-nats", "name: runtime-provision-state", "name: runtime-binding-prepare", `args: ["provision-nats", "--input", "/etc/induforge/runtime-binding/input.json", "--credentials", "/var/run/induforge/bootstrap/nats.json"]`, `args: ["provision-state", "--input", "/etc/induforge/runtime-binding/input.json", "--credentials", "/var/run/induforge/bootstrap/postgres-bootstrap.json"]`, `args: ["prepare", "--input", "/etc/induforge/runtime-binding/input.json"]`, `"--config-root", "/work/bundle"`, "name: runtime-secrets", "name: bootstrap-nats", "name: bootstrap-state", "postgres-bootstrap.json", "mountPath: /work/bundle/secrets", "name: runtime-binding", "emptyDir: {sizeLimit: \"768Mi\"}", "induforge.io/runtime-binding-sha256"} {
		if !strings.Contains(compute, expected) {
			t.Fatalf("compute binding manifest missing %q:\n%s", expected, compute)
		}
	}
	if !(strings.Index(compute, "name: runtime-provision-nats") < strings.Index(compute, "name: runtime-provision-state") && strings.Index(compute, "name: runtime-provision-state") < strings.Index(compute, "name: runtime-binding-prepare")) {
		t.Fatalf("init 顺序错误:\n%s", compute)
	}
	prepareStart := strings.Index(compute, "name: runtime-binding-prepare")
	mainStart := strings.Index(compute, "containers:")
	prepare := compute[prepareStart:mainStart]
	if strings.Contains(prepare, "bootstrap-") || !strings.Contains(prepare, "mountPath: /opt/induforge/release") {
		t.Fatalf("prepare 挂载权限错误:\n%s", prepare)
	}
	natsStart := strings.Index(compute, "name: runtime-provision-nats")
	stateStart := strings.Index(compute, "name: runtime-provision-state")
	nats := compute[natsStart:stateStart]
	state := compute[stateStart:prepareStart]
	if strings.Contains(nats, "postgres-bootstrap.json") || strings.Contains(nats, "runtime-secrets") || strings.Contains(state, "nats.json") || strings.Contains(state, "runtime-secrets") {
		t.Fatalf("bootstrap Secret 未最小投影:\nnats=%s\nstate=%s", nats, state)
	}
	alarm, err := RenderProjectWorkloadManifest(ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "cccccccc-cccc-4ccc-8ccc-cccccccccccc", NodeID: testNodeID, Engine: ServiceAlarm, ReleaseID: testVersionID, ReleaseDigest: digest, Generation: 2})
	if err != nil || !strings.Contains(alarm, `"alarm"`) || strings.Contains(alarm, `"compute"`) {
		t.Fatalf("alarm role isolation failed: %v\n%s", err, alarm)
	}
	if !strings.Contains(alarm, "runtime-binding-prepare") || !strings.Contains(base, "runtime-provision-nats") || !strings.Contains(base, "runtime-provision-state") || !strings.Contains(base, "runtime-api-artifact-prepare") || strings.Contains(base, "sandbox-token") || strings.Contains(base, "runtime-db-password") {
		t.Fatalf("runtime binding/base isolation failed:\nbase=%s\nalarm=%s", base, alarm)
	}
	if strings.Contains(alarm, "sandbox.json") || strings.Contains(alarm, "sandbox-token") || strings.Contains(alarm, "sandbox-secret") {
		t.Fatalf("alarm must not receive sandbox Secret: %s", alarm)
	}
	collector, err := RenderProjectWorkloadManifest(ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd", NodeID: testNodeID, Engine: ServiceCollector, ReleaseID: testVersionID, ReleaseDigest: digest, Generation: 2})
	for _, expected := range []string{"image: induforge/collector-engine:1.0.7", `command: ["/collector_artifact_unpack"]`, `command: ["/industrial_collector"]`, `"--artifact"`, `"--binding"`, `"--index"`, `"--listen"`, "mountPath: /etc/induforge/collector/secrets", "mountPath: /var/lib/induforge/collector/wal", `hostPath: {path: "/var/lib/induforge/node-agent/deployments/99999999-9999-4999-8999-999999999999/state/collector-wal", type: Directory}`, "maxUnavailable: 1, maxSurge: 0"} {
		if err != nil || !strings.Contains(collector, expected) {
			t.Fatalf("collector workload missing %q: %v\n%s", expected, err, collector)
		}
	}
	if strings.Contains(collector, "runtime-binding-prepare") || strings.Contains(collector, `command: ["runtime-engine"]`) || strings.Contains(collector, "if-runtime-provisioner") || strings.Contains(collector, "emptyDir: {sizeLimit: \"1Gi\"}") || strings.Contains(collector, "--secrets-dir") || strings.Contains(collector, "--wal-dir") {
		t.Fatalf("collector workload must not reuse runtime-engine init/config: %v\n%s", err, collector)
	}
	compute, err = RenderProjectWorkloadManifest(ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", NodeID: testNodeID, Engine: ServiceCompute, ReleaseID: testVersionID, ReleaseDigest: digest, Generation: 2})
	for _, expected := range []string{"name: compute-sandbox", "image: induforge/compute-sandbox:1.0.8", "secretKeyRef", "COMPUTE_SANDBOX_TOKEN", "readOnly: true"} {
		if err != nil || !strings.Contains(compute, expected) {
			t.Fatalf("compute sandbox missing %q: %v\n%s", expected, err, compute)
		}
	}
	for _, expected := range []string{"COMPUTE_SANDBOX_RUNTIME_PROFILE", `value: "runtime"`, "COMPUTE_SANDBOX_MAX_PIDS", `value: "8192"`, `COMPUTE_SANDBOX_ARTIFACT_ROOT, value: "/work/artifact"`, `COMPUTE_SANDBOX_ARTIFACT_FILE, value: "runtime-project-artifact.json"`, `securityContext: {runAsUser: 0, runAsGroup: 65532, allowPrivilegeEscalation: false, readOnlyRootFilesystem: true, capabilities: {add: ["SYS_ADMIN", "SETUID", "SETGID", "SETPCAP"], drop: ["ALL"]}`, "seccompProfile: {type: Unconfined}", "appArmorProfile: {type: Unconfined}"} {
		if !strings.Contains(compute, expected) {
			t.Fatalf("compute manifest missing sandbox runtime contract %q: %s", expected, compute)
		}
	}
}

func TestRenderProjectWorkloadManifestBindingChecksumChangesTemplate(t *testing.T) {
	base := ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", NodeID: testNodeID, Engine: ServiceCompute, ReleaseID: testVersionID, ReleaseDigest: "sha256:" + strings.Repeat("a", 64), Generation: 2, BindingRevision: 5, RuntimeBindingChecksum: "sha256:one"}
	first, err := RenderProjectWorkloadManifest(base)
	if err != nil {
		t.Fatal(err)
	}
	base.RuntimeBindingChecksum = "sha256:two"
	second, err := RenderProjectWorkloadManifest(base)
	if err != nil || first == second || !strings.Contains(second, `runtime-binding-sha256: "sha256:two"`) || !strings.Contains(second, `binding-revision: "5"`) {
		t.Fatalf("binding checksum did not alter Pod template: %v\n%s", err, second)
	}
}

func TestRenderProjectWorkloadUsesImmutableReleaseDigest(t *testing.T) {
	base := ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", NodeID: testNodeID, Engine: ServiceBase, ReleaseID: testVersionID, ReleaseDigest: "sha256:" + strings.Repeat("a", 64), Generation: 1, HostPort: func() *int { p := 17800; return &p }(), RuntimeNATSEndpoint: "nats://nats:4222"}
	first, err := RenderProjectWorkloadManifest(base)
	if err != nil || strings.Contains(first, "/release/current") {
		t.Fatalf("immutable release mount failed: %v", err)
	}
	base.ReleaseDigest = "sha256:" + strings.Repeat("b", 64)
	second, err := RenderProjectWorkloadManifest(base)
	if err != nil || first == second || !strings.Contains(second, "sha256-"+strings.Repeat("b", 64)) {
		t.Fatalf("digest did not select distinct release: %v", err)
	}
	base.ReleaseDigest = "sha256:bad"
	if _, err := RenderProjectWorkloadManifest(base); err == nil {
		t.Fatal("invalid digest accepted")
	}
}
