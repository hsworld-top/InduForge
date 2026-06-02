package service

import (
	"testing"

	"github.com/indu-forge/data_service/internal/repository"
)

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
