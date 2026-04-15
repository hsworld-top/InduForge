# Technology Stack: Canvas Layout Optimization

**Project:** InduForge Designer - Canvas Layout and Interaction Optimization
**Researched:** 2026-04-15
**Focus:** Component placement, layout control, spatial arrangement, layer management
**Mode:** Stack Additions for Canvas Optimization (v0.1 milestone)

---

## Executive Summary

The current designer uses Vue 3 + Konva + Element Plus + GSAP for canvas rendering. The existing codebase already has `placement-utils.ts` (coordinate calculation) and `layout-utils.ts` (layout computation), but these are incomplete for the optimization goals described in `docs/designer/placement-and-stacking.md` and `docs/designer/refactor/layout-system.md`.

This research identifies specific library additions needed to address:
1. Inefficient hit testing (O(n) DOM queries)
2. Complex constraint solving for FreeContainer
3. Lack of snapping infrastructure

---

## Recommended Stack Additions

### 1. Spatial Indexing (Hit Testing Acceleration)

**Library:** `rbush`
**Version:** `^4.0.0`
**Purpose:** Efficient 2D spatial queries for hit testing, marquee selection, and collision detection
**Why:** Current codebase uses `document.querySelector` and DOM `getBoundingClientRect` for hit testing, which is O(n) per query. R-tree provides O(log n) spatial indexing.

**Integration Points:**
- New file: `designer/src/editor-core/spatial/SpatialIndex.ts`
- Refactor: `CanvasContainer.vue` - viewport spatial queries
- Refactor: `DesignCanvas.vue` - marquee selection (`collectMarqueeNodeIds`)
- Refactor: `NodeRenderer.vue` - drop target resolution

**Use Cases:**
- Hit testing when dragging/moving nodes
- Marquee selection intersection queries
- Z-order calculation for overlapping nodes
- Snap-to-edge proximity detection

**Installation:**
```bash
pnpm --dir designer add rbush
npm install -D @types/rbush
```

**Alternative Considered:** `quadrant` (simpler but less performant at scale)

---

### 2. Constraint Solver (For FreeContainer Constraints)

**Library:** `kiwi.js`
**Version:** `^1.0.0`
**Purpose:** Solve constraint equations for FreeContainer's constraint-based positioning (top/right/bottom/left/width/height/keepAspect)
**Why:** Current implementation in `docs/designer/refactor/layout-system.md` Section 3.3 uses hand-written if-else constraint solving. A constraint solver handles edge cases (conflicting constraints, underconstrained systems) more robustly.

**Integration Points:**
- New file: `designer/src/editor-core/layout/ConstraintsSolver.ts`
- Refactor: `layout-utils.ts` - replace `calculateConstraints()` with constraint-based approach
- Affects: `FreeContainer` rendering, fitMode combinations

**Use Cases:**
- FreeContainer with `mode: "constraints"` (not `mode: "abs"`)
- Complex multi-constraint scenarios (left + right + width simultaneously)
- keepAspect ratio enforcement
- Constraints + fitMode combinations

**Installation:**
```bash
pnpm --dir designer add kiwi.js
```

**Note:** If `kiwi.js` proves problematic (JS ports of constraint solvers are often buggy), fallback to improved hand-written solver based on the algorithm in `docs/designer/refactor/layout-system.md`. Do NOT use `cassowary` (unmaintained, problematic JS port).

---

### 3. Snapping System (Custom Implementation)

**Recommendation:** Build custom snap system (no external library)
**Why:** Design tool snapping is highly domain-specific. No general-purpose library provides what design tools need (grid snap, guide-line snap, edge snap, center snap). Figma, Sketch, and Adobe XD all implement their own.

**Suggested Implementation Structure:**
```
designer/src/editor-core/snapping/
  SnapEngine.ts      # Main snap coordinator
  GridSnap.ts         # Grid interval snapping
  GuideSnap.ts        # Alignment guide snapping
  EdgeSnap.ts         # Edge-to-edge snapping
  SnapGuide.ts        # Visual guide overlay component
```

**Core Capabilities:**

| Snap Type | Behavior |
|-----------|----------|
| Grid Snap | Snap to configurable grid intervals |
| Edge Snap | Snap to sibling node edges (left/right/top/bottom) |
| Guide Snap | Snap to user-placed alignment guides |
| Center Snap | Snap to sibling centers (horizontal/vertical) |
| Threshold | Configurable snap distance (default: 8px) |

**Integration Points:**
- `MoveNodeCommand` - apply snapping during move
- `handleCanvasPointerMove` - preview snap during drag
- `NodeRenderer.vue` - render snap guides

**Note:** This is a custom implementation. No npm package for this.

---

## No Changes Needed (Keep Existing)

The following existing libraries serve their purpose and do NOT need replacement:

