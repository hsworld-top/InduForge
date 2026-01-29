<template>
  <main
    class="canvas-container"
    ref="containerRef"
    @wheel="handleZoomWheel"
    @mousemove="handleRulerMouseMove"
    @mouseleave="handleRulerMouseLeave"
    @click="handleContainerClick"
  >
    <div class="ruler-layer">
      <div class="ruler ruler-x" :style="rulerXStyle">
        <div class="ruler-crosshair-x" :style="{ left: `${pointerX}px` }" />
        <span
          v-for="mark in rulerMarksX"
          :key="`x-${mark}`"
          class="ruler-label"
          :style="{ left: `${mark * zoom + translateX}px` }"
        >
          {{ mark }}
        </span>
      </div>
      <div class="ruler ruler-y" :style="rulerYStyle">
        <div class="ruler-crosshair-y" :style="{ top: `${pointerY}px` }" />
        <span
          v-for="mark in rulerMarksY"
          :key="`y-${mark}`"
          class="ruler-label"
          :style="{ top: `${mark * zoom + translateY}px` }"
        >
          {{ mark }}
        </span>
      </div>
    </div>
    <div class="canvas-wrapper" ref="wrapperRef">
      <div
        class="canvas"
        ref="canvasRef"
        :style="canvasStyle"
        @dragover="handleDragOver"
        @drop="handleDrop"
      >
        <div class="absolute inset-0 pointer-events-none">
          <slot name="canvas-layer" />
        </div>
        <div class="absolute inset-0">
          <DesignCanvas />
        </div>
      </div>
    </div>
    <div class="canvas-statusbar">
      <span>画布: {{ width }} × {{ height }}</span>
      <el-divider direction="vertical" />
      <span>缩放: {{ Math.round(zoom * 100) }}%</span>
      <el-tooltip content="重置缩放">
        <el-button size="small" text @click="handleZoomReset">
          <IconEpRefresh />
        </el-button>
      </el-tooltip>
    </div>
  </main>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, provide, ref, toRefs } from "vue";
import { storeToRefs } from "pinia";
import { useEditorStore } from "@/stores/editor-store";
import IconEpRefresh from "~icons/ep/refresh";
import { useDragState, endDrag } from "./use-drag-state";
import DesignCanvas from "./DesignCanvas.vue";

const props = defineProps({
  width: {
    type: Number,
    default: 1920,
  },
  height: {
    type: Number,
    default: 1080,
  },
  zoom: {
    type: Number,
    default: 1,
  },
});

const emit = defineEmits(["zoomChange"]);

const { width, height, zoom } = toRefs(props);

const containerRef = ref(null);
const wrapperRef = ref(null);
const canvasRef = ref(null);
const editorStore = useEditorStore();
const { doc, history, selection, currentPage, docVersion } =
  storeToRefs(editorStore);
const dragState = useDragState();
const minorStep = 10;
const majorStep = 100;
const rulerMax = 5000;
const rulerSize = 18;
const containerSize = ref({ width: 0, height: 0 });
const pointerX = ref(-9999);
const pointerY = ref(-9999);
const isNodeTransforming = ref(false);

/**
 * 处理鼠标移动，更新标尺指示线
 * @param {MouseEvent} event - 鼠标事件
 */
const handleRulerMouseMove = (event) => {
  if (isNodeTransforming.value) return;
  if (!containerRef.value) return;
  const rect = containerRef.value.getBoundingClientRect();
  pointerX.value = Math.min(Math.max(0, event.clientX - rect.left), rect.width);
  pointerY.value = Math.min(Math.max(0, event.clientY - rect.top), rect.height);
};

/**
 * 处理鼠标离开，隐藏标尺指示线
 */
const handleRulerMouseLeave = () => {
  if (isNodeTransforming.value) return;
  pointerX.value = -9999;
  pointerY.value = -9999;
};

/**
 * 处理组件拖拽/缩放的标尺指示
 * @param {CustomEvent} event - 自定义事件
 */
