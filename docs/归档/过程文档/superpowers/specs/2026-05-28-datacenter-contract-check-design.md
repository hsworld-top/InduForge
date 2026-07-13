# 数据中心契约检查设计

## 1. 背景

数据中心开发态允许用户配置接入源、数据点、查询、协议变量、计算单元和报警规则，并在后续发布流程中生成运行态数据契约。多数外部接入源位于用户现场网络，开发态通常无法真实连通，因此契约检查不能以外部连通性作为发布硬门槛。

本设计将数据中心契约检查定义为“静态配置、引用关系和运行态契约生成能力”的检查，不替代现场连通性诊断，也不替代 Designer 契约检查和运维发布编排。

## 2. 功能定义

数据中心契约检查用于回答一个问题：当前数据中心配置是否足够完整、引用是否闭合、能否生成运行态可解析的数据契约。

核心能力：

- 支持数据中心开发完成后单独运行检查。
- 支持发布工程前由运维发布流程调用。
- 支持项目级、模块级、对象级检查。
- 支持检查过程进度展示，用户能看到正在检查的模块和对象。
- 支持结构化问题列表，包含级别、影响、原因、建议和跳转修复入口。
- 支持可选连接诊断，但外部接入源连通性失败默认不阻断发布。

## 3. 与其他领域的关系

Designer 契约检查与数据中心契约检查分域提供，由发布流程按顺序调用：

1. 调用 Designer 契约检查，验证页面发布态 Schema、组件绑定、页面资源等。
2. 调用数据中心契约检查，验证数据域配置和运行态数据契约。
3. 两边均无阻断问题后，发布流程继续生成和下发工程制品。

发布流程负责汇总结果和拦截，不复制 Designer 或数据中心内部规则。

### 3.1 与 dev_core 发布编排的关系

工程发布、部署和运维操作属于 `dev_core`。`data_service` 不主动编排发布，也不直接决定工程是否发布；它只提供数据中心发布契约检查接口，并返回结构化检查结果。

`dev_core` 在发布工程时必须调用数据中心契约检查：

1. `dev_core` 接收发布请求，并完成工程访问权限、发布权限和 Designer 发布态契约检查。
2. `dev_core` 使用当前发布请求的 `Authorization` 调用 `data_service`：

```text
POST /api/v1/data/projects/{projectId}/contract-checks/run
```

请求体使用：

```json
{
  "scope": "project",
  "mode": "contract",
  "trigger": "dev_core.publish"
}
```

3. `data_service` 返回 `blocking/summary/issues`。
4. 如果 `blocking=true`，`dev_core` 中止发布，发布记录标记为失败，构建日志写入阻断摘要，并把阻断项返回给运维前端。
5. 如果 `blocking=false`，`dev_core` 继续读取 `data_service` artifact 并生成 IFP。

`dev_core` 不解析数据中心内部规则，不复刻接入源、数据点、查询、计算或报警检查逻辑；只按 `blocking` 进行门禁判断，并展示 `issues`。

### 3.2 与发布制品和部署的边界

数据中心对外提供的发布态数据产物称为 `DataDomainArtifact`。当前接口仍沿用：

```text
GET /api/v1/data/projects/{projectId}/artifact
```

`DataDomainArtifact` 是 `dev_core` 组装最终 `.ifp` 工程制品的输入，不是节点侧完整部署包。数据中心契约检查只保证该数据域产物可生成、引用闭合、结构完整、可序列化，并且不泄漏开发态内部配置或明文密钥。

`DataDomainArtifact` 至少应覆盖：

- `connections`
- `datapoints`
- `queries`
- `compute`
- `alarms`
- `mqtt`
- `protocols`
- `builtinStores`

未来最终 `.ifp` 制品由 `dev_core` 组装，节点侧 `node_agent` 只消费 `.ifp`、部署 profile 和节点资源 profile。数据中心不决定节点实际启动哪些容器、启动几个副本、是否扩容，也不检查节点镜像、端口池和部署 profile 适配性。

