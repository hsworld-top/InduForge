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
`pnpm dev`。

### 2.3 控制台

底部只保留“控制台”。点击后向 Preview Origin 发送固定消息，公共模板在开发态动态加载并打开
完整 Eruda。再次点击同一入口会折叠控制台。

Eruda 使用 Preview 页面内部的固定底部抽屉承载：默认高度约为可视区域的三分之一，顶部拖拽条
支持鼠标、触控笔和键盘调整，高度限制在 `160px` 到视口高度的 `80%`。抽屉折叠后不销毁
Eruda 实例，Console、Elements、Network 和 Resources 继续调试当前 Preview 页面。

模板校验 Origin 与 `document.referrer` 一致，并允许消息来自直接父窗口或受控的 Wujie 宿主 frame。
Designer 只发送固定的 `PREVIEW_DEVTOOLS_COMMAND/toggle` 消息，不读取 Preview DOM。

Eruda 不显示默认悬浮按钮，不进入生产构建。终端、Problems 和 Output 使用 code-server 原生能力。

## 3. 2D/3D 产物视图

- 2D 和 3D 使用独立视图，按 `kind` 过滤公开场景契约并以卡片展示名称、场景 ID、页面路由、
  嵌入方式、公开能力数、数据点引用数和契约版本。
- 空列表展示明确的未创建状态，并保留顶部编辑器入口。
- 编辑器入口只发送工程 ID 与 `2d | 3d` 目标，由 `dev_ide` 打开或激活独立保活标签。
- 当前公开场景契约未声明 Viewer URL 或缩略图 URL，Designer 不根据场景 ID、路由或文件目录
  猜测预览地址。后续补齐运行 Viewer/缩略图契约后，卡片预览区再加载真实产物画面。

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

## 6. Wujie 工具协议

Designer 只向宿主发送工程 ID 与 `2d | 3d` 目标，不传递工具 URL。`dev_ide` 校验消息类型、目标
白名单和当前工程后创建或激活独立标签。源码编辑不再创建宿主标签，统一使用右侧 code-server。
