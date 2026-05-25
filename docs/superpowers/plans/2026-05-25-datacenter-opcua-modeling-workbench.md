# 数据中心 OPC UA 点位建模工作台实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 OPC UA 接入源工作台完善为点位建模入口，支持变量组、变量、NodeId、采样策略、数据点自动同步、建模校验和辅助预览。

**架构：** `data_service` 新增 OPC UA 变量组与变量建模表，服务层负责 CRUD、批量导入、数据点同步和校验。`datacenter` 新增 OPC UA 专用工作台，采用左侧变量组、中间变量表、右侧配置面板的三栏结构；导入与新建变量保存后自动生成 `opcua.node` 数据点。真实采集仍由运行态基于外部发布契约执行。

**技术栈：** Go、pgx、PostgreSQL migration、Vue 3、Element Plus、TypeScript、Vite、Vitest、Go test。

---

## 文件结构

- 创建：`data_service/internal/db/migrations/0027_opcua_node_modeling.sql`
  新增 `data_opcua_node_groups` 与 `data_opcua_nodes`。
- 创建：`data_service/internal/db/migrations/0027_opcua_node_modeling_down.sql`
  回滚 OPC UA 点位建模表。
- 创建：`data_service/internal/repository/opcua_modeling_repository.go`
  参数化实现变量组、变量、批量导入、校验查询和数据点同步所需查询。
- 创建：`data_service/internal/service/opcua_modeling_service.go`
  封装业务规则：变量命名、code 归一化、数据点 path 分配、同步数据点、删除标记失效、建模校验。
- 创建：`data_service/internal/http/handler/opcua_modeling_handler.go`
  暴露变量组、变量、导入、预览和校验接口。
- 修改：`data_service/internal/app/server.go`
  初始化 OPC UA 建模 repository、service、handler。
- 修改：`data_service/internal/http/router/router.go`
  挂载 OPC UA 建模路由。
- 创建：`data_service/tests/integration/opcua_modeling_api_test.go`
  覆盖变量组、变量、数据点同步、删除失效和校验。
- 修改：`data_service/internal/repository/project_snapshot_repository.go`
  将 OPC UA 变量定义纳入运行契约 artifact。
- 修改：`datacenter/src/api/data.api.ts`
  新增 OPC UA 变量组、变量、导入、预览、校验 API 封装。
- 创建：`datacenter/src/components/access-source/workbench/OpcuaWorkbenchPanel.vue`
  OPC UA 三栏建模工作台容器。
- 创建：`datacenter/src/components/opcua/OpcuaGroupTree.vue`
  变量组树与组操作入口。
- 创建：`datacenter/src/components/opcua/OpcuaNodeTable.vue`
  当前组变量表格。
- 创建：`datacenter/src/components/opcua/OpcuaInspectorPanel.vue`
  右侧变量组 / 变量详情、数据点信息和校验问题。
- 创建：`datacenter/src/components/opcua/OpcuaGroupDialog.vue`
  新建 / 编辑变量组弹窗。
- 创建：`datacenter/src/components/opcua/OpcuaNodeDialog.vue`
  新建 / 编辑变量弹窗。
- 创建：`datacenter/src/components/opcua/OpcuaImportDialog.vue`
  从 OPC UA 导入变量弹窗。
- 创建：`datacenter/src/components/opcua/OpcuaPreviewDialog.vue`
  变量预览弹窗。
- 创建：`datacenter/src/components/opcua/OpcuaValidationDrawer.vue`
  建模校验抽屉。
- 修改：`datacenter/src/components/access-source/AccessSourceWorkbench.vue`
  将 `type='opcua'` 路由到新的 OPC UA 工作台。

---

### 任务 1：新增 OPC UA 点位建模数据库表

**文件：**

- 创建：`data_service/internal/db/migrations/0027_opcua_node_modeling.sql`
- 创建：`data_service/internal/db/migrations/0027_opcua_node_modeling_down.sql`

- [ ] **步骤 1：编写迁移**

创建 `0027_opcua_node_modeling.sql`：

