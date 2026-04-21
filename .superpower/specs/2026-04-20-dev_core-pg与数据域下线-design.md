# dev_core 切换 PostgreSQL 与数据域能力下线设计稿

## 1. 背景与目标

本次改造包含两条必须同时成立的主线：

1. `dev_core` 从当前 MySQL + Sequelize 运行模式切换为 PostgreSQL。
2. `dev_core` 彻底下线历史遗留的数据域本地实现，平台侧数据域能力只保留 `data_service` 一个正式拥有者。

用户补充的环境约束如下：

- 根目录 `.env` 后续统一改为 PostgreSQL 连接信息。
- 项目内所有“需要使用 PostgreSQL 配置”的模块，都统一使用同一套主机、端口、用户名、密码：
  - Host: `172.21.242.174`
  - Port: `5432`
  - User: `postgres`
  - Password: `postgres`
- 迁移过程中如果当前库表结构存在可确认的性能问题、冗余字段、缺失索引或不合理约束，允许结合 PostgreSQL 最佳实践做清理与优化，目标是同时保证 `dev_core` 与 `data_service` 的高性能。
- `dev_core` 切到 PostgreSQL 后，可以继续使用 Sequelize，不要求同步替换 ORM 框架。

本设计稿采用的默认假设如下：

- 统一的是 PostgreSQL 实例连接参数，而不是强制把所有模块合并到同一个 database。
- `dev_core` 与 `data_service` 默认仍保留各自逻辑 database 名称，避免把控制面与数据域物理混库：
  - `dev_core` 默认 database 继续使用 `tenant_management`
  - `data_service` 默认 database 继续使用 `data_service`
- 若后续需要合库，再单独做数据库边界调整，不并入本轮。

## 2. 当前现状

### 2.1 `dev_core`

当前 `dev_core` 仍然明显绑定 MySQL：

- `src/config/database.js` 使用 `mysql2/promise` 和 `Sequelize(dialect=mysql)`。
- `scripts/init-database.js` 基于 MySQL 语义创建 database、执行 `database/init.sql`、写入初始化数据。
- `database/init.sql` 大量使用 MySQL 方言，例如：
  - `ENGINE=InnoDB`
  - `CHARSET=utf8mb4`
  - `COLLATE=utf8mb4_unicode_ci`
  - `enum(...)`
  - `tinyint(1)`
  - `datetime(6)`
  - `DEFAULT (uuid())`
- 运行时还有 MySQL 特定兼容逻辑，例如：
  - `ALTER TABLE ... MODIFY COLUMN ... ENUM(...)`
  - 启动时 MySQL 服务可达性检测
  - “数据库不存在则自动创建” 的 MySQL 连接流程

同时，`dev_core` 仍保留了一整套历史数据域实现：

- 路由：`src/routes/v1/data.js`
- 服务：`dataConnectionService`、`dataQueryService`、`dataPointService`、`mqttService`
- 控制器：MQTT 连接、订阅、Tag、TagGroup、数据点控制器
- Sequelize 模型：
  - `DataConnection`
  - `DataRelationalConfig`
  - `DataQuery`
  - `DataQueryLog`
  - `DataPoint`
  - `DataMqttConfig`
  - `DataMqttSubscription`
  - `DataMqttTagGroup`
  - `DataMqttTag`

### 2.2 `data_service`

`data_service` 已经是正式的平台侧数据域服务，并且已经采用 PostgreSQL：

- 通过 `DATA_SERVICE_DATABASE_URL` 连接 PostgreSQL。
- 拥有独立迁移体系和路由装配。
- 已实现的主干领域包括：
  - connections
  - queries
  - datapoints
  - mqtt
  - protocol wave1 / wave2
  - preview sessions
  - compute units

### 2.3 消费方现状

- `datacenter` 开发态已经把 `/api/v1/data` 代理到独立的 `data_service`。
- 但 `datacenter` 使用的接口契约并未与 `data_service` 完全对齐，仍存在若干历史路径依赖。
- `designer` 内仍有：
  - `/api/v1/datapoints/:path/value`
  - `/api/v1/datapoints/values`
  - `/api/v1/datapoints/status`
  - `/api/v1/data/projects/:projectId/datapoints/status`
  - `/api/v1/data/projects/:projectId/datapoints/values`
  这些路径目前并未在 `data_service` 中形成完整正式契约。

## 3. 终态边界

本轮改造完成后，正式边界如下：

