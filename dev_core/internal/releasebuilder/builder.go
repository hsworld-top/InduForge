// Package releasebuilder assembles immutable, signed formal Release bundles.
// It accepts only already-built bytes and never invokes a host build tool.
package releasebuilder

import (
	"archive/tar"
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
)

const (
	manifestFile        = "release-manifest.json"
	checksumsFile       = "checksums.json"
	signatureFile       = "signature.sig"
	clientArtifactFile  = "client-assets.tar.zst"
	runtimeArtifactFile = "runtime-artifact.tar.zst"
	collectorArtifact   = "collector-artifact.tar.zst"
	sbomFile            = "sbom.cdx.json"
	resourcePlanFile    = "resource-recommendation.json"
	healthContractFile  = "health-contract.json"
	schemaPlanFile      = "schema-plan.json"
	// These limits deliberately mirror NodeAgent DefaultReleaseInstallLimits.
	// A Release accepted here must not be rejected later merely for its size.
	MaxReleaseArchiveBytes  int64 = 512 << 20
	MaxReleaseUnpackedBytes int64 = 512 << 20
	MaxReleaseFileBytes     int64 = 128 << 20
)

type buildLimits struct {
	archiveBytes  int64
	unpackedBytes int64
	fileBytes     int64
}

var defaultBuildLimits = buildLimits{
	archiveBytes:  MaxReleaseArchiveBytes,
	unpackedBytes: MaxReleaseUnpackedBytes,
	fileBytes:     MaxReleaseFileBytes,
}

// Input contains the controlled builder output and formal Release metadata.
// SigningKey is used only while signing checksums.json and is never returned.
type Input struct {
	ProjectID, ProjectCode, ReleaseID, Version string
	Client, Runtime, Collector                 []byte
	SBOM, ResourceRecommendation               []byte
	HealthContract, SchemaPlan                 []byte
	Capabilities                               []string
	MinNodeAgentVersion, MinRuntimeVersion     string
	RequiredNodeCapabilities                   []string
	SourceRevision, BuilderID                  string
	Promotable                                 bool
	BuildTime                                  time.Time
	SigningKey                                 ed25519.PrivateKey
	SigningKeyID                               string
}

// Result contains the signed bundle and metadata suitable for an immutable
// application_versions record. All SHA-256 values are lowercase hexadecimal
// without a prefix, matching the database metadata convention.
type Result struct {
	Bundle                         []byte
	OuterSHA256, ManifestSHA256    string
	ChecksumsSHA256                string
	Size                           int64
	Manifest, Checksums, Signature []byte
}

type artifact struct {
	File     string `json:"file"`
	Checksum string `json:"checksum"`
}

type manifest struct {
	SchemaVersion string   `json:"schemaVersion"`
	ProjectID     string   `json:"projectId"`
	ProjectCode   string   `json:"projectCode"`
	ReleaseID     string   `json:"releaseId"`
	Version       string   `json:"version"`
	Capabilities  []string `json:"capabilities,omitempty"`
	BuildTime     string   `json:"buildTime"`
	Artifacts     struct {
		Client    artifact  `json:"client"`
		Runtime   artifact  `json:"runtime"`
		Collector *artifact `json:"collector,omitempty"`
	} `json:"artifacts"`
	Compatibility struct {
		MinNodeAgentVersion      string   `json:"minNodeAgentVersion"`
		MinRuntimeVersion        string   `json:"minRuntimeVersion"`
		RequiredNodeCapabilities []string `json:"requiredNodeCapabilities"`
	} `json:"compatibility"`
	SupplyChain struct {
		SBOMRef        string `json:"sbomRef"`
		SourceRevision string `json:"sourceRevision"`
		BuilderID      string `json:"builderId"`
		SigningKeyID   string `json:"signingKeyId"`
		Promotable     bool   `json:"promotable"`
	} `json:"supplyChain"`
	Preflight struct {
		ResourceRecommendationRef string `json:"resourceRecommendationRef"`
		HealthContractRef         string `json:"healthContractRef"`
		SchemaPlanRef             string `json:"schemaPlanRef"`
	} `json:"preflight"`
}

type checksumEntry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type checksumsDocument struct {
	SchemaVersion string          `json:"schemaVersion"`
	Files         []checksumEntry `json:"files"`
}

type bundleFile struct {
	name string
	data []byte
}

