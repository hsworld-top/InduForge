# Requirements: InduForge

**Defined:** 2026-04-09
**Core Value:** 在工业场景下，确保“设计-发布-部署-运行”链路稳定、可追踪、可运维。

## v1 Requirements

### API Contract

- [ ] **APIC-01**: `dev_core` 的 `/api/v1` 业务接口统一返回 `ApiResponse`（`success/errorCode/message/requestId/data`）。
- [ ] **APIC-02**: 业务异常统一使用 `AppError` 与 `ErrorCodes`，前端可按 `errorCode` 稳定分流。
- [ ] **APIC-03**: 请求全链路携带并回传 `requestId`，便于跨服务排障。

### Publish Contract

- [ ] **PUBC-01**: `designer` 发布态输出满足约定 schema，关键字段可被后端稳定解析。
- [ ] **PUBC-02**: IFP 打包包含运行必需资产（`manifest/pages/assets/data`）并可被节点侧消费。
- [ ] **PUBC-03**: 发布前执行 manifest 与 schema 校验，不合格产物阻断发布。

### Runtime Observability

- [ ] **RTOB-01**: Runtime 提供稳定 `GET /health`，可被 NodeAgent 高频探活。
- [ ] **RTOB-02**: Runtime 提供稳定 `GET /status`，包含运行状态、版本、启动时间与最近错误摘要。
- [ ] **RTOB-03**: 平台侧可展示节点运行状态并与 NodeAgent 状态模型一致。

### Security Baseline

- [ ] **SECU-01**: 开发环境鉴权旁路改为显式可控配置，默认关闭。
- [ ] **SECU-02**: 关键业务路由统一执行能力校验与租户/项目访问校验。
- [ ] **SECU-03**: 关键鉴权路径具备自动化回归测试（含越权与失效 token 场景）。

### Operations Reliability

- [ ] **OPSR-01**: NodeAgent 启动日志改为可追溯策略（追加/轮转），避免覆盖历史日志。
- [ ] **OPSR-02**: NodeAgent Makefile 目标与文档一致，开发/构建命令可直接执行。
- [ ] **OPSR-03**: 发布-部署关键链路具备最小 smoke 验证脚本与执行说明。

## v2 Requirements

### Platform Enhancements

- **PLAT-01**: 提供发布契约可视化诊断面板（差异项、失败原因、修复建议）。
- **PLAT-02**: 增加节点灰度发布策略与批次回滚策略配置。
- **PLAT-03**: 提供跨模块统一可观测大盘（发布、部署、运行、数据链路）。

## Out of Scope

| Feature | Reason |
|---------|--------|
| 全量 UI 重设计 | 当前阶段优先稳定性与契约一致性 |
| 移动端 App | 现阶段资源聚焦 Web 与节点运行链路 |
| 核心栈替换（如迁移新后端框架） | 改造风险高，收益不直接 |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| APIC-01 | Phase 1 | Pending |
| APIC-02 | Phase 1 | Pending |
| APIC-03 | Phase 1 | Pending |
| PUBC-01 | Phase 2 | Pending |
| PUBC-02 | Phase 2 | Pending |
| PUBC-03 | Phase 2 | Pending |
| RTOB-01 | Phase 3 | Pending |
| RTOB-02 | Phase 3 | Pending |
| RTOB-03 | Phase 3 | Pending |
| SECU-01 | Phase 4 | Pending |
| SECU-02 | Phase 4 | Pending |
| SECU-03 | Phase 4 | Pending |
| OPSR-01 | Phase 5 | Pending |
| OPSR-02 | Phase 5 | Pending |
| OPSR-03 | Phase 5 | Pending |

**Coverage:**
- v1 requirements: 15 total
- Mapped to phases: 15
- Unmapped: 0 ✅

---
*Requirements defined: 2026-04-09*
*Last updated: 2026-04-09 after initial definition (assumption-based baseline)*
