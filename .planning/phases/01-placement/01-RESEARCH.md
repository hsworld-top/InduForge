# Phase 1: 放置与拖放统一 - Research

**Researched:** 2026-04-15
**Domain:** Vue 3 Canvas Editor - Drag-and-Drop Placement Unification
**Confidence:** HIGH

## Summary

Phase 1 focuses on unifying component placement and drag-drop logic across the Designer canvas. The core problem is that placement logic is currently scattered across `DesignCanvas.vue` (handleCanvasDrop), `NodeRenderer.vue` (handleDrop), `use-node-drop.ts` composable, and `DragDropManager.ts`, leading to inconsistent behavior where "same visual drop point produces different logical positions."

The solution requires creating a single `placementResolver.ts` entry point using the Strategy pattern to handle different container types (flex/free/grid), while unifying coordinate calculation through `placementUtils.ts`.

**Primary recommendation:** Extract `placementResolver.ts` as a single source of truth for container hit detection and insertIndex calculation, then wire both `DesignCanvas.handleCanvasDrop` and `NodeRenderer.handleDrop` to use it.

## User Constraints (from CONTEXT.md)

### Locked Decisions

1. **Drop target resolution:** Root-level container primary, depth-first secondary
2. **insertIndex:** Y-first + append fallback + nearest neighbor fine-tuning
3. **Drag visual feedback:** Ghost + insert indicator line + container highlight
4. **Zoom/Scroll:** Unified entry via `placementUtils.eventToCanvasPosition`, nested zoom support, scroll compensation

### Claude's Discretion (pending confirmation)

- Ghost transparency value (suggest 0.5 alpha)
- Insert line style (color, border width)
- Container highlight color and border style (D-09 suggests dashed border)

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PLACE-01 | Unify component placement entry, merge DesignCanvas.handleCanvasDrop with NodeRenderer.handleDrop | `placementResolver.ts` becomes single entry; current scattered logic consolidated |
| PLACE-02 | Unify canvas coordinate calculation, same zoom/scroll handling | `placement-utils.ts` provides `eventToCanvasPosition` (existing) with D-10 enforcement |
| PLACE-03 | Fix insertIndex staleness in fast drag scenarios | Y-first + append fallback + nearest neighbor (D-04 to D-06) ensures robustness |
| TECH-01 | Extract placementResolver.ts single entry, Strategy pattern per container type | Architecture section details Strategy pattern implementation |
| TECH-02 | Unify placementUtils coordinate functions | Consolidate `eventToCanvasPosition`, `clampPositionInContainer` into single module |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Vue 3 | 3.5.24 | UI framework | Primary framework per designer/AGENTS.md |
| Vitest | 2.1.8 | Testing framework | Existing test infra in designer, verified via package.json |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| @vue/test-utils | 2.4.6 | Vue component testing | Unit testing Vue components |
| jsdom | (via Vitest) | DOM environment | Test canvas interactions |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Jest | Vitest | Vitest already in project, faster HMR, native Vite support |
| @testing-library/vue | @vue/test-utils | Both viable, @vue/test-utils sufficient for this scope |

**Installation verification:**
```bash
npm view vue version      # 3.5.32
npm view vitest version   # 4.1.4 (project uses 2.1.8)
npm view @vue/test-utils version  # 2.5.7 (project uses 2.4.6)
```

## Architecture Patterns

### Recommended Project Structure

```
designer/src/
├── editor-core/
│   └── utils/
│       ├── placement-utils.ts       # EXISTING - coordinate utils (expand)
│       └── placement-resolver.ts    # NEW - Strategy pattern single entry
├── ui/editors/page/canvas/
│   ├── services/
│   │   └── DragDropManager.ts       # EXISTING - drop decision logic
│   ├── composables/
│   │   ├── use-node-drop.ts         # EXISTING - drop state management
│   │   └── use-drag-state.ts        # EXISTING - drag state
│   ├── DesignCanvas.vue             # REFACTOR - use placementResolver
│   └── NodeRenderer.vue             # REFACTOR - use placementResolver
```

### Pattern 1: Strategy Pattern for Container Types

**What:** A single `placementResolver` that delegates to type-specific strategies for flex, free, and grid containers.

