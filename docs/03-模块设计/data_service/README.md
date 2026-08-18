# data_service 概览

## 模块定位

- `data_service` 是平台侧与开发态的数据域服务，负责承接开发态数据接入、查询、数据点、协议接入、计算、预览会话与数据域 API。

## 核心能力

- 数据连接管理
- 查询定义与执行
- 数据点定义与读写支撑
- 开发态历史存储配置、来源继承与单点覆盖
- MQTT 与多协议数据接入
- 计算单元与数据处理
- 预览会话与开发态数据支撑
- 数据域 API 与数据域存储能力

## Phase 1 协议范围

### 正式协议范围

- `relational`：正式支持配置、查询预览、数据点映射与 artifact 输出。
- `mqtt`：Phase 1 样板协议，正式支持配置、短时 preview、消息样本、Tag/Subscription 管理与 artifact 输出。
- `kafka`：正式纳入 Phase 1，提供配置能力、artifact 输出与基于临时 reader 的真实 topic 短时 preview。
- `http`：正式纳入 Phase 1，提供配置能力、artifact 输出与一次性 HTTP 请求 preview。
- `websocket`：正式纳入 Phase 1，提供配置能力、artifact 输出与短连接 WebSocket 消息 preview。
- `redis`：正式纳入 Phase 1，提供配置能力、artifact 输出与有限 key/value preview。

### Phase 2 工业协议配置范围

- `opcua`
- `s7`
- `modbus`
- `tdengine`
- `opcda`

这些协议已进入平台侧配置阶段：`opcua/s7/modbus/tdengine` 支持通过专用接口创建连接配置并进入 artifact 契约，`opcda` 当前提供发布合约字段校验。data_service 不承担工业协议长连接、轮询采集或节点侧驱动运行；真实连通、采集和诊断由后续节点侧运行器接管。

### 产物与 preview 边界

- `artifact v1` 输出平台侧配置契约，包含 `mqtt`、`protocols.kafka/http/websocket/redis` 与工业协议配置。
- `preview socket`、预览会话与统一协议 preview 只服务开发态调试，不承担长期采集。
- `kafka/http/websocket/redis` 的统一协议 preview 是一次性短任务，请求结束即释放连接，不创建节点侧运行任务。
- `opcua/s7/modbus/tdengine` 本轮只保存配置与 artifact，不在 data_service 内做真实工业协议 preview。

## 正式边界

- `data_service` 是平台侧与开发态数据域服务，不承担平台治理和发布部署编排职责。
- `data_service` 为 `datacenter`、`designer` 开发态预览和其他平台侧模块提供统一数据域能力。
- `data_service` 与运行态的 `runtime_data_service` 共享数据域契约与领域模型，但不是同一部署实体。

## 工业协议执行边界

- `data_service` 保存工业接入源、变量和采集计划，并编排开发态调试命令。
- 真实 OPC UA、S7、Modbus、OPC DA 连接和长期采集迁移到节点侧 Collector。
- 中心允许在没有采集节点时完成全部设计态建模和 Artifact 生成。
- 详细设计见 [工业采集架构](../../02-系统设计/工业采集架构.md)。

## 关键对象与接口域

- 数据连接：connections
- 查询：queries
- 数据点：datapoints
- 历史存储：history-storage（仅配置，不执行运行态写入或建表）
- MQTT：mqtt
- 协议接入：wave1 / wave2
- 预览会话：preview sessions
- 计算能力：compute units

历史存储按接入源、工业采集连接或数据点保存配置。来源配置自动作用于后续新增点，单点可沿用、关闭或自定义；
写入方式固定为每次采样、间隔末值、变化保存和周期快照，目标限定为本工程 IF 时序库与 TDengine。工程快照包含
这些开发态配置和目标绑定，发布工件与本轮运行态链路不包含历史存储配置。

## 质量关注点

- 平台侧数据域定义与运行态数据域定义保持连续。
- 连接、查询、数据点、协议和计算能力具备稳定集成验证。
- 预览会话和开发态数据链路不因局部接口变更而失效。

## 关联文档

- [产品定义](../../01-产品与架构/产品定义.md)
- [平台整体架构](../../01-产品与架构/平台系统架构.md)
- [工业采集架构](../../02-系统设计/工业采集架构.md)
- [测试与质量策略](../../05-研发与交付/测试与质量策略.md)
- [数据中心概览](../datacenter/README.md)
