package ops

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"time"
)

const (
	releaseManifestSchemaVersion = "2.0"
	clientArtifactFile           = "client-assets.tar.zst"
	runtimeArtifactFile          = "runtime-artifact.tar.zst"
	collectorArtifactFile        = "collector-artifact.tar.zst"
)

type releaseArtifactDescriptor struct {
	File                 string          `json:"file"`
	Checksum             string          `json:"checksum"`
	SourceSnapshot       json.RawMessage `json:"sourceSnapshot,omitempty"`
	SourceSnapshotSHA256 string          `json:"sourceSnapshotSha256,omitempty"`
}

type deployableReleaseManifest struct {
	SchemaVersion string   `json:"schemaVersion"`
	ProjectID     string   `json:"projectId"`
	ProjectCode   string   `json:"projectCode"`
	ReleaseID     string   `json:"releaseId"`
	Version       string   `json:"version"`
	Capabilities  []string `json:"capabilities,omitempty"`
	BuildTime     string   `json:"buildTime"`
	Artifacts     struct {
		Client    releaseArtifactDescriptor  `json:"client"`
		Runtime   releaseArtifactDescriptor  `json:"runtime"`
		Collector *releaseArtifactDescriptor `json:"collector,omitempty"`
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
		Promotable     *bool  `json:"promotable"`
	} `json:"supplyChain"`
	Preflight struct {
		ResourceRecommendationRef string `json:"resourceRecommendationRef"`
		HealthContractRef         string `json:"healthContractRef"`
		SchemaPlanRef             string `json:"schemaPlanRef"`
	} `json:"preflight"`
}

// validateDeployableRelease 把“版本构建成功”和“可在物理节点部署”分开。
// 旧 code-first IFP 只有源码快照，不能因为 status=ready 就进入正式运维链路。
func validateDeployableRelease(projectID, releaseID, artifactKey, artifactHash, manifestHash, checksumsHash, signingKeyID string, manifestJSON []byte) error {
	return validateDeployableReleaseForDeployment(projectID, releaseID, artifactKey, artifactHash, manifestHash, checksumsHash, signingKeyID, false, manifestJSON)
}

// validateDeployableReleaseForDeployment 仅校验正式 Release 的不可变元数据；
// 不解析构建内容，也不信任 Manifest 之外的制品位置或摘要。
func validateDeployableReleaseForDeployment(projectID, releaseID, artifactKey, artifactHash, manifestHash, checksumsHash, signingKeyID string, requireCollector bool, manifestJSON []byte) error {
	return validateDeployableArtifactForDeployment(projectID, releaseID, artifactKey, artifactHash, manifestHash, checksumsHash, signingKeyID, requireCollector, true, manifestJSON)
}

// validateDeployableDevelopmentArtifactForDeployment 校验服务端构建并写入开发绑定
// 的临时制品。它仍校验制品身份、摘要、工件和兼容性，但不把生产 Release 冻结
// sourceSnapshot 的字段约束套用到开发快照；开发采集绑定随后由数据域按当前权威
// 快照再次校验，不能借此绕过数据域的项目隔离与完整性检查。
func validateDeployableDevelopmentArtifactForDeployment(projectID, releaseID, artifactKey, artifactHash, manifestHash, checksumsHash, signingKeyID string, requireCollector bool, manifestJSON []byte) error {
	return validateDeployableArtifactForDeployment(projectID, releaseID, artifactKey, artifactHash, manifestHash, checksumsHash, signingKeyID, requireCollector, false, manifestJSON)
}

