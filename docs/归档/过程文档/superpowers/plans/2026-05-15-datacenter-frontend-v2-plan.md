# 数据中心前端 v2 开发实现计划

> 日期：2026-05-15  
> 范围：仅 `datacenter/` 前端。后端 API、`data_service`、PG/TSDB 存储不在本轮。  
> 输入规格：`2026-05-15-datacenter-frontend-v2-design.md` 与 5 份子文档。  
> 技术约束：不替换 Vue 3 / Vite / Tailwind / Element Plus，不引入新组件库或新图标库。图标继续使用现有 Iconify / `~icons/tabler`。时间统一用 `dayjs`。

## 0. 执行原则

1. 先收敛共性基础设施，再按模块推进。模块阶段不各自发明视觉、分页、错误处理、抽屉、草稿和接口封装。
2. 所有后端未到位能力都显式展示「能力未启用」或禁用态，不用旧工作台兜底暴露给用户。
3. 不设置独立的性能阶段。服务端分页、防抖、懒加载、Monaco 分块和大列表策略嵌入到各模块阶段内。
4. 旧代码只按覆盖范围迁移或删除，不顺手重构无关目录。
5. 正式入口依赖 IDE bootstrap。本地调试入口使用 `/debug/...`，缺上下文时显式提示，不白屏。

## 1. 阶段总览

| 编号  | 阶段                                         | 依赖          | 可并行性         |
| ----- | -------------------------------------------- | ------------- | ---------------- |
| P0    | DESIGN.md / PRODUCT.md 优化                  | 无            | 可与 F1、F2 并行 |
| F1    | 共性基础设施：视觉 token 与原子组件          | 无            | 先行             |
| F2    | 共性基础设施：状态、API、Zod、composables    | F1 可部分并行 | 先行             |
| F3    | 共性基础设施：壳层、路由、URL 状态、草稿保护 | F1、F2        | 先行             |
| M0    | 废弃与迁移专项                               | F1、F2、F3    | 贯穿模块阶段     |
| D1-D3 | 数据点工作区                                 | F1-F3         | D1 后可独立验收  |
| A1-A3 | 接入源工作区                                 | F1-F3、M0     | A1 后可独立验收  |
| C1-C3 | 计算单元工作区                               | F1-F3、M0     | C1 后可独立验收  |
| L1-L3 | 报警单元工作区                               | F1-F3         | L1 后可独立验收  |
| K1-K3 | 数据契约检查                                 | F1-F3         | K1 后可独立验收  |

## P0. DESIGN.md / PRODUCT.md 优化

**目标**  
把 v2 需要的视觉规则沉淀到根文档。明确「模块布局自由，视觉基底统一」，并补齐数据中心原子组件规范。

**涉及文件 / 模块**

- `DESIGN.md`
- `PRODUCT.md`
- 对齐参考：`datacenter/src/assets/styles/main.css`
- 对齐参考：`datacenter/src/components/datapoint/DataPointList.vue`

**前置依赖阶段**  
无。可与 F1、F2 并行。

**验证方式**

- 文档评审：确认包含 Pill Button、Status Badge、Icon Action Button、Drawer、Bulk Action Bar、Link Chip、Empty State、Loading State。
- 文档评审：确认 `PRODUCT.md` 的 Design Principles 增加「模块布局自由，视觉基底统一」。
- 本阶段不改代码，不要求 `pnpm` 验证。

**与后端能力的对应**  
无后端依赖。

**风险点**

- `DESIGN.md` 现有圆角为 `12px/18px`，`datacenter` 当前 token 多为 `6px/8px`。需要在文档里明确数据中心 v2 的局部 token 以现有 `--dc-*` 为准，避免强行全局改视觉。

## F1. 共性基础设施：视觉 token 与原子组件

**目标**  
先落统一视觉底座。把数据点工作区已有样式抽成最小原子组件和 class 约定，供五个模块复用。

**涉及文件 / 模块**

- `datacenter/src/assets/styles/main.css`
- `datacenter/src/components/shared/PillButton.vue`
- `datacenter/src/components/shared/StatusBadge.vue`
- `datacenter/src/components/shared/LinkChip.vue`
- `datacenter/src/components/shared/DcDrawer.vue`
- `datacenter/src/components/shared/BulkActionBar.vue`
- `datacenter/src/components/shared/EmptyState.vue`
- `datacenter/src/components/shared/StatusIndicator.vue`
- `datacenter/src/components/layout/DataCenterShell.vue`
- `datacenter/src/components/layout/DataCenterNavRail.vue`

**前置依赖阶段**  
无。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：在 `/debug/datapoint`、`/debug/access-source` 打开页面，确认按钮、状态徽标、抽屉、空状态不破坏现有布局。
- 手工验收：所有 icon-only 按钮有 `aria-label` 或 tooltip。

**与后端能力的对应**  
无直接后端依赖。组件需支持 disabled / loading / capability-disabled 状态，供后续未启用能力降级复用。

**风险点**

- 当前 `AccessSourceList.vue`、`AlarmWorkspace.vue`、`ComputeWorkspace.vue` 内有大量局部样式。F1 只抽公共视觉，不重写业务布局。
- `DcDrawer` 需基于 Element Plus 抽屉或现有 DOM 封装，不引入新浮层库。

## F2. 共性基础设施：状态、API、Zod、composables

**目标**  
拆分状态和 API 边界，让组件只使用 store action。引入 Zod 做响应校验，统一分页、草稿、抽屉、确认框和错误展示。

