package service

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/indu-forge/data_service/internal/repository"
)

func snapshotArtifactSchemaRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位测试文件")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "contracts", "runtime"))
}

func TestBuildArtifactsFromSnapshotUsesOnlyCapturedInput(t *testing.T) {
	projectID, tenantID, pointID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	captured := repository.ProjectSnapshot{DataPoints: []repository.DataPointRecord{{
		ID: pointID, ProjectID: projectID, Path: "metrics.captured", Name: "captured",
		SourceType: "manual", SourceConfig: map[string]any{}, DataType: "float64",
		Tags: []any{}, AttributeDefaults: map[string]string{}, RuntimePermissions: repository.DefaultDataPointRuntimePermissions(),
		RefreshMode: "manual", Status: "active",
	}}}

	// nil repository 是有意的：纯构建路径若尝试读取 live DB 会直接失败。
	service := NewProjectSnapshotServiceWithClockAndSchemaRoot(nil, func() time.Time {
		return time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	}, snapshotArtifactSchemaRoot(t))
	capturedAt := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	first, err := service.BuildArtifactsFromSnapshot(projectID, tenantID, capturedAt, captured, nil)
	if err != nil {
		t.Fatalf("build captured artifacts: %v", err)
	}
	if len(first.RuntimeArtifact.DataPoints) != 1 || first.RuntimeArtifact.DataPoints[0].(map[string]any)["name"] != "captured" {
		t.Fatalf("unexpected captured projection: %#v", first.RuntimeArtifact.DataPoints)
	}

	// 模拟捕获后 live 工程已经变化；再次传入原 captured 值仍必须得到相同制品。
	liveAfterCapture := captured
	liveAfterCapture.DataPoints = append([]repository.DataPointRecord(nil), captured.DataPoints...)
	liveAfterCapture.DataPoints[0].Name = "live-after-capture"
	second, err := service.BuildArtifactsFromSnapshot(projectID, tenantID, capturedAt, captured, nil)
	if err != nil {
		t.Fatalf("rebuild captured artifacts: %v", err)
	}
	if second.RuntimeArtifact.DataPoints[0].(map[string]any)["name"] != "captured" {
		t.Fatalf("artifact followed live state: %#v", second.RuntimeArtifact.DataPoints)
	}
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("same request body must be byte-replayable:\n%s\n%s", firstJSON, secondJSON)
	}
	if liveAfterCapture.DataPoints[0].Name != "live-after-capture" {
		t.Fatal("test setup did not preserve independent live state")
	}
}

func TestBuildArtifactsFromSnapshotIncludesCollectorOnlyWhenRequested(t *testing.T) {
	projectID, tenantID := uuid.NewString(), uuid.NewString()
	connectionID, variableID, datapointID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	snapshot := repository.ProjectSnapshot{
		CollectorConnections: []repository.SnapshotCollectorConnectionRecord{{ID: connectionID, ProtocolFamily: "modbus", DriverID: "modbus.tcp", DriverVersion: "1.0.0", SchemaVersion: 1, IsEnabled: true, Config: map[string]any{"host": "10.0.0.8", "port": 502}, DefaultAcquisition: map[string]any{"intervalMs": 1000, "deadband": 0.0, "changeOnly": false}}},
		CollectorPoints:      []repository.SnapshotCollectorPointRecord{{ID: variableID, ConnectionID: connectionID, Address: map[string]any{"station": 1, "area": "holdingRegister", "address": 0}, AddressSchemaVersion: 1, DataType: "int16", ElementCount: 1, ReadOptions: map[string]any{}, AcquisitionMode: "inherit", Enabled: true}},
		DataPoints:           []repository.DataPointRecord{{ID: datapointID, ProjectID: projectID, Path: "collector.modbus.value", Name: "value", SourceType: "collector.point", SourceID: &variableID, SourceConfig: map[string]any{"connectionId": connectionID}, DataType: "int16", Tags: []any{}, AttributeDefaults: map[string]string{}, RuntimePermissions: repository.DefaultDataPointRuntimePermissions(), RefreshMode: "auto", Status: "active"}},
	}
	service := NewProjectSnapshotServiceWithClockAndSchemaRoot(nil, time.Now, snapshotArtifactSchemaRoot(t))
	capturedAt := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	without, err := service.BuildArtifactsFromSnapshot(projectID, tenantID, capturedAt, snapshot, nil)
	if err != nil || without.CollectorArtifact != nil || len(without.CollectorSourceSnapshot) != 0 {
		t.Fatalf("collector must be omitted when not requested: result=%#v err=%v", without, err)
	}
	with, err := service.BuildArtifactsFromSnapshot(projectID, tenantID, capturedAt, snapshot, &CollectorArtifactRequest{ArtifactID: "collector-version-1", Revision: 1, CollectorVersion: "1.0.0"})
	if err != nil {
		t.Fatalf("build collector artifact: %v", err)
	}
	if with.CollectorArtifact == nil || len(with.CollectorSourceSnapshot) == 0 {
		t.Fatalf("collector bundle incomplete: %#v", with)
	}
}
