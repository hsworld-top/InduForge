# Runtime Data Plane V1 契约

## 1. 定位与范围

本文冻结 Go `runtime-engine` 与 .NET `industrial_collector` 共用的运行数据面 V1。机器可读源码位于
[`contracts/runtime/`](../../../contracts/runtime/)，所有对象均使用 JSON Schema draft 2020-12。

V1 的部署基线是 **`edge-single`：每个 `ProjectDeployment` 的运行数据面只有一个业务 owner，故障后由
人工完成 fencing、检查与恢复**。它不是 HA 方案：不承诺自动故障接管、双机切换、R3、跨节点 WAL 复制或
无损恢复。后续 `site-ha` 必须另行版本化，不能将其行为倒灌到 V1。

本契约不取代 [NodeAgent 与运行系统协议](./node-agent-runtime-protocol.md) 的发布 Saga，也不定义数据库物理
表结构、Secret 存储实现或设备驱动私有协议。

## 2. 机器可读对象

| Schema | Schema version | 作用 |
| --- | --- | --- |
| `point-event.schema.json` | `data.raw.v1`、`data.computed.v1` | Collector 原始点位与 Engine 计算点位共用事件 |
| `alarm-event.schema.json` | `alarm.event.v1` | 结构化报警状态变更与独立 data-gap 事件 |
| `runtime-engine-config.schema.json` | `runtime-engine.config.v1` | Engine Artifact 挂载、角色 owner/epoch、JetStream 消费者与状态库要求 |
| `runtime-project-artifact.schema.json` | `runtime-project-artifact.v1` | `ProjectArtifactV1` 1.0 的 Engine compute/alarm 输入适配对象 |
| `collector-runtime-artifact.schema.json` | `collector-runtime-artifact.v1` | 环境无关、不可变的逻辑采集 Artifact（含协议逻辑点地址） |
| `collector-runtime-binding.schema.json` | `collector-runtime-binding.v1` | deployment 现场资源、Secret 引用与 owner 绑定 |
| `runtime-health-status.schema.json` | `runtime-health-status.v1` | Runtime/Collector 健康与业务新鲜度扩展对象 |

`fixtures/*.valid*.json` 必须通过对应 Schema；`fixtures/*.invalid.json` 必须被拒绝。`common.schema.json`
只提供共享定义，不作为独立消息。

## 3. 隔离、身份与版本

- 一个 `deploymentId` **恰好对应一个** `accountId`；该 Account 只能承载该 deployment 的运行数据。
  一个 Account 不得复用给另一个 deployment，一个 deployment 也不得在 V1 使用多个 Account。
- `deploymentId`、`accountId`、`artifactId`、`bindingId`、`collectorId`、`pointId` 与 `ownerId` 都是稳定身份，
  不能以显示名称代替。所有事件都必须携带前四项中适用的身份。
- `artifactRevision` 与 `bindingRevision` 单调增加且彼此独立。部署时必须精确匹配
  `artifactId + artifactRevision + artifactDigest`；digest 是小写十六进制 SHA-256，格式
  `sha256:<64 hex>`。同一 revision 的 digest 不得变化。
- `schemaVersion` 是协议兼容边界，不能用 Deployment desired revision、Binding revision、Artifact revision 或
  程序版本代替。消费者遇到未知 schemaVersion 必须拒绝并告警，不得猜测字段含义。

`runtime-project-artifact.v1` 与 `runtime-engine.config.v1` 中的 projectId，以及该 Artifact 内 dataPoints、computeUnits、
alarmItems、alarm conditions 的 id 和所有 datapointId 引用，使用小写 canonical UUID；它们对应 data_service 的数据库主键，
不得以 `project-line1` 一类显示/逻辑 ID 代替。deployment/account/owner/collector/consumer 等运行身份仍使用 stableId。
Collector Artifact 的 projectId、pointMappings.datapointId/connectionId/variableId，以及 Binding、raw event source 与 data-gap
中的 connectionId/variableId 同样是相同正式对象的 canonical UUID；collectorId/deployment/account/owner/artifact/binding
仍为运行 stableId。raw event 的 pointId 必须与 mapping.datapointId 及 Runtime Project dataPoints.id 相等。

