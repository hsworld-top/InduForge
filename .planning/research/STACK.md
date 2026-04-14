# STACK

## 研究范围
- 目标：确定 InduForge 在 2026 年继续演进时的“标准工程栈”与升级边界。
- 范围：`dev_core`、`dev_ide`、`datacenter`、`designer`、`runtime/node_agent`、`runtime/node_agent_front`。
- 结论口径：优先沿用仓库已落地栈，做一致性收敛与风险治理，不做大规模重构迁栈。

## 推荐主栈（沿用并收敛）

### 后端与控制面
| 组件 | 当前依赖 | 建议 |
|---|---|---|
| Node.js 运行时 | 各前后端模块均要求 `>=18` | 统一 LTS 版本线（CI 与本地一致） |
| HTTP 框架 | `express@^5.1.0` | 保持 Express 5，重点治理中间件与响应规范 |
| ORM | `sequelize@^6.37.7` | 保持 Sequelize，禁止 `sequelize.sync()`，统一 init 脚本 |
| 认证与安全 | `jsonwebtoken`、`helmet`、`express-rate-limit` | 保持现栈，移除开发旁路依赖，补鉴权回归 |
| 实时通道 | `socket.io@^4.8.3`、`mqtt@^5.14.1` | 保持现栈，治理重复逻辑并固化消息契约 |

### 前端
| 应用 | 当前栈 | 建议 |
|---|---|---|
| `dev_ide` / `datacenter` | Vue 3 + Vite 7 + Pinia + Element Plus | 保持现栈，统一请求封装和错误处理语义 |
| `designer` | Vue 3 + TS + Konva + ECharts + GSAP | 保持现栈，优先稳态与发布契约一致 |
| `runtime/node_agent_front` | Vue 3 + Vite 5 + Element Plus | 维持独立前端，但对齐接口约定与状态模型 |

### 节点执行器
| 组件 | 当前依赖 | 建议 |
|---|---|---|
| `runtime/node_agent` | Go 1.24 + gorilla/mux + Docker SDK + viper | 保持现栈，优先修复执行/日志/健康协议闭环 |

## 不建议当前阶段引入
- 不建议迁移到全新后端框架（如 Nest/Fastify）: 成本高、收益不直接。
- 不建议把多前端合并为单应用: 当前业务域边界已清晰，风险大于收益。
- 不建议同时推进 UI 重设计与契约治理: 会冲散交付焦点。

## 版本与治理策略
- 版本策略：小步升级、模块内验证、避免跨模块联动升级。
- 工程策略：优先“契约一致性 + 可观测 + 回归保障”，再做功能扩张。
- 置信度：高（依据仓库现有依赖与代码地图）。

## 结论
InduForge 的“标准栈”不是换技术，而是收敛实现：
1. 保持 Node/Vue/Go 三层架构；
2. 统一后端响应与错误契约；
3. 打通发布到运行时的契约与观测闭环。
