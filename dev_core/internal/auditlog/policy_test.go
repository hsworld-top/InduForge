package auditlog

import "testing"

func TestStudioOperationScope(t *testing.T) {
	for _, c := range []struct {
		method, path string
		want         bool
	}{
		{"GET", "/api/v1/ops/nodes/metrics", false},
		{"GET", "/api/v1/logs", false},
		{"POST", "/api/v1/projects/abc/datapoints", false},
		{"POST", "/api/v1/projects/abc/scenes", false},
		{"POST", "/api/v1/ops/agent/heartbeat", false},
		{"POST", "/api/v1/users", true},
		{"POST", "/api/v1/ops/runtime-environments/abc/foundation-services/repair", true},
	} {
		if _, _, got := studioOperation(c.method, c.path); got != c.want {
			t.Errorf("%s: %v", c.path, got)
		}
	}
}
