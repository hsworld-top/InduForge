package handler

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/collectorprotocol"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

func TestCollectorCatalogHandlerListWritesPaginatedResponse(t *testing.T) {
	handler := newCollectorCatalogHandlerForTest(t)
	request := httptest.NewRequest("GET", "/api/v1/data/collector/drivers?page=1&pageSize=2", nil)
	request = request.WithContext(auth.WithClaims(request.Context(), &auth.Claims{UserID: "user-1"}))
	recorder := httptest.NewRecorder()

	if err := handler.List(recorder, request); err != nil {
		t.Fatal(err)
	}
	var payload response.ApiResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	data := payload.Data.(map[string]any)
	if len(data["list"].([]any)) != 2 {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestCollectorCatalogHandlerGetReturnsNotFound(t *testing.T) {
	handler := newCollectorCatalogHandlerForTest(t)
	request := httptest.NewRequest("GET", "/api/v1/data/collector/drivers/missing", nil)
	request.SetPathValue("driverId", "missing")
	request = request.WithContext(auth.WithClaims(request.Context(), &auth.Claims{UserID: "user-1"}))
	if err := handler.Get(httptest.NewRecorder(), request); err == nil {
		t.Fatal("expected missing driver error")
	}
}

func newCollectorCatalogHandlerForTest(t *testing.T) *CollectorCatalogHandler {
	t.Helper()
	root := filepath.Clean(filepath.Join("..", "..", "..", "..", "runtime", "collector_protocols"))
	catalog, err := collectorprotocol.LoadCatalog(os.DirFS(root))
	if err != nil {
		t.Fatal(err)
	}
	return NewCollectorCatalogHandler(service.NewCollectorCatalogService(catalog))
}
