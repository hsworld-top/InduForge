package ops

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProductionServiceHelper(t *testing.T) {
	if !strings.Contains(strings.Join(os.Args, " "), "local-service-helper") {
		return
	}
	for {
		time.Sleep(time.Second)
	}
}

func testSupervisor(t *testing.T) *Supervisor {
	return NewSupervisor("unused", t.TempDir(), t.TempDir())
}

func configuredSupervisor(t *testing.T, group ServiceGroup) *Supervisor {
	t.Helper()
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok", "data": map[string]string{"status": "UP"}, "reqId": "test"})
	}))
	t.Cleanup(health.Close)
	root := t.TempDir()
	release := filepath.Join(root, "releases", "v1")
	if err := os.MkdirAll(filepath.Join(release, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(os.Args[0], filepath.Join(release, "bin", "service")); err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{"release":"test"}`)
	if err := os.WriteFile(filepath.Join(release, "release-manifest.json"), manifest, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("releases", "v1"), filepath.Join(root, "current")); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(manifest)
	s, err := NewSupervisorWithConfig(SupervisorConfig{StateDir: filepath.Join(root, "state"), LogDir: filepath.Join(root, "logs"), Services: []ServiceConfig{{Group: group, Component: "test-service", Enabled: true, ReleaseRoot: root, Current: "current", ReleaseDigest: "sha256:" + hex.EncodeToString(digest[:]), Executable: "bin/service", Arguments: []string{"-test.run=TestProductionServiceHelper", "--", "local-service-helper"}, HealthURL: health.URL, HealthTimeout: time.Second, DrainTimeout: time.Second}}})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSupervisorStartsConfiguredProductionServiceNotDemo(t *testing.T) {
	s := configuredSupervisor(t, ServiceCollector)
	started, err := s.Start("collector-a", ServiceCollector, 2)
	if err != nil {
		t.Fatal(err)
	}
	if started.State != "running" || started.PID <= 0 || started.Generation != 2 {
		t.Fatalf("unexpected: %+v", started)
	}
	logs, err := s.Logs("collector-a", 10)
	if err != nil || strings.Contains(strings.Join(logs, "\n"), "demo-workload") {
		t.Fatalf("production command must not use demo: logs=%v err=%v", logs, err)
	}
	stopped, err := s.Stop("collector-a", 3)
	if err != nil {
		t.Fatal(err)
	}
	if stopped.State != "stopped" || stopped.Generation != 3 {
		t.Fatalf("unexpected: %+v", stopped)
	}
}

func TestSupervisorRejectsUnknownOrUnassignedService(t *testing.T) {
	s := configuredSupervisor(t, ServiceCollector)
	if _, err := s.Start("unknown", "shell", 1); err == nil {
		t.Fatal("expected unknown service group rejection")
	}
	if _, err := s.Start("not-local", ServiceDataRuntime, 1); err == nil {
		t.Fatal("expected unassigned service group rejection")
	}
}

func TestInstalledTemplatesClaimCapabilitiesButCannotStartWithoutRelease(t *testing.T) {
	s, err := NewSupervisorWithConfig(SupervisorConfig{
		StateDir: t.TempDir(),
		LogDir:   t.TempDir(),
		Services: []ServiceConfig{
			{Group: ServiceProjectEntry, Component: "project-gateway", Installed: true},
			{Group: ServiceProjectEntry, Component: "runtime-api", Installed: true},
			{Group: ServiceDataRuntime, Component: "runtime-engine", Installed: true},
			{Group: ServiceCollector, Component: "collector", Installed: false},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(s.Capabilities(), ","); got != "project_entry,data_runtime" {
		t.Fatalf("capabilities=%s", got)
	}
	if s.HasService(ServiceProjectEntry) || s.HasService(ServiceDataRuntime) {
		t.Fatal("disabled templates must not be runnable services")
	}
	if _, err := s.Start("service-a", ServiceProjectEntry, 1); err == nil {
		t.Fatal("template without local release config must fail closed")
	}
}

func TestSupervisorRejectsNonLoopbackHealthProbe(t *testing.T) {
	service := ServiceConfig{Group: ServiceCollector, Component: "collector", Enabled: true, ReleaseRoot: t.TempDir(), Current: "current", ReleaseDigest: "sha256:" + strings.Repeat("0", 64), Executable: "bin/collector", HealthURL: "http://example.invalid/health"}
	if _, err := NewSupervisorWithConfig(SupervisorConfig{StateDir: t.TempDir(), LogDir: t.TempDir(), Services: []ServiceConfig{service}}); err == nil {
		t.Fatal("expected non-loopback health URL rejection")
	}
}

func TestProjectGatewayPublicURLIsLocalAndExclusive(t *testing.T) {
	base := ServiceConfig{
		Group:         ServiceProjectEntry,
		Component:     "project-gateway",
		Enabled:       true,
		ReleaseRoot:   t.TempDir(),
		Current:       "current",
		ReleaseDigest: "sha256:" + strings.Repeat("0", 64),
		Executable:    "bin/project-gateway",
		HealthURL:     "http://127.0.0.1:18080/health",
		PublicURL:     "https://gateway.example.com/engineering",
	}
	if err := validateServiceConfig(base); err != nil {
		t.Fatalf("valid project gateway URL rejected: %v", err)
	}
	invalid := []ServiceConfig{
		func() ServiceConfig { value := base; value.Group = ServiceCollector; return value }(),
		func() ServiceConfig {
			value := base
			value.PublicURL = "https://user:pass@gateway.example.com"
			return value
		}(),
		func() ServiceConfig {
			value := base
			value.PublicURL = "https://gateway.example.com/#fragment"
			return value
		}(),
	}
	for _, candidate := range invalid {
		if err := validateServiceConfig(candidate); err == nil {
			t.Fatalf("invalid public URL configuration was accepted: %+v", candidate)
		}
	}
}

func TestSupervisorRejectsReleaseDigestMismatch(t *testing.T) {
	s := configuredSupervisor(t, ServiceCollector)
	service := s.services[ServiceCollector][0]
	service.ReleaseDigest = "sha256:" + strings.Repeat("0", 64)
	s.services[ServiceCollector] = []ServiceConfig{service}
	if _, err := s.Start("collector-b", ServiceCollector, 1); err == nil {
		t.Fatal("expected digest rejection")
	}
}

func TestSupervisorPersistsAndRecoversPIDStatus(t *testing.T) {
	s := configuredSupervisor(t, ServiceCollector)
	started, err := s.Start("collector-recover", ServiceCollector, 1)
	if err != nil {
		t.Fatal(err)
	}
	state, err := os.ReadFile(s.statePath())
	if err != nil || !strings.Contains(string(state), "collector-recover") {
		t.Fatalf("state not persisted: %s err=%v", state, err)
	}
	// The persisted record is intentionally only recovered for a locally
	// configured service group. A changed/removed config cannot adopt it.
	recovered, err := NewSupervisorWithConfig(SupervisorConfig{StateDir: s.stateDir, LogDir: s.logDir, Services: s.services[ServiceCollector]})
	if err != nil {
		t.Fatal(err)
	}
	status, err := recovered.Status("collector-recover")
	if err != nil || status.PID != started.PID || status.State != "running" {
		t.Fatalf("not recovered: %+v err=%v", status, err)
	}
	_, _ = recovered.Stop("collector-recover", 2)
}

func TestProjectEntryStartsEveryDeclaredComponent(t *testing.T) {
	s := configuredSupervisor(t, ServiceProjectEntry)
	base := s.services[ServiceProjectEntry][0]
	api := base
	api.Component = "runtime-api"
	s.services[ServiceProjectEntry] = []ServiceConfig{base, api}
	started, err := s.Start("project-a", ServiceProjectEntry, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(started.ComponentPIDs) != 2 || started.ComponentPIDs["test-service"] == 0 || started.ComponentPIDs["runtime-api"] == 0 {
		t.Fatalf("all project_entry components must run: %+v", started)
	}
	_, _ = s.Stop("project-a", 2)
}
