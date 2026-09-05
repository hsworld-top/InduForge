package ops

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func nativeBundleFixture() (AgentCommand, nativeCollectorBundle, string) {
	command := AgentCommand{DeploymentID: "11111111-1111-4111-8111-111111111111", BindingRevision: 2}
	digest := "sha256:" + strings.Repeat("a", 64)
	binding, _ := json.Marshal(map[string]any{"deploymentId": command.DeploymentID, "bindingRevision": 2, "artifact": map[string]string{"artifactDigest": digest}})
	return command, nativeCollectorBundle{Binding: binding, Index: json.RawMessage(`{"schemaVersion":"collector-runtime-index.v1","resources":{},"secrets":{"secret://nats":"secrets/nats.json"}}`), SecretFiles: map[string][]byte{"nats.json": []byte(`{"token":"test-only"}`)}}, digest
}

func TestNativeCollectorBundleRejectsMismatchedArtifactAndSecretTraversal(t *testing.T) {
	command, bundle, digest := nativeBundleFixture()
	if err := validateNativeCollectorFiles(bundle, command, digest); err != nil {
		t.Fatal(err)
	}
	if err := validateNativeCollectorFiles(bundle, command, "sha256:"+strings.Repeat("b", 64)); err == nil {
		t.Fatal("接受了非冻结制品")
	}
	bundle.Index = json.RawMessage(`{"schemaVersion":"collector-runtime-index.v1","resources":{},"secrets":{"secret://nats":"secrets/../outside"}}`)
	if err := validateNativeCollectorFiles(bundle, command, digest); err == nil {
		t.Fatal("接受了越界凭据引用")
	}
}

func TestNativeCollectorFilesArePrivateImmutableAndWALIndependent(t *testing.T) {
	_, bundle, _ := nativeBundleFixture()
	root := t.TempDir()
	target := filepath.Join(root, "collector-config", "digest")
	if err := writeNativeCollectorFiles(target, []byte(`{}`), bundle); err != nil {
		t.Fatal(err)
	}
	if err := writeNativeCollectorFiles(target, []byte(`{}`), bundle); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(target, "secrets", "nats.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Fatal("凭据可被其他账户读取")
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"token":"tampered"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeNativeCollectorFiles(target, []byte(`{}`), bundle); err == nil {
		t.Fatal("重用被修改的采集配置")
	}
	wal, err := collectorWALPath(root, "11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(wal, "release") || strings.Contains(wal, "collector-config") {
		t.Fatal("WAL 随候选版本切换")
	}
}

func TestNativeCollectorCapabilitiesUsePackagedPlatformPaths(t *testing.T) {
	for _, platform := range []string{"linux", "windows"} {
		root, file, err := nativeCollectorCapability("/installed", platform, "amd64")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(root, "capabilities") || !strings.HasPrefix(file, "industrial_collector") {
			t.Fatal("未使用安装包采集能力")
		}
	}
	if _, _, err := nativeCollectorCapability("/installed", "darwin", "arm64"); err == nil {
		t.Fatal("接受了未发布的平台能力")
	}
}

func TestNativeCollectorRejectsInvalidOperationBeforeStopping(t *testing.T) {
	s := NewSupervisor("", t.TempDir(), t.TempDir())
	a := &Agent{supervisor: s, cfg: Config{DataDir: t.TempDir()}, applied: map[string]int64{}}
	for _, command := range []AgentCommand{
		{ServiceID: "collector-test", Generation: 1, Operation: "unknown", DesiredStatus: "stopped"},
		{ServiceID: "collector-test", Generation: 1, Operation: "stop", DesiredStatus: "running"},
		{ServiceID: "collector-test", Generation: 1, Operation: "deploy", DesiredStatus: "invalid"},
	} {
		if err := a.reconcileNativeCollector(context.Background(), command); err == nil {
			t.Errorf("接受了无效命令: %+v", command)
		}
		if status, _ := s.Status(command.ServiceID); status.Generation != 0 {
			t.Fatal("无效命令修改了采集进程状态")
		}
	}
}

func TestNativeCollectorStopAndDeleteRespectObservedGeneration(t *testing.T) {
	for _, test := range []struct {
		name           string
		operation      string
		generation     int64
		alreadyStopped bool
		wantState      string
		wantGeneration int64
		wantApplied    int64
		wantWAL        bool
	}{
		{"stale stop", "stop", 3, false, "running", 3, 1, true},
		{"stale delete", "delete", 3, false, "running", 3, 1, true},
		{"stale delete of stopped workload", "delete", 3, true, "stopped", 3, 1, true},
		{"stop preserves WAL", "stop", 1, false, "stopped", 2, 2, true},
		{"delete removes WAL", "delete", 1, false, "stopped", 2, 2, false},
		{"retry delete after stop", "delete", 2, true, "stopped", 2, 2, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := configuredSupervisor(t, ServiceCollector)
			t.Cleanup(s.Shutdown)
			const serviceID = "collector-test"
			if _, err := s.Start(serviceID, ServiceCollector, test.generation); err != nil {
				t.Fatal(err)
			}
			if test.alreadyStopped {
				if _, err := s.Stop(serviceID, test.generation); err != nil {
					t.Fatal(err)
				}
			}
			root := t.TempDir()
			const deploymentID = "11111111-1111-4111-8111-111111111111"
			wal, err := collectorWALPath(root, deploymentID)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(wal, 0700); err != nil {
				t.Fatal(err)
			}
			event := filepath.Join(wal, "unacked-event")
			if err := os.WriteFile(event, []byte("test-only"), 0600); err != nil {
				t.Fatal(err)
			}
			// 模拟进程状态已经持久化，而 Agent 的 applied 代次仍落后。
			a := &Agent{supervisor: s, cfg: Config{DataDir: root}, applied: map[string]int64{serviceID: 1}}
			command := AgentCommand{ServiceID: serviceID, DeploymentID: deploymentID, Generation: 2, Operation: test.operation, DesiredStatus: "stopped"}
			if err := a.reconcileNativeCollector(context.Background(), command); err != nil {
				t.Fatal(err)
			}
			status, _ := s.Status(serviceID)
			if status.State != test.wantState || status.Generation != test.wantGeneration {
				t.Errorf("停止结果不符合实际代次: %+v", status)
			}
			if applied := a.appliedGeneration(serviceID); applied != test.wantApplied {
				t.Errorf("applied 代次 = %d，期望 %d", applied, test.wantApplied)
			}
			_, err = os.Stat(event)
			if test.wantWAL && err != nil {
				t.Errorf("不应删除未确认 WAL: %v", err)
			} else if !test.wantWAL && !os.IsNotExist(err) {
				t.Errorf("删除命令未清理 WAL: %v", err)
			}
		})
	}
}

func TestNativeCollectorExtractionCacheUsesPrivateDeploymentState(t *testing.T) {
	root := t.TempDir()
	env, err := nativeCollectorEnvironment(root)
	if err != nil {
		t.Fatal(err)
	}
	cache := env["DOTNET_BUNDLE_EXTRACT_BASE_DIR"]
	info, err := os.Stat(cache)
	if err != nil || info.Mode().Perm() != 0700 || !strings.HasPrefix(cache, filepath.Join(root, "state")) {
		t.Fatal("未使用部署私有可写缓存")
	}
	if err = os.Remove(cache); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(t.TempDir(), cache); err != nil {
		t.Fatal(err)
	}
	if _, err = nativeCollectorEnvironment(root); err == nil {
		t.Fatal("接受了链接缓存目录")
	}
}