当前开发边界限定在 `data_service` 和 `datacenter`：本设计只要求数据中心提供可被运维发布流程调用的检查接口和稳定 `DataDomainArtifact`；`dev_core` 的发布制品装配、部署命令和节点适配检查作为后续运维实现范围。

## 4. 检查边界

### 4.1 覆盖对象

第一版纳入以下对象：

- 接入源：关系库、MQTT、Kafka、HTTP、WebSocket、Redis、IF 实时库、OPC UA、Modbus、S7 等。
- 数据点：手动数据点、接入源生成数据点、查询生成数据点、计算输出数据点。
- 查询：SQL 查询定义、参数、只读约束、数据点关系。
- 协议工作台对象：MQTT Tag、Kafka 变量、HTTP 请求、WebSocket 会话、实时库 key、OPC UA 节点、Modbus 寄存器、S7 变量。
- 计算单元：输入、输出、依赖、启用状态、运行态描述。
- 报警规则：目标数据点、阈值/条件、等级、动作、启用状态、运行态契约。
- artifact dry-run：数据中心侧运行态数据契约生成结果。

### 4.2 明确不做

- 不要求外部设备或外部服务在开发态真实可达。
- 不把外部 Kafka、HTTP、WebSocket、Redis、OPC UA、Modbus、S7 连通失败作为发布阻断。
- 不替代 Designer 契约检查。
- 不替代运维发布编排。
- 不检查最终 `.ifp` 装配过程。
- 不检查节点侧镜像、端口、资源和部署 profile。
- 不做运行态长期健康监控。
- 不检查用户现场网络质量。
- 不因为未启用草稿对象阻断发布。

### 4.3 内置依赖例外

平台自身必需依赖不属于外部现场接入源。如果运行态必需的内置关系库、内置实时库、artifact 存储或运行态配置库配置缺失，会影响服务启动或契约加载，应作为阻断问题处理。

## 5. 检查级别

检查结果使用四个级别，但级别含义以运行态影响为准。

| 级别      | 定义                                                                                   | 发布前是否阻断 |
| --------- | -------------------------------------------------------------------------------------- | -------------- |
| `failed`  | 静态契约无法生成，或运行态必需配置缺失，发布后服务或数据域契约必然无法解析             | 是             |
| `pending` | 启用对象未配置完整，不能形成稳定运行配置，但不涉及外部连通性                           | 是             |
| `warning` | 配置完整、契约可生成，但缺少开发态样本、最近预览失败、外部服务未验证或数据质量存在风险 | 否             |
| `passed`  | 配置完整、引用正确、artifact dry-run 通过                                              | 否             |

未启用对象和草稿对象纳入检查结果，但默认不阻断发布。启用对象、被发布对象引用的对象、运行态 artifact 必需对象出现问题时才阻断。

## 6. 阻断规则

发布门禁不直接根据级别名称猜测，而根据 `blocking` 字段判断。

默认规则：

- `failed` 阻断。
- 启用对象的 `pending` 阻断。
- 未启用对象的 `pending` 不阻断，作为提示展示。
- `warning` 不阻断。
- `category=diagnostic` 默认不阻断。
- 外部接入源连通性失败不阻断。
- 配置缺失、引用断裂、artifact 生成失败阻断。
- 平台内置依赖配置缺失或不可生成运行契约时阻断。

## 7. 检查范围

### 7.1 项目级检查

`scope=project`

检查整个数据中心项目。发布前门禁默认使用该范围。

### 7.2 模块级检查

`scope=module`

检查指定模块，适合数据中心开发态局部自检。模块包括：

- `access_source`
- `datapoint`
- `query`
- `mqtt`
- `kafka`
- `http`
- `websocket`
- `realtime_store`
- `opcua`
- `modbus`
- `s7`
- `compute`
- `alarm`
- `artifact`

### 7.3 对象级检查

`scope=object`

检查单个对象，适合工作台中“检查当前对象”。对象级检查复用同一套规则，只过滤目标对象及其直接依赖。

## 8. 检查模式

### 8.1 契约模式

`mode=contract`

