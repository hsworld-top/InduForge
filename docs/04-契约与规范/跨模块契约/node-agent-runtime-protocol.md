# NodeAgent 与运行系统协议

## 1. 当前正式协议

本协议描述 Center 与外部物理节点 NodeAgent 的当前 HTTP 轮询协议，不描述 K3s、Pod、资源池或未来站点控制器。中心主机使用安装时自带的逻辑内置节点，不安装 NodeAgent。外部 Agent 只主动访问 Center，Center 不连接或嵌入 Agent 进程。Agent 已强制 `serverUrl` 使用 HTTPS；仅本机开发允许 `localhost`、`127.0.0.0/8` 或 `::1` 的 HTTP，并拒绝 userinfo、query、fragment 与非根路径。

Center 统一 JSON 包络响应上限为 1 MiB；超限、无效 JSON 或尾随第二个 JSON 值都会被 Agent 拒绝。

控制面用户模型包含物理节点、运行环境和工程部署。每个组织当前只有一套逻辑运行集群：中心安装流程登记一个不可在界面移除或解绑的中心内置节点，其他已接入 Linux 节点固定作为工作节点加入集群，再由运维人员按需关联运行环境。运行环境只负责命名空间、节点调度范围与基础服务状态聚合，不拥有独立 K3s，也不改变 Agent 的安全边界；Center 下发和 Agent 上报仍始终绑定明确的 `nodeId`。当前工程部署目标仍为一个 `nodeId`，服务类型固定为：

- `project_entry`：本机 Project Gateway + Runtime API；
- `data_runtime`：本机 RuntimeEngine；
- `collector`：可选 Industrial Collector。

静态 `client/` 文件只由 Gateway 托管，不是服务类型，也没有独立状态、PID 或操作端点。

## 2. 接入和身份

1. 管理员以 `POST /api/v1/ops/node-enrollments` 创建一次性接入任务，指定 `platform`、显示名、能力集合和有效期。能力只能是上述三项，至少一项；明文接入码只在创建响应中出现，中心保存哈希。
2. NodeAgent 用 `POST /api/v1/ops/agent/enrollments/claim` 提交接入码、主机名、平台、架构、机器指纹、Agent 版本和本机服务能力。成功响应返回 `node.id` 和一次性可见的 `agentToken`。
3. Agent 将身份和已处理 generation 分别原子写入私有 `dataDir` 的 `ops-agent-identity.json` 与 `ops-agent-applied-generations.json`（权限 0600），成功后清除配置中的接入码。
4. 接入码领取即完成授权，节点可获取自身命令。未使用接入码可通过 `POST /api/v1/ops/node-enrollments/{id}/revoke` 撤销；已经接入的节点通过节点移除流程管理。

中心安装始终自带 K3s Server。组织初始化事务会创建不含 `enrollment_id` 和 `agent_token_hash` 的 `built_in` 节点、逻辑运行集群和默认环境关联；该节点使用集群已有的 `induforge.io/center-node=true` 标签调度，不写入组织专属节点标签。之后凭接入码注册的 Linux 标准节点以 `agent` 来源和 `worker` 身份进入集群，但不会自动关联到某个运行环境。

## 3. 心跳、命令和调和

Agent 在默认 10 秒周期执行：

```text
POST /api/v1/ops/agent/nodes/{nodeId}/heartbeat
GET  /api/v1/ops/agent/nodes/{nodeId}/commands
→ 对每项仅调和本节点的 ServiceID + serviceType + desiredStatus + generation
```

心跳携带资源摘要、Agent 版本、服务观测、可选 `clusterState` 和 `foundationStates[]`；`project_entry` 可上报入口 URL，其余服务不得上报入口。Center 只接受令牌与 URL 安全边界，并由服务记录核对 serviceId 与节点归属。

`clusterState` 使用 `induforge.cluster-state.v1`，以 `clusterId + nodeId + generation` 标识工作节点状态。NodeAgent 只接受 `join-agent`；工作节点必须等待中心内置节点 Ready 后才能取得加入计划。中心 K3s 的初始化和观测由中心安装及控制面负责，不通过 NodeAgent 命令完成。集群令牌只出现在已认证的 Agent 响应中，不落数据库、不进入管理接口或事件。

