# IF 内置运行库设计

## 背景

数据中心需要为没有自有数据基础设施的用户提供平台内置数据能力。用户在设计工程时，应看到稳定的 InduForge 能力名称，而不是直接面对平台内部使用的数据库、缓存或消息组件。

当前 `scripts/` 已经把开发环境、离线安装包和生产拓扑统一到 InduForge 产品体系命名：

- `meta-store`：承载平台元数据、关系数据与时序能力。
- `cache-store`：承载实时值、短期状态与缓存能力。
- `message-hub`：承载 MQTT/WebSocket 等消息接入能力。
- `object-store`：承载工程包、设计资源和运行产物等对象存储能力。该能力属于平台资源与设计中心边界，不作为数据中心接入源暴露。
- `edge`：对外唯一入口，内部服务通过 Docker 网络通信。

因此，内置运行库文档也需要按产品能力描述，而不是按底层组件描述。底层技术可以作为实现说明保留，但不得成为用户配置入口。

数据中心本轮只纳入 `IF关系库`、`IF时序库`、`IF实时库`、`IF消息库`。`IF对象库` 不属于数据中心接入源、数据点来源或 `data_service` 内置运行库模型；它由设计中心、发布链路或平台资源管理能力承载。

## 设计目标

1. 向用户提供统一的内置运行库入口：`IF关系库`、`IF时序库`、`IF实时库`、`IF消息库`。
2. 禁止用户连接平台系统库、系统缓存 DB 或内部消息服务。
3. 开发态 SQL、时序、实时值和消息调试必须真实执行，便于提前发现运行态问题。
4. 运行态由节点侧工程环境提供独立资源；平台侧开发资源只用于设计、预览和调试。
5. 开发态数据默认不发布到运行态，发布产物只携带结构、查询、时序表、保留策略、消息主题和数据契约。
6. 对外配置和交付文档使用 `IF_*` 能力命名，不暴露底层组件名。

## 核心概念

| 名称     | 技术底座            | 主要用途                                    | 开发态实现                                 | 运行态实现                  |
| -------- | ------------------- | ------------------------------------------- | ------------------------------------------ | --------------------------- |
| IF关系库 | meta-store 关系能力 | 用户业务表、页面表单数据、普通 SQL 查询     | `IF_META_STORE_DEV_DATA_DB` 内项目 schema  | 工程运行环境内关系库 schema |
| IF时序库 | meta-store 时序能力 | 数据点历史、趋势查询、计算与报警采样历史    | `IF_META_STORE_DEV_DATA_DB` 内时序 schema  | 工程运行环境内时序 schema   |
| IF实时库 | cache-store         | 当前值、短期缓存、实时状态、订阅状态        | `IF_CACHE_STORE_DEV_RT_DB` + 项目 key 前缀 | 工程运行环境内实时命名空间  |
| IF消息库 | message-hub         | MQTT/WebSocket 消息接入、模拟数据、设备主题 | 开发环境消息中心 + 项目 topic 前缀         | 工程运行环境内消息主题      |

## 总体架构

```text
平台开发态
├─ IF元数据能力
│  ├─ if_core       -> 控制面与设计中心系统库
│  ├─ if_data       -> 数据中心系统库
│  └─ if_dev_data   -> IF关系库 / IF时序库开发沙箱
│
├─ IF缓存能力
│  └─ key prefix: <IF_DEV_CACHE_KEY_PREFIX>:<projectId>:*
│
├─ IF消息能力
│  └─ topic prefix: <IF_MESSAGE_HUB_TOPIC_PREFIX>/<projectId>/*
│
└─ 数据中心
   ├─ SQL 工作台真实执行 SQL
   ├─ 时序趋势、计算、报警引用开发态查询结果
   ├─ MQTT 模拟数据和消息调试写入 IF消息库
   └─ 发布时生成运行态 artifact

节点运行态
├─ relation/time-series store
│  ├─ app schema -> IF关系库
│  └─ ts schema  -> IF时序库
│
├─ realtime store
│  └─ IF实时库
│
├─ message store
│  └─ IF消息库
│
└─ node-agent
   ├─ 按 artifact 初始化运行库结构
   ├─ 执行采集、计算、报警和页面运行逻辑
   └─ 写入本工程内置运行库
```

## 开发态沙箱

开发态需要独立于平台系统库的沙箱资源。当前开发环境模板使用 `IF_META_STORE_DEV_DATA_DB=if_dev_data` 作为开发态运行数据承载库，并在该库内按工程隔离 schema：

