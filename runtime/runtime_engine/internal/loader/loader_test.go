package loader

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/indu-forge/runtime-engine/internal/model"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Join("../../../../contracts/runtime/fixtures", name))
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}

func TestEmbeddedSchemasAcceptAndRejectContractFixtures(t *testing.T) {
	schemas, err := compileSchemas()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		file, schema string
		valid        bool
	}{
		{"runtime-engine-config.valid.json", "runtime-engine-config.schema.json", true}, {"runtime-engine-config.invalid.json", "runtime-engine-config.schema.json", false}, {"runtime-engine-config.invalid.event-consumer.json", "runtime-engine-config.schema.json", false},
		{"runtime-engine-config-v2.valid.json", "runtime-engine-config-v2.schema.json", true}, {"runtime-engine-config-v2.invalid-writable.json", "runtime-engine-config-v2.schema.json", false},
		{"runtime-dlq-event.valid.json", "runtime-dlq-event.schema.json", true}, {"runtime-dlq-event.invalid.json", "runtime-dlq-event.schema.json", false},
		{"runtime-project-artifact.valid.json", "runtime-project-artifact.schema.json", true}, {"runtime-project-artifact.invalid.json", "runtime-project-artifact.schema.json", false}, {"runtime-project-artifact.invalid-derived.json", "runtime-project-artifact.schema.json", false}, {"runtime-project-artifact.invalid.schedule.json", "runtime-project-artifact.schema.json", false},
		{"point-event.valid.raw.json", "point-event.schema.json", true}, {"point-event.valid.computed.json", "point-event.schema.json", true}, {"point-event.invalid.json", "point-event.schema.json", false}, {"point-event.invalid.raw-computation.json", "point-event.schema.json", false}, {"point-event.invalid.computed-source.json", "point-event.schema.json", false},
		{"alarm-event.valid.json", "alarm-event.schema.json", true}, {"alarm-event.valid.data-gap.json", "alarm-event.schema.json", true}, {"alarm-event.invalid.json", "alarm-event.schema.json", false}, {"alarm-event.invalid-clear.json", "alarm-event.schema.json", false}, {"alarm-event.invalid-severity-change.json", "alarm-event.schema.json", false},
		{"collector-runtime-artifact.valid.json", "collector-runtime-artifact.schema.json", true}, {"collector-runtime-artifact.invalid.json", "collector-runtime-artifact.schema.json", false}, {"collector-runtime-artifact.invalid.inherit-overrides.json", "collector-runtime-artifact.schema.json", false}, {"collector-runtime-artifact.invalid.object-data-type.json", "collector-runtime-artifact.schema.json", false},
		{"collector-runtime-binding.valid.json", "collector-runtime-binding.schema.json", true}, {"collector-runtime-binding.valid.no-secret.json", "collector-runtime-binding.schema.json", true}, {"collector-runtime-binding.invalid.json", "collector-runtime-binding.schema.json", false}, {"collector-runtime-binding.invalid.missing-wal.json", "collector-runtime-binding.schema.json", false},
		{"runtime-health-status.valid.json", "runtime-health-status.schema.json", true}, {"runtime-health-status.valid.compute-sandbox.json", "runtime-health-status.schema.json", true}, {"runtime-health-status.valid.native-engine.json", "runtime-health-status.schema.json", true}, {"runtime-health-status.valid.native-project-gateway.json", "runtime-health-status.schema.json", true}, {"runtime-health-status.valid.native-runtime-api.json", "runtime-health-status.schema.json", true}, {"runtime-health-status.invalid.native-engine-missing-process.json", "runtime-health-status.schema.json", false}, {"runtime-health-status.invalid.json", "runtime-health-status.schema.json", false}, {"runtime-health-status.invalid.collector-missing.json", "runtime-health-status.schema.json", false}, {"runtime-health-status.invalid.engine-missing-deployment.json", "runtime-health-status.schema.json", false}, {"runtime-health-status.invalid.noncollector-details.json", "runtime-health-status.schema.json", false},
	}
	for _, test := range cases {
		t.Run(test.file, func(t *testing.T) {
			err := validateJSON(schemas.all[test.schema], fixture(t, test.file), test.file)
			if test.valid && err != nil {
				t.Fatal(err)
			}
			if !test.valid && err == nil {
				t.Fatal("invalid fixture was accepted")
			}
		})
	}
}

