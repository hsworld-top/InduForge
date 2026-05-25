package service

import (
	"context"
	"testing"
)

type fakeProtocolConnection struct {
	ID        string
	ProjectID string
	Type      string
	Name      string
	Config    map[string]any
}

type fakeProtocolDevConnectionRepository struct {
	connection fakeProtocolConnection
}

func (r *fakeProtocolDevConnectionRepository) GetProtocolDevConnection(_ context.Context, projectID, connectionID string) (*ProtocolDevConnection, error) {
	if r.connection.ProjectID != projectID || r.connection.ID != connectionID {
		return nil, nil
	}
	return &ProtocolDevConnection{
		ID:        r.connection.ID,
		ProjectID: r.connection.ProjectID,
		Type:      r.connection.Type,
		Name:      r.connection.Name,
		Config:    r.connection.Config,
	}, nil
}

func TestProtocolDevSessionService_CreateAndCloseSession(t *testing.T) {
	repo := &fakeProtocolDevConnectionRepository{
		connection: fakeProtocolConnection{
			ID:        "conn-1",
			ProjectID: "project-1",
			Type:      "opcua",
			Name:      "opcua-main",
			Config:    map[string]any{"endpoint": "opc.tcp://127.0.0.1:4840"},
		},
	}
	service := NewProtocolDevSessionService(repo, nil, nil)

	session, err := service.CreateSession(context.Background(), "project-1", "conn-1", "user-1", "opcua")
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	if session.SessionID == "" || session.Status != "connected" || session.Protocol != "opcua" {
		t.Fatalf("unexpected session: %#v", session)
	}

	closed, err := service.CloseSession(context.Background(), "project-1", "conn-1", session.SessionID, "user-1", "opcua")
	if err != nil {
		t.Fatalf("close session failed: %v", err)
	}
	if closed.Status != "closed" {
		t.Fatalf("expected closed session, got %#v", closed)
	}
}
