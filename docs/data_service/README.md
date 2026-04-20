# data_service 概览

## 模块定位

- `data_service` 是平台侧与开发态的数据域服务，负责承接开发态数据接入、查询、数据点、协议接入、计算、预览会话与数据域 API。

## 核心能力

- 数据连接管理
- 查询定义与执行
- 数据点定义与读写支撑
- MQTT 与多协议数据接入
- 计算单元与数据处理
- 预览会话与开发态数据支撑
- 数据域 API 与数据域存储能力

## Phase 1 协议范围

### 正式协议范围

- `relational`：正式支持配置、查询预览、数据点映射与 artifact 输出。
- `mqtt`：Phase 1 样板协议，正式支持配置、短时 preview、消息样本、Tag/Subscription 管理与 artifact 输出。
- `kafka`：正式纳入 Phase 1，当前提供配置能力、artifact 输出，以及 `mock` 级 topic preview 样本。
- `http`：正式纳入 Phase 1，当前提供配置能力与 artifact 输出，不提供独立 preview/runtime 采集能力。
- `websocket`：正式纳入 Phase 1，当前提供配置能力与 artifact 输出，不提供独立 preview/runtime 采集能力。
- `redis`：正式纳入 Phase 1，当前提供配置能力与 artifact 输出，不提供独立 preview/runtime 采集能力。

### 非 Phase 1 正式范围

- `opcua`
- `s7`
- `modbus`
- `tdengine`
- `opcda`

这些协议和校验入口当前只保留 Phase 2 的接口边界与规划占位，不再作为 Phase 1 正式可交付能力对外宣称。若调用对应接口，服务端会显式返回“未纳入 Phase 1 正式范围”的错误，而不是继续落库形成误导。

### 产物与 preview 边界

- `artifact v1` 的正式协议区块只包含 `mqtt` 与 `protocols.kafka/http/websocket/redis`。
- `preview socket` 与短时会话只服务开发态调试，不承担长期采集。
- Kafka preview 当前返回 `mock` 样本，目的是给 `datacenter` 和联调链路提供稳定调试口径，而不是承诺完整运行态消费栈。

## 正式边界

- `data_service` 是平台侧与开发态数据域服务，不承担平台治理和发布部署编排职责。
- `data_service` 为 `datacenter`、`designer` 开发态预览和其他平台侧模块提供统一数据域能力。
- `data_service` 与运行态的 `runtime_data_service` 共享数据域契约与领域模型，但不是同一部署实体。

## 关键对象与接口域

- 数据连接：connections
- 查询：queries
- 数据点：datapoints
- MQTT：mqtt
- 协议接入：wave1 / wave2
- 预览会话：preview sessions
- 计算能力：compute units

## 质量关注点

- 平台侧数据域定义与运行态数据域定义保持连续。
- 连接、查询、数据点、协议和计算能力具备稳定集成验证。
- 预览会话和开发态数据链路不因局部接口变更而失效。

## 关联文档

- [产品定义](../产品定义.md)
- [高层设计](../高层设计.md)
- [详细设计](../详细设计.md)
- [测试与质量策略](../测试与质量策略.md)
- [数据中心概览](../datacenter/README.md)
