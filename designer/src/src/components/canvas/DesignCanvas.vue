<template>
  <div
    ref="viewportRef"
    class="design-canvas-viewport"
    :class="{ 'show-grid': showGrid }"
    @click="handleViewportClick"
    @dragenter="handleDragEnter"
    @dragover="handleDragOver"
    @dragleave="handleDragLeave"
    @dragend="handleDragEnd"
    @drop="handleDrop"
    @contextmenu="handleContextMenu"
  >
    <!-- Canvas Layer (Konva) - 辅助功能层 -->
    <!-- z-index: 100, pointer-events: none -->
    <div
      class="canvas-layer"
      :style="{
        width: `${canvasWidth}px`,
        height: `${canvasHeight}px`,
      }"
    >
      <CanvasAuxiliary
        ref="canvasAuxiliaryRef"
        :width="canvasWidth"
        :height="canvasHeight"
        :zoom="zoom"
        :scroll-x="scrollX"
        :scroll-y="scrollY"
        @canvas-click="handleCanvasClick"
        @canvas-ready="handleCanvasReady"
      />
    </div>

    <!-- DOM Layer (Vue Components) - 组件渲染层 -->
    <!-- z-index: 1 -->
    <!-- Task 1.5: DOM Layer 缩放 - 使用 CSS transform: scale() -->
    <div
      ref="domLayerRef"
      class="dom-layer"
      :style="{
        width: `${canvasWidth}px`,
        height: `${canvasHeight}px`,
        backgroundColor: backgroundColor,
        transform: `scale(${zoom})`,
        transformOrigin: 'top left',
      }"
    >
      <DomRenderer
        v-if="currentPage"
        :components="components"
        :selected-id="selectedComponentId"
        :canvas-width="canvasWidth"
        :canvas-height="canvasHeight"
        @select="handleSelect"
        @update="handleUpdate"
        @resize="handleResize"
        @resize-end="handleResizeEnd"
        @contextmenu="handleContextMenu"
        @dragstart="handleWrapperDragStart"
        @drag="handleWrapperDrag"
        @dragend="handleWrapperDragEnd"
        @dragover="handleContainerDragOver"
        @dragleave="handleContainerDragLeave"
        @drop="handleContainerDrop"
      />
      <div
        v-if="insertPlaceholder"
        class="insert-placeholder"
        :style="insertPlaceholderStyle"
      />
    </div>

    <!-- Context Menu - 右键菜单 -->
    <ContextMenu ref="contextMenuRef" :component-id="contextMenuComponentId" />
  </div>
</template>

<script setup>
/**
 * DesignCanvas - 设计画布组件（混合渲染架构）
 *
 * 架构说明：
 * - DOM Layer: 渲染所有组件和布局容器（Vue 组件 + CSS 原生布局）
 * - Canvas Layer: 渲染辅助功能（标尺、对齐线、选择框、拖拽预览）
 *
 * 层级关系：
 * - Canvas Layer (z-index: 100, pointer-events: none) - 上层
 * - DOM Layer (z-index: 1) - 下层
 *
 * Requirements:
 * - Requirement 1: 混合渲染架构
 * - Acceptance Criteria 1.1: 创建两个独立的渲染层
 * - Acceptance Criteria 1.4: 确保两层坐标系统同步且事件不冲突
 */
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from "vue";
import { useDesignStore } from "@/store/design";
import { useCanvas, snapToGrid as snapPositionToGrid } from "@/composables/useCanvas";
import { useCoordinateSync } from "@/composables/useCoordinateSync";
import { calculateInsertPosition } from "@/utils/dropZoneCalculator";
import DomRenderer from "./DomRenderer.vue";
import CanvasAuxiliary from "./CanvasAuxiliary.vue";
import ContextMenu from "./ContextMenu.vue";

// Props
const props = defineProps({
  showGrid: {
    type: Boolean,
    default: true,
  },
});

// Emits
const emit = defineEmits(["contextmenu"]);

// Store
const designStore = useDesignStore();

// Composables
const { canvasState } = useCanvas();

// Refs
const viewportRef = ref(null);
const canvasAuxiliaryRef = ref(null);
const domLayerRef = ref(null);
const contextMenuRef = ref(null);

// Drag state
const isDragging = ref(false);
const draggedComponent = ref(null);
const lastDragPoint = ref(null);
const lastDragClient = ref(null);
const lastDragTarget = ref(null);
const didDrop = ref(false);
const insertPlaceholder = ref(null);

// Context menu state
const contextMenuComponentId = ref(null);

// Canvas state
const zoom = ref(1);
const scrollX = ref(0);
const scrollY = ref(0);

// Zoom center point (for preserving center during zoom)
const zoomCenterX = ref(0);
const zoomCenterY = ref(0);

// 坐标系统同步
const coordinateSync = useCoordinateSync({
  canvasContainerRef: viewportRef,
  zoom,
  scrollX,
  scrollY,
});

// Computed
const currentPage = computed(() => designStore.currentPage);
const pageConfig = computed(() => designStore.pageConfig);
const components = computed(() => designStore.components);
const selectedComponentId = computed(() => designStore.selectedComponentId);

const canvasWidth = computed(() => pageConfig.value?.width || 1920);
const canvasHeight = computed(() => pageConfig.value?.height || 1080);
const backgroundColor = computed(
  () => pageConfig.value?.backgroundColor || "#ffffff"
);
const insertPlaceholderStyle = computed(() => {
  if (!insertPlaceholder.value) return {};
  const { x, y, width, height } = insertPlaceholder.value;
  return {
    left: `${Math.round(x)}px`,
    top: `${Math.round(y)}px`,
    width: `${Math.max(0, Math.round(width))}px`,
    height: `${Math.max(0, Math.round(height))}px`,
  };
});

const snapEnabled = computed(() => {
  if (pageConfig.value && typeof pageConfig.value.snapToGrid === "boolean") {
    return pageConfig.value.snapToGrid;
  }
  return canvasState.snapToGrid;
});

