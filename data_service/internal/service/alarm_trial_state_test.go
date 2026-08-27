package service

import (
	"testing"
	"time"
)

func alarmTrialTime(second int) time.Time {
	return time.Date(2026, 8, 25, 8, 0, second, 0, time.UTC)
}

func alarmTrialSample(second int, value any, quality string) AlarmTrialSample {
	timestamp := alarmTrialTime(second)
	return AlarmTrialSample{ObservedAt: timestamp, SourceTimestamp: &timestamp, Value: value, Quality: quality}
}

func TestAlarmTrialStartsDelayAtZeroAndUpgradesWithIndependentCandidate(t *testing.T) {
	input := SaveAlarmItemInput{Mode: "point", EvaluationMode: "highest_matching", Conditions: []AlarmCondition{
		{ID: "high", Kind: "threshold", Operator: "gt", Label: "高", Severity: "warning", Params: map[string]any{"threshold": 80.0}},
		{ID: "high-high", Kind: "threshold", Operator: "gt", Label: "高高", Severity: "critical", Params: map[string]any{"threshold": 90.0}, TriggerDelayMS: 2000},
	}}
	result, err := runAlarmTrial(input, []AlarmTrialSample{
		alarmTrialSample(0, 85.0, "good"),
		alarmTrialSample(1, 95.0, "good"),
		alarmTrialSample(2, 95.0, "good"),
		alarmTrialSample(3, 95.0, "good"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Steps[0].ActiveCondition == nil || result.Steps[0].ActiveCondition.ID != "high" {
		t.Fatalf("first step = %#v, want high active immediately", result.Steps[0])
	}
	if result.Steps[1].CandidateElapsedMS != 0 || result.Steps[1].ActiveCondition.ID != "high" {
		t.Fatalf("candidate must start at zero while high remains active: %#v", result.Steps[1])
	}
	if result.Steps[2].ActiveCondition.ID != "high" || result.Steps[2].CandidateElapsedMS != 1000 {
		t.Fatalf("high-high must still be pending: %#v", result.Steps[2])
	}
	if result.Steps[3].ActiveCondition.ID != "high-high" {
		t.Fatalf("last step = %#v, want high-high after its own delay", result.Steps[3])
	}
}

func TestAlarmTrialPausesActiveAndCandidateTimers(t *testing.T) {
	input := SaveAlarmItemInput{Mode: "point", EvaluationMode: "highest_matching", Conditions: []AlarmCondition{
		{ID: "high", Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 80.0}},
		{ID: "high-high", Kind: "threshold", Operator: "gt", Severity: "critical", Params: map[string]any{"threshold": 90.0}, TriggerDelayMS: 2000},
	}}
	result, err := runAlarmTrial(input, []AlarmTrialSample{
		alarmTrialSample(0, 85.0, "good"),
		alarmTrialSample(1, 95.0, "good"),
		alarmTrialSample(2, 95.0, "bad"),
		alarmTrialSample(5, 95.0, "good"),
		alarmTrialSample(6, 95.0, "good"),
		alarmTrialSample(7, 95.0, "good"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Steps[2].EvaluationState != alarmEvaluationPaused || result.Steps[2].ActiveCondition.ID != "high" {
		t.Fatalf("bad quality must pause without clearing: %#v", result.Steps[2])
	}
	if result.Steps[3].CandidateElapsedMS != 0 {
		t.Fatalf("paused wall time must not be added after resume: %#v", result.Steps[3])
	}
	if result.Steps[5].ActiveCondition.ID != "high-high" {
		t.Fatalf("candidate should finish only after two good seconds: %#v", result.Steps[5])
	}
}

func TestAlarmTrialDowngradesAfterActiveLevelClearDelay(t *testing.T) {
	input := SaveAlarmItemInput{Mode: "point", EvaluationMode: "highest_matching", Conditions: []AlarmCondition{
		{ID: "high", Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 80.0}},
		{ID: "high-high", Kind: "threshold", Operator: "gt", Severity: "critical", Params: map[string]any{"threshold": 90.0}, ClearDelayMS: 2000},
	}}
	result, err := runAlarmTrial(input, []AlarmTrialSample{
		alarmTrialSample(0, 95.0, "good"),
		alarmTrialSample(1, 85.0, "good"),
		alarmTrialSample(2, 85.0, "good"),
		alarmTrialSample(3, 85.0, "good"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Steps[1].State != "pending_clear" || result.Steps[1].ActiveCondition.ID != "high-high" {
		t.Fatalf("high-high must remain during its clear delay: %#v", result.Steps[1])
	}
	if result.Steps[1].ClearElapsedMS != 0 || result.Steps[2].ActiveCondition.ID != "high-high" {
		t.Fatalf("clear delay must start at zero on the first clear sample: %#v", result.Steps[1:3])
	}
	if result.Steps[3].ActiveCondition == nil || result.Steps[3].ActiveCondition.ID != "high" {
		t.Fatalf("alarm should downgrade to high after clear delay: %#v", result.Steps[3])
	}
}

func TestAlarmTrialRateUsesSourceTimestampsAndDirection(t *testing.T) {
	input := SaveAlarmItemInput{Mode: "point", EvaluationMode: "single", Conditions: []AlarmCondition{{
		ID: "rate", Kind: "rate_of_change", Operator: "gt", Severity: "warning",
		Params: map[string]any{"direction": "rise", "limit": 2.0, "windowMs": 10000.0},
	}}}
	result, err := runAlarmTrial(input, []AlarmTrialSample{
		alarmTrialSample(0, 10.0, "good"),
		alarmTrialSample(2, 16.0, "good"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Steps[1].CalculatedRate == nil || *result.Steps[1].CalculatedRate != 3 {
		t.Fatalf("calculated rate = %#v, want 3 units/s", result.Steps[1].CalculatedRate)
	}
	if result.Steps[1].ActiveCondition == nil {
		t.Fatalf("rise alarm should trigger: %#v", result.Steps[1])
	}
}

func TestAlarmTrialQualityAndStaleConditionsUseMetadata(t *testing.T) {
	quality := SaveAlarmItemInput{Mode: "point", EvaluationMode: "single", Conditions: []AlarmCondition{{ID: "quality", Kind: "quality", Operator: "in", Severity: "major", Params: map[string]any{"qualities": []any{"bad"}}}}}
	qualityResult, err := runAlarmTrial(quality, []AlarmTrialSample{alarmTrialSample(0, 1, "bad")})
	if err != nil || qualityResult.Steps[0].ActiveCondition == nil {
		t.Fatalf("quality alarm result = %#v, err = %v", qualityResult, err)
	}

	sourceTimestamp := alarmTrialTime(0)
	stale := SaveAlarmItemInput{Mode: "point", EvaluationMode: "single", Conditions: []AlarmCondition{{ID: "stale", Kind: "stale", Operator: "age_gte", Severity: "warning", Params: map[string]any{"maxAgeMs": 5000.0}}}}
	staleResult, err := runAlarmTrial(stale, []AlarmTrialSample{{ObservedAt: alarmTrialTime(6), SourceTimestamp: &sourceTimestamp, Value: 1, Quality: "good"}})
	if err != nil || staleResult.Steps[0].ActiveCondition == nil {
		t.Fatalf("stale alarm result = %#v, err = %v", staleResult, err)
	}
}

func TestAlarmTrialStaleSupportsMissingTimestampAndRejectsOutOfOrderSamples(t *testing.T) {
	input := SaveAlarmItemInput{Mode: "point", EvaluationMode: "single", Conditions: []AlarmCondition{{ID: "stale", Kind: "stale", Operator: "age_gte", Severity: "warning", Params: map[string]any{"maxAgeMs": 2000.0}}}}
	result, err := runAlarmTrial(input, []AlarmTrialSample{
		{ObservedAt: alarmTrialTime(0), Value: 1, Quality: "good"},
		{ObservedAt: alarmTrialTime(1), Value: 1, Quality: "good"},
		{ObservedAt: alarmTrialTime(2), Value: 1, Quality: "good"},
	})
	if err != nil || result.Steps[2].ActiveCondition == nil {
		t.Fatalf("missing timestamp should become stale after max age: result=%#v err=%v", result, err)
	}
	_, err = runAlarmTrial(input, []AlarmTrialSample{
		{ObservedAt: alarmTrialTime(1), Value: 1, Quality: "good"},
		{ObservedAt: alarmTrialTime(0), Value: 1, Quality: "good"},
	})
	if err == nil {
		t.Fatal("out-of-order samples must be rejected")
	}
}

func TestAlarmConditionValidationRejectsInvalidAdvancedFields(t *testing.T) {
	if err := validateConditionParams(AlarmCondition{Kind: "transition", Operator: "changed", TriggerDelayMS: 1}); err == nil {
		t.Fatal("transition trigger delay must be rejected")
	}
	if _, err := normalizeConfigurationConditions([]AlarmCondition{{Kind: "state", Operator: "eq", Severity: "warning", Params: map[string]any{"expected": true}, Deadband: 1}}, "boolean", "single", false); err == nil {
		t.Fatal("state deadband must be rejected")
	}
	if _, err := normalizeConfigurationConditions([]AlarmCondition{{Kind: "quality", Operator: "in", Severity: "warning", Params: map[string]any{"qualities": []any{"bad", "unknown"}}}}, "structured", "single", false); err != nil {
		t.Fatalf("metadata alarm should be valid for structured points: %v", err)
	}
}
