package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/indu-forge/dev_core/internal/config"
	"github.com/indu-forge/dev_core/internal/deployment"
)

func TestConfigureReleasePublishingKeepsDefaultDisabled(t *testing.T) {
	if err := configureReleasePublishing(nil, config.Config{}, nil); err != nil {
		t.Fatal(err)
	}
}
func TestConfigureReleasePublishingWiresEnabledDependencies(t *testing.T) {
	_, key, _ := ed25519.GenerateKey(rand.Reader)
	raw, _ := x509.MarshalPKCS8PrivateKey(key)
	file := filepath.Join(t.TempDir(), "key.pem")
	if err := os.WriteFile(file, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: raw}), 0o600); err != nil {
		t.Fatal(err)
	}
	s := deployment.NewService(nil, nil, nil, deployment.ServiceConfig{})
	cfg := config.Config{ReleaseBuilderEnabled: true, ReleaseBuilderImage: "builder", ReleaseBuilderID: "release-builder-v1", ReleaseSigningKeyFile: file, ReleaseSigningKeyID: "key-v1", MinNodeAgentVersion: "1.0.0", MinRuntimeVersion: "1.0.0", DataServiceURL: "http://data.example", CodeWorkspaceVolume: "workspaces", WorkspaceRoot: t.TempDir()}
	if err := configureReleasePublishing(s, cfg, nil); err != nil {
		t.Fatal(err)
	}
}
