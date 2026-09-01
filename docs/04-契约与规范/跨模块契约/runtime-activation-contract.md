# Runtime Activation 契约

## 1. 定位与边界

`runtime-activation.v1` 是 NodeAgent 在正式工程部署时**本地派生并冻结**的内部机器契约。它把已
验证的 Release、部署 Binding 与本地运行基础设施分配和 Secret 元数据合并成一次不可变的激活输入。
Center 当前只向节点提供 DeploymentBinding 和 Release；它不下发 Activation，也不接收 Foundation
分配或 Secret 的两阶段回报，因此本契约不会凭空新增 Center API 或外部输入。

这不是用户可见的资源模型：产品界面仍然只展示**物理节点**、工程版本与该工程在单节点上的部署状态；
不展示 NATS、PostgreSQL、路径、进程命令或其他底层实现。

## 2. 内容与安全边界

契约必须包含 `activationId`、`revision`、节点、部署、工程、Release、站点和账户的 UUID 身份。客户在
工程部署阶段选择访问端口 `P`，平台不设置端口池；当前原生运行组按下表保留连续四个端口：

| 用途                       | 部署端口                       |
| -------------------------- | -----------------------------: |
| Project Gateway 对外入口   |                            `P` |
| Runtime API 回环监听       |                        `P + 1` |
| Runtime Engine 回环监听    |                        `P + 2` |
| Collector 健康检查回环监听 | `P + 3`（仅启用 Collector 时） |

`P` 必须位于 `1024..65532`。Center 在同一节点的部署记录内检查四端口区间是否重叠，NodeAgent 在启动
计划生成阶段检查操作系统真实监听占用；实际进程 bind 仍是防止检查后竞争的最终门禁。

`binding` 是激活快照与 `deployment-binding.v1` 的不可歧义关联：`bindingId` 和
`revision` 指向同一份 Binding 修订，`bindingSha256` 是该修订的 RFC 8785（JCS）UTF-8
规范化 JSON 的 `sha256:<64 lowercase hex>`。NodeAgent 从已认证的 Binding 响应计算该摘要，随后
将它与本地 Foundation/Secret 派生结果一同写入 Activation；重放或恢复时必须重新计算并比对三者。
Binding 身份、修订、摘要或 Release 身份任一不一致时不得激活。摘要字段不包含 Activation 自身，
因此不形成自引用。

未启用 Collector 时不得出现 `collectorHealthLoopback`、`collector` 工件或
`collector-nats` Secret；启用 Collector 时三者必须同时出现。`components` 不允许以
`enabled=false` 声明一个 Collector：不存在即表示未启用，存在即表示由该激活快照负责启动。

`foundation.postgres` 与 `foundation.natsJetStream` 都只传递已准备分配的 ID、revision、可复核摘要和
`ready` 状态；它们不包含 URL、DSN、账号或凭据。实际凭据仅能以 Secret 描述符给出：`ref`、
`schemaVersion`、`sha256`、`revision` 和 `expiresAt`。NodeAgent 必须在安全通道另行取得 Secret，核验
摘要、版本和有效期后才可物化为私有文件。

Release 的内层工件只能使用固定的 Release 文件名和固定的物化布局标识：`client-assets.tar.zst` →
`client`、`runtime-artifact.tar.zst` → `runtime`，可选 `collector-artifact.tar.zst` → `collector`。每项都有
摘要且必须 `readOnlyMount=true`；NodeAgent 必须在启动组件前实际建立只读挂载，而不是仅依赖文件权限。

`components` 只能声明 Project Gateway、Runtime API、Runtime Engine 与可选 Collector 的服务组、启用状态
和依赖关系。契约严格禁止命令、可执行路径、工作目录、环境变量、shell 片段和任何额外字段；这些只能由
节点安装包内受控 Launcher 决定。

## 3. 处理顺序

1. NodeAgent 获取并验证 Binding 与 Release，计算 Binding JCS 摘要。
2. NodeAgent 的本地 Foundation provisioner 验证 PostgreSQL/NATS 分配已准备，取得并校验所需
   Secret 描述符；任一项失败则不生成或切换 Activation。
3. NodeAgent 将这些受验证输入派生为 Activation，原子持久化后复验 schema、身份、revision、时效、
   Binding 摘要和 Release 摘要。
4. 下载并验证 Release，按固定布局物化工件，建立真实只读挂载。
5. 仅通过本地受信任 Launcher 按组件依赖启动；不得把 Center 输入解释为 shell 或任意路径。

机器可读定义和样例位于：

- `contracts/runtime/runtime-activation-v1.schema.json`
- `contracts/runtime/fixtures/runtime-activation-v1.*.json`
