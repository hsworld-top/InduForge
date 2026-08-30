package service

import (
	"testing"

	"github.com/indu-forge/data_service/internal/repository"
)

func TestComputeDefaultOutputPreservesUint64Precision(t *testing.T) {
	value := `9007199254740993`
	outputs, warnings, err := computeRunOutputValues(repository.ComputeUnitRecord{Outputs: []repository.ComputeOutputRecord{{OutputKey: "result", DataType: "uint64", NullPolicy: "default", DefaultValue: &value}}}, nil)
	if err != nil {
		t.Fatalf("apply default: %v", err)
	}
	if len(outputs) != 1 || outputs[0].Value != value || len(warnings) != 1 {
		t.Fatalf("default output was lossy: outputs=%+v warnings=%v", outputs, warnings)
	}
}
