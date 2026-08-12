# runtime 协作规则

## 模块定位

- `runtime/` 是运行时相关模块的公共入口，当前重点覆盖 `node_agent` 与 `node_agent_front`。

## 进入前先看

- `runtime/node_agent/`
- `runtime/node_agent_front/`
- `docs/04-契约与规范/跨模块契约/node-agent-runtime-protocol.md`
- `docs/04-契约与规范/跨模块契约/runtime-health-status-contract.md`

## 开发约束

- 运行时协议、状态字段和探活语义必须保持当前契约一致；变更时同步更新消费者和正式契约，不把节点侧链路改成平台侧实现风格。
- 处理 `runtime/` 下任务时，先判断是后端执行器还是本地前端，再进入对应子模块规则。
- 当前阶段优先保证 NodeAgent 及其本地前端稳定，不提前扩展新的独立运行时宿主。

## 禁止事项

- 不要把 Designer 编辑逻辑或平台管理逻辑下沉到 `runtime/`。
- 不要在 `runtime/` 目录堆放与当前实现无关的大段过程文档。

## 验证命令

- 后端子模块参考 `runtime/node_agent/AGENTS.md`
- 前端子模块参考 `runtime/node_agent_front/AGENTS.md`

## 相关契约

- `docs/04-契约与规范/跨模块契约/node-agent-runtime-protocol.md`
- `docs/04-契约与规范/跨模块契约/runtime-health-status-contract.md`
