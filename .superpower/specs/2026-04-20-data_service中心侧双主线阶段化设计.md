# data_service 中心侧双主线阶段化设计

## 1. 背景与问题陈述

此前围绕“数据开发服务”的讨论，已经形成过一版面向完整目标态的大设计稿，但当前工程现状已经发生实质变化：

1. `dev_core` 已切换到 PostgreSQL。
2. `dev_core` 本地数据域实现已经下线，平台侧数据域主入口已收敛到 `data_service`。
3. `data_service` 已完成 PostgreSQL 迁移、核心路由装配、连接/查询/数据点/MQTT 管理/项目快照/preview session/compute 等主干能力。
4. `datacenter`、`designer`、`dev_core` 已开始依赖 `data_service` 作为中心侧数据域正式服务。

但当前系统仍存在两个明显问题：

1. 设计稿与实现状态脱节。旧稿将“终局目标、阶段目标、节点侧设想、协议全景”混在一起，无法准确指导下一步落地。
2. 当前代码虽然完成了控制面主干能力，但开发态链路与协议运行态仍存在多处降级实现，例如：
   - MQTT 连接测试、启停、状态更多是配置校验和状态字段流转，不是完整 runtime。
   - MQTT Tag 当前值仍可能回落为 `defaultValue`，不是真实短时订阅值。
   - preview 只有 session create/heartbeat/delete 的基础能力，缺少完整 read/subscribe/unsubscribe。
   - 数据点 `writeValue`、`usages` 仍是兼容实现，不是最终可用实现。
   - protocol wave1/wave2 当前主要完成配置入库，不等于真实运行态接入。

因此需要一份新的正式 spec，明确：

1. 中心侧 `data_service` 的正式职责边界。
2. 与 `dev_core`、`datacenter`、`designer` 的联动契约。
3. “开发态链路闭环”和“协议能力分层推进”这两条主线的终局目标。
4. 当前已完成基线与 Phase 1/Phase 2 的阶段划分。

## 2. 目标与非目标

### 2.1 目标

本设计面向中心侧数据开发体系，目标是建立一个职责清晰、阶段可交付的中心侧数据服务体系：

1. `data_service` 作为中心侧唯一正式数据域服务，负责数据配置、开发态预览、SQL 执行预览、采集数据查询与数据产物输出。
2. `dev_core` 保持平台控制面职责，通过契约方式消费 `data_service`，不再恢复任何本地数据域实现。
3. `datacenter` 与 `designer` 统一通过 `/api/v1/data` 语义消费中心侧数据能力，优先保持现有前端语义稳定。
4. 中心侧产出一份节点侧可消费的数据产物，节点侧根据产物独立运行。
5. 中心侧通过“轻量会话预览”支持开发与设计阶段调试，但不演变成运行态采集服务。

### 2.2 非目标

以下能力明确不属于本 spec 范围，也不应在后续实现中被偷偷引回中心侧：

1. 中心侧不是节点侧运行态数据服务，不承担长期采集、持续订阅、持续调度执行。
2. 中心侧不支持高频实时数据流场景，不承诺毫秒级连续流处理能力。
3. 中心侧不支持设备写回，不支持 PLC/OPC/MQTT 设备侧下写。
4. 中心侧 preview 不等于生产运行，不允许把 preview session 当成长时间运行采集任务使用。
5. 中心侧不保证所有协议在每个阶段都拥有同等级 preview 能力。
6. 节点侧数据服务的内部架构、内部缓存、节点运行时编排不属于本设计范围。

## 3. 系统边界与角色

### 3.1 `data_service`

`data_service` 是中心侧正式数据域服务，负责四类能力：

1. 配置层：连接、查询、数据点、协议配置、MQTT 结构管理。
2. 预览层：短时 session、短时订阅、最近值/消息样本、SQL 执行预览、采集数据查询。
3. 产物层：生成节点侧可消费的数据产物。
4. 契约层：对 `dev_core`、`datacenter`、`designer` 暴露稳定的中心侧数据 API。

