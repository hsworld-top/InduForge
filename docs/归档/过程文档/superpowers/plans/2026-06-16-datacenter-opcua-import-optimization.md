# OPC UA 变量导入优化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 完整优化 OPC UA 工作台的浏览导入和批量 NodeId 导入，并让后端批量导入避免半成功。

**架构：** 前端 `OpcuaImportDialog` 负责浏览/文本两种来源的候选行生成、预检、统一确认和二次确认；后端 `OpcuaModelingService` 与仓储新增事务化批量创建接口，批量写入变量并同步数据点。保持现有 API 路径和响应结构不变。

**技术栈：** Vue 3 + Element Plus + TypeScript；Go + pgx + PostgreSQL；pnpm workspace；Go test。

---

## 文件结构

- 修改 `datacenter/src/components/opcua/OpcuaImportDialog.vue`：重构导入弹窗，增加树形浏览、智能解析、统一预检、错误报告、二次确认。
- 修改 `datacenter/src/components/opcua/types.ts`：为浏览树和导入候选行所需字段补充前端类型。
- 修改 `data_service/internal/repository/opcua_modeling_repository.go`：新增批量事务写入方法，并在事务中同步 `data_points`。
- 修改 `data_service/internal/service/opcua_modeling_service.go`：新增批量导入规范化、重复检查、唯一 Code 生成和事务调用。
- 新增或修改 `data_service/internal/service/opcua_modeling_service_test.go`：覆盖批量上限、重复 NodeId 拒绝、Code 自动补后缀等纯服务规则。
- 修改计划文件本身：执行时勾选步骤。

### 任务 1：后端批量导入规则测试

**文件：**

- 创建：`data_service/internal/service/opcua_modeling_service_test.go`
- 修改：`data_service/internal/service/opcua_modeling_service.go`

- [ ] **步骤 1：编写批量规范化纯函数测试**

新增测试覆盖：

```go
func TestNormalizeOpcuaImportNodesGeneratesUniqueCodes(t *testing.T) {
	existing := map[string]struct{}{"speed": {}}
	input := []ImportOpcuaNodeInput{
		{Name: "Speed", NodeID: "ns=2;s=Line1.Speed", DataType: "Double"},
		{Name: "Speed", NodeID: "ns=2;s=Line1.Speed2", DataType: "Double"},
	}
	rows, err := normalizeOpcuaImportRows(input, existing)
	if err != nil {
		t.Fatalf("normalize import rows: %v", err)
	}
	if got := []string{rows[0].Code, rows[1].Code}; !reflect.DeepEqual(got, []string{"speed_2", "speed_3"}) {
		t.Fatalf("codes = %#v", got)
	}
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd data_service && go test ./internal/service -run TestNormalizeOpcuaImportRows -count=1`

预期：FAIL，提示 `normalizeOpcuaImportRows` 未定义。

- [ ] **步骤 3：实现纯规范化函数**

在 `opcua_modeling_service.go` 中新增小函数：

```go
type normalizedOpcuaImportRow struct {
	CreateOpcuaNodeInput
	SortOrder int
}

func normalizeOpcuaImportRows(nodes []ImportOpcuaNodeInput, existingCodes map[string]struct{}) ([]normalizedOpcuaImportRow, error) {
	// 校验输入、生成唯一 Code、补默认值；数据库重复 NodeId 由服务层传入前先检查。
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：`cd data_service && go test ./internal/service -run TestNormalizeOpcuaImportRows -count=1`

预期：PASS。

- [ ] **步骤 5：Commit**

提交：`test(data_service): 覆盖OPC UA批量导入规范化`

### 任务 2：后端事务化批量导入

**文件：**

- 修改：`data_service/internal/repository/opcua_modeling_repository.go`
- 修改：`data_service/internal/service/opcua_modeling_service.go`

- [ ] **步骤 1：新增仓储查询与事务方法**

新增方法：

```go
func (r *OpcuaModelingRepository) ListNodeIdentityMap(ctx context.Context, projectID, connectionID string) (map[string]string, map[string]string, error)
func (r *OpcuaModelingRepository) BatchCreateNodesWithDataPoints(ctx context.Context, params BatchCreateOpcuaNodesParams) ([]OpcuaNodeRecord, error)
```

事务方法在同一个 `pgx.Tx` 内插入 `data_opcua_nodes` 和 upsert `data_points`。

- [ ] **步骤 2：改造服务层 BatchImportNodes**

实现：

```go
const maxOpcuaBatchImportNodes = 1000

