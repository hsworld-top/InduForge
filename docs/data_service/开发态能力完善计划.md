# data_service 开发态能力完善计划

## 背景判断

`data_service` 已经具备平台侧数据域服务的基础骨架，当前包括连接管理、关系库查询、数据点、MQTT 管理、协议 Wave1/Wave2 配置边界、预览会话、Socket.IO 预览通道、计算单元、项目快照与 artifact 输出。

本轮计划暂不推进前端，目标是先把后端开发态能力补齐到可以支撑 `datacenter` 后续真实联调的程度。参考 `docs/datacenter/后续开发计划.md`，后端优先服务以下链路：

1. 调试工程与真实项目上下文联调。
2. 接入源工作区真实数据化。
3. 数据点资产目录增强。
4. 计算单元 SDK 契约落地。
5. 报警规则接口与运行态契约。
6. 数据契约检查后端化。

## 当前后端基线

### 已具备能力

- 服务入口：`cmd/main.go`、`internal/app/server.go` 已完成 HTTP 服务装配、数据库自举、迁移执行、路由注册和资源释放。
- 数据库：`internal/db/migrations` 已覆盖核心表、MQTT、协议配置、预览会话、计算单元、数据点运行态权限和性能索引。
- 统一接口：响应已经收敛到 `code`、`msg`、`data`、`reqId`。
- 权限边界：HTTP 路由统一接入 JWT 与 `project:read` / `project:write` capability。
- 关系库开发态能力：支持连接测试、表列表、表结构、表数据预览、只读 SQL 执行，多方言运行时覆盖 PostgreSQL、MySQL、SQL Server。
- 查询能力：支持 SQL 查询定义 CRUD 与只读执行，支持参数定义与超时。
- MQTT 能力：支持连接、订阅、消息、Tag、Tag 分组、Tag 当前值，创建/更新订阅和 Tag 时已同步生成 `mqtt.*` 数据点。
- 数据点能力：支持列表、详情、按 path 取值、批量状态、批量值、写入默认值、运行态写权限、失效清理。
- 预览链路：支持预览会话、心跳、关闭、Socket.IO 鉴权、MQTT 短时订阅、数据点轮询推送。
- 项目快照：支持项目数据域 snapshot 读写和 Phase 1 artifact 输出。
- 计算能力：支持计算单元创建、JS/Python 手动运行和调试运行，能记录运行结果。

### 主要缺口

- 调试上下文缺口：Socket.IO 当前强依赖合法 token、projectId、previewSessionId；`datacenter/debug` 的本地兜底工程与真实认证上下文之间缺少后端诊断接口和联调口径。
- 接入源聚合缺口：通用 connections 与 MQTT/协议专属配置分散，缺少面向“接入源工作区”的统一详情、统计和最近记录接口。
- 数据点目录缺口：数据点列表只返回定义信息，缺少 `quality`、`lastValue`、`lastUpdatedAt`、来源健康、消费方式等开发态目录字段。
- 数据点同步缺口：MQTT 已做自动数据点同步，但 `db.query -> db.*`、`calc.output -> calc.*` 的自动生成、更新、失效标记尚未完整落地。
- 使用关系缺口：`GetDataPointUsages` 当前返回空列表，无法支撑删除保护、设计中心引用检查、计算输入检查和契约检查。
- 计算 SDK 缺口：计算脚本目前只有 `input -> result`，尚未提供计划中的 `ctx.datapoint.get`、`ctx.sql.query`、`ctx.mqtt.publish` 等开发态 SDK，也未落地副作用治理。
- 计算管理缺口：目前只有创建、运行、调试，缺少列表、详情、更新、删除、启停、运行记录查询、输出数据点同步。
- 报警能力缺口：尚未看到报警规则、规则试算、运行态契约表和接口。
- 契约检查缺口：缺少统一 dry-run 检查接口，无法为发布流程和前端契约检查弹窗提供同一结果。
- 文档漂移风险：部分旧文档示例仍出现 `success/errorCode/message/requestId` 风格，需要后续按统一响应契约校正。

## 开发原则

