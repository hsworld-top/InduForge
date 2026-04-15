# Feature Landscape: Designer Canvas Editor Optimization

**Domain:** Low-code canvas editor component placement, layout control, and layer management
**Project:** InduForge Designer (v0.1 milestone)
**Researched:** 2026-04-15
**Confidence:** MEDIUM

---

## Context Summary

InduForge Designer is a low-code canvas editor with:
- **Canvas + DOM dual-layer architecture** (Canvas for graphics/pipes, DOM for interactive components)
- **Three layout container types**: FreeContainer (absolute positioning), FlexContainer, Grid
- **Stacking order**: `parent.children` array order determines z-index (later = higher)
- **Existing issues** (from `docs/designer/placement-and-stacking.md`):
  - Multiple drop entry points causing inconsistent placement
  - Coordinate calculation inconsistencies between DesignCanvas and NodeRenderer
  - insertIndex snapshot branching logic causing wrong parent/index
  - Root node type ambiguity affecting absolute vs flow positioning

**Research question:** How do professional canvas editors (Figma, Sketch, Adobe XD) handle these patterns, and what are the expected behaviors?

---

## Table Stakes Features

Features users expect from any professional canvas editor. Missing these = product feels incomplete or broken.

### 1. Single Unified Drop Target Resolution

| Aspect | Detail |
|--------|--------|
| **Feature** | Drag-and-drop placement with consistent drop target resolution |
| **Why Expected** | Users drag components expecting predictable placement. Multiple entry points cause "same drop, different result" bugs |
| **Complexity** | High |
| **Dependencies** | Requires centralized `placementResolver` |

**How professional editors handle this:**

| Editor | Approach |
|--------|----------|
| **Figma** | Single drop handler per frame. Drop target determined by hover position at drop time. Auto-layout frames show blue drop indicator. Components placed at exact cursor position or snapped to nearest grid |
| **Sketch** | Layers panel order = z-index. Drop on artboard = new layer at that position. Drop on group = nested layer |
| **Adobe XD** | Artboard as root container. Components placed at drop point with responsive constraints auto-calculated |

**InduForge current issue:** `DesignCanvas.handleCanvasDrop` and `NodeRenderer.handleDrop` are separate entry points with different coordinate calculations.

---

### 2. Layout Container Drop Zone Behavior

| Feature | Why Expected | Complexity | Dependencies |
|---------|--------------|------------|--------------|
| **Visual drop indicator** | Users need to see WHERE component will land | Medium | Requires `placementResolver` integration |
| **Container acceptance rules** | Not all components can go in all containers (e.g., Grid requires specific children types) | Medium | Manifest/component definition |
| **Insert index visualization** | Show which index the component will be inserted at | Medium | DOM ordering simulation |
| **Nested container handling** | Drop on a nested container should place INSIDE it, not at parent level | High | Recursive container detection |

**How professional editors handle this:**

| Editor | Drop Zone Behavior |
|--------|-------------------|
| **Figma** | Auto-layout frames show: (1) Blue border when drag enters, (2) Spacing indicators between existing children, (3) "Drop here" overlay when empty. Non-auto-layout frames: snaps to edges |
| **Sketch** | Drop zones highlight with blue border. Smart guides appear showing alignment to nearby layers |
| **Adobe XD** | Repeat grid shows drop zones between cells. Stacks show insertion indicators |

**Key insight:** Professional editors separate **drop target resolution** (which container + index) from **visual feedback** (highlight, guides). Both use the same resolution logic.

---

### 3. Layer Management (Z-Index / Stacking Order)

| Feature | Why Expected | Complexity | Dependencies |
|---------|--------------|------------|--------------|
| **Layer panel** | Visual representation of DOM order = stacking order | Low | Already have outline tree |
| **Bring to front / Send to back** | Standard layer operations | Low | `ReorderNodeCommand` exists |
| **Move up / Move down** | Fine-grained layer adjustment | Low | `ReorderNodeCommand` exists |
| **Layer visibility toggle** | Hide elements without deleting | Low | Existing locked/hidden in schema |
| **Layer lock toggle** | Prevent accidental edits | Low | Existing locked in schema |

**Existing convention (from `docs/designer/layer-order-convention.md`):**
- `parent.children` array order = DOM order = stacking order
- Index 0 = bottom (rendered first), last index = top (rendered last)
- Operations: 置顶 = move to end, 置底 = move to start, 上移一层 = swap with next, 下移一层 = swap with previous

**Figma behavior:**
- Layers panel is the source of truth for z-order
- Drag layers in panel to reorder
- No separate z-index property - order is always by array position

**Sketch behavior:**
- Layer list determines z-order
- "Bring Forward" / "Send Backward" adjusts list position
- "Bring to Front" / "Send to Back" moves to list end/start