### 3.2 `dev_core`

`dev_core` 是平台控制面，不再实现任何本地数据域逻辑，仅负责：

1. 统一认证与 capability 透传。
2. 工程、成员、发布、部署等平台控制面能力。
3. 在导入、导出、发布链路中调用 `data_service` 的快照和产物能力。

### 3.3 `datacenter`

`datacenter` 是中心侧数据配置前端，主要负责：

1. 数据连接、查询、数据点、协议配置的管理界面。
2. 连接测试、SQL 预览、采集结果查看、轻量会话调试。
3. 消费 `/api/v1/data` 兼容接口，不直连节点侧服务。

### 3.4 `designer`

`designer` 是页面设计器前端，主要通过中心侧数据服务获得：

1. 设计期数据点列表与状态。
2. 设计期 preview session。
3. 设计期短时订阅与最近值预览。

### 3.5 节点侧产物消费方

节点侧只消费中心侧产物，不与中心侧 `data_service` 形成运行时协同依赖。节点侧职责是：

1. 接收中心侧发布产物。
2. 按产物独立运行连接、采集、映射、页面运行引擎交互。
3. 与节点侧前端和页面运行引擎交互。

节点侧不是本 spec 的内部实现对象，只是产物消费方。

## 4. 双主线目标

### 4.1 主线 A：开发态链路闭环

开发态链路闭环的目标是：在中心侧完成“配置 -> 预览 -> 调试 -> 产物生成”的完整闭环，而不越界到运行态长期服务。

这条主线要求：

1. `datacenter` 能完成连接配置、SQL 预览、采集结果查询、协议消息样本查看。
2. `designer` 能创建短时 preview session，并获取页面调试需要的最近值与短时订阅结果。
3. `dev_core` 能在导入/导出/发布链路中稳定获取项目数据快照与产物。
4. 整条链路只提供轻量会话预览，不承诺高频实时，也不提供设备写回。

### 4.2 主线 B：协议能力分层推进

协议能力不再按“大而全协议矩阵一次做完”推进，而采用：

1. MQTT 作为完整样板协议。
2. 其他协议按统一能力层推进，而不是每种协议各写一套独立目标。

协议能力层统一分成：

1. 配置能力：连接配置、参数校验、对象管理。
2. 预览能力：连接测试、最近样本、短时查询或短时订阅。
3. 产物能力：可进入节点侧产物。
4. 扩展能力：仅在后续阶段逐步增加，不作为 Phase 1 强交付项。

## 5. 能力分层模型

### 5.1 配置层

配置层是中心侧基础层，必须稳定、可审计、可发布，包含：

1. `connections`
2. `queries`
3. `datapoints`
4. `mqtt` 的连接/订阅/TagGroup/Tag
5. `kafka` / `http` / `websocket` / `redis` 配置
6. 工业协议配置对象

配置层必须满足：

1. PostgreSQL 持久化。
2. 参数校验明确。
3. 可生成项目快照。
4. 可参与产物输出。

### 5.2 预览层

预览层只服务开发态，不等价于运行态。统一能力边界如下：

1. 允许创建短时 session。
2. 允许短时 read/subscribe/unsubscribe。
3. 允许缓存最近值、最近消息样本、最近查询结果。
4. 允许执行 SQL 预览与采集数据查询。
5. 不支持长期保持连接。
6. 不支持高频连续流。
7. 不支持设备写回。

### 5.3 产物层

产物层负责把中心侧配置整理成节点侧可消费对象。中心侧只负责生成，不负责运行。

产物层必须满足：

1. 产物可独立描述项目数据运行所需的对象与关系。
2. 产物结构具备版本号。
3. 节点侧只依赖产物，不依赖中心侧在线会话。

### 5.4 契约层

契约层负责与上游和前端联动：

1. Phase 1 兼容现有 `/api/v1/data` 语义。
2. 前端优先无感迁移。
3. Phase 2 再清理接口冗余与历史兼容字段。

