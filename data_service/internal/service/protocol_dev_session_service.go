package service

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/indu-forge/data_service/internal/repository"
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

// ProtocolDevConnectionRepositoryAdapter 把现有连接仓储适配成开发态会话所需的最小接口。
type ProtocolDevConnectionRepositoryAdapter struct {
	repository *repository.ConnectionRepository
}

// NewProtocolDevConnectionRepositoryAdapter 创建连接仓储适配器。
func NewProtocolDevConnectionRepositoryAdapter(repository *repository.ConnectionRepository) *ProtocolDevConnectionRepositoryAdapter {
	return &ProtocolDevConnectionRepositoryAdapter{repository: repository}
}

// GetProtocolDevConnection 读取并转换接入源投影。
func (a *ProtocolDevConnectionRepositoryAdapter) GetProtocolDevConnection(ctx context.Context, projectID, connectionID string) (*ProtocolDevConnection, error) {
	if a == nil || a.repository == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "开发态会话连接仓储未初始化")
	}
	record, err := a.repository.GetByProjectAndID(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, nil
	}
	return &ProtocolDevConnection{
		ID:        record.ID,
		ProjectID: record.ProjectID,
		Type:      record.Type,
		Name:      record.Name,
		Config:    record.Config,
	}, nil
}

// ProtocolDevOpcuaModelReader 表示 OPC UA 开发态会话需要读取的建模数据接口。
type ProtocolDevOpcuaModelReader interface {
	ListDevSessionOpcuaGroups(ctx context.Context, projectID, connectionID string) ([]ProtocolDevOpcuaGroup, error)
	ListDevSessionOpcuaNodes(ctx context.Context, projectID, connectionID string, groupID *string) ([]ProtocolDevOpcuaNode, error)
	UpdateNodeLastValue(ctx context.Context, projectID, connectionID, nodeID, userID string, value any, quality string) error
}

// ProtocolDevModbusModelReader 表示 Modbus 开发态会话需要读取的建模数据接口。
type ProtocolDevModbusModelReader interface {
	ListDevSessionModbusRegisters(ctx context.Context, projectID, connectionID string, groupID *string) ([]ProtocolDevModbusRegister, error)
}

// ProtocolDevS7ModelReader 表示 S7 开发态会话需要读取的建模数据接口。
type ProtocolDevS7ModelReader interface {
	ListDevSessionS7Variables(ctx context.Context, projectID, connectionID string, groupID *string) ([]ProtocolDevS7Variable, error)
	EstimateDevSessionS7ReadPlan(ctx context.Context, projectID, connectionID string, groupID *string) (*S7ReadPlanEstimate, error)
	UpdateVariableLastValue(ctx context.Context, projectID, connectionID, variableID, userID string, value any, quality string) error
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
	ID          string  `json:"id"`
	ParentID    *string `json:"parentId,omitempty"`
	Name        string  `json:"name"`
	NodeID      string  `json:"nodeId"`
	NodeType    string  `json:"nodeType"`
	DataType    string  `json:"dataType,omitempty"`
	Modeled     bool    `json:"modeled"`
	HasChildren bool    `json:"hasChildren"`
	BrowseName  string  `json:"browseName,omitempty"`
	DisplayName string  `json:"displayName,omitempty"`
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

// ProtocolDevS7ReadValue 表示 S7 开发态读取返回的一条值。
type ProtocolDevS7ReadValue struct {
	VariableID string  `json:"variableId"`
	Address    string  `json:"address"`
	RawValue   any     `json:"rawValue"`
	Value      any     `json:"value"`
	DataType   string  `json:"dataType"`
	Quality    string  `json:"quality"`
	Timestamp  string  `json:"timestamp"`
	Error      *string `json:"error,omitempty"`
}

// ProtocolDevS7ReadResult 表示 S7 开发态读取结果。
type ProtocolDevS7ReadResult struct {
	Values      []ProtocolDevS7ReadValue `json:"values"`
	Diagnostics []string                 `json:"diagnostics"`
}

// ProtocolDevS7PollResult 表示 S7 开发态短时轮询结果。
type ProtocolDevS7PollResult struct {
	Values      []ProtocolDevS7ReadValue `json:"values"`
	ReadPlan    S7ReadPlanEstimate       `json:"readPlan"`
	Diagnostics []string                 `json:"diagnostics"`
}

// ProtocolDevOpcuaModelingAdapter 把 OPC UA 建模服务适配为会话浏览/读取投影。
type ProtocolDevOpcuaModelingAdapter struct {
	service *OpcuaModelingService
}

// NewProtocolDevOpcuaModelingAdapter 创建 OPC UA 建模适配器。
func NewProtocolDevOpcuaModelingAdapter(service *OpcuaModelingService) *ProtocolDevOpcuaModelingAdapter {
	return &ProtocolDevOpcuaModelingAdapter{service: service}
}

// ListDevSessionOpcuaGroups 返回会话浏览树使用的变量组投影。
func (a *ProtocolDevOpcuaModelingAdapter) ListDevSessionOpcuaGroups(ctx context.Context, projectID, connectionID string) ([]ProtocolDevOpcuaGroup, error) {
	if a == nil || a.service == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "OPC UA 建模服务未初始化")
	}
	groups, err := a.service.ListGroups(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	result := make([]ProtocolDevOpcuaGroup, 0, len(groups))
	for _, group := range groups {
		result = append(result, ProtocolDevOpcuaGroup{ID: group.ID, ParentID: group.ParentID, Name: group.Name})
	}
	return result, nil
}