```sql
CREATE TABLE IF NOT EXISTS data_opcua_node_groups (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    parent_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    description text,
    sort_order integer NOT NULL DEFAULT 0,
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_opcua_node_groups_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_opcua_node_groups_parent_fkey
        FOREIGN KEY (parent_id) REFERENCES data_opcua_node_groups (id) ON DELETE CASCADE,
    CONSTRAINT data_opcua_node_groups_name_key
        UNIQUE (project_id, connection_id, parent_id, name)
);

CREATE INDEX IF NOT EXISTS data_opcua_node_groups_connection_parent_idx
    ON data_opcua_node_groups (project_id, connection_id, parent_id, sort_order, created_at);

CREATE TABLE IF NOT EXISTS data_opcua_nodes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    group_id uuid,
    name text NOT NULL CHECK (char_length(name) <= 100),
    code text NOT NULL CHECK (char_length(code) <= 100),
    node_id text NOT NULL CHECK (char_length(node_id) <= 1000),
    browse_name text,
    display_name text,
    data_type text NOT NULL CHECK (char_length(data_type) <= 50),
    unit text CHECK (unit IS NULL OR char_length(unit) <= 20),
    sampling_ms integer NOT NULL DEFAULT 1000 CHECK (sampling_ms > 0),
    deadband numeric(20, 6),
    access_level text NOT NULL DEFAULT 'Read' CHECK (access_level IN ('Read', 'Write', 'ReadWrite')),
    description text,
    sort_order integer NOT NULL DEFAULT 0,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'invalid')),
    created_by uuid NOT NULL,
    updated_by uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT data_opcua_nodes_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES data_connections (id) ON DELETE CASCADE,
    CONSTRAINT data_opcua_nodes_group_fkey
        FOREIGN KEY (group_id) REFERENCES data_opcua_node_groups (id) ON DELETE SET NULL,
    CONSTRAINT data_opcua_nodes_connection_code_key
        UNIQUE (project_id, connection_id, code),
    CONSTRAINT data_opcua_nodes_connection_node_key
        UNIQUE (project_id, connection_id, node_id)
);

CREATE INDEX IF NOT EXISTS data_opcua_nodes_group_order_idx
    ON data_opcua_nodes (project_id, connection_id, group_id, sort_order, created_at DESC);

CREATE INDEX IF NOT EXISTS data_opcua_nodes_status_idx
    ON data_opcua_nodes (project_id, connection_id, status);
```

- [ ] **步骤 2：编写回滚迁移**

创建 `0027_opcua_node_modeling_down.sql`：

```sql
DROP INDEX IF EXISTS data_opcua_nodes_status_idx;
DROP INDEX IF EXISTS data_opcua_nodes_group_order_idx;
DROP TABLE IF EXISTS data_opcua_nodes;
DROP INDEX IF EXISTS data_opcua_node_groups_connection_parent_idx;
DROP TABLE IF EXISTS data_opcua_node_groups;
```

- [ ] **步骤 3：运行数据库迁移测试**

运行：

```powershell
go test ./tests/integration -run TestMigrations
```

预期：迁移测试通过。

---

### 任务 2：实现后端 repository 与 service

**文件：**

- 创建：`data_service/internal/repository/opcua_modeling_repository.go`
- 创建：`data_service/internal/service/opcua_modeling_service.go`

- [ ] **步骤 1：定义 repository 记录与参数类型**

在 repository 文件中定义：

```go
type OpcuaNodeGroupRecord struct {
	ID           string
	ProjectID    string
	ConnectionID string
	ParentID     *string
	Name         string
	Description  *string
	SortOrder    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type OpcuaNodeRecord struct {
	ID           string
	ProjectID    string
	ConnectionID string
	GroupID      *string
	Name         string
	Code         string
	NodeID       string
	BrowseName   *string
	DisplayName  *string
	DataType     string
	Unit         *string
	SamplingMS   int
	Deadband     *float64
	AccessLevel  string
	Description  *string
	SortOrder    int
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
```

- [ ] **步骤 2：实现变量组 CRUD 查询**

实现方法：

```go
func (r *OpcuaModelingRepository) ListGroups(ctx context.Context, projectID, connectionID string) ([]OpcuaNodeGroupRecord, error)
func (r *OpcuaModelingRepository) CreateGroup(ctx context.Context, params CreateOpcuaNodeGroupParams) (*OpcuaNodeGroupRecord, error)
func (r *OpcuaModelingRepository) UpdateGroup(ctx context.Context, params UpdateOpcuaNodeGroupParams) (*OpcuaNodeGroupRecord, error)
func (r *OpcuaModelingRepository) DeleteGroup(ctx context.Context, projectID, groupID string) error
```

