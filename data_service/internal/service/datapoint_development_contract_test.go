package service

import (
	"testing"
	"time"

	"github.com/indu-forge/data_service/internal/repository"
)

func TestBuildDataPointDevelopmentContract_OnlyExposesPublicContract(t *testing.T) {
	updatedAt := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
	description := "车间温度"
	contract := buildDataPointDevelopmentContract(repository.DataPointRecord{
		ID:          "internal-id",
		Path:        "workshop.temperature",
		Name:        "温度",
		Description: &description,
		DataType:    "number",
		Status:      "active",
		SourceType:  "db.query",
		SourceConfig: map[string]any{
			"sql": "SELECT secret_value FROM private_table",
		},
		RefreshMode: "subscription",
		UpdatedAt:   updatedAt,
	})

	if contract.ID != "workshop.temperature" {
		t.Fatalf("contract ID = %q, want point path", contract.ID)
	}
	if contract.UpdatedAt != updatedAt {
		t.Fatalf("contract updatedAt = %s, want %s", contract.UpdatedAt, updatedAt)
	}
	if len(contract.Methods) != 3 || contract.Methods[0].Name != "get" || contract.Methods[1].Name != "set" || contract.Methods[2].Name != "sub" {
		t.Fatalf("methods = %#v, want get/set/sub", contract.Methods)
	}
}

func TestDevelopmentContractVersion_IsStableForEquivalentContracts(t *testing.T) {
	contracts := []DataPointDevelopmentContract{{ID: "point.a", UpdatedAt: time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)}}
	first := developmentContractVersion(contracts)
	second := developmentContractVersion(contracts)
	if first == "" || first != second {
		t.Fatalf("versions must be stable, got %q and %q", first, second)
	}
}
