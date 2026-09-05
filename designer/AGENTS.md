# designer 协作规则

## 模块定位

- `designer` 是 AI 页面开发工作台，负责承载 Pi Web、Vite 预览外壳、精简 code-server、工程上下文摘要和 2D/3D 工具入口。
- Vite Preview 与 code-server 使用两个持续存活的 iframe；预览设备、刷新和进程启停由 Designer 管理。
- 工程页面是标准 Vue 或 React Vite 源码，源码和路由是页面行为的唯一事实。

## 按需参考

以下路径均相对仓库根目录，仅在任务涉及对应内容时查阅。

- 模块职责与工作台交互：`docs/03-模块设计/designer/README.md`、`docs/03-模块设计/designer/AI页面开发工作台设计.md`。
- 工作区容器与进程：`designer/code-workspace/README.md`。
- 工作区地址与代理：`docs/02-系统设计/工程开发工作空间网络架构.md`、`designer/src/ui/workspaces/code/workspace-runtime.ts`。
- 工程源码与发布产物：`docs/04-契约与规范/跨模块契约/工程前端源码与构建产物契约.md`。

## 开发约束

- 运行时为 Vue 3 + TypeScript + Vite + Pinia + Element Plus。
- Designer 只编排工作台界面，不复制 Pi Web、code-server 或 HT 编辑器内部功能；预览外壳只管理受控 Vite URL 和 Preview Control。
- 工作区 URL 通过 `codeWorkspaceApi` 读取 `dev_core` 返回的服务地址，统一由 `workspace-runtime.ts` 处理开发环境地址与代理；不在页面中推导容器端口或路径。
- 打开 2D、3D 时发送受控宿主消息，由 `dev_ide` 校验工程和权限后处理。
- Pi Web、Vite Preview 和 code-server 使用原生 iframe，不作为 Wujie 子应用；Designer 自身由 `dev_ide` 通过 Wujie 承载。
- 工程、租户、编辑代次、语言和主题取当前宿主上下文；浏览器认证使用同源 `HttpOnly` Cookie，不在前端存储或传递 JWT。
- 预览控制只调用 Preview Control 固定 REST 接口；浏览器调试通过受控 Preview URL 在新标签中完成。
- 标签切换必须保留 iframe 状态；只有用户关闭标签或权限失效时才销毁实例。
- AI 只读取 HT 场景公开契约和数据点上下文，不直接操作 HT 编辑器或修改平台生成的只读上下文。
- 复杂状态变更统一走 store action，跨 iframe 消息必须校验来源、类型和 `projectId`。

## 禁止事项

- 不新增与工程前端源码并行的页面配置模型。
- 不在 Designer 中实现页面运行解释器、HT 私有数据编辑或数据中心业务逻辑。
- 不把长期 token、模型密钥或 refresh token 放入 iframe URL。
- 不把 HT 场景、模型和材质重复复制到工程前端构建产物。

## 验证命令

- 从仓库根目录按变更面选用，不要求每次全跑；测试命令可追加相关测试文件路径。
- TypeScript 改动：`pnpm --dir designer typecheck`。
- `pnpm --dir designer test`
- `pnpm --dir designer build`