const gridSize = computed(() => {
  if (pageConfig.value && typeof pageConfig.value.gridSize === "number") {
    return pageConfig.value.gridSize;
  }
  return canvasState.gridSize || 10;
});

const CANVAS_PADDING = 40;

function findComponentById(list, componentId) {
  for (const item of list || []) {
    if (item.id === componentId) return item;
    if (item.children) {
      const found = findComponentById(item.children, componentId);
      if (found) return found;
    }
  }
  return null;
}

function findParentComponent(list, componentId, parent = null) {
  for (const item of list || []) {
    if (item.id === componentId) return parent;
    if (item.children) {
      const found = findParentComponent(item.children, componentId, item);
      if (found !== undefined) return found;
    }
  }
  return undefined;
}

function isDescendantComponent(component, targetId) {
  if (!component?.children?.length) return false;
  for (const child of component.children) {
    if (child.id === targetId) return true;
    if (isDescendantComponent(child, targetId)) return true;
  }
  return false;
}

function normalizeContainerChildStyle(style = {}) {
  return {
    ...style,
    position: "relative",
    left: null,
    top: null,
    right: null,
    bottom: null,
  };
}

function resolveContainerLayoutMode(container) {
  if (!container) return "flex";
  if (container.type === "Row" || container.type === "ElRow") return "row";
  if (container.type === "Grid") return "grid";
  if (container.type === "FlexLayout" || container.type === "CenterLayout") return "flex";
  if (container.type === "Col" || container.type === "ElCol") return "block";
  const layoutMode = container.props?.layoutMode || container.props?.layout;
  return layoutMode || "flex";
}

function getContainerInsertInfo(container, event, excludeId = null) {
  const children = Array.isArray(container?.children) ? container.children : [];
  const targetChildren = excludeId
    ? children.filter((child) => child.id !== excludeId)
    : children;
  if (!Number.isFinite(event.clientX) || !Number.isFinite(event.clientY)) {
    return { index: targetChildren.length, insertLine: null, rect: null, childRects: [] };
  }

  const targetEl =
    event?.currentTarget?.id === container?.id
      ? event.currentTarget
      : document.getElementById(container?.id || "");
  if (!targetEl) {
    return { index: targetChildren.length, insertLine: null, rect: null, childRects: [] };
  }

  const rect = targetEl.getBoundingClientRect();
  if (!rect) {
    return { index: targetChildren.length, insertLine: null, rect: null, childRects: [] };
  }

  const childRects = [];
  for (const child of targetChildren) {
    const el = document.getElementById(child.id);
    if (!el) {
      return { index: targetChildren.length, insertLine: null, rect, childRects: [] };
    }
    childRects.push(el.getBoundingClientRect());
  }

  const containerInfo = {
    rect,
    layoutMode: resolveContainerLayoutMode(container),
    props: container?.props || {},
  };
  const result = calculateInsertPosition(containerInfo, childRects, {
    x: event.clientX,
    y: event.clientY,
  });
  const index = Number.isFinite(result?.index) ? result.index : targetChildren.length;
  return {
    index: Math.max(0, Math.min(index, targetChildren.length)),
    insertLine: result?.insertLine || null,
    rect,
    childRects,
  };
}

function getContainerInsertIndex(container, event, excludeId = null) {
  return getContainerInsertInfo(container, event, excludeId).index;
}

function getDropContainerId(target) {
  if (!target || typeof target.closest !== "function") return null;
  const containerEl = target.closest(".component-wrapper.is-container");
  return containerEl ? containerEl.id : null;
}

function getDropContainerIdFromPoint(clientX, clientY) {
  if (!Number.isFinite(clientX) || !Number.isFinite(clientY)) return null;
  if (typeof document?.elementsFromPoint !== "function") return null;
  const elements = document.elementsFromPoint(clientX, clientY);
  for (const el of elements) {
    if (!el || typeof el.closest !== "function") continue;
    const containerEl = el.closest(".component-wrapper.is-container");
    if (containerEl?.id) return containerEl.id;
  }
  return null;
}

function ensureRootComponent(componentId) {
  const parent = findParentComponent(components.value, componentId);
  if (parent && parent.id) {
    designStore.moveComponent(componentId, null, components.value.length);
  }
}

function getCanvasPoint(event) {
  if (!viewportRef.value) return { x: 0, y: 0 };
  const rect = viewportRef.value.getBoundingClientRect();
  const x = (event.clientX - rect.left - CANVAS_PADDING + scrollX.value) / zoom.value;
  const y = (event.clientY - rect.top - CANVAS_PADDING + scrollY.value) / zoom.value;
  return { x, y };
}

function getCanvasPointFromClient(clientX, clientY) {
  if (!viewportRef.value) return { x: 0, y: 0 };
  const rect = viewportRef.value.getBoundingClientRect();
  const x = (clientX - rect.left - CANVAS_PADDING + scrollX.value) / zoom.value;
  const y = (clientY - rect.top - CANVAS_PADDING + scrollY.value) / zoom.value;
  return { x, y };
}

function toCanvasRectFromClient(rect) {
  if (!rect) return null;
  const start = getCanvasPointFromClient(rect.x, rect.y);
  const end = getCanvasPointFromClient(rect.x + rect.width, rect.y + rect.height);
  if (
    !Number.isFinite(start.x) ||
    !Number.isFinite(start.y) ||
    !Number.isFinite(end.x) ||
    !Number.isFinite(end.y)
  ) {
    return null;
  }
  return {
    x: start.x,
    y: start.y,
    width: end.x - start.x,
    height: end.y - start.y,
  };
}

function parseSizeValue(value) {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (trimmed.endsWith("px")) {
      const num = Number(trimmed.slice(0, -2));
      return Number.isFinite(num) ? num : null;
    }
    const num = Number(trimmed);
    return Number.isFinite(num) ? num : null;
  }
  return null;
}

