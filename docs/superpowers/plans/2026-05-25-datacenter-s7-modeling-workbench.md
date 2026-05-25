# 数据中心 Siemens S7 建模工作台实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 Siemens S7 接入源工作台完善为面向平台二开人员的 PLC 地址表建模入口，支持 PLC 类型确认、变量组、变量地址表、数据点自动同步、建模校验、开发态读取预览、读取计划估算和运行态契约输出。

**架构：** `data_service` 新增 S7 PLC 档案、变量组和变量建模表，服务层负责 PLC 类型默认能力、S7 地址解析、变量 CRUD、导入、数据点同步、校验、读取计划估算和开发态样例读取。`datacenter` 新增 S7 专用三栏工作台，与当前 Modbus 工作台保持家族化布局：左侧接入源 Header 与变量组，中间变量表和工具栏，右侧 Inspector。

**技术栈：** Go、pgx、PostgreSQL migration、Vue 3、Element Plus、TypeScript、Vite、Go test、pnpm。

---

## 文件结构

后端新增和修改：

- 创建：`data_service/internal/db/migrations/0029_s7_variable_modeling.sql`
  新增 `data_s7_plc_profiles`、`data_s7_variable_groups`、`data_s7_variables`。
- 创建：`data_service/internal/db/migrations/0029_s7_variable_modeling_down.sql`
  回滚 S7 建模表和索引。
- 创建：`data_service/internal/repository/s7_modeling_repository.go`
  参数化实现 PLC 档案、变量组、变量、数据点联查和快照读取所需 SQL。
- 创建：`data_service/internal/service/s7_modeling_service.go`
  封装 PLC 类型默认能力、地址解析、读取长度推导、数据点同步、建模校验、读取计划估算和开发态样例值。
- 创建：`data_service/internal/http/handler/s7_modeling_handler.go`
  暴露 PLC 档案、变量组、变量、导入、校验、预览和读取预估接口。
- 修改：`data_service/internal/http/router/router.go`
  挂载 `/api/v1/data/projects/{projectId}/s7/{connectionId}` 建模路由和 S7 开发态会话路由。
- 修改：`data_service/internal/app/server.go`
  初始化 S7 建模 repository、service、handler，并把 S7 适配器接入协议开发态会话服务。
- 修改：`data_service/internal/service/protocol_dev_session_service.go`
  增加 S7 开发态会话读取和短时轮询能力，沿用现有会话状态机。
- 修改：`data_service/internal/http/handler/protocol_dev_session_handler.go`
  增加 S7 创建/关闭/读取/轮询 HTTP handler。
- 修改：`data_service/internal/repository/project_snapshot_repository.go`
  将 S7 PLC 档案、变量和读取计划纳入运行契约 artifact。

前端新增和修改：

- 修改：`datacenter/src/api/data.api.ts`
  增加 S7 建模 API 与开发态会话 API。
- 创建：`datacenter/src/components/s7/types.ts`
  S7 工作台共享类型。
- 创建：`datacenter/src/components/s7/S7GroupTree.vue`
  左侧变量组树。
- 创建：`datacenter/src/components/s7/S7VariableTable.vue`
  中间变量表格。
- 创建：`datacenter/src/components/s7/S7InspectorPanel.vue`
  右侧配置面板。
- 创建：`datacenter/src/components/s7/S7ProfileDialog.vue`
  PLC 类型 / PLC 档案弹窗。
- 创建：`datacenter/src/components/s7/S7GroupDialog.vue`
  新建 / 编辑变量组弹窗。
- 创建：`datacenter/src/components/s7/S7VariableDialog.vue`
  新建 / 编辑 S7 变量弹窗。
- 创建：`datacenter/src/components/s7/S7ImportDialog.vue`
  地址表导入弹窗。
- 创建：`datacenter/src/components/s7/S7PreviewDialog.vue`
  开发态读取和短时预览弹窗。
- 创建：`datacenter/src/components/s7/S7ValidationDrawer.vue`
  建模校验抽屉。
- 创建：`datacenter/src/components/s7/S7ReadPlanDialog.vue`
  读取计划估算弹窗。
- 创建：`datacenter/src/components/access-source/workbench/S7WorkbenchPanel.vue`
  S7 三栏工作台容器。
- 修改：`datacenter/src/components/access-source/AccessSourceWorkbench.vue`
  将 `type='s7'` 路由到 S7 工作台。

验证命令：

- 后端：`cd data_service; go test ./...`
- 前端类型：`pnpm --filter datacenter typecheck`
- 前端构建：`pnpm --filter datacenter build`

---

## 实现边界

- S7 工作台负责协议建模和开发态辅助读取，不启动正式长期采集。
- S7 接入源创建继续保持轻量，PLC 类型在工作台内通过 `S7ProfileDialog` 确认。
- 平台二开人员是一级用户，界面必须展示 PLC 类型、地址能力、读取能力和风险提示，不暴露底层实现接口。
- 变量保存后自动 upsert `data_points(source_type='s7.variable')`。
- 删除变量时将对应数据点标记为 invalid，不物理删除。
- 读取计划估算只解释运行态如何合并读取，不允许用户手动维护运行计划。
- 运行态 artifact 输出 `profile / variables / readPlans`，运行态不重新解析 `addressText`。

---

### 任务 1：新增 S7 建模数据库表

**文件：**

- 创建：`data_service/internal/db/migrations/0029_s7_variable_modeling.sql`
- 创建：`data_service/internal/db/migrations/0029_s7_variable_modeling_down.sql`

- [ ] **步骤 1：创建 S7 建模迁移**

新增 `data_s7_plc_profiles`：

```sql
CREATE TABLE IF NOT EXISTS data_s7_plc_profiles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    plc_family text NOT NULL DEFAULT 'S7 Compatible',
    communication_mode text NOT NULL DEFAULT 'rack_slot',
    host text NOT NULL CHECK (char_length(host) <= 255),
    port integer NOT NULL DEFAULT 102 CHECK (port > 0 AND port <= 65535),
    rack integer NOT NULL DEFAULT 0 CHECK (rack >= 0),
    slot integer NOT NULL DEFAULT 1 CHECK (slot >= 0),
    local_tsap text,
    remote_tsap text,
    poll_interval_ms integer NOT NULL DEFAULT 1000 CHECK (poll_interval_ms > 0),
    connect_timeout_ms integer NOT NULL DEFAULT 3000 CHECK (connect_timeout_ms > 0),
    read_timeout_ms integer NOT NULL DEFAULT 3000 CHECK (read_timeout_ms > 0),
    pdu_size integer CHECK (pdu_size IS NULL OR pdu_size > 0),
    max_read_bytes integer CHECK (max_read_bytes IS NULL OR max_read_bytes > 0),
    max_gap_bytes integer NOT NULL DEFAULT 8 CHECK (max_gap_bytes >= 0),
    max_concurrent_reads integer NOT NULL DEFAULT 1 CHECK (max_concurrent_reads > 0),
    byte_order text NOT NULL DEFAULT 'big_endian',
    word_order text NOT NULL DEFAULT 'big_endian',
    optimized_block_access boolean NOT NULL DEFAULT false,
    allow_absolute_address boolean NOT NULL DEFAULT true,
    allow_symbol_address boolean NOT NULL DEFAULT false,
    supported_areas jsonb NOT NULL DEFAULT '["DB","M","I","Q"]'::jsonb CHECK (jsonb_typeof(supported_areas) = 'array'),
    options jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(options) = 'object'),
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_s7_plc_profiles_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_s7_plc_profiles_connection_key
        UNIQUE (project_id, connection_id)
);
```

