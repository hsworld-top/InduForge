package postgres

import (
	"errors"
	"strings"
	"testing"
)

func TestSchemaIsSingleV1Baseline(t *testing.T) {
	for _, table := range []string{"schema_meta", "role_fence", "producer_fence", "processed_event", "consumer_checkpoint", "transactional_outbox", "processing_failure", "producer_sequence", "point_history", "point_current", "compute_input_snapshot", "compute_command_audit", "compute_trigger_state", "compute_schedule_state", "alarm_item_state", "alarm_ack_audit"} {
		if !strings.Contains(SchemaSQL, "runtime_engine."+table) {
			t.Fatalf("缺少表 %s", table)
		}
	}
	if strings.Contains(strings.ToUpper(SchemaSQL), "ALTER TABLE") {
		t.Fatal("V1 基线不应包含迁移 ALTER")
	}
}

func TestDiagnosticCodeDoesNotExposeConnectionDetails(t *testing.T) {
	if got := DiagnosticCode(errors.New("PostgreSQL 状态库基线校验失败: runtime_engine")); got != "SCHEMA" {
		t.Fatalf("schema diagnostic=%q", got)
	}
	if got := DiagnosticCode(errors.New("postgres://user:secret@host/database")); got != "UNKNOWN" {
		t.Fatalf("unknown diagnostic=%q", got)
	}
}

func TestRuntimeDLQIDContractVector(t *testing.T) {
	got, err := runtimeDLQID("deployment-line1-prod", "writer-raw-v1", "data.raw.22222222-2222-4222-8222-222222222222", "", FailurePermanentValidation, "ee268a71b05de985c2d7b596eede7bb289ad999b071882bc65e653b9c52e58a6")
	if err != nil {
		t.Fatal(err)
	}
	const want = "7fe0afbd96f4b358d599559934cd9caad251863c9f4276f781b08d939b8874db"
	if got != want {
		t.Fatalf("dlqId=%s, want %s", got, want)
	}
}