func TestSchemaValidationPreservesExactNumbers(t *testing.T) {
	schemas, err := compileSchemas()
	if err != nil {
		t.Fatal(err)
	}
	artifact := strings.Replace(string(fixture(t, "runtime-project-artifact.valid.json")), `"defaultValue": null`, `"defaultValue": 1e1000`, 1)
	if err := validateJSON(schemas.projectArtifact, []byte(artifact), "large-exponent artifact"); err != nil {
		t.Fatalf("schema rejected exact large exponent: %v", err)
	}
	if err := validateJSON(schemas.projectArtifact, []byte(artifact+` {}`), "trailing artifact"); err == nil {
		t.Fatal("schema accepted trailing JSON")
	}
}

func TestProjectArtifactSchemaAlarmItemsBound(t *testing.T) {
	schemas, err := compileSchemas()
	if err != nil {
		t.Fatal(err)
	}
	var artifact map[string]any
	if err := json.Unmarshal(fixture(t, "runtime-project-artifact.valid.json"), &artifact); err != nil {
		t.Fatal(err)
	}
	template := artifact["alarmItems"].([]any)[0]
	items := make([]any, model.MaxAlarmItems)
	for index := range items {
		items[index] = template
	}
	artifact["alarmItems"] = items
	boundary, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateJSON(schemas.projectArtifact, boundary, "alarmItems boundary"); err != nil {
		t.Fatalf("boundary %d rejected: %v", model.MaxAlarmItems, err)
	}
	artifact["alarmItems"] = append(items, template)
	over, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateJSON(schemas.projectArtifact, over, "alarmItems max+1"); err == nil {
		t.Fatal("max+1 alarmItems accepted")
	}
}

func TestProjectArtifactSourceDiscriminatorSchema(t *testing.T) {
	schemas, err := compileSchemas()
	if err != nil {
		t.Fatal(err)
	}
	load := func(t *testing.T) map[string]any {
		t.Helper()
		var artifact map[string]any
		if err := json.Unmarshal(fixture(t, "runtime-project-artifact.valid.json"), &artifact); err != nil {
			t.Fatal(err)
		}
		return artifact
	}
	point := func(t *testing.T, artifact map[string]any, sourceType string) map[string]any {
		t.Helper()
		for _, raw := range artifact["dataPoints"].([]any) {
			candidate := raw.(map[string]any)
			if candidate["sourceType"] == sourceType {
				return candidate
			}
		}
		t.Fatalf("fixture missing %s", sourceType)
		return nil
	}
	assertRejected := func(t *testing.T, name string, mutate func(map[string]any)) {
		t.Helper()
		artifact := load(t)
		mutate(artifact)
		raw, err := json.Marshal(artifact)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateJSON(schemas.projectArtifact, raw, name); err == nil {
			t.Fatalf("%s was accepted by schema", name)
		}
	}
	assertRejected(t, "unknown sourceType", func(artifact map[string]any) {
		point(t, artifact, "manual.input")["sourceType"] = "manual"
	})
	assertRejected(t, "manual non-empty sourceConfig", func(artifact map[string]any) {
		point(t, artifact, "manual.input")["sourceConfig"] = map[string]any{"x": true}
	})
	assertRejected(t, "manual sourceId", func(artifact map[string]any) {
		point(t, artifact, "manual.input")["sourceId"] = "aaaaaaaa-1111-4111-8111-111111111111"
	})
	assertRejected(t, "collector endpoint", func(artifact map[string]any) {
		point(t, artifact, "collector.point")["sourceConfig"] = map[string]any{"connectionId": "aaaaaaaa-1111-4111-8111-111111111111", "endpoint": "tcp://unsafe"}
	})
	assertRejected(t, "calc extra field", func(artifact map[string]any) {
		point(t, artifact, "calc.output")["sourceConfig"] = map[string]any{"computeId": "44444444-4444-4444-8444-444444444444", "outputKey": "result"}
	})
	if err := rejectDuplicateJSONKeys([]byte(`{"sourceConfig":{"connectionId":"aaaaaaaa-1111-4111-8111-111111111111","connectionId":"aaaaaaaa-1111-4111-8111-111111111111"}}`)); err == nil {
		t.Fatal("nested duplicate sourceConfig key was accepted")
	}
}

