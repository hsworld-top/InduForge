# Domain Pitfalls: Designer Canvas Editor Optimization

**Domain:** Vue-based low-code canvas editor component placement and layout interaction
**Researched:** 2026-04-15
**Overall confidence:** MEDIUM-HIGH (based on existing project docs + established patterns)

---

## Critical Pitfalls

Mistakes that cause rewrites, major issues, or blocked user workflows.

### Pitfall 1: Multiple Drop Target Resolution Entry Points

**What goes wrong:** Component placement produces inconsistent results depending on where user drops - canvas blank area vs. on top of a node.

**Why it happens:** Two separate code paths handle drops:
- `DesignCanvas.handleCanvasDrop`: Uses `event.currentTarget.getBoundingClientRect()` for coordinates, always inserts at `rootNodeId` with append index
- `NodeRenderer.handleDrop`: Uses `resolveDropOffset(event, targetElement)` for coordinates when dropped on a node

When root is a FreeContainer filling the canvas, drops mostly hit the root node (NodeRenderer path), but edge cases go through DesignCanvas with different coordinate understanding.

**Consequences:**
- Same visual drop point produces different logical positions
- Components end up at wrong coordinates or wrong parent
- Zoom and scroll handling may differ between paths

**Prevention:**
- Create single `placementResolver.js` that resolves `{ parentId, index, dropPosition }` from event
- Both DesignCanvas and NodeRenderer should call this resolver before `insertNode`
- Consolidate coordinate calculation in `placement-utils.ts` (`eventToCanvasPosition`)

**Detection:**
- Drop same component at canvas edge vs. center produces different results
- Coordinates of inserted component don't match mouse position

---

### Pitfall 2: Stacking Order Desynchronization from Data

**What goes wrong:** Visual z-order doesn't match `parent.children` array order, causing incorrect layer visibility.

**Why it happens:**
- Convention is `parent.children` order = DOM order = z-order (index 0 = bottom, last = top)
- `ReorderNodeCommand` correctly manipulates children array
- But `MoveNodeCommand` or direct mutations may not maintain this invariant
- If any code path uses CSS z-index to override DOM order, data and visuals diverge

**Consequences:**
- "Bring to front" appears to do nothing
- Reordering in layer panel doesn't match canvas
- Drag operations may insert at wrong visual layer

**Prevention:**
- Strictly use `parent.children` order for stacking - no z-index overrides
- Only `ReorderNodeCommand` should modify stacking; all move/insert operations go through it
- Add validation: warn if node has z-index style but is inside sorted container

**Detection:**
- Layer panel order doesn't match visual stacking
- `ReorderNodeCommand` produces expected array but visual order unchanged

---

### Pitfall 3: insertIndex Staleness from Multiple Snapshot Sources

**What goes wrong:** Component inserted at wrong index in flow containers (e.g., inserted into wrong row/column).

**Why it happens:**
- `NodeRenderer.handleDrop` determines insertIndex from multiple sources: `rowInsertSnapshot`, `layoutInsertSnapshot`, `layoutResolvedByPoint`, `DragDropManager.calculateFlexInsertPosition`
- These snapshots are captured during dragover/hover
- If drop happens quickly or target changes, snapshot may not match current `targetNode.id`
- Branches with stale snapshot vs. current target cause "insert into old row" bugs

**Consequences:**
- Component inserted into wrong container or position
- Layout structure corrupted (nodes in unexpected parents)
- Hard to debug - drop visual feedback looked correct

**Prevention:**
- All insertIndex sources must be validated against current `targetNode.id` before use
- If snapshot target differs from current target, recalculate index from current event
- Fall back to "append at end" rather than use stale index

**Detection:**
- Drag component over Layout A, then quickly drop on Layout B - component ends up in A
- Insert index doesn't match insertion line indicator

---

### Pitfall 4: Canvas Coordinate System Inconsistencies with Zoom/Scroll

**What goes wrong:** Dropped component position doesn't match mouse cursor when canvas is zoomed or scrolled.

**Why it happens:**
- `resolveDropOffset` uses `element.getBoundingClientRect()` + `event.clientX/clientY` = viewport-relative coords (correct)
- `DesignCanvas.handleCanvasDrop` uses `event.currentTarget.getBoundingClientRect()` which may be the canvas root, not scroll content
- When canvas has scroll containers, "canvas root" vs "scroll content" are different elements
- Zoom division may be applied inconsistently

**Consequences:**
- At 50% zoom, component appears 2x offset from cursor
- Scrolled canvas causes consistent position offset
- "Drop at cursor" puts component somewhere else

