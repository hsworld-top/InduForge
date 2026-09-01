package ops

import (
	"strings"
	"testing"
)

func TestBuildResolverSecretFilesKeepsResolverShapeInMemory(t *testing.T) {
	files, err := BuildResolverSecretFiles("environment-token", "postgres://runtime")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(files["nats.json"], `"authType":"token"`) || !strings.Contains(files["postgres.json"], `"schemaVersion":"postgres-dsn.v1"`) {
		t.Fatalf("resolver file shapes invalid: %#v", files)
	}
}
