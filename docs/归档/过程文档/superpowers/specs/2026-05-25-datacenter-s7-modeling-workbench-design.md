# Siemens S7 接入源与建模工作台完整设计

> 范围：数据中心接入源中的 Siemens S7 接入、S7 工作台、变量建模、开发态连接验证、数据点自动同步、运行态采集契约。
> 原则：本文描述目标产品设计，不考虑既有实现兼容，不拆分阶段，不把临时交付形态写入产品设计。

## 1. 设计结论

S7 接入源创建保持轻量，只回答“连接到哪个 Siemens S7 PLC”。PLC 系列、通信细节、地址能力、变量建模、读取校验、读取计划、开发态预览都放在 S7 工作台内完成。

协议变量保存后自动同步为平台数据点。S7 工作台不提供手动“同步数据点”主流程，也不在 S7 变量内部配置存储。存储、报警、计算、工作流引用统一围绕数据点配置。

最终分层如下：

```text
S7 接入源
  负责连接入口：名称、PLC 地址、端口、默认轮询周期

S7 工作台
  负责协议建模：PLC 档案、分组、变量地址、读取、预览、校验、读取计划

数据点
  负责平台统一对象：存储、报警、计算、工作流、页面绑定、运行态消费
```

最优方案不是复刻 Modbus，而是把 S7 的 PLC 特性前置成一等模型：

```text
PLC 档案是连接和能力根
地址模型是 S7 专属结构化模型
读取计划是运行态性能契约
变量保存自动生成数据点
平台二开人员围绕 PLC 类型、地址表、读取压力和数据点映射完成项目建模
```

Modbus 的经验只复用两类内容：

```text
工作台交互范式：左侧分组、中间变量表、右侧详情、弹窗闭环
工程边界范式：建模表、数据点同步、校验、预览、读取计划、artifact 输出
```

S7 的领域模型、运行契约和性能优化规则必须保持独立。

## 2. 产品定位

S7 工作台是“西门子 PLC 地址表建模、开发态验证、数据点生成、运行态采集契约生成工具”。

它不是 OPC UA 式节点浏览器。S7 的核心是地址表建模，用户围绕 DB/M/I/Q 等地址区维护变量，再通过短时连接验证这些变量是否可读、是否能形成合理读取计划。

对用户来说，S7 工作台要解决的是：

```text
我连的是哪类 PLC？
这些地址是否合法、能不能读？
这些变量会不会把 PLC 读爆？
这些变量保存后平台怎么统一使用？
发布后运行态按什么计划读取？
```

这里的“二开人员”指平台的一级用户：项目实施人员、应用搭建人员、系统集成商和现场工程师。他们不是来改驱动代码的，而是在平台里把 PLC 地址表建成可发布、可运行、可治理的数据模型。

对平台二开人员来说，S7 工作台要提供稳定边界：

```text
先选清楚 PLC 类型和通信方式
再维护变量组和变量地址表
系统负责解析地址、校验风险、生成数据点
系统展示读取计划和读取压力
发布时运行态直接消费已确认的 S7 契约
```

对运行态来说，S7 artifact 必须避免运行时再猜测：

```text
不从 addressText 临时解析
不从数据点 path 反推变量
不在运行态重新决定 PLC 能力
不让多个节点同时轮询同一个连接计划
```

## 3. 用户操作流程

```text
新建接入源
  -> 选择 Siemens S7
  -> 填写名称、PLC 地址、端口、默认轮询周期
  -> 创建后进入 S7 工作台
  -> 配置或确认 PLC 档案
  -> 连接 PLC
  -> 新建或导入变量
  -> 自动生成/更新数据点
  -> 建模校验
  -> 读取当前值
  -> 短时变量预览
  -> 查看读取计划
  -> 在数据点模块配置存储、报警、计算、工作流引用
  -> 发布运行态采集契约
```

## 4. 接入源创建

### 4.1 设计原则

接入源创建表单不能过重。现场用户通常先知道“这是一个西门子 PLC、IP 是多少”，不一定一开始就明确 S7-200、S7-1200、TSAP、PDU、优化块访问等细节。

因此，接入源创建只要求最小连接入口信息：

```text
名称
PLC 地址
端口
默认轮询周期
高级参数
```

PLC 系列不是创建接入源的必填前置条件。它在 S7 工作台的 PLC 档案中配置，可后续补充和调整。

用户第一次进入 S7 工作台时，如果尚未确认 PLC 类型，左侧基础信息中显示“类型未确认”，中间变量表仍可为空态展示，但涉及地址校验、读取计划和开发态连接的动作需要先完成 PLC 类型配置。

### 4.2 原型

```text
┌──────────────────────── 新建接入源 ────────────────────────┐
│ 类型                                                        │
│ [ MySQL ] [ MQTT ] [ OPC UA ] [ Siemens S7 ] [ Modbus ]     │
│                                                             │
│ 基础信息                                                    │
│ 名称        [ 熔炼炉 PLC                                  ] │
│ PLC 地址     [ 192.168.1.10                              ] │
│ 端口        [ 102        ]                                  │
│ 默认轮询周期 [ 1000       ] ms                              │
│                                                             │
│ 高级参数  ▸                                                 │
│                                                             │
│                                  [取消] [创建并进入工作台]  │
└─────────────────────────────────────────────────────────────┘
```

