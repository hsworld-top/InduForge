# OPC UA 真实地址空间浏览实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 OPC UA 工作台 `/browse` 从已建模变量投影替换为真实 OPC UA Server 地址空间浏览。

**架构：** 在 `data_service` 服务层新增独立 OPC UA 浏览适配器，`ProtocolDevSessionService` 负责 session 校验和 modeled 标记，适配器负责连接、认证、安全配置、递归 browse 与 DataType 读取。

**技术栈：** Go、`github.com/gopcua/opcua`、现有 `apperrors` 与服务层测试。

---

### 任务 1：增加 OPC UA 浏览适配器

**文件：**
- 创建：`data_service/internal/service/protocol_dev_opcua_browser.go`
- 修改：`data_service/go.mod`
- 修改：`data_service/go.sum`

- [ ] 增加 `github.com/gopcua/opcua` 依赖。
- [ ] 定义 `ProtocolDevOpcuaBrowser` 接口和真实实现。
- [ ] 从连接配置解析 endpoint、安全策略、安全模式、认证、证书材料与 browse 限制。
- [ ] 从 `ObjectsFolder` 递归 browse，并读取变量 `DataType`。
- [ ] browse 超深或超量时写入 diagnostics。

### 任务 2：接入会话服务

**文件：**
- 修改：`data_service/internal/service/protocol_dev_session_service.go`
- 修改：`data_service/internal/app/server.go`

- [ ] `ProtocolDevSessionService` 增加 browser 依赖。
- [ ] `NewProtocolDevSessionService` 增加 browser 参数。
- [ ] `BrowseOpcua` 改为调用真实 browser。
- [ ] 已建模变量只用于按 NodeId 设置 `Modeled=true`。
- [ ] 删除旧占位 diagnostics 文案。
- [ ] 在 app 装配中注入真实 browser。

### 任务 3：测试与验证

**文件：**
- 修改：`data_service/internal/service/protocol_dev_session_service_test.go`
- 创建：`data_service/internal/service/protocol_dev_opcua_browser_test.go`

- [ ] 用 fake browser 验证 `BrowseOpcua` 返回真实浏览节点并标记 modeled。
- [ ] 验证缺少 browser 时返回初始化错误。
- [ ] 覆盖浏览配置解析、DataType 映射和 NodeId 去重。
- [ ] 运行 `go test ./internal/service`。
- [ ] 运行 `go test ./...`。

### 任务 4：提交推送

**文件：**
- 本次修改文件集合。

- [ ] 检查 `git status --short`，确认无无关产物。
- [ ] 使用中文 commit message 提交。
- [ ] 推送 `master`。
