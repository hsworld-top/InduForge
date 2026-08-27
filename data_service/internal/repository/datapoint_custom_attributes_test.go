package repository

import (
	"encoding/json"
	"testing"
	"time"
)

func TestProjectSnapshotDataPointAttributeDefaultsRoundTrip(t *testing.T) {
	snapshot := ProjectSnapshot{DataPoints: []DataPointRecord{{
		ID:                 "dp-1",
		AttributeDefaults:  map[string]string{"asset_code": "PUMP-001"},
		RuntimePermissions: DefaultDataPointRuntimePermissions(),
	}}}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	var restored ProjectSnapshot
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if got := restored.DataPoints[0].AttributeDefaults["asset_code"]; got != "PUMP-001" {
		t.Fatalf("snapshot attribute default = %q, want PUMP-001", got)
	}
}

func TestBuildProjectArtifactV1IncludesDataPointAttributeDefaults(t *testing.T) {
	artifact := BuildProjectArtifactV1("project-1", &ProjectSnapshot{
		DataPoints: []DataPointRecord{{
			ID: "dp-1", Path: "line.speed", Name: "线速", DataType: "float64",
			AttributeDefaults: map[string]string{"asset_code": "PUMP-001"},
		}},
	}, time.Now().UTC())

	if got := artifact.DataPoints[0].AttributeDefaults["asset_code"]; got != "PUMP-001" {
		t.Fatalf("artifact attribute default = %q, want PUMP-001", got)
	}
}
