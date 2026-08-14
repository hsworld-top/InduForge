# designer 协作规则

## 模块定位

- `designer` 是 AI 页面开发工作台，负责承载 Pi Web、Vite 预览外壳、精简 code-server、工程上下文摘要和 2D/3D 工具入口。
- Vite Preview 与 code-server 使用两个持续存活的 iframe；预览设备、刷新、进程启停和开发控制台由 Designer 管理。
- 工程页面是标准 Vue 或 React Vite 源码，源码和路由是页面行为的唯一事实。

## 进入前先看

- `designer/package.json`
- `designer/src/`
- `designer/code-workspace/README.md`
- `docs/03-模块设计/designer/README.md`
- `docs/03-模块设计/designer/AI页面开发工作台设计.md`
- `docs/04-契约与规范/跨模块契约/工程前端源码与构建产物契约.md`

## 开发约束

- 运行时为 Vue 3 + TypeScript + Vite + Pinia + Element Plus。
- Designer 只编排工作台界面，不复制 Pi Web、code-server 或 HT 编辑器内部功能；预览外壳只管理受控 Vite URL 和 Preview Control。
- 子应用 URL 由 `dev_core` 返回并由 `dev_ide` 注入，Designer 不拼接容器端口；开发环境可直接读取四个固定 URL。
- 打开 2D、3D 时发送受控宿主消息，由 `dev_ide` 校验工程和权限后处理。
- Pi Web、Vite Preview 和 code-server 使用原生 iframe，不作为 Wujie 子应用；Designer 自身由 `dev_ide` 通过 Wujie 承载。
- 预览控制只调用 Preview Control 固定 REST 接口，开发控制台只向 Preview Origin 发送固定 `postMessage`。
- 标签切换必须保留 iframe 状态；只有用户关闭标签或权限失效时才销毁实例。
- AI 只读取 HT 场景公开契约和数据点上下文，不直接操作 HT 编辑器或修改平台生成的只读上下文。
- 复杂状态变更统一走 store action，跨 iframe 消息必须校验来源、类型和 `projectId`。

## 禁止事项

- 不新增与工程前端源码并行的页面配置模型。
- 不在 Designer 中实现页面运行解释器、HT 私有数据编辑或数据中心业务逻辑。
- 不把长期 token、模型密钥或 refresh token 放入 iframe URL。
- 不把 HT 场景、模型和材质重复复制到工程前端构建产物。

## 验证命令

- `pnpm --dir designer typecheck`
- `pnpm --dir designer test`
- `pnpm --dir designer build`

## 相关文档

- `docs/03-模块设计/designer/README.md`
- `docs/03-模块设计/designer/AI页面开发工作台设计.md`
- `docs/02-系统设计/工程开发工作空间网络架构.md`
- `docs/04-契约与规范/跨模块契约/工程前端源码与构建产物契约.md`