## 6. 协议策略

### 6.1 Phase 1 正式协议

Phase 1 正式交付的协议范围为：

1. `relational`
2. `mqtt`
3. `kafka`
4. `http`
5. `websocket`
6. `redis`

### 6.2 Phase 2 扩展协议

Phase 2 再推进：

1. `opcua`
2. `s7`
3. `modbus`
4. 其他工业协议

### 6.3 样板协议策略

MQTT 作为完整样板协议，要求在中心侧同时具备：

1. 配置对象完整。
2. 轻量预览能力完整。
3. 数据点映射完整。
4. 产物输出完整。

其他协议按统一能力层逐步推进，不要求在 Phase 1 达到 MQTT 同等级深度。

## 7. API 兼容策略

### 7.1 Phase 1

Phase 1 的策略是“兼容优先”：

1. 路径语义优先保持 `/api/v1/data` 不变。
2. 返回结构优先保持前端现有语义。
3. 允许补字段，不轻易删字段。
4. `dev_core`、`datacenter`、`designer` 在 Phase 1 以平滑迁移为第一目标。

### 7.2 Phase 2

Phase 2 再做接口收敛：

1. 清理历史兼容字段。
2. 收敛重复路由。
3. 明确 preview 与配置接口的边界。

## 8. Phase 1 产物契约

### 8.1 产物定位

Phase 1 产物是中心侧发布给节点侧的项目级数据产物。它描述：

1. 节点需要加载哪些连接。
2. 节点需要运行哪些查询、订阅、映射。
3. 页面运行引擎如何引用数据点路径。

### 8.2 顶层结构

Phase 1 产物采用项目级 JSON 对象，建议结构如下：

```json
{
  "version": "1.0",
  "projectId": "uuid",
  "generatedAt": "2026-04-20T12:00:00Z",
  "connections": [],
  "queries": [],
  "datapoints": [],
  "mqtt": {
    "connections": [],
    "subscriptions": [],
    "tagGroups": [],
    "tags": []
  },
  "protocols": {
    "kafka": [],
    "http": [],
    "websocket": [],
    "redis": []
  }
}
```

### 8.3 顶层字段说明

1. `version`：产物契约版本，Phase 1 固定为 `1.0`。
2. `projectId`：项目标识。
3. `generatedAt`：产物生成时间。
4. `connections`：通用连接定义，供关系库及统一连接消费。
5. `queries`：查询定义。
6. `datapoints`：统一数据点定义。
7. `mqtt`：MQTT 专属结构。
8. `protocols`：其他协议的结构化配置集合。

### 8.4 通用连接对象

```json
{
  "id": "uuid",
  "name": "main-db",
  "type": "relational",
  "status": "connected",
  "config": {}
}
```

字段要求：

1. `id`、`name`、`type` 必填。
2. `status` 为产物生成时的配置态状态，不代表节点最终运行状态。
3. `config` 为去控制面冗余后的运行所需配置。

### 8.5 查询对象

```json
{
  "id": "uuid",
  "connectionId": "uuid",
  "name": "query-users",
  "queryType": "sql",
  "config": {},
  "timeoutMs": 30000,
  "isEnabled": true
}
```

字段要求：

1. `connectionId` 必须引用 `connections.id`。
2. `queryType` Phase 1 以 `sql` 为主。
3. `config` 必须保留节点运行所需的 SQL 与参数定义。

### 8.6 数据点对象

```json
{
  "id": "uuid",
  "path": "mqtt.main.temp",
  "name": "temp",
  "sourceType": "mqtt.tag",
  "sourceId": "uuid",
  "dataType": "number",
  "refreshMode": "subscription",
  "status": "active"
}
```

字段要求：

1. `path` 在项目范围内唯一。
2. `sourceType` + `sourceId` 必须可回溯到源对象。
3. `status` 只描述配置可用性，不是节点侧最终运行状态。

### 8.7 MQTT 结构

`mqtt` 节点包含：

