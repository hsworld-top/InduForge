package service

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/google/uuid"
)

// ProtocolDevConnection 是开发态会话需要的最小接入源投影。
type ProtocolDevConnection struct {
	ID        string
	ProjectID string
	Type      string
	Name      string
	Config    map[string]any
}

// ProtocolDevConnectionReader 隔离会话服务和具体仓储实现，后续可由 connection repository 适配。
type ProtocolDevConnectionReader interface {
	GetProtocolDevConnection(ctx context.Context, projectID, connectionID string) (*ProtocolDevConnection, error)
}

// ProtocolDevOpcuaModelReader 表示 OPC UA 开发态会话需要读取的建模数据接口。
type ProtocolDevOpcuaModelReader interface {
	ListDevSessionOpcuaGroups(ctx context.Context, projectID, connectionID string) ([]ProtocolDevOpcuaGroup, error)
	ListDevSessionOpcuaNodes(ctx context.Context, projectID, connectionID string, groupID *string) ([]ProtocolDevOpcuaNode, error)
}

// ProtocolDevModbusModelReader 表示 Modbus 开发态会话需要读取的建模数据接口。
type ProtocolDevModbusModelReader interface {
	ListDevSessionModbusRegisters(ctx context.Context, projectID, connectionID string, groupID *string) ([]ProtocolDevModbusRegister, error)
}

// ProtocolDevOpcuaGroup 是 OPC UA 浏览树需要的变量组投影。
type ProtocolDevOpcuaGroup struct {
	ID       string
	ParentID *string
	Name     string
}

// ProtocolDevOpcuaNode 是 OPC UA 开发态读取需要的变量投影。
type ProtocolDevOpcuaNode struct {
	ID       string
	GroupID  *string
	Name     string
	Code     string
	NodeID   string
	DataType string
}

// ProtocolDevBrowseNode 表示前端浏览树中的节点。
type ProtocolDevBrowseNode struct {
	ID       string  `json:"id"`
	ParentID *string `json:"parentId,omitempty"`
	Name     string  `json:"name"`
	NodeID   string  `json:"nodeId"`
	NodeType string  `json:"nodeType"`
	DataType string  `json:"dataType,omitempty"`
}

// ProtocolDevOpcuaBrowseResult 表示 OPC UA 开发态浏览结果。
type ProtocolDevOpcuaBrowseResult struct {
	Nodes       []ProtocolDevBrowseNode `json:"nodes"`
	Diagnostics []string                `json:"diagnostics"`
}

// ProtocolDevOpcuaReadValue 表示 OPC UA 开发态读取返回的一条值。
type ProtocolDevOpcuaReadValue struct {
	NodeID          string  `json:"nodeId"`
	Value           any     `json:"value"`
	DataType        string  `json:"dataType"`
	Quality         string  `json:"quality"`
	SourceTimestamp string  `json:"sourceTimestamp"`
	ServerTimestamp string  `json:"serverTimestamp"`
	Error           *string `json:"error"`
}

// ProtocolDevOpcuaReadResult 表示 OPC UA 开发态读取结果。
type ProtocolDevOpcuaReadResult struct {
	Values      []ProtocolDevOpcuaReadValue `json:"values"`
	Diagnostics []string                    `json:"diagnostics"`
}

// ProtocolDevModbusRegister 是 Modbus 开发态读取和轮询需要的寄存器投影。
type ProtocolDevModbusRegister struct {
	ID              string
	ProjectID       string
	ConnectionID    string
	GroupID         *string
	Name            string
	Code            string
	UnitID          int
	Area            string
	Address         int
	ProtocolAddress int
	Quantity        int
	DataType        string
	PollIntervalMS  int
	Status          string
	Scale           float64
	Offset          float64
}

