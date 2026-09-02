package ops

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"archive/tar"

	"github.com/klauspost/compress/zstd"
	"golang.org/x/mod/semver"
)

const (
	releaseChecksumsSchema = "release-checksums.v1"
	releasePointerSchema   = "release-pointer.v1"
	releaseManifestSchema  = "2.0"
)

var requiredReleasePayloads = []string{
	"client-assets.tar.zst",
	"health-contract.json",
	"release-manifest.json",
	"resource-recommendation.json",
	"runtime-artifact.tar.zst",
	"sbom.cdx.json",
	"schema-plan.json",
}

const optionalCollectorPayload = "collector-artifact.tar.zst"

// ReleaseStore 是单个 deployment 的本地 Release 根。调用者必须为每个 deployment
// 创建独立 Store，中心输入永远不能影响其中的目录布局。
type ReleaseStore struct {
	root         string
	incomingRoot string
	releasesRoot string
	currentPath  string
}

// ReleaseInstaller 只负责落盘、验签和原子选择 Release；它不启动进程，也不解释中心命令。
type ReleaseInstaller struct {
	store  *ReleaseStore
	limits ReleaseInstallLimits
}

// ReleaseInstallLimits 限制不可信压缩包的磁盘、解压和内存消耗。
type ReleaseInstallLimits struct {
	MaxArchiveBytes int64
	MaxFiles        int
	MaxFileBytes    int64
	MaxTotalBytes   int64
	MaxDecoderBytes uint64
}

// ReleaseInstallInput 是安装一份已由 Agent 主动取得的 Release 所需的最小信任输入。
// KeyID 只用于校验 Manifest 声明，公钥必须来自 Agent 的本地 trust store。
type ReleaseInstallInput struct {
	Archive                 io.Reader
	ExpectedOuterSHA256     string
	ExpectedManifestSHA256  string
	ExpectedChecksumsSHA256 string
	ExpectedProjectID       string
	ExpectedReleaseID       string
	ExpectedVersion         string
	NodeAgentVersion        string
	RuntimeVersion          string
	NodeCapabilities        []string
	BoundServices           []string
	// Engine 非空时表示 v2 单引擎 Binding；Release 清单的全工程能力不能要求
	// 每个分布式节点都安装其他引擎能力，改为校验该引擎的固定最小能力集。
	Engine                string
	VerificationPublicKey ed25519.PublicKey
	KeyID                 string
}

// InstalledRelease 是已验证并被原子选为 current 的本地结果。
type InstalledRelease struct {
	ReleaseDigest string
	ReleaseDir    string
	ActivatedAt   time.Time
}

type releaseChecksumFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type releaseChecksums struct {
	SchemaVersion string                `json:"schemaVersion"`
	Files         []releaseChecksumFile `json:"files"`
}

type releaseArtifactForVerification struct {
	File                 string          `json:"file"`
	Checksum             string          `json:"checksum"`
	SourceSnapshot       json.RawMessage `json:"sourceSnapshot,omitempty"`
	SourceSnapshotSHA256 string          `json:"sourceSnapshotSha256,omitempty"`
}

// collectorSourceSnapshotForVerification 是开发制品内采集快照的最小受信结构。
// 节点只接受签名清单中与工件摘要一致的快照，避免为兼容新字段而放宽严格 JSON 校验。
type collectorSourceSnapshotForVerification struct {
	SchemaVersion    string          `json:"schemaVersion"`
	ProjectID        string          `json:"projectId"`
	ArtifactRevision int64           `json:"artifactRevision"`
	SHA256           string          `json:"sha256"`
	Size             int             `json:"size"`
	Artifact         json.RawMessage `json:"artifact"`
}

type releaseCompatibility struct {
	MinNodeAgentVersion      string   `json:"minNodeAgentVersion"`
	MinRuntimeVersion        string   `json:"minRuntimeVersion"`
	RequiredNodeCapabilities []string `json:"requiredNodeCapabilities"`
}

type releaseManifestForVerification struct {
	SchemaVersion string `json:"schemaVersion"`
	ProjectID     string `json:"projectId"`
	ProjectCode   string `json:"projectCode"`
	ReleaseID     string `json:"releaseId"`
	Version       string `json:"version"`
	Artifacts     struct {
		Client    releaseArtifactForVerification  `json:"client"`
		Runtime   releaseArtifactForVerification  `json:"runtime"`
		Collector *releaseArtifactForVerification `json:"collector,omitempty"`
	} `json:"artifacts"`
	Capabilities  []string             `json:"capabilities,omitempty"`
	Compatibility releaseCompatibility `json:"compatibility"`
	SupplyChain   struct {
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
	BuildTime string `json:"buildTime"`
}

type releasePointer struct {
	SchemaVersion string `json:"schemaVersion"`
	ReleaseDigest string `json:"releaseDigest"`
	ReleaseDir    string `json:"releaseDir"`
	ActivatedAt   string `json:"activatedAt"`
}

// NewReleaseStore 将根目录固定为绝对本地路径。该根由节点本地部署状态派生，不能接受中心路径。
func NewReleaseStore(root string) (*ReleaseStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("release store 根目录不能为空")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("解析 release store 根目录: %w", err)
	}
	return &ReleaseStore{
		root:         filepath.Clean(abs),
		incomingRoot: filepath.Join(abs, "incoming"),
		releasesRoot: filepath.Join(abs, "releases"),
		currentPath:  filepath.Join(abs, "current.json"),
	}, nil
}