新增 `data_s7_variable_groups` 和 `data_s7_variables`，字段覆盖分组、结构化地址、解析规则、采集周期、最近值快照和状态：

```sql
CREATE TABLE IF NOT EXISTS data_s7_variable_groups (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    parent_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    code text NOT NULL CHECK (char_length(code) <= 100),
    description text,
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_s7_variable_groups_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_s7_variable_groups_identity_key
        UNIQUE (id, project_id, connection_id),
    CONSTRAINT data_s7_variable_groups_parent_connection_fkey
        FOREIGN KEY (parent_id, project_id, connection_id)
        REFERENCES data_s7_variable_groups (id, project_id, connection_id) ON DELETE CASCADE,
    CONSTRAINT data_s7_variable_groups_connection_code_key
        UNIQUE (project_id, connection_id, code)
);

CREATE TABLE IF NOT EXISTS data_s7_variables (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    code text NOT NULL CHECK (char_length(code) <= 100),
    description text,
    area text NOT NULL CHECK (area IN ('DB', 'M', 'I', 'Q', 'T', 'C')),
    db_number integer CHECK (db_number IS NULL OR db_number >= 0),
    byte_offset integer NOT NULL CHECK (byte_offset >= 0),
    bit_offset integer CHECK (bit_offset IS NULL OR (bit_offset >= 0 AND bit_offset <= 7)),
    address_text text NOT NULL CHECK (char_length(address_text) <= 120),
    normalized_address text NOT NULL CHECK (char_length(normalized_address) <= 120),
    address_type text NOT NULL CHECK (char_length(address_type) <= 20),
    read_length integer NOT NULL DEFAULT 1 CHECK (read_length > 0),
    data_type text NOT NULL CHECK (char_length(data_type) <= 50),
    length integer CHECK (length IS NULL OR length > 0),
    array_length integer CHECK (array_length IS NULL OR array_length > 0),
    byte_order text NOT NULL DEFAULT 'big_endian',
    word_order text NOT NULL DEFAULT 'big_endian',
    scale numeric(20, 6) NOT NULL DEFAULT 1,
    offset_value numeric(20, 6) NOT NULL DEFAULT 0,
    unit text CHECK (unit IS NULL OR char_length(unit) <= 20),
    poll_interval_ms integer NOT NULL DEFAULT 1000 CHECK (poll_interval_ms > 0),
    quality_rule jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(quality_rule) = 'object'),
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(metadata) = 'object'),
    last_value jsonb,
    quality text NOT NULL DEFAULT 'unknown',
    last_updated_at timestamptz,
    sort_order integer NOT NULL DEFAULT 0,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'invalid')),
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_s7_variables_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_s7_variables_group_connection_fkey
        FOREIGN KEY (group_id, project_id, connection_id)
        REFERENCES data_s7_variable_groups (id, project_id, connection_id) ON DELETE SET NULL,
    CONSTRAINT data_s7_variables_connection_code_key
        UNIQUE (project_id, connection_id, code)
);
```

创建索引：

```sql
CREATE INDEX IF NOT EXISTS data_s7_variable_groups_connection_parent_idx
    ON data_s7_variable_groups (project_id, connection_id, parent_id, sort_order, created_at);

CREATE INDEX IF NOT EXISTS data_s7_variables_group_order_idx
    ON data_s7_variables (project_id, connection_id, group_id, sort_order, created_at DESC);

CREATE INDEX IF NOT EXISTS data_s7_variables_plan_idx
    ON data_s7_variables (project_id, connection_id, area, db_number, poll_interval_ms, byte_offset);

CREATE INDEX IF NOT EXISTS data_s7_variables_status_idx
    ON data_s7_variables (project_id, connection_id, status);
```

- [ ] **步骤 2：创建回滚迁移**

`0029_s7_variable_modeling_down.sql` 按索引、变量表、分组表、档案表顺序删除：

```sql
DROP INDEX IF EXISTS data_s7_variables_status_idx;
DROP INDEX IF EXISTS data_s7_variables_plan_idx;
DROP INDEX IF EXISTS data_s7_variables_group_order_idx;
DROP TABLE IF EXISTS data_s7_variables;
DROP INDEX IF EXISTS data_s7_variable_groups_connection_parent_idx;
DROP TABLE IF EXISTS data_s7_variable_groups;
DROP TABLE IF EXISTS data_s7_plc_profiles;
```

- [ ] **步骤 3：验证迁移文件语法进入编译路径**

运行：

```powershell
cd D:\SVNCode\indu-forge\data_service
go test ./internal/db/... ./internal/repository/... -count=1
```

预期：PASS。

- [ ] **步骤 4：提交数据库迁移**

```powershell
git add data_service/internal/db/migrations/0029_s7_variable_modeling.sql data_service/internal/db/migrations/0029_s7_variable_modeling_down.sql
git commit -m "feat(data_service): 新增 S7 变量建模表"
```

---

### 任务 2：实现 S7 repository

**文件：**

- 创建：`data_service/internal/repository/s7_modeling_repository.go`

- [ ] **步骤 1：定义 repository record 和参数类型**

创建文件并定义：

```go
type S7ProfileRecord struct {
	ID                    string
	ProjectID             string
	ConnectionID          string
	PlcFamily             string
	CommunicationMode     string
	Host                  string
	Port                  int
	Rack                  int
	Slot                  int
	LocalTSAP             *string
	RemoteTSAP            *string
	PollIntervalMS        int
	ConnectTimeoutMS      int
	ReadTimeoutMS         int
	PDUSize               *int
	MaxReadBytes          *int
	MaxGapBytes           int
	MaxConcurrentReads    int
	ByteOrder             string
	WordOrder             string
	OptimizedBlockAccess  bool
	AllowAbsoluteAddress  bool
	AllowSymbolAddress    bool
	SupportedAreas        []byte
	Options               []byte
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type S7VariableGroupRecord struct {
	ID           string
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	Code         string
	Description  *string
	SortOrder    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type S7VariableRecord struct {
	ID              string
	ProjectID       string
	ConnectionID    string
	GroupID         *string
	Name            string
	Code            string
	Description     *string
	Area            string
	DBNumber        *int
	ByteOffset      int
	BitOffset       *int
	AddressText     string
	NormalizedAddress string
	AddressType     string
	ReadLength      int
	DataType        string
	Length          *int
	ArrayLength     *int
	ByteOrder       string
	WordOrder       string
	Scale           float64
	Offset          float64
	Unit            *string
	PollIntervalMS  int
	QualityRule     []byte
	Metadata        []byte
	LastValue        []byte
	Quality         string
	LastUpdatedAt   *time.Time
	SortOrder       int
	Status          string
	DataPointID     *string
	DataPointPath   *string
	DataPointStatus *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
```

- [ ] **步骤 2：实现 PLC 档案读写**

实现方法：

```go
func NewS7ModelingRepository(pool *pgxpool.Pool) *S7ModelingRepository
func (r *S7ModelingRepository) GetProfile(ctx context.Context, projectID, connectionID string) (*S7ProfileRecord, error)
func (r *S7ModelingRepository) UpsertProfile(ctx context.Context, params UpsertS7ProfileParams) (*S7ProfileRecord, error)
```

