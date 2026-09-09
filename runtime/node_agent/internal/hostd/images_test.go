package hostd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type imageTestRunner struct {
	imported bool
	calls    int
	digest   string
	expected string
}

func (r *imageTestRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	r.calls++
	if args[0] == "ctr" {
		if len(args) < 8 || args[1] != "--address" || args[2] != "/run/k3s/containerd/containerd.sock" {
			return nil, fmt.Errorf("missing explicit containerd address")
		}
		data, err := os.ReadFile(args[len(args)-1])
		if err != nil || string(data) != r.expected {
			return nil, fmt.Errorf("bad private archive")
		}
		r.imported = true
		return nil, nil
	}
	if len(args) < 11 || args[1] != "--config" || args[2] != "/dev/null" || args[3] != "--runtime-endpoint" || args[4] != "unix:///run/k3s/containerd/containerd.sock" || args[5] != "--image-endpoint" || args[6] != args[4] {
		return nil, fmt.Errorf("missing explicit CRI endpoints")
	}
	if r.imported {
		return []byte(fmt.Sprintf(`{"status":{"id":%q}}`, r.digest)), nil
	}
	return nil, fmt.Errorf("image missing")
}
func TestImportImagesValidatesArchiveAndContent(t *testing.T) {
	for _, bad := range []bool{false, true} {
		t.Run(fmt.Sprint(bad), func(t *testing.T) {
			body := []byte("archive")
			sum := sha256.Sum256(body)
			digest := "sha256:" + strings.Repeat("a", 64)
			p := ImageArtifact{SHA256: hex.EncodeToString(sum[:]), Size: int64(len(body)), Images: []ImageReference{{Reference: "redis:7", ConfigDigest: digest}}}
			dir := t.TempDir()
			runner := &imageTestRunner{digest: digest, expected: string(body)}
			cfg := DefaultManagerConfig()
			cfg.StateDir = filepath.Join(dir, "state")
			cfg.ImageCacheDir = dir
			cfg.Runner = runner
			m, err := NewManager(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if bad {
				body = []byte("corrupt")
			}
			if err = os.WriteFile(filepath.Join(dir, p.SHA256+".tar"), body, 0600); err != nil {
				t.Fatal(err)
			}
			result, err := m.ImportImages(context.Background(), p)
			if bad {
				if err == nil || runner.imported {
					t.Fatal("corrupt archive imported")
				}
				return
			}
			if err != nil || !result.Ready {
				t.Fatalf("import: %v %+v", err, result)
			}
			if _, err = os.Stat(filepath.Join(dir, p.SHA256+".tar")); !os.IsNotExist(err) {
				t.Fatal("imported archive was not removed")
			}
			if result, err = m.ImportImages(context.Background(), p); err != nil || !result.Ready {
				t.Fatal("local image not reused")
			}
		})
	}
}
func TestImagesOnlyHandlerRejectsClusterOperations(t *testing.T) {
	cfg := DefaultManagerConfig()
	m, err := NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	api, _ := NewAPI(m)
	w := httptest.NewRecorder()
	api.ImagesHandler().ServeHTTP(w, httptest.NewRequest("POST", "/v1/cluster/uninstall", strings.NewReader(`{}`)))
	if w.Code != 404 {
		t.Fatalf("cluster route exposed: %d", w.Code)
	}
}
func TestImageArtifactRejectsPathsAndInvalidDigest(t *testing.T) {
	p := ImageArtifact{SHA256: "../../etc/passwd", Size: 1, Images: []ImageReference{{Reference: "redis", ConfigDigest: "sha256:" + strings.Repeat("a", 64)}}}
	if p.Validate() == nil {
		t.Fatal("unsafe digest accepted")
	}
}
