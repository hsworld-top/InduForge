package ops

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestValidateDeployableReleaseAcceptsFrozenManifest(t *testing.T) {
	err := validateDeployableRelease(
		testProjectID,
		testVersionID,
		"versions/tenant/project/release.tar.zst",
		strings.Repeat("a", 64),
		strings.Repeat("e", 64),
		strings.Repeat("f", 64),
		"induforge-release-2026-01",
		[]byte(validReleaseManifest(testProjectID)),
	)
	if err != nil {
		t.Fatalf("valid formal release rejected: %v", err)
	}
}

func TestValidateDeployableReleaseAcceptsContractExtensionAndUppercaseDigest(t *testing.T) {
	manifest := strings.Replace(
		validReleaseManifest(testProjectID),
		`"buildTime":"2026-08-31T00:00:00Z"`,
		`"buildTime":"2026-08-31T00:00:00Z","signatureRef":"signature.sig"`,
		1,
	)
	if err := validateDeployableRelease(
		testProjectID,
		testVersionID,
		"releases/tenant/project/release.tar.zst",
		strings.Repeat("A", 64),
		strings.Repeat("E", 64),
		strings.Repeat("F", 64),
		"induforge-release-2026-01",
		[]byte(manifest),
	); err != nil {
		t.Fatalf("valid extended formal release rejected: %v", err)
	}
}

func TestValidateDeployableReleaseAcceptsContractStableKeyID(t *testing.T) {
	manifest := strings.Replace(validReleaseManifest(testProjectID), "induforge-release-2026-01", "urn:induforge:key:2026-01", 1)
	if err := validateDeployableRelease(
		testProjectID,
		testVersionID,
		"releases/tenant/project/release.tar.zst",
		strings.Repeat("a", 64),
		strings.Repeat("e", 64),
		strings.Repeat("f", 64),
		"urn:induforge:key:2026-01",
		[]byte(manifest),
	); err != nil {
		t.Fatalf("valid stable signing key ID rejected: %v", err)
	}
}

func TestValidateDeployableReleaseRequiresCollectorWhenDeploymentEnablesIt(t *testing.T) {
	var manifest map[string]any
	if err := json.Unmarshal([]byte(validReleaseManifest(testProjectID)), &manifest); err != nil {
		t.Fatalf("decode test manifest: %v", err)
	}
	artifacts, ok := manifest["artifacts"].(map[string]any)
	if !ok {
		t.Fatal("test manifest artifacts missing")
	}
	delete(artifacts, "collector")
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("encode test manifest: %v", err)
	}
	err = validateDeployableReleaseForDeployment(
		testProjectID,
		testVersionID,
		"releases/tenant/project/release.tar.zst",
		strings.Repeat("a", 64),
		strings.Repeat("e", 64),
		strings.Repeat("f", 64),
		"induforge-release-2026-01",
		true,
		manifestJSON,
	)
	if !errors.Is(err, ErrReleaseNotDeployable) {
		t.Fatalf("collector deployment without collector artifact must be rejected, got %v", err)
	}
}

func TestDeploymentRequirementsRequireCollectorArtifactFromManifest(t *testing.T) {
	var manifest map[string]any
	if err := json.Unmarshal([]byte(validReleaseManifest(testProjectID)), &manifest); err != nil {
		t.Fatal(err)
	}
	manifest["capabilities"] = []any{ServiceBase, ServiceCollector}
	delete(manifest["artifacts"].(map[string]any), "collector")
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	_, err = deploymentRequirementsForRelease(releaseMetadata{ID: testVersionID, ArtifactKey: "releases/tenant/project/release.tar.zst", ArtifactHash: strings.Repeat("a", 64), ArtifactSize: 1, ManifestHash: strings.Repeat("e", 64), ChecksumsHash: strings.Repeat("f", 64), SigningKeyID: "induforge-release-2026-01", Manifest: raw}, testProjectID)
	if !errors.Is(err, ErrReleaseNotDeployable) {
		t.Fatalf("采集引擎必须要求受信 collector 工件: %v", err)
	}
}

