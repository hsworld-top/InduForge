package repository

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func runtimeSchemaRootForTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "contracts", "runtime"))
}

func TestRuntimeProjectArtifactV1IsSchemaValidAndByteDeterministic(t *testing.T) {
	projectID, pointID := uuid.NewString(), uuid.NewString()
	value := `12`
	snapshot := &ProjectSnapshot{DataPoints: []DataPointRecord{{ID: pointID, ProjectID: projectID, Path: "metrics.counter", Name: "counter", SourceType: "manual", SourceConfig: map[string]any{}, DataType: "int32", DefaultValue: &value, Tags: []any{}, AttributeDefaults: map[string]string{}, RuntimePermissions: DefaultDataPointRuntimePermissions(), RefreshMode: "auto", Status: "active"}}}
	generatedAt := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	first, err := BuildRuntimeProjectArtifactV1(runtimeSchemaRootForTest(t), projectID, snapshot, generatedAt)
	if err != nil {
		t.Fatalf("build artifact: %v", err)
	}
	second, err := BuildRuntimeProjectArtifactV1(runtimeSchemaRootForTest(t), projectID, snapshot, generatedAt)
	if err != nil {
		t.Fatalf("build second artifact: %v", err)
	}
	left, _ := json.Marshal(first)
	right, _ := json.Marshal(second)
	if string(left) != string(right) {
		t.Fatalf("artifact bytes differ:\n%s\n%s", left, right)
	}
	point := first.DataPoints[0].(map[string]any)
	if point["sourceType"] != "manual.input" || point["sourceId"] != nil || len(point["sourceConfig"].(map[string]any)) != 0 {
		t.Fatalf("manual datapoint must freeze as manual.input with null/empty source: %#v", point)
	}
}

func TestRuntimeProjectArtifactV1RequiresExplicitSchemaRoot(t *testing.T) {
	projectID := uuid.NewString()
	_, err := BuildRuntimeProjectArtifactV1("", projectID, &ProjectSnapshot{}, time.Now().UTC())
	if err == nil || !strings.Contains(err.Error(), "schema 根目录") {
		t.Fatalf("expected schema root failure, got %v", err)
	}
}

func TestRuntimeProjectArtifactV1RejectsInvalidTypedDefault(t *testing.T) {
	projectID := uuid.NewString()
	pointID := uuid.NewString()
	value := `"not-an-int"`
	snapshot := &ProjectSnapshot{DataPoints: []DataPointRecord{{ID: pointID, ProjectID: projectID, Path: "metrics.counter", Name: "counter", SourceType: "manual", SourceConfig: map[string]any{}, DataType: "int32", DefaultValue: &value, Tags: []any{}, AttributeDefaults: map[string]string{}, RuntimePermissions: DefaultDataPointRuntimePermissions(), RefreshMode: "auto", Status: "active"}}}
	_, err := BuildRuntimeProjectArtifactV1(runtimeSchemaRootForTest(t), projectID, snapshot, time.Now().UTC())
	if err == nil || !strings.Contains(err.Error(), "默认值") {
		t.Fatalf("expected typed default error, got %v", err)
	}
}

func TestRuntimeProjectArtifactProjectsCalcOutputToFrozenComputeIDContract(t *testing.T) {
	projectID, unitID, pointID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	snapshot := &ProjectSnapshot{
		DataPoints:   []DataPointRecord{{ID: pointID, ProjectID: projectID, Path: "calc.unit.result", Name: "result", SourceType: "calc.output", SourceID: &unitID, SourceConfig: map[string]any{"computeUnitId": unitID, "outputKey": "result"}, DataType: "float64", Tags: []any{}, AttributeDefaults: map[string]string{}, RuntimePermissions: DefaultDataPointRuntimePermissions(), RefreshMode: "manual", Status: "active"}},
		ComputeUnits: []ComputeUnitRecord{{ID: unitID, Revision: 1, Name: "unit", Language: "js", ScriptCode: "return 1", TriggerType: "manual", TriggerConfig: map[string]any{}, InputBindings: map[string]any{}, Dependencies: []any{}, TimeoutMS: 1000, IsEnabled: true, Outputs: []ComputeOutputRecord{{ID: uuid.NewString(), DatapointID: pointID, OutputKey: "result", Name: "result", Path: "calc.unit.result", DataType: "float64", NullPolicy: "error"}}}},
	}
	artifact, err := BuildRuntimeProjectArtifactV1(runtimeSchemaRootForTest(t), projectID, snapshot, time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("build artifact: %v", err)
	}
	config := artifact.DataPoints[0].(map[string]any)["sourceConfig"].(map[string]any)
	if len(config) != 1 || config["computeId"] != unitID {
		t.Fatalf("runtime calc.output 必须是 frozen computeId 投影: %#v", config)
	}
	snapshot.DataPoints[0].Status = "inactive"
	if _, err := BuildRuntimeProjectArtifactV1(runtimeSchemaRootForTest(t), projectID, snapshot, time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("inactive calc.output 不能进入运行制品")
	}
}

