package ops

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/indu-forge/node_agent/internal/hostd"
	pkgConfig "github.com/indu-forge/node_agent/internal/pkg/config"
)

const testLoopbackServerURL = "http://127.0.0.1"

func TestClusterStatePayloadDoesNotExposeHostPaths(t *testing.T) {
	payload := clusterStatePayload(hostd.ClusterState{SchemaVersion: "induforge.cluster-state.v1", DataDir: "/secret/host/path", ServiceName: "internal.service", ObservedState: "ready"})
	if _, exists := payload["dataDir"]; exists {
		t.Fatal("center payload exposed host dataDir")
	}
	if _, exists := payload["serviceName"]; exists {
		t.Fatal("center payload exposed host serviceName")
	}
	if payload["observedState"] != "ready" {
		t.Fatalf("observedState=%v", payload["observedState"])
	}
}

func TestNewAgentValidatesCenterServerURL(t *testing.T) {
	valid := []string{
		"https://center.example",
		"https://center.example/",
		"http://localhost:17600",
		"http://127.42.0.1:17600",
		"http://[::1]:17600",
	}
	for _, value := range valid {
		t.Run("valid_"+value, func(t *testing.T) {
			if _, err := NewAgent(Config{Enabled: true, ServerURL: value}, testSupervisor(t)); err != nil {
				t.Fatalf("serverUrl %q rejected: %v", value, err)
			}
		})
	}
	invalid := []string{
		"",
		"center.example",
		"http://center.example",
		"ftp://center.example",
		"https://user:pass@center.example",
		"https://center.example/api/v1",
		"https://center.example?token=unsafe",
		"https://center.example#fragment",
	}
	for _, value := range invalid {
		t.Run("invalid_"+value, func(t *testing.T) {
			if _, err := NewAgent(Config{Enabled: true, ServerURL: value}, testSupervisor(t)); err == nil {
				t.Fatalf("serverUrl %q was accepted", value)
			}
		})
	}
}

func TestAgentRejectsOversizedAndTrailingCenterJSON(t *testing.T) {
	tests := []struct {
		name, response, want string
	}{
		{"oversized", strings.Repeat("x", maxCenterResponseBytes+1), "超过"},
		{"trailing", `{"code":0,"msg":"ok","data":null}{}`, "尾随 JSON"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = writer.Write([]byte(test.response))
			}))
			defer server.Close()
			agent, err := NewAgent(Config{Enabled: true, ServerURL: server.URL}, testSupervisor(t))
			if err != nil {
				t.Fatal(err)
			}
			if err := agent.request(context.Background(), http.MethodGet, "/test", "", nil, nil); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("response was accepted or wrong error: %v", err)
			}
		})
	}
}