默认模式。只检查静态配置、引用关系和 artifact dry-run，不主动连接外部接入源。

### 8.2 契约加诊断模式

`mode=contract_with_diagnostics`

用户主动打开“包含连接诊断”后使用。该模式可以执行连接测试、读取最近预览状态、检查最后值质量。诊断结果默认不阻断发布，除非诊断对象是平台内置必需依赖。

## 9. 检查流程

全量检查采用规则流水线，而不是单个大函数一次性扫描。这样前端可以展示正在检查的阶段和对象，后端也便于扩展规则。

### 9.1 准备上下文

加载项目基础信息、数据点、接入源、查询、协议对象、计算单元、报警规则，并建立索引：

- 按对象 ID 索引。
- 按数据点 path 索引。
- 按 sourceType 和 sourceConfig 索引。
- 按 connectionId 索引。
- 按启用状态索引。

这一步失败通常是系统错误或平台内置依赖问题。

### 9.2 静态配置检查

检查每类对象的必填字段、枚举值、JSON 结构、地址格式、认证结构、超时配置、DB/runtimeKey 等。

示例：

- HTTP 请求 URL 为空：阻断。
- Kafka 接入源缺 brokers：阻断。
- WebSocket URL 协议不可解析：阻断。
- IF 实时库缺 runtimeKey：阻断。
- 外部 Redis 地址完整但开发态无法连接：诊断 warning。

### 9.3 引用关系检查

检查对象之间的引用是否存在、是否同项目、状态是否合理。

示例：

- 数据点引用的 Kafka field 不存在：阻断。
- 报警规则目标数据点不存在：阻断。
- 计算单元输入数据点不存在：阻断。
- 未启用草稿对象引用不存在：提示但不阻断。

### 9.4 运行态契约生成检查

对每类对象尝试生成运行态契约片段。不连接外部服务，只验证生成结构是否完整、可序列化、可被运行态识别。

示例：

- 数据点 sourceConfig 缺运行态必需字段：阻断。
- 接入源类型不被运行态识别：阻断。
- 启用报警规则无法生成规则契约：阻断。

### 9.5 Artifact Dry-Run

复用或调用数据中心运行态 artifact 生成逻辑，检查最终数据契约是否结构完整、路径唯一、引用闭合。

artifact dry-run 失败是发布阻断问题。

### 9.6 可选诊断

仅在 `mode=contract_with_diagnostics` 执行。包括连接测试、开发态预览最近结果、最后值质量等。

外部接入源诊断失败默认输出 `warning`，不阻断发布。

### 9.7 汇总门禁

按检查结果计算整体状态、阻断数量、模块统计和级别统计，并生成可发布结论。

## 10. 结果模型

### 10.1 检查运行结果

```ts
interface ContractCheckRun {
  id: string
  projectId: string
  scope: 'project' | 'module' | 'object'
  mode: 'contract' | 'contract_with_diagnostics'
  status: 'passed' | 'warning' | 'pending' | 'failed'
  blocking: boolean
  startedAt: string
  finishedAt?: string
  progress: ContractCheckProgress
  summary: ContractCheckSummary
  issues: ContractCheckIssue[]
}
```

### 10.2 进度

```ts
interface ContractCheckProgress {
  currentStage: string
  currentModule?: string
  currentObjectName?: string
  checkedCount: number
  totalCount: number
}
```

### 10.3 汇总

```ts
interface ContractCheckSummary {
  failed: number
  pending: number
  warning: number
  passed: number
  blocking: number
}
```

### 10.4 单条结果

```ts
interface ContractCheckIssue {
  id: string
  status: 'passed' | 'warning' | 'pending' | 'failed'
  blocking: boolean
  category: 'static_config' | 'reference' | 'artifact' | 'diagnostic'
  runtimeImpact:
    | 'startup_blocker'
    | 'contract_blocker'
    | 'feature_unavailable'
    | 'data_quality_risk'
    | 'advisory'
  module:
    | 'access_source'
    | 'datapoint'
    | 'query'
    | 'mqtt'
    | 'kafka'
    | 'http'
    | 'websocket'
    | 'realtime_store'
    | 'opcua'
    | 'modbus'
    | 's7'
    | 'compute'
    | 'alarm'
    | 'artifact'
  objectType: string
  objectId?: string
  objectName?: string
  title: string
  detail: string
  suggestion: string
  jumpTarget?: ContractCheckJumpTarget
}

interface ContractCheckJumpTarget {
  routeName: string
  params: Record<string, string>
  query?: Record<string, string>
}
```

