# runtime/node_agent_front 协作规则

## 模块定位

- `runtime/node_agent_front` 是节点本地管理前端，负责节点状态、配置、日志与本地运维界面展示。

## 按任务查阅

以下路径和命令均从仓库根目录解析，只读取任务涉及的文件与章节。

- 页面、状态与日志交互：定位 `runtime/node_agent_front/src/` 中对应视图、状态存储或 API 文件。
- 模块职责与本地管理边界：`docs/03-模块设计/node_agent_front/README.md`。
- 状态或接口语义：核对 `runtime/node_agent_front/src/api/nodeApi.js` 和 `runtime/node_agent/internal/web/handler/routes.go`；展示被托管组件状态时，再查 `docs/04-契约与规范/跨模块契约/runtime-health-status-contract.md`。
- 依赖或构建、校验脚本：`runtime/node_agent_front/package.json`。

## 开发约束

- 运行时为 Vue 3 + Vite + Pinia + Element Plus，Node 依赖由根目录 pnpm workspace 统一管理，模块脚本通过根 workspace 调度。
- 本地前端只消费 NodeAgent 暴露的标准状态字段，不自行发明新的协议语义。
- 列表、状态与日志展示优先保证稳定和可诊断性，避免过度重构页面结构。
- 变更前先确认是否会影响节点本地部署、状态轮询和日志查看链路。

## 禁止事项

- 不要把平台管理端逻辑复制到本地前端。
- 不要把运行时页面渲染能力混入 NodeAgent Front。
- 不要提交 `dist/`、日志或本地缓存。

## 按影响面验证

- JS/Vue 逻辑的静态检查入口为 `pnpm lint:node-agent-front`；页面、依赖或构建改动需要打包验证时使用 `pnpm build:node-agent-front`。
- 初始化、状态轮询或日志链路变化，使用已运行的服务验证受影响交互与失败状态；无可用服务时按根规则说明未验证项。
- 按需选择入口，不要求每次全部执行；纯文档改动不运行业务构建。
