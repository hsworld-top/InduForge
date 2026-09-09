package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func centralTestToken(t *testing.T, claims Claims) string {
	t.Helper()
	payload, err := json.Marshal(struct {
		Claims
		Exp int64 `json:"exp"`
	}{claims, time.Now().Add(time.Hour).Unix()})
	if err != nil {
		t.Fatal(err)
	}
	signed := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`)) + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte("test-secret"))
	_, _ = mac.Write([]byte(signed))
	return signed + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestJWTValidatorRejectsRevokedCenterIdentity(t *testing.T) {
	center := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) }))
	defer center.Close()
	validator, err := NewJWTValidator("test-secret", center.URL)
	if err != nil {
		t.Fatal(err)
	}
	token := centralTestToken(t, Claims{UserID: "user-1", TenantID: "tenant-1", Role: "SYSTEM_ADMIN"})
	if _, err := validator.Validate(token); err == nil {
		t.Fatal("revoked center identity was accepted by data_service")
	}
}

func TestJWTValidatorChecksCurrentIdentityEveryTime(t *testing.T) {
	var revoked atomic.Bool
	token := centralTestToken(t, Claims{UserID: "user-1", TenantID: "tenant-1", Role: "SYSTEM_ADMIN"})
	center := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/v1/auth/me" || r.Header.Get("Authorization") != "Bearer "+token {
			t.Error("unexpected identity request")
		}
		if revoked.Load() {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"id":"user-1","tenantId":"tenant-1","role":"SYSTEM_ADMIN","mustChangePassword":false}}`))
	}))
	defer center.Close()
	validator, err := NewJWTValidator("test-secret", center.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = validator.Validate(token); err != nil {
		t.Fatal(err)
	}
	revoked.Store(true)
	if _, err = validator.Validate(token); err == nil {
		t.Fatal("cached identity survived revocation")
	}
	lease, cancel := validator.withIdentityLease(context.Background(), token, 5*time.Millisecond)
	defer cancel()
	select {
	case <-lease.Done():
	case <-time.After(time.Second):
		t.Fatal("revoked connection lease remained active")
	}
}

func TestJWTValidatorRejectsInvalidCentralIdentity(t *testing.T) {
	tests := map[string]map[string]any{
		"password change required": {"mustChangePassword": true},
		"missing password status":  {"mustChangePassword": nil},
		"platform":                 {"platform": true},
		"user mismatch":            {"id": "other"},
		"tenant mismatch":          {"tenantId": "other"},
		"role mismatch":            {"role": "USER"},
		"disabled":                 {"status": "disabled"},
		"expired":                  {"expiresAt": "2000-01-01T00:00:00Z"},
		"tenant disabled":          {"tenant": map[string]any{"id": "tenant-1", "status": "disabled"}},
		"tenant expired":           {"tenant": map[string]any{"id": "tenant-1", "expiresAt": "2000-01-01T00:00:00Z"}},
	}
	for name, fields := range tests {
		t.Run(name, func(t *testing.T) {
			user := map[string]any{"id": "user-1", "tenantId": "tenant-1", "role": "SYSTEM_ADMIN", "mustChangePassword": false}
			for key, value := range fields {
				user[key] = value
			}
			center := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": user})
			}))
			defer center.Close()
			validator, err := NewJWTValidator("test-secret", center.URL)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = validator.Validate(centralTestToken(t, Claims{UserID: "user-1", TenantID: "tenant-1", Role: "SYSTEM_ADMIN"})); err == nil {
				t.Fatal("invalid identity accepted")
			}
		})
	}
}

func TestJWTValidatorFailsClosedOnCenterFailure(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusInternalServerError, http.StatusFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			center := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "/api/v1/auth/me")
				w.WriteHeader(status)
			}))
			defer center.Close()
			validator, err := NewJWTValidator("test-secret", center.URL)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = validator.Validate(centralTestToken(t, Claims{UserID: "user-1", TenantID: "tenant-1", Role: "SYSTEM_ADMIN"})); err == nil {
				t.Fatal("unavailable identity accepted")
			}
		})
	}
	t.Run("timeout", func(t *testing.T) {
		center := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
		defer center.Close()
		validator, err := NewJWTValidator("test-secret", center.URL)
		if err != nil {
			t.Fatal(err)
		}
		validator.client.Timeout = 20 * time.Millisecond
		if _, err = validator.Validate(centralTestToken(t, Claims{UserID: "user-1", TenantID: "tenant-1", Role: "SYSTEM_ADMIN"})); err == nil {
			t.Fatal("timed out identity accepted")
		}
	})
	for _, body := range []string{`{`, `{"data":{}}`, `{"code":1,"data":{}}`} {
		t.Run(body, func(t *testing.T) {
			center := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer center.Close()
			validator, err := NewJWTValidator("test-secret", center.URL)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = validator.Validate(centralTestToken(t, Claims{UserID: "user-1", TenantID: "tenant-1", Role: "SYSTEM_ADMIN"})); err == nil {
				t.Fatal("malformed identity accepted")
			}
		})
	}
}

func TestValidateCenterURL(t *testing.T) {
	for _, value := range []string{"", "ftp://center", "http://user:pass@center", "http://center/api", "http://center?x=1", "http://center?", "http://center/#fragment"} {
		if err := ValidateCenterURL(value); err == nil {
			t.Errorf("accepted invalid URL %q", value)
		}
	}
	for _, value := range []string{"http://center:18600", "https://center/"} {
		if err := ValidateCenterURL(value); err != nil {
			t.Fatal(err)
		}
	}
}

func TestJWTValidatorRejectsPlatformAndMissingTenantBeforeCenter(t *testing.T) {
	center := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("non-tenant identity reached center") }))
	defer center.Close()
	validator, err := NewJWTValidator("test-secret", center.URL)
	if err != nil {
		t.Fatal(err)
	}
	for _, claims := range []Claims{{UserID: "platform", Role: "SUPER_ADMIN"}, {UserID: "platform", TenantID: "tenant-1", Role: "SUPER_ADMIN"}, {UserID: "user-1", Role: "USER"}} {
		if _, err := validator.Validate(centralTestToken(t, claims)); err == nil {
			t.Fatal("non-tenant identity accepted")
		}
	}
}
