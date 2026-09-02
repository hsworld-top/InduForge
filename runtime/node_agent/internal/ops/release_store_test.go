package ops

import (
	"archive/tar"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/klauspost/compress/zstd"
)

type releaseTestEntry struct {
	name     string
	typeFlag byte
	data     []byte
	link     string
}

const (
	testReleaseProjectID = "11111111-1111-4111-8111-111111111111"
	testReleaseID        = "22222222-2222-4222-8222-222222222222"
)

type signedReleaseFixture struct {
	archive         []byte
	manifestDigest  string
	checksumsDigest string
}

func (f signedReleaseFixture) installInput(publicKey ed25519.PublicKey, keyID string) ReleaseInstallInput {
	return ReleaseInstallInput{
		Archive:                 bytes.NewReader(f.archive),
		ExpectedOuterSHA256:     sha256Digest(f.archive),
		ExpectedManifestSHA256:  f.manifestDigest,
		ExpectedChecksumsSHA256: f.checksumsDigest,
		ExpectedProjectID:       testReleaseProjectID,
		ExpectedReleaseID:       testReleaseID,
		ExpectedVersion:         "2026.08.31-001",
		NodeAgentVersion:        "1.0.0",
		RuntimeVersion:          "1.0.0",
		NodeCapabilities:        []string{"project_entry", "data_runtime"},
		BoundServices:           []string{"project_entry", "data_runtime"},
		VerificationPublicKey:   publicKey,
		KeyID:                   keyID,
	}
}

