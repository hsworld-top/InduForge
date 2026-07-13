# OPC UA / Modbus 开发态会话工作台实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 为 OPC UA 与 Modbus 工作台补齐开发态连接 / 断开会话、在线浏览 / 读取 / 短时订阅或轮询入口，并将工作台布局调整为与 MQTT 类似的左中右结构。

**架构：** `data_service` 新增协议开发态会话服务，第一版使用可测试的建模数据适配器返回浏览、读取和轮询结果，确保接口、状态机和前端体验闭环。`datacenter` 新增会话 API 与前端状态机，把接入源 Header 移到左侧栏，把变量级操作移到中间顶部工具条。

**技术栈：** Go `net/http` + 现有 repository/service/handler/router 分层，Vue 3 `<script setup>` + Element Plus + 现有 `data.api.ts` 请求封装。

---

## 文件结构

后端新增和修改：

- 创建：`data_service/internal/service/protocol_dev_session_service.go`  
  职责：定义开发态会话模型、会话内存存储、OPC UA / Modbus 建模数据适配器、连接 / 断开 / 浏览 / 读取 / 订阅 / 轮询业务逻辑。
- 创建：`data_service/internal/service/protocol_dev_session_service_test.go`  
  职责：覆盖会话状态、项目连接校验、OPC UA 浏览/读取、Modbus 读取/轮询和用户边界。
- 创建：`data_service/internal/http/handler/protocol_dev_session_handler.go`  
  职责：暴露 OPC UA / Modbus 开发态会话 HTTP 接口，复用统一响应结构。
- 修改：`data_service/internal/http/router/router.go`  
  职责：挂载协议开发态会话路由。
- 修改：`data_service/internal/app/server.go`  
  职责：初始化 `ProtocolDevSessionService` 与 handler。

前端新增和修改：

- 修改：`datacenter/src/api/data.api.ts`  
  职责：新增 OPC UA / Modbus 会话 API。
- 修改：`datacenter/src/components/opcua/types.ts`  
  职责：补充 OPC UA 会话、浏览节点、读取值类型。
- 修改：`datacenter/src/components/modbus/types.ts`  
  职责：补充 Modbus 会话、寄存器读取值类型。
- 创建：`datacenter/src/components/access-source/workbench/useProtocolDevSession.ts`  
  职责：封装连接 / 断开状态机、卸载清理、错误状态。
- 修改：`datacenter/src/components/access-source/workbench/OpcuaWorkbenchPanel.vue`  
  职责：移除全宽 Header，改成左中右布局；接入 OPC UA 会话；中间顶部放当前分组操作；预览读取当前分组。
- 修改：`datacenter/src/components/access-source/workbench/ModbusWorkbenchPanel.vue`  
  职责：移除全宽 Header，改成左中右布局；接入 Modbus 会话；中间顶部放当前分组操作与读取计划入口。
- 修改：`datacenter/src/components/opcua/OpcuaPreviewDialog.vue`  
  职责：展示会话读取返回的值、质量码和时间戳。
- 修改：`datacenter/src/components/modbus/ModbusPreviewDialog.vue`  
  职责：展示会话读取返回的原始值、解析值和时间戳。
- 修改：`datacenter/src/components/opcua/OpcuaValidationDrawer.vue`  
  职责：展示当前校验范围文案。
- 修改：`datacenter/src/components/modbus/ModbusValidationDrawer.vue`  
  职责：展示当前校验范围文案。

验证命令：

- `pnpm go:test:data`
- `pnpm --filter datacenter typecheck`
- `pnpm --filter datacenter build`

---

### 任务 1：后端会话服务骨架

**文件：**

- 创建：`data_service/internal/service/protocol_dev_session_service.go`
- 创建：`data_service/internal/service/protocol_dev_session_service_test.go`

- [ ] **步骤 1：编写失败的会话状态测试**

在 `data_service/internal/service/protocol_dev_session_service_test.go` 中创建测试：

```go
package service

import (
	"context"
	"testing"
)

func TestProtocolDevSessionService_CreateAndCloseSession(t *testing.T) {
	repo := &fakeProtocolDevConnectionRepository{
		connection: fakeProtocolConnection{ID: "conn-1", ProjectID: "project-1", Type: "opcua", Name: "opcua-main", Config: map[string]any{"endpoint": "opc.tcp://127.0.0.1:4840"}},
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
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
cd data_service
go test ./internal/service -run TestProtocolDevSessionService_CreateAndCloseSession -count=1
```

预期：FAIL，提示 `NewProtocolDevSessionService` 或相关类型未定义。

- [ ] **步骤 3：实现会话服务最小骨架**

创建 `data_service/internal/service/protocol_dev_session_service.go`：

