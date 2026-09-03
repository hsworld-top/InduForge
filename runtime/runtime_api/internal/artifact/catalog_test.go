package artifact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const projectID = "11111111-1111-4111-8111-111111111111"

func TestLoadCatalogIndexesOnlyRuntimeSafeMetadata(t *testing.T) {
	path := writeArtifact(t, `{
  "schemaVersion":"runtime-project-artifact.v1","projectArtifactVersion":"1.0",
  "projectId":"11111111-1111-4111-8111-111111111111",
  "dataPoints":[{"id":"22222222-2222-4222-8222-222222222222","path":"line.temperature","name":"温度","dataType":"float64","sourceType":"collector.point","sourceId":"33333333-3333-4333-8333-333333333333","runtimePermissions":{"write":{"allowRoles":[],"denyRoles":[],"inherit":true}},"refreshMode":"subscription","status":"active","unit":"C","precisionNum":1,"defaultValue":null,"tags":[],"attributeDefaults":{}}],
  "computeUnits":[{"id":"44444444-4444-4444-8444-444444444444","revision":1,"name":"计算","description":null,"enabled":true,"inputs":[],"outputs":[],"trigger":{"kind":"manual"},"scriptCode":"secret implementation"}],
  "alarmItems":[{"id":"55555555-5555-4555-8555-555555555555","revision":2,"displayName":"高温报警","enabled":true,"mode":"point","evaluationMode":"single","inputs":[{"alias":"value","datapointId":"22222222-2222-4222-8222-222222222222","path":"line.temperature","dataType":"float64"}],"conditions":[]}]}`)
	catalog, err := Load(path, projectID)
	if err != nil {
		t.Fatal(err)
	}
	point, ok := catalog.PointByPath("line.temperature")
	if !ok || point.ID != "22222222-2222-4222-8222-222222222222" || !strings.HasPrefix(catalog.Digest, "sha256:") {
		t.Fatalf("unexpected catalog: %#v %#v", catalog, point)
	}
	computes := catalog.Computes()
	if len(computes) != 1 || strings.Contains(string(computes[0].Trigger), "secret implementation") {
		t.Fatalf("compute metadata leaked implementation: %#v", computes)
	}
	alarm, ok := catalog.AlarmByID("55555555-5555-4555-8555-555555555555")
	if !ok || alarm.Name != "高温报警" || alarm.DisplayName != "高温报警" || len(alarm.Inputs) != 1 || alarm.Inputs[0].Path != "line.temperature" {
		t.Fatalf("unexpected alarm metadata: %#v", alarm)
	}
}

func TestLoadCatalogRejectsProjectMismatchAndDuplicatePath(t *testing.T) {
	path := writeArtifact(t, `{"schemaVersion":"runtime-project-artifact.v1","projectArtifactVersion":"1.0","projectId":"11111111-1111-4111-8111-111111111111","dataPoints":[],"computeUnits":[],"alarmItems":[]}`)
	if _, err := Load(path, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"); err == nil || !strings.Contains(err.Error(), "projectId") {
		t.Fatalf("expected project mismatch, got %v", err)
	}
}

func writeArtifact(t *testing.T, payload string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "runtime-project.json")
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