删除变量组时执行：

```sql
UPDATE data_opcua_nodes
SET group_id = NULL, updated_at = now(), updated_by = $3
WHERE project_id = $1 AND group_id = $2
```

再删除组记录。

- [ ] **步骤 3：实现变量 CRUD 查询**

实现方法：

```go
func (r *OpcuaModelingRepository) ListNodes(ctx context.Context, projectID, connectionID, groupID string) ([]OpcuaNodeRecord, error)
func (r *OpcuaModelingRepository) CreateNode(ctx context.Context, params CreateOpcuaNodeParams) (*OpcuaNodeRecord, error)
func (r *OpcuaModelingRepository) UpdateNode(ctx context.Context, params UpdateOpcuaNodeParams) (*OpcuaNodeRecord, error)
func (r *OpcuaModelingRepository) DeleteNode(ctx context.Context, projectID, nodeID string) error
func (r *OpcuaModelingRepository) GetNode(ctx context.Context, projectID, nodeID string) (*OpcuaNodeRecord, error)
```

- [ ] **步骤 4：实现 service 业务规则**

在 service 中实现：

```go
func normalizeOpcuaNodeName(displayName, browseName, nodeID string) string
func normalizeOpcuaNodeCode(name, nodeID string) string
func (s *OpcuaModelingService) syncNodeDatapoint(ctx context.Context, node repository.OpcuaNodeRecord, userID string) error
```

命名规则：

```go
func normalizeOpcuaNodeName(displayName, browseName, nodeID string) string {
	if strings.TrimSpace(displayName) != "" {
		return strings.TrimSpace(displayName)
	}
	if strings.TrimSpace(browseName) != "" {
		return strings.TrimSpace(browseName)
	}
	parts := strings.FieldsFunc(nodeID, func(r rune) bool {
		return r == '.' || r == '/' || r == ';' || r == '='
	})
	if len(parts) == 0 {
		return strings.TrimSpace(nodeID)
	}
	return parts[len(parts)-1]
}
```

数据点 path 规则：

```text
opcua.<连接名>.<变量组路径>.<变量 code>
```

创建或更新变量后调用 `syncNodeDatapoint`，upsert `data_points`，`source_type='opcua.node'`，`source_id=node.ID`。

- [ ] **步骤 5：删除变量时标记数据点失效**

在 service 的删除方法中：

```go
if err := s.repository.DeleteNode(ctx, projectID, nodeID); err != nil {
	return err
}
_, _ = s.datapoints.MarkInvalidBySource(ctx, projectID, "opcua.node", nodeID, stringPtr(userID))
return nil
```

- [ ] **步骤 6：运行 service 包测试**

运行：

```powershell
go test ./internal/service ./internal/repository
```

预期：通过。

---

### 任务 3：实现后端 HTTP 接口与集成测试

**文件：**

- 创建：`data_service/internal/http/handler/opcua_modeling_handler.go`
- 修改：`data_service/internal/app/server.go`
- 修改：`data_service/internal/http/router/router.go`
- 创建：`data_service/tests/integration/opcua_modeling_api_test.go`

- [ ] **步骤 1：编写集成测试**

覆盖路径：

```go
func TestOpcuaModelingAPI(t *testing.T) {
	// 1. 创建 OPC UA config
	// 2. POST /opcua/{connectionId}/node-groups
	// 3. POST /opcua/{connectionId}/nodes
	// 4. GET /opcua/{connectionId}/nodes
	// 5. 查询 datapoints，断言 source_type = opcua.node
	// 6. POST /opcua/{connectionId}/validate-model
	// 7. DELETE /opcua/{connectionId}/nodes/{nodeId}
	// 8. 查询 datapoints，断言 status = invalid
}
```

- [ ] **步骤 2：实现 handler**

接口：

