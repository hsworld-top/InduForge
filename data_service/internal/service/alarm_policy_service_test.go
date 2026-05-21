package service

import "testing"

func TestEvaluateAlarmPolicyConditions(t *testing.T) {
	policy := AlarmPolicy{
		Mode: "per_target",
		Conditions: []AlarmCondition{
			{ID: "c-h", Type: "H", Name: "高限", IsEnabled: true, Severity: "major", Params: map[string]any{"limit": 80.0}},
			{ID: "c-l", Type: "L", Name: "低限", IsEnabled: true, Severity: "warning", Params: map[string]any{"limit": 20.0}},
		},
	}

	result, err := evaluateAlarmPolicy(&policy, 82.0, nil)
	if err != nil {
		t.Fatalf("evaluateAlarmPolicy() error = %v", err)
	}
	if !result.Triggered {
		t.Fatalf("Triggered = false, want true")
	}
	if len(result.TriggeredConditions) != 1 || result.TriggeredConditions[0].Type != "H" {
		t.Fatalf("TriggeredConditions = %#v, want only H", result.TriggeredConditions)
	}
}

func TestEvaluateAlarmPolicyTextCondition(t *testing.T) {
	policy := AlarmPolicy{
		Mode: "per_target",
		Conditions: []AlarmCondition{
			{ID: "c-text", Type: "string_contains", Name: "文本包含", IsEnabled: true, Severity: "warning", Params: map[string]any{"expected": "error"}},
		},
	}

	result, err := evaluateAlarmPolicy(&policy, "device error", nil)
	if err != nil {
		t.Fatalf("evaluateAlarmPolicy() error = %v", err)
	}
	if !result.Triggered {
		t.Fatalf("Triggered = false, want true")
	}
}

func TestValidateAlarmConditionsRejectsDuplicateThreshold(t *testing.T) {
	conditions := []AlarmCondition{
		{ID: "c-h-1", Type: "H", Name: "高限", IsEnabled: true, Severity: "warning", Params: map[string]any{"limit": 80.0}},
		{ID: "c-h-2", Type: "H", Name: "高限", IsEnabled: true, Severity: "major", Params: map[string]any{"limit": 90.0}},
	}

	err := validateAlarmConditions(conditions, policyAllowedConditionTypes("per_target", []AlarmTargetRef{
		{DatapointID: "dp-1", Path: "metrics.temperature", DataType: "number"},
	}), true)
	if err == nil {
		t.Fatal("validateAlarmConditions() error = nil, want duplicate threshold error")
	}
}

func TestValidateAlarmConditionsAllowsBooleanCondition(t *testing.T) {
	conditions := []AlarmCondition{
		{ID: "c-bool", Type: "bool_equal", Name: "状态判断", IsEnabled: true, Severity: "warning", Params: map[string]any{"expected": true}},
	}

	err := validateAlarmConditions(conditions, policyAllowedConditionTypes("per_target", []AlarmTargetRef{
		{DatapointID: "dp-1", Path: "metrics.running", DataType: "boolean"},
	}), true)
	if err != nil {
		t.Fatalf("validateAlarmConditions() error = %v", err)
	}
}

func TestBuildAlarmPolicyContract(t *testing.T) {
	policy := normalizedAlarmPolicyInput{
		Name: "温度策略",
		Mode: "per_target",
		Targets: []AlarmTargetRef{
			{DatapointID: "dp-1", Path: "metrics.temperature", DataType: "number"},
		},
		Conditions: []AlarmCondition{
			{ID: "c-h", Type: "H", Name: "高限", IsEnabled: true, Severity: "major", Params: map[string]any{"limit": 80.0}},
		},
		IsEnabled: true,
	}

	contract := buildAlarmPolicyContract(policy, "policy-1", "project-1", nil)
	if contract["schemaVersion"] != "alarm.policy.v1" {
		t.Fatalf("schemaVersion = %v", contract["schemaVersion"])
	}
	if contract["effectiveEnabled"] != true {
		t.Fatalf("effectiveEnabled = %v", contract["effectiveEnabled"])
	}
}

func TestPolicyMatchesCoverageTargetOnlyPerTarget(t *testing.T) {
	policy := AlarmPolicy{
		Mode: "per_target",
		Targets: []AlarmTargetRef{
			{DatapointID: "dp-1", Path: "metrics.temperature", DataType: "number"},
		},
	}
	if !policyMatchesCoverageTarget(policy, "dp-1", "") {
		t.Fatal("policyMatchesCoverageTarget() = false, want true by datapoint id")
	}
	if !policyMatchesCoverageTarget(policy, "", "metrics.temperature") {
		t.Fatal("policyMatchesCoverageTarget() = false, want true by path")
	}

	policy.Mode = "derived"
	if policyMatchesCoverageTarget(policy, "dp-1", "metrics.temperature") {
		t.Fatal("policyMatchesCoverageTarget() = true, want false for derived mode")
	}
}

func TestEvaluateDerivedValue(t *testing.T) {
	value, err := evaluateDerivedValue("temperature - pressure * 0.1", map[string]any{
		"temperature": 90.0,
		"pressure":    20.0,
	})
	if err != nil {
		t.Fatalf("evaluateDerivedValue() error = %v", err)
	}
	if value != 88 {
		t.Fatalf("value = %v, want 88", value)
	}
}