```text
if_dev_data
├─ p_<projectId>_app
└─ p_<projectId>_ts
```

`p_<projectId>_app` 用于 IF关系库开发态 SQL 测试。`p_<projectId>_ts` 用于 IF时序库开发态建表、样例数据、趋势 SQL 与聚合 SQL 测试。

开发态实时数据不直接暴露裸连接，使用项目级 key namespace：

```text
<IF_DEV_CACHE_KEY_PREFIX>:<projectId>:rt:<key>
<IF_DEV_CACHE_KEY_PREFIX>:<projectId>:cache:<key>
<IF_DEV_CACHE_KEY_PREFIX>:<projectId>:state:<key>
```

开发态消息主题使用项目级 topic namespace：

```text
<IF_MESSAGE_HUB_TOPIC_PREFIX>/<projectId>/mock-data
<IF_MESSAGE_HUB_TOPIC_PREFIX>/<projectId>/device/<deviceId>/telemetry
<IF_MESSAGE_HUB_TOPIC_PREFIX>/<projectId>/event/<eventName>
```

开发态数据默认只用于调试，不进入发布产物。若后续需要随工程发布样例数据，应单独设计“种子数据”能力，并要求用户显式选择。

## SQL 工作台执行规则

SQL 工作台必须真实执行用户 SQL，但执行上下文必须由平台控制，避免用户访问系统库或其他工程数据。

用户编写 SQL 时不应依赖开发态 schema 名：

```sql
select * from orders limit 10;
```

开发态执行时由服务端注入 search path：

```sql
set search_path to p_<projectId>_app, public;
select * from orders limit 10;
```

运行态执行时使用同一份查询定义，但切换到运行态 schema：

```sql
set search_path to app, public;
select * from orders limit 10;
```

IF时序库同理，开发态使用 `p_<projectId>_ts`，运行态使用 `ts`。

SQL 工作台应限制以下行为：

- 禁止连接或引用平台系统库。
- 禁止访问其他工程 schema。
- 禁止 `DROP DATABASE`、`CREATE DATABASE`、`ALTER SYSTEM` 等高危语句。
- 限制执行超时、最大返回行数和结果集大小。
- 查询、建表和迁移操作使用受限角色执行。
- 参数化执行业务变量，避免拼接用户输入。

## 消息访问规则

IF消息库用于工程内置消息接入，不等同于外部 MQTT 接入源。

- 用户不填写平台内部消息服务地址。
- 数据中心按工程生成 topic 前缀，并只允许访问本工程 topic。
- 模拟数据脚本默认发布到 `induforge/mock-data`，正式工程应映射到工程级 topic。
- 发布 artifact 时保存 topic 约定、消息格式、订阅关系和数据点绑定。

对象能力不属于数据中心接入源。本轮不在数据中心展示 `IF对象库`，不把对象存储作为数据点来源，也不在 `data_service` 内置运行库模型中保存对象库配置。工程资源、设计资源、发布产物和导入导出文件由设计中心、发布服务或平台资源管理模块处理。

## 运行态初始化

发布产物需要携带内置运行库契约，由节点侧 `node_agent` 在工程启动或升级时执行初始化。

示例结构：

```json
{
  "builtinStores": {
    "relation": {
      "enabled": true,
      "schema": "app",
      "ddlVersion": "2026-05-23.1"
    },
    "timeseries": {
      "enabled": true,
      "schema": "ts",
      "extension": "timeseries",
      "retentionDays": 30,
      "ddlVersion": "2026-05-23.1"
    },
    "realtime": {
      "enabled": true,
      "namespace": "rt"
    },
    "message": {
      "enabled": true,
      "topicPrefix": "project"
    }
  }
}
```

节点侧关系/时序初始化内容包括：

- 创建 `app` schema。
- 创建 `ts` schema。
- 启用时序扩展。
- 创建普通业务表。
- 创建时序表。
- 创建 `if_schema_migrations`，记录当前工程库结构版本。
- 创建运行态受限角色。

节点侧实时初始化内容包括：

- 确认工程命名空间。
- 初始化必要的当前值、状态或缓存 key 约定。
- 配置持久化与过期策略。

节点侧消息初始化内容包括：

- 确认工程 topic 前缀。
- 初始化订阅关系、消息格式和数据点映射。
- 限制工程只能访问自身 topic。

## 权限与隔离

