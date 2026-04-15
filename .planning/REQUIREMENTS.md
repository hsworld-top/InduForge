# Requirements: InduForge

**Defined:** 2026-04-15
**Core Value:** 在工业场景下，确保"设计-发布-部署-运行"链路稳定、可追踪、可运维。

## v0.1 Requirements

Designer 画布编辑器优化。聚焦组件放置、布局控制、层级关系的 bug 修复与体验提升。

### 放置与拖放

- [ ] **PLACE-01**: 统一组件放置入口，合并 DesignCanvas.handleCanvasDrop 与 NodeRenderer.handleDrop 的放置逻辑
- [ ] **PLACE-02**: 统一画布坐标计算，所有拖放使用相同的 zoom/scroll 处理
- [ ] **PLACE-03**: 解决 insertIndex 在快速拖拽时过期导致插错位置的边界情况

### 布局容器

- [ ] **LAYOUT-01**: 完善水平布局/垂直布局的 layoutItem 联动（flex grow/shrink/basis）
- [ ] **LAYOUT-02**: 完善自由布局 FreeContainer 的绝对定位与约束定位模式
- [ ] **LAYOUT-03**: 完善 Grid 栅格布局的 row/col/rowSpan/colSpan 计算
- [ ] **LAYOUT-04**: 确保 6 种布局（水平/垂直/折叠/选项卡/表单/区域）嵌套使用时 children 顺序正确

### 层级与空间

- [ ] **LAYER-01**: 修复堆叠顺序与 children 数组不一致的边界情况
- [ ] **LAYER-02**: 确保 MoveNodeCommand 正确维护父容器的 children 顺序
- [ ] **LAYER-03**: 验证空间位置计算（absolutePos）为各种容器类型下的正确性

### 技术债务

- [ ] **TECH-01**: 抽取 placementResolver.ts 单一入口，Strategy 模式处理不同布局类型
- [ ] **TECH-02**: 统一 placementUtils.js 坐标工具函数，消除重复计算逻辑

## v1 Requirements

（原 v1 需求已迁移至 v0.1，v1 里程碑待定）

### API Contract

- [ ] **APIC-01**: `dev_core` 的 `/api/v1` 业务接口统一返回 `ApiResponse`
- [ ] **APIC-02**: 业务异常统一使用 `AppError` 与 `ErrorCodes`
- [ ] **APIC-03**: 请求全链路携带并回传 `requestId`

### Publish Contract

- [ ] **PUBC-01**: Designer 发布态输出满足约定 schema
- [ ] **PUBC-02**: IFP 打包包含运行必需资产
- [ ] **PUBC-03**: 发布前执行 schema 校验

### Runtime Observability

- [ ] **RTOB-01**: Runtime 提供稳定 `GET /health`
- [ ] **RTOB-02**: Runtime 提供稳定 `GET /status`
- [ ] **RTOB-03**: 平台侧状态展示与 NodeAgent 一致

### Security Baseline

- [ ] **SECU-01**: 开发鉴权旁路默认关闭
- [ ] **SECU-02**: 关键路由统一能力校验
- [ ] **SECU-03**: 鉴权路径自动化回归测试

### Operations Reliability

- [ ] **OPSR-01**: NodeAgent 日志可追溯
- [ ] **OPSR-02**: Makefile 目标与文档一致
- [ ] **OPSR-03**: smoke 验证脚本与执行说明

## Out of Scope

| Feature | Reason |
|---------|--------|
| 吸附系统（Snap） | 良好 UX 但非当前 bug，建议 v0.2 考虑 |
| 空间索引（rbush） | 100+ 节点才生效，v0.1 聚焦稳定性 |
| 约束求解器（kiwi.js） | 复杂度高，FreeContainer constraints 手工实现足够 |
| 新增布局类型 | 当前 6 种布局已覆盖，v0.1 不扩展 |
| 全量 UI 重设计 | 当前阶段优先稳定性 |
| 移动端 App | 现阶段聚焦 Web 与节点运行链路 |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| PLACE-01 | Phase 1 | Pending |
| PLACE-02 | Phase 1 | Pending |
| PLACE-03 | Phase 1 | Pending |
| LAYOUT-01 | Phase 2 | Pending |
| LAYOUT-02 | Phase 2 | Pending |
| LAYOUT-03 | Phase 2 | Pending |
| LAYOUT-04 | Phase 2 | Pending |
| LAYER-01 | Phase 3 | Pending |
| LAYER-02 | Phase 3 | Pending |
| LAYER-03 | Phase 3 | Pending |
| TECH-01 | Phase 1 | Pending |
| TECH-02 | Phase 1 | Pending |
| APIC-01 | Future (v1) | Pending |
| APIC-02 | Future (v1) | Pending |
| APIC-03 | Future (v1) | Pending |
| PUBC-01 | Future (v1) | Pending |
| PUBC-02 | Future (v1) | Pending |
| PUBC-03 | Future (v1) | Pending |
| RTOB-01 | Future (v1) | Pending |
| RTOB-02 | Future (v1) | Pending |
| RTOB-03 | Future (v1) | Pending |
| SECU-01 | Future (v1) | Pending |
| SECU-02 | Future (v1) | Pending |
| SECU-03 | Future (v1) | Pending |
| OPSR-01 | Future (v1) | Pending |
| OPSR-02 | Future (v1) | Pending |
| OPSR-03 | Future (v1) | Pending |

**Coverage:**
- v0.1 requirements: 11 total
- Mapped to phases: 11
- Unmapped: 0 ✓

---
*Requirements defined: 2026-04-15*
*Last updated: 2026-04-15 after v0.1 requirements definition*
