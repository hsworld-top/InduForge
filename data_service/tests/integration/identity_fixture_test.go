package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/indu-forge/data_service/internal/app"
	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/config"
)

var identityFixtures sync.Map

// 每个测试仅允许其签发函数登记的完整JWT；未知、篡改、跨测试凭据均不通过中心。
func integrationIdentities(t *testing.T) *sync.Map {
	t.Helper()
	identities := &sync.Map{}
	actual, loaded := identityFixtures.LoadOrStore(t, identities)
	if !loaded {
		t.Cleanup(func() { identityFixtures.Delete(t) })
	}
	return actual.(*sync.Map)
}

func newIntegrationIdentityCenter(t *testing.T) *httptest.Server {
	t.Helper()
	identities := integrationIdentities(t)
	center := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/auth/me" {
			http.NotFound(w, r)
			return
		}
		header := r.Header.Get("Authorization")
		token := strings.TrimPrefix(header, "Bearer ")
		value, found := identities.Load(token)
		if !strings.HasPrefix(header, "Bearer ") || !found {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		claims := value.(auth.Claims)
		if claims.UserID == "" || claims.TenantID == "" || claims.Role == "" || claims.Role == "SUPER_ADMIN" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "", "reqId": "test-center", "data": map[string]any{"id": claims.UserID, "tenantId": claims.TenantID, "role": claims.Role, "mustChangePassword": false}})
	}))
	t.Cleanup(center.Close)
	return center
}

func newIntegrationServer(t *testing.T, cfg config.Config) (*app.Server, error) {
	t.Helper()
	cfg.DevCoreURL = newIntegrationIdentityCenter(t).URL
	return app.NewServer(cfg)
}

func TestIntegrationIdentityFixtureRejectsUnknownAndAlteredTokens(t *testing.T) {
	center := newIntegrationIdentityCenter(t)
	token := mustSignIntegrationJWT(t, "fixture-secret", &auth.Claims{UserID: "fixture-user", TenantID: "fixture-tenant"})
	for _, test := range []struct {
		token  string
		status int
	}{{token, http.StatusOK}, {token + "altered", http.StatusUnauthorized}, {"unknown", http.StatusUnauthorized}} {
		req, err := http.NewRequest(http.MethodGet, center.URL+"/api/v1/auth/me", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+test.token)
		response, err := center.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != test.status {
			t.Fatalf("identity status=%d, want %d", response.StatusCode, test.status)
		}
	}
}
