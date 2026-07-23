# HSL 通用适配层与 Modbus TCP 驱动实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 保留 OPC UA 官方 SDK 实现，解除 DevAgent 的 OPC UA 硬编码，并基于 HslCommunication 完成 Modbus TCP 连接测试、长连接和批量读取闭环。

**架构：** 平台契约改为透传原始连接配置和结构化点位读取请求，具体驱动负责协议解析。OPC UA 继续使用 OPC Foundation SDK；Modbus TCP 通过独立 HSL 适配项目访问 HslCommunication，HSL 类型不进入平台契约或 DevAgent 响应。

**技术栈：** .NET 10、C#、HslCommunication、OPC Foundation SDK、xUnit、JSON Schema、动态端口 TCP 测试服务。

---

## 文件结构

### 修改

- `collector/src/InduForge.Collector.Contracts/DriverRequests.cs`：通用连接和结构化读取请求。
- `collector/src/InduForge.Collector.Contracts/DriverResults.cs`：按点位 Key 关联的读取结果。
- `collector/src/InduForge.Collector.Drivers.OpcUa/*.cs`：适配新契约，保持原有行为。
- `collector/src/InduForge.Collector.DevAgent/CollectorTaskExecutor.cs`：删除 OPC UA 配置和地址硬编码。
- `collector/src/InduForge.Collector.DevAgent/DriverRegistry.cs`：注册 Modbus TCP。
- `collector/src/InduForge.Collector.DevAgent/InduForge.Collector.DevAgent.csproj`：引用驱动并嵌入清单。
- `collector/InduForge.Collector.slnx`：加入新增项目。
- `runtime/collector_protocols/modbus.tcp/*.json`：同步真实配置和能力。
- `data_service/internal/service/collector_point_service.go`：保存阶段 Modbus 语义校验。

### 创建

- `collector/src/InduForge.Collector.Adapters.Hsl/`：唯一直接引用 HslCommunication 的适配项目。
- `collector/src/InduForge.Collector.Drivers.ModbusTcp/`：Modbus TCP 驱动项目。
- `collector/tests/InduForge.Collector.Adapters.Hsl.Tests/`：HSL 适配层测试。
- `collector/tests/InduForge.Collector.Drivers.ModbusTcp.Tests/`：地址、连接、读取和模拟服务测试。

## 任务 1：重构通用连接与读取契约

**文件：**
- 修改：`collector/src/InduForge.Collector.Contracts/DriverRequests.cs`
- 修改：`collector/src/InduForge.Collector.Contracts/DriverResults.cs`
- 修改：`collector/tests/InduForge.Collector.Contracts.Tests/DriverContractsTests.cs`

- [ ] **步骤 1：编写失败的契约测试**

```csharp
var profile = new ConnectionProfile(
    "modbus",
    JsonSerializer.SerializeToElement(new { host = "127.0.0.1", port = 502 }),
    JsonSerializer.SerializeToElement(new { }));
Assert.Equal("127.0.0.1", profile.Config.GetProperty("host").GetString());

var request = new ReadRequest([
    new PointReadRequest(
        "point-1",
        JsonSerializer.SerializeToElement(new { station = 1, area = "holdingRegister", address = 10 }),
        "int16",
        1,
        JsonSerializer.SerializeToElement(new { }))
]);
Assert.Equal("point-1", request.Points[0].Key);
```

- [ ] **步骤 2：运行测试验证失败**

运行：`dotnet test collector/tests/InduForge.Collector.Contracts.Tests/InduForge.Collector.Contracts.Tests.csproj`

预期：旧 `ConnectionProfile` 或 `ReadRequest` 签名导致编译失败。

- [ ] **步骤 3：实现最终契约**

```csharp
public sealed record ConnectionProfile(string ProtocolFamily, JsonElement Config, JsonElement Secrets);

public sealed record PointReadRequest(
    string Key,
    JsonElement Address,
    string DataType,
    int ElementCount,
    JsonElement ReadOptions);

public sealed record ReadRequest(IReadOnlyList<PointReadRequest> Points);
```

```csharp
public sealed record PointReadValue(
    string Key,
    bool Succeeded,
    object? Value,
    string? DataType,
    string Quality,
    DateTimeOffset? SourceTimestamp,
    DateTimeOffset? ServerTimestamp,
    string? ErrorCode,
    string? ErrorMessage);
```

- [ ] **步骤 4：运行测试验证通过**

运行步骤 2 命令，预期全部 PASS。

- [ ] **步骤 5：提交**

