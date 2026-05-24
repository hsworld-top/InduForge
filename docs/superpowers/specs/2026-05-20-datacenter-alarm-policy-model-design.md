# 数据中心报警策略模型与 UI 升级设计

> 日期：2026-05-20  
> 范围：`datacenter/` 报警工作区与 `data_service/` 报警策略接口、服务、存储。  
> 目标：将报警从“单目标单条件规则”升级为“分组管理的报警策略”。支持单点多条件、多点批量套用、多点计算后报警、筛选后全选和批量设置。

## 1. 背景

当前报警实现以 `AlarmRule` 为核心。一条规则只能绑定一个 `targetPath` 和一个 `ruleType`。

这个模型无法覆盖两类实际场景：

- 一个目标点同时配置高高、高、低、低低、偏差、变化率等多个报警条件。
- 多个目标点先计算出一个派生值，再对派生值配置高高、高、低、低低或自定义报警。

同时，报警点数量变多后，仅靠左侧单规则列表不够管理。需要分组、勾选、筛选、筛选后全选和批量设置。

## 2. 目标

- 引入报警分组，用于管理大量报警策略。
- 引入报警策略，替代“单条规则即一个条件”的模型。
- 一个策略可以包含多个报警条件。
- 一个策略可以作用于一个目标点，也可以作用于多个目标点。
- 一个策略可以先基于多个输入点计算，再对计算结果报警。
- 支持批量启停、批量移动分组、批量套用条件集。
- 左侧管理区升级为可筛选、可勾选、可批量操作的管理面板。
- 前后端一次按最终模型实现，不做旧单规则模型的运行时兼容。

## 3. 不做范围

- 不做通知渠道配置。
- 不写运行态报警事件。
- 不实现运行态抑制执行，只保存抑制策略并输出契约。
- 不做多级分组。分组为单层结构。
- 不保留旧“单规则 = 单目标 + 单条件”的前端编辑模式。
- 不新增通用流程引擎或复杂表达式编排器。

## 4. 核心概念

### 报警分组

分组用于管理策略集合。

```ts
type AlarmPolicyGroup = {
  id: string
  projectId: string
  name: string
  description?: string
  isEnabled: boolean
  sortOrder: number
  createdAt: string
  updatedAt: string
}
```

分组规则：

- 分组只做一层。
- 策略的 `groupId` 可为空。
- `groupId = null` 的策略直接展示在左侧根目录。
- 不创建“未分组”虚拟组。
- 分组启停影响组内策略最终生效状态。

### 报警策略

策略是最终业务对象。

```ts
type AlarmPolicy = {
  id: string
  projectId: string
  groupId?: string | null
  name: string
  description?: string
  mode: 'per_target' | 'derived'
  targets: AlarmTargetRef[]
  inputs: AlarmInputRef[]
  derivedExpression?: string
  conditions: AlarmCondition[]
  suppression: AlarmSuppression
  messageTemplate: string
  isEnabled: boolean
  contract: Record<string, unknown>
  createdAt: string
  updatedAt: string
}
```

字段说明：

- `mode = per_target`：逐点判断。多个目标点共用同一套条件。
- `mode = derived`：计算后判断。多个输入点先计算出一个值，再对派生值判断。
- `targets`：策略适用的目标点。逐点判断模式必填。
- `inputs`：计算输入点。计算后判断模式必填。
- `derivedExpression`：派生值表达式。计算后判断模式必填。
- `conditions`：同一策略下的报警条件集。

### 目标点引用

```ts
type AlarmTargetRef = {
  datapointId: string
  path: string
  name?: string
  dataType: string
}
```

### 输入点引用

```ts
type AlarmInputRef = {
  key: string
  datapointId: string
  path: string
  name?: string
  dataType: string
}
```

`key` 用于表达式引用，例如 `temperature - pressure * 0.1`。

### 报警条件

```ts
type AlarmCondition = {
  id: string
  type: 'HH' | 'H' | 'L' | 'LL' | 'deviation_high' | 'deviation_low' | 'rate_of_change' | 'cel'
  name: string
  isEnabled: boolean
  severity: 'info' | 'warning' | 'major' | 'critical'
  params: Record<string, unknown>
}
```

阈值类条件：

```json
{ "limit": 80, "hysteresis": 2, "durationMs": 30000 }
```

偏差类条件：

```json
{ "baselinePath": "metrics.target", "limit": 5, "durationMs": 30000 }
```

变化率条件：

```json
{ "windowMs": 60000, "limit": 10, "direction": "up" }
```

自定义条件：

```json
{ "expression": "value > 80 && quality == \"good\"" }
```

## 5. 左侧管理区