Runtime Project Artifact 的 dataPoint、compute input/output、`datapoint_change` 以及 condition variables 的 `dataType` 统一引用
`common.schema.json#/$defs/dataPointType`：`bool`、`int8`、`uint8`、`int16`、`uint16`、`int32`、`uint32`、`int64`、`uint64`、
`float32`、`float64`、`decimal`、`string`、`bytes`、`datetime`、`object`、`array`。Collector Artifact 的 protocol mapping 则仍只允许
其中的标量 union（不包括 `object`、`array`），不得因公共枚举扩大采集驱动输入范围。

## 4. Subject 与 payload identity

事件的 `subject` 由 NATS 路由使用，payload 的 `deploymentId/accountId/pointId` 才是可审计身份。发布者与
消费者都必须同时校验二者：只匹配 subject 或只相信 payload 都不合格。

| Schema version | Subject | payload identity |
| --- | --- | --- |
| `data.raw.v1` | `data.raw.<pointId>` | `deploymentId`、`accountId`、`pointId`、`ownerId`、`epoch`、`sequence` |
| `data.computed.v1` | `data.computed.<pointId>` | 同上，另有 `computation.computeId/revision/inputEventIds` |
| `alarm.event.v1` | `alarm.event` | transition 为 `alarmId/alarmItemId/pointIds`；data-gap 为 `collectorId/connectionId`，二者均有 `deploymentId/accountId/ownerId/epoch` |

`<pointId>` 必须与 payload `pointId` 完全相同；subject 中不允许使用路径、显示名、数据点 path 或转义后的
别名。消费者还必须确认凭据所在 Account 与 payload `accountId` 相同。

`pointId` 是正式 `data_points.id`（canonical UUID），不是采集变量身份；source 的 connectionId/variableId 同样是
正式 UUID。Artifact 中每个 `pointMappings.datapointId` 必须在一个
Artifact revision 内唯一，并且每个 `(connectionId, variableId)` 只能映射到一个正式 datapoint。运行时事件的
`pointId` 必须等于该 mapping 的 `datapointId`。此复合唯一性和跨对象相等性由 Artifact 构建器校验；JSON Schema
无法表达时不得省略该校验。

`data.raw.v1` 必须有 `source` 且禁止 `computation`；`data.computed.v1` 必须有 `computation` 且禁止 `source`。
Schema 已强制这两个分支互斥；Subject 尾段与 payload `pointId` 的相等性仍是跨字段验收项。

## 5. 点位、报警与时间语义

### 5.1 事件 ID、owner 与重放

`eventId` 是 64 位小写十六进制 SHA-256，不能随机生成。V1 使用 UTF-8 文本、字段间以 ASCII Unit Separator
(`0x1f`) 拼接，空值为零长度字段；输入中的 `0x1f` 不合法。固定公式如下：

```text
data.raw.v1      = SHA-256(schemaVersion, deploymentId, pointId, ownerId, epoch, sequence)
data.computed.v1 = SHA-256(schemaVersion, deploymentId, pointId, ownerId, epoch, sequence,
                           computeId, computeRevision, inputEventIds 的字典序连接结果)
alarm-transition = SHA-256(schemaVersion, kind, deploymentId, alarmId, operation, ownerId, epoch,
                           sourceTimestamp, alarmItemId, pointIds 的字典序连接结果,
                           stateVersion, transitionSequence)
data-gap         = SHA-256(schemaVersion, kind, deploymentId, collectorId, connectionId, ownerId, epoch,
                           fromSequence, toSequence, detectedAt, reason)
```

公式按上述字段顺序拼接；数组先按字节序排序，数组的每个元素都是一个独立字段；整数为无前导零十进制；时间严格使用
payload 中已规范化的 UTC RFC3339 `Z` 字符串；字段之间使用单个 `0x1f`，末尾不添加分隔符。哈希输出不加 `sha256:`
前缀。fixtures 是该编码的真值向量。`sequence` 是 owner/epoch 内单调递增的非负整数；Collector
重启或所有权切换必须提升 `epoch`，不得在同一 `(ownerId, epoch)` 重用 sequence。payload 不含可随投递改变的
`isReplay`、投递次数或消费时间。重复 eventId 是同一事实的 JetStream 原始消息 body bytes 重投；`processed_event` 必须保存
原始消息 body bytes 的 SHA-256（不得解析后再序列化或采用未定义的 canonical JSON），并把同 ID 的不同 bytes 摘要隔离和告警。重放来源、次数和 JetStream delivery metadata 只进入本地状态、
日志或不参与该摘要的传输元数据。

