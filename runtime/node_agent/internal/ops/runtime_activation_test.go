package ops

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDeploymentPortAvailabilityDetectsHostProcess(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	if port < 1025 || port > 65533 {
		t.Skip("ephemeral port does not leave room for runtime offsets")
	}
	if err := validateDeploymentPortAvailability(RuntimeActivationPorts{GatewayPublic: port - 1, RuntimeAPILoopback: port, EngineLoopback: port + 1}); err == nil {
		t.Fatal("occupied customer port must be rejected")
	}
}

func TestDeriveRuntimeActivationUsesExactCenterBindingBytesAndFixedPlan(t *testing.T) {
	installed, bindingRaw, foundation := installedActivationFixture(t, false)
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	derived, err := DeriveRuntimeActivation(RuntimeActivationDerivationInput{
		BindingJSON: bindingRaw, InstalledRelease: installed, SiteID: "77777777-7777-4777-8777-777777777777", AccountID: "88888888-8888-4888-8888-888888888888",
		Foundation: foundation.Foundation, Secrets: foundation.Secrets, ExpiresAt: "2026-09-01T10:00:00Z", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if derived.BindingDigestMode != bindingDigestRawCenterJSON || derived.Activation.Binding.BindingSHA256 != digestBytes(bindingRaw) {
		t.Fatalf("binding digest did not retain exact Center raw bytes: %+v", derived)
	}
	// JSON 语义相同但原始字节不同，必须得到不同摘要；当前实现没有假称 JCS。
	withWhitespace := append([]byte(" \n"), bindingRaw...)
	derivedWhitespace, err := DeriveRuntimeActivation(RuntimeActivationDerivationInput{
		BindingJSON: withWhitespace, InstalledRelease: installed, SiteID: derived.Activation.SiteID, AccountID: derived.Activation.AccountID,
		Foundation: foundation.Foundation, Secrets: foundation.Secrets, ExpiresAt: "2026-09-01T10:00:00Z", Now: now,
	})
	if err != nil || derivedWhitespace.Activation.Binding.BindingSHA256 == derived.Activation.Binding.BindingSHA256 {
		t.Fatalf("raw byte digest boundary was not enforced: result=%+v err=%v", derivedWhitespace, err)
	}

	dataRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(installed.ReleaseDir)))))
	planner, err := NewFixedLaunchPlanner(filepath.Join(t.TempDir(), "node-agent"), dataRoot, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	materializePlanCapabilities(t, planner.installRoot, "amd64", false)
	plan, err := planner.Plan(derived, installed)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Components) != 3 || fixedComponentByName(plan.Components, "demo-workload") != nil {
		t.Fatalf("launch plan must contain only fixed production components: %+v", plan.Components)
	}
	for _, component := range plan.Components {
		if !strings.HasPrefix(component.Executable, filepath.Join(planner.installRoot, "capabilities", "amd64")+string(filepath.Separator)) || strings.Contains(strings.Join(component.Args, " "), "$(") {
			t.Fatalf("launch plan admitted non-local executable or shell syntax: %+v", component)
		}
	}
}

