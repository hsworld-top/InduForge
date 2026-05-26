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
