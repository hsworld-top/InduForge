# 数据中心前端 v2 设计

> 更新时间：2026-05-15
> 适用范围：`datacenter/` 前端的目标态设计。
> 子文档：
>
> - [01 数据点工作区](./datacenter-frontend-v2/01-datapoint.md)
> - [02 接入源工作区](./datacenter-frontend-v2/02-access-source.md)
> - [03 计算单元工作区](./datacenter-frontend-v2/03-compute.md)
> - [04 报警单元工作区](./datacenter-frontend-v2/04-alarm.md)
> - [05 数据契约检查](./datacenter-frontend-v2/05-contract-check.md)

## A. 设计目标与范围

### 适用范围

- 仅 `datacenter/` 前端。
- 后端 API 契约、`data_service` 服务分层、PG/TSDB 存储、性能内核**不在本轮范围**，按现有接口契约与 `docs/data_service/开发态能力完善计划.md` 渐进式补齐。

### 视觉与交互基准

- **数据点工作区（`DataPointList`）的 CSS 设计视为基线**：颜色、背景、阴影、圆角、字体、原子组件视觉风格在四个模块统一。
- **布局不被基线绑定**。每个模块按自身任务特点选布局形态（列表 / 卡片网格 / 双栏 IDE / 规则构建器等）。

### 基线 CSS 与原子组件（v2 全模块共用，遵循 `DESIGN.md` token 体系）

- 颜色：`--dc-primary` `--dc-text` `--dc-surface` 等 token 沿用现有 `:root` 定义。
- 字体层级：headline / title / body / label / meta / mono（参 `DESIGN.md`）。
- 圆角：`--dc-radius-sm/md/lg`。
- 阴影：surface / popover / overlay 三档。
- 间距：`--dc-space-xs/sm/md/lg/xl`。
- 原子组件视觉：
  - `pill-button`（药丸形态筛选 / 排序按钮）
  - `icon-button`（32×32 工具栏图标按钮，带 tooltip）
  - `status-badge`（状态徽标，文本 + 颜色双重表达）
  - `el-table` 行样式（行高 / hover / selected / 截断 / 表头）
  - `LinkChip`（跨模块跳转芯片）
  - 抽屉 / dialog 的标题栏、关闭按钮、内容滚动区
  - 空状态（`el-empty` 包装）、loading（`v-loading`）
  - 批量操作条（选中行时浮动条）

### `DESIGN.md` / `PRODUCT.md` 优化方向

详见 K 节。要点：

- `DESIGN.md` 补齐数据中心特有原子组件规范（pill-button、status-badge、抽屉、批量操作条、LinkChip）。
- `PRODUCT.md` 在「Design Principles」补一条「模块布局自由，视觉基底统一」。

### 原型基线

- 当前 `DataCenterShell` + `DataCenterNavRail` + `DataPointWorkspace` 视为基线。
- 数据中心是 IDE 内打开的标签页。**项目切换、租户、用户菜单、登录续租都由 IDE 宿主承载**。

### 设计目标

1. 四个一级模块共享视觉基底，但各自按任务特点选布局形态。
2. 大列表 / 详情抽屉在 500–1000 接入源 / 海量点假设下仍可用。
3. 后端能力按阶段落地，前端**降级路径**显式可见（"能力未启用"徽标）。

### 非目标

- 不重写 IDE 宿主链路、设计中心、运行态节点侧、`dev_core`。
- 不替换 Element Plus / Vue 3 / Vite / Tailwind 技术栈。
- 不引入项目切换器、用户菜单。
- **不保留旧 SQL/MQTT 工作台兜底入口**——v2 直接以最优实现重做接入源工作区，旧 `SqlQueryWorkbench` / `MqttWorkbench` / `DataCenterNew.vue` 里的 legacy 标签页体系一并废除。
- 不做完整移动端布局。
- 不把数据点工作区的「工具栏 + 表格 + 分页」布局强加到接入源、计算、报警。

---

## B. 信息架构与导航

### 整体框架（沿用现有 Shell）

```
┌─ NavRail（左侧竖向）────────────────────┬───── 主工作区 ─────┐
│  · 数据点（默认）                        │                    │
│  · 接入源                                │  当前一级模块的    │
│  · 计算单元                              │  工作区视图        │
│  · 报警单元                              │                    │
│  ─────                                  │                    │
│  侧栏底部动作槽位（actions slot）：       │                    │
│   · 数据契约检查                          │                    │
│   · 模块级动作（条件出现）                 │                    │
│   · 预览会话状态徽标（条件出现）           │                    │
└─────────────────────────────────────────┴────────────────────┘
```

### 一级模块

