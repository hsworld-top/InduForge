package contextpack

import (
	"strings"
	"testing"

	"github.com/indu-forge/dev_core/internal/project"
	"github.com/indu-forge/dev_core/internal/scenecontract"
)

func TestBuildFilesIncludesSceneContractsWithoutPrivateCanvas(t *testing.T) {
	files, err := buildFiles(
		project.Project{ID: "project-1", Name: "测试工程", Code: "test"},
		nil,
		pointSnapshot{},
		scenecontract.Snapshot{ContractVersion: "scene-version", Contracts: []scenecontract.Contract{{
			ID:              "line-overview",
			Kind:            "2d",
			Name:            "产线总览",
			Parameters:      []scenecontract.Parameter{{Name: "deviceId", Required: true, Schema: map[string]any{"type": "string"}}},
			Events:          []scenecontract.Member{{Name: "device:selected", Schema: map[string]any{"type": "object"}}},
			Commands:        []scenecontract.Command{{Name: "focusDevice", InputSchema: map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}}, OutputSchema: map[string]any{"type": "boolean"}}},
			DatapointRefs:   []string{"line.speed"},
			ContractVersion: "contract-version",
		}}},
	)
	if err != nil {
		t.Fatalf("buildFiles returned error: %v", err)
	}
	content := string(files["scenes/2d/line-overview.md"])
	if !strings.Contains(content, "line.speed") || strings.Contains(content, "canvas") {
		t.Fatalf("unexpected scene context content: %s", content)
	}
	manifest := string(files["manifest.json"])
	if !strings.Contains(manifest, `"sceneCount": 1`) || !strings.Contains(manifest, "scene-version") {
		t.Fatalf("scene metadata missing from manifest: %s", manifest)
	}
	if !strings.Contains(string(files["scenes/manifest.json"]), `"deviceId"`) {
		t.Fatal("machine-readable scene contract missing")
	}
	if !strings.Contains(content, `await scene.setParams({`) || !strings.Contains(content, `"deviceId": "example"`) {
		t.Fatalf("scene parameter example missing: %s", content)
	}
	if !strings.Contains(content, `scene.addEventListener('scene-event'`) || !strings.Contains(content, `scene.invoke('focusDevice'`) {
		t.Fatalf("scene event or command example missing: %s", content)
	}
}

func TestContextDoesNotDuplicateStaticSDKRules(t *testing.T) {
	files, err := buildFiles(project.Project{ID: "project-1"}, nil, pointSnapshot{}, scenecontract.Snapshot{})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"runtime-api.md", "project/development-rules.md", "authorization/usage.md"} {
		if _, found := files[name]; found {
			t.Fatalf("static SDK documentation duplicated: %s", name)
		}
	}
	if !strings.Contains(string(files["README.md"]), "/opt/induforge/project-sdk") {
		t.Fatal("SDK documentation entry missing")
	}
}
