package middleware_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/indu-forge/data_service/internal/auth"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
)

func TestAuthenticate_AllowsValidBearerJWT(t *testing.T) {
	validator := mustNewJWTValidator(t, "secret-123")
	claims := &auth.Claims{
		UserID:       "user-1",
		TenantID:     "tenant-1",
		ProjectIDs:   []string{"project-a", "project-b"},
		Capabilities: []string{"project:read"},
	}
	now := time.Now().UTC()
	token := mustSignJWT(t, "secret-123", claims, now.Add(time.Minute), now.Add(-time.Minute), now.Add(-time.Minute))

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

func TestAuthenticate_RejectsExpiredToken(t *testing.T) {
	validator := mustNewJWTValidator(t, "secret-123")
	now := time.Now().UTC()
	token := mustSignJWT(t, "secret-123", &auth.Claims{UserID: "user-1"}, now.Add(-time.Minute), now.Add(-2*time.Minute), now.Add(-2*time.Minute))

	handler := middleware.Authenticate(validator)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called for expired token")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	handler.ServeHTTP(rr, req)

	assertAuthErrorResponse(t, rr, http.StatusUnauthorized, apperrors.PublicCodeAuthTokenInvalid)
}

func TestAuthenticate_RejectsNbfNotYetValid(t *testing.T) {
	validator := mustNewJWTValidator(t, "secret-123")
	now := time.Now().UTC()
	token := mustSignJWT(t, "secret-123", &auth.Claims{UserID: "user-1"}, now.Add(time.Minute), now.Add(time.Minute), now.Add(-time.Minute))

	handler := middleware.Authenticate(validator)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called for not-yet-valid token")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	handler.ServeHTTP(rr, req)

	assertAuthErrorResponse(t, rr, http.StatusUnauthorized, apperrors.PublicCodeAuthTokenInvalid)
}

func TestAuthenticate_RejectsIatFutureToken(t *testing.T) {
	validator := mustNewJWTValidator(t, "secret-123")
	now := time.Now().UTC()
	token := mustSignJWT(t, "secret-123", &auth.Claims{UserID: "user-1"}, now.Add(time.Hour), now.Add(-time.Minute), now.Add(time.Minute))

	handler := middleware.Authenticate(validator)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called for future iat token")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	handler.ServeHTTP(rr, req)

	assertAuthErrorResponse(t, rr, http.StatusUnauthorized, apperrors.PublicCodeAuthTokenInvalid)
}

func TestAuthenticate_RejectsInvalidBearerJWT(t *testing.T) {
	validator := mustNewJWTValidator(t, "secret-123")
	now := time.Now().UTC()
	validToken := mustSignJWT(t, "secret-123", &auth.Claims{UserID: "user-1"}, now.Add(time.Hour), now.Add(-time.Minute), now.Add(-time.Minute))
	invalidToken := validToken + "tampered"

	handler := middleware.Authenticate(validator)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called for invalid token")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+invalidToken)

	handler.ServeHTTP(rr, req)

	assertAuthErrorResponse(t, rr, http.StatusUnauthorized, apperrors.PublicCodeAuthTokenInvalid)
}

func TestAuthenticate_RejectsMissingBearerPrefix(t *testing.T) {
	validator := mustNewJWTValidator(t, "secret-123")
	handler := middleware.Authenticate(validator)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called for missing bearer prefix")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token abc")

	handler.ServeHTTP(rr, req)

	assertAuthErrorResponse(t, rr, http.StatusUnauthorized, apperrors.PublicCodeAuthTokenRequired)
}

func TestAuthenticate_RejectsEmptySecretAtConstruction(t *testing.T) {
	validator, err := auth.NewJWTValidator("   ")
	if err == nil {
		t.Fatal("expected empty secret to be rejected")
	}
	if validator != nil {
		t.Fatal("expected validator to be nil when secret is empty")
	}
}

func TestAuthenticate_RejectsNilValidatorWithInternalError(t *testing.T) {
	handler := middleware.Authenticate(nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called when validator is nil")
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")

	handler.ServeHTTP(rr, req)

	assertAuthErrorResponse(t, rr, http.StatusInternalServerError, apperrors.PublicCodeAuthSecretRequired)
}

func mustNewJWTValidator(t *testing.T, secret string) *auth.JWTValidator {
	t.Helper()

	validator, err := auth.NewJWTValidator(secret)
	if err != nil {
		t.Fatalf("new validator failed: %v", err)
	}
	return validator
}

func mustSignJWT(t *testing.T, secret string, claims *auth.Claims, exp, nbf, iat time.Time) string {
	t.Helper()

	headerJSON := []byte(`{"alg":"HS256","typ":"JWT"}`)
	payload := map[string]any{
		"userId":       claims.UserID,
		"tenantId":     claims.TenantID,
		"projectIds":   claims.ProjectIDs,
		"capabilities": claims.Capabilities,
		"exp":          exp.Unix(),
		"nbf":          nbf.Unix(),
		"iat":          iat.Unix(),
	}

	payloadJSON, err := json.Marshal(payload)
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

func assertAuthErrorResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedCode int) {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, rr.Code)
	}

	var payload response.ApiResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	if payload.Code != expectedCode {
		t.Fatalf("expected code %d, got %d", expectedCode, payload.Code)
	}
	if payload.Data != nil {
		t.Fatalf("expected error data to be nil, got %#v", payload.Data)
	}
	if payload.Msg == "" {
		t.Fatal("expected msg to be set")
	}
}