func TestRuntimeProjectArtifactProjectsBaseExternalSourcesWithoutSourceConfig(t *testing.T) {
	projectID := uuid.NewString()
	queryID, subscriptionID, connectionID, keyID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	point := func(path, sourceType, sourceID string, sourceConfig map[string]any) DataPointRecord {
		return DataPointRecord{ID: uuid.NewString(), ProjectID: projectID, Path: path, Name: path, SourceType: sourceType, SourceID: &sourceID, SourceConfig: sourceConfig, DataType: "float64", Tags: []any{}, AttributeDefaults: map[string]string{}, RuntimePermissions: DefaultDataPointRuntimePermissions(), RefreshMode: "auto", Status: "active"}
	}
	snapshot := &ProjectSnapshot{DataPoints: []DataPointRecord{
		point("query.temperature", "db.query", queryID, map[string]any{"queryId": queryID, "ignored": "not-in-runtime-artifact"}),
		point("mqtt.temperature", "mqtt.subscription", subscriptionID, map[string]any{"subscriptionId": subscriptionID}),
		point("realtime.temperature", "realtime.key", connectionID, map[string]any{"keyId": keyID, "ignored": "not-in-runtime-artifact"}),
	}}
	artifact, err := BuildRuntimeProjectArtifactV1(runtimeSchemaRootForTest(t), projectID, snapshot, time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("build artifact: %v", err)
	}
	byType := map[string]map[string]any{}
	for _, raw := range artifact.DataPoints {
		item := raw.(map[string]any)
		byType[item["sourceType"].(string)] = item
	}
	for _, sourceType := range []string{"db.query", "mqtt.subscription"} {
		if config := byType[sourceType]["sourceConfig"].(map[string]any); len(config) != 0 {
			t.Fatalf("%s must not export source config: %#v", sourceType, config)
		}
	}
	if config := byType["realtime.key"]["sourceConfig"].(map[string]any); len(config) != 1 || config["keyId"] != keyID {
		t.Fatalf("realtime.key must export only keyId: %#v", config)
	}
}

func TestRuntimeProjectArtifactRejectsRealtimeKeyWithoutKeyID(t *testing.T) {
	projectID, connectionID := uuid.NewString(), uuid.NewString()
	snapshot := &ProjectSnapshot{DataPoints: []DataPointRecord{{ID: uuid.NewString(), ProjectID: projectID, Path: "realtime.temperature", Name: "temperature", SourceType: "realtime.key", SourceID: &connectionID, SourceConfig: map[string]any{}, DataType: "float64", Tags: []any{}, AttributeDefaults: map[string]string{}, RuntimePermissions: DefaultDataPointRuntimePermissions(), RefreshMode: "auto", Status: "active"}}}
	if _, err := BuildRuntimeProjectArtifactV1(runtimeSchemaRootForTest(t), projectID, snapshot, time.Now().UTC()); err == nil || !strings.Contains(err.Error(), "keyId") {
		t.Fatalf("expected missing keyId rejection, got %v", err)
	}
}

func TestRuntimeInputsRejectBooleanLiteralAliases(t *testing.T) {
	pointID := uuid.NewString()
	_, err := runtimeInputs(map[string]any{"datapointVariables": []any{map[string]any{"datapointId": pointID, "alias": "true"}}}, map[string]DataPointRecord{pointID: {ID: pointID, Status: "active"}})
	if err == nil {
		t.Fatal("compute input alias true 必须 fail closed")
	}
}

func TestParseTypedDefaultKeepsUint64BoundaryExact(t *testing.T) {
	valid := `18446744073709551615`
	if _, err := parseTypedDefault(&valid, "uint64"); err != nil {
		t.Fatalf("max uint64 should be accepted: %v", err)
	}
	overflow := `18446744073709551616`
	if _, err := parseTypedDefault(&overflow, "uint64"); err == nil {
		t.Fatal("uint64 overflow must be rejected")
	}
	joined := `1 2`
	if _, err := parseTypedDefault(&joined, "int32"); err == nil {
		t.Fatal("multiple JSON values must be rejected")
	}
}

