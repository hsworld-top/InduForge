# Modbus RTU 采集调试代理设计

## 目标

在现有 HSL 通用适配层和 DevAgent 驱动注册机制上，完成 `modbus.rtu` 的连接测试、长连接、主动断开和点位读取，使其达到当前 Modbus TCP 驱动相同的调试能力，并为后续串口协议复用保留清晰边界。

## 成功标准

- DevAgent 上报 `modbus.rtu`，Schema 基线版本固定为 `1`。
- 支持 `connection.test`、`connection.open`、`connection.close`、`point.read`。
- 支持串口名称、波特率、数据位、校验位、停止位、超时和 Modbus 数据格式。
- 地址结构继续使用 `station`、`area`、`address`、`bitIndex`。
- 读取类型和单点读取上限与 Modbus TCP 保持一致。
- 单点失败只返回该点错误，不中断同批其他点位。
- 连接和读取生命周期可测试，不把 HSL 类型暴露到平台公共契约。
- Collector 全量测试和 Release 构建通过。

## 方案比较

### 方案一：在 Modbus RTU 驱动中直接调用 HSL

代码最少，但驱动会同时承担平台语义、HSL 生命周期和串口细节，后续其他 HSL 串口协议无法复用错误转换和测试替身。

### 方案二：建立万能 HSL 串口客户端

表面复用度高，但不同串口协议的站号、帧格式、连接参数和读取模型差异明显，会形成大量条件分支和无效抽象。

### 方案三：协议专属 HSL RTU 适配器与独立驱动

在 HSL 适配项目中增加 `IHslModbusRtuClient`，仅复用已经稳定的读取请求、结果模型和数据格式枚举；在独立驱动项目中负责连接配置与地址语义。该方案边界清晰，测试方式与 Modbus TCP 一致，是本次采用方案。

## 架构

```text
DevAgent
  ├─ DriverRegistry
  ├─ CollectorConnectionSessionManager
  └─ ModbusRtuDriver
       └─ IHslModbusRtuClient
            └─ HslCommunication.ModBus.ModbusRtu
```

DevAgent 只按公共驱动契约调用。Modbus RTU 驱动解析连接和点位配置，HSL 适配层负责真实串口对象、连接、关闭、读取和错误归一化。

## HSL RTU 适配层

新增连接参数：

- `PortName`
- `BaudRate`
- `DataBits`
- `Parity`
- `StopBits`
- `TimeoutMilliseconds`
- `DataFormat`

新增串口枚举：

- `HslSerialParity`：`None`、`Odd`、`Even`。
- `HslSerialStopBits`：`One`、`Two`。

新增 `IHslModbusRtuClient` 和工厂。接口与 Modbus TCP 客户端保持一致，继续接收 `HslReadRequest`，但内部使用 `ModbusRtu.SerialPortInni` 初始化串口并通过 `Open`/`Close` 管理生命周期。

HSL 返回失败时统一转换为：

- 连接失败：`HSL_CONNECT_FAILED`，可重试。
- 读取失败：`HSL_READ_FAILED`，可重试。
- 不支持的数据类型：`HSL_DATA_TYPE_UNSUPPORTED`，不可重试。

## Modbus RTU 驱动

新增 `InduForge.Collector.Drivers.ModbusRtu` 项目，结构与 Modbus TCP 驱动保持一致：

- `ModbusRtuConnectionOptions`：解析和校验串口配置。
- `ModbusRtuAddress`：解析结构化地址并生成 HSL 地址。
- `ModbusRtuConnectionSession`：持有长连接客户端并串行化会话读取。
- `ModbusRtuDriver`：实现测试连接、打开、关闭和点位读取。
- `ModbusRtuDriverException`：承载稳定错误码与可重试属性。

连接配置校验规则：

- `portName` 必填，去除首尾空格后不能为空。
- `baudRate` 必须大于零。
- `dataBits` 仅允许 `7` 或 `8`。
- `parity` 仅允许 `none`、`odd`、`even`。
- `stopBits` 仅允许 `one`、`two`。
- `timeoutMs` 范围为 `100` 至 `120000`，缺省为 `3000`。
- `dataFormat` 仅允许 `ABCD`、`BADC`、`CDAB`、`DCBA`，缺省为 `ABCD`。

地址解析、数据类型映射和读取数量上限沿用 Modbus TCP 的最终规则，避免两个 Modbus 传输层产生不同语义。

## 串口目录与主机资源快照

串口枚举是 Agent 主机能力，不属于连接或点位地址。新建连接尚未生成 `connectionId`，因此不能复用现有“必须绑定连接”的调试任务模型。

采用协议能力资源快照：

- Agent 使用 `System.IO.Ports.SerialPort.GetPortNames()` 枚举串口，稳定排序并去重。
- Agent 注册和每次心跳重新生成能力快照。
- `AgentProtocolCapability.resources.serialPorts` 只附加到 Manifest 声明 `serial` transport 的驱动。
- Data Service 继续把完整 capability 数组存入现有 JSONB 字段，不新增数据库字段和迁移。
- 前端从当前选中 Agent 的目标驱动 capability 读取串口列表。
- 表单使用可搜索、可创建的下拉框，既能选择枚举结果，也保留手动输入。
- 用户点击刷新时重新加载 Agent 列表；最新串口列表会在下一次心跳后出现。

串口枚举失败只记录 Agent 文件日志并上报空数组，不能影响 Agent 启动、注册、心跳或其他驱动。

## Manifest 与注册

`runtime/collector_protocols/modbus.rtu/manifest.json` 更新为：

- `operations`：四项真实操作。
- `platforms.devAgent`：`windows-x64`。
- `features`：保留 `point.elementCount`。
- `schemaVersion`：保持 `1`。

连接 Schema 增加 `dataFormat`，其他字段保持当前最终结构。DevAgent 嵌入 RTU Manifest，并在默认注册表中创建 `ModbusRtuDriver`。

## 测试

- HSL 适配层测试：工厂配置映射、连接与读取结果转换可通过替身验证。
- 驱动测试：连接配置、地址解析、连接测试、长连接、断开、读取成功和单点失败隔离。
- DevAgent 测试：默认注册、Manifest 一致性和任务执行。
- 完整验证：`dotnet test collector/InduForge.Collector.slnx` 与 `dotnet build collector/InduForge.Collector.slnx -c Release`。

## 前后端接入

- Data Service 的 capability 契约增加可选 `resources.serialPorts`。
- RTU 连接 Schema 继续作为字段和默认值来源。
- 新建连接与连接编辑器都接收当前调试 Agent，并为 `portName` 提供资源选项。
- 未选择 Agent、Agent 离线或没有枚举到串口时，用户仍可手动输入串口名称。

## 不包含

- Modbus 写入。
- 串口热插拔推送。
- 自动重连。
- Runtime Collector 接入。
- 数据库结构变更或迁移逻辑。