`UpsertProfile` 使用 `INSERT ... ON CONFLICT (project_id, connection_id) DO UPDATE`，更新 PLC 类型、连接参数、读取能力、地址能力和 `updated_by`。

- [ ] **步骤 3：实现变量组 CRUD**

实现方法：

```go
func (r *S7ModelingRepository) ListGroups(ctx context.Context, projectID, connectionID string) ([]S7VariableGroupRecord, error)
func (r *S7ModelingRepository) CreateGroup(ctx context.Context, params CreateS7VariableGroupParams) (*S7VariableGroupRecord, error)
func (r *S7ModelingRepository) UpdateGroup(ctx context.Context, params UpdateS7VariableGroupParams) (*S7VariableGroupRecord, error)
func (r *S7ModelingRepository) DeleteGroup(ctx context.Context, projectID, connectionID, groupID string) error
```

删除组前先将变量 `group_id` 置空，再删除组，避免误删变量。

- [ ] **步骤 4：实现变量 CRUD 和数据点联查**

实现方法：

```go
func (r *S7ModelingRepository) ListVariables(ctx context.Context, projectID, connectionID string, groupID *string) ([]S7VariableRecord, error)
func (r *S7ModelingRepository) GetVariable(ctx context.Context, projectID, connectionID, variableID string) (*S7VariableRecord, error)
func (r *S7ModelingRepository) CreateVariable(ctx context.Context, params CreateS7VariableParams) (*S7VariableRecord, error)
func (r *S7ModelingRepository) UpdateVariable(ctx context.Context, params UpdateS7VariableParams) (*S7VariableRecord, error)
func (r *S7ModelingRepository) DeleteVariable(ctx context.Context, projectID, connectionID, variableID string) error
func (r *S7ModelingRepository) UpdateVariableLastValue(ctx context.Context, params UpdateS7VariableLastValueParams) error
```

列表查询左联 `data_points`：

```sql
LEFT JOIN data_points dp
  ON dp.project_id = v.project_id
 AND dp.source_type = 's7.variable'
 AND dp.source_id = v.id
```

- [ ] **步骤 5：验证 repository 编译**

运行：

```powershell
cd D:\SVNCode\indu-forge\data_service
go test ./internal/repository -count=1
```

预期：PASS。

- [ ] **步骤 6：提交 repository**

```powershell
git add data_service/internal/repository/s7_modeling_repository.go
git commit -m "feat(data_service): 增加 S7 建模仓储"
```

---

### 任务 3：实现 S7 service 核心规则

**文件：**

- 创建：`data_service/internal/service/s7_modeling_service.go`
- 可选新增测试：`data_service/internal/service/s7_modeling_service_test.go`

- [ ] **步骤 1：定义前端响应模型**

定义：

```go
type S7Profile struct {
	ID                   string         `json:"id"`
	ProjectID            string         `json:"projectId"`
	ConnectionID         string         `json:"connectionId"`
	PlcFamily            string         `json:"plcFamily"`
	CommunicationMode    string         `json:"communicationMode"`
	Host                 string         `json:"host"`
	Port                 int            `json:"port"`
	Rack                 int            `json:"rack"`
	Slot                 int            `json:"slot"`
	LocalTSAP            *string        `json:"localTsap"`
	RemoteTSAP           *string        `json:"remoteTsap"`
	PollIntervalMS       int            `json:"pollIntervalMs"`
	ConnectTimeoutMS     int            `json:"connectTimeoutMs"`
	ReadTimeoutMS        int            `json:"readTimeoutMs"`
	PDUSize              *int           `json:"pduSize"`
	MaxReadBytes         *int           `json:"maxReadBytes"`
	MaxGapBytes          int            `json:"maxGapBytes"`
	MaxConcurrentReads   int            `json:"maxConcurrentReads"`
	ByteOrder            string         `json:"byteOrder"`
	WordOrder            string         `json:"wordOrder"`
	OptimizedBlockAccess bool           `json:"optimizedBlockAccess"`
	AllowAbsoluteAddress bool           `json:"allowAbsoluteAddress"`
	AllowSymbolAddress   bool           `json:"allowSymbolAddress"`
	SupportedAreas       []string       `json:"supportedAreas"`
	Options              map[string]any `json:"options"`
}

type S7Variable struct {
	ID                string     `json:"id"`
	ProjectID         string     `json:"projectId"`
	ConnectionID      string     `json:"connectionId"`
	GroupID           *string    `json:"groupId"`
	Name              string     `json:"name"`
	Code              string     `json:"code"`
	Area              string     `json:"area"`
	DBNumber          *int       `json:"dbNumber"`
	ByteOffset        int        `json:"byteOffset"`
	BitOffset         *int       `json:"bitOffset"`
	AddressText       string     `json:"addressText"`
	NormalizedAddress string     `json:"normalizedAddress"`
	AddressType       string     `json:"addressType"`
	ReadLength        int        `json:"readLength"`
	DataType          string     `json:"dataType"`
	Length            *int       `json:"length"`
	ArrayLength       *int       `json:"arrayLength"`
	ByteOrder         string     `json:"byteOrder"`
	WordOrder         string     `json:"wordOrder"`
	Scale             float64    `json:"scale"`
	Offset            float64    `json:"offset"`
	Unit              *string    `json:"unit"`
	PollIntervalMS    int        `json:"pollIntervalMs"`
	LastValue         any        `json:"lastValue,omitempty"`
	Quality           string     `json:"quality"`
	LastUpdatedAt     *time.Time `json:"lastUpdatedAt,omitempty"`
	Status            string     `json:"status"`
	DataPointID       *string    `json:"datapointId"`
	DataPointPath     *string    `json:"datapointPath"`
	DataPointStatus   *string    `json:"datapointStatus"`
}
```

- [ ] **步骤 2：实现 PLC 类型默认能力**

实现：

```go
func defaultS7ProfileForFamily(family string, connection repository.ConnectionRecord) S7ProfileDefaults
```

规则：

```text
S7-200 / S7-200 SMART：communicationMode=tsap，maxGapBytes=8，maxConcurrentReads=1。
S7-300 / S7-400：communicationMode=rack_slot，rack=0，slot=2，maxGapBytes=8。
S7-1200 / S7-1500：communicationMode=rack_slot，rack=0，slot=1，maxGapBytes=16，optimizedBlockAccess 提示为 true。
S7 Compatible：communicationMode=rack_slot，rack=0，slot=1，maxGapBytes=8。
```

- [ ] **步骤 3：实现地址解析**

实现：

```go
func ParseS7Address(addressText string, dataType string) (S7ParsedAddress, error)
```

支持输入：

```text
DB1.DBX0.0
DB1.DBB1
DB1.DBW2
DB1.DBD4
M0.0
MB1
MW2
MD4
I0.0
Q0.0
```

输出包含：

```go
type S7ParsedAddress struct {
	Area              string
	DBNumber          *int
	ByteOffset        int
	BitOffset         *int
	AddressType       string
	NormalizedAddress string
}
```

- [ ] **步骤 4：实现数据类型读取长度推导**

实现：

```go
func s7ReadLengthForDataType(dataType string, length *int, arrayLength *int) (int, error)
```

规则：

```text
Bool=1 byte
Byte=1
Word/Int=2
DWord/DInt/Real=4
DateTime=8
String=2 + length
数组长度按单元素长度 * arrayLength
```

