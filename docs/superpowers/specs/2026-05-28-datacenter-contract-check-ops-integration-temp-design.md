# 数据中心契约检查与运维发布部署集成临时设计

## 1. 文档定位

本文档是数据中心契约检查设计的临时补充，用于在运维发布部署能力尚未完全实现前，先冻结数据中心、`dev_core`、`node_agent` 在发布和部署链路中的职责边界。

本文档重点关注数据中心需要提供给运维侧的契约检查能力，以及 `dev_core` 在发布工程前如何调用该能力。本文档不替代节点侧运行态数据引擎设计，也不定义 `node_agent` 的具体拓扑规划实现。

关联文档：

- `docs/superpowers/specs/2026-05-28-datacenter-contract-check-design.md`
- `docs/节点侧运行态数据引擎设计.md`
- `docs/节点侧客户端引擎与工程运行网关设计.md`
- `docs/contracts/node-agent-runtime-protocol.md`

## 2. 核心结论

发布和部署是两个独立操作。

- 发布由 `dev_core` 编排，产出不可变发布版本和 `.ifp` 工程制品。
- 部署由 `dev_core` 编排节点部署记录和节点命令，节点侧由 `node_agent` 下载 `.ifp` 并落地运行。
- 数据中心不承担发布和部署编排，只提供数据域契约检查和数据域 artifact。
- 数据中心契约检查是发布前门禁的一部分，负责判断数据域配置是否能生成运行态可解析的数据契约。
- 节点安装包后期只携带基础镜像和基础运行器，具体工程产物通过部署命令在线下发到节点。

## 3. 分层边界

```text
data_service
  ├─ 数据中心配置
  ├─ 数据中心契约检查
  └─ DataDomainArtifact

dev_core
  ├─ Designer 契约检查编排
  ├─ 数据中心契约检查调用
  ├─ RuntimeSecuritySnapshot
  ├─ IFP 发布制品组装
  ├─ 发布版本记录
  └─ 节点部署命令编排

node_agent
  ├─ 下载并校验 IFP
  ├─ 读取 manifest / datacenter / security / runtimeCapabilities
  ├─ 结合部署 profile 和节点资源 profile 生成拓扑
  └─ 启动 compose 或原生进程计划
```

职责约束：

- `data_service` 不生成最终 `.ifp`。
- `data_service` 不决定部署到哪些节点。
- `data_service` 不决定节点侧启动几个容器或进程。
- `dev_core` 不复制数据域检查规则。
- `node_agent` 不理解数据点、查询、报警和协议建模的业务语义，只消费发布制品和运行拓扑。

## 4. 数据中心产物定义

数据中心对外提供的发布态产物称为 `DataDomainArtifact`。当前实现中接口仍可沿用：

```text
GET /api/v1/data/projects/{projectId}/artifact
```

该 artifact 是 `dev_core` 组装 `.ifp` 的输入，不是节点侧完整部署包。

`DataDomainArtifact` 应包含：

- `version`、`projectId`、`generatedAt`
- `connections`
- `datapoints`
- `queries`
- `compute`
- `alarms`
- `mqtt`
- `protocols`
- `builtinStores`
- 数据域运行能力提示

其中数据域运行能力提示用于帮助 `dev_core` 生成最终 `runtimeCapabilities`。它描述“工程使用了哪些数据能力”，不描述“节点实际启动几个副本”。

示例：

```json
{
  "dataRuntimeHints": {
    "collectors": [
      {
        "protocol": "s7",
        "connectionIds": ["s7-line-1"],
        "shardPolicy": "connection"
      }
    ],
    "builtinStores": ["builtin.realtime", "builtin.timeseries"],
    "features": {
      "query": true,
      "realtime": true,
      "history": true,
      "alarm": true,
      "compute": false
    }
  }
}
```

## 5. IFP 制品与数据中心的关系

最终 `.ifp` 由 `dev_core` 生成。数据中心产物进入 `.ifp` 的 `datacenter.json` 或等价文件。

`.ifp` 至少需要包含：

```text
manifest.json
project.json
datacenter.json
runtime-security.json
runtime-permissions.json
runtime-capabilities.json
package-checksums.json
```