```powershell
git add -- collector/src/InduForge.Collector.Contracts collector/tests/InduForge.Collector.Contracts.Tests
git commit -m "refactor(collector): 通用化连接与点位读取契约"
```

## 任务 2：适配 OPC UA 与 DevAgent 通用转发

**文件：**
- 修改：`collector/src/InduForge.Collector.Drivers.OpcUa/OpcUaDriver.cs`
- 修改：`collector/src/InduForge.Collector.Drivers.OpcUa/OpcUaConnectionFactory.cs`
- 修改：`collector/src/InduForge.Collector.Drivers.OpcUa/OpcUaConnectionSession.cs`
- 修改：`collector/src/InduForge.Collector.DevAgent/CollectorTaskExecutor.cs`
- 修改：`collector/tests/InduForge.Collector.Drivers.OpcUa.Tests/*.cs`
- 修改：`collector/tests/InduForge.Collector.DevAgent.Tests/CollectorTaskExecutorTests.cs`

- [ ] **步骤 1：编写失败测试**

DevAgent 测试传入 Modbus 风格配置，验证测试驱动收到原始字段：

```csharp
Assert.Equal("127.0.0.1", state.Profile!.Config.GetProperty("host").GetString());
Assert.Equal("holdingRegister", state.Request!.Points[0].Address.GetProperty("area").GetString());
Assert.Equal("float32", state.Request.Points[0].DataType);
Assert.Equal(2, state.Request.Points[0].ElementCount);
```

OPC UA 测试改为使用原始 JSON Config 和结构化 `address.nodeId`。

- [ ] **步骤 2：运行测试验证失败**

```powershell
dotnet test collector/tests/InduForge.Collector.Drivers.OpcUa.Tests/InduForge.Collector.Drivers.OpcUa.Tests.csproj
dotnet test collector/tests/InduForge.Collector.DevAgent.Tests/InduForge.Collector.DevAgent.Tests.csproj
```

预期：旧连接字段和 `NodeIds` 引用导致编译失败。

- [ ] **步骤 3：移动 OPC UA 配置解析**

在 OPC UA 驱动内部读取 `profile.Config`，构造 Endpoint、SecurityMode、SecurityPolicy、认证方式和超时；保留现有错误码。

- [ ] **步骤 4：通用化任务创建**

```csharp
new PointReadRequest(
    pointId,
    address.Clone(),
    point.GetProperty("dataType").GetString() ?? string.Empty,
    point.TryGetProperty("elementCount", out var count) ? count.GetInt32() : 1,
    point.TryGetProperty("readOptions", out var options)
        ? options.Clone()
        : JsonSerializer.SerializeToElement(new { }))
```

删除 `CreateProfile()` 的 OPC UA 协议判断、`OpcUaEndpointBuilder` 和任务层 `OpcUaAddressMapper` 调用。结果按 `Key` 回填。

- [ ] **步骤 5：运行步骤 2 测试并提交**

预期全部 PASS，然后提交：

```powershell
git add -- collector/src/InduForge.Collector.Drivers.OpcUa collector/src/InduForge.Collector.DevAgent collector/tests/InduForge.Collector.Drivers.OpcUa.Tests collector/tests/InduForge.Collector.DevAgent.Tests
git commit -m "refactor(collector): 移除 Agent 的 OPC UA 协议硬编码"
```

## 任务 3：建立 HSL 适配项目

**文件：**
- 创建：`collector/src/InduForge.Collector.Adapters.Hsl/InduForge.Collector.Adapters.Hsl.csproj`
- 创建：`collector/src/InduForge.Collector.Adapters.Hsl/HslReadResult.cs`
- 创建：`collector/src/InduForge.Collector.Adapters.Hsl/HslModbusTcpClient.cs`
- 创建：`collector/tests/InduForge.Collector.Adapters.Hsl.Tests/InduForge.Collector.Adapters.Hsl.Tests.csproj`
- 创建：`collector/tests/InduForge.Collector.Adapters.Hsl.Tests/HslModbusTcpClientTests.cs`
- 修改：`collector/InduForge.Collector.slnx`

- [ ] **步骤 1：创建失败测试**

```csharp
Assert.DoesNotContain(
    typeof(HslReadResult).Assembly.GetExportedTypes()
        .SelectMany(type => type.GetProperties())
        .Select(property => property.PropertyType.Namespace),
    value => value?.StartsWith("HslCommunication", StringComparison.Ordinal) == true);
```

并验证连接失败、读取失败和释放都映射为适配层自有结果。

- [ ] **步骤 2：运行测试验证失败**

