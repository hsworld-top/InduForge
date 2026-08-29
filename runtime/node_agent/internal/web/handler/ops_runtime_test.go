package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/indu-forge/node_agent/internal/ops"
)

func TestOpsProcessesRejectsNonLoopback(t *testing.T) {
	h := createTestHandler(t).WithSupervisor(ops.NewSupervisor("unused", t.TempDir(), t.TempDir()))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/processes", nil)
	req.RemoteAddr = "192.0.2.10:1234"
	rec := httptest.NewRecorder()
	h.ListManagedProcesses(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected loopback protection, got %d", rec.Code)
	}
}

func TestOpsProcessesAllowsLoopback(t *testing.T) {
	h := createTestHandler(t).WithSupervisor(ops.NewSupervisor("unused", t.TempDir(), t.TempDir()))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/processes", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	h.ListManagedProcesses(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected loopback request to pass, got %d", rec.Code)
	}
}

func TestOpsProcessMutationRequiresConfirmationHeader(t *testing.T) {
	h := createTestHandler(t).WithSupervisor(ops.NewSupervisor("unused", t.TempDir(), t.TempDir()))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/processes/workload-1/start?role=compute", nil)
	req = mux.SetURLVars(req, map[string]string{"workloadId": "workload-1", "action": "start"})
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	h.ManageProcess(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected mutation without confirmation header to be forbidden, got %d", rec.Code)
	}
}