**Adobe XD behavior:**
- Layer panel shows order
- No z-index property - relies entirely on layer order
- Stacks provide automatic reordering based on layout direction

---

### 4. Spatial Arrangement / Coordinate System

| Feature | Why Expected | Complexity | Dependencies |
|---------|--------------|------------|--------------|
| **Absolute positioning in FreeContainer** | Place elements anywhere with x,y coordinates | Low | `absolutePos` schema exists |
| **Coordinate relative to parent** | Coordinates should be relative to container, not canvas | Medium | Coordinate system normalization |
| **Zoom-aware coordinates** | Design coordinates should account for zoom level | Medium | `placementUtils.js` mentioned |
| **Snap to grid** | Optional alignment to grid | Low | Grid system exists |
| **Smart guides / alignment guides** | Visual alignment to other elements | Medium | Alignment calculation |

**Existing convention (from `docs/designer/placement-and-stacking.md`):**
- All canvas coordinates are relative to page root DOM element
- Coordinates are **pre-divided by zoom** (design coordinates)
- `absolutePos.x/y/w/h` for absolute positioning

**Figma behavior:**
- Coordinates relative to parent frame
- Zoom affects viewport, not stored coordinates
- Smart guides show distance to nearby elements

**Sketch behavior:**
- Coordinates in Points (1pt = 1px at 100%)
- Pixel fitting option to snap to whole pixels
- Grid snapping independent of zoom

---

### 5. Multi-Select and Group Operations

| Feature | Why Expected | Complexity | Dependencies |
|---------|--------------|------------|--------------|
| **Box selection (marquee)** | Select multiple elements by dragging | Medium | Selection logic |
| **Shift+click multi-select** | Add/remove from selection | Low | Selection state |
| **Group selection** | Group elements to move together | Medium | Group node type |
| **Ungroup** | Dissolve group | Low | Group node type |
| **Align selected** | Align multiple elements | Medium | Alignment calculation |
| **Distribute selected** | Even spacing between elements | Medium | Distribution calculation |

**From `docs/designer/refactor/design-interaction.md`:**
- Box selection exists (M key)
- Align/distribute tools mentioned in context toolbar
- Group via Ctrl+G, Ungroup via Ctrl+Shift+G (right-click menu)

---

## Differentiators

Features that set InduForge apart from generic canvas editors. Not expected, but valued for industrial IoT context.

### 1. Industrial Data Binding Integration

| Feature | Value Proposition | Complexity | Dependencies |
|---------|------------------|------------|--------------|
| **Drag data point onto component** | Directly bind data points to component properties | High | DataService, binding system |
| **Data point status visualization** | Show active/invalid/unknown status in designer | Medium | DatapointService |
| **Real-time value preview** | Show live values in preview mode | High | WebSocket subscription |
| **Expression binding** | `{{mqtt.X.y}}` syntax in properties | Medium | Expression engine exists |

**Why differentiator:** Generic editors (Figma, Sketch) have no concept of real-time data binding. This is core to InduForge's industrial value.

---

### 2. Canvas + DOM Dual-Layer Architecture

| Feature | Value Proposition | Complexity | Dependencies |
|---------|------------------|------------|--------------|
| **Canvas graphics over DOM components** | Industrial pipes/backgrounds with interactive overlays | High | Canvas layer + DOM layer separation |
| **Symbol library for industrial components** | Pre-built industrial symbols (valves, pumps, sensors) | Medium | Symbol system |
| **Pipeline with flow animation** | Visual indication of data flow direction | Medium | Canvas rendering |

**Why differentiator:** Generic editors are single-layer (Figma) or DOM-only (Adobe XD). The dual-layer approach is specific to industrial visualization needs.

---

### 3. Multi-View Responsive Design

| Feature | Value Proposition | Complexity | Dependencies |
|---------|------------------|------------|--------------|
| **View schema per breakpoint** | Design once, deploy to multiple screen sizes | High | View switching |
| **Clone PC view to large screen** | Reuse designs across viewports | Medium | View duplication |
| **View-specific component visibility** | Show/hide components per view | Medium | View schema |

**From `docs/designer/refactor/design-interaction.md`:**
- View types: Large (1200+), PC (default), Tablet (992-), Phone (768-, 480-)
- Switch view = switch editing view schema

---

## Anti-Features

Features to explicitly NOT build, based on professional editor patterns and InduForge scope.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Per-element z-index property** | Breaks the `children` array = z-order convention. Leads to visual vs data inconsistency | Use layer operations (bring to front/back) that modify array order |
| **Multiple drop handlers with different logic** | The current bug source. `DesignCanvas` and `NodeRenderer` having separate drop logic causes inconsistency | Single `placementResolver` that both call |
| **Drag过程中修改文档 (modify document during drag)** | Causes state inconsistency. `dragenter`/`dragover` should only update visual feedback, not schema | Only modify document on `drop` |
| **Inline editing in layer panel** | Adds complexity without enough value. Layer panel is for overview, not detailed editing | Double-click on canvas to edit |
| **Manual z-index input field** | Leads to conflicts with array order. Users entering z-index values can conflict with array-based ordering | Layer operations only |