- 只处理 `data_service` 后端和必要正式文档，不修改前端。
- 保持 `/api/v1` 前缀和统一 JSON 包络。
- 新能力沿用 `handler -> service -> repository -> migration` 分层，不把平台治理和发布部署编排放进 `data_service`。
- 开发态能力可以完整，但运行态长期采集、运行态事件处置和节点本地执行仍放到运行态服务。
- 数据点是核心主线：接入源、查询、MQTT、计算、报警和契约检查都围绕数据点定义与运行态 artifact 连续性展开。
- 本阶段不考虑历史兼容，优先选择清晰、稳定、可验证的模型。

## 阶段 1：联调上下文与预览诊断

目标：让 `datacenter/debug` 与真实项目上下文联调时，后端能明确告诉调用方失败原因，而不是只表现为 Socket.IO `403`。

后端任务：

- 补充预览会话诊断接口：校验 token、projectId、capability、session 状态、Redis 依赖、Socket.IO 参数是否齐全。
- 统一 Socket.IO 握手错误码和错误消息，避免前端只能看到笼统 403。
- 为预览会话返回状态摘要：`sessionId`、`projectId`、`userId`、`status`、`expiredAt`、`socketEnabled`。
- 检查 debug 兜底工程场景：无 token 时仍不放宽后端正式鉴权，但提供明确错误响应，方便前端展示非阻塞状态。

建议涉及文件：

- `data_service/internal/service/preview_session_service.go`
- `data_service/internal/http/handler/preview_handler.go`
- `data_service/internal/http/socket/preview_socket_server.go`
- `data_service/internal/http/router/router.go`
- `data_service/tests/integration/preview_session_test.go`

验收标准：

- token 缺失、项目不匹配、session 过期、Redis 未启用分别返回可区分错误。
- Socket.IO 握手失败原因可在服务日志和响应消息中定位。
- `make -C data_service test` 通过。

## 阶段 2：接入源统一读模型

目标：为接入源工作区提供一个后端聚合视图，前端不需要拼接通用连接、MQTT 详情、数据点数量和最近预览记录。

后端任务：

- 新增接入源摘要接口，按项目返回 `relational`、`mqtt`、`kafka`、`http`、`websocket`、`redis` 的统一列表。
- 每个接入源返回：连接基础信息、类型、状态、脱敏配置摘要、数据点数量、最近测试或预览记录、可用动作。
- 新增接入源详情接口，聚合配置、映射入口、预览入口、关联数据点、最近记录。
- 对敏感配置统一脱敏：密码、token、secret、证书内容等只返回存在性或掩码。
- 保留现有连接与 MQTT 接口作为细粒度管理入口。

建议涉及文件：

- 新增 `data_service/internal/service/access_source_service.go`
- 新增 `data_service/internal/repository/access_source_repository.go`
- 新增 `data_service/internal/http/handler/access_source_handler.go`
- 修改 `data_service/internal/http/router/router.go`
- 新增迁移仅在需要记录测试/预览历史时添加。

建议接口：

- `GET /api/v1/data/projects/{projectId}/access-sources`
- `GET /api/v1/data/projects/{projectId}/access-sources/{sourceId}`
- `GET /api/v1/data/projects/{projectId}/access-sources/{sourceId}/records`

验收标准：

- PostgreSQL、MySQL、SQL Server、MQTT 均能在统一列表中展示。
- MQTT 配置不泄露密码。
- 关联数据点数量与 `data_points.source_id/source_type` 一致。
- 空项目返回空列表和统一分页结构。

## 阶段 3：数据点目录增强与来源连续性

目标：让数据点列表成为设计中心和运行态消费数据的可信入口。

后端任务：

- 扩展数据点列表结果，支持返回 `quality`、`lastValue`、`lastUpdatedAt`、`sourceStatus`、`consumeMode`。
- 批量值接口支持按 path 查询，避免前端列表必须先拿 id。
- 建立数据点来源校验器，统一判断来源是否存在、是否启用、是否可读。
- 补齐 `db.query` 数据点自动同步：创建/更新 SQL 查询时生成或更新 `db.{连接名}.{查询名}`；删除查询时标记失效。
- 连接重命名时同步更新相关数据点 path，保留冲突处理策略。
- 完善 `GetDataPointUsages`：先覆盖计算输入、计算输出、后续报警规则；设计中心引用可等设计侧提供索引后再接入。

建议涉及文件：

