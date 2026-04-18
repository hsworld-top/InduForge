package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/indu-forge/data_service/internal/http/response"
)

func TestWriteSuccess_WritesUnifiedApiResponse(t *testing.T) {
	rr := httptest.NewRecorder()

	response.WriteSuccess(rr, "req-123", map[string]any{
		"name": "alice",
	})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var payload response.ApiResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !payload.Success {
		t.Fatal("expected success to be true")
	}
	if payload.ErrorCode != "" {
		t.Fatalf("expected empty errorCode, got %q", payload.ErrorCode)
	}
	if payload.Message != "" {
		t.Fatalf("expected empty message, got %q", payload.Message)
	}
	if payload.RequestID != "req-123" {
		t.Fatalf("expected requestId %q, got %q", "req-123", payload.RequestID)
	}

	data, ok := payload.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected data to be object, got %T", payload.Data)
	}
	if data["name"] != "alice" {
		t.Fatalf("expected data.name %q, got %v", "alice", data["name"])
	}
}

func TestWriteError_WritesUnifiedApiResponse(t *testing.T) {
	rr := httptest.NewRecorder()

	response.WriteError(rr, http.StatusBadRequest, "req-456", "BAD_REQUEST", "参数错误")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var payload response.ApiResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if payload.Success {
		t.Fatal("expected success to be false")
	}
	if payload.ErrorCode != "BAD_REQUEST" {
		t.Fatalf("expected errorCode %q, got %q", "BAD_REQUEST", payload.ErrorCode)
	}
	if payload.Message != "参数错误" {
		t.Fatalf("expected message %q, got %q", "参数错误", payload.Message)
	}
	if payload.RequestID != "req-456" {
		t.Fatalf("expected requestId %q, got %q", "req-456", payload.RequestID)
	}
}
