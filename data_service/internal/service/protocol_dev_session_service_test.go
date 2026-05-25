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

type fakeProtocolDevOpcuaReader struct {
	groups []ProtocolDevOpcuaGroup
	nodes  []ProtocolDevOpcuaNode
}

type fakeProtocolDevModbusReader struct {
	registers []ProtocolDevModbusRegister
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

func (r *fakeProtocolDevOpcuaReader) ListDevSessionOpcuaGroups(_ context.Context, _, _ string) ([]ProtocolDevOpcuaGroup, error) {
	return append([]ProtocolDevOpcuaGroup{}, r.groups...), nil
}

func (r *fakeProtocolDevOpcuaReader) ListDevSessionOpcuaNodes(_ context.Context, _, _ string, groupID *string) ([]ProtocolDevOpcuaNode, error) {
	result := make([]ProtocolDevOpcuaNode, 0, len(r.nodes))
	for _, node := range r.nodes {
		if groupID != nil && (node.GroupID == nil || *node.GroupID != *groupID) {
			continue
		}
		result = append(result, node)
	}
	return result, nil
}

func (r *fakeProtocolDevModbusReader) ListDevSessionModbusRegisters(_ context.Context, _, _ string, groupID *string) ([]ProtocolDevModbusRegister, error) {
	result := make([]ProtocolDevModbusRegister, 0, len(r.registers))
	for _, register := range r.registers {
		if groupID != nil && (register.GroupID == nil || *register.GroupID != *groupID) {
			continue
		}
		result = append(result, register)
	}
	return result, nil
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

func TestProtocolDevSessionService_BrowseAndReadOpcuaNodes(t *testing.T) {
	repo := &fakeProtocolDevConnectionRepository{
		connection: fakeProtocolConnection{
			ID:        "conn-1",
			ProjectID: "project-1",
			Type:      "opcua",
			Name:      "opcua-main",
			Config:    map[string]any{"endpoint": "opc.tcp://127.0.0.1:4840"},
		},
	}
	opcua := &fakeProtocolDevOpcuaReader{
		groups: []ProtocolDevOpcuaGroup{{ID: "g-1", Name: "Furnace01"}},
		nodes: []ProtocolDevOpcuaNode{{
			ID:       "n-1",
			GroupID:  protocolDevStringPtr("g-1"),
			Name:     "Temp",
			Code:     "temp",
			NodeID:   "ns=2;s=Furnace01.Temp",
			DataType: "Double",
		}},
	}
	service := NewProtocolDevSessionService(repo, opcua, nil)
	session, err := service.CreateSession(context.Background(), "project-1", "conn-1", "user-1", "opcua")
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	browse, err := service.BrowseOpcua(context.Background(), "project-1", "conn-1", session.SessionID, "user-1")
	if err != nil {
		t.Fatalf("browse failed: %v", err)
	}
	if len(browse.Nodes) != 2 || browse.Nodes[1].NodeID != "ns=2;s=Furnace01.Temp" {
		t.Fatalf("unexpected browse result: %#v", browse)
	}

	read, err := service.ReadOpcua(context.Background(), "project-1", "conn-1", session.SessionID, "user-1", []string{"n-1"}, nil)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if len(read.Values) != 1 || read.Values[0].Quality != "Good" || read.Values[0].Value == nil {
		t.Fatalf("unexpected read result: %#v", read)
	}
}

func TestProtocolDevSessionService_ReadAndPollModbusRegisters(t *testing.T) {
	repo := &fakeProtocolDevConnectionRepository{
		connection: fakeProtocolConnection{
			ID:        "conn-1",
			ProjectID: "project-1",
			Type:      "modbus",
			Name:      "modbus-main",
			Config:    map[string]any{"host": "127.0.0.1", "port": 502},
		},
	}
	modbus := &fakeProtocolDevModbusReader{
		registers: []ProtocolDevModbusRegister{{
			ID:              "r-1",
			ProjectID:       "project-1",
			ConnectionID:    "conn-1",
			Name:            "Speed",
			Code:            "speed",
			UnitID:          1,
			Area:            "holding_register",
			Address:         40001,
			ProtocolAddress: 0,
			Quantity:        1,
			DataType:        "int16",
			PollIntervalMS:  1000,
			Status:          "active",
			Scale:           1,
		}},
	}
	service := NewProtocolDevSessionService(repo, nil, modbus)
	session, err := service.CreateSession(context.Background(), "project-1", "conn-1", "user-1", "modbus")
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	read, err := service.ReadModbus(context.Background(), "project-1", "conn-1", session.SessionID, "user-1", []string{"r-1"}, nil)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if len(read.Values) != 1 || read.Values[0].RawValue == nil || read.Values[0].Value == nil {
		t.Fatalf("unexpected read result: %#v", read)
	}

	poll, err := service.PollModbus(context.Background(), "project-1", "conn-1", session.SessionID, "user-1", nil)
	if err != nil {
		t.Fatalf("poll failed: %v", err)
	}
	if len(poll.Values) != 1 || poll.ReadPlan.ReadCount != 1 || len(poll.Diagnostics) == 0 {
		t.Fatalf("unexpected poll result: %#v", poll)
	}
}

func protocolDevStringPtr(value string) *string {
	return &value
}
