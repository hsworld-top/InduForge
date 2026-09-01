# NodeAgent 与运行系统协议

## 1. 当前正式协议

本协议描述 Center 与物理节点 NodeAgent 的当前 HTTP 轮询协议，不描述 K3s、Pod、资源池或未来站点控制器。Center 与 NodeAgent 即使同机也必须独立安装、独立账户、独立目录、独立端口和独立网络连接；Agent 只主动访问 Center，Center 不连接或嵌入 Agent 进程。Agent 已强制 `serverUrl` 使用 HTTPS；仅同机开发/安装允许 `localhost`、`127.0.0.0/8` 或 `::1` 的 HTTP，并拒绝 userinfo、query、fragment 与非根路径。

Center 统一 JSON 包络响应上限为 1 MiB；超限、无效 JSON 或尾随第二个 JSON 值都会被 Agent 拒绝。

控制面用户模型包含物理节点、运行环境和工程部署。每个租户当前只有一套中心运行集群：中心安装流程绑定一个不可在界面移除的中心内置节点，其他已审批 Linux 节点固定作为工作节点加入。运行环境只负责命名空间、节点调度范围与基础服务状态聚合，不拥有独立 K3s，也不改变 Agent 的安全边界；Center 下发和 Agent 上报仍始终绑定明确的 `nodeId`。当前工程部署目标仍为一个 `nodeId`，服务类型固定为：

- `project_entry`：本机 Project Gateway + Runtime API；
- `data_runtime`：本机 RuntimeEngine；
- `collector`：可选 Industrial Collector。

静态 `client/` 文件只由 Gateway 托管，不是服务类型，也没有独立状态、PID 或操作端点。

## 2. 接入和身份

1. 管理员以 `POST /api/v1/ops/node-enrollments` 创建一次性接入任务，指定 `platform`、显示名、能力集合和有效期。能力只能是上述三项，至少一项；明文接入码只在创建响应中出现，中心保存哈希。
2. NodeAgent 用 `POST /api/v1/ops/agent/enrollments/claim` 提交接入码、主机名、平台、架构、机器指纹、Agent 版本和本机服务能力。成功响应返回 `node.id` 和一次性可见的 `agentToken`。
3. Agent 将身份和已处理 generation 分别原子写入私有 `dataDir` 的 `ops-agent-identity.json` 与 `ops-agent-applied-generations.json`（权限 0600），成功后清除配置中的接入码。
4. 管理员以 `POST /api/v1/ops/node-enrollments/{id}/approve` 审批后，节点才可获得命令；拒绝使用 `/reject`。拒绝不等于自动清理已落盘本地身份，需按安全事件处置。

中心安装流程通过服务端配置 `IF_OPS_CENTER_NODE_ID` 幂等创建全局集群和中心节点成员。配置的节点必须是已审批 Linux 节点；若已经存在不同中心节点，服务端拒绝启动而不会自动替换。之后审批通过的 Linux 标准节点自动以 `worker` 身份进入该集群。

## 3. 心跳、命令和调和

Agent 在默认 10 秒周期执行：

```text
POST /api/v1/ops/agent/nodes/{nodeId}/heartbeat
GET  /api/v1/ops/agent/nodes/{nodeId}/commands
→ 对每项仅调和本节点的 ServiceID + serviceType + desiredStatus + generation
```

心跳携带资源摘要、Agent 版本、服务观测、可选 `clusterState` 和 `foundationStates[]`；`project_entry` 可上报入口 URL，其余服务不得上报入口。Center 只接受令牌与 URL 安全边界，并由服务记录核对 serviceId 与节点归属。

`clusterState` 使用 `induforge.cluster-state.v1`，以 `clusterId + nodeId + generation` 标识全局集群状态。中心节点只接受 `init-server`，工作节点只接受 `join-agent`；工作节点必须等待中心节点 Ready 后才能取得加入计划。集群令牌由 Center 使用本地根密钥和 `clusterId` 临时派生，只出现在已认证的 Agent 响应中，不落数据库、不进入管理接口或事件。

只有中心节点会收到 `induforge.foundation-plan.v1` 和 `foundationDelete`。每个基础服务计划仍以 `environmentId` 标识逻辑环境，固定渲染到独立命名空间，并通过节点选择器把工作负载分配到已关联的中心或工作节点。`foundationStates[]` 允许中心节点在同一次心跳中报告多个环境。删除环境只异步删除该命名空间并回报 `not-installed`；不会卸载任一节点的 K3s。

命令包含 `nodeId`、`deploymentId`、`serviceId`、`serviceType`、`desiredStatus`、`operation`、`generation` 和版本。Agent 必须拒绝：nodeId 不匹配、未知服务组、未在本机配置的服务组、倒退 generation，以及任何未支持状态/操作。相同 generation 仅在本地进程确实满足期望时跳过，以便 Agent 重启后恢复已配置进程。

中心命令**不包含**可执行路径、命令行参数、环境变量、下载 URL、Secret 或任意 shell 指令。

## 4. 本地 ServiceConfig 与 release 选择

NodeAgent 的 `ops.services` 是管理员维护的声明式 allowlist。每个启用组件至少指定：服务组、组件名、release 根、`current`、`releaseDigest`、相对可执行路径、参数、环境、工作目录、回环 `healthUrl`、健康/排空超时。服务组可有多组件；`project_entry` 至少需要 Gateway 与 Runtime API。

Supervisor 解析 `releaseRoot/current`，确认它位于根内、是目录且包含 `release-manifest.json`；`releaseDigest` 必须等于该 manifest 的 SHA-256。可执行文件和工作目录必须是 release 内相对路径。健康 URL 只允许 `localhost` 或 loopback IP 的 HTTP URL，防止中心或配置把 Agent 变成外网探针。