// ProtocolDevModbusReadValue 表示 Modbus 开发态读取返回的一条值。
type ProtocolDevModbusReadValue struct {
	RegisterID string  `json:"registerId"`
	SlaveID    int     `json:"slaveId"`
	Area       string  `json:"area"`
	Address    int     `json:"address"`
	RawValue   []int   `json:"rawValue"`
	Value      any     `json:"value"`
	DataType   string  `json:"dataType"`
	Timestamp  string  `json:"timestamp"`
	Error      *string `json:"error"`
}

// ProtocolDevModbusReadResult 表示 Modbus 开发态读取结果。
type ProtocolDevModbusReadResult struct {
	Values      []ProtocolDevModbusReadValue `json:"values"`
	Diagnostics []string                     `json:"diagnostics"`
}

// ProtocolDevModbusPollResult 表示 Modbus 开发态短时轮询结果。
type ProtocolDevModbusPollResult struct {
	Values      []ProtocolDevModbusReadValue `json:"values"`
	ReadPlan    ModbusReadPlanEstimate       `json:"readPlan"`
	Diagnostics []string                     `json:"diagnostics"`
}

// ProtocolDevSession 表示 OPC UA / Modbus 工作台的一次开发态短时会话。
type ProtocolDevSession struct {
	SessionID   string         `json:"sessionId"`
	ProjectID    string         `json:"projectId"`
	ConnectionID string         `json:"connectionId"`
	Protocol     string         `json:"protocol"`
	Status       string         `json:"status"`
	ConnectedAt  string         `json:"connectedAt"`
	Endpoint     string         `json:"endpoint"`
	Diagnostics  []string       `json:"diagnostics"`
	Config       map[string]any `json:"-"`
	UserID       string         `json:"-"`
	lastUsedAt   time.Time
}

// ProtocolDevSessionService 管理协议工作台的短时开发态会话。
type ProtocolDevSessionService struct {
	connections ProtocolDevConnectionReader
	opcua       ProtocolDevOpcuaModelReader
	modbus      ProtocolDevModbusModelReader
	mu          sync.Mutex
	sessions    map[string]ProtocolDevSession
	now         func() time.Time
}

// NewProtocolDevSessionService 创建协议开发态会话服务。
func NewProtocolDevSessionService(connections ProtocolDevConnectionReader, opcua ProtocolDevOpcuaModelReader, modbus ProtocolDevModbusModelReader) *ProtocolDevSessionService {
	return &ProtocolDevSessionService{
		connections: connections,
		opcua:       opcua,
		modbus:      modbus,
		sessions:    map[string]ProtocolDevSession{},
		now:         time.Now,
	}
}

// CreateSession 创建一个绑定用户、项目和接入源的开发态会话。
func (s *ProtocolDevSessionService) CreateSession(ctx context.Context, projectID, connectionID, userID, protocol string) (*ProtocolDevSession, error) {
	connection, err := s.loadProtocolConnection(ctx, projectID, connectionID, protocol)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	session := ProtocolDevSession{
		SessionID:   uuid.NewString(),
		ProjectID:    projectID,
		ConnectionID: connectionID,
		Protocol:     protocol,
		Status:       "connected",
		ConnectedAt:  now.Format("2006-01-02 15:04:05"),
		Endpoint:     protocolDevEndpoint(protocol, connection.Config),
		Diagnostics:  []string{},
		Config:       connection.Config,
		UserID:       userID,
		lastUsedAt:   now,
	}
	s.mu.Lock()
	s.sessions[session.SessionID] = session
	s.mu.Unlock()
	return &session, nil
}

// CloseSession 释放开发态会话。
func (s *ProtocolDevSessionService) CloseSession(ctx context.Context, projectID, connectionID, sessionID, userID, protocol string) (*ProtocolDevSession, error) {
	session, err := s.requireSession(projectID, connectionID, sessionID, userID, protocol)
	if err != nil {
		return nil, err
	}
	session.Status = "closed"
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
	return &session, nil
}

