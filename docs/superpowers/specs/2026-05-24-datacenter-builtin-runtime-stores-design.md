# 数据中心 IF 内置运行库设计

> 日期：2026-05-24  
> 范围：`datacenter/` 数据中心前端、`data_service/` 数据域服务、数据域发布 artifact。  
> 目标：在数据中心提供面向运行态的 IF 内置运行库建模，并在开发环境中完成真实可执行测试链路。  
> 关键边界：本轮不实现 `runtime/node_agent` 对 artifact 的消费和运行态初始化。

## 1. 背景

数据中心需要为没有自有数据基础设施的用户提供平台内置数据能力。用户在设计工程时应看到稳定的 InduForge 能力名称，而不是平台内部数据库、缓存或消息服务的连接参数。

当前 `docs/datacenter/IF内置运行库设计.md` 同时描述了关系、时序、实时、消息和对象能力。经讨论，数据中心的职责应聚焦在数据接入、数据建模、数据点、查询、计算和报警。对象能力主要服务设计资源、工程包、发布产物和导入导出文件，不应作为数据中心接入源暴露。

因此，本轮数据中心只纳入四类 IF 内置运行库：

- `IF关系库`
- `IF时序库`
- `IF实时库`
- `IF消息库`

`IF对象库` 保留为平台资源能力，由设计中心、发布链路或后续平台资源管理模块承载，不进入本轮数据中心实施范围。

## 2. 目标

- 在数据中心“新建接入源”中提供“内置运行库”类别。
- 用户可创建或启用 `IF关系库`、`IF时序库`、`IF实时库`、`IF消息库`。
- 用户不填写平台内部数据库、缓存、消息服务地址或访问密钥。
- 开发环境中四类内置运行库必须真实可测试。
- 发布 artifact 携带运行态初始化所需契约。
- 开发态数据不进入发布产物。
- 平台系统库、系统缓存 DB、内部消息服务和跨工程资源不得被用户访问。

## 3. 不做范围

- 不实现 `runtime/node_agent` 初始化运行态 schema、topic、namespace 或运行态角色。
- 不实现运行态数据采集、计算、报警写入内置运行库。
- 不实现 `IF对象库` 在数据中心中的建模、测试或 artifact 字段。
- 不发布开发态样例数据。
- 不做通用文件型数据源。如果未来需要 CSV、Parquet、Excel 等文件数据接入，应作为独立“文件接入源”设计。
- 不在第一版实现完整 SQL AST 白名单引擎；第一版使用服务端上下文隔离、危险语句拦截、超时和结果限制作为基础防线。

## 4. 核心概念

### IF关系库

面向普通业务表、页面表单数据和 SQL 查询。开发态使用 `IF_META_STORE_DEV_DATA_DB` 中的项目级 schema：

```text
p_<projectId>_app
```

用户 SQL 不依赖 schema 名：

```sql
select * from orders limit 10;
```

服务端执行时注入受控上下文：

```sql
set search_path to p_<projectId>_app, public;
select * from orders limit 10;
```

### IF时序库

面向数据点历史、趋势查询、计算采样和报警采样历史。开发态使用 `IF_META_STORE_DEV_DATA_DB` 中的项目级时序 schema：

```text
p_<projectId>_ts
```

时序库需要支持：

- 创建时序表定义。
- 写入调试样例点值。
- 查询趋势和聚合结果。
- 配置默认保留策略，默认建议 30 天。

### IF实时库

面向当前值、短期状态和缓存。开发态通过 `cache-store` 使用项目级 key 前缀：

```text
<IF_DEV_CACHE_KEY_PREFIX>:<projectId>:rt:<key>
<IF_DEV_CACHE_KEY_PREFIX>:<projectId>:cache:<key>
<IF_DEV_CACHE_KEY_PREFIX>:<projectId>:state:<key>
```

第一版测试能力包括 key 写入、读取、删除和 TTL 验证。

### IF消息库

