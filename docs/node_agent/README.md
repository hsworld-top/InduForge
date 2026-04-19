# NodeAgent 后端概览

## 模块定位

- `runtime/node_agent` 是节点侧托管后端，负责发布制品落地、进程托管、运行单元编排、本地状态维护与节点 Web API。

## 核心能力

- 节点初始化与中心绑定
- 发布制品落地与版本切换
- `runtime_gateway`、`runtime_engine` 与 `runtime_data_service` 组的托管
- 本地状态、日志、健康检查与状态回传
- 节点侧本地 API

## 正式边界

- `node_agent` 负责托管与编排，不承担页面语义解释和平台治理职责。
- `node_agent` 不负责平台侧数据域能力。
- 页面运行由 `runtime_engine` 承担，工程级运行态数据域由 `runtime_data_service` 组承担。

## 关键对象与交付件

- `ProjectRuntimeUnit`
- `RuntimeGatewayRegistry`
- 节点命令执行结果
- 节点运行状态与日志

## 质量关注点

- 运行组合的启动、切换、摘流、加流与恢复必须稳定。
- 节点状态、健康信息和运行结果必须可追踪、可回传。
- 节点本地路由和托管状态不能与真实运行状态脱节。

## 关联文档

- [产品定义](../产品定义.md)
- [高层设计](../高层设计.md)
- [详细设计](../详细设计.md)
- [测试与质量策略](../测试与质量策略.md)
- [NodeAgent 启动协议](../contracts/node-agent-runtime-protocol.md)
- [Runtime 健康状态协议](../contracts/runtime-health-status-contract.md)
- [初始化流程](./初始化流程.md)