平台系统库和内部基础设施永远不允许作为用户接入源。禁止列表至少包括：

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
```

开发态推荐角色：

| 角色            | 用途                                  |
| --------------- | ------------------------------------- |
| if_dev_migrator | 创建 schema、表、索引、扩展和迁移记录 |
| if_dev_runtime  | 执行 SQL 工作台允许的读写 SQL         |
| if_dev_readonly | 预览、查询列表和只读调试              |
| if_dev_message  | 访问本工程开发态 topic                |

运行态推荐角色：

| 角色                | 用途                           |
| ------------------- | ------------------------------ |
| if_runtime_migrator | 节点侧初始化与升级库结构       |
| if_runtime_app      | 采集、计算、报警等运行逻辑读写 |
| if_runtime_readonly | 诊断、趋势查询和只读预览       |
| if_runtime_message  | 访问本工程运行态 topic         |

## 接入源界面表达

内置运行库应作为独立类别展示，不与外部数据库、消息连接表单混在一起。

```text
新建接入源

内置运行库
[ IF关系库 ]  随工程运行环境部署，用于业务表和普通 SQL
[ IF时序库 ]  随工程运行环境部署，用于数据点历史和趋势
[ IF实时库 ]  随工程运行环境部署，用于当前值和实时缓存
[ IF消息库 ]  随工程运行环境部署，用于设备消息和模拟数据

外部数据源
[ PostgreSQL ] [ MySQL ] [ SQL Server ] [ Redis ] [ MQTT ] ...
```

用户选择 IF 内置运行库时，平台创建的是工程级内置资源定义，而不是让用户填写平台系统连接串、服务地址或访问密钥。

## 与外部接入源的区别

| 类型            | 用户是否填写连接串 | 开发态是否真实执行 | 运行态来源                 | 是否进入 artifact |
| --------------- | ------------------ | ------------------ | -------------------------- | ----------------- |
| IF关系库        | 否                 | 是                 | 工程运行环境内关系 schema  | 是                |
| IF时序库        | 否                 | 是                 | 工程运行环境内时序 schema  | 是                |
| IF实时库        | 否                 | 是，受限命名空间   | 工程运行环境内实时命名空间 | 是                |
| IF消息库        | 否                 | 是，受限 topic     | 工程运行环境内消息 topic   | 是                |
| 外部 PostgreSQL | 是                 | 是                 | 用户外部资源               | 是                |
| 外部 Redis      | 是                 | 是                 | 用户外部资源               | 是                |
| 外部 MQTT       | 是                 | 是                 | 用户外部资源               | 是                |

## 数据生命周期

- 开发态数据用于 SQL 工作台、趋势测试、消息调试、数据点预览和功能调试。
- 开发态数据默认不发布，不作为运行态初始数据。
- 运行态数据由节点侧采集、计算、报警和页面交互产生。
- IF时序库必须配置保留策略，默认值建议为 30 天，可按工程调整。
- IF实时库只承载当前值、短期状态和缓存，不承诺长期历史追溯。
- IF消息库只承载消息接入、订阅和转发，不作为长期历史存储。

## 与 scripts 的当前边界

- `scripts/dev/init-linux.sh` 负责创建开发态基础设施和 `if_core`、`if_data`、`if_dev_data`，不启动业务项目。
- `dev_core` 在开发环境启动时根据 `DB_AUTO_SCHEMA_SYNC=true` 同步 `if_core` 表结构和默认租户/管理员数据。
- 生产和离线环境保持 `DB_AUTO_SCHEMA_SYNC=false`，由安装脚本在安装阶段执行控制面数据库 bootstrap。
- `scripts/docker/docker-compose.dev.yml` 允许开发环境暴露基础设施端口，便于调试。
- `scripts/docker/docker-compose.offline.yml` 只使用 `induforge/*` 产品体系镜像名，对外只暴露 edge 入口。
- `scripts/release/build-offline-package-linux.sh` 负责生成离线包，镜像默认缓存到 `scripts/docker/images/`。
- `scripts/docker/images/` 只提交 `.gitkeep`，镜像 tar 不提交 Git。

## 待后续细化

1. 开发态沙箱资源创建时机：创建工程时自动初始化，还是首次添加内置运行库时初始化。
2. SQL 安全解析能力：需要确定允许语句白名单、DDL 边界和多语句执行策略。
3. 时序能力版本：开发态和节点侧运行态需保持底层版本一致。
4. 工程备份与迁移：需要定义运行态数据卷、实时持久化、消息持久化和时序数据保留策略。
5. 种子数据能力：如需发布样例数据，应独立建模并要求用户显式选择。
