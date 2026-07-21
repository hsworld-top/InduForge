package repository

import (
	"encoding/json"
	"testing"
)

func TestBuildCollectorPointSnapshotAttemptsKeepsPerPointResults(t *testing.T) {
	firstID := "550e8400-e29b-41d4-a716-446655440101"
	secondID := "550e8400-e29b-41d4-a716-446655440102"
	task := CollectorDevTaskRecord{RequestPayload: json.RawMessage(`{"pointIds":["` + firstID + `","` + secondID + `"]}`)}
	result := []byte(`{"values":[{"pointId":"` + firstID + `","succeeded":true,"value":"running","dataType":"string","quality":"BadOutOfService","sourceTimestamp":"2026-07-20T01:02:03Z","serverTimestamp":"2026-07-20T01:02:04Z"},{"pointId":"` + secondID + `","succeeded":false,"errorCode":"READ_FAILED","errorMessage":"读取失败"}]}`)

	attempts, err := buildCollectorPointSnapshotAttempts(task, "succeeded", result, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 2 {
		t.Fatalf("attempt count = %d, want 2", len(attempts))
	}
	if !attempts[0].Succeeded || attempts[0].ValueText == nil || *attempts[0].ValueText != "running" {
		t.Fatalf("unexpected success attempt: %#v", attempts[0])
	}
	if attempts[0].Quality == nil || *attempts[0].Quality != "BadOutOfService" {
		t.Fatalf("unexpected quality: %#v", attempts[0].Quality)
	}
	if attempts[1].Succeeded || attempts[1].ErrorCode == nil || *attempts[1].ErrorCode != "READ_FAILED" {
		t.Fatalf("unexpected failed attempt: %#v", attempts[1])
	}
}

func TestBuildCollectorPointSnapshotAttemptsDropsMissingProtocolTimestamp(t *testing.T) {
	pointID := "550e8400-e29b-41d4-a716-446655440101"
	task := CollectorDevTaskRecord{RequestPayload: json.RawMessage(`{"pointIds":["` + pointID + `"]}`)}
	result := []byte(`{"values":[{"pointId":"` + pointID + `","succeeded":true,"value":1,"sourceTimestamp":"0001-01-01T00:00:00Z","serverTimestamp":"0001-01-01T00:00:00Z"}]}`)

	attempts, err := buildCollectorPointSnapshotAttempts(task, "succeeded", result, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if attempts[0].SourceTimestamp != nil || attempts[0].ServerTimestamp != nil {
		t.Fatalf("missing protocol timestamp should be nil: %#v", attempts[0])
	}
}

func TestBuildCollectorPointSnapshotAttemptsMarksMissingResult(t *testing.T) {
	pointID := "550e8400-e29b-41d4-a716-446655440101"
	task := CollectorDevTaskRecord{RequestPayload: json.RawMessage(`{"pointIds":["` + pointID + `"]}`)}

	attempts, err := buildCollectorPointSnapshotAttempts(task, "succeeded", []byte(`{"values":[]}`), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 1 || attempts[0].ErrorCode == nil || *attempts[0].ErrorCode != "COLLECTOR_POINT_RESULT_MISSING" {
		t.Fatalf("unexpected attempts: %#v", attempts)
	}
}

func TestBuildCollectorPointSnapshotAttemptsMarksWholeTaskFailure(t *testing.T) {
	pointID := "550e8400-e29b-41d4-a716-446655440101"
	task := CollectorDevTaskRecord{RequestPayload: json.RawMessage(`{"pointIds":["` + pointID + `"]}`)}

	attempts, err := buildCollectorPointSnapshotAttempts(task, "failed", nil, "COLLECTOR_SESSION_LOST", "连接已断开")
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 1 || attempts[0].Succeeded || attempts[0].ErrorMessage == nil || *attempts[0].ErrorMessage != "连接已断开" {
		t.Fatalf("unexpected attempts: %#v", attempts)
	}
}

func TestBuildCollectorPointSnapshotAttemptsRejectsUnknownPoint(t *testing.T) {
	pointID := "550e8400-e29b-41d4-a716-446655440101"
	task := CollectorDevTaskRecord{RequestPayload: json.RawMessage(`{"pointIds":["` + pointID + `"]}`)}
	result := []byte(`{"values":[{"pointId":"550e8400-e29b-41d4-a716-446655440199","succeeded":true,"value":1}]}`)

	if _, err := buildCollectorPointSnapshotAttempts(task, "succeeded", result, "", ""); err == nil {
		t.Fatal("expected unknown pointId to be rejected")
	}
}
