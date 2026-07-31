# Mitsubishi MC 3E TCP 实现计划

**目标：** 完成 Mitsubishi MC 3E Binary TCP 的 DevAgent 与通用前后端目录接入。

**架构：** HSL 协议专属适配器 + 独立工业驱动 + Manifest/Schema 自动接入。

**技术栈：** .NET 10、HslCommunication 12.9.1、xUnit、Go 协议目录、Vue Schema 表单。

## 任务

- [ ] 扩展 HSL Mitsubishi MC 契约并实现客户端。
- [ ] 实现连接配置、原生地址解析、会话和驱动。
- [ ] 添加配置、地址、读取隔离和连接生命周期测试。
- [ ] 创建 `mitsubishi.mc-3e-tcp` Manifest 与 Schema。
- [ ] 注册 DevAgent、解决方案和测试项目。
- [ ] 更新前端驱动中文标签。
- [ ] 运行目标测试、协议目录测试、前端类型检查和 Release 构建。