- 固定 4 项：`数据点 / 接入源 / 计算单元 / 报警单元`。
- 数据点为默认首页。
- 模块切换走 `NavRail`，仅更新 URL。

### 侧栏底部动作槽位

- **数据契约检查**：固定可见，任何模块下可用。
- **模块相关动作**：刷新、新建接入源、新建计算单元、新建规则等，随激活模块条件出现。
- **预览会话状态徽标**：**条件出现**，仅在接入源工作台的协议特定预览（如 MQTT 订阅消息预览、Kafka 短时 reader、HTTP/WebSocket 一次性请求）激活时显示。不作为全局连接状态指示器。

### 主工作区——布局自由，视觉统一

v2 不为四个模块定义统一布局结构。各模块按自身任务特点选形态，但视觉基底（A 节列出的 token 与原子组件）必须一致。

预期布局取向（具体方案在子文档展开）：

- 数据点：单栏列表（工具栏 + 表格 + 抽屉详情 + 分页）。
- 接入源：卡片网格 + 详情抽屉 + 二级工作台（接入源专属全屏）。
- 计算单元：左侧文件夹树 + 右侧编辑器 IDE 形态。
- 报警单元：左侧规则列表 + 右侧规则编辑器。

### 详情交互模式（统一约定）

- 「查看详情」走右侧**抽屉**。默认 480px，可拖至 720px，可"钉为旁路面板"成为第三栏。
- 抽屉**优先单页滚动**，Tab 仅在内容分量大且互斥时使用。
- 子操作（标签管理、权限编辑、契约预览等）走 **dialog**。
- 抽屉与 dialog 的视觉风格统一，遵循 A 节基线。

### 跨模块跳转链路

- 数据点 ↔ 接入源 / 计算单元 / 报警规则
- 接入源详情 → 关联数据点
- 计算单元 ↔ 输入数据点 / 输出 `calc.*` 数据点
- 报警规则 → 目标数据点
- 统一组件 `LinkChip`：行为「切换一级模块 + 打开目标对象详情」，URL 同步更新，浏览器后退可回。

### v2 不保留的

- 旧 `DataCenterNew.vue` 内的 `el-tabs` 多标签页系统。
- `SqlQueryWorkbench` / `MqttWorkbench` 独立工作台。
- 「双击连接进入旧工作台」的链路。
- SQL 编辑、MQTT Tag 管理、消息预览**重新组织进接入源二级工作台**（详细方案见子文档 02）。

---

## C. 路由与会话上下文

### 路由分层

- `/datacenter/`：IDE 宿主入口。正式链路下走 `APP_BOOTSTRAP_REQUEST/RESPONSE`，缺上下文回跳 IDE。
- `/datacenter/debug/`：独立调试入口。允许在本地兜底工程下运行，不要求合法 token；缺上下文显式提示，不白屏。
- 数据中心不维护任何"登录页"或"项目选择页"——这些不属于数据中心职责。

### 路由参数

```
/datacenter/:module/:objectId?/:tab?
```

- `module`: `datapoint | access-source | compute | alarm`（默认 `datapoint`）
- `objectId`: 选中对象 ID（在详情抽屉打开）
- `tab`: 详情抽屉当前 Tab key

筛选条件、搜索关键字、排序通过 `query string` 同步：`?source=mqtt&status=active&q=温度`。

### 会话上下文来源（按优先级）

1. IDE bootstrap 消息（正式入口主路径，承载 projectId / tenantId / token / refreshToken / theme / locale）
2. `localStorage` 缓存（用于刷新恢复，token 不进 localStorage，仅缓存 projectId、tenantId、theme）
3. URL `pid` 参数（仅 `/datacenter/debug/` 接受）
4. 本地兜底工程（仅 `/datacenter/debug/` 且无任何上下文时）

token / refreshToken **不进 URL、不进 localStorage**，统一通过 IDE 消息 + 内存持有；正式入口刷新页面需 IDE 重发 bootstrap。refresh 成功向宿主回发 `AUTH_REFRESHED`，失败回发 `AUTH_EXPIRED`。

### 未保存草稿保护

- 模块切换、刷新、关闭页签时检测当前模块草稿状态。
- 有草稿弹三选一对话框：丢弃 / 保留为草稿 / 取消。
- 草稿存 `localStorage`，按 `projectId + module + objectId` 隔离。

### 刷新即恢复

- 模块、筛选、选中对象、详情 Tab 全部序列化到 URL（query string + path）。
- 刷新页面（拿到 bootstrap 后）回到完全相同状态。

---

## D. 状态分层与数据流

### 全局 store（Pinia，按模块拆分）

