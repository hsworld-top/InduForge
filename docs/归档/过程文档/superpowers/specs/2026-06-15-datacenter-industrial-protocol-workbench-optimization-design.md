# 数据中心工业协议工作台优化设计

> 日期：2026-06-15
> 范围：`datacenter/` 接入源工作台、数据点、存储策略、运行契约预览，以及 `data_service/` 对应读模型和发布契约。
> 对象：OPC UA、Modbus TCP、Siemens S7。OPC DA 不纳入本轮 Linux 目标环境优化。
> 核心目标：以数据点为统一资产，优化工业协议建模、发布前验证、设备冗余、采集冗余和历史归档策略。

## 1. 设计结论

工业协议工作台不做长期 SCADA 监控台。它负责协议变量建模、开发态读取验证、问题诊断、运行契约预览和设备冗余配置。

数据点是平台统一消费对象。页面绑定、报警、计算、运行态权限和历史归档都围绕数据点治理，不在 OPC UA、Modbus、S7 各自工作台内重复配置。

存储策略独立成菜单项。实时当前值默认写入 IF 实时库；历史归档由存储策略配置，目标复用已有接入源，包括 IF 时序库、TDengine 和用户自定义关系库，并为后续存储类型扩展保留 capability 机制。

冗余拆成两层。设备自身冗余在接入源连接配置中维护；平台采集冗余在运行部署或采集策略中统一维护，协议工作台只展示摘要和契约预览。

三类协议保持同一工作台骨架，但字段和流程按协议优化。OPC UA 以服务器地址空间浏览和导入为主；Modbus 以寄存器地址表、粘贴导入和地址段生成为主；S7 以 PLC 档案、Excel/CSV 地址表导入和地址解析预览为主。

## 2. 成熟 SCADA 对照

WinCC、KingSCADA、Ignition SCADA、Kepware/KingIOServer 等成熟系统的共同点不是把所有协议做成同一张表，而是围绕 Tag/点位资产建立连续链路：

- 连接配置清楚表达现场 endpoint、认证、安全和驱动参数。
- 点位来源可浏览、导入、批量维护和校验。
- 点位具备质量、时间戳、读写能力和诊断上下文。
- 归档、报警、计算和页面引用围绕统一点位对象治理。
- 采集性能、主备切换、运行诊断和历史写入可追踪。

本平台不应照搬传统 SCADA 的“变量表内配置所有能力”方式。InduForge 已经存在数据点、计算、报警、设计器和运行态数据引擎边界，应采用更清晰的分层：

```text
接入源
  连接入口、设备冗余、协议级 endpoint 能力

协议工作台
  协议变量建模、导入、校验、读取预览、读取计划/订阅策略预估

数据点
  统一资产身份、来源、状态、权限、引用和治理入口

存储策略
  历史归档规则、存储目标、写入模式、保留策略和容量预估

运行态
  长期采集、实时值写入、历史归档、报警、计算、owner/lease/failover
```

## 3. 数据点核心边界

协议变量保存后自动生成或更新数据点。用户不手动点击“同步数据点”。

协议工作台展示数据点关联，但不配置数据点消费能力：

```text
展示：
- 数据点 path
- 数据点同步状态
- 数据点状态 active / inactive / invalid / error
- 引用数量
- 存储策略数量摘要
- 跳转数据点详情
- 复制数据点 path

不提供：
- 存储配置
- 报警配置
- 计算配置
- 页面绑定配置
- 快速创建报警
- 快速创建计算
```

数据点 path 首次创建时由系统生成。生成规则使用 connectionCode、groupCode 和 variableCode，但 path 一旦生成，后续修改变量名、分组或 code 默认不自动改 path。

确需改 path 时，应在数据点模块执行重命名，并进行引用影响检查。

删除协议变量时，对应数据点标记为 `invalid`，不物理删除。停用变量时，对应数据点为 `inactive`，仍可被页面、报警或计算引用，但契约检查应提示 warning。

状态模型统一为：

```text
协议变量状态：
active / disabled / invalid / error

读取质量：
Good / Uncertain / Bad / Unknown

数据点状态：
active / inactive / invalid / error
```