const handleNodeTransform = (event) => {
  if (!containerRef.value || !event?.detail) return;
  const rect = containerRef.value.getBoundingClientRect();
  const x = Number(event.detail.x) || 0;
  const y = Number(event.detail.y) || 0;
  pointerX.value = Math.min(
    Math.max(0, x * zoom.value + translateX.value + rulerSize),
    rect.width
  );
  pointerY.value = Math.min(
    Math.max(0, y * zoom.value + translateY.value + rulerSize),
    rect.height
  );
  isNodeTransforming.value = true;
};

/**
 * 结束组件拖拽/缩放的标尺指示
 */
const handleNodeTransformEnd = () => {
  isNodeTransforming.value = false;
};

const handleGlobalDragOver = (event) => {
  if (!dragState.dragType) return;
  if (!containerRef.value) return;
  if (!containerRef.value.contains(event.target)) return;
  event.preventDefault();
};

const handleGlobalDrop = (event) => {
  if (!dragState.dragType) return;
  if (!canvasRef.value || !containerRef.value) return;
  if (!containerRef.value.contains(event.target)) return;
  event.preventDefault();

  const componentType = dragState.dragType;
  const target = resolveDropTarget(event, componentType);
  if (!target.nodeId || !target.element) return;

  const { x, y } = calcDropOffset(event, target.element);
  insertNode(componentType, target.nodeId, x, y);
  endDrag();
};
const handleGlobalMouseUp = (event) => {
  if (!dragState.dragType) return;
  if (!containerRef.value) {
    endDrag();
    return;
  }

  if (!containerRef.value.contains(event.target)) {
    endDrag();
    return;
  }

  const componentType = dragState.dragType;
  const target = resolveDropTarget(event, componentType);
  if (!target.nodeId || !target.element) {
    endDrag();
    return;
  }

  const { x, y } = calcDropOffset(event, target.element);
  insertNode(componentType, target.nodeId, x, y);
  endDrag();
};

// 向子组件提供当前缩放比例，用于拖拽落点换?
provide("canvasZoom", zoom);

const rootNodeId = computed(() => currentPage.value?.rootNodeId || "");
const translateX = ref(0);
const translateY = ref(0);

const canvasStyle = computed(() => {
  docVersion.value;
  const config = currentPage.value?.config || {};
  const background = config.background || null;
  const showGrid = Boolean(config.showGrid);
  const gridSize = 10;
  const style = {
    width: `${width.value}px`,
    height: `${height.value}px`,
    transform: `translate(${translateX.value + rulerSize}px, ${
      translateY.value + rulerSize
    }px) scale(${zoom.value})`,
    backgroundColor: "#ffffff",
  };

  if (background?.kind === "color") {
    style.backgroundColor = background.value || "#ffffff";
  } else if (background?.kind === "image") {
    style.backgroundImage = `url(${background.value || ""})`;
    style.backgroundSize = "cover";
    style.backgroundRepeat = "no-repeat";
    style.backgroundPosition = "center";
  } else if (background?.kind === "gradient") {
    style.backgroundImage = background.value || "";
    style.backgroundSize = "cover";
    style.backgroundRepeat = "no-repeat";
    style.backgroundPosition = "center";
  }

  if (showGrid) {
    const gridLayer = `
      linear-gradient(rgba(0, 0, 0, 0.08) 1px, transparent 1px),
      linear-gradient(90deg, rgba(0, 0, 0, 0.08) 1px, transparent 1px)
    `;
    if (style.backgroundImage) {
      style.backgroundImage = `${gridLayer}, ${style.backgroundImage}`;
      style.backgroundSize = `${gridSize}px ${gridSize}px, ${style.backgroundSize || "cover"}`;
      style.backgroundRepeat = `repeat, ${style.backgroundRepeat || "no-repeat"}`;
      style.backgroundPosition = `0 0, ${style.backgroundPosition || "center"}`;
    } else {
      style.backgroundImage = gridLayer;
      style.backgroundSize = `${gridSize}px ${gridSize}px`;
    }
  }

  return style;
});