### 4.3 首次进入的 PLC 类型确认

S7 类型区分不放在外部接入源创建的大表单里，而是在进入 S7 工作台后的 PLC 档案弹窗中完成。这样既保持创建入口轻量，也让 S7 专属能力在工作台内可见、可修改。

触发时机：

```text
首次进入未配置 PLC 档案的 S7 工作台
用户点击左侧基础信息中的“配置 PLC 类型”
用户在连接失败后点击“调整 PLC 档案”
```

弹窗原型：

```text
┌──────────────────────── 配置 PLC 类型 ────────────────────────┐
│ 01 PLC 类型                                                     │
│ PLC 系列 *      [ S7-1200                              v ]      │
│ 通信方式 *      [ Rack / Slot                          v ]      │
│                                                               │
│ 02 连接参数                                                    │
│ PLC 地址        [ 192.168.1.10                              ]  │
│ 端口            [ 102                                      ]  │
│ Rack            [ 0                                        ]  │
│ Slot            [ 1                                        ]  │
│                                                               │
│ 03 读取能力                                                    │
│ PDU 大小         [ 自动协商                                v ]  │
│ 单次最大读取      [ 自动                                    ]  │
│ 合并间隙          [ 16 ] bytes                                │
│ 默认轮询周期      [ 1000 ] ms                                  │
│                                                               │
│ 04 地址能力                                                    │
│ [✓] DB 区  [✓] M 区  [✓] I 区  [✓] Q 区                       │
│ [✓] 绝对地址                                                   │
│ [ ] 符号地址                                                   │
│ [!] S7-1200/1500 如开启优化 DB 访问，绝对 DB 地址可能不可读。    │
│                                                               │
│                                [取消] [保存并刷新读取预估]      │
└───────────────────────────────────────────────────────────────┘
```

当 PLC 系列切换时，系统自动带出默认值：

```text
S7-200 / S7-200 SMART
  优先提示 TSAP / Gateway / Adapter 参数，不默认套 S7-1200 的 Rack/Slot 心智。

S7-300 / S7-400
  默认 Rack=0，Slot=2，合并间隙 8 bytes。

S7-1200 / S7-1500
  默认 Rack=0，Slot=1，合并间隙 16 bytes，并提示优化 DB 访问风险。

S7 Compatible
  保守能力集，默认只启用 DB/M/I/Q 绝对地址和较小读取块。
```

## 5. 当前代码现实与落表边界

当前后端已经有：

```text
data_connections(type='s7')
data_s7_configs(connection_id, host, port, rack, slot, poll_interval_ms, options)
data_points
项目快照 artifact 中已有 protocols.s7 空壳位置
```

这些只能表达“有一个 S7 接入源，能连接到哪里”，还不能表达“这个 PLC 上有哪些变量，运行态该怎么高效读取”。因此 S7 工作台需要新增独立建模表，而不是继续把变量塞进 `data_s7_configs.options`。

推荐新增：

```text
data_s7_plc_profiles
  保存 PLC 档案。它是连接配置的增强档案，面向 PLC 类型、通信方式、地址能力和读取限制。

data_s7_variable_groups
  保存变量组。用于用户组织地址表，不承担全局设备资产职责。

data_s7_variables
  保存 S7 变量。每行是一条用户关心的业务变量，结构化保存地址、类型、采集和解释规则。
```

`data_s7_configs` 继续保留轻量连接入口：

```text
host
port
rack
slot
poll_interval_ms
options
```

PLC 档案不建议只放在 `data_s7_configs.options`，原因是：

```text
PLC 系列、通信方式、PDU、最大读取字节数会影响校验和读取计划，需要可查询、可校验、可进入 artifact。
平台二开人员需要在界面上看懂并确认这些字段，不能在 options 里约定一堆不可见的隐式 key。
运行态需要稳定契约，不应依赖散落 JSON 的非结构化约定。
```

`options` 只保留厂商特有连接参数，例如特殊 TSAP、网关参数、连接保活、私有诊断开关。

## 6. S7 工作台总览

S7 工作台与当前 Modbus 工作台保持家族化结构：只有左、中、右三栏，不增加横向全局 Header。接入源基础信息放在左侧 `WorkbenchSourceHeader`，中间是当前变量组的建模表格，右侧是 Inspector 配置面板。

