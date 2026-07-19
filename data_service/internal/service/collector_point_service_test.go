package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/indu-forge/data_service/internal/collectorprotocol"
	"github.com/indu-forge/data_service/internal/repository"
)

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

func TestCollectorPointServiceRejectsInvalidAddress(t *testing.T) {
	store := &fakeCollectorPointStore{connection: repository.CollectorConnectionRecord{ID: "550e8400-e29b-41d4-a716-446655440002", ProjectID: "550e8400-e29b-41d4-a716-446655440000", DriverID: "opcua.standard", SchemaVersion: 1}}
	service := newRepositoryCollectorPointService(t, store)
	_, err := service.CreatePointsBatch(context.Background(), "550e8400-e29b-41d4-a716-446655440000", store.connection.ID, "550e8400-e29b-41d4-a716-446655440001", []CreateCollectorPointInput{{Code: "temperature", Name: "温度", Address: map[string]any{}, DataType: "float32", ElementCount: 1}})
	if err == nil {
		t.Fatal("expected invalid address")
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
	connection           repository.CollectorConnectionRecord
	created              []repository.CreateCollectorPointParams
	existingAddressTexts []string
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
func (f *fakeCollectorPointStore) GetPointsByIDs(context.Context, string, string, []string) ([]repository.CollectorPointRecord, error) {
	return nil, nil
}
func (f *fakeCollectorPointStore) CreatePointsBatch(_ context.Context, params []repository.CreateCollectorPointParams) ([]repository.CollectorPointRecord, error) {
	f.created = params
	return make([]repository.CollectorPointRecord, len(params)), nil
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
