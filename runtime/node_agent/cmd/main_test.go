package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNodeAgentExecutableProbe(t *testing.T) {
	if !strings.Contains(strings.Join(os.Args, " "), "node-agent-executable-probe") {
		return
	}
}

func TestNodeAgentExecutableStartsAfterWorkingDirectoryChanges(t *testing.T) {
	// 这里不依赖安装包目录：go test 自己的可执行文件足以模拟代理被 Supervisor 从 workload 目录拉起。
	executable := nodeAgentExecutable()
	if !filepath.IsAbs(executable) {
		t.Fatalf("node agent executable must be absolute: %q", executable)
	}
	if _, err := os.Stat(executable); err != nil {
		t.Fatalf("node agent executable must exist: %v", err)
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	workloadDir := t.TempDir()
	if err := os.Chdir(workloadDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	cmd := exec.Command(executable, "-test.run=^TestNodeAgentExecutableProbe$", "--", "node-agent-executable-probe")
	cmd.Dir = workloadDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("executable must remain runnable after workload directory switch: %v, output=%s", err, output)
	}
}

func TestGenerateDefaultConfigProducesValidOpsYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := generateDefaultConfig(path); err != nil {
		t.Fatalf("generate default config: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read default config: %v", err)
	}
	var parsed struct {
		Agent struct {
			Ops struct {
				Role           string `yaml:"role"`
				HeartbeatEvery string `yaml:"heartbeatEvery"`
			} `yaml:"ops"`
		} `yaml:"agent"`
	}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("default config must be valid YAML: %v", err)
	}
	if parsed.Agent.Ops.Role != "collector_linux" || parsed.Agent.Ops.HeartbeatEvery != "10s" {
		t.Fatalf("unexpected default ops config: %+v", parsed.Agent.Ops)
	}
}

func TestCheckSingleInstanceRecoversReusedCurrentPIDLock(t *testing.T) {
	lockFile := filepath.Join(t.TempDir(), "node_agent.lock")
	t.Setenv("NODE_AGENT_ENV", "")
	t.Setenv("NODE_AGENT_LOCK_FILE", lockFile)
	staleContent := fmt.Sprintf("%d\nstale", os.Getpid())
	if err := os.WriteFile(lockFile, []byte(staleContent), 0600); err != nil {
		t.Fatal(err)
	}

	if !checkSingleInstance(true) {
		t.Fatal("daemon must reclaim a lock left by a previous process with the reused current PID")
	}
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != fmt.Sprintf("%d", os.Getpid()) {
		t.Fatalf("lock was not recreated by current process: %q", content)
	}
	cleanupLockFile()
}

func TestCheckSingleInstanceRemovesDeadDaemonLock(t *testing.T) {
	lockFile := filepath.Join(t.TempDir(), "node_agent.lock")
	t.Setenv("NODE_AGENT_ENV", "")
	t.Setenv("NODE_AGENT_LOCK_FILE", lockFile)
	if err := os.WriteFile(lockFile, []byte("999999999"), 0600); err != nil {
		t.Fatal(err)
	}

	if !checkSingleInstance(true) {
		t.Fatal("daemon must remove a lock whose process no longer exists")
	}
	cleanupLockFile()
}
