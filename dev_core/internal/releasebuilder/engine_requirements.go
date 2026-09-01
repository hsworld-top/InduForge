package releasebuilder

import "strings"

// DeriveEngineRequirements 只读取 runtime-project-artifact.v1 的正式字段，未知
// 展示/扩展字段绝不能影响调度：computeUnits、alarmItems、dataPoints[].sourceType。
func DeriveEngineRequirements(document map[string]any) []string {
	compute, alarm, collector := false, false, false
	if units, ok := document["computeUnits"].([]any); ok {
		for _, unit := range units {
			if item, ok := unit.(map[string]any); ok && len(item) > 0 {
				compute = true
				break
			}
		}
	}
	if items, ok := document["alarmItems"].([]any); ok {
		alarm = len(items) > 0
	}
	if points, ok := document["dataPoints"].([]any); ok {
		for _, point := range points {
			item, ok := point.(map[string]any)
			if !ok {
				continue
			}
			source, _ := item["sourceType"].(string)
			if source == "calc.output" {
				compute = true
			}
			if strings.HasPrefix(source, "collector.") {
				collector = true
			}
		}
	}
	result := []string{"base"}
	if compute {
		result = append(result, "compute")
	}
	if alarm {
		result = append(result, "alarm")
	}
	if collector {
		result = append(result, "collector")
	}
	return result
}
