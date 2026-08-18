package sceneasset

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPublicSceneResponsesHideProviderImplementation(t *testing.T) {
	values := []any{
		Scene{SceneID: "main", Kind: "2d", Provider: ProviderHT, EntryPath: "displays/main.json"},
		SceneAsset{ID: "asset-1", Provider: ProviderHT, Type: AssetImage, EntryPath: "pump.png"},
		Revision{Revision: 1, Provider: ProviderHT, ProviderVersion: HTProviderVersion, EntryPath: "displays/main.json", RootHash: "digest"},
	}
	for _, value := range values {
		payload, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("序列化公开响应失败: %v", err)
		}
		content := string(payload)
		for _, forbidden := range []string{`"provider"`, `"providerVersion"`, `"rootHash"`} {
			if strings.Contains(content, forbidden) {
				t.Fatalf("公开响应泄露 Provider 实现字段 %s: %s", forbidden, content)
			}
		}
		if _, ok := value.(Scene); ok && strings.Contains(content, `"entryPath"`) {
			t.Fatalf("场景公开响应泄露逻辑入口: %s", content)
		}
	}
}
