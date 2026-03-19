# 设计器：元素放置与堆叠联动改进方案

本文档针对新版设计器中「元素堆叠、位置计算、元素在页面上放置、元素在布局容器放置」等联动细节导致的各类 bug，给出**问题归因、统一约定和可执行改进步骤**，便于系统修复与回归验证。

---

## 1. 现状与问题归因

### 1.1 涉及模块

| 模块 | 职责 | 关键文件 |
|------|------|----------|
| **放置目标解析** | 判断「拖放时鼠标下是谁」：页面根 vs 某布局/容器 | `DesignCanvas.vue`、`NodeRenderer.vue`、`DragDropManager.js` |
| **位置计算** | 绝对定位的 x/y、流式布局的 index | `DesignCanvas.vue`、`NodeRenderer.vue`（`resolveDropOffset` / `clampDropPosition`）、`editor-store.js`（`insertNode`） |
| **父子与顺序** | parentId、children 顺序、插入 index | `DocumentModel.js`（`_insertNode`）、`nodeCommands.js`（Insert/Move/Reorder） |
| **堆叠顺序** | 同父下谁在上谁在下 | `nodeCommands.js`（`ReorderNodeCommand`）、`layer-order-convention.md` |

### 1.2 典型问题来源

1. **两套「放到页面」入口**
   - **DesignCanvas.handleCanvasDrop**：drop 落在画布空白时，用 `event.currentTarget.getBoundingClientRect()` 算坐标，**始终**插入到 `rootNodeId`，index 为 append。
   - **NodeRenderer.handleDrop**：drop 落在某个节点上（含根节点 FreeContainer）时，用 `resolveDropOffset(event, targetElement)` 算坐标。
   - 若根节点是 FreeContainer 且占满画布，多数时候 drop 会落在根节点上，走 NodeRenderer；只有极边缘或无内容时可能走 DesignCanvas。两处对「画布坐标系」的理解（例如是否考虑滚动、zoom）若不一致，会出现**同一视觉落点、不同逻辑位置**的 bug。

2. **坐标未统一考虑滚动**
   - `resolveDropOffset` 使用 `element.getBoundingClientRect()` 与 `event.clientX/clientY`，得到的是**相对该 element 的坐标**，与是否滚动无关（viewport 坐标系一致）。
   - DesignCanvas 里用的是 `event.currentTarget`（画布根 div）的 rect；若画布在可滚动区域内，且「画布根」与「可滚动内容」不是同一元素，则需明确：**落点应相对谁**（画布内容区 vs 视口内画布可见区）。当前若存在多层滚动容器，可能出现偏差。

3. **容器类型与 insertIndex 多路分支**
   - NodeRenderer.handleDrop 中 insertIndex 由多处逻辑决定：`rowInsertSnapshot`、`layoutInsertSnapshot`、`layoutResolvedByPoint`、`dragDropManager.calculateFlexInsertPosition` 等。
   - 分支多、状态（snapshot）与「当前 drop 目标」可能不同步时，容易出现**插错父或插错 index**（例如插到别的行/列）。

4. **堆叠顺序与数据一致**
   - 堆叠顺序已约定为 `parent.children` 顺序（见 `layer-order-convention.md`），且 `ReorderNodeCommand` 已按此实现。
   - 潜在问题：**移动节点**（MoveNodeCommand）或**插入节点**时若未正确维护 `children` 顺序，或某处用「z-index」覆盖了 DOM 顺序，会导致表现与数据不一致。

5. **根节点类型与「是否绝对定位」**
   - 根节点为 **FreeContainer** 时，其直接子节点应为 `positioning: 'absolute'` + `absolutePos`。
   - `insertNode` 中 `isRootCanvas = (parentNode.id === currentPage.value?.rootNodeId)` 与 `parentNode.type === "FreeContainer"` 都会给子节点设 absolute；需保证**根节点类型与 isRootCanvas 判断一致**，避免根被误判为 flow 容器。

---

## 2. 统一约定（建议固化为规范）

### 2.1 放置目标解析（单一真相源）