- [ ] **步骤 5：实现变量保存时同步数据点**

创建或更新变量后调用 `DataPointRepository.UpsertBySource`，source config 至少包含：

```go
map[string]any{
	"connectionId": variable.ConnectionID,
	"area": variable.Area,
	"dbNumber": variable.DBNumber,
	"byteOffset": variable.ByteOffset,
	"bitOffset": variable.BitOffset,
	"addressText": variable.AddressText,
	"normalizedAddress": variable.NormalizedAddress,
	"readLength": variable.ReadLength,
	"dataType": variable.DataType,
	"byteOrder": variable.ByteOrder,
	"wordOrder": variable.WordOrder,
	"scale": variable.Scale,
	"offset": variable.Offset,
	"pollIntervalMs": variable.PollIntervalMS,
}
```

数据点 path 使用：

```text
s7.<connectionCode>.<groupCode>.<variableCode>
```

- [ ] **步骤 6：实现建模校验**

校验输出类型：

```go
type S7ValidationIssue struct {
	Severity     string  `json:"severity"`
	Code         string  `json:"code"`
	GroupID      *string `json:"groupId,omitempty"`
	VariableID   *string `json:"variableId,omitempty"`
	VariableName string  `json:"variableName,omitempty"`
	Message      string  `json:"message"`
}
```

校验规则覆盖：

```text
变量名为空
code 为空或重复
分组不存在
地址格式非法
DB 区缺少 dbNumber
Bool 缺少 bitOffset
String 缺少 length
数据类型与地址类型不匹配
字节偏移小于 0
采集周期小于等于 0
数据点未生成
数据点已失效
地址范围重叠 warning
读取块碎片过多 warning
PLC 档案未配置 error
PLC 档案不支持变量地址区 error
S7-1200/1500 优化 DB 访问风险 warning
```

- [ ] **步骤 7：实现读取计划估算**

实现：

```go
func BuildS7ReadPlanEstimate(profile S7Profile, variables []S7Variable) S7ReadPlanEstimate
```

按 `connectionId + area + dbNumber + pollIntervalMs + readMode` 分组，按 `byteOffset` 合并，限制：

```text
readLength <= profile.maxReadBytes 或默认 240
gap <= profile.maxGapBytes
不同 DB 不合并
不同周期不合并
```

输出：

```go
type S7ReadPlanEstimate struct {
	VariableCount        int          `json:"variableCount"`
	BlockCount           int          `json:"blockCount"`
	TotalReadBytes       int          `json:"totalReadBytes"`
	ReadsPerSecond       float64      `json:"readsPerSecond"`
	EstimatedCycleMS     int          `json:"estimatedCycleMs"`
	LargestBlockBytes     int          `json:"largestBlockBytes"`
	FragmentedGroupCount  int          `json:"fragmentedGroupCount"`
	Plans                []S7ReadPlan `json:"plans"`
	Diagnostics          []string     `json:"diagnostics"`
}
```

- [ ] **步骤 8：验证 service**

运行：

```powershell
cd D:\SVNCode\indu-forge\data_service
go test ./internal/service -run S7 -count=1
go test ./... -count=1
```

预期：PASS。

- [ ] **步骤 9：提交 service**

```powershell
git add data_service/internal/service/s7_modeling_service.go data_service/internal/service/s7_modeling_service_test.go
git commit -m "feat(data_service): 实现 S7 建模业务规则"
```

---

### 任务 4：实现 S7 HTTP handler 和路由

**文件：**

- 创建：`data_service/internal/http/handler/s7_modeling_handler.go`
- 修改：`data_service/internal/http/router/router.go`
- 修改：`data_service/internal/app/server.go`

- [ ] **步骤 1：实现 handler 请求结构**

请求结构包括：

```go
type s7ProfileRequest struct {
	PlcFamily            string         `json:"plcFamily"`
	CommunicationMode    string         `json:"communicationMode"`
	Host                 string         `json:"host"`
	Port                 *int           `json:"port"`
	Rack                 *int           `json:"rack"`
	Slot                 *int           `json:"slot"`
	LocalTSAP            *string        `json:"localTsap"`
	RemoteTSAP           *string        `json:"remoteTsap"`
	PollIntervalMS       *int           `json:"pollIntervalMs"`
	ConnectTimeoutMS     *int           `json:"connectTimeoutMs"`
	ReadTimeoutMS        *int           `json:"readTimeoutMs"`
	PDUSize              *int           `json:"pduSize"`
	MaxReadBytes         *int           `json:"maxReadBytes"`
	MaxGapBytes          *int           `json:"maxGapBytes"`
	MaxConcurrentReads   *int           `json:"maxConcurrentReads"`
	OptimizedBlockAccess bool           `json:"optimizedBlockAccess"`
	AllowAbsoluteAddress bool           `json:"allowAbsoluteAddress"`
	AllowSymbolAddress   bool           `json:"allowSymbolAddress"`
	SupportedAreas       []string       `json:"supportedAreas"`
	Options              map[string]any `json:"options"`
}
```

变量请求结构包括 `addressText`，后端负责解析为结构化字段。

- [ ] **步骤 2：实现 handler 方法**

方法清单：

```go
func (h *S7ModelingHandler) GetProfile(w http.ResponseWriter, r *http.Request) error
func (h *S7ModelingHandler) UpsertProfile(w http.ResponseWriter, r *http.Request) error
func (h *S7ModelingHandler) ListGroups(w http.ResponseWriter, r *http.Request) error
func (h *S7ModelingHandler) CreateGroup(w http.ResponseWriter, r *http.Request) error
func (h *S7ModelingHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) error
func (h *S7ModelingHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) error
func (h *S7ModelingHandler) ListVariables(w http.ResponseWriter, r *http.Request) error
func (h *S7ModelingHandler) CreateVariable(w http.ResponseWriter, r *http.Request) error
func (h *S7ModelingHandler) BatchImportVariables(w http.ResponseWriter, r *http.Request) error
func (h *S7ModelingHandler) UpdateVariable(w http.ResponseWriter, r *http.Request) error
func (h *S7ModelingHandler) DeleteVariable(w http.ResponseWriter, r *http.Request) error
func (h *S7ModelingHandler) ValidateModel(w http.ResponseWriter, r *http.Request) error
func (h *S7ModelingHandler) PreviewVariables(w http.ResponseWriter, r *http.Request) error
func (h *S7ModelingHandler) EstimateReadPlans(w http.ResponseWriter, r *http.Request) error
```

- [ ] **步骤 3：挂载路由**

在 `router.options` 增加：

```go
s7ModelingHandler *handler.S7ModelingHandler
```

增加：

```go
func WithS7ModelingRoutes(s7ModelingHandler *handler.S7ModelingHandler, jwtValidator *auth.JWTValidator) Option
```

挂载：

