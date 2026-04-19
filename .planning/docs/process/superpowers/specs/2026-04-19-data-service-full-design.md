# Data Service 全量设计规格（评审稿）

> 日期：2026-04-19  
> 范围：开发中心数据服务 `data_service`（仓库根目录新建）  
> 状态：仅设计规格与里程碑，不进入业务实现开发

## 1. 目标、范围与非目标

### 1.1 目标

在仓库根目录新建独立 Go 服务 `data_service/`，承接当前 `dev_core` 数据相关接口，并覆盖以下设计目标：

1. 数据接入协议矩阵（两波落地）。
2. 开发态 preview 会话（30 分钟滑动过期）。
3. compute 引擎（JS/Python 子进程沙盒）。
4. alarm 引擎（含 CEL 规则、状态机、事件流）。
5. runtime_inapp 通知链路与发布契约输出。

### 1.2 已冻结约束

1. 数据请求不经过 `dev_core`，由 Nginx 将 `/api/v1/data/**` 直连 `data_service`。
2. 响应统一 `ApiResponse(success,errorCode,message,requestId,data)`。
3. 数据库仅 PostgreSQL，不做旧 MySQL 迁移工具。
4. 鉴权采用本地 JWT 验签 + capability 校验。
5. OPC DA 本期仅做接口契约与数据模型，不做 Windows 适配器实现。
6. 计算执行器采用 Node/Python 子进程沙盒，首版仅允许预置依赖。
7. 报警自定义规则语言使用 CEL。
8. 通知中心本期仅上线 `runtime_inapp` 渠道。

### 1.3 非目标

1. 不做旧 MySQL 迁移脚本与迁移工具。
2. 不做 npm/pip 在线下载与安装。
3. 不做 Node 侧完整改造，仅做契约联调验证。
4. `design-postgis-tables` 本期只做文档层可行性声明，不单列编码里程碑。

## 2. 架构决策（冻结）

### 2.1 请求分流

1. `/api/v1/data/**` -> `data_service`。
2. 其余 `/api/**` -> `dev_core`。
3. 开发态已调整 `datacenter/vite.config.js`：
1. `/api/v1/data` 默认代理 `http://localhost:19099`（可由 `VITE_DATA_SERVICE_URL` 覆盖）。
2. 其余 `/api` 继续代理 `VITE_API_URL`。

### 2.2 鉴权与权限

1. Go 服务校验平台 JWT（本地验签）。
2. 读取 `tenant/project/capabilities` 做细粒度授权。
3. 不维护角色主数据，角色入口仍在工程卡片“成员与权限”。

### 2.3 数据与契约

1. PostgreSQL 为唯一业务库。
2. 接口语义兼容现有 `dev_core` 数据接口。
3. 通过 DTO 保持 API 语义稳定，允许底层表结构演进。
4. 所有数据库访问必须参数化。

## 3. 技能使用声明（强制）

### 3.1 `design-postgres-tables`（强制）

适用里程碑：

1. M3 核心兼容表与迁移框架。
2. M6 MQTT 全域表。
3. M7/M8 协议矩阵表。
4. M9 preview 会话表。
5. M10 compute 表。
6. M11 alarm/notify 表。

每次应用必须产出：

1. 主键、唯一约束、外键、索引清单。
2. `uuid/timestamptz/jsonb` 字段类型理由。
3. 查询路径到索引路径的映射说明。

### 3.2 `design-postgis-tables`（本期声明引用）

1. 仅用于报警空间模型可行性评估文档。
2. 本期不单列开发任务。
3. 后续若触发地理围栏能力，再独立立项。

### 3.3 `supabase-postgres-best-practices`（每里程碑复审）

1. 每个里程碑收尾执行 SQL/索引复审。
2. 检查索引命中、外键索引完整性、组合索引/部分索引优化空间。

## 4. Go 注释规范（强制执行）

> 维护者当前不熟悉 Go，注释以“可读性优先”。