func validateDeployableArtifactForDeployment(projectID, releaseID, artifactKey, artifactHash, manifestHash, checksumsHash, signingKeyID string, requireCollector, requireFrozenCollectorSnapshot bool, manifestJSON []byte) error {
	if strings.TrimSpace(artifactKey) == "" || !validObjectKey(artifactKey) || !strings.HasSuffix(artifactKey, ".tar.zst") {
		return fmt.Errorf("%w: Release 对象键缺失或无效", ErrReleaseNotDeployable)
	}
	if !validSHA256Hex(artifactHash) || !validSHA256Hex(manifestHash) || !validSHA256Hex(checksumsHash) || !validStableID(signingKeyID) {
		return fmt.Errorf("%w: Release 归档、Manifest、摘要清单或签名密钥元数据缺失或无效", ErrReleaseNotDeployable)
	}

	var manifest deployableReleaseManifest
	// Manifest 是跨模块、可演进的契约。只读取部署决策所需字段，不能因新增字段拒绝
	// 仍满足当前版本必填约束的 Release。
	decoder := json.NewDecoder(bytes.NewReader(manifestJSON))
	if err := decoder.Decode(&manifest); err != nil {
		return fmt.Errorf("%w: Release Manifest 无效: %v", ErrReleaseNotDeployable, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("%w: Release Manifest 只能包含一个 JSON 对象", ErrReleaseNotDeployable)
	}
	if manifest.SchemaVersion != releaseManifestSchemaVersion || (projectID != "" && manifest.ProjectID != projectID) || manifest.ReleaseID != releaseID || strings.TrimSpace(manifest.ProjectCode) == "" || strings.TrimSpace(manifest.Version) == "" || strings.TrimSpace(manifest.BuildTime) == "" {
		return fmt.Errorf("%w: Release Manifest 身份或协议版本不匹配", ErrReleaseNotDeployable)
	}
	if _, err := time.Parse(time.RFC3339, manifest.BuildTime); err != nil {
		return fmt.Errorf("%w: Release Manifest 构建时间无效", ErrReleaseNotDeployable)
	}
	if err := validateReleaseArtifact(manifest.Artifacts.Client, clientArtifactFile); err != nil {
		return err
	}
	if err := validateReleaseArtifact(manifest.Artifacts.Runtime, runtimeArtifactFile); err != nil {
		return err
	}
	collectorRequired := requireCollector
	for _, capability := range manifest.Capabilities {
		if capability == ServiceCollector {
			collectorRequired = true
			break
		}
	}
	if manifest.Artifacts.Collector != nil {
		if err := validateReleaseArtifact(*manifest.Artifacts.Collector, collectorArtifactFile); err != nil {
			return err
		}
		if collectorRequired && requireFrozenCollectorSnapshot && !validCollectorSourceSnapshot(manifest.Artifacts.Collector.SourceSnapshot, manifest.Artifacts.Collector.SourceSnapshotSHA256, projectID) {
			return fmt.Errorf("%w: Release collector sourceSnapshot 无效", ErrReleaseNotDeployable)
		}
	} else if collectorRequired {
		return fmt.Errorf("%w: 部署已启用采集服务，但 Release 未声明 %s", ErrReleaseNotDeployable, collectorArtifactFile)
	}
	if strings.TrimSpace(manifest.Compatibility.MinNodeAgentVersion) == "" || strings.TrimSpace(manifest.Compatibility.MinRuntimeVersion) == "" {
		return fmt.Errorf("%w: Release 缺少运行兼容性声明", ErrReleaseNotDeployable)
	}
	requiredNodeCapabilities := []string{CapabilityProjectEntry, CapabilityDataRuntime}
	if requireCollector {
		requiredNodeCapabilities = append(requiredNodeCapabilities, CapabilityCollector)
	}
	if !sameStrings(manifest.Compatibility.RequiredNodeCapabilities, requiredNodeCapabilities) {
		return fmt.Errorf("%w: Release 节点能力声明与部署服务不匹配", ErrReleaseNotDeployable)
	}
	if strings.TrimSpace(manifest.SupplyChain.SBOMRef) == "" || strings.TrimSpace(manifest.SupplyChain.SourceRevision) == "" || strings.TrimSpace(manifest.SupplyChain.BuilderID) == "" || manifest.SupplyChain.Promotable == nil || !*manifest.SupplyChain.Promotable || manifest.SupplyChain.SigningKeyID != signingKeyID {
		return fmt.Errorf("%w: Release 缺少供应链声明", ErrReleaseNotDeployable)
	}
	if strings.TrimSpace(manifest.Preflight.ResourceRecommendationRef) == "" || strings.TrimSpace(manifest.Preflight.HealthContractRef) == "" || strings.TrimSpace(manifest.Preflight.SchemaPlanRef) == "" {
		return fmt.Errorf("%w: Release 缺少部署前检查声明", ErrReleaseNotDeployable)
	}
	return nil
}

func validCollectorSourceSnapshot(raw []byte, digest, projectID string) bool {
	if len(raw) == 0 || len(raw) > 16<<20 || !validSHA256Checksum(digest) {
		return false
	}
	canonicalSnapshot, ok := canonicalJSONBytes(raw)
	if !ok {
		return false
	}
	sum := sha256.Sum256(canonicalSnapshot)
	if digest != "sha256:"+hex.EncodeToString(sum[:]) {
		return false
	}
	var source struct {
		SchemaVersion    string          `json:"schemaVersion"`
		ProjectID        string          `json:"projectId"`
		ArtifactRevision int64           `json:"artifactRevision"`
		SHA256           string          `json:"sha256"`
		Size             int             `json:"size"`
		Artifact         json.RawMessage `json:"artifact"`
	}
	if json.Unmarshal(canonicalSnapshot, &source) != nil {
		return false
	}
	canonicalArtifact, artifactOK := canonicalJSONBytes(source.Artifact)
	return artifactOK && source.SchemaVersion == "collector-runtime-artifact.v1" && source.ProjectID == projectID && source.ArtifactRevision > 0 && source.Size == len(canonicalArtifact) && source.SHA256 == "sha256:"+sha256Hex(canonicalArtifact)
}

func canonicalJSONBytes(raw []byte) ([]byte, bool) {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return nil, false
	}
	canonical, err := json.Marshal(value)
	return canonical, err == nil
}

func sha256Hex(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

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

func validateReleaseArtifact(artifact releaseArtifactDescriptor, expectedFile string) error {
	if artifact.File != expectedFile || !validSHA256Checksum(artifact.Checksum) {
		return fmt.Errorf("%w: Release 缺少有效的 %s", ErrReleaseNotDeployable, expectedFile)
	}
	return nil
}

func validSHA256Checksum(value string) bool {
	const prefix = "sha256:"
	return strings.HasPrefix(value, prefix) && validSHA256Hex(strings.TrimPrefix(value, prefix))
}

func validSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func validObjectKey(value string) bool {
	if strings.ContainsAny(value, "\\\r\n\t\x00") {
		return false
	}
	cleaned := path.Clean(strings.TrimSpace(value))
	return cleaned == value && cleaned != "." && cleaned != ".." && !strings.HasPrefix(cleaned, "/") && !strings.HasPrefix(cleaned, "../")
}
