package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
)

func TestRequireCapability_AllowsMatchingCapability(t *testing.T) {
	handler := middleware.RequireCapability("project:write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), &auth.Claims{
		UserID:       "user-1",
		Capabilities: []string{"project:read", "project:write"},
	}))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestRequireCapability_ReturnsApiResponseWhenMissing(t *testing.T) {
	handler := middleware.RequireCapability("project:write")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called when capability is missing")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), &auth.Claims{
		UserID:       "user-1",
		Capabilities: []string{"project:read"},
	}))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}

	var payload response.ApiResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if payload.Success {
		t.Fatal("expected success to be false")
	}
	if payload.ErrorCode != "PERMISSION_INSUFFICIENT" {
		t.Fatalf("expected errorCode %q, got %q", "PERMISSION_INSUFFICIENT", payload.ErrorCode)
	}
	if payload.Message != "权限不足" {
		t.Fatalf("expected message %q, got %q", "权限不足", payload.Message)
	}
}