## 11. UI 原型

契约检查入口保留在数据中心左侧菜单左下角。入口状态主要通过图标颜色表达，不在入口中塞入复杂文本。

图标状态：

- 灰色：未检查。
- 主色：检查中。
- 绿色：通过。
- 黄色：存在 warning。
- 橙色：存在 pending 阻断。
- 红色：存在 failed 阻断。

多个级别同时存在时，按最高风险显示颜色：`failed > pending > warning > passed`。

鼠标悬停展示简短提示，例如：`2 个阻断，5 个风险`。

点击入口打开放大版悬浮检查面板。面板覆盖主工作区，但不改变主工作区尺寸，不强遮罩，不阻止用户继续操作主工作区。

```text
┌────左侧菜单────┬────────────────────────主工作区────────────────────────┐
│ 接入源          │                                                        │
│ 数据点          │        ┌────────────────────────────────────────┐      │
│ 查询            │        │ 契约检查                               │      │
│ 计算            │        │ [全量检查] [当前模块] [当前对象]        │      │
│ 报警            │        │ [包含连接诊断]                          │      │
│                │        │                                        │      │
│                │        │ 正在检查：HTTP 接入源配置完整性 18/62   │      │
│ 契约检查入口    │        │ failed 2 | pending 1 | warning 8        │      │
└────────────────┴────────│                                        │──────┘
                          │ 级别 | 模块 | 对象 | 问题 | 影响 | 操作 │
                          │                                        │
                          │ 问题详情：原因 / 影响 / 修复建议        │
                          └────────────────────────────────────────┘
```

面板行为：

- 点击入口打开或关闭面板。
- 检查运行中关闭面板不取消任务。
- 入口图标持续显示检查中状态。
- 检查结果默认保留，重新检查后覆盖。
- 默认隐藏 `passed` 明细，仅在用户打开“显示通过项”时展示。
- 点击“跳转修复”进入对应模块或对象，面板默认自动收起。

面板内容：

- 顶部：标题、上次检查时间、关闭按钮。
- 操作行：全量检查、当前模块、当前对象、包含连接诊断。
- 进度区：当前阶段、当前模块、当前对象、进度条。
- 汇总区：failed、pending、warning、passed、blocking 数量。
- 筛选区：级别、模块、只看阻断、搜索、显示通过项。
- 结果表：级别、模块、对象、问题、运行态影响、操作。
- 详情区：原因、影响、修复建议、跳转修复。

## 12. 使用流程

### 12.1 数据中心开发态自检

1. 用户在数据中心完成某个模块或对象配置。
2. 点击左下角契约检查入口。
3. 打开悬浮检查面板。
4. 选择“当前对象”或“当前模块”。
5. 面板展示正在检查的对象和进度。
6. 检查完成后展示问题列表。
7. 用户点击问题查看详情并跳转修复。
8. 修复后重新运行当前对象或当前模块检查。

### 12.2 项目级全量自检

1. 用户点击“全量检查”。
2. 后端按规则流水线执行项目级检查。
3. 面板展示阶段进度和汇总结果。
4. 用户优先处理阻断问题。
5. 没有阻断问题时，入口图标显示通过或 warning 状态。

### 12.3 发布前门禁

1. 运维发布流程调用 Designer 契约检查。
2. Designer 检查无阻断后，调用数据中心项目级契约检查。
3. 数据中心检查无阻断后，继续生成工程制品。
4. 任一检查存在 `blocking=true` 结果时，发布流程中止并展示汇总。

## 13. 检查规则示例

### 13.1 接入源

