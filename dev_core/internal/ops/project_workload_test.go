package ops

import (
	"strings"
	"testing"
)

func TestRenderProjectWorkloadManifestSeparatesRolesAndHostPort(t *testing.T) {
	port := 17800
	digest := "sha256:" + strings.Repeat("a", 64)
	base, err := RenderProjectWorkloadManifest(ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", NodeID: testNodeID, Engine: ServiceBase, ReleaseID: testVersionID, ReleaseDigest: digest, Generation: 2, HostPort: &port})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"if-env-666666666666", "if-project-999999999999-base", "hostPort: 17800", "induforge.io/node-id", "maxUnavailable: 0", "allowPrivilegeEscalation: false", "IF_RELEASE_ROOT", "IF_WORK_ROOT", "mountPath: /opt/induforge/release", "kind: Service", "image: induforge/project-gateway:1.0.0", "imagePullPolicy: IfNotPresent", `path: "/var/lib/induforge/node-agent/deployments/99999999-9999-4999-8999-999999999999/release/releases/sha256-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`} {
		if !strings.Contains(base, expected) {
			t.Fatalf("base manifest missing %q: %s", expected, base)
		}
	}
	compute, err := RenderProjectWorkloadManifest(ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", NodeID: testNodeID, Engine: ServiceCompute, ReleaseID: testVersionID, ReleaseDigest: digest, Generation: 2})
	if err != nil || !strings.Contains(compute, "IF_ENGINE_ROLE") || !strings.Contains(compute, `"compute"`) || strings.Contains(compute, "hostPort:") {
		t.Fatalf("compute role/port isolation failed: %v\n%s", err, compute)
	}
	for _, expected := range []string{"initContainers:", "name: runtime-provision-nats", "name: runtime-provision-state", "name: runtime-binding-prepare", `args: ["provision-nats", "--input", "/etc/induforge/runtime-binding/input.json", "--credentials", "/var/run/induforge/bootstrap/nats.json"]`, `args: ["provision-state", "--input", "/etc/induforge/runtime-binding/input.json", "--credentials", "/var/run/induforge/bootstrap/postgres-bootstrap.json"]`, `args: ["prepare", "--input", "/etc/induforge/runtime-binding/input.json"]`, "name: runtime-secrets", "name: bootstrap-nats", "name: bootstrap-state", "postgres-bootstrap.json", "mountPath: /work/bundle/secrets", "name: runtime-binding", "emptyDir: {sizeLimit: \"768Mi\"}", "induforge.io/runtime-binding-sha256"} {
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
	if !strings.Contains(alarm, "runtime-binding-prepare") || strings.Contains(base, "runtime-binding-prepare") || strings.Contains(base, "runtime-provision-nats") || strings.Contains(base, "runtime-provision-state") || strings.Contains(base, "deployment-secrets") {
		t.Fatalf("runtime binding/base isolation failed:\nbase=%s\nalarm=%s", base, alarm)
	}
	if strings.Contains(alarm, "sandbox.json") || strings.Contains(alarm, "sandbox-token") || strings.Contains(alarm, "sandbox-secret") {
		t.Fatalf("alarm must not receive sandbox Secret: %s", alarm)
	}
	compute, err = RenderProjectWorkloadManifest(ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", NodeID: testNodeID, Engine: ServiceCompute, ReleaseID: testVersionID, ReleaseDigest: digest, Generation: 2})
	for _, expected := range []string{"name: compute-sandbox", "image: induforge/compute-sandbox:1.0.0", "secretKeyRef", "COMPUTE_SANDBOX_TOKEN", "readOnly: true"} {
		if err != nil || !strings.Contains(compute, expected) {
			t.Fatalf("compute sandbox missing %q: %v\n%s", expected, err, compute)
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
	base := ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", NodeID: testNodeID, Engine: ServiceBase, ReleaseID: testVersionID, ReleaseDigest: "sha256:" + strings.Repeat("a", 64), Generation: 1, HostPort: func() *int { p := 17800; return &p }()}
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
