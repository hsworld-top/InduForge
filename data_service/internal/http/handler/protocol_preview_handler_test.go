package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/indu-forge/data_service/internal/auth"
	"github.com/indu-forge/data_service/internal/service"
)

type fakeProtocolPreviewHandlerService struct {
	projectID    string
	connectionID string
	input        service.ProtocolPreviewInput
}

func (s *fakeProtocolPreviewHandlerService) Preview(ctx context.Context, projectID, connectionID string, input service.ProtocolPreviewInput) (*service.ProtocolPreviewResult, error) {
	s.projectID = projectID
	s.connectionID = connectionID
	s.input = input
	return &service.ProtocolPreviewResult{
		Protocol:     "http",
		ConnectionID: connectionID,
		Status:       "ok",
		Samples:      []any{map[string]any{"ok": true}},
	}, nil
}

func TestProtocolConnectionHandler_PreviewProtocolParsesBody(t *testing.T) {
	fakeService := &fakeProtocolPreviewHandlerService{}
	handler := NewProtocolConnectionHandler(nil, fakeService)
	body, err := json.Marshal(map[string]any{
		"limit":     25,
		"timeoutMs": 12000,
		"options": map[string]any{
			"subscribeMessage": "hello",
		},
	})
	if err != nil {
		t.Fatalf("marshal body failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/data/projects/project-a/protocols/conn-a/preview", bytes.NewReader(body))
	req.SetPathValue("projectId", "project-a")
	req.SetPathValue("connectionId", "conn-a")
	req = req.WithContext(auth.WithClaims(req.Context(), &auth.Claims{UserID: "user-a"}))
	recorder := httptest.NewRecorder()

	if err := handler.PreviewProtocol(recorder, req); err != nil {
		t.Fatalf("preview handler failed: %v", err)
	}

	if fakeService.projectID != "project-a" || fakeService.connectionID != "conn-a" {
		t.Fatalf("unexpected route args project=%q connection=%q", fakeService.projectID, fakeService.connectionID)
	}
	if fakeService.input.Limit != 25 || fakeService.input.TimeoutMS != 12000 {
		t.Fatalf("unexpected preview input: %#v", fakeService.input)
	}
	if fakeService.input.Options["subscribeMessage"] != "hello" {
		t.Fatalf("expected subscribeMessage option, got %#v", fakeService.input.Options)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}