### 3.1 `dev_core` 只保留控制面能力

`dev_core` 只承接以下对象与接口：

- 租户、用户、角色、认证
- 工程与页面管理
- 设计资源
- 发布、部署、节点、命令、日志
- 控制面聚合逻辑

`dev_core` 不再承接以下任何数据域本地实现：

- `/api/v1/data/**` 路由
- 数据连接、查询、数据点、MQTT、协议接入、preview、compute 的本地 CRUD/执行能力
- 对应 Sequelize 模型及表

### 3.2 `data_service` 成为唯一平台侧数据域拥有者

以下能力以后只允许由 `data_service` 提供：

- connections
- relational config / protocol config
- queries
- datapoints
- mqtt connection / subscription / messages / tag / tag-group
- preview sessions
- compute units
- 面向设计态、数据工作台和发布快照的项目级数据域读取接口

### 3.3 配置统一方式

根目录 `.env` 统一维护 PostgreSQL 主连接信息。

建议约定如下：

```env
DB_HOST=172.21.242.174
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=tenant_management

DATA_SERVICE_DATABASE_URL=postgres://postgres:postgres@172.21.242.174:5432/data_service?sslmode=disable
```

设计原则：

- `dev_core` 继续使用 `DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME`
- `data_service` 继续使用 `DATA_SERVICE_DATABASE_URL`
- 两者共用同一套主机、端口、用户名、密码
- 不再保留 MySQL 作为默认数据库实现

## 4. 必须补齐的缺口

要实现“彻底下线 `dev_core` 数据域实现”，不能先删代码，必须先补齐 `data_service` 契约缺口。

### 4.1 连接领域缺口

`data_service` 需要补齐或对齐下列历史能力：

- 连接测试：`connections/test`
- 连接状态更新：`connections/:connectionId/status`
- 关系库表列表读取
- 表结构读取
- 表数据分页预览
- 项目级直接 SQL 执行：`execute-sql`

### 4.2 MQTT 领域缺口

`data_service` 当前 MQTT 只覆盖主干，不足以替换 `dev_core` 现有实现。至少要补齐：

- MQTT 连接列表、详情、更新、删除、停止
- MQTT 订阅完整 CRUD
- MQTT TagGroup 完整 CRUD 与排序
- MQTT Tag 完整 CRUD、排序、批量创建、值读取
- 与数据点自动同步/失效标记的正式实现

### 4.3 数据点领域缺口

至少需要补齐：

- `datapoints/status`
- `datapoints/values`
- 如设计态需要，补齐 path 维度的单点读取/批量读取能力
- 数据点 usages 查询，或者明确从消费侧删除该需求

### 4.4 导入导出与发布快照缺口

这是本轮最容易被漏掉的部分。

当前 `dev_core` 下列流程直接读取本地数据域表：

- 工程导出
- 工程导入
- 发布打包
- 发布清单统计

这些流程必须改成从 `data_service` 获取项目级数据域快照，不能继续依赖 `dev_core` 本地表。

建议在 `data_service` 增加一个正式项目快照接口，例如：

- 读取项目当前 connections / queries / datapoints / mqtt / protocol / compute 快照
- 支持导出场景
- 支持发布打包场景

如果没有这个项目快照接口，就不能删除 `dev_core` 的历史数据域表。

### 4.5 Designer / Diagnostics 缺口

`designer` 目前仍存在一批非正式数据接口依赖。需要统一方案：

- 统一改为走 `data_service` 正式接口
- 或者删除不用的诊断路径
- 禁止继续保留 `dev_core` 兼容路由作为兜底

## 5. `dev_core` PostgreSQL 迁移设计

### 5.1 迁移范围

`dev_core` PG 迁移只覆盖控制面数据库，不再包含历史数据域表。

保留的核心表包括：

- `tenants`
- `users`
- `projects`
- `project_members`
- `logs`
- `design_pages`
- `design_project_settings`
- `design_assets`
- `design_asset_folders`
- `deployments`
- `deployment_pages`
- `nodes`
- `node_deployments`
- `node_commands`

数据域相关历史表将从 `dev_core/database/init.sql` 中删除：

- `data_connections`
- `data_relational_configs`
- `data_queries`
- `data_query_logs`
- `data_points`
- `data_mqtt_configs`
- `data_mqtt_subscriptions`
- `data_mqtt_tag_groups`
- `data_mqtt_tags`
- 以及其余协议/驱动/告警等历史数据域表

