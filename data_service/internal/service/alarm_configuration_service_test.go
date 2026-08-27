package service

import (
	"encoding/json"
	"testing"

	"github.com/indu-forge/data_service/internal/repository"
)

func TestAlarmItemJSONUsesPublicAPICasing(t *testing.T) {
	payload, err := json.Marshal(AlarmItem{
		ID:             "alarm-item-1",
		ProjectID:      "project-1",
		DisplayName:    "温度越限",
		Mode:           "point",
		AlarmType:      "threshold",
		EvaluationMode: "highest_matching",
		DatapointID:    "datapoint-1",
		DatapointName:  "温度",
		DataType:       "float64",
		Inputs:         []AlarmItemInput{},
		Conditions:     []AlarmCondition{},
		Notification:   AlarmNotificationSettings{Mode: "inherit", ChannelIDs: []string{}},
		IsEnabled:      true,
		Revision:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err = json.Unmarshal(payload, &result); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "projectId", "displayName", "mode", "alarmType", "evaluationMode", "notification", "isEnabled", "revision"} {
		if _, ok := result[key]; !ok {
			t.Fatalf("public alarm response is missing %q: %s", key, payload)
		}
	}
	if _, leaked := result["ID"]; leaked {
		t.Fatalf("Go field casing leaked into public alarm response: %s", payload)
	}
	if result["datapointId"] != "datapoint-1" {
		t.Fatalf("alarm item response uses an invalid public shape: %s", payload)
	}
}

func TestResetAlarmItemChildIDsForCreatePreventsBatchPrimaryKeyReuse(t *testing.T) {
	input := SaveAlarmItemInput{
		Inputs:     []AlarmItemInput{{ID: "draft-input", DatapointID: "point-1"}},
		Conditions: []AlarmCondition{{ID: "draft-condition", Kind: "threshold"}},
	}

	resetAlarmItemChildIDsForCreate(&input)

	if input.Inputs[0].ID != "" || input.Conditions[0].ID != "" {
		t.Fatalf("create input must not reuse draft child IDs: %#v", input)
	}
}

func TestNormalizeConfigurationConditionsSupportsSingleConditionKinds(t *testing.T) {
	tests := []struct {
		name      string
		category  string
		condition AlarmCondition
	}{
		{name: "threshold", category: "number", condition: AlarmCondition{Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 80.0}}},
		{name: "range", category: "number", condition: AlarmCondition{Kind: "range", Operator: "outside", Severity: "major", Params: map[string]any{"lower": 10.0, "upper": 20.0}}},
		{name: "state", category: "boolean", condition: AlarmCondition{Kind: "state", Operator: "eq", Severity: "info", Params: map[string]any{"expected": true}}},
		{name: "transition", category: "boolean", condition: AlarmCondition{Kind: "transition", Operator: "changed", Severity: "critical", Params: map[string]any{}}},
		{name: "text", category: "text", condition: AlarmCondition{Kind: "text_match", Operator: "contains", Severity: "warning", Params: map[string]any{"expected": "fault"}}},
		{name: "rate", category: "number", condition: AlarmCondition{Kind: "rate_of_change", Operator: "gt", Severity: "major", Params: map[string]any{"direction": "absolute", "limit": 2.0, "windowMs": 1000.0}}},
		{name: "deviation", category: "number", condition: AlarmCondition{Kind: "deviation", Operator: "gt", Severity: "warning", Params: map[string]any{"baseline": 10.0, "limit": 2.0}}},
		{name: "offline", category: "number", condition: AlarmCondition{Kind: "offline", Operator: "is_offline", Severity: "critical", Params: map[string]any{}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conditions, err := normalizeConfigurationConditions([]AlarmCondition{test.condition}, test.category, "single", false)
			if err != nil {
				t.Fatalf("normalizeConfigurationConditions() error = %v", err)
			}
			if conditions[0].ID == "" || conditions[0].Label == "" {
				t.Fatalf("condition defaults not generated: %#v", conditions[0])
			}
		})
	}
}