// Build validates controlled input, creates an immutable Manifest and signed
// checksums.json, then emits a deterministic tar.zst bundle.
func Build(input Input) (Result, error) {
	return buildWithLimits(input, defaultBuildLimits)
}

func buildWithLimits(input Input, limits buildLimits) (Result, error) {
	if err := validateInput(input, limits); err != nil {
		return Result{}, err
	}

	files := []bundleFile{
		{name: clientArtifactFile, data: input.Client},
		{name: healthContractFile, data: input.HealthContract},
		{name: resourcePlanFile, data: input.ResourceRecommendation},
		{name: runtimeArtifactFile, data: input.Runtime},
		{name: sbomFile, data: input.SBOM},
		{name: schemaPlanFile, data: input.SchemaPlan},
	}
	if input.Collector != nil {
		files = append(files, bundleFile{name: collectorArtifact, data: input.Collector})
	}

	manifestData, err := buildManifest(input, files)
	if err != nil {
		return Result{}, err
	}
	files = append(files, bundleFile{name: manifestFile, data: manifestData})
	checksumsData, err := buildChecksums(files)
	if err != nil {
		return Result{}, err
	}
	signature := ed25519.Sign(input.SigningKey, checksumsData)
	files = append(files,
		bundleFile{name: checksumsFile, data: checksumsData},
		bundleFile{name: signatureFile, data: signature},
	)
	if err := validateBundleSize(files, limits); err != nil {
		return Result{}, err
	}
	bundle, err := pack(files, limits)
	if err != nil {
		return Result{}, err
	}
	if int64(len(bundle)) > limits.archiveBytes {
		return Result{}, errors.New("Release 外层归档超过 NodeAgent 接受上限")
	}
	return Result{
		Bundle:          bundle,
		OuterSHA256:     digest(bundle),
		ManifestSHA256:  digest(manifestData),
		ChecksumsSHA256: digest(checksumsData),
		Size:            int64(len(bundle)),
		Manifest:        manifestData,
		Checksums:       checksumsData,
		Signature:       signature,
	}, nil
}

func validateInput(input Input, limits buildLimits) error {
	if limits.archiveBytes <= 0 || limits.unpackedBytes <= 0 || limits.fileBytes <= 0 || limits.fileBytes > limits.unpackedBytes {
		return errors.New("Release 构建大小限制无效")
	}
	if !validUUID(input.ProjectID) || !validUUID(input.ReleaseID) {
		return errors.New("projectId 和 releaseId 必须为 UUID")
	}
	if !validProjectCode(input.ProjectCode) || !validVersion(input.Version) {
		return errors.New("projectCode 或 version 格式无效")
	}
	if !validStrictSemVer(input.MinNodeAgentVersion) || !validStrictSemVer(input.MinRuntimeVersion) {
		return errors.New("最小 NodeAgent 或 Runtime 版本格式无效")
	}
	if !validGitRevision(input.SourceRevision) || !validStableID(input.BuilderID) || !validStableID(input.SigningKeyID) {
		return errors.New("Release 供应链元数据无效")
	}
	if input.BuildTime.IsZero() || input.BuildTime.UTC().Year() < 1 || input.BuildTime.UTC().Year() > 9999 {
		return errors.New("Release buildTime 无效")
	}
	if len(input.SigningKey) != ed25519.PrivateKeySize {
		return errors.New("Release Ed25519 私钥长度无效")
	}
	if err := validateFiles(input, limits); err != nil {
		return err
	}
	if err := validateStableIDs(input.Capabilities, 128); err != nil {
		return fmt.Errorf("Release capabilities 无效: %w", err)
	}
	wantCapabilities := []string{"project_entry", "data_runtime"}
	if input.Collector != nil {
		wantCapabilities = append(wantCapabilities, "collector")
	}
	if !sameStrings(input.RequiredNodeCapabilities, wantCapabilities) {
		return errors.New("requiredNodeCapabilities 必须与 Release 服务构成一致")
	}
	return nil
}

