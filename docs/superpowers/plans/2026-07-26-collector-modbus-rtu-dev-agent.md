# Modbus RTU 采集调试代理实现计划

> **面向 AI 代理的工作者：** 使用 executing-plans 按顺序实现；每项完成后立即运行最贴近变更面的验证。未经用户明确要求不创建 Git 提交。

**目标：** 在 DevAgent 中完成 Modbus RTU 的连接测试、长连接、断开和点位读取闭环。

**架构：** HSL 适配层新增协议专属 RTU 客户端，独立 Modbus RTU 驱动负责配置和地址语义，DevAgent 通过既有注册表和会话管理器调用。

**技术栈：** .NET 10、C#、HslCommunication 12.9.1、xUnit、WinForms DevAgent。

---

## 文件结构

- 修改 `collector/src/InduForge.Collector.Adapters.Hsl/HslContracts.cs`：增加 RTU 参数、枚举、接口和工厂。
- 创建 `collector/src/InduForge.Collector.Adapters.Hsl/HslModbusRtuClient.cs`：包装 HSL `ModbusRtu`。
- 创建 `collector/src/InduForge.Collector.Drivers.ModbusRtu/`：实现独立 RTU 驱动。
- 创建 `collector/tests/InduForge.Collector.Drivers.ModbusRtu.Tests/`：覆盖配置、地址和驱动行为。
- 修改 `collector/src/InduForge.Collector.DevAgent/`：注册驱动并嵌入 Manifest。
- 修改 `collector/tests/InduForge.Collector.DevAgent.Tests/`：验证默认能力与任务执行。
- 修改 `runtime/collector_protocols/modbus.rtu/`：发布真实能力和连接字段。
- 修改 `collector/InduForge.Collector.slnx`：纳入新项目与测试项目。
- 修改 `data_service/internal/service/collector_dev_service.go`：透传 capability 主机资源快照。
- 修改 `datacenter/src/components/collector-workbench/`：为串口字段提供可刷新、可手输的下拉框。

## 任务 1：扩展 HSL RTU 契约

- [ ] 在 `HslContracts.cs` 增加 `HslSerialParity`、`HslSerialStopBits`、`HslModbusRtuClientOptions`、`IHslModbusRtuClient`、`IHslModbusRtuClientFactory` 和默认工厂。
- [ ] 创建 `HslModbusRtuClient.cs`，使用 `SerialPortInni(portName, baudRate, dataBits, stopBits, parity)` 初始化串口。
- [ ] 将超时映射到 HSL 接收超时，将 `HslDataFormat` 映射到 HSL `DataFormat`。
- [ ] 复用 Modbus TCP 已有数据类型读取规则，保持错误码一致。
- [ ] 运行 `dotnet test collector/tests/InduForge.Collector.Adapters.Hsl.Tests/InduForge.Collector.Adapters.Hsl.Tests.csproj`，预期通过。

## 任务 2：实现 Modbus RTU 配置和地址

- [ ] 创建驱动项目并引用 Contracts 与 HSL Adapters。
- [ ] 实现 `ModbusRtuConnectionOptions.Parse`，覆盖必填项、枚举、超时和数据格式校验。
- [ ] 实现 `ModbusRtuAddress.Parse`，与 Modbus TCP 保持相同站号、区域、位索引、类型和元素数量约束。
- [ ] 添加配置与地址单元测试。
- [ ] 运行 `dotnet test collector/tests/InduForge.Collector.Drivers.ModbusRtu.Tests/InduForge.Collector.Drivers.ModbusRtu.Tests.csproj`，预期通过。

## 任务 3：实现 RTU 驱动和会话

- [ ] 实现 `ModbusRtuDriver`，声明 `connection.test`、`connection.open`、`connection.close`、`point.read`。
- [ ] 实现 `ModbusRtuConnectionSession`，在单个串口连接上串行执行读取并正确关闭释放。
- [ ] 连接测试使用临时客户端，完成后始终关闭和释放。
- [ ] 批量读取逐点隔离错误并按请求 `Key` 返回结果。
- [ ] 使用假客户端覆盖成功、失败、会话和取消路径。
- [ ] 再次运行 RTU 驱动测试，预期通过。

## 任务 4：注册 DevAgent 能力

- [ ] 在 DevAgent 项目中引用 RTU 驱动项目并嵌入 RTU Manifest。
- [ ] 在 `DriverRegistry.CreateDefault` 注册 `ModbusRtuDriver`。
- [ ] 在 `collector/InduForge.Collector.slnx` 中加入驱动和测试项目。
- [ ] 更新 DevAgent 测试，断言 RTU 描述符和 Manifest 完全一致。
- [ ] 运行 `dotnet test collector/tests/InduForge.Collector.DevAgent.Tests/InduForge.Collector.DevAgent.Tests.csproj`，预期通过。

## 任务 5：同步运行时协议定义

- [ ] 在连接 Schema 增加 `dataFormat`，默认 `ABCD`。
- [ ] 将 RTU Manifest operations 更新为四项真实操作。
- [ ] 将 `platforms.devAgent` 更新为 `windows-x64`，保持 Schema 版本 `1`。
- [ ] 检查 Manifest、驱动描述符和 Data Service 现有地址校验规则一致。

## 任务 6：完整验证

- [ ] 运行 `dotnet test collector/InduForge.Collector.slnx`，预期全部通过。
- [ ] 运行 `dotnet build collector/InduForge.Collector.slnx -c Release`，预期零警告零错误。
- [ ] 运行 `git diff --check`，预期无空白错误。
- [ ] 检查 `git status --short`，确认只有本次 RTU 相关文件。

## 任务 7：接入串口资源快照

- [ ] DevAgent 根据 Manifest transport 识别串口驱动，注册和心跳时重新枚举串口。
- [ ] 枚举结果稳定排序、去重；异常写入 Agent 日志并返回空数组。
- [ ] Data Service capability 增加可选 `resources.serialPorts`，继续存入现有 JSONB 数组。
- [ ] 前端 Agent Schema 解析串口资源，新建与编辑表单为 `portName` 提供可刷新下拉框和手动输入。
- [ ] 运行 `go test ./internal/service`、`pnpm --dir datacenter typecheck` 和 `pnpm --dir datacenter build`。
