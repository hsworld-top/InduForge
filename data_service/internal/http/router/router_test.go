package router

import (
	"testing"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/http/handler"
)

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