**When to use:** Container hit detection and insertIndex calculation for different layout types.

**Interface design (TypeScript):**
```typescript
// Source: Adapted from DragDropManager.ts existing patterns
export interface PlacementStrategy {
  /** Container type this strategy handles */
  readonly containerKind: 'flex' | 'free' | 'grid';
  /** Whether this container can accept the given child type */
  canAcceptChild(parentNode: ComponentNode, childType: string): boolean;
  /** Calculate insertIndex from mouse event position */
  resolveInsertIndex(
    event: DragEvent,
    containerNode: ComponentNode,
    containerElement: HTMLElement,
    zoom: number
  ): number;
  /** Get drop position for absolute containers (free layout) */
  resolveDropPosition?(
    event: DragEvent,
    containerElement: HTMLElement,
    zoom: number
  ): { x: number; y: number };
  /** Visual feedback hints (insert line, ghost position) */
  getVisualHint(...): InsertLineHint | GridCellHint | null;
}

export interface PlacementResolution {
  parentId: string;
  insertIndex: number;
  dropPosition: { x: number; y: number } | null;
  containerKind: 'flex' | 'free' | 'grid';
  visualHint: InsertLineHint | GridCellHint | null;
}

// Single entry function
export function resolvePlacement(
  event: DragEvent,
  canvasRoot: HTMLElement,
  doc: CanvasDocLike,
  currentPage: PageNode,
  zoom: number,
  options?: { preferRootLevel?: boolean; depthFirstFallback?: boolean }
): PlacementResolution;
```

**Anti-Pattern to Avoid:** Current scattered implementation where:
- `DesignCanvas.handleCanvasDrop` has its own `resolveLayoutFromPoint()`
- `use-node-drop.ts` has `resolveDropContainer()`, `resolveLayoutInsertFromPoint()`, etc.
- `DragDropManager.resolveDropTarget()` duplicates some logic

### Pattern 2: Y-First + Append Fallback + Nearest Neighbor

**What:** insertIndex calculation follows Y-coordinate priority, falls back to append for stale indices, then fine-tunes with nearest neighbor.

**When to use:** Determining exact insertion position among sibling nodes.

**Algorithm (pseudo-code):**
```typescript
// Source: Based on D-04, D-05, D-06 decisions
function calculateInsertIndex(
  event: DragEvent,
  siblings: ComponentNode[],
  containerElement: HTMLElement,
  zoom: number
): number {
  // 1. Sort siblings by Y (top), then X
  const sorted = [...siblings].sort((a, b) => {
    const posA = getNodeVisualTop(a);
    const posB = getNodeVisualTop(b);
    return posA.y - posB.y || posA.x - posB.x;
  });

  // 2. Find nearest neighbor based on Y+X distance
  const dropPos = eventToCanvasPosition(event, containerElement, zoom);
  let nearestIdx = findNearestByYPlusX(dropPos, sorted);

  // 3. Append fallback if nearestIdx references deleted node
  if (!isValidIndex(nearestIdx, siblings)) {
    nearestIdx = siblings.length; // append
  }

  return nearestIdx;
}
```

### Pattern 3: Unified Coordinate Entry

**What:** All coordinate calculations route through `placement-utils.ts`.

**When to use:** Any mouse event to canvas coordinate conversion.

**Key functions (existing in placement-utils.ts):**
```typescript
// Source: designer/src/editor-core/utils/placement-utils.ts
export function eventToCanvasPosition(
  event: MouseEvent | DragEvent,
  containerElement: HTMLElement,
  zoom = 1,
): CanvasPoint {
  // Already exists - enforces D-10
}

export function clampPositionInContainer(
  position: CanvasPoint,
  containerElement: HTMLElement,
  size: PlacementSize,
  zoom = 1,
): CanvasPoint {
  // Already exists - useful for free containers
}
```

**Issue identified:** `use-node-drop.ts` line 29-30 imports `eventToCanvasPosition` from placement-utils but `DesignCanvas.handleCanvasDrop` (line 442) also calls it directly. Both paths work but the issue is they use different containers and zoom values inconsistently.

### Pattern 4: Ghost + Insert Line + Container Highlight

**What:** Three-layer visual feedback during drag operations.

**When to use:** During active drag-over state.

