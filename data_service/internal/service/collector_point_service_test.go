package service

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/indu-forge/data_service/internal/collectorprotocol"
	"github.com/indu-forge/data_service/internal/repository"
)

func TestToCollectorPointMapsLatestDebugSnapshot(t *testing.T) {
	valueText := "12.5"
	dataType := "float64"
	quality := "Good"
	errorMessage := "上次读取失败"
	readAt := time.Date(2026, 7, 20, 10, 0, 0, 0, time.Local)
	attemptAt := readAt.Add(time.Minute)
	point := toCollectorPoint(repository.CollectorPointRecord{
		ID: "point-1", Address: map[string]any{}, ReadOptions: map[string]any{}, Acquisition: map[string]any{}, Metadata: map[string]any{},
		LatestDebugSnapshot: &repository.CollectorPointDebugSnapshotRecord{
			Value: 12.5, ValueText: &valueText, DataType: &dataType, Quality: &quality, ReadAt: &readAt,
			LastAttemptStatus: "failed", LastAttemptAt: attemptAt, LastErrorMessage: &errorMessage,
		},
	})
	if point.LatestDebugSnapshot == nil {
		t.Fatal("expected latest debug snapshot")
	}
	if point.LatestDebugSnapshot.Quality == nil || *point.LatestDebugSnapshot.Quality != "Good" {
		t.Fatalf("unexpected snapshot: %#v", point.LatestDebugSnapshot)
	}
	if point.LatestDebugSnapshot.LastAttemptAt != "2026-07-20 10:01:00" {
		t.Fatalf("lastAttemptAt = %s", point.LatestDebugSnapshot.LastAttemptAt)
	}
}

func TestCollectorPointServiceCreatesOpcUaAddressText(t *testing.T) {
	store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: "opcua.standard", SchemaVersion: 1}}
	service := newRepositoryCollectorPointService(t, store)
	_, err := service.CreatePointsBatch(context.Background(), "550e8400-e29b-41d4-a716-446655440000", store.connection.ID, "550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{{Code: "temperature", Name: "温度", Address: map[string]any{"nodeId": "ns=2;s=Temperature"}, DataType: "float32", ElementCount: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(store.created) != 1 || store.created[0].AddressText != "ns=2;s=Temperature" {
		t.Fatalf("unexpected params: %#v", store.created)
	}
}

func TestCollectorPointServiceValidatesModbusAddressAndDataTypeCombinations(t *testing.T) {
	tests := []struct {
		name        string
		address     map[string]any
		dataType    string
		wantFailure string
	}{
		{name: "coil rejects numeric type", address: map[string]any{"station": 1, "area": "coil", "address": 10}, dataType: "int16", wantFailure: "线圈和离散输入只支持 bool 数据类型"},
		{name: "coil rejects bit index", address: map[string]any{"station": 1, "area": "coil", "address": 10, "bitIndex": 1}, dataType: "bool", wantFailure: "线圈和离散输入不能配置寄存器位索引"},
		{name: "register bool requires bit index", address: map[string]any{"station": 1, "area": "holdingRegister", "address": 10}, dataType: "bool", wantFailure: "寄存器 bool 变量必须配置位索引"},
		{name: "register numeric rejects bit index", address: map[string]any{"station": 1, "area": "holdingRegister", "address": 10, "bitIndex": 1}, dataType: "int16", wantFailure: "只有寄存器 bool 变量可以配置位索引"},
		{name: "holding register accepts numeric type", address: map[string]any{"station": 1, "area": "holdingRegister", "address": 10}, dataType: "int16"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{
				ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000",
				DriverID: "modbus.tcp", DriverVersion: "1.0.0", SchemaVersion: 2,
			}}
			service := newRepositoryCollectorPointService(t, store)
			result, err := service.CreatePointsBatch(
				context.Background(),
				store.connection.ProjectID,
				store.connection.ID,
				"550e8400-e29b-41d4-a716-446655440001",
				[]CreateCollectorPointInput{{Name: "测试变量", Address: tt.address, DataType: tt.dataType, ElementCount: 1}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantFailure == "" {
				if len(result.List) != 1 || len(result.Failed) != 0 {
					t.Fatalf("合法变量未创建: %+v", result)
				}
				return
			}
			if len(result.List) != 0 || len(result.Failed) != 1 || !strings.Contains(result.Failed[0].Message, tt.wantFailure) {
				t.Fatalf("非法组合未返回预期错误 %q: %+v", tt.wantFailure, result)
			}
		})
	}
}

