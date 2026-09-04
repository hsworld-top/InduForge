package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/indu-forge/data_service/internal/http/middleware"
	"github.com/indu-forge/data_service/internal/http/response"
	"github.com/indu-forge/data_service/internal/service"
)

type handlerProjectTenantBindingStore struct {
	bindings map[string]struct {
		tenant string
		epoch  int64
	}
}

func (s *handlerProjectTenantBindingStore) BindIfUnbound(_ context.Context, projectID, tenantID string, epoch int64) (string, int64, bool, error) {
	if existing, ok := s.bindings[projectID]; ok {
		return existing.tenant, existing.epoch, false, nil
	}
	s.bindings[projectID] = struct {
		tenant string
		epoch  int64
	}{tenantID, epoch}
	return tenantID, epoch, true, nil
}
func (s *handlerProjectTenantBindingStore) Get(_ context.Context, projectID, tenantID string) (int64, error) {
	item, ok := s.bindings[projectID]
	if !ok || item.tenant != tenantID {
		return 0, context.Canceled
	}
	return item.epoch, nil
}

func TestProjectTenantBindingHandler_RejectsUnknownJSONField(t *testing.T) {
	handler := newProjectTenantBindingHandlerForTest()
	request := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(`{"tenantId":"`+uuid.NewString()+`","authoringEpoch":"epoch-1","unexpected":true}`))
	request.SetPathValue("projectId", uuid.NewString())

	if err := handler.Put(httptest.NewRecorder(), request); err == nil {
		t.Fatal("expected unknown field to be rejected")
	}
}

func TestProjectTenantBindingHandler_WritesCreatedResult(t *testing.T) {
	handler := newProjectTenantBindingHandlerForTest()
	projectID, tenantID := uuid.NewString(), uuid.NewString()
	request := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(`{"tenantId":"`+tenantID+`","authoringEpoch":"epoch-1"}`))
	request.SetPathValue("projectId", projectID)
	request = request.WithContext(middleware.WithRequestID(request.Context(), "request-123"))
	recorder := httptest.NewRecorder()

	if err := handler.Put(recorder, request); err != nil {
		t.Fatal(err)
	}
	var payload response.ApiResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != 0 || payload.ReqID != "request-123" {
		t.Fatalf("unexpected response: %#v", payload)
	}
}

func newProjectTenantBindingHandlerForTest() *ProjectTenantBindingHandler {
	return NewProjectTenantBindingHandler(service.NewProjectTenantBindingService(&handlerProjectTenantBindingStore{bindings: map[string]struct {
		tenant string
		epoch  int64
	}{}}))
}
