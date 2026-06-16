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
	groups  []ProtocolDevOpcuaGroup
	nodes   []ProtocolDevOpcuaNode
	updated map[string]any
}

type fakeProtocolDevOpcuaBrowser struct {
	result  *ProtocolDevOpcuaBrowseResult
	session ProtocolDevSession
	err     error
}

type fakeProtocolDevModbusReader struct {
	registers []ProtocolDevModbusRegister
}

type fakeProtocolDevS7Reader struct {
	variables []ProtocolDevS7Variable
	updated   map[string]any
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

func (r *fakeProtocolDevOpcuaReader) UpdateNodeLastValue(_ context.Context, _, _, nodeID, _ string, value any, _ string) error {
	if r.updated == nil {
		r.updated = map[string]any{}
	}
	r.updated[nodeID] = value
	return nil
}

func (b *fakeProtocolDevOpcuaBrowser) Browse(_ context.Context, session ProtocolDevSession) (*ProtocolDevOpcuaBrowseResult, error) {
	b.session = session
	if b.err != nil {
		return nil, b.err
	}
	if b.result == nil {
		return &ProtocolDevOpcuaBrowseResult{Nodes: []ProtocolDevBrowseNode{}, Diagnostics: []string{}}, nil
	}
	clone := &ProtocolDevOpcuaBrowseResult{
		Nodes:       append([]ProtocolDevBrowseNode{}, b.result.Nodes...),
		Diagnostics: append([]string{}, b.result.Diagnostics...),
	}
	return clone, nil
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

func (r *fakeProtocolDevS7Reader) ListDevSessionS7Variables(_ context.Context, _, _ string, groupID *string) ([]ProtocolDevS7Variable, error) {
	result := make([]ProtocolDevS7Variable, 0, len(r.variables))
	for _, variable := range r.variables {
		if groupID != nil && (variable.GroupID == nil || *variable.GroupID != *groupID) {
			continue
		}
		result = append(result, variable)
	}
	return result, nil
}

func (r *fakeProtocolDevS7Reader) EstimateDevSessionS7ReadPlan(_ context.Context, _, _ string, _ *string) (*S7ReadPlanEstimate, error) {
	return &S7ReadPlanEstimate{VariableCount: len(r.variables), BlockCount: 1, Diagnostics: []string{"ok"}}, nil
}

func (r *fakeProtocolDevS7Reader) UpdateVariableLastValue(_ context.Context, _, _, variableID, _ string, value any, _ string) error {
	if r.updated == nil {
		r.updated = map[string]any{}
	}
	r.updated[variableID] = value
	return nil
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
	service := NewProtocolDevSessionService(repo, nil, nil, nil, nil)

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
	browser := &fakeProtocolDevOpcuaBrowser{
		result: &ProtocolDevOpcuaBrowseResult{
			Nodes: []ProtocolDevBrowseNode{
				{ID: "opcua-i=85", Name: "Objects", NodeID: "i=85", NodeType: "folder"},
				{ID: "opcua-ns=2;s=Furnace01.Temp", ParentID: protocolDevStringPtr("opcua-i=85"), Name: "Temp", NodeID: "ns=2;s=Furnace01.Temp", NodeType: "variable", DataType: "Double"},
			},
			Diagnostics: []string{"真实浏览"},
		},
	}
	service := NewProtocolDevSessionService(repo, opcua, browser, nil, nil)
	session, err := service.CreateSession(context.Background(), "project-1", "conn-1", "user-1", "opcua")
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	browse, err := service.BrowseOpcua(context.Background(), "project-1", "conn-1", session.SessionID, "user-1")
	if err != nil {
		t.Fatalf("browse failed: %v", err)
	}
	if browser.session.SessionID != session.SessionID {
		t.Fatalf("browser did not receive session: %#v", browser.session)
	}
	if len(browse.Nodes) != 2 || browse.Nodes[1].NodeID != "ns=2;s=Furnace01.Temp" || !browse.Nodes[1].Modeled || browse.Nodes[0].Modeled {
		t.Fatalf("unexpected browse result: %#v", browse)
	}

	read, err := service.ReadOpcua(context.Background(), "project-1", "conn-1", session.SessionID, "user-1", []string{"n-1"}, nil)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if len(read.Values) != 1 || read.Values[0].Quality != "Good" || read.Values[0].Value == nil || opcua.updated["n-1"] == nil {
		t.Fatalf("unexpected read result: %#v updated=%#v", read, opcua.updated)
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
	service := NewProtocolDevSessionService(repo, nil, nil, modbus, nil)
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

func TestProtocolDevSessionService_ReadAndPollS7Variables(t *testing.T) {
	repo := &fakeProtocolDevConnectionRepository{
		connection: fakeProtocolConnection{
			ID:        "conn-1",
			ProjectID: "project-1",
			Type:      "s7",
			Name:      "s7-main",
			Config:    map[string]any{"host": "127.0.0.1", "port": 102},
		},
	}
	s7 := &fakeProtocolDevS7Reader{
		variables: []ProtocolDevS7Variable{{
			ID:                "v-1",
			Name:              "Speed",
			Code:              "speed",
			Area:              "DB",
			DBNumber:          protocolDevIntPtr(1),
			ByteOffset:        4,
			AddressText:       "DB1.DBD4",
			NormalizedAddress: "DB1.DBD4",
			ReadLength:        4,
			DataType:          "Real",
			Scale:             1,
		}},
	}
	service := NewProtocolDevSessionService(repo, nil, nil, nil, s7)
	session, err := service.CreateSession(context.Background(), "project-1", "conn-1", "user-1", "s7")
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	read, err := service.ReadS7(context.Background(), "project-1", "conn-1", session.SessionID, "user-1", []string{"v-1"}, nil)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if len(read.Values) != 1 || read.Values[0].Value == nil || s7.updated["v-1"] == nil {
		t.Fatalf("unexpected read result: %#v updated=%#v", read, s7.updated)
	}

	poll, err := service.PollS7(context.Background(), "project-1", "conn-1", session.SessionID, "user-1", nil)
	if err != nil {
		t.Fatalf("poll failed: %v", err)
	}
	if len(poll.Values) != 1 || poll.ReadPlan.BlockCount != 1 || len(poll.Diagnostics) == 0 {
		t.Fatalf("unexpected poll result: %#v", poll)
	}
}

func protocolDevStringPtr(value string) *string {
	return &value
}

func protocolDevIntPtr(value int) *int {
	return &value
}