## 4. 统一工作台骨架

OPC UA、Modbus、S7 使用一致的三栏骨架：

```text
左侧：
  接入源摘要、连接/会话状态、分组树

中间：
  工具栏、紧凑总览指标条、变量表、分页

右侧：
  Inspector 详情、数据点摘要、最近读取、策略摘要、诊断问题
```

中间总览指标条保持紧凑，不做大卡片。不同协议展示不同指标：

```text
OPC UA：
节点数、启用数、订阅项、采样摘要、问题数

Modbus：
变量数、从站数、读取次数、reads/s、问题数

S7：
变量数、读取块、总字节、reads/s、问题数
```

右侧 Inspector 固定区块顺序，协议内容各自表达：

```text
1. 当前对象摘要
2. 协议地址 / 节点 / PLC 地址信息
3. 采集策略
4. 发布读写能力
5. 数据点
6. 存储策略摘要
7. 最近开发态读取
8. 读取计划 / 订阅策略摘要
9. 诊断问题
```

变量表必须使用后端分页、搜索、筛选和排序。前端不得只筛当前页。

通用筛选：

```text
全部
异常
未同步或同步异常
可写
已停用
```

协议专属筛选放入更多筛选：

```text
OPC UA：
NodeId / BrowseName / AccessLevel / DataType

Modbus：
从站地址 / 寄存器区 / 地址范围 / 数据类型

S7：
地址区 / DB号 / PLC 档案状态 / 数据类型
```

变量表支持行右键菜单。操作列只保留编辑和更多。

行右键菜单：

```text
编辑
复制为新变量
复制变量信息
复制数据点路径
移动到分组
查看数据点详情
删除变量
```

多选右键或批量菜单：

```text
批量移动到分组
批量设置周期
批量设置发布读写能力
批量启用/停用
批量删除
```

## 5. 分组树

三类协议变量都必须具备分组树能力，便于按产线、设备、区域和功能域管理点位。分组树是协议工作台内的点位组织方式，不替代全局设备资产模型。

分组树能力：

```text
全部变量
未分组
多级分组
搜索分组
新建 / 编辑 / 删除分组
组内变量数
组内问题数徽标
```

不做拖拽。沿用其他接入源的右键或更多菜单：

```text
新建子分组
重命名
移动到其他分组
上移 / 下移
删除
```

删除分组时，组内变量移动到“未分组”。数据点不失效，已有数据点 path 不自动变化。

导入导出不单独处理空分组树。变量行包含完整分组路径，导入时分组路径不存在则自动创建。

## 6. OPC UA 工作台优化

OPC UA 主入口改为服务器地址空间浏览、搜索、勾选导入。手工新建变量保留为补充动作。

主流程：

```text
进入 OPC UA 工作台
-> 连接开发态会话
-> 浏览或搜索服务器节点
-> 勾选变量节点
-> 导入确认
-> 修改变量名、code、分组、采样周期、死区和发布能力
-> 保存后自动生成 opcua.node 数据点
-> 短时预览值和质量
-> 建模校验
-> 运行契约预览
```

OPC UA 变量表建议字段：

```text
变量名
NodeId
BrowseName
DisplayName
DataType
AccessLevel
采样周期
死区
数据点
最近值
质量
状态
引用数
```

OPC UA 连接与会话配置在工作台内完整编辑，分区展示：

```text
基础：
endpoint / port / namespace / 默认采样周期

安全：
SecurityPolicy / SecurityMode / 证书 / 私钥 / 信任服务器证书

认证：
匿名 / 用户名密码 / 证书认证

会话：
sessionTimeout / keepAlive / reconnect / publishInterval
```

主界面 Header 只显示 endpoint、安全模式摘要、认证方式摘要和连接状态。

证书信任与证书状态是一等诊断问题：

```text
服务器证书未信任
客户端证书缺失
证书即将过期
证书主题或 endpoint 不匹配
安全策略与服务器不匹配
```

服务器属性可以刷新同步，但不覆盖用户治理字段。

```text
可刷新：
NodeId
BrowseName
DisplayName
DataType
AccessLevel
Description
ValueRank / ArrayDimensions

不被刷新覆盖：
变量名
code
分组
采样周期
死区
发布能力
数据点 path
标签
用户备注
```

