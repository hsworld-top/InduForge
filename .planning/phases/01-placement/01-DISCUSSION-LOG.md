# Phase 1: 放置与拖放统一 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-15
**Phase:** 01-placement
**Areas discussed:** Drop target resolution, insertIndex 计算策略, 拖拽视觉反馈, Zoom/Scroll 一致性

---

## Drop Target Resolution（容器命中规则）

| Option | Description | Selected |
|--------|-------------|----------|
| 深度优先（最深可见容器） | 鼠标位置下的最深层可见容器作为目标 | |
| 容器边界递归匹配 | 从根容器向下递归，只要坐标在容器边界内就进入 | |
| 仅根级容器（主要）+ 深度优先（辅助） | 只允许直接放置到根容器的直接子级 | ✓ |

**User's choice:** 3为主，1为辅
**Notes:** 以根级容器为主要放置目标，不允许嵌套过深；深度优先作为辅助降级规则

---

## insertIndex 计算策略

| Option | Description | Selected |
|--------|-------------|----------|
| Y 坐标升序 + X 辅助 | 按组件顶部 Y 坐标排序，Y 相同时按 X | |
| 始终 append 到末尾 | 最安全，不会有错位 | |
| 最近邻节点参考 | 根据空间关系找最近的上方或下方节点作为参照 | |
| Y+X 优先 + append 兜底 + 最近邻精调 | 组合策略，先 Y+X，找不到参照时 append，有参照时精调 | ✓ |

**User's choice:** 123组合，先1再2再3
**Notes:** 三步组合策略，优先 Y 坐标，找不到时 append 兜底，最后用最近邻精调

---

## 拖拽视觉反馈

| Option | Description | Selected |
|--------|-------------|----------|
| Ghost 拖影 | 拖拽时显示组件半透明轮廓，跟随鼠标 | |
| 插入指示线 + 容器高亮 | 在放置位置显示插入线，高亮目标容器 | |
| 实时预览组件 | 组件以原样显示，跟随鼠标 | |
| Ghost + 插入指示线 + 容器高亮组合 | Ghost + 插入线 + 容器高亮同时使用 | ✓ |

**User's choice:** 1+2
**Notes:** Ghost 拖影 + 插入指示线 + 容器高亮三者组合使用

---

## Zoom/Scroll 一致性

| Option | Description | Selected |
|--------|-------------|----------|
| 统一入口强制调用 | 所有坐标计算强制调用 placementUtils.eventToCanvasPosition | |
| 支持嵌套 zoom | 处理画布级 zoom + 容器内局部 zoom 叠加 | |
| 处理 scroll 偏移 | scroll 容器的坐标计算需要额外补偿 scroll 偏移 | |
| 统一入口 + 嵌套 zoom + scroll 补偿全要 | 三者全部实现 | ✓ |

**User's choice:** 123都要，1统一入口2嵌套zoom 3scroll补偿
**Notes:** 统一入口处理 zoom，嵌套 zoom 叠加计算，scroll 偏移额外补偿

---

## Deferred Ideas

None — discussion stayed within phase scope.
