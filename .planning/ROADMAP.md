# Roadmap: InduForge

## Overview

本路线图聚焦"在不改动总体架构前提下，完成契约统一、发布闭环、运行可观测与运维稳态化"。阶段顺序按依赖推进，优先降低跨模块联调不确定性。

## Milestones

- 🚧 **v0.1** - Designer 画布编辑器优化 (Phases 1-3)
- 📋 **v1.0** - API/发布/运行/安全/运维收敛 (Phases 4-8, planned)

## Phases

**Phase Numbering:**
- 整数阶段（1,2,3...）为常规里程碑工作
- 小数阶段（2.1,2.2...）用于紧急插入工作

### v0.1 Milestone (In Progress)

- [ ] **Phase 1: 放置与拖放统一** - 单一 placementResolver 入口，统一坐标计算，修复 insertIndex 过期问题
- [ ] **Phase 2: 布局容器完善** - Flex/Free/Grid 布局联动正确，6 种布局嵌套 children 顺序正确
- [ ] **Phase 3: 层级与空间修复** - 堆叠顺序与 children 数组一致，空间位置计算正确

### v1.0 Milestone (Planned)

- [ ] **Phase 4: API 契约统一** - 统一后端响应结构与错误语义。
- [ ] **Phase 5: 发布产物契约闭环** - 固化 Designer 到 IFP 的发布契约与校验。
- [ ] **Phase 6: 运行态可观测打通** - 落实 Runtime/NodeAgent/平台状态协议。
- [ ] **Phase 7: 鉴权基线收敛** - 收敛开发旁路并补关键权限回归。
- [ ] **Phase 8: 运维稳态与交付收尾** - 修复执行链路一致性并补 smoke 验证。

## Phase Details

### v0.1 Milestone

#### Phase 1: 放置与拖放统一
**Goal**: 用户可以在任意容器类型中一致地放置组件，消除"相同视觉落点产生不同逻辑位置"的 bug
**Depends on**: Nothing (first phase)
**Requirements**: PLACE-01, PLACE-02, PLACE-03, TECH-01, TECH-02
**Success Criteria** (what must be TRUE):
  1. 用户从组件面板拖拽组件到画布，在 flex 容器、自由容器、网格容器中均能放置成功，且放置位置由单一入口计算
  2. 用户拖拽时 zoom/scroll 处理一致，坐标转换在所有入口使用相同逻辑（zoom 除法恰好一次，容器相对坐标计算统一）
  3. 快速拖拽时 insertIndex 过期问题已修复，目标不存在时 fallback 到 append 行为正确
  4. placementResolver.ts 单一入口已抽取，支持 Strategy 模式处理 flex/free/grid 布局类型
  5. placementUtils.js 坐标工具函数已统一，消除重复计算逻辑
**Plans**: 4 plans
**Plan list**:
- [x] 01-01: placement-utils scroll 补偿 + 坐标统一 (Wave 1)
- [x] 01-02: placementResolver Strategy 模式抽取 (Wave 1)
- [ ] 01-03: DesignCanvas/NodeRenderer 集成 + insertIndex 修复 (Wave 2)
- [ ] 01-04: Ghost/插入线/容器高亮视觉反馈 (Wave 2)

#### Phase 2: 布局容器完善
**Goal**: Flex/Free/Grid 布局容器的 layoutItem 联动正确，6 种布局嵌套时 children 顺序正确
**Depends on**: Phase 1
**Requirements**: LAYOUT-01, LAYOUT-02, LAYOUT-03, LAYOUT-04
**Success Criteria** (what must be TRUE):
  1. 水平/垂直布局的 flex grow/shrink/basis 属性正确联动，子组件尺寸按预期变化
  2. FreeContainer 的绝对定位与约束定位模式均可正确工作，切换模式时子组件位置正确
  3. Grid 栅格布局的 row/col/rowSpan/colSpan 计算正确，跨行跨列组件占据正确空间
  4. 水平/垂直/折叠/选项卡/表单/区域 6 种布局嵌套时，children 顺序保持正确
**Plans**: TBD
**UI hint**: yes

#### Phase 3: 层级与空间修复
**Goal**: 堆叠顺序与 children 数组一致，MoveNodeCommand 正确维护顺序，空间位置计算正确
**Depends on**: Phase 2
**Requirements**: LAYER-01, LAYER-02, LAYER-03
**Success Criteria** (what must be TRUE):
  1. 组件的视觉 z-order 与父容器的 children 数组顺序一致，无 z-index 样式覆盖导致的错位
  2. MoveNodeCommand 移动组件到新位置后，父容器的 children 数组顺序正确更新
  3. 各种容器类型下的 absolutePos 空间位置计算正确，组件在容器内、容器外的定位均符合预期