1. 所有导出结构体、导出函数、导出接口必须中文注释。
2. 关键业务流程前必须写“目的说明”注释。
3. SQL 函数必须注明“查询意图 + 索引依赖 + 性能风险”。
4. 错误分支必须注明“用户可见行为”。
5. 并发逻辑必须注明“并发意图 + 竞态防护点”。
6. 禁止空泛注释（如“设置变量”“调用函数”）。

## 5. 全量能力清单（需被里程碑覆盖）

### 5.1 协议矩阵

第一波（M6-M7）：

1. 关系库增强（含 TDengine 关系入口增强准备项）。
2. MQTT（在 M6 先完成核心承接，M7 做矩阵补齐与联调验证）。
3. Kafka（仅消费）。
4. HTTP/REST。
5. WebSocket。
6. Redis（受控能力）。

第二波（M8）：

1. OPC UA。
2. S7。
3. Modbus TCP。
4. Modbus RTU。
5. TDengine 增强项（查询/预览/发布编排链路）。
6. OPC DA（仅契约与模型）。

### 5.2 统一数据契约

所有协议统一输出字段：

1. `pointId`
2. `path`
3. `value`
4. `quality`（`good|bad|uncertain` + 协议原始码）
5. `timestamp`
6. `serverTimestamp`
7. `source`
8. `connectionId`
9. `sessionId`（开发态）或运行态上下文标识

### 5.3 compute 关键规则

1. `triggerType` 支持 `manual/timer/datapoint_change`。
2. 调度语义支持项目时区、错过窗口补偿（`skip/catchup_once/catchup_all`）、重叠执行策略（`skip_if_running/queue_one/parallel`）。
3. 输入模型支持变量映射与路径读取并存。
4. 输出固定写入计算结果数据点（`calc.*`）。
5. 沙盒限制：禁网、禁任意文件写入、禁未授权模块导入、资源上限受控。

### 5.4 alarm 关键规则

1. 严重度固定 5 级（5 紧急 -> 1 提示）。
2. `severity` 与 `escalation_level` 分离。
3. 规则类型覆盖 HH/H/L/LL、大偏差、小偏差、变化率、CEL 自定义表达式。
4. 阈值类规则支持滞回（触发阈值/清除阈值）。
5. 状态机覆盖 `RAISED/CLEARED/ACKED/SILENCED/UNSILENCED`。
6. 历史事件不做物理删除（审计保留）。

### 5.5 runtime_inapp 关键规则

1. 通知链路：报警事件 -> 策略命中 -> `runtime_inapp` 投递任务 -> 节点收件箱 -> 前端 ACK 回执。
2. 幂等键建议：`eventId + policyId + channelId + targetUser`。
3. 节点离线支持补投，恢复后按序投递。
4. 通知失败不阻塞报警主流程。

### 5.6 数据保留与可观测基线

1. 默认保留策略：
1. `data_compute_unit_runs` 30 天（可配置）。
2. `data_alarm_events` 180 天（可配置）。
3. `data_alarm_notify_deliveries` 90 天（可配置）。
4. runtime_inapp 历史消息 180 天（可配置）。
2. 可观测指标最小集：
1. 计算维度：执行次数、成功率、超时率、平均耗时。
2. 报警维度：触发率、活动实例数、确认率、消音率。
3. 通知维度：成功率、延迟、重试率、失败原因分布。
4. 调试维度：会话数、平均会话时长、断点执行次数。

## 6. 里程碑计划（能力域编排，11 个）

> 完成定义：每个里程碑必须“可联调通过”，不是仅代码完成。

### M1 基础骨架与调试分流

交付：

1. `data_service/` 基础工程结构。
2. 配置、日志、`/health`、`requestId`。
3. 统一 `ApiResponse`、`AppError`、`ErrorCodes` 适配层。
4. datacenter -> data_service 开发态分流可用。

最小测试包：

1. 启动测试。
2. 健康检查测试。
3. requestId 透传测试。

### M2 鉴权与能力校验基线

交付：

1. JWT 本地验签中间件。
2. capability 校验中间件。
3. 租户/项目隔离校验。

最小测试包：

1. token 有效/无效。
2. capability 允许/拒绝。
3. 跨项目越权拦截。