基础服务计划由中心控制面直接调和，不再下发给 NodeAgent。每个计划仍以 `environmentId` 标识逻辑环境，固定渲染到独立命名空间，并通过节点选择器把工作负载分配到已关联的中心或工作节点。删除环境只异步删除该命名空间；不会卸载任一节点的 K3s。

命令包含 `nodeId`、`deploymentId`、`serviceId`、`serviceType`、`desiredStatus`、`operation`、`generation` 和版本。Agent 必须拒绝：nodeId 不匹配、未知服务组、未在本机配置的服务组、倒退 generation，以及任何未支持状态/操作。相同 generation 仅在本地进程确实满足期望时跳过，以便 Agent 重启后恢复已配置进程。

中心命令**不包含**可执行路径、命令行参数、环境变量、下载 URL、Secret 或任意 shell 指令。

## 4. 本地 ServiceConfig 与 release 选择

NodeAgent 的 `ops.services` 是管理员维护的声明式 allowlist。每个启用组件至少指定：服务组、组件名、release 根、`current`、`releaseDigest`、相对可执行路径、参数、环境、工作目录、回环 `healthUrl`、健康/排空超时。服务组可有多组件；`project_entry` 至少需要 Gateway 与 Runtime API。

Supervisor 解析 `releaseRoot/current`，确认它位于根内、是目录且包含 `release-manifest.json`；`releaseDigest` 必须等于该 manifest 的 SHA-256。可执行文件和工作目录必须是 release 内相对路径。健康 URL 只允许 `localhost` 或 loopback IP 的 HTTP URL，防止中心或配置把 Agent 变成外网探针。

这是 Supervisor 的本地 immutable release/current 选择和 manifest 摘要校验；它不是中心下发启动
参数的接口。运维人员仍须以受控方式提供本地服务配置。RuntimeEngine v2 额外要求实际只读的
release 挂载，普通可写目录会被拒绝。

### 4.1 部署级进程配置

节点内部的 `ActivateWorkload(serviceId, group, generation, services, restart)` 在静态安装能力
allowlist 内派生部署级配置；它不是 HTTP 接口，中心不能调用它传递任意可执行路径或参数。
配置按 serviceId 与进程状态一起写入权限 0600 的 `managed-processes.json`，不会随心跳返回。
Agent 重建 Supervisor 时检查本机已安装组件，再恢复各部署的配置和已存 PID 状态。

启停及候选版本切换串行执行，读取状态仍可并行。候选配置必须先通过 Release 和路径预检，
然后停止旧进程、启动候选、等待健康检查。同一 generation 已运行时，重复 restart 不会再次
启动进程。候选启动失败且旧部署先前正在运行时，恢复旧配置；恢复成功后仍上报旧 generation，
并记录本次错误，不能把恢复旧版本伪报为新版本发布成功。

目前这是节点内部原语；Collector 的中心配置/凭据交付、正式命令调用和 Linux/Windows
端到端部署仍需接入验证。持久化 PID 的恢复也不等同于已完成宿主重启或 PID 重用验收。

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

### 节点连接地址

中心部署配置 `NODE_CONNECTION_URL` 是安装人员确认的节点可达 HTTP(S) 入口，独立于 Studio 的浏览器地址；接入码创建响应提供 `data.nodeConnectionUrl`。未配置时返回空字符串，前端不得使用浏览器 origin 补齐。中心管理器通过 `IF_NODE_CONNECTION_URL` 或持久配置设置此值。Linux 交互安装向导先检查该地址的 `/health`，网络或证书检查失败时允许重新输入且不写入服务；接入码的有效性仍由实际注册接口验证。

节点远程接入必须使用 HTTPS（与 NodeAgent 校验一致）。私有中心随节点包提供 `center-ca.crt`，安装器为节点服务生成独立证书集合，不能跳过证书校验。检测到已有安装时，交互安装器提供重试连接选项而不覆盖配置；文件安装完成不等于接入成功，必须等到中心确认首次在线心跳。

安装器通过 `POST /api/v1/ops/agent/enrollments/validate` 只读验证接入码（code、platform、capabilities）。`data.valid=true` 表示当前有效；预检查不消费凭据、不创建节点，正式领取仍在事务内重新校验。能力匹配采用包含所需能力而非完全相等，允许标准 Linux 包额外带有采集能力。交互安装确认前验证系统、完整性清单、目录与空间，再执行接入预检查。

