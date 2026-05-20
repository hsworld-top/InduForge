# 数据中心报警工作区前后端设计

> 日期：2026-05-20  
> 范围：`datacenter/` 报警工作区与 `data_service/` 报警规则接口、服务、存储。  
> 目标：一次交付最终报警开发态闭环。前后端一起开发，不做旧样例页兼容，不做接口缺失占位。

## 1. 目标

计算模块完成后，继续建设报警单元工作区。报警模块采用规则 IDE 双栏形态，完成规则列表、规则编辑、目标数据点选择、试算、契约预览、启停、删除和后端接口闭环。

本轮以前后端纵向闭环为目标。前端只接最终接口。后端同步补齐规则模型、校验、试算、契约生成和持久化能力。

## 2. 不做范围

- 不保留旧 `AlarmWorkspace.vue` 样例规则构建器。
- 不用本地样例数据冒充真实接口。
- 不做通知渠道配置。
- 不写运行态报警事件。
- 不实现运行态抑制执行，只保存抑制策略并输出契约。
- 不做旧 `threshold / range / expression` 前端可选项兼容。
- 不新增组件库、图标库。

## 3. 整体架构

### 前端

`datacenter` 报警模块拆为：

- `AlarmWorkspace`：布局、路由状态、整体加载。
- `AlarmRuleList`：搜索、筛选、分页、选中。
- `AlarmEditorShell`：右侧编辑器容器。
- `AlarmEditorHeader`：规则名、目标、启停、保存、删除和跨模块入口。
- `AlarmRuleForm`：规则类型和动态条件表单。
- `AlarmSuppressionPanel`：抑制策略。
- `AlarmMessageTemplatePanel`：消息模板。
- `AlarmTestPanel`：样本试算。
- `AlarmContractPanel`：契约 JSON 预览。
- `CreateAlarmRuleDialog`：新建规则入口。
- `DataPointPicker`：目标数据点选择。
- `alarmRuleModel.ts`：前端规则模型、默认值、序列化和反序列化。

复用现有 `StatusBadge`、`EmptyState`、`LinkChip`、`PillButton`、`useDraft`、`useApiError`、`useConfirm`、`useServerPagination`。

### 后端

`data_service` 继续使用现有分层：

- `internal/http/router/router.go` 注册报警规则路由。
- `internal/http/handler/alarm_rule_handler.go` 承接 HTTP 请求。
- `internal/service/alarm_rule_service.go` 负责校验、试算、契约生成。
- `internal/repository/alarm_rule_repository.go` 负责参数化 SQL。
- `internal/db/migrations` 新增报警规则迁移。

接口前缀保持 `/api/v1`。响应保持 `code`、`msg`、`data`、`reqId`。

## 4. 规则模型

`AlarmRule` 字段：

```ts
{
  id: string
  projectId: string
  name: string
  description?: string
  targetDatapointId: string
  targetPath: string
  targetName?: string
  targetDataType: string
  ruleType:
    | "H"
    | "L"
    | "HH"
    | "LL"
    | "deviation_high"
    | "deviation_low"
    | "rate_of_change"
    | "cel"
  condition: Record<string, unknown>
  severity: "info" | "warning" | "major" | "critical"
  isEnabled: boolean
  suppression: Record<string, unknown>
  messageTemplate: string
  contract: Record<string, unknown>
  createdAt: string
  updatedAt: string
}
```

前端使用 `dayjs` 展示时间，格式为 `YYYY-MM-DD HH:mm:ss`。

### condition

阈值类 `H / L / HH / LL`：

```json
{ "limit": 80, "hysteresis": 2, "durationMs": 30000 }
```

方向由 `ruleType` 决定，不再使用 `operator`。

偏差类 `deviation_high / deviation_low`：

```json
{ "baselinePath": "metrics.target", "limit": 5, "durationMs": 30000 }
```

变化率 `rate_of_change`：

```json
{ "windowMs": 60000, "limit": 10, "direction": "up" }
```

CEL `cel`：

```json
{ "expression": "value > 80 && quality == \"good\"" }
```

后端对 CEL 做语法级和变量白名单校验。变量白名单为 `value`、`quality`、`timestamp`、`baselineValue`、`previousValue`。

### suppression

```json
{
  "enabled": false,
  "mode": "duration",
  "durationMs": 300000
}
```

## 5. API

### 列表

`GET /api/v1/data/projects/:projectId/alarm-rules`

查询参数：

- `search`
- `ruleType`
- `severity`
- `enabled`
- `targetPath`
- `page`
- `pageSize`

返回 `data.list` 和 `data.pagination`。

### 创建

`POST /api/v1/data/projects/:projectId/alarm-rules`

