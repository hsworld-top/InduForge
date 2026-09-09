package imagecatalog

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/indu-forge/dev_core/internal/objectstore"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type memoryStore struct {
	objects map[string][]byte
	writes  []string
}

func (s *memoryStore) Open(_ context.Context, k string) (objectstore.ObjectReader, error) {
	b, ok := s.objects[k]
	if !ok {
		return objectstore.ObjectReader{}, fmt.Errorf("missing")
	}
	return objectstore.ObjectReader{Reader: io.NopCloser(bytes.NewReader(b)), Size: int64(len(b))}, nil
}
func (s *memoryStore) Put(_ context.Context, k string, r io.Reader, n int64, _ string) (objectstore.ObjectRef, error) {
	b, e := io.ReadAll(r)
	if e != nil {
		return objectstore.ObjectRef{}, e
	}
	s.objects[k] = b
	s.writes = append(s.writes, k)
	return objectstore.ObjectRef{Size: n}, nil
}
func fixture(t *testing.T) (string, Manifest, *memoryStore) {
	t.Helper()
	dir := t.TempDir()
	b := []byte("offline image archive")
	h := sha256.Sum256(b)
	a := Artifact{Architecture: "arm64", SHA256: hex.EncodeToString(h[:]), Size: int64(len(b)), Archive: "image.tar", Images: []Image{{Reference: "redis:7.2-alpine", ConfigDigest: "sha256:" + strings.Repeat("a", 64)}}}
	m := Manifest{SchemaVersion: 1, Artifacts: []Artifact{a}}
	_ = os.WriteFile(filepath.Join(dir, a.Archive), b, 0600)
	raw, _ := json.Marshal(m)
	_ = os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0600)
	return dir, m, &memoryStore{objects: map[string][]byte{}}
}
func TestImportPublishesVerifiedObjectsBeforeCatalog(t *testing.T) {
	dir, m, s := fixture(t)
	c := Catalog{Store: s}
	if e := c.Import(context.Background(), dir); e != nil {
		t.Fatal(e)
	}
	if len(s.writes) != 2 || s.writes[1] != catalogKey || s.writes[0] != Key(m.Artifacts[0].SHA256) {
		t.Fatal(s.writes)
	}
	a, e := c.Resolve(context.Background(), "arm64", []string{"redis:7.2-alpine"})
	if e != nil || len(a) != 1 || a[0].Archive != "" {
		t.Fatalf("%+v %v", a, e)
	}
	if _, e = c.Resolve(context.Background(), "amd64", []string{"redis:7.2-alpine"}); e == nil {
		t.Fatal("wrong architecture accepted")
	}
}
func TestImportRejectsCorruptionWithoutWrites(t *testing.T) {
	dir, _, s := fixture(t)
	_ = os.WriteFile(filepath.Join(dir, "image.tar"), []byte("corrupt"), 0600)
	if e := (&Catalog{Store: s}).Import(context.Background(), dir); e == nil {
		t.Fatal("corrupt archive accepted")
	}
	if len(s.writes) != 0 {
		t.Fatal(s.writes)
	}
}
func TestArchiveRejectsEscapingSymlink(t *testing.T) {
	dir, m, _ := fixture(t)
	outside := filepath.Join(t.TempDir(), "outside")
	_ = os.WriteFile(outside, []byte("offline image archive"), 0600)
	_ = os.Remove(filepath.Join(dir, "image.tar"))
	_ = os.Symlink(outside, filepath.Join(dir, "image.tar"))
	if e := checkArchive(dir, m.Artifacts[0]); e == nil {
		t.Fatal("escaping link accepted")
	}
}
func TestValidateRejectsDuplicateImageAndDigest(t *testing.T) {
	_, m, _ := fixture(t)
	m.Artifacts = append(m.Artifacts, m.Artifacts[0])
	if e := m.Validate(); e == nil {
		t.Fatal("duplicate accepted")
	}
	m.Artifacts = m.Artifacts[:1]
	m.Artifacts[0].Images[0].ConfigDigest = "sha256:bad"
	if e := m.Validate(); e == nil {
		t.Fatal("bad digest accepted")
	}
}

func TestCatalogCacheDoesNotLeakCallerMutation(t *testing.T) {
	dir, _, s := fixture(t)
	c := &Catalog{Store: s}
	if e := c.Import(context.Background(), dir); e != nil {
		t.Fatal(e)
	}
	first, e := c.Load(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	first.Artifacts[0].Images[0].Reference = "mutated"
	first.Artifacts[0].DownloadPath = "private-node-path"
	delete(s.objects, catalogKey)
	next, e := c.Load(context.Background())
	if e != nil {
		t.Fatal("cache not used", e)
	}
	if next.Artifacts[0].Images[0].Reference == "mutated" || next.Artifacts[0].DownloadPath != "" {
		t.Fatal("cached catalog mutated")
	}
}
func TestImportRetryReusesCompletedObjectsButReplacesWrongSize(t *testing.T) {
	dir, m, s := fixture(t)
	c := &Catalog{Store: s}
	if e := c.Import(context.Background(), dir); e != nil {
		t.Fatal(e)
	}
	s.writes = nil
	if e := c.Import(context.Background(), dir); e != nil {
		t.Fatal(e)
	}
	if len(s.writes) != 1 || s.writes[0] != catalogKey {
		t.Fatalf("completed archive retransmitted: %v", s.writes)
	}
	s.objects[Key(m.Artifacts[0].SHA256)] = []byte("short")
	s.writes = nil
	if e := c.Import(context.Background(), dir); e != nil {
		t.Fatal(e)
	}
	if len(s.writes) != 2 {
		t.Fatal("wrong-size object incorrectly reused", s.writes)
	}
}