**涉及文件 / 模块**

- `datacenter/src/api/data.api.ts`，过渡期只做重导出和兼容入口
- `datacenter/src/api/datapoint.api.ts`
- `datacenter/src/api/access-source.api.ts`
- `datacenter/src/api/compute.api.ts`
- `datacenter/src/api/alarm.api.ts`
- `datacenter/src/api/contract-check.api.ts`
- `datacenter/src/api/schemas/*.schema.ts`
- `datacenter/src/stores/project.store.ts`
- `datacenter/src/stores/data-catalog.store.ts`
- `datacenter/src/stores/access-source.store.ts`
- `datacenter/src/stores/compute.store.ts`
- `datacenter/src/stores/alarm.store.ts`
- `datacenter/src/stores/contract-check.store.ts`
- `datacenter/src/stores/preview-session.store.ts`
- `datacenter/src/stores/ui-prefs.store.ts`
- `datacenter/src/composables/useDraft.ts`
- `datacenter/src/composables/useServerPagination.ts`
- `datacenter/src/composables/useDrawer.ts`
- `datacenter/src/composables/useConfirm.ts`
- `datacenter/src/composables/useApiError.ts`
- `datacenter/src/utils/request.ts`

**前置依赖阶段**  
F1 可部分并行。API 与 store 可先建薄包装，不等待组件完成。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 单测覆盖：schema 解析成功、schema 失败转为明确错误、`useServerPagination` 参数序列化、`useDraft` 按 `projectId + module + objectId` 隔离。
- 手工验收：模拟 `HTTP 2xx + code != 0`，确认统一错误展示包含 `msg` 和 `reqId`。

**与后端能力的对应**

- 先复用现有接口：connections、queries、datapoints、preview sessions、MQTT、compute create/run/debug。
- 新接口未到位时，API 层返回标准「能力未启用」错误对象，组件统一降级。
- Zod 是数据校验依赖，不是组件库。若当前依赖缺失，需在该阶段通过 `pnpm --dir datacenter add zod` 加入。

**风险点**

- 现有 `data.api.ts` 是大文件且带 `@ts-nocheck`。拆分时必须保留过渡重导出，避免一次性改崩旧组件。
- schema 过严会因后端字段缺失导致列表不可用。第一版 schema 对可选字段要保守，关键字段缺失才失败。

## F3. 共性基础设施：壳层、路由、URL 状态、草稿保护

**目标**  
收敛 v2 路由和会话模型。模块、对象、详情 tab、筛选、分页、排序都能从 URL 恢复；未保存草稿在模块切换和关闭前受保护。

**涉及文件 / 模块**

- `datacenter/src/router/route-config.ts`
- `datacenter/src/router/index.ts`
- `datacenter/src/views/DataCenterNew.vue`
- `datacenter/src/config/datacenterModules.ts`
- `datacenter/src/runtime/host-bootstrap.ts`
- `datacenter/src/runtime/runtime-message-handler.ts`
- `datacenter/src/components/layout/DataCenterShell.vue`
- `datacenter/src/components/layout/DataCenterNavRail.vue`
- `datacenter/src/components/shared/LinkChip.vue`
- `datacenter/src/composables/useDraft.ts`
- `datacenter/src/composables/useConfirm.ts`
- `datacenter/src/stores/project.store.ts`
- `datacenter/src/stores/ui-prefs.store.ts`

**前置依赖阶段**  
F1、F2。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：`/debug/datapoint`、`/debug/access-source/:objectId`、`/debug/compute/:objectId/io`、`/debug/alarm/:objectId/test` 能落到正确模块。
- 手工验收：刷新后恢复模块、筛选 query、选中对象和抽屉 tab。
- 手工验收：有草稿时切换模块，出现丢弃 / 保留为草稿 / 取消三选一。

**与后端能力的对应**

- 正式入口需要 IDE bootstrap 下发 `projectId`、`tenantId`、token、theme、locale、capability。
- `/debug` 无 token 时走本地兜底工程或显式提示，不调用需要鉴权的写接口。
- capability 缺失时按最小权限降级，相关按钮显示「能力未启用」。

**风险点**

- 当前模块 ID 是 `datapoints/access-sources/compute-units/alarm-units`，目标路由是 `datapoint/access-source/compute/alarm`。需要提供兼容映射，避免旧链接直接白屏。
- token 不进 URL 和 localStorage。刷新正式入口依赖宿主重发 bootstrap，前端不能自行持久化 token。

## M0. 废弃与迁移专项

**目标**  
明确旧代码迁移顺序、保留期、删除时机和覆盖前兜底方式。迁移期间源码可以短期保留，但 v2 用户入口不再依赖旧工作台作为兜底。

**涉及文件 / 模块**

- `datacenter/src/views/DataCenterNew.vue`
- `datacenter/src/components/database/SqlQueryWorkbench.vue`
- `datacenter/src/components/mqtt/MqttWorkbench.vue`
- `datacenter/src/components/compute/ComputeWorkspace.vue`
- `datacenter/src/components/compute/ComputeUnitPanel.vue`
- `datacenter/src/components/connection/ConnectionContextMenu.vue`
- `datacenter/src/components/connection/TableContextMenu.vue`
- `datacenter/src/components/connection/QueryContextMenu.vue`
- `datacenter/src/components/connection/ConnectionList.vue`
- `datacenter/src/components/database/mysql/MysqlQueryEditor.vue`
- `datacenter/src/components/database/postgres/PostgresQueryEditor.vue`
- `datacenter/src/components/database/sqlserver/SqlServerQueryEditor.vue`
- `datacenter/src/components/mqtt/MqttSubscriptionList.vue`
- `datacenter/src/components/mqtt/MqttMessageViewer.vue`
- `datacenter/src/components/mqtt/MqttTagList.vue`
- `datacenter/src/components/mqtt/MqttTagMonitor.vue`