请求体使用最终 `AlarmRule` 可写字段：

- `name`
- `description`
- `targetPath`
- `ruleType`
- `condition`
- `severity`
- `isEnabled`
- `suppression`
- `messageTemplate`

后端根据目标数据点生成 `targetDatapointId`、`targetDataType` 和 `contract`。

### 详情

`GET /api/v1/data/projects/:projectId/alarm-rules/:id`

返回完整 `AlarmRule`。

### 更新

`PUT /api/v1/data/projects/:projectId/alarm-rules/:id`

请求体支持更新创建接口的可写字段。后端重新校验规则并生成契约。

### 启停

`PATCH /api/v1/data/projects/:projectId/alarm-rules/:id/enabled`

请求：

```json
{ "isEnabled": true }
```

返回更新后的 `AlarmRule`。

### 删除

`DELETE /api/v1/data/projects/:projectId/alarm-rules/:id`

返回：

```json
{ "deleted": true }
```

### 试算

`POST /api/v1/data/projects/:projectId/alarm-rules/:id/test`

请求：

```json
{
  "value": 82.5,
  "timestamp": "2026-05-20T10:00:00Z",
  "context": {
    "quality": "good",
    "baselineValue": 75,
    "previousValue": 70
  }
}
```

返回：

```json
{
  "triggered": true,
  "state": "triggered",
  "severity": "major",
  "ruleType": "H",
  "targetPath": "metrics.temperature",
  "message": "温度过高：82.5",
  "diagnostics": {
    "limit": 80,
    "hysteresis": 2,
    "durationMs": 30000,
    "reason": "value >= limit"
  }
}
```

`state` 取值：

- `triggered`
- `not_triggered`
- `insufficient_input`

变化率和偏差规则缺少必要上下文时返回 `insufficient_input`，不作为系统错误。

### 契约预览

`GET /api/v1/data/projects/:projectId/alarm-rules/:id/contract`

返回后端生成的契约 JSON。

### 草稿校验

`POST /api/v1/data/projects/:projectId/alarm-rules/validate-draft`

用于未保存表单的目标校验、条件校验和契约预览。

请求：

```json
{
  "name": "温度高高报",
  "targetPath": "metrics.temperature",
  "ruleType": "HH",
  "condition": { "limit": 95, "hysteresis": 1, "durationMs": 10000 },
  "severity": "critical",
  "suppression": { "enabled": false, "mode": "duration", "durationMs": 300000 },
  "messageTemplate": "{{targetPath}} 触发 {{ruleType}}"
}
```

返回：

```json
{
  "valid": true,
  "errors": [],
  "target": {},
  "contract": {}
}
```

## 6. 后端校验

后端是最终裁决点：

- `projectId` 和 `id` 必须为有效 UUID。
- `name` 非空，最长 100 字符。
- `targetPath` 必须对应当前项目 active 数据点。
- 非 CEL 规则只能绑定数值型数据点。
- `ruleType` 必须是最终枚举。
- `severity` 必须是最终枚举。
- `condition` 必须与 `ruleType` 匹配。
- `suppression.enabled = true` 时，`durationMs` 必须大于 0。
- `messageTemplate` 可为空；为空时由后端生成默认模板。

错误码优先复用现有规则。确需新增错误码时，同步更新 `docs/统一错误码枚举表.md`。

## 7. 契约结构

契约字段：

```json
{
  "schemaVersion": "alarm.rule.v1",
  "ruleId": "",
  "projectId": "",
  "target": {
    "datapointId": "",
    "path": "",
    "dataType": ""
  },
  "ruleType": "H",
  "condition": {},
  "severity": "major",
  "suppression": {},
  "messageTemplate": "",
  "enabled": true,
  "updatedAt": ""
}
```

契约只描述开发态规则，不生成运行态事件。

## 8. 数据库

新增迁移调整 `data_alarm_rules`：

- 扩展 `rule_type` 约束为 `H / L / HH / LL / deviation_high / deviation_low / rate_of_change / cel`。
- 扩展 `severity` 约束为 `info / warning / major / critical`。
- 增加 `suppression jsonb NOT NULL DEFAULT '{}'::jsonb`。
- 增加 `message_template text NOT NULL DEFAULT ''`。
- 保留 `contract jsonb`。
- 保留唯一约束 `(project_id, name)`。
- 增加 `(project_id, updated_at DESC)` 索引。

如本地库已有旧开发态规则，迁移将旧 `threshold / range / expression` 转为最接近的新类型：

- `threshold` 转 `H`。
- `range` 根据 `condition.high` 转 `H`，保留原 `condition` 中可识别字段。
- `expression` 转 `cel`。

