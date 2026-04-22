package handler

import (
	"encoding/json"
	"testing"
)

func TestParseDataPointRuntimePermissionsInputDefaults(t *testing.T) {
	raw := map[string]json.RawMessage{
		"write": json.RawMessage(`{}`),
	}

	input, err := parseDataPointRuntimePermissionsInput(raw)
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}

	if !input.Write.Inherit {
		t.Fatal("expected default inherit=true")
	}
	if len(input.Write.AllowRoles) != 0 {
		t.Fatalf("expected empty allowRoles, got %v", input.Write.AllowRoles)
	}
	if len(input.Write.DenyRoles) != 0 {
		t.Fatalf("expected empty denyRoles, got %v", input.Write.DenyRoles)
	}
}

func TestParseDataPointRuntimePermissionsInputRejectsUnknownField(t *testing.T) {
	raw := map[string]json.RawMessage{
		"write": json.RawMessage(`{}`),
		"read":  json.RawMessage(`{}`),
	}

	if _, err := parseDataPointRuntimePermissionsInput(raw); err == nil {
		t.Fatal("expected parse to reject unsupported permission field")
	}
}

func TestParseDataPointRuntimePermissionsInputRejectsNullWrite(t *testing.T) {
	raw := map[string]json.RawMessage{
		"write": json.RawMessage(`null`),
	}

	if _, err := parseDataPointRuntimePermissionsInput(raw); err == nil {
		t.Fatal("expected parse to reject write=null")
	}
}

func TestParseDataPointRuntimePermissionsInputRejectsExplicitNullWriteFields(t *testing.T) {
	testCases := []struct {
		name    string
		payload string
	}{
		{
			name:    "inherit null",
			payload: `{"inherit":null}`,
		},
		{
			name:    "allowRoles null",
			payload: `{"allowRoles":null,"inherit":true}`,
		},
		{
			name:    "denyRoles null",
			payload: `{"denyRoles":null,"inherit":true}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			raw := map[string]json.RawMessage{
				"write": json.RawMessage(testCase.payload),
			}
			if _, err := parseDataPointRuntimePermissionsInput(raw); err == nil {
				t.Fatal("expected explicit null write field to be rejected")
			}
		})
	}
}
