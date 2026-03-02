package utils

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func resetMachineIDCacheForTest() {
	cachedMachineID = ""
	machineIDOnce = sync.Once{}
}

func TestGetMachineID_Persisted(t *testing.T) {
	tempDir := t.TempDir()
	idPath := filepath.Join(tempDir, "node_id")
	_ = os.Setenv("NODE_AGENT_MACHINE_ID_FILE", idPath)
	defer os.Unsetenv("NODE_AGENT_MACHINE_ID_FILE")

	resetMachineIDCacheForTest()
	first := GetMachineID()
	if first == "" {
		t.Fatal("首次获取 machineID 不能为空")
	}

	resetMachineIDCacheForTest()
	second := GetMachineID()
	if second == "" {
		t.Fatal("二次获取 machineID 不能为空")
	}
	if first != second {
		t.Fatalf("期望 machineID 持久化一致，first=%s second=%s", first, second)
	}
}
