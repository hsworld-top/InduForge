# runtime/node_agent 协作规则

## 模块定位

- `runtime/node_agent` 是节点侧执行器后端，负责节点初始化、命令轮询、部署落地、状态回传与宿主预留接口。

## 进入前先看

- `runtime/node_agent/go.mod`
- `runtime/node_agent/Makefile`
- `runtime/node_agent/cmd/`
- `runtime/node_agent/internal/`
- `docs/03-模块设计/node_agent/README.md`

## 开发约束

- 运行时为 Go，优先沿用 `cmd/`、`internal/`、本地存储与执行器的现有分层。
- 节点控制面优先保证稳定，`/health`、`/status` 与部署、回滚状态语义要保持一致。
- 运行时目录、版本切换与错误上报规则要满足节点侧长期运行需求；变更时同步更新相关消费者和正式契约。
- 变更执行链路时，优先检查命令轮询、状态回传和本地持久化是否一起受影响。

## 禁止事项

- 不要提前重构 NodeAgent 技术栈。
- 不要把 RuntimeEngine 或平台侧治理逻辑硬编码进 NodeAgent。
- 不要提交本地数据库、日志或运行产物。

## 验证命令

- `make -C runtime/node_agent test`
- `make -C runtime/node_agent vet`

## 相关契约

- `docs/03-模块设计/node_agent/README.md`
- `docs/04-契约与规范/跨模块契约/node-agent-runtime-protocol.md`
- `docs/04-契约与规范/跨模块契约/runtime-health-status-contract.md`
