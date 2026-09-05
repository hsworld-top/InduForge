package ops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNativeCollectorWALRejectsSymlinkAncestors(t *testing.T) {
	const deploymentID = "55555555-5555-4555-8555-555555555555"
	for _, suffix := range []string{"", "state", filepath.Join("state", "collector-wal")} {
		t.Run(suffix, func(t *testing.T) {
			root, outside := t.TempDir(), t.TempDir()
			link := filepath.Join(root, "deployments", deploymentID, suffix)
			if err := os.MkdirAll(filepath.Dir(link), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, link); err != nil {
				t.Skipf("当前环境无法创建软链: %v", err)
			}
			if _, err := prepareNativeCollectorWAL(root, deploymentID); err == nil {
				t.Error("接受了软链采集状态目录")
			}
			entries, err := os.ReadDir(outside)
			if err != nil || len(entries) != 0 {
				t.Fatalf("在部署目录外创建了采集文件: entries=%v err=%v", entries, err)
			}
		})
	}
}

func TestNativeCollectorWALIsPrivateAndPreservesExistingData(t *testing.T) {
	root := t.TempDir()
	const deploymentID = "55555555-5555-4555-8555-555555555555"
	path, err := prepareNativeCollectorWAL(root, deploymentID)
	if err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(path, "pending.wal")
	if err := os.WriteFile(data, []byte("pending"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareNativeCollectorWAL(root, deploymentID); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0700 {
		t.Fatalf("WAL 应仅服务账户可访问: %v", info.Mode())
	}
	content, err := os.ReadFile(data)
	if err != nil || string(content) != "pending" {
		t.Fatalf("重复准备覆盖了未确认数据: %q %v", content, err)
	}
}

func TestCollectorWALRejectsEscapingDeploymentAndOnlyDeleteRemovesIt(t *testing.T) {
	root := t.TempDir()
	if _, err := collectorWALPath(root, "../../outside"); err == nil {
		t.Fatal("escaping deployment id must be rejected")
	}
	deploymentID := "55555555-5555-4555-8555-555555555555"
	path := filepath.Join(root, "deployments", deploymentID, "state", "collector-wal")
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
	// stop 不调用 removeCollectorWAL，因此 WAL 在正常 stop/restart 间保留。
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stop semantic must retain WAL: %v", err)
	}
	if err := removeCollectorWAL(root, deploymentID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("deployment delete must clean WAL: %v", err)
	}
}
