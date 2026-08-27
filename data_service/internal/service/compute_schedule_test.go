package service

import (
	"testing"
	"time"
)

func TestComputeScheduleOccurrencesInterval(t *testing.T) {
	now := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	_, runs := computeScheduleOccurrences(map[string]any{"kind": "interval", "every": 5, "unit": "minutes"}, now, 2)
	if len(runs) != 2 || !runs[0].Equal(now.Add(5*time.Minute)) || !runs[1].Equal(now.Add(10*time.Minute)) {
		t.Fatalf("unexpected interval preview: %#v", runs)
	}
}

func TestNormalizedWeekdaysSortsAndDeduplicates(t *testing.T) {
	got := normalizedWeekdays([]any{float64(7), float64(1), float64(7)})
	if len(got) != 2 || got[0] != 1 || got[1] != 7 {
		t.Fatalf("unexpected weekdays: %#v", got)
	}
}

func TestWeeklyScheduleRejectsInvalidISOWeekday(t *testing.T) {
	if validWeekdaysInput([]any{1, 8}) {
		t.Fatal("weekday 8 must be rejected")
	}
	if !validWeekdaysInput([]any{7, 1, 7}) {
		t.Fatal("valid weekdays including duplicates should be accepted")
	}
}

func TestComputeScheduleOccurrencesSkipsMissingDSTClock(t *testing.T) {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 3, 7, 12, 0, 0, 0, location).UTC()
	_, runs := computeScheduleOccurrences(map[string]any{"kind": "daily", "time": "02:30:00", "timezone": "America/New_York"}, now, 2)
	if len(runs) != 2 {
		t.Fatalf("unexpected runs: %#v", runs)
	}
	if got := runs[0].In(location); got.Day() != 9 || got.Hour() != 2 || got.Minute() != 30 {
		t.Fatalf("missing DST clock should be skipped, got %v", got)
	}
}

func TestComputeScheduleOccurrencesEmitsRepeatedDSTClockOnce(t *testing.T) {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, location).UTC()
	_, runs := computeScheduleOccurrences(map[string]any{"kind": "daily", "time": "01:30:00", "timezone": "America/New_York"}, now, 2)
	if len(runs) != 2 || runs[0].In(location).Day() != 1 || runs[1].In(location).Day() != 2 {
		t.Fatalf("repeated clock must run once per natural day: %#v", runs)
	}
}

func TestComputeScheduleOccurrencesMonthlyLastWeekday(t *testing.T) {
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	_, runs := computeScheduleOccurrences(map[string]any{
		"kind": "monthly", "dayRule": "weekday", "weekOfMonth": -1, "weekday": 1,
		"time": "08:00:00", "timezone": "Asia/Shanghai",
	}, now, 2)
	if len(runs) != 2 {
		t.Fatalf("unexpected monthly runs: %#v", runs)
	}
	location, _ := time.LoadLocation("Asia/Shanghai")
	first := runs[0].In(location)
	if first.Year() != 2026 || first.Month() != time.August || first.Day() != 31 || first.Hour() != 8 {
		t.Fatalf("unexpected first monthly run: %v", first)
	}
}

func TestComputeScheduleOccurrencesYearlySkipsInvalidDay(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, runs := computeScheduleOccurrences(map[string]any{
		"kind": "yearly", "month": 2, "dayRule": "day", "dayOfMonth": 29,
		"time": "00:00:00", "timezone": "UTC",
	}, now, 1)
	if len(runs) != 1 || runs[0].Year() != 2028 || runs[0].Month() != time.February || runs[0].Day() != 29 {
		t.Fatalf("invalid calendar dates must be skipped: %#v", runs)
	}
}

func TestComputeScheduleWindowLimitsPreview(t *testing.T) {
	now := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	_, runs := computeScheduleOccurrences(map[string]any{
		"kind": "interval", "every": 1, "unit": "hours", "maxRuns": 2,
		"startAt": "2026-08-24T03:00:00Z", "endAt": "2026-08-24T04:00:00Z",
	}, now, 5)
	if len(runs) != 2 || !runs[0].Equal(now.Add(3*time.Hour)) || !runs[1].Equal(now.Add(4*time.Hour)) {
		t.Fatalf("unexpected bounded preview: %#v", runs)
	}
}

func TestNormalizeScheduleWindowRejectsInvalidRange(t *testing.T) {
	err := normalizeScheduleWindow(map[string]any{
		"startAt": "2026-08-24T04:00:00Z",
		"endAt":   "2026-08-24T03:00:00Z",
	}, map[string]any{})
	if err == nil {
		t.Fatal("endAt before startAt must be rejected")
	}
}

func TestComputeTriggerScalarTypes(t *testing.T) {
	for _, dataType := range []string{"bool", "string", "int32", "float64", "decimal"} {
		if !isComputeTriggerScalarType(dataType) {
			t.Fatalf("%s must be accepted as trigger scalar", dataType)
		}
	}
	for _, dataType := range []string{"object", "array", "bytes", "datetime"} {
		if isComputeTriggerScalarType(dataType) {
			t.Fatalf("%s must not be accepted as trigger scalar", dataType)
		}
	}
}

func TestComputeTriggerModeMatchesDatapointType(t *testing.T) {
	if !isComputeTriggerModeCompatible("increase", "float64") {
		t.Fatal("numeric increase must be accepted")
	}
	if isComputeTriggerModeCompatible("increase", "string") {
		t.Fatal("string increase must be rejected")
	}
	if !isComputeTriggerModeCompatible("rising_edge", "bool") {
		t.Fatal("bool rising edge must be accepted")
	}
	if isComputeTriggerModeCompatible("rising_edge", "int32") {
		t.Fatal("numeric rising edge must be rejected")
	}
}

func TestConditionExpressionUsesDeclaredAlias(t *testing.T) {
	if !conditionExpressionUsesAlias("temperature > 80 && enabled", "temperature") {
		t.Fatal("declared alias must be detected")
	}
	if conditionExpressionUsesAlias("temperature_backup > 80", "temperature") {
		t.Fatal("alias must match a complete identifier")
	}
}

func TestNormalizeConditionPhasesCanonicalizesSelection(t *testing.T) {
	got, ok := normalizeConditionPhases([]any{"exited", "entered", "active", "entered"})
	if !ok || len(got) != 3 || got[0] != "entered" || got[1] != "active" || got[2] != "exited" {
		t.Fatalf("unexpected phases: %#v, ok=%v", got, ok)
	}
}

func TestNormalizeConditionPhasesRejectsEmptyOrUnknownSelection(t *testing.T) {
	if got, ok := normalizeConditionPhases([]any{}); !ok || len(got) != 0 {
		t.Fatalf("empty selection should normalize to an empty list for caller validation: %#v, ok=%v", got, ok)
	}
	if _, ok := normalizeConditionPhases([]any{"entered", "unknown"}); ok {
		t.Fatal("unknown phase must be rejected")
	}
}