func validateFiles(input Input, limits buildLimits) error {
	for _, file := range []struct {
		name string
		data []byte
	}{
		{clientArtifactFile, input.Client}, {runtimeArtifactFile, input.Runtime}, {sbomFile, input.SBOM},
		{resourcePlanFile, input.ResourceRecommendation}, {healthContractFile, input.HealthContract}, {schemaPlanFile, input.SchemaPlan},
	} {
		if err := validateFile(file.name, file.data, true, limits); err != nil {
			return err
		}
	}
	if input.Collector != nil {
		if err := validateFile(collectorArtifact, input.Collector, true, limits); err != nil {
			return err
		}
	}
	for _, file := range []struct {
		name string
		data []byte
	}{
		{sbomFile, input.SBOM}, {resourcePlanFile, input.ResourceRecommendation},
		{healthContractFile, input.HealthContract}, {schemaPlanFile, input.SchemaPlan},
	} {
		if !json.Valid(file.data) {
			return fmt.Errorf("Release 文件 %s 必须是有效 JSON", file.name)
		}
	}
	return nil
}

func validateFile(name string, data []byte, required bool, limits buildLimits) error {
	if required && len(data) == 0 {
		return fmt.Errorf("Release 文件 %s 不能为空", name)
	}
	if int64(len(data)) > limits.fileBytes {
		return fmt.Errorf("Release 文件 %s 超过大小上限", name)
	}
	return nil
}

func buildManifest(input Input, files []bundleFile) ([]byte, error) {
	fileDigests := make(map[string]string, len(files))
	for _, file := range files {
		fileDigests[file.name] = digest(file.data)
	}
	m := manifest{
		SchemaVersion: "2.0", ProjectID: input.ProjectID, ProjectCode: input.ProjectCode,
		ReleaseID: input.ReleaseID, Version: input.Version, Capabilities: sortedCopy(input.Capabilities),
		BuildTime: input.BuildTime.UTC().Format(time.RFC3339),
	}
	m.Artifacts.Client = artifact{File: clientArtifactFile, Checksum: "sha256:" + fileDigests[clientArtifactFile]}
	m.Artifacts.Runtime = artifact{File: runtimeArtifactFile, Checksum: "sha256:" + fileDigests[runtimeArtifactFile]}
	if input.Collector != nil {
		collector := artifact{File: collectorArtifact, Checksum: "sha256:" + fileDigests[collectorArtifact]}
		m.Artifacts.Collector = &collector
	}
	m.Compatibility.MinNodeAgentVersion = input.MinNodeAgentVersion
	m.Compatibility.MinRuntimeVersion = input.MinRuntimeVersion
	m.Compatibility.RequiredNodeCapabilities = sortedCopy(input.RequiredNodeCapabilities)
	m.SupplyChain.SBOMRef, m.SupplyChain.SourceRevision = sbomFile, input.SourceRevision
	m.SupplyChain.BuilderID, m.SupplyChain.SigningKeyID, m.SupplyChain.Promotable = input.BuilderID, input.SigningKeyID, input.Promotable
	m.Preflight.ResourceRecommendationRef, m.Preflight.HealthContractRef, m.Preflight.SchemaPlanRef = resourcePlanFile, healthContractFile, schemaPlanFile
	return json.Marshal(m)
}