func TestParseTypedDefaultRequiresExactBytesAndUTCDateTime(t *testing.T) {
	for _, testCase := range []struct {
		value, dataType string
		valid           bool
	}{
		{`"AQI="`, "bytes", true},
		{`"AQI"`, "bytes", false},
		{`"2026-08-30T12:00:00Z"`, "datetime", true},
		{`"2026-08-30T20:00:00+08:00"`, "datetime", false},
		{`3.5e38`, "float32", false},
	} {
		_, err := ParseTypedJSONDefault(testCase.value, testCase.dataType)
		if (err == nil) != testCase.valid {
			t.Fatalf("ParseTypedJSONDefault(%s, %s) err=%v", testCase.value, testCase.dataType, err)
		}
	}
}

func TestParseTypedDefaultKeepsDecimalExactWithoutFloat64(t *testing.T) {
	for _, raw := range []string{`1e1000`, `9007199254740992`, `9007199254740993`, `0.123456789012345678901234567890123456789`} {
		value, err := ParseTypedJSONDefault(raw, "decimal")
		if err != nil {
			t.Fatalf("decimal %s should be accepted: %v", raw, err)
		}
		if number, ok := value.(json.Number); !ok || number.String() != raw {
			t.Fatalf("decimal lost exact JSON spelling: %#v", value)
		}
	}
	if _, err := ParseTypedJSONDefault(`1e1000`, "float64"); err == nil {
		t.Fatal("float64 overflow must be rejected")
	}
}

func TestCollectorRuntimeArtifactNeverEmitsEndpointOrSecrets(t *testing.T) {
	projectID, connectionID, variableID, datapointID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	snapshot := &ProjectSnapshot{CollectorConnections: []SnapshotCollectorConnectionRecord{{ID: connectionID, ProtocolFamily: "modbus", DriverID: "modbus.tcp", DriverVersion: "1.0.0", SchemaVersion: 1, IsEnabled: true, Config: map[string]any{"host": "10.0.0.8", "port": 502}, DefaultAcquisition: map[string]any{"intervalMs": 1000, "deadband": 0.0, "changeOnly": false}}}, CollectorPoints: []SnapshotCollectorPointRecord{{ID: variableID, ConnectionID: connectionID, Address: map[string]any{"station": 1, "area": "holdingRegister", "address": 0}, AddressSchemaVersion: 1, DataType: "int16", ElementCount: 1, ReadOptions: map[string]any{}, AcquisitionMode: "inherit", Enabled: true}}, DataPoints: []DataPointRecord{{ID: datapointID, ProjectID: projectID, Path: "collector.modbus.value", Name: "value", SourceType: "collector.point", SourceID: &variableID, SourceConfig: map[string]any{"connectionId": connectionID}, DataType: "int16", Tags: []any{}, AttributeDefaults: map[string]string{}, RuntimePermissions: DefaultDataPointRuntimePermissions(), RefreshMode: "auto", Status: "active"}}}
	artifact, err := BuildCollectorRuntimeArtifactV1(runtimeSchemaRootForTest(t), projectID, "collector-a", 1, "1.0.0", snapshot)
	if err != nil {
		t.Fatalf("build collector artifact: %v", err)
	}
	payload, _ := json.Marshal(artifact)
	for _, forbidden := range []string{"10.0.0.8", "host", "password", "secret"} {
		if strings.Contains(strings.ToLower(string(payload)), strings.ToLower(forbidden)) {
			t.Fatalf("collector artifact leaked %q: %s", forbidden, payload)
		}
	}
}