func (s *OpcuaModelingService) BatchImportNodes(...) ([]OpcuaNode, error) {
	if len(nodes) == 0 { ... }
	if len(nodes) > maxOpcuaBatchImportNodes { ... }
	if err := s.validateProjectConnectionAndUser(...); err != nil { ... }
	nodeIDs, codes, err := s.repository.ListNodeIdentityMap(...)
	// 检查已有 NodeId、本批次重复 NodeId，生成唯一 Code，调用事务仓储。
}
```

- [ ] **步骤 3：运行 Go 格式化**

运行：`gofmt -w data_service/internal/repository/opcua_modeling_repository.go data_service/internal/service/opcua_modeling_service.go data_service/internal/service/opcua_modeling_service_test.go`

预期：无输出。

- [ ] **步骤 4：运行后端相关测试**

运行：`cd data_service && go test ./internal/service -run Opcua -count=1`

预期：PASS。

- [ ] **步骤 5：Commit**

提交：`feat(data_service): 事务化导入OPC UA变量`

### 任务 3：前端导入弹窗重构

**文件：**

- 修改：`datacenter/src/components/opcua/OpcuaImportDialog.vue`
- 修改：`datacenter/src/components/opcua/types.ts`

- [ ] **步骤 1：补充类型**

在 `types.ts` 中补充可选字段：

```ts
export type OpcuaBrowseNode = {
  id: string
  parentId?: string | null
  name: string
  nodeId: string
  nodeType: 'folder' | 'variable'
  dataType?: string
  modeled?: boolean
  browseName?: string | null
  displayName?: string | null
}
```

- [ ] **步骤 2：实现候选行模型与预检**

在 `OpcuaImportDialog.vue` 中新增：

```ts
type OpcuaImportPreviewRow = {
  source: 'browse' | 'manual'
  rowNo: number
  name: string
  code: string
  nodeId: string
  dataType: string
  samplingMs: number
  accessLevel: string
  deadband: number | null
  unit: string | null
  issue: string
  state: 'ready' | 'existing' | 'duplicate' | 'error'
}
```

实现从浏览选中和文本解析生成统一 `previewRows`。

- [ ] **步骤 3：实现 UI**

弹窗包含摘要条、工具条、浏览树/文本输入页签、批量默认值、确认表、只看问题、错误报告、二次确认。

- [ ] **步骤 4：运行前端类型检查**

运行：`pnpm --filter datacenter typecheck`

预期：PASS。

- [ ] **步骤 5：Commit**

提交：`feat(datacenter): 优化OPC UA变量导入弹窗`

### 任务 4：联动与全量验证

**文件：**

- 修改：必要时修复前面任务暴露的问题。

- [ ] **步骤 1：运行后端测试**

运行：`pnpm go:test:data`

预期：PASS，若环境依赖导致失败，记录具体失败原因。

- [ ] **步骤 2：运行前端类型检查**

运行：`pnpm --filter datacenter typecheck`

预期：PASS。

- [ ] **步骤 3：检查工作区**

运行：`git status --short`

预期：只包含本任务相关文件和用户原有 `ConnectionDialog.vue` 未提交改动。

- [ ] **步骤 4：最终提交与推送**

如果有验证修复，提交：`fix(datacenter): 完善OPC UA变量导入验证`。

推送：`git push -u origin codex/opcua-import-optimization`
