package compute

import (
	"encoding/json"
	"testing"
)

func TestSDKDataPointValueSerializesAttributesForMeta(t *testing.T) {
	payload, err := json.Marshal(SDKContext{Datapoints: map[string]SDKDataPointValue{
		"line.speed": {
			Path:       "line.speed",
			Attributes: map[string]string{"asset_code": "PUMP-001"},
		},
	}})
	if err != nil {
		t.Fatalf("marshal SDK context: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal SDK context: %v", err)
	}
	datapoints := decoded["datapoints"].(map[string]any)
	item := datapoints["line.speed"].(map[string]any)
	attributes := item["attributes"].(map[string]any)
	if attributes["asset_code"] != "PUMP-001" {
		t.Fatalf("SDK attributes = %#v", attributes)
	}
}
