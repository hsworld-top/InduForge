# Project Research Summary

**Project:** InduForge Designer - Canvas Layout and Interaction Optimization
**Domain:** Low-code canvas editor with Vue 3 + Konva dual-layer architecture
**Researched:** 2026-04-15
**Confidence:** MEDIUM-HIGH

## Executive Summary

InduForge Designer is a low-code industrial IoT canvas editor using a Vue DOM + Konva canvas dual-layer architecture for interactive component placement and graphics rendering. The current milestone targets fixing scattered drop target resolution logic (multiple entry points causing "same drop, different result" bugs), improving spatial query performance, and implementing a unified placement system following the Strategy pattern.

Research across stack, features, architecture, and pitfalls converges on a clear imperative: consolidate all drop resolution into a single `placementResolver` before adding new capabilities. Stack additions (`rbush` for spatial indexing, `kiwi.js` for constraints) are additive improvements, but the architecture shows the real problem is scattered entry points and uncoordinated coordinate systems. Professional editors (Figma, Sketch, Adobe XD) all use single drop resolution paths with Strategy-based container handling.

Key risks: (1) Multiple existing drop handlers being consolidated may break valid edge cases if not fully mapped first. (2) kiwi.js constraint solver is a JS port with limited maintenance - may need fallback to hand-written solver. (3) Spatial indexing benefit only materializes at 100+ nodes - typical pages may not see improvement. The recommended approach: Phase 1 extracts placementResolver with Strategy pattern, Phase 2 adds spatial indexing, Phase 3 adds snapping, Phase 4 addresses constraints and polish.

## Key Findings

### Recommended Stack

**Summary from STACK.md:** The designer needs three specific additions to address identified performance and functionality gaps. `rbush` (^4.0.0) provides R-tree spatial indexing for O(log n) hit testing vs current O(n) DOM queries. `kiwi.js` (^1.0.0) handles constraint equation solving for FreeContainer's constraint-based positioning. A custom snap system (no library) handles grid/edge/guide/center snapping per industry standard.

**Core technologies:**
- `rbush` (^4.0.0): Spatial indexing for hit testing, marquee selection, collision detection
- `kiwi.js` (^1.0.0): Constraint solver for FreeContainer constraints mode
- Custom SnapEngine: Build per industry standard (Figma/Sketch/XD all custom)
- Keep existing: konva, vue-konva, vue3-grid-layout, vue3-sketch-ruler, gsap
- Extend existing `layout-utils.ts` for FlexLayoutCalculator and GridLayoutCalculator

### Expected Features

**Summary from FEATURES.md:** Users expect predictable drag-drop where "same visual drop = same logical result". Professional editors achieve this through single resolver paths and clear visual feedback. Industrial data binding and dual-layer canvas are differentiators, not table stakes.

**Must have (table stakes) — fix current bugs:**
- Unified `placementResolver` — eliminates DesignCanvas vs NodeRenderer drop divergence
- Coordinate system consistency — zoom division applied exactly once, container-relative coords
- Fix insertIndex snapshot binding — fall back to append if snapshot target mismatches current

**Should have (complete the experience):**
- Visual drop indicators — blue border on drag enter, insert index line, empty container overlay
- Layer panel drag-to-reorder — same resolution logic as component drop

**Defer (v2+):**
- Industrial data binding integration (drag data point onto component)
- Multi-view responsive design with breakpoint schema
- Symbol library for industrial components

**Anti-features (explicitly avoid):**
- Per-element z-index property — breaks children array = z-order convention
- Multiple drop handlers — current bug source
- Document modification during drag — causes state inconsistency

### Architecture Approach

**Summary from ARCHITECTURE.md:** Dual-layer Vue DOM + Konva architecture with Strategy pattern for container-specific placement. `DragDropManager` currently has scattered logic that needs extraction into `placementResolver.ts` with `FlexPlacementStrategy`, `FreePlacementStrategy`, `GridPlacementStrategy`. Command pattern for layout changes via `InsertNodeCommand` and `MoveNodeCommand`. Store (`editor-store.ts`) holds document state with `docVersion` for reactivity.