func NewReleaseInstaller(store *ReleaseStore, limits ReleaseInstallLimits) (*ReleaseInstaller, error) {
	if store == nil {
		return nil, errors.New("release store 不能为空")
	}
	return &ReleaseInstaller{store: store, limits: limits.normalized()}, nil
}

func DefaultReleaseInstallLimits() ReleaseInstallLimits {
	return ReleaseInstallLimits{
		MaxArchiveBytes: 512 << 20,
		MaxFiles:        128,
		MaxFileBytes:    128 << 20,
		MaxTotalBytes:   512 << 20,
		MaxDecoderBytes: 64 << 20,
	}
}

func (l ReleaseInstallLimits) normalized() ReleaseInstallLimits {
	defaults := DefaultReleaseInstallLimits()
	if l.MaxArchiveBytes <= 0 {
		l.MaxArchiveBytes = defaults.MaxArchiveBytes
	}
	if l.MaxFiles <= 0 {
		l.MaxFiles = defaults.MaxFiles
	}
	if l.MaxFileBytes <= 0 {
		l.MaxFileBytes = defaults.MaxFileBytes
	}
	if l.MaxTotalBytes <= 0 {
		l.MaxTotalBytes = defaults.MaxTotalBytes
	}
	if l.MaxDecoderBytes == 0 {
		l.MaxDecoderBytes = defaults.MaxDecoderBytes
	}
	return l
}

