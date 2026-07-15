package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/indu-forge/data_service/internal/collectorprotocol"
)

func TestCollectorCatalogListUsesServerPagination(t *testing.T) {
	service := newRepositoryCollectorCatalogService(t)
	result, err := service.List(ListCollectorDriversInput{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.List) != 2 || result.Pagination.Total != 4 || result.Pagination.TotalPages != 2 {
		t.Fatalf("unexpected page: %#v", result)
	}
}

func TestCollectorCatalogListFiltersProtocolAndTransport(t *testing.T) {
	service := newRepositoryCollectorCatalogService(t)
	result, err := service.List(ListCollectorDriversInput{Page: 1, PageSize: 20, ProtocolFamily: "modbus", Transport: "serial"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.List) != 1 || result.List[0].DriverID != "modbus.rtu" {
		t.Fatalf("unexpected result: %#v", result.List)
	}
}

func TestCollectorCatalogGetReturnsSchemas(t *testing.T) {
	service := newRepositoryCollectorCatalogService(t)
	result, err := service.Get("opcua.standard")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.ConnectionSchema) == 0 || len(result.AddressSchema) == 0 || len(result.UISchema) == 0 {
		t.Fatalf("schemas missing: %#v", result)
	}
}

func newRepositoryCollectorCatalogService(t *testing.T) *CollectorCatalogService {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", "..", "..", "runtime", "collector_protocols"))
	catalog, err := collectorprotocol.LoadCatalog(os.DirFS(root))
	if err != nil {
		t.Fatal(err)
	}
	return NewCollectorCatalogService(catalog)
}
