package repository

import "testing"

func TestDesiredQueryOutputPathReflectsExtractionLevel(t *testing.T) {
	owner := sourceOutputOwner{Kind: "query", PathPrefix: "db.IF关系库.test1"}

	whole := desiredSourceOutputPath(owner, SourceOutputMappingParam{
		Key:      "result",
		Selector: SourceOutputSelectorRecord{Kind: "whole"},
	})
	if whole != "db.IF关系库.test1" {
		t.Fatalf("whole query dataset path mismatch: %q", whole)
	}

	column := desiredSourceOutputPath(owner, SourceOutputMappingParam{
		Key:      "id",
		Selector: SourceOutputSelectorRecord{Kind: "column", Column: stringPointer("id")},
	})
	if column != "db.IF关系库.test1.id" {
		t.Fatalf("query column path mismatch: %q", column)
	}
}

func TestDesiredRealtimeOutputPathOmitsWholeValueSuffix(t *testing.T) {
	owner := sourceOutputOwner{Kind: "realtime", PathPrefix: "redis.生产Redis.device.line1.status"}

	whole := desiredSourceOutputPath(owner, SourceOutputMappingParam{
		Key:      "value",
		Selector: SourceOutputSelectorRecord{Kind: "whole"},
	})
	if whole != "redis.生产Redis.device.line1.status" {
		t.Fatalf("whole realtime value path mismatch: %q", whole)
	}

	field := desiredSourceOutputPath(owner, SourceOutputMappingParam{
		Key:      "temperature",
		Selector: SourceOutputSelectorRecord{Kind: "path", Segments: []any{"temperature"}},
	})
	if field != "redis.生产Redis.device.line1.status.temperature" {
		t.Fatalf("realtime field path mismatch: %q", field)
	}
}

func stringPointer(value string) *string { return &value }
