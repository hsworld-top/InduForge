package repository

import "testing"

func TestCollectorPointPathUsesStableConnectionCode(t *testing.T) {
	if got := collectorPointPath("opc_ua", "lastchange"); got != "collector.opc_ua.lastchange" {
		t.Fatalf("unexpected path: %s", got)
	}
}