func TestCollectorRuntimeArtifactRejectsAmbiguousOrUnsafeMappings(t *testing.T) {
	projectID, connectionID, variableID, datapointID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	snapshot := validCollectorRuntimeSnapshot(projectID, connectionID, variableID, datapointID)
	duplicateID := uuid.NewString()
	duplicate := snapshot.DataPoints[0]
	duplicate.ID = duplicateID
	snapshot.DataPoints = append(snapshot.DataPoints, duplicate)
	if _, err := BuildCollectorRuntimeArtifactV1(runtimeSchemaRootForTest(t), projectID, "collector-a", 1, "1.0.0", snapshot); err == nil {
		t.Fatal("一个采集变量映射多个正式数据点必须失败")
	}

	snapshot = validCollectorRuntimeSnapshot(projectID, connectionID, variableID, datapointID)
	snapshot.DataPoints[0].SourceConfig = map[string]any{"connectionId": connectionID, "host": "leak"}
	if _, err := BuildCollectorRuntimeArtifactV1(runtimeSchemaRootForTest(t), projectID, "collector-a", 1, "1.0.0", snapshot); err == nil {
		t.Fatal("采集 sourceConfig 的现场字段必须失败")
	}

	snapshot = validCollectorRuntimeSnapshot(projectID, connectionID, variableID, datapointID)
	snapshot.CollectorPoints[0].AcquisitionOverrides = map[string]any{"unknown": true}
	snapshot.CollectorPoints[0].AcquisitionMode = "override"
	if _, err := BuildCollectorRuntimeArtifactV1(runtimeSchemaRootForTest(t), projectID, "collector-a", 1, "1.0.0", snapshot); err == nil {
		t.Fatal("未知 acquisition override 字段必须失败")
	}

	snapshot = validCollectorRuntimeSnapshot(projectID, connectionID, variableID, datapointID)
	snapshot.CollectorPoints[0].ReadOptions = map[string]any{"dataFormat": "CDAB"}
	if _, err := BuildCollectorRuntimeArtifactV1(runtimeSchemaRootForTest(t), projectID, "collector-a", 1, "1.0.0", snapshot); err == nil {
		t.Fatal("V1 未消费的 readOptions 字段必须失败")
	}
	if err := validateCollectorProtocolSchema(runtimeSchemaRootForTest(t), "modbus.tcp", "missing-read-options.schema.json", map[string]any{}); err == nil {
		t.Fatal("缺失 readOptions schema 必须 fail closed")
	}
}

func TestCollectorRuntimeArtifactAcceptsEmptyReadOptionsForOPCUA(t *testing.T) {
	projectID, connectionID, variableID, datapointID := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	snapshot := validCollectorRuntimeSnapshot(projectID, connectionID, variableID, datapointID)
	snapshot.CollectorConnections[0].ProtocolFamily = "opcua"
	snapshot.CollectorConnections[0].DriverID = "opcua.standard"
	snapshot.CollectorConnections[0].Config = map[string]any{
		"host": "opc.example.test", "port": 4840, "endpointPath": "/", "securityMode": "None", "securityPolicy": "None", "authenticationType": "anonymous",
	}
	snapshot.CollectorPoints[0].Address = map[string]any{"nodeId": "ns=2;s=Demo.Dynamic.Scalar.Float"}
	snapshot.CollectorPoints[0].DataType = "float64"
	snapshot.DataPoints[0].DataType = "float64"
	snapshot.CollectorPoints[0].ReadOptions = map[string]any{}
	if _, err := BuildCollectorRuntimeArtifactV1(runtimeSchemaRootForTest(t), projectID, "collector-a", 1, "1.0.0", snapshot); err != nil {
		t.Fatalf("OPC UA empty V1 readOptions should pass: %v", err)
	}

	snapshot.CollectorPoints[0].ReadOptions = map[string]any{"samplingIntervalMs": 1000}
	if _, err := BuildCollectorRuntimeArtifactV1(runtimeSchemaRootForTest(t), projectID, "collector-a", 1, "1.0.0", snapshot); err == nil {
		t.Fatal("OPC UA V1 readOptions must reject unsupported fields")
	}
}

func validCollectorRuntimeSnapshot(projectID, connectionID, variableID, datapointID string) *ProjectSnapshot {
	return &ProjectSnapshot{CollectorConnections: []SnapshotCollectorConnectionRecord{{ID: connectionID, ProtocolFamily: "modbus", DriverID: "modbus.tcp", DriverVersion: "1.0.0", SchemaVersion: 1, IsEnabled: true, Config: map[string]any{"host": "10.0.0.8", "port": 502}, DefaultAcquisition: map[string]any{"intervalMs": 1000, "deadband": 0.0, "changeOnly": false}}}, CollectorPoints: []SnapshotCollectorPointRecord{{ID: variableID, ConnectionID: connectionID, Address: map[string]any{"station": 1, "area": "holdingRegister", "address": 0}, AddressSchemaVersion: 1, DataType: "int16", ElementCount: 1, ReadOptions: map[string]any{}, AcquisitionMode: "inherit", Enabled: true}}, DataPoints: []DataPointRecord{{ID: datapointID, ProjectID: projectID, Path: "collector.modbus.value", Name: "value", SourceType: "collector.point", SourceID: &variableID, SourceConfig: map[string]any{"connectionId": connectionID}, DataType: "int16", Tags: []any{}, AttributeDefaults: map[string]string{}, RuntimePermissions: DefaultDataPointRuntimePermissions(), RefreshMode: "auto", Status: "active"}}}
}
