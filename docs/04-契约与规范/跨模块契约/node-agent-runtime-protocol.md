# NodeAgent 与运行系统协议

## 1. 文档定位

本文档定义中心控制面、每台宿主机 NodeAgent、K3s 内 site-controller、工程运行栈和原生采集
进程之间的启动、发布、回滚与状态协议。

## 2. 正式运行基线

- Linux 服务站点以一套 K3s 承载多个工程。
- 每个 `ProjectDeployment` 对应一个 K3s Namespace 和一个 NATS Account；同一工程的多阶段实例
  使用不同 `deploymentId`，允许在同一 Site 并存。
- Windows/Linux 工业采集均使用原生进程，不进入 K3s。
- 每台受管宿主机运行一个 K3s 外部 NodeAgent；每个 K3s 站点运行一个逻辑 site-controller。
- 中心不保存站点 Kubeconfig；NodeAgent 与 site-controller 分别主动建立出站 mTLS 通道。
- Docker Compose 不作为正式生产运行基线；如保留，仅用于本地开发或最小验证。

## 3. 核心对象

所有 Desired State 使用统一 envelope；Observed State 回传相同 `objectId`，并用
`observedRevision` 表明已处理的 `desiredRevision`：

```json
{
  "objectType": "<type>",
  "objectId": "<stable-id>",
  "schemaVersion": "1.0",
  "desiredRevision": 12,
  "generatedAt": "2026-08-30T10:00:00Z",
  "spec": {}
}
```

`desiredRevision` 是协调顺序，领域内的 Binding revision、Release schemaVersion 和 Agent
版本各自保留，不能互相代替。短期下载 URL 不进入持久 Desired State；执行器需要下载时，凭
`artifactRef` 和节点身份换取可续签的短期 URL。

### 3.1 SiteDesiredState

```json
{
  "objectType": "Site",
  "objectId": "site-a",
  "schemaVersion": "1.0",
  "desiredRevision": 12,
  "generatedAt": "2026-08-30T10:00:00Z",
  "spec": {
    "desiredTopologyProfile": "site-ha",
    "k3sVersion": "<managed-version>",
    "enabledCapabilities": ["collector-ha", "runtime-sharding"]
  }
}
```

目标不等于事实。infrastructure-controller leader 另写 `SiteInfrastructureObservedState`：

```json
{
  "objectType": "SiteInfrastructure",
  "objectId": "site-a",
  "observedRevision": 12,
  "observedAt": "2026-08-30T10:02:00Z",
  "status": {
    "observedTopologyProfile": "site-standard",
    "availableCapabilities": ["runtime-sharding"],
    "unmetConditions": ["K3S_SERVER_QUORUM_NOT_READY", "JETSTREAM_R3_NOT_READY"]
  }
}
```

工程发布门禁只读取 Observed。Profile/公共能力变化必须通过独立
`SiteInfrastructurePlan/SiteInfrastructureRun` 执行，不能由普通 Site 字段修改或工程发布暗中触发。

### 3.2 ProjectDeploymentDesiredState

```json
{
  "objectType": "ProjectDeployment",
  "objectId": "deployment-project-a-prod",
  "schemaVersion": "1.0",
  "desiredRevision": 31,
  "generatedAt": "2026-08-30T10:00:00Z",
  "spec": {
    "target": {
      "releaseRef": "release://project-a/2026.07.13-001",
      "bindingId": "binding-project-a-prod",
      "bindingRevision": 7
    },
    "rolloutPolicy": {
      "stateless": "rolling",
      "stateful": "authority-cutover",
      "collector": "authority-cutover"
    }
  }
}
```

`ProjectDeployment` 注册对象在创建时固定 `deploymentId`、`siteId`、`projectId` 和
`deploymentStage`；这些字段不可修改，晋级、迁移或复制部署创建新的 Deployment。
`(siteId, projectId, deploymentStage)` 不作为唯一键，以支持同阶段分区/并行实例。长期 Desired
只表达目标 generation，不保存短生命周期 `deploymentRunId`，也不允许调用方指定 Namespace；
Namespace 由 infrastructure-controller 根据 `deploymentId` 分配。