```text
┌──────────────────────┬──────────────────────────────────────────────┬──────────────────┐
│ 左侧：接入源与分组     │ 中间：变量建模                               │ 右侧：配置面板     │
├──────────────────────┼──────────────────────────────────────────────┼──────────────────┤
│ ← 熔炼炉 PLC           │ 当前组：熔炼炉                                │ 熔炼炉             │
│ Siemens S7             │ 48 个变量 · 自动同步 s7.variable 数据点        │ furnace_area       │
│ 端点 192.168.1.10:102  │ 7 读取块 · 7.00 reads/s · 0 个错误             │                  │
│ 类型 S7-1200           │                                              │ 变量组信息         │
│ Rack 0 / Slot 1        │ [导入] [新建] [读取] [预览] [校验] [读取计划]  │ 变量数：48         │
│ [已连接/断开]          │ [刷新]  [搜索变量                             ]│ 读取块：7          │
│                       │                                              │ reads/s：7.00      │
│ 变量组                 │ ┌──────────────────────────────────────────┐ │                  │
│ [搜索变量组] [+]       │ │变量名 | 地址 | 类型 | 字节范围 | 数据点    │ │ 地址分布         │
│ 全部变量        128    │ │最近值 | 质量 | 周期 | 状态 | 操作        │ │ DB：41           │
│ 熔炼炉           48    │ │炉温 | DB1.DBD4 | Real | DB1 4-7 | ...   │ │ M：5             │
│ 除尘系统         31    │ │723.5 ℃ | Good | 1000ms | active | ...   │ │ I/Q：2           │
│ 输送线           22    │ └──────────────────────────────────────────┘ │                  │
│ 报警             27    │                                              │ 校验问题：无       │
└──────────────────────┴──────────────────────────────────────────────┴──────────────────┘
```

中间顶部工具栏动作：

```text
导入变量
新建变量
读取当前值
变量预览
建模校验
读取计划
刷新
```

不提供“同步数据点”按钮。数据点同步是变量保存流程的一部分。

中间变量表建议列：

```text
变量名 | Code | 地址 | 地址区 | DB 号 | 类型 | 字节范围 | 周期 | 数据点 | 最近值 | 质量 | 状态 | 操作
```

其中：

```text
地址
  展示用户熟悉的 DB1.DBD4、M0.0、I0.0 等文本。

字节范围
  展示结构化读取范围，例如 DB1 4-7 bytes。它服务性能排查，不要求普通用户填写。

最近值 / 质量
  来自读取当前值或短时预览。它是开发态辅助结果，不代表运行态长期采集已经启动。

读取块 / reads/s
  来自读取计划估算，帮助用户看到地址分布对 PLC 压力的影响。
```

右侧详情不是“大表单”，而是上下文 Inspector：

```text
选中分组：展示变量数、数据点数、读取块数、地址区分布、主要风险。
选中变量：展示地址解析、数据解释、数据点、最近读取、相关校验问题。
未选变量时：默认展示当前分组或全部变量概况，不在右侧重复放接入源 Header。
```

## 7. PLC 档案

### 7.1 设计目的

PLC 档案用于描述当前接入源背后的 PLC 类型和连接细节。它不阻塞接入源创建，但会影响地址校验、默认连接参数、读取计划估算和开发态会话连接。

### 7.2 字段

```text
id
projectId
connectionId
plcFamily              PLC 系列
communicationMode      通信方式
host                   PLC 地址
port                   端口
rack                   Rack
slot                   Slot
localTsap              本地 TSAP
remoteTsap             远端 TSAP
pollIntervalMs         默认轮询周期
connectTimeoutMs       连接超时
readTimeoutMs          读取超时
pduSize                PDU 大小
maxReadBytes           单次最大读取字节数
maxConcurrentReads     最大并发读取数
byteOrder              字节序
wordOrder              字序
optimizedBlockAccess   优化 DB 访问提示
allowAbsoluteAddress   是否允许绝对地址
allowSymbolAddress     是否允许符号地址
supportedAreas         支持地址区
options                扩展参数
```

PLC 档案字段分两层：

```text
连接层
  host / port / rack / slot / tsap / timeout

能力层
  plcFamily / supportedAreas / supportedAddressModes / pduSize / maxReadBytes / optimizedBlockAccess
```

连接层用于建立开发态或运行态会话；能力层用于建模校验、读取计划和平台二开。平台二开人员在界面中选择 PLC 系列后，系统应带出默认能力、默认通信方式和风险提示，而不是要求用户理解隐藏配置。

### 7.3 PLC 系列

```text
S7-200
S7-200 SMART
S7-300
S7-400
S7-1200
S7-1500
S7 Compatible
```

PLC 系列影响默认值和能力提示，但不应该只是一个文案字段。它至少影响：

```text
默认 Rack/Slot
通信方式可选项
地址区能力
数据类型能力
优化 DB 访问提示
读取计划限制
诊断提示
```

### 7.4 通信方式

```text
Rack / Slot
TSAP
Gateway
Adapter
```

不同 PLC 系列可显示不同通信方式选项。S7-1200/1500 常用 Rack/Slot；S7-200 和 S7-200 SMART 需要允许 TSAP、Gateway 或 Adapter 类参数，避免把它们强行等同于 S7-1200。

### 7.5 原型