```go
base := "/api/v1/data/projects/{projectId}/s7/{connectionId}"
mux.Handle("GET "+base+"/profile", read(opts.s7ModelingHandler.GetProfile))
mux.Handle("PUT "+base+"/profile", write(opts.s7ModelingHandler.UpsertProfile))
mux.Handle("GET "+base+"/variable-groups", read(opts.s7ModelingHandler.ListGroups))
mux.Handle("POST "+base+"/variable-groups", write(opts.s7ModelingHandler.CreateGroup))
mux.Handle("PUT "+base+"/variable-groups/{groupId}", write(opts.s7ModelingHandler.UpdateGroup))
mux.Handle("DELETE "+base+"/variable-groups/{groupId}", write(opts.s7ModelingHandler.DeleteGroup))
mux.Handle("GET "+base+"/variables", read(opts.s7ModelingHandler.ListVariables))
mux.Handle("POST "+base+"/variables", write(opts.s7ModelingHandler.CreateVariable))
mux.Handle("POST "+base+"/variables/batch-import", write(opts.s7ModelingHandler.BatchImportVariables))
mux.Handle("PUT "+base+"/variables/{variableId}", write(opts.s7ModelingHandler.UpdateVariable))
mux.Handle("DELETE "+base+"/variables/{variableId}", write(opts.s7ModelingHandler.DeleteVariable))
mux.Handle("POST "+base+"/validate-model", read(opts.s7ModelingHandler.ValidateModel))
mux.Handle("POST "+base+"/preview", read(opts.s7ModelingHandler.PreviewVariables))
mux.Handle("GET "+base+"/read-plan-estimate", read(opts.s7ModelingHandler.EstimateReadPlans))
```

- [ ] **步骤 4：初始化 app 依赖**

在 `defaultRouteDependenciesFactory` 中创建：

```go
s7ModelingRepository := repository.NewS7ModelingRepository(pool)
s7ModelingService := service.NewS7ModelingService(s7ModelingRepository, connectionRepository, dataPointRepository)
s7ModelingHandler := handler.NewS7ModelingHandler(s7ModelingService)
```

把 `router.WithS7ModelingRoutes(s7ModelingHandler, jwtValidator)` 加入 route options。

- [ ] **步骤 5：验证路由编译**

运行：

```powershell
cd D:\SVNCode\indu-forge\data_service
go test ./internal/http/... ./internal/app -count=1
```

预期：PASS。

- [ ] **步骤 6：提交 handler 和路由**

```powershell
git add data_service/internal/http/handler/s7_modeling_handler.go data_service/internal/http/router/router.go data_service/internal/app/server.go
git commit -m "feat(data_service): 暴露 S7 建模接口"
```

---

### 任务 5：接入 S7 开发态会话

**文件：**

- 修改：`data_service/internal/service/protocol_dev_session_service.go`
- 修改：`data_service/internal/http/handler/protocol_dev_session_handler.go`
- 修改：`data_service/internal/http/router/router.go`
- 修改：`data_service/internal/app/server.go`
- 修改：`data_service/internal/service/protocol_dev_session_service_test.go`

- [ ] **步骤 1：定义 S7 开发态模型读取接口**

在 `protocol_dev_session_service.go` 增加：

```go
type ProtocolDevS7Variable struct {
	ID                string
	Code              string
	Name              string
	GroupID           *string
	Area              string
	DBNumber          *int
	ByteOffset        int
	BitOffset         *int
	AddressText       string
	NormalizedAddress string
	ReadLength        int
	DataType          string
	Scale             float64
	Offset            float64
	Unit              *string
}

type ProtocolDevS7ModelReader interface {
	ListDevSessionS7Variables(ctx context.Context, projectID, connectionID string, groupID *string) ([]ProtocolDevS7Variable, error)
	EstimateDevSessionS7ReadPlan(ctx context.Context, projectID, connectionID string, groupID *string) (*S7ReadPlanEstimate, error)
}
```

- [ ] **步骤 2：扩展会话服务构造函数**

将构造函数改为接收 S7 reader：

```go
func NewProtocolDevSessionService(connections ProtocolDevConnectionReader, opcua ProtocolDevOpcuaModelReader, modbus ProtocolDevModbusModelReader, s7 ProtocolDevS7ModelReader) *ProtocolDevSessionService
```

同步调整 app 初始化和已有测试。

- [ ] **步骤 3：实现 S7 读取与轮询**

新增：

```go
func (s *ProtocolDevSessionService) ReadS7(ctx context.Context, projectID, connectionID, sessionID, userID string, variableIDs []string, groupID *string) (*ProtocolDevS7ReadResult, error)
func (s *ProtocolDevSessionService) PollS7(ctx context.Context, projectID, connectionID, sessionID, userID string, groupID *string) (*ProtocolDevS7PollResult, error)
func (s *ProtocolDevSessionService) StopPollS7(ctx context.Context, projectID, connectionID, sessionID, userID string) (*ProtocolDevSession, error)
```

读取返回：

```go
type ProtocolDevS7ReadValue struct {
	VariableID        string `json:"variableId"`
	Address           string `json:"address"`
	RawValue          any    `json:"rawValue"`
	Value             any    `json:"value"`
	Quality           string `json:"quality"`
	Timestamp         string `json:"timestamp"`
	Error             *string `json:"error,omitempty"`
}
```

样例值用 `sampleProtocolValue(dataType, code)` 生成，并在读取后调用 S7 service 更新 `last_value / quality / last_updated_at`。

- [ ] **步骤 4：增加 S7 handler 和路由**

在 `protocol_dev_session_handler.go` 增加：

```go
func (h *ProtocolDevSessionHandler) CreateS7(w http.ResponseWriter, r *http.Request) error
func (h *ProtocolDevSessionHandler) CloseS7(w http.ResponseWriter, r *http.Request) error
func (h *ProtocolDevSessionHandler) ReadS7(w http.ResponseWriter, r *http.Request) error
func (h *ProtocolDevSessionHandler) PollS7(w http.ResponseWriter, r *http.Request) error
func (h *ProtocolDevSessionHandler) StopPollS7(w http.ResponseWriter, r *http.Request) error
```

路由：

```go
s7Base := "/api/v1/data/projects/{projectId}/s7/{connectionId}/sessions"
mux.Handle("POST "+s7Base, read(opts.protocolDevSessionHandler.CreateS7))
mux.Handle("DELETE "+s7Base+"/{sessionId}", write(opts.protocolDevSessionHandler.CloseS7))
mux.Handle("POST "+s7Base+"/{sessionId}/read", read(opts.protocolDevSessionHandler.ReadS7))
mux.Handle("POST "+s7Base+"/{sessionId}/poll", read(opts.protocolDevSessionHandler.PollS7))
mux.Handle("DELETE "+s7Base+"/{sessionId}/poll", write(opts.protocolDevSessionHandler.StopPollS7))
```

- [ ] **步骤 5：验证开发态会话**

运行：

```powershell
cd D:\SVNCode\indu-forge\data_service
go test ./internal/service -run ProtocolDevSession -count=1
go test ./... -count=1
```

预期：PASS。

- [ ] **步骤 6：提交开发态会话**

```powershell
git add data_service/internal/service/protocol_dev_session_service.go data_service/internal/service/protocol_dev_session_service_test.go data_service/internal/http/handler/protocol_dev_session_handler.go data_service/internal/http/router/router.go data_service/internal/app/server.go
git commit -m "feat(data_service): 增加 S7 开发态会话"
```

---

### 任务 6：补充运行契约 artifact

**文件：**

- 修改：`data_service/internal/repository/project_snapshot_repository.go`

- [ ] **步骤 1：定义 S7 快照结构**

新增：