若 DataType 或 AccessLevel 变化影响数据点和运行契约，应产生诊断问题。

OPC UA 读写能力以服务器能力为准，只允许收窄，不允许放大：

```text
serverAccessLevel：
从服务器浏览或读取属性得到，不可手动改。

enabledAccessLevel：
平台发布能力，只能是 serverAccessLevel 的子集。
```

OPC UA 不提供导入模板。其主路径是浏览服务器节点。

## 7. Modbus 工作台优化

Modbus 以 TCP 为第一优先级。RTU 保留配置字段和建模边界，但不作为近期核心体验。当前目标环境为 Linux，RTU 开发态预览和串口验证后置。

Modbus 不需要独立设备档案。连接配置和变量级字段足够表达模型：

```text
连接级：
TCP / RTU、host、port、串口参数、默认从站、默认轮询周期、超时、重试

变量级：
从站地址、寄存器区、用户地址、地址基准、协议地址、数量、数据类型、字节序、字序、bit、倍率、偏移、读写权限
```

变量表建议字段：

```text
变量名
Code
从站地址
寄存器区
用户地址
协议地址
数据类型
字节序
周期
数据点
最近值
质量
状态
引用数
```

导入第一版支持粘贴表格和地址段生成。Excel/CSV 不是 Modbus 第一优先级。

粘贴表格字段：

```text
变量名
code
从站地址
寄存器区
用户地址
地址基准
类型
字节序
倍率
单位
周期
```

地址段生成字段：

```text
从站地址
寄存器区
起始地址
变量数量
类型
命名前缀
```

导入确认表必须展示：

```text
状态
变量名
Code
从站
寄存器区
用户地址
协议地址
类型
寄存器数量
问题
```

Modbus 写能力按寄存器区自动约束：

```text
Coil：
可读可写，允许 bool 写入。

Discrete Input：
只读，不允许写。

Input Register：
只读，不允许写。

Holding Register：
可读可写，允许按数据类型写入。
```

用户可以在可写区域选择只读发布或读写发布。不可写区域不能被配置成读写发布。

Modbus 读取计划由系统估算，不允许手工编辑。用户通过地址、周期、从站和少量高级参数影响计划。

## 8. S7 工作台优化

S7 接入源创建保持轻量，只填写名称、PLC 地址、端口和默认轮询周期。PLC 系列、通信方式、PDU、地址能力和优化 DB 访问风险在工作台内的“S7 连接与 PLC 档案”弹窗中配置。

S7 连接与 PLC 档案弹窗分区：

```text
基础连接：
host / port / 通信方式 / rack / slot / TSAP

PLC 档案：
PLC 系列 / 地址能力 / 优化 DB 访问提示

读取能力：
PDU / maxReadBytes / maxGapBytes / 默认周期 / 超时

高级参数：
特殊网关、适配器、私有 options
```

未确认 PLC 档案前，允许创建分组、导入变量草稿和手工录入变量，但限制读取当前值、短时预览、读取计划估算和严格地址能力校验。

变量表建议字段：

```text
变量名
Code
地址
地址区
DB号
字节范围
数据类型
周期
数据点
最近值
质量
状态
引用数
```

S7 导入必须支持 Excel/CSV、字段映射和地址解析预览。它是一等能力。

导入流程：

```text
选择 Excel/CSV
-> 字段映射
-> 地址解析预览
-> 导入确认
-> 允许修正名称、code、分组、地址、类型
-> 仅导入有效项或返回修正
```

字段映射：

```text
变量名
Code
分组
地址
数据类型
单位
倍率
偏移
采集周期
发布能力
描述
```

地址解析预览展示：

```text
原始地址
标准地址
区域
DB号
字节范围
bit 位
问题
```

S7 提供导入模板。模板不包含数据点 path、存储策略、报警规则、计算配置或页面绑定。

S7 写能力默认全部只读，只有用户逐点显式开启后才进入写入契约：