**前置依赖阶段**  
F1、F2、F3。删除动作依赖对应模块阶段完成。

**迁移顺序、保留期、删除时机、兜底方式**

| 旧代码                                        | 迁移顺序                                                                                                                   | 保留期                                       | 删除时机                                                              | 覆盖前兜底方式                                                              |
| --------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------- | --------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| `DataCenterNew.vue` 内 `el-tabs` 多标签页系统 | 先由 F3 接管一级模块路由；A1/A3、D1、C1、L1 分别替换原 tab 内容；最后移除 `tabs/activeTabId/tabCounter` 和所有 tab handler | 保留到 A3、C1、L1 都能从 NavRail 与 URL 进入 | A3 完成 SQL/MQTT 二级工作台、C1 完成计算骨架、L1 完成报警骨架后删除   | v2 入口直接进入新模块；未覆盖能力显示「能力未启用」，不打开旧 tab           |
| `SqlQueryWorkbench` 独立工作台                | A3 中拆可复用能力到 `access-source/workbench/sql`：表浏览、SQL 编辑、保存查询、执行结果                                    | 源文件保留到 A3 验证通过                     | A3 验收后删除原独立入口；可复用子组件迁入新目录后再删旧壳             | 新 SQL 二级工作台未完成前，卡片「打开」进入「能力未启用」占位，不跳旧工作台 |
| `MqttWorkbench` 独立工作台                    | A3 中拆订阅、Tag、消息预览到 `access-source/workbench/mqtt`，预览会话局部化                                                | 源文件保留到 A3 验证通过                     | A3 验收后删除原独立入口；保留复用后的订阅/Tag 子组件                  | 新 MQTT 工作台未完成前，显示协议工作台占位和「能力未启用」                  |
| legacy `ComputeWorkspace` 介绍式三栏布局      | C1 直接替换为双栏 IDE 骨架；原能力说明数据迁到文档或删除                                                                   | 保留到 C1 骨架验收通过                       | C1 后删除介绍式布局                                                   | 计算 CRUD 不足时，双栏 IDE 显禁用态与空状态，不回退介绍页                   |
| legacy `ComputeUnitPanel` 简单表单            | C1/C2 将创建、运行、调试能力迁入新 store 和编辑器底部调试面板                                                              | 保留到 C2 保存 / 调试链路验收通过            | C2 后删除或仅保留测试用例中的 mock，不保留 UI 入口                    | 未覆盖时在新编辑器显示「创建 / 更新未启用」禁用按钮                         |
| 旧右键菜单交互                                | A1/A2 将接入源、表、查询、订阅的关键操作改为显式按钮；C1 只允许计算树节点保留小菜单例外                                    | 保留到 A2/A3 对应显式入口完成                | A3 后删除 connection/table/query 旧菜单；计算树菜单另按 C1 新实现保留 | 覆盖前按钮禁用或隐藏高级动作，不依赖右键作为唯一入口                        |

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 静态检查：`DataCenterNew.vue` 不再导入 `SqlQueryWorkbench`、`MqttWorkbench`、`ConnectionContextMenu`、`TableContextMenu`、`QueryContextMenu`。
- 手工验收：接入源卡片所有关键动作可见；双击和右键不是完成核心任务的唯一方式。

**与后端能力的对应**

- 旧代码删除不要求新后端全部到位。后端缺口由新 UI 降级承接。
- SQL / MQTT 现有接口可迁入新二级工作台；Kafka / HTTP / WebSocket / Redis / 工业协议按 capability 降级。

**风险点**

- `DataCenterNew.vue` 当前承担过多连接、查询、MQTT 和 tab handler。迁移时要按模块切断引用，避免一次性大改。
- 删除旧工作台前必须确认没有测试或路由仍导入旧组件。

## D1. 数据点工作区：骨架与列表

**目标**  
实现 v2 数据点单栏列表：工具栏、固定 10 列、服务端分页、URL 同步。移除当前值、质量、写权限摘要、创建时间等非 v2 字段。

**涉及文件 / 模块**

- `datacenter/src/components/datapoint/DataPointWorkspace.vue`
- `datacenter/src/components/datapoint/DataPointList.vue`
- `datacenter/src/stores/data-catalog.store.ts`
- `datacenter/src/api/datapoint.api.ts`
- `datacenter/src/api/schemas/datapoint.schema.ts`
- `datacenter/src/composables/useServerPagination.ts`
- `datacenter/src/components/shared/PillButton.vue`
- `datacenter/src/components/shared/StatusBadge.vue`
- `datacenter/src/components/shared/BulkActionBar.vue`
- `datacenter/src/router/route-config.ts`

**前置依赖阶段**  
F1、F2、F3。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：`/debug/datapoint?q=温度&status=active` 搜索、状态、来源类型、标签、排序、分页可用且刷新后保持。
- 手工验收：500 条以上数据点时分页稳定，搜索 300ms 防抖，单页 50 / 100 / 200 可切换。

