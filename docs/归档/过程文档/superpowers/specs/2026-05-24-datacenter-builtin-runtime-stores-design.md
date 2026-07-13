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

## 2. 核心结论

一个内置运行库连接代表一个用户可管理的运行库实例，而不是项目级单例开关。

用户创建连接时只选择内置类型并填写名称。系统生成不可变内部标识，用户不可填写、不可修改。名称只用于界面展示，允许后续修改；运行资源、数据点、查询、topic、key 空间和 artifact 引用都不得依赖名称。

| 类型                 | 一个连接代表        | 用户可管理内容                      | 开发态隔离方式  |
| -------------------- | ------------------- | ----------------------------------- | --------------- |
| `builtin.relation`   | 一个逻辑关系库      | 表结构、SQL、查询、数据点           | 独立 schema     |
| `builtin.timeseries` | 一个逻辑时序库      | 时序表、样例点、趋势 SQL、保留策略  | 独立 schema     |
| `builtin.realtime`   | 一个实时 key 空间   | key 定义、当前值测试、TTL           | Redis key 前缀  |
| `builtin.message`    | 一个消息 topic 空间 | topic、变量、payload 解析、发布预览 | MQTT topic 前缀 |

## 3. 目标

- 在数据中心“新建接入源”中提供“内置运行库”类别。
- 同一项目可创建多个 `IF关系库`、`IF时序库`、`IF实时库`、`IF消息库` 实例。
- 用户不填写平台内部数据库、缓存、消息服务地址、运行资源标识或访问密钥。
- 开发环境中四类内置运行库必须真实可测试。
- 发布 artifact 携带运行态初始化所需契约。
- 开发态数据不进入发布产物。
- 平台系统库、系统缓存 DB、内部消息服务和跨工程资源不得被用户访问。

## 4. 不做范围

- 不实现 `runtime/node_agent` 初始化运行态 schema、topic、namespace 或运行态角色。
- 不实现运行态数据采集、计算、报警写入内置运行库。
- 不实现 `IF对象库` 在数据中心中的建模、测试或 artifact 字段。
- 不发布开发态样例数据、实时当前值或历史消息。
- 不做通用文件型数据源。如果未来需要 CSV、Parquet、Excel 等文件数据接入，应作为独立“文件接入源”设计。
- 不在第一版实现完整 SQL AST 白名单引擎；第一版使用服务端上下文隔离、危险语句拦截、超时和结果限制作为基础防线。

## 5. 标识与命名

每条内置运行库连接至少包含以下字段：

| 字段         | 来源                | 是否可改 | 用途                             |
| ------------ | ------------------- | -------- | -------------------------------- |
| `id`         | 系统生成            | 否       | 主键、前端工作台定位             |
| `runtimeKey` | 系统生成            | 否       | 运行资源、artifact、内部隔离前缀 |
| `name`       | 用户填写            | 是       | UI 展示                          |
| `type`       | 用户选择            | 否       | 内置运行库类型                   |
| `category`   | 系统写入            | 否       | 固定为 `builtin`                 |
| `config`     | 系统生成 + 策略配置 | 部分可改 | 保留天数、默认 TTL、测试开关     |

`runtimeKey` 由服务端生成并保存在 `config.runtimeKey`。建议格式由类型前缀和短随机串组成，例如：

```text
rel_a8f3c2
ts_d91b70
rt_2cb7aa
msg_f60d4e
```

用户可以把“订单库”改名为“生产订单库”，但 `runtimeKey`、开发态 schema、key 前缀、topic 前缀和 artifact 引用保持不变。

## 6. 数据模型