// ListDevSessionOpcuaNodes 返回会话读取使用的变量投影。
func (a *ProtocolDevOpcuaModelingAdapter) ListDevSessionOpcuaNodes(ctx context.Context, projectID, connectionID string, groupID *string) ([]ProtocolDevOpcuaNode, error) {
	if a == nil || a.service == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "OPC UA 建模服务未初始化")
	}
	nodes, err := a.service.ListNodes(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}
	result := make([]ProtocolDevOpcuaNode, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, ProtocolDevOpcuaNode{
			ID:       node.ID,
			GroupID:  node.GroupID,
			Name:     node.Name,
			Code:     node.Code,
			NodeID:   node.NodeID,
			DataType: node.DataType,
		})
	}
	return result, nil
}

// UpdateNodeLastValue 写回开发态 OPC UA 读取快照。
func (a *ProtocolDevOpcuaModelingAdapter) UpdateNodeLastValue(ctx context.Context, projectID, connectionID, nodeID, userID string, value any, quality string) error {
	if a == nil || a.service == nil {
		return apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "OPC UA 建模服务未初始化")
	}
	return a.service.UpdateNodeLastValue(ctx, projectID, connectionID, nodeID, userID, value, quality)
}

// ProtocolDevModbusModelingAdapter 把 Modbus 建模服务适配为会话读取/轮询投影。
type ProtocolDevModbusModelingAdapter struct {
	service *ModbusModelingService
}

// NewProtocolDevModbusModelingAdapter 创建 Modbus 建模适配器。
func NewProtocolDevModbusModelingAdapter(service *ModbusModelingService) *ProtocolDevModbusModelingAdapter {
	return &ProtocolDevModbusModelingAdapter{service: service}
}

// ListDevSessionModbusRegisters 返回会话读取使用的寄存器投影。
func (a *ProtocolDevModbusModelingAdapter) ListDevSessionModbusRegisters(ctx context.Context, projectID, connectionID string, groupID *string) ([]ProtocolDevModbusRegister, error) {
	if a == nil || a.service == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "Modbus 建模服务未初始化")
	}
	registers, err := a.service.ListRegisters(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}
	result := make([]ProtocolDevModbusRegister, 0, len(registers))
	for _, register := range registers {
		result = append(result, ProtocolDevModbusRegister{
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
			Quantity:        register.Quantity,
			DataType:        register.DataType,
			PollIntervalMS:  register.PollIntervalMS,
			Status:          register.Status,
			Scale:           register.Scale,
			Offset:          register.Offset,
		})
	}
	return result, nil
}

// ProtocolDevSession 表示 OPC UA / Modbus 工作台的一次开发态短时会话。
type ProtocolDevSession struct {
	SessionID    string         `json:"sessionId"`
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
	connections  ProtocolDevConnectionReader
	opcua        ProtocolDevOpcuaModelReader
	opcuaBrowser ProtocolDevOpcuaBrowser
	modbus       ProtocolDevModbusModelReader
	s7           ProtocolDevS7ModelReader
	mu           sync.Mutex
	sessions     map[string]ProtocolDevSession
	now          func() time.Time
}