**Major components:**
1. `CanvasContainer.vue` — rulers, zoom, pan, drag coordination shell
2. `DesignCanvas.vue` — canvas root, DOM mounting, one of two current drop entry points
3. `NodeRenderer.vue` — recursive DOM node rendering, second drop entry point
4. `DragDropManager.ts` — drop target resolution (needs refactoring into placementResolver)
5. `placement-utils.ts` — coordinate conversion utilities (expand, not replace)
6. `editor-store.ts` — document state, selection, history, canvas mouse position

### Critical Pitfalls

**Top 5 from PITFALLS.md:**

1. **Multiple Drop Target Resolution Entry Points** — DesignCanvas.handleCanvasDrop and NodeRenderer.handleDrop use different coordinate calculations. Same visual drop produces different logical position. Prevention: Single `placementResolver` that both call.

2. **Stacking Order Desynchronization** — Visual z-order doesn't match parent.children array order if any code uses z-index style override. Prevention: Strictly use children array order, only ReorderNodeCommand modifies stacking.

3. **insertIndex Staleness** — Multiple snapshot sources (rowInsertSnapshot, layoutInsertSnapshot) captured during dragover may not match current target on drop. Prevention: Validate snapshots against current target.id, fallback to append.

4. **Canvas Coordinate System Inconsistencies** — Zoom division applied inconsistently between DesignCanvas and NodeRenderer paths. Prevention: Use placement-utils.ts eventToCanvasPosition everywhere, apply zoom exactly once.

5. **Root Node Type Misidentification** — rootNodeId may not match FreeContainer detection, causing children to get wrong positioning mode. Prevention: Single source of truth for "is canvas root", clear parent type -> child positioning mapping.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Extract Unified placementResolver
**Rationale:** This is the critical path blocker. All other improvements (spatial indexing, snapping, constraints) depend on having single drop resolution. Multiple pitfall reports trace back to scattered entry points.

**Delivers:**
- `placementResolver.ts` with Strategy interface
- `FlexPlacementStrategy`, `FreePlacementStrategy`, `GridPlacementStrategy`
- Refactored DragDropManager as thin wrapper
- `placement-utils.ts` expanded with resolveDropTarget, calculateInsertIndex, calculateGridCell

**Addresses:** Features: Unified placementResolver, coordinate system consistency, fix insertIndex snapshot
**Avoids:** Pitfalls 1 (multiple entry points), 3 (insertIndex staleness), 4 (coordinate inconsistencies)

### Phase 2: Spatial Indexing Infrastructure
**Rationale:** rbush integration requires careful SpatialIndex wrapper around Konva coordinate system. This phase is independent but benefits from Phase 1's clean entry points for integration testing.

**Delivers:**
- `SpatialIndex.ts` wrapper for rbush
- Index all nodes on document change
- Replace hit testing in NodeRenderer
- Replace marquee selection with spatial query

**Uses:** Stack: rbush ^4.0.0
**Avoids:** Pitfall 4 (coordinate system already unified in Phase 1)

### Phase 3: Snapping System
**Rationale:** Snapping requires preview during drag which depends on spatial index for proximity detection. Custom implementation (no library) is mid-complexity and benefits from Phase 1's placement result structure.

**Delivers:**
- `SnapEngine.ts`, `GridSnap.ts`, `EdgeSnap.ts`, `GuideSnap.ts`, `SnapGuide.ts`
- Integration with MoveNodeCommand
- Snap preview during drag
- Visual guide overlay

**Uses:** Stack: Custom snap implementation
**Avoids:** Pitfall 1 (single entry point already established)