这是 Supervisor 的本地 immutable release/current 选择和 manifest 摘要校验；它不是中心下发启动
参数的接口。运维人员仍须以受控方式提供本地服务配置。RuntimeEngine v2 额外要求实际只读的
release 挂载，普通可写目录会被拒绝。

## 5. ReleaseStore 安装原语（已交付，尚未接入调和）

`runtime/node_agent/internal/ops` 已提供 `ReleaseStore`/`ReleaseInstaller`。调用者必须为每个
deployment 提供独立本地根，并传入 tar.zst 流、预期 outer `sha256:<64 lowercase hex>`、预期
Manifest/Checksums 摘要、预期 projectId/releaseId，以及本地 trust store 已选择的 Ed25519 公钥及
key ID。该原语：

1. 流式限额写入私有 `incoming/*.partial`，同步文件并核对 outer SHA-256；
2. 只解包固定清单中的根目录普通文件，拒绝目录、绝对路径、`..`、反斜杠、嵌套路径、
   符号/硬链接、设备、FIFO、重复项、过多文件和超限解压；
3. 验证 `signature.sig` 对**原始** `checksums.json` 字节的 Ed25519 签名，验证所有文件摘要和
   大小，并要求 Manifest 的 `supplyChain.signingKeyId` 等于输入 key ID；
4. 以 outer 摘要内容寻址保存为 `releases/sha256-<digest>`，封存为不可写，并通过同卷临时文件、
   同步、rename 原子写入 `current.json`。

Release tar.zst 根必须直接包含标准 Release 文件：

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

`checksums.json` 的唯一格式为：

```json
{
  "schemaVersion": "release-checksums.v1",
  "files": [{ "path": "release-manifest.json", "sha256": "sha256:<64 lowercase hex>", "size": 123 }]
}
```

`files` 必须按 `path` 严格升序，覆盖上列全部标准 Release 文件（可选 Collector 仅在 Manifest
声明时出现），且不得包含 `checksums.json` 或 `signature.sig`。`signature.sig` 必须是精确
checksums 原始字节的 64 字节 Ed25519 原始签名。`current.json` 为：

```json
{
  "schemaVersion": "release-pointer.v1",
  "releaseDigest": "sha256:<64 lowercase hex>",
  "releaseDir": "releases/sha256-<digest>",
  "activatedAt": "2026-08-31T10:00:00Z"
}
```

任一步失败均不切换 current。该原语**尚未**接入 Agent 网络下载、trust 配置、
DeploymentBinding/Secret、真实只读 mount、Supervisor 启动或服务回滚；不得把它描述为 E2E
Release 交付。

## 6. 进程、健康与停止

Supervisor 在私有状态目录持久化服务组 PID、组件 PID、generation、时间和日志路径；启动时只恢复仍属于当前显式配置的进程。每个组件先启动，再对本机 `/health` 探测；任何组件失败会停止已启动的同组组件并使整个服务组失败。

停止时先发送正常终止信号，使 RuntimeEngine 等组件进入自己的 drain 过程；到 `drainTimeout` 后才强制终止。Agent 的 `replicasObserved` 在当前单节点实现只会是 0 或 1，不是副本调度或 HA 证据。

## 7. 工程单节点部署 API

管理接口位于 `/api/v1/ops`：

| 操作              | 路径                                                                       |
| ----------------- | -------------------------------------------------------------------------- |
| 节点与接入任务    | `/node-enrollments`、`/nodes`                                              |
| 查看可下载安装包  | `/node-packages`、`/node-packages/{id}/download`                           |
| 创建/查看工程部署 | `POST/GET /project-deployments`、`GET /project-deployments/{id}`           |
| 操作一项服务组    | `POST /project-deployments/{id}/services/{service}/{start\|stop\|restart}` |
| 读取运行记录      | `/deployment-runs/{id}`、`/deployment-runs/{id}/events`                    |

创建部署请求为 `projectId`、`nodeId`、`applicationVersionId`、客户选择的 `accessPort` 及
`enableCollector`。创建前中心会
校验版本已 `ready`、对象键与归档 SHA-256 有效，且 Manifest 符合 `2.0` 正式 Release 结构；
只有源码快照的旧 code-first IFP 会在创建部署前 fail-closed。中心还会检查同一节点已有部署的端口
区间，节点侧在启动前复核操作系统真实占用。创建后中心生成固定服务记录，
运行记录是否完成由全部服务记录的 desired/observed generation 与状态决定。当前检查与
`ReleaseStore` 都不代表对象存储归档已经由 Agent 下载，也不代表 trust、Binding、Secret、真实
只读挂载和组件启动/回滚已实例化；正式物料竖链仍未交付。

## 8. 未交付：多节点分布式与高可用

当前一个工程部署只绑定一台物理节点；多节点分片、角色迁移、epoch/fencing、检查点、滚动升级、自动故障转移和高可用均未交付且不得通过 API/UI 启用。多台已接入节点、多个 PID、`replicasObserved=1` 或多个单节点部署都不等于 HA。

未来分布式协议须另行冻结 Release/Secret 一致性、业务唯一 owner、跨节点故障检测、手动/自动切换和故障注入验收后，才能取代本协议；在此之前不得将未来对象写入当前 Center→Agent 命令。

## 9. 关联文档

- [工程运行系统架构](../../02-系统设计/工程运行系统架构.md)
- [工程运行微服务设计](../../02-系统设计/工程运行微服务设计.md)
- [Runtime 健康检查与状态协议](./runtime-health-status-contract.md)
- [运维体系实现与验收](../../06-运维与安全/运维体系实现与验收.md)