```go
type SnapshotS7ProfileRecord struct {
	ConnectionID          string         `json:"connectionId"`
	PlcFamily            string         `json:"plcFamily"`
	CommunicationMode    string         `json:"communicationMode"`
	Host                 string         `json:"host"`
	Port                 int            `json:"port"`
	Rack                 int            `json:"rack"`
	Slot                 int            `json:"slot"`
	PollIntervalMS       int            `json:"pollIntervalMs"`
	PDUSize              *int           `json:"pduSize,omitempty"`
	MaxReadBytes         *int           `json:"maxReadBytes,omitempty"`
	MaxGapBytes          int            `json:"maxGapBytes"`
	OptimizedBlockAccess bool           `json:"optimizedBlockAccess"`
	SupportedAreas       []string       `json:"supportedAreas"`
	Options              map[string]any `json:"options"`
}

type SnapshotS7VariableRecord struct {
	ID                string  `json:"id"`
	ConnectionID      string  `json:"connectionId"`
	GroupID           *string `json:"groupId,omitempty"`
	Name              string  `json:"name"`
	Code              string  `json:"code"`
	AddressText       string  `json:"addressText"`
	NormalizedAddress string  `json:"normalizedAddress"`
	Area              string  `json:"area"`
	DBNumber          *int    `json:"dbNumber,omitempty"`
	ByteOffset        int     `json:"byteOffset"`
	BitOffset         *int    `json:"bitOffset,omitempty"`
	ReadLength        int     `json:"readLength"`
	DataType          string  `json:"dataType"`
	ByteOrder         string  `json:"byteOrder"`
	WordOrder         string  `json:"wordOrder"`
	Scale             float64 `json:"scale"`
	Offset            float64 `json:"offset"`
	Unit              *string `json:"unit,omitempty"`
	PollIntervalMS    int     `json:"pollIntervalMs"`
	DataPointPath     *string `json:"datapointPath,omitempty"`
}

type SnapshotS7ReadPlanRecord struct {
	ID             string   `json:"id"`
	ConnectionID   string   `json:"connectionId"`
	Area           string   `json:"area"`
	DBNumber       *int     `json:"dbNumber,omitempty"`
	StartByte      int      `json:"startByte"`
	EndByte        int      `json:"endByte"`
	ReadLength     int      `json:"readLength"`
	PollIntervalMS int      `json:"pollIntervalMs"`
	VariableIDs    []string `json:"variableIds"`
	VariableCount  int      `json:"variableCount"`
	MaxGapBytes    int      `json:"maxGapBytes"`
	ReadMode       string   `json:"readMode"`
}
```

- [ ] **步骤 2：扩展 ProjectSnapshot 和 ArtifactProtocolRecord**

增加字段：

```go
S7Profiles  []SnapshotS7ProfileRecord  `json:"s7Profiles"`
S7Variables []SnapshotS7VariableRecord `json:"s7Variables"`
```

`ArtifactProtocolRecord` 增加：

```go
Profile   *SnapshotS7ProfileRecord  `json:"profile,omitempty"`
Variables []SnapshotS7VariableRecord `json:"variables,omitempty"`
ReadPlans []SnapshotS7ReadPlanRecord `json:"readPlans,omitempty"`
```

- [ ] **步骤 3：实现快照查询与读取计划构建**

实现：

```go
func (r *ProjectSnapshotRepository) listS7Profiles(ctx context.Context, projectID string) ([]SnapshotS7ProfileRecord, error)
func (r *ProjectSnapshotRepository) listS7Variables(ctx context.Context, projectID string) ([]SnapshotS7VariableRecord, error)
func buildSnapshotS7ReadPlansByConnection(profiles []SnapshotS7ProfileRecord, variables []SnapshotS7VariableRecord) map[string][]SnapshotS7ReadPlanRecord
```

读取计划合并规则与 service 一致。

- [ ] **步骤 4：挂入 artifact**

在 `BuildProjectArtifactV1` 的 `case "s7"` 中输出：

```go
profile := s7ProfilesByConnection[connection.ID]
protocolRecord.Profile = profile
protocolRecord.Variables = append([]SnapshotS7VariableRecord{}, s7VariablesByConnection[connection.ID]...)
protocolRecord.ReadPlans = append([]SnapshotS7ReadPlanRecord{}, s7ReadPlansByConnection[connection.ID]...)
protocols.S7 = append(protocols.S7, protocolRecord)
```

- [ ] **步骤 5：验证 artifact 编译**

运行：

```powershell
cd D:\SVNCode\indu-forge\data_service
go test ./internal/repository ./internal/service ./tests/integration -count=1
```

预期：PASS。

- [ ] **步骤 6：提交 artifact**

```powershell
git add data_service/internal/repository/project_snapshot_repository.go
git commit -m "feat(data_service): 输出 S7 运行契约"
```

---

### 任务 7：实现前端 API 与类型

**文件：**

- 修改：`datacenter/src/api/data.api.ts`
- 创建：`datacenter/src/components/s7/types.ts`

- [ ] **步骤 1：定义 S7 前端类型**

`types.ts` 定义：

```ts
export type S7Profile = {
  id?: string
  projectId?: string
  connectionId?: string
  plcFamily: string
  communicationMode: string
  host: string
  port: number
  rack: number
  slot: number
  localTsap?: string
  remoteTsap?: string
  pollIntervalMs: number
  connectTimeoutMs: number
  readTimeoutMs: number
  pduSize?: number | null
  maxReadBytes?: number | null
  maxGapBytes: number
  maxConcurrentReads: number
  optimizedBlockAccess: boolean
  allowAbsoluteAddress: boolean
  allowSymbolAddress: boolean
  supportedAreas: string[]
  options?: Record<string, any>
}

export type S7VariableGroup = {
  id: string
  parentId?: string | null
  name: string
  code: string
  description?: string | null
  sortOrder: number
}

export type S7Variable = {
  id: string
  groupId?: string | null
  name: string
  code: string
  area: string
  dbNumber?: number | null
  byteOffset: number
  bitOffset?: number | null
  addressText: string
  normalizedAddress: string
  addressType: string
  readLength: number
  dataType: string
  length?: number | null
  arrayLength?: number | null
  byteOrder: string
  wordOrder: string
  scale: number
  offset: number
  unit?: string | null
  pollIntervalMs: number
  datapointPath?: string | null
  datapointStatus?: string | null
  lastValue?: any
  quality?: string
  lastUpdatedAt?: string
  status: string
}
```

- [ ] **步骤 2：新增 API 方法**

在 `data.api.ts` 添加：

```ts
export const getS7Profile = (projectId, connectionId) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/profile`, method: 'get' })
export const updateS7Profile = (projectId, connectionId, data) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/profile`, method: 'put', data })
export const getS7VariableGroups = (projectId, connectionId) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/variable-groups`, method: 'get' }).then(normalizeListResponse)
export const createS7VariableGroup = (projectId, connectionId, data) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/variable-groups`, method: 'post', data })
export const updateS7VariableGroup = (projectId, connectionId, groupId, data) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/variable-groups/${groupId}`, method: 'put', data })
export const deleteS7VariableGroup = (projectId, connectionId, groupId) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/variable-groups/${groupId}`, method: 'delete' })
export const getS7Variables = (projectId, connectionId, params = {}) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/variables`, method: 'get', params }).then(normalizeListResponse)
export const createS7Variable = (projectId, connectionId, data) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/variables`, method: 'post', data })
export const batchImportS7Variables = (projectId, connectionId, data) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/variables/batch-import`, method: 'post', data })
export const updateS7Variable = (projectId, connectionId, variableId, data) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/variables/${variableId}`, method: 'put', data })
export const deleteS7Variable = (projectId, connectionId, variableId) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/variables/${variableId}`, method: 'delete' })
export const validateS7Model = (projectId, connectionId) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/validate-model`, method: 'post' })
export const previewS7Variables = (projectId, connectionId, data = {}) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/preview`, method: 'post', data })
export const getS7ReadPlanEstimate = (projectId, connectionId, params = {}) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/read-plan-estimate`, method: 'get', params })
```

