package provision

import (
	"archive/tar"
	"bytes"
	"testing"

	"github.com/klauspost/compress/zstd"
)

func TestUnpackArtifactRejectsTraversalMembers(t *testing.T) {
	for _, name := range []string{"../collector-runtime-artifact.json", "/collector-runtime-artifact.json"} {
		var raw bytes.Buffer
		zw, _ := zstd.NewWriter(&raw)
		tw := tar.NewWriter(zw)
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: 2}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte("{}")); err != nil {
			t.Fatal(err)
		}
		_ = tw.Close()
		_ = zw.Close()
		if err := UnpackArtifact(bytes.NewReader(raw.Bytes()), t.TempDir()+"/out", "collector-runtime-artifact.json", Limits{MaxFiles: 2, MaxFileBytes: 1024, MaxTotalBytes: 1024}); err == nil {
			t.Fatalf("traversal member %q accepted", name)
		}
	}
}
