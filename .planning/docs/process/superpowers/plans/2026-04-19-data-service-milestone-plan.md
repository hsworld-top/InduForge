# 数据服务里程碑计划（仅规划，不开发）

> 适用日期：2026-04-19
> 目标：新建 Go 数据服务承接 `dev_core` 现有数据相关接口，并覆盖设计文档新增能力。
> 当前约束：本阶段只产出计划；未获明确指令前不进入开发实现。

## 1. 已确认决策

1. 新服务定位：独立 Go 数据服务（不经过 `dev_core` 转发数据请求）。
2. 路由分流：由 Nginx 将 `/api/v1/data/**` 直连到数据服务。
3. 鉴权策略：数据服务本地验签 JWT，按 capability 做权限校验。
4. 响应规范：统一 `ApiResponse`（`success`、`errorCode`、`message`、`requestId`、`data`）。
5. 数据库策略：仅 PostgreSQL，不做旧 MySQL 数据迁移。
6. 兼容策略：保持接口语义兼容，库表按 PostgreSQL 最佳实践优化。
7. 计算执行器：JS/Python 都使用子进程沙盒，首版仅允许预置依赖包。
8. 报警表达式：使用 CEL。

## 2. 技能使用声明（强制）

### 2.1 `design-postgres-tables`

- 使用时机：
1. M1 阶段落核心兼容表（`data_connections`、`data_queries`、`data_points`、`data_mqtt_*`）。
2. M2/M3 阶段落新增表（`data_compute_*`、`data_alarm_*`、`data_preview_sessions`）。
- 产出要求：
1. 每张表给出主键、唯一约束、外键、必要索引说明。
2. 明确 `uuid`、`timestamptz`、`jsonb` 使用理由。
3. 明确“查询路径驱动索引”，避免冗余索引。

### 2.2 `design-postgis-tables`

- 使用时机：
1. M3 报警域设计时执行“空间模型评估”。
2. 若报警规则需要地理围栏/区域告警（如 `zone contains point`），则引入 PostGIS 表设计。
- 本期默认结论：
1. 若无空间字段需求，输出“暂不引入 PostGIS 表”的评估结论并记录触发条件。
2. 若引入，必须使用 `geometry/geography + SRID + GiST`，禁止使用 PostgreSQL 内置 `POINT/LINE/POLYGON`。

### 2.3 辅助优化技能：`supabase-postgres-best-practices`

- 使用时机：
1. 每个里程碑完成后进行一次 SQL 与索引复审。
2. 在出现慢查询时执行 `EXPLAIN ANALYZE` 并按规则回调索引方案。

## 3. Go 代码注释规范（强制）

> 因维护者暂不熟悉 Go，以下注释标准在数据服务中必须执行。

1. 所有导出结构体、导出函数、导出接口必须写中文注释，说明“用途 + 输入 + 输出 + 关键边界”。
2. 所有核心流程函数内部，在关键分支前写 1 行中文“目的注释”（不是逐行翻译）。
3. 所有 SQL 查询函数必须写注释说明：
1. 为什么这样写（查询路径）
2. 对应索引是什么
3. 潜在性能风险是什么
4. 所有错误处理分支必须写注释说明“用户可见行为”。
5. 所有并发逻辑（goroutine/channel/mutex）必须写注释说明“并发意图和竞态防护点”。
6. 注释语言统一中文，禁止空泛注释（如“设置变量”“调用函数”）。

## 4. 当前调试入口调整（已完成）

已修改：`datacenter/vite.config.js`

1. `/api/v1/data` 代理到 Go 数据服务（默认 `http://localhost:19099`，可用 `VITE_DATA_SERVICE_URL` 覆盖）。
2. 其余 `/api` 仍代理到 `VITE_API_URL`（通常是 `dev_core`）。

这样可以在开发阶段单独调试新数据服务而不影响其他模块。

## 5. 里程碑计划

### M1：服务骨架 + 数据接口兼容主干

目标：形成“可切流”的最小可用版本。

范围：
1. Go 服务基础设施：配置、日志、requestId、`/health`。
2. PostgreSQL 迁移框架与首批核心表。
3. 兼容接口第一批：
1. `connections`（CRUD、test、tables、execute-sql、table-data、structure）
2. `queries`（CRUD、execute）
3. `datapoints`（list/detail/update/delete/batch/value）

技能应用：
1. `design-postgres-tables`：核心表结构与索引审查。
2. `supabase-postgres-best-practices`：M1 收尾索引复核。

验收标准：
1. datacenter 主流程可联调（不含 compute/alarm）。
2. API 返回全部为 `ApiResponse`。
3. PG 查询全部参数化。

---

### M2：MQTT 兼容域承接

目标：承接 `dev_core` 当前 MQTT 数据域能力。

范围：
1. `mqtt connections/subscriptions/tag-groups/tags/value` 全链路接口。
2. MQTT 连接状态、启停、消息缓存和基础数据点联动。
3. Redis 缓存用于实时值与预览消息。

技能应用：
1. `design-postgres-tables`：`data_mqtt_*` 相关结构与索引校验。
2. `supabase-postgres-best-practices`：热点查询与写路径评估。

验收标准：
1. datacenter 的 MQTT 相关页面可联通。
2. 高频查询无明显慢 SQL（需有 explain 记录）。

---

### M3：compute/alarm/preview 设计能力落地

目标：实现设计文档新增核心能力。

范围：
1. compute：`compute-units` CRUD、run、debug，会话与执行记录。
2. alarm：`alarm-rules`、`alarm-instances/events`、ack/silence、规则试算（CEL）。
3. preview：会话创建、heartbeat、read、subscribe、unsubscribe、delete，30 分钟滑动过期。
4. 执行器：Node/Python 子进程沙盒（预置包，禁在线安装）。

技能应用：
1. `design-postgres-tables`：`data_compute_*`、`data_alarm_*`、`data_preview_sessions` 表设计。
2. `design-postgis-tables`：报警空间模型评估（有地理需求才落地 PostGIS 表）。
3. `supabase-postgres-best-practices`：复杂查询和事件流索引优化。

验收标准：
1. 计算与报警最小闭环跑通。
2. 预览会话续期/过期机制可观测。
3. 沙盒超时与资源限制生效。

---

### M4：发布契约与稳定性收口

目标：让新服务具备可上线替换能力。

范围：
1. 发布编排对象输出契约。
2. 审计日志与基础监控指标。
3. Nginx 灰度切流与回滚预案。
4. 文档补齐（接口清单、环境变量、排障说明）。

技能应用：
1. `supabase-postgres-best-practices`：全量性能复审。
2. `design-postgres-tables`：最终库表评审闭环。

验收标准：
1. 可灰度切流到新数据服务。
2. 出现异常可快速回滚。

## 6. 非目标（本轮明确不做）

1. 旧 MySQL 数据迁移工具。
2. 在线 npm/pip 包下载与安装。
3. 与业务无关的跨模块重构。

## 7. 启动前检查清单（开发开始前）

1. 确认 Nginx 分流配置草案。
2. 确认 JWT 验签公私钥与 capability 字段来源。
3. 确认 PostgreSQL 连接信息与初始化账号。
4. 确认 Redis 可用性与会话 key 前缀。
5. 确认注释规范已纳入代码 review 规则。

