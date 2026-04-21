# 数据中心概览

## 模块定位

- `datacenter` 是平台数据工作台前端，为项目提供数据连接、查询、数据点、协议接入、计算调试和数据消费入口。

## 核心能力

- 数据连接配置与管理
- 查询定义、执行与复用入口
- 数据点定义、查看与调试
- MQTT、协议接入与数据工作台操作
- 计算与数据调试入口

## 正式边界

- `datacenter` 是平台数据工作台前端，不承担平台侧数据域后端实现。
- `datacenter` 通过 `data_service` 获取平台侧与开发态数据域能力。
- `datacenter` 不承担页面渲染、工程聚合和节点托管职责。

## 关键交付件

- 数据连接配置
- 查询定义
- 数据点定义
- 协议接入配置
- 计算与调试入口

## Phase 1 协议消费边界

- `datacenter` 在 Phase 1 只应把 `relational`、`mqtt`、`kafka`、`http`、`websocket`、`redis` 视为正式协议范围。
- MQTT 是唯一要求走到“配置 + 短时 preview + artifact”完整闭环的样板协议。
- Kafka 在工作台里只保证配置与 `mock` 样本预览；HTTP/WebSocket/Redis 只保证配置与 artifact 契约，不承诺 Phase 1 独立 preview/runtime 行为。
- `opcua`、`s7`、`modbus`、`tdengine`、`opcda` 不属于 Phase 1 正式范围。即便后端保留路由占位，前端也不应把这些入口当成正式可用能力依赖。

## 质量关注点

- 数据工作台操作与 `data_service` 提供的正式能力保持一致。
- 数据对象定义与设计态、运行态消费契约保持连续。
- 关键数据工作台入口具备最小可回归验证能力。

## 入口与调试

### 正式入口

- 正式入口应由 `dev_ide` 宿主打开，数据中心在启动时向父窗口发送 `APP_BOOTSTRAP_REQUEST`，再消费宿主回发的 `APP_BOOTSTRAP_RESPONSE`。
- 正式入口不再依赖 URL 里的 `token`、`refreshToken`、`theme`、`pid`、`tenant` 作为主链路上下文。
- refresh 成功后数据中心会向宿主回发 `AUTH_REFRESHED`，refresh 失败或无可用 refreshToken 时会回发 `AUTH_EXPIRED`。

### 独立调试

- 需要单独调试数据中心时，使用 `/datacenter/debug`。
- `/datacenter/debug` 允许脱离 IDE 运行，但只用于本地调试；正式入口 `/datacenter/` 在缺少本地 token 或 projectId 时会回跳 `dev_ide`。
- 调试正式链路时，应从 `dev_ide` 内打开数据中心标签页，再检查 bootstrap 消息、主题同步和续租回传是否正确。

### `handoff` 与恢复

- 子应用被单独打开成浏览器标签页后，只要本地仍保留可复用的 token 与工程上下文，就允许继续独立运行；缺少上下文时仍会依赖 `handoff` 回附 IDE。
- `handoff` 只保存恢复映射，不保存 token 等敏感信息；丢失或过期时会降级回 IDE 首页。
- 子应用内部组件应优先从本地上下文读取 `projectId`、`tenantId` 等工程信息，避免重新依赖旧的 URL 参数。

## 关联文档

- [产品定义](../产品定义.md)
- [高层设计](../高层设计.md)
- [详细设计](../详细设计.md)
- [测试与质量策略](../测试与质量策略.md)
- [data_service 概览](../data_service/README.md)
- [连接说明](./connections.md)
- [查询说明](./queries.md)
- [MQTT 自动发现](./mqtt-auto-discovery.md)
- [数据点设计](./datapoint-design.md)
- [Compute/Alarm 设计](./compute-alarm-design.md)
- [IDE 管理端概览](../dev_ide/README.md)