```text
I 区：
只读，不允许配置可写。

Q 区：
可配置写，但必须显式开启。

M 区：
可配置写，但必须显式开启。

DB 区：
可配置写，但必须显式开启，并提示 S7-1200/1500 优化 DB 访问风险。
```

## 9. 导入、导出与复制

导入确认表允许编辑导入草稿，不直接写正式表。保存前必须做完整校验。导入成功后才生成协议变量和数据点。

导入确认表支持：

```text
单元格内联编辑
错误行定位
只显示错误
只导入有效项
下载或复制错误报告
批量设置分组
批量设置周期
批量设置发布能力
```

三类协议均支持变量导出。导出当前筛选结果，不导出敏感连接信息。

导出格式：

```text
CSV：
UTF-8 BOM，适合快速交换和排错。

XLSX：
用于现场交付、审阅和二次维护。
```

XLSX 包含两个 sheet：

```text
Sheet 1：变量清单
Sheet 2：问题清单
```

变量清单字段：

```text
变量名
Code
分组
协议地址 / NodeId
数据类型
采集周期
发布能力
数据点 path
状态
最近开发态读取值
质量
诊断问题摘要
```

问题清单字段：

```text
严重级别
问题类型
变量名
Code
分组
定位字段
问题说明
建议处理
```

变量复制支持单条“复制为新变量”。复制草稿阶段不生成数据点。保存成功后生成新数据点。第一版不做批量克隆和自动递增复制。

## 10. 读写能力两层模型

协议工作台不提供实际写操作：

```text
不提供：
- 写当前值
- 批量写入
- 强制置位
- 手动控制
```

协议工作台提供协议侧读写能力配置。数据点模块提供运行态权限治理。

```text
协议变量层：
这个现场点从协议上能不能写。

数据点层：
平台里谁能写、哪些角色能写、哪些运行场景能写。
```

发布契约中应同时携带协议侧能力和数据点运行态权限。运行态写入必须同时满足两层条件。

## 11. 开发态验证边界

三类协议统一使用显式开发态会话：

```text
连接
读取一次
短时预览
断开
```

开发态预览不写正式历史，不触发正式报警、计算或工作流。

当前 Linux 目标环境下，S7、Modbus TCP、OPC UA 的开发态预览默认由平台侧 `data_service` 执行。节点侧验证作为后续增强。

界面应明确展示：

```text
开发态验证位置：平台侧 data_service
运行态采集位置：发布后的采集节点 / 数据引擎
```

Modbus RTU 因 Linux 串口设备、权限和部署位置复杂，第一版不与 TCP 同等展示。RTU 预览不可用时提示需要在采集节点验证。

## 12. 诊断问题中心

三类协议统一诊断对象，覆盖连接、建模、数据点、预览、性能和契约风险。

问题类型：

```text
connection
model
datapoint
preview
performance
contract
storage
redundancy
```

严重级别：

```text
error
warning
info
```

定位目标：

```text
connection
group
variable
datapoint
readPlan
endpoint
storagePolicy
```

UI 展示位置：

```text
中间工具栏：
问题计数

右侧 Inspector：
当前对象相关问题

诊断抽屉：
全部问题列表，可定位
```

诊断不只检查字段必填，还必须覆盖发布可行性、读取压力、数据点同步、设备冗余、采集冗余和存储策略异常。

## 13. 设备自身冗余

设备自身冗余描述同一个逻辑接入源背后的多个设备路径或服务端。

示例：

```text
OPC UA：
主 endpoint / 备 endpoint

S7：
主 PLC IP / 备 PLC IP，或主备通信路径

Modbus：
主网关 / 备网关，或主备 TCP endpoint
```

设备冗余放在接入源连接配置层，不放到每个变量或数据点层。变量仍保存逻辑地址，数据点仍引用逻辑 connectionId 和协议变量 ID。

设备冗余第一版支持主备优先级切换，不做双活热备。备用 endpoint 可做轻量健康检查，但不并行采集全量数据。

建议连接配置字段：

```text
redundancy.enabled
redundancy.mode = none / priority_failover
redundancy.endpoints[]
  id
  name
  role = primary / standby
  host / port / endpoint
  priority
  enabled
  healthCheck
failoverPolicy
  timeoutMs
  cooldownMs
  autoFailback
  stableDurationMs
```

