package ops

import (
	"bytes"
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
	File     string `json:"file"`
	Checksum string `json:"checksum"`
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
		MinSiteControllerVersion string   `json:"minSiteControllerVersion"`
		MinRuntimeVersion        string   `json:"minRuntimeVersion"`
		RequiredSiteCapabilities []string `json:"requiredSiteCapabilities"`
	} `json:"compatibility"`
	SupplyChain struct {
		SBOMRef        string `json:"sbomRef"`
		SourceRevision string `json:"sourceRevision"`
		BuilderID      string `json:"builderId"`
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
func validateDeployableRelease(projectID, artifactKey, artifactHash string, manifestJSON []byte) error {
	return validateDeployableReleaseForDeployment(projectID, artifactKey, artifactHash, false, manifestJSON)
}

// validateDeployableReleaseForDeployment 仅校验正式 Release 的不可变元数据；
// 不解析构建内容，也不信任 Manifest 之外的制品位置或摘要。
func validateDeployableReleaseForDeployment(projectID, artifactKey, artifactHash string, requireCollector bool, manifestJSON []byte) error {
	if strings.TrimSpace(artifactKey) == "" || !validObjectKey(artifactKey) {
		return fmt.Errorf("%w: Release 对象键缺失或无效", ErrReleaseNotDeployable)
	}
	if !validSHA256Hex(artifactHash) {
		return fmt.Errorf("%w: Release 归档摘要缺失或无效", ErrReleaseNotDeployable)
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
	if manifest.SchemaVersion != releaseManifestSchemaVersion || manifest.ProjectID != projectID || strings.TrimSpace(manifest.ProjectCode) == "" || strings.TrimSpace(manifest.ReleaseID) == "" || strings.TrimSpace(manifest.Version) == "" || strings.TrimSpace(manifest.BuildTime) == "" {
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
	if manifest.Artifacts.Collector != nil {
		if err := validateReleaseArtifact(*manifest.Artifacts.Collector, collectorArtifactFile); err != nil {
			return err
		}
	} else if requireCollector {
		return fmt.Errorf("%w: 部署已启用采集服务，但 Release 未声明 %s", ErrReleaseNotDeployable, collectorArtifactFile)
	}
	if strings.TrimSpace(manifest.Compatibility.MinSiteControllerVersion) == "" || strings.TrimSpace(manifest.Compatibility.MinRuntimeVersion) == "" {
		return fmt.Errorf("%w: Release 缺少运行兼容性声明", ErrReleaseNotDeployable)
	}
	if strings.TrimSpace(manifest.SupplyChain.SBOMRef) == "" || strings.TrimSpace(manifest.SupplyChain.SourceRevision) == "" || strings.TrimSpace(manifest.SupplyChain.BuilderID) == "" || manifest.SupplyChain.Promotable == nil {
		return fmt.Errorf("%w: Release 缺少供应链声明", ErrReleaseNotDeployable)
	}
	if strings.TrimSpace(manifest.Preflight.ResourceRecommendationRef) == "" || strings.TrimSpace(manifest.Preflight.HealthContractRef) == "" || strings.TrimSpace(manifest.Preflight.SchemaPlanRef) == "" {
		return fmt.Errorf("%w: Release 缺少部署前检查声明", ErrReleaseNotDeployable)
	}
	return nil
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
