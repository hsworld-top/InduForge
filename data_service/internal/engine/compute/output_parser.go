package compute

import (
	"encoding/json"
	"fmt"
	"strings"
)

const resultMarker = "__DATA_SERVICE_RESULT__:"

// parseScriptResult 从脚本输出中抽取标记后的 JSON 结果。
func parseScriptResult(stdout string) (any, error) {
	index := strings.LastIndex(stdout, resultMarker)
	if index < 0 {
		return nil, fmt.Errorf("runner output missing marker")
	}

	payload := strings.TrimSpace(stdout[index+len(resultMarker):])
	if payload == "" {
		return nil, fmt.Errorf("runner output payload is empty")
	}

	var envelope struct {
		Result any `json:"result"`
	}
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
		return nil, fmt.Errorf("decode runner output failed: %w", err)
	}
	return envelope.Result, nil
}