```text
┌──────────────────── PLC 档案 ────────────────────┐
│ PLC 系列        [ S7-1200                 v ]     │
│ 通信方式        [ Rack / Slot             v ]     │
│ Host            [ 192.168.1.10                  ] │
│ Port            [ 102                           ] │
│ Rack            [ 0                             ] │
│ Slot            [ 1                             ] │
│                                                     │
│ 读取参数                                            │
│ 默认轮询周期    [ 1000 ] ms                         │
│ 连接超时        [ 3000 ] ms                         │
│ 读取超时        [ 3000 ] ms                         │
│ PDU 大小        [ 自动  v ]                         │
│                                                     │
│ 地址能力                                            │
│ [✓] DB 区  [✓] M 区  [✓] I 区  [✓] Q 区             │
│ [✓] 绝对地址  [ ] 符号地址                          │
│                                                     │
│                                  [取消] [保存档案]  │
└─────────────────────────────────────────────────────┘
```

### 7.6 平台二开使用边界

PLC 档案需要让平台二开人员明确“哪些配置影响什么”：

```text
PLC 系列
  影响默认 rack/slot、通信方式选项、支持地址区、默认 PDU、最大读取字节数、诊断提示

通信方式
  影响连接参数展示。Rack/Slot、TSAP、Gateway/Adapter 不应混在同一组输入里。

地址能力
  影响变量弹窗、导入校验、读取计划和运行态契约。

读取能力
  影响 maxReadBytes、maxGapBytes、reads/s 和读取块拆分。
```

用户在工作台内看到的是“PLC 类型、地址能力、读取能力、风险提示”，不是底层实现接口。

## 8. 分组设计

S7 接入源通常对应一个 PLC。左侧分组用于表达 PLC 下的设备单元、工段、产线、功能域或报警域。

```text
全部变量
熔炼炉
除尘系统
输送线
报警
能耗
```

分组不是“设备模块”的替代品，但在 S7 工作台内足够承担变量组织。未来如需要全局设备资产模型，可以由设备资产引用数据点，而不是让 S7 工作台提前承担全局设备管理。

## 9. S7 变量模型

### 9.1 变量职责

S7 变量描述“从 PLC 的哪个地址读取一个什么类型的值，以及如何解释这个值”。它是协议建模对象，不是平台消费对象。平台消费对象是自动同步生成的数据点。

### 9.2 字段

```text
id
projectId
connectionId
groupId
name
code
description
status

area
dbNumber
byteOffset
bitOffset
addressText
normalizedAddress
addressType

dataType
length
arrayLength
byteOrder
wordOrder
scale
offset
unit

pollIntervalMs
qualityRule
metadata

dataPointId
dataPointPath
dataPointStatus
lastValue
quality
lastUpdatedAt
sortOrder
createdBy
updatedBy
createdAt
updatedAt
```

推荐 `data_s7_variables` 以结构化字段为准：

```text
area
  DB / M / I / Q / T / C

db_number
  DB 区必填，其他区为空。

byte_offset
  所有按字节读取的数据类型必填。

bit_offset
  Bool 必填，范围 0-7。

address_text
  用户输入和展示用。

normalized_address
  系统标准化后的地址，例如 DB1.DBX0.0、DB1.DBD4、M0.0。

address_type
  DBX / DBB / DBW / DBD / MB / MW / MD / I / Q 等，用于提示和校验。

read_length
  系统根据 dataType、length、arrayLength 推导出的读取字节数。
```

`lastValue / quality / lastUpdatedAt` 是开发态辅助快照，用于下次打开工作台时看到最近一次读取或预览结果。它不等同于运行态正式存储，不参与数据点历史查询。

不要在 S7 变量内放存储配置字段：

```text
storageEnabled
storagePolicy
retention
deadband
```

这些属于数据点层。

### 9.3 地址区

```text
DB    数据块
M     存储区
I     输入区
Q     输出区
T     定时器，可选
C     计数器，可选
```

完整设计可以容纳 T/C，但工作台主路径应围绕 DB/M/I/Q。

### 9.4 地址表达

UI 支持常见地址文本输入，同时后端结构化存储地址字段。

```text
DB1.DBX0.0
DB1.DBB1
DB1.DBW2
DB1.DBD4
M0.0
MB1
MW2
MD4
I0.0
Q0.0
```

`addressText` 用于展示、导入、搜索和人机确认；真实校验、读取计划和运行态契约依赖结构化字段。

### 9.5 数据类型

```text
Bool
Byte
Word
DWord
Int
DInt
Real
String
DateTime
```

String 必须配置长度。Bool 必须具备 bitOffset。Real、DInt、DWord 等按字节偏移读取并结合字节序、字序解释。

### 9.6 后端 API 边界

S7 工作台建议提供与领域对象对应的 API，而不是让前端拼装配置 JSON：

