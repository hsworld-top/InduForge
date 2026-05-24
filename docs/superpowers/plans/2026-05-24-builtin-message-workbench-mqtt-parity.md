# IF消息库工作台 MQTT 化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 IF消息库工作台从表单验证面板改造成接近 MQTT 工作台的 Topic 工作台，支持右键查看消息和变量实时监控弹窗。

**架构：** 后端在内置消息发布路径上增加进程内实时事件广播，并通过 preview socket 增加 `builtin:message:*` 事件。前端复用 MQTT 工作台的信息架构，但使用 IF消息库 topic/variable API 和内置 socket 事件。

**技术栈：** Go data_service、Socket.IO、Vue 3、Element Plus、现有 datacenter workbench 组件。

---

### 任务 1：补内置消息实时事件

**文件：**
- 修改：`data_service/internal/service/builtin_runtime_service.go`
- 修改：`data_service/internal/http/socket/preview_socket_server.go`
- 修改：`data_service/internal/app/server.go`
- 测试：`data_service/internal/service/builtin_runtime_service_test.go`

- [ ] 在 `BuiltinRuntimeService` 中增加 `SubscribeMessageTopic`，返回取消函数。
- [ ] `PublishMessage` 成功后广播 `{connectionId,runtimeKey,topic,fullTopic,payload,qos,timestamp}`。
- [ ] `PreviewSocketServer` 监听 `builtin:message:subscribe/unsubscribe`，按 topic id 注册 socket 订阅并转发 `builtin:message`。
- [ ] 运行 `go test ./...` 与 `go build ./...`。

### 任务 2：重做 IF消息库工作台

**文件：**
- 修改：`datacenter/src/components/access-source/workbench/BuiltinMessageWorkbench.vue`
- 修改：`datacenter/src/api/data.api.ts`
- 可新增：`datacenter/src/components/access-source/workbench/BuiltinMessageViewer.vue`
- 可新增：`datacenter/src/components/access-source/workbench/BuiltinMessageMonitor.vue`

- [ ] 左侧改为 Topic 列表/树，包含筛选、新建、刷新、右键菜单。
- [ ] 右键菜单支持查看消息、变量管理、实时监控、详情、删除占位禁用项。
- [ ] 中间使用 Tab：消息查看复用流式消息列表，变量管理围绕当前 topic 展示。
- [ ] 弹窗监控按当前 topic 的变量定义解析最近消息 payload 并展示当前值。
- [ ] 发布面板保留在工作台内，发布后实时消息 viewer 与监控同步更新。

### 任务 3：浏览器验收

**文件：**
- 不改文件。

- [ ] 创建/选择 IF消息库 topic。
- [ ] 右键 topic 打开“查看消息”。
- [ ] 打开“实时监控”弹窗。
- [ ] 发布 JSON payload，确认消息 Tab 和变量监控刷新。
- [ ] 运行 `pnpm --filter datacenter typecheck`、`lint`、`build`。
