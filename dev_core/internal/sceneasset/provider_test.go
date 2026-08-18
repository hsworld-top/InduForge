package sceneasset

import (
	"context"
	"errors"
	"net/url"
	"testing"
)

func TestHTProviderAnalyzeModelDependencies(t *testing.T) {
	provider := NewHTProvider()
	files := map[string]ProviderFile{
		"pump.obj":             canonicalProviderFile("pump.obj", []byte("mtllib materials/pump.mtl\n"), ""),
		"materials/pump.mtl":   canonicalProviderFile("materials/pump.mtl", []byte("map_Kd ../textures/pump.png\n"), ""),
		"textures/pump.png":    canonicalProviderFile("textures/pump.png", []byte("png"), "image/png"),
		"textures/preview.png": canonicalProviderFile("textures/preview.png", []byte("preview"), "image/png"),
	}

	analysis, err := provider.AnalyzeAsset(context.Background(), AssetModel, files)
	if err != nil {
		t.Fatalf("分析模型资源失败: %v", err)
	}
	if analysis.EntryPath != "pump.obj" {
		t.Fatalf("模型入口错误: got %q", analysis.EntryPath)
	}
	if len(analysis.Dependencies) != len(files) {
		t.Fatalf("资源代次没有保留完整上传清单: got %v", analysis.Dependencies)
	}
}

func TestHTProviderRejectsMissingModelDependency(t *testing.T) {
	provider := NewHTProvider()
	files := map[string]ProviderFile{
		"pump.obj": canonicalProviderFile("pump.obj", []byte("mtllib pump.mtl\n"), ""),
	}

	_, err := provider.AnalyzeAsset(context.Background(), AssetModel, files)
	if !errors.Is(err, ErrDependencyMissing) {
		t.Fatalf("缺少 MTL 应返回依赖错误: got %v", err)
	}
}

func TestHTProviderRejectsMissingJSONDependency(t *testing.T) {
	provider := NewHTProvider()
	files := map[string]ProviderFile{
		"symbols/pump.json": canonicalProviderFile("symbols/pump.json", []byte(`{"image":"pump.png"}`), "application/json"),
	}

	_, err := provider.AnalyzeAsset(context.Background(), AssetSymbol, files)
	if !errors.Is(err, ErrDependencyMissing) {
		t.Fatalf("缺少图片应返回依赖错误: got %v", err)
	}
}

func TestHTProviderResolvesRootJSONDependency(t *testing.T) {
	provider := NewHTProvider()
	files := map[string]ProviderFile{
		"displays/main.json": canonicalProviderFile("displays/main.json", []byte(`{"image":"assets/pump.png"}`), "application/json"),
		"assets/pump.png":    canonicalProviderFile("assets/pump.png", []byte("png"), "image/png"),
	}

	analysis, err := provider.Analyze(context.Background(), "2d", "displays/main.json", files)
	if err != nil {
		t.Fatalf("分析根路径资源失败: %v", err)
	}
	if len(analysis.Dependencies) != 2 || analysis.Dependencies[1] != "displays/main.json" {
		t.Fatalf("依赖闭包错误: got %v", analysis.Dependencies)
	}
}

func TestHTProviderDraftOnlyAcceptsConfiguredRootEntry(t *testing.T) {
	provider := NewHTProvider()
	tests := []struct {
		name      string
		kind      string
		entryPath string
		filename  string
		wantError bool
	}{
		{name: "2D 固定入口", kind: "2d", entryPath: "displays/main.json", filename: "displays/main.json"},
		{name: "2D 第二张图纸", kind: "2d", entryPath: "displays/main.json", filename: "displays/other.json", wantError: true},
		{name: "3D 固定入口", kind: "3d", entryPath: "scenes/main.json", filename: "scenes/main.json"},
		{name: "3D 第二个场景", kind: "3d", entryPath: "scenes/main.json", filename: "scenes/other.json", wantError: true},
		{name: "资源依赖", kind: "2d", entryPath: "displays/main.json", filename: "symbols/library/pump.json"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := provider.ValidateDraftFile(test.kind, test.entryPath, test.filename, []byte("{}"))
			if test.wantError && !errors.Is(err, ErrProtectedPath) {
				t.Fatalf("应拒绝额外根入口: got %v", err)
			}
			if !test.wantError && err != nil {
				t.Fatalf("合法场景文件被拒绝: %v", err)
			}
		})
	}
}

func TestHTProviderEditorURLContainsControlledEntry(t *testing.T) {
	provider := NewHTProvider()
	tests := []struct {
		kind      string
		entryPath string
		page      string
	}{
		{kind: "2d", entryPath: "displays/main scene.json", page: "/designer/scene-studio/index.html"},
		{kind: "3d", entryPath: "scenes/main scene.json", page: "/designer/scene-studio/index3d.html"},
	}
	for _, test := range tests {
		result, err := url.Parse(provider.EditorURL(EditorSession{ID: "session-1", Kind: test.kind, EntryPath: test.entryPath}))
		if err != nil {
			t.Fatalf("编辑器 URL 无效: %v", err)
		}
		if result.Path != test.page || result.Query().Get("sessionId") != "session-1" || result.Query().Get("open") != test.entryPath {
			t.Fatalf("编辑器 URL 未绑定固定入口: got %s", result.String())
		}
	}
}