`autoFailback` 默认关闭。启用自动回切时必须配置稳定时间和冷却时间。

健康检查第一版以连接级为主，可配置少量探测点：

```text
连接级：
OPC UA 建立会话 / 读取 ServerStatus
S7 建立连接 / 轻量读取或握手
Modbus TCP 端口可达 + 可选读探测寄存器

探测点：
用户指定 1-3 个关键变量作为健康探测
```

不做点级或 readPlan 级 endpoint 切换。

设备冗余不影响数据点 path 或 sourceId。运行态事件额外携带：

```text
activeEndpointId
activeEndpointName
failoverState
ownerInstanceId
leaseId
epoch
```

设备冗余配置在协议工作台；设备冗余手动切换、锁定 endpoint 和切换历史属于运行态运维界面。

## 14. 平台采集冗余

平台采集冗余描述同一份运行契约由多个采集实例承接时，谁是当前 Owner。

第一版支持：

```text
none：
单实例采集，无冗余。

standby_failover：
多个采集实例可运行，但同一 connection/readPlan 只有一个 Owner。
Owner 失效后，standby 通过 lease 接管。

sharded：
多个采集实例按 connection 或 readPlan 分片，每个 shard 仍只有一个 Owner。
```

不做双采集仲裁。一个逻辑点同一时刻只有一条权威采集链路。

ownership 粒度默认按 connection。高吞吐场景可高级配置为 readPlan 分片，但必须提示同一设备可能被多个采集实例并发访问。

采集冗余配置不在协议工作台编辑。它属于运行部署或采集策略模块，受节点、授权、部署 profile 和 `engine_coord` 约束。协议工作台展示摘要和契约预览：

```text
模式
ownership 粒度
期望副本数
lease TTL
接管超时
是否允许 readPlan 分片
授权是否允许
```

运行态事件必须带：

```text
ownerInstanceId
leaseId
epoch
endpointId
readPlanId
```

Owner 切换时保留 checkpoint 和 lastSuccess 信息。旧 Owner 的过期数据不得污染写入。

```text
lastSampleAt
lastSuccessAt
lastSequence
ownerEpoch
leaseId
readPlanId
endpointId
```

冗余切换期间支持 `failoverGraceMs`。默认切换期间质量为 `Uncertain`，优先抑制质量类误报，不默认吞业务值。

## 15. 存储策略独立菜单

存储策略独立为数据中心菜单项，不放在协议工作台，也不塞进数据点详情做重表单。

建议菜单：

```text
数据点
接入源
存储策略
计算
报警
```

边界：

```text
数据点：
作为被存储对象，提供 path、类型、质量、来源、状态和引用。

存储策略：
负责把一批数据点写到哪个存储接入源、怎么写、保留多久。

接入源：
作为存储目标，复用已有连接。
```

协议工作台只展示存储摘要：

```text
存储：已命中 1 条历史策略 / 未配置历史归档
```

数据点详情展示命中策略摘要，并跳转存储策略菜单。

存储策略必须面向统一数据点，不只服务 OPC UA、Modbus、S7。

## 16. 存储目标能力声明

存储目标通过 capability 声明是否能作为存储目标，避免基于类型硬编码扩散。

能力建议：

```text
realtimeCurrentValue
timeseriesAppend
relationalAppend
relationalUpsert
messageArchive
```

第一版映射：

```text
IF时序库：
timeseriesAppend

TDengine：
timeseriesAppend

用户自定义关系库：
relationalAppend

IF关系库：
relationalAppend

IF实时库：
realtimeCurrentValue

外部 Redis：
realtimeCurrentValue
```

历史归档策略只展示具备 `timeseriesAppend` 或 `relationalAppend` 的接入源。当前值策略默认使用 IF 实时库，不作为普通用户主要配置项。

## 17. 当前值与历史归档

实时当前值默认写入 IF 实时库，也就是平台产品层的内部实时能力。对用户不暴露底层 Redis。

当前值是运行态基础能力：

```text
默认启用
写入 IF实时库
用于页面实时读取、订阅、运行态状态
可在运行部署或高级设置调整 TTL、质量保留和资源开关
```

