package artifact

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestUnpackRequiresCollectorArtifactEntry(t *testing.T) {
	target := filepath.Join(t.TempDir(), "artifact")
	if err := Unpack(bytes.NewReader(archiveForTest(t, "runtime-project-artifact.json")), target, Limits{MaxFiles: 4, MaxFileBytes: 1024, MaxTotalBytes: 4096}); err == nil {
		t.Fatal("runtime artifact entry must not satisfy collector unpack")
	}
	if err := Unpack(bytes.NewReader(archiveForTest(t, EntryFile)), target, Limits{MaxFiles: 4, MaxFileBytes: 1024, MaxTotalBytes: 4096}); err != nil {
		t.Fatal(err)
	}
	if raw, err := os.ReadFile(filepath.Join(target, EntryFile)); err != nil || string(raw) != `{}` {
		t.Fatalf("collector artifact not unpacked: %v %s", err, raw)
	}
}

func archiveForTest(t *testing.T, name string) []byte {
	t.Helper()
	var output bytes.Buffer
	zw, err := zstd.NewWriter(&output)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(zw)
	if err = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err = tw.Write([]byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err = tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err = zw.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
