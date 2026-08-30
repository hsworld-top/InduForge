package compute

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/indu-forge/runtime-engine/internal/model"
)

// NextSchedule returns the first occurrence strictly after after.  Wall-clock
// schedules use a local occurrence key; callers persist it to ensure a DST
// fall-back repeated clock hour executes once.  Spring-forward nonexistent
// local times are skipped rather than silently shifted.
func NextSchedule(schedule model.Schedule, after time.Time) (time.Time, string, error) {
	after = after.UTC()
	start, end, err := scheduleWindow(schedule)
	if err != nil {
		return time.Time{}, "", err
	}
	if !end.IsZero() && !after.Before(end) {
		return time.Time{}, "", nil
	}
	if schedule.Kind == "interval" {
		if schedule.Every == nil || schedule.Unit == nil {
			return time.Time{}, "", errors.New("interval schedule 非法")
		}
		d := intervalDuration(*schedule.Every, *schedule.Unit)
		if d <= 0 {
			return time.Time{}, "", errors.New("interval schedule 非法")
		}
		if start.IsZero() {
			return time.Time{}, "", errors.New("interval schedule 需要已持久化 activation anchor")
		}
		anchor := start
		next := anchor
		if !after.Before(anchor) {
			steps := after.Sub(anchor)/d + 1
			next = anchor.Add(steps * d)
		}
		if !end.IsZero() && !next.Before(end) {
			return time.Time{}, "", nil
		}
		return next, next.Format(time.RFC3339Nano), nil
	}
	if schedule.Timezone == nil || schedule.Time == nil {
		return time.Time{}, "", errors.New("calendar schedule 非法")
	}
	loc, err := time.LoadLocation(*schedule.Timezone)
	if err != nil {
		return time.Time{}, "", errors.New("timezone 非法")
	}
	hour, minute, second, err := parseClock(*schedule.Time)
	if err != nil {
		return time.Time{}, "", err
	}
	searchFrom := after
	if !start.IsZero() && start.After(searchFrom) {
		searchFrom = start
	}
	date := searchFrom.In(loc).AddDate(0, 0, -1)
	// 12 years covers every valid yearly occurrence even around leap days;
	// invalid dates are simply not candidates.
	for day := 0; day < 5000; day++ {
		current := date.AddDate(0, 0, day)
		if !matchesScheduleDate(schedule, current) {
			continue
		}
		candidate := time.Date(current.Year(), current.Month(), current.Day(), hour, minute, second, 0, loc)
		// time.Date normalizes a nonexistent local time.  Do not run at a
		// different wall time, which would make DST behavior surprising.
		if candidate.In(loc).Year() != current.Year() || candidate.In(loc).Month() != current.Month() || candidate.In(loc).Day() != current.Day() || candidate.In(loc).Hour() != hour || candidate.In(loc).Minute() != minute || candidate.In(loc).Second() != second {
			continue
		}
		candidate = candidate.UTC()
		if !candidate.After(after) || (!start.IsZero() && candidate.Before(start)) {
			continue
		}
		if !end.IsZero() && !candidate.Before(end) {
			return time.Time{}, "", nil
		}
		localKey := fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d@%s", current.Year(), current.Month(), current.Day(), hour, minute, second, *schedule.Timezone)
		return candidate, localKey, nil
	}
	return time.Time{}, "", nil
}

// InitialScheduleOccurrence is called exactly once while creating durable
// schedule state. An interval without startAt is anchored to activationAt;
// later restarts must use the persisted next_run_at, never recompute an epoch.
func InitialScheduleOccurrence(schedule model.Schedule, activationAt time.Time) (time.Time, string, error) {
	if schedule.Kind != "interval" || schedule.StartAt != nil {
		return NextSchedule(schedule, activationAt.Add(-time.Nanosecond))
	}
	if schedule.Every == nil || schedule.Unit == nil || intervalDuration(*schedule.Every, *schedule.Unit) <= 0 {
		return time.Time{}, "", errors.New("interval schedule 非法")
	}
	next := activationAt.UTC()
	return next, next.Format(time.RFC3339Nano), nil
}

