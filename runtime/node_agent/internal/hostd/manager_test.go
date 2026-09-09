package hostd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type fakeRunner struct {
	calls [][]string
	state string
}

type foundationFailureRunner struct{}

type foundationMissingRunner struct{}

func (foundationMissingRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	if len(args) >= 2 && args[0] == "kubectl" && args[1] == "-n" && strings.Contains(strings.Join(args, " "), "get pods") {
		return []byte(`{"items":[]}`), nil
	}
	if strings.Contains(strings.Join(args, " "), "deployment/traefik") {
		return nil, errors.New("missing")
	}
	return nil, nil
}

func (foundationFailureRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	if len(args) >= 2 && args[0] == "kubectl" && args[1] == "-n" && strings.Contains(strings.Join(args, " "), "get pods") {
		return []byte(`{"items":[{"metadata":{"name":"redis-0","labels":{"induforge.io/service":"redis"}},"status":{"conditions":[],"containerStatuses":[{"state":{"waiting":{"reason":"ErrImageNeverPull","message":"missing offline image"}}}]}}]}`), nil
	}
	return nil, errors.New("not ready")
}

func TestFoundationObservationTreatsMissingOfflineImageAsFailed(t *testing.T) {
	manager, err := NewManager(ManagerConfig{AssetsDir: "/assets", BinaryPath: "/k3s", ConfigDir: "/config", SystemdDir: "/systemd", StateDir: "/state", Systemctl: "systemctl", Runner: foundationFailureRunner{}, RuntimeArch: "arm64"})
	if err != nil {
		t.Fatal(err)
	}
	state := manager.observeFoundation(context.Background(), FoundationState{EnvironmentID: "11111111-1111-4111-8111-111111111111"})
	if state.ObservedState != "failed" {
		t.Fatalf("observedState=%s message=%s", state.ObservedState, state.Message)
	}
	found := false
	for _, service := range state.Services {
		if service.Workload == "redis" && service.Status == "failed" && service.Message == "ErrImageNeverPull" {
			found = true
		}
	}
	if !found {
		t.Fatalf("redis failure missing: %+v", state.Services)
	}
}

func TestFoundationObservationTreatsLostHealthyWorkloadAsFailed(t *testing.T) {
	manager, err := NewManager(ManagerConfig{AssetsDir: "/assets", BinaryPath: "/k3s", ConfigDir: "/config", SystemdDir: "/systemd", StateDir: t.TempDir(), Systemctl: "systemctl", Runner: foundationMissingRunner{}, RuntimeArch: "arm64"})
	if err != nil {
		t.Fatal(err)
	}
	state := manager.observeFoundation(context.Background(), FoundationState{EnvironmentID: "11111111-1111-4111-8111-111111111111", ObservedState: "ready"})
	if state.ObservedState != "failed" {
		t.Fatalf("observedState=%s message=%s services=%+v", state.ObservedState, state.Message, state.Services)
	}
	for _, service := range state.Services {
		if service.Status != "failed" || service.Message != "基础服务实例未就绪" {
			t.Fatalf("missing workload was not failed: %+v", service)
		}
	}
}

