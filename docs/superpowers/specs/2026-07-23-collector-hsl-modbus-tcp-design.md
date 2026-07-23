# HSL 通用适配层与 Modbus TCP 驱动设计

## 目标

在不改变现有 OPC UA 技术实现的前提下，解除 DevAgent 对 OPC UA 的硬编码，建立可复用的 HSL 通用适配层，并完成 Modbus TCP 的连接测试、长连接和批量读取闭环。

## 设计原则

- OPC UA 继续使用 OPC Foundation 官方 SDK，不经过 HSL。
- HSL 只作为部分工业协议驱动的底层通信实现，不进入平台通用契约。
- DevAgent 只负责任务路由、会话治理和结果回传，不解析具体协议配置。
- 每个驱动自行解析并校验连接配置、地址配置和数据类型。
- 新增协议时不修改任务执行器中的协议分支。
- 当前项目不保留旧连接模型兼容分支，直接调整为最终结构。

## 首期范围

### 包含

- 通用连接配置透传。
- 现有 OPC UA 驱动适配新连接契约。
- HSL 公共适配层。
- Modbus TCP 驱动。
- Modbus TCP 连接测试。
- Modbus TCP 长连接和主动断开。
- 当前页变量批量读取。
- 单点读取失败隔离。
- 标准数据值、质量码和时间戳转换。
- DevAgent 驱动注册和协议能力上报。
- 协议清单、单元测试和模拟服务集成测试。

### 不包含

- Modbus 写入。
- Modbus 设备树浏览。
- Modbus RTU。
- Siemens S7、三菱、欧姆龙等后续协议。
- 运行采集器接入。
- 数据库存储结构调整。
- 前端新增独立协议工作台。

## 总体架构

```text
DevAgent
  ├─ 任务执行与会话管理
  └─ 平台驱动契约
       ├─ OPC UA 驱动
       │    └─ OPC Foundation SDK
       └─ Modbus TCP 驱动
            └─ HSL 通用适配层
                 └─ HslCommunication
```

平台通用层只认识驱动能力、原始配置、标准读取请求和标准读取结果。OPC UA 与 HSL 的类型、异常和连接对象都限制在各自驱动实现内部。

## 通用连接契约

现有 `ConnectionProfile` 包含 `EndpointUrl`、`SecurityMode` 和 `SecurityPolicy` 等 OPC UA 专属字段，无法完整承载 Modbus、串口和 PLC 配置。

调整后的连接对象只保留：

- `ProtocolFamily`：协议族。
- `Config`：连接配置原始 JSON。
- `Secrets`：敏感配置原始 JSON。

任务执行器不再构造 OPC UA Endpoint，也不再判断协议族是否为 `opcua`。任务执行器将连接配置原样交给目标驱动，由驱动完成强类型映射、默认值应用和语义校验。

## 通用点位读取契约

现有 `ReadRequest` 只包含 OPC UA NodeId，任务执行器还会直接调用 OPC UA 地址解析器，无法表达 Modbus 的结构化地址、数据类型和元素数量。

读取契约调整为按点位传递：

- `Key`：本次任务中的点位关联键，使用采集点 ID。
- `Address`：结构化地址原始 JSON。
- `DataType`：平台标准数据类型。
- `ElementCount`：元素数量。
- `ReadOptions`：协议读取选项原始 JSON。

驱动返回结果时使用 `Key` 关联请求，不再依赖结果数组下标或 NodeId。每个点位结果包含成功状态、值、实际数据类型、质量码、时间戳、错误码和错误消息。

任务执行器只校验请求是否包含基础字段，不再调用 `OpcUaAddressMapper`。具体地址合法性和单点错误隔离统一由目标驱动负责。

Data Service 当前生成的调试任务已经包含 `pointId`、`address`、`dataType`、`elementCount` 和 `readOptions`，因此不需要新增接口字段，只调整 Agent 内部反序列化和驱动契约。

### OPC UA 适配

OPC UA 驱动新增内部连接配置映射器，将原始配置转换为现有 Endpoint、SecurityMode、SecurityPolicy、认证方式和超时配置。

OPC UA 已有连接、浏览、缓存和读取行为保持不变，仅改变配置进入驱动的方式。

### Modbus TCP 配置