### 自动接入与安装确认

接入码即授权，不再提供批准/拒绝接口；未使用接入码可通过 `POST /api/v1/ops/node-enrollments/{id}/revoke` 撤销。领取在事务中登记节点及集群成员，初始离线，首次心跳后在线。安装器读取本次启动后的中心在线确认回执（节点 ID 须匹配已领取身份），只有成功心跳响应才能生成确认回执；领取身份或服务启动均不能代替安装成功。已安装节点支持原配置重试连接，不重复安装。

### 本机卸载确认

卸载器先停止节点协调并通过 Hostd 清理运行组件，再通过带节点凭据的心跳接口发送 `uninstalled: true`。中心保留最后资源信息，在 `resourceSummary.uninstalledAt` 记录中心接收时间，连接状态返回 `uninstalled`。卸载器必须收到该明确确认才显示同步成功；离线或不支持此协议时不阻止本机卸载，但提示中心状态未同步。普通离线不能推断为卸载，界面展示离线前的最后运行组件状态。重新安装后的有效普通心跳覆盖卸载标记并恢复在线。

中心节点管理的“移除登记”对在线、离线节点采用相同行为：检查运行环境和工程占用后，撤销节点凭据、移除集群登记并记录操作人。此入口不下发远端卸载，不删除本机程序或数据。本机备份与卸载统一通过 uninstall.sh 执行。

同一组织内节点名称忽略首尾空格和大小写后唯一；有效未领取的接入码预留名称，已撤销节点及失效接入码释放名称。生成接入码与领取节点在事务内串行检查，历史重复名称不会自动修改。

`DELETE /api/v1/ops/node-enrollments/{id}` 软删除接入码，记录 deleted_at/deleted_by；列表、计数、详情、领取、安装预检查及名称预留查询排除已删除记录。删除已使用接入码不影响对应节点及其名称占用。

## 中心镜像归档按需交付

节点命令响应可携带 `imagePlan: { artifacts: [...] }`。每项包含 `architecture`（`amd64` 或 `arm64`）、`sha256`（归档小写十六进制摘要）、`size`、`images: [{ reference, configDigest }]` 和 `downloadPath`。`configDigest` 为 `sha256:` 前缀的镜像配置摘要，不是镜像标签或远程仓库 manifest 摘要。下载地址严格固定为 `/api/v1/ops/agent/nodes/{nodeId}/images/{sha256}/download`，复用节点 Bearer 凭据；中心仅提供该节点所需的对象归档，支持 HTTP Range，节点拒绝重定向和任意地址。

节点在 K3s 底座完成后、基础服务和工程命令之前处理镜像计划。先检查本地 CRI 镜像配置摘要，缺失或不匹配才下载。文件按归档摘要缓存在 `/var/lib/induforge/image-cache/{sha256}.tar`，部分文件使用 `.part` 后缀供重试续传；完整大小与 SHA256 校验通过后原子发布。节点包只需携带 K3s 自启动底座，业务镜像来自中心对象库，禁止 registry pull，工作负载继续使用 `imagePullPolicy: Never`。

宿主接口 `POST /v1/images/check` 与 `POST /v1/images/import` 只接受 `{ sha256, size, images }`，响应包络 `data: { ready: boolean }`。导入接口不接受路径或 shell，仅读取固定缓存目录中的摘要文件，并复制到 root 私有临时目录重新校验后执行固定 K3s 导入命令。导入结束还需核对 CRI 中实际配置摘要；成功后删除归档与私有副本，复用 containerd 中的镜像内容以避免双份磁盘占用；缓存标记不能替代本地内容检查。中心内置节点使用同一受限接口，`INDUFORGE_HOSTD_IMAGES_ONLY=true` 时不注册集群、卸载、基础服务和时间同步接口，socket 访问组可由 `INDUFORGE_HOSTD_GROUP` 配置，默认 `induforge`。

心跳新增 `imageState: { status, artifacts, message }`；`status` 为 `downloading`、`importing`、`ready` 或 `failed`，`artifacts` 仅列出本轮已核验就绪的归档摘要。长传输期间继续发送心跳。中心必须以所需归档全部就绪为启动门槛；下载或校验失败不触发服务切换，允许下一轮继续准备。
