# runtime/node_agent 模块规则

## 模块定位

- `runtime/node_agent` 是节点侧执行器后端，负责节点初始化、命令轮询、部署落地、状态回传与运行时宿主预留接口。

## 处理该模块时先看哪里

- `runtime/node_agent/go.mod`
- `runtime/node_agent/cmd/`
- `runtime/node_agent/internal/`
- `runtime/node_agent/Makefile`
- `docs/node_agent/README.md`
- `docs/contracts/node-agent-runtime-protocol.md`

## 必须遵守

- 技术栈为 Go，目录约定以 `cmd/`、`internal/`、`data/`、`logs/` 为主。
- 保持节点控制面稳定，运行时托管相关能力优先遵守现有启动协议与状态协议。
- `/health`、`/status` 与部署/回滚状态语义必须和契约文档一致。
- 运行时目录、版本切换、错误上报规则应兼容未来独立宿主接入。

## 不要做

- 不要提前重构 NodeAgent 技术栈。
- 不要在当前阶段把 RuntimeEngine 执行逻辑硬编码进 NodeAgent。
- 不要提交本地数据库、日志或运行产物。

## 验证命令

- `make -C runtime/node_agent test`
- `make -C runtime/node_agent vet`

## 相关参考文档

- `docs/node_agent/README.md`
- `docs/contracts/node-agent-runtime-protocol.md`
- `docs/ai-packages/tasks/runtime_node_agent.task.md`