func TestClaimPersistsOnlyNodeCredentialWithRestrictedPermission(t *testing.T) {
	t.Setenv("NODE_AGENT_MACHINE_ID_FILE", t.TempDir()+"/node_id")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ops/agent/enrollments/claim" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["code"] != "enroll-code" || body["hostname"] == nil || body["platform"] == nil || body["capabilities"] == nil {
			t.Fatalf("unexpected claim: %#v", body)
		}
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{"enrollment":{"id":"e1"},"node":{"id":"n1"},"agentToken":"secret-token","pendingApproval":true}}`))
	}))
	defer server.Close()
	dataDir := t.TempDir()
	configPath := dataDir + "/config.yaml"
	if err := os.WriteFile(configPath, []byte("agent:\n  ops:\n    enrollmentCode: enroll-code\n"), 0600); err != nil {
		t.Fatal(err)
	}
	agent, err := NewAgent(Config{Enabled: true, ServerURL: server.URL, EnrollmentCode: "enroll-code", DataDir: dataDir, ClearEnrollmentCode: func() error { return pkgConfig.ClearOpsEnrollmentCode(configPath) }}, testSupervisor(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := agent.claim(context.Background()); err != nil {
		t.Fatal(err)
	}
	identity := agent.currentIdentity()
	if identity.NodeID != "n1" || identity.AgentToken != "secret-token" {
		t.Fatalf("unexpected identity: %+v", identity)
	}
	info, err := os.Stat(agent.identityPath())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("identity permission=%o", info.Mode().Perm())
	}
	content, err := os.ReadFile(agent.identityPath())
	if err != nil {
		t.Fatal(err)
	}
	if string(content) == "enroll-code" {
		t.Fatal("enrollment code must not be persisted")
	}
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(configContent), "enroll-code") {
		t.Fatalf("enrollment code was not cleared: %s", configContent)
	}
}

func TestReconcileAppliesGenerationOnlyOnce(t *testing.T) {
	supervisor := configuredSupervisor(t, ServiceCollector)
	agent, err := NewAgent(Config{Enabled: true, ServerURL: testLoopbackServerURL, DataDir: t.TempDir()}, supervisor)
	if err != nil {
		t.Fatal(err)
	}
	agent.identity = Identity{NodeID: "node-a", AgentToken: "token"}
	command := AgentCommand{NodeID: "node-a", ServiceID: "collector-one", ServiceType: "collector", DesiredStatus: "running", Generation: 4}
	if err := agent.Reconcile(command); err != nil {
		t.Fatal(err)
	}
	first, _ := agent.supervisor.Status(command.ServiceID)
	if err := agent.Reconcile(command); err != nil {
		t.Fatal(err)
	}
	second, _ := agent.supervisor.Status(command.ServiceID)
	if first.PID != second.PID {
		t.Fatalf("same generation restarted process: %d -> %d", first.PID, second.PID)
	}
	if err := agent.Reconcile(AgentCommand{NodeID: "node-a", ServiceID: "collector-one", ServiceType: "collector", DesiredStatus: "stopped", Generation: 5}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		status, _ := agent.supervisor.Status("collector-one")
		if status.State == "stopped" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("workload was not stopped: %+v", status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestReconcileDeployStartsDesiredRunningWorkload(t *testing.T) {
	agent, err := NewAgent(Config{Enabled: true, ServerURL: testLoopbackServerURL, DataDir: t.TempDir()}, configuredSupervisor(t, ServiceCollector))
	if err != nil {
		t.Fatal(err)
	}
	agent.identity = Identity{NodeID: "node-a", AgentToken: "token"}
	command := AgentCommand{NodeID: "node-a", ServiceID: "collector-deploy", ServiceType: "collector", Operation: "deploy", DesiredStatus: "running", Generation: 1}
	if err := agent.Reconcile(command); err != nil {
		t.Fatal(err)
	}
	status, err := agent.supervisor.Status(command.ServiceID)
	if err != nil || status.State != "running" || status.Generation != 1 {
		t.Fatalf("deploy did not start workload: %+v err=%v", status, err)
	}
	agent.supervisor.Shutdown()
}

func TestReconcileRestoresRunningWorkloadAfterAgentRestart(t *testing.T) {
	dataDir := t.TempDir()
	first, err := NewAgent(Config{Enabled: true, ServerURL: testLoopbackServerURL, DataDir: dataDir}, configuredSupervisor(t, ServiceCollector))
	if err != nil {
		t.Fatal(err)
	}
	first.identity = Identity{NodeID: "node-a", AgentToken: "token"}
	command := AgentCommand{NodeID: "node-a", ServiceID: "collector-restart", ServiceType: "collector", DesiredStatus: "running", Generation: 9}
	if err := first.Reconcile(command); err != nil {
		t.Fatal(err)
	}
	first.supervisor.Shutdown()

	secondSupervisor := configuredSupervisor(t, ServiceCollector)
	second, err := NewAgent(Config{Enabled: true, ServerURL: testLoopbackServerURL, DataDir: dataDir}, secondSupervisor)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.loadApplied(); err != nil {
		t.Fatal(err)
	}
	second.identity = Identity{NodeID: "node-a", AgentToken: "token"}
	if err := second.Reconcile(command); err != nil {
		t.Fatal(err)
	}
	status, err := secondSupervisor.Status(command.ServiceID)
	if err != nil || status.State != "running" || status.Generation != command.Generation {
		t.Fatalf("restart did not restore workload: %+v err=%v", status, err)
	}
	secondSupervisor.Shutdown()
}

func TestReconcileRejectsWorkloadNotAssignedToThisNode(t *testing.T) {
	agent, err := NewAgent(Config{Enabled: true, ServerURL: testLoopbackServerURL, DataDir: t.TempDir()}, configuredSupervisor(t, ServiceCollector))
	if err != nil {
		t.Fatal(err)
	}
	agent.identity = Identity{NodeID: "node-a", AgentToken: "token"}
	err = agent.Reconcile(AgentCommand{NodeID: "node-b", ServiceID: "collector-other", ServiceType: "collector", DesiredStatus: "running", Generation: 1})
	if err == nil || !strings.Contains(err.Error(), "未显式分配") {
		t.Fatalf("expected explicit-node rejection, got %v", err)
	}
}

func TestReconcileReportsInstalledButUnconfiguredServiceAsFailed(t *testing.T) {
	supervisor, err := NewSupervisorWithConfig(SupervisorConfig{
		StateDir: t.TempDir(),
		LogDir:   t.TempDir(),
		Services: []ServiceConfig{{
			Group: ServiceProjectEntry, Component: "project-gateway", Installed: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := NewAgent(Config{Enabled: true, ServerURL: testLoopbackServerURL, DataDir: t.TempDir()}, supervisor)
	if err != nil {
		t.Fatal(err)
	}
	agent.identity = Identity{NodeID: "node-a", AgentToken: "token"}
	command := AgentCommand{
		NodeID: "node-a", ServiceID: "entry-service", ServiceType: "project_entry",
		DesiredStatus: "running", Generation: 1,
	}
	if err := agent.Reconcile(command); err == nil || !strings.Contains(err.Error(), "尚未完成本地 Release 配置") {
		t.Fatalf("expected fail-closed release error, got %v", err)
	}
	status, err := supervisor.Status(command.ServiceID)
	if err != nil || status.State != "failed" || status.Generation != command.Generation {
		t.Fatalf("unconfigured service must be observable as failed: %+v err=%v", status, err)
	}
}

func TestLoadAppliedTreatsJSONNullAsEmptyState(t *testing.T) {
	dataDir := t.TempDir()
	agent, err := NewAgent(Config{Enabled: true, ServerURL: testLoopbackServerURL, DataDir: dataDir}, testSupervisor(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agent.appliedPath(), []byte("null"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := agent.loadApplied(); err != nil {
		t.Fatal(err)
	}
	if err := agent.saveApplied("collector-null-state", 1); err != nil {
		t.Fatalf("null applied state should be repaired to an empty map: %v", err)
	}
}

// TestControlPlaneContractRoundTrip 固化领取、心跳与命令的跨端 JSON 契约。旧的
// hostNode、workloadId 或 serviceGroup 字段不能再次进入 NodeAgent 控制面。
func TestControlPlaneContractRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/ops/agent/enrollments/claim":
			var claim struct {
				Code         string   `json:"code"`
				Hostname     string   `json:"hostname"`
				Platform     string   `json:"platform"`
				Capabilities []string `json:"capabilities"`
			}
			if err := json.NewDecoder(r.Body).Decode(&claim); err != nil || claim.Code != "code" || claim.Hostname == "" || claim.Platform == "" || len(claim.Capabilities) != 1 || claim.Capabilities[0] != "collector" {
				t.Fatalf("claim payload=%+v err=%v", claim, err)
			}
			_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{"node":{"id":"node-a"},"agentToken":"token"}}`))
		case "/api/v1/ops/agent/nodes/node-a/heartbeat":
			var heartbeat struct {
				Services []struct {
					ServiceID string `json:"serviceId"`
					Endpoint  string `json:"endpoint"`
				} `json:"services"`
			}
			if err := json.NewDecoder(r.Body).Decode(&heartbeat); err != nil || len(heartbeat.Services) != 1 || heartbeat.Services[0].ServiceID != "service-a" || heartbeat.Services[0].Endpoint != "" {
				t.Fatalf("heartbeat payload=%+v err=%v", heartbeat, err)
			}
			_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{"node":{"id":"node-a"},"services":[]}}`))
		case "/api/v1/ops/agent/nodes/node-a/commands":
			_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{"commands":[{"nodeId":"node-a","serviceId":"service-a","serviceType":"collector","desiredStatus":"running","generation":1}]}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	supervisor := configuredSupervisor(t, ServiceCollector)
	agent, err := NewAgent(Config{Enabled: true, ServerURL: server.URL, EnrollmentCode: "code", DataDir: t.TempDir()}, supervisor)
	if err != nil {
		t.Fatal(err)
	}
	if err := agent.claim(context.Background()); err != nil {
		t.Fatal(err)
	}
	supervisor.RecordFailure("service-a", ServiceCollector, 1, os.ErrNotExist)
	if err := agent.Heartbeat(context.Background()); err != nil {
		t.Fatal(err)
	}
	commands, err := agent.Commands(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) != 1 || commands[0].NodeID != "node-a" || commands[0].ServiceID != "service-a" || commands[0].ServiceType != "collector" {
		t.Fatalf("commands=%+v", commands)
	}
}

const (
	formalNodeID       = "44444444-4444-4444-8444-444444444444"
	formalDeploymentID = "55555555-5555-4555-8555-555555555555"
	formalServiceID    = "66666666-6666-4666-8666-666666666666"
)

func TestFormalReleaseInstallsThenFailsClosedWithoutLauncher(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	fixture := signedReleaseArchive(t, privateKey, "release-key-a", nil, nil)
	dataDir := t.TempDir()
	releaseRoot := filepath.Join(dataDir, "deployments", formalDeploymentID, "release")
	t.Cleanup(func() { unsealReleaseTree(releaseRoot) })
	command := formalCommand()
	command.ArchiveSHA256 = sha256Digest(fixture.archive)
	command.ManifestSHA256 = fixture.manifestDigest
	command.ChecksumsSHA256 = fixture.checksumsDigest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer node-token" {
			t.Fatalf("agent token missing from %s", r.URL.Path)
		}
		switch r.URL.Path {
		case "/api/v1/ops/agent/nodes/" + formalNodeID + "/deployments/" + formalDeploymentID + "/binding":
			writeSuccess(t, w, formalBinding(command))
		case "/api/v1/ops/agent/nodes/" + formalNodeID + "/deployments/" + formalDeploymentID + "/release":
			w.Header().Set("Content-Type", "application/zstd")
			_, _ = w.Write(fixture.archive)
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	agent := formalAgent(t, server.URL, dataDir, publicKey)
	err = agent.reconcile(context.Background(), command)
	if err == nil || !strings.Contains(err.Error(), "Runtime Foundation 尚未 provisioned/healthy") {
		t.Fatalf("formal release must fail closed without launcher, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(releaseRoot, "current.json")); err != nil {
		t.Fatalf("verified Release was not installed: %v", err)
	}
	status, err := agent.supervisor.Status(formalServiceID)
	if err != nil || status.State != "failed" || status.PID != 0 || !strings.Contains(status.LastError, "Runtime Foundation 尚未 provisioned/healthy") {
		t.Fatalf("formal command must not start static supervisor: status=%+v err=%v", status, err)
	}
}

func TestDeleteDeploymentRemovesCollectorWALWhileStopKeepsIt(t *testing.T) {
	dataDir := t.TempDir()
	agent, err := NewAgent(Config{Enabled: true, ServerURL: testLoopbackServerURL, DataDir: dataDir}, configuredSupervisor(t, ServiceCollector))
	if err != nil {
		t.Fatal(err)
	}
	agent.identity = Identity{NodeID: formalNodeID, AgentToken: "node-token"}
	wal, err := collectorWALPath(dataDir, formalDeploymentID)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(wal, 0o700); err != nil {
		t.Fatal(err)
	}
	stop := formalCommand()
	stop.ServiceType, stop.Operation, stop.DesiredStatus = "collector", "stop", "stopped"
	if err = agent.Reconcile(stop); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(wal); err != nil {
		t.Fatalf("stop must retain WAL: %v", err)
	}
	deleteCommand := formalCommand()
	deleteCommand.ServiceType, deleteCommand.Operation, deleteCommand.DesiredStatus = "collector", "delete", "stopped"
	deleteCommand.Generation = 2
	if err = agent.Reconcile(deleteCommand); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(wal); !os.IsNotExist(err) {
		t.Fatalf("delete must remove WAL: %v", err)
	}
}

func TestFormalBindingRejectsUnknownOrMismatchedFields(t *testing.T) {
	command := formalCommand()
	for name, mutate := range map[string]func(map[string]any){
		"unknown object key": func(binding map[string]any) { binding["objectKey"] = "private/release.tar.zst" },
		"wrong node":         func(binding map[string]any) { binding["nodeId"] = testReleaseProjectID },
		"wrong revision":     func(binding map[string]any) { binding["revision"] = 2 },
		"wrong port": func(binding map[string]any) {
			binding["ports"].(map[string]any)["gatewayPublic"] = 18000
		},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				binding := formalBinding(command)
				mutate(binding)
				writeSuccess(t, w, binding)
			}))
			defer server.Close()
			agent := formalAgent(t, server.URL, t.TempDir(), make(ed25519.PublicKey, ed25519.PublicKeySize))
			if _, err := agent.fetchDeploymentBinding(context.Background(), command); err == nil {
				t.Fatal("invalid binding was accepted")
			}
		})
	}
}

func TestFormalReleaseRejectsRedirectAndNeverContactsRedirectTarget(t *testing.T) {
	redirected := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		redirected = true
		w.WriteHeader(http.StatusTeapot)
	}))
	defer target.Close()
	command := formalCommand()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/binding"):
			writeSuccess(t, w, formalBinding(command))
		case strings.HasSuffix(r.URL.Path, "/release"):
			http.Redirect(w, r, target.URL+"/bundle", http.StatusFound)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	agent := formalAgent(t, server.URL, t.TempDir(), make(ed25519.PublicKey, ed25519.PublicKeySize))
	agent.trustKeys[command.SigningKeyID] = make(ed25519.PublicKey, ed25519.PublicKeySize)
	if err := agent.installBoundRelease(context.Background(), command, ServiceProjectEntry); err == nil || !strings.Contains(err.Error(), "不允许重定向") {
		t.Fatalf("redirected bundle was accepted: %v", err)
	}
	if redirected {
		t.Fatal("agent followed Release redirect")
	}
}

func TestCommandsRejectUnknownFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{"commands":[],"downloadUrl":"https://outside.invalid/bundle"}}`))
	}))
	defer server.Close()
	agent, err := NewAgent(Config{Enabled: true, ServerURL: server.URL, DataDir: t.TempDir()}, testSupervisor(t))
	if err != nil {
		t.Fatal(err)
	}
	agent.identity = Identity{NodeID: formalNodeID, AgentToken: "node-token"}
	if _, err := agent.Commands(context.Background()); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("commands with external URL field were accepted: %v", err)
	}
}

