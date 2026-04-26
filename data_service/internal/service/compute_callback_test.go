package service

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/indu-forge/data_service/internal/repository"
)

func TestMqttTopicMatches(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		topic   string
		want    bool
	}{
		// exact match
		{"exact match", "foo/bar", "foo/bar", true},
		{"exact mismatch", "foo/bar", "foo/baz", false},
		{"exact with extra topic level", "foo/bar", "foo/bar/baz", false},

		// single level wildcard +
		{"plus single level match", "foo/+/baz", "foo/bar/baz", true},
		{"plus single level mismatch", "foo/+/baz", "foo/bar/qux", false},
		{"plus exact", "foo/+", "foo/bar", true},
		{"plus extra levels", "foo/+", "foo/bar/baz", false},

		// multi-level wildcard #
		{"hash match", "foo/#", "foo/bar", true},
		{"hash match deep", "foo/#", "foo/bar/baz/qux", true},
		{"hash must be last", "foo/#/bar", "foo/bar/baz/bar", false},
		{"hash exact", "foo/#", "foo", true},

		// special
		{"empty pattern", "", "foo/bar", false},
		{"empty topic", "foo/bar", "", false},
		{"wildcard star match", "*", "foo/bar", true},
		{"wildcard star match", "*", "*", true},

		// edge
		{"plus vs hash", "foo/+/bar/#", "foo/baz/bar/qux", true},
		{"trailing slash", "foo/bar/", "foo/bar/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mqttTopicMatches(tt.pattern, tt.topic)
			if got != tt.want {
				t.Errorf("mqttTopicMatches(%q, %q) = %v, want %v", tt.pattern, tt.topic, got, tt.want)
			}
		})
	}
}