**Prevention:**
- `eventToCanvasPosition(event, containerElement, zoom)` already exists in `placement-utils.ts` - use it everywhere
- Ensure containerElement is the scroll content element, not just the visual root
- Apply zoom division exactly once, at the final coordinate calculation

**Detection:**
- Zoom != 100% causes position offset proportional to zoom level
- Scrolling canvas changes where component lands relative to cursor

---

### Pitfall 5: Root Node Type Misidentification for Positioning Mode

**What goes wrong:** Child nodes get wrong positioning mode (absolute vs. flow) depending on how root is identified.

**Why it happens:**
- `insertNode` checks both `isRootCanvas = (parentNode.id === currentPage.value?.rootNodeId)` and `parentNode.type === "FreeContainer"`
- If these conditions mismatch (rootNodeId points to non-FreeContainer or vice versa), children get wrong positioning
- Mixed判断: root detected as flow container but children get absolute positioning, or vice versa

**Consequences:**
- Children of root appear at `absolutePos: {x,y}` when they should flow
- Or: children should be absolutely positioned but get flow layout instead
- Layout type mismatch causing structural bugs

**Prevention:**
- Define clear invariant: `rootNodeId` MUST be a FreeContainer or equivalent root
- Single source of truth for "is this the canvas root" check
- Document and enforce parent type -> child positioning mapping:

| Parent Type | Child Positioning | Child Position Data |
|-------------|-------------------|---------------------|
| FreeContainer / rootNodeId | absolute | absolutePos {x,y,w,h} |
| HorizontalLayout / VerticalLayout / Flex / ElLayout | flow | no absolutePos |
| FreeContainer (non-root) | absolute | absolutePos or layoutItem.free.abs |

**Detection:**
- Components dropped on canvas root appear with/without absolutePos unexpectedly
- Layout inspector shows mixed positioning modes in same parent

---

## Moderate Pitfalls

Issues that cause significant bug hunt time or user confusion.

### Pitfall 6: Drop Event Propagation Causing Double Insertion

**What goes wrong:** Component gets inserted twice when dropped, creating duplicate nodes.

**Why it happens:**
- `NodeRenderer.handleDrop` calls `event.stopPropagation()` to prevent bubbling
- If event doesn't reach the expected handler or stopPropagation fails (e.g., capture phase issue), DesignCanvas also fires
- Or: drop on "empty" area hits both node detection AND canvas detection

**Consequences:**
- Duplicate nodes in document
- User sees "ghost" component that can't be selected properly
- Undo creates confusing state

**Prevention:**
- Guard in `DesignCanvas.handleCanvasDrop`: only execute if `event.target === event.currentTarget` or explicitly marked "blank area"
- Add assertion/counter for double insertion detection during development

**Detection:**
- Inserted component appears twice with different IDs
- Undo removes one but not the other

---

### Pitfall 7: Constraints + fitMode Letterbox Ambiguity

**What goes wrong:** When using `fitMode: contain` (letterbox), constraints like "top-right corner" are ambiguous.

**Why it happens:**
- `contain` mode adds black bars to maintain aspect ratio
- "Top-right corner" could mean design canvas top-right or screen top-right
- Constraints calculated relative to design space don't match visible viewport

**Consequences:**
- Elements positioned "in corner" appear cut off or misaligned
- Preview at different resolutions shows broken layouts

**Prevention:**
- Document the convention: constraints are relative to design canvas dimensions, not viewport
- Consider adding `relativeTo: 'viewport'` option for viewport-relative positioning
- Add validation warnings when constraints + fitMode combination likely causes issues

**Detection:**
- Preview at 16:9 vs 4:3 shows elements in wrong corners
- Constraints panel shows different positions than canvas

---

### Pitfall 8: MoveNodeCommand Not Maintaining Stacking Invariants

**What goes wrong:** Moving a node between parents corrupts layer order or parent's children array.

**Why it happens:**
- `MoveNodeCommand` calls `doc._moveNode(nodeId, newParentId, newIndex)`
- If newParent already has children, splice may disrupt existing order
- Moving from one parent to another doesn't preserve relative stacking with siblings

**Consequences:**
- Moving Node A before Node B doesn't match visual expectation
- Siblings end up with unexpected indices
- Reorder operations produce counterintuitive results

**Prevention:**
- After move, validate that `newParent.children` reflects intended order
- Consider normalizing indices (0, 1, 2, ...) after moves to prevent gaps

**Detection:**
- Move A above B visually, but B still appears above A
- Layer panel shows non-sequential indices after multiple moves

---

### Pitfall 9: DuplicateNodeCommand Offset Collisions

