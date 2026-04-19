# InduForge AI 协作总则

本文件是仓库根级公共规则，只保留跨工具、跨模块都稳定成立的约束。处理具体模块前，优先读取离改动目录最近的 `AGENTS.md` 或 `CLAUDE.md`。

## 公共设置

- 使用中文沟通与注释。
- 工作目录为仓库根目录，统一使用根目录 `.env`。
- 包管理器统一使用 `pnpm`。
- 后端 API 前缀统一为 `/api/v1`，健康检查为 `/health`。
- 非业务入口目录 `docker/`、`nginx/`、`miniio/` 默认不作为本轮业务开发目标。

## 目录路由

- `dev_core/`：后端控制面与数据聚合。
- `dev_ide/`：平台管理前端。
- `datacenter/`：数据中心前端。
- `designer/`：低代码设计器。
- `runtime/`：运行时共享层。
- `runtime/node_agent/`：节点执行器后端。
- `runtime/node_agent_front/`：节点本地管理前端。

处理单模块任务时，只加载对应模块规则；跨模块任务按涉及目录分别补读，不要把整仓长文默认塞入上下文。

## 文档与规则文件

- 项目确认后的正式文档统一放在 `docs/`。
- 开发过程文档、任务文档、分析稿、专项计划、审计稿、AI 辅助文档统一放在 `.planning/`，推荐使用 `.planning/docs/process/` 与 `.planning/ai-packages/`。
- 模块目录默认不放普通过程 `.md`，仅允许保留 AI 入口文件 `AGENTS.md`、`CLAUDE.md`。
- 根目录业务文档仅保留 `README.md`；AI 入口与忽略文件属于配置例外。
- 修改或新增业务文档前，需先与需求方确认；本次 AI 规则重构相关文件除外。

## 提交规范

- commit message 必须使用中文，推荐格式为 `type(scope): 中文描述`。
- `type` 推荐值：`feat`、`fix`、`refactor`、`docs`、`chore`、`test`。
- `scope` 推荐值：`dev_core`、`dev_ide`、`datacenter`、`designer`、`runtime`、`node_agent`、`product`。
- 单次提交优先聚焦单一目标；不要把无关改动混入同一提交。
- 若本地最近一次提交信息不符合规范，提交前先修正。

## 通用开发约束

- 后端接口统一使用 `ApiResponse` 格式：`success`、`errorCode`、`message`、`requestId`、`data`。
- 业务错误统一使用 `AppError` 与 `ErrorCodes`。
- 数据库初始化统一使用 `dev_core/scripts/init-database.js`，禁止使用 `sequelize.sync()`。
- `dev_core`、`dev_ide`、`datacenter`、`designer` 统一使用 `dayjs`，字符串时间格式为 `YYYY-MM-DD HH:mm:ss`。
- 类、函数尽量补中文文档注释；复杂业务逻辑前补一行目的说明。
- 数据库操作必须使用参数化查询。
- 避免冗余计算，前端注意懒加载与无意义的全量更新。

## 协作要求

- 需求不清时，先交付可运行的基础版本，再明确待确认问题。
- 完成改动后优先运行与改动最接近的验证命令，再汇报结论。
- 若规则与文档冲突，以模块内最新规则和 `docs/contracts/` 契约文档为准。

<!-- GSD:project-start source:PROJECT.md -->
## Project

InduForge 是一个面向工业互联网场景的低代码开发平台，提供平台管理、数据中心、可视化设计与节点运行时执行链路。当前仓库是一个多模块单仓系统，已具备从设计到发布部署的主干能力。本项目初始化文档用于把既有能力、下一阶段目标和执行路线对齐到同一份可追踪基线。

**Core Value:** 在工业场景下，确保“设计-发布-部署-运行”链路稳定、可追踪、可运维。

### Constraints

- **Tech stack**: 维持现有 Node.js + Vue + Go 混合架构，不引入大规模技术栈迁移，降低改造风险。
- **API Contract**: 后端 API 前缀固定为 `/api/v1`、健康检查 `/health`，并统一 `ApiResponse` 结构，保障跨前端消费一致性。
- **Process**: 数据库初始化必须走 `dev_core/scripts/init-database.js`，禁止 `sequelize.sync()`，避免环境漂移。
- **Tooling**: 包管理统一 `pnpm`，时间处理统一 `dayjs` + `YYYY-MM-DD HH:mm:ss`，减少模块间行为差异。
- **Scope**: 本轮优先“契约收敛与稳定性”，避免并行引入高不确定的新业务域。
<!-- GSD:project-end -->

<!-- GSD:stack-start source:codebase/STACK.md -->
## Technology Stack

