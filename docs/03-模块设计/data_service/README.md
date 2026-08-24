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
- 报警开发态配置、工程通知、报警历史设置与节点配置回写接收
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
- 报警配置：alarm-items / alarm-groups / alarm-settings / alarm-history-settings / alarm-channels

数据点可独立保存字符串 Key/默认值形式的开发态自定义属性。属性进入快照、Artifact 与开发契约，计算脚本可通过
`ctx.datapoint.meta(path).attributes` 读取；采集来源刷新数据点时不覆盖这些属性。开发态默认值不承担节点运行值持久化，
运行态赋值与中心配置相互隔离。

计算触发统一保存为手动、周期、每日、每周或数据点变化配置；每日/每周使用 IANA 时区，并可在保存前预览未来五次计划时间。`data_service` 不承担节点调度，开发态脚本只通过独立 `compute_sandbox` 调试，沙箱不可用时禁用调试且不回退宿主进程。开发态 SDK 仅暴露预取的数据点值/元数据和已声明只读查询。

外部关系库、MQTT、Kafka、HTTP、WebSocket、Redis 和 TDengine 的敏感值统一进入 `data_connection_secrets` AES-GCM 密钥表；API 读取只返回密钥状态，更新使用 `secrets` 与 `clearSecretKeys`。快照只往返密文和密钥版本，Artifact 只输出密钥引用。TDengine 采用结构化 `ws/wss` 配置与独立只读运行时，提供对象分类分页、结构、数据预览、只读 SQL、保存查询和查询数据点。

历史存储按接入源、工业采集连接或数据点保存配置。来源配置自动作用于后续新增点，单点可沿用、关闭或自定义；
写入方式固定为每次采样、间隔末值、变化保存和周期快照，目标限定为本工程 IF 时序库与 TDengine。工程快照包含
这些开发态配置和目标绑定，发布工件与本轮运行态链路不包含历史存储配置。

报警以 `data_alarm_items` 为事实来源，一条普通报警项只关联一个数据点，报警项 ID 同时是未来节点运行时的稳定报警身份。同一点可以分别配置越限、变化率、偏差、离线等多条报警；规范化显示名称和触发指纹在同一点内分别唯一。多点创建在一个事务中生成多条独立报警项，后续可分别维护。数值分级越限使用 `highest_matching`，一组高/高高/低/低低等级只产生一个当前生效等级。组合报警保存至少两个输入点、唯一别名、表达式和一个结果条件。通知采用工程默认与报警项覆盖，外部渠道密钥使用 AES-GCM 密文保存。工程级报警历史默认开启、保留 30 天并保存通知投递记录。工程快照与 Artifact 使用 `alarm.item.v1`，Artifact 只携带密钥引用并始终包含有效 `historyStorage`。本模块还提供 ID/筛选范围批量修改删除，以及普通报警 Excel 模板、导出、预览和事务导入；不执行节点侧报警判断、实例处置、历史写入或通知投递。

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
