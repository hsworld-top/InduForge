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

func TestLoadCatalogRejectsUnknownFeature(t *testing.T) {
	manifest := strings.Replace(validManifest("opcua.standard", "opcua", []string{"connection.test"}), `"operations":["connection.test"]`, `"operations":["connection.test"],"features":["point.unknown"]`, 1)
	_, err := LoadCatalog(completeDriverFS("opcua.standard", manifest))
	if err == nil || !strings.Contains(err.Error(), "不支持的驱动特性") {
		t.Fatalf("expected feature error, got %v", err)
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

func TestRepositoryCatalogLoadsCertifiedDrivers(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", "runtime", "collector_protocols"))
	catalog, err := LoadCatalog(os.DirFS(root))
	if err != nil {
		t.Fatal(err)
	}
	if drivers := catalog.Drivers(); len(drivers) != 115 {
		t.Fatalf("expected 115 certified drivers, got %d", len(drivers))
	}
	if _, ok := catalog.Driver("mitsubishi.mc-3e-tcp"); !ok {
		t.Fatal("expected Mitsubishi MC 3E TCP driver")
	}
	if _, ok := catalog.Driver("omron.fins-tcp"); !ok {
		t.Fatal("expected Omron FINS TCP driver")
	}
	if _, ok := catalog.Driver("omron.fins-udp"); !ok {
		t.Fatal("expected Omron FINS UDP driver")
	}
	for _, driverID := range []string{"omron.cip", "omron.connected-cip", "omron.hostlink-cmode", "omron.hostlink-cmode-over-tcp"} {
		if _, ok := catalog.Driver(driverID); !ok {
			t.Fatalf("expected %s driver", driverID)
		}
	}
	for _, driverID := range []string{"freedom.tcp", "freedom.udp", "freedom.serial", "cimon.hmi-protocol"} {
		if _, ok := catalog.Driver(driverID); !ok {
			t.Fatalf("expected %s driver", driverID)
		}
	}
	for _, driverID := range []string{"yamatake.digitron-serial", "yamatake.digitron-tcp", "yaskawa.memobus-tcp", "yaskawa.memobus-udp"} {
		if _, ok := catalog.Driver(driverID); !ok {
			t.Fatalf("expected %s driver", driverID)
		}
	}
	for _, driverID := range []string{"oriental-motor.eip", "toyo.puc", "turck.reader-tcp"} {
		if _, ok := catalog.Driver(driverID); !ok {
			t.Fatalf("expected %s driver", driverID)
		}
	}
	for _, driverID := range []string{"cjt188.serial", "cjt188.tcp", "rkc.temperature-controller-serial", "rkc.temperature-controller-tcp"} {
		if _, ok := catalog.Driver(driverID); !ok {
			t.Fatalf("expected %s driver", driverID)
		}
	}
	for _, driverID := range []string{"dam3601.serial", "yudian.ai-bus", "delixi.dtsu6606"} {
		if _, ok := catalog.Driver(driverID); !ok {
			t.Fatalf("expected %s driver", driverID)
		}
	}
	for _, driverID := range []string{"dlt645.2007-serial", "dlt645.2007-over-tcp", "dlt645.1997-serial", "dlt645.1997-over-tcp", "dlt698.serial", "dlt698.over-tcp", "dlt698.tcp-net"} {
		if _, ok := catalog.Driver(driverID); !ok {
			t.Fatalf("expected %s driver", driverID)
		}
	}
	if _, ok := catalog.Driver("dcs.nanjing-auto"); !ok {
		t.Fatal("expected dcs.nanjing-auto driver")
	}
	for _, driverID := range []string{"robot.estun-tcp", "robot.fanuc-interface"} {
		if _, ok := catalog.Driver(driverID); !ok {
			t.Fatalf("expected %s driver", driverID)
		}
	}
	for _, driverID := range []string{"siemens.s7-plus", "ec-fan.machine-serial", "mqtt.rpc-device"} {
		if _, ok := catalog.Driver(driverID); !ok {
			t.Fatalf("expected %s driver", driverID)
		}
	}
	if _, ok := catalog.Driver("allen-bradley.ethernet-ip"); !ok {
		t.Fatal("expected Allen-Bradley EtherNet/IP driver")
	}
	if _, ok := catalog.Driver("beckhoff.ads-tcp"); !ok {
		t.Fatal("expected Beckhoff ADS TCP driver")
	}
	if _, ok := catalog.Driver("megmeet.tcp"); !ok {
		t.Fatal("expected MegMeet TCP driver")
	}
	if _, ok := catalog.Driver("fuji.sph-tcp"); !ok {
		t.Fatal("expected Fuji SPH TCP driver")
	}
	if _, ok := catalog.Driver("vigor.serial-over-tcp"); !ok {
		t.Fatal("expected Vigor serial over TCP driver")
	}
	if _, ok := catalog.Driver("yokogawa.link-tcp"); !ok {
		t.Fatal("expected Yokogawa Link TCP driver")
	}
	if _, ok := catalog.Driver("modbus.rtu-over-tcp"); !ok {
		t.Fatal("expected Modbus RTU over TCP driver")
	}
	if _, ok := catalog.Driver("modbus.ascii"); !ok {
		t.Fatal("expected Modbus ASCII driver")
	}
	if _, ok := catalog.Driver("iec.60870-5-104"); !ok {
		t.Fatal("expected IEC 60870-5-104 driver")
	}
	if _, ok := catalog.Driver("panasonic.mewtocol-tcp"); !ok {
		t.Fatal("expected Panasonic Mewtocol TCP driver")
	}
	if _, ok := catalog.Driver("panasonic.mewtocol-serial"); !ok {
		t.Fatal("expected Panasonic Mewtocol serial driver")
	}
	if _, ok := catalog.Driver("panasonic.mc-binary-tcp"); !ok {
		t.Fatal("expected Panasonic MC Binary TCP driver")
	}
	if _, ok := catalog.Driver("lsis.fast-enet"); !ok {
		t.Fatal("expected LSIS Fast Enet driver")
	}
	if _, ok := catalog.Driver("ge.srtp-tcp"); !ok {
		t.Fatal("expected GE SRTP TCP driver")
	}
	if _, ok := catalog.Driver("inovance.modbus-tcp"); !ok {
		t.Fatal("expected Inovance Modbus TCP driver")
	}
	for _, driverID := range []string{"inovance.modbus-serial", "inovance.modbus-rtu-over-tcp"} {
		if _, ok := catalog.Driver(driverID); !ok {
			t.Fatalf("expected %s driver", driverID)
		}
	}
	for _, driverID := range []string{"inovance.connected-cip", "inovance.easy-net", "inovance.computer-link"} {
		if _, ok := catalog.Driver(driverID); !ok {
			t.Fatalf("expected %s driver", driverID)
		}
	}
	if _, ok := catalog.Driver("fatek.program-tcp"); !ok {
		t.Fatal("expected Fatek Program TCP driver")
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
