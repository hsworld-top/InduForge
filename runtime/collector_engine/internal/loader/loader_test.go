package loader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArtifactStrictDecodeAcceptsRequiredWAL(t *testing.T) {
	raw := []byte(`{"schemaVersion":"collector-runtime-artifact.v1","artifactId":"artifact-a","artifactRevision":1,"projectId":"11111111-1111-4111-8111-111111111111","collectorVersion":"1.0","connections":[],"pointMappings":[],"wal":{"fsync":"before-publish","checksum":"crc32c","commitMarker":"after-jetstream-puback"}}`)
	var artifact Artifact
	if err := strictDecode(raw, &artifact); err != nil {
		t.Fatalf("strict decode rejected WAL field: %v", err)
	}
	if artifact.WAL.FSync != "before-publish" || artifact.WAL.Checksum != "crc32c" || artifact.WAL.CommitMarker != "after-jetstream-puback" {
		t.Fatalf("WAL was not retained: %+v", artifact.WAL)
	}
}

func TestValidateCrossUsesNonSensitiveStageCodes(t *testing.T) {
	v := &Loaded{}
	v.Artifact.ArtifactID = "artifact-a"
	v.Artifact.ArtifactRevision = 1
	v.Binding.Artifact.ArtifactID = "other-artifact"
	v.Binding.Artifact.ArtifactRevision = 1
	v.Binding.Artifact.ArtifactDigest = "sha256:invalid"
	err := validateCross(v, []byte(`{}`))
	if err == nil || err.Error() != "collector-loader:artifact-identity" {
		t.Fatalf("unexpected stage error: %v", err)
	}
	if strings.Contains(err.Error(), "artifact-a") {
		t.Fatalf("stage error must not leak artifact values: %v", err)
	}
}

func TestSecureReadAllowsProjectedDataLinkButRejectsEscape(t *testing.T) {
	dir := t.TempDir()
	version := filepath.Join(dir, "..2026_09_02")
	if err := os.Mkdir(version, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(version, "binding.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("..2026_09_02", filepath.Join(dir, "..data")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("..data/binding.json", filepath.Join(dir, "binding.json")); err != nil {
		t.Fatal(err)
	}
	if got, err := secureRead(filepath.Join(dir, "binding.json")); err != nil || string(got) != "{}" {
		t.Fatalf("projected data link must be accepted, got=%q err=%v", got, err)
	}
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "escaped.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := secureRead(filepath.Join(dir, "escaped.json")); err == nil {
		t.Fatal("escaped symlink must be rejected")
	}
}
