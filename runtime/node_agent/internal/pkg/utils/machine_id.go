package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var (
	cachedMachineID string
	machineIDOnce   sync.Once
)

// GetMachineID 生成稳定的机器码 ID。
// 优先读取本地持久化文件；不存在时生成随机 ID 并写入文件。
// 若持久化失败，则回退到主机指纹哈希，保证接口仍可返回可用值。
func GetMachineID() string {
	machineIDOnce.Do(func() {
		idFilePath := resolveMachineIDFilePath()
		if persistedID, ok := readMachineIDFromFile(idFilePath); ok {
			cachedMachineID = persistedID
			return
		}

		newID, err := generateRandomMachineID()
		if err == nil {
			if writeMachineIDToFile(idFilePath, newID) == nil {
				cachedMachineID = newID
				return
			}
		}

		cachedMachineID = buildFingerprintMachineID()
	})

	return cachedMachineID
}

func resolveMachineIDFilePath() string {
	if filePath := strings.TrimSpace(os.Getenv("NODE_AGENT_MACHINE_ID_FILE")); filePath != "" {
		return filePath
	}
	return filepath.Join(".", "data", "node_id")
}

func readMachineIDFromFile(filePath string) (string, bool) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", false
	}
	value := strings.TrimSpace(string(content))
	if value == "" {
		return "", false
	}
	return value, true
}

func writeMachineIDToFile(filePath string, machineID string) error {
	if machineID == "" {
		return fmt.Errorf("machineID is empty")
	}
	if err := EnsureDir(filepath.Dir(filePath)); err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(machineID), 0600)
}

func generateRandomMachineID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "node-" + hex.EncodeToString(buf), nil
}

func buildFingerprintMachineID() string {
	hostname, _ := os.Hostname()
	hostname = strings.TrimSpace(strings.ToLower(hostname))

	macs := make([]string, 0)
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			hw := strings.TrimSpace(strings.ToLower(iface.HardwareAddr.String()))
			if hw != "" {
				macs = append(macs, hw)
			}
		}
	}
	sort.Strings(macs)

	fingerprint := hostname + "|" + strings.Join(macs, ",")
	if fingerprint == "|" {
		fingerprint = "node-agent-unknown"
	}

	sum := sha256.Sum256([]byte(fingerprint))
	return "node-" + hex.EncodeToString(sum[:8])
}