func TestReleaseInstallerVerifiesAndAtomicallyActivates(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	archive := signedReleaseArchive(t, privateKey, "release-key-a", map[string][]byte{
		"runtime-artifact.tar.zst": []byte("runtime-v1"),
		"client-assets.tar.zst":    []byte("client-v1"),
	}, nil)
	store, err := NewReleaseStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { unsealReleaseTree(store.root) })
	installer, err := NewReleaseInstaller(store, ReleaseInstallLimits{MaxArchiveBytes: 1 << 20, MaxFiles: 16, MaxFileBytes: 1 << 20, MaxTotalBytes: 1 << 20, MaxDecoderBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256Digest(archive.archive)
	installed, err := installer.Install(archive.installInput(publicKey, "release-key-a"))
	if err != nil {
		t.Fatal(err)
	}
	if installed.ReleaseDigest != digest || filepath.Base(installed.ReleaseDir) != "sha256-"+strings.TrimPrefix(digest, "sha256:") {
		t.Fatalf("unexpected installed release: %+v", installed)
	}
	if info, err := os.Stat(filepath.Join(installed.ReleaseDir, "release-manifest.json")); err != nil || info.Mode().Perm()&0222 != 0 {
		t.Fatalf("installed Release must be sealed read-only: info=%v err=%v", info, err)
	}
	if info, err := os.Stat(installed.ReleaseDir); err != nil || info.Mode().Perm() != 0555 {
		t.Fatalf("installed Release directory mode=%v err=%v", info, err)
	}
	if info, err := os.Stat(filepath.Join(installed.ReleaseDir, "runtime-artifact.tar.zst")); err != nil || info.Mode().Perm() != 0444 {
		t.Fatalf("runtime artifact mode=%v err=%v", info, err)
	}
	pointerRaw, err := os.ReadFile(filepath.Join(store.root, "current.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pointer releasePointer
	if err := json.Unmarshal(pointerRaw, &pointer); err != nil {
		t.Fatal(err)
	}
	if pointer.SchemaVersion != releasePointerSchema || pointer.ReleaseDigest != digest || pointer.ReleaseDir != "releases/sha256-"+strings.TrimPrefix(digest, "sha256:") || pointer.ActivatedAt == "" {
		t.Fatalf("unexpected pointer: %+v", pointer)
	}
}

func TestReleaseInstallerFailureDoesNotActivate(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewReleaseStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { unsealReleaseTree(store.root) })
	installer, err := NewReleaseInstaller(store, ReleaseInstallLimits{MaxArchiveBytes: 1 << 20, MaxFiles: 16, MaxFileBytes: 1 << 20, MaxTotalBytes: 1 << 20, MaxDecoderBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	good := signedReleaseArchive(t, privateKey, "release-key-a", map[string][]byte{"runtime-artifact.tar.zst": []byte("good")}, nil)
	goodDigest := sha256Digest(good.archive)
	if _, err := installer.Install(good.installInput(publicKey, "release-key-a")); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	bad := signedReleaseArchive(t, privateKey, "release-key-a", map[string][]byte{"runtime-artifact.tar.zst": []byte("bad")}, nil)
	bad.archive[len(bad.archive)-1] ^= 0x01
	badInput := bad.installInput(publicKey, "release-key-a")
	badInput.ExpectedOuterSHA256 = goodDigest
	if _, err := installer.Install(badInput); err == nil {
		t.Fatal("corrupted archive must be rejected")
	}
	after, err := os.ReadFile(store.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("failed installation must not alter current.json")
	}
}

func TestReleaseInstallerRejectsIncompatibleManifestBeforeActivation(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	archive := signedReleaseArchive(t, privateKey, "release-key-a", nil, nil)
	for name, mutate := range map[string]func(*ReleaseInstallInput){
		"version":      func(input *ReleaseInstallInput) { input.ExpectedVersion = "2026.08.31-002" },
		"node agent":   func(input *ReleaseInstallInput) { input.NodeAgentVersion = "0.9.0" },
		"runtime":      func(input *ReleaseInstallInput) { input.RuntimeVersion = "0.9.0" },
		"capabilities": func(input *ReleaseInstallInput) { input.NodeCapabilities = []string{"project_entry"} },
		"binding":      func(input *ReleaseInstallInput) { input.BoundServices = []string{"project_entry"} },
	} {
		t.Run(name, func(t *testing.T) {
			store, err := NewReleaseStore(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { unsealReleaseTree(store.root) })
			installer, err := NewReleaseInstaller(store, ReleaseInstallLimits{MaxArchiveBytes: 1 << 20, MaxFiles: 16, MaxFileBytes: 1 << 20, MaxTotalBytes: 1 << 20, MaxDecoderBytes: 8 << 20})
			if err != nil {
				t.Fatal(err)
			}
			input := archive.installInput(publicKey, "release-key-a")
			mutate(&input)
			if _, err := installer.Install(input); err == nil {
				t.Fatal("incompatible Release was activated")
			}
			if _, err := os.Stat(store.currentPath); !os.IsNotExist(err) {
				t.Fatalf("incompatible Release must not write current.json: %v", err)
			}
		})
	}
}

func TestDevelopmentVersionIsOnlyAcceptedForInternalFixedTag(t *testing.T) {
	if !validVersionValue("__DEV__") {
		t.Fatal("固定内部开发标签必须可安装")
	}
	if validVersionValue("_user-version") {
		t.Fatal("不得放宽用户可输入版本的首字符约束")
	}
}

func TestCompatibilityVersionUsesStrictSemVerPrecedence(t *testing.T) {
	tests := []struct {
		name            string
		actual, minimum string
		want            bool
	}{
		{name: "beta.2 before beta.10", actual: "1.0.0-beta.2", minimum: "1.0.0-beta.10", want: false},
		{name: "beta.10 after beta.2", actual: "1.0.0-beta.10", minimum: "1.0.0-beta.2", want: true},
		{name: "release after prerelease", actual: "1.0.0", minimum: "1.0.0-rc.1", want: true},
		{name: "prerelease before release", actual: "1.0.0-rc.1", minimum: "1.0.0", want: false},
		{name: "build metadata ignored", actual: "1.2.3+build.1", minimum: "1.2.3+build.99", want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := versionAtLeast(test.actual, test.minimum)
			if err != nil || got != test.want {
				t.Fatalf("versionAtLeast(%q, %q) = %v, %v; want %v", test.actual, test.minimum, got, err, test.want)
			}
		})
	}
	for _, value := range []string{"1.02.3", "1.2", "1.2.3+", "1.2.3-beta.01", "v1.2.3", " 1.2.3", "not-a-version"} {
		if validComparableVersion(value) {
			t.Fatalf("invalid or non-canonical SemVer was accepted: %q", value)
		}
	}
}

func TestReleaseInstallerConcurrentActivationUsesIndependentPointers(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewReleaseStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { unsealReleaseTree(store.root) })
	installer, err := NewReleaseInstaller(store, ReleaseInstallLimits{MaxArchiveBytes: 1 << 20, MaxFiles: 16, MaxFileBytes: 1 << 20, MaxTotalBytes: 1 << 20, MaxDecoderBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	first := signedReleaseArchive(t, privateKey, "release-key-a", map[string][]byte{"runtime-artifact.tar.zst": []byte("first")}, nil)
	second := signedReleaseArchive(t, privateKey, "release-key-a", map[string][]byte{"runtime-artifact.tar.zst": []byte("second")}, nil)
	fixtures := []signedReleaseFixture{first, second, first, second, first, second}

	var wait sync.WaitGroup
	errorsFound := make(chan error, len(fixtures))
	for _, fixture := range fixtures {
		fixture := fixture
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, installErr := installer.Install(fixture.installInput(publicKey, "release-key-a"))
			errorsFound <- installErr
		}()
	}
	wait.Wait()
	close(errorsFound)
	for installErr := range errorsFound {
		if installErr != nil {
			t.Fatalf("concurrent install failed: %v", installErr)
		}
	}

	pointerRaw, err := os.ReadFile(store.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	var pointer releasePointer
	if err := json.Unmarshal(pointerRaw, &pointer); err != nil {
		t.Fatal(err)
	}
	firstDigest := sha256Digest(first.archive)
	secondDigest := sha256Digest(second.archive)
	if pointer.ReleaseDigest != firstDigest && pointer.ReleaseDigest != secondDigest {
		t.Fatalf("current.json points to an unexpected Release: %+v", pointer)
	}
}

func TestReleaseInstallerRejectsUntrustedReleaseInputs(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	base := signedReleaseArchive(t, privateKey, "release-key-a", map[string][]byte{"runtime-artifact.tar.zst": []byte("runtime")}, nil)
	tests := []struct {
		name            string
		archive         signedReleaseFixture
		digest          string
		manifestDigest  string
		checksumsDigest string
		projectID       string
		releaseID       string
		keyID           string
		public          ed25519.PublicKey
		limits          ReleaseInstallLimits
	}{
		{name: "outer digest mismatch", archive: base, digest: "sha256:" + strings.Repeat("0", 64), keyID: "release-key-a", public: publicKey},
		{name: "wrong signing key", archive: base, digest: sha256Digest(base.archive), keyID: "release-key-a", public: otherPublicKey(t)},
		{name: "manifest key id mismatch", archive: signedReleaseArchive(t, privateKey, "other-key", map[string][]byte{"runtime-artifact.tar.zst": []byte("runtime")}, nil), keyID: "release-key-a", public: publicKey},
		{name: "manifest digest mismatch", archive: base, manifestDigest: "sha256:" + strings.Repeat("0", 64), keyID: "release-key-a", public: publicKey},
		{name: "checksums digest mismatch", archive: base, checksumsDigest: "sha256:" + strings.Repeat("0", 64), keyID: "release-key-a", public: publicKey},
		{name: "project identity mismatch", archive: base, projectID: "33333333-3333-4333-8333-333333333333", keyID: "release-key-a", public: publicKey},
		{name: "release identity mismatch", archive: base, releaseID: "44444444-4444-4444-8444-444444444444", keyID: "release-key-a", public: publicKey},
		{name: "single file limit", archive: signedReleaseArchive(t, privateKey, "release-key-a", map[string][]byte{"runtime-artifact.tar.zst": bytes.Repeat([]byte("x"), 64)}, nil), keyID: "release-key-a", public: publicKey, limits: ReleaseInstallLimits{MaxArchiveBytes: 1 << 20, MaxFiles: 16, MaxFileBytes: 32, MaxTotalBytes: 1 << 20, MaxDecoderBytes: 8 << 20}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			digest := test.digest
			if digest == "" {
				digest = sha256Digest(test.archive.archive)
			}
			store, err := NewReleaseStore(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { unsealReleaseTree(store.root) })
			installer, err := NewReleaseInstaller(store, test.limits)
			if err != nil {
				t.Fatal(err)
			}
			input := test.archive.installInput(test.public, test.keyID)
			input.ExpectedOuterSHA256 = digest
			if test.manifestDigest != "" {
				input.ExpectedManifestSHA256 = test.manifestDigest
			}
			if test.checksumsDigest != "" {
				input.ExpectedChecksumsSHA256 = test.checksumsDigest
			}
			if test.projectID != "" {
				input.ExpectedProjectID = test.projectID
			}
			if test.releaseID != "" {
				input.ExpectedReleaseID = test.releaseID
			}
			if _, err := installer.Install(input); err == nil {
				t.Fatal("untrusted Release input was accepted")
			}
			if _, err := os.Stat(store.currentPath); !os.IsNotExist(err) {
				t.Fatalf("rejected Release must not create current.json: %v", err)
			}
		})
	}
}

func TestReleaseInstallerRejectsUnsafeTarEntries(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		entry releaseTestEntry
	}{
		{name: "parent traversal", entry: releaseTestEntry{name: "../escape", typeFlag: tar.TypeReg, data: []byte("x")}},
		{name: "absolute path", entry: releaseTestEntry{name: "/escape", typeFlag: tar.TypeReg, data: []byte("x")}},
		{name: "backslash path", entry: releaseTestEntry{name: `a\\b`, typeFlag: tar.TypeReg, data: []byte("x")}},
		{name: "symbolic link", entry: releaseTestEntry{name: "bad-link", typeFlag: tar.TypeSymlink, link: "target"}},
		{name: "hard link", entry: releaseTestEntry{name: "bad-hard-link", typeFlag: tar.TypeLink, link: "target"}},
		{name: "fifo", entry: releaseTestEntry{name: "bad-fifo", typeFlag: tar.TypeFifo}},
		{name: "device", entry: releaseTestEntry{name: "bad-device", typeFlag: tar.TypeChar}},
		{name: "nested path", entry: releaseTestEntry{name: "nested/file", typeFlag: tar.TypeReg, data: []byte("x")}},
		{name: "directory", entry: releaseTestEntry{name: "unsigned-dir", typeFlag: tar.TypeDir}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archive := signedReleaseArchive(t, privateKey, "release-key-a", map[string][]byte{"runtime-artifact.tar.zst": []byte("runtime")}, []releaseTestEntry{test.entry})
			installer := testReleaseInstaller(t)
			if _, err := installer.Install(archive.installInput(publicKey, "release-key-a")); err == nil {
				t.Fatal("unsafe tar entry was accepted")
			}
		})
	}

	t.Run("duplicate path", func(t *testing.T) {
		archive := signedReleaseArchive(t, privateKey, "release-key-a", map[string][]byte{"runtime-artifact.tar.zst": []byte("runtime")}, []releaseTestEntry{{name: "runtime-artifact.tar.zst", typeFlag: tar.TypeReg, data: []byte("duplicate")}})
		installer := testReleaseInstaller(t)
		if _, err := installer.Install(archive.installInput(publicKey, "release-key-a")); err == nil {
			t.Fatal("duplicate tar path was accepted")
		}
	})
}

func TestReleaseChecksumsRejectsUnsortedReservedAndInvalidEntries(t *testing.T) {
	tests := []string{
		`{"schemaVersion":"release-checksums.v1","files":[{"path":"z","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":1},{"path":"a","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":1}]}`,
		`{"schemaVersion":"release-checksums.v1","files":[{"path":"signature.sig","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":1}]}`,
		`{"schemaVersion":"release-checksums.v1","files":[{"path":"item","sha256":"sha256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","size":1}]}`,
	}
	for _, raw := range tests {
		if _, err := parseReleaseChecksums([]byte(raw)); err == nil {
			t.Fatalf("invalid checksums accepted: %s", raw)
		}
	}
}

func TestVerifyReleaseChecksumsRejectsAlteredAndUnlistedFiles(t *testing.T) {
	root := t.TempDir()
	manifest := []byte(`{"supplyChain":{"signingKeyId":"release-key-a"}}`)
	artifact := []byte("runtime")
	if err := os.WriteFile(filepath.Join(root, "release-manifest.json"), manifest, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "runtime-artifact.tar.zst"), artifact, 0600); err != nil {
		t.Fatal(err)
	}
	wrong := sha256.Sum256([]byte("other"))
	checksums := releaseChecksums{SchemaVersion: releaseChecksumsSchema, Files: []releaseChecksumFile{
		{Path: "release-manifest.json", SHA256: sha256Digest(manifest), Size: int64(len(manifest))},
		{Path: "runtime-artifact.tar.zst", SHA256: "sha256:" + hex.EncodeToString(wrong[:]), Size: int64(len(artifact))},
	}}
	if err := verifyReleaseChecksums(root, checksums, 1<<20); err == nil {
		t.Fatal("altered file checksum was accepted")
	}

	digest := sha256.Sum256(artifact)
	checksums.Files[1].SHA256 = "sha256:" + hex.EncodeToString(digest[:])
	if err := os.WriteFile(filepath.Join(root, "unexpected.txt"), []byte("not listed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyReleaseChecksums(root, checksums, 1<<20); err == nil {
		t.Fatal("unlisted Release file was accepted")
	}
}

func testReleaseInstaller(t *testing.T) *ReleaseInstaller {
	t.Helper()
	store, err := NewReleaseStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { unsealReleaseTree(store.root) })
	installer, err := NewReleaseInstaller(store, ReleaseInstallLimits{MaxArchiveBytes: 1 << 20, MaxFiles: 16, MaxFileBytes: 1 << 20, MaxTotalBytes: 1 << 20, MaxDecoderBytes: 8 << 20})
	if err != nil {
		t.Fatal(err)
	}
	return installer
}