```text
GET    /api/v1/data/projects/{projectId}/s7/{connectionId}/profile
PUT    /api/v1/data/projects/{projectId}/s7/{connectionId}/profile

GET    /api/v1/data/projects/{projectId}/s7/{connectionId}/variable-groups
POST   /api/v1/data/projects/{projectId}/s7/{connectionId}/variable-groups
PUT    /api/v1/data/projects/{projectId}/s7/{connectionId}/variable-groups/{groupId}
DELETE /api/v1/data/projects/{projectId}/s7/{connectionId}/variable-groups/{groupId}

GET    /api/v1/data/projects/{projectId}/s7/{connectionId}/variables
POST   /api/v1/data/projects/{projectId}/s7/{connectionId}/variables
POST   /api/v1/data/projects/{projectId}/s7/{connectionId}/variables/batch-import
PUT    /api/v1/data/projects/{projectId}/s7/{connectionId}/variables/{variableId}
DELETE /api/v1/data/projects/{projectId}/s7/{connectionId}/variables/{variableId}

POST   /api/v1/data/projects/{projectId}/s7/{connectionId}/validate-model
POST   /api/v1/data/projects/{projectId}/s7/{connectionId}/preview
GET    /api/v1/data/projects/{projectId}/s7/{connectionId}/read-plan-estimate
```

开发态会话可以沿用协议预览会话范式，但 S7 需要返回更具体的诊断：

```text
连接握手结果
协商 PDU
PLC 系列识别结果
Rack/Slot 或 TSAP 诊断
单次读取耗时
读取块失败原因
```

## 10. 新建变量

```text
┌──────────────────── 新建 S7 变量 ────────────────────┐
│ 基础信息                                               │
│ 变量名       [ 炉温                                  ] │
│ 变量编码     [ furnace_temperature                  ] │
│ 分组         [ 熔炼炉                          v     ] │
│                                                       │
│ 地址                                                   │
│ 地址区       [ DB 区                           v     ] │
│ DB 块号      [ 1                                    ] │
│ 地址类型     [ DBD                            v     ] │
│ 字节偏移     [ 4                                    ] │
│ 位偏移       [ - ]  仅 Bool 使用                      │
│ 预览地址     DB1.DBD4                                 │
│                                                       │
│ 数据解释                                               │
│ 数据类型     [ Real                           v     ] │
│ 长度/数量    [ 1                                    ] │
│ 字节序       [ Big Endian                     v     ] │
│ 字序         [ Big Endian                     v     ] │
│ 倍率         [ 1        ]  偏移 [ 0 ]  单位 [ ℃ ]     │
│                                                       │
│ 采集                                                   │
│ [✓] 启用采集     周期 [继承接入源 1000ms v]            │
│                                                       │
│ 数据点                                                 │
│ 保存后自动生成：s7.furnace_plc.melting.furnace_temperature │
│                                                       │
│                                  [取消] [保存并读取]  │
└───────────────────────────────────────────────────────┘
```

保存变量时必须自动 upsert 数据点。保存失败时不应出现变量已保存但数据点缺失的静默状态。

## 11. 批量导入变量

### 11.1 导入目标

导入用于承接现场地址表。支持 Excel/CSV，允许用户映射字段、预览错误、只导入有效项或修正后导入。

### 11.2 字段映射

```text
变量名
变量编码
分组
地址
数据类型
单位
倍率
偏移
描述
启用状态
采集周期
```

不导入存储配置。存储配置由数据点模块统一处理。

### 11.3 原型

```text
┌──────────────────────── 导入变量 ────────────────────────┐
│ [选择 Excel/CSV 文件]                                     │
│                                                            │
│ 字段映射                                                   │
│ 变量名 -> name     地址 -> address     类型 -> dataType     │
│ 分组   -> group    单位 -> unit        描述 -> description  │
│                                                            │
│ 导入预览                                                   │
│ ┌────┬──────┬──────────┬──────┬──────────────┐             │
│ │状态│变量名│地址       │类型  │问题          │             │
│ ├────┼──────┼──────────┼──────┼──────────────┤             │
│ │ ✓  │炉温  │DB1.DBD4  │Real  │              │             │
│ │ !  │状态  │DB1.DBX0  │Bool  │缺少 bit 位   │             │
│ └────┴──────┴──────────┴──────┴──────────────┘             │
│                                                            │
│                            [取消] [仅导入有效项] [全部修正] │
└────────────────────────────────────────────────────────────┘
```

S7 导入要比 Modbus 更重视地址文本解析和用户可修正性。现场地址表常见问题包括：

```text
地址写法不统一：DB1.DBD4、DB1,DBD4、DB1 4.0、M0.0 混用
Bool 缺少 bit 位：DB1.DBX0
String 未写长度
DB 块号与地址列拆分
变量名重复或无业务含义
TIA 符号名与绝对地址同时存在
```

导入流程应支持三步：

```text
1. 字段映射
   用户把 Excel/CSV 列映射到变量名、Code、分组、地址、类型、单位、倍率、周期等字段。

2. 地址解析预览
   系统把 addressText 解析为 area/dbNumber/byteOffset/bitOffset/readLength，并给出 normalizedAddress。

3. 导入确认
   用户可以直接修改变量名、Code、分组、地址和类型，避免重名或现场表头含糊。
```

导入预览表建议列：