迁移只处理开发态数据，不做历史兼容逻辑留存。

## 9. 前端交互

### 布局

`AlarmWorkspace` 使用双栏：

- 左栏 300px，可折叠到 56px。
- 右侧填满剩余空间。
- 底部面板默认 280px，可拖到 420px。

路由：

- `/debug/alarm`
- `/debug/alarm/:objectId`
- `/debug/alarm/:objectId/test`
- `/debug/alarm/:objectId/contract`

刷新后恢复选中规则和底部 tab。

### 左侧列表

列表顶部提供搜索、新建、启用状态、规则类型和严重度筛选。列表项展示严重度色条、规则名、目标路径、规则类型、启停状态和更新时间。

不做规则分组。没有规则时显示空态。接口错误时展示 `msg` 和 `reqId`。

### 编辑器头条

头条提供规则名 inline 编辑、目标数据点摘要、启停开关、保存、删除、打开目标数据点和检查当前规则。

保存状态：

- 未修改时保存按钮禁用。
- 修改后显示脏状态。
- 保存中显示 loading。
- 保存失败保留草稿。

删除需要二次确认。删除成功后选中列表第一条；没有规则时进入空态。

### 表单

表单包含目标数据点、规则类型、动态条件字段、严重度和描述。

目标数据点选择器只允许 active 数据点。非数值数据点只允许 CEL。

CEL 编辑器懒加载 Monaco。

### 底部面板

底部面板包含：

- `规则配置`：抑制策略、消息模板、模板变量提示。
- `试算`：样本值、timestamp、context、试算结果。
- `契约`：只读 JSON、复制按钮、版本和目标摘要。

试算不写运行态事件。

## 10. UI 规范

UI 采用工业控制台风格：

- 沿用数据中心现有 `--dc-*` token。
- 高密度、克制、可扫描。
- 严重度只用局部色条和 badge，不染整张卡片。
- 表单区用分隔线和小标题，不嵌套卡片。
- 危险操作只在删除等动作中使用低饱和红色。
- 所有 icon-only 按钮必须有 tooltip 或 `aria-label`。
- 页面文案短，不写功能说明式长文案。

## 11. 数据流

1. 进入 `/debug/alarm`。
2. 前端通过 `alarm.store` 拉取列表。
3. 用户选中规则后，路由变为 `/debug/alarm/:objectId`。
4. 前端拉取详情并初始化草稿。
5. 用户编辑字段，`alarmRuleModel.ts` 生成最终 payload。
6. 保存时调用更新接口。
7. 后端校验目标、条件和模板，写库并重新生成契约。
8. 前端刷新详情并清除脏状态。
9. 试算调用后端 `test`。
10. 契约 tab 调用后端 `contract`。

## 12. 测试与验收

### 后端

单测覆盖：

- 每种规则类型的 condition 校验。
- 每种规则类型的试算。
- 非数值数据点绑定非 CEL 失败。
- 偏差规则缺少 `baselineValue` 返回 `insufficient_input`。
- 变化率缺少 `previousValue` 返回 `insufficient_input`。
- 契约字段完整。

集成测试覆盖：

- 创建、列表、详情、更新、启停、删除。
- `test`。
- `contract`。
- `validate-draft`。
- 响应包裹符合 `code/msg/data/reqId`。

命令：

```powershell
go test ./...
```

在 `data_service` 目录执行。

### 前端

测试覆盖：

- Zod schema 解析。
- `alarmRuleModel.ts` 序列化和默认值。
- 规则类型切换后的字段变化。
- 保存脏状态。
- 试算状态展示。
- 契约 JSON 复制状态。

命令：

```powershell
pnpm --dir datacenter build
pnpm --dir datacenter test
```

### 手工验收

- `/debug/alarm`
- `/debug/alarm/:objectId`
- `/debug/alarm/:objectId/test`
- `/debug/alarm/:objectId/contract`

验收点：

- 列表搜索、筛选、分页可用。
- 新建、编辑、保存、启停、删除可用。
- 目标数据点选择可搜索，只允许合法目标。
- 试算展示触发、不触发和输入不足。
- 契约 JSON 与后端返回一致。
- 刷新后恢复选中规则和底部 tab。
- 无旧样例规则构建器入口。

## 13. 风险

- 旧 `data_alarm_rules` 约束和新枚举不一致，需要迁移处理已有开发态数据。
- CEL 校验若做过深会扩大范围，本轮只做语法级和变量白名单。
- 报警规则和数据契约检查有关联，但本轮只在报警详情暴露“检查当前规则”入口，不扩展契约检查历史能力。
- 前后端字段必须一次对齐，避免前端再维护旧 `level/status` 字段别名。