const rulerXStyle = computed(() => {
  const minor = minorStep * zoom.value;
  const major = majorStep * zoom.value;
  return {
    "--ruler-minor": `${minor}px`,
    "--ruler-major": `${major}px`,
    "--ruler-offset": `${translateX.value + rulerSize}px`,
  };
});

const rulerYStyle = computed(() => {
  const minor = minorStep * zoom.value;
  const major = majorStep * zoom.value;
  return {
    "--ruler-minor": `${minor}px`,
    "--ruler-major": `${major}px`,
    "--ruler-offset": `${translateY.value + rulerSize}px`,
  };
});

const rulerMarksX = computed(() => {
  const marks = [];
  const max = rulerMax;
  for (let value = 0; value <= max; value += majorStep) {
    const pos = value * zoom.value + translateX.value + rulerSize;
    if (pos < -majorStep || pos > containerSize.value.width) continue;
    marks.push(value);
  }
  return marks;
});

const rulerMarksY = computed(() => {
  const marks = [];
  const max = rulerMax;
  for (let value = 0; value <= max; value += majorStep) {
    const pos = value * zoom.value + translateY.value + rulerSize;
    if (pos < -majorStep || pos > containerSize.value.height) continue;
    marks.push(value);
  }
  return marks;
});

/**
 * 处理拖拽经过
 * @param {DragEvent} event - 拖拽事件
 */
const handleDragOver = (event) => {
  event.preventDefault();
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = "copy";
  }
};

/**
 * 处理拖拽放置
 * @param {DragEvent} event - 拖拽事件
 */
const handleDrop = (event) => {
  event.preventDefault();
  if (!canvasRef.value) return;

  const payload =
    event.dataTransfer?.getData("application/x-designer-component") ||
    event.dataTransfer?.getData("text/plain");
  const fallbackType = dragState.dragType || "";

  let componentType = "";
  if (payload) {
    try {
      const parsed = JSON.parse(payload);
      componentType = parsed.type || "";
    } catch (error) {
      componentType = payload;
    }
  }

  if (!componentType) {
    componentType = fallbackType;
  }
  if (!componentType) return;

  const target = resolveDropTarget(event, componentType);
  if (!target.nodeId || !target.element) return;

  const { x, y } = calcDropOffset(event, target.element);
  insertNode(componentType, target.nodeId, x, y);
  endDrag();
};

/**
 * 判断组件是否为布局容器类型
 * @param {string} type - 组件类型
 * @returns {boolean}
 */
const isLayoutContainerType = (type) => {
  return [
    "FlexContainer",
    "GridContainer",
    "FreeContainer",
    "ResponsiveLayout",
    "ColumnLayout1",
    "ColumnLayout2",
    "ColumnLayout4",
    "ElContainer",
    "ElLayout",
    "ElLayoutRow",
  ].includes(type);
};

/**
 * 插入组件节点
 * @param {string} type - 组件类型
 * @param {string} parentId - 父节?ID
 * @param {number} x - X 坐标
 * @param {number} y - Y 坐标
 */
const insertNode = (type, parentId, x, y) => {
  if (!parentId) return;
  const parentNode = doc.value?.getNode(parentId);
  const insertIndex = parentNode?.children?.length ?? 0;
  editorStore.insertNode(type, parentId, insertIndex, {
    dropPosition: { x, y },
  });
};

/**
 * 处理画布缩放
 * @param {WheelEvent} event - 滚轮事件
 */
const handleZoomWheel = (event) => {
  if (!event.ctrlKey) return;
  event.preventDefault();

  const step = 0.1;
  const direction = event.deltaY > 0 ? -1 : 1;
  const nextZoom = Math.min(5, Math.max(0.1, zoom.value + step * direction));
  if (nextZoom === zoom.value) return;

  translateX.value = 0;
  translateY.value = 0;
  emit("zoomChange", Number(nextZoom.toFixed(2)));
};

