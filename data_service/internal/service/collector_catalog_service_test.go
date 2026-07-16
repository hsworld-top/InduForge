package service

import (
	"encoding/json"
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

func TestCollectorCatalogOpcUaUsesStructuredBilingualConnectionSchema(t *testing.T) {
	service := newRepositoryCollectorCatalogService(t)
	result, err := service.Get("opcua.standard")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Required   []string                  `json:"required"`
		Properties map[string]map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(result.ConnectionSchema, &schema); err != nil {
		t.Fatal(err)
	}
	if _, exists := schema.Properties["endpointUrl"]; exists {
		t.Fatal("OPC UA connection schema must not expose endpointUrl")
	}
	if schema.Properties["host"]["title"] != "设备 IP / 主机名（host）" || schema.Properties["port"]["title"] != "端口（port）" {
		t.Fatalf("OPC UA bilingual titles missing: %#v", schema.Properties)
	}
	labels, ok := schema.Properties["securityMode"]["x-induforge-enum-labels"].(map[string]any)
	if !ok || labels["None"] != "无签名和加密（None）" {
		t.Fatalf("OPC UA enum labels missing: %#v", schema.Properties["securityMode"])
	}
}

func TestCollectorDriverSummaryNormalizesOptionalCollections(t *testing.T) {
	summary := collectorDriverSummary(collectorprotocol.Manifest{})
	if summary.Transports == nil || summary.Operations == nil || summary.DataTypes == nil || summary.AcquisitionModes == nil || summary.Platforms == nil {
		t.Fatalf("collector driver summary contains nil collection: %#v", summary)
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
