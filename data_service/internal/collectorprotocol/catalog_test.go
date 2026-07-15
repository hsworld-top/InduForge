package collectorprotocol

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadCatalogRejectsDriverDirectoryMismatch(t *testing.T) {
	_, err := LoadCatalog(fstest.MapFS{
		"wrong/manifest.json":          {Data: []byte(validManifest("opcua.standard", "opcua", []string{"connection.test"}))},
		"wrong/connection.schema.json": {Data: []byte(validObjectSchema())},
		"wrong/address.schema.json":    {Data: []byte(validObjectSchema())},
		"wrong/ui.schema.json":         {Data: []byte(`{"order":[]}`)},
	})
	if err == nil || !strings.Contains(err.Error(), "目录名必须等于 driverId") {
		t.Fatalf("expected directory mismatch error, got %v", err)
	}
}

func TestLoadCatalogRejectsUnknownDataType(t *testing.T) {
	manifest := strings.Replace(validManifest("opcua.standard", "opcua", []string{"connection.test"}), `"bool"`, `"object"`, 1)
	_, err := LoadCatalog(completeDriverFS("opcua.standard", manifest))
	if err == nil || !strings.Contains(err.Error(), "不支持的数据类型") {
		t.Fatalf("expected data type error, got %v", err)
	}
}

func TestLoadCatalogRejectsForbiddenPublicTerm(t *testing.T) {
	manifest := strings.Replace(validManifest("opcua.standard", "opcua", []string{"connection.test"}), `"displayName":"OPC UA"`, `"displayName":"HSL OPC UA"`, 1)
	_, err := LoadCatalog(completeDriverFS("opcua.standard", manifest))
	if err == nil || !strings.Contains(err.Error(), "公共协议契约包含禁止词") {
		t.Fatalf("expected forbidden term error, got %v", err)
	}
}

func TestLoadCatalogLoadsCertifiedDrivers(t *testing.T) {
	catalog, err := LoadCatalog(completeDriverFS("opcua.standard", validManifest("opcua.standard", "opcua", []string{"connection.test", "device.browse", "point.read"})))
	if err != nil {
		t.Fatal(err)
	}
	driver, ok := catalog.Driver("opcua.standard")
	if !ok || driver.Manifest.SchemaVersion != 1 || len(driver.Manifest.Operations) != 3 {
		t.Fatalf("unexpected driver definition: %#v", driver)
	}
}

func TestRepositoryCatalogLoadsFourCertifiedDrivers(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", "runtime", "collector_protocols"))
	catalog, err := LoadCatalog(os.DirFS(root))
	if err != nil {
		t.Fatal(err)
	}
	if drivers := catalog.Drivers(); len(drivers) != 4 {
		t.Fatalf("expected four certified drivers, got %d", len(drivers))
	}
}

func completeDriverFS(driverID, manifest string) fstest.MapFS {
	return fstest.MapFS{
		driverID + "/manifest.json":          {Data: []byte(manifest)},
		driverID + "/connection.schema.json": {Data: []byte(validObjectSchema())},
		driverID + "/address.schema.json":    {Data: []byte(validObjectSchema())},
		driverID + "/ui.schema.json":         {Data: []byte(`{"order":[]}`)},
	}
}

func validManifest(driverID, family string, operations []string) string {
	return `{
		"protocolFamily":"` + family + `",
		"driverId":"` + driverID + `",
		"driverVersion":"1.0.0",
		"schemaVersion":1,
		"displayName":"OPC UA",
		"category":"industrial",
		"transports":["tcp"],
		"operations":["` + strings.Join(operations, `","`) + `"],
		"dataTypes":["bool","int16","uint16","int32","uint32","float32","float64","string","bytes","datetime"],
		"acquisitionModes":["polling"],
		"platforms":{"devAgent":["windows-x64"],"runtime":[]}
	}`
}

func validObjectSchema() string {
	return `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,"properties":{}}`
}