面向工程内置消息接入、模拟数据和设备主题。开发态通过 `message-hub` 使用项目级 topic 前缀：

```text
<IF_MESSAGE_HUB_TOPIC_PREFIX>/<projectId>/mock-data
<IF_MESSAGE_HUB_TOPIC_PREFIX>/<projectId>/device/<deviceId>/telemetry
<IF_MESSAGE_HUB_TOPIC_PREFIX>/<projectId>/event/<eventName>
```

第一版测试能力包括发布消息、订阅消息和模拟数据调试。

## 5. 数据模型

第一版复用 `data_connections` 表承载内置运行库定义，避免额外引入一套资源表和列表 API。

新增连接类型：

```text
builtin.relation
builtin.timeseries
builtin.realtime
builtin.message
```

新增连接分类：

```text
builtin
```

内置运行库记录示例：

```json
{
  "id": "uuid",
  "projectId": "project-1",
  "name": "IF关系库",
  "type": "builtin.relation",
  "category": "builtin",
  "status": "connected",
  "config": {
    "store": "relation",
    "devSchema": "p_project_1_app",
    "runtimeSchema": "app",
    "ddlVersion": "2026-05-24.1"
  }
}
```

每个项目同一种内置运行库最多一条。后端在创建时检查 `(project_id, type)` 唯一性；如果记录已存在，返回业务错误，提示用户使用已有内置运行库。

## 6. 前端交互

`ConnectionDialog.vue` 的接入类型选择分为两组：

```text
内置运行库
[ IF关系库 ]    业务表、表单数据和普通 SQL
[ IF时序库 ]    数据点历史、趋势和聚合
[ IF实时库 ]    当前值、短期状态和缓存
[ IF消息库 ]    设备消息、模拟数据和订阅

外部数据源
[ PostgreSQL ] [ MySQL ] [ SQL Server ] [ Redis ] [ MQTT ] ...
```

内置运行库的创建表单不展示 host、port、username、password、broker、DB 编号等内部参数。用户只可配置名称、描述和少量策略字段：

- `IF关系库`：默认 schema 说明、是否允许 DDL 测试。
- `IF时序库`：默认保留天数、时间字段约定。
- `IF实时库`：默认 TTL、key 分组说明。
- `IF消息库`：默认 topic 分组、示例 payload。

工作台按类型展示：

- `builtin.relation`：SQL 工作台。
- `builtin.timeseries`：时序表、样例写入、趋势查询工作台。
- `builtin.realtime`：key/value、TTL 和当前值测试工作台。
- `builtin.message`：topic 发布、订阅和模拟数据工作台。

## 7. 后端 API

复用现有连接 API：

```text
GET    /api/v1/data/projects/:projectId/connections
POST   /api/v1/data/projects/:projectId/connections
GET    /api/v1/data/projects/:projectId/connections/:connectionId
PUT    /api/v1/data/projects/:projectId/connections/:connectionId
DELETE /api/v1/data/projects/:projectId/connections/:connectionId
```

新增或扩展测试 API，按内置类型分发：

```text
POST /api/v1/data/projects/:projectId/builtin/relation/sql/execute
POST /api/v1/data/projects/:projectId/builtin/timeseries/query
POST /api/v1/data/projects/:projectId/builtin/timeseries/sample
POST /api/v1/data/projects/:projectId/builtin/realtime/keys
GET  /api/v1/data/projects/:projectId/builtin/realtime/keys/:key
POST /api/v1/data/projects/:projectId/builtin/message/publish
POST /api/v1/data/projects/:projectId/builtin/message/preview-session
```

所有 API 都从 JWT claims 校验项目访问权，并只使用服务端推导出的项目级 schema、key 前缀和 topic 前缀。

## 8. 开发态资源生命周期

创建内置运行库时初始化对应开发态资源：

