# InduForge

## What This Is

InduForge 是一个面向工业互联网场景的低代码开发平台，提供平台管理、数据中心、可视化设计与节点运行时执行链路。当前仓库是一个多模块单仓系统，已具备从设计到发布部署的主干能力。本项目初始化文档用于把既有能力、下一阶段目标和执行路线对齐到同一份可追踪基线。

## Core Value

在工业场景下，确保“设计-发布-部署-运行”链路稳定、可追踪、可运维。

## Requirements

### Validated

- ✅ 已具备多前端控制面能力（`dev_ide`、`datacenter`、`designer`、`runtime/node_agent_front`）并可独立开发部署。
- ✅ 已具备后端版本化 API 主入口（`/api/v1`）与健康检查（`/health`）能力。
- ✅ 已具备项目发布与节点部署主流程（`dev_core` 发布/部署服务 + `runtime/node_agent` 执行）。
- ✅ 已具备 MQTT + Socket.IO 的实时数据分发基础能力。
- ✅ 已具备多数据库连接与驱动抽象基础（MySQL/PostgreSQL/SQL Server）。

### Active

- [ ] 统一 `dev_core` 业务接口返回为 `ApiResponse`，统一业务错误为 `AppError` + `ErrorCodes`。
- [ ] 完善 Designer 发布产物与 IFP manifest 契约闭环（含 assets 打包、字段校验、版本兼容策略）。
- [ ] 落实 Runtime 健康与状态契约，打通 NodeAgent 与平台侧状态可观测。
- [ ] 收敛鉴权策略（开发旁路可控化）并补足关键鉴权回归验证。
- [ ] 提升部署与运行链路稳健性（日志保留策略、重复逻辑清理、运维脚本一致性）。

### Out of Scope

- 大规模 UI/视觉重设计：当前阶段以稳定性与契约收敛为先，不做全面交互改版。
- 新增独立移动端应用：当前优先保障 Web 与节点运行链路。
- 重构 `docker/`、`nginx/`、`miniio/` 目录：遵循仓库公共规则，本轮不作为业务开发目标。

## Context

- 仓库为 brownfield 项目，已有大量业务实现与历史代码，且已完成代码库地图（`.planning/codebase/*`）。
- 当前主要风险集中在“规范与历史实现并存”：接口返回风格不一致、局部重复代码、发布产物契约尚未完全冻结。
- `docs/contracts/` 已沉淀关键跨模块契约（发布 schema、IFP manifest、NodeAgent 启动协议、Runtime 状态协议），适合作为近期交付对齐基线。
- 团队规则已明确：中文协作、`pnpm`、统一时间库 `dayjs`、后端 API 规范化与参数化查询。

## Constraints

- **Tech stack**: 维持现有 Node.js + Vue + Go 混合架构，不引入大规模技术栈迁移，降低改造风险。
- **API Contract**: 后端 API 前缀固定为 `/api/v1`、健康检查 `/health`，并统一 `ApiResponse` 结构，保障跨前端消费一致性。
- **Process**: 数据库初始化必须走 `dev_core/scripts/init-database.js`，禁止 `sequelize.sync()`，避免环境漂移。
- **Tooling**: 包管理统一 `pnpm`，时间处理统一 `dayjs` + `YYYY-MM-DD HH:mm:ss`，减少模块间行为差异。
- **Scope**: 本轮优先“契约收敛与稳定性”，避免并行引入高不确定的新业务域。

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 以 brownfield 方式初始化并先固化现状基线 | 代码已存在且有可运行主链路，先建立一致基线再扩展最稳妥 | ✅ Good |
| 近期目标聚焦契约统一与运行链路稳定性 | 当前主要风险来自不一致与不可观测，而非功能缺失 | 🔄 Pending |
| 规划文档纳入 git 管理 | 需要跨会话、跨成员持续追踪需求与阶段映射 | ✅ Good |
| 默认启用研究/计划校验/交付验证流程 | 通过流程换取复杂系统改造的确定性 | 🔄 Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? -> Move to Out of Scope with reason
2. Requirements validated? -> Move to Validated with phase reference
3. New requirements emerged? -> Add to Active
4. Decisions to log? -> Add to Key Decisions
5. "What This Is" still accurate? -> Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check -> still the right priority?
3. Audit Out of Scope -> reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-04-09 after initialization (assumption-based baseline)*
