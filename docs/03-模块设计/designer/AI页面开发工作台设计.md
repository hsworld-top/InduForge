# AI 页面开发工作台设计

## 1. 布局

```text
┌──────────────────────┬─┬──────────────────────────────────────────┐
│                      │页│ Vite Preview                             │
│                      │2D│ 2D 产物卡片                              │
│ Pi Web               │3D│ 3D 产物卡片                              │
│                      │编│ 精简 code-server                         │
├──────────────────────┴─┴──────────────────────────────────────────┤
│ 上下文版本 | 2D | 3D | 数据点 | 更新时间                          │
└───────────────────────────────────────────────────────────────────┘
```

- 桌面端 AI 区默认宽度受限于 `420-560px`，分隔线支持拖拽与键盘调整。
- 空间不足时切换为 AI/工作台单面板页签，不同时压缩两个 iframe。
- 中间菜单固定为页面、2D、3D、编辑器；左侧 Pi Web 不再承载 HT 工具按钮。
- 页面和编辑器 iframe 初始化后持续存活，切换视图不销毁实例。
- 2D、3D 视图按公开场景契约展示产物卡片，顶部按钮通过 `dev_ide` 打开对应 HT 编辑器。

### 1.1 首次模板选择

空工作区首次进入时，Designer 使用 Preview Control 查询初始化状态，并展示 Vite 官方模板：

- Vue + JavaScript
- Vue + TypeScript
- React + JavaScript
- React + TypeScript

初始化完成前不创建 Pi Web、Preview 和 code-server iframe，避免 AI 或编辑器提前写入文件。模板只能
选择一次；非空且没有平台初始化标记的工作区必须报错，不得覆盖已有内容。

## 2. 预览视图

### 2.1 工具栏

工具栏提供设备模式、Vite 状态、启动/停止、重启、刷新、新窗口和全屏。

设备按钮点击后弹出菜单：网页、平板 `768x1024`、手机 `390x844`。网页模式填满可用区域，固定设备模式保持真实 CSS viewport 尺寸，空间不足时只缩放视觉画布。
固定设备保持真实 CSS viewport，空间不足时缩放外层画布；切换设备不重载 iframe。

### 2.2 Vite 生命周期

Preview Control 独立运行于容器 `5174`，Vite 使用 `5173`。状态包括：

- `running/managed`：平台启动的 Vite。
- `running/external`：用户从 code-server 终端启动的服务。
- `starting`、`stopping`、`stopped`、`error`。

Designer 在预览激活时每 3 秒轮询，状态变化时每秒轮询，切到编辑器后降低为每 10 秒。
停止 Vite 后显示平台停止态，不能暴露浏览器连接失败页。重启会清理当前 `5173` 进程并统一执行
平台 Vite Runner。用户在终端自行执行 `pnpm dev` 时识别为 `external`。

### 2.3 新标签调试

“新窗口”打开 Designer 自有的 `/designer/preview?projectId=...` 宿主页。宿主按当前
登录身份重新查询受控 Preview URL，不接收页面传入的任意 URL，并复用当前工作空间的
HttpOnly 会话。用户在新标签中使用浏览器原生开发者工具进行 Console、Elements、
Network 和存储调试。

Designer 不再注入 Eruda、调试桥或跨 iframe 控制台消息。平台 Vite Runner 只负责加载工程 Vite
配置并固定监听地址和端口；终端、Problems 和 Output 使用 code-server 原生能力。原始 Vite
地址直接打开时不承诺 2D/3D Viewer 会话解析能力。

## 3. 2D/3D 产物视图

- 2D 和 3D 使用独立视图，按 `kind` 过滤公开场景契约并以卡片展示名称、说明、已提交
  revision；仅在存在数据点引用时展示关联数量。
- 空列表展示明确的未创建状态；用户只填写工程内唯一的场景名称，平台生成不可见的稳定场景 ID，再由短期编辑会话打开对应 Provider 编辑器。
- 卡片提供打开编辑器、场景设置和删除；场景设置维护名称、说明、页面参数、场景事件和
  页面可调用命令，并只读展示 Provider 在 commit 时提取的数据点引用。对象和数组通过递归类型
  编辑器生成 JSON Schema Draft 2020-12，不暴露原始 JSON 编辑框。
- 工程级图片、字体、模型、材质、Symbol 和 Component 资源库只在 HT 内管理，不在 Designer 建设独立资源页。
- 资源替换后既有场景继续固定旧内部代次；用户在当前场景执行“一键更新”后草稿版本递增，再次 commit 才进入新 revision。
- 工程页面通过 `<induforge-scene-2d>` 和 `<induforge-scene-3d>` 加载已提交 revision，只传
  平台生成的稳定 `sceneId`。组件支持 `setParams()`、`scene-event` 和 `invoke()`，不暴露 Provider URL、
  逻辑路径或引擎名称。草稿预览仍在 Provider 编辑器内完成，不进入页面 Viewer。

## 4. 编辑器视图

code-server 保留 Activity Bar、Explorer、Search、SCM、Run、Extensions、Monaco、Terminal、
Problems、Output 和状态栏；隐藏菜单栏、命令中心、布局控制、默认辅助侧栏、欢迎页、推荐、更新和信任提示。Accounts/Manage 使用 code-server 4.131.0 官方 Activity Bar 行为，不通过 DOM 或 CSS 强制隐藏。

裁剪通过固定默认设置与官方启动参数完成，不 Fork code-server、不修改 DOM、不维护工作台扩展。

## 5. 保活与失败隔离

- Pi Web、Preview 和 code-server iframe 相互独立加载。
- Preview 与 code-server 只隐藏不销毁。
- Vite 停止或异常不影响 Pi Web、code-server 和容器健康。
- Preview Control 不可用时禁用进程按钮，Preview 与编辑器其他能力保持可用。
- 上下文或场景摘要部分失败时保留其他已成功数据。

## 6. InduForge AI 嵌入协议

左侧 AI 使用仓库内 `designer/pi-web` 维护的 `@induforge/pi-web`，基于上游 Pi Web `v0.8.8`。嵌入版固定工作目录为
`/workspace`，不提供品牌顶栏、项目选择、语言/主题选择、Plugins、系统提示词查看和 Worktree。

保留模型与 Provider 配置、用户 API Key、会话及会话分叉、文件浏览、Diff、工具调用和 Project
Skills。Skills 只允许读写 `/workspace/.pi/skills/`；用户及工程插件不自动发现，仅允许镜像内
`/opt/induforge/pi/extensions` 的平台只读 Extension。

Designer 在 AI iframe URL 上只附加 `induforgeProjectId`，服务地址仍完整使用
`services.ai.url`，不得推导端口或路径。Pi iframe 加载、发送 READY 或 IDE 设置变化时，Designer
向 Pi Origin 发送：

```ts
interface InduForgePiContext {
  type: 'INDUFORGE_PI_CONTEXT'
  version: 1
  projectId: string
  workspaceRoot: '/workspace'
  locale: 'zh' | 'en'
  theme: 'light' | 'dark'
}
```

Pi Web 加载配置后发送 `INDUFORGE_PI_READY`。双方同时校验消息类型、版本、工程 ID、精确
Origin 和 iframe/父窗口引用。语言与主题只跟随 `dev_ide`，Pi Web 不提供选择入口，也不持久化
独立偏好。

## 7. Wujie 工具协议

Designer 只向宿主发送工程 ID 与 `2d | 3d` 目标，不传递工具 URL。`dev_ide` 校验消息类型、目标
白名单和当前工程后创建或激活独立标签。源码编辑不再创建宿主标签，统一使用右侧 code-server。
