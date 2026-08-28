package service

import (
	"testing"

	enginecompute "github.com/indu-forge/data_service/internal/engine/compute"
)

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

func TestApplyComputeDebugDatapointValuesOverridesDeclaredBindingsOnly(t *testing.T) {
	sdk := enginecompute.SDKContext{
		Datapoints: map[string]enginecompute.SDKDataPointValue{
			"factory.temperature": {Path: "factory.temperature", Value: 12.5},
		},
		PointBindings: map[string]string{"temperature": "factory.temperature"},
		Variables:     map[string]any{"temperature": 12.5},
	}

	if err := applyComputeDebugDatapointValues(&sdk, map[string]any{
		"datapoints": map[string]any{
			"temperature": map[string]any{
				"value":           30.0,
				"quality":         "good",
				"timestamp":       nil,
				"observedAt":      "2026-08-28T01:00:00Z",
				"sourceTimestamp": nil,
			},
			"undeclared": map[string]any{"value": 99.0},
		},
	}); err != nil {
		t.Fatal(err)
	}

	if got := sdk.Datapoints["factory.temperature"].Value; got != 30.0 {
		t.Fatalf("snapshot value = %v, want 30", got)
	}
	if got := sdk.Variables["temperature"]; got != 30.0 {
		t.Fatalf("variable value = %v, want 30", got)
	}
	if sdk.Datapoints["factory.temperature"].ObservedAt == nil || *sdk.Datapoints["factory.temperature"].ObservedAt != "2026-08-28T01:00:00Z" {
		t.Fatalf("observedAt was not overridden: %+v", sdk.Datapoints["factory.temperature"].ObservedAt)
	}
	if _, exists := sdk.Variables["undeclared"]; exists {
		t.Fatal("undeclared debug value must not enter SDK variables")
	}
}

func TestApplyComputeDebugDatapointValuesRejectsPrimitiveSnapshot(t *testing.T) {
	sdk := enginecompute.SDKContext{
		Datapoints:    map[string]enginecompute.SDKDataPointValue{"factory.temperature": {Path: "factory.temperature"}},
		PointBindings: map[string]string{"temperature": "factory.temperature"},
		Variables:     map[string]any{},
	}
	err := applyComputeDebugDatapointValues(&sdk, map[string]any{
		"datapoints": map[string]any{"temperature": 30.0},
	})
	if err == nil {
		t.Fatal("primitive debug snapshot must be rejected")
	}
}
