# 数据中心后续开发计划

## 当前基线

数据中心 UI 初版已经形成可继续联调的基线，范围包括：

- 一级模块：数据点、接入源、计算单元、报警单元。
- 数据点：三栏工作区、来源筛选、状态筛选、列表选择和详情预览。
- 接入源：概览工作区、接入分类、详情面板，并保留旧连接树、SQL 查询和 MQTT 管理工作台入口。
- 计算单元：任务模式、触发方式、脚本能力说明、安全约束，并保留现有创建、运行、调试面板。
- 报警单元：规则构建、规则清单、运行态契约预览和职责边界说明。
- 全局能力：数据契约检查入口，覆盖发布和运行前的关键数据契约检查项。
- 调试链路：`/datacenter/debug/` 在无有效登录态时可回退到本地调试工程，避免路由启动白屏。

当前已验证：

- `pnpm --dir datacenter build` 通过。
- `pnpm --dir datacenter test -- tests/debug-project.test.ts tests/datacenter-modules.test.ts tests/compute-capabilities.test.ts tests/alarm-rule-model.test.ts tests/data-contract-check.test.ts` 通过。
- `http://localhost:18601/datacenter/debug/` 可渲染新 UI，并可切换四个一级模块。

已知非阻断项：

- 本地调试工程 `test-project` 连接 `data_service` 的 Socket.IO 时可能返回 `403`，需要在联调阶段处理项目、租户、令牌与节点侧权限。
- 构建输出中 `index`、`monaco`、`ui` chunk 偏大，属于既有打包体积问题，可在性能优化阶段处理。

## 阶段 1：提交与基线冻结

目标：把当前数据中心 UI 初版作为后续开发的稳定起点。

交付内容：

- 提交当前 `datacenter` UI 改造与本计划文档。
- 不提交 `data_service`、`dev_core`、既有 docs 脏文件和 `.superpowers/` 临时设计稿。
- 在主分支保留一个可回溯的 UI 基线提交。

验收方式：

- 主分支执行 `pnpm --dir datacenter build` 通过。
- 主分支执行本计划中列出的 datacenter 测试命令通过。
- `/datacenter/debug/` 能正常进入数据点默认页。

## 阶段 2：调试工程与真实上下文联调

目标：让 `/datacenter/debug/` 能稳定解析默认工程，并尽量接近真实工程运行上下文。

任务：

- 梳理 debug 工程优先级：URL `pid/id`、默认工程名 `test`、Storage 缓存、本地 `test-project` 兜底。
- 明确 `tenantId`、`projectId`、access token、refresh token 在调试入口的来源和覆盖规则。
- 处理 Socket.IO `403`：确认是缺少 token、租户、项目权限，还是 `data_service` 对 `test-project` 未授权。
- 给 debug 页面增加非阻塞状态提示：当前工程来源、是否为本地兜底工程、数据服务连接状态。

建议文件：

- `datacenter/src/router/debug-project.ts`
- `datacenter/src/router/index.ts`
- `datacenter/src/utils/socketTest.ts`
- `datacenter/src/composables/usePreviewSession.ts`
- `data_service` 相关鉴权或 Socket.IO 握手代码，只有确认根因在后端时再修改。

验收方式：

- 无 token 时页面不白屏。
- 带 `pid` 时使用 URL 工程。
- 有有效 token 时能解析名称为 `test` 的默认工程。
- Socket.IO 失败时只显示状态，不阻断 UI 使用。

## 阶段 3：接入源工作区真实数据化

目标：把接入源概览从结构页推进到可用管理页。

任务：

- 统一关系库连接模型：`type=relational`，具体库类型从 `relationalConfig.dbType` 或兼容字段读取。
- 接入源卡片展示真实连接状态、库类型、地址、数据库名、MQTT Broker、Topic 和映射数据点数量。
- 详情页各 Tab 落实数据来源：
  - 概览：连接状态、最近测试结果、数据点数量。
  - 配置：脱敏后的连接配置。
  - 解析或映射：SQL 查询、MQTT Tag、后续 OPC UA/设备模板的映射入口。
  - 预览：调用现有表预览、查询执行或 MQTT 样本能力。
  - 数据点：按接入源过滤数据点。
  - 记录：连接测试、预览、映射变更记录。
- 保留旧工作台入口，直到新工作区覆盖旧能力。

建议文件：

- `datacenter/src/components/access-source/AccessSourceWorkspace.vue`
- `datacenter/src/components/access-source/AccessSourceList.vue`
- `datacenter/src/components/access-source/AccessSourceDetail.vue`
- `datacenter/src/composables/useConnection.ts`
- `datacenter/src/api/data.api.ts`

验收方式：

- MySQL、PostgreSQL、SQL Server、MQTT 均能在接入源工作区展示。
- 双击接入源仍能进入旧 SQL/MQTT 工作台。
- 连接为空时有明确的创建入口和说明。

## 阶段 4：数据点资产目录增强

目标：让数据点成为设计中心和运行态消费数据的可信入口。

任务：

- 数据点列表补齐真实字段：path、name、sourceType、dataType、status、quality、lastValue、updatedAt。
- 支持按来源、状态、关键字、消费方式筛选。
- 数据点详情补齐：
  - 来源跳转。
  - 当前值预览。
  - 质量与在线状态。
  - 设计中心引用。
  - 运行态查询或订阅契约。