// BrowseOpcua 基于当前会话返回 OPC UA 浏览树，第一版使用已建模分组和变量投影生成结果。
func (s *ProtocolDevSessionService) BrowseOpcua(ctx context.Context, projectID, connectionID, sessionID, userID string) (*ProtocolDevOpcuaBrowseResult, error) {
	if _, err := s.requireSession(projectID, connectionID, sessionID, userID, "opcua"); err != nil {
		return nil, err
	}
	if s.opcua == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "OPC UA 开发态模型读取器未初始化")
	}
	groups, err := s.opcua.ListDevSessionOpcuaGroups(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	nodes, err := s.opcua.ListDevSessionOpcuaNodes(ctx, projectID, connectionID, nil)
	if err != nil {
		return nil, err
	}

	result := &ProtocolDevOpcuaBrowseResult{
		Nodes: make([]ProtocolDevBrowseNode, 0, len(groups)+len(nodes)),
		Diagnostics: []string{
			"当前浏览结果来自已建模变量，真实 OPC UA 地址空间适配器接入后保持同一响应结构。",
		},
	}
	for _, group := range groups {
		result.Nodes = append(result.Nodes, ProtocolDevBrowseNode{
			ID:       "group-" + group.ID,
			ParentID: optionalPrefixedID("group-", group.ParentID),
			Name:     group.Name,
			NodeID:   group.ID,
			NodeType: "folder",
		})
	}
	for _, node := range nodes {
		result.Nodes = append(result.Nodes, ProtocolDevBrowseNode{
			ID:       node.ID,
			ParentID: optionalPrefixedID("group-", node.GroupID),
			Name:     node.Name,
			NodeID:   node.NodeID,
			NodeType: "variable",
			DataType: node.DataType,
		})
	}
	return result, nil
}

// ReadOpcua 读取会话内 OPC UA 变量当前值，第一版生成可预测的开发态样例值。
func (s *ProtocolDevSessionService) ReadOpcua(ctx context.Context, projectID, connectionID, sessionID, userID string, nodeIDs []string, groupID *string) (*ProtocolDevOpcuaReadResult, error) {
	if _, err := s.requireSession(projectID, connectionID, sessionID, userID, "opcua"); err != nil {
		return nil, err
	}
	if s.opcua == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "OPC UA 开发态模型读取器未初始化")
	}
	nodes, err := s.opcua.ListDevSessionOpcuaNodes(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}

	wanted := stringSet(nodeIDs)
	now := s.now().UTC().Format("2006-01-02 15:04:05")
	result := &ProtocolDevOpcuaReadResult{Values: []ProtocolDevOpcuaReadValue{}, Diagnostics: []string{}}
	for _, node := range nodes {
		if len(wanted) > 0 && !wanted[node.ID] && !wanted[node.NodeID] {
			continue
		}
		result.Values = append(result.Values, ProtocolDevOpcuaReadValue{
			NodeID:          node.NodeID,
			Value:           sampleProtocolValue(node.DataType, node.Code),
			DataType:        node.DataType,
			Quality:         "Good",
			SourceTimestamp: now,
			ServerTimestamp: now,
			Error:           nil,
		})
	}
	return result, nil
}

// SubscribeOpcua 复用读取结构返回当前分组的短时订阅快照，后续可替换为真实订阅缓冲区。
func (s *ProtocolDevSessionService) SubscribeOpcua(ctx context.Context, projectID, connectionID, sessionID, userID string, nodeIDs []string, groupID *string) (*ProtocolDevOpcuaReadResult, error) {
	result, err := s.ReadOpcua(ctx, projectID, connectionID, sessionID, userID, nodeIDs, groupID)
	if err != nil {
		return nil, err
	}
	result.Diagnostics = append(result.Diagnostics, "当前订阅结果为开发态短时读取快照，后续接入真实订阅后保持响应结构不变。")
	return result, nil
}

