package service

import "testing"

func TestMqttSavedConnectionSupportsSourceTest(t *testing.T) {
	if got := connectionTestCapability("mqtt"); got.Status != "supported" {
		t.Fatalf("MQTT saved testing unavailable: %+v", got)
	}
	for _, kind := range []string{"builtin.message", "http", "websocket"} {
		if got := connectionTestCapability(kind); got.Status != "unsupported" {
			t.Fatalf("%s unexpectedly enabled", kind)
		}
	}
}