```text
GET    /api/v1/data/projects/{projectId}/opcua/{connectionId}/node-groups
POST   /api/v1/data/projects/{projectId}/opcua/{connectionId}/node-groups
PUT    /api/v1/data/projects/{projectId}/opcua/{connectionId}/node-groups/{groupId}
DELETE /api/v1/data/projects/{projectId}/opcua/{connectionId}/node-groups/{groupId}

GET    /api/v1/data/projects/{projectId}/opcua/{connectionId}/nodes
POST   /api/v1/data/projects/{projectId}/opcua/{connectionId}/nodes
POST   /api/v1/data/projects/{projectId}/opcua/{connectionId}/nodes/batch-import
PUT    /api/v1/data/projects/{projectId}/opcua/{connectionId}/nodes/{nodeId}
DELETE /api/v1/data/projects/{projectId}/opcua/{connectionId}/nodes/{nodeId}

POST   /api/v1/data/projects/{projectId}/opcua/{connectionId}/validate-model
POST   /api/v1/data/projects/{projectId}/opcua/{connectionId}/preview
```

列表响应：

```go
response.WriteSuccess(w, reqID, map[string]any{
	"list": list,
})
```

- [ ] **步骤 3：挂载路由**

在 router 中新增 `mountOpcuaModelingRoutes`，读接口使用 `project:read`，写接口使用 `project:write`。

- [ ] **步骤 4：初始化依赖**

在 `server.go` 创建 repository、service、handler，并传入 router option。

- [ ] **步骤 5：运行集成测试**

运行：

```powershell
go test ./tests/integration -run TestOpcuaModelingAPI
```

预期：通过。

---

### 任务 4：将 OPC UA 点位纳入运行契约 artifact

**文件：**

- 修改：`data_service/internal/repository/project_snapshot_repository.go`
- 修改：相关 snapshot / artifact 测试文件

- [ ] **步骤 1：扩展 artifact 结构**

在 protocols 的 `opcua` 输出中加入节点数组：

```json
{
  "connectionId": "...",
  "type": "opcua",
  "config": {},
  "nodes": [
    {
      "id": "...",
      "groupId": "...",
      "name": "电机转速",
      "code": "motor_speed",
      "nodeId": "ns=2;s=Line1.Motor01.Speed",
      "dataType": "Double",
      "samplingMs": 1000,
      "deadband": 0.1,
      "accessLevel": "Read",
      "datapointPath": "opcua.line1.motor01.motor_speed"
    }
  ]
}
```

- [ ] **步骤 2：补充 repository 查询**

查询 `data_opcua_nodes` 并左连接 `data_points`：

```sql
SELECT n.id, n.group_id, n.name, n.code, n.node_id, n.data_type,
       n.sampling_ms, n.deadband, n.access_level, dp.path
FROM data_opcua_nodes n
LEFT JOIN data_points dp
  ON dp.project_id = n.project_id
 AND dp.source_type = 'opcua.node'
 AND dp.source_id = n.id
WHERE n.project_id = $1
  AND n.connection_id = $2
  AND n.status = 'active'
ORDER BY n.sort_order, n.created_at
```

- [ ] **步骤 3：运行 snapshot 测试**

运行：

```powershell
go test ./internal/repository ./tests/integration -run "Snapshot|Artifact|Opcua"
```

预期：通过。

---

### 任务 5：新增前端 API 封装

**文件：**

- 修改：`datacenter/src/api/data.api.ts`

- [ ] **步骤 1：新增 API 方法**

添加：

```ts
export const getOpcuaNodeGroups = (projectId, connectionId) =>
  request({ url: `/data/projects/${projectId}/opcua/${connectionId}/node-groups`, method: 'get' })

export const createOpcuaNodeGroup = (projectId, connectionId, data) =>
  request({ url: `/data/projects/${projectId}/opcua/${connectionId}/node-groups`, method: 'post', data })

export const updateOpcuaNodeGroup = (projectId, connectionId, groupId, data) =>
  request({ url: `/data/projects/${projectId}/opcua/${connectionId}/node-groups/${groupId}`, method: 'put', data })

export const deleteOpcuaNodeGroup = (projectId, connectionId, groupId) =>
  request({ url: `/data/projects/${projectId}/opcua/${connectionId}/node-groups/${groupId}`, method: 'delete' })

export const getOpcuaNodes = (projectId, connectionId, params = {}) =>
  request({ url: `/data/projects/${projectId}/opcua/${connectionId}/nodes`, method: 'get', params })

export const createOpcuaNode = (projectId, connectionId, data) =>
  request({ url: `/data/projects/${projectId}/opcua/${connectionId}/nodes`, method: 'post', data })

export const batchImportOpcuaNodes = (projectId, connectionId, data) =>
  request({ url: `/data/projects/${projectId}/opcua/${connectionId}/nodes/batch-import`, method: 'post', data })

export const updateOpcuaNode = (projectId, connectionId, nodeId, data) =>
  request({ url: `/data/projects/${projectId}/opcua/${connectionId}/nodes/${nodeId}`, method: 'put', data })

export const deleteOpcuaNode = (projectId, connectionId, nodeId) =>
  request({ url: `/data/projects/${projectId}/opcua/${connectionId}/nodes/${nodeId}`, method: 'delete' })

export const validateOpcuaModel = (projectId, connectionId) =>
  request({ url: `/data/projects/${projectId}/opcua/${connectionId}/validate-model`, method: 'post' })

export const previewOpcuaNodes = (projectId, connectionId, data) =>
  request({ url: `/data/projects/${projectId}/opcua/${connectionId}/preview`, method: 'post', data })
```