- `useProjectStore`：projectId、tenantId、locale、theme、capability 列表，由 IDE bootstrap 注入。
- `useDataCatalogStore`：数据点目录的关键字段缓存（path / name / sourceType / status）。增量更新（按 path），不缓存运行态值。
- `useAccessSourceStore`：接入源摘要列表 + 选中态。
- `useComputeStore`：计算单元列表 + 文件夹树 + 当前编辑单元 + 调试会话。
- `useAlarmStore`：报警规则列表 + 当前编辑规则 + 试算上下文。
- `useContractCheckStore`：最近一次检查结果 + 进行中状态。
- `usePreviewSessionStore`：**局部作用域**——仅在接入源工作台内的协议特定预览激活时持有。
- `useUiPrefsStore`：用户偏好（抽屉宽度、列表每页），持久化到 `localStorage`。

### composables（跨组件复用的最小单元）

- `useDraft(key, schema)`：本地草稿读写，按 `projectId + module + objectId` 隔离。
- `useServerPagination(loader, options)`：服务端分页 + 筛选 + 排序的通用包装。
- `useDrawer()`：抽屉打开 / 关闭 / 钉住的状态机。
- `useConfirm()`：危险动作二次确认。
- `useApiError()`：统一错误展示（`code/msg/data/reqId` 包络）。

**移除**：`useDataPointSubscription`（v2 数据点工作区不订阅 / 不实时推送，开发态值通过"测试取值"按需触发）。

### API 层

- 按模块拆分薄包装：`api/datapoint.api.ts` / `api/access-source.api.ts` / `api/compute.api.ts` / `api/alarm.api.ts` / `api/contract-check.api.ts`。每个文件只导出该模块相关方法。
- 旧 `api/data.api.ts` 拆分后保留过渡期重导出，v2 末期删除。
- 所有响应过 Zod schema 校验。校验失败展示明确错误而非静默吞。
- 错误统一走 `useApiError()`，禁止单点 `try/catch + ElMessage` 散落。

### 数据流约定

- 列表数据：始终走 store。组件不直接调 API。
- 详情数据：进入抽屉时按 ID 拉取详情，缓存在 store 的 `detailById` map 中。
- 写操作：通过 store 的 action 调用 API，成功后 patch store 状态（不重拉全量）。
- 测试取值：复用接入源 / 查询 / MQTT 短时预览能力，前端封装为 store 内的 action。

---

## I-框架. 数据契约检查（顶部动作）

子文档 05 展开详细。主文档只定框架：

- **入口**：NavRail 底部固定按钮，dialog 形态弹出。
- **作用域**：当前项目全量 / 指定模块 / 指定对象（从模块详情触发时）。
- **结果分级**：`passed / warning / pending / failed`。
- **结果定位**：每条结果可点击 LinkChip 跳转到对应模块的对应对象。
- **不作为一级模块**。

---

## J. 开发态预览会话（局部能力）

### 定位

预览会话**不是数据中心的全局基础设施**，仅在接入源工作台内由协议特定场景按需启动：

- MQTT 订阅消息预览、Tag 监视
- Kafka 短时 reader 预览
- HTTP / WebSocket 一次性预览
- Redis 受限 key/value 预览

离开接入源工作台或关闭预览面板，预览会话立即结束。

### 数据点 / 计算 / 报警模块不开预览会话

- 数据点工作区：开发态值通过"测试取值"按需触发一次（详情抽屉里），不订阅。
- 计算单元：调试有自己的同步执行链路，不走 Socket.IO 长连接。
- 报警单元：试算在数据中心计算，不写运行态事件。

### Socket.IO 通道

- 仅在 MQTT 等长连接预览场景使用。
- 复用现有 `preview_socket_server` 路径与鉴权（capability 校验）。
- 会话 ID 由 `usePreviewSession` 拿到后注入 Socket handshake。

### 连接状态徽标

- 在 NavRail 底部条件出现（仅接入源工作台内激活预览时）。
- 4 态：`disconnected / connecting / connected / reconnecting`。
- 颜色 + 文本双重表达；点击弹小卡片：`sessionId` / socket URL / 最近错误 / 重试次数 / 手动重连按钮。

### 降级策略

- 后端 preview 路由未启用：徽标显「未启用」，相关 UI 区块禁用并 tooltip 解释。
- 鉴权失败：徽标显「权限不足」，禁用订阅。
- 心跳超时：自动 `reconnecting`，超过 3 次切到 `disconnected` 状态并停止重试，需用户手动触发。

---

## K. 视觉规范与组件库

### 技术栈不变

- Element Plus + Vue 3 + Vite + Tailwind + Iconify。
- 不引入新组件库 / 新图标库。
- Iconify 图标统一使用 `~icons/tabler` 集（与现有保持一致）。

### `DESIGN.md` 优化清单

