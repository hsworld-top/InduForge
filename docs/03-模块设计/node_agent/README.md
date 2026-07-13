# NodeAgent 后端概览

## 模块定位

`runtime/node_agent` 是站点和原生采集节点的产品级管理代理，负责中心绑定、期望状态执行、K3s 资源应用、原生进程托管、健康聚合、升级和回滚。

## 部署形态

- Linux 服务站点：systemd 服务，运行在 K3s 外部。
- Linux 采集节点：systemd 服务，托管 Collector 子进程。
- Windows 采集节点：Windows Service，托管 Collector 和可选 OPC DA Worker。

## 核心能力

- 节点初始化、设备身份和中心绑定。
- K3s 版本与健康检查。
- 工程 Namespace、Release Loader、工作负载和 Ingress 资源应用。
- `native-processes.json` 生成和原生进程生命周期管理。
- Collector 能力包、工程采集配置、密钥和 WAL 目录管理。
- 发布、升级、回滚和故障恢复。
- 工程、Pod、基础设施和原生进程健康状态聚合。
- 本地诊断 API 和状态回传。

## 正式边界

NodeAgent 负责：

- 将平台 Desired State 转换为 K3s 资源或原生进程计划。
- 管理产品级部署生命周期和观察状态。

NodeAgent 不负责：

- 页面 Schema、查询、计算和报警语义。
- HSL、OPC UA、OPC DA 等工业协议执行。
- 处理工程 JetStream 业务消息。
- 解析和管理工程页面资源内容。

## 关键对象

- `SiteDesiredState`
- `ProjectReleaseDesiredState`
- `NativeCollectorDesiredState`
- `ObservedState`
- `ReleaseApplyResult`
- `native-processes.json`

## 质量关注点

- K3s 异常时 NodeAgent 仍必须可运行和诊断。
- 新 Release 失败不得停止旧工程版本。
- 原生采集升级失败必须恢复旧配置和进程。
- 状态回传必须区分期望状态、观察状态和最后成功 Release。
- WAL、密钥和工程版本目录必须按工程隔离。

## 关联文档

- [平台系统架构](../../01-产品与架构/平台系统架构.md)
- [节点管理与交付架构](../../02-系统设计/节点管理与交付架构.md)
- [工业采集架构](../../02-系统设计/工业采集架构.md)
- [NodeAgent 与运行系统协议](../../04-契约与规范/跨模块契约/node-agent-runtime-protocol.md)
- [Runtime 健康状态协议](../../04-契约与规范/跨模块契约/runtime-health-status-contract.md)
- [初始化流程](./初始化流程.md)