- 对失效点和来源缺失点提供清晰状态，不在前端静默吞掉。

建议文件：

- `datacenter/src/components/datapoint/DataPointWorkspace.vue`
- `datacenter/src/components/datapoint/DataPointList.vue`
- `datacenter/src/api/data.api.ts`

验收方式：

- 数据点筛选不会触发多余循环请求。
- 点击数据点能展示详情。
- 旧标签页内的数据点列表仍可使用。

## 阶段 5：计算单元 SDK 契约落地

目标：把当前计算单元 UI 中展示的能力变成可执行、可验证的脚本能力契约。

任务：

- 定义计算脚本上下文对象：
  - `ctx.datapoint.get(path)`
  - `ctx.sql.query(source, sql, args)`
  - `ctx.sql.execute(source, sql, args)`
  - `ctx.mqtt.publish(source, topic, payload)`
  - `ctx.kafka.publish(source, topic, payload)`
  - `ctx.http.get/post/put`
  - `ctx.math.*`
  - `ctx.text.*`
  - `ctx.json.*`
- 明确任务模式：
  - 输出数据点。
  - 无输出副作用任务。
  - 混合任务。
- 明确触发方式：
  - 手动或调试。
  - 定时执行。
  - Bool 数据点边沿触发。
  - 数据点变化触发。
- 建立副作用治理：
  - SQL 参数化。
  - 接入源授权。
  - Topic 白名单。
  - HTTP 域名白名单。
  - 超时、重试和幂等键。

建议文件：

- `datacenter/src/components/compute/ComputeWorkspace.vue`
- `datacenter/src/components/compute/computeCapabilities.ts`
- `datacenter/src/components/compute/ComputeUnitPanel.vue`
- `datacenter/src/api/data.api.ts`
- `data_service` 计算单元相关 handler、service、model。

验收方式：

- 无输出任务可以保存、调试并展示副作用执行结果。
- 输出数据点任务能生成或更新 `calc.*` 数据点。
- 调试结果中区分返回值、日志、副作用和错误。

## 阶段 6：报警规则接口与运行态契约

目标：把报警单元从本地规则样例推进到可保存、可校验、可发布的规则配置。

任务：

- 后端新增或确认报警规则接口：
  - 列表。
  - 创建。
  - 更新。
  - 删除。
  - 校验目标数据点。
  - 样本试算。
  - 生成运行态契约。
- 前端规则构建器接入真实接口。
- 保持职责边界：
  - 数据中心只构建规则和预览契约。
  - 运行态节点执行规则、产生事件、处理确认和消音。
- 不在数据中心增加事件时间线、最近事件、ACK 或静默中心。

建议文件：

- `datacenter/src/components/alarm/AlarmWorkspace.vue`
- `datacenter/src/components/alarm/alarmRuleModel.ts`
- `datacenter/src/api/data.api.ts`
- `data_service` 报警规则相关 handler、service、model。

验收方式：

- 可以保存报警规则草稿。
- 可以校验目标数据点是否存在。
- 可以生成节点侧可执行契约 JSON。
- 页面不展示运行态事件管理功能。

## 阶段 7：数据契约检查后端化

目标：把前端本地检查模型升级为发布和运行前的真实检查能力。

任务：

- 定义契约检查接口，输入为 projectId 和检查范围。
- 检查项覆盖：
  - 数据点定义。
  - 接入源连接。
  - 设计中心引用。
  - 运行态查询或订阅契约。
  - 计算单元副作用。
  - 报警规则契约。
- 返回统一状态：`passed`、`warning`、`pending`、`failed`。
- 支持从发布流程调用 dry-run。

建议文件：

- `datacenter/src/components/contract/DataContractCheckDialog.vue`
- `datacenter/src/components/contract/dataContractCheck.ts`
- `datacenter/src/api/data.api.ts`
- `dev_core` 发布流程相关代码。
- `data_service` 契约检查相关代码。

验收方式：

- 弹窗能展示真实检查结果。
- 检查失败能定位到具体模块和对象。
- 发布流程可读取同一检查结果。

## 阶段 8：性能、可访问性与移动端收口

目标：让数据中心在真实项目规模下保持可用。

任务：

- 拆分 Monaco、Element Plus 大 chunk，评估动态导入。
- 数据点列表增加分页、虚拟列表或服务端筛选。
- 接入源和数据点请求增加加载态、错误态和重试入口。
- 检查键盘焦点、表单标签、按钮可访问名称。
- 保持移动端可查看关键内容，不隐藏全局数据契约检查入口。

验收方式：

- 生产构建无新增编译警告。
- 大量数据点下列表不卡死。
- 1280px、1024px、768px 宽度下页面布局可用。

## 推荐执行顺序

1. 先冻结当前 UI 基线。
2. 优先修复 debug 工程和 Socket.IO 真实上下文。
3. 先接入源、再数据点、再计算、再报警。
4. 数据契约检查最后后端化，因为它依赖前面模块的真实数据。
5. 每个阶段独立提交，提交前至少执行 `pnpm --dir datacenter build` 和相关测试。