```go
package service

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	apperrors "github.com/indu-forge/data_service/internal/errors"
	"github.com/google/uuid"
)

type protocolDevConnection struct {
	ID        string
	ProjectID string
	Type      string
	Name      string
	Config    map[string]any
}

type ProtocolDevConnectionReader interface {
	GetProtocolDevConnection(ctx context.Context, projectID, connectionID string) (*protocolDevConnection, error)
}

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

type ProtocolDevSessionService struct {
	connections ProtocolDevConnectionReader
	opcua       ProtocolDevOpcuaModelReader
	modbus      ProtocolDevModbusModelReader
	mu          sync.Mutex
	sessions    map[string]ProtocolDevSession
	now         func() time.Time
}

func NewProtocolDevSessionService(connections ProtocolDevConnectionReader, opcua ProtocolDevOpcuaModelReader, modbus ProtocolDevModbusModelReader) *ProtocolDevSessionService {
	return &ProtocolDevSessionService{
		connections: connections,
		opcua:       opcua,
		modbus:      modbus,
		sessions:    map[string]ProtocolDevSession{},
		now:         time.Now,
	}
}

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

func (s *ProtocolDevSessionService) loadProtocolConnection(ctx context.Context, projectID, connectionID, protocol string) (*protocolDevConnection, error) {
	if strings.TrimSpace(projectID) == "" || strings.TrimSpace(connectionID) == "" || strings.TrimSpace(protocol) == "" {
		return nil, apperrors.NewAppError(apperrors.ErrorCodeBadRequest, http.StatusBadRequest, "开发态会话参数不完整")
	}
	connection, err := s.connections.GetProtocolDevConnection(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
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
	case float64:
		return strconv.Itoa(int(typed))
	default:
		return ""
	}
}
```

同时在文件 import 中加入 `strconv`。测试内补充 fake 类型：

```go
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

func (r *fakeProtocolDevConnectionRepository) GetProtocolDevConnection(_ context.Context, projectID, connectionID string) (*protocolDevConnection, error) {
	if r.connection.ProjectID != projectID || r.connection.ID != connectionID {
		return nil, nil
	}
	return &protocolDevConnection{
		ID:        r.connection.ID,
		ProjectID: r.connection.ProjectID,
		Type:      r.connection.Type,
		Name:      r.connection.Name,
		Config:    r.connection.Config,
	}, nil
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：

```powershell
cd data_service
go test ./internal/service -run TestProtocolDevSessionService_CreateAndCloseSession -count=1
```

预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add data_service/internal/service/protocol_dev_session_service.go data_service/internal/service/protocol_dev_session_service_test.go
git commit -m "feat(datacenter): 增加协议开发态会话服务骨架"
```

---

### 任务 2：后端 OPC UA 浏览与读取

**文件：**

- 修改：`data_service/internal/service/protocol_dev_session_service.go`
- 修改：`data_service/internal/service/protocol_dev_session_service_test.go`

- [ ] **步骤 1：编写 OPC UA 浏览/读取测试**

追加测试：

```go
func TestProtocolDevSessionService_BrowseAndReadOpcuaNodes(t *testing.T) {
	repo := &fakeProtocolDevConnectionRepository{
		connection: fakeProtocolConnection{ID: "conn-1", ProjectID: "project-1", Type: "opcua", Name: "opcua-main", Config: map[string]any{"endpoint": "opc.tcp://127.0.0.1:4840"}},
	}
	opcua := &fakeProtocolDevOpcuaReader{
		groups: []ProtocolDevOpcuaGroup{{ID: "g-1", Name: "Furnace01"}},
		nodes: []ProtocolDevOpcuaNode{{ID: "n-1", GroupID: stringPtr("g-1"), Name: "Temp", Code: "temp", NodeID: "ns=2;s=Furnace01.Temp", DataType: "Double"}},
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
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
cd data_service
go test ./internal/service -run TestProtocolDevSessionService_BrowseAndReadOpcuaNodes -count=1
```

预期：FAIL，提示 `ProtocolDevOpcuaGroup`、`BrowseOpcua` 或 `ReadOpcua` 未定义。

- [ ] **步骤 3：实现 OPC UA 建模数据适配**

在服务文件中追加：

```go
type ProtocolDevOpcuaModelReader interface {
	ListDevSessionOpcuaGroups(ctx context.Context, projectID, connectionID string) ([]ProtocolDevOpcuaGroup, error)
	ListDevSessionOpcuaNodes(ctx context.Context, projectID, connectionID string, groupID *string) ([]ProtocolDevOpcuaNode, error)
}

type ProtocolDevOpcuaGroup struct {
	ID       string
	ParentID *string
	Name     string
}

type ProtocolDevOpcuaNode struct {
	ID       string
	GroupID  *string
	Name     string
	Code     string
	NodeID   string
	DataType string
}

type ProtocolDevBrowseNode struct {
	ID       string `json:"id"`
	ParentID *string `json:"parentId,omitempty"`
	Name     string `json:"name"`
	NodeID   string `json:"nodeId"`
	NodeType string `json:"nodeType"`
	DataType string `json:"dataType,omitempty"`
}

type ProtocolDevOpcuaBrowseResult struct {
	Nodes       []ProtocolDevBrowseNode `json:"nodes"`
	Diagnostics []string                `json:"diagnostics"`
}

type ProtocolDevOpcuaReadValue struct {
	NodeID          string `json:"nodeId"`
	Value           any    `json:"value"`
	DataType        string `json:"dataType"`
	Quality         string `json:"quality"`
	SourceTimestamp string `json:"sourceTimestamp"`
	ServerTimestamp string `json:"serverTimestamp"`
	Error           *string `json:"error"`
}

type ProtocolDevOpcuaReadResult struct {
	Values      []ProtocolDevOpcuaReadValue `json:"values"`
	Diagnostics []string                    `json:"diagnostics"`
}

func (s *ProtocolDevSessionService) BrowseOpcua(ctx context.Context, projectID, connectionID, sessionID, userID string) (*ProtocolDevOpcuaBrowseResult, error) {
	if _, err := s.requireSession(projectID, connectionID, sessionID, userID, "opcua"); err != nil {
		return nil, err
	}
	groups, err := s.opcua.ListDevSessionOpcuaGroups(ctx, projectID, connectionID)
	if err != nil {
		return nil, err
	}
	nodes, err := s.opcua.ListDevSessionOpcuaNodes(ctx, projectID, connectionID, nil)
	if err != nil {
		return nil, err
	}
	result := &ProtocolDevOpcuaBrowseResult{Nodes: []ProtocolDevBrowseNode{}, Diagnostics: []string{"当前浏览结果来自已建模变量，真实 OPC UA 地址空间适配器接入后保持同一响应结构。"}}
	for _, group := range groups {
		result.Nodes = append(result.Nodes, ProtocolDevBrowseNode{ID: "group-" + group.ID, ParentID: group.ParentID, Name: group.Name, NodeID: group.ID, NodeType: "folder"})
	}
	for _, node := range nodes {
		parentID := optionalPrefixedID("group-", node.GroupID)
		result.Nodes = append(result.Nodes, ProtocolDevBrowseNode{ID: node.ID, ParentID: parentID, Name: node.Name, NodeID: node.NodeID, NodeType: "variable", DataType: node.DataType})
	}
	return result, nil
}

func (s *ProtocolDevSessionService) ReadOpcua(ctx context.Context, projectID, connectionID, sessionID, userID string, nodeIDs []string, groupID *string) (*ProtocolDevOpcuaReadResult, error) {
	if _, err := s.requireSession(projectID, connectionID, sessionID, userID, "opcua"); err != nil {
		return nil, err
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
```