## 代码库定位
- 本仓库是工业低代码平台多模块单仓，核心目录为 `dev_core`、`dev_ide`、`datacenter`、`designer`、`runtime/node_agent`、`runtime/node_agent_front`。
- 后端统一入口由 `dev_core/src/index.js` 启动，核心 API 前缀为 `/api/v1`，健康检查为 `/health`（见 `dev_core/src/app.js`）。
- `runtime/node_agent` 是 Go 编写的节点执行器，`runtime/node_agent_front` 是其本地 Vue 管理界面。
## 语言与运行时
- JavaScript/Node.js：`dev_core`、`dev_ide`、`datacenter`、`runtime/node_agent_front`。
- TypeScript：`designer`（例如 `designer/src/stores/editor-store.ts`）。
- Go：`runtime/node_agent`（见 `runtime/node_agent/go.mod`，Go 1.24.0）。
## 后端技术栈（dev_core）
- Web 框架：`express`（`dev_core/package.json`，入口 `dev_core/src/app.js`）。
- ORM：`sequelize`（`dev_core/src/config/database.js` + `dev_core/src/models/index.js`）。
- 数据库驱动：`mysql2`、`pg`、`mssql`（驱动工厂 `dev_core/src/services/drivers/DriverFactory.js`）。
- 缓存与队列：`ioredis`（`dev_core/src/utils/redis.js`）。
- 对象存储：`minio`（`dev_core/src/services/storageService.js`）。
- 实时通信：`socket.io`（`dev_core/src/services/socketService.js`）。
- 设备消息协议：`mqtt`（`dev_core/src/services/mqttService.js` + `dev_core/src/services/protocols/MqttProtocol.js`）。
- 安全与中间件：`helmet`、`cors`、`express-rate-limit`、`jsonwebtoken`、`bcryptjs`。
- 时间库：`dayjs`（符合仓库规则，`dev_core/src/config/database.js` 等）。
## 前端技术栈
- `dev_ide`：Vue 3 + Vite + Pinia + Element Plus + vue-router + vue-i18n（`dev_ide/package.json`、`dev_ide/src/main.js`）。
- `datacenter`：Vue 3 + Vite + Pinia + Element Plus + Monaco + SQL 解析/格式化（`datacenter/package.json`）。
- `designer`：Vue 3 + TypeScript + Vite + Pinia + Element Plus + Konva + ECharts + GSAP + Vitest（`designer/package.json`）。
- `runtime/node_agent_front`：Vue 3 + Vite + Pinia + Element Plus（`runtime/node_agent_front/package.json`）。
## 节点执行器技术栈（runtime/node_agent）
- HTTP 路由：`gorilla/mux`（`runtime/node_agent/internal/web/handler/routes.go`）。
- 容器能力：Docker SDK（`runtime/node_agent/internal/agent/executor/docker.go`）。
- 本地执行：进程执行器（`runtime/node_agent/internal/agent/executor/process.go`）。
- 本地持久化：SQLite（`modernc.org/sqlite`，`runtime/node_agent/internal/agent/store/store.go`）。
- 配置框架：`viper`（`runtime/node_agent/cmd/main.go`）。
## 构建与包管理
- Node 模块统一使用 `pnpm`（根 AGENTS 规则）。
- 前端构建工具统一为 Vite（各子模块 `vite.config.js` / `vite.config.ts`）。
- Go 模块由 `runtime/node_agent/Makefile` 提供 `build/test/vet`。
## 环境配置
- 根环境变量模板：`.env_example`。
- 关键端口：`PORT=9099`（后端）、`VITE_IDE_PORT=9091`、`VITE_DATACENTER_PORT=9092`、`VITE_DESIGNER_PORT=9093`、`NODE_AGENT_PORT=8081`、`VITE_NODE_AGENT_FRONT_PORT=13000`。
- 数据与中间件：MySQL、Redis、MinIO、MQTT 在 `.env_example` 中提供默认项。
## 结论
- 当前仓库是“Node.js 业务聚合后端 + 多 Vue 前端 + Go 节点执行器”的混合技术栈。
- 业务核心在 `dev_core` 与 `designer`，运行时控制链路在 `dev_core -> runtime/node_agent -> runtime/node_agent_front`。
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

