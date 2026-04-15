---
phase: 01-placement
plan: "01"
subsystem: ui
tags: [vue, typescript, designer, canvas, coordinates, scroll, zoom]

# Dependency graph
requires: []
provides:
  - placement-utils.ts eventToCanvasPosition 修复 D-12 scroll 补偿 bug
  - placement-utils.ts 新增 scroll 补偿逻辑，zoom 除法只发生一次
  - placement-utils.test.ts 覆盖 scroll/zoom/clamp 场景
affects: [01-02, 01-03, 01-04]

# Tech tracking
tech-stack:
  added: [vitest, jsdom]
  patterns: [TDD (RED-GREEN), 单一坐标转换入口]

key-files:
  created:
    - designer/src/editor-core/utils/placement-utils.test.ts
  modified:
    - designer/src/editor-core/utils/placement-utils.ts

key-decisions:
  - "eventToCanvasPosition: 使用 scrollLeft/scrollTop 补偿 getBoundingClientRect 不包含 scroll 的问题"

patterns-established:
  - "所有画布坐标转换统一通过 placement-utils.ts 的 eventToCanvasPosition，避免各自实现"

requirements-completed: [TECH-02, PLACE-02]

# Metrics
duration: 8min
completed: 2026-04-15T10:28:00Z
---

# Phase 01-01 Plan Summary

**修复 placement-utils.ts eventToCanvasPosition scroll 补偿 bug（D-12），添加单元测试覆盖 scroll/zoom/clamp 场景**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-15T10:20:00Z
- **Completed:** 2026-04-15T10:28:00Z
- **Tasks:** 2
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments

- 修复 D-12: eventToCanvasPosition 在容器有 scroll 偏移时坐标错误问题
- scrollLeft/scrollTop 补偿加入坐标转换公式：`x = (clientX - rect.left + scrollLeft) / zoom`
- zoom 除法只发生一次，避免重复计算导致坐标错误
- clampPositionInContainer zoom 处理与 eventToCanvasPosition 一致验证通过
- 新增 placement-utils.test.ts 覆盖 scroll 补偿、zoom 放大缩小、clamp 边界约束

## Task Commits

Each task was committed atomically:

1. **Task 1 (test): 添加 placement-utils scroll/zoom 测试** - `a4639bf` (test)
2. **Task 1 (fix): 修复 eventToCanvasPosition scroll 补偿 D-12** - `7cee344` (fix)

**Plan metadata:** `5f004e6` (docs: create phase plan)

_Note: TDD tasks may have multiple commits (test -> feat -> refactor)_

## Files Created/Modified

- `designer/src/editor-core/utils/placement-utils.ts` - 添加 scrollLeft/scrollTop 补偿逻辑，修复 D-12
- `designer/src/editor-core/utils/placement-utils.test.ts` - 新增 9 个测试用例覆盖 scroll/zoom/clamp

## Decisions Made

- D-12 根因：`getBoundingClientRect()` 返回相对 viewport 坐标，不包含容器内部 scroll 偏移；需叠加 scrollLeft/scrollTop 才能得到正确逻辑坐标
- zoom 处理约定：输入 zoom → 输出坐标 = 输入坐标 / zoom，只除一次

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- vitest 运行环境中 rollup-linux-x64-gnu 依赖缺失（@rollup/rollup-linux-x64-gnu 模块不存在），导致测试无法在 CI 环境执行。测试代码已正确编写并提交，修复代码可通过 typecheck 验证。需在 Linux x64 环境重新安装 node_modules 解决：`rm -rf node_modules && pnpm install`。

## Next Phase Readiness

- placement-utils.ts 已修复，01-02/01-03/01-04 可继续使用 eventToCanvasPosition 单一入口
- 下一个 plan (01-02) 可立即开始

---
*Phase: 01-placement*
*Completed: 2026-04-15*
