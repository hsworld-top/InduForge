# runtime/node_agent 协作规则

## 模块定位

- `runtime/node_agent` 是节点侧执行器后端，负责节点初始化、命令轮询、部署落地、状态回传与受限宿主操作。

## 按任务查阅

以下路径和命令均从仓库根目录解析，只读取任务涉及的文件与章节。

- 模块职责与安装边界：`docs/03-模块设计/node_agent/README.md`。
- 命令、部署与进程托管：`runtime/node_agent/internal/ops/` 对应实现与测试，以及 `docs/04-契约与规范/跨模块契约/node-agent-runtime-protocol.md` 对应章节。
- 宿主操作：`runtime/node_agent/internal/hostd/` 中对应计划校验与执行实现。
- 节点本地 API：`runtime/node_agent/internal/web/handler/routes.go` 及对应处理器；组件健康探测另查 `docs/04-契约与规范/跨模块契约/runtime-health-status-contract.md`。
- 依赖、启动装配或校验入口：按需查 `runtime/node_agent/go.mod`、`runtime/node_agent/cmd/`、`runtime/node_agent/Makefile`。

## 开发约束

- 运行时为 Go，优先沿用入口、本地存储与执行器的现有分层。
- NodeAgent 自身接口为 `/health` 与 `/api/v1/node/status`；被托管 Runtime 组件使用各自的 `/health` 与 `/api/v1/status`。节点状态与组件状态不是同一接口或模型，部署、回滚和探测变更应核对实际消费者。
- 运行时目录、版本切换与错误上报规则要满足节点侧长期运行需求；变更时同步更新相关消费者和正式契约。
- 变更执行链路时，优先检查命令轮询、状态回传和本地持久化是否一起受影响。

## 禁止事项

- 任务未涉及技术栈演进时，不扩大为 NodeAgent 技术栈重构。
- 不要把 RuntimeEngine 或平台侧治理逻辑硬编码进 NodeAgent。
- 宿主操作只接受受限声明式计划，不把中心输入扩展为任意 shell、路径、参数或环境变量执行通道。
- 不要提交本地数据库、日志或运行产物。

## 按影响面验证

- 局部逻辑优先通过 `test-pkg` 指定受影响包，例如 `make -C runtime/node_agent test-pkg PKG=./internal/ops`；跨包执行链路变更需要模块回归时使用 `make -C runtime/node_agent test`。
- 静态检查入口为 `make -C runtime/node_agent vet`；宿主权限、平台差异或部署链路变更，还需针对实际影响补充对应验证。
- 按需选择入口，必要验证通过后不重复全量检查；纯文档改动不运行业务测试。
