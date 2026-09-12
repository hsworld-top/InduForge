package contextpack

import (
	"strings"
	"testing"

	"github.com/indu-forge/dev_core/internal/project"
	"github.com/indu-forge/dev_core/internal/runtimeaccess"
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

func TestBuildFilesWritesMachineReadableAndMarkdownContextTogether(t *testing.T) {
	files, err := buildFiles(
		project.Project{ID: "project-1", Name: "测试工程", Code: "test"},
		[]runtimeaccess.Role{{ID: "role-1", ProjectID: "project-1", Code: "OPERATOR", Name: "操作员", Capabilities: []string{"point:read"}}},
		pointSnapshot{ProjectID: "project-1", ContractVersion: "points-v1", DataPoints: []dataPointContext{{
			ID: "line.speed", Name: "线速度", DataType: "number", Status: "active", Attributes: map[string]string{"unit": "m/s"},
			Methods: []dataPointMethod{{Name: "get", Description: "读取当前值", Parameters: map[string]any{"type": "object"}}},
		}}},
		scenecontract.Snapshot{},
		resourceContext{Items: []map[string]any{}, Available: true},
		resourceContext{Items: []map[string]any{}, Available: true},
		resourceContext{Items: []map[string]any{{"assetId": "asset-1", "name": "产线图标", "type": "image", "compatibleKind": "2d", "thumbnailUrl": "/api/v1/projects/project-1/scene-assets/asset-1/thumbnail", "objectKey": "must-not-leak"}}, Available: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"project/overview.json", "authorization/roles.json", "points/catalog/0001.json", "points/catalog/0001.md", "alarms/catalog.json", "computes/catalog.json", "object-library/README.md", "object-library/catalog.json", "object-library/catalog.md"} {
		if _, ok := files[name]; !ok {
			t.Fatalf("context file %s missing", name)
		}
	}
	if !strings.Contains(string(files["authorization/roles.json"]), "OPERATOR") {
		t.Fatalf("role JSON missing role code: %s", files["authorization/roles.json"])
	}
	if !strings.Contains(string(files["points/catalog/0001.json"]), "m/s") || !strings.Contains(string(files["points/catalog/0001.md"]), "读取当前值") {
		t.Fatalf("point fields were not preserved in both formats")
	}
	if !strings.Contains(string(files["object-library/catalog.json"]), "产线图标") || strings.Contains(string(files["object-library/catalog.json"]), "must-not-leak") {
		t.Fatalf("object library metadata was not filtered or preserved: %s", files["object-library/catalog.json"])
	}
	if !strings.Contains(string(files["manifest.json"]), `"objectLibraryCount": 1`) {
		t.Fatalf("object library count missing from manifest: %s", files["manifest.json"])
	}
}