```text
状态 | 变量名 | Code | 分组 | 原始地址 | 标准地址 | 类型 | 字节范围 | 问题
```

不建议在导入阶段要求用户配置存储、报警或计算。

## 12. 数据点自动同步

### 12.1 原则

协议工作台负责“怎么采”。数据点负责“平台怎么用”。S7 变量、Modbus 寄存器、OPC UA 节点、MQTT Tag 都应该自动生成平台数据点，避免用户重复维护。

### 12.2 同步规则

```text
新增变量
  自动创建数据点

修改变量名、编码、分组
  自动更新数据点名称、路径和来源摘要

修改地址、类型、采集周期
  自动更新数据点 sourceConfig

禁用变量
  数据点保留，状态变为 inactive 或 disabled

删除变量
  默认将数据点标记为 invalid 或 detached，不物理删除

恢复变量
  重新绑定或更新对应数据点
```

### 12.3 数据点来源

```text
sourceType     s7.variable
sourceId       S7 变量 ID
sourceIdPath   connectionId/groupId/variableId
connectionId   S7 接入源 ID
dataType       变量数据类型映射后的平台类型
path           s7.<connectionCode>.<groupCode>.<variableCode>
```

### 12.4 同步状态

工作台展示自动同步状态，不提供主流程按钮：

```text
128 个变量 · 128 个数据点已自动同步
```

变量详情中展示：

```text
数据点
path: s7.furnace_plc.melting.furnace_temperature
状态: 已同步
```

同步失败时展示为校验或诊断问题：

```text
数据点同步异常：变量编码重复，无法生成唯一 path
```

## 13. 存储配置边界

### 13.1 结论

S7 和 Modbus 等协议工作台不配置存储。存储策略统一在数据点模块配置。

### 13.2 原因

如果每个协议变量都配置存储，会导致：

```text
MQTT、OPC UA、S7、Modbus 各自一套存储字段
存储策略重复实现
用户需要在协议工作台和数据点模块之间理解两套配置
报警、计算、工作流引用数据点时无法形成统一治理
```

数据点才是平台统一消费对象，因此存储应围绕数据点配置。

### 13.3 数据点存储配置

```text
storage.enabled
storage.storeId
storage.policy
storage.intervalMs
storage.changeOnly
storage.deadband
storage.retention
storage.compression
```

协议变量只保留采集相关字段：

```text
enabled
pollIntervalMs
address
dataType
scale
offset
unit
```

## 14. 建模校验

### 14.1 校验内容

```text
变量名必填
变量编码唯一
分组存在
地址格式合法
DB 区必须有 dbNumber
Bool 必须有 bitOffset
String 必须有 length
数据类型与地址类型匹配
字节偏移不能为负数
采集周期必须大于 0
同一变量必须能生成唯一数据点 path
同一地址重复使用需要提示
读取范围重叠需要提示
读取计划碎片过多需要提示
PLC 档案与变量地址能力不匹配需要提示
```

### 14.2 原型

```text
┌────────────── 建模校验 ──────────────┐
│ 结果：3 个错误，5 个警告              │
│                                       │
│ 错误                                  │
│ ✕ DB1.DBX0 缺少 bitOffset             │
│ ✕ 炉温变量编码重复                    │
│ ✕ String 类型缺少长度                 │
│                                       │
│ 警告                                  │
│ ! DB1.DBD4 与 DB1.DBW6 读取范围重叠   │
│ ! 变量分布过碎，预计读取块较多        │
│                                       │
│ [定位变量] [导出报告] [关闭]          │
└───────────────────────────────────────┘
```

## 15. 开发态连接与读取

### 15.1 连接/断开

S7 工作台使用明确的“连接/断开”心智，不使用“测试连接”作为主动作。

连接后可执行：

```text
读取当前值
短时变量预览
读取计划估算
连接诊断
```

断开后停止轮询并释放开发态连接资源。

### 15.2 读取当前值

读取当前值是单次动作，支持：

```text
读取当前选中变量
读取当前分组变量
读取勾选变量
```

读取结果包含：

```text
原始值
换算后值
质量
读取耗时
错误信息
更新时间
```

### 15.3 变量预览

变量预览是开发态短时轮询，不写入正式存储，不触发正式报警、计算和工作流。

```text
┌──────────────────── 变量预览 ────────────────────┐
│ 范围 [当前分组 v]  周期 [1000ms]  [开始] [停止]   │
│                                                    │
│ ┌────────┬──────────┬──────┬────────────┬──────┐  │
│ │变量名   │地址       │当前值│更新时间     │质量 │  │
│ ├────────┼──────────┼──────┼────────────┼──────┤  │
│ │炉温     │DB1.DBD4  │723.5 │14:22:10    │Good │  │
│ │运行状态 │DB1.DBX0.0│true  │14:22:10    │Good │  │
│ └────────┴──────────┴──────┴────────────┴──────┘  │
│                                                    │
│ 错误日志                                           │
│ 14:22:08 DB3.DBD0 read timeout                     │
└────────────────────────────────────────────────────┘
```