**Implementation approach:**
- Ghost: Semi-transparent clone of dragged component (opacity ~0.5)
- Insert line: Horizontal/vertical line at calculated insertIndex position
- Container highlight: Dashed border on target container (distinguishes from solid selection border)

**Specific ideas from discuss-phase:**
- Insert indicator line should display beside nearest neighbor, not overlap nodes
- Container highlight uses dashed border (not solid like selection)

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Container hit detection | Custom elementFromPoint wrapping | Leverage `DragDropManager.resolveDropTarget()` | Already handles root-level priority |
| Flex insertIndex calculation | Inline mid-point comparison | `DragDropManager.calculateFlexInsertPosition()` | Already exists with direction awareness |
| Grid cell calculation | Manual grid-template parsing | `parseGridTemplateParts()` + `calculateGridCell()` | Already tested via parse-grid-template.test.ts |
| Scroll offset compensation | Manual scroll calculations | `getBoundingClientRect()` + zoom division | placement-utils handles this |

**Key insight:** DragDropManager already has substantial logic for drop target resolution. The refactor should wrap/extend it rather than replace it wholesale.

## Common Pitfalls

### Pitfall 1: Mixed Coordinate Systems (page vs client)

**What goes wrong:** Mouse events use client coordinates; canvas uses page/logical coordinates; mixing causes offset bugs.

**Why it happens:** `event.clientX/clientY` vs `event.pageX/pageY` vs calculated canvas coordinates all differ based on scroll and zoom.

**How to avoid:**
- Always use `eventToCanvasPosition()` for mouse-to-canvas conversion
- D-10 mandates single entry: no local zoom/scroll calculations
- Test with canvas scrolled and zoomed

**Warning signs:** Dropped component appears offset from cursor; works at 1x zoom but not 0.5x

### Pitfall 2: insertIndex Staleness (PLACE-03)

**What goes wrong:** Fast drag-drop: insertIndex calculated at dragover but siblings change before drop event fires.

**Why it happens:** DOM mutations (other drag operations, animations) modify children array between events.

**How to avoid:**
- D-05: Append fallback when target node doesn't exist
- Validate insertIndex against current children length before use
- Consider using node ID references instead of array indices

**Warning signs:** Component inserted at wrong position; insert at index 3 goes to index 0

### Pitfall 3: Deep Nesting vs Root-Level Container Conflict

**What goes wrong:** D-01 and D-02 can conflict; depth-first finds nested container but root-level should be primary.

**Why it happens:** Container hierarchy creates ambiguity when hovering near nested containers.

**How to avoid:**
- D-01 primary: root-level container direct children only
- D-02 fallback: depth-first only when root-level logic unclear
- Explicit priority: prefer shallowest valid container

**Warning signs:** Drop target changes unexpectedly when moving mouse slightly

### Pitfall 4: ElLayout/Auto-Insert Edge Cases

**What goes wrong:** ElLayout auto-creates ElLayoutRow; ElLayoutRow auto-creates ElCol; these auto-insertions conflict with manual drop.

**Why it happens:** `canAcceptByAutoInsert` logic in use-node-drop.ts handles this but has gaps.

**How to avoid:**
- When dropping on ElLayout (not ElLayoutRow), auto-create row
- When dropping on ElLayoutRow (not ElCol), auto-create column
- Validate auto-insertion chain completes before returning success

### Pitfall 5: Missing Scroll Compensation (D-12)

**What goes wrong:** `getBoundingClientRect()` returns viewport-relative rect, not accounting for scroll offset within scrollable containers.

**Why it happens:** Scrollable canvas containers have offset between viewport and content coordinates.

**How to avoid:**
- Use `containerElement.getBoundingClientRect()` which IS scroll-aware
- For nested scroll containers, accumulate scroll offsets
- `placement-utils.ts` `eventToCanvasPosition` does NOT currently handle scroll - needs enhancement

**Warning signs:** Canvas with scrollbar: drops appear offset horizontally/vertically

## Code Examples

### Example 1: Extending placement-utils.ts for Scroll Compensation