func TestValidateSnapshotAlarmItemsRejectsInvalidFingerprintAndLevelShape(t *testing.T) {
	condition := AlarmCondition{ID: "condition", Kind: "threshold", Operator: "gt", Label: "高", Severity: "warning", Params: map[string]any{"threshold": 80.0}}
	input := SaveAlarmItemInput{Mode: "point", EvaluationMode: "highest_matching", Conditions: []AlarmCondition{condition}}
	fingerprint, err := alarmTriggerFingerprint(input)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := repository.ProjectSnapshot{
		DataPoints: []repository.DataPointRecord{{ID: "dp-1", DataType: "float64"}},
		AlarmItems: []repository.AlarmItemRecord{{
			ID: "alarm-item", DatapointID: alarmTestStringPointer("dp-1"), DisplayName: "温度报警", NameKey: "温度报警", Mode: "point", AlarmType: "threshold", EvaluationMode: "highest_matching", TriggerFingerprint: fingerprint,
			Conditions: []repository.AlarmItemConditionRecord{{ID: condition.ID, Kind: condition.Kind, Operator: condition.Operator, Label: condition.Label, Severity: condition.Severity, Params: condition.Params}},
		}},
	}
	if err = validateSnapshotAlarmItems(snapshot); err != nil {
		t.Fatalf("valid alarm snapshot rejected: %v", err)
	}
	snapshot.AlarmItems[0].TriggerFingerprint = "invalid"
	if err = validateSnapshotAlarmItems(snapshot); err == nil {
		t.Fatal("mismatched trigger fingerprint should be rejected")
	}
	snapshot.AlarmItems[0].TriggerFingerprint = fingerprint
	snapshot.AlarmItems[0].Conditions = append(snapshot.AlarmItems[0].Conditions, repository.AlarmItemConditionRecord{ID: "invalid", Kind: "range", Operator: "outside", Severity: "warning", Params: map[string]any{"lower": 1.0, "upper": 2.0}})
	if err = validateSnapshotAlarmItems(snapshot); err == nil {
		t.Fatal("highest matching alarm with non-threshold condition should be rejected")
	}
}

func TestHighestMatchingConditionsSortAndSelectDeepestLevel(t *testing.T) {
	conditions, err := normalizeConfigurationConditions([]AlarmCondition{
		{ID: "hh", Kind: "threshold", Operator: "gt", Label: "高高", Severity: "critical", Params: map[string]any{"threshold": 90.0}},
		{ID: "l", Kind: "threshold", Operator: "lt", Label: "低", Severity: "warning", Params: map[string]any{"threshold": 20.0}},
		{ID: "h", Kind: "threshold", Operator: "gt", Label: "高", Severity: "warning", Params: map[string]any{"threshold": 80.0}},
		{ID: "ll", Kind: "threshold", Operator: "lt", Label: "低低", Severity: "critical", Params: map[string]any{"threshold": 10.0}},
	}, "number", "highest_matching", false)
	if err != nil {
		t.Fatalf("normalize highest matching conditions: %v", err)
	}
	wantOrder := []string{"h", "hh", "l", "ll"}
	for index, want := range wantOrder {
		if conditions[index].ID != want {
			t.Fatalf("conditions[%d].ID = %s, want %s", index, conditions[index].ID, want)
		}
	}
	values := []struct {
		value float64
		want  string
	}{{50, ""}, {85, "h"}, {95, "hh"}, {85, "h"}, {50, ""}, {15, "l"}, {5, "ll"}}
	for _, item := range values {
		selected := selectAlarmCondition("highest_matching", conditions, item.value, nil)
		if item.want == "" && selected != nil {
			t.Fatalf("value %v selected %s, want normal", item.value, selected.ID)
		}
		if item.want != "" && (selected == nil || selected.ID != item.want) {
			t.Fatalf("value %v selected %#v, want %s", item.value, selected, item.want)
		}
	}
}