/**
 * 重置缩放比例
 */
const handleZoomReset = () => {
  translateX.value = 0;
  translateY.value = 0;
  emit("zoomChange", 1);
};
/**
 * \u70b9\u51fb\u753b\u5e03\u5916\u90e8\u7a7a\u767d\u533a\u57df\u65f6\u663e\u793a\u9875\u9762\u4fe1\u606f
 * @param {MouseEvent} event - \u9f20\u6807\u4e8b\u4ef6
 */
const handleContainerClick = (event) => {
  const target = event.target;
  if (canvasRef.value && canvasRef.value.contains(target)) {
    return;
  }
  selection.value?.clearSelection();
};


/**
 * 构建布局配置
 * @param {import('@/editor-core').ComponentNode | null} parentNode - 父节? * @param {{x: number, y: number, width: number, height: number}} dropInfo - 放置信息
 * @returns {import('@/editor-core').LayoutItem | null}
 */
const buildLayoutItem = (parentNode, dropInfo) => {
  if (!parentNode) {
    return buildFreeLayoutItem(dropInfo);
  }

  if (
    parentNode.type === "GridContainer" ||
    parentNode.type === "ColumnLayout1" ||
    parentNode.type === "ColumnLayout2" ||
    parentNode.type === "ColumnLayout4"
  ) {
    return buildGridLayoutItem(parentNode);
  }

  if (
    parentNode.type === "FlexContainer" ||
    parentNode.type === "ResponsiveLayout" ||
    parentNode.type === "ElContainer" ||
    parentNode.type === "ElLayout" ||
    parentNode.type === "ElLayoutRow" ||
    parentNode.type === "ElHeader" ||
    parentNode.type === "ElAside" ||
    parentNode.type === "ElMain" ||
    parentNode.type === "ElFooter" ||
    parentNode.type === "ElCol"
  ) {
    return buildFlexLayoutItem();
  }

  if (parentNode.type === "FreeContainer") {
    return buildFreeLayoutItem(dropInfo);
  }

  return buildFreeLayoutItem(dropInfo);
};

/**
 * 构建自由布局配置
 * @param {{x: number, y: number, width: number, height: number}} dropInfo - 放置信息
 * @returns {import('@/editor-core').LayoutItem}
 */
const buildFreeLayoutItem = (dropInfo) => {
  return {
    free: {
      mode: "abs",
      abs: {
        x: Math.max(0, Math.round(dropInfo.x)),
        y: Math.max(0, Math.round(dropInfo.y)),
        w: dropInfo.width,
        h: dropInfo.height,
        z: 1,
      },
    },
  };
};

/**
 * 构建 Flex 布局配置
 * @returns {import('@/editor-core').LayoutItem}
 */
const buildFlexLayoutItem = () => {
  return {
    flex: {
      grow: 0,
      shrink: 0,
      basis: "auto",
    },
  };
};

/**
 * 构建 Grid 布局配置
 * @param {import('@/editor-core').ComponentNode} parentNode - 父节? * @returns {import('@/editor-core').LayoutItem}
 */
const buildGridLayoutItem = (parentNode) => {
  const columns = resolveGridCount(parentNode.props?.columns);
  const colCount = Math.max(1, columns);
  const index = parentNode.children?.length ?? 0;
  const row = Math.floor(index / colCount) + 1;
  const col = (index % colCount) + 1;

  return {
    grid: {
      row,
      col,
      rowSpan: 1,
      colSpan: 1,
    },
  };
};

/**
 * 解析 Grid 列数
 * @param {string | number | undefined} value - 列配? * @returns {number}
 */