增加开发态会话 API：

```ts
export const createS7DevSession = (projectId, connectionId) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/sessions`, method: 'post' })
export const closeS7DevSession = (projectId, connectionId, sessionId) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/sessions/${sessionId}`, method: 'delete' })
export const readS7DevSession = (projectId, connectionId, sessionId, data = {}) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/sessions/${sessionId}/read`, method: 'post', data })
export const pollS7DevSession = (projectId, connectionId, sessionId, data = {}) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/sessions/${sessionId}/poll`, method: 'post', data })
export const stopS7DevSessionPoll = (projectId, connectionId, sessionId) => request({ url: `/data/projects/${projectId}/s7/${connectionId}/sessions/${sessionId}/poll`, method: 'delete' })
```

- [ ] **步骤 3：验证前端类型**

运行：

```powershell
cd D:\SVNCode\indu-forge
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 4：提交 API 和类型**

```powershell
git add datacenter/src/api/data.api.ts datacenter/src/components/s7/types.ts
git commit -m "feat(datacenter): 增加 S7 工作台 API"
```

---

### 任务 8：实现 S7 三栏工作台容器

**文件：**

- 创建：`datacenter/src/components/access-source/workbench/S7WorkbenchPanel.vue`
- 修改：`datacenter/src/components/access-source/AccessSourceWorkbench.vue`

- [ ] **步骤 1：创建工作台容器骨架**

布局必须与当前 Modbus 保持一致：

```vue
<template>
  <section class="s7-workbench">
    <aside class="s7-workbench__side">
      <WorkbenchSourceHeader
        :title="connection.name || '未命名 S7 接入源'"
        fallback-title="未命名 S7 接入源"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #status>
          <button type="button" class="s7-workbench__connect-action" :class="{ 'is-connected': session.connected.value }" @click="toggleSession">
            <span>{{ session.connected.value ? '已连接' : '连接' }}</span>
          </button>
        </template>
      </WorkbenchSourceHeader>
      <S7GroupTree :groups="groups" :variables="variables" :selected-group-id="selectedGroupId" />
    </aside>

    <main class="s7-workbench__main">
      <div class="s7-workbench__bar">
        <div class="s7-workbench__title">
          <strong>{{ currentGroup?.name || '全部变量' }}</strong>
          <span>{{ filteredVariables.length }} 个变量 · 自动同步 s7.variable 数据点</span>
        </div>
      </div>
      <S7VariableTable :variables="filteredVariables" />
    </main>

    <S7InspectorPanel :profile="profile" :group="currentGroup" :variable="selectedVariable" :variables="variables" :estimate="readPlanEstimate" :issues="scopedValidationIssues" />
  </section>
</template>
```

- [ ] **步骤 2：接入加载状态**

`onMounted` 调用：

```ts
await Promise.all([loadProfile(), loadGroups(), loadVariables(), loadReadPlanEstimate()])
```

首次 `getS7Profile` 返回空或 `plcFamily` 缺失时，打开 `S7ProfileDialog`。

- [ ] **步骤 3：接入工具栏动作**

工具栏包含图标按钮：

```text
配置 PLC 类型
导入变量
新建变量
读取当前值
变量预览
建模校验
读取计划
刷新
```

读取和预览按钮在未连接时 disabled，并用 title 提示“连接后可读取当前分组变量”。

- [ ] **步骤 4：接入 AccessSourceWorkbench**

修改：

```ts
import S7WorkbenchPanel from './workbench/S7WorkbenchPanel.vue'