### 5.2 PostgreSQL 结构原则

本轮 PostgreSQL 结构设计遵循 `design-postgres-tables` 技能约束，`dev_core` 与 `data_service` 都按 PostgreSQL 优先原则清理和优化表结构。

`dev_core` PostgreSQL 结构采用以下约定：

- 主键统一使用 `uuid`
- 布尔值使用 `boolean`
- 结构化字段统一使用 `jsonb`
- 时间字段统一优先使用 `timestamptz`
- 尽量避免直接依赖 PostgreSQL 原生 enum 类型，优先使用 Sequelize 枚举 + `CHECK` 约束可兼容写法
- 不再使用 MySQL 风格的 `varchar(n)`、`char(n)`、`tinyint(1)`、`datetime(6)` 作为默认建模方式
- 主业务表保持规范化优先，只在有明确收益的读取场景下再考虑定向反规范化
- 外键列必须显式补索引，不能依赖 PostgreSQL 自动处理

### 5.3 PostgreSQL 优化与清理策略

本轮不是机械“方言替换”，允许在迁移时顺手清理历史库结构，但范围必须严格围绕性能、约束一致性和正式边界，不做无关重构。

#### 5.3.1 `dev_core` 优化原则

- 只保留控制面正式对象，借迁移机会删除历史数据域表及其相关索引、外键、冗余字段
- 对高频筛选路径补齐组合索引，例如：
  - `projects(tenant_id, status)`
  - `users(tenant_id, role)`
  - `deployments(project_id, created_at desc)`
  - `node_deployments(project_id, status)`
  - `node_commands(node_id, status, requested_at desc)`
- 对 JSON 结构字段仅保留真正需要灵活扩展的配置对象，其余可稳定字段尽量拆回明确列
- 时间线、状态流转、部署查询场景优先按“工程 + 状态 + 时间”设计索引，避免控制面查询退化

#### 5.3.2 `data_service` 优化原则

- 保持数据域主表 + 专属配置表的拆分，不把协议专属字段重新混回 `data_connections`
- 针对典型访问路径补强索引：
  - `data_connections(project_id, type)`
  - `data_connections(project_id, status)`
  - `data_queries(project_id, connection_id)`
  - `data_points(project_id, path)`
  - `data_mqtt_subscriptions(project_id, connection_id)`
  - `data_preview_sessions(project_id, user_id, status)`
  - `data_compute_units(project_id, status)`（如状态字段存在）
- 对高频对象配置使用 `jsonb`，并只在确实存在 containment / key existence 查询时增加 GIN 索引
- 对 update-heavy 表优先控制宽列更新频率，避免不必要的索引字段被频繁改写

#### 5.3.3 命名与映射策略

- PostgreSQL 物理表名、列名优先采用 `snake_case`
- `dev_core` 因为继续使用 Sequelize，可以通过 `field` 映射兼容现有 JavaScript 层命名，避免业务层一次性大面积重写
- 不允许继续新增大小写混用或依赖双引号的 PostgreSQL 标识符

### 5.4 `dev_core` 使用 Sequelize 的策略

`dev_core` 切换到 PostgreSQL 后，继续使用 Sequelize，原因如下：

- 当前 `dev_core` 已经围绕 Sequelize 组织了模型、关联、事务和大部分服务层逻辑
- 本轮核心风险在于数据库方言切换和数据域下线，不在于 ORM 能力不足
- 同时替换 ORM 会把“数据库切换风险”和“持久层重构风险”叠加，不符合本轮收口目标

因此本轮策略明确为：

- 保留 Sequelize
- 将 dialect 切到 `postgres`
- 修正模型字段类型与字段映射
- 改写少量原生 SQL 为 PostgreSQL 版本
- 仅在 Sequelize 明显不适配的特定查询场景保留参数化原生 SQL

### 5.5 初始化策略

`dev_core/scripts/init-database.js` 需要改造成 PostgreSQL 版本：

- 改用 `pg`
- 如 database 不存在，改用 PostgreSQL 的 system database 连接检查与创建流程
- 执行新的 PostgreSQL `init.sql`
- 写入默认租户、超级管理员、系统管理员

### 5.6 运行时连接策略

`dev_core/src/config/database.js` 需要整体重写：

- Sequelize dialect 改为 `postgres`
- 移除 `mysql2/promise`
- 移除全部 MySQL 特定连接诊断、健康检查提示与兼容修复逻辑
- 改成 PostgreSQL 连接检查、连接池和错误诊断

