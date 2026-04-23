package response_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/indu-forge/data_service/internal/errors"
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

	var raw map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to unmarshal raw response: %v", err)
	}

	if payload.Code != 0 {
		t.Fatalf("expected code %d, got %d", 0, payload.Code)
	}
	if payload.Msg == "" {
		t.Fatal("expected msg to be non-empty")
	}
	if payload.ReqID != "req-123" {
		t.Fatalf("expected reqId %q, got %q", "req-123", payload.ReqID)
	}
	if _, exists := raw["success"]; exists {
		t.Fatal("expected legacy field success to be removed")
	}
	if _, exists := raw["errorCode"]; exists {
		t.Fatal("expected legacy field errorCode to be removed")
	}
	if _, exists := raw["message"]; exists {
		t.Fatal("expected legacy field message to be removed")
	}
	if _, exists := raw["requestId"]; exists {
		t.Fatal("expected legacy field requestId to be removed")
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

	var raw map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to unmarshal raw response: %v", err)
	}

	if payload.Code != 20001 {
		t.Fatalf("expected code %d, got %d", 20001, payload.Code)
	}
	if payload.Msg != "参数错误" {
		t.Fatalf("expected msg %q, got %q", "参数错误", payload.Msg)
	}
	if payload.Data != nil {
		t.Fatalf("expected error data to be nil, got %#v", payload.Data)
	}
	if payload.ReqID != "req-456" {
		t.Fatalf("expected reqId %q, got %q", "req-456", payload.ReqID)
	}
	if _, exists := raw["success"]; exists {
		t.Fatal("expected legacy field success to be removed")
	}
	if _, exists := raw["errorCode"]; exists {
		t.Fatal("expected legacy field errorCode to be removed")
	}
	if _, exists := raw["message"]; exists {
		t.Fatal("expected legacy field message to be removed")
	}
	if _, exists := raw["requestId"]; exists {
		t.Fatal("expected legacy field requestId to be removed")
	}
}

func TestWriteAppError_UsesStronglyTypedErrorCode(t *testing.T) {
	rr := httptest.NewRecorder()

	response.WriteAppError(rr, http.StatusForbidden, "req-789", apperrors.ErrorCodePermissionInsufficient, "权限不足")

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}

	var payload response.ApiResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if payload.Code != apperrors.PublicCodePermissionInsufficient {
		t.Fatalf("expected code %d, got %d", apperrors.PublicCodePermissionInsufficient, payload.Code)
	}
	if payload.Msg != "权限不足" {
		t.Fatalf("expected msg %q, got %q", "权限不足", payload.Msg)
	}
	if payload.Data != nil {
		t.Fatalf("expected error data to be nil, got %#v", payload.Data)
	}
	if payload.ReqID != "req-789" {
		t.Fatalf("expected reqId %q, got %q", "req-789", payload.ReqID)
	}
}
