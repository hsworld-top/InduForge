package ops

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/indu-forge/dev_core/internal/objectstore"
	"github.com/klauspost/compress/zstd"
)

type centerTestObjects []byte

func (o centerTestObjects) Open(context.Context, string) (objectstore.ObjectReader, error) {
	return objectstore.ObjectReader{Size: int64(len(o)), Reader: io.NopCloser(bytes.NewReader(o))}, nil
}

func centerTestArchive(t *testing.T, name string, kind byte) []byte {
	t.Helper()
	var out bytes.Buffer
	z, err := zstd.NewWriter(&out)
	if err != nil {
		t.Fatal(err)
	}
	w := tar.NewWriter(z)
	size := int64(4)
	if kind != tar.TypeReg {
		size = 0
	}
	if err = w.WriteHeader(&tar.Header{Name: name, Typeflag: kind, Size: size, Mode: 0644}); err != nil {
		t.Fatal(err)
	}
	if size > 0 {
		if _, err = w.Write([]byte("test")); err != nil {
			t.Fatal(err)
		}
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	if err = z.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestCenterReleasePreparation(t *testing.T) {
	archive := centerTestArchive(t, "runtime-artifact.tar.zst", tar.TypeReg)
	digest := sha256.Sum256(archive)
	release := releaseMetadata{ArtifactKey: "release", ArtifactHash: hex.EncodeToString(digest[:]), ArtifactSize: int64(len(archive))}
	root := t.TempDir()
	store := centerReleaseStore{objects: centerTestObjects(archive), localRoot: root, hostRoot: "/host/releases"}
	host, err := store.prepare(context.Background(), testProjectID, release)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(host, "/host/releases/") {
		t.Fatal(host)
	}
	file := filepath.Join(root, testProjectID, "sha256-"+release.ArtifactHash, "runtime-artifact.tar.zst")
	if content, err := os.ReadFile(file); err != nil || string(content) != "test" {
		t.Fatalf("%q %v", content, err)
	}
	if _, err = store.prepare(context.Background(), testProjectID, release); err != nil {
		t.Fatal(err)
	}
	release.ArtifactHash = strings.Repeat("0", 64)
	if _, err = store.prepare(context.Background(), testProjectID, release); err == nil {
		t.Fatal("digest mismatch accepted")
	}
	entries, err := os.ReadDir(filepath.Join(root, testProjectID))
	if err != nil || len(entries) != 1 {
		t.Fatalf("failed preparation leaked entries: %v %v", entries, err)
	}
}

func TestCenterReleaseRejectsUnsafeArchive(t *testing.T) {
	for _, item := range []struct {
		name string
		kind byte
	}{{"../escape", tar.TypeReg}, {"/escape", tar.TypeReg}, {"link", tar.TypeSymlink}} {
		t.Run(item.name, func(t *testing.T) {
			archive := centerTestArchive(t, item.name, item.kind)
			if err := unpackCenterRelease(context.Background(), bytes.NewReader(archive), t.TempDir()); err == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
}