### 3.3 DeploymentBinding

```json
{
  "bindingId": "binding-project-a-prod",
  "schemaVersion": "1.0",
  "bindingRevision": 7,
  "deploymentId": "deployment-project-a-prod",
  "continuityPolicy": "production-standard",
  "requiredSiteCapabilities": ["collector-ha"],
  "workloadProfile": "standard",
  "dataDurabilityClass": "standard",
  "domains": ["project-a.example.local"],
  "secretRefs": ["secret://site-a/deployment-project-a-prod/runtime"],
  "resourcePolicy": "production-standard",
  "connectionBindings": {
    "line1-plc": "site-resource://collector-01/connection-7"
  }
}
```

Release 保持环境无关；真实连接、Secret、域名、资源、副本和保留策略只进入
DeploymentBinding。Binding 只引用 `deploymentId`，不重复保存 Site、Project 或 Stage，也不包含
Release ID；它可跨 Release 复用并独立版本化。
ProjectDeploymentDesiredState/DeploymentRun 同时引用 Release 与 Binding revision；同一 Release
在 development、staging 和 production 间晋级时不重新构建。

### 3.4 NativeCollectorAssignmentDesiredState

```json
{
  "objectType": "NativeCollectorAssignment",
  "objectId": "assignment-line1-primary",
  "schemaVersion": "1.0",
  "desiredRevision": 9,
  "generatedAt": "2026-08-30T10:00:00Z",
  "spec": {
    "nodeId": "collector-win-01",
    "deploymentId": "deployment-project-a-prod",
    "projectId": "project-a",
    "releaseId": "2026.07.13-001",
    "artifactRef": "release://project-a/2026.07.13-001/collector",
    "bindingId": "binding-project-a-prod",
    "bindingRevision": 7,
    "collectorAssignmentId": "assignment-line1-primary",
    "assignmentRevision": 4,
    "ownerRole": "ACTIVE",
    "epoch": 102,
    "collectorVersion": "1.0.0",
    "platform": "windows-x64",
    "logicalConnections": [
      {
        "logicalId": "line1-plc",
        "resourceRef": "site-resource://collector-01/connection-7",
        "secretRef": "secret://site-a/deployment-project-a-prod/line1-plc"
      }
    ]
  }
}
```

节点准备完成只表示进程和连接配置可用，不等于获得业务所有权。长期 Assignment Desired 不保存
Run ID；NodeAgent 必须按 Assignment revision 与 epoch 执行 active/standby 切换，并通过当前
DeploymentRun 的独立 `ParticipantCommand/ParticipantStatus` 关联准备、切换和补偿结果。无硬
fencing 的工业协议在主备歧义期间默认禁止设备写入。

### 3.5 SiteBootstrapPlan

首次建站由中心签发限时 `SiteBootstrapPlan`，指定 `bootstrapEpoch`、唯一 bootstrap owner、owner
Lease、Site Profile、K3s 版本、Server/Agent 节点、固定注册地址和控制器基线。owner 本地生成
静态 K3s server token；新增 Server 用绑定
`siteId + bootstrapEpoch + nodeId + expiresAt`、目标节点公钥加密的一次性 capsule 安全取得该
token，一次性的是传递授权而不是 server token。Agent 节点使用每节点限时 K3s bootstrap token，
成功加入后必须执行 `k3s token delete`。中心只中转密文，不保存可复用 server token。

Plan/Run 使用统一 envelope、`commandId + desiredRevision + bootstrapEpoch` 幂等并持久化阶段
checkpoint。owner 失联时必须显式 revoke、提升 epoch，并在电源、网络或人工现场完成旧 owner
fencing 后才能 takeover；不能确认旧 owner 已停止时，不得创建另一套首集群。server token 按
轮换版本备份并与对应 etcd 快照关联。

### 3.6 DeploymentPlan、Run 与 Snapshot