- `data_service/internal/service/datapoint_service.go`
- `data_service/internal/service/datapoint_management.go`
- `data_service/internal/service/query_service.go`
- `data_service/internal/repository/datapoint_repository.go`
- `data_service/internal/repository/datapoint_management_repository.go`
- `data_service/internal/http/handler/datapoint_handler.go`

验收标准：

- 保存 SQL 查询自动产生 `db.query` 数据点。
- 删除 SQL 查询后关联数据点变为 `invalid`，不会物理删除。
- 数据点列表能一次返回目录展示所需字段。
- 批量状态和值接口对缺失、失效、来源异常有稳定结果。

## 阶段 4：计算单元管理与 SDK 契约

目标：把计算单元从“能跑脚本”推进到“可管理、可调试、可产出数据点、可生成运行态契约”。

后端任务：

- 补齐计算单元 CRUD：列表、详情、更新、删除、启停。
- 补齐运行记录查询：按单元、状态、时间分页查询。
- 定义开发态 SDK 上下文，首批实现只读和受控副作用：
  - `ctx.datapoint.get(path)`
  - `ctx.sql.query(source, sql, args)`
  - `ctx.mqtt.publish(source, topic, payload)`
  - `ctx.kafka.publish(source, topic, payload)` 可先返回未实现或 mock，按 Phase 1 边界处理。
- 明确副作用治理：参数化 SQL、接入源归属校验、Topic 白名单、超时限制、错误分级。
- 计算单元保存时根据 `outputBindings` 生成或更新 `calc.*` 数据点；删除或移除输出时标记失效。
- 调试结果返回值、日志、副作用、错误分开表达。

建议涉及文件：

- `data_service/internal/service/compute_service.go`
- `data_service/internal/repository/compute_repository.go`
- `data_service/internal/engine/compute/*`
- `data_service/internal/http/handler/compute_handler.go`
- `data_service/internal/http/router/router.go`
- 新增 `data_service/internal/service/compute_context.go`

验收标准：

- 计算单元可列表、编辑、禁用、删除。
- 调试结果能区分 `output`、`logs`、`sideEffects`、`error`。
- 输出数据点自动生成并可通过数据点接口读取。
- 禁用计算单元不能运行。

## 阶段 5：报警规则配置与契约预览

目标：先提供开发态规则配置、目标校验、样本试算和运行态契约，不在数据中心实现运行态事件中心。

后端任务：

- 新增报警规则表：规则基础信息、目标数据点、规则类型、阈值/滞回、严重度、启用状态、契约版本。
- 新增报警规则 CRUD。
- 新增目标数据点校验接口，检查数据点存在性、数据类型、状态、运行态消费权限。
- 新增样本试算接口，输入样本值和时间戳，返回是否触发、清除条件、诊断信息。
- 新增运行态契约预览接口，输出节点侧可执行 JSON。
- 报警自定义表达式先按设计采用 CEL；若 Go 依赖引入延后，第一阶段只实现阈值类规则，CEL 返回明确未启用错误。

建议涉及文件：

- 新增 `data_service/internal/service/alarm_rule_service.go`
- 新增 `data_service/internal/repository/alarm_rule_repository.go`
- 新增 `data_service/internal/http/handler/alarm_rule_handler.go`
- 新增 `data_service/internal/db/migrations/0010_alarm_rules.sql`
- 修改 `data_service/internal/http/router/router.go`

建议接口：

- `GET /api/v1/data/projects/{projectId}/alarm-rules`
- `POST /api/v1/data/projects/{projectId}/alarm-rules`
- `GET /api/v1/data/projects/{projectId}/alarm-rules/{id}`
- `PUT /api/v1/data/projects/{projectId}/alarm-rules/{id}`
- `DELETE /api/v1/data/projects/{projectId}/alarm-rules/{id}`
- `POST /api/v1/data/projects/{projectId}/alarm-rules/{id}/validate-target`
- `POST /api/v1/data/projects/{projectId}/alarm-rules/{id}/test`
- `GET /api/v1/data/projects/{projectId}/alarm-rules/{id}/contract`

验收标准：

- 可以保存报警规则草稿。
- 目标数据点不存在或类型不匹配时返回明确错误。
- 样本试算不写运行态事件。
- 契约预览包含规则版本和目标数据点 path。

## 阶段 6：数据契约检查后端化

