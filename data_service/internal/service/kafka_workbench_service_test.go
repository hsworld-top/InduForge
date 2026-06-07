package service

import (
	"context"
	"strings"
	"testing"

	"github.com/indu-forge/data_service/internal/repository"
)

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

func TestBuildKafkaRawDataPointPathUsesMappingName(t *testing.T) {
	path := buildKafkaRawDataPointPath("test1")
	if path != "kafka.test1" {
		t.Fatalf("unexpected raw data point path: %s", path)
	}
}

func TestNormalizeUpdateTopicMappingRebuildsRawPathFromName(t *testing.T) {
	service := &KafkaWorkbenchService{}
	oldPath := "kafka.old.message"
	current := repository.KafkaTopicMappingRecord{
		ID:               "mapping-1",
		ProjectID:        "project-1",
		ConnectionID:     "connection-1",
		Name:             "old",
		Topic:            "device.telemetry",
		OutputMode:       "raw_message",
		RawOutputScope:   "value",
		RawDataPointPath: &oldPath,
		PartitionMode:    "all",
		StartPosition:    "latest",
		Decode:           "json",
		SampleLimit:      100,
		TimeoutMS:        5000,
	}

	params, err := service.normalizeUpdateTopicMapping(
		context.Background(),
		"project-1",
		"user-1",
		current,
		UpdateKafkaTopicMappingInput{
			Name:           "test1",
			Topic:          "device.telemetry",
			RawOutputScope: "value",
			PartitionMode:  "all",
			StartPosition:  "latest",
			Decode:         "json",
			SampleLimit:    100,
			TimeoutMS:      5000,
		},
	)
	if err != nil {
		t.Fatalf("normalize update failed: %v", err)
	}
	if params.RawDataPointPathInput != "kafka.test1" {
		t.Fatalf("unexpected raw path: %s", params.RawDataPointPathInput)
	}
}

func TestNormalizeUpdateTopicMappingRejectsOutputModeChange(t *testing.T) {
	service := &KafkaWorkbenchService{}
	current := repository.KafkaTopicMappingRecord{
		ID:             "mapping-1",
		ProjectID:      "project-1",
		ConnectionID:   "connection-1",
		Name:           "设备遥测",
		Topic:          "device.telemetry",
		ConsumerGroup:  "",
		OutputMode:     "raw_message",
		RawOutputScope: "value",
		PartitionMode:  "all",
		StartPosition:  "latest",
		Decode:         "json",
		SampleLimit:    100,
		TimeoutMS:      5000,
	}

	_, err := service.normalizeUpdateTopicMapping(
		context.Background(),
		"project-1",
		"user-1",
		current,
		UpdateKafkaTopicMappingInput{
			Name:           "设备遥测",
			Topic:          "device.telemetry",
			OutputMode:     "field_mapping",
			RawOutputScope: "value",
			PartitionMode:  "all",
			StartPosition:  "latest",
			Decode:         "json",
			SampleLimit:    100,
			TimeoutMS:      5000,
		},
	)

	if err == nil {
		t.Fatal("expected output mode change to fail")
	}
	if !strings.Contains(err.Error(), "输出模式创建后不可修改") {
		t.Fatalf("unexpected error: %v", err)
	}
}