**与后端能力的对应**

- 需现有 `GET datapoints` 支持 projectId、分页、搜索、状态、sourceType、标签、排序。
- 若后端只返回旧字段，前端 schema 做兼容映射；缺失的引用数、来源失效原因显示 `-` 或隐藏。

**风险点**

- 现有返回结构可能是 `data.datapoints`，规范要求 `data.list` 和 `data.pagination`。API 层需要集中适配，不让组件判断多种结构。
- 标签筛选和来源类型动态枚举可能依赖列表聚合字段，缺失时只展示「全部」。

## D2. 数据点工作区：详情抽屉与跨模块跳转

**目标**  
实现单页滚动详情抽屉。支持复制 path、标签编辑入口、来源 LinkChip、引用 LinkChip、运行态权限编辑入口和 URL 对象恢复。

**涉及文件 / 模块**

- `datacenter/src/components/datapoint/DataPointWorkspace.vue`
- `datacenter/src/components/datapoint/DataPointList.vue`
- `datacenter/src/components/datapoint/DataPointDetailDrawer.vue`
- `datacenter/src/components/datapoint/DataPointTagDialog.vue`
- `datacenter/src/components/datapoint/RuntimePermissionDialog.vue`
- `datacenter/src/components/shared/DcDrawer.vue`
- `datacenter/src/components/shared/LinkChip.vue`
- `datacenter/src/stores/data-catalog.store.ts`
- `datacenter/src/stores/ui-prefs.store.ts`
- `datacenter/src/api/datapoint.api.ts`

**前置依赖阶段**  
D1。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：点击行打开 `/debug/datapoint/:objectId`，刷新后抽屉仍打开。
- 手工验收：抽屉 480px 默认宽度、可拖到 720px、可钉住为旁路面板。
- 手工验收：LinkChip 能切到接入源 / 计算 / 报警并打开目标对象。

**与后端能力的对应**

- 详情读取可复用列表详情或新增 `GET datapoints/:id`。若后端无详情接口，先用列表项字段渲染可用区块。
- `GetDataPointUsages` 未到位时，引用区块不展示。
- 运行态权限更新复用现有 `updateDatapointRuntimePermissions`；后端不支持时编辑按钮禁用。

**风险点**

- 抽屉内编辑标签和权限容易扩展成复杂治理。v2 只做入口和必要表单，不加历史记录。
- LinkChip 要避免循环 push 相同路由导致重复加载。

## D3. 数据点工作区：高级能力与降级路径

**目标**  
完成测试取值、批量打标签、批量清理失效、来源失效原因和能力未启用降级。数据点工作区不做实时订阅。

**涉及文件 / 模块**

- `datacenter/src/components/datapoint/DataPointDetailDrawer.vue`
- `datacenter/src/components/datapoint/DataPointList.vue`
- `datacenter/src/components/datapoint/DataPointTagDialog.vue`
- `datacenter/src/stores/data-catalog.store.ts`
- `datacenter/src/api/datapoint.api.ts`
- `datacenter/src/api/access-source.api.ts`
- `datacenter/src/api/compute.api.ts`
- `datacenter/src/composables/useConfirm.ts`
- `datacenter/src/composables/useApiError.ts`

**前置依赖阶段**  
D2。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：选中多行后浮出批量操作条，清空选择不改变分页。
- 手工验收：`db.query`、`mqtt.tag`、`calc.output` 的测试取值调用对应 API；`alarm.state` 禁用。
- 手工验收：后端 preview 不支持时按钮禁用并显示 tooltip。

**与后端能力的对应**

- `db.query` 测试取值依赖 SQL 执行或保存查询执行接口。
- `mqtt.tag` 依赖最新缓存值接口。
- `calc.output` 依赖 compute run/debug。
- Phase 2 协议和 `alarm.state` 默认「能力未启用」或不支持。

**风险点**

- 测试取值跨多个来源类型，不能把协议分支散落在组件里，应由 store action 封装。
- 批量清理失效要二次确认，并且只允许 invalid 行。

## A1. 接入源工作区：骨架与卡片列表

**目标**  
实现接入源卡片网格、搜索、类型 pill、状态 pill、刷新、新增连接入口。卡片空白点击打开详情，按钮点击进入二级工作台。

**涉及文件 / 模块**

- `datacenter/src/components/access-source/AccessSourceWorkspace.vue`
- `datacenter/src/components/access-source/AccessSourceList.vue`
- `datacenter/src/components/access-source/AccessSourceCard.vue`
- `datacenter/src/stores/access-source.store.ts`
- `datacenter/src/api/access-source.api.ts`
- `datacenter/src/api/schemas/access-source.schema.ts`
- `datacenter/src/config/connectionTypes.ts`
- `datacenter/src/components/shared/PillButton.vue`
- `datacenter/src/components/shared/StatusBadge.vue`
- `datacenter/src/components/shared/EmptyState.vue`

**前置依赖阶段**  
F1、F2、F3、M0。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：`/debug/access-source?type=mqtt&status=connected` 搜索和筛选刷新后保持。
- 手工验收：卡片显示协议图标、状态、名称、脱敏端点、数据点数量；Phase 2 协议有「仅配置」角标。
- 手工验收：右键不是必要入口，卡片可见按钮覆盖打开、编辑、新增。

**与后端能力的对应**