左侧管理区从列表升级为策略管理面板。

布局：

- 默认宽度 `400px`。
- 最小宽度 `360px`。
- 最大宽度 `460px`。
- 支持拖拽调整宽度。
- 小屏幕下切换为抽屉。

根目录直接展示分组节点和未分组策略项：

```text
分组 A
  策略 1
  策略 2
分组 B
  策略 3
未分组策略 4
未分组策略 5
```

规则：

- 有分组的策略展示在对应分组下。
- 未分组策略直接展示在根目录。
- 不展示“未分组”虚拟组。
- 分组节点可展开、收起、勾选。
- 策略节点可勾选、选中、启停。
- 分组勾选表示选择该分组下当前筛选命中的策略。
- 分组存在部分命中或部分勾选时展示半选状态。

## 6. 筛选与全选

左侧顶部提供筛选区。

筛选项：

- 名称搜索。
- 分组。
- 启用状态。
- 报警级别。
- 条件类型。
- 目标点。
- 判断模式。

全选规则：

- 全选只作用于当前筛选结果。
- 当前筛选结果指服务端按筛选条件命中的全部策略，不等同于当前可视区域。
- 全选只选择策略项，不选择分组本身。
- 分组节点的勾选状态由筛选结果中的策略项反推。
- 清空筛选后，已勾选策略保持勾选。
- 批量操作只作用于已勾选策略。

选择状态：

- 普通勾选使用策略 ID 集合。
- 筛选后全选使用筛选快照。
- 用户在筛选后全选状态下取消个别策略时，记录排除策略 ID。
- 批量操作提交时，后端按筛选快照重新计算命中策略，并排除取消勾选的策略。

## 7. 右侧工作区

右侧根据当前状态切换视图。

### 单策略编辑

选中一个策略时展示策略编辑器。

区域：

- 头部：策略名称、所属分组、启用状态、保存、删除。
- 目标区：选择单个目标点、多个目标点或点集。
- 计算区：选择逐点判断或计算后判断。
- 条件区：维护条件集。
- 扩展区：抑制策略、消息模板。
- 底部：试算、契约预览。

### 批量设置

勾选多个策略时展示批量设置视图。

支持：

- 批量启用。
- 批量停用。
- 批量移动分组。
- 批量套用条件集。
- 批量设置高限。
- 批量设置低限。
- 批量设置报警级别。

批量套用条件集时，以用户确认的条件集覆盖目标策略的同类型条件。不存在的同类型条件则新增。

### 分组概览

选中分组节点时展示分组概览。

内容：

- 分组名称。
- 分组启用状态。
- 组内策略数量。
- 各级别策略数量。
- 条件类型分布。
- 分组级批量操作入口。

## 8. 条件编辑 UI

条件区使用矩阵式编辑，不再按单条规则切换表单。

列：

- 启用。
- 类型。
- 名称。
- 阈值或表达式摘要。
- 持续时间。
- 回差。
- 级别。
- 操作。

交互：

- 同一策略内可同时存在 HH、H、L、LL、偏差、变化率、自定义条件。
- 阈值类条件支持快速填入高高、高、低、低低。
- 条件可单独启停。
- 条件类型决定参数字段。
- 非数值目标禁止添加非自定义条件。
- 逐点判断模式下，条件对每个目标点分别生效。
- 计算后判断模式下，条件对派生值生效。

## 9. 批量创建

新建策略支持两种入口。

### 从目标点批量创建

流程：

1. 选择多个目标点。
2. 选择所属分组。
3. 选择逐点判断。
4. 配置条件集。
5. 保存为一个策略。

结果：一个策略作用于多个目标点，共用同一套条件。

### 从计算表达式创建

流程：

1. 添加多个输入点。
2. 为每个输入点设置变量名。
3. 编写派生表达式。
4. 配置条件集。
5. 保存为一个策略。

结果：一个策略先计算派生值，再对派生值报警。

## 10. API

接口前缀保持 `/api/v1`。响应保持 `code`、`msg`、`data`、`reqId`。

### 分组接口

- `GET /api/v1/data/projects/:projectId/alarm-policy-groups`
- `POST /api/v1/data/projects/:projectId/alarm-policy-groups`
- `PUT /api/v1/data/projects/:projectId/alarm-policy-groups/:id`
- `PATCH /api/v1/data/projects/:projectId/alarm-policy-groups/:id/enabled`
- `DELETE /api/v1/data/projects/:projectId/alarm-policy-groups/:id`

删除分组时，组内策略的 `groupId` 置空。策略回到根目录。

### 策略接口