中心负责生成并审批不可变 `DeploymentPlan/DeploymentRunRequest`，冻结
`deploymentId + desiredRevision + releaseRef + bindingId/bindingRevision + rolloutPolicy`。站点
site-controller leader 是 `DeploymentRun.status` 的唯一推进者，中心只镜像状态；
infrastructure-controller、K3s 工作负载和 Collector NodeAgent 只写各自的 `ParticipantStatus`。

发布切换不是跨 K3s、数据库和原生进程的分布式原子事务，而是站点内可恢复 Saga：

```text
RunRequest → Prepare → 持久 CommitDecision → 幂等 Commit → Observe → Snapshot
                              └─ 失败 → Compensate / ManualIntervention
```

controller 之间通过 CRD/站点控制库协调；NodeAgent 通过主动出站 mTLS 连接
`site-control-gateway`。命令、回执、`ParticipantStatus` 和唯一 `CommitDecision` 必须先写入
`SiteControlStore` 再投递，不能依赖中心在线，也不能使用工程 NATS Account 作为唯一协调通道。
参与者至少包含 site-controller、infrastructure-controller 与全部相关 Collector NodeAgent。

每次成功运行生成不可变 `DeploymentSnapshot`：

```json
{
  "snapshotId": "snapshot-project-a-prod-20260830-01",
  "deploymentId": "deployment-project-a-prod",
  "releaseRef": "release://project-a/2026.07.13-001",
  "binding": { "bindingId": "binding-project-a-prod", "bindingRevision": 7 },
  "resolvedSecretVersions": ["secret://site-a/deployment-project-a-prod/runtime@12"],
  "siteResourceRevision": 19,
  "collectorAssignmentRevisions": {
    "assignment-line1-primary": 4
  },
  "compatibility": {
    "schema": "backward-compatible",
    "checkpoint": "v3"
  },
  "committedAt": "2026-08-30T10:10:00Z"
}
```

回滚是以目标 Snapshot 创建新的 RunRequest/DeploymentRun；不得只写 `rollbackReleaseId`，也不得
修改既有 Snapshot。

## 4. NodeAgent 与 site-controller 职责

### 4.1 NodeAgent

- 与中心建立设备身份和状态通道。
- 上报本机身份、CPU、内存、磁盘、网络、时钟和关键服务。
- 安装、启停、升级和修复本机 K3s Server/Agent。
- 托管本机 Windows/Linux 原生 Collector 子进程。
- 管理本机 Release 缓存、日志、WAL、密钥与磁盘水位。
- 在 K3s 不可用时继续提供本机诊断和恢复通道。

### 4.2 site-controller

- 与中心协调站点和工程期望状态。
- 将工程期望状态转换为已分配 Namespace 内的 Kubernetes 资源。
- 应用 ServiceAccount、Secret、ConfigMap、PVC、Job、Deployment 和 Service；不创建 Namespace、
  集群级 Ingress/域名、NATS Account 或公共数据空间。
- 维护 Desired/Observed/LastSuccessfulSnapshot 与 DeploymentRun。
- 聚合工程和 K3s 工作负载状态。
- 使用 Kubernetes Lease 选举活动协调者。

### 4.3 infrastructure-controller

- 创建/回收工程 Namespace，协调 Ingress/域名、NATS Account、工程数据空间、遥测 Gateway、
  备份控制和站点公共数据底座。
- 使用独立 ServiceAccount、CRD 和审计域，不与工程控制器共用无限制高权限身份。
- 首次由签名 `SiteBootstrapPlan` 引导安装，后续按独立维护计划升级。

NodeAgent、site-controller 与 infrastructure-controller 都不解释页面 Schema、数据点、报警规则、
计算脚本和工业地址。

声明状态持久化在 Kubernetes API/CRD，Release/配置使用站点持久存储；运行 epoch、检查点、
CommitDecision 和幂等账本使用支持原子比较更新的运行控制库。Pod 本地目录只作可丢缓存。
状态采用单域单写：NodeAgent 写 `HostNodeObservedState`；infrastructure-controller leader 写
`SiteInfrastructureObservedState`；site-controller leader 写 ProjectDeployment/DeploymentRun
Observed；中心或 site-observer 只读汇总。两个 controller 分别使用独立 Lease、workload identity
和 leader-only 写会话，不能共享一套可同时写入的证书。

