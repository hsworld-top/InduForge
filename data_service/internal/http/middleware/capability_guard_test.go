package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
	if payload.Code != apperrors.PublicCodePermissionInsufficient {
		t.Fatalf("expected code %d, got %d", apperrors.PublicCodePermissionInsufficient, payload.Code)
	}
	if payload.Msg != "权限不足" {
		t.Fatalf("expected msg %q, got %q", "权限不足", payload.Msg)
	}
	if payload.Data != nil {
		t.Fatalf("expected error data to be nil, got %#v", payload.Data)
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
	if payload.Code != apperrors.PublicCodeBadRequest {
		t.Fatalf("expected code %d, got %d", apperrors.PublicCodeBadRequest, payload.Code)
	}
}

func TestRequireCapability_RejectsCrossProjectAccess(t *testing.T) {
	handler := middleware.RequireCapability("project:write")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called when project boundary is violated")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/data/projects/project-b/assets", nil)
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
	if payload.Code != apperrors.PublicCodePermissionProjectMismatch {
		t.Fatalf("expected code %d, got %d", apperrors.PublicCodePermissionProjectMismatch, payload.Code)
	}
	if payload.Msg != "项目范围不足" {
		t.Fatalf("expected msg %q, got %q", "项目范围不足", payload.Msg)
	}
}

func TestRequireCapability_RejectsProjectRouteWithoutProjectID(t *testing.T) {
	handler := middleware.RequireCapability("project:write")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called when project id is missing")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/data/projects", nil)
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
	if payload.Code != apperrors.PublicCodePermissionProjectMismatch {
		t.Fatalf("expected code %d, got %d", apperrors.PublicCodePermissionProjectMismatch, payload.Code)
	}
}
