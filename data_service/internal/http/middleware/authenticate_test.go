package middleware_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
)

func TestAuthenticate_AllowsValidBearerJWT(t *testing.T) {
	validator := auth.NewJWTValidator("secret-123")
	claims := &auth.Claims{
		UserID:       "user-1",
		TenantID:     "tenant-1",
		ProjectIDs:   []string{"project-a", "project-b"},
		Capabilities: []string{"project:read"},
	}
	token := mustSignJWT(t, "secret-123", claims)

	handler := middleware.Authenticate(validator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClaims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			t.Fatal("expected claims in context")
		}
		if gotClaims.UserID != claims.UserID {
			t.Fatalf("expected userId %q, got %q", claims.UserID, gotClaims.UserID)
		}
		if gotClaims.TenantID != claims.TenantID {
			t.Fatalf("expected tenantId %q, got %q", claims.TenantID, gotClaims.TenantID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestAuthenticate_RejectsInvalidBearerJWT(t *testing.T) {
	validator := auth.NewJWTValidator("secret-123")
	validToken := mustSignJWT(t, "secret-123", &auth.Claims{UserID: "user-1"})
	invalidToken := validToken + "tampered"

	handler := middleware.Authenticate(validator)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called for invalid token")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+invalidToken)

	handler.ServeHTTP(rr, req)

	assertAuthErrorResponse(t, rr, http.StatusUnauthorized, "AUTH_TOKEN_INVALID")
}

func TestAuthenticate_RejectsMissingBearerPrefix(t *testing.T) {
	validator := auth.NewJWTValidator("secret-123")
	handler := middleware.Authenticate(validator)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called for missing bearer prefix")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token abc")

	handler.ServeHTTP(rr, req)

	assertAuthErrorResponse(t, rr, http.StatusUnauthorized, "AUTH_TOKEN_REQUIRED")
}

func mustSignJWT(t *testing.T, secret string, claims *auth.Claims) string {
	t.Helper()

	headerJSON := []byte(`{"alg":"HS256","typ":"JWT"}`)
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims failed: %v", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := encodedHeader + "." + encodedPayload

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signingInput))
	signature := mac.Sum(nil)

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func assertAuthErrorResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedErrorCode string) {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, rr.Code)
	}

	var payload response.ApiResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if payload.Success {
		t.Fatal("expected success to be false")
	}
	if payload.ErrorCode != expectedErrorCode {
		t.Fatalf("expected errorCode %q, got %q", expectedErrorCode, payload.ErrorCode)
	}
	if payload.Message == "" {
		t.Fatal("expected error message to be set")
	}
}