func TestHighestMatchingConditionsRejectInvalidLevels(t *testing.T) {
	tests := [][]AlarmCondition{
		{{Kind: "range", Operator: "outside", Severity: "warning", Params: map[string]any{"lower": 1.0, "upper": 2.0}}},
		{{Kind: "threshold", Operator: "gt", Severity: "critical", Params: map[string]any{"threshold": 80.0}}, {Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 90.0}}},
		{{Kind: "threshold", Operator: "lt", Severity: "warning", Params: map[string]any{"threshold": 80.0}, Deadband: 10}, {Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 90.0}, Deadband: 10}},
		{{Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 80.0}}, {Kind: "threshold", Operator: "lt", Severity: "warning", Params: map[string]any{"threshold": 80.0}}},
	}
	for _, conditions := range tests {
		if _, err := normalizeConfigurationConditions(conditions, "number", "highest_matching", false); err == nil {
			t.Fatalf("conditions %#v should be rejected", conditions)
		}
	}
}

func TestAlarmTriggerFingerprintIgnoresPresentationAndNotification(t *testing.T) {
	base := SaveAlarmItemInput{Mode: "point", EvaluationMode: "single", DisplayName: "温度报警", Conditions: []AlarmCondition{{Kind: "threshold", Operator: "gt", Label: "高温", Severity: "warning", Params: map[string]any{"threshold": 80.0}, TriggerDelayMS: 1000, Deadband: 1}}}
	first, err := alarmTriggerFingerprint(base)
	if err != nil {
		t.Fatal(err)
	}
	base.DisplayName = "另一个名称"
	base.Description = alarmTestStringPointer("说明")
	base.Conditions[0].Label = "另一标签"
	base.Conditions[0].Severity = "critical"
	base.Conditions[0].Params["configurationMode"] = "advanced"
	base.Notification = AlarmNotificationSettings{Mode: "off"}
	second, err := alarmTriggerFingerprint(base)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("presentation-only changes changed fingerprint: %s != %s", first, second)
	}
	base.Conditions[0].TriggerDelayMS = 2000
	third, _ := alarmTriggerFingerprint(base)
	if third == second {
		t.Fatal("trigger delay must participate in fingerprint")
	}
}