func TestValidateDeployableReleaseRejectsSourceSnapshotAndMalformedMaterial(t *testing.T) {
	valid := validReleaseManifest(testProjectID)
	tests := []struct {
		name         string
		projectID    string
		artifactKey  string
		artifactHash string
		manifest     string
	}{
		{name: "source snapshot", projectID: testProjectID, artifactKey: "versions/release.tar.zst", artifactHash: strings.Repeat("a", 64), manifest: `{"schemaVersion":"2.0.0","projectId":"` + testProjectID + `","source":"code-first"}`},
		{name: "project mismatch", projectID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", artifactKey: "versions/release.tar.zst", artifactHash: strings.Repeat("a", 64), manifest: valid},
		{name: "unsafe object key", projectID: testProjectID, artifactKey: "../release.ifp", artifactHash: strings.Repeat("a", 64), manifest: valid},
		{name: "parent object key", projectID: testProjectID, artifactKey: "..", artifactHash: strings.Repeat("a", 64), manifest: valid},
		{name: "legacy non-bundle object", projectID: testProjectID, artifactKey: "versions/release.ifp", artifactHash: strings.Repeat("a", 64), manifest: valid},
		{name: "archive digest", projectID: testProjectID, artifactKey: "versions/release.tar.zst", artifactHash: "sha256:short", manifest: valid},
		{name: "client digest", projectID: testProjectID, artifactKey: "versions/release.tar.zst", artifactHash: strings.Repeat("a", 64), manifest: strings.Replace(valid, "sha256:"+strings.Repeat("b", 64), "sha256:short", 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateDeployableRelease(test.projectID, testVersionID, test.artifactKey, test.artifactHash, strings.Repeat("e", 64), strings.Repeat("f", 64), "induforge-release-2026-01", []byte(test.manifest))
			if !errors.Is(err, ErrReleaseNotDeployable) {
				t.Fatalf("expected ErrReleaseNotDeployable, got %v", err)
			}
		})
	}
}

func TestDevelopmentArtifactAllowsCollectorSnapshotWithoutProductionDigest(t *testing.T) {
	manifest := collectorRequiredManifestWithoutSnapshot(t)
	production := releaseMetadata{ID: testVersionID, ArtifactKey: "releases/tenant/project/release.tar.zst", ArtifactHash: strings.Repeat("a", 64), ArtifactSize: 1, ManifestHash: strings.Repeat("e", 64), ChecksumsHash: strings.Repeat("f", 64), SigningKeyID: "induforge-release-2026-01", Manifest: manifest}
	if _, err := deploymentRequirementsForRelease(production, testProjectID); !errors.Is(err, ErrReleaseNotDeployable) {
		t.Fatalf("生产 Release 缺少冻结 sourceSnapshot 不应通过: %v", err)
	}
	development := &DevelopmentArtifact{ReleaseID: testVersionID, Version: "__DEV__", ArtifactKey: "development/tenant/project/dev.tar.zst", ArtifactHash: strings.Repeat("a", 64), ArtifactSize: 1, ManifestHash: strings.Repeat("e", 64), ChecksumsHash: strings.Repeat("f", 64), SigningKeyID: "induforge-release-2026-01", Manifest: manifest}
	metadata, err := developmentReleaseMetadata(development, testProjectID)
	if err != nil {
		t.Fatalf("受控开发制品不应复用生产冻结快照限制: %v", err)
	}
	if required, err := deploymentRequirementsForDevelopmentArtifact(metadata, testProjectID); err != nil || !sameStrings(required, []string{ServiceBase, ServiceCollector}) {
		t.Fatalf("开发制品引擎需求无效 required=%v err=%v", required, err)
	}
	if _, _, err := collectorSourceSnapshotFromManifest(manifest, testProjectID); err == nil {
		t.Fatal("生产运行上下文接受了缺少冻结摘要的采集快照")
	}
	if snapshot, required, err := developmentCollectorSourceSnapshotFromManifest(manifest, testProjectID); err != nil || !required || len(snapshot) == 0 {
		t.Fatalf("开发运行上下文未接受受控采集快照 required=%v err=%v", required, err)
	}
}

func collectorRequiredManifestWithoutSnapshot(t *testing.T) []byte {
	t.Helper()
	manifest := validReleaseManifest(testProjectID)
	manifest = strings.Replace(manifest, `"capabilities":["runtime.auth","runtime.datapoint"]`, `"capabilities":["collector"]`, 1)
	manifest = strings.Replace(manifest, `"requiredNodeCapabilities":["project_entry","data_runtime"]`, `"requiredNodeCapabilities":["project_entry","data_runtime","collector"]`, 1)
	source := collectorSourceSnapshotForManifest(testProjectID)
	sourceDigest := sha256.Sum256(source)
	manifest = strings.Replace(manifest, `,"sourceSnapshotSha256":"sha256:`+hex.EncodeToString(sourceDigest[:])+`"`, "", 1)
	if strings.Contains(manifest, "sourceSnapshotSha256") {
		t.Fatal("未移除生产 sourceSnapshot 摘要")
	}
	return []byte(manifest)
}

func validReleaseManifest(projectID string) string {
	source := collectorSourceSnapshotForManifest(projectID)
	sourceDigest := sha256.Sum256(source)
	return `{
  "schemaVersion":"2.0",
  "projectId":"` + projectID + `",
  "projectCode":"factory_dashboard",
  "releaseId":"` + testVersionID + `",
  "version":"2026.08.31-001",
  "artifacts":{
    "client":{"file":"client-assets.tar.zst","checksum":"sha256:` + strings.Repeat("b", 64) + `"},
    "runtime":{"file":"runtime-artifact.tar.zst","checksum":"sha256:` + strings.Repeat("c", 64) + `"},
    "collector":{"file":"collector-artifact.tar.zst","checksum":"sha256:` + strings.Repeat("d", 64) + `","sourceSnapshot":` + string(source) + `,"sourceSnapshotSha256":"sha256:` + hex.EncodeToString(sourceDigest[:]) + `"}
  },
  "capabilities":["runtime.auth","runtime.datapoint"],
  "compatibility":{"minNodeAgentVersion":"1.0.0","minRuntimeVersion":"1.0.0","requiredNodeCapabilities":["project_entry","data_runtime"]},
  "supplyChain":{"sbomRef":"sbom.cdx.json","sourceRevision":"git:abc","builderId":"induforge-release-builder-v1","signingKeyId":"induforge-release-2026-01","promotable":true},
  "preflight":{"resourceRecommendationRef":"resource-recommendation.json","healthContractRef":"health-contract.json","schemaPlanRef":"schema-plan.json"},
  "buildTime":"2026-08-31T00:00:00Z"
}`
}

func collectorSourceSnapshotForManifest(projectID string) []byte {
	artifact := []byte(`{"schemaVersion":"collector-runtime-artifact.v1","artifactId":"collector-a","artifactRevision":1,"projectId":"` + projectID + `"}`)
	sum := sha256.Sum256(artifact)
	raw, _ := json.Marshal(map[string]any{"schemaVersion": "collector-runtime-artifact.v1", "projectId": projectID, "artifactRevision": 1, "sha256": "sha256:" + hex.EncodeToString(sum[:]), "size": len(artifact), "artifact": json.RawMessage(artifact)})
	return raw
}
