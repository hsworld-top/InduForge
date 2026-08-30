package compute

import (
	"testing"
	"time"

	"github.com/indu-forge/runtime-engine/internal/model"
)

func TestScheduleDSTAndIntervalAnchor(t *testing.T) {
	zone, clock := "America/New_York", "01:30:00"
	daily := model.Schedule{Kind: "daily", Timezone: &zone, Time: &clock}
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	first, key, err := NextSchedule(daily, from)
	if err != nil || first.IsZero() || key == "" {
		t.Fatalf("dst next: %v %v", first, err)
	}
	second, key2, err := NextSchedule(daily, first)
	if err != nil || key2 == key || !second.After(first) {
		t.Fatalf("fall-back duplicate: %v %q %q %v", second, key, key2, err)
	}
	every := int64(5)
	unit := "minutes"
	interval := model.Schedule{Kind: "interval", Every: &every, Unit: &unit}
	if _, _, err := NextSchedule(interval, from); err == nil {
		t.Fatal("interval must require persisted anchor")
	}
	initial, _, err := InitialScheduleOccurrence(interval, from)
	if err != nil || !initial.Equal(from) {
		t.Fatalf("activation anchor: %v %v", initial, err)
	}
}
