# 数据中心 Modbus 寄存器建模工作台实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 Modbus 接入源工作台完善为寄存器变量建模入口，支持寄存器组、多从站变量、数据点自动同步、建模校验、辅助预览和运行态读取预估。

**架构：** `data_service` 新增 Modbus 寄存器组与寄存器变量建模表，服务层负责 CRUD、批量导入、地址换算、数据点同步、校验和读取计划预估。`datacenter` 新增 Modbus 专用三栏工作台，与 MQTT / OPC UA 保持家族化界面语言；真实采集、重连、HA 和发布仍由运行态及外部发布入口负责。

**技术栈：** Go、pgx、PostgreSQL migration、Vue 3、Element Plus、TypeScript、Vite、Go test、pnpm。

---

## 文件结构

- 创建：`data_service/internal/db/migrations/0028_modbus_register_modeling.sql`
  新增 `data_modbus_register_groups` 与 `data_modbus_registers`。
- 创建：`data_service/internal/db/migrations/0028_modbus_register_modeling_down.sql`
  回滚 Modbus 建模表。
- 创建：`data_service/internal/repository/modbus_modeling_repository.go`
  参数化实现寄存器组、变量、数据点联查和快照读取所需 SQL。
- 创建：`data_service/internal/service/modbus_modeling_service.go`
  封装地址换算、类型寄存器数量推导、数据点同步、建模校验、读取计划预估。
- 创建：`data_service/internal/http/handler/modbus_modeling_handler.go`
  暴露寄存器组、变量、批量导入、校验、预览和读取预估接口。
- 修改：`data_service/internal/http/router/router.go`
  挂载 `/api/v1/data/projects/{projectId}/modbus/{connectionId}` 建模路由。
- 修改：`data_service/internal/app/server.go`
  初始化 Modbus 建模 repository、service、handler。
- 修改：`data_service/internal/repository/project_snapshot_repository.go`
  将 Modbus 寄存器变量和读取计划纳入运行契约 artifact。
- 修改：`datacenter/src/api/data.api.ts`
  增加 Modbus 建模 API 类型与方法。
- 创建：`datacenter/src/components/modbus/types.ts`
  前端 Modbus 工作台共享类型。
- 创建：`datacenter/src/components/modbus/ModbusGroupTree.vue`
  左侧寄存器组树。
- 创建：`datacenter/src/components/modbus/ModbusRegisterTable.vue`
  中间变量表格。
- 创建：`datacenter/src/components/modbus/ModbusInspectorPanel.vue`
  右侧配置面板与读取预估摘要。
- 创建：`datacenter/src/components/modbus/ModbusGroupDialog.vue`
  新建 / 编辑寄存器组弹窗。
- 创建：`datacenter/src/components/modbus/ModbusRegisterDialog.vue`
  新建 / 编辑变量弹窗。
- 创建：`datacenter/src/components/modbus/ModbusImportDialog.vue`
  粘贴表格 / 地址段生成导入弹窗。
- 创建：`datacenter/src/components/modbus/ModbusPreviewDialog.vue`
  辅助预览弹窗。
- 创建：`datacenter/src/components/modbus/ModbusValidationDrawer.vue`
  建模校验抽屉。
- 创建：`datacenter/src/components/modbus/ModbusReadPlanDialog.vue`
  运行态读取预估弹窗。
- 创建：`datacenter/src/components/access-source/workbench/ModbusWorkbenchPanel.vue`
  Modbus 三栏工作台容器。
- 修改：`datacenter/src/components/access-source/AccessSourceWorkbench.vue`
  将 `type='modbus'` 路由到新工作台。

---

### 任务 1：新增 Modbus 建模数据库表

- [ ] 创建迁移，字段覆盖寄存器组、变量、地址、解析规则、轮询周期和状态。
- [ ] 创建回滚迁移，按依赖顺序删除索引和表。
- [ ] 验证：`go test ./...` 至少能通过迁移编译加载。

### 任务 2：实现后端 repository 与 service

- [ ] 实现寄存器组 CRUD。
- [ ] 实现变量 CRUD、批量导入、数据点联查。
- [ ] 实现用户地址到协议地址换算和数据类型到寄存器数量推导。
- [ ] 保存 / 更新变量时同步 `source_type='modbus.register'` 数据点。
- [ ] 删除变量时标记对应数据点失效。
- [ ] 实现建模校验、辅助预览和读取计划预估。
- [ ] 验证：`go test ./...` 编译通过。

### 任务 3：挂载后端路由并补充运行契约

- [ ] 在 server 初始化 Modbus 建模依赖。
- [ ] 在 router 增加 Modbus 建模路由。
- [ ] 在 project snapshot 增加 Modbus 寄存器变量快照。
- [ ] artifact 的 `protocols.modbus` 输出 `registers` 与 `readPlans`。
- [ ] 验证：`go test ./...`。

### 任务 4：实现前端 API 与 Modbus 工作台组件

- [ ] 在 `data.api.ts` 增加类型和 API 方法。
- [ ] 新增 Modbus 三栏工作台及组树、变量表、右侧面板。
- [ ] 新增寄存器组、变量、导入、预览、校验、读取预估弹窗。
- [ ] 修改 AccessSourceWorkbench 路由。
- [ ] 验证：`pnpm --filter datacenter typecheck`。

### 任务 5：UI 打磨与最终验证

- [ ] 让 Modbus 与 MQTT / OPC UA 保持一致的工作台语言：顶部概览、左树、中表、右配置面板、低饱和边框和紧凑操作区。
- [ ] 检查按钮文案，避免“覆盖变量”等误导词，统一使用“本次读取变量数”。
- [ ] 运行：`pnpm --filter datacenter typecheck`、`pnpm --filter datacenter build`、`go test ./...`。