- 启用的外部接入源缺少运行态必需配置：`failed`，阻断。
- 外部接入源开发态连接失败：`warning`，不阻断。
- 内置实时库缺 runtimeKey：`failed`，阻断。
- 接入源类型不被 artifact 识别：`failed`，阻断。

### 13.2 数据点

- path 为空或重复：`failed`，阻断。
- sourceType 不合法：`failed`，阻断。
- 启用数据点引用的来源对象不存在：`failed`，阻断。
- 数据点没有最近值：`warning`，不阻断。

### 13.3 查询

- SQL 为空：`failed`，阻断。
- 查询所属连接不存在：`failed`，阻断。
- SQL 不满足只读约束：`failed`，阻断。
- 外部关系库开发态不可达：`warning`，不阻断。

### 13.4 协议工作台对象

- Kafka 变量引用的 Topic 映射不存在：`failed`，阻断。
- HTTP 请求 URL 为空：`failed`，阻断。
- WebSocket 会话 URL 不可解析：`failed`，阻断。
- 实时库 key 元数据不存在但数据点仍引用它：`failed`，阻断。
- OPC UA、Modbus、S7 变量地址格式非法：`failed`，阻断。

### 13.5 计算单元

- 启用计算单元输入数据点不存在：`failed`，阻断。
- 启用计算单元没有输出数据点：`failed`，阻断。
- 未启用计算单元配置不完整：`warning` 或非阻断 `pending`。
- 脚本运行逻辑不在契约检查中完整执行。

### 13.6 报警规则

- 启用报警规则目标数据点不存在：`failed`，阻断。
- 目标数据点类型不支持当前规则：`failed`，阻断。
- 启用报警规则阈值或等级配置缺失：`pending`，阻断。
- 未启用报警草稿配置不完整：提示但不阻断。

### 13.7 Artifact

- 数据中心运行态契约无法生成：`failed`，阻断。
- 生成结果缺必要 section：`failed`，阻断。
- 生成结果可序列化但存在未验证外部连接：`warning`，不阻断。

## 14. API 设计

第一版建议由 `data_service` 提供接口：

```text
POST /api/v1/data/projects/{projectId}/contract-checks/run
GET  /api/v1/data/projects/{projectId}/contract-checks/latest
GET  /api/v1/data/projects/{projectId}/contract-checks/{runId}
```

运行请求：

```ts
interface RunContractCheckRequest {
  scope: 'project' | 'module' | 'object'
  mode?: 'contract' | 'contract_with_diagnostics'
  module?: ContractCheckIssue['module']
  objectType?: string
  objectId?: string
  trigger?: 'datacenter.ui' | 'dev_core.publish'
}
```

第一版可以同步返回完整结果；如果全量检查耗时明显，再扩展为异步 runId + 轮询或 SSE 推送。前端结果模型保持一致，避免后续重做 UI。

## 15. 数据存储

第一版建议实时计算并在内存或短期缓存中保留最近一次结果。若需要跨刷新保留结果，可新增检查运行记录表。

建议持久化字段：

- run id
- project id
- scope
- mode
- status
- blocking
- summary
- issues
- started_at
- finished_at
- created_by

检查结果属于诊断快照，不作为业务配置源。

## 16. 成功标准

- 数据中心可以独立运行项目级、模块级、对象级契约检查。
- 检查结果能区分阻断问题和非阻断风险。
- 外部接入源开发态不可达不会阻断发布。
- 配置缺失、引用断裂、artifact dry-run 失败会阻断发布。
- 未启用草稿对象的问题会展示但不阻断。
- 悬浮检查面板不压缩主工作区，不强遮罩。
- 发布流程可以先调用 Designer 契约检查，再调用数据中心契约检查。

## 17. 风险与约束

- 检查规则需要和 artifact 生成逻辑共用核心路径，避免检查通过但生成失败。
- 跳转修复需要各模块提供稳定路由和对象定位参数。
- 连接诊断不能混入默认发布门禁，否则会误伤现场网络不可达场景。
- 结果模型要保持可扩展，后续可接入 Designer、运维发布和节点侧运行诊断。