当前 `DESIGN.md` 缺失的数据中心特有原子组件规范，v2 推进中向 `DESIGN.md` 补齐以下章节：

1. **Pill Button**（药丸筛选按钮）
   - 形状：圆角 `12px`，高度 28px，padding `0 10px`，font 13px。
   - 状态：default / hover / active（active 用 `--dc-primary-soft` 弱底 + 主色文字）。
   - 内部布局：可选前置 icon（14px） + 文本。
2. **Status Badge**（状态徽标）
   - 5 类：success / warning / danger / info / muted。
   - 表达：文本 + 颜色双重，禁止仅颜色。
   - 尺寸：高度 22px，圆角 `6px`，font 12px。
3. **Icon Action Button**（行内 / 工具栏图标按钮）
   - 已在 `DESIGN.md` 提及，补齐"行内 24×24"小尺寸变体（用于表格操作列）。
4. **Drawer**（抽屉）
   - 宽度 480 默认 / 720 最大，可拖。
   - 头部：标题（title 字号）+ 操作图标群 + 关闭按钮。
   - 内容区：滚动，padding `16px`。
   - 钉住模式：去掉抽屉边阴影，嵌入主区右侧成为第三栏。
   - **优先单页滚动**，Tab 仅在内容分量大且互斥时使用。
5. **Bulk Action Bar**（批量操作浮动条）
   - 列表选中 ≥ 1 行时浮现在底部分页栏上方。
   - 内容：选中计数 + 主要动作 + 清空选择。
6. **Link Chip**（跨模块跳转芯片）
   - 形状：圆角 `8px`，高度 24px，font 12px。
   - 内部：模块图标 + 对象名 + 跳转箭头。
   - hover 加底色，点击切换模块 + 打开目标对象详情。
7. **Empty State**（空状态）
   - `el-empty` 包装，文案 + 可选辅助动作（创建 / 刷新）。
8. **Loading State**（加载状态）
   - 列表用 `v-loading`，详情用骨架屏（按需引入）。

### `PRODUCT.md` 优化清单

在「Design Principles」补一条：

> **6. 模块布局自由，视觉基底统一**：子应用各模块按任务特点自由选择布局形态（表格 / 卡片 / IDE / 双栏 / 多栏等），但颜色、字体、圆角、阴影、原子组件视觉风格遵循统一 token 与组件库。

### i18n / 主题

- 沿用 `datacenter/src/i18n/runtime.ts`，默认中文，保留英文键作为可选。
- 主题（明 / 暗）跟随 IDE 宿主下发，本地不维护切换 UI。

### a11y

- 所有 icon-only 按钮必须有 `aria-label` 或 tooltip（保持现有约定）。
- 状态徽标必须文本 + 颜色双重表达。
- 焦点环使用 `--dc-primary` 弱 ring，不靠 hover 单独表达可点击。

---

## L. 性能（前端侧）

### 大列表策略

- 服务端分页是默认模式（每页 50 / 100 / 200，UI 提供选项）。
- 单页超过 100 行启用虚拟滚动（`vue-virtual-scroller`，按需引入）。

### 输入防抖

- 筛选 / 搜索：300ms。
- 排序切换：立即触发。
- 表单字段：500ms。

### 路由级懒加载

- 四个一级模块按 chunk 分割。
- Monaco 单独 chunk，仅计算单元 / SQL 编辑场景加载。
- DataContractCheckDialog 懒加载，首次打开时按需。

### 首屏指标目标

- 首屏可交互（FCP+TTI）≤ 2s（在 IDE 标签页内打开）。
- 切换一级模块的视觉响应 ≤ 200ms（不含数据加载）。
- 列表筛选反馈 ≤ 500ms。

### Bundle 控制

- 主 chunk ≤ 600KB（gzip）。
- 当前已知大 chunk：index / monaco / ui，针对性拆分。

---

## M. 测试基线（前端侧）

### vitest 覆盖

- 所有 store / composable 的纯逻辑函数。
- 关键模型转换函数（数据点 path 解析、规则模型序列化、契约检查结果归类等）。
- 组件交互测试：抽屉打开 / 关闭 / 钉住、批量操作、筛选条件 URL 同步、规则构建器表单。

### 类型与契约

- API 响应 Zod schema 校验在测试中跑通。
- TypeScript 严格模式，关键模型类型不允许 `any`。

### 手工验收清单

各子文档自带模块级清单。共性项：

- 列表筛选 / 排序 / 分页基线动作。
- 详情抽屉打开 / 钉住 / 关闭。
- 跨模块跳转 LinkChip。
- 数据契约检查弹窗。

### 不纳入

- e2e（Playwright / Cypress）。
- 视觉回归（Chromatic）。