const resolveGridCount = (value) => {
  if (typeof value === "number" && Number.isFinite(value)) {
    return Math.max(1, Math.floor(value));
  }

  if (typeof value === "string") {
    const repeatMatch = value.match(/repeat\((\d+)/i);
    if (repeatMatch) {
      const count = Number(repeatMatch[1]);
      if (Number.isFinite(count)) return Math.max(1, Math.floor(count));
    }
    const tokens = value.trim().split(/\s+/).filter(Boolean);
    if (tokens.length > 0) return tokens.length;
  }

  return 1;
};

/**
 * 解析拖拽落点目标容器
 * @param {DragEvent} event - 拖拽事件
 * @param {string} componentType - 组件类型
 * @returns {{ nodeId: string, element: HTMLElement } | { nodeId: string, element: HTMLElement | null }}
 */
const resolveDropTarget = (event, componentType) => {
  if (!doc.value) {
    return { nodeId: rootNodeId.value, element: canvasRef.value };
  }

  const hit = document.elementFromPoint(event.clientX, event.clientY);
  let current = hit;

  while (current && current !== canvasRef.value) {
    const nodeId = current.dataset?.nodeId;
    if (nodeId && isContainerNode(nodeId) && canAcceptChild(nodeId, componentType)) {
      return { nodeId, element: current };
    }
    current = current.parentElement;
  }

  return { nodeId: rootNodeId.value, element: canvasRef.value };
};

/**
 * 判断节点是否为容? * @param {string} nodeId - 节点 ID
 * @returns {boolean}
 */
const isContainerNode = (nodeId) => {
  const node = doc.value?.getNode(nodeId);
  if (!node) return false;
  const manifest = componentRegistry.get(node.type);
  return Boolean(manifest?.isContainer);
};

/**
 * 判断容器是否允许子组? * @param {string} parentId - 父节?ID
 * @param {string} childType - 子组件类? * @returns {boolean}
 */
const canAcceptChild = (parentId, childType) => {
  const node = doc.value?.getNode(parentId);
  if (!node) return false;
  if (node.type === "ElLayout") {
    return childType === "ElLayoutRow";
  }
  if (node.type === "ElLayoutRow") {
    return childType === "ElCol";
  }
  const manifest = componentRegistry.get(node.type);
  const allowed = manifest?.allowedChildren;
  if (!Array.isArray(allowed) || allowed.length === 0) return true;
  return allowed.includes(childType);
};

/**
 * 计算落点相对坐标
 * @param {DragEvent} event - 拖拽事件
 * @param {HTMLElement} element - 目标元素
 * @returns {{ x: number, y: number }}
 */
const calcDropOffset = (event, element) => {
  const rect = element.getBoundingClientRect();
  const offsetX = (event.clientX - rect.left) / zoom.value;
  const offsetY = (event.clientY - rect.top) / zoom.value;
  return {
    x: Math.max(0, Math.round(offsetX)),
    y: Math.max(0, Math.round(offsetY)),
  };
};

/**
 * 获取默认尺寸
 * @param {string} type - 组件类型
 * @param {Object | undefined} manifest - 组件清单
 * @returns {{width: number, height: number}}
 */
const resolveDefaultSize = (type, manifest) => {
  if (manifest?.defaultSize) {
    return {
      width: manifest.defaultSize.width || 120,
      height: manifest.defaultSize.height || 32,
    };
  }

  const sizeMap = {
    FlexContainer: { width: 360, height: 200 },
    FreeContainer: { width: 360, height: 200 },
    GridContainer: { width: 360, height: 200 },
    ElContainer: { width: 360, height: 240 },
    ElLayout: { width: 360, height: 200 },
    Text: { width: 120, height: 32 },
    Button: { width: 120, height: 36 },
  };

  return sizeMap[type] || { width: 160, height: 80 };
};
onMounted(() => {
  window.addEventListener("dragover", handleGlobalDragOver);
  window.addEventListener("drop", handleGlobalDrop);
  window.addEventListener("mouseup", handleGlobalMouseUp);
  window.addEventListener("designer:node-transform", handleNodeTransform);
  window.addEventListener("designer:node-transform-end", handleNodeTransformEnd);
  if (containerRef.value && typeof ResizeObserver !== "undefined") {
    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (!entry) return;
      const { width: w, height: h } = entry.contentRect;
      containerSize.value = { width: w, height: h };
    });
    observer.observe(containerRef.value);
    containerRef.value.__rulerObserver = observer;
  }
});