`ownerId + epoch` 是副作用 fencing token。Engine 与 Collector 在写状态、推进 checkpoint、产生 computed/alarm
事件前均要用它做 CAS；收到旧 epoch 的事件可审计保存，但不得推进当前状态或产生新副作用。

### 5.2 报警状态机

`alarm.event.v1` 有两个互斥 `kind`。`alarm-transition` 必须带 `alarmId/alarmItemId/pointIds/transition`；
`data-gap` 不得带这些字段，必须带 `dataGap`。普通报警操作只有 `RAISE`、`SEVERITY_CHANGE`、`ACK`、`CLEAR`：

- `RAISE` 的 `alarmState=OPEN`、`previousSeverity/clearedAt/ackedAt/acknowledgedBy` 全为 `null`。
- `SEVERITY_CHANGE` 保持 `OPEN`，并要求非空 `previousSeverity`。
- `ACK` 必须转为 `ACKED`，并带 `ackedAt` 与 `acknowledgedBy`。
- `CLEAR` 必须转为 `CLEARED`，并带 `clearedAt`。

每个 transition 都携带 `severity`、正式 UUID 的 `alarmItemId/conditionId/pointIds`、采样 `value/quality`、`message`、opened/updated/cleared
时间、`stateVersion` 与 `transitionSequence`。severity 使用 `^[a-z][a-z0-9_]{0,29}$`，内置与自定义键由
Artifact/报警设置校验，运行事件不得硬编码等级集合。`pointIds` 支持 derived alarm 的多输入且去重；普通 point
alarm 恰好一个 pointId 由 Artifact/报警设置校验。stateVersion 和 transitionSequence 必须单调递增，且相邻
transition 的状态机合法性由报警状态库用 owner/epoch CAS 验证。`DATA_GAP` 是独立操作，仅限 `kind=data-gap`。
`alarmId` 则是报警状态库在首次 `RAISE` 时生成、在该报警实例生命周期内保持不变的运行实例 stableId；它不是
`alarm_items.id`，也不要求或假定为 UUID。

### 5.3 值、质量与时间

- `value` 只允许 JSON `null`、boolean、number、string、array、object，递归元素也遵循同一规则。JSON 文本
  不存在 `NaN`、`Infinity`、`-Infinity`；实现必须在序列化/反序列化边界拒绝非有限浮点数，不能将它们编码为字符串。
- `quality` 只能是 `good`、`bad`、`unknown`。设备断线、超时、转换失败与未识别状态在进入本契约前归一化为
  `bad` 或 `unknown`，不得扩展字符串枚举。
- `sourceTimestamp` 是设备或采集源给出（或 Collector 在无法获得设备时间时生成）的样本时间；
  `serverTimestamp` 是产生该事件的 Collector/Engine 本地服务时间；`receivedAt` 是生产者首次摄取该事实时写入的
  不可变 ingest 时间。消费者自身接收时间只进入本地状态，不能回写 payload。三者都必须为 UTC RFC3339，使用 `Z`，
  可带 1–9 位小数秒。
- V1 允许回放、重复投递和乱序。消费者以 `eventId` 去重，按 `(epoch, sourceTimestamp, sequence)` 判断是否可以
  更新当前值；较早事件不得覆盖较新当前值，但仍可作为历史事实写入。不得因为乱序而重新分配 eventId。

## 6. Artifact、Binding 与 Secret 边界

Artifact 是可审计、环境无关且不可变的逻辑采集模型：协议能力、逻辑连接、变量、正式 datapoint 映射、
`addressSchemaVersion`、协议逻辑地址、`readOptions`、`elementCount`、正式 UUID 的 connection/variable/datapoint 映射、连接默认采集配置、点位的
`acquisitionMode/acquisitionOverrides/effectiveAcquisition`、启用标记以及 WAL 耐久语义均属于 Artifact。逻辑地址
例如 OPC UA NodeId、Modbus area/address/station，不等同现场 endpoint。Artifact 不得包含现场 host/port/endpoint、
设备账号、密码、Token、证书、私钥、NATS 凭据或短期下载 URL。连接必须冻结
`protocolFamily/driverId/driverVersion/schemaVersion/enabled/defaultAcquisition`，用于节点 preflight 驱动版本和
Schema；`addressSchemaVersion` 是 `integer >= 1`，`elementCount` 是 `integer >= 1`。