- 需现有 `GET connections`。数据点数量、状态、地址脱敏字段缺失时显示 `-` 或本地脱敏。
- 工业协议仅配置，进入工作台后显示只读配置或「能力未启用」。

**风险点**

- 当前 `AccessSourceWorkspace.vue` 内有本地查询数据点数量逻辑。A1 只保留必要兼容，后续应收敛到 store。
- 卡片网格 500 到 1000 个对象必须依赖服务端分页或分批加载，不能一次性无限渲染。

## A2. 接入源工作区：详情抽屉与新增 / 编辑

**目标**  
实现单页滚动详情抽屉、连通性测试、关联数据点跳转、编辑配置、删除确认和单一 ConnectionDialog 两步流程。

**涉及文件 / 模块**

- `datacenter/src/components/access-source/AccessSourceDetailDrawer.vue`
- `datacenter/src/components/dialogs/ConnectionDialog.vue`
- `datacenter/src/components/connection/forms/*.vue`
- `datacenter/src/stores/access-source.store.ts`
- `datacenter/src/api/access-source.api.ts`
- `datacenter/src/components/shared/DcDrawer.vue`
- `datacenter/src/components/shared/LinkChip.vue`
- `datacenter/src/composables/useConfirm.ts`
- `datacenter/src/composables/useApiError.ts`

**前置依赖阶段**  
A1。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：`/debug/access-source/:objectId` 刷新后详情抽屉恢复。
- 手工验收：新增连接为「选协议 -> 填配置」两步，编辑从卡片和抽屉都进入第二步。
- 手工验收：测试连接成功展示响应耗时和时间；失败展示简短错误和可展开详情。
- 手工验收：关联数据点按钮跳转到 `/debug/datapoint?sourceId=...`。

**与后端能力的对应**

- CRUD 复用现有 connections 接口和协议 config 创建接口。
- 连通性测试优先复用 `testConnection` 或协议握手接口。不支持协议按钮禁用。
- 删除保护依赖后端引用检查；未到位时只做前端二次确认。

**风险点**

- 现有 `ConnectionDialog` 可能混合旧类型字段。A2 只整理到 v2 需要的两步，不增加导入模板。
- 删除接入源可能影响数据点，后端引用检查未到位前风险要在确认文案中说明。

## A3. 接入源工作区：二级工作台、预览会话与降级

**目标**  
替代旧 SQL / MQTT 独立工作台。二级工作台按协议分支渲染：关系库、MQTT、Kafka、HTTP、WebSocket、Redis、工业协议只读配置。

**涉及文件 / 模块**

- `datacenter/src/components/access-source/AccessSourceWorkbench.vue`
- `datacenter/src/components/access-source/workbench/SqlWorkbench.vue`
- `datacenter/src/components/access-source/workbench/MqttWorkbenchPanel.vue`
- `datacenter/src/components/access-source/workbench/ProtocolPreviewPanel.vue`
- `datacenter/src/components/access-source/workbench/ReadOnlyConfigPanel.vue`
- `datacenter/src/components/database/mysql/MysqlTableList.vue`
- `datacenter/src/components/database/mysql/MysqlQueryEditor.vue`
- `datacenter/src/components/database/postgres/PostgresTableList.vue`
- `datacenter/src/components/database/postgres/PostgresQueryEditor.vue`
- `datacenter/src/components/database/sqlserver/SqlServerTableList.vue`
- `datacenter/src/components/database/sqlserver/SqlServerQueryEditor.vue`
- `datacenter/src/components/mqtt/MqttSubscriptionList.vue`
- `datacenter/src/components/mqtt/MqttMessageViewer.vue`
- `datacenter/src/components/mqtt/MqttTagList.vue`
- `datacenter/src/components/mqtt/MqttTagMonitor.vue`
- `datacenter/src/stores/preview-session.store.ts`
- `datacenter/src/composables/usePreviewSession.ts`
- `datacenter/src/composables/useMqttSocket.ts`
- `datacenter/src/api/access-source.api.ts`

**前置依赖阶段**  
A2、M0。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：`/debug/access-source/:objectId/workbench` 能进入，返回后保持原筛选。
- 手工验收：关系库可表浏览、SQL 编辑、执行、保存查询。
- 手工验收：MQTT 可订阅管理、Tag 管理、消息预览；离开工作台关闭 preview session。
- 手工验收：NavRail 底部预览会话徽标只在预览激活时出现。

**与后端能力的对应**

- 关系库复用表结构、SQL 执行、queries 接口。
- MQTT 复用 subscriptions、tags、preview session、Socket.IO。
- Kafka / HTTP / WebSocket / Redis 预览能力未到位时显示「能力未启用」。
- 工业协议 Phase 2 只读配置，不启用预览。

**风险点**

- Monaco 和 SQL 编辑器要路由级懒加载，避免拖大首屏。
- Socket 重连状态必须局部化，离开接入源工作台立即停止。
- A3 是删除旧 `SqlQueryWorkbench` / `MqttWorkbench` 的前置里程碑，不能只做占位就删除旧源码。

## C1. 计算单元工作区：双栏 IDE 骨架与树列表

**目标**  
替换 legacy 三栏介绍页，落地左侧文件夹树 + 右侧编辑器主区骨架。支持搜索、新建单元、新建文件夹、选中、URL 恢复和输出数据点路径展示。

**涉及文件 / 模块**