Namespace 只能由 infrastructure-controller 创建并标记 `deploymentId/owner`；site-controller
只管理已经分配的 Namespace。Kubernetes RBAC 无法按名称前缀约束 Namespace `create`，因此还须
用准入策略验证资源归属，不能以文字前缀代替权限边界。

## 5. K3s 工程发布流程

1. 中心冻结并审批 `DeploymentPlan/RunRequest`；site-controller leader 按 runRequestId 创建或恢复
   `DeploymentRun`，重复请求返回既有 Run。
2. 校验 Deployment 注册身份、Release、Binding revision、Observed Site capabilities、协议版本和
   目标 DeploymentSnapshot 兼容性。
3. infrastructure-controller 准备或校验 deployment 的 Namespace、Ingress/域名、NATS Account、
   数据空间和 Secret 版本，并写自己的 `ParticipantStatus=PREPARED`。
4. site-controller 在已分配 Namespace 内创建 Release Loader Job；Loader 下载、校验并展开资源到
   工程 PVC。
5. Loader 成功后应用新版本运行工作负载，并等待 Deployment、Service 和无副作用业务探针通过。
6. 全部相关 Collector NodeAgent 以 standby 准备新 Assignment，并写
   `ParticipantStatus=PREPARED`。
7. 所有参与者 Prepare 成功后，site-controller 在 SiteControlStore 持久化唯一 CommitDecision，
   再并行投递幂等 Commit：排空旧 owner、保存 checkpoint、递增 epoch 并切换唯一业务所有权。
8. 所有参与者 Commit 完成后进入观察窗口；成功则生成 DeploymentSnapshot 并更新
   `lastSuccessfulSnapshotId`。
9. Prepare 失败保留旧版本；Commit 后失败按持久决议执行 Compensate，无法自动恢复则进入
   `MANUAL_INTERVENTION_REQUIRED`。中心断线不改变已持久化决议。

无状态服务可以滚动更新；writer、alarm、compute 和定时任务必须先预热、排空、持久化
checkpoint，再通过 Lease + epoch 切换唯一业务所有权。Pod Ready 不代表有状态切换完成。

## 6. 原生采集发布流程

1. NodeAgent 接收 `NativeCollectorAssignmentDesiredState`。
2. 校验 deployment、Release、Binding、Assignment revision、ownerRole、epoch、目标平台和驱动能力。
3. 必要时安装或升级 Collector 能力包。
4. 以 `artifactRef` 换取短期 URL，下载逻辑采集模型，并用 Binding 实例化现场资源和 Secret；
   校验签名和 SHA-256。
5. 生成 `native-processes.json`。
6. 预创建工程日志、WAL、密钥和版本目录。
7. 以 standby 模式启动新版本 Collector 并执行本地健康检查，回传 `PREPARED`。
8. 仅在收到 SiteControlStore 中已持久 CommitDecision 对应的 ParticipantCommand 且获得新 epoch
   后激活；重复命令按 runId/phase 幂等，失败时执行 Compensate 恢复旧进程或保持设备写入暂停。
9. 回传进程状态、Release、Binding/Assignment revision、epoch 和驱动能力。

## 7. native-processes.json

```json
{
  "nodeId": "collector-win-01",
  "processes": [
    {
      "id": "collector-deployment-project-a-prod",
      "deploymentId": "deployment-project-a-prod",
      "projectId": "project-a",
      "executable": "industrial-collector",
      "arguments": ["--mode=runtime", "--deployment=deployment-project-a-prod", "--release=2026.07.13-001"],
      "workingDirectory": "<project-release-dir>",
      "restartPolicy": "always",
      "healthCheck": {
        "type": "http",
        "address": "http://127.0.0.1:19101/health"
      }
    }
  ]
}
```

OPC DA Worker 作为工程级可选进程加入同一计划。