1. `connections`
2. `subscriptions`
3. `tagGroups`
4. `tags`

其中引用关系要求：

1. `subscriptions.connectionId -> mqtt.connections.id`
2. `tagGroups.subscriptionId -> mqtt.subscriptions.id`
3. `tags.subscriptionId -> mqtt.subscriptions.id`
4. `tags.groupId -> mqtt.tagGroups.id`

### 8.8 其他协议结构

Phase 1 对 `kafka`、`http`、`websocket`、`redis` 先定义对象级结构，不在本阶段展开到完整运行态字段矩阵。要求只有三条：

1. 每个对象必须有 `id`、`name`、`type`、`config`。
2. 必须能被节点侧识别和消费。
3. 必须可关联回 `datapoints` 或运行对象。

## 9. 阶段规划

### 9.1 Phase 0：已完成基线

Phase 0 是当前已经完成的基础面，包含：

1. `dev_core` PostgreSQL 切换完成。
2. `dev_core` 本地数据域实现下线。
3. `data_service` PostgreSQL 迁移与核心路由装配完成。
4. `connections`、`queries`、`datapoints`、`mqtt` 管理、`project snapshot`、`preview session`、`compute` 主干已落地。
5. 当前验证已通过：
   - `go test ./...`
   - `go build ./...`

### 9.2 Phase 1：正式交付阶段

Phase 1 的目标不是把中心侧做成运行态，而是把中心侧开发链路和主协议能力做成正式可交付。

必须完成：

1. 开发态链路闭环：
   - `datacenter` 配置、测试、预览、查询主链路稳定。
   - `designer` preview session/read/subscribe 主链路稳定。
   - `dev_core` 快照和发布主链路稳定。
2. MQTT 样板协议闭环：
   - 真实短时测试、短时订阅、最近值/消息样本。
   - Tag/TagGroup/Subscription 与数据点映射稳定。
3. `relational + MQTT + Kafka + HTTP/WebSocket + Redis` 进入正式协议范围。
4. Phase 1 API 保持兼容优先。
5. Phase 1 产物结构稳定。

### 9.3 Phase 2：扩展与收敛阶段

Phase 2 再处理：

1. 工业协议配置与最小预览能力。
2. Phase 1 历史兼容接口收敛。
3. 产物契约进一步版本化。
4. 对部分协议补充更细的对象结构和运行边界。

## 10. 验收标准

### 10.1 Phase 1 验收

Phase 1 达成时，应满足以下标准：

1. `dev_core` 不再依赖任何本地数据域实现。
2. `datacenter` 能完成连接配置、SQL 预览、消息样本查看、采集数据查询。
3. `designer` 能通过 preview session 获取设计态需要的短时数据预览。
4. MQTT 不再用 `defaultValue` 伪装实时值主链路。
5. 中心侧产物能稳定提供给节点侧消费。
6. 不出现把中心侧 preview 当长期运行态服务的设计反向膨胀。

### 10.2 持续验证要求

至少保持以下验证可通过：

1. `go test ./...`
2. `go build ./...`
3. `dev_core` 与 `data_service` 的导入/导出/发布联调验证。
4. `datacenter` 与 `designer` 的 preview 主链路联调验证。

## 11. 风险与设计原则

### 11.1 当前主要风险

1. 前端仍可能依赖历史兼容语义，接口收敛不能过早推进。
2. 如果不明确非目标，中心侧很容易再次被拉向“伪运行态服务”。
3. 如果不定义产物版本，后续节点侧消费会逐步失控。
4. 如果不以 MQTT 做样板协议，其他协议会继续各写各的，难以收敛。

### 11.2 设计原则

1. 中心侧配置优先，运行态让位于节点侧。
2. 轻量预览优先，拒绝长期运行。
3. 兼容优先，但兼容是阶段性策略，不是永久负担。
4. 协议按能力层推进，不按幻觉式“大而全一次到位”推进。
5. 产物契约必须先稳定，再考虑更多协议扩展。