V1 的采集模式不另造 `poll/subscribe`。连接以 `defaultAcquisition(intervalMs, deadband, changeOnly)` 给出默认值；
点位 `acquisitionMode=inherit` 时不得有 overrides，`override` 时必须带同三字段的 overrides。构建器必须输出并校验
每点无歧义的 `effectiveAcquisition`：inherit 等于连接默认值，override 是 connection default 与 overrides 的字段合并。
`deadband` 必须为非负 number，`intervalMs` 为 1..86400000。Artifact 本体不得自带 digest，避免自哈希循环；
Binding/Engine config 的 `artifactRef.artifactDigest` 是对挂载 Artifact 原始文件 bytes 的 SHA-256。

连接声明 `requiredOperations` 且必须包含 `point.read`。构建器必须按 `(driverId, schemaVersion)` 调用
`contracts/collector-protocols/<driverId>/address.schema.json` 与对应 connection/read-options Schema 验证 `address` 和
`readOptions`；通用 Runtime Schema 不能替代该按驱动的验证。

Binding 只针对一个 deployment，引用一个固定 Artifact identity，并补充现场 `resourceRef`、可选的零到多个
`secretRefs`、NATS credential 的 `credentialSecretRef`、`ownership` 与本 deployment/node 的 WAL 容量策略。无认证
连接可以没有 `secretRefs`；Binding 中仍只能出现 Secret 引用，绝不能出现 Secret 值。`resourceRef` 解析 endpoint
等现场连接配置，解析结果不得序列化回 status。Secret 解析在 NodeAgent/Collector 的安全目录或受限运行环境中完成，
日志、状态和错误对象不得回显敏感值。

Collector Binding 与 Engine JetStream 配置都必须引用站点 NATS 的 `serverResourceRef` 和最小权限
`credentialSecretRef`；Account 名称不是可连接地址。Engine 的 `projectArtifact` 引用和只读 `artifactMount` 指向
`runtime-project-artifact.v1` 文件，后者是构建期从 `ProjectArtifactV1` 1.0 规范化得到的**可执行**适配输入，而不是只含 ID 的索引：
dataPoints 固定 id/path/name/dataType，并保留 sourceType/sourceId/sourceConfig、runtimePermissions、refresh/status、unit/default/tags/attributes 等 SDK 快照元数据；
computeUnits 固定脚本、语言、revision、超时、依赖、标准化 datapoint 输入与完整输出/nullPolicy；alarmItems 固定 point/derived 输入别名、derivedExpression、评价模式和完整条件。它不包含现场 Binding 或 Secret。

每个 dataPoint 必带 nullable `precisionNum`。`runtimePermissions` 严格等于 `{write:{allowRoles,denyRoles,inherit}}`：两个 role
数组均唯一且元素为 stable role ID，`inherit` 为 boolean；不得编造 read/history 等平铺权限。`sourceId` 是 UUID 或 null：
`collector.point` 指向 collector point/variable UUID，`calc.output` 指向 compute unit UUID。sourceConfig 只保留无 Secret 的来源元数据，
例如 collector connection UUID 或 computeId；不得含现场 endpoint、凭据或其他 Secret。refreshMode 仅为 auto/manual/subscription，
status 仅为 active/inactive/invalid。