- `datacenter/src/components/compute/ComputeWorkspace.vue`
- `datacenter/src/components/compute/ComputeTree.vue`
- `datacenter/src/components/compute/ComputeEditorShell.vue`
- `datacenter/src/components/compute/CreateComputeUnitDialog.vue`
- `datacenter/src/stores/compute.store.ts`
- `datacenter/src/api/compute.api.ts`
- `datacenter/src/api/schemas/compute.schema.ts`
- `datacenter/src/components/shared/StatusBadge.vue`
- `datacenter/src/components/shared/EmptyState.vue`
- `datacenter/src/composables/useDraft.ts`

**前置依赖阶段**  
F1、F2、F3、M0。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：`/debug/compute/:objectId` 选中计算单元，刷新后仍选中。
- 手工验收：左侧 280px，可折叠；空文件夹有空状态；输出路径按文件夹 + 名称形成 `calc.*`。
- 手工验收：后端 list / folder 未到位时显示「能力未启用」，不回退旧介绍页。

**与后端能力的对应**

- 现有后端有 create/run/debug，缺 list/update/delete/enable/folder。
- list 缺失时可显示空状态和禁用新建后的持久列表能力；不造完整本地假数据作为正式能力。

**风险点**

- 文件夹树拖拽和右键小菜单是计算模块唯一例外。实现时不要复用旧接入源右键菜单。
- 输出 `calc.*` 路径同步依赖后端，前端只能预览路径，不能承诺已生成数据点。

## C2. 计算单元工作区：编辑器详情、依赖、触发和输入

**目标**  
完成编辑器头条、名称 inline 编辑、语言只读、启停、保存、删除、依赖勾选、触发方式和输入映射底部 Tab。

**涉及文件 / 模块**

- `datacenter/src/components/compute/ComputeEditorShell.vue`
- `datacenter/src/components/compute/ComputeEditorHeader.vue`
- `datacenter/src/components/compute/ComputeDependencyPanel.vue`
- `datacenter/src/components/compute/ComputeTriggerPanel.vue`
- `datacenter/src/components/compute/ComputeInputPanel.vue`
- `datacenter/src/components/compute/DataPointPicker.vue`
- `datacenter/src/components/MonacoEditor.vue`
- `datacenter/src/stores/compute.store.ts`
- `datacenter/src/stores/data-catalog.store.ts`
- `datacenter/src/api/compute.api.ts`
- `datacenter/src/api/datapoint.api.ts`

**前置依赖阶段**  
C1。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：改名或移动时输出路径实时预览；保存成功后脏状态清除。
- 手工验收：触发方式只允许手动、定时、数据点变化，动态显示对应字段。
- 手工验收：输入映射可搜索数据点并添加 / 删除。
- 手工验收：Monaco 只在计算模块加载。

**与后端能力的对应**

- 依赖白名单接口未到位时显示「无可用库」并禁用刷新。
- `dependencies` 字段未到位时，本地草稿暂存，保存时忽略该字段并提示。
- update/delete/enable 未到位时按钮禁用。

**风险点**

- 依赖校验涉及脚本静态分析，前端第一版只展示后端返回错误，不自造完整解析器。
- Python / JavaScript 语言创建后只读，避免保存时切语言导致后端模型不一致。

## C3. 计算单元工作区：调试、副作用治理与降级路径

**目标**  
完成底部调试 Tab、dry-run / 真实执行二次确认、副作用治理条件区、删除引用检查、跨模块跳转。

**涉及文件 / 模块**

- `datacenter/src/components/compute/ComputeDebugPanel.vue`
- `datacenter/src/components/compute/ComputeSideEffectPanel.vue`
- `datacenter/src/components/compute/ComputeEditorShell.vue`
- `datacenter/src/stores/compute.store.ts`
- `datacenter/src/stores/data-catalog.store.ts`
- `datacenter/src/api/compute.api.ts`
- `datacenter/src/api/datapoint.api.ts`
- `datacenter/src/components/shared/LinkChip.vue`
- `datacenter/src/composables/useConfirm.ts`
- `datacenter/src/composables/useApiError.ts`

**前置依赖阶段**  
C2。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：调试结果分返回值、日志、副作用、错误四区展示。
- 手工验收：真实执行必须二次确认；dry-run 不支持时显示禁用。
- 手工验收：数据点详情跳转到计算单元，报警目标为 `calc.*` 时可跳回计算单元。

**与后端能力的对应**

- debug 复用现有 `debugComputeUnit`。
- dry-run 标志、结构化日志、副作用列表、引用检查未到位时按区块降级。
- 输出数据点自动同步依赖后端；前端只展示路径和同步状态。

**风险点**

- 调试真实执行可能产生副作用。默认 dry-run，真实执行必须清楚标识。
- 副作用治理只在检测到 `ctx.sql/ctx.mqtt/ctx.http` 调用时出现，避免常态表单过重。

## L1. 报警单元工作区：骨架与规则列表

**目标**  
实现左侧规则列表 + 右侧编辑器主区骨架。支持搜索、新建规则入口、选中、启停状态显示和后端未启用的顶部提示。

**涉及文件 / 模块**

- `datacenter/src/components/alarm/AlarmWorkspace.vue`
- `datacenter/src/components/alarm/AlarmRuleList.vue`
- `datacenter/src/components/alarm/AlarmEditorShell.vue`
- `datacenter/src/components/alarm/CreateAlarmRuleDialog.vue`
- `datacenter/src/components/alarm/alarmRuleModel.ts`
- `datacenter/src/stores/alarm.store.ts`
- `datacenter/src/api/alarm.api.ts`
- `datacenter/src/api/schemas/alarm.schema.ts`
- `datacenter/src/components/shared/StatusBadge.vue`
- `datacenter/src/components/shared/EmptyState.vue`

