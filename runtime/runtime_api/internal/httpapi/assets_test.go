package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeAssetsSupportMetadataRangeAndETag(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "video.mp4"), []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(`{"schemaVersion":"runtime-assets.v1","assets":[{"assetId":"asset-1","name":"video.mp4","contentType":"video/mp4","path":"video.mp4"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := NewFileAssetStore(root)
	if err != nil {
		t.Fatal(err)
	}
	server, _, token := newTestRuntimeAPI(t, &fakeStore{})
	server.config.Assets = store
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()
	request, _ := http.NewRequest(http.MethodGet, httpServer.URL+"/api/v1/runtime/assets/asset-1/content", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-InduForge-Deployment-Id", "deployment-1")
	request.Header.Set("X-InduForge-Project-Id", testProjectID)
	request.Header.Set("Range", "bytes=2-5")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusPartialContent {
		t.Fatalf("status=%d, want 206", response.StatusCode)
	}
	if got := response.Header.Get("ETag"); got == "" {
		t.Fatal("缺少 ETag")
	}
}

func TestNewFileAssetStoreRejectsMissingOrMismatchedResources(t *testing.T) {
	root := t.TempDir()
	writeManifest := func(raw string) error { return os.WriteFile(filepath.Join(root, "manifest.json"), []byte(raw), 0o600) }
	if err := writeManifest(`{"schemaVersion":"runtime-assets.v1","assets":[{"assetId":"missing","name":"missing.bin","contentType":"application/octet-stream","path":"missing.bin"}]}`); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFileAssetStore(root); err == nil || !strings.Contains(err.Error(), "资源文件不存在") {
		t.Fatalf("missing resource was accepted: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.bin"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeManifest(`{"schemaVersion":"runtime-assets.v1","assets":[{"assetId":"file","name":"file.bin","contentType":"application/octet-stream","path":"file.bin","size":99}]}`); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFileAssetStore(root); err == nil || !strings.Contains(err.Error(), "大小不匹配") {
		t.Fatalf("mismatched resource was accepted: %v", err)
	}
}