function clampValue(value, min, max) {
  return Math.min(Math.max(value, min), max);
}

function resolvePlaceholderSize(info, dragData) {
  const dragStyle = dragData?.style || {};
  let width = parseSizeValue(dragStyle.width);
  let height = parseSizeValue(dragStyle.height);

  const refRect = info?.childRects?.length
    ? info.childRects[Math.min(info.index, info.childRects.length - 1)]
    : info?.rect;

  if (!Number.isFinite(width)) width = refRect?.width || 80;
  if (!Number.isFinite(height)) height = refRect?.height || 32;

  const maxWidth = info?.rect?.width || width;
  const maxHeight = info?.rect?.height || height;
  width = Math.min(Math.max(width, 16), maxWidth);
  height = Math.min(Math.max(height, 16), maxHeight);

  return { width, height };
}

function buildPlaceholderClientRect(info, dragData, event) {
  if (!info?.rect) return null;
  const { width, height } = resolvePlaceholderSize(info, dragData);
  if (!Number.isFinite(width) || !Number.isFinite(height)) return null;

  const line = info.insertLine;
  let x;
  let y;

  if (line) {
    const isVertical = Math.abs(line.height) > Math.abs(line.width);
    if (isVertical) {
      x = line.x - width / 2;
      y = line.y + (line.height - height) / 2;
    } else {
      x = line.x + (line.width - width) / 2;
      y = line.y - height / 2;
    }
  } else if (Number.isFinite(event?.clientX) && Number.isFinite(event?.clientY)) {
    x = event.clientX - width / 2;
    y = event.clientY - height / 2;
  } else {
    x = info.rect.left + 8;
    y = info.rect.top + 8;
  }

  const maxX = info.rect.right - width;
  const maxY = info.rect.bottom - height;
  x = clampValue(x, info.rect.left, maxX);
  y = clampValue(y, info.rect.top, maxY);

  return { x, y, width, height };
}

function showInsertPlaceholder(rect) {
  if (!rect) {
    insertPlaceholder.value = null;
    return;
  }
  insertPlaceholder.value = rect;
  const insertLine = canvasAuxiliaryRef.value?.getInsertLine?.();
  if (insertLine) {
    insertLine.hide();
  }
}

function hideInsertLine() {
  const insertLine = canvasAuxiliaryRef.value?.getInsertLine?.();
  if (insertLine) {
    insertLine.hide();
  }
  insertPlaceholder.value = null;
}

function setDropEffect(event) {
  const dt = event?.dataTransfer;
  if (!dt) return;
  const allowed = (dt.effectAllowed || "").toLowerCase();
  if (!allowed || allowed === "uninitialized") {
    dt.dropEffect = "move";
    return;
  }
  if (allowed.includes("move")) {
    dt.dropEffect = "move";
    return;
  }
  if (allowed.includes("copy")) {
    dt.dropEffect = "copy";
    return;
  }
  dt.dropEffect = "none";
}

function getPreviewPoint(point, component) {
  if (!point) return null;
  if (!component || component.source !== "canvas") return point;
  const offsetX = Number.isFinite(component.offsetX) ? component.offsetX / zoom.value : 0;
  const offsetY = Number.isFinite(component.offsetY) ? component.offsetY / zoom.value : 0;
  return { x: point.x - offsetX, y: point.y - offsetY };
}

function updateLastDragPoint(event) {
  if (!event || !viewportRef.value) return null;
  const rect = viewportRef.value.getBoundingClientRect();
  const clientX = event.clientX;
  const clientY = event.clientY;
  if (!Number.isFinite(clientX) || !Number.isFinite(clientY)) return null;
  if (
    clientX < rect.left ||
    clientX > rect.right ||
    clientY < rect.top ||
    clientY > rect.bottom
  ) {
    return null;
  }
  const point = getCanvasPoint(event);
  if (!Number.isFinite(point.x) || !Number.isFinite(point.y)) return null;
  lastDragPoint.value = point;
  lastDragClient.value = { x: clientX, y: clientY };
  lastDragTarget.value = event.target || null;
  return point;
}

function parseDragData(event) {
  if (!event?.dataTransfer) return null;
  const dt = event.dataTransfer;
  const candidates = [
    dt.getData("application/json"),
    dt.getData("text/plain"),
    dt.getData("application/x-designer-component"),
  ];
  for (const raw of candidates) {
    if (!raw) continue;
    try {
      return JSON.parse(raw);
    } catch (err) {
      // ignore and try next
    }
  }
  return null;
}

function getSelectedComponent() {
  if (!designStore.selectedComponentId) return null;
  return findComponentById(components.value, designStore.selectedComponentId);
}

function handleWrapperDragStart(payload) {
  const event = payload?.event;
  const component = payload?.component;
  if (!event || !component) return;

  didDrop.value = false;
  const rect = event.currentTarget?.getBoundingClientRect?.();
  const offsetX = rect ? event.clientX - rect.left : 0;
  const offsetY = rect ? event.clientY - rect.top : 0;

  draggedComponent.value = {
    ...component,
    source: "canvas",
    offsetX,
    offsetY,
  };
  isDragging.value = true;
  updateLastDragPoint(event);
}

function handleWrapperDrag(payload) {
  if (!isDragging.value) return;
  const event = payload?.event;
  if (!event) return;
  updateLastDragPoint(event);
}

function handleWrapperDragEnd(payload) {
  handleDragEnd(payload?.event || payload);
}

/**
 * 处理组件选择
 */
/**
 * 处理组件选择
 * Task 5.1: 支持单选和多选
 * @param {string} id - 组件ID
 * @param {boolean} isMultiSelect - 是否多选模式（Ctrl+点击）
 */
function handleSelect(id, isMultiSelect = false) {
  if (isMultiSelect) {
    // 多选模式：切换选中状态
    designStore.toggleComponentSelection(id);
  } else {
    // 单选模式
    designStore.selectComponent(id);
  }
}

