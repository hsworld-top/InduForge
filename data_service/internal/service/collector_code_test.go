package service

import "testing"

func TestCollectorCodeFromNameUsesStableReadableSegments(t *testing.T) {
	if got := collectorCodeFromName("1号产线 OPC UA", "connection"); got != "1号产线_opc_ua" {
		t.Fatalf("unexpected code: %s", got)
	}
}

func TestAllocateCollectorPointCodeResolvesConnectionLocalConflicts(t *testing.T) {
	used := map[string]struct{}{"lastchange": {}}
	if got := allocateCollectorPointCode("LastChange", used); got != "lastchange_2" {
		t.Fatalf("unexpected code: %s", got)
	}
}
