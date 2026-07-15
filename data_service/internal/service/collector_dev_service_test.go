package service

import "testing"

func TestValidateCollectorTaskInputUsesSavedObjectReferences(t *testing.T) {
	err := validateCollectorTaskInput(CollectorTaskInput{AgentID: "agent-1", ConnectionID: "550e8400-e29b-41d4-a716-446655440002", Operation: "point.read", Input: map[string]any{"pointIds": []any{"550e8400-e29b-41d4-a716-446655440003"}}, TimeoutSeconds: 30})
	if err != nil {
		t.Fatal(err)
	}
	if err := validateCollectorTaskInput(CollectorTaskInput{AgentID: "agent-1", ConnectionID: "550e8400-e29b-41d4-a716-446655440002", Operation: "opcua" + ".read", Input: map[string]any{}}); err == nil {
		t.Fatal("expected protocol-specific operation to be rejected")
	}
}

func TestCollectorAgentSupportsDriverVersionSchemaAndOperation(t *testing.T) {
	capabilities := []CollectorProtocolCapability{{DriverID: "opcua.standard", DriverVersion: "1.0.0", SchemaVersions: []int{1}, Operations: []string{"connection.test", "point.read"}}}
	if !collectorAgentSupports(capabilities, "opcua.standard", "1.0.0", 1, "point.read") {
		t.Fatal("expected capability match")
	}
	if collectorAgentSupports(capabilities, "opcua.standard", "1.0.0", 2, "point.read") {
		t.Fatal("unexpected schema version match")
	}
}