同时实现 `stringSet`、`optionalPrefixedID`、`sampleProtocolValue`，并在测试中补 fake reader 方法。

- [ ] **步骤 4：运行测试验证通过**

运行：

```powershell
cd data_service
go test ./internal/service -run TestProtocolDevSessionService_BrowseAndReadOpcuaNodes -count=1
```

预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add data_service/internal/service/protocol_dev_session_service.go data_service/internal/service/protocol_dev_session_service_test.go
git commit -m "feat(datacenter): 增加 OPC UA 开发态浏览读取能力"
```

---

### 任务 3：后端 Modbus 读取与轮询

**文件：**

- 修改：`data_service/internal/service/protocol_dev_session_service.go`
- 修改：`data_service/internal/service/protocol_dev_session_service_test.go`

- [ ] **步骤 1：编写 Modbus 读取测试**

追加测试：

```go
func TestProtocolDevSessionService_ReadAndPollModbusRegisters(t *testing.T) {
	repo := &fakeProtocolDevConnectionRepository{
		connection: fakeProtocolConnection{ID: "conn-1", ProjectID: "project-1", Type: "modbus", Name: "modbus-main", Config: map[string]any{"host": "127.0.0.1", "port": 502}},
	}
	modbus := &fakeProtocolDevModbusReader{
		registers: []ProtocolDevModbusRegister{{ID: "r-1", Name: "Speed", Code: "speed", UnitID: 1, Area: "holding", Address: 40001, DataType: "int16"}},
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
	if len(poll.Values) != 1 {
		t.Fatalf("unexpected poll result: %#v", poll)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
cd data_service
go test ./internal/service -run TestProtocolDevSessionService_ReadAndPollModbusRegisters -count=1
```

预期：FAIL，提示 Modbus 会话读取类型或方法未定义。

- [ ] **步骤 3：实现 Modbus 读取和轮询**

在服务文件中追加：

```go
type ProtocolDevModbusModelReader interface {
	ListDevSessionModbusRegisters(ctx context.Context, projectID, connectionID string, groupID *string) ([]ProtocolDevModbusRegister, error)
}

type ProtocolDevModbusRegister struct {
	ID       string
	GroupID  *string
	Name     string
	Code     string
	UnitID   int
	Area     string
	Address  int
	DataType string
}

type ProtocolDevModbusReadValue struct {
	RegisterID string `json:"registerId"`
	SlaveID    int    `json:"slaveId"`
	Area       string `json:"area"`
	Address    int    `json:"address"`
	RawValue   []int  `json:"rawValue"`
	Value      any    `json:"value"`
	DataType   string `json:"dataType"`
	Timestamp  string `json:"timestamp"`
	Error      *string `json:"error"`
}

type ProtocolDevModbusReadResult struct {
	Values      []ProtocolDevModbusReadValue `json:"values"`
	Diagnostics []string                     `json:"diagnostics"`
}

func (s *ProtocolDevSessionService) ReadModbus(ctx context.Context, projectID, connectionID, sessionID, userID string, registerIDs []string, groupID *string) (*ProtocolDevModbusReadResult, error) {
	if _, err := s.requireSession(projectID, connectionID, sessionID, userID, "modbus"); err != nil {
		return nil, err
	}
	return s.readModbusRegisters(ctx, projectID, connectionID, registerIDs, groupID)
}

func (s *ProtocolDevSessionService) PollModbus(ctx context.Context, projectID, connectionID, sessionID, userID string, groupID *string) (*ProtocolDevModbusReadResult, error) {
	if _, err := s.requireSession(projectID, connectionID, sessionID, userID, "modbus"); err != nil {
		return nil, err
	}
	return s.readModbusRegisters(ctx, projectID, connectionID, nil, groupID)
}

func (s *ProtocolDevSessionService) readModbusRegisters(ctx context.Context, projectID, connectionID string, registerIDs []string, groupID *string) (*ProtocolDevModbusReadResult, error) {
	registers, err := s.modbus.ListDevSessionModbusRegisters(ctx, projectID, connectionID, groupID)
	if err != nil {
		return nil, err
	}
	wanted := stringSet(registerIDs)
	now := s.now().UTC().Format("2006-01-02 15:04:05")
	result := &ProtocolDevModbusReadResult{Values: []ProtocolDevModbusReadValue{}, Diagnostics: []string{}}
	for _, register := range registers {
		if len(wanted) > 0 && !wanted[register.ID] {
			continue
		}
		raw := []int{register.Address % 256, (register.Address / 256) % 256}
		result.Values = append(result.Values, ProtocolDevModbusReadValue{
			RegisterID: register.ID,
			SlaveID:    register.UnitID,
			Area:       register.Area,
			Address:    register.Address,
			RawValue:   raw,
			Value:      sampleProtocolValue(register.DataType, register.Code),
			DataType:   register.DataType,
			Timestamp:  now,
			Error:      nil,
		})
	}
	return result, nil
}
```

测试文件中补 `fakeProtocolDevModbusReader`。

- [ ] **步骤 4：运行测试验证通过**

运行：

```powershell
cd data_service
go test ./internal/service -run TestProtocolDevSessionService_ReadAndPollModbusRegisters -count=1
```

预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add data_service/internal/service/protocol_dev_session_service.go data_service/internal/service/protocol_dev_session_service_test.go
git commit -m "feat(datacenter): 增加 Modbus 开发态读取轮询能力"
```

---

### 任务 4：仓储适配与路由装配

**文件：**

- 修改：`data_service/internal/repository/connection_repository.go`
- 修改：`data_service/internal/repository/opcua_modeling_repository.go`
- 修改：`data_service/internal/repository/modbus_modeling_repository.go`
- 创建：`data_service/internal/http/handler/protocol_dev_session_handler.go`
- 修改：`data_service/internal/http/router/router.go`
- 修改：`data_service/internal/app/server.go`

- [ ] **步骤 1：编写 handler 路由测试**

创建或追加 `data_service/internal/http/handler/protocol_dev_session_handler_test.go`：

```go
package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProtocolDevSessionHandler_CreateOpcuaSessionRequiresAuth(t *testing.T) {
	handler := NewProtocolDevSessionHandler(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/data/projects/project-1/opcua/conn-1/sessions", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	err := handler.CreateOpcuaSession(rec, req)
	if err == nil {
		t.Fatal("expected auth error")
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：

```powershell
cd data_service
go test ./internal/http/handler -run TestProtocolDevSessionHandler_CreateOpcuaSessionRequiresAuth -count=1
```

预期：FAIL，提示 handler 未定义。

- [ ] **步骤 3：实现 repository 适配方法**

在 `connection_repository.go` 增加：

```go
func (r *ConnectionRepository) GetProtocolDevConnection(ctx context.Context, projectID, connectionID string) (*service.ProtocolDevConnection, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, project_id, name, type, metadata
		FROM data_connections
		WHERE project_id = $1 AND id = $2
	`, projectID, connectionID)
	// 按现有 ConnectionRecord 扫描风格实现，并把 metadata 作为 Config 返回。
}
```

如果不能从 repository 引用 service 类型，则把 `protocolDevConnection` 移到 `repository` 或新增轻量 DTO，保持单向依赖不破坏现有结构。最终代码必须保持 `repository -> service` 无反向依赖。

在 `opcua_modeling_repository.go` 增加：

```go
func (r *OpcuaModelingRepository) ListDevSessionOpcuaGroups(ctx context.Context, projectID, connectionID string) ([]service.ProtocolDevOpcuaGroup, error) {
	// 查询 data_opcua_node_groups 的 id,parent_id,name，按 sort_order 返回。
}

func (r *OpcuaModelingRepository) ListDevSessionOpcuaNodes(ctx context.Context, projectID, connectionID string, groupID *string) ([]service.ProtocolDevOpcuaNode, error) {
	// 查询 active data_opcua_nodes 的 id,group_id,name,code,node_id,data_type。
}
```

在 `modbus_modeling_repository.go` 增加：

```go
func (r *ModbusModelingRepository) ListDevSessionModbusRegisters(ctx context.Context, projectID, connectionID string, groupID *string) ([]service.ProtocolDevModbusRegister, error) {
	// 查询 active data_modbus_registers 的 id,group_id,name,code,unit_id,area,address,data_type。
}
```

如果出现 import cycle，按下面结构调整：服务层定义 repository-facing 接口使用本包私有 DTO，repository 方法返回 repository DTO，服务构造时用 adapter 包装。不要让 repository import service。

- [ ] **步骤 4：实现 handler 和 router**

创建 `protocol_dev_session_handler.go`：

```go
type ProtocolDevSessionHandler struct {
	service *service.ProtocolDevSessionService
}

func NewProtocolDevSessionHandler(service *service.ProtocolDevSessionService) *ProtocolDevSessionHandler {
	return &ProtocolDevSessionHandler{service: service}
}

func (h *ProtocolDevSessionHandler) CreateOpcuaSession(w http.ResponseWriter, r *http.Request) error {
	claims, err := requireClaims(r)
	if err != nil {
		return err
	}
	result, err := h.service.CreateSession(r.Context(), r.PathValue("projectId"), r.PathValue("connectionId"), claims.UserID, "opcua")
	if err != nil {
		return normalizeRepresentativeHandlerError(err)
	}
	response.WriteSuccess(w, middleware.RequestID(r.Context()), result)
	return nil
}
```

同文件继续实现：

- `CloseOpcuaSession`
- `BrowseOpcua`
- `ReadOpcua`
- `SubscribeOpcua`
- `StopSubscribeOpcua`
- `CreateModbusSession`
- `CloseModbusSession`
- `ReadModbus`
- `PollModbus`
- `StopPollModbus`

`SubscribeOpcua` 第一版调用 `ReadOpcua` 并返回同结构，`StopSubscribeOpcua` 返回 `{"stopped": true}`。`StopPollModbus` 返回 `{"stopped": true}`。

在 `router.go` 增加 `WithProtocolDevSessionRoutes` 并挂载：

```go
opcuaBase := "/api/v1/data/projects/{projectId}/opcua/{connectionId}/sessions"
mux.Handle("POST "+opcuaBase, write(opts.protocolDevSessionHandler.CreateOpcuaSession))
mux.Handle("DELETE "+opcuaBase+"/{sessionId}", write(opts.protocolDevSessionHandler.CloseOpcuaSession))
mux.Handle("GET "+opcuaBase+"/{sessionId}/browse", read(opts.protocolDevSessionHandler.BrowseOpcua))
mux.Handle("POST "+opcuaBase+"/{sessionId}/read", read(opts.protocolDevSessionHandler.ReadOpcua))
mux.Handle("POST "+opcuaBase+"/{sessionId}/subscribe", read(opts.protocolDevSessionHandler.SubscribeOpcua))
mux.Handle("DELETE "+opcuaBase+"/{sessionId}/subscribe", write(opts.protocolDevSessionHandler.StopSubscribeOpcua))
```

Modbus 同理挂载 `modbus/{connectionId}/sessions`。

在 `server.go` 初始化 service 和 handler，并加入 route option 与 summary。

- [ ] **步骤 5：运行后端测试**

运行：

```powershell
pnpm go:test:data
```

预期：PASS。

- [ ] **步骤 6：Commit**

```powershell
git add data_service/internal/repository data_service/internal/http data_service/internal/app/server.go
git commit -m "feat(datacenter): 暴露协议开发态会话接口"
```

---

### 任务 5：前端会话 API 与状态机

**文件：**

- 修改：`datacenter/src/api/data.api.ts`
- 修改：`datacenter/src/components/opcua/types.ts`
- 修改：`datacenter/src/components/modbus/types.ts`
- 创建：`datacenter/src/components/access-source/workbench/useProtocolDevSession.ts`

- [ ] **步骤 1：补充前端类型**

在 `opcua/types.ts` 追加：

```ts
export type ProtocolDevSession = {
  sessionId: string
  status: 'connected' | 'closed' | 'error'
  protocol: string
  connectedAt: string
  endpoint: string
  diagnostics: string[]
}

export type OpcuaBrowseNode = {
  id: string
  parentId?: string | null
  name: string
  nodeId: string
  nodeType: 'folder' | 'variable'
  dataType?: string
}

export type OpcuaReadValue = {
  nodeId: string
  value: unknown
  dataType: string
  quality: string
  sourceTimestamp: string
  serverTimestamp: string
  error?: string | null
}
```

在 `modbus/types.ts` 追加：

```ts
export type ModbusReadValue = {
  registerId: string
  slaveId: number
  area: string
  address: number
  rawValue: number[]
  value: unknown
  dataType: string
  timestamp: string
  error?: string | null
}
```

- [ ] **步骤 2：新增 API 方法**

在 `data.api.ts` 的 OPC UA 方法附近追加：

```ts
export const createOpcuaDevSession = (projectId, connectionId) =>
  request({ url: `/data/projects/${projectId}/opcua/${connectionId}/sessions`, method: 'post' })

export const closeOpcuaDevSession = (projectId, connectionId, sessionId) =>
  request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/sessions/${sessionId}`,
    method: 'delete',
  })

export const browseOpcuaDevSession = (projectId, connectionId, sessionId) =>
  request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/sessions/${sessionId}/browse`,
    method: 'get',
  })

export const readOpcuaDevSession = (projectId, connectionId, sessionId, data = {}) =>
  request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/sessions/${sessionId}/read`,
    method: 'post',
    data,
  })