### M3 PostgreSQL 迁移框架与核心兼容表

交付：

1. 迁移框架（up/down）。
2. 核心表：
1. `data_connections`
2. `data_relational_configs`
3. `data_queries`
4. `data_points`
3. 索引基线与唯一约束策略。

最小测试包：

1. migration up/down。
2. 索引存在性。
3. 核心 CRUD。

### M4 接口承接 I（connections）

交付接口：

1. `connections` CRUD。
2. `test/tables/execute-sql/table-data/structure`。

最小测试包：

1. happy path。
2. 参数错误。
3. SQL 注入回归。

### M5 接口承接 II（queries + datapoints）

交付接口：

1. `queries` CRUD + execute。
2. `datapoints` list/detail/update/delete/batch/value/usages。

最小测试包：

1. 查询执行与数据点联动。
2. 分页过滤。
3. 批量删除边界。

### M6 接口承接 III（MQTT 全域）

交付接口：

1. `mqtt connection/subscription/tag-group/tag/value/start/stop/status` 全域。

交付表：

1. `data_mqtt_configs`
2. `data_mqtt_subscriptions`
3. `data_mqtt_tag_groups`
4. `data_mqtt_tags`
5. `data_mqtt_tag_values`（按实现需要）

最小测试包：

1. 启停状态。
2. 订阅与值读取。
3. 缓存与清理。

### M7 协议矩阵第一波

目标协议：

1. Kafka（仅消费）。
2. HTTP/REST。
3. WebSocket。
4. Redis（受控读写 + Pub/Sub 查看）。
5. MQTT（矩阵联调补齐，不重复建设已有 M6 核心能力）。
6. 关系库增强（含 TDengine 入口增强准备项）。

交付接口域：

1. `/api/v1/data/projects/:projectId/kafka/...`
2. `/api/v1/data/projects/:projectId/http/...`
3. `/api/v1/data/projects/:projectId/websocket/...`
4. `/api/v1/data/projects/:projectId/redis/...`

交付表：

1. `data_kafka_configs` + `data_kafka_topics`
2. `data_http_configs` + `data_http_mappings`
3. `data_websocket_configs` + `data_websocket_topics`
4. `data_redis_configs` + `data_redis_keys`

最小测试包：

1. 每协议至少 1 条 E2E 样例。
2. Kafka 消费链路样例（不含生产）。
3. MQTT 与关系库能力在矩阵下的联调回归样例。

### M8 协议矩阵第二波

目标协议：

1. OPC UA。
2. S7。
3. Modbus TCP/RTU。
4. TDengine 增强接入。
5. OPC DA（仅契约与模型）。

交付接口域：

1. `/api/v1/data/projects/:projectId/opcua/...`
2. `/api/v1/data/projects/:projectId/s7/...`
3. `/api/v1/data/projects/:projectId/modbus/...`
4. `/api/v1/data/projects/:projectId/opcda/...`（契约占位）

交付表：

1. `data_opcua_configs` + `data_opcua_nodes`
2. `data_s7_configs` + `data_s7_tags`
3. `data_modbus_configs` + `data_modbus_points`
4. `data_opcda_configs` + `data_opcda_items`（模型占位）

最小测试包：

1. 每协议至少 1 条联调样例。
2. OPC DA 契约与模型校验样例。

### M9 Preview 会话域

交付接口：

1. `create/heartbeat/read/subscribe/unsubscribe/delete`。

交付存储：

1. Redis Key：
1. `preview:session:{sessionId}`
2. `preview:session:{sessionId}:connections`
3. `preview:session:{sessionId}:subscriptions`
2. 表：`data_preview_sessions`（可选 `data_preview_session_logs`）。

交付前端（最小页）：

1. preview 会话最小页面（创建、续期、订阅状态、销毁）。

规则：

1. 30 分钟滑动过期。

最小测试包：

1. 创建。
2. 续期。
3. 过期。
4. 回收。

### M10 Compute 引擎

交付接口：

1. `compute-units` CRUD。
2. `/compute-units/:id/run`。
3. `/compute-units/:id/debug`。

