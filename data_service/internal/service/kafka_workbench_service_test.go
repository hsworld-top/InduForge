package service

import "testing"

func TestNormalizeKafkaTopicMappingRequiresOffsetForOffsetStart(t *testing.T) {
	partition := 0
	_, err := normalizeKafkaTopicMappingRuntimeFields("offset", "single", &partition, nil)
	if err == nil {
		t.Fatal("expected offset start without startOffset to fail")
	}
}

func TestNormalizeKafkaTopicMappingRequiresSinglePartitionForOffsetStart(t *testing.T) {
	offset := int64(42)
	_, err := normalizeKafkaTopicMappingRuntimeFields("offset", "all", nil, &offset)
	if err == nil {
		t.Fatal("expected offset start without single partition to fail")
	}
}

func TestNormalizeKafkaTopicMappingAcceptsOffsetWithSinglePartition(t *testing.T) {
	partition := 0
	offset := int64(42)
	result, err := normalizeKafkaTopicMappingRuntimeFields("offset", "single", &partition, &offset)
	if err != nil {
		t.Fatalf("expected valid offset config: %v", err)
	}
	if result.StartOffset == nil || *result.StartOffset != 42 {
		t.Fatalf("unexpected start offset: %#v", result.StartOffset)
	}
}

func TestNormalizeKafkaTopicMappingClearsOffsetForNonOffsetStart(t *testing.T) {
	partition := 0
	offset := int64(42)
	result, err := normalizeKafkaTopicMappingRuntimeFields("latest", "single", &partition, &offset)
	if err != nil {
		t.Fatalf("expected latest config to ignore offset: %v", err)
	}
	if result.StartOffset != nil {
		t.Fatalf("expected start offset to be cleared, got %#v", result.StartOffset)
	}
}

func TestNormalizeKafkaOutputModeDefaultsToFieldMapping(t *testing.T) {
	mode, scope, err := normalizeKafkaOutputConfig("", "")
	if err != nil {
		t.Fatalf("expected default output config: %v", err)
	}
	if mode != "field_mapping" || scope != "value" {
		t.Fatalf("unexpected output config: %s %s", mode, scope)
	}
}

func TestNormalizeKafkaOutputModeAcceptsRawMessage(t *testing.T) {
	mode, scope, err := normalizeKafkaOutputConfig("raw_message", "full_message")
	if err != nil {
		t.Fatalf("expected raw output config: %v", err)
	}
	if mode != "raw_message" || scope != "full_message" {
		t.Fatalf("unexpected raw output config: %s %s", mode, scope)
	}
}

func TestNormalizeKafkaOutputModeRejectsUnknownMode(t *testing.T) {
	if _, _, err := normalizeKafkaOutputConfig("topic_value", "value"); err == nil {
		t.Fatal("expected unknown output mode to fail")
	}
}