func TestComputeMqttPublishAllowed(t *testing.T) {
	makeUnit := func(triggerConfig, inputBindings map[string]any) repository.ComputeUnitRecord {
		return repository.ComputeUnitRecord{
			TriggerConfig: triggerConfig,
			InputBindings: inputBindings,
		}
	}

	tests := []struct {
		name   string
		unit   repository.ComputeUnitRecord
		source string
		topic  string
		want   bool
	}{
		{
			name:   "sideEffects false returns false",
			unit:   makeUnit(map[string]any{"sideEffects": map[string]any{"mqttPublish": map[string]any{"enabled": false}}}, nil),
			source: "src1",
			topic:  "test/topic",
			want:   false,
		},
		{
			name:   "sideEffects nil returns false",
			unit:   makeUnit(map[string]any{}, nil),
			source: "src1",
			topic:  "test/topic",
			want:   false,
		},
		{
			name:   "enabled true no sourceIds no topics returns true",
			unit:   makeUnit(map[string]any{"sideEffects": map[string]any{"mqttPublish": map[string]any{"enabled": true}}}, nil),
			source: "src1",
			topic:  "test/topic",
			want:   true,
		},
		{
			name:   "sourceIds mismatch returns false",
			unit:   makeUnit(map[string]any{"sideEffects": map[string]any{"mqttPublish": map[string]any{"enabled": true, "sourceIds": []string{"src2", "src3"}}}}, nil),
			source: "src1",
			topic:  "test/topic",
			want:   false,
		},
		{
			name:   "sourceIds match returns true",
			unit:   makeUnit(map[string]any{"sideEffects": map[string]any{"mqttPublish": map[string]any{"enabled": true, "sourceIds": []string{"src1", "src2"}}}}, nil),
			source: "src1",
			topic:  "test/topic",
			want:   true,
		},
		{
			name:   "topics mismatch returns false",
			unit:   makeUnit(map[string]any{"sideEffects": map[string]any{"mqttPublish": map[string]any{"enabled": true, "topics": []string{"other/topic"}}}}, nil),
			source: "src1",
			topic:  "test/topic",
			want:   false,
		},
		{
			name:   "topics match returns true",
			unit:   makeUnit(map[string]any{"sideEffects": map[string]any{"mqttPublish": map[string]any{"enabled": true, "topics": []string{"test/topic"}}}}, nil),
			source: "src1",
			topic:  "test/topic",
			want:   true,
		},
		{
			name:   "topics wildcard plus",
			unit:   makeUnit(map[string]any{"sideEffects": map[string]any{"mqttPublish": map[string]any{"enabled": true, "topics": []string{"test/+"}}}}, nil),
			source: "src1",
			topic:  "test/topic",
			want:   true,
		},
		{
			name:   "topics wildcard hash",
			unit:   makeUnit(map[string]any{"sideEffects": map[string]any{"mqttPublish": map[string]any{"enabled": true, "topics": []string{"test/#"}}}}, nil),
			source: "src1",
			topic:  "test/deep/nested",
			want:   true,
		},
		{
			name:   "effects alias for sideEffects",
			unit:   makeUnit(map[string]any{"effects": map[string]any{"mqtt": map[string]any{"enabled": true}}}, nil),
			source: "src1",
			topic:  "test/topic",
			want:   true,
		},
		{
			name:   "inputBindings takes precedence over TriggerConfig when both have mqttPublish",
			unit:   makeUnit(map[string]any{"sideEffects": map[string]any{"mqttPublish": map[string]any{"enabled": false}}}, map[string]any{"sideEffects": map[string]any{"mqttPublish": map[string]any{"enabled": true, "topics": []string{"test/topic"}}}}),
			source: "src1",
			topic:  "test/topic",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeMqttPublishAllowed(tt.unit, tt.source, tt.topic)
			if got != tt.want {
				t.Errorf("computeMqttPublishAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeMQTTPayload(t *testing.T) {
	t.Run("string passthrough", func(t *testing.T) {
		payload, err := computeMQTTPayload("hello")
		if err != nil {
			t.Fatalf("computeMQTTPayload() error = %v", err)
		}
		if payload != "hello" {
			t.Fatalf("payload = %v, want hello", payload)
		}
	})

	t.Run("bytes passthrough", func(t *testing.T) {
		input := []byte("hello")
		payload, err := computeMQTTPayload(input)
		if err != nil {
			t.Fatalf("computeMQTTPayload() error = %v", err)
		}
		got, ok := payload.([]byte)
		if !ok {
			t.Fatalf("payload type = %T, want []byte", payload)
		}
		if !bytes.Equal(got, input) {
			t.Fatalf("payload = %q, want %q", got, input)
		}
	})

	t.Run("map marshals to JSON bytes", func(t *testing.T) {
		payload, err := computeMQTTPayload(map[string]any{"key": "value"})
		if err != nil {
			t.Fatalf("computeMQTTPayload() error = %v", err)
		}
		got, ok := payload.([]byte)
		if !ok {
			t.Fatalf("payload type = %T, want []byte", payload)
		}
		var decoded map[string]string
		if err := json.Unmarshal(got, &decoded); err != nil {
			t.Fatalf("payload is not valid JSON: %v", err)
		}
		if decoded["key"] != "value" {
			t.Fatalf("decoded key = %q, want value", decoded["key"])
		}
	})

	t.Run("nil marshals to null", func(t *testing.T) {
		payload, err := computeMQTTPayload(nil)
		if err != nil {
			t.Fatalf("computeMQTTPayload() error = %v", err)
		}
		got, ok := payload.([]byte)
		if !ok {
			t.Fatalf("payload type = %T, want []byte", payload)
		}
		if string(got) != "null" {
			t.Fatalf("payload = %q, want null", got)
		}
	})

	t.Run("unmarshalable payload returns error", func(t *testing.T) {
		_, err := computeMQTTPayload(make(chan int))
		if err == nil {
			t.Fatal("computeMQTTPayload() error = nil, want error")
		}
	})
}

func TestComputeMQTTPayloadStruct(t *testing.T) {
	payload, err := computeMQTTPayload(struct {
		A int `json:"a"`
	}{A: 1})
	if err != nil {
		t.Fatalf("computeMQTTPayload() error = %v", err)
	}
	got, ok := payload.([]byte)
	if !ok {
		t.Fatalf("payload type = %T, want []byte", payload)
	}
	if string(got) != `{"a":1}` {
		t.Fatalf("payload = %q, want {\"a\":1}", got)
	}
}

func TestMergeComputeSDKQueryParameters(t *testing.T) {
	tests := []struct {
		name     string
		base     map[string]any
		override map[string]any
	}{
		{
			name:     "override replaces base",
			base:     map[string]any{"a": 1, "b": 2},
			override: map[string]any{"b": 3, "c": 4},
		},
		{
			name:     "empty base",
			base:     nil,
			override: map[string]any{"a": 1},
		},
		{
			name:     "empty override",
			base:     map[string]any{"a": 1},
			override: nil,
		},
		{
			name:     "both empty",
			base:     nil,
			override: nil,
		},
		{
			name:     "override adds new keys",
			base:     map[string]any{"a": 1},
			override: map[string]any{"b": 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeComputeSDKQueryParameters(tt.base, tt.override)

			// verify override values are present
			for k, v := range tt.override {
				if result[k] != v {
					t.Errorf("override key %q: got %v, want %v", k, result[k], v)
				}
			}

			// verify base values not in override are preserved
			for k, v := range tt.base {
				if _, inOverride := tt.override[k]; !inOverride {
					if result[k] != v {
						t.Errorf("base key %q: got %v, want %v", k, result[k], v)
					}
				}
			}
		})
	}
}
