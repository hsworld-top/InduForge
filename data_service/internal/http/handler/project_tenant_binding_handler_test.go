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

type handlerProjectTenantBindingStore struct{ bindings map[string]string }

func (s *handlerProjectTenantBindingStore) BindIfUnbound(_ context.Context, projectID, tenantID string) (string, bool, error) {
	if existing, ok := s.bindings[projectID]; ok {
		return existing, false, nil
	}
	s.bindings[projectID] = tenantID
	return tenantID, true, nil
}

func TestProjectTenantBindingHandler_RejectsUnknownJSONField(t *testing.T) {
	handler := newProjectTenantBindingHandlerForTest()
	request := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(`{"tenantId":"`+uuid.NewString()+`","unexpected":true}`))
	request.SetPathValue("projectId", uuid.NewString())

	if err := handler.Put(httptest.NewRecorder(), request); err == nil {
		t.Fatal("expected unknown field to be rejected")
	}
}

func TestProjectTenantBindingHandler_WritesCreatedResult(t *testing.T) {
	handler := newProjectTenantBindingHandlerForTest()
	projectID, tenantID := uuid.NewString(), uuid.NewString()
	request := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(`{"tenantId":"`+tenantID+`"}`))
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
	return NewProjectTenantBindingHandler(service.NewProjectTenantBindingService(&handlerProjectTenantBindingStore{bindings: map[string]string{}}))
}
