package router

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/http/handler"
	"github.com/indu-forge/data_service/internal/service"
)

type routerProjectTenantBindingStore struct {
	bindings map[string]struct {
		tenant string
		epoch  int64
	}
}

func (s *routerProjectTenantBindingStore) BindIfUnbound(_ context.Context, projectID, tenantID string, epoch int64) (string, int64, bool, error) {
	if existing, ok := s.bindings[projectID]; ok {
		return existing.tenant, existing.epoch, false, nil
	}
	s.bindings[projectID] = struct {
		tenant string
		epoch  int64
	}{tenantID, epoch}
	return tenantID, epoch, true, nil
}
func (s *routerProjectTenantBindingStore) Get(_ context.Context, projectID, tenantID string) (int64, error) {
	item, ok := s.bindings[projectID]
	if !ok || item.tenant != tenantID {
		return 0, context.Canceled
	}
	return item.epoch, nil
}

func TestNewRouterKafkaWorkbenchRoutesDoNotConflict(t *testing.T) {
	validator, err := auth.NewJWTValidator("router-test-secret")
	if err != nil {
		t.Fatalf("create jwt validator failed: %v", err)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("kafka workbench routes should not conflict: %v", recovered)
		}
	}()

	_ = NewRouter(WithKafkaWorkbenchRoutes(handler.NewKafkaWorkbenchHandler(nil), validator))
}

func TestNewRouterCollectorCatalogRoutesDoNotConflict(t *testing.T) {
	validator, err := auth.NewJWTValidator("router-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("collector catalog routes should not conflict: %v", recovered)
		}
	}()
	_ = NewRouter(WithCollectorCatalogRoutes(handler.NewCollectorCatalogHandler(nil), validator))
}

func TestNewRouterCollectorRoutesDoNotConflict(t *testing.T) {
	validator, err := auth.NewJWTValidator("router-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("collector routes should not conflict: %v", recovered)
		}
	}()
	_ = NewRouter(WithCollectorRoutes(handler.NewCollectorHandler(nil), validator))
}

func TestNewRouterCollectorPointRoutesDoNotConflict(t *testing.T) {
	validator, err := auth.NewJWTValidator("router-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("collector point routes should not conflict: %v", recovered)
		}
	}()
	_ = NewRouter(WithCollectorPointRoutes(handler.NewCollectorPointHandler(nil), validator))
}

func TestNewRouterCollectorImportRoutesDoNotConflict(t *testing.T) {
	validator, err := auth.NewJWTValidator("router-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("collector import routes should not conflict: %v", recovered)
		}
	}()
	_ = NewRouter(WithCollectorImportRoutes(handler.NewCollectorImportHandler(nil), validator))
}

func TestNewRouterAlarmRoutesDoNotConflict(t *testing.T) {
	validator, err := auth.NewJWTValidator("router-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("alarm policy routes should not conflict: %v", recovered)
		}
	}()
	_ = NewRouter(WithAlarmRoutes(handler.NewAlarmHandler(nil), validator))
}

func TestInternalProjectTenantBindingRouteRejectsBearerToken(t *testing.T) {
	bindingHandler := handler.NewProjectTenantBindingHandler(service.NewProjectTenantBindingService(&routerProjectTenantBindingStore{bindings: map[string]struct {
		tenant string
		epoch  int64
	}{}}))
	router := NewRouter(WithProjectTenantBindingInternalRoutes(bindingHandler, "internal-token"))
	projectID, tenantID := uuid.NewString(), uuid.NewString()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/internal/data/project-bindings/"+projectID, bytes.NewBufferString(`{"tenantId":"`+tenantID+`"}`))
	request.Header.Set("Authorization", "Bearer not-an-internal-token")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected ordinary bearer token to be rejected with 401, got %d", recorder.Code)
	}
}

func TestInternalCollectorBindingRouteRejectsBearerToken(t *testing.T) {
	bundleHandler := handler.NewCollectorBindingBundleHandler(nil, nil)
	router := NewRouter(WithCollectorBindingBundleInternalRoutes(bundleHandler, "internal-token"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/internal/data/projects/11111111-1111-4111-8111-111111111111/collector-binding", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer not-an-internal-token")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected ordinary bearer token to be rejected with 401, got %d", recorder.Code)
	}
}

func TestProjectSnapshotInternalRoutesRequireInternalToken(t *testing.T) {
	snapshotHandler := handler.NewProjectSnapshotHandler(nil)
	router := NewRouter(WithProjectSnapshotInternalRoutes(snapshotHandler, "internal-token"))
	cases := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/internal/data/projects/11111111-1111-4111-8111-111111111111/snapshot?tenantId=22222222-2222-4222-8222-222222222222"},
		{http.MethodPut, "/api/v1/internal/data/projects/11111111-1111-4111-8111-111111111111/snapshot"},
		{http.MethodPost, "/api/v1/internal/data/projects/11111111-1111-4111-8111-111111111111/artifact"},
		{http.MethodPost, "/api/v1/internal/data/projects/11111111-1111-4111-8111-111111111111/snapshot/artifacts"},
	}
	for _, item := range cases {
		request := httptest.NewRequest(item.method, item.path, bytes.NewBufferString(`{}`))
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status=%d", item.method, item.path, recorder.Code)
		}
	}
}