// Install 流式保存 outer tar.zst，先校验外层摘要，再安全解包、校验原始 checksums
// 签名和每个文件。任一失败都不会改写已存在的 current.json。
func (i *ReleaseInstaller) Install(input ReleaseInstallInput) (InstalledRelease, error) {
	if input.Archive == nil {
		return InstalledRelease{}, errors.New("release archive 不能为空")
	}
	digestHex, err := parseSHA256(input.ExpectedOuterSHA256)
	if err != nil {
		return InstalledRelease{}, fmt.Errorf("outer Release 摘要无效: %w", err)
	}
	if len(input.VerificationPublicKey) != ed25519.PublicKeySize {
		return InstalledRelease{}, errors.New("Release Ed25519 公钥长度无效")
	}
	if strings.TrimSpace(input.KeyID) == "" {
		return InstalledRelease{}, errors.New("Release signing keyId 不能为空")
	}
	if strings.TrimSpace(input.ExpectedProjectID) == "" || strings.TrimSpace(input.ExpectedReleaseID) == "" {
		return InstalledRelease{}, errors.New("Release projectId 和 releaseId 不能为空")
	}
	if !validVersionValue(input.ExpectedVersion) || !validComparableVersion(input.NodeAgentVersion) || !validComparableVersion(input.RuntimeVersion) {
		return InstalledRelease{}, errors.New("Release 版本或本机 NodeAgent/Runtime 版本无效，拒绝默认兼容")
	}
	if _, err := normalizedNodeCapabilities(input.NodeCapabilities); err != nil {
		return InstalledRelease{}, fmt.Errorf("本机节点能力无效: %w", err)
	}
	if _, err := normalizedNodeCapabilities(input.BoundServices); err != nil {
		return InstalledRelease{}, fmt.Errorf("DeploymentBinding 服务能力无效: %w", err)
	}
	if _, err := parseSHA256(input.ExpectedManifestSHA256); err != nil {
		return InstalledRelease{}, fmt.Errorf("Release manifest 摘要无效: %w", err)
	}
	if _, err := parseSHA256(input.ExpectedChecksumsSHA256); err != nil {
		return InstalledRelease{}, fmt.Errorf("Release checksums 摘要无效: %w", err)
	}
	if err := i.store.ensureRoots(); err != nil {
		return InstalledRelease{}, err
	}

	partial, actualDigest, err := i.download(input.Archive, digestHex)
	if err != nil {
		return InstalledRelease{}, err
	}
	defer os.Remove(partial)
	if actualDigest != digestHex {
		return InstalledRelease{}, fmt.Errorf("outer Release SHA-256 不匹配: 期望 sha256:%s，实际 sha256:%s", digestHex, actualDigest)
	}

	finalName := "sha256-" + digestHex
	finalDir := filepath.Join(i.store.releasesRoot, finalName)
	if info, statErr := os.Stat(finalDir); statErr == nil {
		if !info.IsDir() {
			return InstalledRelease{}, fmt.Errorf("内容寻址 Release 路径不是目录: %s", finalDir)
		}
		if err := i.verifyRelease(finalDir, input); err != nil {
			return InstalledRelease{}, fmt.Errorf("已有内容寻址 Release 校验失败: %w", err)
		}
		// 已封存目录为 0555；在完成验签后临时恢复 owner 写权限，仅用于原子补物化，
		// 随后立即重新 seal，绝不修改任何已校验 payload。
		if err := os.Chmod(finalDir, 0755); err != nil {
			return InstalledRelease{}, fmt.Errorf("解封已有内容寻址 Release: %w", err)
		}
		// 升级旧节点时，已验签内容寻址 Release 可能尚未包含物化目录；只允许
		// 基于同一已验证 tar 补齐，已有目录由 materialize 做结构校验且绝不覆盖。
		if err := i.materializeRuntimeArtifact(finalDir); err != nil {
			return InstalledRelease{}, fmt.Errorf("补物化已有内容寻址 Release: %w", err)
		}
		if err := sealRelease(finalDir); err != nil {
			return InstalledRelease{}, fmt.Errorf("封存已有内容寻址 Release: %w", err)
		}
		return i.store.activate(input.ExpectedOuterSHA256, finalName)
	} else if !os.IsNotExist(statErr) {
		return InstalledRelease{}, statErr
	}

	staging, err := os.MkdirTemp(i.store.root, ".release-staging-")
	if err != nil {
		return InstalledRelease{}, fmt.Errorf("创建 Release 临时目录: %w", err)
	}
	defer os.RemoveAll(staging)
	if err := i.extract(partial, staging); err != nil {
		return InstalledRelease{}, err
	}
	if err := i.verifyRelease(staging, input); err != nil {
		return InstalledRelease{}, err
	}
	if err := i.materializeRuntimeArtifact(staging); err != nil {
		return InstalledRelease{}, err
	}
	if err := syncDirectory(staging); err != nil {
		return InstalledRelease{}, fmt.Errorf("同步 Release 临时目录: %w", err)
	}
	if err := os.Rename(staging, finalDir); err != nil {
		// 相同内容可能由崩溃恢复或串行重试已落盘；绝不能覆盖它，必须重新验证。
		if _, statErr := os.Stat(finalDir); statErr != nil {
			return InstalledRelease{}, fmt.Errorf("提交内容寻址 Release: %w", err)
		}
		if err := i.verifyRelease(finalDir, input); err != nil {
			return InstalledRelease{}, fmt.Errorf("并发已有 Release 校验失败: %w", err)
		}
	}
	if err := sealRelease(finalDir); err != nil {
		return InstalledRelease{}, fmt.Errorf("封存内容寻址 Release: %w", err)
	}
	if err := syncDirectory(finalDir); err != nil {
		return InstalledRelease{}, fmt.Errorf("同步内容寻址 Release: %w", err)
	}
	if err := syncDirectory(i.store.releasesRoot); err != nil {
		return InstalledRelease{}, fmt.Errorf("同步 Release 根目录: %w", err)
	}
	return i.store.activate(input.ExpectedOuterSHA256, finalName)
}

// materializeRuntimeArtifact 在已验签、已校验 outer Release 后安全解包运行子制品。
// 结果写入同一摘要 Release 的 runtime-artifact 目录，供 K3s 以只读 hostPath 挂载。
func (i *ReleaseInstaller) materializeRuntimeArtifact(root string) error {
	archive, err := os.Open(filepath.Join(root, "runtime-artifact.tar.zst"))
	if err != nil {
		return fmt.Errorf("读取 runtime artifact: %w", err)
	}
	defer archive.Close()
	decoder, err := zstd.NewReader(archive, zstd.WithDecoderMaxMemory(i.limits.MaxDecoderBytes))
	if err != nil {
		return fmt.Errorf("打开 runtime artifact: %w", err)
	}
	defer decoder.Close()
	tmp, err := os.MkdirTemp(root, ".runtime-artifact-")
	if err != nil {
		return fmt.Errorf("创建 runtime artifact 临时目录: %w", err)
	}
	defer os.RemoveAll(tmp)
	reader := tar.NewReader(decoder)
	found, count := false, 0
	var total int64
	for {
		h, nextErr := reader.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return fmt.Errorf("读取 runtime artifact 条目: %w", nextErr)
		}
		if h.Typeflag != tar.TypeReg || h.Name != "runtime-project-artifact.json" || h.Size < 0 || h.Size > i.limits.MaxFileBytes {
			return errors.New("runtime artifact 包含非法条目")
		}
		count++
		total += h.Size
		if count != 1 || total > i.limits.MaxTotalBytes {
			return errors.New("runtime artifact 超过安全限制")
		}
		if err := writeArchiveRegularFile(filepath.Join(tmp, h.Name), reader, h.Size); err != nil {
			return fmt.Errorf("写入 runtime artifact: %w", err)
		}
		found = true
	}
	if !found {
		return errors.New("runtime artifact 缺少项目制品")
	}
	if err := syncDirectory(tmp); err != nil {
		return err
	}
	target := filepath.Join(root, "runtime-artifact")
	if _, err := os.Stat(target); err == nil {
		info, statErr := os.Lstat(filepath.Join(target, "runtime-project-artifact.json"))
		if statErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("已有 runtime artifact 物化目录无效")
		}
		entries, readErr := os.ReadDir(target)
		if readErr != nil || len(entries) != 1 {
			return errors.New("已有 runtime artifact 物化目录结构无效")
		}
		return nil
	}
	if err := os.Rename(tmp, target); err != nil {
		return fmt.Errorf("提交 runtime artifact: %w", err)
	}
	return syncDirectory(root)
}

