package service

import (
	"reflect"
	"testing"
	"time"

	"github.com/indu-forge/data_service/internal/repository"
)

func TestExtractMqttJSONPathValueRootArray(t *testing.T) {
	tag := repository.MqttTagRecord{
		DataType:  "array",
		ParseType: "jsonpath",
		ParseRule: "$",
	}

	value, err := extractMqttTagValue(tag, `[{"N":"temperature","V":23.5,"Q":192}]`)
	if err != nil {
		t.Fatalf("extractMqttTagValue() error = %v", err)
	}
	expected := []any{map[string]any{"N": "temperature", "V": float64(23.5), "Q": float64(192)}}
	if !reflect.DeepEqual(value, expected) {
		t.Fatalf("extractMqttTagValue() = %#v, want %#v", value, expected)
	}
}

func TestExtractMqttBatchJSONPathValueRootArray(t *testing.T) {
	tag := repository.MqttTagRecord{
		DataType:  "number",
		ParseType: "batch_jsonpath",
		ParseRule: `{"arrayPath":"$","namePath":"N","matchName":"temperature","valuePath":"V","qualityPath":"Q","timePath":"T"}`,
	}

	value, err := extractMqttTagValue(tag, `[{"N":"temperature","V":23.5,"Q":192,"T":"2026-06-02 10:00:00"}]`)
	if err != nil {
		t.Fatalf("extractMqttTagValue() error = %v", err)
	}
	if value != float64(23.5) {
		t.Fatalf("extractMqttTagValue() = %v, want 23.5", value)
	}
}

func TestBuildMqttTagSnapshotFromBatchJSONPathKeepsBadQualityValue(t *testing.T) {
	tag := repository.MqttTagRecord{
		ID:             "tag-1",
		SubscriptionID: "sub-1",
		DataType:       "number",
		ParseType:      "batch_jsonpath",
		ParseRule:      `{"arrayPath":"$","namePath":"N","matchName":"temperature","valuePath":"V","qualityPath":"Q"}`,
	}
	message := repository.MqttMessageRecord{
		SubscriptionID: "sub-1",
		Topic:          "device/demo",
		Payload:        `[{"N":"temperature","V":23.5,"Q":28}]`,
		ReceivedAt:     testTime(),
	}

	snapshot := BuildMqttTagSnapshotFromMessage(tag, message)
	if snapshot.Quality != "bad" {
		t.Fatalf("snapshot.Quality = %q, want bad", snapshot.Quality)
	}
	if snapshot.QualityCode != float64(28) {
		t.Fatalf("snapshot.QualityCode = %v, want 28", snapshot.QualityCode)
	}
	if snapshot.ParsedValue != float64(23.5) {
		t.Fatalf("snapshot.ParsedValue = %v, want 23.5", snapshot.ParsedValue)
	}
}

func TestBuildMqttTagSnapshotUpdateFromBatchJSONPathSkipsMissingName(t *testing.T) {
	tag := repository.MqttTagRecord{
		ID:             "tag-982-id",
		SubscriptionID: "sub-1",
		DataType:       "number",
		ParseType:      "batch_jsonpath",
		ParseRule:      `{"arrayPath":"$","namePath":"N","matchName":"tag982","valuePath":"V","qualityPath":"Q"}`,
	}
	message := repository.MqttMessageRecord{
		SubscriptionID: "sub-1",
		Topic:          "device/demo",
		Payload:        `[{"N":"tag6","V":51,"Q":192},{"N":"tag7","V":154587.47,"Q":192}]`,
		ReceivedAt:     testTime(),
	}

	_, ok := BuildMqttTagSnapshotUpdateFromMessage(tag, message)
	if ok {
		t.Fatal("expected missing batch variable to be skipped")
	}
}

func TestBuildMqttTagSnapshotUpdateFromBatchJSONPathSkipsMissingArrayPath(t *testing.T) {
	tag := repository.MqttTagRecord{
		ID:             "nested-tag-id",
		SubscriptionID: "sub-1",
		DataType:       "number",
		ParseType:      "batch_jsonpath",
		ParseRule:      `{"arrayPath":"$.data.data","namePath":"N","matchName":"temperature","valuePath":"V","qualityPath":"Q"}`,
	}
	message := repository.MqttMessageRecord{
		SubscriptionID: "sub-1",
		Topic:          "device/demo",
		Payload:        `[{"N":"temperature","V":23.5,"Q":192}]`,
		ReceivedAt:     testTime(),
	}

	_, ok := BuildMqttTagSnapshotUpdateFromMessage(tag, message)
	if ok {
		t.Fatal("expected message with another batch array path to be skipped")
	}
}

func TestBuildMqttTagSnapshotUpdateFromBatchJSONPathSkipsNonArrayPath(t *testing.T) {
	tag := repository.MqttTagRecord{
		ID:             "root-tag-id",
		SubscriptionID: "sub-1",
		DataType:       "number",
		ParseType:      "batch_jsonpath",
		ParseRule:      `{"arrayPath":"$","namePath":"N","matchName":"temperature","valuePath":"V","qualityPath":"Q"}`,
	}
	message := repository.MqttMessageRecord{
		SubscriptionID: "sub-1",
		Topic:          "device/demo",
		Payload:        `{"data":{"data":[{"N":"temperature","V":23.5,"Q":192}]}}`,
		ReceivedAt:     testTime(),
	}

	_, ok := BuildMqttTagSnapshotUpdateFromMessage(tag, message)
	if ok {
		t.Fatal("expected message with incompatible batch array path to be skipped")
	}
}

func TestExtractMqttBatchJSONPathValueNestedArray(t *testing.T) {
	tag := repository.MqttTagRecord{
		DataType:  "string",
		ParseType: "batch_jsonpath",
		ParseRule: `{"arrayPath":"$.items","namePath":"name","matchName":"runState","valuePath":"value"}`,
	}

	value, err := extractMqttTagValue(tag, `{"device":"motor-1","items":[{"name":"runState","value":"running"}]}`)
	if err != nil {
		t.Fatalf("extractMqttTagValue() error = %v", err)
	}
	if value != "running" {
		t.Fatalf("extractMqttTagValue() = %v, want running", value)
	}
}

func testTime() time.Time {
	return time.Date(2026, 6, 3, 10, 0, 0, 0, time.UTC)
}
