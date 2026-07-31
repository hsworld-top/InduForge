# Allen-Bradley EtherNet/IP CIP 实现计划

> **面向 AI 代理的工作者：** 在当前会话按步骤实现并逐项验证。

**目标：** 完成 Allen-Bradley Logix EtherNet/IP CIP 的 DevAgent、Data Service 和 Datacenter 接入。

**架构：** HSL Adapter 隔离 `AllenBradleyNet`，独立驱动负责配置、标签地址、会话和逐点失败隔离，前后端继续由协议目录驱动。

**技术栈：** .NET 10、HslCommunication 12.9.1、Go、JSON Schema、Vue 3、TypeScript。

---

### 任务 1：HSL Adapter

- [ ] 定义连接参数、读取请求、客户端和工厂契约。
- [ ] 实现连接、关闭、标签类型读取和错误映射。
- [ ] 添加公共契约测试并验证 Adapter。

### 任务 2：驱动与测试

- [ ] 创建 EtherNet/IP 驱动项目和测试项目。
- [ ] 实现配置、路由、标签地址、会话和描述符。
- [ ] 覆盖默认配置、路由校验、长度上限、连接失败和单点失败隔离。

### 任务 3：DevAgent 与协议目录

- [ ] 注册驱动并嵌入 Manifest。
- [ ] 新增连接、地址和 UI Schema。
- [ ] 验证 Manifest、传输和默认注册一致。

### 任务 4：Data Service 与前端

- [ ] 增加保存校验、地址文本格式化和目录数量测试。
- [ ] 增加协议族和驱动中文名称。
- [ ] 运行 Go、TypeScript、Vitest、生产构建和 Collector Release 构建。
