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

func TestClaimPersistsOnlyNodeCredentialWithRestrictedPermission(t *testing.T) {
	t.Setenv("NODE_AGENT_MACHINE_ID_FILE", t.TempDir()+"/node_id")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ops/agent/enrollments/claim" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["code"] != "enroll-code" || body["host"] == nil || body["agent"] == nil {
			t.Fatalf("unexpected claim: %#v", body)
		}
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{"enrollment":{"id":"e1"},"hostNode":{"id":"h1"},"agentToken":"secret-token","pendingApproval":true}}`))
	}))
	defer server.Close()
	dataDir := t.TempDir()
	configPath := dataDir + "/config.yaml"
	if err := os.WriteFile(configPath, []byte("agent:\n  ops:\n    enrollmentCode: enroll-code\n"), 0600); err != nil {
		t.Fatal(err)
	}
	agent, err := NewAgent(Config{Enabled: true, ServerURL: server.URL, EnrollmentCode: "enroll-code", Role: "collector_linux", DataDir: dataDir, ClearEnrollmentCode: func() error { return pkgConfig.ClearOpsEnrollmentCode(configPath) }}, testSupervisor(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := agent.claim(context.Background()); err != nil {
		t.Fatal(err)
	}
	identity := agent.currentIdentity()
	if identity.HostNodeID != "h1" || identity.AgentToken != "secret-token" {
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
	agent, err := NewAgent(Config{Enabled: true, Role: "runtime_linux", DemoRuntime: true, DataDir: t.TempDir()}, testSupervisor(t))
	if err != nil {
		t.Fatal(err)
	}
	command := DesiredWorkload{WorkloadID: "compute-one", Role: "compute", DesiredStatus: "running", Generation: 4}
	if err := agent.Reconcile(command); err != nil {
		t.Fatal(err)
	}
	first, _ := agent.supervisor.Status("compute-one")
	if err := agent.Reconcile(command); err != nil {
		t.Fatal(err)
	}
	second, _ := agent.supervisor.Status("compute-one")
	if first.PID != second.PID {
		t.Fatalf("same generation restarted process: %d -> %d", first.PID, second.PID)
	}
	if err := agent.Reconcile(DesiredWorkload{WorkloadID: "compute-one", Role: "compute", DesiredStatus: "stopped", Generation: 5}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		status, _ := agent.supervisor.Status("compute-one")
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
	agent, err := NewAgent(Config{Enabled: true, Role: "collector_linux", DataDir: t.TempDir()}, testSupervisor(t))
	if err != nil {
		t.Fatal(err)
	}
	command := DesiredWorkload{WorkloadID: "collector-deploy", Role: "collector", Operation: "deploy", DesiredStatus: "running", Generation: 1}
	if err := agent.Reconcile(command); err != nil {
		t.Fatal(err)
	}
	status, err := agent.supervisor.Status(command.WorkloadID)
	if err != nil || status.State != "running" || status.Generation != 1 {
		t.Fatalf("deploy did not start workload: %+v err=%v", status, err)
	}
	agent.supervisor.Shutdown()
}

func TestReconcileRestoresRunningWorkloadAfterAgentRestart(t *testing.T) {
	dataDir := t.TempDir()
	first, err := NewAgent(Config{Enabled: true, Role: "collector_linux", DataDir: dataDir}, testSupervisor(t))
	if err != nil {
		t.Fatal(err)
	}
	command := DesiredWorkload{WorkloadID: "collector-restart", Role: "collector", DesiredStatus: "running", Generation: 9}
	if err := first.Reconcile(command); err != nil {
		t.Fatal(err)
	}
	first.supervisor.Shutdown()

	secondSupervisor := testSupervisor(t)
	second, err := NewAgent(Config{Enabled: true, Role: "collector_linux", DataDir: dataDir}, secondSupervisor)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.loadApplied(); err != nil {
		t.Fatal(err)
	}
	if err := second.Reconcile(command); err != nil {
		t.Fatal(err)
	}
	status, err := secondSupervisor.Status(command.WorkloadID)
	if err != nil || status.State != "running" || status.Generation != command.Generation {
		t.Fatalf("restart did not restore workload: %+v err=%v", status, err)
	}
	secondSupervisor.Shutdown()
}

func TestLoadAppliedTreatsJSONNullAsEmptyState(t *testing.T) {
	dataDir := t.TempDir()
	agent, err := NewAgent(Config{Enabled: true, Role: "collector_linux", DataDir: dataDir}, testSupervisor(t))
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