**What goes wrong:** Rapid duplication places copies at same position, making them overlap and hard to select.

**Why it happens:**
- Default offset is `{ x: 20, y: 20 }`
- If two duplications happen synchronously (double-click vs. Ctrl+D held), both calculate offset from original
- Original position + 20 = same for both

**Consequences:**
- Duplicate nodes completely overlap
- User must drag to separate them
- Confusing UX

**Prevention:**
- Add small random jitter to offset (plus/minus 5px)
- Or: after duplicate, immediately select new node and show toast "Ctrl+D to duplicate again"

**Detection:**
- Two identical nodes at exactly same position
- Ctrl+D produces overlapping duplicates when held

---

## Minor Pitfalls

Edge cases that cause confusion or require workarounds.

### Pitfall 10: ReorderNodeCommand Boundary Index Math

**What goes wrong:** "Up" moves down, "down" moves up due to index interpretation.

**Why it happens:**
- Convention: index 0 = bottom (back), index N = top (front)
- "Up" should increase index (move toward front)
- But ReorderNodeCommand maps:
  - "up" -> `Math.min(length - 1, oldIndex + 1)` - correct
  - "down" -> `Math.max(0, oldIndex - 1)` - correct
- But undo swaps these and may not match user expectation

**Consequences:**
- "Move up" keyboard shortcut moves visually down in some contexts
- Undo of reorder appears to do opposite operation

**Prevention:**
- Document clearly: larger index = visually on top
- Add visual feedback showing where element will land before release

**Detection:**
- Keyboard shortcut for "layer up" moves element down
- Undo of reorder undoes the opposite

---

### Pitfall 11: Nested Container InsertIndex Calculation

**What goes wrong:** Dropping into deeply nested layout (e.g., ElLayout inside ElLayout) calculates wrong index.

**Why it happens:**
- Multiple snapshot types (rowInsertSnapshot, layoutInsertSnapshot) may conflict in nested scenarios
- Inner container receives insert meant for outer, or vice versa
- Parent chain resolution fails in complex hierarchies

**Consequences:**
- Component ends up in deeply nested sub-container that user didn't intend
- Layout structure hard to fix without removing and re-adding

**Prevention:**
- Log parent chain during drop for debugging
- Validate inserted node's actual parent matches expected parent

**Detection:**
- Component found in deeply nested sub-container that user didn't expect
- Tree view shows unexpected nesting

---

### Pitfall 12: DragLeave Fires Prematurely in Complex Layouts

**What goes wrong:** Drop target highlight disappears even though user is still over valid drop area.

**Why it happens:**
- CSS `pointer-events` on child elements may fire dragleave when entering child
- Complex nested containers cause dragenter/dragleave to fire at wrong times
- Particularly problematic with FlexContainer where children fill container

**Consequences:**
- Visual feedback shows "invalid drop" when position is valid
- User releases over wrong target or cancels

**Prevention:**
- Use `relatedTarget` check instead of relying solely on dragleave
- Implement custom dragover tracking with elementFromPoint

**Detection:**
- Drop target highlight flickers or disappears during drag
- Valid drop positions not highlighted

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Single placement resolver | Multiple entry points being consolidated may break existing valid paths | Map all current flows before refactoring; add integration tests |
| Unified coordinate system | Existing calculations assume local coords; fixing breaks those | Audit all `clientX/Y`, `getBoundingClientRect` usages first |
| Stacking invariant enforcement | Old operations may violate new strict rules | Add schema validation; warn on invalid z-index usage |
| Layout container compatibility | Mixed layout types may reject certain components | Define explicit component-accepts-container matrix |
| Undo/redo with new commands | Stack changes may break expected undo behavior | Test undo across all operations after each phase |

---

## Sources

- **HIGH confidence:** Project internal docs (placement-and-stacking.md, layer-order-convention.md, size-convention.md, layout-system.md)
- **HIGH confidence:** Source code analysis (nodeCommands.ts, placement-utils.ts, editor-store.ts preview)
- **MEDIUM confidence:** Rendering architecture (rendering.md) - established patterns, may need verification against current implementation
- **LOW confidence:** General low-code canvas pitfalls - WebSearch API unavailable, unable to verify external sources

---

## Open Questions

1. How does `relatedTarget` behave in Konva/DOM hybrid layer model during drag operations?
2. Are there existing integration tests for placement scenarios that can guide refactoring?
3. Does `fitMode: cover` have similar letterbox ambiguity issues as `contain`?
4. Are there documented cases of the double-insertion bug in issue tracker?
5. Does the current DragDropManager have any retry/dedupe logic for rapid drops?