Modbus TCP 驱动解析现有协议 Schema 中的：

- `host`
- `port`
- `connectTimeoutMs`

连接测试和长连接共享相同的配置解析与校验逻辑。主机为空、端口越界或超时越界时，由驱动返回明确、不可重试的配置错误。

## HSL 通用适配层

新增独立项目 `InduForge.Collector.Adapters.Hsl`，直接引用 `HslCommunication`，负责不同 HSL 协议驱动可复用的基础能力。

首期职责：

- 统一 HSL 操作结果判定。
- 将 HSL 错误转换为平台驱动异常。
- 统一连接、关闭和释放生命周期。
- 提供标准数据类型读取辅助方法。
- 将读取失败转换为单点诊断，不让单个地址导致整批任务失败。
- 屏蔽 HSL 类型，禁止其出现在 `InduForge.Collector.Contracts`、DevAgent 任务模型和接口响应中。

首期不建立复杂基类层级。公共层只提取 Modbus TCP 已经实际使用、并确认后续协议可复用的最小能力。

## Modbus TCP 驱动

新增项目 `InduForge.Collector.Drivers.ModbusTcp`，实现：

- `IIndustrialDriver`
- `IConnectionSessionDriver`
- `IPointReader`

会话对象实现：

- `IIndustrialConnectionSession`
- `IPointReaderSession`

驱动能力清单为：

- `connection.test`
- `connection.open`
- `connection.close`
- `point.read`

不声明 `device.browse`，因此前端不会展示 OPC UA 风格的设备浏览入口。

## 地址模型

继续使用现有 `modbus.tcp/address.schema.json`：

- `station`：站号，范围 `0-255`。
- `area`：`coil`、`discreteInput`、`inputRegister`、`holdingRegister`。
- `address`：协议地址，范围 `0-65535`。
- `bitIndex`：寄存器位索引，范围 `0-15`。

驱动内部将结构化地址转换为 HSL 可识别的地址表达式。转换逻辑集中在 Modbus TCP 驱动内部，不复用展示用途的 `addressText`。

### 区域约束

- `coil`：读取布尔值，不允许配置 `bitIndex`。
- `discreteInput`：读取布尔值，不允许配置 `bitIndex`。
- `inputRegister`：读取寄存器值，布尔类型必须提供 `bitIndex`。
- `holdingRegister`：读取寄存器值，布尔类型必须提供 `bitIndex`。
- `bitIndex` 只对寄存器区域有效。

不符合区域和数据类型组合的变量在保存阶段与驱动执行阶段都应被拒绝。

## 数据类型映射

首期支持协议清单中已声明的数据类型：

- `bool`
- `int8`、`uint8`
- `int16`、`uint16`
- `int32`、`uint32`
- `int64`、`uint64`
- `float32`、`float64`
- `string`
- `bytes`

`datetime` 暂不作为 Modbus TCP 第一阶段的原生读取类型；若协议清单当前声明了但没有明确寄存器编码规则，应从首期能力中移除，后续通过显式转换配置实现。

### 元素数量

- 标量类型默认 `elementCount=1`。
- 数组读取使用变量的 `elementCount`。
- 字符串和字节读取长度由 `elementCount` 表示。
- 驱动根据数据类型计算实际寄存器占用数量。

### 字节序

Modbus 多寄存器值存在字节序和字交换差异。首期在连接配置中增加高级字段 `dataFormat`，由连接级配置统一控制，默认使用 HSL 的标准 ABCD 格式。

首期支持：

- `ABCD`
- `BADC`
- `CDAB`
- `DCBA`

变量级字节序覆盖不在首期范围内。

## 批量读取策略

外部仍使用单次 `point.read` 批量任务，但驱动接收新的结构化点位读取契约。Modbus TCP 驱动按以下规则处理：

1. 解析并校验每个变量地址。
2. 按站号和区域分组。
3. 对可连续合并的寄存器区间生成批量读取段。
4. 控制单次请求长度，不超过 Modbus 协议和 HSL 实现允许的范围。
5. 从批量响应中切分并转换各变量值。
6. 某个批量段失败时，对该段内变量执行降级读取，以隔离具体失败地址。
7. 按点位 `Key` 返回成功值或失败信息，并附加批次级诊断。