`datacenter.json` 保存数据域契约本体。`runtime-capabilities.json` 保存运行能力图，由 `dev_core` 基于 Designer 产物、数据中心 artifact、运行态安全快照和发布配置生成。

两者边界：

- `datacenter.json` 回答“数据域对象是什么、引用关系是什么、运行态如何解析数据能力”。
- `runtime-capabilities.json` 回答“运行这个工程需要哪些运行角色、内置库、采集能力和内部组件”。
- 部署 profile 回答“本次部署在指定节点上按什么执行方式、站点类型和副本策略运行”。

## 6. 运维发布前调用数据中心检查

工程发布入口在 `dev_core`。发布工程前，`dev_core` 必须按顺序执行：

1. 校验工程基础信息、版本号和操作者权限。
2. 调用 Designer 契约检查。
3. 调用数据中心契约检查。
4. 两个领域均无阻断项后，读取 Designer 发布态产物和 `DataDomainArtifact`。
5. 生成 `.ifp`。
6. 保存发布版本记录和 artifact 元数据。

数据中心检查调用：

```text
POST /api/v1/data/projects/{projectId}/contract-checks/run
```

请求体：

```json
{
  "scope": "project",
  "mode": "contract",
  "trigger": "dev_core.publish"
}
```

`mode=contract` 不主动连接用户现场外部接入源，只检查静态配置、引用关系和 artifact 生成能力。

`dev_core` 处理规则：

- `blocking=true`：中止发布，发布记录标记失败，构建日志写入阻断摘要。
- `blocking=false`：继续读取 `DataDomainArtifact` 并组装 `.ifp`。
- `dev_core` 只展示数据中心返回的问题，不重新实现数据域规则。

## 7. 数据中心检查能力边界

数据中心契约检查需要保证以下内容：

- 数据点 path 唯一且可发布。
- 启用数据点引用的来源对象存在。
- 查询引用的接入源存在，SQL 或查询配置满足运行态只读与参数规则。
- MQTT/Kafka/HTTP/WebSocket/实时库/OPCUA/Modbus/S7 等工作台对象能生成运行态契约。
- 计算单元输入、输出和依赖闭合。
- 报警规则目标数据点、条件、等级和运行态契约完整。
- IF 内置运行库契约完整，例如 `runtimeKey`、schema、namespace、topic prefix。
- `DataDomainArtifact` dry-run 可以成功生成并完成 JSON 序列化。
- artifact 不泄漏开发态本机端口、开发库名、内部密钥或明文密码。

数据中心契约检查不保证以下内容：

- 用户现场外部接入源在发布时一定可达。
- 节点侧镜像是否已安装。
- 节点资源是否足够。
- 节点端口池是否可分配。
- 部署 profile 是否适配目标节点。
- `node_agent` 最终生成的 compose 或原生进程计划是否能启动。

这些内容属于部署前节点适配检查或节点侧部署执行检查。

## 8. 发布后部署阶段的数据中心影响

部署阶段不再调用数据中心开发态配置作为事实源。节点侧运行只消费 `.ifp`。

部署命令需要携带或可解析以下 artifact 信息：

```json
{
  "artifact": {
    "url": "/api/v1/publish/deployment/{deploymentId}/download",
    "sha256": "hex",
    "size": 123456,
    "version": "1.0.0"
  },
  "deploymentProfile": {
    "executionMode": "service-linux-container",
    "siteType": "server",
    "siteId": "server-main",
    "scaleProfile": "single"
  }
}
```

`node_agent` 下载 `.ifp` 后读取 `datacenter.json` 和 `runtime-capabilities.json`，再结合部署 profile、节点资源 profile 生成：

- `runtime-topology.json`
- `docker-compose.runtime.yml`
- `native-processes.json`
- 工程 `.env`
- 健康检查计划

数据中心不参与这一阶段的在线决策。

## 9. runtimeCapabilities 与部署 profile 的边界

`runtimeCapabilities` 属于发布制品，描述工程能力需求。它应尽量稳定，同一个发布版本部署到不同节点时保持一致。

