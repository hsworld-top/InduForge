# NodeAgent 后端概览

## 模块定位

`runtime/node_agent` 是**每台宿主机一个**的产品级管理代理，负责主机身份、中心绑定、宿主机观测、
本机 K3s 生命周期、原生进程托管以及宿主机失联恢复。它运行在 K3s 外部，K3s 控制面或
site-controller 故障时仍必须能够诊断和修复本机。

站点级工程编排由 K3s 内的 `site-controller` 负责。两者不得合并成一个同时拥有宿主机 root
权限与整个集群管理权限的“Site NodeAgent”。

## 部署形态

- Linux 服务节点：每台主机一个 systemd 服务，运行在 K3s 外部。
- Linux 采集节点：systemd 服务，托管 Collector 子进程。
- Windows 采集节点：Windows Service，托管 Collector 和可选 OPC DA Worker。

## 核心能力

- 节点初始化、设备身份和中心绑定。
- CPU、内存、磁盘、网络、时钟、进程和本机故障观测。
- 本机 K3s server/agent 的安装、启停、版本、健康与恢复。
- `native-processes.json` 生成和原生进程生命周期管理。
- Collector 能力包、工程采集配置、密钥和 WAL 目录管理。
- NodeAgent 自身和原生进程的升级、回滚与故障恢复。
- 有界持久的心跳、资源摘要、操作结果和事件回传。
- 默认仅监听回环地址的本地诊断接口。

## 正式边界

NodeAgent 负责执行宿主机作用域的：

- `HostNodeDesiredState`、`InfrastructureHostDesiredState` 和 `NativeCollectorAssignmentDesiredState`。
- 将本机期望状态转换为系统服务、K3s 节点动作或原生进程计划。
- 回传 `HostNodeObservedState`、`NodeResourceSnapshot`、原生进程健康和操作结果。

NodeAgent 不负责：

- 工程 Namespace、Deployment、Job、Service、Ingress、Secret/PVC 的编排；这些属于 `site-controller`。
- 站点级 ProjectDeployment 状态机、Release 切换和分区所有权协调。
- 页面 Schema、查询、计算和报警语义。
- HSL、OPC UA、OPC DA 等工业协议执行。
- 处理工程 JetStream 业务消息。
- 解析和管理工程页面资源内容。

## 关键对象

- `HostNodeDesiredState`
- `InfrastructureHostDesiredState`
- `NativeCollectorAssignmentDesiredState`
- `HostNodeObservedState`
- `NodeResourceSnapshot`
- `NativeProcessHealth`
- `NodeOperationResult`
- `native-processes.json`

## 质量关注点

- K3s 异常时 NodeAgent 仍必须可运行和诊断。
- 中心断开时不影响已运行工程，遥测队列必须同时受时间和容量上限约束。
- 原生采集升级失败必须恢复旧配置和进程。
- 状态回传必须区分期望状态、观察状态、健康状态和数据新鲜度。
- NodeAgent 不保存中心用户密码，浏览器不保存长期节点凭据或私钥。
- WAL、密钥和版本目录必须按 `deploymentId` 隔离。

## 关联文档

- [平台系统架构](../../01-产品与架构/平台系统架构.md)
- [节点管理与交付架构](../../02-系统设计/节点管理与交付架构.md)
- [运维与节点运行态总体架构](../../02-系统设计/运维与节点运行态总体架构.md)
- [平台运维体系设计](../../06-运维与安全/平台运维体系设计.md)
- [工业采集架构](../../02-系统设计/工业采集架构.md)
- [NodeAgent 与运行系统协议](../../04-契约与规范/跨模块契约/node-agent-runtime-protocol.md)
- [Runtime 健康状态协议](../../04-契约与规范/跨模块契约/runtime-health-status-contract.md)
- [初始化流程](./初始化流程.md)