func signedReleaseArchive(t *testing.T, privateKey ed25519.PrivateKey, keyID string, files map[string][]byte, additions []releaseTestEntry) signedReleaseFixture {
	t.Helper()
	files = cloneFiles(files)
	standardFiles := map[string][]byte{
		"client-assets.tar.zst":        []byte("client"),
		"runtime-artifact.tar.zst":     []byte("runtime"),
		"sbom.cdx.json":                []byte(`{"bomFormat":"CycloneDX"}`),
		"resource-recommendation.json": []byte(`{"schemaVersion":"resource-recommendation.v1"}`),
		"health-contract.json":         []byte(`{"schemaVersion":"health-contract.v1"}`),
		"schema-plan.json":             []byte(`{"schemaVersion":"schema-plan.v1"}`),
	}
	for name, payload := range standardFiles {
		if _, exists := files[name]; !exists {
			files[name] = payload
		}
	}
	artifacts := map[string]any{
		"client": map[string]any{
			"file":     "client-assets.tar.zst",
			"checksum": sha256Digest(files["client-assets.tar.zst"]),
		},
		"runtime": map[string]any{
			"file":     "runtime-artifact.tar.zst",
			"checksum": sha256Digest(files["runtime-artifact.tar.zst"]),
		},
	}
	if collector, exists := files[optionalCollectorPayload]; exists {
		artifacts["collector"] = map[string]any{
			"file":     optionalCollectorPayload,
			"checksum": sha256Digest(collector),
		}
	}
	manifest, err := json.Marshal(map[string]any{
		"schemaVersion": releaseManifestSchema,
		"projectId":     testReleaseProjectID,
		"projectCode":   "factory_dashboard",
		"releaseId":     testReleaseID,
		"version":       "2026.08.31-001",
		"artifacts":     artifacts,
		"capabilities":  []string{"runtime.datapoint"},
		"compatibility": map[string]any{
			"minNodeAgentVersion":      "1.0.0",
			"minRuntimeVersion":        "1.0.0",
			"requiredNodeCapabilities": []string{"project_entry", "data_runtime"},
		},
		"supplyChain": map[string]any{
			"sbomRef":        "sbom.cdx.json",
			"sourceRevision": "git:0123456789abcdef0123456789abcdef01234567",
			"builderId":      "release-builder",
			"signingKeyId":   keyID,
			"promotable":     true,
		},
		"preflight": map[string]any{
			"resourceRecommendationRef": "resource-recommendation.json",
			"healthContractRef":         "health-contract.json",
			"schemaPlanRef":             "schema-plan.json",
		},
		"buildTime": "2026-08-31T10:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	files["release-manifest.json"] = manifest
	checksumFiles := make([]releaseChecksumFile, 0, len(files))
	for name, payload := range files {
		digest := sha256.Sum256(payload)
		checksumFiles = append(checksumFiles, releaseChecksumFile{Path: name, SHA256: "sha256:" + hex.EncodeToString(digest[:]), Size: int64(len(payload))})
	}
	checksumsRaw, err := json.Marshal(releaseChecksums{SchemaVersion: releaseChecksumsSchema, Files: sortedChecksumFiles(checksumFiles)})
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(privateKey, checksumsRaw)

	var compressed bytes.Buffer
	encoder, err := zstd.NewWriter(&compressed)
	if err != nil {
		t.Fatal(err)
	}
	writer := tar.NewWriter(encoder)
	for _, name := range sortedNames(files) {
		writeTarEntry(t, writer, releaseTestEntry{name: name, typeFlag: tar.TypeReg, data: files[name]})
	}
	for _, entry := range additions {
		writeTarEntry(t, writer, entry)
	}
	writeTarEntry(t, writer, releaseTestEntry{name: "checksums.json", typeFlag: tar.TypeReg, data: checksumsRaw})
	writeTarEntry(t, writer, releaseTestEntry{name: "signature.sig", typeFlag: tar.TypeReg, data: signature})
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := encoder.Close(); err != nil {
		t.Fatal(err)
	}
	return signedReleaseFixture{
		archive:         compressed.Bytes(),
		manifestDigest:  sha256Digest(manifest),
		checksumsDigest: sha256Digest(checksumsRaw),
	}
}

func writeTarEntry(t *testing.T, writer *tar.Writer, entry releaseTestEntry) {
	t.Helper()
	header := &tar.Header{Name: entry.name, Typeflag: entry.typeFlag, Linkname: entry.link, Mode: 0600}
	if entry.typeFlag == tar.TypeReg || entry.typeFlag == tar.TypeRegA {
		header.Size = int64(len(entry.data))
	}
	if err := writer.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if len(entry.data) > 0 {
		if _, err := writer.Write(entry.data); err != nil {
			t.Fatal(err)
		}
	}
}

func cloneFiles(source map[string][]byte) map[string][]byte {
	result := make(map[string][]byte, len(source)+1)
	for name, payload := range source {
		result[name] = append([]byte(nil), payload...)
	}
	return result
}

func sortedNames(files map[string][]byte) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	for left := range names {
		for right := left + 1; right < len(names); right++ {
			if names[right] < names[left] {
				names[left], names[right] = names[right], names[left]
			}
		}
	}
	return names
}

func sha256Digest(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func otherPublicKey(t *testing.T) ed25519.PublicKey {
	t.Helper()
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return public
}

func unsealReleaseTree(root string) {
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			_ = os.Chmod(path, 0700)
		} else {
			_ = os.Chmod(path, 0600)
		}
		return nil
	})
}