## 全仓公共约束（来自根 AGENTS）
- 沟通与注释使用中文。
- 根目录统一 `.env`，包管理器统一 `pnpm`。
- 后端 API 前缀 `/api/v1`，健康检查 `/health`。
- 后端错误/响应建议统一：`ApiResponse` + `AppError` + `ErrorCodes`。
- 时间库统一 `dayjs`，字符串格式 `YYYY-MM-DD HH:mm:ss`。
- 数据库初始化应通过 `dev_core/scripts/init-database.js`，禁止 `sequelize.sync()`。
## 模块级命名约定
- Vue 组件名：`PascalCase`（见各模块 AGENTS）。
- 文件名：通常 `kebab-case`。
- composable：`useXxx`（如 `datacenter/src/composables/*`）。
- API 文件：`*.api.js`（`dev_ide/src/api/*.api.js`）。
## 后端实现模式
- 路由层 -> 服务层 -> 模型层：
- 示例：`dev_core/src/routes/v1/deployment.js` -> `dev_core/src/services/deploymentService.js` -> `dev_core/src/models/*`。
- 权限模型：`authenticate`、`checkProjectAccess`、`requireCapability`（`dev_core/src/middlewares/auth.js`）。
- 版本化路由：`dev_core/src/routes/register.js` 挂载 v1/v2。
## 前端实现模式
- 统一 axios 请求封装 + token 刷新：
- `dev_ide/src/utils/request.js`
- `datacenter/src/utils/request.js`
- `designer/src/utils/request.ts`
- 路由守卫处理登录态与页面跳转（如 `dev_ide/src/router/index.js`、`designer/src/router/index.ts`）。
- 大型页面采用懒加载和模块分割（如 `designer/src/ui/shell/DesignerView.vue`）。
## 编辑器工程模式（designer）
- Store 统一调度（`designer/src/stores/editor-store.ts`）。
- 内核命令模式（`designer/src/editor-core/commands/*`）。
- 页面 Schema 归一化与持久化动作拆分到 `stores/editor/*` 子模块。
## 代码质量工具
- ESLint Flat Config：`dev_ide/eslint.config.js`、`datacenter/eslint.config.js`。
- designer 使用 `@antfu/eslint-config`：`designer/eslint.config.mjs`。
- 格式化统一 `prettier`（各模块脚本中定义）。
## 观察到的偏差（与“应有规范”对比）
- `dev_core/src/routes/v1/data.js`、`dev_core/src/routes/v1/node-register.js` 中存在直接 `res.json` 返回，未统一走 `ApiResponse`。
- `dev_core/src/middlewares/auth.js` 在 development 下存在无 token 自动注入默认管理员逻辑。
- 局部文件出现重复定义/重复调用（例如 `dev_core/src/services/socketService.js` 中 `setupDataPointSubscription` 定义重复）。
## 结论
- 仓库总体遵循“分层 + 统一请求封装 + 角色权限”模式。
- 目前处于“新规范与历史实现并存”阶段，建议后续逐步收敛响应格式、权限策略和重复代码段。
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

## 架构风格
- 仓库采用多应用单仓（multi-app monorepo）结构，而不是严格的 pnpm workspace 根工作区。
- 平台控制面与数据聚合集中在 `dev_core`，多个前端应用按业务域拆分。
- 运行时执行层与控制面解耦，`runtime/node_agent` 可独立部署在节点侧。
## 分层视图
- 展示层：
- `dev_ide`（平台管理）
- `datacenter`（数据中心）
- `designer`（低代码设计器）
- `runtime/node_agent_front`（节点本地运维）
- 控制层：
- `dev_core/src/routes/v1/*.js`（版本化 API）
- `dev_core/src/middlewares/*.js`（认证、审计、国际化、请求ID）
- 领域服务层：
- `dev_core/src/services/*.js`（发布、部署、数据连接、MQTT、设计资产等）
- 数据访问层：
- Sequelize 模型与关联：`dev_core/src/models/index.js`
- 驱动抽象层：`dev_core/src/services/drivers/*.js`
- 执行层：
- 进程/容器执行器：`runtime/node_agent/internal/agent/executor/*.go`
## 后端启动与路由装配
- 入口：`dev_core/src/index.js`。
- 应用构建：`dev_core/src/app.js`。
- 路由注册：`dev_core/src/routes/register.js`。
- API 版本：
- v1：业务主路由（`/api/v1`）
- v2：测试路由（`/api/v2/test`，`dev_core/src/routes/v2/index.js`）
## 核心业务流
- 发布流：
- `publishService.publish`（`dev_core/src/services/publishService.js`）
- 校验工程 -> 汇总页面/数据点 -> 生成 manifest -> 打包 IFP -> 记录 Deployment
- 部署流：
- `deploymentService.deployToNodes/deployDevToNodesByProject`（`dev_core/src/services/deploymentService.js`）
- 创建 NodeDeployment/NodeCommand -> 节点心跳取指令 -> 节点回传状态
- 节点执行流：
- `runtime/node_agent/cmd/main.go` 启动 HTTP 服务
- `APIHandler` 处理项目部署/启停/回滚（`runtime/node_agent/internal/web/handler/api.go`）
- `Orchestrator` 调用 `Executor` 完成执行（`runtime/node_agent/internal/agent/orchestrator/orchestrator.go`）
## 实时通信流
- MQTT 输入：`MqttProtocol` -> `MqttService.handleMqttMessage`。
- 消息分发：`socketService.broadcastMqttMessage`、数据点广播、运维广播。
- 前端消费：通过 socket.io 房间订阅（项目、订阅、数据点、运维租户维度）。
## 设计器内部架构
- 主状态：`designer/src/stores/editor-store.ts`（文档、历史、选区、页面、锁）。
- 文档模型：`designer/src/editor-core/document/*`。
- 命令系统：`designer/src/editor-core/commands/*`。
- 壳层编排：`designer/src/ui/shell/DesignerView.vue`。
## 运行时模式
- NodeAgent 支持 `offline` 与 `online` 模式（`runtime/node_agent/cmd/main.go` + `internal/pkg/config`）。
- online 模式通过中心心跳接收 deploy/start/stop/restart/rollback 指令（`heartbeat_runtime.go`）。
- 离线模式提供本地工程管理 API 与本地前端。
## 架构特征总结
- 优点：模块边界清晰、运行时可独立部署、前后端职责分层明确。
- 当前成本：接口风格不完全统一、历史代码与新规范并存、跨模块契约仍需持续收敛。
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, or `.github/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
