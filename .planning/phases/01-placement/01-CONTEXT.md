# Phase 1: 放置与拖放统一 - Context

**Gathered:** 2026-04-15
**Status:** Ready for planning

<domain>
## Phase Boundary

用户可以从组件面板拖拽组件到画布，在 flex 容器、自由容器、网格容器中均能放置成功，且放置位置由单一入口计算。消除"相同视觉落点产生不同逻辑位置"的 bug。

</domain>

<decisions>
## Implementation Decisions

### Drop Target Resolution（容器命中规则）
- **D-01:** 以根级容器为主要放置目标，不允许嵌套过深（仅根级容器直接子级）
- **D-02:** 深度优先（最深可见容器）为辅助规则，当根级容器逻辑不明确时降级使用
- **D-03:** placementResolver.ts 单一入口统一处理容器命中判断，Strategy 模式按容器类型（flex/free/grid）分发

### insertIndex 计算策略
- **D-04:** Y 坐标优先：按组件顶部 Y 坐标升序排序，Y 相同时按 X 辅助排序
- **D-05:** append 兜底：当 insertIndex 过期（目标节点不存在）时 fallback 到 children 数组末尾
- **D-06:** 最近邻精调：在 Y+X 基础上，以最近邻节点为参照精确调整插入位置

### 拖拽视觉反馈
- **D-07:** Ghost 拖影：拖拽时显示组件半透明轮廓，跟随鼠标移动
- **D-08:** 插入指示线：在放置位置显示插入线（横线/竖线）指示精确插入点
- **D-09:** 容器高亮：目标容器在拖拽进入时高亮显示，视觉区分放置目标

### Zoom/Scroll 一致性
- **D-10:** 统一入口：所有坐标计算强制调用 placementUtils.eventToCanvasPosition，禁止各自处理 zoom
- **D-11:** 嵌套 zoom：支持画布级 zoom + 容器内局部 zoom 的叠加计算
- **D-12:** scroll 补偿：scroll 容器的 getBoundingClientRect 不包含 scroll 偏移，需要额外补偿

### Claude's Discretion
- Ghost 拖影的具体透明度数值
- 插入指示线的样式（颜色、线宽）
- 容器高亮的颜色和边框样式

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Designer 画布编辑器
- `designer/src/editor-core/utils/placement-utils.ts` — 现有放置工具函数，包含 eventToCanvasPosition、clampPositionInContainer
- `designer/src/ui/editors/page/canvas/DesignCanvas.vue` — 画布级拖放入口 handleCanvasDrop
- `designer/src/ui/editors/page/canvas/NodeRenderer.vue` — 节点级拖放入口 handleDrop
- `designer/src/ui/editors/page/canvas/composables/use-node-drop.ts` — 拖放状态管理

### Requirements
- `.planning/REQUIREMENTS.md` §放置与拖放 — PLACE-01、PLACE-02、PLACE-03、TECH-01、TECH-02

</canonical_refs>

<codebase>
## Existing Code Insights

### Reusable Assets
- `placement-utils.ts`（已存在）：`eventToCanvasPosition`、`clampPositionInContainer` 可直接使用或扩展
- `use-drag-state.ts`：拖拽状态管理可复用
- `use-node-drop.ts`：节点放置 composable

### Established Patterns
- Strategy 模式在 designer 中有先例（如 descriptor registry）
- placementResolver.ts 单一入口已规划（TECH-01），本 phase 实现

### Integration Points
- `DesignCanvas.handleCanvasDrop`：画布级放置入口
- `NodeRenderer.handleDrop`：节点级放置入口
- 两者将统一收敛到 placementResolver

</codebase>

<specifics>
## Specific Ideas

- 插入指示线应该在最近邻节点旁边显示，而不是覆盖在节点上
- 容器高亮采用虚线边框，与选中态实线边框区分

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 01-placement*
*Context gathered: 2026-04-15*