// StopSubscribeOpcua 结束 OPC UA 短时订阅占位，会话仍保持连接。
func (s *ProtocolDevSessionService) StopSubscribeOpcua(ctx context.Context, projectID, connectionID, sessionID, userID string) (*ProtocolDevSession, error) {
	session, err := s.requireSession(projectID, connectionID, sessionID, userID, "opcua")
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// ReadModbus 读取会话内 Modbus 寄存器当前值，第一版基于建模数据生成开发态样例值。
func (s *ProtocolDevSessionService) ReadModbus(ctx context.Context, projectID, connectionID, sessionID, userID string, registerIDs []string, groupID *string) (*ProtocolDevModbusReadResult, error) {
	if _, err := s.requireSession(projectID, connectionID, sessionID, userID, "modbus"); err != nil {
		return nil, err
	}
	registers, err := s.listModbusRegisters(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}
	values := s.buildModbusReadValues(registers, registerIDs)
	return &ProtocolDevModbusReadResult{Values: values, Diagnostics: []string{}}, nil
}

// PollModbus 返回当前会话的短时轮询快照，并附带与运行态一致的读取计划估算。
func (s *ProtocolDevSessionService) PollModbus(ctx context.Context, projectID, connectionID, sessionID, userID string, groupID *string) (*ProtocolDevModbusPollResult, error) {
	if _, err := s.requireSession(projectID, connectionID, sessionID, userID, "modbus"); err != nil {
		return nil, err
	}
	registers, err := s.listModbusRegisters(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}
	values := s.buildModbusReadValues(registers, nil)
	readPlan := BuildModbusReadPlanEstimate(toModbusRegisters(registers))
	diagnostics := append([]string{}, readPlan.Diagnostics...)
	diagnostics = append(diagnostics, "当前轮询结果为开发态短时读取快照，不写入历史库、不触发报警或计算。")
	return &ProtocolDevModbusPollResult{Values: values, ReadPlan: readPlan, Diagnostics: diagnostics}, nil
}

// StopPollModbus 结束 Modbus 短时轮询占位，会话仍保持连接。
func (s *ProtocolDevSessionService) StopPollModbus(ctx context.Context, projectID, connectionID, sessionID, userID string) (*ProtocolDevSession, error) {
	session, err := s.requireSession(projectID, connectionID, sessionID, userID, "modbus")
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *ProtocolDevSessionService) loadProtocolConnection(ctx context.Context, projectID, connectionID, protocol string) (*ProtocolDevConnection, error) {
	if strings.TrimSpace(projectID) == "" || strings.TrimSpace(connectionID) == "" || strings.TrimSpace(protocol) == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "开发态会话参数不完整")
	}
	if s.connections == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态会话连接仓储未初始化")
	}
	connection, err := s.connections.GetProtocolDevConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if connection == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "接入源不存在")
	}
	if connection.Type != protocol {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "接入源类型与会话协议不匹配")
	}
	return connection, nil
}

func (s *ProtocolDevSessionService) requireSession(projectID, connectionID, sessionID, userID, protocol string) (ProtocolDevSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[sessionID]
	if !ok || session.ProjectID != projectID || session.ConnectionID != connectionID || session.UserID != userID || session.Protocol != protocol {
		return ProtocolDevSession{}, apperrors.NewAppError(apperrors.ErrorCodeNotFound, http.StatusNotFound, "开发态会话不存在或已断开")
	}
	session.lastUsedAt = s.now().UTC()
	s.sessions[sessionID] = session
	return session, nil
}

func (s *ProtocolDevSessionService) listModbusRegisters(ctx context.Context, projectID, connectionID string, groupID *string) ([]ProtocolDevModbusRegister, error) {
	if s.modbus == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "Modbus 开发态模型读取器未初始化")
	}
	return s.modbus.ListDevSessionModbusRegisters(ctx, projectID, connectionID, groupID)
}

