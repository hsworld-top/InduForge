# IDE 管理端概览

## 1. 模块定位

`dev_ide` 是平台管理、运维和工程工具宿主前端。它负责平台管理、工程入口、发布部署、节点状态，
以及 Designer、数据中心和 HT 编辑器等子应用的标签化接入。

`dev_ide` 不实现 AI 对话、页面源码编辑、Vite 预览、HT 场景编辑或数据域业务逻辑。

## 2. 核心能力

- 平台、租户、用户和工程管理入口。
- 发布、部署、版本、节点和运行状态管理。
- 获取受控子应用 URL，并完成工程身份、鉴权、主题和语言上下文注入。
- 管理工程工具标签、重复激活、实例保活、关闭和顶层恢复。
- 汇总子应用初始化失败、权限失效和服务不可用状态。

“运行管理”当前用于创建运行集群记录、接入任务、审批节点、查看心跳和触发 Demo 工作负载操作；
页面中的生产/高可用标签不表示 K3s 或生产运行已交付。实际范围、安装包与验收见
[运维体系实现与验收](../../06-运维与安全/运维体系实现与验收.md)。

## 3. 工程工具标签

同一工程的顶层标签键固定为：

```text
{projectId}:ai
{projectId}:2d
{projectId}:3d
```

分别对应 AI 页面开发工作台、HT 2D 编辑器和 HT 3D 编辑器。重复打开请求必须激活已有标签，
不能创建重复实例。

标签标题包含工程名称，例如“产线监控 · AI开发”“产线监控 · 2D”和“产线监控 · 3D”。标签
切换时隐藏但不销毁实例，只有用户关闭标签、工程被删除或权限失效时才销毁。

源码、Vite 预览、版本管理和终端不再创建 `{projectId}:code` 顶层标签。Designer 右侧使用
持续存活的 Preview 与 code-server iframe 分别承载预览和源码开发能力。

## 4. 子应用打开协议

Designer 可以通过 Wujie props 注入的 `onOpenWorkspace` 回调向宿主发送：

```ts
interface WorkspaceOpenRequest {
  type: 'WORKSPACE_OPEN_REQUEST'
  projectId: string
  target: '2d' | '3d'
}
```

`dev_ide` 收到请求后必须：

1. 校验回调来源对应当前 Designer 实例和当前工程。
2. 校验当前用户对工程及目标工具的访问权限。
3. 复用或创建稳定标签。
4. 从控制面获取环境无关、短期授权的访问 URL。
5. 激活标签并保持其他未关闭实例状态。

子应用不能要求宿主接受任意 URL，也不能自行拼接容器端口。

## 5. Wujie 与 iframe 边界

`dev_ide` 通过 Wujie 承载 Designer。Wujie props 可注入：

- 当前用户、租户、工程 ID 和工程名称。
- 主题、语言和平台能力配置。
- `onOpenWorkspace` 等宿主白名单回调。

Designer 内部的 Pi Web 和 code-server 使用原生 iframe，不作为 Wujie 子应用：

```text
dev_ide
└─ Wujie：Designer
   ├─ iframe：Pi Web
   └─ iframe：code-server
```

原因是 Pi Web 和 code-server 都是独立完整站点；其中 code-server 依赖 WebSocket、Webview、
Service Worker、剪贴板、下载和多层 iframe。`dev_ide` 不对这些运行时做 DOM、location、fetch
或资源代理。

## 6. 鉴权与上下文注入

- Wujie Designer 通过受控 props 获取平台上下文，不在 URL 中传递长期凭据。
- Pi Web、code-server 和 Vite Preview 使用工程级短期工作空间票据或由网关建立的 HttpOnly 会话。
- Pi Web 和 code-server 必须使用受控独立 Origin，并通过 CSP `frame-ancestors` 限制平台嵌入来源。
- iframe 消息必须校验 `origin`、消息类型、工程 ID、实例 ID 和会话状态。
- 顶层刷新时由 `dev_ide` 根据已授权标签状态恢复，不依赖敏感查询参数。
- 工作空间票据过期后重新授权，不把长期 token、refresh token 或模型密钥写入 iframe URL。
- 浏览器会话由 IDE 宿主统一续租；首次 `401` 触发单次刷新并重试原请求，Wujie 子应用复用同一刷新任务，失败后才统一退出登录。

## 7. 与 Designer 的边界

Designer 负责：

- Pi Web、Preview 和 code-server iframe 的工作台布局与保活。
- Preview/编辑器菜单、设备模式和 Vite 生命周期控制。
- 上下文摘要以及 HT 2D/3D 工具入口。

`dev_ide` 只负责承载 Designer Wujie 实例及处理 HT 标签打开请求，不复制工作台布局，不单独
打开 code-server，也不转发任意 VS Code 命令。

具体交互见 [AI 页面开发工作台设计](../designer/AI页面开发工作台设计.md)。

## 8. 质量关注点

- 同一顶层标签键只能存在一个活动实例。
- 标签切换不能导致 Pi 会话、Preview 页面、code-server 编辑缓冲区或 HT 未保存状态丢失。
- 工程 A 的消息和 URL 不能打开或控制工程 B 的工具。
- 短期票据过期后应重新授权，不把长期凭据写入 iframe URL。
- 子应用加载失败不能阻塞其他管理页面和工程标签。
- 顶层标签恢复不能创建第二套 Preview 或 code-server iframe。

## 9. 关联文档

- [Designer 模块概览](../designer/README.md)
- [AI 页面开发工作台设计](../designer/AI页面开发工作台设计.md)
- [工程开发工作空间网络架构](../../02-系统设计/工程开发工作空间网络架构.md)
- [数据中心概览](../datacenter/README.md)
- [dev_core 后端概览](../dev_core/README.md)
