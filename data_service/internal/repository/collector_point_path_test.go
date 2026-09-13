package repository

import "testing"

func TestCollectorPointPathUsesStableConnectionID(t *testing.T) {
	if got := collectorPointPath("8a3d2f54-0b9f-4f0c-9bb4-4c9d0b4a3d10", "lastchange"); got != "collector.8a3d2f54-0b9f-4f0c-9bb4-4c9d0b4a3d10.lastchange" {
		t.Fatalf("unexpected path: %s", got)
	}
}