// NewProtocolDevSessionService 创建协议开发态会话服务。
func NewProtocolDevSessionService(connections ProtocolDevConnectionReader, opcua ProtocolDevOpcuaModelReader, opcuaBrowser ProtocolDevOpcuaBrowser, modbus ProtocolDevModbusModelReader, s7 ProtocolDevS7ModelReader) *ProtocolDevSessionService {
	return &ProtocolDevSessionService{
		connections:  connections,
		opcua:        opcua,
		opcuaBrowser: opcuaBrowser,
		modbus:       modbus,
		s7:           s7,
		sessions:     map[string]ProtocolDevSession{},
		now:          time.Now,
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
		SessionID:    uuid.NewString(),
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
	if protocol == "opcua" && s.opcuaBrowser != nil {
		if err := s.opcuaBrowser.Open(ctx, session); err != nil {
			return nil, err
		}
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
	if protocol == "opcua" && s.opcuaBrowser != nil {
		s.opcuaBrowser.Close(sessionID)
	}
	return &session, nil
}

// BrowseOpcua 基于当前会话返回真实 OPC UA 地址空间中指定父节点的直接子节点。
func (s *ProtocolDevSessionService) BrowseOpcua(ctx context.Context, projectID, connectionID, sessionID, userID, parentNodeID string) (*ProtocolDevOpcuaBrowseResult, error) {
	session, err := s.requireSession(projectID, connectionID, sessionID, userID, "opcua")
	if err != nil {
		return nil, err
	}
	if s.opcua == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "OPC UA 开发态模型读取器未初始化")
	}
	if s.opcuaBrowser == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "OPC UA 真实浏览适配器未初始化")
	}
	modeledNodeIDs, err := s.modeledOpcuaNodeIDSet(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	result, err := s.opcuaBrowser.Browse(ctx, session, parentNodeID)
	if err != nil {
		return nil, err
	}
	return markModeledBrowseNodes(result, modeledNodeIDs), nil
}

// BrowseOpcuaSubtree 基于当前会话递归收集指定目录下的变量，用于用户主动选择“导入子树变量”。
func (s *ProtocolDevSessionService) BrowseOpcuaSubtree(ctx context.Context, projectID, connectionID, sessionID, userID, parentNodeID string) (*ProtocolDevOpcuaBrowseResult, error) {
	session, err := s.requireSession(projectID, connectionID, sessionID, userID, "opcua")
	if err != nil {
		return nil, err
	}
	if s.opcua == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "OPC UA 开发态模型读取器未初始化")
	}
	if s.opcuaBrowser == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "OPC UA 真实浏览适配器未初始化")
	}
	modeledNodeIDs, err := s.modeledOpcuaNodeIDSet(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	result, err := s.opcuaBrowser.BrowseSubtree(ctx, session, parentNodeID)
	if err != nil {
		return nil, err
	}
	return markModeledBrowseNodes(result, modeledNodeIDs), nil
}

func (s *ProtocolDevSessionService) modeledOpcuaNodeIDSet(ctx context.Context, projectID, connectionID string) (map[string]bool, error) {
	// 建模数据只用于标记真实地址空间中已导入的变量，不再参与生成浏览树，避免误导用户。
	nodes, err := s.opcua.ListDevSessionOpcuaNodes(ctx, projectID, connectionID, nil)
	if err != nil {
		return nil, err
	}
	modeledNodeIDs := map[string]bool{}
	for _, node := range nodes {
		if text := strings.TrimSpace(node.NodeID); text != "" {
			modeledNodeIDs[text] = true
		}
	}
	return modeledNodeIDs, nil
}

