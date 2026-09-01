package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
)

func TestRequireInternalToken_RejectsMissingAndWrongTokenWithoutLeakage(t *testing.T) {
	const secret = "test-internal-token"
	handler := middleware.RequireInternalToken(secret)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler must not be called")
	}))

	for name, provided := range map[string]string{"missing": "", "wrong": "not-the-token"} {
		t.Run(name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPut, "/api/v1/internal/data/project-bindings/project", nil)
			request = request.WithContext(middleware.WithRequestID(request.Context(), "request-123"))
			if provided != "" {
				request.Header.Set("X-InduForge-Internal-Token", provided)
			}

			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("expected status 401, got %d", recorder.Code)
			}
			var payload response.ApiResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Code != apperrors.PublicCodeAuthTokenInvalid || payload.ReqID != "request-123" || payload.Data != nil {
				t.Fatalf("unexpected auth response: %#v", payload)
			}
			if strings.Contains(recorder.Body.String(), secret) || (provided != "" && strings.Contains(recorder.Body.String(), provided)) {
				t.Fatalf("response leaked token: %s", recorder.Body.String())
			}
		})
	}
}

func TestRequireInternalToken_AllowsExpectedToken(t *testing.T) {
	handler := middleware.RequireInternalToken("test-internal-token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/", nil)
	request.Header.Set("X-InduForge-Internal-Token", "test-internal-token")

	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", recorder.Code)
	}
}
