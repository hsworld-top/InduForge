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

type routerProjectTenantBindingStore struct{ bindings map[string]string }

func (s *routerProjectTenantBindingStore) BindIfUnbound(_ context.Context, projectID, tenantID string) (string, bool, error) {
	if existing, ok := s.bindings[projectID]; ok {
		return existing, false, nil
	}
	s.bindings[projectID] = tenantID
	return tenantID, true, nil
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
	bindingHandler := handler.NewProjectTenantBindingHandler(service.NewProjectTenantBindingService(&routerProjectTenantBindingStore{bindings: map[string]string{}}))
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