export const subscribeOpcuaDevSession = (projectId, connectionId, sessionId, data = {}) =>
  request({
    url: `/data/projects/${projectId}/opcua/${connectionId}/sessions/${sessionId}/subscribe`,
    method: 'post',
    data,
  })
```

在 Modbus 方法附近追加：

```ts
export const createModbusDevSession = (projectId, connectionId) =>
  request({ url: `/data/projects/${projectId}/modbus/${connectionId}/sessions`, method: 'post' })

export const closeModbusDevSession = (projectId, connectionId, sessionId) =>
  request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/sessions/${sessionId}`,
    method: 'delete',
  })

export const readModbusDevSession = (projectId, connectionId, sessionId, data = {}) =>
  request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/sessions/${sessionId}/read`,
    method: 'post',
    data,
  })

export const pollModbusDevSession = (projectId, connectionId, sessionId, data = {}) =>
  request({
    url: `/data/projects/${projectId}/modbus/${connectionId}/sessions/${sessionId}/poll`,
    method: 'post',
    data,
  })
```

- [ ] **步骤 3：创建会话 composable**

创建 `useProtocolDevSession.ts`：

```ts
import { computed, onBeforeUnmount, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getApiErrorMessage } from '@/utils/request'

type SessionApi = {
  create: () => Promise<any>
  close: (sessionId: string) => Promise<any>
}

