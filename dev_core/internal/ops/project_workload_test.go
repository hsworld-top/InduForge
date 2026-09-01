package ops

import (
	"strings"
	"testing"
)

func TestRenderProjectWorkloadManifestSeparatesRolesAndHostPort(t *testing.T) {
	port := 17800
	base, err := RenderProjectWorkloadManifest(ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", NodeID: testNodeID, Engine: ServiceBase, ReleaseID: testVersionID, Generation: 2, HostPort: &port})
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"if-env-666666666666", "if-project-999999999999-base", "hostPort: 17800", "induforge.io/node-id", "maxUnavailable: 0", "allowPrivilegeEscalation: false", "IF_RELEASE_ROOT", "IF_WORK_ROOT", "mountPath: /opt/induforge/release", "kind: Service", "image: induforge/project-gateway:1.0.0", "imagePullPolicy: IfNotPresent", `path: "/var/lib/induforge/node-agent/deployments/99999999-9999-4999-8999-999999999999/release"`} {
		if !strings.Contains(base, expected) {
			t.Fatalf("base manifest missing %q: %s", expected, base)
		}
	}
	compute, err := RenderProjectWorkloadManifest(ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", NodeID: testNodeID, Engine: ServiceCompute, ReleaseID: testVersionID, Generation: 2})
	if err != nil || !strings.Contains(compute, "IF_ENGINE_ROLE") || !strings.Contains(compute, `"compute"`) || strings.Contains(compute, "hostPort:") {
		t.Fatalf("compute role/port isolation failed: %v\n%s", err, compute)
	}
	for _, expected := range []string{"initContainers:", "name: runtime-binding-prepare", `command: ["if-runtime-provisioner", "prepare", "--input", "/etc/induforge/runtime-binding/input.json"]`, "args:", "name: runtime-secrets", "name: bootstrap-secrets", "postgres-bootstrap.json", "mountPath: /work/bundle/secrets", "name: runtime-binding", "induforge.io/runtime-binding-sha256"} {
		if !strings.Contains(compute, expected) {
			t.Fatalf("compute binding manifest missing %q:\n%s", expected, compute)
		}
	}
	alarm, err := RenderProjectWorkloadManifest(ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "cccccccc-cccc-4ccc-8ccc-cccccccccccc", NodeID: testNodeID, Engine: ServiceAlarm, ReleaseID: testVersionID, Generation: 2})
	if err != nil || !strings.Contains(alarm, `"alarm"`) || strings.Contains(alarm, `"compute"`) {
		t.Fatalf("alarm role isolation failed: %v\n%s", err, alarm)
	}
	if !strings.Contains(alarm, "runtime-binding-prepare") || strings.Contains(base, "runtime-binding-prepare") || strings.Contains(base, "deployment-secrets") {
		t.Fatalf("runtime binding/base isolation failed:\nbase=%s\nalarm=%s", base, alarm)
	}
	if strings.Contains(alarm, "sandbox.json") || strings.Contains(alarm, "sandbox-token") || strings.Contains(alarm, "sandbox-secret") {
		t.Fatalf("alarm must not receive sandbox Secret: %s", alarm)
	}
	compute, err = RenderProjectWorkloadManifest(ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", NodeID: testNodeID, Engine: ServiceCompute, ReleaseID: testVersionID, Generation: 2})
	for _, expected := range []string{"name: compute-sandbox", "image: induforge/compute-sandbox:1.0.0", "secretKeyRef", "COMPUTE_SANDBOX_TOKEN", "readOnly: true"} {
		if err != nil || !strings.Contains(compute, expected) {
			t.Fatalf("compute sandbox missing %q: %v\n%s", expected, err, compute)
		}
	}
}

func TestRenderProjectWorkloadManifestBindingChecksumChangesTemplate(t *testing.T) {
	base := ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", NodeID: testNodeID, Engine: ServiceCompute, ReleaseID: testVersionID, Generation: 2, BindingRevision: 5, RuntimeBindingChecksum: "sha256:one"}
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