- `IF关系库`：创建 `p_<projectId>_app` schema 和 `if_schema_migrations`。
- `IF时序库`：创建 `p_<projectId>_ts` schema 和时序迁移记录。
- `IF实时库`：确认项目 key 前缀可用，不暴露 Redis DB。
- `IF消息库`：确认项目 topic 前缀可用，不暴露 broker 地址。

删除内置运行库时只删除建模记录。第一版不自动清空 schema、key 或 topic 历史数据，避免误删调试结果。删除前必须检查数据点、查询、计算和报警引用；存在引用时拒绝删除。

## 9. SQL 与执行安全

`IF关系库` 和 `IF时序库` 的 SQL 执行必须由服务端控制上下文：

- 强制连接到开发态数据承载库。
- 强制注入项目级 `search_path`。
- 禁止用户显式引用系统库、系统 schema 和其他工程 schema。
- 禁止危险语句：`DROP DATABASE`、`CREATE DATABASE`、`ALTER SYSTEM`、`COPY ... PROGRAM`。
- 限制执行超时、最大返回行数和响应体大小。
- 业务变量使用参数化绑定，不拼接用户输入。

第一版允许项目 schema 内的基础 DDL 和 DML，以满足开发环境建模和测试：

- `CREATE TABLE`
- `ALTER TABLE`
- `CREATE INDEX`
- `INSERT`
- `UPDATE`
- `DELETE`
- `SELECT`

跨库、跨 schema、系统级变更不允许。

## 10. Artifact 契约

`ProjectArtifactV1` 增加 `builtinStores` 字段：

```json
{
  "builtinStores": {
    "relation": {
      "enabled": true,
      "schema": "app",
      "ddlVersion": "2026-05-24.1",
      "queries": []
    },
    "timeseries": {
      "enabled": true,
      "schema": "ts",
      "retentionDays": 30,
      "ddlVersion": "2026-05-24.1",
      "tables": []
    },
    "realtime": {
      "enabled": true,
      "namespace": "rt",
      "defaultTtlSeconds": 300,
      "keys": []
    },
    "message": {
      "enabled": true,
      "topicPrefix": "project",
      "subscriptions": [],
      "bindings": []
    }
  }
}
```

artifact 不包含开发态 schema 名、Redis DB、MQTT broker、内部用户名、密码或 token。运行态实际资源名称由后续 `node_agent` 按运行环境映射。

## 11. 权限与隔离

禁止访问或暴露以下资源：

```text
if_core
if_data
if_dev_data
postgres
template0
template1
information_schema
pg_catalog
pg_toast
任何 if_platform_* / induforge_* 系统库
任何平台内部缓存 DB
任何平台内部消息服务地址
任何平台内部对象存储密钥
```

对象库不属于数据中心本轮边界，因此不出现在接入源、数据点来源或 data_service 内置运行库模型中。

## 12. 验证标准

- 前端可创建四类内置运行库，且表单不出现内部连接参数。
- 同一项目重复创建同类内置运行库会收到明确业务错误。
- `IF关系库` 可在开发环境执行项目 schema 内 SQL。
- `IF时序库` 可创建样例时序表、写入样例数据并查询趋势结果。
- `IF实时库` 可写入、读取和删除项目命名空间内 key。
- `IF消息库` 可发布和订阅项目 topic 前缀内消息。
- artifact 中包含 `builtinStores`，且不包含开发态数据和内部连接密钥。
- `pnpm --filter datacenter typecheck` 通过。
- `go test ./...` 或最小相关 Go 包测试通过。

## 13. 实施拆分建议

本设计建议拆为四个实施阶段：

1. 内置运行库连接模型和前端入口。
2. 开发态关系库和时序库真实执行。
3. 开发态实时库和消息库真实执行。
4. artifact 契约输出和回归测试。

阶段之间可以独立验证。第一阶段完成后，用户已能看到稳定产品名；第二、三阶段完成后，开发环境测试闭环完整；第四阶段完成后，后续 node_agent 运行态初始化有稳定输入。