- `GET /api/v1/data/projects/:projectId/alarm-policies`
- `GET /api/v1/data/projects/:projectId/alarm-policies/tree`
- `POST /api/v1/data/projects/:projectId/alarm-policies`
- `GET /api/v1/data/projects/:projectId/alarm-policies/:id`
- `PUT /api/v1/data/projects/:projectId/alarm-policies/:id`
- `PATCH /api/v1/data/projects/:projectId/alarm-policies/:id/enabled`
- `DELETE /api/v1/data/projects/:projectId/alarm-policies/:id`
- `POST /api/v1/data/projects/:projectId/alarm-policies/:id/test`
- `GET /api/v1/data/projects/:projectId/alarm-policies/:id/contract`
- `POST /api/v1/data/projects/:projectId/alarm-policies/validate-draft`

列表返回 `data.list` 和 `data.pagination`。

列表查询参数：

- `search`
- `groupId`
- `enabled`
- `severity`
- `conditionType`
- `targetPath`
- `mode`
- `page`
- `pageSize`

`alarm-policies/tree` 用于左侧管理区。它使用相同筛选参数，但不使用页码分页，返回当前筛选命中的轻量树：

```json
{
  "groups": [],
  "rootPolicies": [],
  "matchedPolicyCount": 0,
  "totalPolicyCount": 0
}
```

左侧使用服务端筛选和前端虚拟滚动展示树数据。

### 批量接口

- `POST /api/v1/data/projects/:projectId/alarm-policies/batch-enable`
- `POST /api/v1/data/projects/:projectId/alarm-policies/batch-disable`
- `POST /api/v1/data/projects/:projectId/alarm-policies/batch-move`
- `POST /api/v1/data/projects/:projectId/alarm-policies/batch-apply-conditions`

批量接口请求体使用策略 ID 列表：

```json
{ "policyIds": ["id-1", "id-2"] }
```

筛选后全选提交筛选快照：

```json
{
  "selection": {
    "mode": "filtered",
    "filters": {
      "search": "温度",
      "enabled": true,
      "severity": "major"
    },
    "excludePolicyIds": ["id-3"]
  }
}
```

后端按 `filters` 重新查询命中策略，再排除 `excludePolicyIds`。

批量移动：

```json
{ "policyIds": ["id-1", "id-2"], "groupId": "group-id" }
```

移动到根目录时：

```json
{ "policyIds": ["id-1", "id-2"], "groupId": null }
```

批量套用条件：

```json
{
  "policyIds": ["id-1", "id-2"],
  "conditions": []
}
```

## 11. 后端校验

后端是最终裁决点。

- `projectId`、`id`、`groupId` 必须为有效 UUID。
- `name` 非空，最长 100 字符。
- `groupId` 为空时表示根目录策略。
- `mode` 必须为 `per_target` 或 `derived`。
- `per_target` 模式必须有至少一个目标点。
- `derived` 模式必须有至少一个输入点和一个派生表达式。
- 目标点和输入点必须属于当前项目且状态 active。
- 非数值目标不能使用阈值、偏差、变化率条件。
- 条件类型必须是最终枚举。
- 条件参数必须与条件类型匹配。
- 条件集至少包含一个启用条件。
- 批量接口中的策略 ID 必须全部属于当前项目。
- 分组启停不改写策略自身 `isEnabled`，只影响最终生效状态。

错误码优先复用现有规则。确需新增错误码时，同步更新 `docs/统一错误码枚举表.md`。

## 12. 契约结构

契约升级为策略契约。

```json
{
  "schemaVersion": "alarm.policy.v1",
  "policyId": "",
  "projectId": "",
  "groupId": null,
  "mode": "per_target",
  "targets": [],
  "inputs": [],
  "derivedExpression": "",
  "conditions": [],
  "suppression": {},
  "messageTemplate": "",
  "enabled": true,
  "effectiveEnabled": true,
  "updatedAt": ""
}
```

`effectiveEnabled = group.isEnabled && policy.isEnabled`。

契约只描述开发态策略，不生成运行态事件。

## 13. 数据库

新增或调整表：

- `data_alarm_policy_groups`
- `data_alarm_policies`

`data_alarm_policy_groups`：

- `id`
- `project_id`
- `name`
- `description`
- `is_enabled`
- `sort_order`
- `created_at`
- `updated_at`

`data_alarm_policies`：

- `id`
- `project_id`
- `group_id`
- `name`
- `description`
- `mode`
- `targets jsonb`
- `inputs jsonb`
- `derived_expression text`
- `conditions jsonb`
- `suppression jsonb`
- `message_template text`
- `is_enabled`
- `contract jsonb`
- `created_at`
- `updated_at`

索引：