/**
 * 处理右键菜单
 * Task 7.4: 实现右键菜单
 * @param {MouseEvent} event - 鼠标事件
 * @param {string} componentId - 组件ID（可选）
 */
function handleContextMenu(event, componentId = null) {
  event.preventDefault();

  // 如果右键点击的是组件，且该组件未被选中，则先选中它
  if (componentId && !designStore.selectedComponentIds.includes(componentId)) {
    designStore.selectComponent(componentId);
  }

  // 设置右键菜单的组件ID
  contextMenuComponentId.value = componentId;

  // 显示右键菜单
  if (contextMenuRef.value) {
    contextMenuRef.value.show(event);
  }
}

/**
 * 处理键盘事件
 * Task 5.1: 实现全选（Ctrl+A）
 * Task 7.2: 实现撤销/重做快捷键
 * @param {KeyboardEvent} event - 键盘事件
 */
function handleKeyDown(event) {
  // Ctrl+Z 或 Cmd+Z：撤销
  if (
    (event.ctrlKey || event.metaKey) &&
    event.key === "z" &&
    !event.shiftKey
  ) {
    event.preventDefault();
    designStore.undo();
    console.log("↩️ Undo");
    return;
  }

  // Ctrl+Y 或 Cmd+Shift+Z：重做
  if (
    ((event.ctrlKey || event.metaKey) && event.key === "y") ||
    ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key === "z")
  ) {
    event.preventDefault();
    designStore.redo();
    console.log("↪️ Redo");
    return;
  }

  // Ctrl+A 或 Cmd+A：全选
  if ((event.ctrlKey || event.metaKey) && event.key === "a") {
    event.preventDefault();
    designStore.selectAllComponents();
    console.log("📋 Select all components");
  }

  // Delete 或 Backspace：删除选中的组件
  if (event.key === "Delete" || event.key === "Backspace") {
    if (designStore.selectedComponentIds.length > 1) {
      // 多选：批量删除
      event.preventDefault();
      designStore.batchDeleteComponents(designStore.selectedComponentIds);
      designStore.saveHistory("批量删除组件");
      console.log("🗑️ Batch delete selected components");
    } else if (designStore.selectedComponentId) {
      // 单选：删除单个
      event.preventDefault();
      designStore.deleteSelectedComponent();
      designStore.saveHistory("删除组件");
    }
  }

  // Ctrl+C 或 Cmd+C：复制
  if ((event.ctrlKey || event.metaKey) && event.key === "c") {
    if (designStore.selectedComponentIds.length > 0) {
      event.preventDefault();
      designStore.copySelectedComponent();
      console.log("📋 Copy selected components");
    }
  }

  // Ctrl+V 或 Cmd+V：粘贴
  if ((event.ctrlKey || event.metaKey) && event.key === "v") {
    if (designStore.selectedComponentIds.length > 1) {
      event.preventDefault();
      const newIds = designStore.batchCopyComponents(
        designStore.selectedComponentIds
      );
      designStore.saveHistory("批量复制组件");
      console.log("📋 Paste selected components");
    } else if (designStore.clipboard) {
      event.preventDefault();
      designStore.pasteComponent();
      designStore.saveHistory("粘贴组件");
    }
  }

  // Ctrl+D 或 Cmd+D：复制并粘贴
  if ((event.ctrlKey || event.metaKey) && event.key === "d") {
    event.preventDefault();
    if (designStore.selectedComponentIds.length > 1) {
      designStore.batchCopyComponents(designStore.selectedComponentIds);
      designStore.saveHistory("批量复制组件");
    } else if (designStore.selectedComponentId) {
      designStore.duplicateSelectedComponent();
      designStore.saveHistory("复制组件");
    }
    console.log("📋 Duplicate selected components");
  }

  // Ctrl+Plus/Equal：放大
  if (
    (event.ctrlKey || event.metaKey) &&
    (event.key === "+" || event.key === "=")
  ) {
    event.preventDefault();
    const newZoom = Math.min(zoom.value + 0.1, 5); // 最大500%
    setZoom(newZoom);
    console.log(`🔍 Zoom in: ${Math.round(newZoom * 100)}%`);
  }

  // Ctrl+Minus：缩小
  if ((event.ctrlKey || event.metaKey) && event.key === "-") {
    event.preventDefault();
    const newZoom = Math.max(zoom.value - 0.1, 0.1); // 最小10%
    setZoom(newZoom);
    console.log(`🔍 Zoom out: ${Math.round(newZoom * 100)}%`);
  }

  // Ctrl+0：重置缩放（100%）
  if ((event.ctrlKey || event.metaKey) && event.key === "0") {
    event.preventDefault();
    setZoom(1);
    console.log("🔍 Reset zoom: 100%");
  }

  // Escape：取消选择
  if (event.key === "Escape") {
    designStore.clearSelection();
    console.log("❌ Clear selection");
  }
}

/**
 * 处理组件更新
 */
function handleUpdate(id, updates) {
  designStore.updateComponent(id, updates);
  // 保存历史记录
  designStore.saveHistory(`更新组件 ${id}`);
}

function handleResize(id, updates) {
  designStore.updateComponent(id, updates);
}

function handleResizeEnd(id) {
  if (!id) return;
  designStore.saveHistory(`resize component ${id}`);
}

/**
 * 处理 Canvas 空白区域点击
 */
function handleCanvasClick(event) {
  console.log("Canvas blank area clicked:", event);
  // 取消选择所有组件
  designStore.selectComponent(null);
}

function handleViewportClick(event) {
  if (event?.target?.closest?.(".component-wrapper")) return;
  designStore.selectComponent(null);
}

/**
 * 处理 Canvas 准备就绪
 */