- **原则**：拖放结束时，**只在一个地方**根据「鼠标位置」解析出：**最终父节点 parentId**、**插入索引 index**（流式容器）、**落点坐标 dropPosition**（仅绝对定位容器需要）。
- **推荐**：由 **DragDropManager**（或抽成 `placementResolver.js`）提供单一方法，例如：
  - 输入：`(event, canvasRootElement, doc, currentPage)`  
  - 输出：`{ parentId, index, dropPosition: { x, y } | null }`  
  - 内部用 `document.elementFromPoint(clientX, clientY)` + `closest('[data-node-id]')` 找到最内层节点，再判断是否为容器、是否接受该组件类型；若为流式布局则算 index，若为 FreeContainer/根则算 dropPosition（并做 clamp）。

这样 DesignCanvas 与 NodeRenderer 的 drop 都**只负责**：取 payload、调该解析器、再调 `editorStore.insertNode(type, parentId, index, { dropPosition })`，避免两套坐标与目标逻辑。

### 2.2 画布坐标系

- **约定**：所有「画布内」的绝对坐标（如 `absolutePos.x/y`）均为**相对当前页根节点对应 DOM 元素**的坐标，且**已除以 zoom**（设计态缩放）。
- **滚动**：若画布存在滚动（例如 `.canvas-scroll-content`），则落点应相对**该滚动内容区**的左上角计算（即 getBoundingClientRect 的 element 应为该内容区），这样滚动后落点与视觉一致。若当前实现中 drop 的 currentTarget 已是滚动内容区内的根节点，则通常无需再加 scrollOffset；否则需要在 DesignCanvas 或统一解析处显式加上 scrollLeft/scrollTop。

### 2.3 堆叠顺序（与现有约定一致）

- 继续遵循 `layer-order-convention.md`：**同一父节点下，`children` 数组顺序 = DOM 渲染顺序 = 叠放顺序**（index 越大越在上层）。
- 插入、移动、排序时，**只通过修改 `parent.children` 的顺序**来改变堆叠，不在样式上使用 z-index 覆盖（除非有明确需求如浮动工具栏）。

### 2.4 父类型与子节点定位方式

| 父节点类型 | 子节点 positioning | 子节点位置信息 |
|------------|--------------------|----------------|
| FreeContainer / 页面根（rootNodeId） | `absolute` | `absolutePos`（x, y, w, h） |
| HorizontalLayout / VerticalLayout / FlexContainer / ElLayout / ElCol / … | `flow` | 无 absolutePos；顺序由 children index 决定 |
| FreeContainer（非根） | `absolute` | `absolutePos` 或 layoutItem.free.abs |

确保 **insertNode** 中「是否根画布」「是否 FreeContainer」分支与上述一致，且不重复、不遗漏。

---

## 3. 可执行改进步骤

### 3.1 短期（止血、减少 bug）

1. **统一「放到页面根」的坐标计算**
   - 在 DesignCanvas.handleCanvasDrop 中，若希望与 NodeRenderer 行为一致，可改为：用 `elementFromPoint` 找到当前落点下的节点；若为根节点（FreeContainer），则**复用与 NodeRenderer 相同的坐标计算**（即调用同一套 `resolveDropOffset` + `clampDropPosition`，或抽到共用的 `placementUtils.js`），再 `insertNode(type, rootNodeId, undefined, { dropPosition })`。
   - 或：**收敛为单一入口**——画布空白处的 drop 也先走「解析器」得到 parentId=rootNodeId、index、dropPosition，再统一走 `insertNode`，避免两处各自算坐标。

2. **明确 drop 事件不重复执行**
   - 已通过 NodeRenderer 的 `event.stopPropagation()` 避免冒泡到 DesignCanvas；建议在 DesignCanvas.handleCanvasDrop 开头加一层防护：若 `event.target` 不是画布根或明确「空白区域」（例如通过 `event.target === event.currentTarget` 或 data 属性标记），则不再执行插入，避免极端情况下双重插入。

