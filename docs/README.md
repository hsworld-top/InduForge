# InduForge 文档中心

## 1. 使用原则

- `docs/` 只保留已确认的正式文档、稳定规范与模块说明。
- 开发过程文档、专项计划、分析稿、审计稿、测试清单、AI 辅助资料不放入仓库，统一使用外部协作空间管理。
- 当前代码、数据库、接口与已实现能力优先于过时文档；跨模块协议以 `docs/contracts/` 为准。

## 2. 正式文档入口

### 平台级正式文档

- [产品定义](./产品定义.md)
- [数据库设计](./database-design.md)
- [高层设计](./高层设计.md)
- [详细设计](./详细设计.md)
- [测试与质量策略](./测试与质量策略.md)
- [研发与交付视角模型](./研发与交付视角模型.md)

### 跨模块契约

- [发布态 Schema 契约](./contracts/designer-publish-schema.md)
- [IFP Manifest 契约](./contracts/ifp-manifest-contract.md)
- [NodeAgent 启动协议](./contracts/node-agent-runtime-protocol.md)
- [Runtime 健康状态协议](./contracts/runtime-health-status-contract.md)

### 模块概览与稳定说明

- [dev_core 后端概览](./backend/README.md)
- [IDE 管理端概览](./dev_ide/README.md)
- [数据中心概览](./datacenter/README.md)
- [data_service 概览](./data_service/README.md)
- [设计器概览](./designer/README.md)
- [NodeAgent 后端概览](./node_agent/README.md)
- [NodeAgent Front 概览](./node_agent_front/README.md)
- [Designer 组件开发](./designer/component-development.md)
- [Designer 层级约定](./designer/layer-order-convention.md)
- [Designer 放置与堆叠](./designer/placement-and-stacking.md)
- [Designer 尺寸约定](./designer/size-convention.md)
- [Datacenter 连接说明](./datacenter/connections.md)
- [Datacenter 查询说明](./datacenter/queries.md)
- [Datacenter MQTT 自动发现](./datacenter/mqtt-auto-discovery.md)
- [Datacenter 数据点设计](./datacenter/datapoint-design.md)
- [Datacenter Compute/Alarm 设计](./datacenter/compute-alarm-design.md)
- [NodeAgent 初始化流程](./node_agent/初始化流程.md)

## 3. 过程文档入口

- 本仓库不再维护开发过程文档与 AI 辅助资料索引。

## 4. 文档维护规则

- 未确认的规划、专项计划、分析稿、审计稿、测试清单不要放在 `docs/`。
- 这类资料统一迁移到团队外部协作空间，不在仓库内沉淀。
- 正式文档发生设计取舍或结构调整时，先讨论结论，再更新正文。