func TestFixedLaunchPlanRejectsCenterPathAndCommandInjection(t *testing.T) {
	installed, bindingRaw, foundation := installedActivationFixture(t, false)
	var binding map[string]any
	if err := json.Unmarshal(bindingRaw, &binding); err != nil {
		t.Fatal(err)
	}
	binding["command"] = "/bin/sh -c evil"
	binding["releaseRoot"] = "/tmp/center-controlled"
	malicious, err := json.Marshal(binding)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DeriveRuntimeActivation(RuntimeActivationDerivationInput{
		BindingJSON: malicious, InstalledRelease: installed, SiteID: "77777777-7777-4777-8777-777777777777", AccountID: "88888888-8888-4888-8888-888888888888",
		Foundation: foundation.Foundation, Secrets: foundation.Secrets, ExpiresAt: "2026-09-01T10:00:00Z", Now: time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC),
	}); err == nil {
		t.Fatal("Binding command field must be rejected")
	}
	derived, err := DeriveRuntimeActivation(RuntimeActivationDerivationInput{
		BindingJSON: bindingRaw, InstalledRelease: installed, SiteID: "77777777-7777-4777-8777-777777777777", AccountID: "88888888-8888-8888-8888-888888888888",
		Foundation: foundation.Foundation, Secrets: foundation.Secrets, ExpiresAt: "2026-09-01T10:00:00Z", Now: time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("invalid local account id must be rejected rather than enter arguments")
	}
	_ = derived

	valid, err := DeriveRuntimeActivation(RuntimeActivationDerivationInput{
		BindingJSON: bindingRaw, InstalledRelease: installed, SiteID: "77777777-7777-4777-8777-777777777777", AccountID: "88888888-8888-4888-8888-888888888888",
		Foundation: foundation.Foundation, Secrets: foundation.Secrets, ExpiresAt: "2026-09-01T10:00:00Z", Now: time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	dataRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(installed.ReleaseDir)))))
	planner, err := NewFixedLaunchPlanner(t.TempDir(), dataRoot, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	escaped := installed
	escaped.ReleaseDir = t.TempDir()
	if _, err := planner.Plan(valid, escaped); err == nil {
		t.Fatal("deployment fixed plan must reject a release path outside its local directory")
	}
}

func TestFixedLaunchGateRejectsDeclaredFoundationAndValidatesSecrets(t *testing.T) {
	installed, bindingRaw, foundation := installedActivationFixture(t, false)
	secretsDir := t.TempDir()
	for index := range foundation.Secrets {
		payload := []byte("secret-" + foundation.Secrets[index].Name)
		foundation.Secrets[index].SHA256 = digestBytes(payload)
		if err := os.WriteFile(filepath.Join(secretsDir, foundation.Secrets[index].Name), payload, 0600); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	derived, err := DeriveRuntimeActivation(RuntimeActivationDerivationInput{
		BindingJSON: bindingRaw, InstalledRelease: installed, SiteID: "77777777-7777-4777-8777-777777777777", AccountID: "88888888-8888-4888-8888-888888888888",
		Foundation: foundation.Foundation, Secrets: foundation.Secrets, ExpiresAt: "2026-09-01T10:00:00Z", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	declared := newRuntimeFoundationState(derived.Activation, "", now)
	resolver := testSecretResolver{root: secretsDir}
	if err := ValidateFixedLaunchGate(derived.Activation, declared, resolver); err == nil || !strings.Contains(err.Error(), "provisioned/healthy") {
		t.Fatalf("declared Foundation must fail closed: %v", err)
	}
	healthy := declared
	healthy.Status = "healthy"
	if err := ValidateFixedLaunchGate(derived.Activation, healthy, resolver); err != nil {
		t.Fatalf("healthy Foundation with private verified secrets was rejected: %v", err)
	}
	if err := os.Chmod(filepath.Join(secretsDir, foundation.Secrets[0].Name), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFixedLaunchGate(derived.Activation, healthy, resolver); err == nil {
		t.Fatal("group-readable Secret must be rejected")
	}
}

func TestDeriveRuntimeActivationIncludesCollectorOnlyWhenBound(t *testing.T) {
	installed, bindingRaw, foundation := installedActivationFixture(t, true)
	foundation.Secrets = append(foundation.Secrets, RuntimeSecretReference{Name: "collector-nats", Ref: "secret-collector-nats", SchemaVersion: "runtime-nats-credentials.v1", SHA256: "sha256:" + strings.Repeat("9", 64), Revision: 1, ExpiresAt: "2026-09-01T10:00:00Z"})
	derived, err := DeriveRuntimeActivation(RuntimeActivationDerivationInput{
		BindingJSON: bindingRaw, InstalledRelease: installed, SiteID: "77777777-7777-4777-8777-777777777777", AccountID: "88888888-8888-4888-8888-888888888888",
		Foundation: foundation.Foundation, Secrets: foundation.Secrets, ExpiresAt: "2026-09-01T10:00:00Z", Now: time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC),
	})
	if err != nil || derived.Activation.Artifacts.Collector == nil || activationComponentByName(derived.Activation.Components, "collector") == nil {
		t.Fatalf("bound collector should be derived with its artifact: activation=%+v err=%v", derived.Activation, err)
	}
	dataRoot := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(installed.ReleaseDir)))))
	planner, err := NewFixedLaunchPlanner(t.TempDir(), dataRoot, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	materializePlanCapabilities(t, planner.installRoot, "amd64", true)
	plan, err := planner.Plan(derived, installed)
	if err != nil || fixedComponentByName(plan.Components, "collector") == nil {
		t.Fatalf("collector fixed plan was not emitted: plan=%+v err=%v", plan, err)
	}
}

