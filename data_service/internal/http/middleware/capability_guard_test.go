package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
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
		ProjectIDs:   []string{"project-a"},
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
		ProjectIDs:   []string{"project-a"},
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
	if payload.ErrorCode != string(apperrors.ErrorCodePermissionInsufficient) {
		t.Fatalf("expected errorCode %q, got %q", apperrors.ErrorCodePermissionInsufficient, payload.ErrorCode)
	}
	if payload.Message != "权限不足" {
		t.Fatalf("expected message %q, got %q", "权限不足", payload.Message)
	}
}

func TestRequireCapability_RejectsEmptyRequiredCapability(t *testing.T) {
	handler := middleware.RequireCapability("")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called when required capability is empty")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), &auth.Claims{
		UserID:       "user-1",
		Capabilities: []string{"project:read"},
		ProjectIDs:   []string{"project-a"},
	}))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var payload response.ApiResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if payload.ErrorCode != string(apperrors.ErrorCodeBadRequest) {
		t.Fatalf("expected errorCode %q, got %q", apperrors.ErrorCodeBadRequest, payload.ErrorCode)
	}
}

func TestRequireCapability_RejectsCrossProjectAccess(t *testing.T) {
	handler := middleware.RequireCapability("project:write")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called when project boundary is violated")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/projects/project-b", nil)
	req = mux.SetURLVars(req, map[string]string{"projectId": "project-b"})
	req = req.WithContext(auth.WithClaims(req.Context(), &auth.Claims{
		UserID:       "user-1",
		Capabilities: []string{"project:write"},
		ProjectIDs:   []string{"project-a"},
	}))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}

	var payload response.ApiResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if payload.ErrorCode != string(apperrors.ErrorCodePermissionProjectMismatch) {
		t.Fatalf("expected errorCode %q, got %q", apperrors.ErrorCodePermissionProjectMismatch, payload.ErrorCode)
	}
	if payload.Message != "项目范围不足" {
		t.Fatalf("expected message %q, got %q", "项目范围不足", payload.Message)
	}
}