```typescript
// Source: Based on D-12 requirement
// ENHANCEMENT NEEDED: Add scroll-aware position calculation
export function eventToCanvasPosition(
  event: MouseEvent | DragEvent,
  containerElement: HTMLElement,
  zoom = 1,
): CanvasPoint {
  if (!containerElement || typeof event.clientX !== "number") {
    return { x: 0, y: 0 };
  }
  const rect = containerElement.getBoundingClientRect();

  // Current: no scroll compensation
  const x = (event.clientX - rect.left) / zoom;
  const y = (event.clientY - rect.top) / zoom;

  // D-12 requirement: compensate for scroll offset
  // Note: getBoundingClientRect does NOT include scroll - this is the bug
  const scrollLeft = containerElement.scrollLeft || 0;
  const scrollTop = containerElement.scrollTop || 0;

  return {
    x: Math.max(0, Math.round((event.clientX - rect.left + scrollLeft) / zoom)),
    y: Math.max(0, Math.round((event.clientY - rect.top + scrollTop) / zoom)),
  };
}
```

### Example 2: Strategy Pattern Skeleton for placementResolver

```typescript
// Source: Based on DragDropManager.ts patterns + D-03 decision
import { getChildPositioning, isContainerType } from "@/components/descriptors/registry";

interface InsertLineHint {
  orientation: 'horizontal' | 'vertical';
  offset: number;
  lineBox: { left: number; top: number; width: number; height: number };
}

abstract class ContainerStrategy {
  abstract readonly containerKind: 'flex' | 'free' | 'grid';
  abstract resolveInsertIndex(event: DragEvent, containerNode: ComponentNode, containerElement: HTMLElement, zoom: number): number;
  abstract getVisualHint(event: DragEvent, containerNode: ComponentNode, containerElement: HTMLElement, zoom: number): InsertLineHint | null;
}

class FlexContainerStrategy extends ContainerStrategy {
  readonly containerKind = 'flex';

  resolveInsertIndex(event: DragEvent, containerNode: ComponentNode, containerElement: HTMLElement, zoom: number): number {
    // Use DragDropManager.calculateFlexInsertPosition logic
    // Sort by Y then X (D-04), find midpoint crossings
  }

  getVisualHint(event: DragEvent, containerNode: ComponentNode, containerElement: HTMLElement, zoom: number): InsertLineHint {
    // Calculate line position at insertIndex
  }
}

class FreeContainerStrategy extends ContainerStrategy {
  readonly containerKind = 'free';

  resolveInsertIndex(): number {
    return (containerNode.children || []).length; // Always append for absolute
  }

  getVisualHint(): InsertLineHint | null {
    return null; // No insert line for free containers
  }
}

// Single entry point
export function resolvePlacement(
  event: DragEvent,
  canvasRoot: HTMLElement,
  doc: CanvasDocLike,
  currentPage: PageNode,
  zoom: number,
): PlacementResolution {
  // 1. Find target container (root-level primary, depth-first fallback)
  // 2. Select strategy based on container type
  // 3. Calculate insertIndex using strategy
  // 4. Return resolution with visual hints
}
```

### Example 3: insertIndex with Append Fallback