## 16. 读取计划

### 16.1 设计目的

S7 地址读取应尽量按区域、DB 块和连续地址合并。读取计划用于让用户理解变量分布对采集性能的影响。

### 16.2 计划维度

```text
区域
DB 号
起始字节
结束字节
读取长度
包含变量数
预计耗时
风险提示
```

读取计划不是让用户手动配置的“运行计划”，而是系统对运行态读取行为的解释和契约预览。它服务三个对象：

```text
用户
  看懂变量分布是否过散，是否会造成 PLC 读取压力。

平台二开人员
  在建模阶段就能看到读取块、读取字节数、reads/s 和优化 DB 访问风险，便于调整地址表或轮询周期。

运行态
  直接消费 readPlans，避免运行时重新扫描变量并临时决定怎么读。
```

### 16.3 合并规则

S7 读取计划按以下 key 分组：

```text
connectionId
area
dbNumber
pollIntervalMs
readMode
```

其中：

```text
DB 区按 dbNumber 分组，不同 DB 块绝不合并。
M/I/Q 区没有 dbNumber，但不同区域绝不合并。
不同采集周期不合并。
不同 readMode 不合并，例如标准绝对地址读取与符号读取不合并。
```

组内按 `byteOffset` 排序后合并连续或小间隔地址。合并后的读取块必须满足：

```text
读取长度 <= PLC 档案 maxReadBytes
读取长度 <= 协商 PDU 可承载长度
读取块内变量数量 <= 单块建议上限
读取间隙 <= maxGapBytes
```

`maxGapBytes` 不是越大越好。间隙过大会多读无用字节，间隙过小会导致读取块碎片过多。推荐由 PLC 档案给默认值：

```text
S7-300 / S7-400：默认 8 bytes
S7-1200 / S7-1500：默认 16 bytes
S7 Compatible：默认 8 bytes
```

### 16.4 性能指标

读取计划估算应输出：

```text
variableCount          启用变量数
blockCount             读取块数
totalReadBytes         单轮读取总字节数
readsPerSecond         预计每秒读取次数
estimatedCycleMs       预计单轮耗时
largestBlockBytes      最大读取块字节数
fragmentedGroupCount   地址碎片较多的分组数
diagnostics            风险提示
```

风险提示示例：

```text
DB5 地址跨度 0-920，但只包含 9 个变量，建议整理地址或接受额外读取块。
1s 周期下预计 65 reads/s，可能对 PLC 或网络造成压力。
S7-1200 开启优化块访问时，绝对 DB 地址可能不可读，请确认 DB 块关闭优化访问。
单块读取 240 bytes 接近 PDU 限制，运行态可能拆分读取。
```

### 16.5 原型

```text
┌──────────────────── 读取计划估算 ────────────────────┐
│ 范围：全部启用变量                                    │
│ 变量数：128  读取块：18  总字节：1460  预计：18 reads/s │
│ PDU：自动协商 240 bytes  合并间隙：16 bytes             │
│                                                        │
│ ┌──────┬─────┬────────────┬──────┬──────┬────────┐   │
│ │区域  │DB号 │读取范围     │字节  │变量数│周期    │   │
│ ├──────┼─────┼────────────┼──────┼──────┼────────┤   │
│ │DB    │1    │0 - 64       │65    │23    │1000ms  │   │
│ │DB    │2    │0 - 120      │121   │41    │1000ms  │   │
│ │M     │-    │0 - 32       │33    │12    │1000ms  │   │
│ └──────┴─────┴────────────┴──────┴──────┴────────┘   │
│                                                        │
│ 诊断                                                   │
│ ! DB5 地址分散，建议整理或接受额外读取块。              │
│ ! S7-1200/1500 如开启优化 DB 访问，绝对地址可能不可读。 │
└────────────────────────────────────────────────────────┘
```

## 17. 右侧详情与诊断

右侧面板根据当前上下文展示：

```text
PLC 档案摘要
当前变量详情
数据点同步状态
读取结果
建模校验问题
连接诊断
读取计划摘要
```

变量详情示例：

```text
变量
  炉温
  DB1.DBD4
  Real

采集
  启用
  继承 1000ms

数据点
  s7.furnace_plc.melting.furnace_temperature
  已同步

最近读取
  723.5 ℃
  Good
  2026-05-25 14:22:10
```

## 18. 与 Modbus 的关系

S7 和 Modbus 都属于地址表建模型协议，但不应强行复用同一套字段命名，也不应强行共用同一个变量模型。

共同点：

```text
分组
变量/寄存器表
导入
读取当前值
短时轮询预览
建模校验
读取计划
自动同步数据点
存储归数据点层
```

差异点：

```text
S7 有 PLC 系列、Rack/Slot、TSAP、PDU、DB/M/I/Q 地址区、优化 DB 访问限制
Modbus 有 TCP/RTU、Slave ID、功能码、寄存器区、起始地址和数量
S7 读取计划按 area/db/byte range 合并
Modbus 读取计划按 slave/function/address range 合并
S7 需要处理 byteOffset/bitOffset/readLength
Modbus 需要处理 unitId/protocolAddress/quantity
```