function handleCanvasReady(canvasLayers) {
  console.log("✅ Canvas layers ready:", canvasLayers);
  // 可以在这里保存 Canvas 图层的引用，用于后续的辅助功能渲染
  // 例如：绘制标尺、对齐线、选择框等

  // Task 1.4: 初始化坐标系统同步
  initCoordinateSync();
}

/**
 * 处理拖拽进入画布
 * Task 4.1: 在 Canvas Layer 显示拖拽预览
 */
function handleDragEnter(event) {
  event.preventDefault();
  setDropEffect(event);
  didDrop.value = false;
  if (!isDragging.value || !draggedComponent.value) {
    const parsed = parseDragData(event);
    if (parsed) {
      if (parsed?.source === "canvas" && parsed.id) {
        draggedComponent.value = findComponentById(components.value, parsed.id) || parsed;
      } else {
        draggedComponent.value = parsed;
      }
      isDragging.value = true;
    } else {
      const selected = getSelectedComponent();
      if (selected) {
        draggedComponent.value = { ...selected, source: "canvas", offsetX: 0, offsetY: 0 };
        isDragging.value = true;
      }
    }
  }
  updateLastDragPoint(event);
  console.log("Drag enter canvas");
}

/**
 * 处理拖拽在画布上移动
 * Task 4.1: 实现拖拽预览跟随鼠标
 */
function handleDragOver(event) {
  event.preventDefault();
  setDropEffect(event);

  const hoverContainerId = getDropContainerId(event.target);
  if (!hoverContainerId) {
    hideInsertLine();
  }

  if (!isDragging.value || !draggedComponent.value) {
    const parsed = parseDragData(event);
    if (parsed) {
      if (parsed?.source === "canvas" && parsed.id) {
        draggedComponent.value = findComponentById(components.value, parsed.id) || parsed;
      } else {
        draggedComponent.value = parsed;
      }
      isDragging.value = true;
      console.info("[DesignCanvas] dragover: start dragging", parsed);
    } else {
      // dataTransfer 为空时兜底使用当前选中组件
      const selected = getSelectedComponent();
      if (selected) {
        draggedComponent.value = { ...selected, source: "canvas", offsetX: 0, offsetY: 0 };
        isDragging.value = true;
        console.info("[DesignCanvas] dragover: fallback selected", selected.id);
      }
    }
  }

  // 更新拖拽预览位置
  if (isDragging.value) {
    const point = updateLastDragPoint(event);
    if (!point) return;

    if (canvasAuxiliaryRef.value) {
      const dragPreview = canvasAuxiliaryRef.value.getDragPreview();
      if (dragPreview) {
        const previewPoint = getPreviewPoint(point, draggedComponent.value);
        if (!previewPoint) return;
        const { x, y } = previewPoint;
        // 如果预览未显示，先显示
        if (!dragPreview.isVisible && draggedComponent.value) {
          dragPreview.show(draggedComponent.value, x, y);
        } else {
          dragPreview.updatePosition(x, y);
        }
      }
    }
  }
}

/**
 * 处理拖拽离开画布
 */
function handleDragLeave(event) {
  // 只在真正离开画布时隐藏预览
  if (!event.currentTarget.contains(event.relatedTarget)) {
    hideDragPreview();
    hideInsertLine();
  }
}

function handleDragEnd(event) {
  if (didDrop.value) {
    didDrop.value = false;
    hideDragPreview();
    hideInsertLine();
    isDragging.value = false;
    draggedComponent.value = null;
    lastDragPoint.value = null;
    lastDragClient.value = null;
    lastDragTarget.value = null;
    return;
  }

  // 如果 drop 未触发，但已有拖拽信息，按最后位置补救更新
  if (isDragging.value && draggedComponent.value) {
    console.info("[DesignCanvas] dragend fallback");
    const component = draggedComponent.value;
    const client = lastDragClient.value || { x: event?.clientX, y: event?.clientY };
    const fallbackContainerId =
      getDropContainerId(lastDragTarget.value) ||
      getDropContainerIdFromPoint(client?.x, client?.y);
    if (fallbackContainerId) {
      const container = findComponentById(components.value, fallbackContainerId);
      if (container) {
        const inferredSource = component?.source === "canvas" ? "canvas" : "library";
        const containerEl = document.getElementById(fallbackContainerId);
        handleContainerDrop({
          container,
          dragData: component,
          source: inferredSource,
          event: {
            currentTarget: containerEl,
            clientX: client?.x,
            clientY: client?.y,
          },
        });
        didDrop.value = true;
        hideDragPreview();
        hideInsertLine();
        isDragging.value = false;
        draggedComponent.value = null;
        lastDragPoint.value = null;
        lastDragClient.value = null;
        lastDragTarget.value = null;
        return;
      }
    }
    const point = lastDragPoint.value;
    if (!point) {
      hideDragPreview();
      hideInsertLine();
      isDragging.value = false;
      draggedComponent.value = null;
      lastDragPoint.value = null;
      lastDragClient.value = null;
      lastDragTarget.value = null;
      return;
    }
    const { x, y } = point;

    if (component?.source === "canvas" && component.id) {
      ensureRootComponent(component.id);
      const offsetX = (component.offsetX || 0) / zoom.value;
      const offsetY = (component.offsetY || 0) / zoom.value;
      let newLeft = x - offsetX;
      let newTop = y - offsetY;

      if (!Number.isFinite(newLeft) || !Number.isFinite(newTop)) {
        hideDragPreview();
        hideInsertLine();
        isDragging.value = false;
        draggedComponent.value = null;
        lastDragPoint.value = null;
        return;
      }

      if (snapEnabled.value) {
        const snapped = snapPositionToGrid({ x: newLeft, y: newTop }, gridSize.value);
        newLeft = snapped.x;
        newTop = snapped.y;
      }

      designStore.updateComponent(component.id, {
        style: {
          position: "absolute",
          left: Math.round(newLeft),
          top: Math.round(newTop),
        },
      });
      designStore.saveHistory(`移动组件 ${component.id}`);
      designStore.selectComponent(component.id);
    }
  }

  hideDragPreview();
  hideInsertLine();
  isDragging.value = false;
  draggedComponent.value = null;
  lastDragPoint.value = null;
  lastDragClient.value = null;
  lastDragTarget.value = null;
}