---

## Feature Dependencies

```
placementResolver (NEW)
    ├── Requires: editor-store (for insertNode)
    ├── Requires: DragDropManager (existing)
    ├── Requires: placementUtils.js (existing utilities)
    │
    ├── Enables: Unified drop target resolution
    │       └── Eliminates: DesignCanvas vs NodeRenderer drop divergence
    │
    └── Enables: Consistent visual drop indicators
            └── Requires: Visual feedback system (drop zone highlighting)

insertNode (existing, needs fixes)
    ├── Requires: Parent type detection (FreeContainer vs flow containers)
    ├── Requires: Coordinate system normalization (zoom-aware)
    │
    └── Fixes: Root node type ambiguity

ReorderNodeCommand (existing)
    ├── Uses: parent.children array order
    ├── Enables: 置顶/置底/上移/下移
    │
    └── Fixes: Ensure no z-index override anywhere

Layer Panel (existing outline tree enhancement)
    ├── Uses: children array for order
    ├── Enables: Drag to reorder
    │
    └── Optional: Could use same drag logic as placementResolver
```

---

## MVP Recommendation

Based on the existing issues and professional editor patterns, prioritize in this order:

### Must Have (Table Stakes - Fix Current Bugs)

1. **Unified `placementResolver`** - Single entry point for drop target resolution
   - Consolidates `DesignCanvas.handleCanvasDrop` + `NodeRenderer.handleDrop`
   - Returns: `{ parentId, index, dropPosition }`
   - Uses `document.elementFromPoint()` + `closest('[data-node-id]')` to find target

2. **Fix coordinate system consistency**
   - Ensure zoom is always divided out of stored coordinates
   - Verify scroll offset handling is consistent
   - Use `placementUtils.js` everywhere instead of ad-hoc rect calculations

3. **Fix insertIndex snapshot binding**
   - Verify snapshots are cleared when hover moves to different container
   - Fall back to "append" if snapshot doesn't match current target

### Should Have (Complete the Experience)

4. **Visual drop indicators**
   - Blue border highlight when drag enters container
   - Show insert index line between existing children
   - "Drop here" overlay for empty containers

5. **Layer panel enhancement**
   - Drag to reorder layers
   - Same resolution logic as component drop

### Nice to Have (Polish)

6. **Smart guides for alignment**
7. **Snap to grid (optional)**
8. **Group/ungroup in layer panel**

---

## Gap Analysis: What I Could Not Fully Verify

| Gap | Reason | Confidence | Recommendation |
|-----|--------|------------|----------------|
| Figma's exact drop resolution algorithm | Figma docs returning 404s during research | LOW | Validate with Figma user testing or community docs |
| Sketch's nested drop behavior | Sketch docs returned navigation structure only | LOW | Assume similar to Figma: hover target determines parent |
| Adobe XD's constraint system details | Docs were too high-level | LOW | Focus on Figma patterns as primary reference |
| InduForge's `placementUtils.js` implementation | Not examined in this research pass | MEDIUM | Need to verify existing utility coverage |

---

## Sources

### Verified Sources (HIGH/MEDIUM Confidence)

- **Sketch Canvas Documentation** - `https://www.sketch.com/docs/canvas/` - Zooming, rulers, grids, snapping, smart guides
- **Sketch Layer Documentation** - `https://www.sketch.com/docs/layers/` - Layer behavior and snapping
- **Adobe XD Layers** - `https://helpx.adobe.com/xd/help/layers.html` - Layer panel, grouping, z-order via panel order
- **Adobe XD Components** - `https://helpx.adobe.com/xd/help/work-with-components-xd.html` - Component placement, responsive constraints

### Project Internal Sources (HIGH Confidence - Already Read)

- `docs/designer/refactor/design-interaction.md` - Canvas architecture, drag-drop, panels
- `docs/designer/layer-order-convention.md` - Stacking order via `children` array
- `docs/designer/placement-and-stacking.md` - Current issues and proposed unified resolver
- `.planning/PROJECT.md` - Current milestone context

### Unverified Sources (LOW Confidence - Need Validation)

- Figma auto-layout behavior details
- Figma frame drop zone resolution algorithm
- Sketch nested container drop behavior
- Adobe XD stacks and repeat grid specifics

**Recommendation:** Before implementation, do a focused research spike on Figma's auto-layout documentation (currently blocked by 404s from Figma's side) to validate the single-resolver pattern alignment.