func formalAgent(t *testing.T, serverURL, dataDir string, publicKey ed25519.PublicKey) *Agent {
	t.Helper()
	supervisor, err := NewSupervisorWithConfig(SupervisorConfig{StateDir: t.TempDir(), LogDir: t.TempDir(), Services: []ServiceConfig{
		{Group: ServiceProjectEntry, Component: "project-gateway", Installed: true},
		{Group: ServiceProjectEntry, Component: "runtime-api", Installed: true},
		{Group: ServiceDataRuntime, Component: "runtime-engine", Installed: true},
	}})
	if err != nil {
		t.Fatal(err)
	}
	agent, err := NewAgent(Config{
		Enabled:        true,
		ServerURL:      serverURL,
		DataDir:        dataDir,
		AgentVersion:   "1.0.0",
		RuntimeVersion: "1.0.0",
		TrustKeys: []TrustedSigningKey{{
			KeyID:     "release-key-a",
			PublicKey: base64.StdEncoding.EncodeToString(publicKey),
		}},
	}, supervisor)
	if err != nil {
		t.Fatal(err)
	}
	agent.identity = Identity{NodeID: formalNodeID, AgentToken: "node-token"}
	return agent
}

func TestEngineServiceGroupRoutesAllK3sEngines(t *testing.T) {
	tests := map[string]ServiceGroup{
		"base": ServiceProjectEntry, "compute": ServiceDataRuntime,
		"alarm": ServiceDataRuntime, "collector": ServiceCollector,
	}
	for engine, want := range tests {
		got, ok := engineServiceGroup(engine)
		if !ok || got != want {
			t.Fatalf("engineServiceGroup(%q)=(%q,%v), want (%q,true)", engine, got, ok, want)
		}
	}
	if _, ok := engineServiceGroup("project_entry"); ok {
		t.Fatal("旧原生服务组不能作为 K3s 引擎类型")
	}
}

