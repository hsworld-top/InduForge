# Architecture Research: Canvas Layout Integration

**Domain:** Designer canvas editor low-code platform
**Researched:** 2026-04-15
**Confidence:** MEDIUM-HIGH

## Executive Summary

The Designer canvas editor uses a dual-layer architecture: **Vue DOM components** for UI elements and **Konva canvas** for graphics. The current architecture has scattered placement logic, multiple layout helpers, and no unified placement resolver. The milestone goal is to unify component placement and drag-drop interactions with improved layout container implementation.

## Existing Architecture Analysis

### Dual-Layer Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    CanvasContainer.vue                      │
│  ┌─────────────┐  ┌─────────────────────────────────────┐  │
│  │ RulerLayer  │  │         DesignCanvas                │  │
│  │  (Konva)    │  │  ┌───────────────────────────────┐  │  │
│  └─────────────┘  │  │     NodeRenderer (Vue DOM)     │  │  │
│                   │  │  ┌───────────────────────────┐ │  │  │
│                   │  │  │  [data-node-id] Components │ │  │  │
│                   │  │  │  FlexContainer            │ │  │  │
│                   │  │  │  FreeContainer            │ │  │  │
│                   │  │  │  GridContainer           │ │  │  │
│                   │  │  └───────────────────────────┘ │  │  │
│                   │  └───────────────────────────────┘  │  │
│                   │  ┌───────────────────────────────┐  │  │
│                   │  │  Konva GraphicsLayer           │  │  │
│                   │  │  (Canvas.Line, Pipe, Symbol)   │  │  │
│                   │  └───────────────────────────────┘  │  │
│                   └─────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Component Boundaries

| Component | File | Responsibility |
|-----------|------|----------------|
| CanvasContainer | `CanvasContainer.vue` | Rulers, zoom, pan, drag coordination |
| DesignCanvas | `DesignCanvas.vue` | Canvas root, DOM mounting |
| NodeRenderer | `NodeRenderer.vue` | Recursive DOM node rendering |
| CanvasInsertLineOverlay | `CanvasInsertLineOverlay.vue` | Insert position indicator |
| DragDropManager | `services/DragDropManager.ts` | Drop target resolution |
| use-node-drop | `composables/use-node-drop.ts` | Drag-over/drop handling |
| use-canvas-viewport-placement | `composables/use-canvas-viewport-placement.ts` | Viewport centering |

### Document Model (Schema)

```typescript
// Flat structure - nodes stored by ID, parent-child via children array
interface ComponentNode {
  id: string;
  type: string;                    // 'FlexContainer' | 'FreeContainer' | 'GridContainer' | etc.
  props: Record<string, any>;
  style: Record<string, any>;
  layoutItem: LayoutItem | null;   // How this node participates in layout
  children: string[];               // Child node IDs
  // ... bindings, permissions, events
}

// LayoutItem determines layout behavior
type LayoutItem =
  | { flex: { grow: number; shrink: number; basis: string } }
  | { free: { mode: 'abs'; abs: { x, y, w, h, z } } | { mode: 'constraints'; constraints: {...} } }
  | { grid: { row: number; col: number; rowSpan: number; colSpan: number } };
```

### Current Data Flow for Component Placement

```
User drags component
        │
        ▼
DragEvent + canvasRoot element
        │
        ▼
DragDropManager.resolveDropTarget()
  - elementFromPoint to find target
  - Walk up tree to find container
  - Determine container type (flex/grid/free)
        │
        ▼
  For Flex: calculateFlexInsertPosition()
    - Get DOM rect of children
    - Determine before/after/inside position
        │
        ▼
  For Free: calculateFreePosition()
    - Return x,y relative to container
        │
        ▼
InsertNodeCommand.execute()
  - Creates node with appropriate layoutItem
  - Updates doc.nodesById
        │
        ▼
NodeRenderer reacts to docVersion change
  - Re-renders component tree
```

## Integration Points

### 1. Entry Points for Layout Improvements

| Point | File | Method/Property | Current Behavior |
|-------|------|------------------|------------------|
| Drop resolution | `DragDropManager.ts` | `resolveDropTarget()` | Returns parentId, index, containerType |
| Drop decision | `DragDropManager.ts` | `calculateDropDecision()` | Switch on nodeType, returns DropDecision |
| Insert position | `DragDropManager.ts` | `calculateFlexInsertPosition()` | DOM rect-based midpoint calculation |
| Free position | `DragDropManager.ts` | `calculateFreePosition()` | Simple clientX - rect.left |
| Layout item build | `editor-node-layout-helpers.ts` | `buildLayoutItem()` | Type-switch to build flex/free/grid item |
| Coordinate conversion | `placement-utils.ts` | `eventToCanvasPosition()` | Client to canvas coords with zoom |
| Insert line rendering | `CanvasInsertLineOverlay.vue` | `insertLineStyle` | Shows horizontal/vertical line |

