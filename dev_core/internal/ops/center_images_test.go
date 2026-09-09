package ops

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/indu-forge/dev_core/internal/imagecatalog"
	"github.com/indu-forge/dev_core/internal/objectstore"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type centerImageObjects struct {
	data  []byte
	opens int
}

func (s *centerImageObjects) Open(_ context.Context, _ string) (objectstore.ObjectReader, error) {
	s.opens++
	return objectstore.ObjectReader{Reader: io.NopCloser(bytes.NewReader(s.data)), Size: int64(len(s.data))}, nil
}
func (s *centerImageObjects) Put(context.Context, string, io.Reader, int64, string) (objectstore.ObjectRef, error) {
	return objectstore.ObjectRef{}, fmt.Errorf("unexpected upload")
}
func TestCenterImagesChecksDownloadsAndImportsViaLocalHostd(t *testing.T) {
	dir, err := os.MkdirTemp("", "if-img-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "h.sock")
	cache := filepath.Join(dir, "cache")
	t.Setenv("IF_OPS_CENTER_HOSTD_SOCKET", socket)
	t.Setenv("IF_OPS_CENTER_IMAGE_CACHE", cache)
	data := []byte("compressed-or-plain-archive")
	hash := sha256.Sum256(data)
	a := imagecatalog.Artifact{SHA256: hex.EncodeToString(hash[:]), Size: int64(len(data)), Images: []imagecatalog.Image{{Reference: "redis:7.2-alpine", ConfigDigest: "sha256:" + strings.Repeat("a", 64)}}}
	ready := false
	imports := 0
	l, e := net.Listen("unix", socket)
	if e != nil {
		t.Fatal(e)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p struct {
			SHA256 string               `json:"sha256"`
			Size   int64                `json:"size"`
			Images []imagecatalog.Image `json:"images"`
		}
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		if e := d.Decode(&p); e != nil {
			t.Error(e)
		}
		if r.URL.Path == "/v1/images/import" {
			imports++
			got, e := os.ReadFile(filepath.Join(cache, p.SHA256+".tar"))
			if e != nil || !bytes.Equal(got, data) {
				t.Errorf("archive=%q err=%v", got, e)
			}
			ready = true
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]bool{"ready": ready}})
	})}
	go server.Serve(l)
	defer server.Close()
	objects := &centerImageObjects{data: data}
	repo := &PostgreSQLRepository{imageCatalog: &imagecatalog.Catalog{Store: objects}}
	for i := 0; i < 2; i++ {
		if e = repo.prepareCenterImages(context.Background(), []imagecatalog.Artifact{a}); e != nil {
			t.Fatal(e)
		}
	}
	if imports != 1 || objects.opens != 1 {
		t.Fatalf("imports=%d downloads=%d", imports, objects.opens)
	}
}
