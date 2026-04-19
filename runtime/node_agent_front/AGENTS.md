# runtime/node_agent_front 模块规则

## 模块定位

- `runtime/node_agent_front` 是节点本地管理前端，负责节点状态、配置、日志和本地运维界面展示。

## 处理该模块时先看哪里

- `runtime/node_agent_front/package.json`
- `runtime/node_agent_front/src/`
- `docs/node_agent_front/README.md`
- `.planning/docs/process/runtime/README_nodeagent.md`

## 必须遵守

- 技术栈为 Vue 3 + Vite + Pinia + Element Plus。
- 本地前端只消费 NodeAgent 暴露的标准状态字段，不自行发明新的协议语义。
- 列表与状态展示优先保证稳定和可诊断性，避免过度重构页面结构。

## 不要做

- 不要把平台管理端逻辑复制到本地前端。
- 不要把运行时页面渲染能力混入 NodeAgent Front。
- 不要提交 `dist/`、日志或本地缓存。

## 验证命令

- `pnpm --dir runtime/node_agent_front build`

## 相关参考文档

- `docs/node_agent_front/README.md`
- `.planning/ai-packages/tasks/runtime_node_agent.task.md`
