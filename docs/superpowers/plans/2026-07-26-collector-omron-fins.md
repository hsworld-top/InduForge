# 欧姆龙 FINS TCP/UDP 实现计划

> **面向 AI 代理的工作者：** 在当前会话按步骤实现并逐项验证。

**目标：** 完成欧姆龙 FINS TCP/UDP 的 DevAgent、Data Service 和 Datacenter 通用接入。

**架构：** 两个 Driver ID 共用一个 HSL FINS 适配器和一个驱动项目；TCP 使用真实长连接，UDP 用 CPU 状态探测建立逻辑会话。

**技术栈：** .NET 10、HslCommunication 12.9.1、Go、JSON Schema、Vue 3、TypeScript。

---

### 任务 1：HSL FINS 适配器

**文件：**
- 修改：`collector/src/InduForge.Collector.Adapters.Hsl/HslContracts.cs`
- 创建：`collector/src/InduForge.Collector.Adapters.Hsl/HslOmronFinsClient.cs`
- 修改：`collector/tests/InduForge.Collector.Adapters.Hsl.Tests/HslPublicContractTests.cs`

- [ ] 定义 FINS 传输、PLC 类型、连接参数、读取请求和客户端工厂契约。
- [ ] 实现 TCP 连接、UDP CPU 状态探测、关闭和各平台类型读取。
- [ ] 验证公共契约测试和 Adapter 构建通过。

### 任务 2：FINS 驱动与测试

**文件：**
- 创建：`collector/src/InduForge.Collector.Drivers.OmronFins/`
- 创建：`collector/tests/InduForge.Collector.Drivers.OmronFins.Tests/`
- 修改：`collector/InduForge.Collector.slnx`

- [ ] 实现 TCP/UDP 描述符、配置解析、地址解析、会话和驱动。
- [ ] 覆盖默认值、非法配置、长度上限、连接失败和单点失败隔离。
- [ ] 运行 FINS 驱动测试。

### 任务 3：DevAgent 注册

**文件：**
- 修改：`collector/src/InduForge.Collector.DevAgent/DriverRegistry.cs`
- 修改：`collector/src/InduForge.Collector.DevAgent/InduForge.Collector.DevAgent.csproj`
- 修改：`collector/tests/InduForge.Collector.DevAgent.Tests/DriverRegistryTests.cs`

- [ ] 注册两个驱动并嵌入对应 Manifest。
- [ ] 验证默认注册、传输类型和 Manifest 一致性。

### 任务 4：协议目录与保存校验

**文件：**
- 创建：`runtime/collector_protocols/omron.fins-tcp/`
- 创建：`runtime/collector_protocols/omron.fins-udp/`
- 修改：`data_service/internal/service/collector_point_service.go`
- 修改：`data_service/internal/service/collector_point_service_test.go`
- 修改：`data_service/internal/collectorprotocol/catalog_test.go`
- 修改：`data_service/internal/service/collector_catalog_service_test.go`

- [ ] 添加两个 Schema 版本为 1 的协议目录。
- [ ] 增加地址、数据类型和读取长度保存校验及地址文本格式化。
- [ ] 运行协议目录和服务测试。

### 任务 5：前端名称与全链路验证

**文件：**
- 修改：`datacenter/src/components/collector-workbench/collector-workbench-model.ts`

- [ ] 添加 FINS TCP/UDP 中文名称。
- [ ] 运行 Datacenter 类型检查、测试和生产构建。
- [ ] 运行 Adapter、FINS Driver、DevAgent 测试和 Collector Release 构建。
- [ ] 运行 `git diff --check`。