可以共享：

```text
WorkbenchSourceHeader
左侧分组树交互模式
中间表格选中/搜索/操作模式
右侧 Inspector 思路
导入、预览、校验、读取计划弹窗结构
API 响应 list/issue/estimate 的外形
```

不能共享：

```text
字段命名
地址解析逻辑
读取计划合并算法
数据类型数量推导
运行态 artifact 内部结构
```

S7 设计从平台二开角度应优先稳定“用户能看懂、能调整、能发布”的建模闭环，不是追求和 Modbus 抽成同一套泛型协议模型。

## 19. 运行态契约

发布运行态时，S7 artifact 应包含：

```text
接入源连接参数
PLC 档案
变量分组
变量表
数据点映射
读取计划
采集周期
数据解释规则
质量规则
```

artifact 建议结构：

```json
{
  "profile": {},
  "groups": [],
  "variables": [],
  "readPlans": []
}
```

`variables` 保留用户建模结果：

```text
id
connectionId
groupId
name
code
addressText
normalizedAddress
area
dbNumber
byteOffset
bitOffset
readLength
dataType
byteOrder
wordOrder
scale
offset
unit
pollIntervalMs
datapointPath
```

`readPlans` 给运行态高效执行：

```text
id
connectionId
area
dbNumber
startByte
endByte
readLength
pollIntervalMs
variableIds
variableCount
maxGapBytes
readMode
```

运行态不应重新猜测地址，也不应从数据点 path 反推 S7 变量。运行态采集应直接消费 S7 建模 artifact。

存储、报警、计算和工作流契约通过数据点引用关联。

### 19.1 高可用与并发边界

多运行态节点时，S7 最大风险不是多算一次，而是多个节点同时读取同一台 PLC，造成 PLC 通信资源、PDU 队列或网络压力异常。

设计原则：

```text
同一个 S7 connection 或 readPlan 同一时刻只有一个运行态 Owner。
```

运行态可采用：

```text
按 connectionId 分片
基于 readPlan 的 lease
lease TTL + fencing token
节点掉线后自动转移
转移冷却时间，避免频繁抖动
```

默认推荐按连接分片。只有当一个 PLC 的变量量非常大，并且现场确认 PLC 能承受并发读取时，才允许按 readPlan 拆分并发执行。

### 19.2 运行态执行建议

运行态读取时应：

```text
按 readPlans 分组调度
同一连接默认串行或小并发读取
读取失败时按块降级，不让单个变量阻断整个连接
把块级错误映射回变量质量
按变量 dataType/byteOrder/wordOrder 解码
按 scale/offset 生成工程值
把结果写入实时通道和数据点存储链路
```

运行态不负责：

```text
自动修正地址
自动改写变量定义
自动新增数据点
猜测 PLC 系列
绕过工作台校验临时拼读取块
```

## 20. 错误与异常处理

### 20.1 连接异常

```text
PLC 不可达
端口不可达
Rack/Slot 错误
TSAP 错误
连接超时
驱动握手失败
```

### 20.2 读取异常

```text
地址不存在
DB 块不可访问
偏移越界
数据类型解释失败
读取超时
部分变量读取失败
```

### 20.3 数据点同步异常

```text
变量编码为空
变量编码重复
生成 path 冲突
数据点来源已被其他对象占用
数据点更新失败
```

异常需要出现在右侧详情、校验抽屉或预览日志中，不能静默吞掉。

## 21. 成功标准

```text
用户可以轻量创建 Siemens S7 接入源。
用户可以在 S7 工作台补充 PLC 档案。
用户可以维护 S7 分组和变量地址表。
用户可以导入 Excel/CSV 地址表并预览错误。
导入时可以把原始地址解析成标准地址和结构化字段。
变量保存后自动生成或更新 s7.variable 数据点。
S7 工作台不出现存储配置字段。
数据点模块可以按 s7.variable 来源筛选并配置存储。
用户可以连接/断开开发态 S7 会话。
连接后可以读取当前值和短时轮询预览。
变量表可以展示最近值、质量和最近更新时间。
用户可以运行建模校验并定位问题变量。
用户可以查看读取计划估算、读取块数量、总字节、reads/s 和风险提示。
发布 artifact 能包含运行态采集所需的 S7 建模契约。
artifact 中包含 profile、variables、readPlans 和数据点映射。
平台二开人员可以在界面中完成 PLC 类型确认、地址表建模、读取压力评估和运行契约发布。
运行态可以直接消费 readPlans，不需要重新解析 addressText。
多运行态节点不会默认同时轮询同一个 S7 连接。
```

## 22. 非目标

以下能力不属于 S7 工作台本身：

```text
数据点存储策略配置
报警规则配置
计算单元编排
工作流编排
页面绑定配置
全局设备资产管理
正式运行态长期采集进程
```

这些能力通过数据点和运行态服务完成，S7 工作台只负责协议建模和开发态验证。