func formalCommand() AgentCommand {
	return AgentCommand{
		NodeID: formalNodeID, DeploymentID: formalDeploymentID, ReleaseID: testReleaseID, ServiceID: formalServiceID,
		ServiceType: "project_entry", DesiredStatus: "running", Operation: "deploy", Generation: 1,
		Version: "2026.08.31-001", ArchiveSHA256: "sha256:" + strings.Repeat("a", 64),
		ManifestSHA256: "sha256:" + strings.Repeat("b", 64), ChecksumsSHA256: "sha256:" + strings.Repeat("c", 64),
		SigningKeyID: "release-key-a", BindingRevision: 1, ReplicasDesired: 1,
	}
}

func formalBinding(command AgentCommand) map[string]any {
	return map[string]any{
		"schemaVersion": deploymentBindingSchema, "bindingId": "33333333-3333-4333-8333-333333333333",
		"revision": command.BindingRevision, "nodeId": command.NodeID, "deploymentId": command.DeploymentID,
		"projectId":       testReleaseProjectID,
		"release":         map[string]any{"id": command.ReleaseID, "archiveSha256": command.ArchiveSHA256, "manifestSha256": command.ManifestSHA256, "checksumsSha256": command.ChecksumsSHA256, "signingKeyId": command.SigningKeyID},
		"enabledServices": []string{"project_entry", "data_runtime"},
		"ports":           map[string]any{"gatewayPublic": gatewayPublicPort, "runtimeApiLoopback": runtimeAPILoopbackPort, "engineLoopback": engineLoopbackPort},
		"secrets":         []any{}, "issuedAt": "2026-08-31T10:00:00Z",
	}
}

func writeSuccess(t *testing.T, writer http.ResponseWriter, data any) {
	t.Helper()
	if err := json.NewEncoder(writer).Encode(map[string]any{"code": 0, "msg": "ok", "data": data}); err != nil {
		t.Fatal(err)
	}
}

func TestFormalKubernetesStopDoesNotBlockNativeCollectorCommands(t *testing.T) {
	for _, engine := range []string{"base", "compute", "alarm"} {
		t.Run(engine, func(t *testing.T) {
			agent := formalAgent(t, "https://unused.invalid", t.TempDir(), make(ed25519.PublicKey, ed25519.PublicKeySize))
			command := formalCommand()
			command.ServiceType = engine
			command.Operation = "stop"
			command.DesiredStatus = "stopped"
			command.Generation = 2
			if err := agent.Reconcile(command); err != nil {
				t.Fatalf("K3s 停止命令阻塞节点协调: %v", err)
			}
			if status, _ := agent.supervisor.Status(command.ServiceID); status.Generation != 0 {
				t.Fatal("NodeAgent 修改了 K3s 服务状态")
			}
		})
	}
}
