package service

import (
	"strings"
	"testing"
)

func TestDevelopmentSQLLimits(t *testing.T) {
	limits := developmentSQLLimits()
	if limits.MaxRows != 500 || limits.MaxBytes != 5*1024*1024 || limits.TimeoutSec != 30 {
		t.Fatalf("unexpected SQL limits: %+v", limits)
	}
	if size := jsonResultSize(map[string]any{"value": strings.Repeat("x", 128)}); size <= 128 {
		t.Fatalf("expected JSON envelope bytes, got %d", size)
	}
	if accepted, reason, _ := admitSQLResultRow(500, 0, 500, developmentSQLMaxBytes, []any{1}); accepted || reason != "rows" {
		t.Fatalf("expected row truncation, accepted=%v reason=%q", accepted, reason)
	}
	if accepted, reason, _ := admitSQLResultRow(0, 0, 500, 8, map[string]any{"value": "too large"}); accepted || reason != "bytes" {
		t.Fatalf("expected byte truncation, accepted=%v reason=%q", accepted, reason)
	}
}