- `data_alarm_policy_groups(project_id, sort_order)`
- `data_alarm_policies(project_id, group_id)`
- `data_alarm_policies(project_id, updated_at DESC)`
- `data_alarm_policies(project_id, is_enabled)`

约束：

- 同一项目下分组名唯一。
- 同一项目下策略名唯一。
- `group_id` 可为空。
- 删除分组时策略 `group_id` 置空。

## 14. 前端模块

`datacenter` 报警模块调整为：

- `AlarmWorkspace`：整体布局和状态编排。
- `AlarmManagementPanel`：左侧管理区。
- `AlarmPolicyTree`：分组与策略混合树。
- `AlarmPolicyFilters`：筛选区。
- `AlarmBulkToolbar`：批量操作栏。
- `AlarmPolicyEditor`：单策略编辑器。
- `AlarmConditionMatrix`：条件矩阵。
- `AlarmTargetSelector`：目标点选择。
- `AlarmInputExpressionPanel`：输入点和派生表达式。
- `AlarmGroupOverview`：分组概览。
- `AlarmBulkEditor`：批量设置视图。
- `alarmPolicyModel.ts`：前端策略模型、默认值、序列化和反序列化。

复用现有 `StatusBadge`、`EmptyState`、`LinkChip`、`PillButton`、`useDraft`、`useApiError`、`useConfirm`、`useServerPagination`。

时间展示统一使用 `dayjs`，格式为 `YYYY-MM-DD HH:mm:ss`。

## 15. UI 规范

UI 采用工业控制台风格。

- 高密度、克制、可扫描。
- 左侧管理区承载管理动作，不做装饰性侧栏。
- 分组和策略使用清晰层级线，不使用大卡片堆叠。
- 条件矩阵使用表格密度，避免把每个条件做成大卡片。
- 严重度只用局部色条和 badge。
- 批量操作栏固定在左侧筛选区下方。
- icon-only 按钮必须有 tooltip 或 `aria-label`。
- 文案短，不写功能说明式长文案。
- 不使用营销式 hero、渐变装饰和大面积单色氛围背景。

## 16. 数据流

1. 进入 `/debug/alarm`。
2. 前端拉取分组和策略列表。
3. 左侧按筛选条件生成混合树。
4. 用户勾选策略或分组。
5. 全选只选择当前筛选命中的策略。
6. 单选策略时，右侧拉取详情并初始化草稿。
7. 多选策略时，右侧进入批量设置。
8. 保存策略时，后端校验目标、输入、表达式和条件集。
9. 后端生成 `alarm.policy.v1` 契约并写库。
10. 前端刷新列表、详情和勾选状态。

## 17. 测试与验收

### 后端

单测覆盖：

- 分组创建、更新、启停、删除。
- 删除分组后策略回到根目录。
- 逐点判断策略校验。
- 计算后判断策略校验。
- 条件集多条件校验。
- 批量启用、停用、移动、套用条件。
- 契约 `effectiveEnabled` 生成。

命令：

```powershell
go test ./...
```

在 `data_service` 目录执行。

### 前端

测试覆盖：

- Zod schema 解析。
- `alarmPolicyModel.ts` 序列化和默认值。
- 未分组策略直接展示在根目录。
- 筛选后全选只选择筛选命中的策略。
- 分组半选状态。
- 批量操作 payload。
- 条件矩阵增删改。
- 逐点判断与计算后判断切换。

命令：

```powershell
pnpm --dir datacenter test
pnpm --dir datacenter build
```

### 手工验收

- 左侧宽度适合展示分组、策略、筛选和批量工具。
- 根目录同时展示分组和未分组策略。
- 没有“未分组”虚拟组。
- 筛选后点击全选，只选中筛选结果中的策略。
- 勾选分组时，只选中该分组下当前筛选命中的策略。
- 单策略可同时配置 HH、H、L、LL、偏差、变化率和自定义条件。
- 多个点可共用同一套条件。
- 多个输入点可计算派生值后报警。
- 批量启停、移动分组、套用条件可用。
- 契约预览为 `alarm.policy.v1`。

## 18. 风险

- 当前已实现的单规则接口和前端模型需要整体替换为策略模型，改动范围大。
- 批量套用条件必须定义覆盖规则，否则容易误改用户已有条件。
- 计算表达式需要变量名校验，避免输入点重名或引用不存在变量。
- 左侧筛选、勾选和分页若混用会增加状态复杂度。管理区使用树查询和虚拟滚动，批量操作使用筛选快照和排除项，避免“当前页全选”和“筛选后全选”语义混淆。
- 分组启停与策略启停需要同时展示 `isEnabled` 和 `effectiveEnabled`，避免用户误解。