func (s *ReleaseStore) ensureRoots() error {
	for _, directory := range []string{s.root, s.incomingRoot, s.releasesRoot} {
		if err := os.MkdirAll(directory, 0700); err != nil {
			return fmt.Errorf("创建 Release 目录 %s: %w", directory, err)
		}
	}
	return nil
}

func (i *ReleaseInstaller) download(source io.Reader, expectedDigest string) (string, string, error) {
	partial, err := os.CreateTemp(i.store.incomingRoot, expectedDigest+"-*.partial")
	if err != nil {
		return "", "", fmt.Errorf("创建 Release partial 文件: %w", err)
	}
	path := partial.Name()
	defer partial.Close()
	if err := partial.Chmod(0600); err != nil {
		return "", "", err
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(partial, hash), io.LimitReader(source, i.limits.MaxArchiveBytes+1))
	if err != nil {
		return "", "", fmt.Errorf("下载 Release 失败: %w", err)
	}
	if written > i.limits.MaxArchiveBytes {
		return "", "", fmt.Errorf("Release archive 超过 %d 字节上限", i.limits.MaxArchiveBytes)
	}
	if err := partial.Sync(); err != nil {
		return "", "", fmt.Errorf("同步 Release partial 文件: %w", err)
	}
	return path, hex.EncodeToString(hash.Sum(nil)), nil
}

func (i *ReleaseInstaller) extract(archivePath, destination string) error {
	archive, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()
	decoder, err := zstd.NewReader(archive, zstd.WithDecoderMaxMemory(i.limits.MaxDecoderBytes))
	if err != nil {
		return fmt.Errorf("打开 Release zstd 流: %w", err)
	}
	defer decoder.Close()

	reader := tar.NewReader(decoder)
	seen := map[string]struct{}{}
	var count int
	var total int64
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("读取 Release tar 条目: %w", err)
		}
		name, err := releaseArchiveName(header.Name)
		if err != nil {
			return err
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("Release tar 包含重复条目: %s", name)
		}
		seen[name] = struct{}{}
		count++
		if count > i.limits.MaxFiles {
			return fmt.Errorf("Release tar 条目超过 %d 个上限", i.limits.MaxFiles)
		}
		target := filepath.Join(destination, filepath.FromSlash(name))
		if !releasePathWithin(destination, target) {
			return fmt.Errorf("Release tar 条目越界: %s", name)
		}

		switch header.Typeflag {
		case tar.TypeReg, tar.TypeRegA:
			if header.Size < 0 || header.Size > i.limits.MaxFileBytes {
				return fmt.Errorf("Release 文件 %s 超过单文件上限", name)
			}
			total += header.Size
			if total > i.limits.MaxTotalBytes {
				return fmt.Errorf("Release 解压总量超过 %d 字节上限", i.limits.MaxTotalBytes)
			}
			if err := writeArchiveRegularFile(target, reader, header.Size); err != nil {
				return fmt.Errorf("写入 Release 文件 %s: %w", name, err)
			}
		default:
			return fmt.Errorf("Release tar 不允许的条目类型 %d: %s", header.Typeflag, name)
		}
	}
	return syncDirectory(destination)
}

func releaseArchiveName(raw string) (string, error) {
	if raw == "" || strings.ContainsRune(raw, '\x00') || strings.Contains(raw, "\\") || strings.HasPrefix(raw, "/") {
		return "", fmt.Errorf("Release tar 路径非法: %q", raw)
	}
	name := strings.TrimSuffix(raw, "/")
	if name == "" || strings.Contains(name, "/") || path.Clean(name) != name || name == "." || name == ".." || strings.HasPrefix(name, "../") {
		return "", fmt.Errorf("Release tar 必须在根目录放置普通文件: %q", raw)
	}
	return name, nil
}

