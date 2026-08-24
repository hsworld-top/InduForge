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