export const useProtocolDevSession = (api: SessionApi) => {
  const status = ref<'idle' | 'connecting' | 'connected' | 'disconnecting' | 'error'>('idle')
  const sessionId = ref('')
  const diagnostics = ref<string[]>([])

  const connected = computed(() => status.value === 'connected' && Boolean(sessionId.value))

  const connect = async () => {
    if (status.value === 'connecting' || connected.value) return
    status.value = 'connecting'
    try {
      const response = await api.create()
      const data = response?.data?.data || response?.data || {}
      sessionId.value = data.sessionId || ''
      diagnostics.value = data.diagnostics || []
      status.value = 'connected'
    } catch (error) {
      status.value = 'error'
      ElMessage.error(getApiErrorMessage(error, '连接失败'))
    }
  }

  const disconnect = async () => {
    if (!sessionId.value) {
      status.value = 'idle'
      return
    }
    status.value = 'disconnecting'
    const closingId = sessionId.value
    sessionId.value = ''
    try {
      await api.close(closingId)
      status.value = 'idle'
    } catch (error) {
      status.value = 'error'
      ElMessage.error(getApiErrorMessage(error, '断开连接失败'))
    }
  }

  onBeforeUnmount(() => {
    if (sessionId.value) {
      void api.close(sessionId.value)
    }
  })

  return { status, sessionId, diagnostics, connected, connect, disconnect }
}
```

- [ ] **步骤 4：运行前端类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 5：Commit**

```powershell
git add datacenter/src/api/data.api.ts datacenter/src/components/opcua/types.ts datacenter/src/components/modbus/types.ts datacenter/src/components/access-source/workbench/useProtocolDevSession.ts
git commit -m "feat(datacenter): 增加协议开发态会话前端 API"
```

---

### 任务 6：OPC UA 工作台布局与会话接入

**文件：**

- 修改：`datacenter/src/components/access-source/workbench/OpcuaWorkbenchPanel.vue`
- 修改：`datacenter/src/components/opcua/OpcuaPreviewDialog.vue`
- 修改：`datacenter/src/components/opcua/OpcuaValidationDrawer.vue`

- [ ] **步骤 1：调整模板为左中右布局**

把 `OpcuaWorkbenchPanel.vue` 顶部全宽 Header 移入左侧栏：

```vue
<section class="opcua-workbench">
  <aside class="opcua-workbench__side">
    <WorkbenchSourceHeader ...>
      <template #status>
        <button type="button" class="opcua-workbench__connect-action" :class="{ 'is-connected': session.connected.value }" @click="toggleSession">
          {{ session.connected.value ? '已连接' : '连接' }}
        </button>
      </template>
    </WorkbenchSourceHeader>
    <OpcuaGroupTree ... />
  </aside>

  <main class="opcua-workbench__main">
    <div class="opcua-workbench__bar">
      <div class="opcua-workbench__title">...</div>
      <div class="opcua-workbench__actions">
        <button :title="importActionTitle" @click="importVisible = true"><IconTablerUpload /></button>
        <button @click="openCreateNode"><IconTablerPlus /></button>
        <button :disabled="!session.connected.value" title="预览当前分组变量" @click="openPreview"><IconTablerActivityHeartbeat /></button>
        <button :title="validationActionTitle" @click="runValidation"><IconTablerChecklist /></button>
        <button @click="reloadAll"><IconTablerRefresh /></button>
      </div>
      <el-input ... />
    </div>
    <OpcuaNodeTable ... />
  </main>

  <OpcuaInspectorPanel ... />
