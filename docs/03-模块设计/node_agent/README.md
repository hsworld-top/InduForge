# NodeAgent 模块概览

## 定位

`runtime/node_agent` 是安装在**外部物理节点**上的独立节点代理。中心安装自带的 K3s Server
由控制面直接映射为逻辑中心内置节点，不安装 NodeAgent。额外服务器通过 NodeAgent 主动访问
Center；Center 不反向连接 Agent。

当前控制面用户模型包含物理节点、运行环境、基础服务和工程部署。运行环境可以包含一台或多台
Linux 标准运行节点；单节点只是一个成员的特例。用户只管理物理节点和服务分配，不接触底层实现：

- `project_entry`：Project Gateway 与 Runtime API；静态 `client/` 是 Gateway 托管的文件，
  不是服务。
- `data_runtime`：RuntimeEngine。
- `collector`：可选 Industrial Collector。

Linux 包内的 root Hostd 以 K3s `v1.36.4+k3s1` 作为固定运行底座：外部 Linux 节点只作为
工作节点加入中心安装时已经建立的 Server。中心控制面应用各运行环境的八项固定基础服务清单。
运行环境不拥有独立 K3s；它只是命名空间、服务和调度边界，同一节点可供多个环境使用。K3s、Pod
和命名空间属于内部实现，不是用户概念。基础数据库当前仍是单实例；主备、集群和自动故障转移尚未交付。

## 已交付能力

- 一次性接入码领取、审批后心跳和 Agent 主动命令轮询；中心命令只含服务期望状态/代次，不含
  shell、路径、参数、环境变量、下载 URL 或 Secret。
- 固定 schema 的工作节点加入计划、离线资产双重摘要校验、真实工作负载观测、故障回传，
  以及受边界约束的完整卸载清理。
- 本机声明式服务组 allowlist、进程 PID 持久化、回环 HTTP 健康探测和 SIGTERM drain。
- Supervisor 的部署级启动原语 `ActivateWorkload`：按 serviceId 保存独立配置，串行启停，
  同代次幂等，候选启动失败后恢复旧配置，并保留仍在运行的旧进程 PID 和 generation。
  该原语已有本机进程测试，尚未接入中心的原生 Collector 配置下发链路。
- `ReleaseStore` 安装原语：输入不可信的 tar.zst 流、预期 outer SHA-256、受信 Ed25519 公钥和
  key ID；它限额落盘到 `incoming/*.partial`，安全解包，验证签名/摘要，内容寻址保存并原子写入
  `current.json`。

`ReleaseStore` 不会发起网络请求、加载信任根、读取 DeploymentBinding/Secret、创建只读挂载、
启动进程或回滚进程。它是正式物料链的基础原语，**不是**完成的工程部署或端到端发布。

## Release 文件和签名

输入 tar.zst 的根目录直接放标准 Release 文件，例如：

```text
release-manifest.json
client-assets.tar.zst
runtime-artifact.tar.zst
health-contract.json
resource-recommendation.json
schema-plan.json
sbom.cdx.json
collector-artifact.tar.zst       # 可选
checksums.json
signature.sig
```

`release-manifest.json` 的 `supplyChain.signingKeyId` 必须与安装输入的本地受信 key ID 相同。
`checksums.json` 的冻结格式为：

```json
{
  "schemaVersion": "release-checksums.v1",
  "files": [{ "path": "release-manifest.json", "sha256": "sha256:<64 lowercase hex>", "size": 123 }]
}
```

`files` 必须按 `path` 严格升序，覆盖以上所有标准 Release 文件（可选 Collector 仅在 Manifest
声明时出现），且不得列出 `checksums.json` 或 `signature.sig`。`signature.sig` 是对**精确原始**
`checksums.json` 字节签名的 64 字节 Ed25519 原始签名。安装器只接受这些根目录普通文件，拒绝
目录、绝对路径、`..`、反斜杠、符号/硬链接、设备、FIFO、重复项和超限解压。

成功安装的内容目录为 `releases/sha256-<outer-digest>`；`current.json` 使用
`release-pointer.v1`，含 `releaseDigest`、相对 `releaseDir` 和 `activatedAt`。失败绝不切换
current，也不应据此报告服务运行。

## 明确未交付项

- 工程 Release 已能按节点主动下载 Binding 和同源流并完成本地信任校验，但 Secret 装配与正式
  launcher 尚未交付，因此还不能形成端到端工程运行。
- 每 deployment 的端口、Secret、资源隔离与原生 Collector 调和接线；Supervisor 部署级进程隔离原语已具备。
- Linux RuntimeEngine 所需的真实只读 mount，以及启动失败后的服务级回滚编排。
- Windows 的 Gateway/Runtime API/RuntimeEngine。当前 Windows 包只声明 Collector，其他服务组
  必须 fail-closed。

## 关联文档

- [NodeAgent 与运行系统协议](../../04-契约与规范/跨模块契约/node-agent-runtime-protocol.md)
- [Runtime 健康状态协议](../../04-契约与规范/跨模块契约/runtime-health-status-contract.md)
- [运维体系实现与验收](../../06-运维与安全/运维体系实现与验收.md)
