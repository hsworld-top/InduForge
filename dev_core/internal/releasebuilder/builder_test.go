package releasebuilder

import (
	"archive/tar"
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestBuildProducesDeterministicSignedFormalRelease(t *testing.T) {
	input := validInput(t)
	first, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Bundle, second.Bundle) || first.OuterSHA256 != second.OuterSHA256 {
		t.Fatal("same controlled inputs must produce the same Release bundle")
	}
	if got, want := tarEntryNames(t, first.Bundle), []string{checksumsFile, clientArtifactFile, healthContractFile, manifestFile, resourcePlanFile, runtimeArtifactFile, sbomFile, schemaPlanFile, signatureFile}; !sameStringSlice(got, want) {
		t.Fatalf("outer tar order is not deterministic: got %v want %v", got, want)
	}
	if first.Size != int64(len(first.Bundle)) || first.OuterSHA256 != sha256Hex(first.Bundle) || first.ManifestSHA256 != sha256Hex(first.Manifest) || first.ChecksumsSHA256 != sha256Hex(first.Checksums) {
		t.Fatalf("returned Release digests or size are inconsistent: %#v", first)
	}
	if len(first.Signature) != ed25519.SignatureSize || !ed25519.Verify(input.SigningKey.Public().(ed25519.PublicKey), first.Checksums, first.Signature) {
		t.Fatal("checksums signature is invalid")
	}

	files := unpack(t, first.Bundle)
	wantNames := []string{checksumsFile, clientArtifactFile, healthContractFile, manifestFile, resourcePlanFile, runtimeArtifactFile, sbomFile, schemaPlanFile, signatureFile}
	if got := sortedNames(files); !sameStringSlice(got, wantNames) {
		t.Fatalf("unexpected fixed Release layout: got %v want %v", got, wantNames)
	}
	if !bytes.Equal(files[manifestFile], first.Manifest) || !bytes.Equal(files[checksumsFile], first.Checksums) || !bytes.Equal(files[signatureFile], first.Signature) {
		t.Fatal("bundle does not contain returned signed metadata")
	}

	var manifest map[string]any
	if err := json.Unmarshal(first.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["schemaVersion"] != "2.0" || manifest["projectId"] != input.ProjectID || manifest["releaseId"] != input.ReleaseID {
		t.Fatalf("formal manifest identity invalid: %s", first.Manifest)
	}
	supplyChain := manifest["supplyChain"].(map[string]any)
	if supplyChain["signingKeyId"] != input.SigningKeyID || supplyChain["sourceRevision"] != input.SourceRevision {
		t.Fatalf("formal manifest supply-chain fields invalid: %s", first.Manifest)
	}
	var checksums checksumsDocument
	if err := json.Unmarshal(first.Checksums, &checksums); err != nil {
		t.Fatal(err)
	}
	if checksums.SchemaVersion != "release-checksums.v1" || len(checksums.Files) != 7 {
		t.Fatalf("unexpected checksums manifest: %s", first.Checksums)
	}
	for index, entry := range checksums.Files {
		if entry.Path == checksumsFile || entry.Path == signatureFile || !strings.HasPrefix(entry.SHA256, "sha256:") {
			t.Fatalf("checksums contains a forbidden entry: %#v", entry)
		}
		payload, ok := files[entry.Path]
		if !ok || entry.Size != int64(len(payload)) || entry.SHA256 != "sha256:"+sha256Hex(payload) {
			t.Fatalf("checksums entry does not bind the actual bundle content: %#v", entry)
		}
		if index > 0 && checksums.Files[index-1].Path >= entry.Path {
			t.Fatalf("checksums entries are not strictly sorted: %s", first.Checksums)
		}
	}
}

func TestBuildFailsClosedForTamperingAndSecretLeakage(t *testing.T) {
	input := validInput(t)
	result, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	tamperedChecksums := append([]byte(nil), result.Checksums...)
	tamperedChecksums[0] ^= 1
	if ed25519.Verify(input.SigningKey.Public().(ed25519.PublicKey), tamperedChecksums, result.Signature) {
		t.Fatal("tampered checksums must not retain a valid signature")
	}

	privateMaterial := hex.EncodeToString(input.SigningKey)
	for name, data := range unpack(t, result.Bundle) {
		if strings.Contains(string(data), privateMaterial) || bytes.Contains(data, input.SigningKey) {
			t.Fatalf("private key leaked into Release file %s", name)
		}
	}
	if _, err := Build(Input{}); err == nil {
		t.Fatal("empty or untrusted input must be rejected")
	}
}

func TestBuildOptionalCollectorAndInputValidation(t *testing.T) {
	input := validInput(t)
	input.Collector = []byte("collector-artifact")
	input.CollectorSourceSnapshot = validCollectorSnapshotForTest(t, input.ProjectID)
	input.RequiredNodeCapabilities = []string{"collector", "data_runtime", "project_entry"}
	result, err := Build(input)
	if err != nil {
		t.Fatal(err)
	}
	files := unpack(t, result.Bundle)
	if !bytes.Equal(files[collectorArtifact], input.Collector) {
		t.Fatal("collector artifact missing from optional collector Release")
	}
	var checksums checksumsDocument
	if err := json.Unmarshal(result.Checksums, &checksums); err != nil || len(checksums.Files) != 8 {
		t.Fatalf("collector must be included in signed checksums: %v %s", err, result.Checksums)
	}

	input.RequiredNodeCapabilities = []string{"project_entry", "data_runtime"}
	if _, err := Build(input); err == nil || !strings.Contains(err.Error(), "requiredNodeCapabilities") {
		t.Fatalf("collector capability mismatch must fail closed: %v", err)
	}
	input = validInput(t)
	input.ProjectID = "not-a-uuid"
	if _, err := Build(input); err == nil {
		t.Fatal("malformed UUID must be rejected")
	}
	input = validInput(t)
	input.SBOM = nil
	if _, err := Build(input); err == nil || !strings.Contains(err.Error(), sbomFile) {
		t.Fatalf("missing fixed preflight file must be rejected: %v", err)
	}
	input = validInput(t)
	input.HealthContract = []byte("not-json")
	if _, err := Build(input); err == nil || !strings.Contains(err.Error(), healthContractFile) {
		t.Fatalf("malformed fixed JSON input must be rejected: %v", err)
	}
}

func validCollectorSnapshotForTest(t *testing.T, projectID string) []byte {
	t.Helper()
	artifact := []byte(`{"schemaVersion":"collector-runtime-artifact.v1","artifactId":"collector-a","artifactRevision":1,"projectId":"` + projectID + `"}`)
	sum := sha256.Sum256(artifact)
	raw, err := json.Marshal(map[string]any{"schemaVersion": "collector-runtime-artifact.v1", "projectId": projectID, "artifactRevision": 1, "sha256": "sha256:" + hex.EncodeToString(sum[:]), "size": len(artifact), "artifact": json.RawMessage(artifact)})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestBuildFailsClosedForNodeAgentSizeLimits(t *testing.T) {
	input := validInput(t)
	if _, err := buildWithLimits(input, buildLimits{archiveBytes: 1024, unpackedBytes: 80, fileBytes: 64}); err == nil || !strings.Contains(err.Error(), "解包总量") {
		t.Fatalf("raw Release total over the NodeAgent limit must fail before packing: %v", err)
	}
	if _, err := buildWithLimits(input, buildLimits{archiveBytes: 1, unpackedBytes: 4096, fileBytes: 4096}); err == nil || !strings.Contains(err.Error(), "外层归档") {
		t.Fatalf("outer Release over the NodeAgent limit must fail after packing: %v", err)
	}
	input = validInput(t)
	input.Client = bytes.Repeat([]byte{'x'}, 17)
	if _, err := buildWithLimits(input, buildLimits{archiveBytes: 1024, unpackedBytes: 1024, fileBytes: 16}); err == nil || !strings.Contains(err.Error(), clientArtifactFile) {
		t.Fatalf("single file over the NodeAgent limit must fail closed: %v", err)
	}
}

func TestBuildRequiresStrictCompatibilitySemVer(t *testing.T) {
	for _, version := range []string{"1.0", "1.0.0.1", "01.0.0", "1.01.0", "1.0.00", "1.0.0-01", "v1.0.0", "1.0.0+"} {
		input := validInput(t)
		input.MinNodeAgentVersion = version
		if _, err := Build(input); err == nil || !strings.Contains(err.Error(), "最小 NodeAgent") {
			t.Fatalf("invalid minimum NodeAgent SemVer %q must be rejected: %v", version, err)
		}
	}
	for _, version := range []string{"1.0.0", "1.2.3-rc.1", "1.2.3+build.7", "1.2.3-rc.1+build.7"} {
		input := validInput(t)
		input.MinRuntimeVersion = version
		if _, err := Build(input); err != nil {
			t.Fatalf("valid minimum Runtime SemVer %q rejected: %v", version, err)
		}
	}
}

// ReleaseBuilder 是正式清单的唯一生产者；这里同时固定与 NodeAgent 共用的
// SemVer 约束，以及激活快照中 Collector 端口的条件约束，防止三端各自漂移。
func TestFormalRuntimeContractFixtures(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位 ReleaseBuilder 契约测试目录")
	}
	contractRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "../../../contracts/runtime"))
	tests := []struct {
		name, schema, fixture string
		valid                 bool
	}{
		{"Release 兼容版本允许 build metadata", "release-manifest-v2.schema.json", "release-manifest-v2.valid.json", true},
		{"Release 拒绝非 SemVer 兼容版本", "release-manifest-v2.schema.json", "release-manifest-v2.invalid-compatibility-semver.json", false},
		{"未启用 Collector 不含其端口", "runtime-activation-v1.schema.json", "runtime-activation-v1.valid.json", true},
		{"未启用 Collector 禁止其端口", "runtime-activation-v1.schema.json", "runtime-activation-v1.invalid-collector-port-without-service.json", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			compiler := jsonschema.NewCompiler()
			compiler.DefaultDraft(jsonschema.Draft2020)
			const contractBase = "https://induforge.dev/contracts/runtime/"
			for _, name := range []string{"common.schema.json", test.schema} {
				if err := compiler.AddResource(contractBase+name, readRuntimeContractJSON(t, filepath.Join(contractRoot, name))); err != nil {
					t.Fatalf("注册契约 %s: %v", name, err)
				}
			}
			schema, err := compiler.Compile(contractBase + test.schema)
			if err != nil {
				t.Fatalf("编译契约 %s: %v", test.schema, err)
			}
			err = schema.Validate(readRuntimeContractJSON(t, filepath.Join(contractRoot, "fixtures", test.fixture)))
			if test.valid && err != nil {
				t.Fatalf("合法样例未通过: %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("非法样例被契约接受")
			}
		})
	}
	t.Run("启用 Collector 必须给出健康端口", func(t *testing.T) {
		compiler := jsonschema.NewCompiler()
		compiler.DefaultDraft(jsonschema.Draft2020)
		const contractBase = "https://induforge.dev/contracts/runtime/"
		for _, name := range []string{"common.schema.json", "runtime-activation-v1.schema.json"} {
			if err := compiler.AddResource(contractBase+name, readRuntimeContractJSON(t, filepath.Join(contractRoot, name))); err != nil {
				t.Fatalf("注册契约 %s: %v", name, err)
			}
		}
		schema, err := compiler.Compile(contractBase + "runtime-activation-v1.schema.json")
		if err != nil {
			t.Fatal(err)
		}
		document := readRuntimeContractJSON(t, filepath.Join(contractRoot, "fixtures", "runtime-activation-v1.valid.json")).(map[string]any)
		document["components"] = append(document["components"].([]any), map[string]any{
			"name": "collector", "serviceGroup": "collector", "enabled": true, "dependsOn": []any{},
		})
		document["artifacts"].(map[string]any)["collector"] = map[string]any{
			"releaseFile": "collector-artifact.tar.zst", "digest": "sha256:8888888888888888888888888888888888888888888888888888888888888888", "materializedLayout": "collector", "readOnlyMount": true,
		}
		document["secrets"] = append(document["secrets"].([]any), map[string]any{
			"name": "collector-nats", "ref": "secret-collector-nats", "schemaVersion": "runtime-nats-credentials.v1", "sha256": "sha256:9999999999999999999999999999999999999999999999999999999999999999", "revision": 1, "expiresAt": "2026-09-01T10:00:00Z",
		})
		if err := schema.Validate(document); err == nil {
			t.Fatal("启用 Collector 但缺少 collectorHealthLoopback 被接受")
		}
		document["ports"].(map[string]any)["collectorHealthLoopback"] = float64(17803)
		if err := schema.Validate(document); err != nil {
			t.Fatalf("启用 Collector 的完整激活快照未通过: %v", err)
		}
	})
}