</section>
```

- [ ] **步骤 2：接入会话状态机**

在 script 中加入：

```ts
import { useProtocolDevSession } from './useProtocolDevSession'

const session = useProtocolDevSession({
  create: () => dataAPI.createOpcuaDevSession(props.projectId, props.connection.id),
  close: (sessionId: string) =>
    dataAPI.closeOpcuaDevSession(props.projectId, props.connection.id, sessionId),
})

const toggleSession = () => {
  if (session.connected.value) void session.disconnect()
  else void session.connect()
}
```

移除 `testing` 与 `runConnectionTest`。`变量预览` 未连接时禁用并显示 tooltip 或 title `连接后可预览当前分组变量`。

- [ ] **步骤 3：让预览读取会话值**

改 `openPreview`：

```ts
const openPreview = async () => {
  if (!session.connected.value || !session.sessionId.value) {
    ElMessage.warning('请先连接 OPC UA 开发态会话')
    return
  }
  try {
    const response = await dataAPI.subscribeOpcuaDevSession(
      props.projectId,
      props.connection.id,
      session.sessionId.value,
      {
        groupId: selectedGroupId.value || null,
      },
    )
    const data = response?.data?.data || response?.data || {}
    previewNodes.value = data.values || []
    previewDiagnostics.value = data.diagnostics || []
    previewVisible.value = true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '变量预览失败'))
  }
}
```

`OpcuaPreviewDialog.vue` 表格列改为读取值结构：`nodeId / value / dataType / quality / sourceTimestamp / serverTimestamp / error`。

- [ ] **步骤 4：按当前上下文过滤校验问题**

增加：

```ts
const scopedValidationIssues = computed(() => {
  if (selectedNodeId.value)
    return validationIssues.value.filter((item) => item.nodeId === selectedNodeId.value)
  if (selectedGroupId.value) {
    const ids = new Set(
      nodes.value.filter((node) => node.groupId === selectedGroupId.value).map((node) => node.id),
    )
    return validationIssues.value.filter((item) => item.nodeId && ids.has(item.nodeId))
  }
  return validationIssues.value
})