### 5.7 必须改掉的 MySQL 方言

当前已确认至少存在以下 MySQL 方言，需要在迁移时一并处理：

- `ON DUPLICATE KEY UPDATE`
- `ALTER TABLE ... MODIFY COLUMN ... ENUM(...)`
- `ENGINE=InnoDB`
- `CHARSET=utf8mb4`
- `COLLATE=utf8mb4_unicode_ci`
- `tinyint(1)`
- `datetime(6)`
- MySQL 风格建库流程

其中已明确在业务代码里出现的 `ON DUPLICATE KEY UPDATE` 位于：

- `src/routes/v1/project.js`
- `src/services/designService.js`

这部分需要改写为 PostgreSQL 可执行的 upsert 语义。

## 6. 实施顺序

本轮必须分阶段实施，顺序不可颠倒。

### 阶段 A：补齐 `data_service`

目标：

- 补齐替代 `dev_core` 所需的所有正式数据域接口
- 增加项目级数据域快照接口
- 补齐 `designer` / `datacenter` 依赖的点位状态和值接口

阶段完成标准：

- `datacenter` 与 `designer` 不再依赖 `dev_core` 的任何数据域实现
- `dev_core` 的导入导出和发布链路可以只通过 `data_service` 读取数据域信息

### 阶段 B：切消费方

目标：

- `datacenter` 完整对齐 `data_service` 契约
- `designer` 诊断与数据访问改到 `data_service`
- `dev_core` 的导出、导入、发布、统计都改为调用 `data_service`

阶段完成标准：

- 仓库内不再有任何业务模块依赖 `dev_core` 本地数据域表

### 阶段 C：删除 `dev_core` 数据域实现

目标：

- 删除 `src/routes/v1/data.js`
- 删除相关 controller / service / model
- 删除 `buildV1Router` 中 `/data` 挂载
- 删除 `database/init.sql` 中所有历史数据域表

阶段完成标准：

- `dev_core` 不再暴露 `/api/v1/data`
- `dev_core` 不再初始化数据域表

### 阶段 D：`dev_core` 切 PostgreSQL

目标：

- PostgreSQL 化 `database/init.sql`
- PostgreSQL 化 `scripts/init-database.js`
- PostgreSQL 化 `src/config/database.js`
- 保留 Sequelize，完成 PostgreSQL 方言切换与字段映射调整
- 修复全部 MySQL 方言 SQL 和原始查询
- 更新根 `.env` 与 `.env_example`

阶段完成标准：

- `pnpm --dir dev_core test` 可在 PostgreSQL 配置下运行
- `data_service` 在统一 PG 连接参数下可正常连接和迁移

## 7. 配置调整口径

本轮配置文档与模板需要统一更新为 PostgreSQL 口径。

需要更新的至少包括：

- 根 `.env`
- 根 `.env_example`
- `dev_core/database/README.md`
- `docs/backend/database-init.md`
- 其他仍写明 MySQL 默认值的文档

更新原则：

- 根 `.env` 以你的本机实际值为准
- `.env_example` 保留统一默认值，但口径改为 PostgreSQL
- 文档不再把 MySQL 作为默认数据库方案描述

## 8. 风险与回避策略

### 8.1 最大风险

最大风险不是 PG 迁移，而是“以为只要删掉 `dev_core` 的 `/api/v1/data` 就算完成”。

真正的高风险点在于：

- 发布打包链路断裂
- 工程导入导出断裂
- `datacenter` 某些老接口未补齐
- `designer` 点位诊断链路失效

### 8.2 回避策略

- 先补 `data_service`，再删 `dev_core`
- 先切消费方，确认没有调用残留，再删除本地模型和表
- 最后再做 `dev_core` PostgreSQL 化，避免两个方向同时失稳

## 9. 本轮结论

基于当前仓库现状，本轮正确的实施策略是：

1. 以 `data_service` 为唯一平台侧数据域正式实现。
2. 以“项目级数据域快照接口”替代 `dev_core` 对历史本地数据表的直接依赖。
3. 以 PostgreSQL 替换 `dev_core` 当前 MySQL 实现，并将根 `.env` 的数据库连接口径统一为你指定的 PG 实例。
4. 最终彻底删除 `dev_core` 的 `/api/v1/data` 路由、本地数据域模型与数据域表。
