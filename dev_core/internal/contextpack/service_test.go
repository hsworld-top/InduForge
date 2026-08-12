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
			EmbedMode:       "both",
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
}
