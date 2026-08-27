package sceneasset

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestBuildSelectedAssetFilesRewritesAndReusesDependencies(t *testing.T) {
	selection := map[string]any{
		"induforgeResourceType": "selection",
		"formatVersion":         float64(1),
		"datas": []any{map[string]any{
			"c": "ht.Node",
			"p": map[string]any{"image": "assets/library/source/pump.png"},
		}},
	}
	sceneFiles := map[string]revisionFile{
		"assets/library/source/pump.png": {
			ObjectID: "object-1", Hash: "hash-1", ObjectKey: "objects/hash-1", Type: "image/png", Size: 12,
		},
	}

	providerFiles, stored, err := buildSelectedAssetFiles(selection, sceneFiles)
	if err != nil {
		t.Fatalf("build selection files: %v", err)
	}
	dependencyPath := "dependencies/assets/library/source/pump.png"
	if stored[dependencyPath].ObjectKey != "objects/hash-1" || stored[dependencyPath].Hash != "hash-1" {
		t.Fatalf("dependency was not reused: %#v", stored[dependencyPath])
	}
	var rewritten map[string]any
	if err := json.Unmarshal(providerFiles["selection.json"].Content, &rewritten); err != nil {
		t.Fatalf("decode selection: %v", err)
	}
	datas := rewritten["datas"].([]any)
	properties := datas[0].(map[string]any)["p"].(map[string]any)
	if properties["image"] != dependencyPath {
		t.Fatalf("unexpected rewritten image: %#v", properties["image"])
	}
	if _, err := NewHTProvider().AnalyzeAsset(t.Context(), AssetSymbol, providerFiles); err != nil {
		t.Fatalf("provider should accept generated selection package: %v", err)
	}
}

func TestBuildSelectedAssetFilesRejectsMissingDependency(t *testing.T) {
	selection := map[string]any{"datas": []any{map[string]any{"p": map[string]any{"image": "assets/missing.png"}}}}
	_, _, err := buildSelectedAssetFiles(selection, nil)
	if !errors.Is(err, ErrDependencyMissing) {
		t.Fatalf("expected missing dependency, got %v", err)
	}
}

func TestBuildSelectedAssetFilesKeepsJSONDependencyAsNonEntryFile(t *testing.T) {
	selection := map[string]any{"datas": []any{map[string]any{"p": map[string]any{"image": "symbols/library/source/pump.json"}}}}
	sceneFiles := map[string]revisionFile{
		"symbols/library/source/pump.json": {
			ObjectID: "object-json", Hash: "hash-json", Type: "application/json", Content: []byte(`{"image":"pump.png"}`), Size: 20,
		},
		"symbols/library/source/pump.png": {
			ObjectID: "object-image", Hash: "hash-image", ObjectKey: "objects/hash-image", Type: "image/png", Size: 12,
		},
	}

	providerFiles, _, err := buildSelectedAssetFiles(selection, sceneFiles)
	if err != nil {
		t.Fatalf("build selection files: %v", err)
	}
	analysis, err := NewHTProvider().AnalyzeAsset(t.Context(), AssetSymbol, providerFiles)
	if err != nil {
		t.Fatalf("provider should accept selection with JSON dependency: %v", err)
	}
	if analysis.EntryPath != "selection.json" {
		t.Fatalf("unexpected entry path: %s", analysis.EntryPath)
	}
}
