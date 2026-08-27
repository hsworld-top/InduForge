package sceneasset

import (
	"errors"
	"testing"
)

func TestValidatePublicContractAcceptsStructuredMembers(t *testing.T) {
	contract := map[string]any{
		"description": "设备场景",
		"parameters":  []any{map[string]any{"name": "deviceId", "required": true, "schema": map[string]any{"type": "string"}}},
		"events":      []any{map[string]any{"name": "device:selected", "schema": map[string]any{"type": "object"}}},
		"commands": []any{map[string]any{"name": "focusDevice", "inputSchema": map[string]any{"type": "object"},
			"outputSchema": map[string]any{"type": "boolean"}}},
	}
	if _, err := validatePublicContract(contract); err != nil {
		t.Fatalf("valid contract rejected: %v", err)
	}
}

func TestMergeSceneContractsRejectsSchemaConflict(t *testing.T) {
	publicContract := emptyPublicContract()
	publicContract["events"] = []any{map[string]any{"name": "changed", "schema": map[string]any{"type": "string"}}}
	managedContract := emptyPublicContract()
	managedContract["events"] = []any{map[string]any{"name": "changed", "schema": map[string]any{"type": "number"}}}
	if _, err := mergeSceneContracts(publicContract, managedContract); !errors.Is(err, ErrContractInvalid) {
		t.Fatalf("Schema 冲突应拒绝提交: %v", err)
	}
}

func TestValidatePublicContractRejectsLegacyAndInvalidSchema(t *testing.T) {
	for _, contract := range []map[string]any{
		{"description": "", "parameters": []any{}, "events": []any{}, "commands": []any{}, "embedMode": "both"},
		{"description": "", "parameters": []any{map[string]any{"name": "bad name", "required": false, "schema": map[string]any{"type": "string"}}}, "events": []any{}, "commands": []any{}},
		{"description": "", "parameters": []any{map[string]any{"name": "value", "required": false, "schema": map[string]any{"type": "unknown"}}}, "events": []any{}, "commands": []any{}},
	} {
		if _, err := validatePublicContract(contract); err == nil {
			t.Fatalf("invalid contract accepted: %#v", contract)
		}
	}
}
