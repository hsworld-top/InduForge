# runtime 共享规则

## 模块定位

- `runtime/` 承接运行时共享层，当前主要覆盖 NodeAgent 及其前端，后续可继续承接 `engine_*` 共享规则。

## 处理该模块时先看哪里

- `runtime/node_agent/`
- `runtime/node_agent_front/`
- `.planning/docs/process/runtime/README_nodeagent.md`
- `docs/contracts/node-agent-runtime-protocol.md`
- `docs/contracts/runtime-health-status-contract.md`

## 必须遵守

- 运行时协议、探活字段和状态定义以 `docs/contracts/` 为准。
- 当前阶段优先保证 NodeAgent 与本地前端稳定，不提前扩独立运行时宿主。
- `/health` 与 `/status` 相关约定保持兼容，避免新旧状态语义漂移。

## 不要做

- 不要在运行时目录下补写与当前实现无关的大段规划文档。
- 不要把 Designer 编辑逻辑或平台管理逻辑混入 `runtime/`。

## 验证命令

- 后端子模块参考 `runtime/node_agent/AGENTS.md`
- 前端子模块参考 `runtime/node_agent_front/AGENTS.md`

## 相关参考文档

- `.planning/docs/process/runtime/README_nodeagent.md`
- `.planning/ai-packages/runtime_node_agent-ai-package.md`
- `.planning/ai-packages/runtime_engine-ai-package.md`