func TestNormalizeConfigurationConditionsRemovesObsoletePriority(t *testing.T) {
	conditions, err := normalizeConfigurationConditions([]AlarmCondition{{Kind: "state", Operator: "eq", Severity: "warning", Params: map[string]any{"expected": true, "priority": 90}}}, "boolean", "single", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := conditions[0].Params["priority"]; exists {
		t.Fatalf("obsolete priority should be removed: %#v", conditions[0].Params)
	}
}

func TestNormalizeAlarmLevelSettingsSupportsCustomSeverityAndEscalation(t *testing.T) {
	definitions := defaultAlarmSeverityDefinitions()
	definitions = append(definitions, AlarmSeverityDefinition{Key: "shutdown", DisplayName: "停机", Color: "#991b1b", SortOrder: 50})
	settings, err := normalizeAlarmLevelSettings("project", SaveAlarmLevelSettingsInput{
		SeverityDefinitions: definitions,
		EscalationRules: []AlarmEscalationRule{{
			SourceSeverity:        "critical",
			TargetSeverity:        "shutdown",
			UnacknowledgedSeconds: 300,
			IsEnabled:             true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(settings.EscalationRules) != 1 || settings.EscalationRules[0].ID == "" {
		t.Fatalf("normalized escalation rules = %#v", settings.EscalationRules)
	}
	allowed, order := map[string]bool{}, map[string]int{}
	for _, definition := range settings.SeverityDefinitions {
		allowed[definition.Key] = true
		order[definition.Key] = definition.SortOrder
	}
	_, err = normalizeConfigurationConditionsWithPolicy([]AlarmCondition{{Kind: "state", Operator: "eq", Severity: "shutdown", Params: map[string]any{"expected": true}}}, "boolean", "single", false, allowed, order)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeAlarmLevelSettingsRejectsMissingBuiltinAndDowngrade(t *testing.T) {
	definitions := defaultAlarmSeverityDefinitions()[1:]
	if _, err := normalizeAlarmLevelSettings("project", SaveAlarmLevelSettingsInput{SeverityDefinitions: definitions}); err == nil {
		t.Fatal("missing builtin severity should be rejected")
	}
	definitions = defaultAlarmSeverityDefinitions()
	if _, err := normalizeAlarmLevelSettings("project", SaveAlarmLevelSettingsInput{
		SeverityDefinitions: definitions,
		EscalationRules:     []AlarmEscalationRule{{ID: "077880a5-0056-4cf0-9a97-7069f78defd6", SourceSeverity: "critical", TargetSeverity: "warning", UnacknowledgedSeconds: 60, IsEnabled: true}},
	}); err == nil {
		t.Fatal("severity downgrade should be rejected")
	}
}

func TestDefaultAlarmItemNameUsesCompactSuffix(t *testing.T) {
	if got := defaultAlarmItemName("threshold"); got != "越限" {
		t.Fatalf("threshold suffix = %q, want 越限", got)
	}
	if got := defaultAlarmItemName("rate_of_change"); got != "变化率" {
		t.Fatalf("rate suffix = %q, want 变化率", got)
	}
}

func TestAlarmTriggerFingerprintDistinguishesThresholdAndRateOfChange(t *testing.T) {
	threshold := SaveAlarmItemInput{Mode: "point", EvaluationMode: "single", Conditions: []AlarmCondition{{Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 80.0}}}}
	rate := SaveAlarmItemInput{Mode: "point", EvaluationMode: "single", Conditions: []AlarmCondition{{Kind: "rate_of_change", Operator: "gt", Severity: "warning", Params: map[string]any{"direction": "absolute", "limit": 2.0, "windowMs": 60000.0}}}}
	thresholdFingerprint, err := alarmTriggerFingerprint(threshold)
	if err != nil {
		t.Fatal(err)
	}
	rateFingerprint, err := alarmTriggerFingerprint(rate)
	if err != nil {
		t.Fatal(err)
	}
	if thresholdFingerprint == rateFingerprint {
		t.Fatal("threshold and rate-of-change alarms on one datapoint must remain distinct")
	}
}

func TestDerivedFingerprintIncludesInputIdentityAndAlias(t *testing.T) {
	input := SaveAlarmItemInput{Mode: "derived", EvaluationMode: "single", DerivedExpression: "a && b", Inputs: []AlarmItemInput{{DatapointID: "dp-a", InputKey: "a"}, {DatapointID: "dp-b", InputKey: "b"}}, Conditions: []AlarmCondition{{Kind: "state", Operator: "eq", Severity: "warning", Params: map[string]any{"expected": true}}}}
	first, _ := alarmTriggerFingerprint(input)
	input.Inputs[1].DatapointID = "dp-c"
	second, _ := alarmTriggerFingerprint(input)
	if first == second {
		t.Fatal("derived input datapoint identity must participate in fingerprint")
	}
}

func TestAlarmConditionsOverlapDistinguishesSeparatedHighAndLowRegions(t *testing.T) {
	high := []AlarmCondition{{Kind: "threshold", Operator: "gt", Params: map[string]any{"threshold": 80.0}}}
	separatedLow := []repository.AlarmItemConditionRecord{{Kind: "threshold", Operator: "lt", Params: map[string]any{"threshold": 20.0}}}
	overlappingLow := []repository.AlarmItemConditionRecord{{Kind: "threshold", Operator: "lt", Params: map[string]any{"threshold": 90.0}}}
	if alarmConditionsOverlap(high, separatedLow) {
		t.Fatal("separated high and low trigger regions should not overlap")
	}
	if !alarmConditionsOverlap(high, overlappingLow) {
		t.Fatal("crossed high and low trigger regions should produce an overlap warning")
	}
}

func TestAlarmConditionsOverlapAvoidsNonNumericFalseWarnings(t *testing.T) {
	tests := []struct {
		name     string
		current  AlarmCondition
		existing repository.AlarmItemConditionRecord
	}{
		{name: "different states", current: AlarmCondition{Kind: "state", Operator: "eq", Params: map[string]any{"expected": true}}, existing: repository.AlarmItemConditionRecord{Kind: "state", Operator: "eq", Params: map[string]any{"expected": false}}},
		{name: "different text equality", current: AlarmCondition{Kind: "text_match", Operator: "eq", Params: map[string]any{"expected": "running"}}, existing: repository.AlarmItemConditionRecord{Kind: "text_match", Operator: "eq", Params: map[string]any{"expected": "stopped"}}},
		{name: "disjoint qualities", current: AlarmCondition{Kind: "quality", Operator: "in", Params: map[string]any{"qualities": []any{"bad"}}}, existing: repository.AlarmItemConditionRecord{Kind: "quality", Operator: "in", Params: map[string]any{"qualities": []any{"unknown"}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if alarmConditionsOverlap([]AlarmCondition{test.current}, []repository.AlarmItemConditionRecord{test.existing}) {
				t.Fatal("mutually exclusive conditions must not produce overlap warnings")
			}
		})
	}
}

func alarmTestStringPointer(value string) *string { return &value }

func TestEvaluateAlarmConditionsPausesInvalidQualityButKeepsOfflineIndependent(t *testing.T) {
	conditions := []AlarmCondition{
		{ID: "threshold", Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 10.0}},
		{ID: "offline", Kind: "offline", Operator: "is_offline", Severity: "critical", Params: map[string]any{}},
	}
	result := evaluateAlarmConditions(conditions, 20.0, map[string]any{"quality": "bad", "offline": true})
	if !result.Triggered || len(result.TriggeredConditions) != 1 || result.TriggeredConditions[0].Kind != "offline" {
		t.Fatalf("result = %#v, want only offline condition triggered", result)
	}
}

func TestEvaluateAlarmConditionSupportsRateAndExplicitTransition(t *testing.T) {
	rate := AlarmCondition{Kind: "rate_of_change", Operator: "gt", Params: map[string]any{"direction": "absolute", "limit": 2.0, "windowMs": 1000.0}}
	triggered, reason := evaluateAlarmCondition(rate, 15.0, "good", false, map[string]any{"previousValue": 10.0})
	if !triggered || reason != "evaluated" {
		t.Fatalf("rate trial = (%v, %s), want triggered", triggered, reason)
	}
	transition := AlarmCondition{Kind: "transition", Operator: "from_to", Params: map[string]any{"from": false, "to": true}}
	triggered, reason = evaluateAlarmCondition(transition, true, "good", false, map[string]any{"previousValue": false})
	if !triggered || reason != "evaluated" {
		t.Fatalf("explicit transition trial = (%v, %s), want triggered", triggered, reason)
	}
	if err := validateConditionParams(AlarmCondition{Kind: "transition", Operator: "from_to", Params: map[string]any{"from": true, "to": true}}); err == nil {
		t.Fatal("same from/to state must be rejected")
	}
}

func TestEvaluateAlarmConditionUsesDeadbandWhileAlarmIsActive(t *testing.T) {
	condition := AlarmCondition{Kind: "threshold", Operator: "gt", Params: map[string]any{"threshold": 10.0}, Deadband: 1}
	triggered, _ := evaluateAlarmCondition(condition, 9.5, "good", false, map[string]any{})
	if triggered {
		t.Fatal("inactive high threshold must not trigger below its threshold")
	}
	triggered, _ = evaluateAlarmCondition(condition, 9.5, "good", false, map[string]any{"alarmActive": true})
	if !triggered {
		t.Fatal("active high threshold must remain active inside its deadband")
	}
}

func TestDerivedAlarmExpressionValidationAndEvaluation(t *testing.T) {
	aliases := map[string]bool{"temperature": true, "pressure": true}
	if err := validateDerivedAlarmExpression("temperature > 80 && pressure > 1.2", aliases); err != nil {
		t.Fatalf("valid derived expression rejected: %v", err)
	}
	if err := validateDerivedAlarmExpression("os.Execute(\"bad\")", aliases); err == nil {
		t.Fatal("function call must not be allowed in a derived expression")
	}
	value, err := evaluateDerivedAlarmExpression("temperature > 80 && pressure > 1.2", map[string]any{"temperature": 90.0, "pressure": 1.5})
	if err != nil || value != true {
		t.Fatalf("derived expression result = %#v, err = %v", value, err)
	}
}

func TestDerivedStateConditionRequiresBooleanExpectedValue(t *testing.T) {
	_, err := normalizeConfigurationConditions(
		[]AlarmCondition{{Kind: "state", Operator: "eq", Severity: "warning", Params: map[string]any{"expected": "true1"}}},
		"", "single", true,
	)
	if err == nil {
		t.Fatal("derived state condition must reject arbitrary text")
	}
	_, err = normalizeConfigurationConditions(
		[]AlarmCondition{{Kind: "state", Operator: "eq", Severity: "warning", Params: map[string]any{"expected": true}}},
		"", "single", true,
	)
	if err != nil {
		t.Fatalf("derived boolean state condition rejected: %v", err)
	}
}

func TestDerivedExpressionChecksInputAndResultTypes(t *testing.T) {
	inputs := []AlarmItemInput{{DatapointID: "run", InputKey: "running", DataType: "bool"}, {DatapointID: "temp", InputKey: "temperature", DataType: "float64"}}
	state := AlarmCondition{Kind: "state", Operator: "eq", Severity: "warning", Params: map[string]any{"expected": true}}
	if err := validateDerivedAlarmExpressionTypes("running && temperature > 80", inputs, state); err != nil {
		t.Fatalf("valid mixed input expression rejected: %v", err)
	}
	if err := validateDerivedAlarmExpressionTypes("running + temperature", inputs, state); err == nil {
		t.Fatal("boolean input must not be accepted in arithmetic")
	}
	threshold := AlarmCondition{Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 80.0}}
	if err := validateDerivedAlarmExpressionTypes("running && temperature > 80", inputs, threshold); err == nil {
		t.Fatal("boolean expression result must not be accepted by threshold condition")
	}
}

func TestNormalizeNotificationShape(t *testing.T) {
	inherit, err := normalizeNotificationShape(AlarmNotificationSettings{})
	if err != nil || inherit.Mode != "inherit" {
		t.Fatalf("inherit = %#v, err = %v", inherit, err)
	}
	raise, clear := true, true
	repeat := 300
	custom, err := normalizeNotificationShape(AlarmNotificationSettings{Mode: "custom", NotifyOnRaise: &raise, NotifyOnClear: &clear, RepeatIntervalSeconds: &repeat, ChannelIDs: []string{"runtime_inapp", "runtime_inapp"}})
	if err != nil || len(custom.ChannelIDs) != 1 {
		t.Fatalf("custom = %#v, err = %v", custom, err)
	}
}

func TestValidateChannelConfigOnlyChecksDevelopmentFormat(t *testing.T) {
	for _, channelType := range []string{"webhook", "dingtalk", "wecom"} {
		if err := validateChannelConfig(channelType, map[string]any{"webhookUrl": "https://example.com/hook"}); err != nil {
			t.Fatalf("validateChannelConfig(%s) error = %v", channelType, err)
		}
	}
	if err := validateChannelConfig("email", map[string]any{"webhookUrl": "https://example.com"}); err == nil {
		t.Fatal("unsupported channel type should be rejected")
	}
}

func TestAlarmHistorySettingsDefaultsAndValidation(t *testing.T) {
	settings := defaultAlarmHistorySettings("project-1")
	if !settings.IsEnabled || settings.RetentionDays == nil || *settings.RetentionDays != 30 || !settings.StoreNotificationDeliveries {
		t.Fatalf("default history settings = %#v", settings)
	}
	permanent := SaveAlarmHistorySettingsInput{IsEnabled: true, RetentionDays: nil, StoreNotificationDeliveries: true}
	if err := validateAlarmHistorySettings(permanent); err != nil {
		t.Fatalf("permanent history settings rejected: %v", err)
	}
}

func TestNormalizeSyncOperationSupportsAlarmHistorySettings(t *testing.T) {
	service := &AlarmSettingsService{}
	retentionDays := 90
	payload, err := json.Marshal(SaveAlarmHistorySettingsInput{IsEnabled: false, RetentionDays: &retentionDays, StoreNotificationDeliveries: true})
	if err != nil {
		t.Fatal(err)
	}
	operation, err := service.normalizeSyncOperation(nil, "project-1", "actor-1", AlarmConfigSyncOperation{Resource: "history_settings", Action: "upsert", ID: "history", Data: payload})
	if err != nil {
		t.Fatalf("normalize history settings: %v", err)
	}
	if operation.HistorySettings == nil || operation.HistorySettings.IsEnabled || operation.HistorySettings.RetentionDays == nil || *operation.HistorySettings.RetentionDays != 90 {
		t.Fatalf("normalized history settings = %#v", operation.HistorySettings)
	}
}
