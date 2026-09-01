package releasebuilder

import "strings"

// DeriveEngineRequirements 从发布实际打包的工程文档递归识别引擎需求。基础引擎
// 恒定存在；计算、报警、采集只由保存内容中的正式配置键触发，调用方不得传入覆盖值。
func DeriveEngineRequirements(document map[string]any) []string {
	found := map[string]bool{"base": true}
	var walk func(any, string)
	walk = func(value any, key string) {
		key = strings.ToLower(key)
		if value != nil {
			switch {
			case strings.Contains(key, "alarm"):
				found["alarm"] = true
			case strings.Contains(key, "collector") || strings.Contains(key, "acquisition") || strings.Contains(key, "devicepoint"):
				found["collector"] = true
			case strings.Contains(key, "calculation") || strings.Contains(key, "expression") || strings.Contains(key, "computed"):
				found["compute"] = true
			}
		}
		switch item := value.(type) {
		case map[string]any:
			for childKey, child := range item {
				walk(child, childKey)
			}
		case []any:
			for _, child := range item {
				walk(child, key)
			}
		}
	}
	walk(document, "")
	result := []string{"base"}
	for _, engine := range []string{"compute", "alarm", "collector"} {
		if found[engine] {
			result = append(result, engine)
		}
	}
	return result
}
