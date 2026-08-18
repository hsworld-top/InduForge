package service

import "testing"

func TestNormalizeConditionsSupportsDevelopmentConditionKinds(t *testing.T) {
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
		{name: "rate", category: "number", condition: AlarmCondition{Kind: "rate_of_change", Operator: "gt", Severity: "major", Params: map[string]any{"limit": 2.0, "windowMs": 1000.0}}},
		{name: "deviation", category: "number", condition: AlarmCondition{Kind: "deviation", Operator: "gt", Severity: "warning", Params: map[string]any{"baseline": 10.0, "limit": 2.0}}},
		{name: "offline", category: "number", condition: AlarmCondition{Kind: "offline", Operator: "is_offline", Severity: "critical", Params: map[string]any{}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conditions, err := normalizeConditions([]AlarmCondition{test.condition}, test.category, "per_target")
			if err != nil {
				t.Fatalf("normalizeConditions() error = %v", err)
			}
			if conditions[0].ID == "" || conditions[0].Label == "" {
				t.Fatalf("condition defaults not generated: %#v", conditions[0])
			}
		})
	}
}

func TestNormalizeConditionsRejectsInvalidDelayDeadbandAndSeverity(t *testing.T) {
	tests := []AlarmCondition{
		{Kind: "threshold", Operator: "gt", Severity: "unknown", Params: map[string]any{"threshold": 1.0}},
		{Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 1.0}, TriggerDelayMS: -1},
		{Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 1.0}, Deadband: -1},
		{Kind: "range", Operator: "between", Severity: "warning", Params: map[string]any{"lower": 2.0, "upper": 1.0}},
	}
	for _, condition := range tests {
		if _, err := normalizeConditions([]AlarmCondition{condition}, "number", "per_target"); err == nil {
			t.Fatalf("normalizeConditions(%#v) error = nil", condition)
		}
	}
}

func TestEvaluateAlarmConditionsPausesInvalidQualityButKeepsOfflineIndependent(t *testing.T) {
	conditions := []AlarmCondition{
		{ID: "threshold", Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 10.0}},
		{ID: "offline", Kind: "offline", Operator: "is_offline", Severity: "critical", Params: map[string]any{}},
	}
	result := evaluateAlarmConditions(conditions, 20.0, map[string]any{"quality": "bad", "offline": true})
	if !result.Triggered || len(result.TriggeredConditions) != 1 || result.TriggeredConditions[0].Kind != "offline" {
		t.Fatalf("result = %#v, want only offline condition triggered", result)
	}
	if result.ConditionResults[0]["reason"] != "quality_paused" {
		t.Fatalf("threshold reason = %v", result.ConditionResults[0]["reason"])
	}
}

func TestNormalizeNotificationShape(t *testing.T) {
	inherit, err := normalizeNotificationShape(AlarmNotificationSettings{})
	if err != nil || inherit.Mode != "inherit" {
		t.Fatalf("inherit = %#v, err = %v", inherit, err)
	}
	off, err := normalizeNotificationShape(AlarmNotificationSettings{Mode: "off"})
	if err != nil || off.Mode != "off" {
		t.Fatalf("off = %#v, err = %v", off, err)
	}
	raise, clear := true, true
	repeat := 300
	custom, err := normalizeNotificationShape(AlarmNotificationSettings{Mode: "custom", NotifyOnRaise: &raise, NotifyOnClear: &clear, RepeatIntervalSeconds: &repeat, ChannelIDs: []string{"runtime_inapp", "runtime_inapp"}})
	if err != nil || len(custom.ChannelIDs) != 1 {
		t.Fatalf("custom = %#v, err = %v", custom, err)
	}
	invalidRepeat := 0
	if _, err = normalizeNotificationShape(AlarmNotificationSettings{Mode: "custom", NotifyOnRaise: &raise, NotifyOnClear: &clear, RepeatIntervalSeconds: &invalidRepeat, ChannelIDs: []string{"runtime_inapp"}}); err == nil {
		t.Fatal("zero repeat interval should be rejected")
	}
}

func TestBuildAlarmContractUsesFinalSchemaAndModeDefault(t *testing.T) {
	ordinary := SaveAlarmPolicyInput{Name: "温度高报警", Mode: "per_target", Bindings: []AlarmBinding{{DatapointID: "dp-1", Role: "target"}}, Conditions: []AlarmCondition{{Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 80.0}}}, Notification: AlarmNotificationSettings{Mode: "inherit"}}
	contract := buildAlarmContract("project-1", "policy-1", ordinary)
	if contract["schemaVersion"] != "alarm.policy.v1" || contract["isEnabled"] != true {
		t.Fatalf("ordinary contract = %#v", contract)
	}
	derived := ordinary
	derived.Mode = "derived"
	derived.DerivedExpression = "a && b"
	contract = buildAlarmContract("project-1", "policy-2", derived)
	if contract["isEnabled"] != false {
		t.Fatalf("derived default enabled = %v, want false", contract["isEnabled"])
	}
	lifecycle := contract["lifecycle"].(map[string]any)
	if lifecycle["autoClear"] != true || lifecycle["acknowledgement"] != "audit_only" {
		t.Fatalf("lifecycle = %#v", lifecycle)
	}
}

func TestBuildAlarmContractKeepsAssignedPolicyID(t *testing.T) {
	input := SaveAlarmPolicyInput{
		Name:       "温度报警",
		Mode:       "per_target",
		Bindings:   []AlarmBinding{{DatapointID: "dp-1", Role: "target"}},
		Conditions: []AlarmCondition{{Kind: "threshold", Operator: "gt", Severity: "warning", Params: map[string]any{"threshold": 80.0}}},
	}
	contract := buildAlarmContract("project-1", "policy-1", input)
	if contract["policyId"] != "policy-1" {
		t.Fatalf("contract policyId = %v, want policy-1", contract["policyId"])
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