交付表：

1. `data_compute_units`
2. `data_compute_unit_inputs`
3. `data_compute_unit_outputs`
4. `data_compute_unit_runs`

执行器：

1. Node 子进程沙盒。
2. Python 子进程沙盒。
3. 仅预置包。

交付前端（最小页）：

1. compute 最小页面（列表、编辑、运行、结果查看）。

最小测试包：

1. JS 执行。
2. Python 执行。
3. 超时与资源限制。
4. 调度策略（时区/重叠/补偿）至少 1 组样例。
5. 调试会话超时自动释放样例。

### M11 Alarm + runtime_inapp + 发布契约

交付接口：

1. `alarm-rules` CRUD + `/alarm-rules/:id/test`。
2. `alarm-instances`、`alarm-events` 查询。
3. `/alarm-events/:id/ack`、`/alarm-events/:id/silence`。
4. `alarm-channels`（仅 `runtime_inapp`）+ `/alarm-channels/:id/test`。
5. `alarm-notify-policies` CRUD。
6. `alarm-notify-deliveries` 查询 + `/alarm-notify-deliveries/:id/replay`。
7. 发布契约导出接口。

交付表：

1. `data_alarm_rules`
2. `data_alarm_instances`
3. `data_alarm_events`
4. `data_alarm_channels`
5. `data_alarm_notify_policies`
6. `data_alarm_notify_deliveries`

交付前端（最小页）：

1. alarm 最小页面（规则、实例、事件、确认/消音）。
2. runtime_inapp 最小通知页（列表、未读、已读 ACK）。

能力：

1. CEL 规则编译与执行治理（编译校验、变量校验、超时保护）。
2. 报警状态机闭环。
3. runtime_inapp 投递闭环与幂等去重。
4. Node 侧契约联调验证。
5. 数据保留策略配置与清理任务基线。

最小测试包：

1. 触发/清除/确认/消音流程。
2. 通知入箱与幂等。
3. 发布契约一致性校验。

## 7. 依赖关系与并行策略

### 7.1 主阻塞链

1. M1 -> M2 -> M3 -> M4 -> M5 -> M6 -> M7 -> M8 -> M11

### 7.2 并行链

1. M9 可在 M5 后并行推进。
2. M10 依赖 M5 + M9。
3. M11 依赖 M8 + M10。

## 8. 里程碑治理（入口/出口）

每个里程碑统一要求：

1. 入口：前序里程碑全部绿灯。
2. 出口：
1. 代码 + 测试通过。
2. datacenter 最小联调通过。
3. 接口文档更新。
4. 变更风险清单更新。
5. SQL/索引复审记录归档。

## 9. 设计覆盖映射（对齐两份原始设计文档）

1. `data-development-service-design.md` 的协议矩阵、统一数据契约、preview 机制、发布契约：由 M7/M8/M9/M11 覆盖。
2. `compute-alarm-design.md` 的 compute 模型、调度语义、沙盒约束：由 M10 覆盖。
3. `compute-alarm-design.md` 的 alarm 状态机、事件语义、通知策略、runtime_inapp 链路：由 M11 覆盖。
4. 工程级 RBAC 透传与 capability 模式：由 M2 与全链路鉴权约束覆盖。

## 10. 关键风险与应对

1. 接口兼容偏差：以契约测试锁定字段与状态语义。
2. 执行器安全风险：默认禁网、资源限额、白名单包。
3. 协议扩展复杂度：两波推进 + 每协议最小样例验收。
4. 数据库性能回退：每里程碑执行 SQL/索引复审。
5. 报警噪声风险：滞回、on-delay/off-delay、去重与聚合策略。

## 11. 评审输出模板

每次里程碑评审按三类输出：

1. 必改项（阻塞下阶段）。
2. 建议项（后续优化）。
3. 暂缓项（登记技术债）。

## 12. 下一步（仍处于规划阶段）

1. 本规格确认后，进入 implementation plan 拆解（task 级）。
2. 仅在你明确下达“开始开发”指令后进入编码阶段。