func readRuntimeContractJSON(t *testing.T, path string) any {
	t.Helper()
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取契约 %s: %v", path, err)
	}
	var document any
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatalf("解析契约 %s: %v", path, err)
	}
	return document
}

func TestBuildAllowsOnlyInternalDevelopmentVersionMarker(t *testing.T) {
	development := validInput(t)
	development.Version = "__DEV__"
	if _, err := Build(development); err != nil {
		t.Fatalf("internal development marker must build: %v", err)
	}
	invalid := validInput(t)
	invalid.Version = "__OTHER__"
	if _, err := Build(invalid); err == nil || !strings.Contains(err.Error(), "projectCode 或 version") {
		t.Fatalf("arbitrary underscore-prefixed version must remain rejected: %v", err)
	}
}

func validInput(t *testing.T) Input {
	t.Helper()
	seed := bytes.Repeat([]byte{0x42}, ed25519.SeedSize)
	return Input{
		ProjectID:                "11111111-1111-4111-8111-111111111111",
		ProjectCode:              "factory_dashboard",
		ReleaseID:                "22222222-2222-4222-8222-222222222222",
		Version:                  "2026.08.31-001",
		Client:                   []byte("client-artifact"),
		Runtime:                  []byte("runtime-artifact"),
		SBOM:                     []byte(`{"bomFormat":"CycloneDX"}`),
		ResourceRecommendation:   []byte(`{"cpu":"1"}`),
		HealthContract:           []byte(`{"health":"ok"}`),
		SchemaPlan:               []byte(`{"changes":[]}`),
		Capabilities:             []string{"runtime.scene", "runtime.auth"},
		MinNodeAgentVersion:      "1.0.0",
		MinRuntimeVersion:        "1.0.0",
		RequiredNodeCapabilities: []string{"data_runtime", "project_entry"},
		SourceRevision:           "git:" + strings.Repeat("a", 40),
		BuilderID:                "induforge-release-builder-v1",
		Promotable:               true,
		BuildTime:                time.Date(2026, 8, 31, 8, 0, 0, 0, time.FixedZone("CST", 8*3600)),
		SigningKey:               ed25519.NewKeyFromSeed(seed),
		SigningKeyID:             "induforge-release-2026-01",
	}
}

func unpack(t *testing.T, bundle []byte) map[string][]byte {
	t.Helper()
	decoder, err := zstd.NewReader(bytes.NewReader(bundle))
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()
	reader := tar.NewReader(decoder)
	files := map[string][]byte{}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return files
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		files[header.Name] = data
	}
}

func tarEntryNames(t *testing.T, bundle []byte) []string {
	t.Helper()
	decoder, err := zstd.NewReader(bytes.NewReader(bundle))
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()
	reader := tar.NewReader(decoder)
	var names []string
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return names
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, header.Name)
	}
}

func sortedNames(files map[string][]byte) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sameStringSlice(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sha256Hex(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}