func TestCollectorPointServiceFindsExistingAddressIndexes(t *testing.T) {
	store := &fakeCollectorPointStore{
		connection:           repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: "opcua.standard", SchemaVersion: 1},
		existingAddressTexts: []string{"ns=2;s=Temperature"},
	}
	service := newRepositoryCollectorPointService(t, store)
	indexes, err := service.FindExistingPointAddressIndexes(context.Background(), store.connection.ProjectID, store.connection.ID, []map[string]any{
		{"nodeId": "ns=2;s=Temperature"},
		{"nodeId": "ns=2;s=Pressure"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(indexes) != 1 || indexes[0] != 0 {
		t.Fatalf("unexpected indexes: %#v", indexes)
	}
}

func TestCollectorPointServiceReturnsInvalidAddressAsBatchFailure(t *testing.T) {
	store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: "opcua.standard", SchemaVersion: 1}}
	service := newRepositoryCollectorPointService(t, store)
	result, err := service.CreatePointsBatch(context.Background(), "550e8400-e29b-41d4-a716-446655440000", store.connection.ID, "550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{{Code: "temperature", Name: "温度", Address: map[string]any{}, DataType: "float32", ElementCount: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.List) != 0 || len(result.Failed) != 1 || result.Failed[0].Code != "VALIDATION_FAILED" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestCollectorPointServiceRejectsMalformedOpcUaNodeID(t *testing.T) {
	store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: "opcua.standard", SchemaVersion: 2}}
	service := newRepositoryCollectorPointService(t, store)

	result, err := service.CreatePointsBatch(context.Background(), store.connection.ProjectID, store.connection.ID, "550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{{Name: "LastChange", Address: map[string]any{"nodeId": "111"}, DataType: "bool", ElementCount: 1}})

	if err != nil {
		t.Fatal(err)
	}
	if len(result.List) != 0 || len(result.Failed) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if !strings.Contains(result.Failed[0].Message, "OPC UA NodeId 格式无效") {
		t.Fatalf("unexpected failure: %#v", result.Failed[0])
	}
	if len(store.created) != 0 {
		t.Fatalf("invalid point must not be persisted: %#v", store.created)
	}
}

func TestCollectorPointServiceCreatesValidRowsAndReturnsDuplicateFailures(t *testing.T) {
	store := &fakeCollectorPointStore{
		connection:            repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: "opcua.standard", SchemaVersion: 1},
		existingConflictNames: []string{"existing"},
	}
	service := newRepositoryCollectorPointService(t, store)
	result, err := service.CreatePointsBatch(context.Background(), store.connection.ProjectID, store.connection.ID, "550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{
		{Name: "Temperature", Address: map[string]any{"nodeId": "ns=2;s=Temperature"}, DataType: "float32", ElementCount: 1},
		{Name: "temperature", Address: map[string]any{"nodeId": "ns=2;s=Temperature2"}, DataType: "float32", ElementCount: 1},
		{Name: "Existing", Address: map[string]any{"nodeId": "ns=2;s=Existing"}, DataType: "float32", ElementCount: 1},
		{Name: "Pressure", Address: map[string]any{"nodeId": "ns=2;s=Pressure"}, DataType: "float32", ElementCount: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.List) != 2 || len(result.Failed) != 2 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Failed[0].Index != 1 || result.Failed[0].Code != "DUPLICATE_NAME" || result.Failed[1].Index != 2 {
		t.Fatalf("unexpected failures: %#v", result.Failed)
	}
}

func TestCollectorPointServiceAllowsLaterValidDuplicateNameCandidate(t *testing.T) {
	store := &fakeCollectorPointStore{
		connection:                   repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: "opcua.standard", SchemaVersion: 1},
		existingConflictAddressTexts: []string{"ns=2;s=ExistingAddress"},
	}
	pointService := newRepositoryCollectorPointService(t, store)
	result, err := pointService.CreatePointsBatch(context.Background(), store.connection.ProjectID, store.connection.ID, "550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{
		{Name: "Temperature", Address: map[string]any{"nodeId": "ns=2;s=ExistingAddress"}, DataType: "float32", ElementCount: 1},
		{Name: "temperature", Address: map[string]any{"nodeId": "ns=2;s=ValidAddress"}, DataType: "float32", ElementCount: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.List) != 1 || result.List[0].AddressText != "ns=2;s=ValidAddress" {
		t.Fatalf("unexpected created points: %#v", result.List)
	}
	if len(result.Failed) != 1 || result.Failed[0].Index != 0 || result.Failed[0].Code != "DUPLICATE_ADDRESS" {
		t.Fatalf("unexpected failures: %#v", result.Failed)
	}
}