运行：`dotnet test collector/tests/InduForge.Collector.Adapters.Hsl.Tests/InduForge.Collector.Adapters.Hsl.Tests.csproj`

预期：项目或类型不存在。

- [ ] **步骤 3：添加依赖与最小实现**

```xml
<PackageReference Include="HslCommunication" Version="12.9.1" />
```

`HslModbusTcpClient` 公开连接、关闭和标准类型读取方法；内部统一处理 `OperateResult.IsSuccess`，对外不返回 HSL 类型。

- [ ] **步骤 4：运行测试并提交**

预期全部 PASS，然后提交：

```powershell
git add -- collector/src/InduForge.Collector.Adapters.Hsl collector/tests/InduForge.Collector.Adapters.Hsl.Tests collector/InduForge.Collector.slnx
git commit -m "feat(collector): 增加 HSL 通信适配层"
```

## 任务 4：实现 Modbus TCP 配置与地址解析

**文件：**
- 创建：`collector/src/InduForge.Collector.Drivers.ModbusTcp/InduForge.Collector.Drivers.ModbusTcp.csproj`
- 创建：`collector/src/InduForge.Collector.Drivers.ModbusTcp/ModbusTcpConnectionOptions.cs`
- 创建：`collector/src/InduForge.Collector.Drivers.ModbusTcp/ModbusTcpAddress.cs`
- 创建：`collector/tests/InduForge.Collector.Drivers.ModbusTcp.Tests/InduForge.Collector.Drivers.ModbusTcp.Tests.csproj`
- 创建：`collector/tests/InduForge.Collector.Drivers.ModbusTcp.Tests/ModbusTcpAddressTests.cs`
- 修改：`collector/InduForge.Collector.slnx`

- [ ] **步骤 1：编写失败测试**

覆盖空主机、非法端口、非法 `dataFormat`、四种区域、站号和地址边界、寄存器位索引、Coil 禁止位索引、寄存器 bool 必须提供位索引。

```csharp
var parsed = ModbusTcpAddress.Parse(new PointReadRequest(
    "point-1",
    JsonSerializer.SerializeToElement(new
    {
        station = 1,
        area = "holdingRegister",
        address = 100,
        bitIndex = 3,
    }),
    "bool",
    1,
    JsonSerializer.SerializeToElement(new { })));
Assert.Equal("s=1;x=3;100", parsed.HslAddress);
```

- [ ] **步骤 2：运行测试验证失败**

运行：`dotnet test collector/tests/InduForge.Collector.Drivers.ModbusTcp.Tests/InduForge.Collector.Drivers.ModbusTcp.Tests.csproj`

预期：项目或解析类型不存在。

- [ ] **步骤 3：实现配置与地址解析**

默认端口 `502`、连接超时 `5000ms`、数据格式 `ABCD`。解析结果包含站号、区域、起始地址、位索引和 HSL 地址，异常使用稳定 `MODBUS_*` 错误码。

- [ ] **步骤 4：运行测试并提交**

预期全部 PASS，然后提交：

```powershell
git add -- collector/src/InduForge.Collector.Drivers.ModbusTcp collector/tests/InduForge.Collector.Drivers.ModbusTcp.Tests collector/InduForge.Collector.slnx
git commit -m "feat(collector): 实现 Modbus TCP 配置与地址解析"
```

## 任务 5：实现 Modbus TCP 连接与批量读取

**文件：**
- 创建：`collector/src/InduForge.Collector.Drivers.ModbusTcp/ModbusTcpDriver.cs`
- 创建：`collector/src/InduForge.Collector.Drivers.ModbusTcp/ModbusTcpConnectionSession.cs`
- 创建：`collector/tests/InduForge.Collector.Drivers.ModbusTcp.Tests/ModbusTcpDriverTests.cs`
- 创建：`collector/tests/InduForge.Collector.Drivers.ModbusTcp.Tests/ModbusTcpTestServer.cs`

- [ ] **步骤 1：编写失败测试和模拟服务**

模拟服务使用 `TcpListener(IPAddress.Loopback, 0)` 动态端口，实现功能码 `01`、`02`、`03`、`04`。测试连接成功、不可达端口、四种区域、不同站号、整数、浮点、字符串、字节、数组和无效地址隔离。

- [ ] **步骤 2：运行测试验证失败**

运行任务 4 的测试命令，预期驱动入口和会话不存在。

- [ ] **步骤 3：实现驱动与会话**