第一版优先保证结果正确和单点隔离，再通过测试固定批量合并规则。不得由前端循环逐点发请求。

## 质量码与时间

- 读取成功统一返回 `Good`。
- 地址无效、设备异常响应、类型转换失败和通信失败统一返回失败诊断，由现有 DevAgent 结果映射为 `Bad`。
- Modbus 不提供源设备时间戳，`SourceTimestamp` 和 `ServerTimestamp` 均使用读取完成时的 Agent 本机时间。
- 同一批次成功值使用同一个读取完成时间，避免产生没有意义的逐点时间差。

## 异常处理

驱动异常统一包含：

- 稳定错误码。
- 用户可理解的中文消息。
- 是否可重试。
- 原始异常作为内部异常链，进入 DevAgent 文件日志但不直接返回前端。

错误分类至少包括：

- 配置无效：不可重试。
- 地址无效：不可重试，只影响对应变量。
- 连接超时：可重试。
- 连接断开：可重试。
- 设备异常响应：根据 HSL 返回信息判断，默认可重试。
- 数据类型转换失败：不可重试，只影响对应变量。

## DevAgent 注册

- DevAgent 引用 Modbus TCP 驱动项目。
- 嵌入 `modbus.tcp/manifest.json`。
- `DriverRegistry.CreateDefault()` 注册 OPC UA 与 Modbus TCP 工厂和清单。
- 注册校验继续保证代码声明能力与 Manifest 中 `operations` 完全一致。
- Agent 心跳上报两个驱动的真实能力。

## 协议 Schema 调整

更新 `runtime/collector_protocols/modbus.tcp`：

- `connection.schema.json` 增加 `dataFormat` 高级配置。
- `manifest.json` 写入实际 operations。
- `platforms.devAgent` 增加 `windows-x64`。
- 数据类型清单与首期真实支持范围保持一致。
- Schema Version 在结构发生变化时递增，不保留旧版本兼容逻辑。

## 测试策略

### 通用契约和 DevAgent

- 原始 Config 和 Secrets 能完整进入驱动。
- 结构化 Address、DataType、ElementCount 和 ReadOptions 能完整进入驱动。
- 驱动结果通过点位 Key 正确回填，不依赖数组下标。
- 任务执行器不再拒绝非 OPC UA 协议。
- 任务执行器不再引用 OPC UA 地址解析器。
- OPC UA 现有任务回归通过。
- DriverRegistry 同时上报 OPC UA 与 Modbus TCP。
- 长连接复用和离开工作台后的统一关闭对两类驱动均有效。

### HSL 公共适配层

- 成功和失败结果映射。
- 可重试属性映射。
- HSL 原始异常不会泄漏到外部响应。
- 连接对象能够安全释放。

### Modbus TCP 驱动

- 连接配置校验。
- 四种区域地址映射。
- 站号和位索引处理。
- 支持的数据类型转换。
- 四种 `dataFormat`。
- 元素数量和寄存器长度计算。
- 单点失败隔离。
- 长连接复用。

### 集成测试

使用测试内可启动和停止的 Modbus TCP 模拟服务，覆盖：

- 连接成功和端口不可达。
- Coil 与 Discrete Input 读取。
- Input Register 与 Holding Register 读取。
- 多变量同批读取。
- 不同站号分组。
- 无效地址不影响其他变量。
- 断开连接后读取失败。

测试端口由测试进程动态分配，不使用根目录开发服务端口，避免与本机服务和 WSL 基础设施冲突。

## 验收标准

- OPC UA 现有连接、浏览、长连接和读取功能无回退。
- DevAgent 不再包含只允许 OPC UA 的协议硬编码。
- Modbus TCP 连接能够在工作台执行连接测试、建立长连接和主动断开。
- 长连接状态能够沿用现有左侧连接状态展示。
- 当前页变量读取能够返回 Modbus TCP 的值、Good/Bad、读取时间和失败信息。
- 一个无效 Modbus 地址不会导致同批其他变量失败。
- 前端根据能力清单不展示 Modbus 设备浏览入口。
- 协议 Manifest 与驱动实际能力一致。
- Collector 全量测试和构建通过。
- Data Service 与 Datacenter 受影响范围的类型检查和测试通过。
