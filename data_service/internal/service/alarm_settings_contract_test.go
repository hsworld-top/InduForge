package service

import "testing"

func TestDefaultAlarmNotificationChannelUsesCanonicalTestStatus(t *testing.T) {
	channel := defaultAlarmNotificationChannel("project-1")
	if channel.LastTestStatus != "not_tested" {
		t.Fatalf("LastTestStatus = %q, want not_tested", channel.LastTestStatus)
	}
	if channel.Config == nil || channel.SecretStatus == nil {
		t.Fatalf("built-in channel maps must not be nil: %#v", channel)
	}
}