func scheduleWindow(schedule model.Schedule) (time.Time, time.Time, error) {
	var start, end time.Time
	var err error
	if schedule.StartAt != nil {
		start, err = time.Parse(time.RFC3339Nano, *schedule.StartAt)
		if err != nil || !strings.HasSuffix(*schedule.StartAt, "Z") {
			return time.Time{}, time.Time{}, errors.New("schedule startAt 非法")
		}
	}
	if schedule.EndAt != nil {
		end, err = time.Parse(time.RFC3339Nano, *schedule.EndAt)
		if err != nil || !strings.HasSuffix(*schedule.EndAt, "Z") {
			return time.Time{}, time.Time{}, errors.New("schedule endAt 非法")
		}
	}
	if !start.IsZero() && !end.IsZero() && !end.After(start) {
		return time.Time{}, time.Time{}, errors.New("schedule window 非法")
	}
	return start.UTC(), end.UTC(), nil
}
func intervalDuration(every int64, unit string) time.Duration {
	if every < 1 {
		return 0
	}
	switch unit {
	case "seconds":
		if every > int64((1<<63-1)/int64(time.Second)) {
			return 0
		}
		return time.Duration(every) * time.Second
	case "minutes":
		if every > int64((1<<63-1)/int64(time.Minute)) {
			return 0
		}
		return time.Duration(every) * time.Minute
	case "hours":
		if every > int64((1<<63-1)/int64(time.Hour)) {
			return 0
		}
		return time.Duration(every) * time.Hour
	}
	return 0
}
func parseClock(value string) (int, int, int, error) {
	if len(value) != 8 || value[2] != ':' || value[5] != ':' {
		return 0, 0, 0, errors.New("schedule time 非法")
	}
	toInt := func(a, b byte) int {
		if a < '0' || a > '9' || b < '0' || b > '9' {
			return -1
		}
		return int(a-'0')*10 + int(b-'0')
	}
	h, m, s := toInt(value[0], value[1]), toInt(value[3], value[4]), toInt(value[6], value[7])
	if h < 0 || h > 23 || m < 0 || m > 59 || s < 0 || s > 59 {
		return 0, 0, 0, errors.New("schedule time 非法")
	}
	return h, m, s, nil
}
func matchesScheduleDate(schedule model.Schedule, day time.Time) bool {
	switch schedule.Kind {
	case "daily":
		return true
	case "weekly":
		for _, w := range schedule.Weekdays {
			if w < 1 || w > 7 {
				return false
			}
			if int(day.Weekday()) == w%7 {
				return true
			}
		}
		return false
	case "monthly":
		return matchesMonthly(schedule, day)
	case "yearly":
		return schedule.Month != nil && *schedule.Month >= 1 && *schedule.Month <= 12 && int(day.Month()) == *schedule.Month && matchesMonthly(schedule, day)
	}
	return false
}
func matchesMonthly(schedule model.Schedule, day time.Time) bool {
	if schedule.DayRule == nil {
		return false
	}
	if *schedule.DayRule == "day" {
		return schedule.DayOfMonth != nil && *schedule.DayOfMonth >= 1 && *schedule.DayOfMonth <= 31 && day.Day() == *schedule.DayOfMonth
	}
	if *schedule.DayRule != "weekday" || schedule.Weekday == nil || *schedule.Weekday < 1 || *schedule.Weekday > 7 || schedule.WeekOfMonth == nil || (*schedule.WeekOfMonth != -1 && (*schedule.WeekOfMonth < 1 || *schedule.WeekOfMonth > 5)) || int(day.Weekday()) != *schedule.Weekday%7 {
		return false
	}
	if *schedule.WeekOfMonth == -1 {
		return day.AddDate(0, 0, 7).Month() != day.Month()
	}
	return (day.Day()-1)/7+1 == *schedule.WeekOfMonth
}