compute language 只允许当前 sandbox 可执行的 `js` 与 `python`；依赖是 packageName/importName/version/language 对象。trigger 是严格判别对象：`manual` 没有附加配置；schedule 严格只有 `interval`、`daily`、`weekly`、`monthly`、`yearly` 五个分支，任何分支的无关字段都必须拒绝。interval 仅为 every/unit；daily/weekly/monthly/yearly 使用 `timezone`（必须可由装载器以 Go `time.LoadLocation` 解析的 IANA timezone）和 `time`（HH:mm:ss）；weekly 必带 weekdays；monthly/yearly 的 dayRule=day 仅带 dayOfMonth，dayRule=weekday 仅带 weekOfMonth（-1 或 1..5）和 weekday，yearly 还必带 month。startAt/endAt 是可选 `common.utcRfc3339` window，maxRuns 为 1..1000000；装载器必须解析为 UTC 并拒绝 `endAt <= startAt`。`datapoint_change` 必带 datapointId/path/dataType/mode/debounceMs；deadband 只能随数值 `value_change`、`increase`、`decrease` 出现。装载器还校验 any/value_change 只用 bool/string/数值标量、increase/decrease 只用数值、rising_edge/falling_edge 只用 bool。`condition` 必带 expression（最多 1000）、entered/active/exited phases、alias/datapointId/path/dataType variables 与不超过 3600000ms 的 debounce。报警项必须有 `enabled:boolean`。报警条件 kind 固定为 threshold、range、state、transition、text_match、rate_of_change、deviation、offline、quality、stale，每项均带 operator、专用 params、severity、triggerDelayMs、clearDelayMs、deadband；transition 的 changed/rising/falling 仅接受空 params，from_to 仅接受 from/to；deviation 的 operator 固定 gt；offline 固定 is 与空 params。装载器使用 Go RE2 编译 regex，且复核 range 的 lower < upper、from_to 的 from/to 不同。`mode=point` 只允许 single 或 highest_matching：single 恰一输入/条件，highest_matching 仅数值 threshold 分级；`mode=derived` 只允许 single、至少两个 inputs、恰一个条件且禁止 offline/quality/stale。装载器还须校验所有 datapoint/compute 引用存在、输入 alias/outputKey 唯一、依赖与输入一致、输出 path/dataType 与目标 datapoint 一致，以及不存在歧义写入。

此外，装载器必须拒绝 enabled compute 对同一 output datapoint 的多写，并要求每个 enabled compute 都有唯一的
`producerAssignments` compute fence；该 fence 的 ownership 是 producer token，不能与 consumer role token 比较或混用。

WAL 的 `fsync/checksum/commitMarker` 是 Artifact 的不可变耐久语义；`maxBytes/highWatermarkBytes` 是 deployment/主机
容量策略，故在 Binding。V1 单个 deployment WAL 的 `maxBytes` 为 1MiB..1GiB（`1073741824`），与 Collector `DurableWalOptions`
的冻结上限一致。Artifact 构建器与 Binding 校验器必须验证 `0 < highWatermarkBytes < maxBytes`；此数值跨字段
比较由实现完成，JSON Schema 无法表达。Engine config 固定 `siteId/projectId/executionForm=k3s-workload`；装载器必须验证 config projectId 与挂载 runtime-project-artifact 的 projectId 相等。Engine 必须从受信只读 `release-pvc` 挂载读取 `artifactMount.artifactFile`，
并重新计算 digest，必须与 `projectArtifact` 引用完全相同后才可启动。`artifactMount.artifactFile` 的 Schema 使用 Go RE2-safe 的分段相对路径正则；装载器还必须拒绝 NUL、`.`/`..`、双斜线、symlink 和非普通文件。`producerAssignments` 是带 discriminator 的受信 producer fence：`collector` 记录 collectorId/ownership，原始事件按 payload 的 collectorId/ownerId/epoch 校验；`compute` 记录 computeId/role=compute/ownership，computed 事件按该 Assignment 与 Artifact computeId 校验；`alarm` 记录 role=alarm/ownership，仅允许报警 transition producer 使用。data-gap 由 Collector producer fence 校验，computed 由 compute fence 校验，alarm-transition 由 alarm producer fence 校验；三类都不能与 consumer role token 混为一谈。

每个 collector producer 还必须恰有一个 `collectorArtifact`：其中固定 Artifact id/revision/digest 与同一
`artifactMount.mountPath` 下的 RE2-safe 相对 artifactFile。装载器对该文件执行与 Project Artifact 相同的 16MiB、
no-follow、普通文件、TOCTOU、最具体只读 mount、原始 bytes digest 和 Schema 校验；其 projectId 必须同时等于
config 和 Project Artifact。装载后，enabled mapping 必须一一映射到 active `collector.point`：datapointId、sourceId
(variableId)、sourceConfig.connectionId 和 dataType 全相等，反向也必须恰有一个 mapping。由此 raw ingress 可按
collectorId + pointId + connectionId + variableId 校验，不需要也不得比较 producer/consumer ownership token。

Collector 只在 Binding 的 `ownerId/epoch` 与已持久化发布决议一致时激活；standby 可以校验 Artifact/Binding，
但不得连接为业务写 owner 或发布数据。

## 7. JetStream 可靠投递

`runtime-engine.config.v1` 的每个业务 Consumer 都必须是 Durable Pull Consumer，并且强制：

- `ackPolicy=explicit`；仅在本地事务成功提交后 Ack。
- 显式提供 `AckWait`、`maxDeliver`、递增或非递减 `backoff`，以及仅本 deployment Account 可访问的
  `dlq.<consumer>` Subject。
