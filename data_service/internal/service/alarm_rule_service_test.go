package service

import (
	"testing"

	"github.com/indu-forge/data_service/internal/repository"
)

func TestEvaluateAlarmRuleFinalTypes(t *testing.T) {
	tests := []struct {
		name      string
		ruleType  string
		condition map[string]any
		value     any
		context   map[string]any
		wantState string
	}{
		{"H triggered", "H", map[string]any{"limit": 80.0}, 81.0, nil, "triggered"},
		{"L not triggered", "L", map[string]any{"limit": 10.0}, 11.0, nil, "not_triggered"},
		{"deviation needs baseline", "deviation_high", map[string]any{"limit": 5.0}, 80.0, nil, "insufficient_input"},
		{"rate needs previous", "rate_of_change", map[string]any{"limit": 2.0, "windowMs": 60000.0}, 80.0, nil, "insufficient_input"},
		{"cel triggered", "cel", map[string]any{"expression": "value > 80"}, 81.0, nil, "triggered"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluateAlarmRule(tt.ruleType, tt.condition, tt.value, tt.context)
			if err != nil {
				t.Fatalf("evaluateAlarmRule() error = %v", err)
			}
			if result.State != tt.wantState {
				t.Fatalf("State = %q, want %q", result.State, tt.wantState)
			}
		})
	}
}

func TestBuildAlarmRuleContractFinalShape(t *testing.T) {
	input := normalizedAlarmRuleInput{
		Name:            "温度高报",
		TargetPath:      "metrics.temperature",
		RuleType:        "H",
		Condition:       map[string]any{"limit": 80.0},
		Severity:        "major",
		Suppression:     map[string]any{"enabled": false},
		MessageTemplate: "温度 {{value}}",
		IsEnabled:       true,
	}
	target := fakeAlarmDatapoint("dp-1", "metrics.temperature", "number")

	contract := buildAlarmRuleContract(input, target, "rule-1", "project-1")
	if contract["schemaVersion"] != "alarm.rule.v1" {
		t.Fatalf("schemaVersion = %v", contract["schemaVersion"])
	}
	if contract["ruleType"] != "H" {
		t.Fatalf("ruleType = %v", contract["ruleType"])
	}
	if contract["messageTemplate"] != "温度 {{value}}" {
		t.Fatalf("messageTemplate = %v", contract["messageTemplate"])
	}
}

func fakeAlarmDatapoint(id, path, dataType string) *repository.DataPointRecord {
	return &repository.DataPointRecord{
		ID:       id,
		Path:     path,
		DataType: dataType,
	}
}