### Phase 4: Constraints and Layout Polish
**Rationale:** kiwi.js constraint solver has maintenance risk and may need fallback. This is lower priority and should follow core functionality stabilization.

**Delivers:**
- `LayoutCalculator.ts` with calculateConstraints
- FreeContainer constraint mode refactor
- FlexLayoutCalculator formalization
- GridLayoutCalculator for grid containers

**Uses:** Stack: kiwi.js ^1.0.0 (or improved hand-written solver fallback)
**Avoids:** Pitfall 7 (constraints + fitMode letterbox ambiguity documented)

### Phase Ordering Rationale

1. **Dependencies drive order:** placementResolver is prerequisite for all subsequent integration. Spatial indexing and snapping can theoretically run parallel but benefit from sequential integration.
2. **Risk management:** kiwi.js has maintenance risk so constraints come last. If solver needs rewrite, core UX is already shipped.
3. **Pitfall concentration:** Phases 1 directly addresses 5 critical pitfalls. Each subsequent phase builds on that foundation.
4. **Scale validation:** Spatial indexing benefit only at 100+ nodes. Phase 2 should include measurement to validate full investment.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3 (Snapping):** Threshold UX (8px default) needs user testing for industrial design use case. May need to validate snap distance preferences.
- **Phase 4 (Constraints):** kiwi.js maintenance status uncertain. Need verification before committing. Alternative fallback already documented.

Phases with standard patterns (skip research-phase):
- **Phase 1:** Strategy pattern well-understood, project docs provide clear integration points.
- **Phase 2:** R-tree library (rbush) is standard, npm verification complete.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Based on npm verification, established libraries (rbush), industry standard patterns |
| Features | MEDIUM | Figma/Sketch/XD patterns partially unverified (Figma docs 404s), internal docs HIGH |
| Architecture | HIGH | Based on reading actual source files, clear integration points identified |
| Pitfalls | MEDIUM-HIGH | Based on internal docs and source analysis, LOW confidence on external best practices comparison |

**Overall confidence:** MEDIUM-HIGH

### Gaps to Address

- **kiwi.js maintenance risk:** JS port may need fallback. Verify constraint solver works for keepAspect and conflicting constraints before Phase 4 commitment. If problematic, use improved hand-written solver per layout-system.md algorithm.
- **rbush coordinate system:** Need to verify works with Konva's coordinate system. May need coordinate transformation layer in SpatialIndex wrapper.
- **Performance at scale:** Spatial indexing benefit at typical page sizes (20-50 nodes) unmeasured. Add instrumentation in Phase 2 before full rollout.
- **Snap threshold UX:** Default 8px threshold may need tuning based on industrial design user testing.
- **Figma auto-layout details:** Docs returning 404s during research. Validate single-resolver pattern alignment before implementation.

## Sources

### Primary (HIGH confidence)
- Project internal docs: `docs/designer/placement-and-stacking.md`, `docs/designer/layer-order-convention.md`, `.planning/docs/process/designer/refactor/layout-system.md`, `.planning/docs/process/designer/refactor/schema-design.md`
- Source code analysis: `designer/src/editor-core/utils/placement-utils.ts`, `designer/src/stores/editor/editor-node-layout-helpers.ts`, `designer/src/ui/editors/page/canvas/services/DragDropManager.ts`
- npm verification: rbush, kiwi.js (versions confirmed)

### Secondary (MEDIUM confidence)
- Sketch Canvas/Layer Documentation — `https://www.sketch.com/docs/canvas/`, `https://www.sketch.com/docs/layers/`
- Adobe XD Layers/Components — `https://helpx.adobe.com/xd/help/layers.html`

### Tertiary (LOW confidence — needs validation)
- Figma auto-layout behavior details (docs returning 404s)
- Figma drop zone resolution algorithm
- Industry performance benchmarks for spatial indexing at canvas editor scale

---
*Research completed: 2026-04-15*
*Ready for roadmap: yes*