目标：为发布前 dry-run 和数据中心全局检查入口提供同一后端检查结果。

后端任务：

- 新增契约检查服务，输入 projectId、检查范围和可选对象 id。
- 检查项覆盖：
  - 数据点定义完整性。
  - 数据点来源存在性。
  - 接入源配置完整性。
  - 查询只读性和可执行性。
  - MQTT 订阅、Tag 与数据点同步关系。
  - 计算单元输入/输出数据点。
  - 报警规则目标和契约可生成性。
  - artifact 输出完整性。
- 返回统一状态：`passed`、`warning`、`pending`、`failed`。
- 每条检查结果包含模块、对象类型、对象 id、标题、详情、建议动作。
- 为发布流程预留 dry-run 调用接口，但不把发布编排写入 `data_service`。

建议涉及文件：

- 新增 `data_service/internal/service/contract_check_service.go`
- 新增 `data_service/internal/http/handler/contract_check_handler.go`
- 修改 `data_service/internal/http/router/router.go`
- 必要时新增检查结果临时记录表；第一版可实时计算不落库。

建议接口：

- `POST /api/v1/data/projects/{projectId}/contract-checks/run`
- `GET /api/v1/data/projects/{projectId}/contract-checks/latest`

验收标准：

- 空项目、正常项目、存在失效数据点的项目都能返回结构化检查结果。
- 失败项能定位到具体对象。
- artifact 生成失败能被契约检查捕获。

## 阶段 7：快照与 artifact v1 收口

目标：让开发态配置能稳定转成运行态数据域契约。

后端任务：

- 扩展 artifact v1，将计算单元输出、报警规则契约、数据点运行态权限纳入稳定结构。
- 对 Phase 1 协议边界保持清晰：Kafka/HTTP/WebSocket/Redis 保留配置与 artifact；无完整 preview/runtime 的协议不得伪装成可运行采集。
- 补齐 snapshot replace 对新对象的导入导出。
- 对 artifact 增加版本、生成时间、校验摘要和 schemaVersion。

建议涉及文件：

- `data_service/internal/repository/project_snapshot_repository.go`
- `data_service/internal/service/project_snapshot_service.go`
- `data_service/tests/integration/project_snapshot_artifact_test.go`
- `data_service/tests/integration/project_snapshot_roundtrip_test.go`

验收标准：

- snapshot roundtrip 不丢失数据点权限、计算输出和报警规则契约。
- artifact 可被契约检查复用。
- Phase 2 协议继续显式返回边界错误，不落入正式 artifact。

## 阶段 8：质量、文档与清理

目标：把后端开发态能力稳定为可持续联调的基线。

后端任务：

- 补齐集成测试，覆盖连接、查询、数据点、MQTT、预览、计算、报警、契约检查和 artifact。
- 统一文档中的响应示例，全部改为 `code/msg/data/reqId`。
- 更新 `docs/统一REST接口规范与清单.md` 的新增接口。
- 若新增错误码，同步更新 `docs/统一错误码枚举表.md`。
- 检查所有数据库访问是否参数化。

验收标准：

- `make -C data_service test` 通过。
- `make -C data_service build` 通过。
- REST 清单覆盖所有新增接口。
- 文档中不再新增旧式响应包络。

## 推荐执行顺序

1. 阶段 1：先解决 debug 和 preview 诊断，降低联调噪音。
2. 阶段 2：接入源统一读模型，让前端工作区可真实取数。
3. 阶段 3：数据点目录增强，打通查询、MQTT、数据点三者连续性。
4. 阶段 4：计算单元管理和 SDK，补齐 `calc.*` 数据点。
5. 阶段 5：报警规则配置和契约预览。
6. 阶段 6：数据契约检查。
7. 阶段 7：snapshot/artifact 收口。
8. 阶段 8：测试、文档、清理。

## 首轮建议切片

首轮不要同时做计算、报警和契约检查。建议先做以下最小闭环：

1. 预览诊断接口和 Socket.IO 错误收口。
2. 接入源统一列表与详情。
3. SQL 查询自动同步 `db.query` 数据点。
4. 数据点列表返回当前值与质量。

这一组完成后，`datacenter` 的 debug、接入源、数据点三个工作区就能进入真实联调；随后再推进计算和报警会更稳。
