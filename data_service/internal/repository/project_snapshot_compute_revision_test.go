package repository

import (
	"testing"
)

func TestSnapshotComputeOutputTargetChangesAreExecutableChanges(t *testing.T) {
	outputs := []ComputeOutputRecord{{OutputKey: "result", DatapointID: "new-point"}}
	if !snapshotComputeOutputTargetsChanged(map[string]string{"result": "old-point"}, outputs) {
		t.Fatal("快照替换输出目标数据点必须触发 revision")
	}
	if snapshotComputeOutputTargetsChanged(map[string]string{"result": "new-point"}, outputs) {
		t.Fatal("同一输出目标数据点不应触发 revision")
	}
}

func TestSnapshotDefaultOutputRejectsJSONNullAndResidualDefault(t *testing.T) {
	nullDefault := `null`
	pointID := "point"
	unitID := "unit"
	snapshot := ProjectSnapshot{
		DataPoints:   []DataPointRecord{{ID: pointID, Path: "calc.unit.result", DataType: "uint64", DefaultValue: &nullDefault, Status: "active", SourceType: "calc.output", SourceID: &unitID, SourceConfig: map[string]any{"computeUnitId": unitID, "outputKey": "result"}}},
		ComputeUnits: []ComputeUnitRecord{{ID: unitID, Outputs: []ComputeOutputRecord{{DatapointID: pointID, OutputKey: "result", Path: "calc.unit.result", DataType: "uint64", NullPolicy: "default", DefaultValue: &nullDefault}}}},
	}
	if err := validateSnapshotComputeOutputTargets(snapshot); err == nil {
		t.Fatal("default 输出不能接受 JSON null")
	}
	value := `1`
	snapshot.DataPoints[0].DefaultValue = &value
	snapshot.ComputeUnits[0].Outputs[0].NullPolicy = "skip"
	snapshot.ComputeUnits[0].Outputs[0].DefaultValue = nil
	if err := validateSnapshotComputeOutputTargets(snapshot); err == nil {
		t.Fatal("非 default 输出不能残留默认值")
	}
}

func TestComputeOutputTargetInvariantIsSharedBySnapshotAndArtifact(t *testing.T) {
	unitID, pointID := "unit", "point"
	defaultValue := `7`
	output := ComputeOutputRecord{DatapointID: pointID, OutputKey: "result", Path: "calc.unit.result", Name: "result", DataType: "int32", NullPolicy: "default", DefaultValue: &defaultValue}
	good := DataPointRecord{ID: pointID, Path: output.Path, DataType: output.DataType, DefaultValue: &defaultValue, Status: "active", SourceType: "calc.output", SourceID: &unitID, SourceConfig: map[string]any{"computeUnitId": unitID, "outputKey": output.OutputKey}}
	unit := ComputeUnitRecord{ID: unitID, Revision: 1, Name: "unit", Language: "js", ScriptCode: "return 7", TriggerType: "manual", TriggerConfig: map[string]any{}, InputBindings: map[string]any{}, Dependencies: []any{}, TimeoutMS: 1000, IsEnabled: true, Outputs: []ComputeOutputRecord{output}}
	assertInvalid := func(name string, point DataPointRecord) {
		t.Helper()
		snapshot := ProjectSnapshot{DataPoints: []DataPointRecord{point}, ComputeUnits: []ComputeUnitRecord{unit}}
		if err := validateSnapshotComputeOutputTargets(snapshot); err == nil {
			t.Fatalf("%s: snapshot preflight must reject invalid output projection", name)
		}
		if _, err := buildRuntimeComputeUnit(unit, map[string]DataPointRecord{pointID: point}); err == nil {
			t.Fatalf("%s: runtime artifact must reject invalid output projection", name)
		}
	}
	if err := validateSnapshotComputeOutputTargets(ProjectSnapshot{DataPoints: []DataPointRecord{good}, ComputeUnits: []ComputeUnitRecord{unit}}); err != nil {
		t.Fatalf("valid output projection rejected: %v", err)
	}
	if _, err := buildRuntimeComputeUnit(unit, map[string]DataPointRecord{pointID: good}); err != nil {
		t.Fatalf("valid runtime output projection rejected: %v", err)
	}

	inactive := good
	inactive.Status = "inactive"
	assertInvalid("inactive", inactive)
	manual := good
	manual.SourceType, manual.SourceID, manual.SourceConfig = "manual", nil, map[string]any{}
	assertInvalid("manual source", manual)
	collector := good
	collector.SourceType = "collector.point"
	assertInvalid("collector source", collector)
	wrongSourceID := good
	otherUnitID := "other-unit"
	wrongSourceID.SourceID = &otherUnitID
	assertInvalid("wrong sourceId", wrongSourceID)
	wrongConfig := good
	wrongConfig.SourceConfig = map[string]any{"computeUnitId": unitID, "outputKey": "other"}
	assertInvalid("wrong sourceConfig", wrongConfig)
	extraConfig := good
	extraConfig.SourceConfig = map[string]any{"computeUnitId": unitID, "outputKey": output.OutputKey, "extra": true}
	assertInvalid("extra sourceConfig", extraConfig)
	wrongPath := good
	wrongPath.Path = "calc.unit.other"
	assertInvalid("path mismatch", wrongPath)
	wrongType := good
	wrongType.DataType = "float64"
	assertInvalid("data type mismatch", wrongType)
	wrongDefault := good
	otherDefault := `8`
	wrongDefault.DefaultValue = &otherDefault
	assertInvalid("default mismatch", wrongDefault)
}

func TestSnapshotReplaceRejectsInactiveOutputAndBooleanAlias(t *testing.T) {
	pointID := "point"
	snapshot := ProjectSnapshot{
		DataPoints:   []DataPointRecord{{ID: pointID, DataType: "float64", Status: "inactive"}},
		ComputeUnits: []ComputeUnitRecord{{Outputs: []ComputeOutputRecord{{DatapointID: pointID, DataType: "float64", NullPolicy: "error"}}}},
	}
	if err := validateSnapshotComputeOutputTargets(snapshot); err == nil {
		t.Fatal("snapshot Replace 不能持久化 inactive compute output")
	}
	unit := ComputeUnitRecord{InputBindings: map[string]any{"datapointVariables": []any{map[string]any{"datapointId": pointID, "alias": "false"}}}}
	if _, err := snapshotComputeDatapointRefs(unit); err == nil {
		t.Fatal("snapshot Replace 不能持久化 bool literal compute alias")
	}
}