### 2. Existing Layout Utilities

| File | Functions | Purpose |
|------|-----------|---------|
| `placement-utils.ts` | `eventToCanvasPosition()`, `clampPositionInContainer()` | Coordinate conversion |
| `layout-utils.ts` | `resolveAbsoluteLayout()`, `buildFlowResetStyle()`, `parseSizeToNumber()`, `clampElContainerPropsBySize()` | Layout computation |
| `editor-node-layout-helpers.ts` | `buildFlexLayoutItem()`, `buildFreeLayoutItem()`, `buildGridLayoutItem()`, `resolveDefaultSize()` | Layout item construction |
| `layout-manifests-flexbox.ts` | FlexContainer manifest | Component descriptor |
| `layout-manifests-page-layouts.ts` | Page layout manifests | Component descriptors |

### 3. Store Integration

| Store Property | Type | Purpose |
|----------------|------|---------|
| `doc` | `DocumentModel` | Schema storage, node CRUD |
| `docVersion` | `number` | Triggers reactivity on changes |
| `selection` | `SelectionModel` | Selected elements |
| `history` | `History` | Undo/redo stack |
| `canvasMousePos` | `{x,y}` | Current mouse position |
| `hoveredNodeType` | `string` | Type of hovered node |

## Layout Container Implementation Status

### FlexContainer
- **Schema**: `layoutItem: { flex: { grow, shrink, basis } }`
- **Manifest**: Defined in `layout-manifests-flexbox.ts`
- **Rendering**: Via NodeRenderer recursive traversal
- **Insert Logic**: `calculateFlexInsertPosition()` based on DOM children
- **Status**: Basic implementation exists

### FreeContainer
- **Schema**: `layoutItem: { free: { mode: 'abs', abs: {x,y,w,h,z} } | { mode: 'constraints', constraints: {...} } }`
- **Positioning**: `absolutePos` on node + `layoutItem.free.abs`
- **Constraints**: Documented in `docs/designer/refactor/layout-system.md`
- **Status**: abs mode exists, constraints mode documented but not fully implemented

### GridContainer
- **Schema**: `layoutItem: { grid: { row, col, rowSpan, colSpan } }`
- **Manifest**: Defined in layout manifests
- **Insert Logic**: `calculateGridCell()` - divides container into grid cells
- **Status**: Basic implementation exists

## New vs Modified Components

### New Components Needed

| Component | Location | Purpose |
|-----------|----------|---------|
| `placementResolver.ts` | `editor-core/utils/` | Unified drop target resolution |
| `LayoutCalculator.ts` | `editor-core/utils/` | Constraints calculation per design doc |
| `InsertLineRenderer.ts` | `ui/editors/page/canvas/` | Dedicated insert line rendering (if Vue overlay insufficient) |
| `DragFeedback.ts` | `ui/editors/page/canvas/` | Drag preview ghost element |

### Modified Components

| Component | Change |
|-----------|--------|
| `DragDropManager.ts` | Extract `calculateDropDecision` into `placementResolver` |
| `use-node-drop.ts` | Use new `placementResolver` instead of DragDropManager directly |
| `editor-node-layout-helpers.ts` | Add constraints calculation |
| `placement-utils.ts` | Expand with more coordinate utilities |
| `CanvasInsertLineOverlay.vue` | Support for grid cell highlighting |

## Data Flow Changes

### Current Flow (Scattered)

```
DragEvent
    │
    ▼
DragDropManager.calculateDropDecision()  ◄── Type switch statement
    │
    ▼
InsertNodeCommand
```

### Proposed Flow (Unified)

```
DragEvent
    │
    ▼
placementResolver.resolve()  ◄── Single entry point
    │
    ├──► FlexLayoutStrategy.calculate()
    ├──► FreeLayoutStrategy.calculate()
    └──► GridLayoutStrategy.calculate()
    │
    ▼
PlacementResult { parentId, index, dropPosition, containerType, visualHint }
    │
    ▼
InsertNodeCommand / MoveNodeCommand
```

### Visual Hint Structure

```typescript
interface PlacementVisualHint {
  type: 'insert-line' | 'grid-cell' | 'free-position' | 'none';
  // For insert-line:
  orientation?: 'horizontal' | 'vertical';
  offset?: number;
  index?: number;
  // For grid-cell:
  row?: number;
  col?: number;
  highlightRect?: { left, top, width, height };
  // For free-position:
  x?: number;
  y?: number;
}
```

## Suggested Build Order (Considering Dependencies)

### Phase 1: Foundation (No dependencies)
1. **Extract `placementResolver.ts`** - Move drop decision logic from DragDropManager
   - Create `PlacementStrategy` interface
   - Implement `FlexPlacementStrategy`, `FreePlacementStrategy`, `GridPlacementStrategy`
   - Keep DragDropManager as thin wrapper during transition

