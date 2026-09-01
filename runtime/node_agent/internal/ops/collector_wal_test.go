package ops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureCollectorWALUsesDeploymentScopedWritableDirectory(t *testing.T) {
	root := t.TempDir()
	deploymentID := "55555555-5555-4555-8555-555555555555"
	var ownerPath string
	path, err := ensureCollectorWALWithChown(root, deploymentID, func(value string, uid, gid int) error {
		ownerPath = value
		if uid != collectorContainerUID || gid != collectorContainerGID {
			t.Fatalf("collector owner=%d:%d", uid, gid)
		}
		return nil
	})
	if err != nil || path != filepath.Join(root, "deployments", deploymentID, "state", "collector-wal") || ownerPath != path {
		t.Fatalf("wal path=%q owner=%q err=%v", path, ownerPath, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o770 {
		t.Fatalf("wal permissions=%v err=%v", info.Mode(), err)
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