func writeArchiveRegularFile(target string, source io.Reader, size int64) error {
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	written, err := io.Copy(file, io.LimitReader(source, size+1))
	if err != nil {
		return err
	}
	if written != size {
		return fmt.Errorf("tar 文件长度不匹配: 期望 %d，实际 %d", size, written)
	}
	return file.Sync()
}

func (i *ReleaseInstaller) verifyRelease(root string, input ReleaseInstallInput) error {
	checksumsRaw, err := readReleaseRegularFile(root, "checksums.json", i.limits.MaxFileBytes)
	if err != nil {
		return err
	}
	if digestBytes(checksumsRaw) != input.ExpectedChecksumsSHA256 {
		return errors.New("Release checksums.json SHA-256 与部署绑定不匹配")
	}
	signature, err := readReleaseRegularFile(root, "signature.sig", int64(ed25519.SignatureSize))
	if err != nil {
		return err
	}
	if len(signature) != ed25519.SignatureSize || !ed25519.Verify(input.VerificationPublicKey, checksumsRaw, signature) {
		return errors.New("Release checksums.json Ed25519 签名无效")
	}
	checksums, err := parseReleaseChecksums(checksumsRaw)
	if err != nil {
		return err
	}
	if err := verifyReleaseChecksums(root, checksums, i.limits.MaxFileBytes); err != nil {
		return err
	}
	manifestRaw, err := readReleaseRegularFile(root, "release-manifest.json", i.limits.MaxFileBytes)
	if err != nil {
		return err
	}
	if digestBytes(manifestRaw) != input.ExpectedManifestSHA256 {
		return errors.New("Release manifest SHA-256 与部署绑定不匹配")
	}
	var manifest releaseManifestForVerification
	decoder := json.NewDecoder(strings.NewReader(string(manifestRaw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return fmt.Errorf("Release manifest 无效: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("Release manifest 包含尾随 JSON")
	}
	if manifest.SchemaVersion != releaseManifestSchema {
		return fmt.Errorf("Release manifest schemaVersion 不受支持: %q", manifest.SchemaVersion)
	}
	if manifest.ProjectID != input.ExpectedProjectID || manifest.ReleaseID != input.ExpectedReleaseID {
		return errors.New("Release manifest 的 projectId/releaseId 与部署绑定不匹配")
	}
	if !validProjectCode(manifest.ProjectCode) || manifest.Version != input.ExpectedVersion {
		return errors.New("Release manifest 的 projectCode 或 version 与部署命令不匹配")
	}
	if err := validateManifestCompatibility(manifest.Compatibility, input); err != nil {
		return err
	}
	if manifest.SupplyChain.SigningKeyID != input.KeyID {
		return fmt.Errorf("Release manifest signingKeyId 不匹配: %q", manifest.SupplyChain.SigningKeyID)
	}
	if !manifest.SupplyChain.Promotable {
		return errors.New("Release manifest 未声明 promotable=true")
	}
	if manifest.SupplyChain.SBOMRef != "sbom.cdx.json" ||
		manifest.Preflight.ResourceRecommendationRef != "resource-recommendation.json" ||
		manifest.Preflight.HealthContractRef != "health-contract.json" ||
		manifest.Preflight.SchemaPlanRef != "schema-plan.json" {
		return errors.New("Release manifest 的供应链或预检引用不是冻结文件名")
	}
	declared := make(map[string]releaseChecksumFile, len(checksums.Files))
	for _, entry := range checksums.Files {
		declared[entry.Path] = entry
	}
	if err := verifyManifestArtifact(manifest.Artifacts.Client, "client-assets.tar.zst", declared); err != nil {
		return err
	}
	if err := verifyManifestArtifact(manifest.Artifacts.Runtime, "runtime-artifact.tar.zst", declared); err != nil {
		return err
	}
	if manifest.Artifacts.Collector != nil {
		if err := verifyManifestArtifact(*manifest.Artifacts.Collector, optionalCollectorPayload, declared); err != nil {
			return err
		}
		if err := verifyCollectorSourceSnapshot(*manifest.Artifacts.Collector, manifest.ProjectID); err != nil {
			return err
		}
	} else if _, exists := declared[optionalCollectorPayload]; exists {
		return errors.New("Release 包含 collector 产物，但 manifest 未声明")
	}
	return nil
}

func verifyCollectorSourceSnapshot(artifact releaseArtifactForVerification, projectID string) error {
	// 生产早期 Release 可以没有开发快照；一旦声明则必须完整且可校验。
	if len(artifact.SourceSnapshot) == 0 && artifact.SourceSnapshotSHA256 == "" {
		return nil
	}
	if len(artifact.SourceSnapshot) == 0 || len(artifact.SourceSnapshot) > 16<<20 || !validSHA256(artifact.SourceSnapshotSHA256) {
		return errors.New("Release collector sourceSnapshot 无效")
	}
	if digestBytes(artifact.SourceSnapshot) != artifact.SourceSnapshotSHA256 {
		return errors.New("Release collector sourceSnapshot 摘要不匹配")
	}
	var snapshot collectorSourceSnapshotForVerification
	if err := strictDecodeJSON(artifact.SourceSnapshot, &snapshot); err != nil {
		return fmt.Errorf("Release collector sourceSnapshot 格式无效: %w", err)
	}
	if snapshot.SchemaVersion != "collector-runtime-artifact.v1" || snapshot.ProjectID != projectID || snapshot.ArtifactRevision < 1 ||
		snapshot.Size != len(snapshot.Artifact) || len(snapshot.Artifact) == 0 || !json.Valid(snapshot.Artifact) ||
		snapshot.SHA256 != digestBytes(snapshot.Artifact) {
		return errors.New("Release collector sourceSnapshot 内容无效")
	}
	return nil
}

func parseReleaseChecksums(raw []byte) (releaseChecksums, error) {
	var document releaseChecksums
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return document, fmt.Errorf("Release checksums.json 无效: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return document, errors.New("Release checksums.json 包含尾随 JSON")
	}
	if document.SchemaVersion != releaseChecksumsSchema {
		return document, errors.New("Release checksums.json schemaVersion 或 files 无效")
	}
	allowed := make(map[string]struct{}, len(requiredReleasePayloads)+1)
	for _, name := range requiredReleasePayloads {
		allowed[name] = struct{}{}
	}
	allowed[optionalCollectorPayload] = struct{}{}
	previous := ""
	present := make(map[string]struct{}, len(document.Files))
	for _, entry := range document.Files {
		name, err := releaseArchiveName(entry.Path)
		_, allowedName := allowed[entry.Path]
		if err != nil || name != entry.Path || !allowedName {
			return document, fmt.Errorf("Release checksum 路径非法: %q", entry.Path)
		}
		if previous != "" && entry.Path <= previous {
			return document, errors.New("Release checksums files 必须按 path 严格升序且不可重复")
		}
		if _, err := parseSHA256(entry.SHA256); err != nil || entry.Size < 0 {
			return document, fmt.Errorf("Release checksum 条目无效: %s", entry.Path)
		}
		previous = entry.Path
		present[entry.Path] = struct{}{}
	}
	for _, required := range requiredReleasePayloads {
		if _, exists := present[required]; !exists {
			return document, fmt.Errorf("Release checksums 缺少标准文件: %s", required)
		}
	}
	return document, nil
}

func verifyReleaseChecksums(root string, checksums releaseChecksums, maxFileBytes int64) error {
	declared := make(map[string]releaseChecksumFile, len(checksums.Files))
	for _, entry := range checksums.Files {
		declared[entry.Path] = entry
		actualDigest, actualSize, err := hashReleaseRegularFile(root, entry.Path, maxFileBytes)
		if err != nil {
			return err
		}
		if actualSize != entry.Size {
			return fmt.Errorf("Release 文件大小不匹配: %s", entry.Path)
		}
		if entry.SHA256 != actualDigest {
			return fmt.Errorf("Release 文件 SHA-256 不匹配: %s", entry.Path)
		}
	}
	if _, ok := declared["release-manifest.json"]; !ok {
		return errors.New("Release checksums 未覆盖 release-manifest.json")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if entry.Name() != "runtime-artifact" {
				return fmt.Errorf("Release 包含非标准目录: %s", entry.Name())
			}
			info, statErr := os.Lstat(filepath.Join(root, entry.Name(), "runtime-project-artifact.json"))
			children, readErr := os.ReadDir(filepath.Join(root, entry.Name()))
			if statErr != nil || readErr != nil || len(children) != 1 || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
				return errors.New("Release runtime artifact 物化目录无效")
			}
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return fmt.Errorf("Release 包含非普通根文件: %s", entry.Name())
		}
		if entry.Name() == "checksums.json" || entry.Name() == "signature.sig" {
			continue
		}
		if _, ok := declared[entry.Name()]; !ok {
			return fmt.Errorf("Release checksums 未覆盖文件: %s", entry.Name())
		}
	}
	return nil
}

func verifyManifestArtifact(artifact releaseArtifactForVerification, expectedFile string, declared map[string]releaseChecksumFile) error {
	if artifact.File != expectedFile {
		return fmt.Errorf("Release manifest 产物文件名无效: %q", artifact.File)
	}
	entry, exists := declared[expectedFile]
	if !exists || entry.SHA256 != artifact.Checksum {
		return fmt.Errorf("Release manifest 产物摘要与 checksums 不匹配: %s", expectedFile)
	}
	return nil
}

func hashReleaseRegularFile(root, name string, limit int64) (string, int64, error) {
	if _, err := releaseArchiveName(name); err != nil {
		return "", 0, err
	}
	filePath := filepath.Join(root, filepath.FromSlash(name))
	if !releasePathWithin(root, filePath) {
		return "", 0, fmt.Errorf("Release 文件路径越界: %s", name)
	}
	info, err := os.Lstat(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("读取 Release 文件 %s: %w", name, err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > limit {
		return "", 0, fmt.Errorf("Release 文件不是受限普通文件: %s", name)
	}
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, limit+1))
	if err != nil || written > limit {
		return "", 0, fmt.Errorf("计算 Release 文件摘要失败或超限: %s", name)
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), written, nil
}

func readReleaseRegularFile(root, name string, limit int64) ([]byte, error) {
	if _, err := releaseArchiveName(name); err != nil {
		return nil, err
	}
	path := filepath.Join(root, filepath.FromSlash(name))
	if !releasePathWithin(root, path) {
		return nil, fmt.Errorf("Release 文件路径越界: %s", name)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("读取 Release 文件 %s: %w", name, err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > limit {
		return nil, fmt.Errorf("Release 文件不是受限普通文件: %s", name)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	payload, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(payload)) > limit {
		return nil, fmt.Errorf("读取 Release 文件 %s 失败或超限", name)
	}
	return payload, nil
}

func sealRelease(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if entry.Name() != "runtime-artifact" {
				return fmt.Errorf("Release 提交前出现非法目录: %s", entry.Name())
			}
			child := filepath.Join(root, entry.Name(), "runtime-project-artifact.json")
			info, statErr := os.Lstat(child)
			if statErr != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
				return errors.New("runtime artifact 物化目录无效")
			}
			if err := os.Chmod(child, 0444); err != nil {
				return err
			}
			if err := os.Chmod(filepath.Join(root, entry.Name()), 0555); err != nil {
				return err
			}
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return fmt.Errorf("Release 提交前出现非法文件: %s", entry.Name())
		}
		// K3s 以固定非 root UID 挂载内容寻址目录；other 只读位允许容器读取，
		// 写位仍全部关闭，且上层 Agent 私有目录继续阻止宿主机普通用户遍历。
		if err := os.Chmod(filepath.Join(root, entry.Name()), 0444); err != nil {
			return err
		}
	}
	return os.Chmod(root, 0555)
}

