# NodeAgent 与运行系统协议

## 1. 当前正式协议

本协议描述 Center 与物理节点 NodeAgent 的当前 HTTP 轮询协议，不描述 K3s、Pod、资源池或未来站点控制器。Center 与 NodeAgent 即使同机也必须独立安装、独立账户、独立目录、独立端口和独立网络连接；Agent 只主动访问 Center，Center 不连接或嵌入 Agent 进程。Agent 已强制 `serverUrl` 使用 HTTPS；仅同机开发/安装允许 `localhost`、`127.0.0.0/8` 或 `::1` 的 HTTP，并拒绝 userinfo、query、fragment 与非根路径。

Center 统一 JSON 包络响应上限为 1 MiB；超限、无效 JSON 或尾随第二个 JSON 值都会被 Agent 拒绝。

用户模型只有物理节点和工程单节点部署。工程部署目标为一个 `nodeId`，服务类型固定为：

- `project_entry`：本机 Project Gateway + Runtime API；
- `data_runtime`：本机 RuntimeEngine；
- `collector`：可选 Industrial Collector。

静态 `client/` 文件只由 Gateway 托管，不是服务类型，也没有独立状态、PID 或操作端点。

## 2. 接入和身份

1. 管理员以 `POST /api/v1/ops/node-enrollments` 创建一次性接入任务，指定 `platform`、显示名、能力集合和有效期。能力只能是上述三项，至少一项；明文接入码只在创建响应中出现，中心保存哈希。
2. NodeAgent 用 `POST /api/v1/ops/agent/enrollments/claim` 提交接入码、主机名、平台、架构、机器指纹、Agent 版本和本机服务能力。成功响应返回 `node.id` 和一次性可见的 `agentToken`。
3. Agent 将身份和已处理 generation 分别原子写入私有 `dataDir` 的 `ops-agent-identity.json` 与 `ops-agent-applied-generations.json`（权限 0600），成功后清除配置中的接入码。
4. 管理员以 `POST /api/v1/ops/node-enrollments/{id}/approve` 审批后，节点才可获得命令；拒绝使用 `/reject`。拒绝不等于自动清理已落盘本地身份，需按安全事件处置。

## 3. 心跳、命令和调和

Agent 在默认 10 秒周期执行：

```text
POST /api/v1/ops/agent/nodes/{nodeId}/heartbeat
GET  /api/v1/ops/agent/nodes/{nodeId}/commands
→ 对每项仅调和本节点的 ServiceID + serviceType + desiredStatus + generation
```

心跳携带资源摘要、Agent 版本和服务观测；`project_entry` 可上报入口 URL，其余服务不得上报入口。Center 只接受令牌与 URL 安全边界，并由服务记录核对 serviceId 与节点归属。

命令包含 `nodeId`、`deploymentId`、`serviceId`、`serviceType`、`desiredStatus`、`operation`、`generation` 和版本。Agent 必须拒绝：nodeId 不匹配、未知服务组、未在本机配置的服务组、倒退 generation，以及任何未支持状态/操作。相同 generation 仅在本地进程确实满足期望时跳过，以便 Agent 重启后恢复已配置进程。

中心命令**不包含**可执行路径、命令行参数、环境变量、下载 URL、Secret 或任意 shell 指令。

## 4. 本地 ServiceConfig 与 release 选择

NodeAgent 的 `ops.services` 是管理员维护的声明式 allowlist。每个启用组件至少指定：服务组、组件名、release 根、`current`、`releaseDigest`、相对可执行路径、参数、环境、工作目录、回环 `healthUrl`、健康/排空超时。服务组可有多组件；`project_entry` 至少需要 Gateway 与 Runtime API。

Supervisor 解析 `releaseRoot/current`，确认它位于根内、是目录且包含 `release-manifest.json`；`releaseDigest` 必须等于该 manifest 的 SHA-256。可执行文件和工作目录必须是 release 内相对路径。健康 URL 只允许 `localhost` 或 loopback IP 的 HTTP URL，防止中心或配置把 Agent 变成外网探针。

这是**本地 immutable release/current 选择和 manifest 摘要校验**，不是完整 Release 下载/验签/解包协议。完整交付链尚未实现；运维人员必须先以受控方式提供 release、配置和 Secret。RuntimeEngine v2 额外要求实际只读的 release 挂载，普通可写目录会被拒绝。

## 5. 进程、健康与停止

Supervisor 在私有状态目录持久化服务组 PID、组件 PID、generation、时间和日志路径；启动时只恢复仍属于当前显式配置的进程。每个组件先启动，再对本机 `/health` 探测；任何组件失败会停止已启动的同组组件并使整个服务组失败。

停止时先发送正常终止信号，使 RuntimeEngine 等组件进入自己的 drain 过程；到 `drainTimeout` 后才强制终止。Agent 的 `replicasObserved` 在当前单节点实现只会是 0 或 1，不是副本调度或 HA 证据。

## 6. 工程单节点部署 API

管理接口位于 `/api/v1/ops`：

| 操作              | 路径                                                                       |
| ----------------- | -------------------------------------------------------------------------- |
| 节点与接入任务    | `/node-enrollments`、`/nodes`                                              |
| 查看可下载安装包  | `/node-packages`、`/node-packages/{id}/download`                           |
| 创建/查看工程部署 | `POST/GET /project-deployments`、`GET /project-deployments/{id}`           |
| 操作一项服务组    | `POST /project-deployments/{id}/services/{service}/{start\|stop\|restart}` |
| 读取运行记录      | `/deployment-runs/{id}`、`/deployment-runs/{id}/events`                    |

创建部署请求为 `projectId`、`nodeId`、`applicationVersionId` 及 `enableCollector`。创建前中心会
校验版本已 `ready`、对象键与归档 SHA-256 有效，且 Manifest 符合 `2.0` 正式 Release 结构；
只有源码快照的旧 code-first IFP 会在创建部署前 fail-closed。创建后中心生成固定服务记录，
运行记录是否完成由全部服务记录的 desired/observed generation 与状态决定。当前检查仍不代表
对象存储中的归档已被 NodeAgent 下载、验签、解包或实例化；完整物料竖链仍是未交付项。

## 7. 未交付：多节点分布式与高可用

当前一个工程部署只绑定一台物理节点；多节点分片、角色迁移、epoch/fencing、检查点、滚动升级、自动故障转移和高可用均未交付且不得通过 API/UI 启用。多台已接入节点、多个 PID、`replicasObserved=1` 或多个单节点部署都不等于 HA。

未来分布式协议须另行冻结 Release/Secret 一致性、业务唯一 owner、跨节点故障检测、手动/自动切换和故障注入验收后，才能取代本协议；在此之前不得将未来对象写入当前 Center→Agent 命令。

## 8. 关联文档

- [工程运行系统架构](../../02-系统设计/工程运行系统架构.md)
- [工程运行微服务设计](../../02-系统设计/工程运行微服务设计.md)
- [Runtime 健康检查与状态协议](./runtime-health-status-contract.md)
- [运维体系实现与验收](../../06-运维与安全/运维体系实现与验收.md)