3. **insertIndex 与目标容器强绑定**
   - 在 NodeRenderer.handleDrop 中，所有用于 `insertIndex` 的 snapshot（rowInsertSnapshot、layoutInsertSnapshot 等）必须与当前 `targetNode.id` 一致再使用；若不一致则回退到「末尾」或由 DragDropManager 按当前 event 再算一次 index，避免用了「上一次悬停」的 index 导致插错位置。

### 3.2 中期（结构清晰、易维护）

4. **抽出「放置解析」单一入口**
   - 新建 `editor-core/placement/placementResolver.js`（或放在 `ui/Canvas/` 下），实现：
     - `resolveDropTarget(event, options)` → `{ parentId, index, dropPosition }`
     - 内部使用现有的 DragDropManager、resolveDropOffset、clampDropPosition 等逻辑。
   - DesignCanvas 与 NodeRenderer 的 drop 只负责：解析 payload（组件类型等）→ 调用 `resolveDropTarget` → 调用 `insertNode(type, parentId, index, { dropPosition })`。这样所有「放哪里、什么顺序、什么坐标」都在一处维护。

5. **画布坐标工具函数**
   - 已提供 `editor-core/utils/placementUtils.js`：
     - `eventToCanvasPosition(event, containerElement, zoom)`：返回相对 containerElement 且已除 zoom 的 `{ x, y }`，如需可在此处加 scroll 修正。
     - `clampPositionInContainer(position, containerElement, size, zoom)`：将落点限制在容器内。
   - 建议 DesignCanvas 与 NodeRenderer 中所有「落点坐标」计算逐步迁移到上述工具，避免多处手写 rect/clientX/clientY/zoom。

6. **回归用例（手动或自动化）**
   - 列出核心场景，便于每次改完验证：
     - 拖到页面空白：落在根节点，absolutePos 与鼠标位置一致（含 zoom）。
     - 拖到 HorizontalLayout：插入到正确 index，无 absolutePos。
     - 拖到 ElLayout：自动创建行/列并插入到正确格子。
     - 从布局内拖出到页面根：变为 absolute，位置为拖放落点。
     - 置顶/置底/上移一层/下移一层：仅改变 children 顺序，表现与数据一致。

### 3.3 长期（可选）

7. **拖拽过程与放置结果解耦**
   - 拖拽过程中只做「高亮 / 插入线」等视觉反馈，**不修改文档**；仅在 **drop** 时调用一次 `resolveDropTarget` + `insertNode`，避免 dragenter/dragover 多次触发导致的状态不一致。

8. **文档层校验**
   - 在 DocumentModel 或 command 执行后做轻量校验：同一父下 children 无重复、无孤儿节点；根子节点若为 absolute 则必有 absolutePos 等，便于在开发阶段发现联动错误。

---

## 4. 与现有文档的对应关系

- **层级顺序**：完全遵循 `docs/designer/layer-order-convention.md`，本方案只强调「插入/移动时务必维护 children 顺序」。
- **尺寸**：遵循 `docs/designer/size-convention.md`；放置时默认尺寸由 manifest / `resolveDefaultSize` 提供，绝对定位时用 `absolutePos.w/h` 与样式宽高联动。
- **老版对比**：老版编辑器的布局与属性机制见 `docs/designer/legacy-page-editor-analysis.md`；新版采用「FreeContainer 根 + absolute/flow 分支」，本方案在该架构下统一放置与堆叠行为。

---

## 5. 小结

- **问题本质**：放置目标、插入索引、落点坐标存在多入口和多套计算，加上容器类型与 snapshot 分支多，容易产生「插错父/错 index/错坐标」和堆叠与数据不一致。
- **改进方向**：**单一放置解析入口** + **统一画布坐标计算** + **严格按 parent.children 顺序表示堆叠**，并做少量防护与回归场景验证。
- **实施顺序**：先做 3.1 的止血（统一坐标、防重复执行、insertIndex 与 target 绑定），再考虑 3.2 的抽取与工具函数，最后按需做 3.3 的解耦与校验。

按上述步骤落地后，元素堆叠、位置计算、页面放置与布局内放置的联动会更可预期，bug 也更容易定位和收敛。