第一版复用 `data_connections` 表承载内置运行库定义，避免额外引入一套运行库实例列表 API。

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
  "name": "订单库",
  "type": "builtin.relation",
  "category": "builtin",
  "status": "connected",
  "config": {
    "store": "relation",
    "runtimeKey": "rel_a8f3c2",
    "devSchema": "p_project_1_rel_a8f3c2",
    "runtimeSchema": "rel_a8f3c2",
    "ddlVersion": "2026-05-24.2"
  }
}
```

同一项目允许存在多条同类型内置运行库连接，因此不得建立 `(project_id, type)` 唯一约束。需要通过 `(project_id, id)` 定位具体实例。

## 7. 四类运行库设计

### 7.1 IF关系库

`IF关系库` 面向普通业务表、页面表单数据和 SQL 查询。每个连接对应一个逻辑关系库。

开发态使用 `IF_META_STORE_DEV_DATA_DB` 中的独立 schema：

```text
p_<projectId>_<runtimeKey>
```

用户 SQL 不依赖 schema 名：

```sql
select * from orders limit 10;
```

服务端执行时按连接注入受控上下文：

```sql
set search_path to p_<projectId>_<runtimeKey>, public;
select * from orders limit 10;
```

工作台参考外部 SQL 工作台，支持：

- 表列表与表结构。
- SQL 多标签编辑、格式化、执行。
- DDL/DML 测试。
- 保存查询。
- 执行历史。
- 查询生成数据点。

### 7.2 IF时序库

`IF时序库` 面向数据点历史、趋势查询、计算采样和报警采样历史。每个连接对应一个逻辑时序库。

开发态使用独立 schema：

```text
p_<projectId>_<runtimeKey>
```

时序库工作台应复用 SQL 工作台骨架，并增加时序增强能力：

- 时序表模板。
- 时间字段、标签字段和值字段约定。
- 写入调试样例点值。
- 趋势查询和聚合查询模板。
- 默认保留策略，默认建议 30 天。

`IF时序库` 不应做成脱离 SQL 的独立玩具面板。它本质仍需要表结构、SQL 查询和结果预览能力。

### 7.3 IF实时库

`IF实时库` 面向当前值、短期状态和缓存。每个连接对应一个实时 key 空间。

开发态通过 `cache-store` 使用连接级 key 前缀：

```text
<IF_DEV_CACHE_KEY_PREFIX>:<projectId>:<runtimeKey>:<userKey>
```

用户只看到并管理 `userKey`，例如：

```text
device/line1/status
cache/order/latest
state/current-user
```

实时库工作台参考外部 Redis 工作台，但产品语义不是 Redis 管理器，而是“实时 key 建模 + 当前值测试”：

- 创建和维护 key 定义。
- 配置数据类型、默认 TTL、说明和运行态权限。
- 设置是否生成数据点。
- 按前缀浏览开发态 key。
- 写入、读取、删除开发态当前值。
- 查看 TTL 和当前值内容。

artifact 发布 key 定义，不发布开发态当前值。

### 7.4 IF消息库

`IF消息库` 面向工程内置消息接入、模拟数据和设备主题。每个连接对应一个消息 topic 空间。

开发态通过 `message-hub` 使用连接级 topic 前缀：

```text
<IF_MESSAGE_HUB_TOPIC_PREFIX>/<projectId>/<runtimeKey>/<userTopic>
```

用户只看到并管理 `userTopic`，例如：

```text
device/+/telemetry
alarm/event
mock-data
```

消息库工作台参考外部 MQTT 工作台，但不暴露 broker 地址、账号、端口和外部连接生命周期：

- 建立 topic 或订阅定义。
- topic 树或分组管理。
- 发布 JSON 测试消息。
- 查看最近消息或预览消息。
- 在 topic 下管理变量。
- 变量配置 payload path、数据类型、单位、说明。
- 变量可生成数据点。

artifact 发布 topic、变量和绑定定义，不发布开发态历史消息。

## 8. 前端交互

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

内置运行库的创建表单不展示 host、port、username、password、broker、DB 编号、schema 标识、key 前缀或 topic 前缀。用户只填写名称、说明和少量策略字段：

- `IF关系库`：名称、说明、是否允许 DDL 测试。
- `IF时序库`：名称、说明、默认保留天数、时间字段约定。
- `IF实时库`：名称、说明、默认 TTL。
- `IF消息库`：名称、说明、默认 topic 分组或示例 payload。

详情页可只读展示“系统标识”，用于排障和 artifact 对照，但不作为创建输入。

## 9. 后端 API

复用现有连接 API 管理内置运行库实例：

```text
GET    /api/v1/data/projects/:projectId/connections
POST   /api/v1/data/projects/:projectId/connections
GET    /api/v1/data/projects/:projectId/connections/:connectionId
PUT    /api/v1/data/projects/:projectId/connections/:connectionId
DELETE /api/v1/data/projects/:projectId/connections/:connectionId
```

`IF关系库` 和 `IF时序库` 应优先复用连接级 SQL API：

```text
GET  /api/v1/data/projects/:projectId/connections/:connectionId/tables
GET  /api/v1/data/projects/:projectId/connections/:connectionId/tables/:tableName/structure
POST /api/v1/data/projects/:projectId/connections/:connectionId/execute-sql
```

服务端根据连接类型分发：

- 外部 SQL 连接：走外部数据库。
- `builtin.relation`：走该连接的内置关系库 schema。
- `builtin.timeseries`：走该连接的内置时序库 schema。

实时库需要连接级 key API：

```text
GET    /api/v1/data/projects/:projectId/connections/:connectionId/realtime/keys
POST   /api/v1/data/projects/:projectId/connections/:connectionId/realtime/keys
GET    /api/v1/data/projects/:projectId/connections/:connectionId/realtime/keys/:key
DELETE /api/v1/data/projects/:projectId/connections/:connectionId/realtime/keys/:key
```

消息库需要连接级 topic 和变量 API：

```text
GET  /api/v1/data/projects/:projectId/connections/:connectionId/message/topics
POST /api/v1/data/projects/:projectId/connections/:connectionId/message/topics
POST /api/v1/data/projects/:projectId/connections/:connectionId/message/publish
GET  /api/v1/data/projects/:projectId/connections/:connectionId/message/topics/:topicId/variables
POST /api/v1/data/projects/:projectId/connections/:connectionId/message/topics/:topicId/variables
```

所有 API 都从 JWT claims 校验项目访问权，并只使用服务端推导出的连接级 schema、key 前缀和 topic 前缀。

## 10. 开发态资源生命周期

创建内置运行库时初始化对应开发态资源：

- `IF关系库`：创建连接级 schema 和 `if_schema_migrations`。
- `IF时序库`：创建连接级 schema、时序迁移记录和保留策略配置。
- `IF实时库`：确认连接级 key 前缀可用，不暴露 Redis DB。
- `IF消息库`：确认连接级 topic 前缀可用，不暴露 broker 地址。

删除内置运行库时只删除建模记录。第一版不自动清空 schema、key 或 topic 历史数据，避免误删调试结果。删除前必须检查数据点、查询、计算和报警引用；存在引用时拒绝删除。

如果后续需要清理开发态残留资源，应单独提供“清理开发态数据”显式操作。

## 11. SQL 与执行安全

`IF关系库` 和 `IF时序库` 的 SQL 执行必须由服务端控制上下文：

- 强制连接到开发态数据承载库。
- 强制按 `connectionId` 注入连接级 `search_path`。
- 禁止用户显式引用系统库、系统 schema 和其他工程 schema。
- 禁止危险语句：`DROP DATABASE`、`CREATE DATABASE`、`ALTER SYSTEM`、`COPY ... PROGRAM`。
- 限制执行超时、最大返回行数和响应体大小。
- 业务变量使用参数化绑定，不拼接用户输入。

第一版允许连接级 schema 内的基础 DDL 和 DML，以满足开发环境建模和测试：

- `CREATE TABLE`
- `ALTER TABLE`
- `CREATE INDEX`
- `INSERT`
- `UPDATE`
- `DELETE`
- `SELECT`

跨库、跨 schema、系统级变更不允许。

## 12. Artifact 契约

`ProjectArtifactV1` 增加 `builtinStores` 字段。由于同一项目可创建多个同类内置运行库，artifact 使用集合结构：

```json
{
  "builtinStores": {
    "relations": [
      {
        "id": "connection-id",
        "runtimeKey": "rel_a8f3c2",
        "name": "订单库",
        "schema": "rel_a8f3c2",
        "ddlVersion": "2026-05-24.2",
        "queries": []
      }
    ],
    "timeseries": [
      {
        "id": "connection-id",
        "runtimeKey": "ts_d91b70",
        "name": "设备历史库",
        "schema": "ts_d91b70",
        "retentionDays": 30,
        "ddlVersion": "2026-05-24.2",
        "tables": []
      }
    ],
    "realtimeSpaces": [
      {
        "id": "connection-id",
        "runtimeKey": "rt_2cb7aa",
        "name": "产线实时空间",
        "namespace": "rt_2cb7aa",
        "defaultTtlSeconds": 300,
        "keys": []
      }
    ],
    "messageSpaces": [
      {
        "id": "connection-id",
        "runtimeKey": "msg_f60d4e",
        "name": "设备消息空间",
        "topicPrefix": "msg_f60d4e",
        "topics": [],
        "variables": [],
        "bindings": []
      }
    ]
  }
}
```

artifact 不包含开发态 schema 名、Redis DB、MQTT broker、内部用户名、密码、token、实时当前值或历史消息。运行态实际资源名称由后续 `node_agent` 按运行环境映射。

## 13. 权限与隔离

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

## 14. 验证标准

- 前端可创建四类内置运行库，且表单不出现内部连接参数或内部标识输入。
- 同一项目可创建多个同类型内置运行库。
- 用户修改名称后，SQL、数据点、key、topic 和 artifact 引用保持稳定。
- `IF关系库` 可在开发环境执行连接级 schema 内 SQL，并可查看表列表和表结构。
- `IF时序库` 可创建样例时序表、写入样例数据并查询趋势结果。
- `IF实时库` 可维护 key 定义，并可写入、读取、删除连接级命名空间内开发态当前值。
- `IF消息库` 可维护 topic 和变量，并可发布、预览连接级 topic 前缀内消息。
- artifact 中包含集合结构的 `builtinStores`，且不包含开发态数据和内部连接密钥。
- `pnpm --filter datacenter typecheck` 通过。
- `go test ./...` 或最小相关 Go 包测试通过。

## 15. 实施拆分建议

本设计建议拆为五个实施阶段：

1. 调整内置运行库连接模型，取消同类型单例限制，生成不可变 `runtimeKey`。
2. 将关系库和时序库接入连接级 SQL 工作台 API。
3. 建立实时库 key 定义与当前值测试工作台。
4. 建立消息库 topic、变量和发布预览工作台。
5. 调整 artifact 契约输出和回归测试。

阶段之间可以独立验证。第一阶段完成后，用户已能创建多个同类型内置运行库；第二阶段完成后，关系/时序 SQL 工作台闭环完整；第三、四阶段完成后，实时和消息建模能力完整；第五阶段完成后，后续 node_agent 运行态初始化有稳定输入。