const validationScopeLabel = computed(() => {
  if (selectedNode.value) return `当前变量：${selectedNode.value.name}`
  if (currentGroup.value) return `当前分组：${currentGroup.value.name}`
  return '全部变量'
})
```

传给抽屉：

```vue
<OpcuaValidationDrawer
  v-model="validationVisible"
  :issues="scopedValidationIssues"
  :scope-label="validationScopeLabel"
  @locate="locateNode"
/>
```

- [ ] **步骤 5：更新样式**

把 `.opcua-workbench` 改为三栏 grid：

```css
.opcua-workbench {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr) 286px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
  overflow: hidden;
}

.opcua-workbench__side {
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--dc-border);
  overflow: hidden;
}

.opcua-workbench__side :deep(.opcua-groups) {
  flex: 1;
  min-height: 0;
}

.opcua-workbench__bar {
  grid-template-columns: minmax(0, 1fr) auto 230px;
}
```

- [ ] **步骤 6：运行前端验证**

运行：

```powershell
pnpm --filter datacenter typecheck
pnpm --filter datacenter build
```

预期：PASS。

- [ ] **步骤 7：Commit**

```powershell
git add datacenter/src/components/access-source/workbench/OpcuaWorkbenchPanel.vue datacenter/src/components/opcua/OpcuaPreviewDialog.vue datacenter/src/components/opcua/OpcuaValidationDrawer.vue
git commit -m "feat(datacenter): 调整 OPC UA 工作台开发态会话布局"
```

---

### 任务 7：Modbus 工作台布局与会话接入

**文件：**

- 修改：`datacenter/src/components/access-source/workbench/ModbusWorkbenchPanel.vue`
- 修改：`datacenter/src/components/modbus/ModbusPreviewDialog.vue`
- 修改：`datacenter/src/components/modbus/ModbusValidationDrawer.vue`

- [ ] **步骤 1：调整模板为左中右布局**

把 `ModbusWorkbenchPanel.vue` 顶部全宽 Header 移入左侧栏，结构与 OPC UA 对齐：

```vue
<section class="modbus-workbench">
  <aside class="modbus-workbench__side">
    <WorkbenchSourceHeader ...>
      <template #status>
        <button type="button" class="modbus-workbench__connect-action" :class="{ 'is-connected': session.connected.value }" @click="toggleSession">
          {{ session.connected.value ? '已连接' : '连接' }}
        </button>
      </template>
    </WorkbenchSourceHeader>
    <ModbusGroupTree ... />
  </aside>

  <main class="modbus-workbench__main">
    <div class="modbus-workbench__bar">
      <div class="modbus-workbench__title">...</div>
      <div class="modbus-workbench__actions">
        <button :title="importActionTitle" @click="importVisible = true"><IconTablerUpload /></button>
        <button @click="openCreateRegister"><IconTablerPlus /></button>
        <button :disabled="!session.connected.value" title="预览当前分组变量" @click="openPreview"><IconTablerActivityHeartbeat /></button>
        <button :title="validationActionTitle" @click="runValidation"><IconTablerChecklist /></button>
        <button title="当前分组读取计划" @click="openReadPlan"><IconTablerRoute /></button>
        <button @click="reloadAll"><IconTablerRefresh /></button>
      </div>
      <el-input ... />
    </div>
    <ModbusRegisterTable ... />
  </main>

  <ModbusInspectorPanel ... />
</section>
```

- [ ] **步骤 2：接入 Modbus 会话状态机**

加入：

```ts
const session = useProtocolDevSession({
  create: () => dataAPI.createModbusDevSession(props.projectId, props.connection.id),
  close: (sessionId: string) =>
    dataAPI.closeModbusDevSession(props.projectId, props.connection.id, sessionId),
})

const toggleSession = () => {
  if (session.connected.value) void session.disconnect()
  else void session.connect()
}
```

移除 `testing` 与 `runConnectionTest`。

- [ ] **步骤 3：让预览读取当前分组会话值**

改 `openPreview`：

```ts
const openPreview = async () => {
  if (!session.connected.value || !session.sessionId.value) {
    ElMessage.warning('请先连接 Modbus 开发态会话')
    return
  }
  try {
    const response = await dataAPI.pollModbusDevSession(
      props.projectId,
      props.connection.id,
      session.sessionId.value,
      {
        groupId: selectedGroupId.value || null,
      },
    )
    const data = unwrapData(response)
    previewRegisters.value = data.values || []
    previewDiagnostics.value = data.diagnostics || []
    previewVisible.value = true
  } catch (error) {
    ElMessage.error(getApiErrorMessage(error, '变量预览失败'))
  }
}
```

`ModbusPreviewDialog.vue` 表格列改为：`registerId / slaveId / area / address / rawValue / value / dataType / timestamp / error`。

- [ ] **步骤 4：按当前上下文过滤校验问题**

增加：

```ts
const scopedValidationIssues = computed(() => {
  if (selectedRegisterId.value)
    return validationIssues.value.filter((item) => item.registerId === selectedRegisterId.value)
  if (selectedGroupId.value) {
    const ids = new Set(
      registers.value
        .filter((item) => item.groupId === selectedGroupId.value)
        .map((item) => item.id),
    )
    return validationIssues.value.filter((item) => item.registerId && ids.has(item.registerId))
  }
  return validationIssues.value
})
```

传给 `ModbusValidationDrawer`，并增加 `scope-label`。

- [ ] **步骤 5：更新样式**

把 `.modbus-workbench` 改为三栏 grid：

```css
.modbus-workbench {
  height: 100%;
  min-height: 0;
  display: grid;
  grid-template-columns: 264px minmax(0, 1fr) 292px;
  border: 1px solid var(--dc-border);
  border-radius: var(--dc-radius-md);
  background: var(--dc-surface-raised);
  box-shadow: var(--dc-shadow-surface);
  overflow: hidden;
}