- [ ] **步骤 2：运行类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：通过。

---

### 任务 6：实现 OPC UA 工作台三栏容器

**文件：**

- 创建：`datacenter/src/components/access-source/workbench/OpcuaWorkbenchPanel.vue`
- 修改：`datacenter/src/components/access-source/AccessSourceWorkbench.vue`

- [ ] **步骤 1：创建容器组件**

模板骨架：

```vue
<template>
  <section class="opcua-workbench">
    <header class="opcua-workbench__header">
      <WorkbenchSourceHeader
        :title="connection.name || '未命名 OPC UA 接入源'"
        fallback-title="未命名 OPC UA 接入源"
        :meta="sourceMetaRows"
        @back="$emit('back')"
      >
        <template #actions>
          <el-button size="small" @click="runTest">测试连接</el-button>
          <el-button type="primary" size="small" @click="importVisible = true">从 OPC UA 导入变量</el-button>
          <el-button size="small" @click="openCreateNode">新建变量</el-button>
          <el-button size="small" @click="previewVisible = true">变量预览</el-button>
          <el-button size="small" @click="runValidation">建模校验</el-button>
          <el-button size="small" @click="reloadAll">刷新</el-button>
        </template>
      </WorkbenchSourceHeader>
    </header>
    <div class="opcua-workbench__body">
      <OpcuaGroupTree />
      <OpcuaNodeTable />
      <OpcuaInspectorPanel />
    </div>
  </section>
</template>
```

- [ ] **步骤 2：接入路由分支**

在 `AccessSourceWorkbench.vue` 中将 `opcua` 指向新组件：

```vue
<OpcuaWorkbenchPanel
  v-if="selectedConnection?.type === 'opcua'"
  :project-id="projectId"
  :connection="selectedConnection"
  @back="handleBack"
/>
```

- [ ] **步骤 3：运行构建**

运行：

```powershell
pnpm --filter datacenter build
```

预期：通过。

---

### 任务 7：实现变量组树与变量表格

**文件：**

- 创建：`datacenter/src/components/opcua/OpcuaGroupTree.vue`
- 创建：`datacenter/src/components/opcua/OpcuaNodeTable.vue`

- [ ] **步骤 1：变量组树**

组件 props / emits：

```ts
defineProps<{
  groups: OpcuaNodeGroup[]
  selectedGroupId: string
}>()

defineEmits<{
  (event: 'select', groupId: string): void
  (event: 'create'): void
  (event: 'edit', group: OpcuaNodeGroup): void
  (event: 'delete', group: OpcuaNodeGroup): void
}>()
```

界面包含搜索框、新建按钮、树列表和变量数量。

- [ ] **步骤 2：变量表格**

表格列：

```text
变量名 | NodeId | 类型 | 数据点 | 最近值 | 质量 | 状态 | 操作
```

组件 props / emits：

```ts
defineProps<{
  nodes: OpcuaNode[]
  loading: boolean
  selectedNodeId: string
}>()

defineEmits<{
  (event: 'select', node: OpcuaNode): void
  (event: 'edit', node: OpcuaNode): void
  (event: 'delete', node: OpcuaNode): void
}>()
```

- [ ] **步骤 3：运行类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：通过。

---

### 任务 8：实现右侧配置面板

**文件：**

