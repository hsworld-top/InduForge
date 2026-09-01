package ops

import (
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

func validReleaseManifest(projectID string) string {
	return `{
  "schemaVersion":"2.0",
  "projectId":"` + projectID + `",
  "projectCode":"factory_dashboard",
  "releaseId":"` + testVersionID + `",
  "version":"2026.08.31-001",
  "artifacts":{
    "client":{"file":"client-assets.tar.zst","checksum":"sha256:` + strings.Repeat("b", 64) + `"},
    "runtime":{"file":"runtime-artifact.tar.zst","checksum":"sha256:` + strings.Repeat("c", 64) + `"},
    "collector":{"file":"collector-artifact.tar.zst","checksum":"sha256:` + strings.Repeat("d", 64) + `"}
  },
  "capabilities":["runtime.auth","runtime.datapoint"],
  "compatibility":{"minNodeAgentVersion":"1.0.0","minRuntimeVersion":"1.0.0","requiredNodeCapabilities":["project_entry","data_runtime"]},
  "supplyChain":{"sbomRef":"sbom.cdx.json","sourceRevision":"git:abc","builderId":"induforge-release-builder-v1","signingKeyId":"induforge-release-2026-01","promotable":true},
  "preflight":{"resourceRecommendationRef":"resource-recommendation.json","healthContractRef":"health-contract.json","schemaPlanRef":"schema-plan.json"},
  "buildTime":"2026-08-31T00:00:00Z"
}`
}