.modbus-workbench__side {
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--dc-border);
  overflow: hidden;
}

.modbus-workbench__side :deep(.modbus-groups) {
  flex: 1;
  min-height: 0;
}

.modbus-workbench__bar {
  grid-template-columns: minmax(0, 1fr) auto 230px;
}
```

- [ ] **步骤 6：运行前端验证**

运行：

```powershell
pnpm --filter datacenter typecheck
pnpm --filter datacenter build
```

预期：PASS。

- [ ] **步骤 7：Commit**

```powershell
git add datacenter/src/components/access-source/workbench/ModbusWorkbenchPanel.vue datacenter/src/components/modbus/ModbusPreviewDialog.vue datacenter/src/components/modbus/ModbusValidationDrawer.vue
git commit -m "feat(datacenter): 调整 Modbus 工作台开发态会话布局"
```

---

### 任务 8：全量验证与文档对齐

**文件：**

- 修改：`docs/superpowers/specs/2026-05-25-datacenter-protocol-dev-session-workbench-design.md`
- 修改：`docs/superpowers/plans/2026-05-25-datacenter-protocol-dev-session-workbench.md`

- [ ] **步骤 1：运行后端全量测试**

运行：

```powershell
pnpm go:test:data
```

预期：PASS。

- [ ] **步骤 2：运行前端类型检查和构建**

运行：

```powershell
pnpm --filter datacenter typecheck
pnpm --filter datacenter build
```

预期：PASS。

- [ ] **步骤 3：检查文档和实现接口一致性**

核对接口列表：

```text
POST   /api/v1/data/projects/{projectId}/opcua/{connectionId}/sessions
DELETE /api/v1/data/projects/{projectId}/opcua/{connectionId}/sessions/{sessionId}
GET    /api/v1/data/projects/{projectId}/opcua/{connectionId}/sessions/{sessionId}/browse
POST   /api/v1/data/projects/{projectId}/opcua/{connectionId}/sessions/{sessionId}/read
POST   /api/v1/data/projects/{projectId}/opcua/{connectionId}/sessions/{sessionId}/subscribe
DELETE /api/v1/data/projects/{projectId}/opcua/{connectionId}/sessions/{sessionId}/subscribe
POST   /api/v1/data/projects/{projectId}/modbus/{connectionId}/sessions
DELETE /api/v1/data/projects/{projectId}/modbus/{connectionId}/sessions/{sessionId}
POST   /api/v1/data/projects/{projectId}/modbus/{connectionId}/sessions/{sessionId}/read
POST   /api/v1/data/projects/{projectId}/modbus/{connectionId}/sessions/{sessionId}/poll
DELETE /api/v1/data/projects/{projectId}/modbus/{connectionId}/sessions/{sessionId}/poll
```

如果实现接口名称或响应字段和规格不同，优先改代码保持规格；只有代码发现规格确实不合理时才修改规格。

- [ ] **步骤 4：检查 git 状态**

运行：

```powershell
git status --short
```

预期：只显示本任务相关文件，且没有构建产物、临时日志或环境文件。

- [ ] **步骤 5：Commit**

```powershell
git add docs/superpowers/specs/2026-05-25-datacenter-protocol-dev-session-workbench-design.md docs/superpowers/plans/2026-05-25-datacenter-protocol-dev-session-workbench.md
git commit -m "docs(datacenter): 对齐协议开发态会话实施说明"
```

---

## 自检

规格覆盖度：

- 工作台布局左中右：任务 6、任务 7。
- OPC UA 连接 / 断开、浏览、读取、订阅：任务 1、任务 2、任务 4、任务 5、任务 6。
- Modbus 连接 / 断开、读取、轮询、读取计划：任务 1、任务 3、任务 4、任务 5、任务 7。
- 当前分组作用域：任务 6、任务 7。
- 会话生命周期和卸载清理：任务 1、任务 5。
- 不混入运行态长期采集：任务 1、任务 2、任务 3 的模拟适配器边界与任务 8 的文档对齐。

占位符扫描：

- 本计划不包含未落实的占位表达。
- 第一版适配器明确使用建模数据生成可测开发态读数；不是空实现。

类型一致性：

- 后端统一使用 `sessionId`、`status`、`diagnostics`。
- OPC UA 读取值统一使用 `nodeId/value/dataType/quality/sourceTimestamp/serverTimestamp/error`。
- Modbus 读取值统一使用 `registerId/slaveId/area/address/rawValue/value/dataType/timestamp/error`。