- 到达 `maxDeliver` 的消息必须带原始 subject、eventId、失败原因、投递次数进入 DLQ，并使健康状态降级；DLQ failure
  记录和最终 DLQ 发布先在数据库事务中写入 Outbox，提交成功后才 Ack 原消息；不得静默丢弃或无限重试。
- 至少一次投递是既定语义。所有副作用必须以 `eventId`、业务键和 owner/epoch 防重，不能依赖“只投递一次”。

每项 `jetStream.consumers` 显式声明 `role`、稳定 `consumerKey`、配置中的实际 `stream` 名称、durableName 与 filterSubject；不得由 durableName 或 Subject 字符串猜测角色。装载器必须校验 raw/derived consumer 的 stream 分别等于 dataRawStream/dataDerivedStream、role 已启用、consumerKey 与 durableName 均唯一，writer/alarm/compute 的 raw 与 computed Subject 覆盖完整，并将 consumerKey 用作 `processed_event` 的去重命名空间；backoff 非递减且长度不得超过 maxDeliver。

V1 的 EVENT stream 只承载 `alarm.event.v1`（alarm transition 与 DATA_GAP）输出，不是 Engine 入站 consumer。
因此 `jetStream.consumers` 只允许 `data.raw.>` 与 `data.computed.>`，Schema 和装载器均拒绝 event stream consumer。
DLQ body 使用 `runtime.dlq.event.v1`：含 deployment/account/consumer、原 subject、可空 eventId、稳定 reasonCode、
deliveryCount、原 body 的 SHA-256 与受 1MiB 解码上限约束的 base64、occurredAt；不得包含 Secret、endpoint 或 DSN。
`dlqId`/Outbox dedupe key 为 `SHA-256(schemaVersion, deploymentId, consumerKey, originalSubject, eventId 或空字段,
reasonCode, bodySha256)`，字段以 0x1f 拼接，忽略会随重试变化的 deliveryCount/occurredAt，以避免同一失败重复创建 Outbox。

## 8. PostgreSQL 事务、Outbox 与 checkpoint

V1 的 Engine 状态库只能是 PostgreSQL，且必须支持原子比较更新（CAS）。运行态物理表可由模块实现决定，但逻辑上
必须有 `processed_event`、当前/报警状态、`checkpoint` 与 `transactional_outbox` 四类记录。`processed_event` 的唯一键
是 `(consumerKey, eventId)`，不能只用 eventId，否则 writer/alarm/compute 会互相吞掉同一原始事件；它保存原始消息
bytes 的 SHA-256，拒绝相同 consumerKey/eventId 的不同 bytes。任何 owner 状态写入都
使用类似 `WHERE owner_id = :ownerId AND epoch = :epoch AND version = :expectedVersion` 的条件更新；影响行数不是一时
必须回滚并停止该 owner 的副作用。

处理一条消息的固定顺序是：

1. 读取 JetStream 消息，开始 PostgreSQL 事务，并校验当前消费者 role owner/epoch。
2. 以 `(consumerKey, eventId)` 插入 `processed_event`（唯一约束），并保存 JetStream 原始消息 body bytes SHA-256。若已存在且摘要相同，提交空事务并 Ack；若摘要不同，隔离并告警。
3. 在同一事务内应用业务状态、以 CAS 更新 checkpoint，并写入所有待发布的 `transactional_outbox` 记录。
4. 提交事务后才 Ack JetStream。事务失败、CAS 失败或 Outbox 写入失败均不得 Ack。
5. 独立 Outbox publisher 在提交后发布；收到 PubAck 后将该 Outbox 行标为已发布。重复发布仍由下游 eventId 幂等处理。

checkpoint 只表示已安全处理到的位置，不能绕过 `processed_event` 去重，也不能在业务状态和 Outbox 之前持久化。

事件中的 producer owner/epoch 只用于校验来源 Collector Assignment；每个 Engine role 在 `roleAssignments` 中的
consumer owner/epoch 只用于自己的 CAS fencing，两者是独立 token，禁止比较它们是否相等。Assignment
必须与 `roles` 一一对应，由配置装载器做数组键唯一和集合相等校验。`artifactMount` 固定 Artifact 从 release PVC 的
只读本地来源，运行进程不能从网络 URL 或可写工作目录读取该输入。