/**
 * 处理放置到画布
 * Task 4.2: 实现画布接收拖放
 */
function handleDrop(event) {
  event.preventDefault();
  event.stopPropagation(); // 阻止事件冒泡，防止 DesignCenter 也处理此事件
  didDrop.value = true;

  try {
    console.info("[DesignCanvas] drop received");
    // 获取拖拽数据
    const component =
      parseDragData(event) ||
      (() => {
        const selected = getSelectedComponent();
        return selected ? { ...selected, source: "canvas", offsetX: 0, offsetY: 0 } : null;
      })();

    if (!component) {
      console.warn("No drag data found");
      return;
    }

    const dropContainerId =
      getDropContainerId(event.target) ||
      getDropContainerIdFromPoint(event.clientX, event.clientY);
    if (dropContainerId) {
      const container = findComponentById(components.value, dropContainerId);
      if (container) {
        const inferredSource = component?.source === "canvas" ? "canvas" : "library";
        handleContainerDrop({ container, dragData: component, source: inferredSource, event });
        return;
      }
    }

    // 计算放置位置（画布坐标系）
    const point = getCanvasPoint(event);
    lastDragPoint.value = point;
    const { x, y } = point;

    // 本地拖拽移动：只更新组件位置不重新创建
    if (component?.source === "canvas" && component.id) {
      const target = findComponentById(components.value, component.id);
      if (!target) {
        console.warn("[DesignCanvas] Cannot move component, id not found:", component.id);
      } else {
        const parent = findParentComponent(components.value, component.id);
        const dropContainerId = getDropContainerId(event.target);
        if (parent?.id && dropContainerId === parent.id) {
          designStore.selectComponent(component.id);
          return;
        }
        ensureRootComponent(component.id);

        const offsetX = (component.offsetX || 0) / zoom.value;
        const offsetY = (component.offsetY || 0) / zoom.value;
        let newLeft = x - offsetX;
        let newTop = y - offsetY;

        // 容器百分比宽度转换为像素，避免落点偏移
        if (
          target.type === "Container" &&
          typeof target.style?.width === "string" &&
          target.style.width.includes("%")
        ) {
          const percentage = parseFloat(target.style.width) / 100;
          const canvasWidth = pageConfig.value?.width || 1920;
          const pxWidth = Math.round(canvasWidth * percentage - CANVAS_PADDING * 2);
          designStore.updateComponent(component.id, {
            style: { width: pxWidth },
          });
        }

        if (snapEnabled.value) {
          const snapped = snapPositionToGrid({ x: newLeft, y: newTop }, gridSize.value);
          newLeft = snapped.x;
          newTop = snapped.y;
        }

        designStore.updateComponent(component.id, {
          style: {
            position: "absolute",
            left: Math.round(newLeft),
            top: Math.round(newTop),
          },
        });
        designStore.saveHistory(`移动组件 ${target.name || target.type}`);
        designStore.selectComponent(component.id);
      }
      return;
    }

    // 更新组件位置
    component.style = {
      ...component.style,
      left: x,
      top: y,
      position: "absolute",
    };

    if (snapEnabled.value) {
      const snapped = snapPositionToGrid({ x: component.style.left, y: component.style.top }, gridSize.value);
      component.style.left = snapped.x;
      component.style.top = snapped.y;
    }

    // 如果是容器组件且宽度是百分比字符串，计算实际像素值
    if (
      component.type === "Container" &&
      typeof component.style.width === "string" &&
      component.style.width.includes("%")
    ) {
      const percentage = parseFloat(component.style.width) / 100;
      const canvasWidth = pageConfig.value?.width || 1920;
      component.style.width = Math.round(canvasWidth * percentage - 80); // 减去左右 padding
    }

    // 添加组件到 Store
    designStore.addComponent(component);
    designStore.saveHistory(`添加组件 ${component.name}`);

    console.log("✅ Component dropped:", component);
  } catch (error) {
    console.error("❌ Failed to drop component:", error);
  } finally {
    // 清理拖拽状态
    hideDragPreview();
    hideInsertLine();
    isDragging.value = false;
    draggedComponent.value = null;
    lastDragPoint.value = null;
    lastDragClient.value = null;
    lastDragTarget.value = null;
  }
}

/**
 * 隐藏拖拽预览
 */
function hideDragPreview() {
  if (canvasAuxiliaryRef.value) {
    const dragPreview = canvasAuxiliaryRef.value.getDragPreview();
    if (dragPreview) {
      dragPreview.hide();
    }
  }
}

function handleContainerDragOver(payload) {
  const { container, event } = payload || {};
  if (!container || !event) return;
  if (container.locked) {
    hideInsertLine();
    return;
  }

  const dragData = parseDragData(event) || draggedComponent.value;
  if (dragData?.source === "canvas" && dragData?.id) {
    if (dragData.id === container.id) {
      hideInsertLine();
      return;
    }
    const movingComponent = findComponentById(components.value, dragData.id);
    if (isDescendantComponent(movingComponent, container.id)) {
      hideInsertLine();
      return;
    }
  }

  const excludeId = dragData?.source === "canvas" ? dragData.id : null;
  const info = getContainerInsertInfo(container, event, excludeId);
  const placeholderClientRect = buildPlaceholderClientRect(info, dragData, event);
  showInsertPlaceholder(toCanvasRectFromClient(placeholderClientRect));
}

function handleContainerDragLeave() {
  hideInsertLine();
}

/**
 * 处理放置到容器
 * Task 4.3: 实现拖拽到容器内
 */
