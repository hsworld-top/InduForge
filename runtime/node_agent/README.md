# InduForge NodeAgent

NodeAgent 是物理节点上的独立 Go 服务。它通过主动 HTTPS 轮询接入 Center；同机安装时，Center
和 NodeAgent 仍必须使用独立身份、目录、端口、进程和网络边界。

当前正式服务组为工程入口（Gateway + Runtime API）、数据运行（RuntimeEngine）和可选采集
（Collector）。静态前端文件由 Gateway 托管，不是 NodeAgent 可独立启动的服务。

Linux 标准节点还包含独立的 root Hostd。NodeAgent 只通过权限受限的 Unix socket 提交固定
`induforge.cluster-plan.v1` 和 `induforge.foundation-plan.v1` 声明；Hostd 负责校验离线资产、
安装或加入 K3s、应用固定基础服务清单并回传真实工作负载状态。Center 和普通 Agent 命令均不能
向 Hostd 注入 shell、宿主路径或任意镜像。

## 构建和验证

```bash
cd runtime/node_agent
make test
make vet
```

不应为验证而自行启动 NodeAgent 或任何 Runtime 子进程。

## 现行控制面边界

Agent 领取身份后主动请求 Center 心跳和命令。非本机 Center 只允许 HTTPS；中心命令只能调和
本节点已声明服务组的期望状态/代次，不能提供 shell、可执行文件、路径、参数、环境变量、下载
URL 或 Secret。默认配置的组件均为 `enabled: false`，安装能力不等于已部署或运行。

## 运行环境基础设施

当前节点包固定携带 K3s `v1.36.4+k3s1` 及 amd64 / arm64 对应的 airgap 镜像，并携带
TimescaleDB、Redis、EMQX、NATS JetStream、SeaweedFS、Nginx 基础镜像。中心安装流程指定的内置
节点初始化唯一 K3s Server，其他已审批 Linux 节点固定作为工作节点加入；运行环境只是在该集群上
划分命名空间和节点调度范围，这些底层实现不出现在最终用户界面。

基础服务只接受完整八项单实例分配；IF 历史库与 IF 时序库共享一个 TimescaleDB 工作负载，必须
位于同一节点。重复计划是幂等的，提升 generation 且保持原分配可用于重新部署 / 修复；当前拒绝
通过修复入口迁移有状态服务。SeaweedFS S3 和 EMQX MQTT 均默认拒绝匿名访问。

物理移除工作节点时清理已记录数据目录下的 K3s Agent 状态与本地 PVC，并清理 shim、挂载点、CNI 接口和规则；取消环境节点分配不会卸载 K3s。
默认保留 NodeAgent 身份、配置和日志以便重装；`--purge` 才额外清除这些内容。数据目录来自已校验
的本地集群状态，卸载器不会接受任意递归删除目标。

Linux 包同时携带 Ubuntu 24.04 的 Chrony `4.5-1ubuntu4.2` 及离线安装依赖。中心节点只发布自身
系统时间，并以 `-x` 禁止 Chrony 修改中心时钟；工作节点唯一时间源为中心节点。Agent 在心跳中
上报当前偏差和同步状态，时间同步故障会产生运维事件，但不会阻断 K3s、基础服务或工程命令。

## ReleaseStore 基础原语

正式 Release 命令已接入本流程：Agent 先以自身 token 从 Center 取得节点专属
`DeploymentBinding`，再从同源 Release 端点取得 `application/zstd` 流。Center 不会返回对象存储键、
外部下载 URL、路径、命令或 Secret；Agent 不跟随重定向，签名公钥只来自 `config.yaml` 的本地 trust
store。`internal/ops` 的 `ReleaseStore`/`ReleaseInstaller` 会：

1. 限额流式写入私有 `incoming/*.partial` 并核对 outer SHA-256；
2. 只安全解包根目录的标准普通文件，拒绝目录、路径逃逸、链接、设备、FIFO、重复项和解压炸弹；
3. 用 `signature.sig` 验证原始 `checksums.json`，并核对 Manifest 与每一个列出的文件；
4. 将成功内容写到 `releases/sha256-<digest>`，封存为不可写，并以原子 `current.json` 选择版本。

Release 根目录只允许 `release-manifest.json`、`client-assets.tar.zst`、`runtime-artifact.tar.zst`、
`health-contract.json`、`resource-recommendation.json`、`schema-plan.json`、`sbom.cdx.json`、可选
`collector-artifact.tar.zst`、`checksums.json` 和 `signature.sig`。文件格式为
`release-checksums.v1`：`files` 以路径严格升序列出 `{path,sha256,size}`，且不得包含
`checksums.json` 或 `signature.sig`；
`signature.sig` 是精确 checksums 原始字节的 64-byte Ed25519 签名；Manifest 必须声明
`supplyChain.signingKeyId`。详情见
[NodeAgent 与运行系统协议](../../docs/04-契约与规范/跨模块契约/node-agent-runtime-protocol.md)。

这不代表已实现完整工程运行：Secret 安装、内层工件物化、真实只读 mount、Runtime Foundation、组件
启动、升级或回滚尚未交付。当前正式 Release 通过全部校验后会因正式 launcher 尚未交付而记录失败，
绝不回退到旧静态 Supervisor。尤其 RuntimeEngine native v2 仍需要真实只读挂载，普通不可写目录不能替代。

## 平台范围

- Linux 包含 Gateway、Runtime API、RuntimeEngine 能力模板和可选 Collector，但默认不会启动。
- Windows 当前仅包含并声明 Collector；若收到 Gateway、Runtime API 或 RuntimeEngine 部署，必须
  fail-closed。Windows 正式安装/停止/重装验收仍待真实主机完成。
