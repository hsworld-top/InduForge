package loader

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestEmbeddedSchemasMatchRuntimeEngine guards the deliberately minimal schema copy.
func TestEmbeddedSchemasMatchRuntimeEngine(t *testing.T) {
	for _, name := range []string{"common.schema.json", "collector-runtime-artifact.schema.json", "collector-runtime-binding.schema.json", "point-event.schema.json"} {
		got, err := schemaFiles.ReadFile("schemas/runtime/" + name)
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join("..", "..", "..", "runtime_engine", "internal", "loader", "schemas", "runtime", name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("embedded schema %s diverged from runtime_engine contract", name)
		}
	}
}