```csharp
Operations:
[
    DriverOperations.ConnectionTest,
    DriverOperations.ConnectionOpen,
    DriverOperations.ConnectionClose,
    DriverOperations.PointRead,
]
```

会话按站号和区域分组，优先合并连续地址；批量段失败时逐点降级读取，结果按 `Key` 返回。

- [ ] **步骤 4：运行测试并提交**

预期全部 PASS，然后提交：

```powershell
git add -- collector/src/InduForge.Collector.Drivers.ModbusTcp collector/tests/InduForge.Collector.Drivers.ModbusTcp.Tests
git commit -m "feat(collector): 完成 Modbus TCP 连接与批量读取"
```

## 任务 6：注册驱动并同步协议能力

**文件：**
- 修改：`collector/src/InduForge.Collector.DevAgent/DriverRegistry.cs`
- 修改：`collector/src/InduForge.Collector.DevAgent/InduForge.Collector.DevAgent.csproj`
- 修改：`collector/tests/InduForge.Collector.DevAgent.Tests/DriverRegistryTests.cs`
- 修改：`runtime/collector_protocols/modbus.tcp/connection.schema.json`
- 修改：`runtime/collector_protocols/modbus.tcp/manifest.json`

- [ ] **步骤 1：编写失败测试**

```csharp
var descriptor = DriverRegistry.CreateDefault().Describe("modbus.tcp");
Assert.Contains(DriverOperations.ConnectionOpen, descriptor.Operations);
Assert.Contains(DriverOperations.PointRead, descriptor.Operations);
Assert.DoesNotContain(DriverOperations.DeviceBrowse, descriptor.Operations);
```

- [ ] **步骤 2：运行 DevAgent 测试验证失败**

运行：`dotnet test collector/tests/InduForge.Collector.DevAgent.Tests/InduForge.Collector.DevAgent.Tests.csproj`

预期：`modbus.tcp` 未注册。

- [ ] **步骤 3：更新 Schema、Manifest 和注册表**

Schema 增加 `dataFormat=ABCD/BADC/CDAB/DCBA`。Manifest 写入四项真实 operation、`windows-x64` 和真实数据类型，递增 Schema Version。

- [ ] **步骤 4：运行测试并提交**

预期全部 PASS，然后提交：

```powershell
git add -- collector/src/InduForge.Collector.DevAgent collector/tests/InduForge.Collector.DevAgent.Tests runtime/collector_protocols/modbus.tcp
git commit -m "feat(collector): 注册 Modbus TCP 调试驱动"
```

## 任务 7：补充 Data Service 保存校验

**文件：**
- 修改：`data_service/internal/service/collector_point_service.go`
- 修改：`data_service/internal/service/collector_point_service_test.go`

- [ ] **步骤 1：编写失败测试**

覆盖 Coil 带 `bitIndex`、寄存器 bool 缺少 `bitIndex`、站号越界、地址越界和合法 Holding Register。

- [ ] **步骤 2：运行目标测试验证失败**

```powershell
Push-Location data_service
go test ./internal/service -run CollectorPoint
Pop-Location
```

预期：非法组合当前仍被接受。

- [ ] **步骤 3：实现保存语义校验**

在地址格式化前调用 Modbus 专属校验器，返回 `BAD_REQUEST` 和明确中文消息；不增加数据库迁移或启动兼容逻辑。

- [ ] **步骤 4：运行测试并提交**

预期全部 PASS，然后提交：

```powershell
git add -- data_service/internal/service/collector_point_service.go data_service/internal/service/collector_point_service_test.go
git commit -m "fix(data-service): 校验 Modbus 变量地址组合"
```

## 任务 8：完整验证并进入 S7

- [ ] **步骤 1：运行 Collector 全量测试和 Release 构建**

```powershell
pnpm dotnet:test:collector
dotnet build collector/InduForge.Collector.slnx -c Release
```

预期全部成功，无编译错误。

- [ ] **步骤 2：运行 Data Service 全量测试和构建**

```powershell
Push-Location data_service
go test ./...
go build ./...
Pop-Location
```

预期全部成功。

- [ ] **步骤 3：检查能力一致性与工作区**

```powershell
git diff --check
git status --short
git log -8 --oneline
```

确认 Manifest 与驱动 Descriptor 的 ID、版本、Schema 和 operations 完全一致，Modbus 不展示设备浏览。

- [ ] **步骤 4：进入 S7 接入**

Modbus TCP 验收完成后，复用通用连接、结构化读取契约和 HSL 适配层，单独形成 Siemens S7 TCP 规格；首期实现连接测试、长连接和地址型批量读取。