func TestEmbeddedSchemaSnapshotMatchesContractSource(t *testing.T) {
	entries, err := os.ReadDir("schemas/runtime")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		embedded, err := os.ReadFile(filepath.Join("schemas/runtime", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		source, err := os.ReadFile(filepath.Join("../../../../contracts/runtime", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if string(embedded) != string(source) {
			t.Fatalf("embedded Schema %s is not synchronized with contracts/runtime", entry.Name())
		}
	}
}

func TestLoadValidArtifactAndRejectDigestOrSymlinkEscape(t *testing.T) {
	temporary, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mount := filepath.Join(temporary, "release")
	if err := os.Mkdir(mount, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := fixture(t, "runtime-project-artifact.valid.json")
	if err := os.WriteFile(filepath.Join(mount, "artifact.json"), artifact, 0o600); err != nil {
		t.Fatal(err)
	}
	config := fixtureConfig(t, mount, "artifact.json", artifact)
	configPath := filepath.Join(temporary, "config.json")
	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(testOptions(configPath, temporary))
	if err != nil {
		t.Fatalf("valid load failed: %v", err)
	}
	if !strings.HasPrefix(loaded.ConfigSHA256, "sha256:") || !strings.HasPrefix(loaded.ArtifactSHA256, "sha256:") {
		t.Fatalf("raw byte digests not recorded: %+v", loaded)
	}
	if artifact, ok := loaded.CollectorArtifacts["collector-line1-a"]; !ok || len(artifact.PointMappings) != 1 {
		t.Fatalf("collector artifact was not loaded and verified: %+v", loaded.CollectorArtifacts)
	}

	var broken map[string]any
	if err := json.Unmarshal(config, &broken); err != nil {
		t.Fatal(err)
	}
	broken["projectArtifact"].(map[string]any)["artifactDigest"] = "sha256:" + strings.Repeat("0", 64)
	brokenBytes, err := json.Marshal(broken)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, brokenBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(testOptions(configPath, temporary)); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("expected digest error, got %v", err)
	}

	outside := filepath.Join(temporary, "outside.json")
	if err := os.WriteFile(outside, artifact, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(mount, "escape.json")); err != nil {
		t.Fatal(err)
	}
	escaping := fixtureConfig(t, mount, "escape.json", artifact)
	if err := os.WriteFile(configPath, escaping, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(testOptions(configPath, temporary)); err == nil || (!strings.Contains(err.Error(), "逃逸") && !strings.Contains(err.Error(), "符号链接")) {
		t.Fatalf("expected escaping path rejection, got %v", err)
	}

	// Collector Artifact 与 Project Artifact 使用同一安全装载路径，不能以 symlink 绕过受信 release-pvc 根。
	config = fixtureConfig(t, mount, "artifact.json", artifact)
	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatal(err)
	}
	collectorPath := filepath.Join(mount, "collector", "collector-artifact.json")
	collectorOutside := filepath.Join(temporary, "collector-outside.json")
	if err := os.WriteFile(collectorOutside, fixture(t, "collector-runtime-artifact.valid.json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(collectorPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(collectorOutside, collectorPath); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(testOptions(configPath, temporary)); err == nil || !strings.Contains(err.Error(), "符号链接") {
		t.Fatalf("expected collector symlink rejection, got %v", err)
	}
}

func TestLoadNativeV2RequiresReadOnlyReleaseRootAndRejectsSymlink(t *testing.T) {
	temporary, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	releaseRoot := filepath.Join(temporary, "release-v2")
	if err := os.Mkdir(releaseRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := fixture(t, "runtime-project-artifact.valid.json")
	if err := os.WriteFile(filepath.Join(releaseRoot, "artifact.json"), artifact, 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(temporary, "config-v2.json")
	if err := os.WriteFile(configPath, fixtureConfigFrom(t, "runtime-engine-config-v2.valid.json", releaseRoot, "artifact.json", artifact), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(testOptions(configPath, temporary))
	if err != nil {
		t.Fatalf("native v2 load failed: %v", err)
	}
	if loaded.Config.SchemaVersion != "runtime-engine.config.v2" || loaded.Config.NodeID != "node-line1-01" {
		t.Fatalf("native config was not selected: %+v", loaded.Config)
	}

	mountInfoPath := filepath.Join(temporary, "mountinfo")
	if err := os.WriteFile(mountInfoPath, []byte("36 25 0:32 / "+releaseRoot+" rw - tmpfs tmpfs rw\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	requireReadOnly := true
	if _, err := Load(Options{ConfigPath: configPath, ConfigRoot: temporary, RequireReadOnlyMount: &requireReadOnly, MountInfoPath: mountInfoPath}); err == nil || !strings.Contains(err.Error(), "native-release 根必须为只读") {
		t.Fatalf("writable native release root was accepted: %v", err)
	}

	linkedRoot := filepath.Join(temporary, "release-link")
	if err := os.Symlink(releaseRoot, linkedRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, fixtureConfigFrom(t, "runtime-engine-config-v2.valid.json", linkedRoot, "artifact.json", artifact), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(testOptions(configPath, temporary)); err == nil || !strings.Contains(err.Error(), "符号链接") {
		t.Fatalf("symlink native release root was accepted: %v", err)
	}
}

func testOptions(configPath, configRoot string) Options {
	requireReadOnlyMount := false
	return Options{ConfigPath: configPath, ConfigRoot: configRoot, RequireReadOnlyMount: &requireReadOnlyMount}
}

func TestStrictArtifactPaths(t *testing.T) {
	for _, path := range []string{"relative", "/opt//release", "/opt/../release", "/opt/./release", "/opt/\x00release"} {
		if err := validateMountPath(path); err == nil {
			t.Fatalf("mount path %q was accepted", path)
		}
	}
	for _, path := range []string{"/absolute.json", "../artifact.json", "dir//artifact.json", "dir/./artifact.json", "dir/\x00artifact.json"} {
		if err := validateArtifactFile(path); err == nil {
			t.Fatalf("artifact file %q was accepted", path)
		}
	}
}

func TestReadRegularFileRejectsDuplicateKeysAndInsideRootSymlink(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	duplicate := filepath.Join(root, "duplicate.json")
	if err := os.WriteFile(duplicate, []byte(`{"a":1,"a":2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegularFile(duplicate, maxConfigBytes, "test"); err == nil || !strings.Contains(err.Error(), "重复键") {
		t.Fatalf("expected duplicate key rejection, got %v", err)
	}
	target := filepath.Join(root, "target.json")
	if err := os.WriteFile(target, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(root, "linked.json")
	if err := os.Symlink(target, linked); err != nil {
		t.Fatal(err)
	}
	if _, err := requireAbsoluteExistingFile(linked, "test"); err == nil || !strings.Contains(err.Error(), "符号链接") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

func fixtureConfig(t *testing.T, mount, artifactFile string, artifact []byte) []byte {
	return fixtureConfigFrom(t, "runtime-engine-config.valid.json", mount, artifactFile, artifact)
}

func fixtureConfigFrom(t *testing.T, fixtureName, mount, artifactFile string, artifact []byte) []byte {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(fixture(t, fixtureName), &value); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(artifact)
	value["projectArtifact"].(map[string]any)["artifactDigest"] = "sha256:" + hex.EncodeToString(digest[:])
	value["artifactMount"].(map[string]any)["mountPath"] = mount
	value["artifactMount"].(map[string]any)["artifactFile"] = artifactFile
	collectorArtifact := fixture(t, "collector-runtime-artifact.valid.json")
	collectorBinding := value["producerAssignments"].([]any)[0].(map[string]any)["collectorArtifact"].(map[string]any)
	collectorDigest := sha256.Sum256(collectorArtifact)
	collectorBinding["artifact"].(map[string]any)["artifactDigest"] = "sha256:" + hex.EncodeToString(collectorDigest[:])
	collectorFile := collectorBinding["artifactFile"].(string)
	if err := os.MkdirAll(filepath.Dir(filepath.Join(mount, collectorFile)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mount, collectorFile), collectorArtifact, 0o600); err != nil {
		t.Fatal(err)
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}
