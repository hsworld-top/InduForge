# 工业驱动目录简化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [x]`）语法来跟踪进度。

**目标：** 将新建工业连接弹窗从“能力类别 → 厂商/协议 → 驱动”简化为“厂商/协议 → 驱动”。

**架构：** 保留 Manifest 的 `category` 作为搜索元数据，但 `buildCollectorDriverTree` 只按 `protocolFamily` 聚合。弹窗只渲染协议族和驱动两种节点，连接 Schema 加载流程保持不变。

**技术栈：** Vue 3、TypeScript、Element Plus、Vitest、ESLint。

---

## 文件结构

- 修改：`datacenter/src/components/collector-workbench/collector-workbench-model.ts` — 构建两级驱动树并保留分类搜索能力。
- 修改：`datacenter/src/components/collector-workbench/CollectorConnectionWizard.vue` — 移除分类节点渲染、展开键和样式。
- 修改：`datacenter/tests/collector-workbench.test.ts` — 验证协议族为顶层以及分类关键词仍可搜索。
- 修改：`docs/02-系统设计/统一工业采集工作台与驱动模型设计.md` — 同步用户可见目录结构。

### 任务 1：用测试固定两级目录行为

**文件：**
- 修改：`datacenter/tests/collector-workbench.test.ts:64`

- [x] **步骤 1：修改树结构测试**

```ts
it('builds a protocol family and driver tree with bilingual labels', () => {
  const tree = buildCollectorDriverTree(drivers)
  expect(tree[0]?.label).toBe('Modbus')
  expect(tree[1]?.label).toBe('Siemens Plc [西门子]')
  expect(tree[1]?.children?.[0]?.label).toBe('Siemens S7 TCP [西门子 S7 以太网]')
  expect(buildCollectorDriverTree(drivers, 'PLC')[0]?.protocolFamily).toBe('siemens')
  expect(buildCollectorDriverTree(drivers, '串口')[0]?.protocolFamily).toBe('modbus')
})
```

- [x] **步骤 2：运行测试验证失败**

运行：`pnpm --filter datacenter test -- collector-workbench.test.ts`

预期：树顶层仍是 `PLC [可编程控制器]`，新断言失败。

### 任务 2：实现两级目录并清理分类 UI

**文件：**
- 修改：`datacenter/src/components/collector-workbench/collector-workbench-model.ts:22`
- 修改：`datacenter/src/components/collector-workbench/collector-workbench-model.ts:110`
- 修改：`datacenter/src/components/collector-workbench/CollectorConnectionWizard.vue:37`

- [x] **步骤 1：将树节点类型收窄为 `family | driver`。**

- [x] **步骤 2：按 `protocolFamily` 直接聚合驱动，同时把 `category` 和分类中文名称保留在搜索词中。**

- [x] **步骤 3：删除 `IconTablerCategory`、分类节点渲染分支和 `.is-category` 样式，协议族节点继续显示图标和驱动数量。**

- [x] **步骤 4：运行定向测试。**

运行：`pnpm --filter datacenter test -- collector-workbench.test.ts`

预期：相关测试全部通过。

### 任务 3：同步文档并完成静态验证

**文件：**
- 修改：`docs/02-系统设计/统一工业采集工作台与驱动模型设计.md:101`

- [x] **步骤 1：将用户可见目录说明改为 `协议族 → 驱动`，明确 `category` 只用于搜索和内部管理。**

- [x] **步骤 2：运行 TypeScript 类型检查。**

运行：`pnpm --filter datacenter typecheck`

预期：退出码为 `0`。

- [x] **步骤 3：运行定向 ESLint。**

运行：`pnpm --dir datacenter exec eslint src/components/collector-workbench/collector-workbench-model.ts src/components/collector-workbench/CollectorConnectionWizard.vue tests/collector-workbench.test.ts`

预期：退出码为 `0`。

- [x] **步骤 4：运行生产构建。**

运行：`pnpm --filter datacenter build`

预期：构建成功；允许保留项目已有的大 chunk 警告。

- [x] **步骤 5：提交实现。**

```powershell
git add -- datacenter/src/components/collector-workbench/collector-workbench-model.ts datacenter/src/components/collector-workbench/CollectorConnectionWizard.vue datacenter/tests/collector-workbench.test.ts docs/02-系统设计/统一工业采集工作台与驱动模型设计.md docs/superpowers/plans/2026-07-15-industrial-driver-catalog-simplification.md
git commit -m "feat(datacenter): 简化工业驱动选择目录"
```
