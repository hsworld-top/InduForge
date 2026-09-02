package loader

import "testing"

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