func (s *ReleaseStore) activate(digest, finalName string) (InstalledRelease, error) {
	if _, err := parseSHA256(digest); err != nil {
		return InstalledRelease{}, err
	}
	if finalName != "sha256-"+strings.TrimPrefix(digest, "sha256:") {
		return InstalledRelease{}, errors.New("Release 内容寻址目录与摘要不一致")
	}
	activatedAt := time.Now().UTC()
	pointer := releasePointer{
		SchemaVersion: releasePointerSchema,
		ReleaseDigest: digest,
		ReleaseDir:    filepath.ToSlash(filepath.Join("releases", finalName)),
		ActivatedAt:   activatedAt.Format(time.RFC3339Nano),
	}
	payload, err := json.Marshal(pointer)
	if err != nil {
		return InstalledRelease{}, err
	}
	file, err := os.CreateTemp(s.root, ".current-*.tmp")
	if err != nil {
		return InstalledRelease{}, err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return InstalledRelease{}, err
	}
	if _, err := file.Write(payload); err != nil {
		file.Close()
		return InstalledRelease{}, err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return InstalledRelease{}, err
	}
	if err := file.Close(); err != nil {
		return InstalledRelease{}, err
	}
	if err := replaceFileAtomically(temporary, s.currentPath); err != nil {
		return InstalledRelease{}, fmt.Errorf("原子切换 current.json: %w", err)
	}
	if err := syncDirectory(s.root); err != nil {
		return InstalledRelease{}, fmt.Errorf("同步 current.json 父目录: %w", err)
	}
	return InstalledRelease{ReleaseDigest: digest, ReleaseDir: filepath.Join(s.releasesRoot, finalName), ActivatedAt: activatedAt}, nil
}