function handleContainerDrop(payload) {
  try {
    const { container, dragData, source, event } = payload;

    console.log("? Drop to container:", container.type, container.id);
    console.log("  Drag data:", dragData);
    console.log("  Source:", source);

    let component;

    if (source === "canvas") {
      const componentId = dragData?.id;
      if (!componentId) {
        console.warn("[DesignCanvas] Missing drag component id");
        return;
      }

      if (componentId === container.id) {
        console.warn("[DesignCanvas] Cannot drop component into itself");
        return;
      }

      const movingComponent = findComponentById(components.value, componentId);
      if (!movingComponent) {
        console.warn("[DesignCanvas] Cannot move component, id not found:", componentId);
        return;
      }

      if (isDescendantComponent(movingComponent, container.id)) {
        console.warn("[DesignCanvas] Cannot move component into its descendant:", componentId, container.id);
        return;
      }

      const parent = findParentComponent(components.value, componentId);
      if (parent?.id === container.id) {
        designStore.selectComponent(componentId);
        return;
      }

      const { index: insertIndex } = getContainerInsertInfo(container, event, componentId);
      didDrop.value = true;
      designStore.moveComponent(componentId, container.id, insertIndex);
      designStore.updateComponent(componentId, {
        style: normalizeContainerChildStyle(movingComponent.style || {}),
      });
      designStore.saveHistory(`???? ${movingComponent.name || movingComponent.type} ???`);
      designStore.selectComponent(componentId);
      console.log("? Component moved into container");
      return;
    }

    if (source === "library") {
      // ???????dragData ??????????
      component = dragData;

      if (!component) {
        console.warn("[DesignCanvas] Missing component data from library");
        return;
      }

      // ????????????????
      if (
        container.type === "Container" ||
        container.type === "FlexLayout" ||
        container.type === "Grid"
      ) {
        component.style = normalizeContainerChildStyle(component.style || {});
      }
    } else {
      console.warn("[DesignCanvas] Unknown drag source:", source);
      return;
    }

    const { index: insertIndex } = getContainerInsertInfo(container, event);
    didDrop.value = true;
    // ???????
    designStore.addComponent(component, container.id, insertIndex);
    designStore.saveHistory(`???? ${component.type} ???`);

    console.log("? Component added to container");
  } catch (error) {
    console.error("? Failed to drop component to container:", error);
  } finally {
    // ??????
    hideDragPreview();
    hideInsertLine();
    isDragging.value = false;
    draggedComponent.value = null;
  }
}

/**
 * 初始化坐标系统同步
 * Task 1.4: 实现坐标系统同步
 */
function initCoordinateSync() {
  if (!domLayerRef.value) {
    console.warn("[DesignCanvas] DOM Layer not ready for coordinate sync");
    return;
  }

  // 获取所有组件 ID
  const componentIds = components.value.map((comp) => comp.id);

  // 初始化所有组件的边界
  coordinateSync.updateAllComponentBounds(componentIds);

  // 监听组件尺寸变化
  coordinateSync.observeComponentResize(componentIds, (componentId, bounds) => {
    console.log(`[CoordinateSync] Component ${componentId} resized:`, bounds);
    // TODO: 更新 Canvas Layer 的选择框、对齐线等
  });

  // 监听组件位置变化
  coordinateSync.observeComponentPosition(
    domLayerRef.value,
    (componentId, bounds) => {
      console.log(`[CoordinateSync] Component ${componentId} moved:`, bounds);
      // TODO: 更新 Canvas Layer 的选择框、对齐线等
    }
  );

  console.log("✅ Coordinate sync initialized");
}

/**
 * 更新坐标同步（当组件列表变化时）
 */
function updateCoordinateSync() {
  if (!domLayerRef.value) return;

  const componentIds = components.value.map((comp) => comp.id);

  // 重新监听组件
  coordinateSync.observeComponentResize(componentIds, (componentId, bounds) => {
    console.log(`[CoordinateSync] Component ${componentId} resized:`, bounds);
  });

  // 更新所有组件边界
  coordinateSync.updateAllComponentBounds(componentIds);
}

// Watch components changes to update coordinate sync
watch(
  components,
  () => {
    nextTick(() => {
      updateCoordinateSync();
    });
  },
  { deep: true }
);

watch(
  () => canvasState.scale,
  (nextScale) => {
    if (!Number.isFinite(nextScale) || nextScale <= 0) return;
    if (Math.abs(nextScale - zoom.value) < 0.0001) return;
    if (!viewportRef.value) {
      zoom.value = nextScale;
      return;
    }
    setZoom(nextScale);
  }
);

// Lifecycle
onMounted(async () => {
  console.log("✅ DesignCanvas mounted - Hybrid Rendering Architecture");
  console.log("  - DOM Layer: Rendering components and layout containers");
  console.log(
    "  - Canvas Layer: Auxiliary features (rulers, guides, selection box)"
  );

  // ✅ Task 1.3 - 初始化 Konva Stage 和 Layer (完成)
  // ✅ Task 1.4 - 实现坐标系统同步 (完成)
  // TODO: Task 1.5 - 实现缩放和滚动同步

  // 监听滚动事件
  if (viewportRef.value) {
    viewportRef.value.addEventListener("scroll", handleScroll);
    viewportRef.value.addEventListener("wheel", handleZoomWheel, { passive: false });
    // 补充全局 dragend 监听，避免某些浏览器未在视口上触发
    window.addEventListener("dragend", handleDragEnd);
  }

  // Task 5.1: 监听键盘事件（全选）
  window.addEventListener("keydown", handleKeyDown);
});

onUnmounted(() => {
  console.log("👋 DesignCanvas unmounted");

  // 清理滚动事件监听器
  if (viewportRef.value) {
    viewportRef.value.removeEventListener("scroll", handleScroll);
    viewportRef.value.removeEventListener("wheel", handleZoomWheel);
  }

  // 清理键盘事件监听器
  window.removeEventListener("keydown", handleKeyDown);
  window.removeEventListener("dragend", handleDragEnd);
});