func buildChecksums(files []bundleFile) ([]byte, error) {
	entries := make([]checksumEntry, 0, len(files))
	seen := make(map[string]struct{}, len(files))
	for _, file := range files {
		if _, ok := seen[file.name]; ok || file.name == checksumsFile || file.name == signatureFile {
			return nil, errors.New("Release 摘要清单包含重复或禁止文件")
		}
		seen[file.name] = struct{}{}
		entries = append(entries, checksumEntry{Path: file.name, SHA256: "sha256:" + digest(file.data), Size: int64(len(file.data))})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return json.Marshal(checksumsDocument{SchemaVersion: "release-checksums.v1", Files: entries})
}

func validateBundleSize(files []bundleFile, limits buildLimits) error {
	var total int64
	for _, file := range files {
		size := int64(len(file.data))
		if size > limits.unpackedBytes-total {
			return errors.New("Release 解包总量超过 NodeAgent 接受上限")
		}
		total += size
	}
	return nil
}

func pack(files []bundleFile, limits buildLimits) ([]byte, error) {
	sort.Slice(files, func(i, j int) bool { return files[i].name < files[j].name })
	var compressed bytes.Buffer
	encoder, err := zstd.NewWriter(&compressed, zstd.WithEncoderConcurrency(1), zstd.WithEncoderCRC(false))
	if err != nil {
		return nil, fmt.Errorf("创建 Release zstd 编码器: %w", err)
	}
	tarWriter := tar.NewWriter(encoder)
	for _, file := range files {
		if err := validateFile(file.name, file.data, true, limits); err != nil {
			_ = tarWriter.Close()
			_ = encoder.Close()
			return nil, err
		}
		header := &tar.Header{Name: file.name, Mode: 0644, Size: int64(len(file.data)), ModTime: time.Unix(0, 0).UTC(), AccessTime: time.Time{}, ChangeTime: time.Time{}, Format: tar.FormatUSTAR}
		if err := tarWriter.WriteHeader(header); err != nil {
			_ = tarWriter.Close()
			_ = encoder.Close()
			return nil, fmt.Errorf("写入 Release tar 头: %w", err)
		}
		if _, err := tarWriter.Write(file.data); err != nil {
			_ = tarWriter.Close()
			_ = encoder.Close()
			return nil, fmt.Errorf("写入 Release tar 内容: %w", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		_ = encoder.Close()
		return nil, fmt.Errorf("完成 Release tar: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("完成 Release zstd: %w", err)
	}
	return compressed.Bytes(), nil
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func validUUID(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && value == parsed.String() && parsed.Version() >= 1 && parsed.Version() <= 5 && parsed.Variant() == uuid.RFC4122
}

func validProjectCode(value string) bool {
	return validPattern(value, 128, func(c byte) bool {
		return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-'
	})
}

func validVersion(value string) bool {
	return validVersionWithLimit(value, 128)
}

func validVersionWithLimit(value string, limit int) bool {
	return validPattern(value, limit, func(c byte) bool {
		return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-' || c == '+'
	})
}

// validStrictSemVer accepts SemVer 2.0.0 without a leading "v". The Release
// display version intentionally remains broader, but compatibility gates must
// be comparable by the NodeAgent without ambiguous forms such as 01.0.0.
func validStrictSemVer(value string) bool {
	if len(value) == 0 || len(value) > 64 {
		return false
	}
	coreAndPre, metadata, hasMetadata := strings.Cut(value, "+")
	if hasMetadata && (metadata == "" || strings.Contains(metadata, "+") || !validSemVerIdentifiers(metadata, false)) {
		return false
	}
	core, prerelease, hasPrerelease := strings.Cut(coreAndPre, "-")
	if hasPrerelease && (prerelease == "" || !validSemVerIdentifiers(prerelease, true)) {
		return false
	}
	segments := strings.Split(core, ".")
	if len(segments) != 3 {
		return false
	}
	for _, segment := range segments {
		if !validSemVerNumber(segment) {
			return false
		}
	}
	return true
}

func validSemVerIdentifiers(value string, rejectNumericLeadingZero bool) bool {
	for _, identifier := range strings.Split(value, ".") {
		if identifier == "" {
			return false
		}
		numeric := true
		for i := range identifier {
			c := identifier[i]
			if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
			if c < '0' || c > '9' {
				numeric = false
			}
		}
		if rejectNumericLeadingZero && numeric && !validSemVerNumber(identifier) {
			return false
		}
	}
	return true
}

func validSemVerNumber(value string) bool {
	if value == "" || (len(value) > 1 && value[0] == '0') {
		return false
	}
	for i := range value {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}

func validPattern(value string, limit int, allowed func(byte) bool) bool {
	if len(value) == 0 || len(value) > limit || !((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z') || (value[0] >= '0' && value[0] <= '9')) {
		return false
	}
	for i := range value {
		if !allowed(value[i]) {
			return false
		}
	}
	return true
}

func validGitRevision(value string) bool {
	if !strings.HasPrefix(value, "git:") || len(value) < 44 || len(value) > 68 {
		return false
	}
	for _, c := range value[4:] {
		if !(c >= 'a' && c <= 'f' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func validStableID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for i := range value {
		c := value[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || (i > 0 && (c == '.' || c == '_' || c == '-' || c == ':')) {
			continue
		}
		return false
	}
	return true
}

func validateStableIDs(values []string, limit int) error {
	if len(values) > limit {
		return errors.New("数量超过上限")
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		if !validStableID(value) {
			return errors.New("标识符格式无效")
		}
		if _, ok := seen[value]; ok {
			return errors.New("标识符重复")
		}
		seen[value] = struct{}{}
	}
	return nil
}

func sameStrings(actual, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	actualSet := map[string]struct{}{}
	for _, value := range actual {
		actualSet[value] = struct{}{}
	}
	for _, value := range expected {
		if _, ok := actualSet[value]; !ok {
			return false
		}
	}
	return len(actualSet) == len(actual)
}

func sortedCopy(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}