| Library | Current Version | Purpose | Why Keep |
|---------|-----------------|---------|----------|
| `konva` | ^10.0.12 | Canvas rendering | Provides necessary shape/transform/Layer capabilities |
| `vue-konva` | ^3.2.6 | Vue bindings | Works with current architecture |
| `vue3-grid-layout` | ^1.0.0 | Grid layout | Already used for certain layout modes |
| `vue3-sketch-ruler` | ^2.3.1 | Design rulers | Already in use for ruler overlays |
| `gsap` | ^3.13.0 | Animation | Used for transitions, keep for motion |

---

## Layout Computation: Extend Existing (No New Library)

**Current State:** `layout-utils.ts` already handles:
- `resolveAbsoluteLayout()` - absolute positioning
- `buildFlowResetStyle()` - flow layout reset
- `parseSizeToNumber()` - dimension parsing
- `resolveElContainerMain()` - Element Plus container main
- `clampElContainerPropsBySize()` - container size clamping
- `buildContainerSectionSizePatch()` - section size scaling

**What to Extend (no new library):**

| Extension | Purpose | File |
|-----------|---------|------|
| FlexLayoutCalculator | Proper flexbox algorithm for `FlexContainer` (direction, justify, align, gap, grow, shrink, basis) | `layout-utils.ts` |
| GridLayoutCalculator | CSS Grid-style for `GridContainer` (columns, rows, rowSpan, colSpan) | `layout-utils.ts` |
| LayoutModeDetector | Detect layout mode from node type and props | `layout-utils.ts` |

**Why No Library:**
- The Element Plus and custom layout patterns are specific to this codebase
- Flexbox algorithms are straightforward to implement (already partially done)
- External flexbox libs (like `yoga-layout`) are heavy (C++ based) and overkill

---

## What NOT to Add

| Library | Reason to Avoid |
|---------|----------------|
| `yoga-layout` | C++ based, heavy, designed for React Native not canvas editors |
| `gridstack` | Dashboard grid layout, not suitable for design tool canvas |
| `panzoom` | General pan/zoom, not design snapping (conflicts with existing zoom) |
| `moveable` | React-specific, would require React integration |
| `react-grid-layout` | React-specific |
| `snap.svg` | SVG manipulation, not canvas-based design tools |

---

## Integration Roadmap

### Phase 1: Spatial Indexing (HIGH Priority)
```
Week 1-2:
- Add rbush to designer
- Create SpatialIndex.ts wrapper
- Index all nodes on document change
- Replace hit testing in NodeRenderer
- Replace collectMarqueeNodeIds with spatial query
```

### Phase 2: Snap System (MEDIUM Priority)
```
Week 3-4:
- Create SnapEngine.ts structure
- Implement GridSnap with configurable intervals
- Implement EdgeSnap for sibling alignment
- Integrate with MoveNodeCommand
- Add snap preview during drag
- Add SnapGuide visual overlay
```

### Phase 3: Constraint Solver (MEDIUM Priority)
```
Week 5-6:
- Add kiwi.js
- Create ConstraintsSolver.ts
- Refactor FreeContainer constraint mode
- Handle edge cases (conflicts, underconstrained)
- Test constraints + fitMode combinations
```

### Phase 4: Layout Calculators (LOW Priority)
```
Week 7-8:
- Formalize FlexLayoutCalculator
- Add GridLayoutCalculator
- Test Flex/Grid container nesting
```

---

## Confidence Assessment

| Area | Confidence | Basis |
|------|------------|-------|
| rbush recommendation | HIGH | Standard R-tree implementation, widely used in canvas editors, verified on npm |
| kiwi.js recommendation | MEDIUM | Known constraint solver but JS port less maintained; may need fallback |
| Snap system approach | HIGH | Industry standard - Figma/Sketch/XD all use custom snapping |
| Layout computation | MEDIUM | Based on existing codebase analysis; not exhaustive library comparison |

---

## Open Questions

1. **kiwi.js maintenance risk**: The kiwi.js library has seen limited updates. Verify it works correctly for the constraint use cases (particularly keepAspect and conflicting constraints) before committing. Alternative: improved hand-written solver.

2. **rbush coordinate system**: Need to verify rbush works correctly with Konva's coordinate system (may use different origin than DOM). May need coordinate transformation layer in SpatialIndex wrapper.

3. **Performance at scale**: Spatial indexing benefit only materializes at 100+ nodes. Measure actual performance improvement for typical page sizes (likely 20-50 nodes) before full investment.

4. **snap threshold UX**: Default 8px threshold may need tuning based on user testing for industrial design use case.

---

## Sources

- [rbush npm](https://www.npmjs.com/package/rbush) - Spatial indexing
- [kiwi.js npm](https://www.npmjs.com/package/kiwi.js) - Constraint solver
- [docs/designer/refactor/layout-system.md](./layout-system.md) - Layout mode definitions
- [docs/designer/placement-and-stacking.md](./placement-and-stacking.md) - Current pain points
- Existing codebase analysis: `designer/src/editor-core/utils/placement-utils.ts`, `designer/src/editor-core/utils/layout-utils.ts`