/**
 * 处理滚动事件
 */
function handleScroll(event) {
  if (!viewportRef.value) return;

  scrollX.value = viewportRef.value.scrollLeft;
  scrollY.value = viewportRef.value.scrollTop;
}

function handleZoomWheel(event) {
  if (!event?.ctrlKey && !event?.metaKey) return;
  if (!viewportRef.value) return;
  event.preventDefault();

  const rect = viewportRef.value.getBoundingClientRect();
  const centerX = event.clientX - rect.left;
  const centerY = event.clientY - rect.top;
  const direction = event.deltaY > 0 ? -1 : 1;
  const step = 0.1;
  const nextZoom = zoom.value + direction * step;

  setZoom(nextZoom, centerX, centerY);
}

/**
 * 设置缩放比例（保持中心点不变）
 * Task 1.5: 实现缩放中心点保持
 *
 * 功能说明：
 * - 在缩放时保持画布的视觉中心点不变
 * - 计算缩放前后的中心点偏移，并调整滚动位置
 * - 确保用户看到的画布区域在缩放前后保持一致
 *
 * Requirements:
 * - Requirement 21: 缩放功能
 * - Acceptance Criteria 21.4: 缩放后保持画布中心点不变
 *
 * @param {number} newZoom - 新的缩放比例 (0.1 ~ 5.0)
 * @param {number} centerX - 缩放中心点 X 坐标（相对于视口，可选）
 * @param {number} centerY - 缩放中心点 Y 坐标（相对于视口，可选）
 */
function setZoom(newZoom, centerX = null, centerY = null) {
  if (!viewportRef.value) {
    console.warn("[DesignCanvas] Viewport not ready for zoom");
    return;
  }

  // 限制缩放范围 (10% ~ 500%)
  const clampedZoom = Math.max(0.1, Math.min(5.0, newZoom));

  if (clampedZoom === zoom.value) {
    return; // 缩放比例未变化，无需处理
  }

  const oldZoom = zoom.value;
  const viewport = viewportRef.value;

  // 如果未指定缩放中心点，使用视口中心
  const viewportWidth = viewport.clientWidth;
  const viewportHeight = viewport.clientHeight;

  const zoomCenterXPos = centerX !== null ? centerX : viewportWidth / 2;
  const zoomCenterYPos = centerY !== null ? centerY : viewportHeight / 2;

  // 计算缩放中心点在画布坐标系中的位置（缩放前）
  const canvasX = (viewport.scrollLeft + zoomCenterXPos) / oldZoom;
  const canvasY = (viewport.scrollTop + zoomCenterYPos) / oldZoom;

  // 更新缩放比例
  zoom.value = clampedZoom;
  if (canvasState.scale !== clampedZoom) {
    canvasState.scale = clampedZoom;
  }

  // 等待 DOM 更新后调整滚动位置
  nextTick(() => {
    // 计算新的滚动位置，使得缩放中心点在画布坐标系中的位置保持不变
    const newScrollLeft = canvasX * clampedZoom - zoomCenterXPos;
    const newScrollTop = canvasY * clampedZoom - zoomCenterYPos;

    // 更新滚动位置
    viewport.scrollLeft = newScrollLeft;
    viewport.scrollTop = newScrollTop;

    // 更新滚动状态（触发 Canvas Layer 同步）
    scrollX.value = newScrollLeft;
    scrollY.value = newScrollTop;

    console.log(
      `🔍 Zoom updated: ${oldZoom.toFixed(2)} → ${clampedZoom.toFixed(2)}`
    );
    console.log(
      `  Center point preserved at canvas (${canvasX.toFixed(1)}, ${canvasY.toFixed(1)})`
    );
    console.log(
      `  Scroll adjusted: (${newScrollLeft.toFixed(1)}, ${newScrollTop.toFixed(1)})`
    );
  });
}

// 暴露方法给父组件
defineExpose({
  canvasAuxiliaryRef,
  zoom,
  scrollX,
  scrollY,
  setZoom,
  coordinateSync,
  updateCoordinateSync,
  hideDragPreview,
});
</script>

<style scoped>
/**
 * 设计画布视口
 * - 提供滚动容器
 * - 显示背景网格（可选）
 * - 包含 Canvas Layer 和 DOM Layer
 */
.design-canvas-viewport {
  width: 100%;
  height: 100%;
  overflow: auto;
  background-color: #f5f5f5;
  display: flex;
  align-items: flex-start;
  justify-content: flex-start;
  padding: 40px;
  position: relative;
}

/**
 * 背景网格（可选）
 * - 20px x 20px 网格
 * - 浅灰色线条
 */
.design-canvas-viewport.show-grid {
  background-image:
    linear-gradient(to right, #e0e0e0 1px, transparent 1px),
    linear-gradient(to bottom, #e0e0e0 1px, transparent 1px);
  background-size: 20px 20px;
}

/**
 * Canvas Layer (Konva)
 * - 位于 DOM Layer 上方
 * - z-index: 100
 * - pointer-events: none（不阻挡 DOM 事件）
 * - 用于渲染辅助功能：标尺、对齐线、选择框、拖拽预览
 */
.canvas-layer {
  position: absolute;
  top: 40px;
  left: 40px;
  pointer-events: none;
  /* z-index: 100; */
  /* Konva Stage 将在这里初始化 */
}

/**
 * DOM Layer (Vue Components)
 * - 位于 Canvas Layer 下方
 * - z-index: 1
 * - 渲染所有组件和布局容器
 * - 使用 Vue 组件 + CSS 原生布局
 */
.dom-layer {
  position: relative;
  z-index: 1;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.insert-placeholder {
  position: absolute;
  border: 2px dashed #f56c6c;
  background-color: rgba(245, 108, 108, 0.08);
  border-radius: 4px;
  pointer-events: none;
  z-index: 5;
}
</style>