**前置依赖阶段**  
F1、F2、F3。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：`/debug/alarm/:objectId` 能恢复选中规则。
- 手工验收：后端报警 CRUD 未启用时，列表和编辑器显示禁用态，不白屏。
- 手工验收：1k 规则量级用搜索 + 服务端分页或分批加载，不做分组列表。

**与后端能力的对应**

- 报警规则 CRUD 当前按未实现处理。
- 若后端无列表，前端可显示空状态和「能力未启用」，不使用本地样例假装真实数据。

**风险点**

- 现有 `AlarmWorkspace.vue` 是样例规则构建器。L1 要避免把样例数据误当后端数据。
- 规则分组被规格砍掉，不迁移旧分组侧栏。

## L2. 报警单元工作区：规则编辑表单与底部面板

**目标**  
完成编辑器头条、目标数据点选择、规则类型动态字段、严重度、启停、保存、删除、抑制策略和消息模板折叠区。

**涉及文件 / 模块**

- `datacenter/src/components/alarm/AlarmEditorHeader.vue`
- `datacenter/src/components/alarm/AlarmRuleForm.vue`
- `datacenter/src/components/alarm/AlarmSuppressionPanel.vue`
- `datacenter/src/components/alarm/AlarmMessageTemplatePanel.vue`
- `datacenter/src/components/alarm/DataPointPicker.vue`
- `datacenter/src/stores/alarm.store.ts`
- `datacenter/src/stores/data-catalog.store.ts`
- `datacenter/src/api/alarm.api.ts`
- `datacenter/src/api/datapoint.api.ts`
- `datacenter/src/composables/useDraft.ts`
- `datacenter/src/composables/useConfirm.ts`

**前置依赖阶段**  
L1。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：H / L / HH / LL / 大偏差 / 小偏差 / 变化率 / CEL 切换时字段正确显示。
- 手工验收：目标数据点选择器可搜索，跳转按钮打开数据点详情。
- 手工验收：消息模板默认值合理，抑制策略默认收起。

**与后端能力的对应**

- CRUD、目标数据点校验未到位时保存按钮禁用或显示「能力未启用」。
- CEL 后端依赖未到位时 CEL 选项禁用。

**风险点**

- 表单模型复杂，必须以 `alarmRuleModel.ts` 集中序列化，避免每个组件拼不同 payload。
- 默认回差、延时和模板提示要清楚，但不实现通知渠道和运行态事件。

## L3. 报警单元工作区：试算、契约预览与降级路径

**目标**  
完成底部试算 Tab、契约预览 Tab、复制 JSON、CEL 语法校验入口和跨模块 LinkChip。

**涉及文件 / 模块**

- `datacenter/src/components/alarm/AlarmTestPanel.vue`
- `datacenter/src/components/alarm/AlarmContractPanel.vue`
- `datacenter/src/components/alarm/AlarmEditorShell.vue`
- `datacenter/src/components/MonacoEditor.vue`
- `datacenter/src/stores/alarm.store.ts`
- `datacenter/src/api/alarm.api.ts`
- `datacenter/src/api/datapoint.api.ts`
- `datacenter/src/components/shared/LinkChip.vue`
- `datacenter/src/composables/useApiError.ts`

**前置依赖阶段**  
L2。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：手输样本和最近一次预览值两种试算来源可切换。
- 手工验收：试算结果显示触发 / 不触发 / 输入不足、推导和消息模板渲染。
- 手工验收：契约 JSON 只读可复制；接口未启用时显示灰态说明。

**与后端能力的对应**

- 样本试算接口未到位时，试算 Tab 显示「能力未启用」。
- 契约预览依赖 `GET alarm-rules/:id/contract`，未到位时禁用。
- 最近一次预览值复用数据点测试取值能力，未支持来源禁用。

**风险点**

- 试算不写运行态事件，不能调用运行态报警链路。
- CEL 编辑器要懒加载，避免影响报警列表首屏。

## K1. 数据契约检查：入口、弹窗骨架与作用域

**目标**  
把数据契约检查固定到 NavRail 底部。弹窗单页滚动，支持当前项目全量、指定模块、指定对象三种作用域。

**涉及文件 / 模块**

- `datacenter/src/components/layout/DataCenterNavRail.vue`
- `datacenter/src/views/DataCenterNew.vue`
- `datacenter/src/components/contract/DataContractCheckDialog.vue`
- `datacenter/src/components/contract/dataContractCheck.ts`
- `datacenter/src/stores/contract-check.store.ts`
- `datacenter/src/api/contract-check.api.ts`
- `datacenter/src/components/shared/StatusBadge.vue`
- `datacenter/src/components/shared/EmptyState.vue`

**前置依赖阶段**  
F1、F2、F3。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：任意模块点击 NavRail 底部按钮能打开弹窗。
- 手工验收：三种作用域切换不丢表单状态；关闭后焦点回到入口按钮。

**与后端能力的对应**

- 后端检查接口未实现时，弹窗顶部显示「能力未启用」。
- 可保留前端 mock 结果用于骨架展示，但必须标明不是后端检查结果。

**风险点**

- 数据契约检查不是一级模块，不能加入 NavRail 主模块列表。
- 弹窗不需要历史记录、导出报告和检查项配置。