func markModeledBrowseNodes(result *ProtocolDevOpcuaBrowseResult, modeledNodeIDs map[string]bool) *ProtocolDevOpcuaBrowseResult {
	if result == nil {
		return &ProtocolDevOpcuaBrowseResult{Nodes: []ProtocolDevBrowseNode{}, Diagnostics: []string{}}
	}
	for index := range result.Nodes {
		if modeledNodeIDs[strings.TrimSpace(result.Nodes[index].NodeID)] {
			result.Nodes[index].Modeled = true
		}
	}
	return result
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
		value := sampleProtocolValue(node.DataType, node.Code)
		quality := "Good"
		_ = s.opcua.UpdateNodeLastValue(ctx, projectID, connectionID, node.ID, userID, value, quality)
		result.Values = append(result.Values, ProtocolDevOpcuaReadValue{
			NodeID:          node.NodeID,
			Value:           value,
			DataType:        node.DataType,
			Quality:         quality,
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

// ReadS7 读取会话内 S7 变量当前值，第一版基于建模数据生成开发态样例值并写回最近值。
func (s *ProtocolDevSessionService) ReadS7(ctx context.Context, projectID, connectionID, sessionID, userID string, variableIDs []string, groupID *string) (*ProtocolDevS7ReadResult, error) {
	if _, err := s.requireSession(projectID, connectionID, sessionID, userID, "s7"); err != nil {
		return nil, err
	}
	variables, err := s.listS7Variables(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}
	values := s.buildS7ReadValues(ctx, projectID, connectionID, userID, variables, variableIDs)
	return &ProtocolDevS7ReadResult{Values: values, Diagnostics: []string{}}, nil
}

// PollS7 返回当前会话的短时轮询快照，并附带与运行态一致的读取计划估算。
func (s *ProtocolDevSessionService) PollS7(ctx context.Context, projectID, connectionID, sessionID, userID string, groupID *string) (*ProtocolDevS7PollResult, error) {
	if _, err := s.requireSession(projectID, connectionID, sessionID, userID, "s7"); err != nil {
		return nil, err
	}
	variables, err := s.listS7Variables(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}
	values := s.buildS7ReadValues(ctx, projectID, connectionID, userID, variables, nil)
	readPlan, err := s.s7.EstimateDevSessionS7ReadPlan(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}
	diagnostics := append([]string{}, readPlan.Diagnostics...)
	diagnostics = append(diagnostics, "当前轮询结果为开发态短时读取快照，不写入历史库、不触发报警或计算。")
	return &ProtocolDevS7PollResult{Values: values, ReadPlan: *readPlan, Diagnostics: diagnostics}, nil
}

// StopPollS7 结束 S7 短时轮询占位，会话仍保持连接。
func (s *ProtocolDevSessionService) StopPollS7(ctx context.Context, projectID, connectionID, sessionID, userID string) (*ProtocolDevSession, error) {
	session, err := s.requireSession(projectID, connectionID, sessionID, userID, "s7")
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

func (s *ProtocolDevSessionService) listS7Variables(ctx context.Context, projectID, connectionID string, groupID *string) ([]ProtocolDevS7Variable, error) {
	if s.s7 == nil {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeInternal, http.StatusInternalServerError, "S7 开发态模型读取器未初始化")
	}
	return s.s7.ListDevSessionS7Variables(ctx, projectID, connectionID, groupID)
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

func (s *ProtocolDevSessionService) buildS7ReadValues(ctx context.Context, projectID, connectionID, userID string, variables []ProtocolDevS7Variable, variableIDs []string) []ProtocolDevS7ReadValue {
	wanted := stringSet(variableIDs)
	now := s.now().UTC().Format("2006-01-02 15:04:05")
	values := make([]ProtocolDevS7ReadValue, 0, len(variables))
	for _, variable := range variables {
		if len(wanted) > 0 && !wanted[variable.ID] && !wanted[variable.Code] {
			continue
		}
		raw := sampleS7RawValue(variable)
		value := sampleS7Value(variable, raw)
		quality := "Good"
		_ = s.s7.UpdateVariableLastValue(ctx, projectID, connectionID, variable.ID, userID, value, quality)
		values = append(values, ProtocolDevS7ReadValue{
			VariableID: variable.ID,
			Address:    firstNonEmpty(variable.NormalizedAddress, variable.AddressText),
			RawValue:   raw,
			Value:      value,
			DataType:   variable.DataType,
			Quality:    quality,
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
	if protocol == "s7" {
		host := firstProtocolString(config, "host", "ip")
		port := firstProtocolString(config, "port")
		if port == "" {
			port = "102"
		}
		if host == "" {
			return ""
		}
		return host + ":" + port
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

func sampleS7RawValue(variable ProtocolDevS7Variable) any {
	base := variable.ByteOffset + len(variable.Code)
	switch strings.ToLower(strings.TrimSpace(variable.DataType)) {
	case "bool", "boolean":
		return base%2 == 0
	case "real", "float", "float32", "double":
		return float64(1000+base) / 10
	case "string":
		return firstNonEmpty(variable.Code, variable.Name, "preview")
	default:
		return base
	}
}

func sampleS7Value(variable ProtocolDevS7Variable, raw any) any {
	switch typed := raw.(type) {
	case int:
		return float64(typed)*variable.Scale + variable.Offset
	case float64:
		scale := variable.Scale
		if scale == 0 {
			scale = 1
		}
		return typed*scale + variable.Offset
	default:
		return typed
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