if (type === 's7') {
  return S7WorkbenchPanel
}
```

- [ ] **步骤 5：验证**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 6：提交容器**

```powershell
git add datacenter/src/components/access-source/workbench/S7WorkbenchPanel.vue datacenter/src/components/access-source/AccessSourceWorkbench.vue
git commit -m "feat(datacenter): 增加 S7 三栏工作台"
```

---

### 任务 9：实现 S7 工作台基础组件

**文件：**

- 创建：`datacenter/src/components/s7/S7GroupTree.vue`
- 创建：`datacenter/src/components/s7/S7VariableTable.vue`
- 创建：`datacenter/src/components/s7/S7InspectorPanel.vue`

- [ ] **步骤 1：实现变量组树**

`S7GroupTree.vue` 参考 `ModbusGroupTree.vue`，文案改为“变量组”，事件：

```ts
defineEmits<{
  (event: 'select', groupId: string): void
  (event: 'create'): void
  (event: 'edit', group: S7VariableGroup): void
  (event: 'delete', group: S7VariableGroup): void
}>()
```

左侧固定包含“全部变量”，每个组显示变量数量。

- [ ] **步骤 2：实现变量表**

列：

```text
变量名 | 地址 | 类型 | 字节范围 | 数据点 | 最近值 | 质量 | 周期 | 状态 | 操作
```

地址列显示：

```vue
<strong>{{ row.normalizedAddress || row.addressText }}</strong>
<span>{{ row.area }} {{ formatByteRange(row) }}</span>
```

字节范围函数：

```ts
const formatByteRange = (row: S7Variable) => {
  const end = row.byteOffset + row.readLength - 1
  const db = row.area === 'DB' && row.dbNumber !== null && row.dbNumber !== undefined ? `DB${row.dbNumber} ` : ''
  return `${db}${row.byteOffset}-${end}`
}
```

- [ ] **步骤 3：实现 Inspector**

选中变量时展示：

```text
变量信息
S7 地址
数据解释
数据点信息
最近读取
校验问题
```

未选变量时展示：

```text
变量组信息
地址分布
读取计划摘要
风险提示
```

- [ ] **步骤 4：验证**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 5：提交基础组件**

```powershell
git add datacenter/src/components/s7/S7GroupTree.vue datacenter/src/components/s7/S7VariableTable.vue datacenter/src/components/s7/S7InspectorPanel.vue
git commit -m "feat(datacenter): 实现 S7 工作台基础组件"
```

---

### 任务 10：实现 S7 弹窗组件

**文件：**

- 创建：`datacenter/src/components/s7/S7ProfileDialog.vue`
- 创建：`datacenter/src/components/s7/S7GroupDialog.vue`
- 创建：`datacenter/src/components/s7/S7VariableDialog.vue`
- 创建：`datacenter/src/components/s7/S7ImportDialog.vue`
- 创建：`datacenter/src/components/s7/S7PreviewDialog.vue`
- 创建：`datacenter/src/components/s7/S7ValidationDrawer.vue`
- 创建：`datacenter/src/components/s7/S7ReadPlanDialog.vue`

- [ ] **步骤 1：实现 PLC 类型弹窗**

`S7ProfileDialog.vue` 分四段：

```text
01 PLC 类型
02 连接参数
03 读取能力
04 地址能力
```

PLC 系列选项：

```ts
const plcFamilyOptions = ['S7-200', 'S7-200 SMART', 'S7-300', 'S7-400', 'S7-1200', 'S7-1500', 'S7 Compatible']
```

切换系列时调用：

```ts
const applyFamilyDefaults = (family: string) => {
  if (family === 'S7-300' || family === 'S7-400') {
    form.communicationMode = 'rack_slot'
    form.rack = 0
    form.slot = 2
    form.maxGapBytes = 8
  } else if (family === 'S7-1200' || family === 'S7-1500') {
    form.communicationMode = 'rack_slot'
    form.rack = 0
    form.slot = 1
    form.maxGapBytes = 16
    form.optimizedBlockAccess = true
  } else if (family === 'S7-200' || family === 'S7-200 SMART') {
    form.communicationMode = 'tsap'
    form.maxGapBytes = 8
  } else {
    form.communicationMode = 'rack_slot'
    form.rack = 0
    form.slot = 1
    form.maxGapBytes = 8
  }
}
```

- [ ] **步骤 2：实现变量弹窗**

`S7VariableDialog.vue` 分四段：

```text
01 基础信息
02 S7 地址
03 数据解释
04 采样
```

地址输入只要求用户填 `addressText`，下方显示解析预览：

```text
标准地址：DB1.DBD4
字节范围：DB1 4-7 bytes
```

解析预览可先前端轻量解析，保存时以后端解析结果为准。

- [ ] **步骤 3：实现导入弹窗**

导入方式先支持粘贴表格文本，列：

```text
变量名,Code,分组,地址,类型,单位,倍率,偏移,采集周期,描述
```

弹窗内显示导入确认表：

```text
状态 | 变量名 | Code | 分组 | 原始地址 | 标准地址 | 类型 | 字节范围 | 问题
```

- [ ] **步骤 4：实现预览、校验和读取计划弹窗**

预览弹窗展示：

```text
变量名 | 地址 | 当前值 | 更新时间 | 质量
```

读取计划弹窗展示：

```text
区域 | DB号 | 读取范围 | 字节 | 变量数 | 周期
```

校验抽屉支持定位变量事件：

```ts
defineEmits<{ (event: 'locate', variableId: string): void }>()
```

- [ ] **步骤 5：验证**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：PASS。

- [ ] **步骤 6：提交弹窗组件**

```powershell
git add datacenter/src/components/s7
git commit -m "feat(datacenter): 增加 S7 工作台弹窗"
```

---

### 任务 11：串联前端工作流和 UI 细节

**文件：**

- 修改：`datacenter/src/components/access-source/workbench/S7WorkbenchPanel.vue`
- 修改：`datacenter/src/components/s7/*.vue`

- [ ] **步骤 1：串联 CRUD**

工作台容器实现：

```ts
const saveProfile = async (payload) => {
  await dataAPI.updateS7Profile(props.projectId, props.connection.id, payload)
  profileDialogVisible.value = false
  await Promise.all([loadProfile(), loadReadPlanEstimate(), runValidationSilently()])
}

const saveVariable = async (payload) => {
  if (variableDialogMode.value === 'edit' && editingVariable.value) {
    await dataAPI.updateS7Variable(props.projectId, props.connection.id, editingVariable.value.id, payload)
  } else {
    await dataAPI.createS7Variable(props.projectId, props.connection.id, payload)
  }
  variableDialogVisible.value = false
  await reloadAll()
}
```

- [ ] **步骤 2：串联开发态读取**

使用现有 `useProtocolDevSession`：

```ts
const session = useProtocolDevSession({
  create: () => dataAPI.createS7DevSession(props.projectId, props.connection.id),
  close: (sessionId: string) => dataAPI.closeS7DevSession(props.projectId, props.connection.id, sessionId),
})
```

读取当前分组：

```ts
const readCurrentScope = async () => {
  const result = await dataAPI.readS7DevSession(props.projectId, props.connection.id, session.sessionId.value, {
    groupId: selectedGroupId.value || null,
  })
  applyReadValues(result.data?.values || [])
}
```

- [ ] **步骤 3：最近值回填变量表**

实现：

```ts
const applyReadValues = (values: S7ReadValue[]) => {
  const valueMap = new Map(values.map((item) => [item.variableId, item]))
  variables.value = variables.value.map((variable) => {
    const next = valueMap.get(variable.id)
    if (!next) return variable
    return {
      ...variable,
      lastValue: next.value,
      quality: next.quality,
      lastUpdatedAt: next.timestamp,
    }
  })
}
```

- [ ] **步骤 4：UI 家族化打磨**

检查：

```text
不增加横向全局 Header
左侧 Header 显示名称、类型、端点、PLC 类型、Rack/Slot
图标按钮使用 title / aria-label
卡片半径不超过现有工作台风格
右侧 Inspector 不嵌套卡片
长地址、数据点 path 使用省略和 title
```

- [ ] **步骤 5：验证**

运行：

```powershell
pnpm --filter datacenter typecheck
pnpm --filter datacenter build
```

预期：PASS。构建如出现 chunk size 警告，只记录，不作为失败。

- [ ] **步骤 6：提交前端串联**

```powershell
git add datacenter/src/components/access-source/workbench/S7WorkbenchPanel.vue datacenter/src/components/s7
git commit -m "feat(datacenter): 串联 S7 工作台交互"
```

---

### 任务 12：最终联调和验收

**文件：**

- 涉及前面所有 S7 文件

- [ ] **步骤 1：后端全量验证**

运行：

```powershell
cd D:\SVNCode\indu-forge\data_service
go test ./...
```

预期：PASS。

- [ ] **步骤 2：前端类型和构建验证**

运行：

```powershell
cd D:\SVNCode\indu-forge
pnpm --filter datacenter typecheck
pnpm --filter datacenter build
```

预期：PASS。

- [ ] **步骤 3：手动验收路径**

验收：

```text
创建 S7 接入源后进入 S7 工作台。
左侧 Header 显示 S7 基础信息，无横向全局 Header。
首次进入可打开 PLC 类型弹窗，选择 S7-1200 后带出 Rack=0、Slot=1、合并间隙=16。
新建变量组后左侧树刷新并选中。
新建变量输入 DB1.DBD4 / Real 后保存成功，变量表显示标准地址和字节范围 DB1 4-7。
变量保存后生成 s7.variable 数据点。
建模校验能识别 Bool 缺少 bitOffset、String 缺少 length、地址区不支持等问题。
读取计划能输出读取块、总字节、reads/s 和优化 DB 访问风险。
连接开发态会话后读取当前分组，变量表最近值和质量回填。
发布 artifact 中 protocols.s7 包含 profile、variables、readPlans。
```

- [ ] **步骤 4：提交最终修正**

如验证阶段产生修正：

```powershell
git add <修正文件>
git commit -m "fix(datacenter): 修正 S7 工作台联调问题"
```

如果没有修正，不创建空提交。

---

## 计划自检

- 规格覆盖：PLC 类型确认、三栏工作台、变量组、变量、导入、数据点同步、校验、开发态读取、读取计划、artifact、高可用边界均有任务覆盖。
- 类型一致性：后端统一使用 `S7Profile / S7Variable / S7ReadPlanEstimate`；前端统一使用 `S7Profile / S7VariableGroup / S7Variable`。
- 验证闭环：每组后端任务包含 `go test`，每组前端任务包含 `pnpm --filter datacenter typecheck`，最终包含 `build`。
- 修改边界：计划新增 S7 文件，修改共享入口文件；不要求改动当前未提交的 Modbus/OPC UI 文件。