历史归档是用户显式配置：

```text
独立菜单配置
目标复用具备历史写入能力的接入源
支持批量按筛选配置
```

历史归档第一版优先支持：

```text
1. IF时序库
2. TDengine
3. 关系库追加表
```

策略绑定内部使用 `dataPointId`，展示、导出和写出保留 `dataPointPath`。

历史写入统一逻辑模型：

```text
timestamp
dataPointId
dataPointPath
value
quality
tags
sourceType
sourceId
ownerEpoch
ownerInstanceId
leaseId
activeEndpointId
readPlanId
sequence
```

物理结构按目标优化：

```text
IF时序库：
统一时序表，按 dataPointId/path 分区或索引。

TDengine：
stable + tag，每个数据点子表，或按策略建 stable。

关系库：
窄表追加。
```

关系库第一版默认窄表，不做宽表映射。

窄表字段：

```text
timestamp
datapoint_id
datapoint_path
value
quality
source_type
source_id
epoch
sequence
```

目标表或 stable 默认自动创建，允许高级用户选择已有表并配置字段映射。

## 18. 历史归档策略

历史策略支持批量配置。主流程：

```text
选择存储目标
选择数据点范围
配置写入模式和保留策略
预览影响点数和写入量
验证目标表或试写
保存策略
```

数据点范围支持：

```text
按标签
按来源类型
按接入源
按 path 前缀
手动选择
```

按来源类型展开 source-specific 条件：

```text
OPC UA：
接入源、变量组、AccessLevel、DataType

Modbus：
接入源、寄存器组、从站、寄存器区、数据类型

S7：
接入源、变量组、地址区、DB号、数据类型

MQTT：
接入源、订阅、Topic、Tag 分组

Kafka：
Topic、字段映射

HTTP / WebSocket：
请求或消息分组

SQL 查询：
数据库连接、查询名称

计算输出：
计算单元、输出名

报警状态：
报警规则或策略
```

绑定模式：

```text
静态绑定：
明确选择一批 dataPointId。

动态规则：
保存筛选条件，新数据点满足规则时自动纳入。
```

协议分组条件内部优先使用 groupId，不依赖分组名称。界面展示分组路径。分组删除后，策略诊断提示当前规则引用的分组已删除。

写入模式：

```text
every_sample：
每次采样写历史。

on_change：
值变化超过阈值或死区时写。

periodic_snapshot：
按固定周期写最新值。
```

策略字段：

```text
writeMode
minIntervalMs
deadband
snapshotIntervalMs
includeQualities
retentionDays
targetConnectionId
targetTableMode = auto_create / existing_mapping
status = enabled / disabled / error
```

质量入库默认：

```text
Good、Uncertain 写历史值。
Bad 不写 value 历史，但记录质量事件。
```

策略可显式选择包含 Bad。

保留策略必须存在。不同目标能力不同：

```text
IF时序库：
支持策略级保留天数。

TDengine：
映射到 database/stable/table 层保留能力。

关系库：
第一版记录 retentionDays；如启用自动清理，需要清理任务和索引要求。
```

删除存储策略默认只删除策略定义，不删除目标库里的历史数据或目标表。

停用存储策略不删除历史数据、目标表或数据点，只是不再新写历史。

## 19. 存储验证、去重与预估

历史归档策略必须支持发布前验证目标表或试写。

开发态试写：

```text
使用模拟数据或最近开发态读取值
写入目标接入源的开发态连接
验证字段映射、权限、表结构和基础写入能力
```

对外部库可写入带测试标记的数据，或使用事务回滚、临时表验证，按目标能力选择。

历史写入由 writer 层做幂等与 epoch 校验。

去重键建议：

```text
dataPointId + timestamp + sourceId + sequence
```

或：

```text
dataPointId + sourceEventId
```

校验规则：

```text
ownerEpoch 必须是当前有效 epoch。
过期 owner 事件拒绝写入或标记 stale。
同一去重键重复到达时忽略或覆盖，按目标能力决定。
```

存储策略必须提供基础写入频率和容量预估：