func TestCollectorPointExportWritesReimportableCSV(t *testing.T) {
	description := "现场温度"
	store := &fakeCollectorPointStore{
		connection: repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", Name: "锅炉 OPC UA", DriverID: "opcua.standard", SchemaVersion: 1},
		exportRecords: []repository.CollectorPointExportRecord{{
			CollectorPointRecord: repository.CollectorPointRecord{
				ID: "550e8400-e29b-41d4-a716-446655440003", Name: "温度", Description: &description,
				Address: map[string]any{"nodeId": "ns=2;s=Temperature"}, DataType: "float32",
				ElementCount: 1, Enabled: true, ReadOptions: map[string]any{}, Acquisition: map[string]any{"intervalMs": 1000}, Metadata: map[string]any{},
			},
			GroupPath: "锅炉/温度",
		}},
	}
	pointService := newRepositoryCollectorPointService(t, store)
	plan, err := pointService.PreparePointExport(context.Background(), store.connection.ProjectID, store.connection.ID, CollectorPointExportRequest{Format: "csv", Scope: "current_page", Page: 1, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	buffer := bytes.NewBuffer(nil)
	if err := plan.WriteCSV(context.Background(), buffer); err != nil {
		t.Fatal(err)
	}
	payload := buffer.String()
	for _, expected := range []string{"\ufeffgroupPath,name,description,dataType,elementCount,enabled,address.nodeId", "锅炉/温度,温度,现场温度,float32,1,true,ns=2;s=Temperature"} {
		if !strings.Contains(payload, expected) {
			t.Fatalf("CSV 缺少内容 %q: %s", expected, payload)
		}
	}
}

func newRepositoryCollectorPointService(t *testing.T, store CollectorPointStore) *CollectorPointService {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", "..", "..", "runtime", "collector_protocols"))
	catalog, err := collectorprotocol.LoadCatalog(os.DirFS(root))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewCollectorPointService(store, catalog)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

type fakeCollectorPointStore struct {
	connection                   repository.CollectorConnectionRecord
	created                      []repository.CreateCollectorPointParams
	existingAddressTexts         []string
	existingConflictNames        []string
	existingConflictAddressTexts []string
	exportRecords                []repository.CollectorPointExportRecord
}

func (f *fakeCollectorPointStore) GetConnection(context.Context, string, string) (*repository.CollectorConnectionRecord, error) {
	return &f.connection, nil
}
func (f *fakeCollectorPointStore) ListPointGroups(context.Context, string, string, *string) ([]repository.CollectorPointGroupRecord, error) {
	return nil, nil
}
func (f *fakeCollectorPointStore) CreatePointGroup(context.Context, repository.CreateCollectorPointGroupParams) (*repository.CollectorPointGroupRecord, error) {
	return nil, nil
}
func (f *fakeCollectorPointStore) UpdatePointGroup(context.Context, repository.UpdateCollectorPointGroupParams) (*repository.CollectorPointGroupRecord, error) {
	return nil, nil
}
func (f *fakeCollectorPointStore) DeletePointGroup(context.Context, string, string, string) error {
	return nil
}
func (f *fakeCollectorPointStore) ListPoints(context.Context, string, string, repository.CollectorPointListFilter) ([]repository.CollectorPointRecord, int, error) {
	return nil, 0, nil
}
func (f *fakeCollectorPointStore) ListPointCodes(context.Context, string, string) ([]string, error) {
	return nil, nil
}
func (f *fakeCollectorPointStore) ListExistingPointAddressTexts(context.Context, string, string, []string) ([]string, error) {
	return f.existingAddressTexts, nil
}
func (f *fakeCollectorPointStore) ListExistingPointConflicts(context.Context, string, string, []string, []string) (repository.CollectorPointExistingConflicts, error) {
	return repository.CollectorPointExistingConflicts{Names: f.existingConflictNames, AddressTexts: f.existingConflictAddressTexts}, nil
}
func (f *fakeCollectorPointStore) StreamPointsForExport(_ context.Context, _ string, _ string, _ repository.CollectorPointExportFilter, visit func(repository.CollectorPointExportRecord) error) error {
	for _, record := range f.exportRecords {
		if err := visit(record); err != nil {
			return err
		}
	}
	return nil
}
func (f *fakeCollectorPointStore) GetPointsByIDs(context.Context, string, string, []string) ([]repository.CollectorPointRecord, error) {
	return nil, nil
}
func (f *fakeCollectorPointStore) CreatePointsBatch(_ context.Context, params []repository.CreateCollectorPointParams) ([]repository.CollectorPointRecord, error) {
	f.created = params
	records := make([]repository.CollectorPointRecord, 0, len(params))
	for _, item := range params {
		records = append(records, repository.CollectorPointRecord{
			ID: item.ID, ProjectID: item.ProjectID, ConnectionID: item.ConnectionID, GroupID: item.GroupID,
			Code: item.Code, Name: item.Name, Description: item.Description, Address: item.Address,
			AddressText: item.AddressText, AddressSchemaVersion: item.AddressSchemaVersion, DataType: item.DataType,
			ElementCount: item.ElementCount, ReadOptions: item.ReadOptions, Acquisition: item.Acquisition,
			Enabled: item.Enabled, SortOrder: item.SortOrder, Metadata: item.Metadata,
		})
	}
	return records, nil
}
func (f *fakeCollectorPointStore) UpdatePointsBatch(context.Context, []repository.UpdateCollectorPointParams) ([]repository.CollectorPointRecord, error) {
	return nil, nil
}
func (f *fakeCollectorPointStore) DeletePointsBatch(context.Context, string, string, []string) error {
	return nil
}
func (f *fakeCollectorPointStore) MovePointsBatch(context.Context, string, string, *string, []string) error {
	return nil
}
