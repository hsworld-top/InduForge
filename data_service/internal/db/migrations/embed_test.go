package migrations

import (
	"strings"
	"testing"
)

func TestEmbeddedMigrationsContainUnifiedCollectorModel(t *testing.T) {
	payload, err := Files.ReadFile("0060_collector_unified_model.sql")
	if err != nil {
		t.Fatal(err)
	}
	migration := string(payload)
	for _, table := range []string{
		"data_collector_connections",
		"data_collector_connection_secrets",
		"data_collector_point_groups",
		"data_collector_points",
		"data_collector_import_sessions",
	} {
		if !strings.Contains(migration, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Fatalf("missing table %s", table)
		}
	}
	for _, constraint := range []string{
		"data_collector_connections_driver_id_check",
		"data_collector_points_data_type_check",
		"data_collector_points_element_count_check",
	} {
		if !strings.Contains(migration, "CONSTRAINT "+constraint) {
			t.Fatalf("missing constraint %s", constraint)
		}
	}
}

func TestEmbeddedMigrationsContainUnifiedCollectorModelDown(t *testing.T) {
	payload, err := Files.ReadFile("0060_collector_unified_model_down.sql")
	if err != nil {
		t.Fatal(err)
	}
	down := string(payload)
	for _, table := range []string{
		"data_collector_points",
		"data_collector_import_sessions",
		"data_collector_point_groups",
		"data_collector_connection_secrets",
		"data_collector_connections",
	} {
		if !strings.Contains(down, "DROP TABLE IF EXISTS "+table) {
			t.Fatalf("missing down table %s", table)
		}
	}
}

func TestEmbeddedMigrationsContainLegacyCollectorConversion(t *testing.T) {
	payload, err := Files.ReadFile("0061_collector_legacy_migration.sql")
	if err != nil {
		t.Fatal(err)
	}
	migration := string(payload)
	for _, driverID := range []string{"opcua.standard", "modbus.tcp", "modbus.rtu", "siemens.s7-tcp"} {
		if !strings.Contains(migration, driverID) {
			t.Fatalf("missing legacy driver conversion %s", driverID)
		}
	}
	if !strings.Contains(migration, "source_type = 'collector.point'") {
		t.Fatal("missing legacy data point source conversion")
	}
}