## K2. 数据契约检查：执行、最近结果与结果跳转

**目标**  
接入开始检查、复用最近一次结果、汇总、分类折叠、四态渲染和 LinkChip 定位。

**涉及文件 / 模块**

- `datacenter/src/components/contract/DataContractCheckDialog.vue`
- `datacenter/src/components/contract/ContractCheckResultList.vue`
- `datacenter/src/stores/contract-check.store.ts`
- `datacenter/src/api/contract-check.api.ts`
- `datacenter/src/api/schemas/contract-check.schema.ts`
- `datacenter/src/components/shared/LinkChip.vue`
- `datacenter/src/composables/useApiError.ts`

**前置依赖阶段**  
K1。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：开始检查 loading，成功后显示 passed / warning / pending / failed 四态。
- 手工验收：复用最近一次按钮在无结果时禁用，有结果时显示时间。
- 手工验收：点击结果跳转关闭弹窗，切换模块并打开目标对象。

**与后端能力的对应**

- 需要 `POST contract-checks/run` 和 `GET contract-checks/latest`。
- 接口未到位时开始检查按钮禁用或显示本地 mock，复用按钮禁用。
- 分类规则缺失时对应分类灰态展示。

**风险点**

- LinkChip 跳转要和 F3 路由状态一致，不能在弹窗内维护独立选中态。
- 检查结果来自后端缓存，前端不要自行存历史。

## K3. 数据契约检查：模块上下文触发与降级完善

**目标**  
允许从数据点、接入源、计算单元、报警详情中触发「检查当前对象」，并统一能力未启用、权限不足、分类未支持的展示。

**涉及文件 / 模块**

- `datacenter/src/components/datapoint/DataPointDetailDrawer.vue`
- `datacenter/src/components/access-source/AccessSourceDetailDrawer.vue`
- `datacenter/src/components/compute/ComputeEditorHeader.vue`
- `datacenter/src/components/alarm/AlarmEditorHeader.vue`
- `datacenter/src/components/contract/DataContractCheckDialog.vue`
- `datacenter/src/stores/contract-check.store.ts`
- `datacenter/src/components/shared/StatusBadge.vue`
- `datacenter/src/composables/useApiError.ts`

**前置依赖阶段**  
K2、D2、A2、C2、L2。

**验证方式**

- `pnpm --dir datacenter build`
- `pnpm --dir datacenter test`
- 手工验收：从四个模块详情触发检查时，弹窗自动选择指定对象作用域。
- 手工验收：能力未启用、权限不足、分类未支持三种状态文案区分清楚。

**与后端能力的对应**

- 指定对象检查依赖后端支持 scope object。
- 后端只支持项目全量时，前端提示「当前接口仅支持全量检查」，对象级按钮禁用。

**风险点**

- 模块详情触发检查会引入交叉依赖，建议只传 scope 参数给 contract store，不让模块互相 import。
- 权限不足不能伪装成未启用，需要使用统一错误展示。

## 2. 阶段间里程碑

| 里程碑               | 完成条件                                                           |
| -------------------- | ------------------------------------------------------------------ |
| M1：v2 壳层可用      | F1、F2、F3 完成，四个一级模块可通过 URL 进入，刷新可恢复           |
| M2：数据点可独立交付 | D1、D2、D3 完成，数据点列表、详情、批量、测试取值和 LinkChip 可用  |
| M3：接入源替换旧入口 | A1、A2、A3 完成，旧 SQL/MQTT 独立工作台不再导入                    |
| M4：计算 IDE 成型    | C1、C2、C3 完成，legacy ComputeWorkspace / ComputeUnitPanel 可删除 |
| M5：报警开发态闭环   | L1、L2、L3 完成，后端未启用时降级路径完整                          |
| M6：契约检查闭环     | K1、K2、K3 完成，NavRail 与模块详情入口都可触发检查                |

## 3. 统一验收清单

1. `pnpm --dir datacenter build` 通过。
2. `pnpm --dir datacenter test` 通过。
3. 本地 dev server 跑通：
   - `/debug/datapoint`
   - `/debug/datapoint/:objectId`
   - `/debug/access-source`
   - `/debug/access-source/:objectId`
   - `/debug/access-source/:objectId/workbench`
   - `/debug/compute/:objectId`
   - `/debug/compute/:objectId/io`
   - `/debug/alarm/:objectId`
   - `/debug/alarm/:objectId/test`
4. 手工确认 NavRail 底部「数据契约检查」在任意模块可用。
5. 手工确认 URL 恢复、抽屉钉住、LinkChip 跨模块跳转、草稿保护和统一错误展示。
6. 静态确认未引入新组件库 / 新图标库，时间格式继续用 `dayjs`。

## 4. 摘要

- 总阶段数：20 个。包含 P0 文档阶段、F1-F3 共性基础设施、M0 废弃与迁移、D/A/C/L/K 各 3 个模块阶段。
- 关键里程碑：M1 v2 壳层可用；M2 数据点可独立交付；M3 接入源替换旧入口；M4 计算 IDE 成型；M5 报警开发态闭环；M6 契约检查闭环。
- 最早可独立验证的阶段：若包含文档，P0 可最早独立验收；若只看代码，F1 可最早独立验证，验证方式为 `pnpm --dir datacenter build`、`pnpm --dir datacenter test` 和 `/debug/datapoint`、`/debug/access-source` 的视觉基线检查。
