package deployment

import (
	"crypto/ed25519"
	"fmt"
	"strings"
	"time"

	"github.com/indu-forge/dev_core/internal/releasebuilder"
)

func assembleFormalRelease(project Project, version Version, source ReleaseSource, signing SigningConfig, config ServiceConfig, now time.Time) (releasebuilder.Result, error) {
	if len(signing.Key) != ed25519.PrivateKeySize || strings.TrimSpace(signing.KeyID) == "" || strings.TrimSpace(config.MinNodeAgentVersion) == "" || strings.TrimSpace(config.MinRuntimeVersion) == "" {
		return releasebuilder.Result{}, fmt.Errorf("正式 Release 签名或最小运行版本未配置")
	}
	caps := []string{"project_entry", "data_runtime"}
	if len(source.Collector) > 0 {
		caps = append(caps, "collector")
	}
	return releasebuilder.Build(releasebuilder.Input{ProjectID: project.ID, ProjectCode: project.Code, ReleaseID: version.ID, Version: version.Version, Client: source.Client, Runtime: source.Runtime, Collector: source.Collector, SBOM: source.SBOM, ResourceRecommendation: source.ResourceRecommendation, HealthContract: source.HealthContract, SchemaPlan: source.SchemaPlan, ProjectDocument: source.ProjectDocument, MinNodeAgentVersion: config.MinNodeAgentVersion, MinRuntimeVersion: config.MinRuntimeVersion, RequiredNodeCapabilities: caps, SourceRevision: source.SourceRevision, BuilderID: source.BuilderID, Promotable: true, BuildTime: now.UTC(), SigningKey: signing.Key, SigningKeyID: signing.KeyID})
}