func parseSHA256(value string) (string, error) {
	if !strings.HasPrefix(value, "sha256:") {
		return "", errors.New("必须为 sha256:<64hex>")
	}
	hexValue := strings.TrimPrefix(value, "sha256:")
	if len(hexValue) != sha256.Size*2 || strings.ToLower(hexValue) != hexValue {
		return "", errors.New("SHA-256 必须为 64 位小写十六进制")
	}
	if _, err := hex.DecodeString(hexValue); err != nil {
		return "", err
	}
	return hexValue, nil
}

func digestBytes(value []byte) string {
	digest := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func releasePathWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

// sortedChecksumFiles 用于测试和未来构建器复用，确保冻结格式的 path 次序稳定。
func sortedChecksumFiles(files []releaseChecksumFile) []releaseChecksumFile {
	result := append([]releaseChecksumFile(nil), files...)
	sort.Slice(result, func(left, right int) bool { return result[left].Path < result[right].Path })
	return result
}

// validateManifestCompatibility 在 current 指针切换前复验本机可承载性。版本无法
// 可靠比较时宁可拒绝安装，也不能把未知不兼容的 Release 标成 current。
func validateManifestCompatibility(compatibility releaseCompatibility, input ReleaseInstallInput) error {
	if !validComparableVersion(compatibility.MinNodeAgentVersion) || !validComparableVersion(compatibility.MinRuntimeVersion) {
		return errors.New("Release compatibility 最低版本不可比较")
	}
	nodeAgentCompatible, err := versionAtLeast(input.NodeAgentVersion, compatibility.MinNodeAgentVersion)
	if err != nil || !nodeAgentCompatible {
		return errors.New("NodeAgent 版本不满足 Release compatibility")
	}
	runtimeCompatible, err := versionAtLeast(input.RuntimeVersion, compatibility.MinRuntimeVersion)
	if err != nil || !runtimeCompatible {
		return errors.New("Runtime 版本不满足 Release compatibility")
	}
	available, err := normalizedNodeCapabilities(input.NodeCapabilities)
	if err != nil {
		return err
	}
	bound, err := normalizedNodeCapabilities(input.BoundServices)
	if err != nil {
		return fmt.Errorf("DeploymentBinding 服务能力无效: %w", err)
	}
	requiredValues := compatibility.RequiredNodeCapabilities
	if input.Engine != "" {
		var ok bool
		if _, ok = engineServiceGroup(input.Engine); !ok {
			return fmt.Errorf("DeploymentBinding 引擎无效: %s", input.Engine)
		}
		requiredValues = engineBindingCapabilities(input.Engine)
	}
	required, err := normalizedNodeCapabilities(requiredValues)
	if err != nil {
		return fmt.Errorf("Release requiredNodeCapabilities 无效: %w", err)
	}
	for capability := range required {
		if _, exists := available[capability]; !exists {
			return fmt.Errorf("本机不具备 Release 所需能力: %s", capability)
		}
		if _, exists := bound[capability]; !exists {
			return fmt.Errorf("DeploymentBinding 未启用 Release 所需能力: %s", capability)
		}
	}
	return nil
}

func normalizedNodeCapabilities(values []string) (map[string]struct{}, error) {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !validServiceGroup(ServiceGroup(value)) {
			return nil, fmt.Errorf("未知节点能力: %s", value)
		}
		if _, exists := result[value]; exists {
			return nil, fmt.Errorf("节点能力重复: %s", value)
		}
		result[value] = struct{}{}
	}
	return result, nil
}