```typescript
// Source: Based on D-05 requirement
function resolveInsertIndexWithFallback(
  targetIndex: number,
  siblings: string[],
  doc: CanvasDocLike
): number {
  // D-05: Append fallback when insertIndex is stale
  if (targetIndex < 0 || targetIndex > siblings.length) {
    return siblings.length; // append
  }

  // Validate: if index references non-existent node, append
  const nodeAtIndex = doc.getNode(siblings[targetIndex]);
  if (!nodeAtIndex) {
    return siblings.length;
  }

  // D-06: Nearest neighbor fine-tuning could adjust index
  // by checking actual DOM positions of adjacent nodes
  return targetIndex;
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Scattered drop handlers | Unified placementResolver | This phase (v0.1) | Consistent behavior |
| Local zoom calculations | placement-utils single entry | This phase (v0.1) | D-10 compliance |
| Index-only insertIndex | Y-first + append + nearest | This phase (v0.1) | Fix PLACE-03 |

**Deprecated/outdated:**
- `resolveLayoutFromPoint()` in DesignCanvas.vue - replaced by placementResolver
- Inline `eventToCanvasPosition` calls without scroll compensation - replaced by enhanced placement-utils

## Assumptions Log

> List all claims tagged `[ASSUMED]` in this research. Planner and discuss-phase use this section.

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Ghost opacity 0.5 | Visual Feedback | User preference may differ |
| A2 | Insert line "beside neighbor, not overlapping" | Common Pitfalls | May need to overlap to be visible |
| A3 | Dashed border for container highlight | Visual Feedback | Solid border may be more visible |

## Open Questions

1. **Ghost rendering approach**
   - What we know: Need semi-transparent component outline following cursor
   - What's unclear: Use CSS clone, Konva ghost layer, or DOM overlay
   - Recommendation: DOM overlay with `pointer-events: none` is simplest

2. **Nested zoom calculation**
   - What we know: D-11 mentions nested zoom (canvas + container-level)
   - What's unclear: Current implementation doesn't support container-level zoom
   - Recommendation: Defer container-level zoom to future phase; focus on canvas-level first

3. **Strategy strategy selection for hybrid containers**
   - What we know: Some containers may have mixed positioning modes
   - What's unclear: How to detect and handle containers that are both flex and absolute
   - Recommendation: Use `getChildPositioning()` as primary determinant

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | Designer dev server | Yes | 22+ | — |
| pnpm | Package management | Yes | 9+ | — |
| Vitest | Testing | Yes | 2.1.8 | — |
| jsdom | DOM simulation | Yes (via Vitest) | — | — |

**Missing dependencies with no fallback:**
- None identified

**Missing dependencies with fallback:**
- None identified

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 2.1.8 |
| Config file | designer/vitest.config.js |
| Quick run command | `pnpm --dir designer test` |
| Full suite command | `pnpm --dir designer test --run` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| PLACE-01 | Unified placement entry | Integration | `pnpm --dir designer test --run -- src/**/placement*.test.ts` | No - create |
| PLACE-02 | Unified coordinate calculation | Unit | `pnpm --dir designer test --run -- src/editor-core/utils/placement-utils.test.ts` | No - create |
| PLACE-03 | insertIndex fallback | Unit | `pnpm --dir designer test --run -- src/**/insert-index*.test.ts` | No - create |
| TECH-01 | Strategy pattern | Unit | `pnpm --dir designer test --run -- src/editor-core/utils/placement-resolver.test.ts` | No - create |
| TECH-02 | Coordinate unification | Unit | Same as PLACE-02 | No - create |

### Sampling Rate
- **Per task commit:** `pnpm --dir designer test --run` (full suite, ~30s)
- **Per wave merge:** Full suite green
- **Phase gate:** All phase tests green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `designer/src/editor-core/utils/placement-utils.test.ts` — covers PLACE-02, TECH-02
- [ ] `designer/src/editor-core/utils/placement-resolver.test.ts` — covers TECH-01
- [ ] `designer/src/ui/editors/page/canvas/services/placement.integration.test.ts` — covers PLACE-01, PLACE-03
- [ ] `designer/src/ui/editors/page/canvas/services/drag-drop-manager.test.ts` — covers existing DragDropManager behavior

## Security Domain

**Note:** This phase is UI-only with no network, authentication, or data persistence changes. Security domain does not apply.

### Applicable ASVS Categories
N/A - Phase 1 is pure client-side drag-drop UI with no security-relevant changes.

## Sources

### Primary (HIGH confidence)
- `designer/src/editor-core/utils/placement-utils.ts` — existing coordinate utilities
- `designer/src/ui/editors/page/canvas/services/DragDropManager.ts` — existing drop target resolution
- `designer/src/ui/editors/page/canvas/composables/use-node-drop.ts` — existing drop composable
- `designer/src/ui/editors/page/canvas/DesignCanvas.vue` — existing canvas drop handler
- `designer/package.json` — confirmed vitest 2.1.8, vue 3.5.24

### Secondary (MEDIUM confidence)
- [Vitest docs](https://vitest.dev/) — testing patterns
- [Vue 3 docs](https://vuejs.org/) — Composition API patterns

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — confirmed via package.json and existing tests
- Architecture: HIGH — based on existing code patterns and discuss-phase decisions
- Pitfalls: HIGH — identified from codebase analysis and known drag-drop edge cases

**Research date:** 2026-04-15
**Valid until:** 2026-05-15 (30 days — stable domain, no fast-moving tech)
