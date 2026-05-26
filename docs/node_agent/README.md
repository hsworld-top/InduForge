# NodeAgent 后端概览

## 模块定位

- `runtime/node_agent` 是节点侧托管后端，负责发布制品落地、进程托管、运行单元编排、本地状态维护与节点 Web API。

## 核心能力

- 节点初始化与中心绑定
- 发布制品落地与版本切换
- `project_nginx`、`runtime_api` 与 `runtime_data_service` 组的托管
- 基于工程发布制品生成 `runtime-topology.json`、`docker-compose.runtime.yml` 或原生采集进程计划
- 按能力裁剪启动服务站点 compose 内的客户端引擎、数据引擎角色、IF 内置库和必要的内部组件
- 支持服务站点和采集站点两类运行拓扑；采集站点只启动采集子栈并上行到服务站点
- 支持 Windows Service 与 Linux systemd 形态的原生采集站点，由 `node_agent` 托管 collector 子进程和本地 WAL
- 为服务站点工程分配独立宿主机端口，并只将 `project_nginx` 映射到宿主机
- 本地状态、日志、健康检查与状态回传
- 节点侧本地 API

## 正式边界

- `node_agent` 负责托管与编排，不承担页面语义解释和平台治理职责。
- `node_agent` 不负责平台侧数据域能力。
- `node_agent` 不理解采集协议、查询语义、报警规则和计算脚本，只消费运行拓扑与健康状态。
- 页面运行由 `client_engine` 静态运行时承担，工程动态 API 由 `runtime_api` 承担，工程级运行态数据域由 `runtime_data_service` 组承担。

## 关键对象与交付件

- `ProjectRuntimeUnit`
- `ProjectEntryRecord`
- `RuntimeTopology`
- `RuntimeSite`
- `docker-compose.runtime.yml`
- `native-processes.json`
- 节点命令执行结果
- 节点运行状态与日志

## 质量关注点

- 运行组合的启动、切换、入口启停与恢复必须稳定。
- 原生采集站点的 Windows Service、systemd、子进程托管和 WAL 补投必须可观测、可恢复。
- 节点状态、健康信息和运行结果必须可追踪、可回传。
- 节点工程入口记录和托管状态不能与真实运行状态脱节。
- 按能力裁剪后的 compose 或原生进程计划必须可重放，便于回滚、离线排障和故障恢复。

## 关联文档

- [产品定义](../产品定义.md)
- [高层设计](../高层设计.md)
- [详细设计](../详细设计.md)
- [节点侧运行态数据引擎设计](../节点侧运行态数据引擎设计.md)
- [测试与质量策略](../测试与质量策略.md)
- [NodeAgent 启动协议](../contracts/node-agent-runtime-protocol.md)
- [Runtime 健康状态协议](../contracts/runtime-health-status-contract.md)
- [初始化流程](./初始化流程.md)