- 创建：`datacenter/src/components/opcua/OpcuaInspectorPanel.vue`

- [ ] **步骤 1：实现变量组选中状态**

展示：

```text
变量组信息
名称
父级
变量数
数据点数量
校验问题
```

- [ ] **步骤 2：实现变量选中状态**

展示：

```text
变量信息
NodeId / 类型 / Code
采样策略
数据点信息
校验问题
```

- [ ] **步骤 3：空态**

未选中对象时显示：

```text
选择变量组或变量查看配置详情
```

- [ ] **步骤 4：运行构建**

运行：

```powershell
pnpm --filter datacenter build
```

预期：通过。

---

### 任务 9：实现变量组和变量编辑弹窗

**文件：**

- 创建：`datacenter/src/components/opcua/OpcuaGroupDialog.vue`
- 创建：`datacenter/src/components/opcua/OpcuaNodeDialog.vue`

- [ ] **步骤 1：变量组弹窗**

字段：

```text
变量组名称 *
父级变量组
说明
```

保存 payload：

```ts
{
  name: form.name.trim(),
  parentId: form.parentId || null,
  description: form.description.trim(),
}
```

- [ ] **步骤 2：变量弹窗**

字段：

```text
变量名 *
标识符 code *
变量组 *
NodeId *
BrowseName
数据类型 *
单位
采样周期
死区
访问级别
说明
```

保存 payload：

```ts
{
  name,
  code,
  groupId,
  nodeId,
  browseName,
  dataType,
  unit,
  samplingMs,
  deadband,
  accessLevel,
  description,
}
```

- [ ] **步骤 3：保存后刷新工作台**

父组件监听 `success`：

```ts
await reloadNodes()
await reloadGroups()
selectedNodeId.value = saved.id
```

- [ ] **步骤 4：运行类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：通过。

---

### 任务 10：实现导入、预览和校验弹窗

**文件：**

- 创建：`datacenter/src/components/opcua/OpcuaImportDialog.vue`
- 创建：`datacenter/src/components/opcua/OpcuaPreviewDialog.vue`
- 创建：`datacenter/src/components/opcua/OpcuaValidationDrawer.vue`

- [ ] **步骤 1：导入弹窗**

导入弹窗支持节点树数据。若后端 browse 暂无真实适配，使用手工批量 NodeId 输入区作为同一弹窗内的降级输入，不额外创建第二套入口。

确认区字段：

```text
变量名，可编辑
NodeId
类型
导入到变量组
采样周期
死区
```

提交 `batchImportOpcuaNodes`。

- [ ] **步骤 2：预览弹窗**

字段：

```text
读取一次
开始短时监听
停止
变量名
NodeId
最近值
质量
诊断日志
```

预览成功后向父组件 emit：

```ts
emit('value-update', { nodeId, value, quality, timestamp })
```

- [ ] **步骤 3：校验抽屉**

展示校验结果：

```text
severity
groupName
nodeName
message
定位变量
```

点击定位变量时 emit：

```ts
emit('locate', issue.nodeId)
```

- [ ] **步骤 4：运行构建**

运行：

```powershell
pnpm --filter datacenter build
```

预期：通过。

---

### 任务 11：端到端联调与最终验证

**文件：**

- 修改：前述所有相关文件

- [ ] **步骤 1：运行后端测试**

运行：

```powershell
go test ./...
```

预期：通过。

- [ ] **步骤 2：运行前端类型检查**

运行：

```powershell
pnpm --filter datacenter typecheck
```

预期：通过。

- [ ] **步骤 3：运行前端构建**

运行：

```powershell
pnpm --filter datacenter build
```

预期：通过。

- [ ] **步骤 4：手动验收路径**

验收：

```text
1. 创建 OPC UA 接入源。
2. 进入 OPC UA 工作台。
3. 创建变量组 Motor01。
4. 新建变量“电机转速”，填写 NodeId 与采样策略。
5. 保存后表格显示数据点 path。
6. 查询数据点列表，存在 source_type='opcua.node' 的记录。
7. 编辑变量名，数据点名称同步更新。
8. 删除变量，数据点状态变为 invalid。
9. 执行建模校验，重复 NodeId 能被识别并定位。
10. 打开外部发布运行契约，artifact 中包含 OPC UA 节点定义。
```

预期：所有路径符合设计文档。
