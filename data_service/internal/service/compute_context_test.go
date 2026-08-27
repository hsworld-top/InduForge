package service

import "testing"

func TestSafeComputeVariableAliasUsesCommonJavaScriptAndPythonSubset(t *testing.T) {
	valid := []string{"temperature", "line_1_speed", "_internalValue"}
	for _, alias := range valid {
		if !isSafeComputeVariableAlias(alias) {
			t.Fatalf("valid alias rejected: %s", alias)
		}
	}

	invalid := []string{"$temperature", "1temperature", "line-speed", "__private", "ctx", "dp", "argv", "require", "class", "def"}
	for _, alias := range invalid {
		if isSafeComputeVariableAlias(alias) {
			t.Fatalf("invalid alias accepted: %s", alias)
		}
	}
}

func TestExtractComputeDatapointVariableBindingsDropsInvalidAndDuplicateItems(t *testing.T) {
	bindings := extractComputeDatapointVariableBindings(map[string]any{
		"datapointVariables": []any{
			map[string]any{"alias": "temperature", "path": "factory.temperature"},
			map[string]any{"alias": "temperature", "path": "factory.temperature"},
			map[string]any{"alias": "ctx", "path": "factory.invalid"},
			map[string]any{"alias": "speed", "datapointPath": "factory.speed"},
		},
	})
	if len(bindings) != 2 {
		t.Fatalf("bindings = %+v, want two valid unique bindings", bindings)
	}
	if bindings[0].Alias != "temperature" || bindings[1].Alias != "speed" {
		t.Fatalf("unexpected bindings: %+v", bindings)
	}
}