onBeforeUnmount(() => {
  window.removeEventListener("dragover", handleGlobalDragOver);
  window.removeEventListener("drop", handleGlobalDrop);
  window.removeEventListener("mouseup", handleGlobalMouseUp);
  window.removeEventListener("designer:node-transform", handleNodeTransform);
  window.removeEventListener(
    "designer:node-transform-end",
    handleNodeTransformEnd
  );
  if (containerRef.value?.__rulerObserver) {
    containerRef.value.__rulerObserver.disconnect();
    containerRef.value.__rulerObserver = null;
  }
});
</script>

<style scoped>
.canvas-container {
  position: relative;
}

.canvas-wrapper {
  padding: 0 !important;
  align-items: flex-start !important;
  justify-content: flex-start !important;
}

.canvas {
  position: relative;
  transform-origin: 0 0;
}

.dark .canvas {
}
.ruler-layer {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 5;
}

.ruler-crosshair-x {
  position: absolute;
  top: 0;
  height: 18px;
  width: 2px;
  background: #409eff;
  opacity: 0.9;
  z-index: 3;
}

.ruler-crosshair-y {
  position: absolute;
  left: 0;
  width: 18px;
  height: 2px;
  background: #409eff;
  opacity: 0.9;
  z-index: 3;
}

.ruler-zero {
  position: absolute;
  left: 2px;
  top: 2px;
  font-size: 10px;
  color: #606266;
  background: #f5f7fa;
  padding: 0 2px;
}

.ruler {
  position: absolute;
  color: #606266;
  font-size: 10px;
  background-color: #f5f7fa;
  border-color: #dcdfe6;
  overflow: hidden;
}

.ruler-x {
  left: 0;
  top: 0;
  height: 18px;
  right: 0;
  border-bottom: 1px solid #dcdfe6;
  z-index: 2;
}

.ruler-y {
  left: 0;
  top: 0;
  width: 18px;
  bottom: 0;
  border-right: 1px solid #dcdfe6;
  z-index: 1;
}

.ruler-x::before,
.ruler-x::after,
.ruler-y::before,
.ruler-y::after {
  content: "";
  position: absolute;
  pointer-events: none;
}

.ruler-x::before {
  left: 0;
  right: 0;
  bottom: 0;
  height: 6px;
  background-image: linear-gradient(to right, #c0c4cc 1px, transparent 1px);
  background-size: var(--ruler-minor, 10px) 100%;
  background-position: var(--ruler-offset, 0) 0;
}

.ruler-x::after {
  left: 0;
  right: 0;
  bottom: 0;
  height: 12px;
  background-image: linear-gradient(to right, #909399 1px, transparent 1px);
  background-size: var(--ruler-major, 100px) 100%;
  background-position: var(--ruler-offset, 0) 0;
}

.ruler-y::before {
  top: 0;
  bottom: 0;
  right: 0;
  width: 6px;
  background-image: linear-gradient(to bottom, #c0c4cc 1px, transparent 1px);
  background-size: 100% var(--ruler-minor, 10px);
  background-position: 0 var(--ruler-offset, 0);
}

.ruler-y::after {
  top: 0;
  bottom: 0;
  right: 0;
  width: 12px;
  background-image: linear-gradient(to bottom, #909399 1px, transparent 1px);
  background-size: 100% var(--ruler-major, 100px);
  background-position: 0 var(--ruler-offset, 0);
}

.ruler-label {
  position: absolute;
  padding: 2px 2px 0 2px;
  line-height: 1;
  white-space: nowrap;
}

.ruler-y .ruler-label {
  transform: rotate(-90deg);
  transform-origin: left top;
  left: 2px;
}

</style>