func TestApplyFoundationRecreatesCredentialJobBeforeApplyingManifest(t *testing.T) {
	manager, cluster := preparedManager(t)
	if err := os.MkdirAll(filepath.Join(cluster.DataDir, "agent"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cluster.DataDir, "agent", "client-kubelet.crt"), []byte("registered"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := manager.saveState(ClusterState{
		SchemaVersion: "induforge.cluster-state.v1", Generation: cluster.Generation, ClusterID: cluster.ClusterID,
		NodeID: cluster.NodeID, Operation: OperationInitServer, K3sVersion: cluster.K3sVersion,
		DataDir: cluster.DataDir, ServiceName: cluster.ServiceName(), ObservedState: "ready",
	}); err != nil {
		t.Fatal(err)
	}
	plan := FoundationPlan{
		SchemaVersion: FoundationSchemaVersion, Generation: 1,
		EnvironmentID: "11111111-1111-4111-8111-111111111111", NodeID: cluster.NodeID,
		Assignments: map[string]string{
			"postgres": cluster.NodeID, "redis": cluster.NodeID, "emqx": cluster.NodeID,
			"nats": cluster.NodeID, "object": cluster.NodeID, "nginx": cluster.NodeID,
		},
	}
	if _, err := manager.ApplyFoundation(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	runner := manager.cfg.Runner.(*fakeRunner)
	deleteCall := []string{manager.cfg.BinaryPath, "kubectl", "-n", foundationNamespace(plan.EnvironmentID), "delete", "job/emqx-runtime-credential", "--wait=true"}
	applyCall := []string{manager.cfg.BinaryPath, "kubectl", "apply", "-f", manager.foundationManifestPath(plan.EnvironmentID)}
	deleteIndex, applyIndex := -1, -1
	for index, call := range runner.calls {
		if reflect.DeepEqual(call, deleteCall) {
			deleteIndex = index
		}
		if reflect.DeepEqual(call, applyCall) {
			applyIndex = index
		}
	}
	if deleteIndex < 0 || applyIndex < 0 || deleteIndex > applyIndex {
		t.Fatalf("credential job must be recreated before manifest apply: %#v", runner.calls)
	}
}

func TestDeleteFoundationDoesNotBlockNodeHeartbeat(t *testing.T) {
	manager, plan := preparedManager(t)
	if err := os.MkdirAll(filepath.Join(plan.DataDir, "agent"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plan.DataDir, "agent", "client-kubelet.crt"), []byte("registered"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Apply(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	runner := manager.cfg.Runner.(*fakeRunner)
	runner.calls = nil
	environmentID := "1289555a-c3f3-4536-8b66-16383e30824e"
	state, err := manager.DeleteFoundation(context.Background(), FoundationDeleteRequest{EnvironmentID: environmentID, NodeID: plan.NodeID})
	if err != nil {
		t.Fatal(err)
	}
	if state.ObservedState != "not-installed" || state.EnvironmentID != environmentID {
		t.Fatalf("unexpected delete state: %+v", state)
	}
	want := []string{manager.cfg.BinaryPath, "kubectl", "delete", "namespace", foundationNamespace(environmentID), "--ignore-not-found=true", "--wait=false"}
	if len(runner.calls) != 2 || !reflect.DeepEqual(runner.calls[1], want) {
		t.Fatalf("delete must be asynchronous, calls=%#v", runner.calls)
	}
}

func (runner *fakeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	runner.calls = append(runner.calls, append([]string{name}, args...))
	if strings.Contains(strings.Join(args, " "), "get job/emqx-runtime-credential") {
		return []byte(`{"metadata":{"name":"emqx-runtime-credential","labels":{"induforge.io/component":"credential-bootstrap"}}}`), nil
	}
	if len(args) == 6 && args[0] == "--mount=/proc/1/ns/mnt" && args[2] == "/bin/rm" && args[3] == "-rf" && args[4] == "--" {
		return nil, os.RemoveAll(args[5])
	}
	if len(args) > 0 && args[0] == "is-active" {
		return []byte(runner.state + "\n"), nil
	}
	return nil, nil
}

func TestManagerApplyIsVerifiedAndIdempotent(t *testing.T) {
	root := t.TempDir()
	assets := filepath.Join(root, "assets", "arm64")
	if err := os.MkdirAll(assets, 0755); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(assets, "k3s")
	images := filepath.Join(assets, "k3s-airgap-images-arm64.tar.zst")
	if err := os.WriteFile(binary, []byte("binary"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(images, []byte("images"), 0600); err != nil {
		t.Fatal(err)
	}
	checksumLine := checksum(t, binary) + "  k3s\n" + checksum(t, images) + "  k3s-airgap-images-arm64.tar.zst\n"
	if err := os.WriteFile(filepath.Join(assets, "SHA256SUMS"), []byte(checksumLine), 0600); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{state: "active"}
	manager, err := NewManager(ManagerConfig{
		AssetsDir: filepath.Join(root, "assets"), BinaryPath: filepath.Join(root, "bin", "k3s"),
		ConfigDir: filepath.Join(root, "config"), SystemdDir: filepath.Join(root, "systemd"),
		StateDir: filepath.Join(root, "state"), Systemctl: "systemctl", Runner: runner, RuntimeArch: "arm64",
	})
	if err != nil {
		t.Fatal(err)
	}
	plan := validPlan()
	plan.DataDir = filepath.Join(root, "data", "k3s")
	if err := os.MkdirAll(filepath.Join(plan.DataDir, "agent"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plan.DataDir, "agent", "client-kubelet.crt"), []byte("registered"), 0600); err != nil {
		t.Fatal(err)
	}
	state, err := manager.Apply(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if state.ObservedState != "ready" || state.NodeName != "if-5f0c6c9cd3f1" {
		t.Fatalf("unexpected state: %+v", state)
	}
	config, err := os.ReadFile(filepath.Join(root, "config", "config.yaml"))
	if err != nil || !strings.Contains(string(config), "cluster-init: true") {
		t.Fatalf("config=%s err=%v", config, err)
	}
	killall, err := os.ReadFile(filepath.Join(root, "config", "killall.sh"))
	if err != nil || !strings.Contains(string(killall), "K3S_DATA_DIR='"+plan.DataDir+"'") || !strings.Contains(string(killall), "-address /run/k3s/containerd/containerd.sock") || !strings.Contains(string(killall), `unmount_prefix "$K3S_DATA_DIR"`) {
		t.Fatalf("killall script does not target recorded data dir: %s err=%v", killall, err)
	}
	unit, err := os.ReadFile(filepath.Join(root, "systemd", plan.ServiceName()))
	if err != nil || !strings.Contains(string(unit), "ExecStopPost="+filepath.Join(root, "config", "killall.sh")) {
		t.Fatalf("systemd unit does not clean host resources: %s err=%v", unit, err)
	}
	firstCalls := append([][]string(nil), runner.calls...)
	if _, err := manager.Apply(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != len(firstCalls)+1 || !reflect.DeepEqual(runner.calls[len(runner.calls)-1], []string{"systemctl", "is-active", plan.ServiceName()}) {
		t.Fatalf("idempotent apply invoked mutation: %#v", runner.calls[len(firstCalls):])
	}
	for _, call := range firstCalls {
		for _, argument := range call {
			if argument == plan.Token {
				t.Fatal("cluster token must never be passed as a process argument")
			}
		}
	}
}

func TestManagerRejectsRoleOrEnvironmentReplacement(t *testing.T) {
	manager, plan := preparedManager(t)
	if _, err := manager.Apply(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	replacement := plan
	replacement.Generation = 2
	replacement.ClusterID = "74bc3380-5c23-4698-b1e1-18b9fd3c8057"
	if _, err := manager.Apply(context.Background(), replacement); err == nil {
		t.Fatal("expected environment replacement rejection")
	}
	replacement = plan
	replacement.Generation = 2
	replacement.Operation = OperationJoinAgent
	replacement.ServerURL = "https://172.16.125.129:6443"
	if _, err := manager.Apply(context.Background(), replacement); err == nil {
		t.Fatal("expected role replacement rejection")
	}
}

func TestManagerUninstallAlwaysCleansOnlyRecordedClusterDataDir(t *testing.T) {
	manager, plan := preparedManager(t)
	if _, err := manager.Apply(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	// 模拟从尚未生成 killall 脚本的早期节点包原地升级。
	if err := os.Remove(manager.killallPath()); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(plan.DataDir, "preserved")
	if err := os.WriteFile(marker, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Uninstall(context.Background(), UninstallRequest{ClusterID: "wrong", NodeID: plan.NodeID}); err == nil {
		t.Fatal("expected identity mismatch rejection")
	}
	state, err := manager.Uninstall(context.Background(), UninstallRequest{ClusterID: plan.ClusterID, NodeID: plan.NodeID})
	if err != nil || state.ObservedState != "not-installed" {
		t.Fatalf("state=%+v err=%v", state, err)
	}
	if state.ClusterID != plan.ClusterID || state.NodeID != plan.NodeID || state.Generation != 0 {
		t.Fatalf("uninstall tombstone lost center identity: %+v", state)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("uninstall must remove exact recorded cluster data dir, err=%v", err)
	}
	if info, err := os.Stat(plan.DataDir); err != nil || !info.IsDir() {
		t.Fatalf("uninstall must preserve writable cluster data root: info=%v err=%v", info, err)
	}
	runner := manager.cfg.Runner.(*fakeRunner)
	callCount := len(runner.calls)
	observed, err := manager.Status(context.Background())
	if err != nil || observed.ObservedState != "not-installed" || len(runner.calls) != callCount {
		t.Fatalf("tombstone status must not invoke systemctl: state=%+v err=%v calls=%d/%d", observed, err, len(runner.calls), callCount)
	}
	reinstalled, err := manager.Apply(context.Background(), plan)
	if err != nil || reinstalled.Generation != plan.Generation || reinstalled.ObservedState != "starting" {
		t.Fatalf("same identity must reinstall from tombstone: state=%+v err=%v", reinstalled, err)
	}
}

func preparedManager(t *testing.T) (*Manager, ClusterPlan) {
	t.Helper()
	root := t.TempDir()
	assets := filepath.Join(root, "assets", "arm64")
	if err := os.MkdirAll(assets, 0755); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(assets, "k3s")
	images := filepath.Join(assets, "k3s-airgap-images-arm64.tar.zst")
	_ = os.WriteFile(binary, []byte("binary"), 0755)
	_ = os.WriteFile(images, []byte("images"), 0600)
	checksums := checksum(t, binary) + "  k3s\n" + checksum(t, images) + "  k3s-airgap-images-arm64.tar.zst\n"
	_ = os.WriteFile(filepath.Join(assets, "SHA256SUMS"), []byte(checksums), 0600)
	manager, err := NewManager(ManagerConfig{
		AssetsDir: filepath.Join(root, "assets"), BinaryPath: filepath.Join(root, "bin", "k3s"), ConfigDir: filepath.Join(root, "config"),
		SystemdDir: filepath.Join(root, "systemd"), StateDir: filepath.Join(root, "state"), Systemctl: "systemctl",
		Runner: &fakeRunner{state: "active"}, RuntimeArch: "arm64",
	})
	if err != nil {
		t.Fatal(err)
	}
	plan := validPlan()
	plan.DataDir = filepath.Join(root, "data", "k3s")
	return manager, plan
}

func checksum(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
