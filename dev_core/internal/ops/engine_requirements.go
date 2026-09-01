package ops

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// deploymentEngineRequirements 只从由 Builder 固化的 Release manifest 读取可选引擎。
// 基础引擎始终存在；调用方不能通过请求体开启未被工程内容声明的计算、报警或采集。
func deploymentEngineRequirements(manifest []byte) ([]string, error) {
	result := []string{ServiceBase}
	if len(manifest) == 0 {
		return result, nil
	}
	var document struct {
		Capabilities []string `json:"capabilities"`
		Engines      []string `json:"engines"`
	}
	if err := json.Unmarshal(manifest, &document); err != nil {
		return nil, fmt.Errorf("Release 清单无效: %w", err)
	}
	declared := make(map[string]struct{}, len(document.Capabilities)+len(document.Engines))
	for _, value := range append(document.Capabilities, document.Engines...) {
		switch strings.TrimSpace(strings.ToLower(value)) {
		case ServiceCompute, ServiceAlarm, ServiceCollector:
			declared[strings.TrimSpace(strings.ToLower(value))] = struct{}{}
		}
	}
	for _, engine := range []string{ServiceCompute, ServiceAlarm, ServiceCollector} {
		if _, ok := declared[engine]; ok {
			result = append(result, engine)
		}
	}
	return result, nil
}

func validateEnginePlacements(required []string, placements map[string]string) error {
	if len(placements) == 0 {
		return fmt.Errorf("基础引擎必须选择部署节点")
	}
	requiredSet := make(map[string]struct{}, len(required))
	for _, engine := range required {
		requiredSet[engine] = struct{}{}
		if strings.TrimSpace(placements[engine]) == "" {
			return fmt.Errorf("%s 引擎必须选择部署节点", engine)
		}
	}
	for engine, nodeID := range placements {
		if strings.TrimSpace(nodeID) == "" {
			continue
		}
		if _, ok := requiredSet[engine]; !ok {
			return fmt.Errorf("工程未使用 %s 引擎，不能创建该工作负载", engine)
		}
	}
	return nil
}

func sortedEngines(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}