func installedActivationFixture(t *testing.T, collector bool) (InstalledRelease, []byte, RuntimeActivationFoundationInput) {
	t.Helper()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	if collector {
		files[optionalCollectorPayload] = []byte("collector")
	}
	archive := signedReleaseArchive(t, private, "release-key-a", files, nil)
	dataRoot := t.TempDir()
	store, err := NewReleaseStore(filepath.Join(dataRoot, "deployments", formalDeploymentID, "release"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { unsealReleaseTree(store.root) })
	installer, err := NewReleaseInstaller(store, DefaultReleaseInstallLimits())
	if err != nil {
		t.Fatal(err)
	}
	input := archive.installInput(public, "release-key-a")
	if collector {
		input.NodeCapabilities = append(input.NodeCapabilities, "collector")
		input.BoundServices = append(input.BoundServices, "collector")
	}
	installed, err := installer.Install(input)
	if err != nil {
		t.Fatal(err)
	}
	command := formalCommand()
	command.ArchiveSHA256, command.ManifestSHA256, command.ChecksumsSHA256 = installed.ReleaseDigest, archive.manifestDigest, archive.checksumsDigest
	binding := formalBinding(command)
	if collector {
		binding["enabledServices"] = []string{"project_entry", "data_runtime", "collector"}
		binding["ports"] = map[string]any{"gatewayPublic": gatewayPublicPort, "runtimeApiLoopback": runtimeAPILoopbackPort, "engineLoopback": engineLoopbackPort, "collectorHealthLoopback": collectorHealthPort}
	}
	raw, err := json.Marshal(binding)
	if err != nil {
		t.Fatal(err)
	}
	return installed, raw, runtimeFoundationFixture(t)
}

type testSecretResolver struct{ root string }

func (r testSecretResolver) Resolve(reference RuntimeSecretReference) (ResolvedRuntimeSecret, error) {
	return ResolvedRuntimeSecret{Name: reference.Name, Ref: reference.Ref, Path: filepath.Join(r.root, reference.Name)}, nil
}

func fixedComponentByName(components []FixedLaunchComponent, name string) *FixedLaunchComponent {
	for index := range components {
		if components[index].Name == name {
			return &components[index]
		}
	}
	return nil
}

func activationComponentByName(components []RuntimeActivationComponent, name string) *RuntimeActivationComponent {
	for index := range components {
		if components[index].Name == name {
			return &components[index]
		}
	}
	return nil
}

func materializePlanCapabilities(t *testing.T, installRoot, arch string, collector bool) {
	t.Helper()
	paths := []string{
		filepath.Join(installRoot, "capabilities", arch, "project-gateway"),
		filepath.Join(installRoot, "capabilities", arch, "runtime-api"),
		filepath.Join(installRoot, "capabilities", arch, "runtime-engine"),
	}
	if collector {
		paths = append(paths, filepath.Join(installRoot, "capabilities", arch, "collector", "industrial_collector"))
	}
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("binary"), 0700); err != nil {
			t.Fatal(err)
		}
	}
}