```text
命中数据点数
预计 events/s
预计 rows/day
按 retentionDays 的预计行数
目标连接写入能力是否未知或超阈值
```

`on_change` 第一版可按采集周期乘以保守变化率估算，或显示依赖变化率、无法精确估算。

## 20. 运行契约预览

协议工作台提供运行契约预览，但不执行发布。

契约预览必须包含：

```text
协议建模：
连接配置、变量/节点/寄存器、数据点映射、采集策略

读取计划或订阅策略：
readPlans、subscription summary、reads/s、风险诊断

设备冗余：
模式、endpoints、健康检查、探测点、回切策略

采集冗余：
模式、ownership、期望副本数、lease TTL、接管超时、授权状态

当前值：
IF实时库启用状态、TTL、质量保留策略

历史归档：
策略数量、命中点数、目标接入源、写入模式、保留策略、质量规则、字段映射、写入预估

诊断问题：
连接、建模、数据点、冗余、存储、契约风险
```

如果设备冗余和采集冗余叠加，预览必须说明最终语义：

```text
同一 connection 同一时刻只有一个采集 Owner；
该 Owner 在 endpoint 故障时按主备策略切换设备路径。
```

存储策略也必须进入运行契约检查。动态规则命中 0 个点、目标接入源不可用、字段映射不完整、写入量超阈值都应产生诊断问题。

## 21. 接入源列表健康摘要

接入源列表不做长期在线监控，但要显示建模进度和健康摘要。

OPC UA / Modbus / S7 接入源显示：

```text
变量数
启用数
数据点数
问题数
最近校验时间
运行契约是否可发布
设备冗余摘要
采集冗余摘要
历史归档摘要
预计写入量
```

设备冗余摘要：

```text
无
主备已配置
当前备用承载
主路径异常
```

采集冗余摘要：

```text
无
主备接管
分片
授权不足
```

历史归档摘要：

```text
未配置
已配置 N 条
策略异常
预计 X rows/day
```

启用点没有历史策略不一定是错误，可提示 info：

```text
未配置历史归档，仅保留实时当前值。
```

## 22. 非目标

本轮不做：

```text
长期 SCADA 监控台
协议工作台内配置存储、报警、计算或页面绑定
协议工作台内快速创建报警或计算
协议工作台直接写设备值或批量控制
OPC DA 工作台
Modbus RTU 与 TCP 同等完整开发态预览
设备双活热备和双路数据仲裁
平台双采集实例同时采同一点再仲裁
手工编辑 readPlan
点级设备冗余映射
协议工作台执行运行态 endpoint 手动切换或锁定
关系库历史宽表映射第一版
删除存储策略时删除历史数据或目标表
```

## 23. 分阶段建议

阶段 1：统一骨架和诊断基础

```text
变量表后端搜索/分页/筛选
分组树右键操作
行右键菜单
Inspector 区块顺序
诊断对象模型
接入源健康摘要基础字段
```

阶段 2：协议差异化补强

```text
OPC UA 浏览/搜索/勾选导入
OPC UA 证书与服务器属性刷新
Modbus 粘贴导入和地址段生成
S7 连接与 PLC 档案合并弹窗
S7 Excel/CSV 导入和地址解析预览
```

阶段 3：运行契约预览

```text
readPlan / subscription summary
数据点 path 稳定策略检查
发布读写能力检查
运行契约预览入口
契约诊断整合
```

阶段 4：存储策略菜单

```text
存储 capability 声明
历史归档策略列表和详情
静态绑定和动态规则
IF时序库、TDengine、关系库追加表目标
试写验证
写入频率和保留预估
```

阶段 5：设备冗余和采集冗余可视化

```text
设备主备 endpoint 配置
连接级健康检查和探测点
契约预览展示设备冗余
采集冗余摘要接入运行部署策略
冗余事件和历史写入上下文字段
```

阶段 6：导出和现场交付

```text
CSV 导出
XLSX 变量清单和问题清单
S7 导入模板
错误报告导出
```

每阶段完成后都应至少验证：

```text
pnpm --filter datacenter typecheck
pnpm --filter datacenter build
go test ./...
```

如果只改文档，不需要执行代码级验证。
