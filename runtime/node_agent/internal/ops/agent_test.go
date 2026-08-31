package ops

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	pkgConfig "github.com/indu-forge/node_agent/internal/pkg/config"
)

const testLoopbackServerURL = "http://127.0.0.1"

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