func (s *ProtocolDevSessionService) buildModbusReadValues(registers []ProtocolDevModbusRegister, registerIDs []string) []ProtocolDevModbusReadValue {
	wanted := stringSet(registerIDs)
	now := s.now().UTC().Format("2006-01-02 15:04:05")
	values := make([]ProtocolDevModbusReadValue, 0, len(registers))
	for _, register := range registers {
		if len(wanted) > 0 && !wanted[register.ID] && !wanted[register.Code] {
			continue
		}
		raw := sampleModbusRawValue(register)
		values = append(values, ProtocolDevModbusReadValue{
			RegisterID: register.ID,
			SlaveID:    register.UnitID,
			Area:       register.Area,
			Address:    register.Address,
			RawValue:   raw,
			Value:      sampleModbusValue(register, raw),
			DataType:   register.DataType,
			Timestamp:  now,
			Error:      nil,
		})
	}
	return values
}

func protocolDevEndpoint(protocol string, config map[string]any) string {
	if protocol == "opcua" {
		return firstProtocolString(config, "endpoint", "url")
	}
	if firstProtocolString(config, "mode") == "rtu" {
		return "RTU"
	}
	host := firstProtocolString(config, "host")
	port := firstProtocolString(config, "port")
	if host == "" {
		return ""
	}
	if port == "" {
		return host
	}
	return host + ":" + port
}

func firstProtocolString(config map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := config[key]; ok {
			text := strings.TrimSpace(toProtocolString(value))
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func toProtocolString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.Itoa(int(typed))
	default:
		return ""
	}
}

func optionalPrefixedID(prefix string, value *string) *string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	prefixed := prefix + strings.TrimSpace(*value)
	return &prefixed
}

func stringSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		text := strings.TrimSpace(value)
		if text != "" {
			result[text] = true
		}
	}
	return result
}

func sampleProtocolValue(dataType, seed string) any {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bool", "boolean":
		return len(seed)%2 == 0
	case "int", "int16", "int32", "integer", "uint", "uint16", "uint32":
		return 100 + len(seed)
	case "float", "float32", "double":
		return float64(1000+len(seed)) / 10
	default:
		if strings.TrimSpace(seed) != "" {
			return seed
		}
		return "preview"
	}
}

func sampleModbusRawValue(register ProtocolDevModbusRegister) []int {
	quantity := register.Quantity
	if quantity <= 0 {
		quantity = 1
	}
	raw := make([]int, quantity)
	base := register.ProtocolAddress
	if base <= 0 {
		base = register.Address
	}
	for index := range raw {
		raw[index] = base + index + len(register.Code)
	}
	return raw
}

func sampleModbusValue(register ProtocolDevModbusRegister, raw []int) any {
	if len(raw) == 0 {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(register.DataType)) {
	case "bool", "boolean":
		return raw[0]%2 == 1
	case "float", "float32", "double":
		scale := register.Scale
		if scale == 0 {
			scale = 1
		}
		return float64(raw[0])*scale + register.Offset
	default:
		return raw[0]
	}
}

func toModbusRegisters(registers []ProtocolDevModbusRegister) []ModbusRegister {
	result := make([]ModbusRegister, 0, len(registers))
	for _, register := range registers {
		quantity := register.Quantity
		if quantity <= 0 {
			quantity = 1
		}
		pollInterval := register.PollIntervalMS
		if pollInterval <= 0 {
			pollInterval = 1000
		}
		status := strings.TrimSpace(register.Status)
		if status == "" {
			status = "active"
		}
		result = append(result, ModbusRegister{
			ID:              register.ID,
			ProjectID:       register.ProjectID,
			ConnectionID:    register.ConnectionID,
			GroupID:         register.GroupID,
			Name:            register.Name,
			Code:            register.Code,
			UnitID:          register.UnitID,
			Area:            register.Area,
			Address:         register.Address,
			ProtocolAddress: register.ProtocolAddress,
			Quantity:        quantity,
			DataType:        register.DataType,
			PollIntervalMS:  pollInterval,
			Status:          status,
			Scale:           register.Scale,
			Offset:          register.Offset,
		})
	}
	return result
}
