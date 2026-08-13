# Designer 模块设计

Designer 是 AI 页面开发工作台。页面布局、路由和组件由 AI 直接维护 Vue/Vite 源码；Designer
负责组织 Pi Web、页面预览、2D/3D 产物、源码编辑器和工程上下文摘要。

## 模块职责

- 左侧只承载 Pi Web。
- 中间菜单提供“页面 / 2D / 3D / 编辑器”四个工作台视图。
- 预览视图承载 Vite iframe、设备尺寸、刷新、全屏、进程控制和开发控制台。
- 2D/3D 视图按公开场景契约展示产物卡片，并在顶部提供对应 HT 编辑器入口。
- 编辑器视图承载精简 code-server，使用其原生文件、搜索、Git、运行、扩展和终端。
- 底部展示上下文版本、2D/3D 场景数、数据点数和更新时间。
- 向 `dev_ide` 发送受控的 2D/3D 标签打开请求。

## 边界

- Designer 不维护页面 Schema 或页面解释器。
- Designer 不读取或修改 Pi Web、Preview、code-server 和 HT 编辑器的跨域 DOM。
- AI 读取平台生成的 HT 场景契约与数据点上下文，不直接操作 HT 编辑器。
- Runtime SDK 是工程页面访问鉴权、权限、数据点和场景能力的唯一前端抽象。
- Preview Control 只管理开发态 `5173` 端口，不参与工程发布和运行站点。

## 嵌入方式

| 关系 | 方式 |
| --- | --- |
| `dev_ide -> Designer` | Wujie，注入工程上下文与工具回调。 |
| `Designer -> Pi Web` | 原生 iframe。 |
| `Designer -> Vite Preview` | 原生 iframe。 |
| `Designer -> code-server` | 原生 iframe。 |
| `dev_ide -> HT 2D/3D` | 独立保活 iframe 标签。 |

Preview 与 code-server iframe 初始化后始终存在。页面、2D、3D、编辑器切换只改变可见性和交互状态，不能修改
`src` 或销毁实例，以保留 HMR、页面状态、Canvas、编辑缓冲区和终端进程。

## 服务契约

Designer 消费严格的四服务工作空间状态：

```ts
services: {
  ai: WorkspaceServiceState
  code: WorkspaceServiceState
  preview: WorkspaceServiceState
  previewControl: WorkspaceServiceState
}
```

开发环境从根 `.env` 读取四个地址并直连 WSL 容器。生产环境由 `dev_core` 和 Traefik 返回受控
URL，前端不得推导端口或域名。

## 开发链路

```text
用户描述需求
-> Pi Web 修改 /workspace 源码
-> Vite HMR 更新 Preview iframe
-> 用户可切换设备、刷新或打开 Eruda
-> code-server 随时编辑文件、运行 Git 或终端命令
```

发布链路仍为 `pnpm build -> dist -> client-assets -> project-nginx`，不携带 Pi Web、code-server、
Preview Control 或 Eruda。
