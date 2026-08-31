package realtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/indu-forge/runtime-api/internal/artifact"
)

func TestHubValidatesIdentityAndRoutesOnlySubscribedPaths(t *testing.T) {
	catalog := testCatalog(t)
	hub := NewHub("deployment-1", "account-1", catalog)
	events, cancel, err := hub.Subscribe([]string{"line.temperature"})
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	payload := map[string]any{
		"schemaVersion": "data.raw.v1", "deploymentId": "deployment-1", "accountId": "account-1",
		"pointId": "22222222-2222-4222-8222-222222222222", "eventId": "event-1", "value": 42.5,
		"quality": "good", "sourceTimestamp": "2026-08-31T10:00:00Z", "serverTimestamp": "2026-08-31T10:00:01Z", "sequence": 1,
	}
	body, _ := json.Marshal(payload)
	if err := hub.Accept("data.raw.22222222-2222-4222-8222-222222222222", body); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.Path != "line.temperature" || event.Quality != "good" {
			t.Fatalf("unexpected event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected realtime event")
	}
	payload["accountId"] = "other"
	body, _ = json.Marshal(payload)
	if err := hub.Accept("data.raw.22222222-2222-4222-8222-222222222222", body); err == nil {
		t.Fatal("cross-account event must be rejected")
	}
}

func testCatalog(t *testing.T) *artifact.Catalog {
	t.Helper()
	path := filepath.Join(t.TempDir(), "artifact.json")
	payload := `{"schemaVersion":"runtime-project-artifact.v1","projectArtifactVersion":"1.0","projectId":"11111111-1111-4111-8111-111111111111","dataPoints":[{"id":"22222222-2222-4222-8222-222222222222","path":"line.temperature","name":"温度","dataType":"float64","sourceType":"collector.point","sourceId":"33333333-3333-4333-8333-333333333333","runtimePermissions":{"write":{"allowRoles":[],"denyRoles":[],"inherit":true}},"refreshMode":"subscription","status":"active","unit":"C","precisionNum":1,"defaultValue":null,"tags":[],"attributeDefaults":{}}],"computeUnits":[],"alarmItems":[]}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	catalog, err := artifact.Load(path, "11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}
