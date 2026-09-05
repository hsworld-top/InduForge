# Designer 模块设计

Designer 是 AI 页面开发工作台。页面布局、路由和组件由 AI 直接维护 Vue/React Vite 源码；Designer
负责组织 Pi Web、页面预览、2D/3D 产物、源码编辑器和工程上下文摘要。

## 模块职责

- 左侧只承载 Pi Web。
- 中间菜单提供“页面 / 2D / 3D / 编辑器”四个工作台视图。
- 预览视图承载 Vite iframe、设备尺寸、刷新、全屏、新标签打开和进程控制。
- 2D/3D 视图按公开场景契约展示产物卡片，并在顶部提供对应 HT 编辑器入口。
- Designer 只管理场景卡片和公开契约；工程级场景资源库、上传和 Symbol/Component 编辑均在 HT 内完成。
- 编辑器视图承载精简 code-server，使用其原生文件、搜索、Git、运行、扩展和终端。
- 底部展示上下文版本、2D/3D 场景数、数据点数和更新时间。
- 向 `dev_ide` 发送受控的 2D/3D 标签打开请求。
- 空工作区首次进入时展示四套官方 Vite 模板，初始化完成前不创建任何工程 iframe。

## 边界

- Designer 不维护页面 Schema 或页面解释器。
- Designer 不读取或修改 Pi Web、Preview、code-server 和 HT 编辑器的跨域 DOM。
- AI 读取平台生成的 HT 场景契约与数据点上下文，不直接操作 HT 编辑器。
- HT 只通过短期场景/资源会话访问 PostgreSQL 与 MinIO，不读取工程工作区中的物理资源目录。
- Runtime SDK 是工程页面访问鉴权、权限、数据点和场景能力的唯一前端抽象。
- Preview Control 只管理开发态 `5173` 端口，不参与工程发布和运行站点。

## 嵌入方式

| 关系                                | 方式                                              |
| ----------------------------------- | ------------------------------------------------- |
| `dev_ide -> Designer`               | Wujie，注入工程上下文与工具回调。                 |
| `Designer -> Pi Web`                | 原生 iframe。                                     |
| `Designer -> Vite Preview`          | 原生 iframe。                                     |
| `Designer -> code-server`           | 原生 iframe。                                     |
| `dev_ide -> InduForge 2D/3D Studio` | 独立保活 iframe 标签；浏览器不感知内部 Provider。 |

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

根目录前端启动命令默认使用 `frontend-linux` 模式：Designer 通过工作区 API 读取 `dev_core`
返回的四服务地址，再由 `workspace-runtime.ts` 改写为本地工作区代理地址。显式本地开发模式
才从根 `.env` 读取四个固定地址，连接本机映射的工作区服务。

生产环境使用控制面返回的受控服务 URL，页面不得自行推导端口或域名。各模式的配置和代理边界见
[工程开发工作空间网络架构](../../02-系统设计/工程开发工作空间网络架构.md)。代码提供了这些路径，
目标环境的 Cookie、跨域、WebSocket 和 iframe 加载仍需分别验收。

## 开发链路

```text
用户首次选择 Vue/React 与 JavaScript/TypeScript 模板
-> Preview Control 初始化 /workspace 和本地 Git
-> 启动受控 Vite Runner
用户描述需求
-> Pi Web 修改 /workspace 源码
-> Vite HMR 更新 Preview iframe
-> 用户可切换设备、刷新或在新标签中打开 Preview 并使用浏览器开发者工具
-> code-server 随时编辑文件、运行 Git 或终端命令
```

发布链路仍为 `pnpm build -> dist -> client-assets -> project-nginx`，不携带 Pi Web、code-server、
Preview Control 或开发工作空间能力。
