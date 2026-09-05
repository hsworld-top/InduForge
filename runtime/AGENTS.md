# runtime 协作规则

## 模块定位

- `runtime/` 包含节点代理、本地管理前端、工程运行组件、工业采集与页面 SDK；各模块职责独立。

## 按任务查阅

以下路径和命令均从仓库根目录解析，只进入任务涉及的模块。

| 任务                            | 入口                                                                  |
| ------------------------------- | --------------------------------------------------------------------- |
| 节点初始化、部署执行与状态回传  | `runtime/node_agent/AGENTS.md`                                        |
| 节点本地管理界面                | `runtime/node_agent_front/AGENTS.md`                                  |
| 数据、计算与报警运行逻辑        | `runtime/runtime_engine/README.md`                                    |
| 工业协议采集与 WAL              | `runtime/collector_engine/go.mod`、`docs/02-系统设计/工业采集架构.md` |
| 发布工程动态 API 与实时订阅     | `runtime/runtime_api/README.md`                                       |
| 工程静态入口与 Runtime API 代理 | `runtime/project_gateway/README.md`                                   |
| 页面运行与预览 SDK              | `runtime/web-sdk/README.md`、`runtime/web-sdk/package.json`           |

- 节点命令、部署或执行边界变更：查 `docs/04-契约与规范/跨模块契约/node-agent-runtime-protocol.md` 对应章节。
- 被托管组件的健康或状态字段变更：查 `docs/04-契约与规范/跨模块契约/runtime-health-status-contract.md`，同时定位实际生产者和消费者。

## 开发约束

- 运行时协议、状态字段和探活语义必须保持当前契约一致；变更时同步更新消费者和正式契约，不把节点侧链路改成平台侧实现风格。
- 沿用各现有组件的职责边界；只有任务涉及新增宿主或执行形态时才扩展相应架构，不能把已有独立组件视为尚未实施的预留模块。

## 禁止事项

- 不要把 Designer 编辑逻辑或平台管理逻辑下沉到 `runtime/`。
- 不要在 `runtime/` 目录堆放与当前实现无关的大段过程文档。

## 按影响面验证

- NodeAgent 与本地前端使用各自子模块规则中的入口。
- RuntimeEngine、Runtime API、Project Gateway：按模块选用根脚本 `pnpm test:runtime-engine`、`pnpm test:runtime-api`、`pnpm test:project-gateway`；各模块也提供对应的 `build:` 和 `lint:` 根脚本。
- Collector：在 `runtime/collector_engine` 模块内按需执行 `go test`、`go build` 或 `go vet`，模块全量范围使用 `./...`；该模块没有 Makefile。
- SDK：`pnpm test:runtime-web-sdk`。
- 局部变更优先缩小到受影响的包或测试；契约、共享代码或跨模块链路变化时增加对应消费者验证。以上入口无需全部运行，纯文档改动不运行业务测试。