func validProjectCode(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for index, character := range value {
		letterOrDigit := (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9')
		if (index == 0 && !letterOrDigit) || (index > 0 && !letterOrDigit && !strings.ContainsRune("._-", character)) {
			return false
		}
	}
	return true
}

func validVersionValue(value string) bool {
	// __DEV__ 是控制面固定的内部开发制品标签，不是用户可输入的 Release 版本。
	if value == "__DEV__" {
		return true
	}
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for index, character := range value {
		letterOrDigit := (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9')
		if (index == 0 && !letterOrDigit) || (index > 0 && !letterOrDigit && !strings.ContainsRune("._+-", character)) {
			return false
		}
	}
	return true
}

// compatibility 版本按 SemVer 2.0.0 比较。契约要求 MAJOR.MINOR.PATCH 且不带 v
// 前缀；build metadata 是标准 SemVer 的一部分，但不参与版本先后比较。
func validComparableVersion(value string) bool {
	_, err := normalizedComparableSemver(value)
	return err == nil
}

func versionAtLeast(actual, minimum string) (bool, error) {
	actualVersion, err := normalizedComparableSemver(actual)
	if err != nil {
		return false, err
	}
	minimumVersion, err := normalizedComparableSemver(minimum)
	if err != nil {
		return false, err
	}
	return semver.Compare(actualVersion, minimumVersion) >= 0, nil
}

func normalizedComparableSemver(value string) (string, error) {
	if value == "" || value != strings.TrimSpace(value) || len(value) > 64 || strings.HasPrefix(value, "v") {
		return "", errors.New("版本格式不支持")
	}
	normalized := "v" + value
	if !semver.IsValid(normalized) {
		return "", errors.New("版本不是严格 SemVer")
	}
	// x/mod 允许 v1 与 v1.2 这两个 Go 专用简写；Release compatibility 必须固定
	// MAJOR.MINOR.PATCH，以免与中心契约的版本含义出现歧义。
	core := strings.TrimPrefix(strings.SplitN(strings.SplitN(normalized, "+", 2)[0], "-", 2)[0], "v")
	if len(strings.SplitN(core, ".", 4)) != 3 {
		return "", errors.New("版本必须包含 MAJOR.MINOR.PATCH")
	}
	return normalized, nil
}
