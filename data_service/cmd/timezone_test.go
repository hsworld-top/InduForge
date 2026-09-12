package main

import (
	"testing"
	"time"
)

// 此用例也在无系统 zoneinfo、无 Go 安装目录的 scratch 容器中执行。
func TestEmbeddedTimezone(t *testing.T) {
	for _, tc := range []struct {
		name   string
		month  time.Month
		offset int
	}{
		{"Asia/Shanghai", time.January, 8 * 3600},
		{"America/New_York", time.January, -5 * 3600},
		{"America/New_York", time.July, -4 * 3600},
	} {
		loc, err := time.LoadLocation(tc.name)
		if err != nil {
			t.Fatalf("load %s: %v", tc.name, err)
		}
		_, offset := time.Date(2026, tc.month, 1, 12, 0, 0, 0, loc).Zone()
		if offset != tc.offset {
			t.Fatalf("%s offset = %d, want %d", tc.name, offset, tc.offset)
		}
	}
	if _, err := time.LoadLocation("Invalid/Timezone"); err == nil {
		t.Fatal("invalid timezone accepted")
	}
}