2. **Expand `placement-utils.ts`**
   - Add `resolveDropTarget()` combining eventToCanvasPosition + clampPositionInContainer
   - Add `calculateInsertIndex()` for flex containers
   - Add `calculateGridCell()` for grid containers

**New file**: `designer/src/editor-core/utils/placement-resolver.ts`
**Modified**: `placement-utils.ts`, `DragDropManager.ts`

### Phase 2: Layout Item Builders (Depends on Phase 1)
3. **Enhance `editor-node-layout-helpers.ts`**
   - Add `buildFreeLayoutItemWithConstraints()` helper
   - Add `resolveLayoutItemForContainer()` - auto-detect correct layout type

4. **Add `LayoutCalculator.ts`**
   - Implement `calculateConstraints()` per design doc
   - Implement `calculateAbsPosition()`
   - Implement `calculateFit()` for fitMode

**New file**: `designer/src/editor-core/utils/layout-calculator.ts`
**Modified**: `editor-node-layout-helpers.ts`

### Phase 3: UI Integration (Depends on Phase 1-2)
5. **Update `use-node-drop.ts`**
   - Replace direct DragDropManager calls with `placementResolver`
   - Support new visual hint types

6. **Enhance `CanvasInsertLineOverlay.vue`**
   - Support grid cell highlighting mode
   - Clean up insert line rendering logic

**Modified**: `use-node-drop.ts`, `CanvasInsertLineOverlay.vue`

### Phase 4: Polish & Edge Cases (Depends on all)
7. **Add `DragFeedback.ts`**
   - Ghost element during drag
   - Snap indicators

8. **Handle boundary cases**
   - Empty container drop
   - Nested container drop
   - Cross-container move

**New file**: `designer/src/ui/editors/page/canvas/components/DragFeedback.vue`
**Modified**: `CanvasContainer.vue`, `use-node-drop.ts`

## Architecture Patterns to Follow

### Strategy Pattern for Placement
```typescript
interface PlacementStrategy {
  canHandle(containerType: string): boolean;
  resolve(
    event: MouseEvent | DragEvent,
    containerElement: HTMLElement,
    containerNode: ComponentNode,
    zoom: number
  ): PlacementResult;
}
```

### Command Pattern for Layout Changes
```typescript
class SetLayoutItemCommand implements Command {
  constructor(
    private nodeId: string,
    private newLayoutItem: LayoutItem
  ) {}
  execute(doc: DocumentModel): void { /* ... */ }
  undo(doc: DocumentModel): void { /* ... */ }
}
```

### Computed Properties for Derived Layout
```typescript
const computedBounds = computed(() => {
  if (!node.value || !parentSize.value) return null;
  return calculateConstraints(parentSize.value, node.value.layoutItem?.free?.constraints);
});
```

## Anti-Patterns to Avoid

1. **Don't put layout calculation in store actions** - Keep store focused on state, extract logic to utils
2. **Don't use DOM measurements for layout decisions in batch** - Cache measurements, don't re-measure on every frame
3. **Don't mix coordinate systems** - Clearly separate: client coords, canvas coords, container-relative coords
4. **Don't extend DragDropManager with new switch cases** - Add new strategies instead

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Architecture analysis | HIGH | Based on reading actual source files |
| Integration points | HIGH | Clear entry points identified in source |
| Build order | MEDIUM | Dependency analysis sound, but actual implementation may reveal gaps |
| Schema design | HIGH | Documented in schema-design.md and editor-core.md |

## Open Questions

1. **Konva usage extent**: Current grep shows "canvas layer" references but no direct Konva Stage usage found. Need to verify if Konva is actively used for graphics rendering or if it's prepared but not yet integrated.
2. **Constraints implementation completeness**: The layout-system.md documents constraints calculation, but is it fully implemented in the runtime?
3. **GridContainer rendering**: Is grid layout applied via CSS grid or custom calculation?
4. **fitMode implementation**: Is contain/cover/stretch implemented in designer preview, runtime, or both?

## Sources

- `designer/src/editor-core/index.ts` - Editor core exports
- `designer/src/editor-core/utils/placement-utils.ts` - Current placement utilities
- `designer/src/editor-core/utils/layout-utils.ts` - Layout computation utilities
- `designer/src/stores/editor/editor-node-layout-helpers.ts` - Layout item builders
- `designer/src/ui/editors/page/canvas/services/DragDropManager.ts` - Drop target resolution
- `designer/src/ui/editors/page/canvas/composables/use-node-drop.ts` - Drop handling composable
- `designer/src/ui/editors/page/canvas/composables/use-canvas-viewport-placement.ts` - Viewport placement
- `designer/src/ui/editors/page/canvas/CanvasContainer.vue` - Canvas container
- `docs/designer/refactor/layout-system.md` - Layout system design
- `docs/designer/refactor/schema-design.md` - Schema v2 design
- `docs/designer/refactor/editor-core.md` - Editor core architecture
