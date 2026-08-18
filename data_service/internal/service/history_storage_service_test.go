package service

import "testing"

func TestValidateHistoryStorageConfigurationFields(t *testing.T) {
	interval := int64(60000)
	zeroInterval := int64(0)
	deadband := 0.5
	negativeDeadband := -0.1
	maxSilence := int64(3600000)
	zeroSilence := int64(0)
	tests := []struct {
		name    string
		input   HistoryStorageConfigurationInput
		wantErr bool
	}{
		{name: "every sample", input: HistoryStorageConfigurationInput{WriteMode: "every_sample", OfflineBehavior: "store_stale"}},
		{name: "interval latest", input: HistoryStorageConfigurationInput{WriteMode: "interval_latest", IntervalMS: &interval, OfflineBehavior: "store_stale"}},
		{name: "on change", input: HistoryStorageConfigurationInput{WriteMode: "on_change", Deadband: &deadband, MaxSilenceMS: &maxSilence, OfflineBehavior: "store_stale"}},
		{name: "periodic", input: HistoryStorageConfigurationInput{WriteMode: "periodic_snapshot", IntervalMS: &interval, OfflineBehavior: "skip"}},
		{name: "interval missing", input: HistoryStorageConfigurationInput{WriteMode: "interval_latest", OfflineBehavior: "store_stale"}, wantErr: true},
		{name: "interval zero", input: HistoryStorageConfigurationInput{WriteMode: "periodic_snapshot", IntervalMS: &zeroInterval}, wantErr: true},
		{name: "deadband on every sample", input: HistoryStorageConfigurationInput{WriteMode: "every_sample", Deadband: &deadband, OfflineBehavior: "store_stale"}, wantErr: true},
		{name: "interval on change", input: HistoryStorageConfigurationInput{WriteMode: "on_change", IntervalMS: &interval}, wantErr: true},
		{name: "negative deadband", input: HistoryStorageConfigurationInput{WriteMode: "on_change", Deadband: &negativeDeadband}, wantErr: true},
		{name: "zero max silence", input: HistoryStorageConfigurationInput{WriteMode: "on_change", MaxSilenceMS: &zeroSilence}, wantErr: true},
		{name: "offline behavior on interval", input: HistoryStorageConfigurationInput{WriteMode: "interval_latest", IntervalMS: &interval, OfflineBehavior: "skip"}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := validateHistoryStorageConfigurationFields(&test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr=%v", err, test.wantErr)
			}
		})
	}
}

func TestValidateHistoryStorageConfigurationDefaultsOfflineBehavior(t *testing.T) {
	mode, offline, err := validateHistoryStorageConfigurationFields(&HistoryStorageConfigurationInput{WriteMode: "on_change"})
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if mode != "on_change" || offline != "store_stale" {
		t.Fatalf("mode=%q offline=%q", mode, offline)
	}
}

func TestNormalizeHistoryStorageTargetInputs(t *testing.T) {
	permanent := HistoryStorageTargetInput{ConnectionID: "00000000-0000-0000-0000-000000000001", IsPrimary: true, SortOrder: 9, RetentionDays: nil}
	days := int64(30)
	secondary := HistoryStorageTargetInput{ConnectionID: "00000000-0000-0000-0000-000000000002", SortOrder: 3, RetentionDays: &days}
	ids, targets, err := normalizeHistoryStorageTargetInputs([]HistoryStorageTargetInput{permanent, secondary})
	if err != nil {
		t.Fatalf("normalize failed: %v", err)
	}
	if len(ids) != 2 || len(targets) != 2 {
		t.Fatalf("ids=%#v targets=%#v", ids, targets)
	}
	if targets[0].RetentionDays != nil || targets[0].SortOrder != 0 || targets[1].SortOrder != 1 {
		t.Fatalf("目标顺序或永久保留语义错误: %#v", targets)
	}
}

func TestNormalizeHistoryStorageTargetInputsRejectsInvalidValues(t *testing.T) {
	days := int64(30)
	zero := int64(0)
	primary := HistoryStorageTargetInput{ConnectionID: "00000000-0000-0000-0000-000000000001", IsPrimary: true, RetentionDays: &days}
	tests := []struct {
		name   string
		values []HistoryStorageTargetInput
	}{
		{name: "empty"},
		{name: "invalid id", values: []HistoryStorageTargetInput{{ConnectionID: "invalid", IsPrimary: true}}},
		{name: "no primary", values: []HistoryStorageTargetInput{{ConnectionID: primary.ConnectionID}}},
		{name: "duplicate", values: []HistoryStorageTargetInput{primary, primary}},
		{name: "invalid retention", values: []HistoryStorageTargetInput{{ConnectionID: primary.ConnectionID, IsPrimary: true, RetentionDays: &zero}}},
		{name: "multiple primary", values: []HistoryStorageTargetInput{primary, {ConnectionID: "00000000-0000-0000-0000-000000000002", IsPrimary: true}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := normalizeHistoryStorageTargetInputs(test.values); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNormalizeHistoryStorageScope(t *testing.T) {
	if _, err := normalizeHistoryStorageScope("access_source", "not-a-uuid", false); err == nil {
		t.Fatal("expected invalid id error")
	}
	if _, err := normalizeHistoryStorageScope("datapoint", "00000000-0000-0000-0000-000000000001", false); err == nil {
		t.Fatal("expected datapoint scope rejection")
	}
	if scope, err := normalizeHistoryStorageScope("collector_connection", "00000000-0000-0000-0000-000000000001", false); err != nil || scope.Type != "collector_connection" {
		t.Fatalf("scope=%#v err=%v", scope, err)
	}
}