启动时可用受信 assignment 显式调用独立 CAS 激活 owner；`Store.Open` 不得自动写库或激活任何 producer/consumer token。
roleAssignment 与 producerAssignment 的 ownership.epoch 均为至少 1 的独立 fencing token；只允许在各自侧做 CAS，
不得以两者是否相等作为启动或事件验收条件。

除 Schema 可直接验证的字段外，配置装载器必须拒绝：roles 与 roleAssignments 的集合不相等、重复 role、重复
connectionId/datapointId、mapping 的 `(connectionId, variableId)` 不唯一、mapping 指向不存在连接、Subject 尾段不等于
pointId、非递减 backoff 不满足、backoff 长度超过 maxDeliver，以及 effectiveAcquisition 与默认/override 合并结果
不一致。

## 9. Collector WAL 与数据缺口

Collector 对每个 deployment 使用隔离 WAL。每条记录在发布前必须完成 fsync，带 CRC32C；只有取得 JetStream
PubAck 后才能写 `after-jetstream-puback` commit marker 并允许回收。重连后的补投按 WAL 顺序发送，保留整个原始
payload（包括不可变 `receivedAt`）、eventId、sourceTimestamp 与 owner/epoch/sequence；不得修改 payload 或增加重放标记。

Binding 固定 `maxBytes` 和 `highWatermarkBytes`。达到高水位必须降级并上报；达到容量上限时不得静默覆盖
未确认记录，必须暂停相关采集或按人工批准的恢复动作处理。若介质损坏、人工清理或容量策略导致事实不可恢复，Collector
必须生成 `kind=data-gap, operation=DATA_GAP` 的结构化 `alarm.event.v1` 事件（不需要也不得伪造 `alarmItemId`，含
Collector、连接、受影响时间/sequence 范围和原因），并在 status 中反映。

`fromSequence <= toSequence`、`0 < highWatermarkBytes < maxBytes` 必须由实现校验。WAL 必须为数据缺口的持久诊断/
控制记录预留独立空间；容量已满时不得依赖同一已满 WAL 再写入控制记录。

Binding 的 `diagnosticReserveBytes` 是必填正整数。raw admission 必须始终为它保留容量；data-gap 诊断记录可以使用该
预留，但单条 diagnostic frame 加其 ACK 标记不得超过此额度，收到 ACK 并完成 compaction 后才能恢复该额度。它不替代
每个 pending PubAck 所需的正常 WAL 预留。

## 10. 健康与业务新鲜度

`runtime-health-status.v1` 是既有 [Runtime 健康检查与状态协议](./runtime-health-status-contract.md) 中 HTTP
`data` 对象的机器可读扩展，**不是** `/health` 或 `/api/v1/status` 的 `code/msg/data/reqId` HTTP envelope。它保留
site/deployment、execution form、node/process、生命周期与健康语义，并增加业务新鲜度。`runtime-api`、
`runtime-engine`、`industrial-collector` 是 deployment 角色，必须有 deploymentId/accountId/businessFreshness；
共享或节点级 `compute-sandbox` 可以没有这些字段，也不应伪造业务新鲜度。Collector 必须上报 assignment、
Binding/Artifact revision、owner/epoch、WAL backlog/bytes/oldest age/watermark 与上游发布状态；非 Collector 角色不得
伪装这些字段。`healthState` 只回答组件能否提供其职责（进程、依赖、WAL、连接等）；`businessFreshness.state` 只回答最新业务
事件的证据是否新鲜。两者独立：运行健康但设备不更新可以是 `HEALTHY + STALE`；WAL 高水位可为 `DEGRADED + FRESH`。
不得由其中一个字段推导或覆盖另一个。中心可根据 `observedAt` 计算状态上报的新鲜度，但不能把它写回组件健康。
`projectId`、`projectCode`、`deploymentStage` 为可选的既有 status 扩展字段，不改变 `deploymentId` 在共享组件中可缺省的
语义。

## 11. 验收要求

实现方至少验证：合法/非法 fixtures、所有 Schema 的 draft 2020-12 编译、Subject/payload identity 对照、重复和
乱序事件、旧 epoch fencing、事务提交前失败、Outbox 重投、WAL PubAck 前崩溃与满盘数据缺口。`edge-single` 的
故障恢复必须有人工操作记录；不得将“进程重启成功”表述为自动 HA 恢复。
