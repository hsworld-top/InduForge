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
