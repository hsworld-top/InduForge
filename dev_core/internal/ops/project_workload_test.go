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
	for _, expected := range []string{"if-env-666666666666", "if-project-999999999999-base", "hostPort: 17800", "induforge.io/node-id", "maxUnavailable: 0", "allowPrivilegeEscalation: false", "IF_RELEASE_ROOT", "mountPath: /opt/induforge/release", `path: "/var/lib/induforge/node-agent/deployments/99999999-9999-4999-8999-999999999999/release"`} {
		if !strings.Contains(base, expected) {
			t.Fatalf("base manifest missing %q: %s", expected, base)
		}
	}
	compute, err := RenderProjectWorkloadManifest(ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", NodeID: testNodeID, Engine: ServiceCompute, ReleaseID: testVersionID, Generation: 2})
	if err != nil || !strings.Contains(compute, "IF_ENGINE_ROLE") || !strings.Contains(compute, `"compute"`) || strings.Contains(compute, "hostPort:") {
		t.Fatalf("compute role/port isolation failed: %v\n%s", err, compute)
	}
	alarm, err := RenderProjectWorkloadManifest(ProjectWorkload{EnvironmentID: testEnvironmentID, DeploymentID: "99999999-9999-4999-8999-999999999999", ServiceID: "cccccccc-cccc-4ccc-8ccc-cccccccccccc", NodeID: testNodeID, Engine: ServiceAlarm, ReleaseID: testVersionID, Generation: 2})
	if err != nil || !strings.Contains(alarm, `"alarm"`) || strings.Contains(alarm, `"compute"`) {
		t.Fatalf("alarm role isolation failed: %v\n%s", err, alarm)
	}
}