示例：

```json
{
  "schemaVersion": "1.0",
  "requiredRoles": ["runtime_api", "writer"],
  "optionalRoles": ["collector", "alarm", "compute"],
  "collectors": [
    {
      "protocol": "s7",
      "connectionIds": ["s7-line-1"],
      "shardPolicy": "connection"
    }
  ],
  "builtinStores": ["builtin.realtime", "builtin.timeseries"],
  "internalComponents": {
    "engineBus": "required_when_multi_role",
    "engineCoord": "required_when_ha_or_sharding"
  },
  "features": {
    "query": true,
    "realtime": true,
    "history": true,
    "alarm": true,
    "compute": false
  }
}
```

部署 profile 属于部署操作，描述本次如何运行：

```json
{
  "executionMode": "service-linux-container",
  "siteType": "server",
  "siteId": "server-main",
  "replicas": {
    "runtime_api": 1,
    "writer": 1,
    "alarm": 1
  }
}
```

数据中心契约检查只检查能否生成能力需求，不检查副本数是否合理。

## 10. 结果模型补充

数据中心契约检查结果需要能被运维发布流程直接消费。已有 `blocking` 字段继续作为发布门禁唯一依据。

建议在结果中保留以下信息：

```ts
interface ContractCheckRun {
  projectId: string
  scope: 'project' | 'module' | 'object'
  mode: 'contract' | 'contract_with_diagnostics'
  trigger?: 'datacenter.ui' | 'dev_core.publish'
  status: 'passed' | 'warning' | 'pending' | 'failed'
  blocking: boolean
  summary: ContractCheckSummary
  issues: ContractCheckIssue[]
  artifact?: {
    dryRunPassed: boolean
    schemaVersion: string
    sections: string[]
    warnings: string[]
  }
}
```

`artifact` 摘要只描述 dry-run 结果，不返回完整 artifact，避免检查接口和 artifact 获取接口职责混在一起。

## 11. 当前实现需要重点 review 的风险

当前发布链路需要重点防止以下问题：

- `data_service` artifact 中存在 `compute`、`alarms`、`builtinStores`，但 `.ifp` 的 `datacenter.json` 未完整写入。
- `data_service` artifact 中存在 OPCUA/S7/Modbus/TDengine 协议区块，但 `dev_core` 打包时只保留 Kafka/HTTP/WebSocket/Redis。
- `datacenter.json` 与 `manifest.json` 中的数据点数量、能力列表不一致。
- 发布前只调用 artifact 接口，不先调用数据中心契约检查。
- artifact dry-run 通过，但最终 `.ifp` 组装时丢字段，导致节点侧无法规划拓扑。
- 部署命令没有携带 artifact 下载描述和校验摘要，导致节点侧部署协议不闭合。

这些风险不要求数据中心替运维修复，但数据中心契约检查需要提供足够明确的结构化结果，让 `dev_core` 可以在发布前阻断。

## 12. 成功标准

- 数据中心可以独立运行项目级契约检查。
- 运维发布流程可以调用数据中心项目级契约检查。
- 外部现场接入源不可达不会阻断发布。
- 数据域配置缺失、引用断裂、artifact dry-run 失败会阻断发布。
- `dev_core` 能基于数据中心返回的 `blocking` 中止或继续发布。
- `DataDomainArtifact` 能作为 `.ifp` 中 `datacenter.json` 的完整输入。
- 发布制品能清楚区分 `datacenter.json`、`runtime-capabilities.json` 和部署 profile。
- 节点侧只依赖 `.ifp` 和部署命令，不回读数据中心开发态数据库。

## 13. 后续归并建议

本临时文档确认后，建议拆分归并：

- 数据中心契约检查调用和结果模型归并到 `2026-05-28-datacenter-contract-check-design.md`。
- IFP 制品结构归并到发布部署 API 或工程发布制品设计。
- `runtimeCapabilities` 与部署 profile 边界归并到节点侧运行态数据引擎设计。
- NodeAgent 下载、校验和拓扑规划细节归并到 `node-agent-runtime-protocol.md`。