## 8. 运行目录

### 8.1 Linux NodeAgent

```text
/opt/induforge/node-agent/
/var/lib/induforge/node-agent/
/var/log/induforge/node-agent/
```

### 8.2 原生采集节点

```text
deployments/<deploymentId>/
├─ releases/<releaseId>/collector/
├─ current-state.json
├─ logs/
├─ wal/
└─ secrets/
```

## 9. NATS 和凭据

- 站点共享一套 NATS JetStream。
- 每个 ProjectDeployment 使用独立 Account，避免同一工程的预发/生产消息互通。
- Collector 只获取目标 deployment 最小权限凭据。
- NodeAgent 可以部署凭据文件，但不使用该凭据处理工程业务消息。

## 10. 健康状态

下例是中心/site-observer 基于多个领域 ObservedState 生成的只读视图，不是任何 controller 可整体
覆写的权威对象：

```json
{
  "siteId": "site-a",
  "observedRevision": 12,
  "observedAt": "2026-08-30T10:00:00Z",
  "healthState": "HEALTHY",
  "k3s": { "healthState": "HEALTHY" },
  "jetstream": { "healthState": "HEALTHY" },
  "deployments": [
    {
      "deploymentId": "deployment-project-a-prod",
      "projectId": "project-a",
      "desiredReleaseId": "2026.07.13-001",
      "observedReleaseId": "2026.07.13-001",
      "lastSuccessfulSnapshotId": "snapshot-project-a-prod-20260830-01",
      "healthState": "HEALTHY"
    }
  ],
  "nativeProcesses": [
    {
      "id": "collector-deployment-project-a-prod",
      "lifecycleState": "RUNNING",
      "healthState": "HEALTHY"
    }
  ]
}
```

字段来源固定为：NodeAgent → HostNodeObservedState，infrastructure-controller leader →
SiteInfrastructureObservedState，site-controller leader → ProjectDeployment/DeploymentRunObservedState，
Collector NodeAgent → ParticipantStatus/本机进程状态。聚合器不得用聚合结果反写任何来源对象。

详细字段遵循 [Runtime 健康检查与状态协议](./runtime-health-status-contract.md)。
CPU、内存、磁盘等 NodeResourceSnapshot 和运维事件遵循
[平台运维体系设计](../../06-运维与安全/平台运维体系设计.md)。

## 11. 回滚

- 回滚目标必须是指定的不可变 DeploymentSnapshot，而不是单独的 Release ID。
- 中心为该 Snapshot 创建新的 DeploymentPlan/RunRequest，site-controller 按正常 Saga 推进；K3s、
  infrastructure-controller 和原生 Collector 必须恢复同一组 Release、Binding/Secret、SiteResource、
  Assignment 与 Schema/checkpoint 兼容边界。
- 不允许在当前 Release 目录中原地修改配置，也不允许只回滚 Pod 而保留新连接/新 Assignment。
- 回滚完成后必须重新执行健康门禁和观察窗口，并生成新的审计结果。

## 12. 安全约束

- 中心不保存站点 K3s 凭据；site-controller 与 infrastructure-controller 使用职责隔离、范围受限
  的集群内 ServiceAccount。
- NodeAgent 拆分普通协调进程和最小特权 helper；helper 只接受签名白名单命令、校验参数和目标
  路径并记录本地提权审计。
- Desired State 只保存 `artifactRef`；Release 下载时换取可续签的短期授权。
- 工程 Secret 只进入目标 Namespace。
- 原生采集密钥只写入节点安全目录。
- 本地诊断端口默认绑定 `127.0.0.1`。

## 13. 关联文档

- [节点管理与交付架构](../../02-系统设计/节点管理与交付架构.md)
- [运维与节点运行态总体架构](../../02-系统设计/运维与节点运行态总体架构.md)
- [平台运维体系设计](../../06-运维与安全/平台运维体系设计.md)
- [工程运行系统架构](../../02-系统设计/工程运行系统架构.md)
- [工业采集架构](../../02-系统设计/工业采集架构.md)
