# runtime/node_agent_front 协作规则

## 模块定位

- `runtime/node_agent_front` 是节点本地管理前端，负责节点状态、配置、日志与本地运维界面展示。

## 进入前先看

- `runtime/node_agent_front/package.json`
- `runtime/node_agent_front/src/`
- `docs/node_agent_front/README.md`

## 开发约束

- 运行时为 Vue 3 + Vite + Pinia + Element Plus，但依赖和脚本链路独立维护，按本模块现状处理。
- 本地前端只消费 NodeAgent 暴露的标准状态字段，不自行发明新的协议语义。
- 列表、状态与日志展示优先保证稳定和可诊断性，避免过度重构页面结构。
- 变更前先确认是否会影响节点本地部署、状态轮询和日志查看链路。

## 禁止事项

- 不要把平台管理端逻辑复制到本地前端。
- 不要把运行时页面渲染能力混入 NodeAgent Front。
- 不要提交 `dist/`、日志或本地缓存。

## 验证命令

- `pnpm --dir runtime/node_agent_front build`

## 相关契约

- `docs/node_agent_front/README.md`
- `docs/contracts/runtime-health-status-contract.md`