**Plans**: TBD

### v1.0 Milestone (Planned)

#### Phase 4: API 契约统一
**Goal**: 让前端面对统一的成功/失败语义，降低跨模块联调成本。
**Depends on**: Phase 3 (v0.1 complete)
**Requirements**: APIC-01, APIC-02, APIC-03
**Success Criteria** (what must be TRUE):
1. `dev_core` 目标路由统一返回 `ApiResponse`。
2. 业务异常统一映射到 `AppError` + `ErrorCodes`。
3. 接口日志与返回体均可追踪 `requestId`。
**Plans**: 3 plans

Plans:
- [ ] 04-01: 盘点并改造非统一响应路由（v1 关键路径）
- [ ] 04-02: 建立统一错误映射与中间件收口
- [ ] 04-03: 增补接口契约与回归测试

#### Phase 5: 发布产物契约闭环
**Goal**: 让发布制品可被节点侧稳定消费，避免"发布成功但运行失败"。
**Depends on**: Phase 4
**Requirements**: PUBC-01, PUBC-02, PUBC-03
**Success Criteria** (what must be TRUE):
1. Designer 输出结构满足约定 schema。
2. IFP 产物包含 manifest/pages/assets/data 等运行必需内容。
3. 发布前校验失败会阻断发布并给出可定位信息。
**Plans**: 3 plans

Plans:
- [ ] 05-01: 对齐发布 schema 与导出字段
- [ ] 05-02: 完善 IFP 打包内容与 manifest 生成
- [ ] 05-03: 实现发布前契约校验与失败提示

#### Phase 6: 运行态可观测打通
**Goal**: 平台可以持续看到节点运行状态并快速定位故障。
**Depends on**: Phase 5
**Requirements**: RTOB-01, RTOB-02, RTOB-03
**Success Criteria** (what must be TRUE):
1. Runtime `GET /health` 稳定可用于探活。
2. Runtime `GET /status` 返回约定状态字段。
3. 平台侧状态展示与 NodeAgent 状态一致。
**UI hint**: yes
**Plans**: 3 plans

Plans:
- [ ] 06-01: 完成 Runtime 健康/状态接口实现与契约测试
- [ ] 06-02: 对齐 NodeAgent 状态采集与上报逻辑
- [ ] 06-03: 平台侧状态展示与错误摘要联动

#### Phase 7: 鉴权基线收敛
**Goal**: 保证开发与生产权限模型一致，减少越权与联调偏差。
**Depends on**: Phase 6
**Requirements**: SECU-01, SECU-02, SECU-03
**Success Criteria** (what must be TRUE):
1. 开发旁路必须显式开启且默认关闭。
2. 关键路由完成能力与访问校验收敛。
3. 权限回归用例覆盖失效 token 与越权访问。
**Plans**: 2 plans

Plans:
- [ ] 07-01: 收敛鉴权旁路与关键路由守卫
- [ ] 07-02: 增加权限回归测试与异常场景验证

#### Phase 8: 运维稳态与交付收尾
**Goal**: 提升部署执行可复盘性，形成可持续交付基线。
**Depends on**: Phase 7
**Requirements**: OPSR-01, OPSR-02, OPSR-03
**Success Criteria** (what must be TRUE):
1. NodeAgent 日志不再覆盖，能追溯历史关键事件。
2. Makefile 目标和文档一致，常用命令一次可执行。
3. 发布-部署链路有最小 smoke 验证脚本与操作说明。
**Plans**: 2 plans

Plans:
- [ ] 08-01: 修复日志策略与构建脚本一致性
- [ ] 08-02: 建立发布-部署 smoke 验证并沉淀文档

## Progress

**Execution Order:**
v0.1 phases execute first: 1 -> 2 -> 3
Then v1.0 phases: 4 -> 5 -> 6 -> 7 -> 8

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. 放置与拖放统一 | v0.1 | 0/4 | Not started | - |
| 2. 布局容器完善 | v0.1 | 0/TBD | Not started | - |
| 3. 层级与空间修复 | v0.1 | 0/TBD | Not started | - |
| 4. API 契约统一 | v1.0 | 0/3 | Not started | - |
| 5. 发布产物契约闭环 | v1.0 | 0/3 | Not started | - |
| 6. 运行态可观测打通 | v1.0 | 0/3 | Not started | - |
| 7. 鉴权基线收敛 | v1.0 | 0/2 | Not started | - |
| 8. 运维稳态与交付收尾 | v1.0 | 0/2 | Not started | - |

---

*Roadmap created: 2026-04-15*
